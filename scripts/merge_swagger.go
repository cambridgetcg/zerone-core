//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// merge_swagger merges all per-module swagger.json files from tmp-swagger-gen/
// into a single OpenAPI 2.0 spec at docs/swagger-ui/swagger.json.

func main() {
	merged := map[string]interface{}{
		"swagger": "2.0",
		"info": map[string]interface{}{
			"title":       "Zerone API",
			"description": "REST API for the Zerone blockchain — a knowledge-verified AI agent economy.",
			"version":     "1.0.0",
		},
		"host":        "localhost:1317",
		"basePath":    "/",
		"schemes":     []string{"http", "https"},
		"paths":       map[string]interface{}{},
		"definitions": map[string]interface{}{},
	}

	paths := merged["paths"].(map[string]interface{})
	defs := merged["definitions"].(map[string]interface{})

	err := filepath.Walk("tmp-swagger-gen", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		var spec map[string]interface{}
		if err := json.Unmarshal(data, &spec); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}

		if p, ok := spec["paths"].(map[string]interface{}); ok {
			for k, v := range p {
				paths[k] = v
			}
		}
		if d, ok := spec["definitions"].(map[string]interface{}); ok {
			for k, v := range d {
				defs[k] = v
			}
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	deduplicateOperationIDs(paths)

	if err := os.MkdirAll("docs/swagger-ui", 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
		os.Exit(1)
	}

	out, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile("docs/swagger-ui/swagger.json", out, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Merged %d paths, %d definitions → docs/swagger-ui/swagger.json\n", len(paths), len(defs))
}

var swaggerOperationMethods = []string{"delete", "get", "head", "options", "patch", "post", "put"}

type swaggerOperation struct {
	path      string
	method    string
	operation map[string]interface{}
	baseID    string
}

// deduplicateOperationIDs makes the merged document's operation IDs globally
// unique without renaming IDs which were already unique. For each collision,
// the lexicographically first path/method retains the original ID; later
// operations receive a stable suffix derived from their HTTP method and path.
func deduplicateOperationIDs(paths map[string]interface{}) {
	operations := collectSwaggerOperations(paths)
	sort.Slice(operations, func(i, j int) bool {
		if operations[i].path != operations[j].path {
			return operations[i].path < operations[j].path
		}
		return operations[i].method < operations[j].method
	})

	counts := make(map[string]int, len(operations))
	used := make(map[string]struct{}, len(operations))
	for _, operation := range operations {
		counts[operation.baseID]++
		used[operation.baseID] = struct{}{}
	}

	seen := make(map[string]int, len(counts))
	for _, operation := range operations {
		seen[operation.baseID]++
		if counts[operation.baseID] == 1 || seen[operation.baseID] == 1 {
			continue
		}

		baseCandidate := operation.baseID + "__" + operationIDSuffix(operation.method, operation.path)
		candidate := baseCandidate
		for collision := 2; ; collision++ {
			if _, exists := used[candidate]; !exists {
				break
			}
			candidate = fmt.Sprintf("%s__%d", baseCandidate, collision)
		}

		operation.operation["operationId"] = candidate
		used[candidate] = struct{}{}
	}
}

func collectSwaggerOperations(paths map[string]interface{}) []swaggerOperation {
	operations := make([]swaggerOperation, 0, len(paths))
	for path, rawPathItem := range paths {
		pathItem, ok := rawPathItem.(map[string]interface{})
		if !ok {
			continue
		}
		for _, method := range swaggerOperationMethods {
			rawOperation, ok := pathItem[method]
			if !ok {
				continue
			}
			operation, ok := rawOperation.(map[string]interface{})
			if !ok {
				continue
			}
			operationID, ok := operation["operationId"].(string)
			if !ok || operationID == "" {
				continue
			}
			operations = append(operations, swaggerOperation{
				path:      path,
				method:    method,
				operation: operation,
				baseID:    operationID,
			})
		}
	}
	return operations
}

func operationIDSuffix(method, path string) string {
	raw := strings.ToLower(method) + "_" + strings.Trim(path, "/")
	var suffix strings.Builder
	separatorPending := false
	for _, r := range raw {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if separatorPending && suffix.Len() > 0 {
				suffix.WriteByte('_')
			}
			suffix.WriteRune(r)
			separatorPending = false
			continue
		}
		separatorPending = suffix.Len() > 0
	}
	if suffix.Len() == 0 {
		return "operation"
	}
	return suffix.String()
}
