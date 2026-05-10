package keeper

import (
	"context"

	"github.com/zerone-chain/zerone/x/work_creed/types"
)

// InitGenesis writes all pinned sub-creeds from genesis state into the
// store. Caller (app.go genesis populator at chain init) is responsible
// for deriving the inception pins from the build-time
// CanonicalSubCreeds + .sub-creed-hashes; Phase 1+ may also call
// SetSubCreedPin via msg handlers post-genesis.
func (k Keeper) InitGenesis(ctx context.Context, gs types.GenesisState) {
	for _, p := range gs.PinnedSubCreeds {
		if p == nil {
			continue
		}
		if err := k.SetSubCreedPin(ctx, p); err != nil {
			panic(err)
		}
	}
}

// ExportGenesis dumps the latest pin per phase. Phase 0 has only one
// pin per phase by definition; Phase 1+ when versions accumulate, this
// will export only the LATEST pin (history retrieval needs explicit
// queries via grpc_query).
func (k Keeper) ExportGenesis(ctx context.Context) types.GenesisState {
	gs := types.DefaultGenesis()
	k.IterateSubCreedPins(ctx, func(p *types.PinnedSubCreed) bool {
		gs.PinnedSubCreeds = append(gs.PinnedSubCreeds, p)
		return false
	})
	return *gs
}
