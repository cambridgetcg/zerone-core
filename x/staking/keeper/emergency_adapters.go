package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	emergencytypes "github.com/zerone-chain/zerone/x/emergency/types"
	"github.com/zerone-chain/zerone/x/staking/types"
)

// EmergencyStakingAdapter wraps the staking Keeper to satisfy the
// emergency module's StakingKeeper interface.
type EmergencyStakingAdapter struct {
	k Keeper
}

// NewEmergencyStakingAdapter returns an adapter for the emergency module.
func NewEmergencyStakingAdapter(k Keeper) *EmergencyStakingAdapter {
	return &EmergencyStakingAdapter{k: k}
}

// Compile-time interface check.
var _ emergencytypes.StakingKeeper = (*EmergencyStakingAdapter)(nil)

// GetValidator returns a validator as a ValidatorInfo for the emergency module.
func (a *EmergencyStakingAdapter) GetValidator(ctx context.Context, addr string) (*emergencytypes.ValidatorInfo, bool) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	val, found := a.k.GetValidator(sdkCtx, addr)
	if !found {
		return nil, false
	}
	return &emergencytypes.ValidatorInfo{
		Address:    val.OperatorAddress,
		TotalStake: val.TotalStake,
		Tier:       uint32(val.Tier),
		IsActive:   val.IsActive,
	}, true
}

// GetGuardianValidators returns all active Guardian-tier validators.
func (a *EmergencyStakingAdapter) GetGuardianValidators(ctx context.Context) ([]emergencytypes.ValidatorInfo, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	var guardians []emergencytypes.ValidatorInfo

	a.k.IterateValidators(sdkCtx, func(val *types.Validator) bool {
		if val.Tier == types.TierGuardian && val.IsActive {
			guardians = append(guardians, emergencytypes.ValidatorInfo{
				Address:    val.OperatorAddress,
				TotalStake: val.TotalStake,
				Tier:       uint32(val.Tier),
				IsActive:   val.IsActive,
			})
		}
		return false
	})

	return guardians, nil
}
