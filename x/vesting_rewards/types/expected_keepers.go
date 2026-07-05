package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BankKeeper defines the expected bank module keeper interface.
type BankKeeper interface {
	MintCoins(ctx context.Context, moduleName string, amounts sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx context.Context, senderModule string, recipientModule string, amt sdk.Coins) error
	GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	GetSupply(ctx context.Context, denom string) sdk.Coin
}

// StakingKeeper defines the expected staking module keeper interface.
type StakingKeeper interface {
	GetActiveValidatorCount(ctx context.Context) uint32
}

// KnowledgeKeeper exposes the verification rate so block rewards can be
// coupled to knowledge throughput (T9 / thesis claim 1). Nil-safe: when this
// keeper is not wired, block rewards fall back to the pure decay schedule.
type KnowledgeKeeper interface {
	// GetVerificationRate is the legacy accept-rate (accepted/terminal). Retained
	// for the supply-coupling audit query; no longer drives block emission.
	GetVerificationRate(ctx context.Context) uint64
	// GetSurvivedChallengeRate is the survival-gate coupling signal:
	// survived/(survived+disproven) facts in BPS. Block emission couples to this,
	// so issuance follows truth surviving challenge, not acceptance volume.
	GetSurvivedChallengeRate(ctx context.Context) uint64
}
