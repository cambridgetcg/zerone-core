package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Migrator handles in-place vesting-rewards state migrations.
type Migrator struct {
	keeper Keeper
}

// NewMigrator returns a vesting-rewards migrator.
func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// Migrate1to2 permanently retires the legacy founder revenue tap and
// transaction-presence block reward. It clears only their compatibility
// parameters, preserves unrelated params and historical records, and writes
// through the validated v2 storage boundary. No balances move.
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	params, err := m.keeper.getStoredParams(ctx)
	if err != nil {
		return fmt.Errorf("read legacy vesting_rewards params: %w", err)
	}
	params.FounderShareBps = 0
	params.FounderAddress = ""
	params.BlockReward = "0"
	params.FloorReward = "0"
	params.EmptyBlockRewardRate = 0
	if err := m.keeper.setParams(ctx, params, true); err != nil {
		return fmt.Errorf("persist vesting_rewards v2 retirement: %w", err)
	}
	return nil
}
