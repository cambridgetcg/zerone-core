package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	alignmenttypes "github.com/zerone-chain/zerone/x/alignment/types"
)

// AlignmentOntologyAdapter wraps the ontology Keeper to satisfy
// alignmenttypes.OntologyKeeper interface.
type AlignmentOntologyAdapter struct {
	k Keeper
}

// NewAlignmentOntologyAdapter returns an adapter for the alignment module.
func NewAlignmentOntologyAdapter(k Keeper) *AlignmentOntologyAdapter {
	return &AlignmentOntologyAdapter{k: k}
}

// Ensure compile-time interface compliance.
var _ alignmenttypes.OntologyKeeper = (*AlignmentOntologyAdapter)(nil)

// GetDomainCount returns the number of active domains.
func (a *AlignmentOntologyAdapter) GetDomainCount(ctx context.Context) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	domains := a.k.GetAllDomains(sdkCtx)
	return uint64(len(domains))
}
