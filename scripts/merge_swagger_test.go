//go:build ignore

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestDeduplicateOperationIDsPreservesUniqueAndSuffixesCollisions(t *testing.T) {
	paths := swaggerPaths([]swaggerTestOperation{
		{path: "/reserved", method: "get", operationID: "Shared__get_b"},
		{path: "/c/{thing-id}", method: "get", operationID: "Shared"},
		{path: "/unique", method: "post", operationID: "Unique"},
		{path: "/b", method: "get", operationID: "Shared"},
		{path: "/a", method: "post", operationID: "Shared"},
	})

	deduplicateOperationIDs(paths)

	assertOperationID(t, paths, "/a", "post", "Shared")
	assertOperationID(t, paths, "/b", "get", "Shared__get_b__2")
	assertOperationID(t, paths, "/c/{thing-id}", "get", "Shared__get_c_thing_id")
	assertOperationID(t, paths, "/reserved", "get", "Shared__get_b")
	assertOperationID(t, paths, "/unique", "post", "Unique")
	assertUniqueOperationIDs(t, paths)

	beforeSecondPass, err := json.Marshal(paths)
	if err != nil {
		t.Fatalf("marshal first pass: %v", err)
	}
	deduplicateOperationIDs(paths)
	afterSecondPass, err := json.Marshal(paths)
	if err != nil {
		t.Fatalf("marshal second pass: %v", err)
	}
	if !reflect.DeepEqual(beforeSecondPass, afterSecondPass) {
		t.Fatal("operation ID deduplication is not idempotent")
	}
}

func TestDeduplicateOperationIDsIsIndependentOfInputOrder(t *testing.T) {
	forward := []swaggerTestOperation{
		{path: "/z", method: "get", operationID: "Repeated"},
		{path: "/a", method: "post", operationID: "Repeated"},
		{path: "/m", method: "patch", operationID: "Repeated"},
		{path: "/only", method: "get", operationID: "Only"},
	}
	reverse := make([]swaggerTestOperation, len(forward))
	for i := range forward {
		reverse[len(forward)-1-i] = forward[i]
	}

	forwardPaths := swaggerPaths(forward)
	reversePaths := swaggerPaths(reverse)
	deduplicateOperationIDs(forwardPaths)
	deduplicateOperationIDs(reversePaths)

	if !reflect.DeepEqual(forwardPaths, reversePaths) {
		t.Fatalf("deduplication depends on map insertion order:\nforward: %#v\nreverse: %#v", forwardPaths, reversePaths)
	}
	assertOperationID(t, forwardPaths, "/a", "post", "Repeated")
	assertOperationID(t, forwardPaths, "/m", "patch", "Repeated__patch_m")
	assertOperationID(t, forwardPaths, "/z", "get", "Repeated__get_z")
}

func TestGeneratedSwaggerOperationIDsAreGloballyUnique(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate merge_swagger_test.go")
	}
	swaggerPath := filepath.Join(filepath.Dir(filename), "..", "docs", "swagger-ui", "swagger.json")
	data, err := os.ReadFile(swaggerPath)
	if err != nil {
		t.Fatalf("read generated Swagger: %v", err)
	}

	var spec struct {
		Paths map[string]interface{} `json:"paths"`
	}
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatalf("parse generated Swagger: %v", err)
	}
	if len(spec.Paths) == 0 {
		t.Fatal("generated Swagger has no paths")
	}

	assertAllOperationsHaveIDs(t, spec.Paths)
	assertUniqueOperationIDs(t, spec.Paths)
	assertOperationID(t, spec.Paths, "/zerone/alignment/v1/params", "get", "Params")
	assertOperationID(t, spec.Paths, "/zerone/auth/v1/params", "get", "Params__get_zerone_auth_v1_params")
	assertOperationID(t, spec.Paths, "/zerone/knowledge/v1/domains/{name}", "get", "Domain")
	assertOperationID(t, spec.Paths, "/zerone/ontology/v1/domain/{name}", "get", "Domain__get_zerone_ontology_v1_domain_name")
}

type swaggerTestOperation struct {
	path        string
	method      string
	operationID string
}

func swaggerPaths(operations []swaggerTestOperation) map[string]interface{} {
	paths := make(map[string]interface{}, len(operations))
	for _, operation := range operations {
		pathItem, ok := paths[operation.path].(map[string]interface{})
		if !ok {
			pathItem = make(map[string]interface{})
			paths[operation.path] = pathItem
		}
		pathItem[operation.method] = map[string]interface{}{"operationId": operation.operationID}
	}
	return paths
}

func assertOperationID(t *testing.T, paths map[string]interface{}, path, method, want string) {
	t.Helper()
	pathItem, ok := paths[path].(map[string]interface{})
	if !ok {
		t.Fatalf("missing path %q", path)
	}
	operation, ok := pathItem[method].(map[string]interface{})
	if !ok {
		t.Fatalf("missing %s operation for %q", method, path)
	}
	if got, _ := operation["operationId"].(string); got != want {
		t.Fatalf("%s %s operationId = %q, want %q", method, path, got, want)
	}
}

func assertAllOperationsHaveIDs(t *testing.T, paths map[string]interface{}) {
	t.Helper()
	for path, rawPathItem := range paths {
		pathItem, ok := rawPathItem.(map[string]interface{})
		if !ok {
			t.Fatalf("path item %q is not an object", path)
		}
		for _, method := range swaggerOperationMethods {
			rawOperation, exists := pathItem[method]
			if !exists {
				continue
			}
			operation, ok := rawOperation.(map[string]interface{})
			if !ok {
				t.Fatalf("%s operation for %q is not an object", method, path)
			}
			operationID, ok := operation["operationId"].(string)
			if !ok || operationID == "" {
				t.Fatalf("%s %s has no non-empty operationId", method, path)
			}
		}
	}
}

func assertUniqueOperationIDs(t *testing.T, paths map[string]interface{}) {
	t.Helper()
	seen := make(map[string]swaggerOperation)
	for _, operation := range collectSwaggerOperations(paths) {
		if previous, exists := seen[operation.baseID]; exists {
			t.Fatalf(
				"duplicate operationId %q on %s %s and %s %s",
				operation.baseID,
				previous.method,
				previous.path,
				operation.method,
				operation.path,
			)
		}
		seen[operation.baseID] = operation
	}
}
