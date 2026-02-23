package types

import "context"

// ValidatorInfo is a minimal view of a validator for emergency guardian checks.
type ValidatorInfo struct {
	Address    string
	TotalStake string
	Tier       uint32
	IsActive   bool
}

// StakingKeeper defines the expected staking module interface.
type StakingKeeper interface {
	GetValidator(ctx context.Context, addr string) (*ValidatorInfo, bool)
	GetGuardianValidators(ctx context.Context) ([]ValidatorInfo, error)
}
