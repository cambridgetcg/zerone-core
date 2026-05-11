package cross_stack_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	substratebridgekeeper "github.com/zerone-chain/zerone/x/substrate_bridge/keeper"
	substratebridgetypes "github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

// TestSubstrateBridge_HappyPathSettlement drives the full lifecycle:
//  1. Register adapter via direct keeper call (simulating gov-passed LIP).
//  2. Seed cited_fact in x/knowledge.
//  3. Submit external attestation with 2 pending claims via msg server.
//  4. Resolve pending claims as VERIFIED (calls into substrate_bridge.OnClaimResolved).
//  5. BeginBlocker drains READY → SETTLED.
//  6. Verify attestation has SETTLED status and non-zero reward.
func TestSubstrateBridge_HappyPathSettlement(t *testing.T) {
	h := NewTestHarness(t)

	// 1. Register adapter.
	require.NoError(t, h.SubstrateBridgeKeeper.WriteAdapter(h.Ctx, &substratebridgetypes.AdapterRegistration{
		AdapterId:              "test-wiki",
		Status:                 substratebridgetypes.AdapterStatus_ADAPTER_STATUS_ACTIVE,
		MinAttestationBondUzrn: "222000",
		MinPerClaimBondUzrn:    "222",
		AxisBounds:             &substratebridgetypes.AxisBounds{AxisSubstrateMax: 1_000_000},
	}))

	// 2. Seed cited fact.
	require.NoError(t, h.KnowledgeKeeper.SetDomain(h.Ctx, &knowledgetypes.Domain{
		Name:   "test-domain",
		Status: knowledgetypes.DomainStatus_DOMAIN_STATUS_ACTIVE,
	}))
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
		Id:              "seed-fact",
		Domain:          "test-domain",
		Status:          knowledgetypes.FactStatus_FACT_STATUS_VERIFIED,
		VerifiedAtBlock: 1,
	}))

	// 3. Submit attestation.
	link := &substratebridgetypes.SubstrateLink{
		AdapterId: "test-wiki",
		CitedFacts: []*substratebridgetypes.FactCitation{
			{FactId: "seed-fact", CitationType: substratebridgetypes.CitationType_CITATION_TYPE_SUPPORTS},
		},
		PendingClaims: []*substratebridgetypes.PendingClaim{
			{ClaimContent: "claim A", Domain: "test-domain"},
			{ClaimContent: "claim B", Domain: "test-domain"},
		},
		RecursionWeight: &substratebridgetypes.AxisProjection{AxisSubstrate: 100_000},
	}
	link.LinkHash = substratebridgekeeper.ComputeLinkHash(link)

	submitter := testAddr("sb_happy_submitter")
	// Fund submitter for bond.
	require.NoError(t, h.FundAccount(submitter, sdk.NewCoins(sdk.NewInt64Coin("uzrn", 10_000_000))))

	srv := substratebridgekeeper.NewMsgServerImpl(h.SubstrateBridgeKeeper)
	resp, err := srv.SubmitExternalAttestation(h.Ctx, &substratebridgetypes.MsgSubmitExternalAttestation{
		Submitter:   submitter.String(),
		AdapterId:   "test-wiki",
		WorkClassId: "translation",
		Link:        link,
		BondUzrn:    "1000000",
	})
	require.NoError(t, err)
	attID := resp.AttestationId

	att, found := h.SubstrateBridgeKeeper.GetAttestation(h.Ctx, attID)
	require.True(t, found)
	require.Equal(t, substratebridgetypes.AttestationStatus_ATTESTATION_STATUS_AWAITING_RESOLUTION, att.Status)

	// 4. Resolve pending claims as VERIFIED via OnClaimResolved.
	pendingClaims := h.SubstrateBridgeKeeper.PendingClaimsFor(h.Ctx, attID)
	require.NotEmpty(t, pendingClaims, "expected pending claims to be indexed")
	for _, claimID := range pendingClaims {
		require.NoError(t, h.SubstrateBridgeKeeper.OnClaimResolved(h.Ctx, claimID, true))
	}

	att, found = h.SubstrateBridgeKeeper.GetAttestation(h.Ctx, attID)
	require.True(t, found)
	require.Equal(t, substratebridgetypes.AttestationStatus_ATTESTATION_STATUS_READY, att.Status)

	// 5. BeginBlocker drains READY → SETTLED.
	require.NoError(t, h.SubstrateBridgeKeeper.BeginBlocker(h.Ctx))

	att, found = h.SubstrateBridgeKeeper.GetAttestation(h.Ctx, attID)
	require.True(t, found)
	require.Equal(t, substratebridgetypes.AttestationStatus_ATTESTATION_STATUS_SETTLED, att.Status)
	require.NotEqual(t, "0", att.RewardUzrn)
	require.NotEmpty(t, att.RewardUzrn)
}

// TestSubstrateBridge_RejectionThreshold drives the fraud path: most
// pending claims are REJECTED → attestation transitions to REJECTED →
// bond slashed.
func TestSubstrateBridge_RejectionThreshold(t *testing.T) {
	h := NewTestHarness(t)
	require.NoError(t, h.SubstrateBridgeKeeper.WriteAdapter(h.Ctx, &substratebridgetypes.AdapterRegistration{
		AdapterId: "fraud-adapter",
		Status:    substratebridgetypes.AdapterStatus_ADAPTER_STATUS_ACTIVE,
	}))

	// Create attestation with 4 pending claims.
	att := &substratebridgetypes.ExternalAttestation{
		AttestationId: "fraud-att",
		AdapterId:     "fraud-adapter",
		Status:        substratebridgetypes.AttestationStatus_ATTESTATION_STATUS_AWAITING_RESOLUTION,
		Link: &substratebridgetypes.SubstrateLink{
			PendingClaims: []*substratebridgetypes.PendingClaim{{}, {}, {}, {}},
		},
		BondUzrn: "1000000",
	}
	require.NoError(t, h.SubstrateBridgeKeeper.WriteAttestation(h.Ctx, att))

	// Pre-set: 3 rejections, 1 verification → rejection ratio 75% > 50% threshold.
	att.RejectedCount = 3
	att.VerifiedCount = 1
	require.NoError(t, h.SubstrateBridgeKeeper.WriteAttestation(h.Ctx, att))

	require.NoError(t, h.SubstrateBridgeKeeper.SettleAttestation(h.Ctx, "fraud-att"))

	final, found := h.SubstrateBridgeKeeper.GetAttestation(h.Ctx, "fraud-att")
	require.True(t, found)
	require.Equal(t, substratebridgetypes.AttestationStatus_ATTESTATION_STATUS_REJECTED, final.Status)
	require.NotEmpty(t, final.RejectionReason)
	// Bond should be slashed (slash_uzrn should match bond_uzrn).
	require.Equal(t, att.BondUzrn, final.SlashUzrn)
}

// TestSubstrateBridge_LineagePropagatesAcrossClasses drives a cross-class
// lineage chain: attestation A (class translation) settled → attestation B
// (class curriculum) cites A → settle B → A's accumulator non-zero.
func TestSubstrateBridge_LineagePropagatesAcrossClasses(t *testing.T) {
	h := NewTestHarness(t)

	// Write two attestations in different work classes. A is already SETTLED
	// (upstream); B is READY (downstream, cites A).
	require.NoError(t, h.SubstrateBridgeKeeper.WriteAttestation(h.Ctx, &substratebridgetypes.ExternalAttestation{
		AttestationId:    "att-A",
		WorkClassId:      "translation",
		Submitter:        testAddr("sb_lineage_alice").String(),
		SubmittedAtBlock: 10,
		Status:           substratebridgetypes.AttestationStatus_ATTESTATION_STATUS_SETTLED,
	}))
	// att-B has 2 pending claims already resolved as verified so the reward is
	// non-zero and lineage propagation fires.
	require.NoError(t, h.SubstrateBridgeKeeper.WriteAttestation(h.Ctx, &substratebridgetypes.ExternalAttestation{
		AttestationId:    "att-B",
		WorkClassId:      "curriculum",
		Submitter:        testAddr("sb_lineage_bob__").String(),
		SubmittedAtBlock: 20,
		Status:           substratebridgetypes.AttestationStatus_ATTESTATION_STATUS_READY,
		VerifiedCount:    2,
		Link: &substratebridgetypes.SubstrateLink{
			RecursionWeight: &substratebridgetypes.AxisProjection{AxisSubstrate: 50_000},
			PendingClaims: []*substratebridgetypes.PendingClaim{
				{ClaimContent: "curriculum claim 1", Domain: "curriculum"},
				{ClaimContent: "curriculum claim 2", Domain: "curriculum"},
			},
		},
	}))

	// Create lineage edge: B cites A (B downstream, A upstream).
	require.NoError(t, h.SubstrateBridgeKeeper.CreateLineageEdge(h.Ctx, &substratebridgetypes.LineageEdge{
		UpstreamAttestationId:   "att-A",
		DownstreamAttestationId: "att-B",
		CitationType:            substratebridgetypes.CitationType_CITATION_TYPE_SUPPORTS,
		ContributionShareBps:    10000,
	}))

	// Settle B — triggers lineage propagation back to A.
	require.NoError(t, h.SubstrateBridgeKeeper.SettleAttestation(h.Ctx, "att-B"))

	// Verify B is now SETTLED (or PARTIAL if any rejected, but 0 rejections → SETTLED).
	attB, found := h.SubstrateBridgeKeeper.GetAttestation(h.Ctx, "att-B")
	require.True(t, found)
	require.Equal(t, substratebridgetypes.AttestationStatus_ATTESTATION_STATUS_SETTLED, attB.Status)

	// A's accumulator should be non-zero (received lineage royalty from B's settlement).
	acc, found := h.SubstrateBridgeKeeper.GetLineageAccumulator(h.Ctx, "att-A")
	require.True(t, found, "upstream accumulator should exist after lineage propagation")
	require.NotEqual(t, "0", acc.CumulativeUzrn)
}
