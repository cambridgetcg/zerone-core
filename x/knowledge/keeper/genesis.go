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
	// InitChain does not run ValidateGenesis, so check the v3 reviewer balance
	// here, before any write. A genesis without the field passes unchanged.
	if err := types.ValidateReviewerBalanceGenesis(gs); err != nil {
		return err
	}
	// Nor the seat-deposit state (reviewer-entry step B): with the marker
	// off no round may carry seat terms, a returns plan or a seat deposit,
	// and with it on its predecessor markers must be selected. Both import
	// paths below write only after this.
	if err := types.ValidateSeatDepositGenesis(gs); err != nil {
		return err
	}
	if gs.AccountingState == nil {
		if err := k.initGenesisLegacy(ctx, gs); err != nil {
			return err
		}
		return k.initGenesisV3Markers(ctx, gs)
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
	if err := k.initGenesisV3Markers(cache, gs); err != nil {
		return err
	}
	write()
	return nil
}

// ExportGenesis includes every retained primary accounting key, not a
// reconstruction from bounty creation amounts. It cannot attest that an earlier
// exporter or relaunch never lost history; the carrier must establish lineage.
func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	gs := k.exportGenesisLegacy(ctx)
	k.exportGenesisV3Markers(ctx, gs)
	if err := k.exportGenesisAccounting(ctx, gs); err != nil {
		// There is no error result in the module's export interface. Accounting
		// inconsistencies are not activation-marker faults; existing marker
		// faults retain their activation.ErrMarkerFault chain in the legacy body.
		panic(fmt.Errorf("export knowledge accounting: %w", err))
	}
	return gs
}

// initGenesisV3Markers writes the consensus-repairs-v3 selections. It sits
// outside initGenesisLegacy, whose body stays pinned to the v2 foundation, and
// runs on both import paths. Only an explicit true writes a marker; absence
// keeps legacy semantics.
func (k Keeper) initGenesisV3Markers(ctx context.Context, gs *types.GenesisState) error {
	if gs.DoctrineIdentityEnabled {
		if err := k.EnableDoctrineIdentity(ctx); err != nil {
			return err
		}
	}
	if gs.DormantFactContestEnabled {
		if err := k.EnableDormantFactContest(ctx); err != nil {
			return err
		}
	}
	if gs.ReviewerBalanceParamEnabled {
		if err := k.EnableReviewerBalanceParam(ctx); err != nil {
			return err
		}
	}
	if gs.SeatDepositEnabled {
		if err := k.EnableSeatDeposit(ctx); err != nil {
			return err
		}
	}
	return nil
}

// exportGenesisV3Markers reads the consensus-repairs-v3 markers back into gs.
// A marker fault panics with its activation.ErrMarkerFault chain, as the
// markers in exportGenesisLegacy do, so a malformed marker is never exported
// as off.
func (k Keeper) exportGenesisV3Markers(ctx context.Context, gs *types.GenesisState) {
	doctrineIdentity, err := k.DoctrineIdentityEnabled(ctx)
	if err != nil {
		panic(err)
	}
	dormantFactContest, err := k.DormantFactContestEnabled(ctx)
	if err != nil {
		panic(err)
	}
	reviewerBalanceParam, err := k.ReviewerBalanceParamEnabled(ctx)
	if err != nil {
		panic(err)
	}
	seatDeposit, err := k.SeatDepositEnabled(ctx)
	if err != nil {
		panic(err)
	}
	gs.DoctrineIdentityEnabled = doctrineIdentity
	gs.DormantFactContestEnabled = dormantFactContest
	gs.ReviewerBalanceParamEnabled = reviewerBalanceParam
	gs.SeatDepositEnabled = seatDeposit
}
