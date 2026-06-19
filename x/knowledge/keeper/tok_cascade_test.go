package keeper_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func TestRecordStatusTransition_AppendsForwardOnly(t *testing.T) {
	k, ctx, _, _ := setupKnowledgeTestFull(t)

	require.NoError(t, k.RecordStatusTransition(ctx, &types.StatusTransition{
		FactId:         "fact-a",
		PriorStatus:    types.FactStatus_FACT_STATUS_ACTIVE,
		NewStatus:      types.FactStatus_FACT_STATUS_VERIFIED,
		BlockHeight:    100,
		CauseEventType: "verification",
		CauseId:        "round-1",
	}))

	require.NoError(t, k.RecordStatusTransition(ctx, &types.StatusTransition{
		FactId:         "fact-a",
		PriorStatus:    types.FactStatus_FACT_STATUS_VERIFIED,
		NewStatus:      types.FactStatus_FACT_STATUS_DISPROVEN,
		BlockHeight:    200,
		CauseEventType: "challenge_disproven",
		CauseId:        "challenge-7",
	}))

	history := k.GetStatusHistory(ctx, "fact-a")
	require.Len(t, history, 2)
	require.Equal(t, uint64(1), history[0].Seq, "first transition seq=1")
	require.Equal(t, uint64(2), history[1].Seq, "second transition seq=2")
	require.Equal(t, types.FactStatus_FACT_STATUS_VERIFIED, history[0].NewStatus)
	require.Equal(t, types.FactStatus_FACT_STATUS_DISPROVEN, history[1].NewStatus)
}

func TestGetStatusHistory_EmptyForUntouchedFact(t *testing.T) {
	k, ctx, _, _ := setupKnowledgeTestFull(t)
	history := k.GetStatusHistory(ctx, "never-existed")
	require.Empty(t, history)
}

func TestRecordStatusTransition_SkipsNoOpTransition(t *testing.T) {
	k, ctx, _, _ := setupKnowledgeTestFull(t)
	// Same prior and new status → no-op.
	require.NoError(t, k.RecordStatusTransition(ctx, &types.StatusTransition{
		FactId:      "fact-noop",
		PriorStatus: types.FactStatus_FACT_STATUS_VERIFIED,
		NewStatus:   types.FactStatus_FACT_STATUS_VERIFIED,
	}))
	history := k.GetStatusHistory(ctx, "fact-noop")
	require.Empty(t, history, "no transition written when status unchanged")
}

func TestSetFact_RecordsStatusTransition(t *testing.T) {
	k, ctx, _, _ := setupKnowledgeTestFull(t)

	// Initial set: UNSPECIFIED → ACTIVE (first write records initial transition).
	require.NoError(t, k.SetFact(ctx, &types.Fact{
		Id: "fact-x", Domain: "physics",
		Status: types.FactStatus_FACT_STATUS_ACTIVE,
	}))
	// Second set: ACTIVE → VERIFIED.
	require.NoError(t, k.SetFact(ctx, &types.Fact{
		Id: "fact-x", Domain: "physics",
		Status: types.FactStatus_FACT_STATUS_VERIFIED,
	}))

	history := k.GetStatusHistory(ctx, "fact-x")
	require.Len(t, history, 2, "two transitions: UNSPECIFIED→ACTIVE, ACTIVE→VERIFIED")
	require.Equal(t, types.FactStatus_FACT_STATUS_UNSPECIFIED, history[0].PriorStatus)
	require.Equal(t, types.FactStatus_FACT_STATUS_ACTIVE, history[0].NewStatus)
	require.Equal(t, types.FactStatus_FACT_STATUS_ACTIVE, history[1].PriorStatus)
	require.Equal(t, types.FactStatus_FACT_STATUS_VERIFIED, history[1].NewStatus)
}

func TestSetFact_NoTransitionOnUnchangedStatus(t *testing.T) {
	k, ctx, _, _ := setupKnowledgeTestFull(t)

	require.NoError(t, k.SetFact(ctx, &types.Fact{
		Id: "fact-y", Domain: "math",
		Status: types.FactStatus_FACT_STATUS_VERIFIED,
	}))
	// First write records UNSPECIFIED → VERIFIED.
	history1 := k.GetStatusHistory(ctx, "fact-y")
	require.Len(t, history1, 1, "first write records initial transition")
	// Re-write same status with different unrelated field.
	require.NoError(t, k.SetFact(ctx, &types.Fact{
		Id: "fact-y", Domain: "math",
		Status:     types.FactStatus_FACT_STATUS_VERIFIED,
		Confidence: 750_000,
	}))

	history := k.GetStatusHistory(ctx, "fact-y")
	require.Len(t, history, 1, "no new transition written when status unchanged")
}

func TestSetFact_FirstWriteRecordsFromUnspecified(t *testing.T) {
	k, ctx, _, _ := setupKnowledgeTestFull(t)

	require.NoError(t, k.SetFact(ctx, &types.Fact{
		Id: "fact-genesis", Domain: "math",
		Status: types.FactStatus_FACT_STATUS_VERIFIED,
	}))
	history := k.GetStatusHistory(ctx, "fact-genesis")
	require.Len(t, history, 1, "first write records initial transition")
	require.Equal(t, types.FactStatus_FACT_STATUS_UNSPECIFIED, history[0].PriorStatus)
	require.Equal(t, types.FactStatus_FACT_STATUS_VERIFIED, history[0].NewStatus)
}

func TestRecordCascadeEvent_AppendsForwardOnly(t *testing.T) {
	k, ctx, _, _ := setupKnowledgeTestFull(t)

	require.NoError(t, k.RecordCascadeEvent(ctx, &types.CascadeEvent{
		DisprovenFactId:  "axiom-d",
		DescendantFactId: "child-1",
		ChallengeClaimId: "challenge-7",
		EdgeRelation:     "SUPPORTS",
		PriorStatus:      types.FactStatus_FACT_STATUS_VERIFIED,
		NewStatus:        types.FactStatus_FACT_STATUS_CONTESTED,
		BlockHeight:      200,
	}))
	require.NoError(t, k.RecordCascadeEvent(ctx, &types.CascadeEvent{
		DisprovenFactId:  "axiom-d",
		DescendantFactId: "child-2",
		ChallengeClaimId: "challenge-7",
		EdgeRelation:     "REQUIRES",
		PriorStatus:      types.FactStatus_FACT_STATUS_VERIFIED,
		NewStatus:        types.FactStatus_FACT_STATUS_CONTESTED,
		BlockHeight:      200,
	}))

	events := k.GetCascadeEventsForDisproof(ctx, "axiom-d")
	require.Len(t, events, 2)
	require.Equal(t, uint64(1), events[0].Seq)
	require.Equal(t, uint64(2), events[1].Seq)
	require.Equal(t, "child-1", events[0].DescendantFactId)
	require.Equal(t, "child-2", events[1].DescendantFactId)
}

func TestGetCascadeEventsForDisproof_EmptyForUntouchedFact(t *testing.T) {
	k, ctx, _, _ := setupKnowledgeTestFull(t)
	events := k.GetCascadeEventsForDisproof(ctx, "never-disproven")
	require.Empty(t, events)
}

func TestCascadeEvent_ReverseIndexFindable(t *testing.T) {
	k, ctx, _, _ := setupKnowledgeTestFull(t)

	require.NoError(t, k.RecordCascadeEvent(ctx, &types.CascadeEvent{
		DisprovenFactId:  "axiom-a",
		DescendantFactId: "leaf-x",
		EdgeRelation:     "SUPPORTS",
	}))
	require.NoError(t, k.RecordCascadeEvent(ctx, &types.CascadeEvent{
		DisprovenFactId:  "axiom-b",
		DescendantFactId: "leaf-x",
		EdgeRelation:     "CITES",
	}))

	// leaf-x was hit by two disproofs; reverse index lets us find both.
	disproofs := k.GetDisproofsAffectingDescendant(ctx, "leaf-x")
	require.Len(t, disproofs, 2)
	require.Contains(t, disproofs, "axiom-a")
	require.Contains(t, disproofs, "axiom-b")
}

// ─── Task 8: CascadeReplay selector validation ──────────────────────────────

func TestValidateToKSelector_CascadeReplay_RequiresDisprovenFactId(t *testing.T) {
	sel := &types.ToKSelector{Variant: &types.ToKSelector_CascadeReplay{
		CascadeReplay: &types.CascadeReplaySelector{},
	}}
	err := keeper.ValidateToKSelector(sel)
	require.Error(t, err)
	require.Contains(t, err.Error(), "disproven_fact_id")
}

func TestValidateToKSelector_CascadeReplay_CapsMaxDepth(t *testing.T) {
	sel := &types.ToKSelector{Variant: &types.ToKSelector_CascadeReplay{
		CascadeReplay: &types.CascadeReplaySelector{
			DisprovenFactId: "fact-1", MaxDepth: 100,
		},
	}}
	capped, err := keeper.ValidateAndCapToKSelector(sel)
	require.NoError(t, err)
	require.Equal(t, uint32(3), capped.GetCascadeReplay().MaxDepth, "cascade depth caps at 3")
}

func TestValidateToKSelector_CascadeReplay_ZeroDepthDefaults(t *testing.T) {
	sel := &types.ToKSelector{Variant: &types.ToKSelector_CascadeReplay{
		CascadeReplay: &types.CascadeReplaySelector{
			DisprovenFactId: "fact-1", MaxDepth: 0,
		},
	}}
	capped, err := keeper.ValidateAndCapToKSelector(sel)
	require.NoError(t, err)
	require.Equal(t, uint32(1), capped.GetCascadeReplay().MaxDepth, "zero-depth defaults to first-hop only")
}

// ─── Task 9: GatherCascade ───────────────────────────────────────────────────

func TestGatherCascade_ReturnsDisproofPlusCascadedDescendants(t *testing.T) {
	k, ctx, _, _ := setupKnowledgeTestFull(t)

	// Pre-seed: disproven axiom + 2 cascaded descendants.
	require.NoError(t, k.SetFact(ctx, &types.Fact{
		Id: "disproven-x", Domain: "physics",
		Status: types.FactStatus_FACT_STATUS_DISPROVEN, VerifiedAtBlock: 100,
	}))
	require.NoError(t, k.SetFact(ctx, &types.Fact{
		Id: "child-1", Domain: "physics",
		Status: types.FactStatus_FACT_STATUS_CONTESTED, VerifiedAtBlock: 100,
	}))
	require.NoError(t, k.SetFact(ctx, &types.Fact{
		Id: "child-2", Domain: "physics",
		Status: types.FactStatus_FACT_STATUS_CONTESTED, VerifiedAtBlock: 100,
	}))

	// Pre-seed: cascade events.
	require.NoError(t, k.RecordCascadeEvent(ctx, &types.CascadeEvent{
		DisprovenFactId: "disproven-x", DescendantFactId: "child-1",
		EdgeRelation: "RELATION_TYPE_SUPPORTS", BlockHeight: 200,
	}))
	require.NoError(t, k.RecordCascadeEvent(ctx, &types.CascadeEvent{
		DisprovenFactId: "disproven-x", DescendantFactId: "child-2",
		EdgeRelation: "RELATION_TYPE_REQUIRES", BlockHeight: 200,
	}))

	// Pre-seed: relations (child-1 SUPPORTS disproven-x, child-2 REQUIRES disproven-x).
	require.NoError(t, k.SetFactRelation(ctx, &types.FactRelation{
		SourceFactId: "child-1", TargetFactId: "disproven-x",
		Relation: types.RelationType_RELATION_TYPE_SUPPORTS,
	}))
	require.NoError(t, k.SetFactRelation(ctx, &types.FactRelation{
		SourceFactId: "child-2", TargetFactId: "disproven-x",
		Relation: types.RelationType_RELATION_TYPE_REQUIRES,
	}))

	sel := &types.CascadeReplaySelector{
		DisprovenFactId: "disproven-x", MaxDepth: 1,
	}
	nodeIDs, edges, cascadeEvents, _, _, err := k.GatherCascade(ctx, sel)
	require.NoError(t, err)
	require.Equal(t, []string{"child-1", "child-2", "disproven-x"}, nodeIDs)
	require.Len(t, cascadeEvents, 2)
	// Edges include the CONTRADICTS that flipped the axiom (if recorded
	// as a relation) and the SUPPORTS/REQUIRES edges that cascaded.
	require.NotEmpty(t, edges)
}

func TestGatherCascade_RejectsNonDisprovenRoot(t *testing.T) {
	k, ctx, _, _ := setupKnowledgeTestFull(t)

	require.NoError(t, k.SetFact(ctx, &types.Fact{
		Id: "still-verified", Domain: "physics",
		Status: types.FactStatus_FACT_STATUS_VERIFIED,
	}))

	sel := &types.CascadeReplaySelector{
		DisprovenFactId: "still-verified", MaxDepth: 1,
	}
	_, _, _, _, _, err := k.GatherCascade(ctx, sel)
	require.Error(t, err, "TC4: cascade replay must reject non-DISPROVEN roots")
	require.Contains(t, err.Error(), "DISPROVEN")
}

// ─── Task 10: ComputeToKSnapshotRootV2 ───────────────────────────────────────

func TestComputeToKSnapshotRootV2_DistinctFromV1(t *testing.T) {
	nodeIDs := []string{"a", "b"}
	edges := []*types.ToKEdge{{FromFactId: "b", ToFactId: "a", Relation: "SUPPORTS"}}

	v1 := keeper.ComputeToKSnapshotRoot(nodeIDs, edges)
	v2 := keeper.ComputeToKSnapshotRootV2(nodeIDs, edges, nil, nil, nil)

	require.NotEqual(t, v1, v2, "V1 and V2 roots must differ even with empty cascade fields")
	require.Len(t, v2, 32)
}

func TestComputeToKSnapshotRootV2_DomainSeparated(t *testing.T) {
	cascadeEvent := []*types.CascadeEvent{{
		DisprovenFactId: "x", DescendantFactId: "y", EdgeRelation: "SUPPORTS",
	}}

	rWithCascade := keeper.ComputeToKSnapshotRootV2([]string{"x", "y"}, nil, cascadeEvent, nil, nil)
	rWithoutCascade := keeper.ComputeToKSnapshotRootV2([]string{"x", "y"}, nil, nil, nil, nil)

	require.NotEqual(t, rWithCascade, rWithoutCascade, "cascade events must affect root")
}

func TestComputeToKSnapshotRootV2_Deterministic(t *testing.T) {
	nodeIDs := []string{"a", "b"}
	cascade := []*types.CascadeEvent{{
		DisprovenFactId: "a", DescendantFactId: "b", EdgeRelation: "SUPPORTS", BlockHeight: 100,
	}}
	r1 := keeper.ComputeToKSnapshotRootV2(nodeIDs, nil, cascade, nil, nil)
	r2 := keeper.ComputeToKSnapshotRootV2(nodeIDs, nil, cascade, nil, nil)
	require.Equal(t, r1, r2)
}

// ─── Task 11: AssembleToKBundle V1/V2 dispatch ──────────────────────────────

func TestAssembleToKBundle_CascadeReplay(t *testing.T) {
	k, ctx, _, _ := setupKnowledgeTestFull(t)

	require.NoError(t, k.SetFact(ctx, &types.Fact{
		Id: "axiom-c", Domain: "physics",
		Status: types.FactStatus_FACT_STATUS_DISPROVEN, VerifiedAtBlock: 100,
	}))
	require.NoError(t, k.SetFact(ctx, &types.Fact{
		Id: "child-c", Domain: "physics",
		Status: types.FactStatus_FACT_STATUS_CONTESTED, VerifiedAtBlock: 100,
	}))
	require.NoError(t, k.RecordCascadeEvent(ctx, &types.CascadeEvent{
		DisprovenFactId: "axiom-c", DescendantFactId: "child-c",
		EdgeRelation: "RELATION_TYPE_SUPPORTS", BlockHeight: 200,
	}))

	sel := &types.ToKSelector{Variant: &types.ToKSelector_CascadeReplay{
		CascadeReplay: &types.CascadeReplaySelector{
			DisprovenFactId: "axiom-c", MaxDepth: 1,
		},
	}}
	bundle, err := k.AssembleToKBundle(ctx, sel, 0)
	require.NoError(t, err)
	require.NotEmpty(t, bundle.SnapshotRoot)
	require.Len(t, bundle.SnapshotRoot, 32)
	require.Len(t, bundle.CascadeEvents, 1)
	require.Equal(t, "child-c", bundle.CascadeEvents[0].DescendantFactId)
	require.Equal(t, "v2", bundle.Provenance.TokRootVersion)

	rederived := keeper.ComputeToKSnapshotRootV2(
		bundle.IncludedNodeIds, bundle.IncludedEdges,
		bundle.CascadeEvents, bundle.Vindications, bundle.StatusHistory,
	)
	require.Equal(t, bundle.SnapshotRoot, rederived)
}

func TestAssembleToKBundle_RootedSubtree_StaysV1(t *testing.T) {
	k, ctx, _, _ := setupKnowledgeTestFull(t)
	require.NoError(t, k.SetFact(ctx, &types.Fact{
		Id: "axiom-v1", Domain: "physics",
		Status: types.FactStatus_FACT_STATUS_VERIFIED,
	}))
	sel := &types.ToKSelector{Variant: &types.ToKSelector_RootedSubtree{
		RootedSubtree: &types.RootedSubtreeSelector{RootFactId: "axiom-v1", MaxDepth: 1},
	}}
	bundle, err := k.AssembleToKBundle(ctx, sel, 0)
	require.NoError(t, err)
	require.Equal(t, "v1", bundle.Provenance.TokRootVersion, "non-cascade selectors stay V1")
	require.Empty(t, bundle.CascadeEvents)
}

// ─── Task 12: JSONL cascade fields ────────────────────────────────────────────

func TestSerialiseToK_JSONL_IncludesCascadeFields(t *testing.T) {
	bundle := &types.ToKBundle{
		IncludedNodeIds: []string{"a", "b"},
		IncludedEdges:   []*types.ToKEdge{{FromFactId: "b", ToFactId: "a", Relation: "CONTRADICTS"}},
		Nodes:           []*types.Fact{{Id: "a"}, {Id: "b"}},
		CascadeEvents: []*types.CascadeEvent{{
			DisprovenFactId: "a", DescendantFactId: "b", EdgeRelation: "SUPPORTS",
		}},
		Vindications: []*types.ToKVindicationRecord{{
			FactId: "a", Verifier: "v1", RefundAmount: "100", BonusAmount: "10",
		}},
		StatusHistory: []*types.StatusTransition{{
			FactId: "a", PriorStatus: types.FactStatus_FACT_STATUS_VERIFIED,
			NewStatus: types.FactStatus_FACT_STATUS_DISPROVEN,
		}},
	}
	payload, err := keeper.SerialiseToK_JSONL(bundle)
	require.NoError(t, err)
	lines := bytes.Split(payload, []byte("\n"))
	if len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}
	require.Len(t, lines, 6)
	require.Contains(t, string(lines[3]), `"kind":"cascade_event"`)
	require.Contains(t, string(lines[4]), `"kind":"vindication"`)
	require.Contains(t, string(lines[5]), `"kind":"transition"`)
}

// ─── Task 13: BundleToK gRPC handler ─────────────────────────────────────────

func TestQueryBundleToK_CascadeReplay(t *testing.T) {
	k, ctx, _, _ := setupKnowledgeTestFull(t)

	require.NoError(t, k.SetFact(ctx, &types.Fact{
		Id: "axiom-rpc", Domain: "physics",
		Status: types.FactStatus_FACT_STATUS_DISPROVEN,
	}))

	q := keeper.NewQueryServerImpl(k)
	resp, err := q.BundleToK(ctx, &types.QueryBundleToKRequest{
		Selector: &types.ToKSelector{Variant: &types.ToKSelector_CascadeReplay{
			CascadeReplay: &types.CascadeReplaySelector{DisprovenFactId: "axiom-rpc", MaxDepth: 1},
		}},
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Bundle)
	require.Equal(t, "v2", resp.Bundle.Provenance.TokRootVersion)
}

func TestQueryBundleToK_CascadeReplay_NotDisprovenReturnsFailedPrecondition(t *testing.T) {
	k, ctx, _, _ := setupKnowledgeTestFull(t)

	require.NoError(t, k.SetFact(ctx, &types.Fact{
		Id: "still-active", Domain: "physics",
		Status: types.FactStatus_FACT_STATUS_VERIFIED,
	}))

	q := keeper.NewQueryServerImpl(k)
	_, err := q.BundleToK(ctx, &types.QueryBundleToKRequest{
		Selector: &types.ToKSelector{Variant: &types.ToKSelector_CascadeReplay{
			CascadeReplay: &types.CascadeReplaySelector{DisprovenFactId: "still-active", MaxDepth: 1},
		}},
	})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.FailedPrecondition, st.Code(), "non-DISPROVEN root → FailedPrecondition")
}

// ─── Task 14: RouteBCapabilities ──────────────────────────────────────────────

func TestRouteBCapabilities_AdvertisesCascadeReplay(t *testing.T) {
	k, ctx, _, _ := setupKnowledgeTestFull(t)
	q := keeper.NewQueryServerImpl(k)
	resp, err := q.RouteBCapabilities(ctx, &types.QueryRouteBCapabilitiesRequest{})
	require.NoError(t, err)
	require.Contains(t, resp.TokCapabilities.SupportedSelectors, "cascade_replay",
		"TC4: cascade_replay must be advertised")
	require.Contains(t, resp.TokCapabilities.TokDoctrineVersion, "TC4",
		"doctrine version must reflect TC4 binding")
}

// ─── Task 16: cascade_replayed event ──────────────────────────────────────────

func TestCascadeReplay_EmitsCascadeReplayedEvent(t *testing.T) {
	k, ctx, _, _ := setupKnowledgeTestFull(t)

	require.NoError(t, k.SetFact(ctx, &types.Fact{
		Id: "axiom-voice", Domain: "physics",
		Status: types.FactStatus_FACT_STATUS_DISPROVEN,
	}))

	sel := &types.ToKSelector{Variant: &types.ToKSelector_CascadeReplay{
		CascadeReplay: &types.CascadeReplaySelector{DisprovenFactId: "axiom-voice", MaxDepth: 1},
	}}
	_, err := k.AssembleToKBundle(ctx, sel, 0)
	require.NoError(t, err)

	events := sdk.UnwrapSDKContext(ctx).EventManager().Events()
	var sawReplayed bool
	for _, e := range events {
		if e.Type == keeper.EventTypeCascadeReplayed {
			sawReplayed = true
			for _, a := range e.Attributes {
				if a.Key == keeper.AttrToKCommitment {
					require.Equal(t, "TC4", a.Value)
				}
			}
		}
	}
	require.True(t, sawReplayed, "TC4: cascade_replayed must be emitted on cascade-replay bundle")
}