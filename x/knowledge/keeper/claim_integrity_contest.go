package keeper

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
//
// Review fixes (group G1): a survival is judged from the height its challenge
// opened, never from the height it ended, so a refutation while the challenge
// was open is not cleared by it (SEC-1); a refutation at that height counts
// against the fact (F1); and an EXPIRED counter-fact still holds its target
// (SEC-5). R2 is read before R1, and a reason either finds is conclusive
// (second pass). An entry under a fact's prefix whose payload names another
// fact is skipped rather than failing the check (third pass).
//
// Review G1R: a check that cannot finish never releases, at a challenge's end
// too; G1's SEC-6 fallback, which restored the standing before the challenge,
// is reverted because any read an attacker can inflate made it a laundering
// lever. R2 reads a claim-born fact's dependencies by point lookups from its
// claim, so it never reads what others write under the fact's prefix; a
// failed item inside either rule no longer stops the rule.

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

// counterFactStands reports whether a contradicting fact still holds its
// target (R1): it stands, or it is EXPIRED. Patronage or metabolism can
// revive an EXPIRED fact, so it has not fallen (decision (b), SEC-5); only
// SUPERSEDED, DISPROVEN, REVOKED and PRUNED have. Admission keeps factStands.
func counterFactStands(status types.FactStatus) bool {
	return factStands(status) || status == types.FactStatus_FACT_STATUS_EXPIRED
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
// (relation, fact, claim or transition) counts toward the ceilings that
// hasOtherActiveChallenge uses: 100k records and 64 MiB in total.
type contestScan struct {
	store          corestore.KVStore
	records, bytes int
}

// contestScanMaxRecords and contestScanMaxBytes are the ceilings of one
// contest check: recordInventoryMaxRecords and recordInventoryMaxBytes. Only
// tests change them, to reach the bound without writing 100k records or
// 64 MiB.
var (
	contestScanMaxRecords = recordInventoryMaxRecords
	contestScanMaxBytes   = recordInventoryMaxBytes
)

func (k Keeper) newContestScan(ctx context.Context) *contestScan {
	return &contestScan{store: k.storeService.OpenKVStore(ctx)}
}

func (s *contestScan) charge(size int) error {
	s.records++
	if s.records > contestScanMaxRecords {
		return fmt.Errorf("contest scan exceeds record bound")
	}
	if size > contestScanMaxBytes-s.bytes {
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

// claim reads and validates one claim record. It is nil when absent.
func (s *contestScan) claim(id string) (*types.Claim, error) {
	key := types.ClaimKey(id)
	bz, err := s.store.Get(key)
	if err != nil {
		return nil, err
	}
	if err = s.charge(len(key) + len(bz)); err != nil || bz == nil {
		return nil, err
	}
	if err = validateRawClaimRecord(bz); err != nil {
		return nil, err
	}
	claim := &types.Claim{}
	if err = proto.Unmarshal(bz, claim); err != nil {
		return nil, err
	}
	if claim.Id != id || hasUnknownRecordFields(claim.ProtoReflect()) {
		return nil, fmt.Errorf("claim identity or fields are invalid")
	}
	return claim, nil
}

// latestTransition is factID's latest recorded transition that match
// accepts, or nil when none is recorded.
func (s *contestScan) latestTransition(factID string, match func(*types.StatusTransition) bool) (*types.StatusTransition, error) {
	var latest *types.StatusTransition
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
			latest = t
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return latest, nil
}

// disprovenAt is the height of the fact's latest transition to DISPROVEN, or
// 0 if none is recorded.
func (s *contestScan) disprovenAt(factID string) (uint64, error) {
	t, err := s.latestTransition(factID, func(t *types.StatusTransition) bool {
		return t.NewStatus == types.FactStatus_FACT_STATUS_DISPROVEN
	})
	if err != nil || t == nil {
		return 0, err
	}
	return t.BlockHeight, nil
}

// lastSurvivedAt is the evidence cutoff of the fact's latest survival: the
// height its challenge opened, the SubmittedAtBlock of the challenge claim
// that the latest challenge_survived transition names (SEC-1). It is the
// cutoff challengeEndStatus judged that survival with, so a refutation while
// that challenge was open is never cleared by it. It is 0 when no survival is
// recorded; knowledge 11 recorded survivals as "unknown", so they clear
// nothing. A survival whose challenge claim is missing, unreadable, names
// another fact or opened after the survival is an error.
func (s *contestScan) lastSurvivedAt(factID string) (uint64, error) {
	survival, err := s.latestTransition(factID, func(t *types.StatusTransition) bool {
		return t.CauseEventType == causeChallengeSurvived
	})
	if err != nil || survival == nil {
		return 0, err
	}
	id := survival.CauseId
	if id == "" {
		return 0, fmt.Errorf("survival at %d names no challenge claim", survival.BlockHeight)
	}
	challenge, err := s.claim(id)
	if err != nil {
		return 0, fmt.Errorf("survival challenge claim %s: %w", id, err)
	}
	if challenge == nil {
		return 0, fmt.Errorf("survival challenge claim %s is missing", id)
	}
	if challenge.ProvisionalFactId != factID {
		return 0, fmt.Errorf("survival challenge claim %s challenged %s", id, challenge.ProvisionalFactId)
	}
	if challenge.SubmittedAtBlock == 0 || challenge.SubmittedAtBlock > survival.BlockHeight {
		return 0, fmt.Errorf("survival challenge claim %s opened at %d, survived at %d", id, challenge.SubmittedAtBlock, survival.BlockHeight)
	}
	return challenge.SubmittedAtBlock, nil
}

// maxReportedContestFailures caps the failed reads one rule names in
// contest_check_incomplete; the rest are counted.
const maxReportedContestFailures = 8

// contestFailures collects the failed reads of one rule. A failed item never
// stops the rule (review G1R): a reason found after it is conclusive. A scan
// that stopped (the bound, or an iterator error) is always named.
type contestFailures struct {
	named   []error
	more    int
	stopped error
}

func (f *contestFailures) add(err error) {
	if len(f.named) < maxReportedContestFailures {
		f.named = append(f.named, err)
		return
	}
	f.more++
}

func (f *contestFailures) stop(err error) {
	if f.stopped == nil {
		f.stopped = err
	}
}

// err names the collected failures, or is nil when none was collected.
func (f *contestFailures) err() error {
	parts := make([]string, 0, len(f.named)+2)
	for _, err := range f.named {
		parts = append(parts, err.Error())
	}
	if f.more > 0 {
		parts = append(parts, fmt.Sprintf("and %d more", f.more))
	}
	if f.stopped != nil {
		parts = append(parts, f.stopped.Error())
	}
	if len(parts) == 0 {
		return nil
	}
	return errors.New(strings.Join(parts, "; "))
}

// contestReason decides R2, then R1 (plan §5.3). survivedAt, the evidence
// cutoff of the fact's last survival (0 if none), is called at most once, and
// only when a refuted dependency makes it matter.
//
// A reason found by either rule is conclusive: it is returned with no error,
// whatever else could not be read. Inside each rule a failed item (a
// malformed or unreadable record, an edge naming a missing fact) is recorded
// and the rule goes on, so a reason found after it is still conclusive
// (review G1R). Only when neither rule finds a reason does a failure make the
// check incomplete, and the error then names the failed reads, up to
// maxReportedContestFailures per rule. Both rules share one scan budget, and
// R2 is read first (review G1, second pass): R1 reads every incoming edge of
// every type, which any later fact can add.
//
// R2 reads nothing others can grow cheaply (review G1R). For a fact accepted from a
// claim it reads that claim and, for each target the claim names, the stored
// edge F→target by key: one read per named target (the claim is one record
// its submitter wrote, at most types.MaxRelationsPerClaim relations since
// that cap), plus the status histories of F and of each refuted dependency,
// which grow only by status changes. F's claim wrote every outgoing edge of
// F; the one other writer, before activation, was an augmentation whose ID
// equalled F's, and its REFORMULATES edge is outside the cascade set. So the
// point reads find every dependency a scan of F's forward prefix would find,
// without reading the entries others add under that prefix: a passing
// augmentation wrote a REFORMULATES edge whose source is its user-chosen ID
// (ApplyFinalizedAugmentationVerdict), so long IDs "<F>/…" could fill the
// prefix past the byte bound. Only a fact with no claim (genesis, doctrine or
// an authority injection) has its forward prefix scanned; entries there whose
// payload names another source are skipped, and still charged.
// validateToKStoredRecord rebuilds every key from its payload, so the payload
// names the edge's owner. R1 skips an entry under F's reverse prefix that
// names another target the same way.
//
// The bound can still be reached: by R1, on a fact with very many incoming
// edges, or by the prefix scan of a claimless fact. The check is then
// incomplete and never releases (challengeEndStatus); that is the disclosed
// cost, and nothing an attacker adds can turn it into a release.
func (k Keeper) contestReason(ctx context.Context, fact *types.Fact, survivedAt func(*contestScan) (uint64, error)) (bool, error) {
	s := k.newContestScan(ctx)
	refuted, dependencyErr := s.refutedDependency(fact, survivedAt)
	if refuted {
		return true, nil
	}
	standing, conflictErr := s.standingCounterFact(fact.Id)
	if standing {
		return true, nil
	}
	switch {
	case dependencyErr != nil && conflictErr != nil:
		return true, fmt.Errorf("%w; %w", dependencyErr, conflictErr)
	case dependencyErr != nil:
		return true, dependencyErr
	case conflictErr != nil:
		return true, conflictErr
	}
	return false, nil
}

// refutedDependency decides R2: an outgoing dependency edge (the cascade set)
// to a DISPROVEN fact that fell at or after the opening of the fact's last
// survived challenge. A fall at the opening height counts: heights cannot
// order two events of one block (F1). A dependency that cannot be read is
// recorded and the next is tried; without the survival cutoff no refuted
// dependency can be judged, so a failure to read it ends the rule.
func (s *contestScan) refutedDependency(fact *types.Fact, survivedAt func(*contestScan) (uint64, error)) (bool, error) {
	var failed contestFailures
	var survived uint64
	known := false
	for _, id := range s.dependencies(fact, &failed) {
		dependency, found, err := s.fact(id)
		if err != nil {
			failed.add(fmt.Errorf("dependency %s: %w", id, err))
			continue
		}
		if !found {
			failed.add(fmt.Errorf("dependency %s is missing", id))
			continue
		}
		if dependency.Status != types.FactStatus_FACT_STATUS_DISPROVEN {
			continue
		}
		if !known {
			if survived, err = survivedAt(s); err != nil {
				failed.add(fmt.Errorf("survival history: %w", err))
				return false, failed.err()
			}
			known = true
		}
		if survived == 0 {
			return true, nil
		}
		at, err := s.disprovenAt(id)
		if err != nil {
			failed.add(fmt.Errorf("dependency %s history: %w", id, err))
			continue
		}
		if at >= survived {
			return true, nil
		}
	}
	return false, failed.err()
}

// dependencies lists the targets of fact's stored outgoing edges in the
// cascade set (see contestReason). For a fact accepted from a claim it reads
// the claim and looks up the stored edge to each target the claim names, so
// entries others write under the fact's forward prefix are never read. A
// target whose stored edge is absent or outside the cascade set is not a
// dependency: the stored edge is what the cascade follows. Only a fact with
// no claim has its forward prefix scanned. Failed reads go to failed.
func (s *contestScan) dependencies(fact *types.Fact, failed *contestFailures) []string {
	if fact.ClaimId == "" {
		return s.scannedDependencies(fact.Id, failed)
	}
	claim, err := s.claim(fact.ClaimId)
	if err != nil {
		failed.add(fmt.Errorf("claim %s of %s: %w", fact.ClaimId, fact.Id, err))
		return nil
	}
	if claim == nil {
		failed.add(fmt.Errorf("claim %s of %s is missing", fact.ClaimId, fact.Id))
		return nil
	}
	var ids []string
	named := make(map[string]bool, len(claim.Relations))
	for _, rel := range claim.Relations {
		if rel == nil || named[rel.TargetFactId] {
			continue
		}
		named[rel.TargetFactId] = true
		edge, err := s.relation(types.FactRelationKey(fact.Id, rel.TargetFactId))
		if err != nil {
			failed.add(fmt.Errorf("outgoing relation %s→%s: %w", fact.Id, rel.TargetFactId, err))
			continue
		}
		// Keys are "<source>/<target>", so F→"x/y" and "<F>/x"→y share a
		// key; an entry whose payload names another pair is not F's edge.
		if edge != nil && edge.SourceFactId == fact.Id && edge.TargetFactId == rel.TargetFactId && isCascadeRelation(edge.Relation) {
			ids = append(ids, rel.TargetFactId)
		}
	}
	return ids
}

// scannedDependencies is dependencies for a fact with no claim: a scan of its
// forward prefix, skipping entries whose payload names another source.
func (s *contestScan) scannedDependencies(factID string, failed *contestFailures) []string {
	var ids []string
	err := s.each(types.FactRelationsBySourcePrefix(factID), func(key, value []byte) error {
		rel, err := decodeContestRelation(key, value)
		if err != nil {
			failed.add(fmt.Errorf("outgoing relation: %w", err))
			return nil
		}
		if rel.SourceFactId != factID {
			return nil // another source's edge (see contestReason)
		}
		if isCascadeRelation(rel.Relation) {
			ids = append(ids, rel.TargetFactId)
		}
		return nil
	})
	if err != nil {
		failed.stop(err)
	}
	return ids
}

// relation reads and validates the relation stored at key. It is nil when
// absent.
func (s *contestScan) relation(key []byte) (*types.FactRelation, error) {
	bz, err := s.store.Get(key)
	if err != nil {
		return nil, err
	}
	if err = s.charge(len(key) + len(bz)); err != nil || bz == nil {
		return nil, err
	}
	return decodeContestRelation(key, bz)
}

// standingCounterFact decides R1: an incoming CONTRADICTS edge from a fact
// that has not fallen (counterFactStands). It reads every incoming edge, of
// any type, until it finds one. An entry that cannot be read, or that names a
// missing or unreadable fact, is recorded and the scan goes on.
func (s *contestScan) standingCounterFact(factID string) (bool, error) {
	var failed contestFailures
	err := s.each(types.FactRelationsByTargetPrefix(factID), func(key, value []byte) error {
		rel, err := decodeContestRelation(key, value)
		if err != nil {
			failed.add(fmt.Errorf("incoming relation: %w", err))
			return nil
		}
		if rel.TargetFactId != factID {
			return nil // another target's edge (see contestReason)
		}
		if rel.Relation != types.RelationType_RELATION_TYPE_CONTRADICTS {
			return nil
		}
		source, found, err := s.fact(rel.SourceFactId)
		if err != nil {
			failed.add(fmt.Errorf("contradicting fact %s: %w", rel.SourceFactId, err))
			return nil
		}
		if !found {
			failed.add(fmt.Errorf("contradicting fact %s is missing", rel.SourceFactId))
			return nil
		}
		if counterFactStands(source.Status) {
			return errContestReasonFound
		}
		return nil
	})
	if errors.Is(err, errContestReasonFound) {
		return true, nil
	}
	if err != nil {
		failed.stop(err)
	}
	return false, failed.err()
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
// R1, an incoming CONTRADICTS edge from a fact that has not fallen
// (counterFactStands); or R2, an outgoing dependency edge to a DISPROVEN fact
// that fell at or after survivedAt (or any such fall when survivedAt is 0).
// survivedAt is the opening height of a survived challenge. Pending
// contradictions are not reasons. A reason either rule finds is conclusive,
// even after failed items (see contestReason for what each rule reads). It
// never returns an error: when neither finds one, a read error, a malformed
// record or the scan bound means the reason remains, and
// contest_check_incomplete says why. It writes nothing.
func (k Keeper) contestReasonRemains(ctx context.Context, fact *types.Fact, survivedAt uint64) bool {
	return k.contestReasonOrIncomplete(ctx, fact, func(*contestScan) (uint64, error) { return survivedAt, nil })
}

// contestReasonRemainsSinceLastSurvival is contestReasonRemains with
// survivedAt read from the fact's history and its last survival's challenge
// claim (lastSurvivedAt).
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

// challengeEndStatus is the status a challenged fact takes when challenge
// ends without refuting it (§5.2 rows 7 and 8): RestoredStatusFor, except
// that with claim integrity on an established fact returns to CONTESTED while
// a contest reason remains. A conjecture is exempt and returns to
// PROVISIONAL.
//
// A survival (REJECT) clears only dependencies refuted before this challenge
// opened (its SubmittedAtBlock, the evidence the panel was given; SEC-1), and
// lastSurvivedAt later reads the same height. Any other ending clears
// nothing, so the reasons are judged since the fact's last survival. A
// standing counter-fact is never cleared by a survival (§9 D2).
//
// A check that cannot finish never releases: the fact returns to CONTESTED,
// whatever its standing before the challenge, and contest_check_incomplete
// says why (review G1R). G1's SEC-6 fallback restored the standing before the
// challenge instead, and that made every read an attacker can inflate a
// laundering lever: long augmentation IDs extending F's ID filled the byte
// bound of R2's prefix scan, and a shielded F returned to ACTIVE although a
// dependency fell while its challenge was open. The disclosed cost: a fact
// whose check is too large to finish, such as one with more than 100k
// incoming relations, stays CONTESTED once a challenge of it ends; only a
// challenge that refutes it moves it.
func (k Keeper) challengeEndStatus(ctx context.Context, fact *types.Fact, challenge *types.Claim, integrity, survived bool) types.FactStatus {
	if !integrity || IsConjecture(fact) {
		return RestoredStatusFor(fact)
	}
	remains := false
	if survived {
		remains = k.contestReasonRemains(ctx, fact, challenge.SubmittedAtBlock)
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
