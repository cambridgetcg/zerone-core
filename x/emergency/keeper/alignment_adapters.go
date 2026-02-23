package keeper

import (
	"context"

	alignmenttypes "github.com/zerone-chain/zerone/x/alignment/types"
)

// AlignmentEmergencyAdapter wraps the emergency Keeper to satisfy
// alignmenttypes.EmergencyKeeper interface.
type AlignmentEmergencyAdapter struct {
	k Keeper
}

// NewAlignmentEmergencyAdapter returns an adapter for the alignment module.
func NewAlignmentEmergencyAdapter(k Keeper) *AlignmentEmergencyAdapter {
	return &AlignmentEmergencyAdapter{k: k}
}

// Ensure compile-time interface compliance.
var _ alignmenttypes.EmergencyKeeper = (*AlignmentEmergencyAdapter)(nil)

// IsHalted returns true if the chain is in emergency halt.
func (a *AlignmentEmergencyAdapter) IsHalted(ctx context.Context) bool {
	return a.k.IsHalted(ctx)
}
