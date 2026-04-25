package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BankKeeper escrows the bounty during inquiry lifetime.
type BankKeeper interface {
	SendCoinsFromAccountToModule(ctx context.Context, sender sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipient sdk.AccAddress, amt sdk.Coins) error
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
}

// KnowledgeKeeper exposes the read this module needs to verify a
// claim and observe whether it has produced an accepted fact.
//
// The contract is intentionally narrow:
//   - ClaimSubmitter — the bech32 of whoever owns the claim, so we
//     refuse cross-author answer attachments.
//   - AcceptedFactForClaim — the fact id (and ok bool) iff the
//     claim has produced an accepted fact (verified into the corpus).
//     If ok=false, the claim is still in flight, was rejected, or
//     was never created.
type KnowledgeKeeper interface {
	ClaimSubmitter(ctx context.Context, claimID string) (string, bool)
	AcceptedFactForClaim(ctx context.Context, claimID string) (string, bool)
}
