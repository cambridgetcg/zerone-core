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

// ActiveDomainCountEnabled exposes the keeper's sole marker reader to the
// alignment entry point. No marker is read by a selected sensor call.
func (a *AlignmentOntologyAdapter) ActiveDomainCountEnabled(ctx context.Context) (bool, error) {
	return a.k.ActiveDomainCountEnabled(ctx)
}

var _ alignmenttypes.ActiveDomainCountSelector = (*AlignmentOntologyAdapter)(nil)

// GetDomainCount uses the entry-point selection when present. Standalone reads
// select from the marker themselves; this historical interface has no error
// return, so a marker fault panics rather than masquerading as legacy state.
// EndBlock reads and returns marker faults before reaching this interface.
func (a *AlignmentOntologyAdapter) GetDomainCount(ctx context.Context) uint64 {
	selection, selected := alignmenttypes.InputSelectionFromContext(ctx)
	if !selected {
		var err error
		selection.ActiveDomainCount, err = a.ActiveDomainCountEnabled(ctx)
		if err != nil {
			panic(err)
		}
	}
	if !selection.ActiveDomainCount {
		return a.getDomainCountLegacy(ctx)
	}
	return a.getDomainCountFixed(ctx)
}

// getDomainCountFixed counts exact active status for alignment only. All-status
// query/export readers remain unchanged, including proposed and retired records.
func (a *AlignmentOntologyAdapter) getDomainCountFixed(ctx context.Context) uint64 {
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
