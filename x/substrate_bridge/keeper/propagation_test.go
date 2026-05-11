package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"

	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

func TestPropagateLineage_DirectCitePaysProportional(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeperWithBank(t) // harness includes stubBankKeeper
	setupTwoAttestations(t, k, ctx)
	require.NoError(t, k.CreateLineageEdge(ctx, &types.LineageEdge{
		UpstreamAttestationId:   "att-upstream",
		DownstreamAttestationId: "att-downstream",
		CitationType:            types.CitationType_CITATION_TYPE_SUPPORTS,
		ContributionShareBps:    10000,
	}))

	// Downstream settles with reward 100 ZRN = 100_000_000 uzrn.
	downstreamReward := sdkmath.NewInt(100_000_000)
	err := k.PropagateLineage(ctx, "att-downstream", downstreamReward)
	require.NoError(t, err)

	// 30% lineage budget = 30M uzrn.
	// Direct upstream (depth 1, SUPPORTS=2× weight, share 10000bps):
	//   share = 30M × 10000bps × 2× / 10000 = 60M uzrn
	//   but clamp to budget remaining: 30M
	acc, found := k.GetLineageAccumulator(ctx, "att-upstream")
	require.True(t, found)
	require.Equal(t, "30000000", acc.CumulativeUzrn)
}

func TestPropagateLineage_MultiHopWithDecay(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeperWithBank(t)
	// Three attestations: F → T → D (chain).
	require.NoError(t, k.WriteAttestation(ctx, &types.ExternalAttestation{
		AttestationId: "F", Submitter: "alice", SubmittedAtBlock: 10,
		Status: types.AttestationStatus_ATTESTATION_STATUS_SETTLED,
	}))
	require.NoError(t, k.WriteAttestation(ctx, &types.ExternalAttestation{
		AttestationId: "T", Submitter: "bob", SubmittedAtBlock: 20,
		Status: types.AttestationStatus_ATTESTATION_STATUS_SETTLED,
	}))
	require.NoError(t, k.WriteAttestation(ctx, &types.ExternalAttestation{
		AttestationId: "D", Submitter: "carol", SubmittedAtBlock: 30,
		Status: types.AttestationStatus_ATTESTATION_STATUS_SETTLED,
	}))
	// T cites F; D cites T.
	require.NoError(t, k.CreateLineageEdge(ctx, &types.LineageEdge{
		UpstreamAttestationId: "F", DownstreamAttestationId: "T",
		CitationType: types.CitationType_CITATION_TYPE_CITES, ContributionShareBps: 10000,
	}))
	require.NoError(t, k.CreateLineageEdge(ctx, &types.LineageEdge{
		UpstreamAttestationId: "T", DownstreamAttestationId: "D",
		CitationType: types.CitationType_CITATION_TYPE_CITES, ContributionShareBps: 10000,
	}))

	downstreamReward := sdkmath.NewInt(100_000_000)
	require.NoError(t, k.PropagateLineage(ctx, "D", downstreamReward))

	// T receives: 30M × 1× × 10000bps = 30M
	accT, _ := k.GetLineageAccumulator(ctx, "T")
	require.Equal(t, "30000000", accT.CumulativeUzrn)
	// F receives propagated: T's share × 30% decay = 9M
	accF, _ := k.GetLineageAccumulator(ctx, "F")
	require.Equal(t, "9000000", accF.CumulativeUzrn)
}

func TestPropagateLineage_HaltsAtMaxDepth(t *testing.T) {
	// Chain of 8 attestations, max depth 5. Confirm propagation stops.
	// (Test body builds the chain, propagates from leaf, asserts only
	// the closest 5 ancestors received payments; deeper got 0.)
	t.Skip("Will be implemented as part of executor's discretion; the assertion is: " +
		"k.GetLineageAccumulator(ctx, deepest_ancestor) returns either not-found or 0")
}
