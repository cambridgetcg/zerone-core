package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// InitGenesis imports an explicit accounting inventory atomically. FX7 is
// genesis-only: field presence, not a live activation marker, selects this path.
// An older genesis still imports exactly as before, without inventing balances
// or admission history that it did not carry.
func (k Keeper) InitGenesis(ctx context.Context, gs *types.GenesisState) error {
	if gs.AccountingState == nil {
		return k.initGenesisLegacy(ctx, gs)
	}
	if err := types.ValidateGenesisAccounting(gs); err != nil {
		return err
	}
	// Bank genesis must precede knowledge. Check before even the legacy
	// bootstrap allocation writes, and especially before training-fund top-up.
	if err := k.validateGenesisAccountingBacking(ctx, gs); err != nil {
		return err
	}
	cache, write := sdk.UnwrapSDKContext(ctx).CacheContext()
	if err := k.initGenesisLegacy(cache, gs); err != nil {
		return err
	}
	if err := k.importGenesisAccounting(cache, gs.AccountingState); err != nil {
		return fmt.Errorf("import knowledge accounting: %w", err)
	}
	write()
	return nil
}

// ExportGenesis includes every retained primary accounting key, not a
// reconstruction from bounty creation amounts. It cannot attest that an earlier
// exporter or relaunch never lost history; the carrier must establish lineage.
func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	gs := k.exportGenesisLegacy(ctx)
	if err := k.exportGenesisAccounting(ctx, gs); err != nil {
		// There is no error result in the module's export interface. Accounting
		// inconsistencies are not activation-marker faults; existing marker
		// faults retain their activation.ErrMarkerFault chain in the legacy body.
		panic(fmt.Errorf("export knowledge accounting: %w", err))
	}
	return gs
}
