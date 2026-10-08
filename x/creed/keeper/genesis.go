package keeper

import (
	"context"
	"slices"

	"github.com/zerone-chain/zerone/x/creed/types"
)

// InitGenesis restores params, earlier pins, the current pin and council seats.
// GenesisState.Validate owns structural validation; this storage routine does
// not revalidate. Imported histories may have gaps between version numbers.
func (k Keeper) InitGenesis(ctx context.Context, gs *types.GenesisState) {
	if gs == nil {
		return
	}
	if gs.Params != nil {
		_ = k.SetParams(ctx, gs.Params)
	}
	// History first (older versions), then the genesis pin (current).
	for _, p := range gs.History {
		if p == nil {
			continue
		}
		_ = k.SetPin(ctx, p)
	}
	if gs.GenesisPin != nil {
		_ = k.SetPin(ctx, gs.GenesisPin)
	}
	// Creed Council seats. Genesis-installed members carry an
	// empty AdmittedViaLip (no LIP precedes genesis); their
	// admitted_at_height defaults to 0 (the genesis block) unless
	// the genesis file specifies otherwise.
	for _, m := range gs.CouncilMembers {
		if m == nil {
			continue
		}
		_ = k.SetCouncilMember(ctx, m)
	}
}

// ExportGenesis dumps the current pinned state. Pin history is
// included in ascending version order; the highest version is the
// current canonical pin. Council membership exports in
// undefined order (the consumer should sort if order matters).
func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	params := k.GetParams(ctx)
	gs := &types.GenesisState{
		Params: params,
	}
	// Council members.
	k.IterateCouncilMembers(ctx, func(m *types.CreedCouncilMember) bool {
		gs.CouncilMembers = append(gs.CouncilMembers, m)
		return false
	})
	k.IteratePinsDescending(ctx, func(p *types.PinnedCreed) bool {
		if gs.GenesisPin == nil {
			gs.GenesisPin = p
		} else {
			gs.History = append(gs.History, p)
		}
		return false
	})
	slices.Reverse(gs.History)
	return gs
}
