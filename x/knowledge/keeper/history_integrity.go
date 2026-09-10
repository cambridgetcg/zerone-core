package keeper

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	corestore "cosmossdk.io/core/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

const knowledgeHistoryMaxEntries = 100000
const knowledgeHistoryMaxBytes = 64 * 1024 * 1024

func (k Keeper) getFactChecked(ctx context.Context, id string) (*types.Fact, bool, error) {
	bz, err := k.storeService.OpenKVStore(ctx).Get(types.FactKey(id))
	if err != nil {
		return nil, false, err
	}
	if bz == nil {
		return nil, false, nil
	}
	if err = validateToKStoredRecord(types.FactKey(id), bz); err != nil {
		return nil, false, err
	}
	v := &types.Fact{}
	if err = proto.Unmarshal(bz, v); err != nil {
		return nil, false, err
	}
	if v.Id != id {
		return nil, false, fmt.Errorf("fact key/payload mismatch")
	}
	return v, true, nil
}
func (k Keeper) getRelationsChecked(ctx context.Context, prefix []byte) (out []*types.FactRelation, err error) {
	it, err := k.storeService.OpenKVStore(ctx).Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil, err
	}
	defer func() {
		err = errors.Join(err, it.Close())
		if err != nil {
			out = nil
		}
	}()
	for ; it.Valid(); it.Next() {
		if err = validateToKStoredRecord(it.Key(), it.Value()); err != nil {
			return nil, err
		}
		v := &types.FactRelation{}
		if err = proto.Unmarshal(it.Value(), v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, historyIteratorError(it)
}

// SetFact preserves the pre-upgrade path. Activated writes commit history,
// primary state and secondary indexes together, or preserve every prior byte.
func (k Keeper) SetFact(ctx context.Context, fact *types.Fact) error {
	enabled, err := k.RecordIntegrityEnabled(ctx)
	if err != nil {
		return err
	}
	if !enabled {
		return k.legacySetFact(ctx, fact)
	}
	return k.setFactWithHistory(ctx, fact, true)
}

func (k Keeper) SetFactSkipTransition(ctx context.Context, fact *types.Fact) error {
	enabled, err := k.RecordIntegrityEnabled(ctx)
	if err != nil {
		return err
	}
	if !enabled {
		return k.legacySetFactSkipTransition(ctx, fact)
	}
	return k.setFactWithHistory(ctx, fact, false)
}

func (k Keeper) setFactWithHistory(ctx context.Context, fact *types.Fact, record bool) error {
	if fact == nil || fact.Id == "" {
		return fmt.Errorf("fact requires id")
	}
	if err := validateHistoryStatuses(fact.Status, fact.Status); err != nil {
		return err
	}
	if hasUnknownRecordFields(fact.ProtoReflect()) {
		return fmt.Errorf("unknown fact fields")
	}
	cached, write := sdk.UnwrapSDKContext(ctx).CacheContext()
	store := k.storeService.OpenKVStore(cached)
	old, err := store.Get(types.FactKey(fact.Id))
	if err != nil {
		return err
	}
	var prior *types.Fact
	if old != nil {
		if err := validateToKStoredRecord(types.FactKey(fact.Id), old); err != nil {
			return err
		}
		prior = &types.Fact{}
		if err = proto.Unmarshal(old, prior); err != nil {
			return err
		}
		if prior.Id != fact.Id {
			return fmt.Errorf("fact key/payload mismatch")
		}
	}
	if record {
		priorStatus := types.FactStatus_FACT_STATUS_UNSPECIFIED
		if prior != nil {
			priorStatus = prior.Status
		}
		if priorStatus != fact.Status {
			cause, causeID := inferStatusTransitionCause(cached, fact)
			if err = k.recordStatusTransitionChecked(cached, &types.StatusTransition{FactId: fact.Id, PriorStatus: priorStatus, NewStatus: fact.Status, BlockHeight: uint64(cached.BlockHeight()), CauseEventType: cause, CauseId: causeID}); err != nil {
				return err
			}
		}
	}
	if prior != nil {
		if prior.Submitter != "" && prior.Submitter != fact.Submitter {
			if err = store.Delete(types.FactBySubmitterKey(prior.Submitter, prior.Id)); err != nil {
				return err
			}
		}
		if prior.Domain != "" && prior.Domain != fact.Domain {
			if err = store.Delete(types.FactByDomainKey(prior.Domain, prior.Id)); err != nil {
				return err
			}
		}
	}
	bz, err := marshalOpts.Marshal(fact)
	if err != nil {
		return err
	}
	if err = store.Set(types.FactKey(fact.Id), bz); err != nil {
		return err
	}
	if fact.Submitter != "" {
		if err = store.Set(types.FactBySubmitterKey(fact.Submitter, fact.Id), []byte{1}); err != nil {
			return err
		}
	}
	if fact.Domain != "" {
		if err = store.Set(types.FactByDomainKey(fact.Domain, fact.Id), []byte{1}); err != nil {
			return err
		}
	}
	write()
	return nil
}

func (k Keeper) RecordStatusTransition(ctx context.Context, t *types.StatusTransition) error {
	enabled, err := k.RecordIntegrityEnabled(ctx)
	if err != nil {
		return err
	}
	if !enabled {
		return k.legacyRecordStatusTransition(ctx, t)
	}
	return k.recordStatusTransitionChecked(ctx, t)
}

func (k Keeper) recordStatusTransitionChecked(ctx context.Context, t *types.StatusTransition) error {
	if t == nil || t.FactId == "" {
		return fmt.Errorf("status transition requires fact_id")
	}
	if err := validateHistoryStatuses(t.PriorStatus, t.NewStatus); err != nil {
		return err
	}
	if len(t.ProtoReflect().GetUnknown()) != 0 {
		return fmt.Errorf("unknown transition fields")
	}
	if t.PriorStatus == t.NewStatus {
		return nil
	}
	cached, write := sdk.UnwrapSDKContext(ctx).CacheContext()
	store := k.storeService.OpenKVStore(cached)
	bz, err := store.Get(types.StatusTransitionSeqKey(t.FactId))
	if err != nil {
		return err
	}
	var seq uint64
	if bz != nil {
		seq, err = decodeHistoryCounter(bz)
		if err != nil {
			return err
		}
	}
	last, err := lastHistorySequence(store, types.StatusTransitionPrefixForFact(t.FactId), false)
	if err != nil {
		return err
	}
	if seq < last {
		return fmt.Errorf("status counter %d is behind recorded sequence %d", seq, last)
	}
	if seq == math.MaxUint64 {
		return fmt.Errorf("status sequence exhausted")
	}
	next := proto.Clone(t).(*types.StatusTransition)
	next.Seq = seq + 1
	key := types.StatusTransitionKey(next.FactId, next.Seq)
	if existing, err := store.Get(key); err != nil {
		return err
	} else if existing != nil {
		return fmt.Errorf("status transition already exists")
	}
	buf := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(buf, next.Seq)
	if err = store.Set(types.StatusTransitionSeqKey(next.FactId), buf[:n]); err != nil {
		return err
	}
	bz, err = marshalOpts.Marshal(next)
	if err != nil {
		return err
	}
	if err = store.Set(key, bz); err != nil {
		return err
	}
	write()
	t.Seq = next.Seq
	return nil
}

func (k Keeper) RecordCascadeEvent(ctx context.Context, ev *types.CascadeEvent) error {
	enabled, err := k.RecordIntegrityEnabled(ctx)
	if err != nil {
		return err
	}
	if !enabled {
		return k.legacyRecordCascadeEvent(ctx, ev)
	}
	if ev == nil || ev.DisprovenFactId == "" || ev.DescendantFactId == "" {
		return fmt.Errorf("cascade event requires root and descendant")
	}
	if err := validateHistoryStatuses(ev.PriorStatus, ev.NewStatus); err != nil {
		return err
	}
	if len(ev.ProtoReflect().GetUnknown()) != 0 {
		return fmt.Errorf("unknown cascade fields")
	}
	cached, write := sdk.UnwrapSDKContext(ctx).CacheContext()
	store := k.storeService.OpenKVStore(cached)
	seq, err := lastHistorySequence(store, types.CascadeEventPrefixForDisproof(ev.DisprovenFactId), true)
	if err != nil {
		return err
	}
	if seq == math.MaxUint64 {
		return fmt.Errorf("cascade sequence exhausted")
	}
	next := proto.Clone(ev).(*types.CascadeEvent)
	next.Seq = seq + 1
	if bytes.HasPrefix(types.CascadeEventByDescendantKey(next.DescendantFactId, next.DisprovenFactId), migrationMarkerPrefix) {
		return fmt.Errorf("cascade descendant collides with reserved migration namespace")
	}
	key := types.CascadeEventKey(next.DisprovenFactId, next.Seq)
	if existing, err := store.Get(key); err != nil {
		return err
	} else if existing != nil {
		return fmt.Errorf("cascade event already exists")
	}
	bz, err := marshalOpts.Marshal(next)
	if err != nil {
		return err
	}
	if err = store.Set(key, bz); err != nil {
		return err
	}
	if err = store.Set(types.CascadeEventByDescendantKey(next.DescendantFactId, next.DisprovenFactId), []byte{1}); err != nil {
		return err
	}
	write()
	ev.Seq = next.Seq
	return nil
}

// Ordered keys already contain the sequence. A reverse seek replaces the old
// scan of all previous events; the one-time activation audit validates history.
func lastHistorySequence(store corestore.KVStore, prefix []byte, cascade bool) (seq uint64, err error) {
	it, err := store.ReverseIterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return 0, err
	}
	defer func() { err = errors.Join(err, it.Close()) }()
	if !it.Valid() {
		return 0, historyIteratorError(it)
	}
	if err = validateToKStoredRecord(it.Key(), it.Value()); err != nil {
		return 0, err
	}
	if cascade {
		v := &types.CascadeEvent{}
		if err = proto.Unmarshal(it.Value(), v); err != nil {
			return 0, err
		}
		seq = v.Seq
	} else {
		v := &types.StatusTransition{}
		if err = proto.Unmarshal(it.Value(), v); err != nil {
			return 0, err
		}
		seq = v.Seq
	}
	if seq == 0 {
		return 0, fmt.Errorf("history sequence zero")
	}
	return seq, historyIteratorError(it)
}

func decodeHistoryCounter(bz []byte) (uint64, error) {
	v, n := binary.Uvarint(bz)
	if n <= 0 || n != len(bz) {
		return 0, fmt.Errorf("malformed status counter")
	}
	buf := make([]byte, binary.MaxVarintLen64)
	m := binary.PutUvarint(buf, v)
	if !bytes.Equal(buf[:m], bz) {
		return 0, fmt.Errorf("noncanonical status counter")
	}
	return v, nil
}

func (k Keeper) GetStatusHistoryChecked(ctx context.Context, id string) (out []*types.StatusTransition, err error) {
	err = k.scanKnowledgeHistory(ctx, types.StatusTransitionPrefixForFact(id), func(key, value []byte) error {
		if err := validateToKStoredRecord(key, value); err != nil {
			return err
		}
		v := &types.StatusTransition{}
		if err := proto.Unmarshal(value, v); err != nil {
			return err
		}
		if v.FactId != id || v.Seq == 0 || !bytes.Equal(key, types.StatusTransitionKey(v.FactId, v.Seq)) {
			return fmt.Errorf("status history key/payload mismatch")
		}
		out = append(out, v)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
func (k Keeper) GetCascadeEventsForDisproofChecked(ctx context.Context, id string) (out []*types.CascadeEvent, err error) {
	err = k.scanKnowledgeHistory(ctx, types.CascadeEventPrefixForDisproof(id), func(key, value []byte) error {
		if err := validateToKStoredRecord(key, value); err != nil {
			return err
		}
		v := &types.CascadeEvent{}
		if err := proto.Unmarshal(value, v); err != nil {
			return err
		}
		if v.DisprovenFactId != id || v.DescendantFactId == "" || v.Seq == 0 || !bytes.Equal(key, types.CascadeEventKey(v.DisprovenFactId, v.Seq)) {
			return fmt.Errorf("cascade history key/payload mismatch")
		}
		out = append(out, v)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
func (k Keeper) scanKnowledgeHistory(ctx context.Context, prefix []byte, read func([]byte, []byte) error) (err error) {
	it, err := k.storeService.OpenKVStore(ctx).Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrToKInconsistentState, err)
	}
	defer func() { err = errors.Join(err, it.Close()) }()
	n, size := 0, 0
	for ; it.Valid(); it.Next() {
		if bytes.Equal(prefix, types.CascadeEventByDescendantPrefix) && bytes.HasPrefix(it.Key(), migrationMarkerPrefix) {
			continue
		}
		n++
		size += len(it.Key()) + len(it.Value())
		if n > knowledgeHistoryMaxEntries || size > knowledgeHistoryMaxBytes {
			return ErrToKResourceLimit
		}
		if err = read(it.Key(), it.Value()); err != nil {
			return fmt.Errorf("%w: %w", ErrToKInconsistentState, err)
		}
	}
	return historyIteratorError(it)
}

// ValidateKnowledgeHistoryState is a bounded, read-only activation/export audit.
// Missing past events and sequence gaps remain unknown, never synthesized.
// Counters ahead of the last record are retained; counters behind it refuse.
func (k Keeper) ValidateKnowledgeHistoryState(ctx context.Context) error {
	maxSeq := map[string]uint64{}
	counters := map[string]uint64{}
	pairs := map[string]bool{}
	indices := map[string]bool{}
	n, size := 0, 0
	for _, prefix := range [][]byte{types.StatusTransitionKeyPrefix, types.StatusTransitionSeqKeyPrefix, types.CascadeEventKeyPrefix, types.CascadeEventByDescendantPrefix} {
		err := k.scanKnowledgeHistory(ctx, prefix, func(key, value []byte) error {
			n++
			size += len(key) + len(value)
			if n > knowledgeHistoryMaxEntries || size > knowledgeHistoryMaxBytes {
				return ErrToKResourceLimit
			}
			if err := validateToKStoredRecord(key, value); err != nil {
				return err
			}
			switch key[0] {
			case types.StatusTransitionKeyPrefix[0]:
				v := &types.StatusTransition{}
				if err := proto.Unmarshal(value, v); err != nil {
					return err
				}
				if v.FactId == "" || v.Seq == 0 || !bytes.Equal(key, types.StatusTransitionKey(v.FactId, v.Seq)) {
					return fmt.Errorf("malformed status transition")
				}
				if v.Seq > maxSeq[v.FactId] {
					maxSeq[v.FactId] = v.Seq
				}
			case types.StatusTransitionSeqKeyPrefix[0]:
				id := string(key[1:])
				if id == "" {
					return fmt.Errorf("empty counter fact id")
				}
				seq, err := decodeHistoryCounter(value)
				if err != nil {
					return err
				}
				counters[id] = seq
			case types.CascadeEventKeyPrefix[0]:
				v := &types.CascadeEvent{}
				if err := proto.Unmarshal(value, v); err != nil {
					return err
				}
				if v.DisprovenFactId == "" || v.DescendantFactId == "" || v.Seq == 0 || !bytes.Equal(key, types.CascadeEventKey(v.DisprovenFactId, v.Seq)) {
					return fmt.Errorf("malformed cascade event")
				}
				if bytes.HasPrefix(types.CascadeEventByDescendantKey(v.DescendantFactId, v.DisprovenFactId), migrationMarkerPrefix) {
					return fmt.Errorf("cascade history collides with reserved migration namespace")
				}
				pairs[string(types.CascadeEventByDescendantKey(v.DescendantFactId, v.DisprovenFactId))] = true
			case types.CascadeEventByDescendantPrefix[0]:
				if !bytes.Equal(value, []byte{1}) {
					return fmt.Errorf("malformed cascade reverse index")
				}
				indices[string(key)] = true
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	for id, seq := range maxSeq {
		if counters[id] < seq {
			return fmt.Errorf("status counter for %s missing or behind history", id)
		}
	}
	for key := range pairs {
		if !indices[key] {
			return fmt.Errorf("cascade reverse index missing")
		}
	}
	for key := range indices {
		if !pairs[key] {
			return fmt.Errorf("orphan cascade reverse index")
		}
	}
	return nil
}

func (k Keeper) ExportKnowledgeHistory(ctx context.Context) (transitions []*types.StatusTransition, cascades []*types.CascadeEvent, counters []*types.StatusTransitionCounter, err error) {
	if err = k.ValidateKnowledgeHistoryState(ctx); err != nil {
		return nil, nil, nil, err
	}
	err = k.scanKnowledgeHistory(ctx, types.StatusTransitionKeyPrefix, func(_, value []byte) error {
		v := &types.StatusTransition{}
		if err := proto.Unmarshal(value, v); err != nil {
			return err
		}
		transitions = append(transitions, v)
		return nil
	})
	if err != nil {
		return nil, nil, nil, err
	}
	err = k.scanKnowledgeHistory(ctx, types.CascadeEventKeyPrefix, func(_, value []byte) error {
		v := &types.CascadeEvent{}
		if err := proto.Unmarshal(value, v); err != nil {
			return err
		}
		cascades = append(cascades, v)
		return nil
	})
	if err != nil {
		return nil, nil, nil, err
	}
	err = k.scanKnowledgeHistory(ctx, types.StatusTransitionSeqKeyPrefix, func(key, value []byte) error {
		v, err := decodeHistoryCounter(value)
		if err != nil {
			return err
		}
		counters = append(counters, &types.StatusTransitionCounter{FactId: string(key[1:]), Sequence: v})
		return nil
	})
	if err != nil {
		return nil, nil, nil, err
	}
	return transitions, cascades, counters, nil
}

// ImportKnowledgeHistory is only for genesis construction. SetFact may have
// emitted transient initial transitions while loading facts; replace those with
// the supplied history, including an explicitly empty/unknown historical log.
// Primary histories/counters are preserved; reverse markers are derived.
func (k Keeper) ImportKnowledgeHistory(ctx context.Context, transitions []*types.StatusTransition, cascades []*types.CascadeEvent, counters []*types.StatusTransitionCounter) error {
	if err := types.ValidateKnowledgeHistoryGenesis(&types.GenesisState{StatusTransitions: transitions, CascadeEvents: cascades, StatusTransitionCounters: counters}); err != nil {
		return err
	}
	cached, write := sdk.UnwrapSDKContext(ctx).CacheContext()
	store := k.storeService.OpenKVStore(cached)
	var keys [][]byte
	for _, prefix := range [][]byte{types.StatusTransitionKeyPrefix, types.StatusTransitionSeqKeyPrefix, types.CascadeEventKeyPrefix, types.CascadeEventByDescendantPrefix} {
		if err := k.scanKnowledgeHistory(cached, prefix, func(key, _ []byte) error {
			keys = append(keys, bytes.Clone(key))
			if len(keys) > knowledgeHistoryMaxEntries {
				return ErrToKResourceLimit
			}
			return nil
		}); err != nil {
			return err
		}
	}
	for _, key := range keys {
		if err := store.Delete(key); err != nil {
			return err
		}
	}
	seen := map[string]bool{}
	put := func(key, value []byte) error {
		if seen[string(key)] {
			return fmt.Errorf("duplicate imported history key %x", key)
		}
		seen[string(key)] = true
		if len(seen) > knowledgeHistoryMaxEntries {
			return ErrToKResourceLimit
		}
		return store.Set(key, value)
	}
	for _, v := range transitions {
		if v == nil || v.FactId == "" || v.Seq == 0 {
			return fmt.Errorf("invalid imported status transition")
		}
		bz, err := marshalOpts.Marshal(v)
		if err != nil {
			return err
		}
		if err = put(types.StatusTransitionKey(v.FactId, v.Seq), bz); err != nil {
			return err
		}
	}
	for _, v := range cascades {
		if v == nil || v.DisprovenFactId == "" || v.DescendantFactId == "" || v.Seq == 0 {
			return fmt.Errorf("invalid imported cascade")
		}
		bz, err := marshalOpts.Marshal(v)
		if err != nil {
			return err
		}
		if err = put(types.CascadeEventKey(v.DisprovenFactId, v.Seq), bz); err != nil {
			return err
		}
		if err = store.Set(types.CascadeEventByDescendantKey(v.DescendantFactId, v.DisprovenFactId), []byte{1}); err != nil {
			return err
		}
	}
	for _, v := range counters {
		if v == nil || v.FactId == "" {
			return fmt.Errorf("invalid imported history counter")
		}
		buf := make([]byte, binary.MaxVarintLen64)
		n := binary.PutUvarint(buf, v.Sequence)
		if err := put(types.StatusTransitionSeqKey(v.FactId), buf[:n]); err != nil {
			return err
		}
	}
	if err := k.ValidateKnowledgeHistoryState(cached); err != nil {
		return err
	}
	write()
	return nil
}
