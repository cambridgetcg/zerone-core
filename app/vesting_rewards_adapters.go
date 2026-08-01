package app

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// vestingRewardsStakingAdapter adapts the SDK x/staking keeper to the
// x/vesting_rewards expected StakingKeeper interface. It supplies:
//
//   - GetActiveValidatorCount: bonded validator count for historical reward
//     records and compatibility queries.
//   - GetValidatorByConsAddr: consensus-address → validator resolution so
//     a legacy reward calculation never targets an unspendable consensus
//     address. Consensus v2 does not mint proposer rewards.
type vestingRewardsStakingAdapter struct {
	sk *stakingkeeper.Keeper
}

// GetActiveValidatorCount returns the number of validators in the last
// (active/bonded) validator set.
func (a vestingRewardsStakingAdapter) GetActiveValidatorCount(ctx context.Context) uint32 {
	var count uint32
	_ = a.sk.IterateLastValidators(ctx, func(_ int64, _ stakingtypes.ValidatorI) bool {
		count++
		return false
	})
	return count
}

// GetValidatorByConsAddr resolves a consensus address to its validator record.
func (a vestingRewardsStakingAdapter) GetValidatorByConsAddr(ctx context.Context, consAddr sdk.ConsAddress) (stakingtypes.Validator, error) {
	return a.sk.GetValidatorByConsAddr(ctx, consAddr)
}
