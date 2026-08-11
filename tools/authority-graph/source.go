package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	maxManifestBytes = 4 << 20
	maxSourceBytes   = 16 << 20

	customStakingKeeperImport = "github.com/zerone-chain/zerone/x/staking/keeper"
)

type sourceDiscovery struct {
	CustomStakingConsumers []string
	Constructors           []string
	AbsentTargetModules    []string
}

type constructorExpectation struct {
	ID         string
	Field      string
	ImportPath string
}

var expectedConstructors = []constructorExpectation{
	{"sdk-staking", "StakingKeeper", "github.com/cosmos/cosmos-sdk/x/staking/keeper"},
	{"custom-staking", "ZeroneStakingKeeper", customStakingKeeperImport},
	{"sdk-gov", "GovKeeper", "github.com/cosmos/cosmos-sdk/x/gov/keeper"},
	{"custom-gov", "ZeroneGovKeeper", "github.com/zerone-chain/zerone/x/gov/keeper"},
	{"ontology", "ZeroneOntologyKeeper", "github.com/zerone-chain/zerone/x/ontology/keeper"},
	{"knowledge", "KnowledgeKeeper", "github.com/zerone-chain/zerone/x/knowledge/keeper"},
}

var expectedAdapters = map[string]string{
	"NewStakingKeeperAdapter":              "knowledge",
	"NewGovStakingKeeperAdapter":           "gov",
	"NewQualificationStakingKeeperAdapter": "qualification",
	"NewEmergencyStakingAdapter":           "emergency",
	"NewAlignmentStakingAdapter":           "alignment",
	"NewClaimingPotStakingAdapter":         "claiming-pot",
}

func validateSource(root string, m manifest, issues *issueSet) (int, sourceDiscovery) {
	verifiedAnchors := validateSourceAnchors(root, m, issues)
	discovery := discoverAppAuthority(root, issues)
	discovery.AbsentTargetModules = validateTargetModulesAbsent(root, issues)
	return verifiedAnchors, discovery
}

func validateSourceAnchors(root string, m manifest, issues *issueSet) int {
	verified := 0
	for _, anchor := range m.SourceAnchors {
		before := len(issues.items)
		data, reason := readRepositoryFile(root, anchor.Path, maxSourceBytes)
		if reason != "" {
			issues.add("SOURCE_ANCHOR_READ_FAILED", fmt.Sprintf("source anchor %s: %s", safeID(anchor.ID), reason))
			continue
		}
		digest := sha256Hex(data)
		if digest != anchor.SHA256 {
			issues.add("SOURCE_ANCHOR_SHA256_MISMATCH", fmt.Sprintf("source anchor %s does not match its declared SHA-256", safeID(anchor.ID)))
		}
		for i, snippet := range anchor.RequiredSnippets {
			if snippet == "" || !strings.Contains(string(data), snippet) {
				issues.add("SOURCE_ANCHOR_REQUIRED_SNIPPET_MISSING", fmt.Sprintf("source anchor %s is missing required snippet %d", safeID(anchor.ID), i+1))
			}
		}
		for i, snippet := range anchor.ForbiddenSnippets {
			if snippet == "" || strings.Contains(string(data), snippet) {
				issues.add("SOURCE_ANCHOR_FORBIDDEN_SNIPPET_PRESENT", fmt.Sprintf("source anchor %s contains forbidden snippet %d", safeID(anchor.ID), i+1))
			}
		}
		if len(issues.items) == before {
			verified++
		}
	}
	if m.SourceDesign.RepositoryPath != "" {
		for _, anchor := range m.SourceAnchors {
			if anchor.ID == "authoritative-state-design" && (anchor.Path != m.SourceDesign.RepositoryPath || anchor.SHA256 != m.SourceDesign.SHA256) {
				issues.add("SOURCE_DESIGN_ANCHOR_MISMATCH", "authoritative-state-design must match sourceDesign path and SHA-256")
			}
		}
	}
	return verified
}

func validateTargetModulesAbsent(root string, issues *issueSet) []string {
	targets := []expectedNode{
		{"controller", "x/controller", "", "", nil},
		{"verifier-profile", "x/verifier_profile", "", "", nil},
		{"electorate", "x/electorate", "", "", nil},
		{"legacy-claims", "x/legacy_claims", "", "", nil},
	}
	absent := make([]string, 0, len(targets))
	for _, target := range targets {
		if !isSafeRepositoryPath(target.Module) {
			issues.add("TARGET_MODULE_PATH_INVALID", fmt.Sprintf("target module %s has an unsafe path", target.ID))
			continue
		}
		_, err := os.Lstat(filepath.Join(root, filepath.FromSlash(target.Module)))
		switch {
		case os.IsNotExist(err):
			absent = append(absent, target.ID)
		case err != nil:
			issues.add("TARGET_MODULE_INSPECTION_FAILED", fmt.Sprintf("target module %s could not be inspected", target.ID))
		default:
			issues.add("TARGET_ONLY_MODULE_PRESENT", fmt.Sprintf("target-only module %s must remain absent", target.ID))
		}
	}
	sort.Strings(absent)
	return absent
}

func discoverAppAuthority(root string, issues *issueSet) sourceDiscovery {
	files := listProductionGoFiles(root, "app", issues)
	adapterCounts := make(map[string]int)
	constructorCallCounts := make(map[string]int)
	constructorAssignmentCounts := make(map[string]int)

	constructorByImport := make(map[string]constructorExpectation, len(expectedConstructors))
	constructorByField := make(map[string]constructorExpectation, len(expectedConstructors))
	for _, constructor := range expectedConstructors {
		constructorByImport[constructor.ImportPath] = constructor
		constructorByField[constructor.Field] = constructor
	}

	for _, relativePath := range files {
		data, reason := readRepositoryFile(root, relativePath, maxSourceBytes)
		if reason != "" {
			issues.add("SOURCE_GO_READ_FAILED", fmt.Sprintf("%s: %s", relativePath, reason))
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), relativePath, data, parser.SkipObjectResolution)
		if err != nil {
			issues.add("SOURCE_GO_PARSE_FAILED", fmt.Sprintf("%s is not valid Go source", relativePath))
			continue
		}
		imports := resolvedImports(parsed, relativePath, issues)

		ast.Inspect(parsed, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			qualifier, ok := selector.X.(*ast.Ident)
			if !ok {
				return true
			}
			importPath, ok := imports[qualifier.Name]
			if !ok {
				return true
			}
			if constructor, ok := constructorByImport[importPath]; ok && selector.Sel.Name == "NewKeeper" {
				constructorCallCounts[constructor.ID]++
			}
			if importPath == customStakingKeeperImport && strings.HasPrefix(selector.Sel.Name, "New") && strings.HasSuffix(selector.Sel.Name, "Adapter") {
				adapterCounts[selector.Sel.Name]++
				if !hasExactCustomStakingArgument(call.Args) {
					issues.add("SOURCE_CUSTOM_STAKING_ADAPTER_ARGUMENT_INVALID", fmt.Sprintf("%s must receive only app.ZeroneStakingKeeper", selector.Sel.Name))
				}
			}
			return true
		})

		ast.Inspect(parsed, func(node ast.Node) bool {
			assignment, ok := node.(*ast.AssignStmt)
			if !ok || len(assignment.Lhs) != len(assignment.Rhs) {
				return true
			}
			for i, lhs := range assignment.Lhs {
				field, ok := appField(lhs)
				if !ok {
					continue
				}
				constructor, expectedField := constructorByField[field]
				if !expectedField {
					continue
				}
				if callUsesConstructor(assignment.Rhs[i], imports, constructor.ImportPath) {
					constructorAssignmentCounts[constructor.ID]++
				}
			}
			return true
		})
	}

	validateAdapterSet(adapterCounts, issues)
	constructors := make([]string, 0, len(expectedConstructors))
	for _, constructor := range expectedConstructors {
		if constructorCallCounts[constructor.ID] != 1 || constructorAssignmentCounts[constructor.ID] != 1 {
			issues.add(
				"SOURCE_KEEPER_CONSTRUCTOR_SET_MISMATCH",
				fmt.Sprintf("constructor %s must occur once and assign app.%s once", constructor.ID, constructor.Field),
			)
			continue
		}
		constructors = append(constructors, constructor.ID)
	}
	sort.Strings(constructors)

	consumers := make([]string, 0, len(expectedAdapters))
	for adapter, consumer := range expectedAdapters {
		if adapterCounts[adapter] == 1 {
			consumers = append(consumers, consumer)
		}
	}
	sort.Strings(consumers)
	return sourceDiscovery{
		CustomStakingConsumers: consumers,
		Constructors:           constructors,
	}
}

func listProductionGoFiles(root, directory string, issues *issueSet) []string {
	if !isSafeRepositoryPath(directory) {
		issues.add("SOURCE_SCAN_PATH_INVALID", "Go source scan path is unsafe")
		return nil
	}
	absDirectory := filepath.Join(root, filepath.FromSlash(directory))
	info, err := os.Lstat(absDirectory)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		issues.add("SOURCE_SCAN_FAILED", "app source directory is missing, linked, or not a directory")
		return nil
	}
	var files []string
	err = filepath.WalkDir(absDirectory, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk failed")
		}
		relative, err := filepath.Rel(root, current)
		if err != nil {
			return fmt.Errorf("relative path failed")
		}
		relative = filepath.ToSlash(relative)
		if entry.Type()&os.ModeSymlink != 0 {
			issues.add("SOURCE_SCAN_SYMLINK_REJECTED", fmt.Sprintf("source scan encountered linked path %s", relative))
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type().IsRegular() && strings.HasSuffix(entry.Name(), ".go") && !strings.HasSuffix(entry.Name(), "_test.go") {
			files = append(files, relative)
		}
		return nil
	})
	if err != nil {
		issues.add("SOURCE_SCAN_FAILED", "app source directory could not be traversed")
	}
	sort.Strings(files)
	if len(files) == 0 {
		issues.add("SOURCE_SCAN_FAILED", "app source scan found no production Go files")
	}
	return files
}

func resolvedImports(file *ast.File, relativePath string, issues *issueSet) map[string]string {
	imports := make(map[string]string, len(file.Imports))
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			issues.add("SOURCE_GO_IMPORT_INVALID", fmt.Sprintf("%s contains an invalid import", relativePath))
			continue
		}
		name := path.Base(importPath)
		if spec.Name != nil {
			name = spec.Name.Name
		}
		if name == "." && isAuthorityKeeperImport(importPath) {
			issues.add("SOURCE_GO_IMPORT_INVALID", fmt.Sprintf("%s dot-imports an authority keeper", relativePath))
			continue
		}
		if name == "_" {
			continue
		}
		if prior, duplicate := imports[name]; duplicate && prior != importPath {
			issues.add("SOURCE_GO_IMPORT_INVALID", fmt.Sprintf("%s has an ambiguous import alias", relativePath))
			continue
		}
		imports[name] = importPath
	}
	return imports
}

func isAuthorityKeeperImport(importPath string) bool {
	for _, constructor := range expectedConstructors {
		if constructor.ImportPath == importPath {
			return true
		}
	}
	return false
}

func hasExactCustomStakingArgument(args []ast.Expr) bool {
	if len(args) != 1 {
		return false
	}
	selector, ok := args[0].(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "ZeroneStakingKeeper" {
		return false
	}
	identifier, ok := selector.X.(*ast.Ident)
	return ok && identifier.Name == "app"
}

func appField(expr ast.Expr) (string, bool) {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	identifier, ok := selector.X.(*ast.Ident)
	if !ok || identifier.Name != "app" {
		return "", false
	}
	return selector.Sel.Name, true
}

func callUsesConstructor(expr ast.Expr, imports map[string]string, expectedImport string) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "NewKeeper" {
		return false
	}
	qualifier, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}
	return imports[qualifier.Name] == expectedImport
}

func validateAdapterSet(counts map[string]int, issues *issueSet) {
	names := make([]string, 0, len(counts))
	for name, count := range counts {
		names = append(names, fmt.Sprintf("%s:%d", name, count))
	}
	sort.Strings(names)
	valid := len(counts) == len(expectedAdapters)
	for expected := range expectedAdapters {
		if counts[expected] != 1 {
			valid = false
		}
	}
	if !valid {
		issues.add("SOURCE_CUSTOM_STAKING_ADAPTER_SET_MISMATCH", fmt.Sprintf("custom staking adapters must match the reviewed six; discovered [%s]", strings.Join(names, ", ")))
	}
}

func readRepositoryFile(root, relative string, limit int64) ([]byte, string) {
	if !isSafeRepositoryPath(relative) {
		return nil, "path is not a safe repository-relative path"
	}
	rootInfo, err := os.Stat(root)
	if err != nil || !rootInfo.IsDir() {
		return nil, "repository root is missing or not a directory"
	}
	parts := strings.Split(relative, "/")
	current := root
	for i, part := range parts {
		current = filepath.Join(current, filepath.FromSlash(part))
		info, err := os.Lstat(current)
		if err != nil {
			return nil, "file is missing or unreadable"
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, "linked paths are not accepted"
		}
		if i < len(parts)-1 && !info.IsDir() {
			return nil, "a parent path is not a directory"
		}
		if i == len(parts)-1 && !info.Mode().IsRegular() {
			return nil, "path is not a regular file"
		}
	}
	file, err := os.Open(current)
	if err != nil {
		return nil, "file is unreadable"
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, "file could not be read"
	}
	if int64(len(data)) > limit {
		return nil, "file exceeds the size limit"
	}
	return data, ""
}

func isSafeRepositoryPath(value string) bool {
	if value == "" || value == "." || strings.Contains(value, "\\") || strings.HasPrefix(value, "/") {
		return false
	}
	if path.Clean(value) != value {
		return false
	}
	for _, component := range strings.Split(value, "/") {
		if component == "" || component == "." || component == ".." {
			return false
		}
	}
	return true
}

func sha256Hex(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func safeID(value string) string {
	if value == "" {
		return "<missing>"
	}
	for _, r := range value {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' && r != '_' {
			return "<invalid>"
		}
	}
	return value
}
