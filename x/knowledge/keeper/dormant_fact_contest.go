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
// it can be open at once. The end of a review-neutral challenge therefore
// restores and credits its target only when this challenge holds it
// (challengeHolds): the target is CHALLENGED and its latest status transition
// is this challenge's own admission. A target a challenge does not hold is
// left alone, whatever its status. That includes a dormant target that
// recovered to ACTIVE through patronage, or through another challenge it
// survived, while this one was open: its end writes no status, transition or
// survival cutoff, and the survival sweep pays any pending survival reward of
// a standing fact. The first challenge that refutes a dormant fact disproves
// it. A later accepted one leaves the DISPROVEN fact as it is and becomes no
// fact of its own: its claim is ACCEPTED and settles as accepted, but the
// refutation has been made, and a challenge claim has no dedup hold to stop
// the same refutation text being minted again. One challenger opens at most
// one challenge of a fact per block, because the claim identity is the
// challenger, the fact and the height; where the contest applies the
// challenge doors refuse a repeat before any coins move. The open-questions
// listing flags a conjecture under challenge by its CHALLENGED status, so a
// dormant conjecture with an open refutation round is listed without that
// flag.
//
// The challenge doors apply the contest only to a review-neutral claim
// (policy 1). On the legacy policy a MALFORMED or INCONCLUSIVE aggregated
// verdict restores no challenged fact (completeRound is pinned), and an
// accepted challenge earns a confidence-scaled bonus, so a target held
// CHALLENGED could be held for good and a target not held could pay that
// bonus to every concurrent challenger. A legacy-policy challenge is
// therefore admitted, held and ended exactly as with the marker off: the
// round paths read the hold record, skip a second disproval and withhold a
// counter-fact only for a review-neutral challenge (contestGoverns), so a
// legacy-policy end reads and writes what it did, for the same gas. The
// open-questions listing treats a PRUNED conjecture as open only with review
// neutrality on as well. Native genesis selects both. Contradictions apply
// the contest on either policy.
//
// Known limits, not changed here:
//   - The citation rule needs one of consensus-repairs-v2's creation markers,
//     calibration smoothing or the zero-confidence floor. With both off, a new
//     fact is made by the frozen creation body (rounds_repairs_legacy.go,
//     pinned), which has no dormant gate, so an accepted contradiction feeds
//     its AT_RISK, EXPIRED or PRUNED target citation energy and its author a
//     "cited" edge. Native genesis cannot select that (InitChain requires
//     every v2 selection), and a v3 carrier would run after v2's. A
//     hand-made genesis can. Nothing ties this marker to v2's: the v3
//     registry is kept independent of v1 and v2 (each v3 entry enables from
//     a chain with every earlier marker absent), so refusing it is for the
//     v3 carrier and its genesis requirement to decide.
//   - An accepted challenge disproves its target as today. DISPROVEN is not
//     metabolised, so the domain's counters (AtRiskCount, TotalEnergy) keep
//     a target that falls while AT_RISK or EXPIRED where it was when it fell,
//     as they do for any disproven fact.
//   - The falsification cascade moves an AT_RISK descendant of a disproven
//     fact to CONTESTED (vindication.go), and metabolism skips a CONTESTED
//     fact. A CONTESTED fact is challengeable, so a later challenge of it
//     holds it, and if that challenge is rejected the fact returns ACTIVE
//     with its at-risk stamp cleared while AtRiskCount still counts it. That
//     holds for every disproven parent; the marker only makes a dormant
//     parent disprovable.
//   - Revival does not read contest reasons. Patronage or metabolism returns
//     an AT_RISK or EXPIRED fact ACTIVE (RestoredStatusFor) even while an
//     accepted contradiction's counter-fact stands (R1), because
//     contestAcceptedTargets contests only a VERIFIED or ACTIVE target. With
//     the marker off that holds for AT_RISK; with it on, an EXPIRED target can
//     be contradicted as well, so it holds for EXPIRED too. A PRUNED fact
//     does not revive. Closing it means revival consults contestReasonRemains,
//     a change to metabolism and patronage outside this repair.
//   - challengeHolds needs the fact's status history. A review-neutral
//     challenge that holds a fact whose history is missing or unreadable
//     (only a hand-made genesis can import one; an exported genesis carries
//     the history) leaves it CHALLENGED when it ends, and reports
//     contest_check_incomplete.
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
// its target and a second challenge that holds it, all in one block, and one
// actor can arrange all three: patronage is open to anyone. The dormant
// challenge's REJECT, MALFORMED or INCONCLUSIVE end then restores the target
// while the second challenge is still open. On the review-neutral policy that
// end credits no energy, corroboration or karma; what is lost is the second
// challenge's hold, and its end may then release the survival reward.
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

// contestGoverns reports whether the dormant-fact contest governs the end of
// challenge: the marker is on and challenge is review-neutral (policy 1),
// the only policy whose admission may leave a target unheld. A legacy-policy
// challenge was held at admission exactly as with the marker off, and the
// round paths end it exactly as with the marker off, reading nothing more.
// It reads only the snapshot and the claim, so it charges no gas.
func (k roundsExecution) contestGoverns(challenge *types.Claim) bool {
	return k.dormantFactContest && challenge != nil && challenge.ReviewPolicyVersion == types.ReviewPolicyNeutral
}

// challengeEndActs reports whether the end of challenge may restore and
// credit fact. Where the contest does not govern challenge (contestGoverns)
// it always may, as before. Where it does, only a CHALLENGED fact that
// challenge holds (challengeHolds). Any other fact is left as it is and earns
// nothing: a dormant target that was never moved, one that fell, became
// contested or recovered to VERIFIED or ACTIVE while the contest was open,
// one another challenge holds, or one another challenge already refuted. A
// hold record that cannot be read is reported with contest_check_incomplete,
// and the fact is left as it is: restoring it could undo another challenge's
// hold.
//
// It writes nothing except the contest_check_incomplete event.
func (k roundsExecution) challengeEndActs(ctx context.Context, fact *types.Fact, challenge *types.Claim) bool {
	if !k.contestGoverns(challenge) {
		return true
	}
	if fact == nil || fact.Status != types.FactStatus_FACT_STATUS_CHALLENGED {
		return false
	}
	held, err := k.challengeHolds(ctx, fact, challenge)
	if err != nil {
		emitContestCheckIncomplete(ctx, fact.Id, fmt.Errorf("admission record of challenge %s: %w", challenge.Id, err))
		return false
	}
	return held
}

// refutedAlready reports whether an accepted challenge finds its target
// already DISPROVEN where the contest governs it. Such a refutation becomes no
// fact: another challenge open at the same time refuted the target first, and
// each further acceptance would mint a VERIFIED copy of the refutation, with
// birth energy and a place in the domain's counters, which a challenge claim's
// missing dedup hold would not stop. The target read is the one
// handleChallengeDisproven makes, with the same error.
func (k roundsExecution) refutedAlready(ctx context.Context, challenge *types.Claim) (bool, error) {
	if !k.contestGoverns(challenge) || challenge.ProvisionalFactId == "" {
		return false, nil
	}
	target, found, err := k.getFactChecked(ctx, challenge.ProvisionalFactId)
	if err != nil {
		return false, err
	}
	return found && target.Status == types.FactStatus_FACT_STATUS_DISPROVEN, nil
}

// refuseRepeatedChallenge refuses a challenge whose claim identity already
// exists. Where the contest applies, a target is not always held CHALLENGED,
// so one challenger could open a second challenge of the same fact in one
// block, and GenerateClaimID(challenger, fact, height) would give it the first
// one's claim and round: the second would overwrite them and strand the first
// stake. The challenge doors call it before any coins move. It reads one key.
func (k Keeper) refuseRepeatedChallenge(ctx context.Context, challengeClaimID string) error {
	prior, err := k.storeService.OpenKVStore(ctx).Get(types.ClaimKey(challengeClaimID))
	if err != nil {
		return fmt.Errorf("read prior challenge claim: %w", err)
	}
	if prior != nil {
		return fmt.Errorf("challenge claim identity %s already exists: a challenger opens at most one challenge of a fact per block", challengeClaimID)
	}
	return nil
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
