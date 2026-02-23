package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	evidencemgmttypes "github.com/zerone-chain/zerone/x/evidence_mgmt/types"
)

// EvidenceMgmtStakingAdapter wraps the staking Keeper to satisfy
// evidencemgmttypes.StakingKeeper interface.
type EvidenceMgmtStakingAdapter struct {
	k Keeper
}

// NewEvidenceMgmtStakingAdapter returns an adapter for the evidence_mgmt module.
func NewEvidenceMgmtStakingAdapter(k Keeper) *EvidenceMgmtStakingAdapter {
	return &EvidenceMgmtStakingAdapter{k: k}
}

// Ensure compile-time interface compliance.
var _ evidencemgmttypes.StakingKeeper = (*EvidenceMgmtStakingAdapter)(nil)

// GetValidatorTier returns the validator tier for the given address.
// Returns tier 0 if not a validator.
func (a *EvidenceMgmtStakingAdapter) GetValidatorTier(ctx context.Context, addr string) (uint32, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	val, found := a.k.GetValidator(sdkCtx, addr)
	if !found {
		return 0, nil
	}
	return uint32(val.Tier), nil
}
