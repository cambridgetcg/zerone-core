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
// Being contested buys no standing. A challenge moves its target to
// CHALLENGED, as it does today. When a challenge of a target admitted in a
// dormant status ends without refuting it, the target returns to exactly that
// status, with its energy and AtRiskSinceEpoch untouched, and gains no
// survival energy, corroboration, calibration credit, karma edge or release
// of a pending survival reward. This holds wherever the round paths restore a
// challenged fact: a rejection; a malformed, inconclusive or starved ending
// with review neutrality on; a starved ending with it off. A malformed or
// inconclusive aggregated verdict with review neutrality off restores no
// challenged fact, dormant or not, and stays so: completeRound's body is
// pinned to its foundation (TestQ2LegacyBodiesPinnedToFoundation). An
// accepted challenge disproves its target as today. A contradiction never
// changes a dormant target's status: admission changes no fact, and every
// outcome acts only on VERIFIED, ACTIVE or CONTESTED targets.
//
// The fact record does not hold the status a challenge found: CHALLENGED
// overwrites it, and AtRiskSinceEpoch, Energy and the epoch cannot rebuild it
// (competition prunes with any AtRiskSinceEpoch; a lineage cascade sets
// AT_RISK without stamping it; a parameter change moves the boundaries).
// The admission's own status transition holds it: SetFact records
// {prior, CHALLENGED} at the height the challenge claim was submitted, in
// both record-integrity states. challengeAdmissionStatus reads that one
// record by point lookups (the fact's transition counter, then the
// transition at that sequence), so nothing another fact writes can stand in
// for it.
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

// dormantChallengeTarget reports whether MsgChallengeFact admits fact because
// of the dormant-fact contest: the marker is on, the fact is not a conjecture,
// and it is AT_RISK, EXPIRED or PRUNED.
func dormantChallengeTarget(fact *types.Fact, dormant bool) bool {
	return dormant && fact != nil && !IsConjecture(fact) && isDormantStatus(fact.Status)
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

// conjectureResolved is IsConjectureResolved, except that with the marker on
// a PRUNED conjecture is still open: lapsing is not an answer.
func conjectureResolved(fact *types.Fact, dormant bool) bool {
	if dormant && fact != nil && fact.Status == types.FactStatus_FACT_STATUS_PRUNED {
		return false
	}
	return IsConjectureResolved(fact)
}

// errDormantContestNotChallengeable is MsgChallengeFact's refusal with the
// marker on. integrity says whether a CONTESTED fact is challengeable.
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

// challengeAdmissionStatus returns the status challenge found its target in
// when it was admitted: the prior status of the target's latest recorded
// transition, which must be the move to CHALLENGED at the height the
// challenge was submitted and, when a cause is recorded, name this challenge.
// It reads the fact's transition counter and the transition at that sequence,
// both by exact key. Any absent, malformed or non-matching record is an
// error: the caller then cannot tell what the challenge found.
func (k Keeper) challengeAdmissionStatus(ctx context.Context, factID string, challenge *types.Claim) (types.FactStatus, error) {
	unknown := types.FactStatus_FACT_STATUS_UNSPECIFIED
	store := k.storeService.OpenKVStore(ctx)
	counter, err := store.Get(types.StatusTransitionSeqKey(factID))
	if err != nil {
		return unknown, err
	}
	if counter == nil {
		return unknown, fmt.Errorf("no status transition is recorded")
	}
	seq, err := decodeHistoryCounter(counter)
	if err != nil {
		return unknown, err
	}
	key := types.StatusTransitionKey(factID, seq)
	bz, err := store.Get(key)
	if err != nil {
		return unknown, err
	}
	if bz == nil {
		return unknown, fmt.Errorf("status transition %d is missing", seq)
	}
	if err := validateToKStoredRecord(key, bz); err != nil {
		return unknown, err
	}
	t := &types.StatusTransition{}
	if err := proto.Unmarshal(bz, t); err != nil {
		return unknown, err
	}
	if t.FactId != factID || t.Seq != seq {
		return unknown, fmt.Errorf("status transition %d names fact %q sequence %d", seq, t.FactId, t.Seq)
	}
	if t.NewStatus != types.FactStatus_FACT_STATUS_CHALLENGED || t.BlockHeight != challenge.SubmittedAtBlock {
		return unknown, fmt.Errorf("latest status transition is %s at %d, not CHALLENGED at %d", t.NewStatus, t.BlockHeight, challenge.SubmittedAtBlock)
	}
	if t.CauseId != "" && t.CauseId != challenge.Id {
		return unknown, fmt.Errorf("latest status transition names cause %q, not challenge %s", t.CauseId, challenge.Id)
	}
	return t.PriorStatus, nil
}

// challengeAdmission classifies a challenged fact at the end of its
// challenge.
type challengeAdmission int

const (
	// admittedStanding: the marker is off, the fact is no longer CHALLENGED,
	// or the challenge found it in a status that is not dormant. The end of
	// the challenge restores it as before.
	admittedStanding challengeAdmission = iota
	// admittedDormant: the challenge found it AT_RISK, EXPIRED or PRUNED. It
	// returns to exactly that status.
	admittedDormant
	// admittedUnknown: the record of what the challenge found cannot be read.
	// The fact stays CHALLENGED and earns nothing; contest_check_incomplete
	// says why. Restoring it as before could promote a dormant fact to
	// ACTIVE, which is what the marker forbids.
	admittedUnknown
)

// classifyChallengeAdmission decides how the end of challenge treats fact.
// With the marker on and fact CHALLENGED it reads challengeAdmissionStatus;
// the returned status is meaningful only for admittedDormant. It writes
// nothing except, for admittedUnknown, a contest_check_incomplete event.
func (k roundsExecution) classifyChallengeAdmission(ctx context.Context, fact *types.Fact, challenge *types.Claim) (challengeAdmission, types.FactStatus) {
	none := types.FactStatus_FACT_STATUS_UNSPECIFIED
	if !k.dormantFactContest || fact == nil || challenge == nil ||
		fact.Status != types.FactStatus_FACT_STATUS_CHALLENGED {
		return admittedStanding, none
	}
	found, err := k.challengeAdmissionStatus(ctx, fact.Id, challenge)
	if err != nil {
		emitContestCheckIncomplete(ctx, fact.Id, fmt.Errorf("admission record of challenge %s: %w", challenge.Id, err))
		return admittedUnknown, none
	}
	if isDormantStatus(found) {
		return admittedDormant, found
	}
	return admittedStanding, none
}

// OpenQuestionsForDomainChecked reads the dormant-fact contest marker once,
// returns a fault rather than choose, and lists the open questions. The read
// charges no gas, so the query costs what it cost before the marker.
func (k Keeper) OpenQuestionsForDomainChecked(ctx context.Context, domain string, limit uint32) ([]*types.OpenQuestion, uint32, uint32, bool, error) {
	dormant, err := k.DormantFactContestEnabled(ctx)
	if err != nil {
		return nil, 0, 0, false, err
	}
	questions, total, examined, scanLimitReached := k.openQuestionsForDomain(ctx, domain, limit, dormant)
	return questions, total, examined, scanLimitReached, nil
}
