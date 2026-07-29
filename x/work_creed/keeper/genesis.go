package keeper

import (
	"context"

	"github.com/zerone-chain/zerone/x/work_creed/types"
)

// InitGenesis writes only pins explicitly present in genesis. The default and
// published zerone-1 state are empty. tools/ceremony-inject can prepare pins
// for a future genesis, but no post-genesis Msg exists in Phase 0.
func (k Keeper) InitGenesis(ctx context.Context, gs *types.GenesisState) {
	for _, p := range gs.PinnedSubCreeds {
		if p == nil {
			continue
		}
		if err := k.SetSubCreedPin(ctx, p); err != nil {
			panic(err)
		}
	}
}

// ExportGenesis dumps the stored pin per phase. Phase 0 has no amendment
// message or history query.
func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	gs := types.DefaultGenesis()
	k.IterateSubCreedPins(ctx, func(p *types.PinnedSubCreed) bool {
		gs.PinnedSubCreeds = append(gs.PinnedSubCreeds, p)
		return false
	})
	return gs
}
