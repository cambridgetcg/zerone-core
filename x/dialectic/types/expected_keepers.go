package types

import "context"

// RoundReveal is one validator's revealed verdict in a verification
// round. Mirrors x/knowledge.RevealEntry but in this module's
// vocabulary so the synthesizer doesn't depend on the upstream
// proto.
type RoundReveal struct {
	Voter string
	Vote  string // "accept" / "reject" / "malformed"
}

// RoundOutcome is a verification round in the shape this synthesizer
// needs.
type RoundOutcome struct {
	RoundID string
	ClaimID string
	Verdict string // "accept" / "reject" / "malformed" / "unspecified"
	Reveals []RoundReveal
}

// FactInfo is just (id, claim_id, domain) — what the dialectic
// synthesizer needs to navigate from a fact to its round.
type FactInfo struct {
	ID      string
	ClaimID string
	Domain  string
}

// KnowledgeKeeper is the read-only contract dialectic needs from
// x/knowledge. Implementations are adapters declared in x/knowledge.
type KnowledgeKeeper interface {
	// GetFactInfo returns the fact's claim_id and domain. ok=false
	// if not found.
	GetFactInfo(ctx context.Context, factID string) (FactInfo, bool)
	// GetRoundForClaim returns the verification round associated
	// with a claim. ok=false if no round exists.
	GetRoundForClaim(ctx context.Context, claimID string) (RoundOutcome, bool)
	// IterateFactsByDomain calls cb with each fact_id in the domain.
	// Returning true from cb stops iteration.
	IterateFactsByDomain(ctx context.Context, domain string, cb func(factID string) bool)
	// IterateAllRounds is used by PairwiseDisagreement, which has
	// to walk every round both agents could have voted in. Bounded
	// by chain history; v2 should add a per-voter index.
	IterateAllRounds(ctx context.Context, cb func(RoundOutcome) bool)
}
