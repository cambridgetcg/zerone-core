package keeper

import (
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// ─── What a conjecture is, in one place ─────────────────────────────────────
//
// A conjecture must never acquire standing it did not earn through actual
// verification of its truth — which, by construction, never happens: the panel
// that admitted it was asked whether it was well-posed, not whether it was so.
// "Standing" means citability, training-corpus inclusion, training value,
// non-zero confidence, calibration credit, or any reward.
//
// THE RULE THAT MAKES THAT HOLD: every guard keys on ClaimType, never on
// Status.
//
// The first version of this feature keyed its guards on
// FACT_STATUS_PROVISIONAL, and an adversarial review took it apart in five
// independent ways, because Status is a mutable field that cheap public
// messages can move:
//
//   - MsgSubmitContradiction flips any fact to CONTESTED for 1 uzrn, after
//     which reverseContradictionsFromClaim restores it to VERIFIED.
//   - MsgPatronizeFact drives ApplyPatronageEnergyBoost, which writes ACTIVE.
//   - The metabolism recovery arm writes ACTIVE on any energy income.
//   - Metabolic decay reaches AT_RISK on its own in ~41 epochs, with nobody
//     doing anything at all.
//   - MsgChallengeProvisionalFact itself moves the fact to CHALLENGED — so the
//     mechanism's one intended action opened every hole it was meant to close.
//
// Each of those left the fact a conjecture and made every status-keyed
// exemption evaporate at once. ClaimType is written once at fact creation and
// has no writer anywhere in the module, so a guard keyed on it cannot be
// stepped around by moving the fact through the status graph.

// IsConjecture reports whether a fact is an open question rather than
// something the chain believes. This is the ONLY correct predicate for
// deciding whether a conjecture exemption applies. Prefer it over any status
// comparison.
func IsConjecture(fact *types.Fact) bool {
	return fact != nil && fact.ClaimType == types.ClaimType_CLAIM_TYPE_CONJECTURE
}

// IsConjectureResolved reports whether a conjecture has reached a terminal
// state — refuted, or metabolised out of the graph. Used to decide whether the
// refutation door is still open.
func IsConjectureResolved(fact *types.Fact) bool {
	if fact == nil {
		return true
	}
	switch fact.Status {
	case types.FactStatus_FACT_STATUS_DISPROVEN,
		types.FactStatus_FACT_STATUS_PRUNED,
		types.FactStatus_FACT_STATUS_REVOKED:
		return true
	}
	return false
}

// RestoredStatusFor answers "what status should this fact return to when
// something restores it?" — the shared decision behind four call sites that
// previously each hardcoded their own answer: the contradiction reversal
// (rounds.go), challenge survival (rounds.go), the metabolism recovery arm and
// the patronage energy boost (metabolism.go).
//
// A conjecture returns to PROVISIONAL. Nothing that merely restores a fact —
// an abandoned contradiction, a failed probe, an energy top-up — is evidence
// about whether the conjecture is TRUE, and PROVISIONAL is the only status
// that keeps the refutation door open and the standing gates shut.
func RestoredStatusFor(fact *types.Fact) types.FactStatus {
	if IsConjecture(fact) {
		return types.FactStatus_FACT_STATUS_PROVISIONAL
	}
	return types.FactStatus_FACT_STATUS_ACTIVE
}

// errIfConjectureCitation refuses a claim that rests any part of its proof
// chain on a conjecture.
//
// This is not tidiness. computeProvenance derives a claim's
// dependency_confidence_floor as the MINIMUM effective confidence across all
// its cited edges, and a conjecture's confidence is zero by construction. So a
// claim citing one solid fact and one conjecture computes a floor of zero —
// and a zero floor is read everywhere as "no floor at all", because the clamp
// in createFactFromClaim is guarded by `floor > 0`. Citing a question would
// therefore not weaken a proof chain, it would silently exempt it from the
// ceiling its real foundations impose. A question is not evidence, and it may
// not appear anywhere that evidence is counted — in ANY status.
func errIfConjectureCitation(target *types.Fact) error {
	if !IsConjecture(target) {
		return nil
	}
	return types.ErrInvalidClaim.Wrapf(
		"fact %s is a conjecture and cannot be cited — an unsettled proposition is not support for anything, whatever status it currently holds; refute it with `tx knowledge challenge-provisional` instead",
		target.Id)
}
