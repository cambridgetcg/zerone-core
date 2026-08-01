package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
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
	// GetValidatorByConsAddr resolves a consensus address (the block header's
	// ProposerAddress) to the validator record for historical reward queries.
	// The raw consensus address is not controlled by an operator secp256k1 key.
	GetValidatorByConsAddr(ctx context.Context, consAddr sdk.ConsAddress) (stakingtypes.Validator, error)
}

// DistributionKeeper defines the expected x/distribution keeper interface.
// The legacy reward API uses it to honor delegator withdraw-address mappings;
// consensus v2 has no automatic proposer or founder payout.
type DistributionKeeper interface {
	// GetDelegatorWithdrawAddr returns the withdraw address for a delegator;
	// x/distribution defaults it to the delegator address itself when unset.
	GetDelegatorWithdrawAddr(ctx context.Context, delAddr sdk.AccAddress) (sdk.AccAddress, error)
}

// KnowledgeKeeper exposes legacy acceptance telemetry and the challenged-fact
// survival signal retained for audit and vesting compatibility.
type KnowledgeKeeper interface {
	// GetVerificationRate is the legacy accept-rate (accepted/terminal). Retained
	// for the supply-coupling audit query; no longer drives block emission.
	GetVerificationRate(ctx context.Context) uint64
	// GetSurvivedChallengeRate is the historical survival-gate signal:
	// survived/(survived+disproven) facts in BPS. Consensus v2 does not couple
	// automatic issuance to this value.
	GetSurvivedChallengeRate(ctx context.Context) uint64
	// IsFactDisproven reports whether the fact has been adjudicated
	// FACT_STATUS_DISPROVEN by the PoT layer. Clawback is a consequence of
	// adjudication, never an assertion by the party requesting it — see
	// FalsifyClaim. Returns false for unknown ids, so an id that names no
	// fact can never authorise a clawback.
	IsFactDisproven(ctx context.Context, factID string) bool
}
