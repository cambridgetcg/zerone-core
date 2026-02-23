package keeper

import (
	"context"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	aptypes "github.com/zerone-chain/zerone/x/autopoiesis/types"
	apkeeper "github.com/zerone-chain/zerone/x/autopoiesis/keeper"
	stakingtypes "github.com/zerone-chain/zerone/x/staking/types"
)

// AutopoiesisStakingAdapter wraps the autopoiesis Keeper to satisfy
// stakingtypes.AutopoiesisKeeper (returns uint64 without error).
type AutopoiesisStakingAdapter struct {
	k apkeeper.Keeper
}

// NewAutopoiesisStakingAdapter returns an adapter for the staking module.
func NewAutopoiesisStakingAdapter(k apkeeper.Keeper) *AutopoiesisStakingAdapter {
	return &AutopoiesisStakingAdapter{k: k}
}

// Compile-time interface check.
var _ stakingtypes.AutopoiesisKeeper = (*AutopoiesisStakingAdapter)(nil)

// GetMultiplier satisfies stakingtypes.AutopoiesisKeeper.
// Drops the error from the underlying keeper, defaulting to 1.0x on error.
func (a *AutopoiesisStakingAdapter) GetMultiplier(ctx context.Context, path string) uint64 {
	val, err := a.k.GetMultiplier(ctx, path)
	if err != nil {
		return aptypes.BPSScale // default 1.0x
	}
	return val
}

// StakingForAutopoiesisAdapter wraps the staking Keeper to satisfy
// aptypes.StakingKeeper (GetTotalBondedStake, GetActiveValidatorCount).
type StakingForAutopoiesisAdapter struct {
	k Keeper
}

// NewStakingForAutopoiesisAdapter returns an adapter providing staking data
// to the autopoiesis module.
func NewStakingForAutopoiesisAdapter(k Keeper) *StakingForAutopoiesisAdapter {
	return &StakingForAutopoiesisAdapter{k: k}
}

// Compile-time interface check.
var _ aptypes.StakingKeeper = (*StakingForAutopoiesisAdapter)(nil)

// GetTotalBondedStake returns total bonded stake across active validators.
func (a *StakingForAutopoiesisAdapter) GetTotalBondedStake(ctx sdk.Context) *big.Int {
	return a.k.GetTotalBondedStake(ctx)
}

// GetActiveValidatorCount returns the number of active validators.
func (a *StakingForAutopoiesisAdapter) GetActiveValidatorCount(ctx sdk.Context) int {
	return len(a.k.GetActiveValidatorSet(ctx))
}
