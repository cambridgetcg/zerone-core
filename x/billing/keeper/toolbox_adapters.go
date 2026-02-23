package keeper

import (
	"context"

	toolboxtypes "github.com/zerone-chain/zerone/x/toolbox/types"
)

// ToolboxBillingAdapter wraps the billing Keeper to satisfy
// toolboxtypes.BillingKeeper interface.
type ToolboxBillingAdapter struct {
	k Keeper
}

// NewToolboxBillingAdapter returns an adapter that bridges the billing keeper
// to the toolbox module's expected BillingKeeper interface.
func NewToolboxBillingAdapter(k Keeper) *ToolboxBillingAdapter {
	return &ToolboxBillingAdapter{k: k}
}

// Ensure compile-time interface compliance.
var _ toolboxtypes.BillingKeeper = (*ToolboxBillingAdapter)(nil)

// GetZRNPriceUSD returns the current ZRN price in micro-USD.
// The billing keeper's GetZRNPriceUSD returns uint64 (no error),
// so we wrap it to match the toolbox interface (uint64, error).
func (a *ToolboxBillingAdapter) GetZRNPriceUSD(ctx context.Context) (uint64, error) {
	return a.k.GetZRNPriceUSD(ctx), nil
}
