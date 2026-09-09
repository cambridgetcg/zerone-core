package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/internal/accountingmigration"
)

// Migrator handles in-place store migrations.
type Migrator struct {
	keeper Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// Migrate1to2 migrates from version 1 to version 2.
// Stub — implement when v2 state changes are needed.
func (m Migrator) Migrate1to2(_ sdk.Context) error {
	return nil
}

// Migrate2to3 enables atomic future execution without replaying historical
// terminal proposals or inventing a claimant ledger for legacy LIP escrow.
func (m Migrator) Migrate2to3(ctx sdk.Context) error {
	if err := accountingmigration.Require(ctx); err != nil {
		return err
	}
	if err := m.keeper.ValidateAccountingMigration(ctx); err != nil {
		return err
	}
	return m.keeper.EnableAccountingSafety(ctx)
}
