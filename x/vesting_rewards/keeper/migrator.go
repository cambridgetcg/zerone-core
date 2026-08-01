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

// Migrate1to2 permanently retires the legacy founder revenue tap. It clears
// both compatibility fields regardless of their v1 values, preserves every
// unrelated parameter and all historical reward records, then writes through
// the validated v2 storage boundary. There was no accrued founder balance in
// this module: v1 transfers, if any, were synchronous and remain history.
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	params, err := m.keeper.getStoredParams(ctx)
	if err != nil {
		return fmt.Errorf("read legacy vesting_rewards params: %w", err)
	}
	params.FounderShareBps = 0
	params.FounderAddress = ""
	if err := m.keeper.setParams(ctx, params, true); err != nil {
		return fmt.Errorf("persist founder renunciation: %w", err)
	}
	return nil
}
