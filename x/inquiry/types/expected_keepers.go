package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BankKeeper escrows the bounty during inquiry lifetime. Also used to
// mint into the frontier-bounty pool when the chain sponsors inquiries
// (commitment 18) and to round-trip bounties between the inquiry pool
// and the frontier pool.
type BankKeeper interface {
	SendCoinsFromAccountToModule(ctx context.Context, sender sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipient sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error
	MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
}

// DomainSparsity is the narrow per-row read of the chain's frontier
// that the inquiry BeginBlocker needs in order to know which domains
// the chain should sponsor exploration in. Defined here (rather than
// imported from x/governance_synthesis) to keep the dependency
// direction one-way: governance_synthesis already reads from inquiry,
// so inquiry must NOT import governance_synthesis.
type DomainSparsity struct {
	Domain      string
	SparsityBps uint64
}

// FrontierProvider is wired post-init in app.go. It returns the
// chain's frontier ordered by sparsity descending, cut to at most
// `limit` rows. The inquiry BeginBlocker calls this each cadence
// tick to decide which domains to sponsor.
//
// Optional: if nil, the chain still functions; the commitment 18
// path is simply inactive (rather than crashing). This mirrors the
// frontier composition itself, which degrades to empty when an
// upstream keeper is missing.
type FrontierProvider func(ctx context.Context, limit uint32) []DomainSparsity

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
