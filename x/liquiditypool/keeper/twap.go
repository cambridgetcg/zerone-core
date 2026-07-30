package keeper

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/liquiditypool/types"
)

// twapScale is the precision scale for TWAP accumulators (1e12).
var twapScale = new(big.Int).Exp(big.NewInt(10), big.NewInt(12), nil)

const (
	// These two independent bounds keep stale-history cleanup a small,
	// deterministic fraction of block work even after many pools close.
	twapGCDeletesPerBlock = 64
	twapGCMarkersPerBlock = 64
)

// ProcessTWAPGarbageCollection deletes a bounded number of checkpoints for
// closed pools. Accumulators are removed synchronously at close, so queued
// checkpoints are unreachable historical storage, not live oracle state.
func (k Keeper) ProcessTWAPGarbageCollection(ctx sdk.Context) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(
		types.TWAPGarbageCollectPrefix,
		prefixEndBytes(types.TWAPGarbageCollectPrefix),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to iterate TWAP garbage-collection queue: %v", err))
	}
	type marker struct {
		key    []byte
		poolID string
	}
	markers := make([]marker, 0, twapGCMarkersPerBlock)
	for ; iter.Valid() && len(markers) < twapGCMarkersPerBlock; iter.Next() {
		key := append([]byte(nil), iter.Key()...)
		if len(key) <= len(types.TWAPGarbageCollectPrefix) {
			iter.Close()
			panic(fmt.Sprintf("malformed TWAP garbage-collection key %X", key))
		}
		markers = append(markers, marker{
			key:    key,
			poolID: string(key[len(types.TWAPGarbageCollectPrefix):]),
		})
	}
	iter.Close()

	remainingDeletes := twapGCDeletesPerBlock
	for _, queued := range markers {
		prefix := types.TWAPObservationPoolPrefix(queued.poolID)
		observationIter, err := store.Iterator(prefix, prefixEndBytes(prefix))
		if err != nil {
			panic(fmt.Sprintf("failed to iterate queued TWAP observations: %v", err))
		}
		keys := make([][]byte, 0, remainingDeletes)
		for ; observationIter.Valid() && len(keys) < remainingDeletes; observationIter.Next() {
			keys = append(keys, append([]byte(nil), observationIter.Key()...))
		}
		hasMore := observationIter.Valid()
		observationIter.Close()
		for _, key := range keys {
			if err := store.Delete(key); err != nil {
				panic(fmt.Sprintf("failed to garbage-collect TWAP observation: %v", err))
			}
		}
		remainingDeletes -= len(keys)
		if !hasMore {
			if err := store.Delete(queued.key); err != nil {
				panic(fmt.Sprintf("failed to complete TWAP garbage collection: %v", err))
			}
		}
		if remainingDeletes == 0 {
			break
		}
	}
}

// UpdateTWAPAccumulator advances a pool's cumulative prices and persists one
// bounded observation for the current block. BeginBlock calls it through the
// finite open-pool index, so closed tombstones add no recurring work.
func (k Keeper) UpdateTWAPAccumulator(ctx sdk.Context, pool *types.Pool) {
	if pool.Status == types.PoolStatus_POOL_STATUS_CLOSED {
		return
	}
	if ctx.BlockHeight() < 0 {
		panic("liquiditypool cannot update TWAP at a negative block height")
	}
	currentBlock := uint64(ctx.BlockHeight())

	acc, found := k.GetTWAPAccumulator(ctx, pool.PoolId)
	if !found {
		acc = &types.TWAPAccumulator{
			PoolId:       pool.PoolId,
			LastBlock:    currentBlock,
			StartBlock:   currentBlock,
			CumPriceAToB: "0",
			CumPriceBToA: "0",
		}
		k.SetTWAPAccumulator(ctx, acc)
		k.SetTWAPObservation(ctx, &types.TWAPObservation{
			PoolId:       pool.PoolId,
			BlockHeight:  currentBlock,
			CumPriceAToB: "0",
			CumPriceBToA: "0",
		})
		return
	}
	if currentBlock <= acc.LastBlock {
		return
	}

	reserveA, err := types.ParsePositiveAmount(pool.ReserveA)
	if err != nil {
		panic(fmt.Sprintf("invalid reserve_a for TWAP pool %s: %v", pool.PoolId, err))
	}
	reserveB, err := types.ParsePositiveAmount(pool.ReserveB)
	if err != nil {
		panic(fmt.Sprintf("invalid reserve_b for TWAP pool %s: %v", pool.PoolId, err))
	}
	cumAtoB, err := types.ParseCumulativeAmount(acc.CumPriceAToB)
	if err != nil {
		panic(fmt.Sprintf("invalid cumulative A/B price for pool %s: %v", pool.PoolId, err))
	}
	cumBtoA, err := types.ParseCumulativeAmount(acc.CumPriceBToA)
	if err != nil {
		panic(fmt.Sprintf("invalid cumulative B/A price for pool %s: %v", pool.PoolId, err))
	}

	blocksDelta := currentBlock - acc.LastBlock
	priceAtoB := new(big.Int).Mul(reserveB, twapScale)
	priceAtoB.Div(priceAtoB, reserveA)
	priceAtoB.Mul(priceAtoB, new(big.Int).SetUint64(blocksDelta))
	cumAtoB.Add(cumAtoB, priceAtoB)

	priceBtoA := new(big.Int).Mul(reserveA, twapScale)
	priceBtoA.Div(priceBtoA, reserveB)
	priceBtoA.Mul(priceBtoA, new(big.Int).SetUint64(blocksDelta))
	cumBtoA.Add(cumBtoA, priceBtoA)

	acc.CumPriceAToB = cumAtoB.String()
	acc.CumPriceBToA = cumBtoA.String()
	acc.LastBlock = currentBlock
	k.SetTWAPAccumulator(ctx, acc)
	k.SetTWAPObservation(ctx, &types.TWAPObservation{
		PoolId:       pool.PoolId,
		BlockHeight:  currentBlock,
		CumPriceAToB: acc.CumPriceAToB,
		CumPriceBToA: acc.CumPriceBToA,
	})
	k.pruneTWAPObservations(ctx, pool.PoolId, currentBlock, k.GetParams(ctx).TwapWindowBlocks)
}

func (k Keeper) pruneTWAPObservations(ctx sdk.Context, poolID string, currentBlock, retention uint64) {
	if currentBlock <= retention {
		return
	}
	cutoff := currentBlock - retention
	store := k.storeService.OpenKVStore(ctx)
	prefix := types.TWAPObservationPoolPrefix(poolID)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		panic(fmt.Sprintf("failed to iterate TWAP observations for pruning: %v", err))
	}
	var keys [][]byte
	for ; iter.Valid(); iter.Next() {
		height, ok := types.TWAPObservationHeight(iter.Key())
		if !ok {
			badKey := append([]byte(nil), iter.Key()...)
			iter.Close()
			panic(fmt.Sprintf("malformed TWAP observation key %X", badKey))
		}
		if height >= cutoff {
			break
		}
		keys = append(keys, append([]byte(nil), iter.Key()...))
	}
	iter.Close()
	for _, key := range keys {
		if err := store.Delete(key); err != nil {
			panic(fmt.Sprintf("failed to prune TWAP observation: %v", err))
		}
	}
}

// GetTWAP returns the block-weighted arithmetic average over the requested
// retained window. A zero window selects Params.twap_window_blocks. If the
// pool is younger than the request, window_used truthfully reports the shorter
// available span.
func (k Keeper) GetTWAP(ctx sdk.Context, poolID, baseDenom string, window uint64) (*big.Int, uint64, error) {
	pool, found := k.GetPool(ctx, poolID)
	if !found {
		return nil, 0, types.ErrPoolNotFound
	}
	if pool.Status == types.PoolStatus_POOL_STATUS_CLOSED {
		return nil, 0, types.ErrPoolNotActive
	}
	if baseDenom != pool.DenomA && baseDenom != pool.DenomB {
		return nil, 0, types.ErrDenomNotInPool
	}

	retention := k.GetParams(ctx).TwapWindowBlocks
	if window == 0 {
		window = retention
	}
	if window > retention {
		return nil, 0, types.ErrTWAPWindowUnavailable.Wrapf(
			"requested %d blocks exceeds retained %d", window, retention,
		)
	}

	acc, found := k.GetTWAPAccumulator(ctx, poolID)
	if !found {
		return k.getSpotPrice(pool, baseDenom)
	}
	currentCum, err := cumulativeForBase(acc.CumPriceAToB, acc.CumPriceBToA, baseDenom == pool.DenomA)
	if err != nil {
		return nil, 0, types.ErrInvalidPoolState.Wrap(err.Error())
	}
	target := uint64(0)
	if acc.LastBlock > window {
		target = acc.LastBlock - window
	}

	var selected *types.TWAPObservation
	k.IterateTWAPObservations(ctx, poolID, func(observation *types.TWAPObservation) bool {
		if observation.BlockHeight >= target {
			selected = proto.Clone(observation).(*types.TWAPObservation)
			return true
		}
		return false
	})
	if selected == nil || selected.BlockHeight >= acc.LastBlock {
		return k.getSpotPrice(pool, baseDenom)
	}

	previousCum, err := cumulativeForBase(
		selected.CumPriceAToB,
		selected.CumPriceBToA,
		baseDenom == pool.DenomA,
	)
	if err != nil {
		return nil, 0, types.ErrInvalidPoolState.Wrap(err.Error())
	}
	if currentCum.Cmp(previousCum) < 0 {
		return nil, 0, types.ErrInvalidPoolState.Wrap("TWAP cumulative price decreased")
	}
	span := acc.LastBlock - selected.BlockHeight
	delta := new(big.Int).Sub(currentCum, previousCum)
	numerator := new(big.Int).Mul(delta, big.NewInt(1_000_000))
	divisor := new(big.Int).Mul(new(big.Int).SetUint64(span), twapScale)
	return numerator.Div(numerator, divisor), span, nil
}

func cumulativeForBase(aToB, bToA string, baseIsA bool) (*big.Int, error) {
	if baseIsA {
		return types.ParseCumulativeAmount(aToB)
	}
	return types.ParseCumulativeAmount(bToA)
}

// getSpotPrice returns spot price of baseDenom in quote terms, scaled by 1e6.
func (k Keeper) getSpotPrice(pool *types.Pool, baseDenom string) (*big.Int, uint64, error) {
	if baseDenom != pool.DenomA && baseDenom != pool.DenomB {
		return nil, 0, types.ErrDenomNotInPool
	}
	reserveA, err := types.ParsePositiveAmount(pool.ReserveA)
	if err != nil {
		return nil, 0, types.ErrInvalidPoolState.Wrapf("reserve_a: %v", err)
	}
	reserveB, err := types.ParsePositiveAmount(pool.ReserveB)
	if err != nil {
		return nil, 0, types.ErrInvalidPoolState.Wrapf("reserve_b: %v", err)
	}

	scale := big.NewInt(1_000_000)
	if baseDenom == pool.DenomA {
		price := new(big.Int).Mul(reserveB, scale)
		return price.Div(price, reserveA), 0, nil
	}
	price := new(big.Int).Mul(reserveA, scale)
	return price.Div(price, reserveB), 0, nil
}

func (k Keeper) GetSpotPrice(ctx sdk.Context, poolID, baseDenom string) (*big.Int, error) {
	pool, found := k.GetPool(ctx, poolID)
	if !found {
		return nil, types.ErrPoolNotFound
	}
	if pool.Status == types.PoolStatus_POOL_STATUS_CLOSED {
		return nil, types.ErrPoolNotActive
	}
	price, _, err := k.getSpotPrice(pool, baseDenom)
	return price, err
}
