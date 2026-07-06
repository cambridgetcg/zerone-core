package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// StakingKeeper defines the staking module interface required by governance.
type StakingKeeper interface {
	// GetTotalBondedStake returns the total bonded stake as a decimal string.
	GetTotalBondedStake(ctx context.Context) (string, error)
	// GetDelegatorTotalBonded returns the total bonded tokens for a delegator as a decimal string.
	GetDelegatorTotalBonded(ctx context.Context, addr string) (string, error)
	// CountActiveGuardians returns the number of active Guardian-tier validators.
	CountActiveGuardians(ctx context.Context) (uint64, error)
	// IsGuardian returns true if the address is Guardian tier (tier 4) and active.
	IsGuardian(ctx context.Context, addr string) (bool, error)
	// IsJailed returns true if the validator at the given address is jailed.
	IsJailed(ctx context.Context, addr string) (bool, error)
	// GetSlashCount returns the number of times a validator has been slashed.
	GetSlashCount(ctx context.Context, addr string) (uint64, error)
}

// BankKeeper defines the bank module interface required by governance.
type BankKeeper interface {
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins
}

// VestingRewardsKeeper defines the vesting rewards module interface for research fund disbursement.
type VestingRewardsKeeper interface {
	DisburseFromResearchFund(ctx sdk.Context, recipient sdk.AccAddress, amount sdk.Coins) error
}

// UpgradeKeeper defines the upgrade module interface for scheduling software upgrades.
type UpgradeKeeper interface {
	ScheduleUpgrade(ctx context.Context, plan *UpgradePlan) error
}

// ParamRouter dispatches parameter changes from passed LIPs to the target module keepers.
type ParamRouter interface {
	ApplyParamChange(ctx context.Context, module, key, value string) error
}

// FundingRecorder is the interface used by the SybilFundingDecorator.
type FundingRecorder interface {
	RecordFunding(ctx sdk.Context, sender, recipient, amount string, blockHeight uint64)
}

// EmergencyKeeper defines the emergency module interface for governance condition checking.
type EmergencyKeeper interface {
	CountHaltsForReason(ctx context.Context, reason string) uint64
}

// AlignmentKeeper defines the alignment module interface for health-aware governance.
type AlignmentKeeper interface {
	GetHealthCategory(ctx context.Context) string
}

// CreedKeeper defines the x/creed interface required by governance
// for the CategoryCreedAmendment LIP class. On a passed creed-
// amendment LIP, x/gov calls AnchorPinFromBytes with the pinned
// payload that was attached to the LIP body. The keeper rebuilds
// the PinnedCreed shape internally.
//
// IsActiveCouncilMember exposes the AI-side voter pool so future
// two-pool quorum tracking (a natural progression of this stage)
// can route votes to the correct pool at tally time.
type CreedKeeper interface {
	// AnchorPinFromBytes records a new pin from the canonical
	// hash + JSON-encoded commitment registry that the LIP carried.
	// The current version is auto-incremented; the LIP id is
	// recorded as the source.
	AnchorPinFromBytes(ctx context.Context, sourceLip string, canonicalHash []byte, commitmentsJSON []byte) error

	// IsActiveCouncilMember returns true if the address holds an
	// active Creed Council seat. Used to route votes to the AI-
	// side pool when two-pool quorum is enabled.
	IsActiveCouncilMember(ctx context.Context, address string) bool
}

// SubstrateBridgeKeeper defines the x/substrate_bridge interface required
// by governance for the CategoryAdapterRegistration LIP class (commitment
// 20 — issuance follows participation). On a passed adapter-registration
// LIP, x/gov calls WriteAdapter with the adapter spec carried by the LIP.
//
// Phase-0 note: this interface is declared now to establish the vocabulary
// and allow the gov keeper to hold a typed reference. Full dispatch wiring
// (adapter payload attachment to LIP body, on-pass WriteAdapter call) will
// be completed when the generic LIP-dispatch mechanism stabilises.
type SubstrateBridgeKeeper interface {
	// WriteAdapter persists or overwrites an adapter registration.
	// adapterID is the stable, canonical identifier for the adapter
	// (e.g., "github-commits-v1"). lipID is the governance LIP that
	// authorised the write; it is stored on the adapter record for
	// on-chain audit. The adapter proto bytes are the serialised
	// substrate_bridge.AdapterRegistration payload attached to the LIP.
	WriteAdapterFromGov(ctx context.Context, lipID string, adapterProtoBytes []byte) error
}
