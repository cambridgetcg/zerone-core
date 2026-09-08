package integration_test

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestEventAudit_AllHandlersEmitEvents verifies that every successful message
// handler in every Zerone module emits at least one event. A handler that
// unconditionally returns an error may opt out with an explicit in-body
// "event-audit: fail-closed" annotation: SDK transaction events are discarded
// on error, so pretending such an event is durable would be misleading.
//
// It scans msg_server.go files, identifies handler functions, and checks
// that each contains an EmitEvent call.
func TestEventAudit_AllHandlersEmitEvents(t *testing.T) {
	root := findProjectRoot(t)
	modulesDir := filepath.Join(root, "x")

	// Collect all msg_server.go files.
	var msgServerFiles []string
	err := filepath.Walk(modulesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() == "msg_server.go" && strings.Contains(path, "/keeper/") {
			msgServerFiles = append(msgServerFiles, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk modules directory: %v", err)
	}

	if len(msgServerFiles) == 0 {
		t.Fatal("no msg_server.go files found under x/")
	}

	// Regex to identify handler functions (methods on msgServer).
	handlerRe := regexp.MustCompile(`^func \(.*\b(?:ms|m|k)\b.*\) (\w+)\(`)
	emitRe := regexp.MustCompile(`EmitEvent|EmitTypedEvent`)
	failClosedRe := regexp.MustCompile(`event-audit:\s*fail-closed`)
	approvedFailClosed := map[string]bool{
		"emergency.ProposeRevert": true,
		"emergency.VoteRevert":    true,
		"gov.AttachUpgradePlan":   true,
	}

	var missing []string
	var unapprovedExemptions []string

	for _, file := range msgServerFiles {
		moduleName := extractModuleName(file)
		handlers := extractHandlers(t, file, handlerRe, emitRe, failClosedRe)
		for _, h := range handlers {
			handlerName := fmt.Sprintf("%s.%s", moduleName, h.name)
			if h.failClosed && !approvedFailClosed[handlerName] {
				unapprovedExemptions = append(unapprovedExemptions, handlerName)
			}
			// A thin same-receiver return wrapper may delegate the emission. Trace
			// the actual AST call into production files; a comment, unrelated
			// emitter, missing helper or recursion cycle cannot satisfy the audit.
			delegatesEvent := !h.hasEvent && !h.failClosed && delegatedHandlerEmits(t, filepath.Dir(file), h.name)
			if !h.hasEvent && !delegatesEvent && !(h.failClosed && approvedFailClosed[handlerName]) {
				missing = append(missing, handlerName)
			}
		}
	}

	if len(unapprovedExemptions) > 0 {
		t.Errorf("handlers use an unreviewed fail-closed event exemption (%d):\n  %s",
			len(unapprovedExemptions), strings.Join(unapprovedExemptions, "\n  "))
	}
	if len(missing) > 0 {
		t.Errorf("handlers missing event emission or explicit fail-closed annotation (%d):\n  %s",
			len(missing), strings.Join(missing, "\n  "))
	}
}

// delegatedHandlerEmits is syntactic event-path coverage, not proof that every
// successful execution reaches the emitter; committed transport tests supply
// that behavioral evidence. Only a single return of a same-receiver call counts
// as delegation, and the resolved helper must contain an actual event call.
func delegatedHandlerEmits(t *testing.T, directory, handler string) bool {
	t.Helper()
	packages, err := parser.ParseDir(token.NewFileSet(), directory, func(info os.FileInfo) bool {
		return strings.HasSuffix(info.Name(), ".go") && !strings.HasSuffix(info.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	methods := map[string]*ast.FuncDecl{}
	var entry string
	for _, pkg := range packages {
		for _, file := range pkg.Files {
			for _, declaration := range file.Decls {
				fn, ok := declaration.(*ast.FuncDecl)
				if !ok || fn.Recv == nil || len(fn.Recv.List) != 1 || len(fn.Recv.List[0].Names) != 1 {
					continue
				}
				receiver := fn.Recv.List[0].Type
				if pointer, ok := receiver.(*ast.StarExpr); ok {
					receiver = pointer.X
				}
				name, ok := receiver.(*ast.Ident)
				if !ok {
					continue
				}
				key := pkg.Name + "." + name.Name + "." + fn.Name.Name
				if methods[key] != nil {
					return false // ambiguous build-tagged implementations
				}
				methods[key] = fn
				if fn.Name.Name == handler {
					if entry != "" {
						return false
					}
					entry = key
				}
			}
		}
	}
	seen := map[string]bool{}
	for key := entry; key != "" && !seen[key]; {
		seen[key] = true
		fn := methods[key]
		if fn == nil || fn.Body == nil {
			return false
		}
		emits := false
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if ok {
				if selector, ok := call.Fun.(*ast.SelectorExpr); ok && (selector.Sel.Name == "EmitEvent" || selector.Sel.Name == "EmitTypedEvent") {
					emits = true
				}
			}
			return true
		})
		if emits {
			return true
		}
		if len(fn.Body.List) != 1 {
			return false
		}
		returned, ok := fn.Body.List[0].(*ast.ReturnStmt)
		if !ok || len(returned.Results) != 1 {
			return false
		}
		call, ok := returned.Results[0].(*ast.CallExpr)
		if !ok {
			return false
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		receiver, ok := selector.X.(*ast.Ident)
		if !ok || receiver.Name != fn.Recv.List[0].Names[0].Name {
			return false
		}
		key = key[:strings.LastIndex(key, ".")+1] + selector.Sel.Name
	}
	return false
}

func TestEventAudit_DelegationRequiresActualReachableHelper(t *testing.T) {
	for _, tc := range []struct {
		name, helper string
		want         bool
	}{
		{"emitter", "func(m *msgServer) helper() { ctx.EventManager().EmitEvent(event) }", true},
		{"removed emitter", "func(m *msgServer) helper() {}", false},
		{"comment only", "func(m *msgServer) helper() { /* EmitEvent(event) */ }", false},
		{"unrelated receiver", "func(m *other) helper() { ctx.EmitEvent(event) }", false},
		{"cycle", "func(m *msgServer) helper() { return m.RateFact() }", false},
		{"missing helper", "func(m *msgServer) unrelated() { ctx.EmitEvent(event) }", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			source := "package keeper\nfunc(m *msgServer) RateFact() { return m.helper() }\n" + tc.helper
			if err := os.WriteFile(filepath.Join(dir, "msg_server.go"), []byte(source), 0600); err != nil {
				t.Fatal(err)
			}
			if got := delegatedHandlerEmits(t, dir, "RateFact"); got != tc.want {
				t.Fatalf("delegated emission = %t, want %t", got, tc.want)
			}
		})
	}
}

// TestEventAudit_EventTypeFormat verifies that all event type strings
// follow the zerone.<module>.<action> convention.
func TestEventAudit_EventTypeFormat(t *testing.T) {
	root := findProjectRoot(t)
	modulesDir := filepath.Join(root, "x")

	// Match sdk.NewEvent("...") calls.
	eventTypeRe := regexp.MustCompile(`sdk\.NewEvent\("([^"]+)"`)
	validFormatRe := regexp.MustCompile(`^zerone\.[a-z_]+\.[a-z_]+$`)

	var violations []string

	err := filepath.Walk(modulesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		// Skip test files and protobuf generated files.
		if strings.HasSuffix(path, "_test.go") || strings.Contains(path, ".pb.go") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		matches := eventTypeRe.FindAllStringSubmatch(string(data), -1)
		for _, match := range matches {
			eventType := match[1]
			if !validFormatRe.MatchString(eventType) {
				relPath, _ := filepath.Rel(root, path)
				violations = append(violations, fmt.Sprintf("%s: %q", relPath, eventType))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk modules directory: %v", err)
	}

	if len(violations) > 0 {
		t.Errorf("event types not matching zerone.<module>.<action> format (%d):\n  %s",
			len(violations), strings.Join(violations, "\n  "))
	}
}

// TestEventAudit_NoSensitiveData verifies that events don't contain
// attributes that might leak sensitive data.
func TestEventAudit_NoSensitiveData(t *testing.T) {
	root := findProjectRoot(t)
	modulesDir := filepath.Join(root, "x")

	// Attribute names that should never appear in events.
	sensitiveAttrs := []string{
		"private_key", "secret", "password", "mnemonic", "seed_phrase",
		"raw_content", "plaintext",
	}
	attrRe := regexp.MustCompile(`sdk\.NewAttribute\("([^"]+)"`)

	var violations []string

	err := filepath.Walk(modulesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		matches := attrRe.FindAllStringSubmatch(string(data), -1)
		for _, match := range matches {
			attrName := match[1]
			for _, sensitive := range sensitiveAttrs {
				if strings.Contains(strings.ToLower(attrName), sensitive) {
					relPath, _ := filepath.Rel(root, path)
					violations = append(violations, fmt.Sprintf("%s: sensitive attribute %q", relPath, attrName))
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk modules directory: %v", err)
	}

	if len(violations) > 0 {
		t.Errorf("events contain potentially sensitive attributes:\n  %s",
			strings.Join(violations, "\n  "))
	}
}

// TestEventAudit_AttributeValuesAreStrings verifies that event attributes
// use string values (not fmt.Sprintf with %v or %d directly in NewAttribute).
// All attribute values MUST be strings per CometBFT requirement.
func TestEventAudit_AttributeValuesAreStrings(t *testing.T) {
	root := findProjectRoot(t)
	modulesDir := filepath.Join(root, "x")

	// Check for raw numeric types passed directly to NewAttribute.
	// Valid: sdk.NewAttribute("key", "value")
	// Valid: sdk.NewAttribute("key", fmt.Sprintf("%d", x))
	// Invalid: sdk.NewAttribute("key", 42)
	badAttrRe := regexp.MustCompile(`sdk\.NewAttribute\("[^"]+",\s*\d+\s*\)`)

	var violations []string

	err := filepath.Walk(modulesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		matches := badAttrRe.FindAllString(string(data), -1)
		for _, match := range matches {
			relPath, _ := filepath.Rel(root, path)
			violations = append(violations, fmt.Sprintf("%s: %s", relPath, match))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk modules directory: %v", err)
	}

	if len(violations) > 0 {
		t.Errorf("event attributes with non-string values:\n  %s",
			strings.Join(violations, "\n  "))
	}
}

// TestEventAudit_DocumentationCompleteness verifies that every event type
// in the codebase has a corresponding entry in docs/EVENTS.md.
func TestEventAudit_DocumentationCompleteness(t *testing.T) {
	root := findProjectRoot(t)
	eventsDocPath := filepath.Join(root, "docs", "EVENTS.md")

	// Collect all event types from the codebase.
	// Handles both inline sdk.NewEvent("type" and multiline sdk.NewEvent(\n"type"
	codebaseEvents := make(map[string]bool)

	for _, sourceDir := range []string{
		filepath.Join(root, "x"),
		filepath.Join(root, "app"),
	} {
		err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || strings.Contains(path, ".pb.go") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for _, eventType := range extractEventTypes(string(data)) {
				codebaseEvents[eventType] = true
			}
			return nil
		})
		if err != nil {
			t.Fatalf("failed to walk source directory %s: %v", sourceDir, err)
		}
	}

	// Parse documented events from EVENTS.md (### zerone.module.action headings).
	docData, err := os.ReadFile(eventsDocPath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", eventsDocPath, err)
	}

	docHeadingRe := regexp.MustCompile(`(?m)^### (zerone\.\w+\.\w+)`)
	documentedEvents := make(map[string]bool)
	for _, match := range docHeadingRe.FindAllStringSubmatch(string(docData), -1) {
		documentedEvents[match[1]] = true
	}

	// Check for undocumented events.
	var undocumented []string
	for event := range codebaseEvents {
		if !documentedEvents[event] {
			undocumented = append(undocumented, event)
		}
	}

	if len(undocumented) > 0 {
		t.Errorf("events in codebase but missing from docs/EVENTS.md (%d):\n  %s",
			len(undocumented), strings.Join(undocumented, "\n  "))
	}

	// Check for phantom docs (documented but not in codebase).
	var phantom []string
	for event := range documentedEvents {
		if !codebaseEvents[event] {
			phantom = append(phantom, event)
		}
	}

	if len(phantom) > 0 {
		t.Errorf("events in docs/EVENTS.md but not found in codebase (%d):\n  %s",
			len(phantom), strings.Join(phantom, "\n  "))
	}
}

// TestEventAudit_AttributeCompleteness verifies minimum required attributes
// for all events based on their context.
func TestEventAudit_AttributeCompleteness(t *testing.T) {
	root := findProjectRoot(t)
	modulesDir := filepath.Join(root, "x")

	attrRe := regexp.MustCompile(`sdk\.NewAttribute\("([^"]+)"`)
	strQuoteRe := regexp.MustCompile(`"(zerone\.[a-z_]+\.[a-z_]+)"`)

	type eventInfo struct {
		eventType string
		attrs     map[string]bool
		file      string
	}

	var allEvents []eventInfo

	err := filepath.Walk(modulesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || strings.Contains(path, ".pb.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		content := string(data)
		relPath, _ := filepath.Rel(root, path)

		// Split by sdk.NewEvent to find each event block.
		parts := strings.Split(content, "sdk.NewEvent(")
		for i := 1; i < len(parts); i++ {
			part := parts[i]
			if len(part) > 1000 {
				part = part[:1000]
			}

			// Extract event type from the first quoted string after sdk.NewEvent(
			typeMatch := strQuoteRe.FindStringSubmatch(part[:min(200, len(part))])
			if typeMatch == nil {
				continue
			}

			attrs := make(map[string]bool)
			for _, am := range attrRe.FindAllStringSubmatch(part, -1) {
				attrs[am[1]] = true
			}

			allEvents = append(allEvents, eventInfo{
				eventType: typeMatch[1],
				attrs:     attrs,
				file:      relPath,
			})
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk modules directory: %v", err)
	}

	var violations []string

	for _, ev := range allEvents {
		// Rule 1: Every event must have at least one attribute.
		if len(ev.attrs) == 0 {
			violations = append(violations, fmt.Sprintf(
				"%s: %s has no attributes", ev.file, ev.eventType))
		}
	}

	if len(violations) > 0 {
		t.Errorf("event attribute completeness violations (%d):\n  %s",
			len(violations), strings.Join(violations, "\n  "))
	}
}

// --- helpers ---

// extractEventTypes finds all event type strings in Go source, handling both
// inline sdk.NewEvent("type" and multiline sdk.NewEvent(\n\t"type" patterns.
func extractEventTypes(content string) []string {
	eventTypeRe := regexp.MustCompile(`"(zerone\.[a-z_]+\.[a-z_]+)"`)
	var types []string

	// Split by sdk.NewEvent( and look for the first quoted zerone.* string
	// in the ~200 chars after it.
	parts := strings.Split(content, "sdk.NewEvent(")
	for i := 1; i < len(parts); i++ {
		snippet := parts[i]
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		if m := eventTypeRe.FindStringSubmatch(snippet); m != nil {
			types = append(types, m[1])
		}
	}
	return types
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type handlerInfo struct {
	name       string
	hasEvent   bool
	failClosed bool
}

func extractHandlers(t *testing.T, path string, handlerRe, emitRe, failClosedRe *regexp.Regexp) []handlerInfo {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open %s: %v", path, err)
	}
	defer file.Close()

	// Delegation pattern: handlers that delegate to keeper methods which
	// emit events internally (e.g., return ms.Keeper.Foo(ctx, msg)).
	delegateRe := regexp.MustCompile(`\.\w+\.\w+\(|\.Handle\w+\(|\.VoteProposal\(|\.graduateMentorship\(`)

	var handlers []handlerInfo
	braceDepth := 0
	inHandler := false
	enteredBody := false // true once we've seen the opening {

	// Non-handler methods and internal helpers to skip.
	skipMethods := map[string]bool{
		"NewMsgServerImpl":         true,
		"NewResearchMsgServerImpl": true,
		"markAccountInactive":      true,
		"checkEligibility":         true,
		"addVoteAudit":             true,
		"isValidStatusTransition":  true,
		"intersectPermissions":     true,
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		if !inHandler {
			if m := handlerRe.FindStringSubmatch(line); m != nil {
				name := m[1]
				if skipMethods[name] {
					continue
				}
				inHandler = true
				braceDepth = 0
				enteredBody = false
				handlers = append(handlers, handlerInfo{name: name})
			}
		}

		if inHandler {
			braceDepth += strings.Count(line, "{") - strings.Count(line, "}")

			if braceDepth > 0 {
				enteredBody = true
			}

			if emitRe.MatchString(line) || delegateRe.MatchString(line) {
				handlers[len(handlers)-1].hasEvent = true
			}
			if failClosedRe.MatchString(line) {
				handlers[len(handlers)-1].failClosed = true
			}

			if enteredBody && braceDepth <= 0 {
				inHandler = false
			}
		}
	}

	return handlers
}

func extractModuleName(path string) string {
	// Extract module name from path like .../x/knowledge/keeper/msg_server.go
	parts := strings.Split(filepath.ToSlash(path), "/")
	for i, p := range parts {
		if p == "x" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return "unknown"
}

func findProjectRoot(t *testing.T) string {
	t.Helper()
	// Walk up from current working directory to find go.mod.
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root (go.mod)")
		}
		dir = parent
	}
}
