package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	apkeeper "github.com/zerone-chain/zerone/x/autopoiesis/keeper"
	aptypes "github.com/zerone-chain/zerone/x/autopoiesis/types"
	vestingtypes "github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

// AutopoiesisVestingAdapter wraps the autopoiesis Keeper to satisfy
// vestingtypes.AutopoiesisKeeper (sdk.Context, returns uint64 without error).
type AutopoiesisVestingAdapter struct {
	k apkeeper.Keeper
}

// NewAutopoiesisVestingAdapter returns an adapter for the vesting_rewards module.
func NewAutopoiesisVestingAdapter(k apkeeper.Keeper) *AutopoiesisVestingAdapter {
	return &AutopoiesisVestingAdapter{k: k}
}

// Compile-time interface check.
var _ vestingtypes.AutopoiesisKeeper = (*AutopoiesisVestingAdapter)(nil)

// GetMultiplier satisfies vestingtypes.AutopoiesisKeeper.
// Bridges sdk.Context → context.Context and drops the error.
func (a *AutopoiesisVestingAdapter) GetMultiplier(ctx sdk.Context, path string) uint64 {
	val, err := a.k.GetMultiplier(ctx, path)
	if err != nil {
		return aptypes.BPSScale // default 1.0x
	}
	return val
}
