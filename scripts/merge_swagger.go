//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
		"host":     "localhost:1317",
		"basePath": "/",
		"schemes":  []string{"http", "https"},
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
