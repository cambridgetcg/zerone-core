package keeper

import (
	"context"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	alignmenttypes "github.com/zerone-chain/zerone/x/alignment/types"
)

// AlignmentStakingAdapter wraps the staking Keeper to satisfy
// alignmenttypes.StakingKeeper interface.
type AlignmentStakingAdapter struct {
	k Keeper
}

// NewAlignmentStakingAdapter returns an adapter for the alignment module.
func NewAlignmentStakingAdapter(k Keeper) *AlignmentStakingAdapter {
	return &AlignmentStakingAdapter{k: k}
}

// Ensure compile-time interface compliance.
var _ alignmenttypes.StakingKeeper = (*AlignmentStakingAdapter)(nil)

// GetTotalStaked returns the total bonded stake.
func (a *AlignmentStakingAdapter) GetTotalStaked(ctx context.Context) *big.Int {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return a.k.GetTotalBondedStake(sdkCtx)
}

// GetActiveValidatorCount returns the number of active validators.
func (a *AlignmentStakingAdapter) GetActiveValidatorCount(ctx context.Context) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	validators := a.k.GetActiveValidatorSet(sdkCtx)
	return uint64(len(validators))
}

// GetTargetValidatorCount returns the target number of validators.
// Hardcoded to 111 (Zerone's target validator set size).
func (a *AlignmentStakingAdapter) GetTargetValidatorCount(_ context.Context) uint64 {
	return 111
}
