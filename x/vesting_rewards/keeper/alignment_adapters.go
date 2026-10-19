package keeper

import (
	"context"
	"math/big"

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

// GetTotalSupply returns current bank supply in uzrn, the custom staking denom.
// The cumulative MintWithCap ledger is not supply: it excludes direct bank and
// genesis minting, and burns do not reduce it. An unwired bank yields zero so
// alignment keeps its neutral zero-denominator behavior, never a ledger proxy.
// This consensus-changing correction requires a scheduled binary swap; it has
// no stored activation marker and does not rewrite historical observations.
func (a *AlignmentVestingRewardsAdapter) GetTotalSupply(ctx context.Context) *big.Int {
	if a.k.bankKeeper == nil {
		return new(big.Int)
	}
	return a.k.bankKeeper.GetSupply(ctx, "uzrn").Amount.BigInt()
}
