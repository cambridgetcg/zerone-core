package keeper

import (
	"context"

	evidencemgmttypes "github.com/zerone-chain/zerone/x/evidence_mgmt/types"
)

// EvidenceMgmtDisputesAdapter wraps the disputes Keeper to satisfy
// evidencemgmttypes.DisputesKeeper interface.
type EvidenceMgmtDisputesAdapter struct {
	k Keeper
}

// NewEvidenceMgmtDisputesAdapter returns an adapter for the evidence_mgmt module.
func NewEvidenceMgmtDisputesAdapter(k Keeper) *EvidenceMgmtDisputesAdapter {
	return &EvidenceMgmtDisputesAdapter{k: k}
}

// Ensure compile-time interface compliance.
var _ evidencemgmttypes.DisputesKeeper = (*EvidenceMgmtDisputesAdapter)(nil)

// CreateDispute creates a dispute via the disputes module.
// Stub implementation — returns nil error for now, as the full bridge requires
// escrow logic that depends on bank keeper wiring.
func (a *EvidenceMgmtDisputesAdapter) CreateDispute(_ context.Context, _, _, _, _ string) (string, error) {
	return "stub-dispute-id", nil
}
