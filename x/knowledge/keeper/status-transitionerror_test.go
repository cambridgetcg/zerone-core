package keeper_test

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func sequenceErrorTransition() *types.StatusTransition {
	return &types.StatusTransition{
		FactId:         "sequence-error-fact",
		Seq:            99,
		PriorStatus:    types.FactStatus_FACT_STATUS_ACTIVE,
		NewStatus:      types.FactStatus_FACT_STATUS_VERIFIED,
		BlockHeight:    9,
		CauseEventType: "verification",
		CauseId:        "sequence-error-round",
	}
}

func TestRecordStatusTransition_SequenceStorageFailuresAreAtomic(t *testing.T) {
	for _, tc := range []struct {
		name, op string
		prefix   []byte
	}{
		{"sequence-read", "get", types.StatusTransitionSeqKeyPrefix},
		{"sequence-write", "set", types.StatusTransitionSeqKeyPrefix},
		{"transition-write", "set", types.StatusTransitionKeyPrefix},
	} {
		t.Run(tc.name, func(t *testing.T) {
			k, ctx, s := setupFeedbackStore(t)
			require.NoError(t, k.RecordStatusTransition(ctx, sequenceErrorTransition()))
			before := feedbackSnapshot(t, s, ctx)
			record := sequenceErrorTransition()
			original := proto.Clone(record)
			s.op, s.prefix = tc.op, tc.prefix

			require.ErrorIs(t, k.RecordStatusTransition(ctx, record), feedbackStoreErr)
			require.Equal(t, before, feedbackSnapshot(t, s, ctx), "failed allocation must not change history or its counter")
			require.True(t, proto.Equal(original, record), "failed allocation must not report an assigned sequence")
		})
	}
}

func TestRecordStatusTransition_SequenceRejectsInvalidCounters(t *testing.T) {
	for _, tc := range []struct {
		name string
		buf  []byte
	}{
		{"empty", []byte{}},
		{"truncated", []byte{0x80}},
		{"overflow", []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x02}},
		{"trailing-bytes", []byte{0x01, 0x00}},
		{"noncanonical-zero", []byte{0x80, 0x00}},
		{"noncanonical-one", []byte{0x81, 0x00}},
		{"exhausted", binary.AppendUvarint(nil, math.MaxUint64)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			k, ctx, s := setupFeedbackStore(t)
			require.NoError(t, k.RecordStatusTransition(ctx, sequenceErrorTransition()))
			record := sequenceErrorTransition()
			original := proto.Clone(record)
			raw := s.KVStoreService.OpenKVStore(ctx)
			require.NoError(t, raw.Set(types.StatusTransitionSeqKey(record.FactId), tc.buf))
			before := feedbackSnapshot(t, s, ctx)

			require.Error(t, k.RecordStatusTransition(ctx, record))
			require.Equal(t, before, feedbackSnapshot(t, s, ctx), "invalid counters must not reset or overwrite existing history")
			require.True(t, proto.Equal(original, record))
		})
	}
}

func TestRecordStatusTransition_SequencePreservesLastAllocatedAndGaps(t *testing.T) {
	for _, tc := range []struct {
		name string
		last *uint64
		next uint64
	}{
		{"absent", nil, 1},
		{"zero", proto.Uint64(0), 1},
		{"one", proto.Uint64(1), 2},
		{"gap", proto.Uint64(127), 128},
		{"last-available", proto.Uint64(math.MaxUint64 - 1), math.MaxUint64},
	} {
		t.Run(tc.name, func(t *testing.T) {
			k, ctx, s := setupFeedbackStore(t)
			record := sequenceErrorTransition()
			raw := s.KVStoreService.OpenKVStore(ctx)
			var prior []byte
			if tc.last != nil {
				if *tc.last > 0 {
					require.NoError(t, k.RecordStatusTransition(ctx, sequenceErrorTransition()))
					var err error
					prior, err = raw.Get(types.StatusTransitionKey(record.FactId, 1))
					require.NoError(t, err)
				}
				require.NoError(t, raw.Set(types.StatusTransitionSeqKey(record.FactId), binary.AppendUvarint(nil, *tc.last)))
			}

			require.NoError(t, k.RecordStatusTransition(ctx, record))
			require.Equal(t, tc.next, record.Seq)
			counter, err := raw.Get(types.StatusTransitionSeqKey(record.FactId))
			require.NoError(t, err)
			require.Equal(t, binary.AppendUvarint(nil, tc.next), counter, "counter stores last allocated, not next available")
			history := k.GetStatusHistory(ctx, record.FactId)
			if prior != nil {
				require.Len(t, history, 2, "gaps must not be filled with fabricated transitions")
				unchanged, err := raw.Get(types.StatusTransitionKey(record.FactId, 1))
				require.NoError(t, err)
				require.Equal(t, prior, unchanged)
			} else {
				require.Len(t, history, 1)
			}
			require.True(t, proto.Equal(record, history[len(history)-1]))
		})
	}
}

func TestRecordStatusTransition_SequenceMarshalFailureIsAtomic(t *testing.T) {
	k, ctx, s := setupFeedbackStore(t)
	require.NoError(t, k.RecordStatusTransition(ctx, sequenceErrorTransition()))
	before := feedbackSnapshot(t, s, ctx)
	record := sequenceErrorTransition()
	record.CauseEventType = string([]byte{0xff})
	original := proto.Clone(record)

	require.Error(t, k.RecordStatusTransition(ctx, record))
	require.Equal(t, before, feedbackSnapshot(t, s, ctx))
	require.True(t, proto.Equal(original, record))
}

func TestRecordStatusTransition_SequenceNoOpDoesNotReadCounter(t *testing.T) {
	k, ctx, s := setupFeedbackStore(t)
	record := sequenceErrorTransition()
	record.NewStatus = record.PriorStatus
	original := proto.Clone(record)
	before := feedbackSnapshot(t, s, ctx)
	s.op, s.prefix = "get", types.StatusTransitionSeqKeyPrefix

	require.NoError(t, k.RecordStatusTransition(ctx, record))
	require.Equal(t, before, feedbackSnapshot(t, s, ctx))
	require.True(t, proto.Equal(original, record))
}
