package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/liquiditypool/types"
)

// --- Pool CRUD ---

func (k Keeper) SetPool(ctx sdk.Context, pool *types.Pool) {
	store := k.storeService.OpenKVStore(ctx)

	// Remove the previous secondary indexes before changing the canonical
	// record. This also repairs a denom change instead of leaving a stale pair
	// that resolves to the replacement pool.
	if previous, found := k.GetPool(ctx, pool.PoolId); found {
		if err := store.Delete(types.DenomPairKey(previous.DenomA, previous.DenomB)); err != nil {
			panic(fmt.Sprintf("failed to delete previous denom index: %v", err))
		}
		if err := store.Delete(types.OpenPoolIndexKey(previous.PoolId)); err != nil {
			panic(fmt.Sprintf("failed to delete previous open-pool index: %v", err))
		}
	}

	bz, err := proto.Marshal(pool)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal pool: %v", err))
	}
	if err := store.Set(types.PoolKey(pool.PoolId), bz); err != nil {
		panic(fmt.Sprintf("failed to set pool: %v", err))
	}

	// UNSPECIFIED remains indexable only so v3 state can be read while the
	// v3→v4 migration assigns an explicit status. New/genesis state validation
	// rejects it.
	if pool.Status != types.PoolStatus_POOL_STATUS_CLOSED {
		if err := store.Set(types.DenomPairKey(pool.DenomA, pool.DenomB), []byte(pool.PoolId)); err != nil {
			panic(fmt.Sprintf("failed to set denom index: %v", err))
		}
		if err := store.Set(types.OpenPoolIndexKey(pool.PoolId), []byte(pool.PoolId)); err != nil {
			panic(fmt.Sprintf("failed to set open-pool index: %v", err))
		}
	}
}

func (k Keeper) GetPool(ctx sdk.Context, poolID string) (*types.Pool, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.PoolKey(poolID))
	if err != nil || bz == nil {
		return nil, false
	}
	var pool types.Pool
	if err := proto.Unmarshal(bz, &pool); err != nil {
		return nil, false
	}
	return &pool, true
}

func (k Keeper) GetPoolByDenoms(ctx sdk.Context, denomA, denomB string) *types.Pool {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.DenomPairKey(denomA, denomB))
	if err != nil || bz == nil {
		return nil
	}
	pool, found := k.GetPool(ctx, string(bz))
	if !found || pool.Status == types.PoolStatus_POOL_STATUS_CLOSED {
		return nil
	}
	if !sameDenomPair(pool.DenomA, pool.DenomB, denomA, denomB) {
		return nil
	}
	return pool
}

func sameDenomPair(a1, b1, a2, b2 string) bool {
	return (a1 == a2 && b1 == b2) || (a1 == b2 && b1 == a2)
}

func (k Keeper) DeletePool(ctx sdk.Context, pool *types.Pool) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Delete(types.PoolKey(pool.PoolId)); err != nil {
		panic(fmt.Sprintf("failed to delete pool: %v", err))
	}
	if err := store.Delete(types.DenomPairKey(pool.DenomA, pool.DenomB)); err != nil {
		panic(fmt.Sprintf("failed to delete denom index: %v", err))
	}
	if err := store.Delete(types.OpenPoolIndexKey(pool.PoolId)); err != nil {
		panic(fmt.Sprintf("failed to delete open-pool index: %v", err))
	}
	k.DeleteTWAPHistory(ctx, pool.PoolId)
}

func (k Keeper) IteratePools(ctx sdk.Context, cb func(*types.Pool) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.PoolKeyPrefix, prefixEndBytes(types.PoolKeyPrefix))
	if err != nil {
		panic(fmt.Sprintf("failed to iterate pools: %v", err))
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var pool types.Pool
		if err := proto.Unmarshal(iter.Value(), &pool); err != nil {
			panic(fmt.Sprintf("failed to decode pool at key %X: %v", iter.Key(), err))
		}
		if cb(&pool) {
			break
		}
	}
}

// IterateOpenPools walks only the bounded secondary index. Closed tombstones
// remain queryable/auditable without growing BeginBlock work over time.
func (k Keeper) IterateOpenPools(ctx sdk.Context, cb func(*types.Pool) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.OpenPoolIndexPrefix, prefixEndBytes(types.OpenPoolIndexPrefix))
	if err != nil {
		panic(fmt.Sprintf("failed to iterate open pools: %v", err))
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		poolID := string(iter.Value())
		pool, found := k.GetPool(ctx, poolID)
		if !found || pool.Status == types.PoolStatus_POOL_STATUS_CLOSED {
			panic(fmt.Sprintf("stale open-pool index for %q", poolID))
		}
		if cb(pool) {
			break
		}
	}
}

func (k Keeper) CountPools(ctx sdk.Context) uint64 {
	var count uint64
	k.IteratePools(ctx, func(_ *types.Pool) bool {
		count++
		return false
	})
	return count
}

func (k Keeper) CountOpenPools(ctx sdk.Context) uint64 {
	var count uint64
	k.IterateOpenPools(ctx, func(_ *types.Pool) bool {
		count++
		return false
	})
	return count
}

// --- TWAP Accumulator CRUD ---

func (k Keeper) SetTWAPAccumulator(ctx sdk.Context, acc *types.TWAPAccumulator) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(acc)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal TWAP accumulator: %v", err))
	}
	if err := store.Set(types.TWAPKey(acc.PoolId), bz); err != nil {
		panic(fmt.Sprintf("failed to set TWAP accumulator: %v", err))
	}
}

func (k Keeper) GetTWAPAccumulator(ctx sdk.Context, poolID string) (*types.TWAPAccumulator, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.TWAPKey(poolID))
	if err != nil || bz == nil {
		return nil, false
	}
	var acc types.TWAPAccumulator
	if err := proto.Unmarshal(bz, &acc); err != nil {
		return nil, false
	}
	return &acc, true
}

func (k Keeper) IterateTWAPAccumulators(ctx sdk.Context, cb func(*types.TWAPAccumulator) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.TWAPKeyPrefix, prefixEndBytes(types.TWAPKeyPrefix))
	if err != nil {
		panic(fmt.Sprintf("failed to iterate TWAP accumulators: %v", err))
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var acc types.TWAPAccumulator
		if err := proto.Unmarshal(iter.Value(), &acc); err != nil {
			panic(fmt.Sprintf("failed to decode TWAP accumulator at key %X: %v", iter.Key(), err))
		}
		if cb(&acc) {
			break
		}
	}
}

func (k Keeper) SetTWAPObservation(ctx sdk.Context, observation *types.TWAPObservation) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(observation)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal TWAP observation: %v", err))
	}
	if err := store.Set(types.TWAPObservationKey(observation.PoolId, observation.BlockHeight), bz); err != nil {
		panic(fmt.Sprintf("failed to set TWAP observation: %v", err))
	}
}

func (k Keeper) IterateTWAPObservations(ctx sdk.Context, poolID string, cb func(*types.TWAPObservation) bool) {
	store := k.storeService.OpenKVStore(ctx)
	prefix := types.TWAPObservationKeyPrefix
	if poolID != "" {
		prefix = types.TWAPObservationPoolPrefix(poolID)
	}
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		panic(fmt.Sprintf("failed to iterate TWAP observations: %v", err))
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var observation types.TWAPObservation
		if err := proto.Unmarshal(iter.Value(), &observation); err != nil {
			panic(fmt.Sprintf("failed to decode TWAP observation at key %X: %v", iter.Key(), err))
		}
		if cb(&observation) {
			break
		}
	}
}

func (k Keeper) DeleteTWAPHistory(ctx sdk.Context, poolID string) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Delete(types.TWAPKey(poolID)); err != nil {
		panic(fmt.Sprintf("failed to delete TWAP accumulator: %v", err))
	}
	prefix := types.TWAPObservationPoolPrefix(poolID)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		panic(fmt.Sprintf("failed to iterate TWAP observations for deletion: %v", err))
	}
	var keys [][]byte
	for ; iter.Valid(); iter.Next() {
		keys = append(keys, append([]byte(nil), iter.Key()...))
	}
	iter.Close()
	for _, key := range keys {
		if err := store.Delete(key); err != nil {
			panic(fmt.Sprintf("failed to delete TWAP observation: %v", err))
		}
	}
	if err := store.Delete(types.TWAPGarbageCollectionKey(poolID)); err != nil {
		panic(fmt.Sprintf("failed to delete TWAP garbage-collection marker: %v", err))
	}
}

// ScheduleTWAPHistoryDeletion makes a closed pool's oracle state immediately
// unusable while deferring the potentially large observation-key deletion to
// bounded BeginBlock work. A max-retention final LP exit therefore cannot
// exceed the transaction gas ceiling merely because the pool is old.
func (k Keeper) ScheduleTWAPHistoryDeletion(ctx sdk.Context, poolID string) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Delete(types.TWAPKey(poolID)); err != nil {
		panic(fmt.Sprintf("failed to delete TWAP accumulator: %v", err))
	}
	if err := store.Set(types.TWAPGarbageCollectionKey(poolID), []byte{1}); err != nil {
		panic(fmt.Sprintf("failed to schedule TWAP history deletion: %v", err))
	}
}

func (k Keeper) IsTWAPHistoryDeletionScheduled(ctx sdk.Context, poolID string) bool {
	store := k.storeService.OpenKVStore(ctx)
	value, err := store.Get(types.TWAPGarbageCollectionKey(poolID))
	if err != nil {
		panic(fmt.Sprintf("failed to read TWAP garbage-collection marker: %v", err))
	}
	return value != nil
}

func (k Keeper) IterateTWAPGarbageCollection(ctx sdk.Context, cb func(string) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(
		types.TWAPGarbageCollectPrefix,
		prefixEndBytes(types.TWAPGarbageCollectPrefix),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to iterate TWAP garbage-collection queue: %v", err))
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) <= len(types.TWAPGarbageCollectPrefix) {
			panic(fmt.Sprintf("malformed TWAP garbage-collection key %X", key))
		}
		poolID := string(key[len(types.TWAPGarbageCollectPrefix):])
		if cb(poolID) {
			break
		}
	}
}

func (k Keeper) DeleteSecondaryIndexes(ctx sdk.Context) {
	store := k.storeService.OpenKVStore(ctx)
	for _, prefix := range [][]byte{types.DenomIndexPrefix, types.OpenPoolIndexPrefix} {
		iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
		if err != nil {
			panic(fmt.Sprintf("failed to iterate secondary indexes: %v", err))
		}
		var keys [][]byte
		for ; iter.Valid(); iter.Next() {
			keys = append(keys, append([]byte(nil), iter.Key()...))
		}
		iter.Close()
		for _, key := range keys {
			if err := store.Delete(key); err != nil {
				panic(fmt.Sprintf("failed to delete secondary index: %v", err))
			}
		}
	}
}
