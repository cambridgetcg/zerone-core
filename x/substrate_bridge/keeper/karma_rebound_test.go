package keeper_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

// ─── K-alpha review fix: settlement-time re-bound ───────────────────────────
//
// MaxCitedFactsPerLink is enforced at admission for NEW links, but K-alpha
// ships no migration, so an attestation admitted under a pre-cap binary can
// reach the BeginBlocker drain carrying arbitrary fan-out. The emitter must
// re-bound: recognition truncates deterministically at the first
// MaxCitedFactsPerLink cited facts instead of trusting admission history.

func TestSettleAttestation_LegacyOverCapLinkTruncatedAtSettlement(t *testing.T) {
	overCap := types.MaxCitedFactsPerLink + 4 // 20 — inadmissible today, possible as legacy state

	kk := &stubKarmaKnowledgeKeeper{facts: map[string]*knowledgetypes.Fact{}}
	for i := 0; i < overCap; i++ {
		id := fmt.Sprintf("fact-%d", i)
		kk.facts[id] = &knowledgetypes.Fact{Id: id, Submitter: testSubmitter(fmt.Sprintf("author-%d", i))}
	}
	k, ctx := setupKarmaExternalKeeper(t, kk)
	writeActiveAdapter(t, k, ctx, "wiki-v1")

	// Written directly into state — the legacy shape ValidateLink never saw.
	require.NoError(t, k.WriteAttestation(ctx, &types.ExternalAttestation{
		AttestationId: "att-legacy-overcap", AdapterId: "wiki-v1", Submitter: testSubmitter("attester"),
		BondUzrn: "1000000",
		Status:   types.AttestationStatus_ATTESTATION_STATUS_READY,
		Link: &types.SubstrateLink{
			CitedFacts: citedFacts(overCap),
		},
	}))

	require.NoError(t, k.SettleAttestation(ctx, "att-legacy-overcap"))
	settled, _ := k.GetAttestation(ctx, "att-legacy-overcap")
	require.Equal(t, types.AttestationStatus_ATTESTATION_STATUS_SETTLED, settled.Status,
		"the over-cap legacy link still settles — only recognition is bounded")

	edges := karmaEdgeEvents(ctx)
	require.Len(t, edges, types.MaxCitedFactsPerLink,
		"recognition fan-out re-bounds at the cap even for pre-cap state")

	// Deterministic truncation: exactly the first MaxCitedFactsPerLink
	// entries in link order earn edges.
	seen := map[string]bool{}
	for _, ev := range edges {
		seen[attrValue(t, ev, types.AttrKarmaRefID)] = true
		// The register confession rides the external edge too.
		require.Equal(t, types.KarmaRegisterPricedCoherence, attrValue(t, ev, types.AttrKarmaRegister))
	}
	for i := 0; i < types.MaxCitedFactsPerLink; i++ {
		require.True(t, seen[fmt.Sprintf("fact-%d", i)], "fact-%d (inside the cap) must be recognized", i)
	}
	for i := types.MaxCitedFactsPerLink; i < overCap; i++ {
		require.False(t, seen[fmt.Sprintf("fact-%d", i)], "fact-%d (beyond the cap) must be truncated", i)
	}
}
