package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	claimingpottypes "github.com/zerone-chain/zerone/x/claiming_pot/types"
)

// ClaimingPotStakingAdapter wraps the staking Keeper to satisfy
// claimingpottypes.StakingKeeper interface.
type ClaimingPotStakingAdapter struct {
	k Keeper
}

// NewClaimingPotStakingAdapter returns an adapter for the claiming_pot module.
func NewClaimingPotStakingAdapter(k Keeper) *ClaimingPotStakingAdapter {
	return &ClaimingPotStakingAdapter{k: k}
}

// Ensure compile-time interface compliance.
var _ claimingpottypes.StakingKeeper = (*ClaimingPotStakingAdapter)(nil)

// GetValidatorTier returns the validator tier for the given address.
func (a *ClaimingPotStakingAdapter) GetValidatorTier(ctx context.Context, addr string) (uint32, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	val, found := a.k.GetValidator(sdkCtx, addr)
	if !found {
		return 0, nil
	}
	return uint32(val.Tier), nil
}
