package keeper

import (
	"context"
	"errors"
	"fmt"

	corestore "cosmossdk.io/core/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Contradiction outcomes with claim integrity on (knowledge v12, plan §5.2
// rows 2 and 4, §5.3; Task 12, P3b).
//
// Admitting a contradiction changes no fact. Accepting one contests each
// established target. When one ends without acceptance, a CONTESTED target is
// released only if no contest reason remains. Whether a reason remains is
// computed from relation edges and recorded transitions, never from events or
// from other pending claims, and a check that cannot finish keeps the fact
// CONTESTED. With the marker off none of this runs: knowledge 11 flips at
// admission and reverseContradictionsFromClaim restores.
//
// Task 13 (P3c) adds the decision paths out of CONTESTED: a challenge of the
// fact itself (challengeEndStatus) and the fall of a counter-fact
// (settleCounterFactTargets). Both use the same predicate.

const (
	causeContradictionAccepted = "contradiction_accepted"
	causeContradictionReversed = "contradiction_reversed"
	causeCounterfactDisproven  = "counterfact_disproven"
)

// errContestReasonFound ends a scan as soon as a reason is known.
var errContestReasonFound = errors.New("contest reason found")

// factStands reports whether a fact is accepted and neither refuted nor
// decayed out of the graph: VERIFIED, ACTIVE, CONTESTED, CHALLENGED or
// AT_RISK.
func factStands(status types.FactStatus) bool {
	switch status {
	case types.FactStatus_FACT_STATUS_VERIFIED,
		types.FactStatus_FACT_STATUS_ACTIVE,
		types.FactStatus_FACT_STATUS_CONTESTED,
		types.FactStatus_FACT_STATUS_CHALLENGED,
		types.FactStatus_FACT_STATUS_AT_RISK:
		return true
	}
	return false
}

// isCascadeRelation names the dependency edges a refutation cascades along,
// the same set as cascadeFalsificationChecked.
func isCascadeRelation(relation types.RelationType) bool {
	switch relation {
	case types.RelationType_RELATION_TYPE_SUPPORTS,
		types.RelationType_RELATION_TYPE_REQUIRES,
		types.RelationType_RELATION_TYPE_REFINES,
		types.RelationType_RELATION_TYPE_GENERALIZES,
		types.RelationType_RELATION_TYPE_CITES:
		return true
	}
	return false
}

// contestScan bounds one contest-reason check. Every KV entry it reads
// (relation, fact or transition) counts toward the ceilings that
// hasOtherActiveChallenge uses: 100k records and 64 MiB in total.
type contestScan struct {
	store          corestore.KVStore
	records, bytes int
}

func (k Keeper) newContestScan(ctx context.Context) *contestScan {
	return &contestScan{store: k.storeService.OpenKVStore(ctx)}
}

func (s *contestScan) charge(size int) error {
	s.records++
	if s.records > recordInventoryMaxRecords {
		return fmt.Errorf("contest scan exceeds record bound")
	}
	if size > recordInventoryMaxBytes-s.bytes {
		return fmt.Errorf("contest scan exceeds byte bound")
	}
	s.bytes += size
	return nil
}

// each charges, then reads, every entry under prefix in key order. read may
// return errContestReasonFound to stop early.
func (s *contestScan) each(prefix []byte, read func(key, value []byte) error) (err error) {
	it, err := s.store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, it.Close()) }()
	for ; it.Valid(); it.Next() {
		if err = s.charge(len(it.Key()) + len(it.Value())); err != nil {
			return err
		}
		if err = read(it.Key(), it.Value()); err != nil {
			return err
		}
	}
	return historyIteratorError(it)
}

// fact reads and validates one fact record. found is false when it is absent.
func (s *contestScan) fact(id string) (fact *types.Fact, found bool, err error) {
	key := types.FactKey(id)
	bz, err := s.store.Get(key)
	if err != nil {
		return nil, false, err
	}
	if err = s.charge(len(key) + len(bz)); err != nil || bz == nil {
		return nil, false, err
	}
	if err = validateToKStoredRecord(key, bz); err != nil {
		return nil, false, err
	}
	fact = &types.Fact{}
	if err = proto.Unmarshal(bz, fact); err != nil {
		return nil, false, err
	}
	return fact, true, nil
}

func decodeContestRelation(key, value []byte) (*types.FactRelation, error) {
	if err := validateToKStoredRecord(key, value); err != nil {
		return nil, err
	}
	rel := &types.FactRelation{}
	if err := proto.Unmarshal(value, rel); err != nil {
		return nil, err
	}
	return rel, nil
}

// latestTransitionAt is the block height of factID's latest recorded
// transition that match accepts, or 0 when none is recorded.
func (s *contestScan) latestTransitionAt(factID string, match func(*types.StatusTransition) bool) (uint64, error) {
	var at uint64
	err := s.each(types.StatusTransitionPrefixForFact(factID), func(key, value []byte) error {
		if err := validateToKStoredRecord(key, value); err != nil {
			return err
		}
		t := &types.StatusTransition{}
		if err := proto.Unmarshal(value, t); err != nil {
			return err
		}
		if t.FactId != factID {
			return fmt.Errorf("status history key/payload mismatch")
		}
		if match(t) {
			at = t.BlockHeight
		}
		return nil
	})
	return at, err
}

// disprovenAt is the height of the fact's latest transition to DISPROVEN, or
// 0 if none is recorded.
func (s *contestScan) disprovenAt(factID string) (uint64, error) {
	return s.latestTransitionAt(factID, func(t *types.StatusTransition) bool {
		return t.NewStatus == types.FactStatus_FACT_STATUS_DISPROVEN
	})
}

// lastSurvivedAt is the height of the fact's latest challenge_survived
// transition, or 0 if none is recorded. Knowledge 11 recorded survivals as
// "unknown", so they clear nothing.
func (s *contestScan) lastSurvivedAt(factID string) (uint64, error) {
	return s.latestTransitionAt(factID, func(t *types.StatusTransition) bool {
		return t.CauseEventType == causeChallengeSurvived
	})
}

// contestReason decides R1 and R2 (plan §5.3). survivedAt is called at most
// once, and only when a refuted dependency makes it matter. Any error means
// the check could not finish.
func (k Keeper) contestReason(ctx context.Context, fact *types.Fact, survivedAt func(*contestScan) (uint64, error)) (bool, error) {
	s := k.newContestScan(ctx)

	// R1: an accepted contradicting fact that still stands.
	err := s.each(types.FactRelationsByTargetPrefix(fact.Id), func(key, value []byte) error {
		rel, err := decodeContestRelation(key, value)
		if err != nil {
			return fmt.Errorf("incoming relation: %w", err)
		}
		if rel.TargetFactId != fact.Id {
			return fmt.Errorf("incoming relation names target %s", rel.TargetFactId)
		}
		if rel.Relation != types.RelationType_RELATION_TYPE_CONTRADICTS {
			return nil
		}
		source, found, err := s.fact(rel.SourceFactId)
		if err != nil {
			return fmt.Errorf("contradicting fact %s: %w", rel.SourceFactId, err)
		}
		if !found {
			return fmt.Errorf("contradicting fact %s is missing", rel.SourceFactId)
		}
		if factStands(source.Status) {
			return errContestReasonFound
		}
		return nil
	})
	if errors.Is(err, errContestReasonFound) {
		return true, nil
	}
	if err != nil {
		return true, err
	}

	// R2: a dependency refuted after the fact's last survived challenge.
	var dependencies []string
	err = s.each(types.FactRelationsBySourcePrefix(fact.Id), func(key, value []byte) error {
		rel, err := decodeContestRelation(key, value)
		if err != nil {
			return fmt.Errorf("outgoing relation: %w", err)
		}
		if rel.SourceFactId != fact.Id {
			return fmt.Errorf("outgoing relation names source %s", rel.SourceFactId)
		}
		if isCascadeRelation(rel.Relation) {
			dependencies = append(dependencies, rel.TargetFactId)
		}
		return nil
	})
	if err != nil {
		return true, err
	}
	var survived uint64
	known := false
	for _, id := range dependencies {
		dependency, found, err := s.fact(id)
		if err != nil {
			return true, fmt.Errorf("dependency %s: %w", id, err)
		}
		if !found {
			return true, fmt.Errorf("dependency %s is missing", id)
		}
		if dependency.Status != types.FactStatus_FACT_STATUS_DISPROVEN {
			continue
		}
		if !known {
			if survived, err = survivedAt(s); err != nil {
				return true, fmt.Errorf("survival history: %w", err)
			}
			known = true
		}
		if survived == 0 {
			return true, nil
		}
		at, err := s.disprovenAt(id)
		if err != nil {
			return true, fmt.Errorf("dependency %s history: %w", id, err)
		}
		if at > survived {
			return true, nil
		}
	}
	return false, nil
}

func emitContestCheckIncomplete(ctx context.Context, factID string, cause error) {
	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(sdk.NewEvent(
		"zerone.knowledge.contest_check_incomplete",
		sdk.NewAttribute("fact_id", factID),
		sdk.NewAttribute("reason", cause.Error()),
	))
}

func (k Keeper) contestReasonOrIncomplete(ctx context.Context, fact *types.Fact, survivedAt func(*contestScan) (uint64, error)) bool {
	remains, err := k.contestReason(ctx, fact, survivedAt)
	if err != nil {
		emitContestCheckIncomplete(ctx, fact.Id, err)
		return true
	}
	return remains
}

// contestReasonRemains reports whether fact must stay CONTESTED (plan §5.3):
// R1, an incoming CONTRADICTS edge from a standing fact; or R2, an outgoing
// dependency edge to a DISPROVEN fact that fell after survivedAt (or any
// such fall when survivedAt is 0). Pending contradictions are not reasons. It
// never returns an error: a read error, a malformed record or the scan bound
// means the reason remains, and contest_check_incomplete says why. It writes
// nothing.
func (k Keeper) contestReasonRemains(ctx context.Context, fact *types.Fact, survivedAt uint64) bool {
	return k.contestReasonOrIncomplete(ctx, fact, func(*contestScan) (uint64, error) { return survivedAt, nil })
}

// contestReasonRemainsSinceLastSurvival is contestReasonRemains with
// survivedAt read from the fact's history (lastSurvivedAt).
func (k Keeper) contestReasonRemainsSinceLastSurvival(ctx context.Context, fact *types.Fact) bool {
	return k.contestReasonOrIncomplete(ctx, fact, func(s *contestScan) (uint64, error) { return s.lastSurvivedAt(fact.Id) })
}

// storedContradiction reports whether the stored edge counterFactID→targetID
// is CONTRADICTS. It reads the reverse-index entry, the record R1 scans. The
// store keeps one edge per (source, target) pair and the last relation
// written wins, so a claim that names a target twice may store another type.
// An absent edge is not a contradiction. An unreadable one, or one whose
// payload names another pair (validateToKStoredRecord rebuilds the key from
// the payload), is an error.
func (k Keeper) storedContradiction(ctx context.Context, counterFactID, targetID string) (bool, error) {
	key := types.FactRelationReverseKey(targetID, counterFactID)
	bz, err := k.storeService.OpenKVStore(ctx).Get(key)
	if err != nil || bz == nil {
		return false, err
	}
	rel, err := decodeContestRelation(key, bz)
	if err != nil {
		return false, err
	}
	return rel.Relation == types.RelationType_RELATION_TYPE_CONTRADICTS, nil
}

// contestAcceptedTargets runs when claim is accepted and counterFactID was
// created from it (§5.2 row 2). Each VERIFIED or ACTIVE target of its
// CONTRADICTS relations becomes CONTESTED, with cause
// contradiction_accepted/<claim ID>, when the stored edge counterFactID→target
// is CONTRADICTS: acceptance and R1 read the same edge, so a target is never
// contested without the edge that holds it. The counter-fact stays VERIFIED
// (§9 D2). Errors are returned; completeRound runs in a cache that an error
// discards.
func (k Keeper) contestAcceptedTargets(ctx context.Context, claim *types.Claim, counterFactID string) error {
	for _, rel := range claim.Relations {
		if rel.Relation != types.RelationType_RELATION_TYPE_CONTRADICTS {
			continue
		}
		target, found, err := k.getFactChecked(ctx, rel.TargetFactId)
		if err != nil {
			return err
		}
		if !found || (target.Status != types.FactStatus_FACT_STATUS_VERIFIED && target.Status != types.FactStatus_FACT_STATUS_ACTIVE) {
			continue
		}
		contradicts, err := k.storedContradiction(ctx, counterFactID, target.Id)
		if err != nil {
			return fmt.Errorf("counter-fact edge %s→%s: %w", counterFactID, target.Id, err)
		}
		if !contradicts {
			continue
		}
		target.Status = types.FactStatus_FACT_STATUS_CONTESTED
		if err := k.SetFact(withTransitionCause(ctx, target.Id, causeContradictionAccepted, claim.Id), target); err != nil {
			return err
		}
		sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(sdk.NewEvent(
			"zerone.knowledge.contradiction_accepted",
			sdk.NewAttribute("fact_id", target.Id),
			sdk.NewAttribute("counter_claim_id", claim.Id),
			sdk.NewAttribute("counter_fact_id", counterFactID),
		))
	}
	return nil
}

// releaseContested returns a CONTESTED target to VERIFIED (PROVISIONAL for a
// conjecture), with cause eventType/causeID, when no contest reason remains
// since its last survived challenge. It reports whether it wrote.
func (k Keeper) releaseContested(ctx context.Context, target *types.Fact, eventType, causeID string) (bool, error) {
	if k.contestReasonRemainsSinceLastSurvival(ctx, target) {
		return false, nil
	}
	target.Status = types.FactStatus_FACT_STATUS_VERIFIED
	if IsConjecture(target) {
		target.Status = types.FactStatus_FACT_STATUS_PROVISIONAL
	}
	if err := k.SetFact(withTransitionCause(ctx, target.Id, eventType, causeID), target); err != nil {
		return false, err
	}
	return true, nil
}

// settleContradictionTargets runs when claim ends without acceptance (REJECT,
// MALFORMED, INCONCLUSIVE or starved; §5.2 row 4). Each CONTESTED target of
// its CONTRADICTS relations returns to VERIFIED (PROVISIONAL for a
// conjecture), with cause contradiction_reversed/<claim ID>, only when no
// contest reason remains since its last survived challenge. A target whose
// record cannot be read is left as it is and reported like an incomplete
// check. The writes are atomic: an error leaves every target unchanged.
func (k Keeper) settleContradictionTargets(ctx context.Context, claim *types.Claim) error {
	cached, write := sdk.UnwrapSDKContext(ctx).CacheContext()
	for _, rel := range claim.Relations {
		if rel.Relation != types.RelationType_RELATION_TYPE_CONTRADICTS {
			continue
		}
		target, found, err := k.getFactChecked(cached, rel.TargetFactId)
		if err != nil {
			emitContestCheckIncomplete(cached, rel.TargetFactId, fmt.Errorf("target: %w", err))
			continue
		}
		if !found || target.Status != types.FactStatus_FACT_STATUS_CONTESTED {
			continue
		}
		released, err := k.releaseContested(cached, target, causeContradictionReversed, claim.Id)
		if err != nil {
			return err
		}
		if !released {
			continue
		}
		cached.EventManager().EmitEvent(sdk.NewEvent(
			"zerone.knowledge.contradiction_reversed",
			sdk.NewAttribute("fact_id", target.Id),
			sdk.NewAttribute("reverted_by_claim", claim.Id),
		))
	}
	write()
	return nil
}

// endContradiction runs when claim ends without acceptance. With claim
// integrity on its targets settle through the contest-reason predicate; with
// it off, knowledge 11's reversal runs unchanged.
func (k Keeper) endContradiction(ctx context.Context, claim *types.Claim, integrity bool) error {
	if !integrity {
		k.reverseContradictionsFromClaim(ctx, claim)
		return nil
	}
	return k.settleContradictionTargets(ctx, claim)
}

// challengeEndStatus is the status a challenged fact takes when its challenge
// ends without refuting it (§5.2 rows 7 and 8): RestoredStatusFor, except
// that with claim integrity on an established fact returns to CONTESTED while
// a contest reason remains. A conjecture is exempt and returns to
// PROVISIONAL. A survival (REJECT) clears dependencies refuted at or before
// its own height, the current one; any other ending clears nothing, so the
// reasons are judged since the fact's last survived challenge. A standing
// counter-fact is never cleared by a survival (§9 D2). A check that cannot
// finish returns CONTESTED.
func (k Keeper) challengeEndStatus(ctx context.Context, fact *types.Fact, integrity, survived bool) types.FactStatus {
	if !integrity || IsConjecture(fact) {
		return RestoredStatusFor(fact)
	}
	var remains bool
	if survived {
		remains = k.contestReasonRemains(ctx, fact, uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()))
	} else {
		remains = k.contestReasonRemainsSinceLastSurvival(ctx, fact)
	}
	if remains {
		return types.FactStatus_FACT_STATUS_CONTESTED
	}
	return RestoredStatusFor(fact)
}

// settleCounterFactTargets runs when counter-fact K has just been disproven,
// in handleChallengeDisproven's cache and after its DISPROVEN write and
// cascade (§5.2 row 5). Each CONTESTED target of K's stored CONTRADICTS
// edges, the edges R1 reads, returns to VERIFIED (PROVISIONAL for a
// conjecture), with cause counterfact_disproven/<K>, when no contest reason
// remains since its last survived challenge. Another standing counter-fact,
// or a dependency refuted since then (K itself included), keeps it CONTESTED.
// A check that cannot finish never releases: if K's relations cannot be
// listed no target settles and contest_check_incomplete names K; a target
// that cannot be read is reported and left as it is. Neither undoes the
// disproval. Write errors are returned.
func (k Keeper) settleCounterFactTargets(ctx context.Context, counterFactID string) error {
	var targets []string
	err := k.newContestScan(ctx).each(types.FactRelationsBySourcePrefix(counterFactID), func(key, value []byte) error {
		rel, err := decodeContestRelation(key, value)
		if err != nil {
			return err
		}
		if rel.Relation == types.RelationType_RELATION_TYPE_CONTRADICTS {
			targets = append(targets, rel.TargetFactId)
		}
		return nil
	})
	if err != nil {
		emitContestCheckIncomplete(ctx, counterFactID, fmt.Errorf("counter-fact relations: %w", err))
		return nil
	}
	for _, id := range targets {
		target, found, err := k.getFactChecked(ctx, id)
		if err != nil {
			emitContestCheckIncomplete(ctx, id, fmt.Errorf("target: %w", err))
			continue
		}
		if !found || target.Status != types.FactStatus_FACT_STATUS_CONTESTED {
			continue
		}
		released, err := k.releaseContested(ctx, target, causeCounterfactDisproven, counterFactID)
		if err != nil {
			return err
		}
		if released {
			sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(sdk.NewEvent(
				"zerone.knowledge.contradiction_reversed",
				sdk.NewAttribute("fact_id", target.Id),
				sdk.NewAttribute("reverted_by_fact", counterFactID),
			))
		}
	}
	return nil
}
