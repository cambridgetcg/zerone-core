package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
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