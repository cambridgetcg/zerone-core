package keeper

import (
	"context"
	"fmt"

	toolboxtypes "github.com/zerone-chain/zerone/x/toolbox/types"
)

// ToolboxHomeAdapter wraps the home Keeper to satisfy
// toolboxtypes.HomeKeeper interface.
type ToolboxHomeAdapter struct {
	k Keeper
}

// NewToolboxHomeAdapter returns an adapter that bridges the home keeper
// to the toolbox module's expected HomeKeeper interface.
func NewToolboxHomeAdapter(k Keeper) *ToolboxHomeAdapter {
	return &ToolboxHomeAdapter{k: k}
}

// Ensure compile-time interface compliance.
var _ toolboxtypes.HomeKeeper = (*ToolboxHomeAdapter)(nil)

// GetHomesByOwner returns all home IDs for an owner.
// The home keeper's GetHomesByOwner returns []string (no error),
// so we wrap it to match the toolbox interface ([]string, error).
func (a *ToolboxHomeAdapter) GetHomesByOwner(ctx context.Context, owner string) ([]string, error) {
	return a.k.GetHomesByOwner(ctx, owner), nil
}

// GetHomeCreatedAtBlock returns the block at which a home was created.
// Bridges via GetHome → .CreatedAtBlock.
func (a *ToolboxHomeAdapter) GetHomeCreatedAtBlock(ctx context.Context, homeID string) (uint64, error) {
	home, found := a.k.GetHome(ctx, homeID)
	if !found {
		return 0, fmt.Errorf("home %s not found", homeID)
	}
	return home.CreatedAtBlock, nil
}

// GetHomeStatus returns the status of a home.
// Bridges via GetHome → .Status.
func (a *ToolboxHomeAdapter) GetHomeStatus(ctx context.Context, homeID string) (string, error) {
	home, found := a.k.GetHome(ctx, homeID)
	if !found {
		return "", fmt.Errorf("home %s not found", homeID)
	}
	return home.Status, nil
}
