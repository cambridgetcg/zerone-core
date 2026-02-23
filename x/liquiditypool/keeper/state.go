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
	bz, err := proto.Marshal(pool)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal pool: %v", err))
	}
	if err := store.Set(types.PoolKey(pool.PoolId), bz); err != nil {
		panic(fmt.Sprintf("failed to set pool: %v", err))
	}
	// Update denom pair index
	indexKey := types.DenomPairKey(pool.DenomA, pool.DenomB)
	if err := store.Set(indexKey, []byte(pool.PoolId)); err != nil {
		panic(fmt.Sprintf("failed to set denom index: %v", err))
	}
}

func (k Keeper) GetPool(ctx sdk.Context, poolId string) (*types.Pool, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.PoolKey(poolId))
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
	indexKey := types.DenomPairKey(denomA, denomB)
	bz, err := store.Get(indexKey)
	if err != nil || bz == nil {
		return nil
	}
	poolId := string(bz)
	pool, found := k.GetPool(ctx, poolId)
	if !found {
		return nil
	}
	return pool
}

func (k Keeper) DeletePool(ctx sdk.Context, pool *types.Pool) {
	store := k.storeService.OpenKVStore(ctx)
	_ = store.Delete(types.PoolKey(pool.PoolId))
	_ = store.Delete(types.DenomPairKey(pool.DenomA, pool.DenomB))
}

func (k Keeper) IteratePools(ctx sdk.Context, cb func(*types.Pool) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.PoolKeyPrefix, prefixEndBytes(types.PoolKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var pool types.Pool
		if err := proto.Unmarshal(iter.Value(), &pool); err != nil {
			continue
		}
		if cb(&pool) {
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

func (k Keeper) GetTWAPAccumulator(ctx sdk.Context, poolId string) (*types.TWAPAccumulator, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.TWAPKey(poolId))
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
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var acc types.TWAPAccumulator
		if err := proto.Unmarshal(iter.Value(), &acc); err != nil {
			continue
		}
		if cb(&acc) {
			break
		}
	}
}
