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

// AlignmentBankSupplyEnabled exposes the keeper's sole marker reader to the
// alignment entry point. No marker is read by a selected sensor call.
func (a *AlignmentVestingRewardsAdapter) AlignmentBankSupplyEnabled(ctx context.Context) (bool, error) {
	return a.k.AlignmentBankSupplyEnabled(ctx)
}

var _ alignmenttypes.BankSupplySelector = (*AlignmentVestingRewardsAdapter)(nil)

// GetTotalSupply uses the entry-point selection when present. Standalone reads
// select from the marker themselves; this historical interface has no error
// return, so a marker fault panics rather than masquerading as legacy state.
// EndBlock reads and returns marker faults before reaching this interface.
func (a *AlignmentVestingRewardsAdapter) GetTotalSupply(ctx context.Context) *big.Int {
	selection, selected := alignmenttypes.InputSelectionFromContext(ctx)
	if !selected {
		var err error
		selection.BankSupply, err = a.AlignmentBankSupplyEnabled(ctx)
		if err != nil {
			panic(err)
		}
	}
	if !selection.BankSupply {
		return a.getTotalSupplyLegacy(ctx)
	}
	return a.getTotalSupplyFixed(ctx)
}

// getTotalSupplyFixed reads current uzrn bank supply, not the cumulative mint
// ledger. An unwired bank retains the neutral zero-denominator behavior.
func (a *AlignmentVestingRewardsAdapter) getTotalSupplyFixed(ctx context.Context) *big.Int {
	if a.k.bankKeeper == nil {
		return new(big.Int)
	}
	return a.k.bankKeeper.GetSupply(ctx, "uzrn").Amount.BigInt()
}
