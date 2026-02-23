package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	billingtypes "github.com/zerone-chain/zerone/x/billing/types"
)

// ResearchFundDepositorAdapter wraps VestingRewards Keeper to satisfy
// billingtypes.ResearchFundDepositor interface.
type ResearchFundDepositorAdapter struct {
	k Keeper
}

// NewResearchFundDepositorAdapter returns an adapter that bridges the vesting rewards
// keeper to the billing module's expected ResearchFundDepositor interface.
func NewResearchFundDepositorAdapter(k Keeper) *ResearchFundDepositorAdapter {
	return &ResearchFundDepositorAdapter{k: k}
}

// Ensure compile-time interface compliance.
var _ billingtypes.ResearchFundDepositor = (*ResearchFundDepositorAdapter)(nil)

func (a *ResearchFundDepositorAdapter) DepositToResearchFund(ctx context.Context, sourceModule string, amount sdk.Coins) error {
	return a.k.DepositToResearchFund(sdk.UnwrapSDKContext(ctx), sourceModule, amount)
}
