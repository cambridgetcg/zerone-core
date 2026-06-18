package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

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