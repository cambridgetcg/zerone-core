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
