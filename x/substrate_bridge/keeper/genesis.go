package keeper

import (
	"context"

	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

func (k Keeper) InitGenesis(ctx context.Context, gs *types.GenesisState) error {
	if gs.Params != nil {
		if err := k.SetParams(ctx, *gs.Params); err != nil {
			return err
		}
	}
	for _, a := range gs.Adapters {
		if err := k.WriteAdapter(ctx, a); err != nil {
			return err
		}
	}
	// A chain born with this code carries no attestation history, so the
	// (empty) source-ref index is already correct and enforcement is armed
	// from genesis. Existing chains that predate this code never re-run
	// InitGenesis; they arm via the substrate-dedupe-v1 upgrade handler,
	// which seeds the index from their history first.
	//
	// NOTE: GenesisState round-trips only Params + Adapters — neither the
	// attestation store nor the source-ref index survives an export/import.
	// A relaunch from exported state therefore starts with an empty index
	// while previously minted ZRN persists in the bank genesis, so a
	// historically-settled (adapter_id, source_id) could mint again on the
	// new chain. Preserving the wall across export requires adding
	// attestations + the index to GenesisState (a proto change tracked as a
	// follow-up); today the module is launched fresh (the zerone-2 relaunch
	// disables it entirely), so this is a documented limitation, not a live
	// hole.
	k.SetDedupeArmed(ctx)
	return nil
}

func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	params := k.GetParams(ctx)
	gs := &types.GenesisState{Params: &params}
	k.IterateAdapters(ctx, func(a *types.AdapterRegistration) bool {
		gs.Adapters = append(gs.Adapters, a)
		return false
	})
	return gs
}
