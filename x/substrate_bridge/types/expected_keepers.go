package types

import (
	"context"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	qualificationtypes "github.com/zerone-chain/zerone/x/qualification/types"
)

// KnowledgeKeeper is the subset of x/knowledge.Keeper used by
// substrate_bridge for cited-fact validation and attribution.
// Implementations: x/knowledge/keeper.Keeper.
type KnowledgeKeeper interface {
	GetFact(ctx context.Context, factID string) (*knowledgetypes.Fact, bool)
}

// QualificationKeeper is the subset of x/qualification.Keeper used
// for submitter qualification checks.
type QualificationKeeper interface {
	GetDomainQualification(ctx context.Context, address, domain string) (*qualificationtypes.DomainQualification, bool)
}

// BankKeeper escrows submitter bonds and disburses rewards.
type BankKeeper interface {
	SendCoinsFromAccountToModule(ctx context.Context, sender sdk.AccAddress, recipientModule string, coins sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipient sdk.AccAddress, coins sdk.Coins) error
	BurnCoins(ctx context.Context, moduleName string, coins sdk.Coins) error
}

// VestingRewardsKeeper is the chain's single cap-gated mint entry point.
// Attestation rewards mint through it into the audit bounty pool —
// issuance follows participation, and the 222,222,222 ZRN cap is enforced
// once, chain-wide. Returns the amount actually minted, which may be
// clipped below the request when the cap is nearly exhausted.
type VestingRewardsKeeper interface {
	MintWithCap(ctx sdk.Context, recipientModule string, amount *big.Int) (*big.Int, error)
}
