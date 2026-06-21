package cross_stack_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// TC4: the graph carries its disprovals.
//
// Verified by: cascade-replay returns the DISPROVEN root, the cascaded
// descendants, and the cascade events that fired when the disproof landed.
// The bundle's snapshot root commits to all of (nodes, edges, cascade_events,
// vindications, transitions) under V2 semantics; re-derivable from the IDs
// + cascade-event canon alone.
func TestToKSubstrate_TC4_GraphCarriesDisprovals(t *testing.T) {
	h := NewTestHarness(t)
	domain := "tc4_test_domain"
	require.NoError(t, h.KnowledgeKeeper.SetDomain(h.Ctx, &knowledgetypes.Domain{
		Name: domain, Status: knowledgetypes.DomainStatus_DOMAIN_STATUS_ACTIVE,
	}))

	// Build: axiom (will be disproven) + descendant.
	axiom := &knowledgetypes.Fact{
		Id: "tc4-axiom", Domain: domain,
		Status:        knowledgetypes.FactStatus_FACT_STATUS_VERIFIED,
		VerifiedAtBlock: 100, Confidence: 900_000,
	}
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, axiom))

	descendant := submitAndAcceptChainedClaim(t, h, domain, "depends on tc4-axiom",
		[]*knowledgetypes.ClaimRelation{{
			TargetFactId:          axiom.Id,
			Relation:               knowledgetypes.RelationType_RELATION_TYPE_REQUIRES,
			Inference:              knowledgetypes.InferenceType_INFERENCE_TYPE_DEDUCTIVE,
			InferenceStrengthBps:   1_000_000,
		}}, "tc4-descendant")

	// Disprove the axiom via challenge.
	challengeClaim := &knowledgetypes.Claim{
		Id: "tc4-challenge", Submitter: "challenger",
		FactContent: "axiom is wrong",
		Domain:      domain,
		Category:    "empirical",
		Status:      knowledgetypes.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION,
		Stake:       "11000000",
		ProvisionalFactId: axiom.Id,
		Relations: []*knowledgetypes.ClaimRelation{{
			TargetFactId: axiom.Id,
			Relation:     knowledgetypes.RelationType_RELATION_TYPE_CONTRADICTS,
		}},
	}
	require.NoError(t, h.KnowledgeKeeper.SetClaim(h.Ctx, challengeClaim))
	round := &knowledgetypes.VerificationRound{
		Id: "tc4-round", ClaimId: challengeClaim.Id,
		Phase: knowledgetypes.VerificationPhase_VERIFICATION_PHASE_COMPLETE,
	}
	require.NoError(t, h.KnowledgeKeeper.CompleteRound(h.Ctx, round, &knowledgekeeper.VerificationResult{
		Verdict:    knowledgetypes.Verdict_VERDICT_ACCEPT,
		Confidence: 900_000, AcceptCount: 3,
	}))

	// ─── Cascade-replay must surface the disproval-graph. ────────────────
	q := knowledgekeeper.NewQueryServerImpl(h.KnowledgeKeeper)
	resp, err := q.BundleToK(h.Ctx, &knowledgetypes.QueryBundleToKRequest{
		Selector: &knowledgetypes.ToKSelector{Variant: &knowledgetypes.ToKSelector_CascadeReplay{
			CascadeReplay: &knowledgetypes.CascadeReplaySelector{
				DisprovenFactId:      axiom.Id,
				MaxDepth:             1,
				IncludeStatusHistory: true,
			},
		}},
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Bundle, "TC4: cascade-replay must return a bundle")
	require.Equal(t, "v2", resp.Bundle.Provenance.TokRootVersion)

	// Bundle includes the disproven axiom and the cascaded descendant.
	require.Contains(t, resp.Bundle.IncludedNodeIds, axiom.Id)
	require.Contains(t, resp.Bundle.IncludedNodeIds, descendant.Id)

	// Cascade events surface the disproof's cascade.
	require.NotEmpty(t, resp.Bundle.CascadeEvents, "TC4: cascade events must be in bundle")
	require.Equal(t, descendant.Id, resp.Bundle.CascadeEvents[0].DescendantFactId)

	// Status history surfaces the axiom's transition VERIFIED → DISPROVEN.
	var sawAxiomDisprovenTransition bool
	for _, tr := range resp.Bundle.StatusHistory {
		if tr.FactId == axiom.Id && tr.NewStatus == knowledgetypes.FactStatus_FACT_STATUS_DISPROVEN {
			sawAxiomDisprovenTransition = true
		}
	}
	require.True(t, sawAxiomDisprovenTransition,
		"TC4: status history must include the VERIFIED → DISPROVEN transition")

	// Re-derivability: V2 root from IDs + canon.
	rederived := knowledgekeeper.ComputeToKSnapshotRootV2(
		resp.Bundle.IncludedNodeIds, resp.Bundle.IncludedEdges,
		resp.Bundle.CascadeEvents, resp.Bundle.Vindications, resp.Bundle.StatusHistory,
	)
	require.Equal(t, resp.Bundle.SnapshotRoot, rederived,
		"TC4: V2 root must be re-derivable from IDs + cascade canon")
}

// TC4 (cont'd): DISPROVEN nodes are not pruned from non-cascade selectors.
//
// Verified by: a RootedSubtree from an axiom that is itself DISPROVEN still
// includes the axiom (it is the root). Filter is by relation type, not by
// status. The doctrine: "Disproven facts remain in the graph with their
// full disproval rationale; they are not pruned."
func TestToKSubstrate_TC4_NoPruneDisprovenFromNonCascadeSelectors(t *testing.T) {
	h := NewTestHarness(t)
	domain := "tc4_noprune_domain"
	require.NoError(t, h.KnowledgeKeeper.SetDomain(h.Ctx, &knowledgetypes.Domain{
		Name: domain, Status: knowledgetypes.DomainStatus_DOMAIN_STATUS_ACTIVE,
	}))

	// A DISPROVEN root that still has descendants.
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
		Id: "noprune-disproven", Domain: domain,
		Status:          knowledgetypes.FactStatus_FACT_STATUS_DISPROVEN,
		VerifiedAtBlock: 100,
	}))
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
		Id: "noprune-leaf", Domain: domain,
		Status:          knowledgetypes.FactStatus_FACT_STATUS_CONTESTED,
		VerifiedAtBlock: 100,
	}))
	require.NoError(t, h.KnowledgeKeeper.SetFactRelation(h.Ctx, &knowledgetypes.FactRelation{
		SourceFactId: "noprune-leaf",
		TargetFactId: "noprune-disproven",
		Relation:     knowledgetypes.RelationType_RELATION_TYPE_SUPPORTS,
	}))

	q := knowledgekeeper.NewQueryServerImpl(h.KnowledgeKeeper)

	// RootedSubtree from the DISPROVEN root — must include the root itself
	// and its descendant (the descendant supports the disproven root).
	resp, err := q.BundleToK(h.Ctx, &knowledgetypes.QueryBundleToKRequest{
		Selector: &knowledgetypes.ToKSelector{Variant: &knowledgetypes.ToKSelector_RootedSubtree{
			RootedSubtree: &knowledgetypes.RootedSubtreeSelector{
				RootFactId: "noprune-disproven", MaxDepth: 5,
			},
		}},
	})
	require.NoError(t, err)
	require.Contains(t, resp.Bundle.IncludedNodeIds, "noprune-disproven",
		"TC4: DISPROVEN root must NOT be pruned from non-cascade selectors")
	require.Contains(t, resp.Bundle.IncludedNodeIds, "noprune-leaf",
		"TC4: descendant of DISPROVEN root must NOT be pruned")
}