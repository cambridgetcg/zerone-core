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
// With the doctrine-identity marker on (doctrine_identity_activation.go), four
// places read this instead of the label: ProcessMetabolism (the energy
// exemption), ResurrectDoctrineFacts (the rescue), and the two steps that grow
// and then clamp confidence, AdvanceConfidence and UpdateAllFitnessScores,
// which skip doctrine so its confidence is neither grown nor clamped.
//
// No stored confidence is reset. On a new chain doctrine keeps the value it is
// born with: types.DoctrineConfidence for genesis doctrine, and for the
// authority's facts the value the authority gave, clamped to the effective cap
// at creation (ClampConfidence, at MsgAddFact or when a guardian-veto
// injection matures). On a chain that turns the marker on later, doctrine
// keeps whatever it holds at that height: after a growth epoch, the effective
// cap it was clamped to (ClampConfidence; MaxConfidence, 880,000 by default).
//
// The label itself is reserved at admission with the marker on
// (doctrine_reserved.go): no conjecture, contradiction or legacy claim
// admitted while the marker is on may carry it, and a challenge of doctrine
// opened while it is on does not inherit it. Claims admitted before the marker
// was on keep their category. One accepted later makes a claim-derived fact
// labelled doctrine, which this predicate does not count as doctrine.
func isDoctrineFact(f *types.Fact) bool {
	return f.ClaimId == "" && (f.Category == types.DoctrineCategory || f.MethodId == types.DoctrineMethodId)
}
