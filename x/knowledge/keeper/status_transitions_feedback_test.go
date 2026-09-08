package keeper_test

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func TestStatusSequenceCheckedAndAtomic(t *testing.T) {
	for _, bad := range [][]byte{{}, {0x80}, {0x81, 0}, {0, 1}, binary.AppendUvarint(nil, math.MaxUint64)} {
		k, ctx, s := setupFeedbackStore(t)
		require.NoError(t, s.KVStoreService.OpenKVStore(ctx).Set(types.StatusTransitionSeqKey("f"), bad))
		before := feedbackSnapshot(t, s, ctx)
		tr := &types.StatusTransition{FactId: "f", PriorStatus: types.FactStatus_FACT_STATUS_ACTIVE, NewStatus: types.FactStatus_FACT_STATUS_CHALLENGED}
		require.Error(t, k.RecordStatusTransition(ctx, tr))
		require.Zero(t, tr.Seq)
		require.Equal(t, before, feedbackSnapshot(t, s, ctx))
	}
	for _, op := range []string{"get", "set"} {
		k, ctx, s := setupFeedbackStore(t)
		before := feedbackSnapshot(t, s, ctx)
		s.op, s.prefix = op, types.StatusTransitionSeqKeyPrefix
		tr := &types.StatusTransition{FactId: "f", PriorStatus: types.FactStatus_FACT_STATUS_ACTIVE, NewStatus: types.FactStatus_FACT_STATUS_CHALLENGED}
		require.ErrorIs(t, k.RecordStatusTransition(ctx, tr), feedbackStoreErr)
		require.Equal(t, before, feedbackSnapshot(t, s, ctx))
		require.Zero(t, tr.Seq)
	}
	k, ctx, s := setupFeedbackStore(t)
	require.NoError(t, s.KVStoreService.OpenKVStore(ctx).Set(types.StatusTransitionSeqKey("f"), binary.AppendUvarint(nil, 19)))
	tr := &types.StatusTransition{FactId: "f", PriorStatus: types.FactStatus_FACT_STATUS_ACTIVE, NewStatus: types.FactStatus_FACT_STATUS_CHALLENGED}
	require.NoError(t, k.RecordStatusTransition(ctx, tr))
	require.EqualValues(t, 20, tr.Seq)
}
