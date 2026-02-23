package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	disputestypes "github.com/zerone-chain/zerone/x/disputes/types"
)

// DisputesStakingKeeperAdapter wraps the staking Keeper to satisfy
// disputestypes.StakingKeeper interface.
type DisputesStakingKeeperAdapter struct {
	k Keeper
}

// NewDisputesStakingKeeperAdapter returns an adapter for the disputes module.
func NewDisputesStakingKeeperAdapter(k Keeper) *DisputesStakingKeeperAdapter {
	return &DisputesStakingKeeperAdapter{k: k}
}

// Ensure compile-time interface compliance.
var _ disputestypes.StakingKeeper = (*DisputesStakingKeeperAdapter)(nil)

// GetQualifiedValidators returns active validator addresses suitable for arbiter duty.
func (a *DisputesStakingKeeperAdapter) GetQualifiedValidators(ctx context.Context, domain string, count int) ([]string, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	activeVals := a.k.GetActiveValidatorSet(sdkCtx)

	var addrs []string
	for _, val := range activeVals {
		addrs = append(addrs, val.OperatorAddress)
		if len(addrs) >= count {
			break
		}
	}
	return addrs, nil
}
