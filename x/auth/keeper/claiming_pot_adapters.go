package keeper

import (
	"context"

	claimingpottypes "github.com/zerone-chain/zerone/x/claiming_pot/types"
)

// ClaimingPotAuthAdapter wraps the zerone auth Keeper to satisfy
// claimingpottypes.AuthKeeper interface.
type ClaimingPotAuthAdapter struct {
	k Keeper
}

// NewClaimingPotAuthAdapter returns an adapter for the claiming_pot module.
func NewClaimingPotAuthAdapter(k Keeper) *ClaimingPotAuthAdapter {
	return &ClaimingPotAuthAdapter{k: k}
}

// Ensure compile-time interface compliance.
var _ claimingpottypes.AuthKeeper = (*ClaimingPotAuthAdapter)(nil)

// GetRegistrationBlock returns the block at which the address was registered.
// Returns 0 for now — will be properly wired when account metadata tracking is available.
func (a *ClaimingPotAuthAdapter) GetRegistrationBlock(_ context.Context, _ string) (uint64, error) {
	return 0, nil
}
