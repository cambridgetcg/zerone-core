package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/internal/accountingmigration"
)

type Migrator struct{ keeper Keeper }

func NewMigrator(k Keeper) Migrator { return Migrator{keeper: k} }

// Migrate1to2 validates existing claims and backing before enabling repaired
// accounting. It neither changes claimants nor resolves historical discrepancies.
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	if err := accountingmigration.Require(ctx); err != nil {
		return err
	}
	return m.keeper.EnableAccountingSafety(ctx)
}
