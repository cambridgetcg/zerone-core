package keeper

import (
	"context"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/liquiditypool/types"
)

// LiquidityPoolKeeperAdapter adapts the liquiditypool Keeper to the billing
// module's LiquidityPoolKeeper interface.
type LiquidityPoolKeeperAdapter struct {
	keeper Keeper
}

// NewLiquidityPoolKeeperAdapter wraps a Keeper for cross-module use.
func NewLiquidityPoolKeeperAdapter(k Keeper) *LiquidityPoolKeeperAdapter {
	return &LiquidityPoolKeeperAdapter{keeper: k}
}

// GetTWAP returns the TWAP for a base denom (e.g., "uzrn") against any quote denom
// found in an active pool. Returns scaled by 1e6.
func (a *LiquidityPoolKeeperAdapter) GetTWAP(goCtx context.Context, baseDenom string, windowBlocks uint64) (*big.Int, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Find a pool containing baseDenom by iterating all pools
	var targetPool *types.Pool
	a.keeper.IteratePools(ctx, func(p *types.Pool) bool {
		if p.DenomA == baseDenom || p.DenomB == baseDenom {
			targetPool = p
			return true // found
		}
		return false
	})

	if targetPool == nil {
		return nil, types.ErrNoPool
	}

	twap, _, err := a.keeper.GetTWAP(ctx, targetPool.PoolId, baseDenom, windowBlocks)
	return twap, err
}

// GetLastPriceUpdateHeight returns the most recent block at which any TWAP
// accumulator was updated.
func (a *LiquidityPoolKeeperAdapter) GetLastPriceUpdateHeight(goCtx context.Context) uint64 {
	ctx := sdk.UnwrapSDKContext(goCtx)
	var maxBlock uint64
	a.keeper.IterateTWAPAccumulators(ctx, func(acc *types.TWAPAccumulator) bool {
		if acc.LastBlock > maxBlock {
			maxBlock = acc.LastBlock
		}
		return false
	})
	return maxBlock
}
