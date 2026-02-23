package types

import (
	"context"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// StakingKeeper defines the expected staking module interface.
type StakingKeeper interface {
	GetTotalBondedStake(ctx sdk.Context) *big.Int
	GetActiveValidatorCount(ctx sdk.Context) int
}

// KnowledgeKeeper defines the expected knowledge module interface.
type KnowledgeKeeper interface {
	GetVerificationRate(ctx context.Context) uint64
}

// EmergencyKeeper defines the expected emergency module interface.
type EmergencyKeeper interface {
	IsHalted(ctx context.Context) bool
}
