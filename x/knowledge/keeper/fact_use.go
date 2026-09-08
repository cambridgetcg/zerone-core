package keeper

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	corestore "cosmossdk.io/core/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Feedback uses checked reads: legacy GetParams/GetFact treat corrupt state as
// defaults/absence, which is not safe for admission or monotonic accounting.
func (k Keeper) getFactUseParams(ctx context.Context) (*types.Params, error) {
	bz, err := k.storeService.OpenKVStore(ctx).Get(types.ParamsKey)
	if err != nil {
		return nil, err
	}
	if bz == nil {
		p := types.DefaultParams()
		return &p, nil
	}
	p := new(types.Params)
	if err := proto.Unmarshal(bz, p); err != nil {
		return nil, fmt.Errorf("decode feedback params: %w", err)
	}
	return p, nil
}

func (k Keeper) getFeedbackFact(ctx context.Context, id string) (*types.Fact, bool, error) {
	bz, err := k.storeService.OpenKVStore(ctx).Get(types.FactKey(id))
	if err != nil || bz == nil {
		return nil, false, err
	}
	f := new(types.Fact)
	if err := proto.Unmarshal(bz, f); err != nil {
		return nil, false, err
	}
	if f.Id != id {
		return nil, false, fmt.Errorf("fact key/payload mismatch")
	}
	return f, true, nil
}

// writeFeedbackFact updates only an existing fact payload. Feedback cannot alter
// identity, domain, submitter or status; no secondary index or inferred history
// write is needed. Restoration records its explicit transition separately.
func (k Keeper) writeFeedbackFact(ctx context.Context, fact *types.Fact) error {
	bz, err := marshalOpts.Marshal(fact)
	if err != nil {
		return err
	}
	return k.storeService.OpenKVStore(ctx).Set(types.FactKey(fact.Id), bz)
}

func decodeFactUseReceipt(key, value []byte) (*types.FactUseReceipt, error) {
	epoch, consumer, fact, err := types.ParseFactUseReceiptKey(key)
	if err != nil {
		return nil, err
	}
	r := new(types.FactUseReceipt)
	if err := proto.Unmarshal(value, r); err != nil {
		return nil, err
	}
	if r.Epoch != epoch || r.Consumer != consumer || r.FactId != fact || r.Version != types.FactUseReceiptVersion {
		return nil, fmt.Errorf("fact-use receipt key/payload/version mismatch")
	}
	return r, nil
}

func (k Keeper) GetFactUseReceipt(ctx context.Context, epoch uint64, consumer, fact string) (*types.FactUseReceipt, bool, error) {
	if _, err := types.CanonicalFactUseConsumer(consumer); err != nil {
		return nil, false, err
	}
	if err := types.ValidateFactUseFactID(fact); err != nil {
		return nil, false, err
	}
	key := types.FactUseReceiptKey(epoch, consumer, fact)
	bz, err := k.storeService.OpenKVStore(ctx).Get(key)
	if err != nil || bz == nil {
		return nil, false, err
	}
	r, err := decodeFactUseReceipt(key, bz)
	if err != nil {
		return nil, false, err
	}
	p, err := k.getFactUseParams(ctx)
	if err != nil {
		return nil, false, err
	}
	if err := r.Validate(p.FitnessEpochBlocks); err != nil {
		return nil, false, err
	}
	return r, true, nil
}

func (k Keeper) setFactUseReceipt(ctx context.Context, r *types.FactUseReceipt) error {
	bz, err := marshalOpts.Marshal(r)
	if err != nil {
		return err
	}
	return k.storeService.OpenKVStore(ctx).Set(types.FactUseReceiptKey(r.Epoch, r.Consumer, r.FactId), bz)
}

// Absent legacy pruning state is represented as an empty, non-nil value.
func (k Keeper) GetFactUsePruningState(ctx context.Context) (*types.FactUsePruningState, error) {
	bz, err := k.storeService.OpenKVStore(ctx).Get(types.FactUsePruningStateKey)
	if err != nil {
		return nil, err
	}
	p := new(types.FactUsePruningState)
	if bz != nil {
		if err := proto.Unmarshal(bz, p); err != nil {
			return nil, err
		}
	}
	if len(p.NextKey) > 0 {
		if _, _, _, err := types.ParseFactUseReceiptKey(p.NextKey); err != nil {
			return nil, err
		}
	}
	return p, nil
}

func (k Keeper) setFactUsePruningState(ctx context.Context, p *types.FactUsePruningState) error {
	bz, err := marshalOpts.Marshal(p)
	if err != nil {
		return err
	}
	return k.storeService.OpenKVStore(ctx).Set(types.FactUsePruningStateKey, bz)
}

func (k Keeper) HasRetainedFactUseReceipts(ctx context.Context) (bool, error) {
	it, err := k.storeService.OpenKVStore(ctx).Iterator(types.FactUseReceiptPrefix, prefixEndBytes(types.FactUseReceiptPrefix))
	if err != nil {
		return false, err
	}
	found := it.Valid()
	return found, errors.Join(feedbackIteratorError(it), it.Close())
}

func readFactUseCount(store corestore.KVStore, key []byte, max uint64) (uint64, error) {
	bz, err := store.Get(key)
	if err != nil || bz == nil {
		return 0, err
	}
	if len(bz) != 8 {
		return 0, fmt.Errorf("invalid fact-use counter encoding")
	}
	n := binary.BigEndian.Uint64(bz)
	if n == 0 || n > max {
		return 0, fmt.Errorf("invalid fact-use counter value")
	}
	return n, nil
}

func writeFactUseCount(store corestore.KVStore, key []byte, n uint64) error {
	if n == 0 {
		return store.Delete(key)
	}
	return store.Set(key, binary.BigEndian.AppendUint64(nil, n))
}

// Count canonical receipt keys, not values, under the hard retained cap. This
// bounded scan makes capacity and both admission quotas fail closed even if a
// derived counter is missing or understates retained state. No new key is used.
// Report admission examines at most 2,001 keys; pruning has its separate 100 cap.
func (k Keeper) factUseAdmissionCounts(ctx context.Context, epoch uint64, consumer string) (total, global, account uint64, err error) {
	store := k.storeService.OpenKVStore(ctx)
	it, err := store.Iterator(types.FactUseReceiptPrefix, prefixEndBytes(types.FactUseReceiptPrefix))
	if err != nil {
		return 0, 0, 0, err
	}
	defer func() { err = errors.Join(err, it.Close()) }()
	for ; it.Valid(); it.Next() {
		total++
		if total > types.MaxFactUseRetainedReceipts {
			return 0, 0, 0, fmt.Errorf("fact-use retained capacity exceeded")
		}
		e, c, _, parseErr := types.ParseFactUseReceiptKey(it.Key())
		if parseErr != nil {
			return 0, 0, 0, parseErr
		}
		if e == epoch {
			global++
			if c == consumer {
				account++
			}
		}
	}
	if err = feedbackIteratorError(it); err != nil {
		return 0, 0, 0, err
	}
	storedGlobal, err := readFactUseCount(store, types.FactUseEpochCountKey(epoch), types.MaxFactUsePerEpoch)
	if err != nil {
		return 0, 0, 0, err
	}
	storedAccount, err := readFactUseCount(store, types.FactUseConsumerCountKey(epoch, consumer), types.MaxFactUsePerConsumerEpoch)
	if err != nil {
		return 0, 0, 0, err
	}
	if storedGlobal != global || storedAccount != account {
		return 0, 0, 0, fmt.Errorf("fact-use derived quota count mismatch")
	}
	return total, global, account, nil
}

// ExportFactUseReceipts exports actual records, including rated and expired but
// unpruned markers. It never promotes legacy QueryReceiptPrefix entries.
func (k Keeper) ExportFactUseReceipts(ctx context.Context) (out []*types.FactUseReceipt, err error) {
	p, err := k.getFactUseParams(ctx)
	if err != nil {
		return nil, err
	}
	it, err := k.storeService.OpenKVStore(ctx).Iterator(types.FactUseReceiptPrefix, prefixEndBytes(types.FactUseReceiptPrefix))
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, it.Close()) }()
	for ; it.Valid(); it.Next() {
		if len(out) >= types.MaxFactUseRetainedReceipts {
			return nil, fmt.Errorf("fact-use retained capacity exceeded")
		}
		r, err := decodeFactUseReceipt(it.Key(), it.Value())
		if err != nil {
			return nil, err
		}
		if err := r.Validate(p.FitnessEpochBlocks); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, feedbackIteratorError(it)
}

// InitFactUseReceipts requires empty receipt/counter namespaces, validates actual
// records, and derives quota counts deterministically. It never resets an
// existing EverReported latch, nor synthesizes one for inconsistent imports.
// Invoke after params and facts are imported. All effects use an SDK cache.
func (k Keeper) InitFactUseReceipts(ctx context.Context, receipts []*types.FactUseReceipt, pruning *types.FactUsePruningState) error {
	cache, write := sdk.UnwrapSDKContext(ctx).CacheContext()
	store := k.storeService.OpenKVStore(cache)
	for _, prefix := range [][]byte{types.FactUseReceiptPrefix, types.FactUseEpochCountPrefix, types.FactUseConsumerCountPrefix} {
		it, err := store.Iterator(prefix, prefixEndBytes(prefix))
		if err != nil {
			return err
		}
		found := it.Valid()
		if err := errors.Join(feedbackIteratorError(it), it.Close()); err != nil {
			return err
		}
		if found {
			return fmt.Errorf("fact-use import requires empty receipt and counter namespaces")
		}
	}
	if len(receipts) > types.MaxFactUseRetainedReceipts {
		return fmt.Errorf("too many retained fact-use receipts")
	}
	p, err := k.getFactUseParams(cache)
	if err != nil {
		return err
	}
	state := new(types.FactUsePruningState)
	if pruning != nil {
		state = proto.Clone(pruning).(*types.FactUsePruningState)
	}
	old, err := k.GetFactUsePruningState(cache)
	if err != nil {
		return err
	}
	if old.EverReported && !state.EverReported {
		return fmt.Errorf("cannot clear ever_reported during import")
	}
	if len(state.NextKey) > 0 {
		if _, _, _, err := types.ParseFactUseReceiptKey(state.NextKey); err != nil {
			return err
		}
	}
	if err := types.ValidateFactUseNonEconomicState(p, state, len(receipts) > 0); err != nil {
		return err
	}
	seen := make(map[string]bool, len(receipts))
	for _, r := range receipts {
		if err := r.Validate(p.FitnessEpochBlocks); err != nil {
			return err
		}
		key := types.FactUseReceiptKey(r.Epoch, r.Consumer, r.FactId)
		if seen[string(key)] {
			return fmt.Errorf("duplicate fact-use receipt")
		}
		seen[string(key)] = true
		if _, found, err := k.getFeedbackFact(cache, r.FactId); err != nil {
			return err
		} else if !found {
			return fmt.Errorf("receipt fact %s missing", r.FactId)
		}
		for _, quota := range []struct {
			key []byte
			max uint64
		}{
			{types.FactUseEpochCountKey(r.Epoch), types.MaxFactUsePerEpoch},
			{types.FactUseConsumerCountKey(r.Epoch, r.Consumer), types.MaxFactUsePerConsumerEpoch},
		} {
			n, err := readFactUseCount(store, quota.key, quota.max)
			if err != nil {
				return err
			}
			if n >= quota.max {
				return fmt.Errorf("imported fact-use quota exceeded")
			}
			if err := writeFactUseCount(store, quota.key, n+1); err != nil {
				return err
			}
		}
		if err := k.setFactUseReceipt(cache, r); err != nil {
			return err
		}
	}
	if err := k.setFactUsePruningState(cache, state); err != nil {
		return err
	}
	write()
	return nil
}

// PruneFactUseReceipts examines at most 100 records, not 100 deletions. The
// inclusive NextKey resumes at the last examined key: if retained, it is
// re-examined (and counted); if deleted, iteration seeks the next extant key.
// This avoids inspecting a 101st key just to persist a cursor. Reaching the end
// resets it for the NEXT block, so insertions before it cannot be permanently
// skipped. No iterator writes; failures discard the SDK cache.
func (k Keeper) PruneFactUseReceipts(ctx context.Context) error {
	cache, write := sdk.UnwrapSDKContext(ctx).CacheContext()
	state, err := k.GetFactUsePruningState(cache)
	if err != nil {
		return err
	}
	p, err := k.getFactUseParams(cache)
	if err != nil {
		return err
	}
	height := cache.BlockHeight()
	if height < 0 {
		return fmt.Errorf("negative fact-use pruning height")
	}
	store := k.storeService.OpenKVStore(cache)
	start := types.FactUseReceiptPrefix
	if len(state.NextKey) > 0 {
		start = bytes.Clone(state.NextKey)
	}
	it, err := store.Iterator(start, prefixEndBytes(types.FactUseReceiptPrefix))
	if err != nil {
		return err
	}
	var expired []*types.FactUseReceipt
	var scanErr error
	examined := 0
	for ; it.Valid() && examined < types.FactUsePruneBatchSize; it.Next() {
		examined++
		r, err := decodeFactUseReceipt(it.Key(), it.Value())
		if err == nil {
			err = r.Validate(p.FitnessEpochBlocks)
		}
		if err != nil {
			scanErr = err
			break
		}
		if !state.EverReported {
			scanErr = fmt.Errorf("retained fact-use receipt without ever_reported")
			break
		}
		state.NextKey = bytes.Clone(it.Key())
		if r.ExpiryHeight <= uint64(height) {
			expired = append(expired, r)
		}
	}
	if !it.Valid() {
		state.NextKey = nil
	}
	if err := errors.Join(scanErr, feedbackIteratorError(it), it.Close()); err != nil {
		return err
	}
	for _, r := range expired {
		for _, quota := range []struct {
			key []byte
			max uint64
		}{
			{types.FactUseEpochCountKey(r.Epoch), types.MaxFactUsePerEpoch},
			{types.FactUseConsumerCountKey(r.Epoch, r.Consumer), types.MaxFactUsePerConsumerEpoch},
		} {
			n, err := readFactUseCount(store, quota.key, quota.max)
			if err != nil {
				return err
			}
			if n == 0 {
				return fmt.Errorf("fact-use quota underflow while pruning")
			}
			if err := writeFactUseCount(store, quota.key, n-1); err != nil {
				return err
			}
		}
		if err := store.Delete(types.FactUseReceiptKey(r.Epoch, r.Consumer, r.FactId)); err != nil {
			return err
		}
	}
	if err := k.setFactUsePruningState(cache, state); err != nil {
		return err
	}
	write()
	return nil
}
