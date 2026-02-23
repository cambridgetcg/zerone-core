package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	ktypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// writeAxiomFile is a test helper that writes axioms to a temp JSON file.
func writeAxiomFile(t *testing.T, dir string, axioms []*ktypes.GenesisAxiom) string {
	t.Helper()
	path := filepath.Join(dir, "axioms.json")
	data, err := json.MarshalIndent(axioms, "", "  ")
	if err != nil {
		t.Fatalf("marshal axioms: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write axioms: %v", err)
	}
	return path
}

func TestValidateReal777(t *testing.T) {
	// Validate the real 777-axiom file.
	err := runValidate([]string{"../../x/knowledge/types/genesis_axioms.json"})
	if err != nil {
		t.Fatalf("validate 777 axioms: %v", err)
	}
}

func TestValidateDuplicateID(t *testing.T) {
	dir := t.TempDir()
	axioms := []*ktypes.GenesisAxiom{
		{AxiomID: "MATH-001", Statement: "Zero is a natural number", ClaimType: "axiom", Domain: "mathematics", EpistemicCategory: "analytic", Confidence: 1.0, Dependencies: []string{}},
		{AxiomID: "MATH-001", Statement: "Duplicate", ClaimType: "axiom", Domain: "mathematics", EpistemicCategory: "analytic", Confidence: 1.0, Dependencies: []string{}},
	}
	path := writeAxiomFile(t, dir, axioms)
	err := runValidate([]string{path})
	if err == nil {
		t.Fatal("expected error for duplicate IDs")
	}
}

func TestValidateMissingDep(t *testing.T) {
	dir := t.TempDir()
	axioms := []*ktypes.GenesisAxiom{
		{AxiomID: "MATH-001", Statement: "Zero is a natural number", ClaimType: "axiom", Domain: "mathematics", EpistemicCategory: "analytic", Confidence: 1.0, Dependencies: []string{"MATH-999"}},
	}
	path := writeAxiomFile(t, dir, axioms)
	err := runValidate([]string{path})
	if err == nil {
		t.Fatal("expected error for missing dependency")
	}
}

func TestValidateCycle(t *testing.T) {
	dir := t.TempDir()
	axioms := []*ktypes.GenesisAxiom{
		{AxiomID: "MATH-001", Statement: "A", ClaimType: "derived_claim", Domain: "mathematics", EpistemicCategory: "formal", Confidence: 0.9, Dependencies: []string{"MATH-002"}},
		{AxiomID: "MATH-002", Statement: "B", ClaimType: "derived_claim", Domain: "mathematics", EpistemicCategory: "formal", Confidence: 0.9, Dependencies: []string{"MATH-001"}},
	}
	path := writeAxiomFile(t, dir, axioms)
	err := runValidate([]string{path})
	if err == nil {
		t.Fatal("expected error for cycle")
	}
}

func TestInjectIntoGenesis(t *testing.T) {
	dir := t.TempDir()

	// Minimal axiom set
	axioms := []*ktypes.GenesisAxiom{
		{AxiomID: "MATH-001", Statement: "Zero is a natural number", ClaimType: "axiom", Domain: "mathematics", EpistemicCategory: "analytic", Confidence: 1.0, Dependencies: []string{}},
		{AxiomID: "MATH-002", Statement: "Every natural number has a successor", ClaimType: "axiom", Domain: "mathematics", EpistemicCategory: "analytic", Confidence: 1.0, Dependencies: []string{}},
	}
	axiomPath := writeAxiomFile(t, dir, axioms)

	// Minimal genesis.json with empty knowledge state
	genesisPath := filepath.Join(dir, "genesis.json")
	genesis := map[string]interface{}{
		"genesis_time":   "2024-01-01T00:00:00Z",
		"chain_id":       "test-1",
		"initial_height": "1",
		"app_state": map[string]interface{}{
			"knowledge": map[string]interface{}{
				"params": map[string]interface{}{},
				"facts":  []interface{}{},
			},
			"bank": map[string]interface{}{},
		},
	}
	genesisData, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		t.Fatalf("marshal genesis: %v", err)
	}
	if err := os.WriteFile(genesisPath, genesisData, 0644); err != nil {
		t.Fatalf("write genesis: %v", err)
	}

	// Run inject
	err = runInject([]string{axiomPath, genesisPath})
	if err != nil {
		t.Fatalf("inject: %v", err)
	}

	// Read back and verify facts were injected
	resultData, err := os.ReadFile(genesisPath)
	if err != nil {
		t.Fatalf("read result: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resultData, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	appState := result["app_state"].(map[string]interface{})
	knowledge := appState["knowledge"].(map[string]interface{})
	facts := knowledge["facts"].([]interface{})

	if len(facts) != 2 {
		t.Fatalf("expected 2 facts, got %d", len(facts))
	}

	// Verify bank is still there (other modules preserved)
	if _, ok := appState["bank"]; !ok {
		t.Fatal("bank module was lost during inject")
	}
}
