package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

// Migrator handles in-place vesting_rewards store migrations.
type Migrator struct {
	keeper Keeper
}

// NewMigrator returns a vesting_rewards migrator.
func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// Migrate1to2 permanently retires the founder auto-split and the
// proposer-controlled transaction-presence reward. The migration never moves
// balances: it only clears compatibility parameters that could otherwise
// create new control- or identity-derived claims after activation.
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	params := m.keeper.GetParams(ctx)
	params.FounderShareBps = 0
	params.FounderAddress = ""
	params.BlockReward = "0"
	params.FloorReward = "0"
	params.EmptyBlockRewardRate = 0
	if err := types.ValidateParams(params); err != nil {
		return fmt.Errorf("vesting_rewards v2 params migration: %w", err)
	}
	m.keeper.SetParams(ctx, params)
	return nil
}
