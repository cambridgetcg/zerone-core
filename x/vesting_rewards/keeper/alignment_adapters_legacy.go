package keeper

import (
	"context"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// getTotalSupplyLegacy preserves the GetTotalSupply body verbatim from
// 77521a0d (before #122). Do not route it through the bank-supply repair.
func (a *AlignmentVestingRewardsAdapter) getTotalSupplyLegacy(ctx context.Context) *big.Int {
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
