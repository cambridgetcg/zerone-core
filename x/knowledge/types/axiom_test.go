package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDomainPrefixMapConsistency — the shared domain/stratum maps (kept
// after the axiom-tier removal) stay consistent. The axiom-machinery tests
// that lived in this file (ValidateAxiomID, ParseAxioms, AxiomsToFacts,
// ValidateAxiomDAG, SeedAxiomFacts, the embedded-axioms load test, etc.)
// were removed with the axiom tier: truth is, not proven; axioms are
// assumptions, and the chain no longer seeds a bedrock of assumed-true
// facts.
func TestDomainPrefixMapConsistency(t *testing.T) {
	// Every domain in DomainPrefixMap should have a reverse mapping.
	for domain, prefix := range DomainPrefixMap {
		reverseDomain, ok := PrefixToDomainMap[prefix]
		require.True(t, ok, "prefix %q has no reverse mapping", prefix)
		require.Equal(t, domain, reverseDomain, "reverse mapping mismatch for %q", prefix)
	}

	// Every domain in BootstrapDomainStrata should be in DomainPrefixMap.
	for domain := range BootstrapDomainStrata {
		_, ok := DomainPrefixMap[domain]
		require.True(t, ok, "domain %q in BootstrapDomainStrata but not in DomainPrefixMap", domain)
	}
}