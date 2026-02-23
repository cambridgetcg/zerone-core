package keeper

import (
	"context"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	alignmenttypes "github.com/zerone-chain/zerone/x/alignment/types"
)

// AlignmentVestingRewardsAdapter wraps the vesting_rewards Keeper to satisfy
// alignmenttypes.VestingRewardsKeeper interface.
type AlignmentVestingRewardsAdapter struct {
	k Keeper
}

// NewAlignmentVestingRewardsAdapter returns an adapter for the alignment module.
func NewAlignmentVestingRewardsAdapter(k Keeper) *AlignmentVestingRewardsAdapter {
	return &AlignmentVestingRewardsAdapter{k: k}
}

// Ensure compile-time interface compliance.
var _ alignmenttypes.VestingRewardsKeeper = (*AlignmentVestingRewardsAdapter)(nil)

// GetTotalSupply returns the total uzrn token supply via the bank keeper.
func (a *AlignmentVestingRewardsAdapter) GetTotalSupply(ctx context.Context) *big.Int {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	_ = sdkCtx
	// Access the bank keeper through vesting_rewards to get total supply.
	// The vesting_rewards keeper wraps bankKeeper.GetSupply internally.
	// Since bankKeeper is private, we use GetTotalMinted as a proxy.
	totalMinted := a.k.GetTotalMinted(sdk.UnwrapSDKContext(ctx))
	if totalMinted == nil {
		return new(big.Int)
	}
	return totalMinted
}
