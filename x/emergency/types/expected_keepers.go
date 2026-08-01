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

// RecoveryAuthorizationTargetVerifier binds a Guardian ceremony to an
// existing SDK-governance proposal, its exact Any bytes, and the exact upgrade
// plan being scheduled or cancelled. The app wires this after keeper
// construction to avoid a module dependency cycle.
type RecoveryAuthorizationTargetVerifier interface {
	VerifyRecoveryAuthorizationTarget(
		ctx context.Context,
		proposalID uint64,
		actionSHA256 string,
		upgradePlanSHA256 string,
		actionType string,
	) error
}
