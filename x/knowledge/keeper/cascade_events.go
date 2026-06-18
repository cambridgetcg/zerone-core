package keeper

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// RecordCascadeEvent appends a CascadeEvent to the per-disproof log and
// updates the reverse index.
//
// TC4: cascade events are bundled with the substrate. The chain emits a
// `falsification_cascade` voice event when this fires; the record is what
// makes TC4 bundle-able. Commitment 10: forward-only.
func (k Keeper) RecordCascadeEvent(ctx context.Context, ev *types.CascadeEvent) error {
	if ev == nil || ev.DisprovenFactId == "" || ev.DescendantFactId == "" {
		return fmt.Errorf("cascade event requires disproven_fact_id and descendant_fact_id")
	}

	store := k.storeService.OpenKVStore(ctx)

	// Allocate seq by counting existing entries for this disproof root.
	iter, err := store.Iterator(
		types.CascadeEventPrefixForDisproof(ev.DisprovenFactId),
		prefixEndBytes(types.CascadeEventPrefixForDisproof(ev.DisprovenFactId)),
	)
	if err != nil {
		return fmt.Errorf("cascade event seq allocation: %w", err)
	}
	var lastSeq uint64
	for ; iter.Valid(); iter.Next() {
		existing := &types.CascadeEvent{}
		if err := proto.Unmarshal(iter.Value(), existing); err == nil {
			if existing.Seq > lastSeq {
				lastSeq = existing.Seq
			}
		}
	}
	iter.Close()
	ev.Seq = lastSeq + 1

	// Persist cascade event.
	bz, err := marshalOpts.Marshal(ev)
	if err != nil {
		return fmt.Errorf("marshal cascade event: %w", err)
	}
	if err := store.Set(types.CascadeEventKey(ev.DisprovenFactId, ev.Seq), bz); err != nil {
		return err
	}

	// Persist reverse-index marker.
	return store.Set(types.CascadeEventByDescendantKey(ev.DescendantFactId, ev.DisprovenFactId), []byte{0x01})
}

// GetCascadeEventsForDisproof returns all cascade events for a single
// disproof, sorted by seq.
func (k Keeper) GetCascadeEventsForDisproof(ctx context.Context, disprovenFactID string) []*types.CascadeEvent {
	store := k.storeService.OpenKVStore(ctx)
	prefix := types.CascadeEventPrefixForDisproof(disprovenFactID)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var out []*types.CascadeEvent
	for ; iter.Valid(); iter.Next() {
		ev := &types.CascadeEvent{}
		if err := proto.Unmarshal(iter.Value(), ev); err == nil {
			out = append(out, ev)
		}
	}
	return out
}

// GetDisproofsAffectingDescendant returns the disproven_fact_ids of every
// disproof that cascaded onto the given descendant. Used by training-data
// auditors to surface "this fact was hit by N disproofs."
func (k Keeper) GetDisproofsAffectingDescendant(ctx context.Context, descendantFactID string) []string {
	store := k.storeService.OpenKVStore(ctx)
	prefix := types.CascadeEventByDescendantPrefixFor(descendantFactID)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var out []string
	for ; iter.Valid(); iter.Next() {
		// Key suffix after the prefix is the disproven_fact_id bytes.
		key := iter.Key()
		out = append(out, string(key[len(prefix):]))
	}
	return out
}

// IterateCascadeEvents iterates every cascade event. Used by genesis export.
func (k Keeper) IterateCascadeEvents(ctx context.Context, f func(*types.CascadeEvent) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.CascadeEventKeyPrefix, prefixEndBytes(types.CascadeEventKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		ev := &types.CascadeEvent{}
		if err := proto.Unmarshal(iter.Value(), ev); err != nil {
			continue
		}
		if f(ev) {
			return
		}
	}
}