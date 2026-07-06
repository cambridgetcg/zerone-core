package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/liquiditypool/types"
)

// Migrator handles in-place store migrations.
type Migrator struct {
	keeper Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// Migrate1to2 backfills TWAPAccumulator.StartBlock for accumulators created
// before the field existed. Without it, GetTWAP's divisor (LastBlock -
// StartBlock) would treat StartBlock=0 as "accumulating since genesis" and
// dilute the average for every pool created later. The pool's creation
// height is exact: CreatePool seeds the accumulator in the same block.
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	m.keeper.IteratePools(ctx, func(pool *types.Pool) bool {
		acc, found := m.keeper.GetTWAPAccumulator(ctx, pool.PoolId)
		if !found || acc.StartBlock != 0 {
			return false
		}
		acc.StartBlock = pool.CreatedAtBlock
		m.keeper.SetTWAPAccumulator(ctx, acc)
		return false
	})
	return nil
}
