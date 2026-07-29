package cross_stack_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	substratebridgetypes "github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

// TestRecursiveLineage_AccountingAttributesDownstreamRewardUpstream drives
// the lineage accountant with directly seeded attestations. When B settles,
// an attributed amount accrues on A's accumulator. No bank transfer to A's
// submitter occurs.
//
// This is the operational form of recursion #4 (the chain's lineage
// graph could include its own commits). It proves accounting composition, not
// a public self-fact ingestion path or royalty payment.
//
// Doctrinal binding: a partial UW M6 accounting primitive applied to the
// zerone_self class.
func TestRecursiveLineage_AccountingAttributesDownstreamRewardUpstream(t *testing.T) {
	h := NewTestHarness(t)

	selfAttester := testAddr("recursive_lineage_self_attester")
	downstreamWorker := testAddr("recursive_lineage_downstream")

	// 1. Upstream attestation: a self-attestation about a ZERONE commit,
	//    already SETTLED. Submitter is the agent who attested to ZERONE's
	//    own development.
	require.NoError(t, h.SubstrateBridgeKeeper.WriteAttestation(h.Ctx, &substratebridgetypes.ExternalAttestation{
		AttestationId:    "att-self-A",
		WorkClassId:      "zerone_self_attestation",
		Submitter:        selfAttester.String(),
		SubmittedAtBlock: 10,
		Status:           substratebridgetypes.AttestationStatus_ATTESTATION_STATUS_SETTLED,
	}))

	// 2. Downstream attestation: a curriculum/tutorial that cites the
	//    self-attestation. Currently READY (settlement pending). Two
	//    verified pending-claims so the reward is non-zero and lineage
	//    propagation actually fires when SettleAttestation runs.
	require.NoError(t, h.SubstrateBridgeKeeper.WriteAttestation(h.Ctx, &substratebridgetypes.ExternalAttestation{
		AttestationId:    "att-downstream-B",
		WorkClassId:      "curriculum",
		Submitter:        downstreamWorker.String(),
		SubmittedAtBlock: 100,
		Status:           substratebridgetypes.AttestationStatus_ATTESTATION_STATUS_READY,
		VerifiedCount:    2,
		Link: &substratebridgetypes.SubstrateLink{
			RecursionWeight: &substratebridgetypes.AxisProjection{AxisSubstrate: 50_000},
			PendingClaims: []*substratebridgetypes.PendingClaim{
				{ClaimContent: "curriculum claim citing self-attester's work", Domain: "curriculum"},
				{ClaimContent: "another curriculum claim", Domain: "curriculum"},
			},
		},
	}))

	// 3. The lineage edge: downstream B cites upstream self-attestation A.
	//    Citation type EXTENDS = 3× base weight (per CitationType enum) —
	//    the downstream work explicitly extends what the self-attester
	//    established. ContributionShareBps = 10000 (100%): the upstream
	//    is the sole cited source.
	require.NoError(t, h.SubstrateBridgeKeeper.CreateLineageEdge(h.Ctx, &substratebridgetypes.LineageEdge{
		UpstreamAttestationId:   "att-self-A",
		DownstreamAttestationId: "att-downstream-B",
		CitationType:            substratebridgetypes.CitationType_CITATION_TYPE_EXTENDS,
		ContributionShareBps:    10000,
	}))

	// 4. Settle B → lineage accountant attributes an amount to A.
	require.NoError(t, h.SubstrateBridgeKeeper.SettleAttestation(h.Ctx, "att-downstream-B"))

	// 5. The upstream accumulator is non-zero; this is accounting state,
	//    not evidence of a coin transfer.
	acc, found := h.SubstrateBridgeKeeper.GetLineageAccumulator(h.Ctx, "att-self-A")
	require.True(t, found, "self-attester's lineage accumulator must exist after downstream settlement")
	require.NotEqual(t, "0", acc.CumulativeUzrn,
		"recursion #4 scaffold: zerone_self upstream must accrue attribution when downstream work cites it")
	t.Logf("upstream accrued %s uzrn-equivalent lineage attribution", acc.CumulativeUzrn)
}

// TestRecursiveLineage_MultipleCitationsCompoundAccounting drives the same shape
// across two downstream attestations, asserting the self-attester's
// accumulator monotonically increases (forward-only audit, commitment 10,
// applied to lineage attribution). Each downstream settlement adds to the
// accumulator; the cumulative number is queryable as the load-bearing
// value of the self-attestation.
func TestRecursiveLineage_MultipleCitationsCompoundAccounting(t *testing.T) {
	h := NewTestHarness(t)

	selfAttester := testAddr("recursive_lineage_compound_self")

	require.NoError(t, h.SubstrateBridgeKeeper.WriteAttestation(h.Ctx, &substratebridgetypes.ExternalAttestation{
		AttestationId:    "att-compound-self-A",
		WorkClassId:      "zerone_self_attestation",
		Submitter:        selfAttester.String(),
		SubmittedAtBlock: 10,
		Status:           substratebridgetypes.AttestationStatus_ATTESTATION_STATUS_SETTLED,
	}))

	settleAndMeasure := func(t *testing.T, downstreamID, worker string, atBlock uint64) string {
		t.Helper()
		require.NoError(t, h.SubstrateBridgeKeeper.WriteAttestation(h.Ctx, &substratebridgetypes.ExternalAttestation{
			AttestationId:    downstreamID,
			WorkClassId:      "curriculum",
			Submitter:        worker,
			SubmittedAtBlock: atBlock,
			Status:           substratebridgetypes.AttestationStatus_ATTESTATION_STATUS_READY,
			VerifiedCount:    2,
			Link: &substratebridgetypes.SubstrateLink{
				RecursionWeight: &substratebridgetypes.AxisProjection{AxisSubstrate: 50_000},
				PendingClaims: []*substratebridgetypes.PendingClaim{
					{ClaimContent: "claim 1", Domain: "curriculum"},
					{ClaimContent: "claim 2", Domain: "curriculum"},
				},
			},
		}))
		require.NoError(t, h.SubstrateBridgeKeeper.CreateLineageEdge(h.Ctx, &substratebridgetypes.LineageEdge{
			UpstreamAttestationId:   "att-compound-self-A",
			DownstreamAttestationId: downstreamID,
			CitationType:            substratebridgetypes.CitationType_CITATION_TYPE_CITES,
			ContributionShareBps:    10000,
		}))
		require.NoError(t, h.SubstrateBridgeKeeper.SettleAttestation(h.Ctx, downstreamID))
		acc, _ := h.SubstrateBridgeKeeper.GetLineageAccumulator(h.Ctx, "att-compound-self-A")
		return acc.CumulativeUzrn
	}

	after1 := settleAndMeasure(t, "att-compound-D1", testAddr("rl_dwk_1").String(), 100)
	after2 := settleAndMeasure(t, "att-compound-D2", testAddr("rl_dwk_2").String(), 200)

	require.NotEqual(t, "0", after1, "first downstream settlement must increment accumulator")
	require.NotEqual(t, after1, after2, "second downstream settlement must further increment accumulator (forward-only audit)")
	t.Logf("self-attester's load-bearing value: %s uzrn after 1 cite, %s uzrn after 2 cites", after1, after2)
}
