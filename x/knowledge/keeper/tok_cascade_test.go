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