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
// Alignment uses this for governance participation and the capture-risk
// denominator; proposed and retired domains must not inflate either signal.
// Keep GetAllDomains unfiltered for queries and genesis export.
func (a *AlignmentOntologyAdapter) GetDomainCount(ctx context.Context) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	domains := a.k.GetAllDomains(sdkCtx)
	var count uint64
	for _, domain := range domains {
		if domain.Status == "active" {
			count++
		}
	}
	return count
}
