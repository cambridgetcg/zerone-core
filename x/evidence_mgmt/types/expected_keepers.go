package types

import "context"

// StakingKeeper defines the expected staking module interface for verifier tier checks.
type StakingKeeper interface {
	GetValidatorTier(ctx context.Context, addr string) (uint32, error)
}

// DisputesKeeper defines the expected disputes module interface for challenge→dispute bridge.
type DisputesKeeper interface {
	CreateDispute(ctx context.Context, challenger, targetID, reason, bond string) (string, error)
}
