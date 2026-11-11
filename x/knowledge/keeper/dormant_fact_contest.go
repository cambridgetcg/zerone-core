package keeper

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Dormant-fact contest (consensus-repairs-v3, marker 0x95,
// DormantFactContestEnabled).
//
// A fact that gains no energy goes AT_RISK, then EXPIRED, then PRUNED
// (metabolism.go). Nothing in that path asks whether the fact is false. With
// the marker on, such a fact stays open to falsification:
//
//   - MsgChallengeFact admits a non-conjecture target that is AT_RISK, EXPIRED
//     or PRUNED. A conjecture keeps its own door, MsgChallengeProvisionalFact.
//   - MsgSubmitContradiction's standing check also admits EXPIRED and PRUNED.
//   - A PRUNED conjecture is not resolved: MsgChallengeProvisionalFact admits
//     it and the open-questions listing keeps it. DISPROVEN and REVOKED still
//     resolve a conjecture.
//
// Attention, never existence. A contest of a dormant fact does not move it:
// the challenge doors do not set a target they find AT_RISK, EXPIRED or
// PRUNED to CHALLENGED, and nothing that ends the contest without refuting
// the fact touches it. So for as long as the contest is open the fact is
// what it would have been without it: metabolism, niche competition and
// patronage keep acting on it, it gains no standing (training value, the
// dedup hold on its text, R1's counter-fact hold), and when the contest ends
// without refuting it, it has the status, energy and at-risk stamp it would
// have had anyway. When no metabolism epoch falls inside the contest, that is
// exactly the record it had when the contest was admitted. A contest that
// refutes it disproves it as today. A contradiction never moves a dormant
// target either, and with the marker on a CONTRADICTS edge to a dormant
// target earns it no citation energy and its author no "cited" edge.
//
// Because a dormant target is not held CHALLENGED, more than one challenge of
// it can be open at once. The end of a challenge therefore restores and
// credits a target only when this challenge holds it (challengeHolds): the
// target is CHALLENGED and its latest status transition is this challenge's
// own admission. A target a challenge does not hold is left alone. The first
// challenge that refutes a dormant fact disproves it, and a later one leaves
// the DISPROVEN fact as it is. The open-questions listing flags a conjecture
// under challenge by its CHALLENGED status, so a dormant conjecture with an
// open refutation round is listed without that flag.
//
// The challenge doors apply the contest only to a review-neutral claim
// (policy 1). On the legacy policy a MALFORMED or INCONCLUSIVE aggregated
// verdict restores no challenged fact (completeRound is pinned), and an
// accepted challenge earns a confidence-scaled bonus, so a target held
// CHALLENGED could be held for good and a target not held could pay that
// bonus to every concurrent challenger. A legacy-policy challenge is
// therefore admitted, held and ended exactly as with the marker off, and the
// open-questions listing treats a PRUNED conjecture as open only with review
// neutrality on as well. Native genesis selects both. Contradictions apply
// the contest on either policy.
//
// Known limits, not changed here:
//   - An accepted challenge disproves its target as today. DISPROVEN is not
//     metabolised, so the domain's counters (AtRiskCount, TotalEnergy) keep
//     a target that falls while AT_RISK or EXPIRED where it was when it fell,
//     as they do for any disproven fact.
//   - The falsification cascade moves an AT_RISK descendant of a disproven
//     fact to CONTESTED (vindication.go), metabolism skips a CONTESTED fact,
//     and a later rejected challenge of it returns it ACTIVE with its at-risk
//     stamp cleared. That holds for every disproven parent; the marker only
//     makes a dormant parent disprovable.
//
// With the marker off none of this runs: every predicate below reduces to the
// one it extends, and the round paths restore as before.

// isDormantStatus reports whether status is one that metabolism gives a fact
// that gained no energy: AT_RISK, EXPIRED or PRUNED.
func isDormantStatus(status types.FactStatus) bool {
	switch status {
	case types.FactStatus_FACT_STATUS_AT_RISK,
		types.FactStatus_FACT_STATUS_EXPIRED,
		types.FactStatus_FACT_STATUS_PRUNED:
		return true
	}
	return false
}

// dormantContestApplies reports whether a challenge door applies the contest
// to a claim of reviewPolicy: the marker is on and the claim is
// review-neutral.
func dormantContestApplies(dormant bool, reviewPolicy uint32) bool {
	return dormant && reviewPolicy == types.ReviewPolicyNeutral
}

// dormantChallengeTarget reports whether MsgChallengeFact admits fact because
// of the dormant-fact contest: the contest applies, the fact is not a
// conjecture, and it is AT_RISK, EXPIRED or PRUNED.
func dormantChallengeTarget(fact *types.Fact, applies bool) bool {
	return applies && fact != nil && !IsConjecture(fact) && isDormantStatus(fact.Status)
}

// contradictionTargetAdmissible is the standing check of a contradiction's
// target under claim integrity: factStands, and with the marker on also
// EXPIRED and PRUNED.
func contradictionTargetAdmissible(status types.FactStatus, dormant bool) bool {
	if factStands(status) {
		return true
	}
	return dormant && (status == types.FactStatus_FACT_STATUS_EXPIRED ||
		status == types.FactStatus_FACT_STATUS_PRUNED)
}

// conjectureResolved is IsConjectureResolved, except that where the contest
// applies a PRUNED conjecture is still open: lapsing is not an answer.
func conjectureResolved(fact *types.Fact, applies bool) bool {
	if applies && fact != nil && fact.Status == types.FactStatus_FACT_STATUS_PRUNED {
		return false
	}
	return IsConjectureResolved(fact)
}

// errDormantContestNotChallengeable is MsgChallengeFact's refusal where the
// contest applies. integrity says whether a CONTESTED fact is challengeable.
func errDormantContestNotChallengeable(fact *types.Fact, integrity bool) error {
	if IsConjecture(fact) && isDormantStatus(fact.Status) {
		return fmt.Errorf("fact %s is a conjecture (status: %s); refute it with `tx knowledge challenge-provisional`", fact.Id, fact.Status)
	}
	accepted := "verified, active, at-risk, expired or pruned"
	if integrity {
		accepted = "verified, active, contested, at-risk, expired or pruned"
	}
	return fmt.Errorf("fact %s is not in a challengeable state (status: %s) (only %s facts accept probes)", fact.Id, fact.Status, accepted)
}

// challengeHolds reports whether challenge holds fact: the fact is CHALLENGED
// and its latest recorded transition is the move to CHALLENGED at the height
// the challenge was submitted, naming this challenge when a cause is
// recorded. It reads the fact's transition counter and the transition at
// that sequence, both by exact key, so nothing another fact writes can stand
// in for it. A well-formed record of another admission is not this
// challenge's hold. An absent or malformed record, or a latest transition
// that is not a move to CHALLENGED, is an error: the caller cannot tell who
// holds the fact.
//
// Without claim integrity no cause is recorded, so two admissions in one
// block cannot be told apart. That needs a dormant challenge, a recovery of
// its target and a second challenge that holds it, all in one block.
func (k Keeper) challengeHolds(ctx context.Context, fact *types.Fact, challenge *types.Claim) (bool, error) {
	if fact.Status != types.FactStatus_FACT_STATUS_CHALLENGED {
		return false, nil
	}
	store := k.storeService.OpenKVStore(ctx)
	counter, err := store.Get(types.StatusTransitionSeqKey(fact.Id))
	if err != nil {
		return false, err
	}
	if counter == nil {
		return false, fmt.Errorf("no status transition is recorded")
	}
	seq, err := decodeHistoryCounter(counter)
	if err != nil {
		return false, err
	}
	key := types.StatusTransitionKey(fact.Id, seq)
	bz, err := store.Get(key)
	if err != nil {
		return false, err
	}
	if bz == nil {
		return false, fmt.Errorf("status transition %d is missing", seq)
	}
	if err := validateToKStoredRecord(key, bz); err != nil {
		return false, err
	}
	t := &types.StatusTransition{}
	if err := proto.Unmarshal(bz, t); err != nil {
		return false, err
	}
	if t.FactId != fact.Id || t.Seq != seq {
		return false, fmt.Errorf("status transition %d names fact %q sequence %d", seq, t.FactId, t.Seq)
	}
	if t.NewStatus != types.FactStatus_FACT_STATUS_CHALLENGED {
		return false, fmt.Errorf("latest status transition is %s, not CHALLENGED", t.NewStatus)
	}
	if t.BlockHeight != challenge.SubmittedAtBlock {
		return false, nil
	}
	if t.CauseId != "" && t.CauseId != challenge.Id {
		return false, nil
	}
	return true, nil
}

// challengeEndActs reports whether the end of challenge may restore and
// credit fact. With the marker off it always may, as before. With it on:
//
//   - a CHALLENGED fact only when challenge holds it (challengeHolds). A
//     record that cannot be read is reported with contest_check_incomplete,
//     and the fact is left as it is: restoring it could undo another
//     challenge's hold.
//   - an AT_RISK, EXPIRED, PRUNED, DISPROVEN, CONTESTED or other non-standing
//     fact never: no challenge holds it, so it is a dormant target that was
//     never moved, one that fell or became contested while the contest was
//     open, or one another challenge already refuted. The contest leaves it
//     as it is and it earns nothing.
//   - a VERIFIED, ACTIVE or PROVISIONAL fact that no challenge holds as
//     before. In a real flow that is a dormant target that recovered through
//     patronage or metabolism while the contest was open; on the
//     review-neutral policy, the only one a dormant contest runs on, its end
//     then credits no energy, corroboration or karma, and may hand over a
//     pending survival reward that the sweep would also pay a standing fact.
//
// It writes nothing except the contest_check_incomplete event.
func (k roundsExecution) challengeEndActs(ctx context.Context, fact *types.Fact, challenge *types.Claim) bool {
	if !k.dormantFactContest {
		return true
	}
	if fact == nil || challenge == nil {
		return false
	}
	switch fact.Status {
	case types.FactStatus_FACT_STATUS_CHALLENGED:
		held, err := k.challengeHolds(ctx, fact, challenge)
		if err != nil {
			emitContestCheckIncomplete(ctx, fact.Id, fmt.Errorf("admission record of challenge %s: %w", challenge.Id, err))
			return false
		}
		return held
	case types.FactStatus_FACT_STATUS_VERIFIED,
		types.FactStatus_FACT_STATUS_ACTIVE,
		types.FactStatus_FACT_STATUS_PROVISIONAL:
		return true
	}
	return false
}

// OpenQuestionsForDomainChecked reads the dormant-fact contest marker once,
// returns a fault rather than choose, and lists the open questions. The
// marker read charges no gas, so with the marker off the query costs what it
// cost before the marker. With it on the query also reads review neutrality,
// through the caller's gas meter like every read of that marker, because a
// PRUNED conjecture is open only where its refutation door admits it.
func (k Keeper) OpenQuestionsForDomainChecked(ctx context.Context, domain string, limit uint32) ([]*types.OpenQuestion, uint32, uint32, bool, error) {
	dormant, err := k.DormantFactContestEnabled(ctx)
	if err != nil {
		return nil, 0, 0, false, err
	}
	if dormant {
		neutral, err := k.ReviewNeutralityEnabled(ctx)
		if err != nil {
			return nil, 0, 0, false, err
		}
		dormant = neutral
	}
	questions, total, examined, scanLimitReached := k.openQuestionsForDomain(ctx, domain, limit, dormant)
	return questions, total, examined, scanLimitReached, nil
}
