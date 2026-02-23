package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	qualificationtypes "github.com/zerone-chain/zerone/x/qualification/types"
)

// QualificationStakingKeeperAdapter wraps the staking Keeper to satisfy
// qualificationtypes.StakingKeeper interface.
type QualificationStakingKeeperAdapter struct {
	k Keeper
}

// NewQualificationStakingKeeperAdapter returns an adapter that bridges the staking keeper
// to the qualification module's expected interface.
func NewQualificationStakingKeeperAdapter(k Keeper) *QualificationStakingKeeperAdapter {
	return &QualificationStakingKeeperAdapter{k: k}
}

// Ensure compile-time interface compliance.
var _ qualificationtypes.StakingKeeper = (*QualificationStakingKeeperAdapter)(nil)

// IsValidator checks if the given address is a registered validator.
func (a *QualificationStakingKeeperAdapter) IsValidator(ctx context.Context, addr string) bool {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	_, found := a.k.GetValidator(sdkCtx, addr)
	return found
}
