package keeper

import "github.com/zerone-chain/zerone/x/knowledge/types"

// isDoctrineFact reports whether a fact is the chain's own doctrine, judged by
// where the fact came from rather than by its label alone.
//
// Category is free text. On several paths a fact made from an accepted claim
// copies its claim's category without a check: a conjecture, a contradiction,
// the legacy claim path, and a challenge of a doctrine fact, which inherits the
// target's "doctrine" category. Such a fact always has a claim: both builders,
// createFactFromClaim and createFactFromClaimLegacy, set ClaimId to the claim's
// ID, and a claim's ID is never empty (GenerateClaimID returns 32 hex
// characters, and genesis refuses a claim without one).
//
// Genuine doctrine has no claim. types.BuildDoctrineFact (genesis doctrine)
// leaves ClaimId empty, and so do the authority's own facts: MsgAddFact and a
// matured guardian-veto injection. So a fact counts as doctrine only when it
// has no claim and carries the doctrine category or the doctrine_authorship
// method.
//
// Only ProcessMetabolism reads this, and only with the doctrine-identity marker
// on (doctrine_identity_activation.go). ResurrectDoctrineFacts and admission
// still use the label.
func isDoctrineFact(f *types.Fact) bool {
	return f.ClaimId == "" && (f.Category == types.DoctrineCategory || f.MethodId == types.DoctrineMethodId)
}
