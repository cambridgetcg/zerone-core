package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateAxiomID_Valid(t *testing.T) {
	valid := []string{
		"MATH-001", "PHYS-111", "AGRT-004a", "AGRT-004a-ii",
		"CS-001", "THEO-007", "AP-001", "LOGIC-077",
	}
	for _, id := range valid {
		require.NoError(t, ValidateAxiomID(id), "expected valid: %s", id)
	}
}

func TestValidateAxiomID_Invalid(t *testing.T) {
	invalid := []string{
		"",             // empty
		"math-001",     // lowercase prefix
		"MATH001",      // no dash
		"MATH-01",      // only 2 digits
		"UNKNOWN-001",  // unknown prefix
		"MATH 001",     // space instead of dash
	}
	for _, id := range invalid {
		require.Error(t, ValidateAxiomID(id), "expected invalid: %s", id)
	}
}

func TestParseAxioms_ValidJSON(t *testing.T) {
	data := `[
		{"axiom_id": "MATH-001", "statement": "test", "type": "axiom", "domain": "mathematics", "epistemic_category": "analytic", "confidence": 1.0, "dependencies": []},
		{"axiom_id": "MATH-002", "statement": "test2", "type": "axiom", "domain": "mathematics", "epistemic_category": "analytic", "confidence": 1.0, "dependencies": ["MATH-001"]}
	]`
	axioms, err := ParseAxioms([]byte(data))
	require.NoError(t, err)
	require.Len(t, axioms, 2)
	require.Equal(t, "MATH-001", axioms[0].AxiomID)
	require.Equal(t, []string{"MATH-001"}, axioms[1].Dependencies)
}

func TestParseAxioms_InvalidJSON(t *testing.T) {
	_, err := ParseAxioms([]byte(`not json`))
	require.Error(t, err)
}

func TestValidateAxioms_Empty(t *testing.T) {
	err := ValidateAxioms(nil, []string{"mathematics"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty")
}

func TestValidateAxioms_DuplicateID(t *testing.T) {
	axioms := []*GenesisAxiom{
		{AxiomID: "MATH-001", Statement: "a", ClaimType: "axiom", Domain: "mathematics", EpistemicCategory: "analytic", Confidence: 1.0},
		{AxiomID: "MATH-001", Statement: "b", ClaimType: "axiom", Domain: "mathematics", EpistemicCategory: "analytic", Confidence: 1.0},
	}
	err := ValidateAxioms(axioms, []string{"mathematics"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "duplicate")
}

func TestValidateAxiomDAG_NoCycle(t *testing.T) {
	axioms := []*GenesisAxiom{
		{AxiomID: "MATH-001", Dependencies: nil},
		{AxiomID: "MATH-002", Dependencies: []string{"MATH-001"}},
		{AxiomID: "MATH-003", Dependencies: []string{"MATH-001", "MATH-002"}},
	}
	require.NoError(t, ValidateAxiomDAG(axioms))
}

func TestValidateAxiomDAG_Cycle(t *testing.T) {
	axioms := []*GenesisAxiom{
		{AxiomID: "MATH-001", Dependencies: []string{"MATH-002"}},
		{AxiomID: "MATH-002", Dependencies: []string{"MATH-001"}},
	}
	err := ValidateAxiomDAG(axioms)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cycle")
}

func TestValidateAxiomDAG_SelfDependency(t *testing.T) {
	axioms := []*GenesisAxiom{
		{AxiomID: "MATH-001", Dependencies: []string{"MATH-001"}},
	}
	err := ValidateAxiomDAG(axioms)
	require.Error(t, err)
	require.Contains(t, err.Error(), "self-dependency")
}

func TestValidateStratumConsistency_Valid(t *testing.T) {
	axioms := []*GenesisAxiom{
		{AxiomID: "MATH-001", ClaimType: "axiom", Dependencies: nil},
		{AxiomID: "MATH-002", ClaimType: "definition", Dependencies: []string{"MATH-001"}},
		{AxiomID: "MATH-003", ClaimType: "derived_claim", Dependencies: []string{"MATH-002"}},
	}
	require.NoError(t, ValidateStratumConsistency(axioms))
}

func TestValidateStratumConsistency_Violation(t *testing.T) {
	axioms := []*GenesisAxiom{
		{AxiomID: "MATH-001", ClaimType: "derived_claim", Dependencies: nil},
		{AxiomID: "MATH-002", ClaimType: "axiom", Dependencies: []string{"MATH-001"}},
	}
	err := ValidateStratumConsistency(axioms)
	require.Error(t, err)
	require.Contains(t, err.Error(), "stratum violation")
}

func TestValidateDerivedConfidence_Valid(t *testing.T) {
	axioms := []*GenesisAxiom{
		{AxiomID: "MATH-001", ClaimType: "axiom", Confidence: 1.0},
		{AxiomID: "MATH-002", ClaimType: "derived_claim", Confidence: 0.95, Dependencies: []string{"MATH-001"}},
	}
	require.NoError(t, ValidateDerivedConfidence(axioms))
}

func TestValidateDerivedConfidence_Exceeds(t *testing.T) {
	axioms := []*GenesisAxiom{
		{AxiomID: "MATH-001", ClaimType: "empirical_axiom", Confidence: 0.90},
		{AxiomID: "MATH-002", ClaimType: "derived_claim", Confidence: 0.95, Dependencies: []string{"MATH-001"}},
	}
	err := ValidateDerivedConfidence(axioms)
	require.Error(t, err)
	require.Contains(t, err.Error(), "exceeding")
}

func TestResolveAxiomCategory(t *testing.T) {
	tests := []struct {
		name     string
		axiom    *GenesisAxiom
		expected string
		disagree bool
	}{
		{
			name:     "explicit analytic",
			axiom:    &GenesisAxiom{EpistemicCategory: "analytic", ClaimType: "axiom"},
			expected: "analytic",
		},
		{
			name:     "auto-derived from claim type",
			axiom:    &GenesisAxiom{ClaimType: "empirical_axiom"},
			expected: "empirical",
		},
		{
			name:     "override disagrees",
			axiom:    &GenesisAxiom{EpistemicCategory: "formal", ClaimType: "axiom"},
			expected: "formal",
			disagree: true,
		},
		{
			name:     "fallback analytic",
			axiom:    &GenesisAxiom{},
			expected: "analytic",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cat, disagrees := ResolveAxiomCategory(tc.axiom)
			require.Equal(t, tc.expected, cat)
			require.Equal(t, tc.disagree, disagrees)
		})
	}
}

func TestAxiomConfidenceBPS(t *testing.T) {
	// Explicit confidence
	a := &GenesisAxiom{Confidence: 0.97, ClaimType: "axiom"}
	require.Equal(t, uint64(970_000), axiomConfidenceBPS(a))

	// Zero confidence, use default for claim type
	b := &GenesisAxiom{Confidence: 0, ClaimType: "empirical_axiom"}
	require.Equal(t, uint64(970_000), axiomConfidenceBPS(b))

	// Zero confidence, no claim type — fallback 1.0
	c := &GenesisAxiom{}
	require.Equal(t, uint64(1_000_000), axiomConfidenceBPS(c))
}

func TestAxiomsToFacts(t *testing.T) {
	axioms := []*GenesisAxiom{
		{
			AxiomID:           "MATH-001",
			Statement:         "Every object is identical to itself.",
			ClaimType:         "axiom",
			Domain:            "mathematics",
			EpistemicCategory: "analytic",
			Confidence:        1.0,
			Dependencies:      nil,
		},
		{
			AxiomID:           "PHYS-001",
			Statement:         "Space is modeled as a 3D continuum.",
			ClaimType:         "definition",
			Domain:            "physics",
			EpistemicCategory: "analytic",
			Confidence:        1.0,
			Dependencies:      []string{"MATH-001"},
		},
	}

	facts := AxiomsToFacts(axioms)
	require.Len(t, facts, 2)

	// Check first fact (MATH axiom)
	f := facts[0]
	require.Equal(t, "MATH-001", f.Id)
	require.Equal(t, "Every object is identical to itself.", f.Content)
	require.Equal(t, "mathematics", f.Domain)
	require.Equal(t, "analytic", f.Category)
	require.Equal(t, uint64(1_000_000), f.Confidence)
	require.Equal(t, FactStatus_FACT_STATUS_VERIFIED, f.Status)
	require.Equal(t, "genesis", f.Submitter)
	require.Equal(t, uint64(1_000_000), f.Fundamentality)
	require.Equal(t, "fundamental", f.Stratum) // mathematics → fundamental
	require.Equal(t, "canonical", f.Maturity)
	require.Equal(t, uint64(0), f.SubmittedAtBlock)
	require.Equal(t, uint64(0), f.VerifiedAtBlock)

	// Check second fact has references from dependencies
	f2 := facts[1]
	require.Equal(t, []string{"MATH-001"}, f2.References)
	require.Equal(t, "physical", f2.Stratum) // physics → physical
}

func TestComputeDAGStats(t *testing.T) {
	axioms := []*GenesisAxiom{
		{AxiomID: "MATH-001", Dependencies: nil},
		{AxiomID: "MATH-002", Dependencies: nil},
		{AxiomID: "MATH-003", Dependencies: []string{"MATH-001"}},
		{AxiomID: "MATH-004", Dependencies: []string{"MATH-001", "MATH-002"}},
		{AxiomID: "MATH-005", Dependencies: []string{"MATH-003", "MATH-004"}},
	}

	stats, err := ComputeDAGStats(axioms)
	require.NoError(t, err)
	require.Equal(t, 2, stats.RootCount)        // MATH-001, MATH-002
	require.Equal(t, 2, stats.MaxDepth)          // MATH-005 at depth 2
	require.Equal(t, 5, stats.TotalEdges)        // 003→001, 004→001, 004→002, 005→003, 005→004
	require.Len(t, stats.Orphans, 0)             // no orphans
}

func TestComputeCrossDomainMatrix(t *testing.T) {
	axioms := []*GenesisAxiom{
		{AxiomID: "MATH-001", Domain: "mathematics"},
		{AxiomID: "PHYS-001", Domain: "physics", Dependencies: []string{"MATH-001"}},
		{AxiomID: "PHYS-002", Domain: "physics", Dependencies: []string{"MATH-001"}},
	}

	matrix := ComputeCrossDomainMatrix(axioms)
	require.Len(t, matrix.Entries, 1) // physics→mathematics
	require.Equal(t, "physics", matrix.Entries[0].FromDomain)
	require.Equal(t, "mathematics", matrix.Entries[0].ToDomain)
	require.Equal(t, 2, matrix.Entries[0].Count)
}

func TestEmbeddedAxiomsLoad(t *testing.T) {
	// Verify embedded JSON loads and parses
	require.NotEmpty(t, GenesisAxiomsJSON, "embedded axioms JSON must not be empty")

	axioms, err := ParseAxioms(GenesisAxiomsJSON)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(axioms), 777, "expected at least 777 axioms, got %d", len(axioms))

	// Check domain coverage
	domains := make(map[string]bool)
	for _, a := range axioms {
		domains[a.Domain] = true
	}
	expectedDomains := []string{
		"mathematics", "physics", "theology", "philosophy", "logic",
		"chemistry", "biology", "computer_science", "economics",
		"agent_rights", "agent_purpose",
	}
	for _, d := range expectedDomains {
		require.True(t, domains[d], "expected domain %q in axioms", d)
	}
}

func TestSeedAxiomFacts(t *testing.T) {
	facts, err := SeedAxiomFacts()
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(facts), 777)

	// All facts should be verified with canonical maturity
	for _, f := range facts {
		require.Equal(t, FactStatus_FACT_STATUS_VERIFIED, f.Status, "fact %s", f.Id)
		require.Equal(t, "canonical", f.Maturity, "fact %s", f.Id)
		require.Equal(t, "genesis", f.Submitter, "fact %s", f.Id)
		require.Greater(t, f.Confidence, uint64(0), "fact %s", f.Id)
		require.NotEmpty(t, f.Content, "fact %s", f.Id)
		require.NotEmpty(t, f.Domain, "fact %s", f.Id)
	}
}

func TestDomainPrefixMapConsistency(t *testing.T) {
	// Every domain in DomainPrefixMap should have a reverse mapping
	for domain, prefix := range DomainPrefixMap {
		reverseDomain, ok := PrefixToDomainMap[prefix]
		require.True(t, ok, "prefix %q has no reverse mapping", prefix)
		require.Equal(t, domain, reverseDomain, "reverse mapping mismatch for %q", prefix)
	}

	// Every domain in BootstrapDomainStrata should be in DomainPrefixMap
	for domain := range BootstrapDomainStrata {
		_, ok := DomainPrefixMap[domain]
		require.True(t, ok, "domain %q in BootstrapDomainStrata but not in DomainPrefixMap", domain)
	}
}
