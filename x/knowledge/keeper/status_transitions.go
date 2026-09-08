package keeper

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"google.golang.org/protobuf/proto"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// RecordStatusTransition appends a forward-only StatusTransition entry for
// the given fact. The seq is auto-allocated from a per-fact counter.
//
// TC4: "Fact.status is preserved in graph manifests with full transition
// history." This is the function that makes "full transition history" real.
// Commitment 10: forward-only audit; transitions never modify in place.
func (k Keeper) RecordStatusTransition(ctx context.Context, t *types.StatusTransition) error {
	if t == nil || t.FactId == "" {
		return fmt.Errorf("status transition requires fact_id")
	}
	if t.PriorStatus == t.NewStatus {
		// No-op: same status. Don't write a transition for an unchanged status
		// — this keeps the history meaningful.
		return nil
	}

	cache, write := sdk.UnwrapSDKContext(ctx).CacheContext()
	store := k.storeService.OpenKVStore(cache)
	seqKey := types.StatusTransitionSeqKey(t.FactId)
	buf, err := store.Get(seqKey)
	if err != nil {
		return fmt.Errorf("read status sequence: %w", err)
	}
	var last uint64
	if buf != nil {
		last, err = decodeStatusSequence(buf)
		if err != nil {
			return err
		}
	}
	if last == math.MaxUint64 {
		return fmt.Errorf("status sequence exhausted")
	}
	prefix := types.StatusTransitionPrefixForFact(t.FactId)
	it, err := store.ReverseIterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return err
	}
	var historyErr error
	if it.Valid() {
		previous := new(types.StatusTransition)
		if e := proto.Unmarshal(it.Value(), previous); e != nil {
			historyErr = e
		} else if !bytes.Equal(it.Key(), types.StatusTransitionKey(t.FactId, previous.Seq)) || previous.Seq == 0 || previous.Seq > last {
			historyErr = fmt.Errorf("status counter missing or behind retained history")
		}
	}
	if err := errors.Join(historyErr, feedbackIteratorError(it), it.Close()); err != nil {
		return err
	}
	next := last + 1
	key := types.StatusTransitionKey(t.FactId, next)
	if found, err := store.Has(key); err != nil {
		return err
	} else if found {
		return fmt.Errorf("status sequence would overwrite existing history")
	}
	record := proto.Clone(t).(*types.StatusTransition)
	record.Seq = next
	bz, err := marshalOpts.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal status transition: %w", err)
	}
	if err := store.Set(key, bz); err != nil {
		return err
	}
	if err := store.Set(seqKey, binary.AppendUvarint(nil, next)); err != nil {
		return err
	}
	write()
	t.Seq = next
	return nil
}

// decodeStatusSequence rejects truncated, overflowed and noncanonical counters.
// A counter may legitimately exceed the last surviving history sequence.
func decodeStatusSequence(buf []byte) (uint64, error) {
	n, size := binary.Uvarint(buf)
	if size <= 0 || size != len(buf) || !bytes.Equal(buf, binary.AppendUvarint(nil, n)) {
		return 0, fmt.Errorf("invalid status sequence counter")
	}
	return n, nil
}

// GetStatusHistory returns all status transitions for a fact, sorted by seq.
// Empty slice for facts with no recorded transitions (e.g. genesis facts
// that never changed status post-genesis).
func (k Keeper) GetStatusHistory(ctx context.Context, factID string) []*types.StatusTransition {
	store := k.storeService.OpenKVStore(ctx)
	prefix := types.StatusTransitionPrefixForFact(factID)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var out []*types.StatusTransition
	for ; iter.Valid(); iter.Next() {
		t := &types.StatusTransition{}
		if err := proto.Unmarshal(iter.Value(), t); err != nil {
			continue
		}
		out = append(out, t)
	}
	return out
}

// IterateStatusTransitions calls f for every transition in the store.
// Used by genesis export and audit tooling.
func (k Keeper) IterateStatusTransitions(ctx context.Context, f func(*types.StatusTransition) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.StatusTransitionKeyPrefix, prefixEndBytes(types.StatusTransitionKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		t := &types.StatusTransition{}
		if err := proto.Unmarshal(iter.Value(), t); err != nil {
			continue
		}
		if f(t) {
			return
		}
	}
}

// SetFactSkipTransition is the same as SetFact but skips the auto
// status-transition record. Use when the caller has already written a
// precise StatusTransition with full cause attribution (e.g.
// cascadeFalsification, handleChallengeDisproven).
func (k Keeper) SetFactSkipTransition(ctx context.Context, fact *types.Fact) error {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := marshalOpts.Marshal(fact)
	if err != nil {
		return fmt.Errorf("failed to marshal fact: %w", err)
	}
	if err := store.Set(types.FactKey(fact.Id), bz); err != nil {
		return err
	}
	// Secondary indexes
	if fact.Submitter != "" {
		if err := store.Set(types.FactBySubmitterKey(fact.Submitter, fact.Id), []byte{0x01}); err != nil {
			return err
		}
	}
	if fact.Domain != "" {
		if err := store.Set(types.FactByDomainKey(fact.Domain, fact.Id), []byte{0x01}); err != nil {
			return err
		}
	}
	return nil
}

// inferStatusTransitionCause looks at the SDK context to infer what action
// triggered the status change. Best-effort; defaults to "unknown" if no
// signal can be read.
//
// The caller of SetFact often has rich context (round_id, challenge_id) that
// SetFact does not. As a future refinement, callers can stash the cause in
// the SDK context via a typed key. For now: best-effort string inspection.
func inferStatusTransitionCause(ctx sdk.Context, fact *types.Fact) (string, string) {
	switch fact.Status {
	case types.FactStatus_FACT_STATUS_DISPROVEN:
		return "challenge_disproven", ""
	case types.FactStatus_FACT_STATUS_CONTESTED:
		return "cascade", ""
	case types.FactStatus_FACT_STATUS_SUPERSEDED:
		return "supersession", ""
	case types.FactStatus_FACT_STATUS_VERIFIED:
		return "verification", fact.ClaimId
	default:
		return "unknown", ""
	}
}
