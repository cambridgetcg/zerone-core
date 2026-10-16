package keeper

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// The contest-reason predicate (plan §5.3, Task 12) on hand-built state. F is
// CONTESTED; every other record is written directly, with explicit heights,
// so each rule and each failure is checked in isolation. The scenarios in
// claim_integrity_contest_test.go run the same predicate through real rounds.

const contestF = "contest-f"

type contestPredicateFixture struct {
	handoffFixture
	t *testing.T
}

func newContestPredicateFixture(t *testing.T) *contestPredicateFixture {
	t.Helper()
	f := &contestPredicateFixture{setupHandoff(t), t}
	require.NoError(t, f.k.EnableRecordIntegrity(f.ctx))
	f.putFact(contestF, types.FactStatus_FACT_STATUS_CONTESTED)
	return f
}

func (f *contestPredicateFixture) putFact(id string, status types.FactStatus) {
	f.t.Helper()
	require.NoError(f.t, f.k.SetFactSkipTransition(f.ctx, &types.Fact{Id: id, ClaimId: id + "-claim", Status: status}))
}

func (f *contestPredicateFixture) putRelation(source, target string, relation types.RelationType) {
	f.t.Helper()
	require.NoError(f.t, f.k.SetFactRelation(f.ctx, &types.FactRelation{SourceFactId: source, TargetFactId: target, Relation: relation}))
}

func (f *contestPredicateFixture) putTransition(factID string, prior, next types.FactStatus, cause string, height uint64) {
	f.t.Helper()
	require.NoError(f.t, f.k.recordStatusTransitionChecked(f.ctx, &types.StatusTransition{
		FactId: factID, PriorStatus: prior, NewStatus: next, CauseEventType: cause, CauseId: "cause-of-" + factID, BlockHeight: height,
	}))
}

// disprove records dependency D refuted at height.
func (f *contestPredicateFixture) disprove(id string, height uint64) {
	f.t.Helper()
	f.putFact(id, types.FactStatus_FACT_STATUS_DISPROVEN)
	f.putTransition(id, types.FactStatus_FACT_STATUS_CHALLENGED, types.FactStatus_FACT_STATUS_DISPROVEN, causeChallengeDisproven, height)
}

// survive records that F survived, at survived, a challenge opened at opened:
// the challenge claim, its challenge_opened transition and its
// challenge_survived transition.
func (f *contestPredicateFixture) survive(opened, survived uint64) {
	f.t.Helper()
	id := fmt.Sprintf("challenge-opened-%d", opened)
	require.NoError(f.t, f.k.SetClaim(f.ctx, &types.Claim{Id: id, Submitter: "challenger", ProvisionalFactId: contestF, SubmittedAtBlock: opened}))
	f.putCausedTransition(types.FactStatus_FACT_STATUS_ACTIVE, types.FactStatus_FACT_STATUS_CHALLENGED, causeChallengeOpened, id, opened)
	f.putCausedTransition(types.FactStatus_FACT_STATUS_CHALLENGED, types.FactStatus_FACT_STATUS_ACTIVE, causeChallengeSurvived, id, survived)
}

func (f *contestPredicateFixture) putCausedTransition(prior, next types.FactStatus, cause, causeID string, height uint64) {
	f.t.Helper()
	require.NoError(f.t, f.k.recordStatusTransitionChecked(f.ctx, &types.StatusTransition{
		FactId: contestF, PriorStatus: prior, NewStatus: next, CauseEventType: cause, CauseId: causeID, BlockHeight: height,
	}))
}

func (f *contestPredicateFixture) factF() *types.Fact {
	f.t.Helper()
	fact, found, err := f.k.getFactChecked(f.ctx, contestF)
	require.NoError(f.t, err)
	require.True(f.t, found)
	return fact
}

// check runs one predicate on a fresh event manager. It returns the verdict
// and every contest_check_incomplete event, and requires that nothing was
// written.
func (f *contestPredicateFixture) check(run func(context.Context, *types.Fact) bool) (bool, []map[string]string) {
	f.t.Helper()
	ctx := f.ctx.WithEventManager(sdk.NewEventManager())
	before := f.snapshot(f.t)
	remains := run(ctx, f.factF())
	require.Equal(f.t, before, f.snapshot(f.t), "the predicate never writes")
	var incomplete []map[string]string
	for _, event := range ctx.EventManager().Events() {
		require.Equal(f.t, "zerone.knowledge.contest_check_incomplete", event.Type, "the predicate emits no other event")
		attrs := map[string]string{}
		for _, attr := range event.Attributes {
			attrs[attr.Key] = attr.Value
		}
		incomplete = append(incomplete, attrs)
	}
	return remains, incomplete
}

func (f *contestPredicateFixture) remainsAt(survivedAt uint64) bool {
	f.t.Helper()
	remains, incomplete := f.check(func(ctx context.Context, fact *types.Fact) bool {
		return f.k.contestReasonRemains(ctx, fact, survivedAt)
	})
	require.Empty(f.t, incomplete)
	return remains
}

func (f *contestPredicateFixture) remainsSinceLastSurvival() bool {
	f.t.Helper()
	remains, incomplete := f.check(f.k.contestReasonRemainsSinceLastSurvival)
	require.Empty(f.t, incomplete)
	return remains
}

func sortedFactStatuses() []types.FactStatus {
	var out []types.FactStatus
	for value := range types.FactStatus_name {
		out = append(out, types.FactStatus(value))
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func sortedRelationTypes() []types.RelationType {
	var out []types.RelationType
	for value := range types.RelationType_name {
		if types.RelationType(value) != types.RelationType_RELATION_TYPE_UNSPECIFIED {
			out = append(out, types.RelationType(value))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

var cascadeRelationTypes = map[types.RelationType]bool{
	types.RelationType_RELATION_TYPE_SUPPORTS:    true,
	types.RelationType_RELATION_TYPE_REQUIRES:    true,
	types.RelationType_RELATION_TYPE_REFINES:     true,
	types.RelationType_RELATION_TYPE_GENERALIZES: true,
	types.RelationType_RELATION_TYPE_CITES:       true,
}

// R1: an incoming CONTRADICTS edge whose source has not fallen (VERIFIED,
// ACTIVE, CONTESTED, CHALLENGED, AT_RISK or EXPIRED; an EXPIRED fact can be
// revived, decision (b)). Other incoming types, and F's own outgoing
// CONTRADICTS edges, are not reasons.
func TestContestReasonStandingCounterFact(t *testing.T) {
	standing := map[types.FactStatus]bool{
		types.FactStatus_FACT_STATUS_VERIFIED:   true,
		types.FactStatus_FACT_STATUS_ACTIVE:     true,
		types.FactStatus_FACT_STATUS_CONTESTED:  true,
		types.FactStatus_FACT_STATUS_CHALLENGED: true,
		types.FactStatus_FACT_STATUS_AT_RISK:    true,
		types.FactStatus_FACT_STATUS_EXPIRED:    true,
	}
	for _, status := range sortedFactStatuses() {
		t.Run(status.String(), func(t *testing.T) {
			f := newContestPredicateFixture(t)
			f.putFact("counter-k", status)
			f.putRelation("counter-k", contestF, types.RelationType_RELATION_TYPE_CONTRADICTS)
			require.Equal(t, standing[status], f.remainsAt(0))
			require.Equal(t, standing[status], f.remainsSinceLastSurvival())
		})
	}
	for _, relation := range sortedRelationTypes() {
		if relation == types.RelationType_RELATION_TYPE_CONTRADICTS {
			continue
		}
		t.Run("incoming "+relation.String(), func(t *testing.T) {
			f := newContestPredicateFixture(t)
			f.putFact("source-s", types.FactStatus_FACT_STATUS_VERIFIED)
			f.putRelation("source-s", contestF, relation)
			require.False(t, f.remainsAt(0))
		})
	}
	t.Run("outgoing CONTRADICTS", func(t *testing.T) {
		f := newContestPredicateFixture(t)
		f.putFact("other-o", types.FactStatus_FACT_STATUS_VERIFIED)
		f.putRelation(contestF, "other-o", types.RelationType_RELATION_TYPE_CONTRADICTS)
		require.False(t, f.remainsAt(0))
	})
	t.Run("no relations", func(t *testing.T) {
		require.False(t, newContestPredicateFixture(t).remainsAt(0))
	})
}

// R2: an outgoing dependency edge (the cascade set) to a DISPROVEN fact D,
// unless F survived a direct challenge that opened after D's latest
// transition to DISPROVEN. survivedAt is that opening height; a fall at the
// same height counts against F (F1: heights cannot order two events of one
// block). A D with no recorded fall is cleared by any survival.
func TestContestReasonRefutedDependencyIsOrderedBySurvival(t *testing.T) {
	for _, relation := range sortedRelationTypes() {
		t.Run(relation.String(), func(t *testing.T) {
			f := newContestPredicateFixture(t)
			f.disprove("dependency-d", 50)
			f.putRelation(contestF, "dependency-d", relation)
			if !cascadeRelationTypes[relation] {
				require.False(t, f.remainsAt(0), "only dependency edges carry a refutation")
				return
			}
			for survivedAt, remains := range map[uint64]bool{0: true, 49: true, 50: true, 51: false} {
				require.Equal(t, remains, f.remainsAt(survivedAt), "survivedAt %d", survivedAt)
			}
		})
	}
	for _, status := range sortedFactStatuses() {
		if status == types.FactStatus_FACT_STATUS_DISPROVEN {
			continue
		}
		t.Run("dependency "+status.String(), func(t *testing.T) {
			f := newContestPredicateFixture(t)
			f.putFact("dependency-d", status)
			f.putRelation(contestF, "dependency-d", types.RelationType_RELATION_TYPE_REQUIRES)
			require.False(t, f.remainsAt(0))
		})
	}
	t.Run("no recorded fall", func(t *testing.T) {
		f := newContestPredicateFixture(t)
		f.putFact("dependency-d", types.FactStatus_FACT_STATUS_DISPROVEN)
		f.putRelation(contestF, "dependency-d", types.RelationType_RELATION_TYPE_REQUIRES)
		require.True(t, f.remainsAt(0))
		require.False(t, f.remainsAt(1))
	})
	t.Run("latest fall counts", func(t *testing.T) {
		f := newContestPredicateFixture(t)
		f.disprove("dependency-d", 30)
		f.putTransition("dependency-d", types.FactStatus_FACT_STATUS_DISPROVEN, types.FactStatus_FACT_STATUS_CONTESTED, "synthetic", 40)
		f.putTransition("dependency-d", types.FactStatus_FACT_STATUS_CONTESTED, types.FactStatus_FACT_STATUS_DISPROVEN, causeChallengeDisproven, 70)
		f.putRelation(contestF, "dependency-d", types.RelationType_RELATION_TYPE_CITES)
		require.True(t, f.remainsAt(50))
		require.True(t, f.remainsAt(70))
		require.False(t, f.remainsAt(71))
	})
	t.Run("one uncleared dependency is enough", func(t *testing.T) {
		f := newContestPredicateFixture(t)
		f.disprove("dependency-a", 20)
		f.disprove("dependency-b", 80)
		f.putRelation(contestF, "dependency-a", types.RelationType_RELATION_TYPE_REQUIRES)
		f.putRelation(contestF, "dependency-b", types.RelationType_RELATION_TYPE_SUPPORTS)
		require.True(t, f.remainsAt(50))
		require.True(t, f.remainsAt(80))
		require.False(t, f.remainsAt(81))
	})
}

// Without a supplied height, survivedAt is the opening height of the
// challenge behind F's latest challenge_survived transition: its challenge
// claim's SubmittedAtBlock (SEC-1). A refutation while that challenge was
// open (65), at its survival (70), or at its opening height (60, F1) is not
// cleared. An inconclusive challenge or a knowledge-11 survival (recorded as
// "unknown") clears nothing.
func TestContestReasonSinceLastSurvival(t *testing.T) {
	for _, tc := range []struct {
		fall    uint64
		remains bool
	}{{55, false}, {59, false}, {60, true}, {65, true}, {70, true}, {85, true}} {
		t.Run(fmt.Sprintf("fall at %d", tc.fall), func(t *testing.T) {
			f := newContestPredicateFixture(t)
			f.survive(40, 45)
			f.survive(60, 70)
			f.putTransition(contestF, types.FactStatus_FACT_STATUS_CHALLENGED, types.FactStatus_FACT_STATUS_ACTIVE, causeChallengeInconclusive, 80)
			f.putTransition(contestF, types.FactStatus_FACT_STATUS_CHALLENGED, types.FactStatus_FACT_STATUS_ACTIVE, "unknown", 90)
			f.disprove("dependency-d", tc.fall)
			f.putRelation(contestF, "dependency-d", types.RelationType_RELATION_TYPE_REQUIRES)
			require.Equal(t, tc.remains, f.remainsSinceLastSurvival())
		})
	}
	t.Run("never survived", func(t *testing.T) {
		f := newContestPredicateFixture(t)
		f.disprove("dependency-d", 10)
		f.putRelation(contestF, "dependency-d", types.RelationType_RELATION_TYPE_REQUIRES)
		require.True(t, f.remainsSinceLastSurvival())
	})
}

// Plan §5.3 failure handling. The base state has no reason: an incoming
// SUPPORTS edge, a refuted counter-fact contradicting F, and a dependency
// refuted at 50 that F survived at 60. Each fault makes the check unable to
// finish, so the reason remains and exactly one contest_check_incomplete
// event names F and the cause.
func TestContestReasonFailsTowardContested(t *testing.T) {
	putRaw := func(f *contestPredicateFixture, key, value []byte) { f.ctx.KVStore(f.keys[0]).Set(key, value) }
	relationBytes := func(source, target string, relation types.RelationType) []byte {
		bz, err := proto.Marshal(&types.FactRelation{SourceFactId: source, TargetFactId: target, Relation: relation})
		require.NoError(t, err)
		return bz
	}
	for _, tc := range []struct {
		name   string
		inject func(*contestPredicateFixture)
		reason string
	}{
		{"unreadable incoming relation", func(f *contestPredicateFixture) {
			putRaw(f, types.FactRelationReverseKey(contestF, "garbled"), []byte{0xff})
		}, "incoming relation"},
		{"incoming relation stored under another key", func(f *contestPredicateFixture) {
			putRaw(f, types.FactRelationReverseKey(contestF, "listed"), relationBytes("elsewhere", contestF, types.RelationType_RELATION_TYPE_CONTRADICTS))
		}, "key/payload mismatch"},
		{"missing contradicting fact", func(f *contestPredicateFixture) {
			f.putRelation("ghost-k", contestF, types.RelationType_RELATION_TYPE_CONTRADICTS)
		}, "contradicting fact ghost-k is missing"},
		{"unreadable contradicting fact", func(f *contestPredicateFixture) {
			putRaw(f, types.FactKey("refuted-k"), []byte{0xff})
		}, "contradicting fact refuted-k"},
		{"read error on contradicting fact", func(f *contestPredicateFixture) {
			f.kFault.op, f.kFault.prefix = "get", types.FactKey("refuted-k")
		}, "injected read failure"},
		{"unreadable outgoing relation", func(f *contestPredicateFixture) {
			putRaw(f, types.FactRelationKey(contestF, "garbled"), []byte{0xff})
		}, "outgoing relation"},
		{"missing dependency", func(f *contestPredicateFixture) {
			f.putRelation(contestF, "ghost-d", types.RelationType_RELATION_TYPE_REQUIRES)
		}, "dependency ghost-d is missing"},
		{"unreadable dependency history", func(f *contestPredicateFixture) {
			putRaw(f, types.StatusTransitionKey("dependency-d", 1), []byte{0xff})
		}, "dependency dependency-d history"},
		{"unreadable survival history", func(f *contestPredicateFixture) {
			putRaw(f, types.StatusTransitionKey(contestF, 1), []byte{0xff})
		}, "survival history"},
		{"survival challenge claim missing", func(f *contestPredicateFixture) {
			f.ctx.KVStore(f.keys[0]).Delete(types.ClaimKey("challenge-opened-60"))
		}, "survival challenge claim challenge-opened-60 is missing"},
		{"unreadable survival challenge claim", func(f *contestPredicateFixture) {
			putRaw(f, types.ClaimKey("challenge-opened-60"), []byte{0xff})
		}, "survival challenge claim challenge-opened-60"},
		{"survival challenge claim names another fact", func(f *contestPredicateFixture) {
			require.NoError(f.t, f.k.SetClaim(f.ctx, &types.Claim{Id: "challenge-opened-60", Submitter: "challenger", ProvisionalFactId: "elsewhere", SubmittedAtBlock: 60}))
		}, "survival challenge claim challenge-opened-60 challenged elsewhere"},
		{"survival challenge claim opened after the survival", func(f *contestPredicateFixture) {
			require.NoError(f.t, f.k.SetClaim(f.ctx, &types.Claim{Id: "challenge-opened-60", Submitter: "challenger", ProvisionalFactId: contestF, SubmittedAtBlock: 66}))
		}, "survival challenge claim challenge-opened-60 opened at 66"},
		{"survival with no challenge claim ID", func(f *contestPredicateFixture) {
			f.putCausedTransition(types.FactStatus_FACT_STATUS_CHALLENGED, types.FactStatus_FACT_STATUS_ACTIVE, causeChallengeSurvived, "", 90)
		}, "survival at 90 names no challenge claim"},
		{"read error on survival challenge claim", func(f *contestPredicateFixture) {
			f.kFault.op, f.kFault.prefix = "get", types.ClaimKey("challenge-opened-60")
		}, "injected read failure"},
		{"record bound", func(f *contestPredicateFixture) {
			for i := 0; i <= recordInventoryMaxRecords; i++ {
				f.putRelation(fmt.Sprintf("bound-%06d", i), contestF, types.RelationType_RELATION_TYPE_CITES)
			}
		}, "contest scan exceeds record bound"},
		{"byte bound", func(f *contestPredicateFixture) {
			putRaw(f, types.FactRelationReverseKey(contestF, "oversized"), make([]byte, recordInventoryMaxBytes))
		}, "contest scan exceeds byte bound"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newContestPredicateFixture(t)
			f.putFact("supporter-s", types.FactStatus_FACT_STATUS_VERIFIED)
			f.putRelation("supporter-s", contestF, types.RelationType_RELATION_TYPE_SUPPORTS)
			f.putFact("refuted-k", types.FactStatus_FACT_STATUS_DISPROVEN)
			f.putRelation("refuted-k", contestF, types.RelationType_RELATION_TYPE_CONTRADICTS)
			f.disprove("dependency-d", 50)
			f.putRelation(contestF, "dependency-d", types.RelationType_RELATION_TYPE_REQUIRES)
			f.survive(60, 65)
			require.False(t, f.remainsSinceLastSurvival(), "the base state has no contest reason")

			tc.inject(f)
			remains, incomplete := f.check(f.k.contestReasonRemainsSinceLastSurvival)
			require.True(t, remains, "an incomplete check means the reason remains")
			require.Len(t, incomplete, 1)
			require.Equal(t, contestF, incomplete[0]["fact_id"])
			require.Contains(t, incomplete[0]["reason"], tc.reason)
			require.False(t, strings.Contains(incomplete[0]["reason"], "contest reason found"))
		})
	}
}

// A reason found by either rule is conclusive, whatever the other rule could
// read (review G1, second pass). R2 is read first, so a refuted dependency is
// found even when F has more incoming relations than one check may read (the
// heavily cited fact behind a shield challenge), and a standing counter-fact is
// still found when F's dependencies cannot be read. Only when neither rule
// finds a reason does a failed read make the check incomplete, and then one
// event carries every failed read.
func TestContestReasonFindingIsConclusive(t *testing.T) {
	const bound = 8
	cited := func(t *testing.T) *contestPredicateFixture {
		f := newContestPredicateFixture(t)
		SetContestScanMaxRecordsForTest(t, bound)
		for i := 0; i <= bound; i++ {
			f.putRelation(fmt.Sprintf("citing-%02d", i), contestF, types.RelationType_RELATION_TYPE_CITES)
		}
		return f
	}
	t.Run("refuted dependency past the bound", func(t *testing.T) {
		f := cited(t)
		f.disprove("dependency-d", 50)
		f.putRelation(contestF, "dependency-d", types.RelationType_RELATION_TYPE_REQUIRES)
		for _, survivedAt := range []uint64{0, 49, 50} {
			require.True(t, f.remainsAt(survivedAt), "survivedAt %d", survivedAt)
		}
		require.True(t, f.remainsSinceLastSurvival(), "never survived")
	})
	t.Run("cleared dependency past the bound", func(t *testing.T) {
		f := cited(t)
		f.disprove("dependency-d", 50)
		f.putRelation(contestF, "dependency-d", types.RelationType_RELATION_TYPE_REQUIRES)
		remains, incomplete := f.check(func(ctx context.Context, fact *types.Fact) bool {
			return f.k.contestReasonRemains(ctx, fact, 51)
		})
		require.True(t, remains, "with no reason found, the bound still means the reason remains")
		require.Len(t, incomplete, 1)
		require.Contains(t, incomplete[0]["reason"], "contest scan exceeds record bound")
	})
	for _, tc := range []struct {
		name   string
		inject func(*contestPredicateFixture)
	}{
		{"unreadable outgoing relation", func(f *contestPredicateFixture) {
			f.ctx.KVStore(f.keys[0]).Set(types.FactRelationKey(contestF, "garbled"), []byte{0xff})
		}},
		{"missing dependency", func(f *contestPredicateFixture) {
			f.putRelation(contestF, "ghost-d", types.RelationType_RELATION_TYPE_REQUIRES)
		}},
		{"unreadable survival history", func(f *contestPredicateFixture) {
			f.disprove("dependency-d", 50)
			f.putRelation(contestF, "dependency-d", types.RelationType_RELATION_TYPE_REQUIRES)
			f.ctx.KVStore(f.keys[0]).Set(types.StatusTransitionKey(contestF, 1), []byte{0xff})
		}},
	} {
		t.Run("standing counter-fact past "+tc.name, func(t *testing.T) {
			f := newContestPredicateFixture(t)
			f.putFact("counter-k", types.FactStatus_FACT_STATUS_VERIFIED)
			f.putRelation("counter-k", contestF, types.RelationType_RELATION_TYPE_CONTRADICTS)
			tc.inject(f)
			require.True(t, f.remainsSinceLastSurvival())
		})
	}
	t.Run("neither rule finishes", func(t *testing.T) {
		f := newContestPredicateFixture(t)
		f.ctx.KVStore(f.keys[0]).Set(types.FactRelationKey(contestF, "garbled"), []byte{0xff})
		f.ctx.KVStore(f.keys[0]).Set(types.FactRelationReverseKey(contestF, "garbled"), []byte{0xff})
		remains, incomplete := f.check(f.k.contestReasonRemainsSinceLastSurvival)
		require.True(t, remains)
		require.Len(t, incomplete, 1)
		require.Contains(t, incomplete[0]["reason"], "outgoing relation")
		require.Contains(t, incomplete[0]["reason"], "incoming relation")
	})
}

// An entry under F's prefix that names another fact is not one of F's edges
// (review G1, third pass). validateToKStoredRecord rebuilds the key from the
// payload, so such an entry's source (or target, in the reverse index) starts
// with "<F>/": a passing augmentation whose user-chosen ID extends F's writes
// one (ApplyFinalizedAugmentationVerdict). Both rules skip it, so it neither
// hides a refuted dependency nor becomes a reason. It is still read, so it
// still counts toward the bound.
func TestContestReasonSkipsForeignEdgesUnderTheFactsPrefix(t *testing.T) {
	foreign := contestF + "/0"
	t.Run("foreign source before a refuted dependency", func(t *testing.T) {
		f := newContestPredicateFixture(t)
		f.disprove("dependency-d", 50)
		f.putRelation(foreign, "dependency-d", types.RelationType_RELATION_TYPE_REFORMULATES)
		f.putRelation(contestF, "dependency-d", types.RelationType_RELATION_TYPE_REQUIRES)
		for _, survivedAt := range []uint64{0, 50} {
			require.True(t, f.remainsAt(survivedAt), "survivedAt %d", survivedAt)
		}
		require.False(t, f.remainsAt(51), "cleared by a later survival")
		require.True(t, f.remainsSinceLastSurvival(), "never survived")
	})
	t.Run("foreign dependency edge is not F's", func(t *testing.T) {
		f := newContestPredicateFixture(t)
		f.disprove("dependency-x", 50)
		f.putRelation(foreign, "dependency-x", types.RelationType_RELATION_TYPE_REQUIRES)
		require.False(t, f.remainsAt(0))
		require.False(t, f.remainsSinceLastSurvival())
	})
	t.Run("foreign target under the reverse prefix", func(t *testing.T) {
		f := newContestPredicateFixture(t)
		f.putFact("counter-k", types.FactStatus_FACT_STATUS_VERIFIED)
		f.putRelation("counter-k", foreign, types.RelationType_RELATION_TYPE_CONTRADICTS)
		require.False(t, f.remainsSinceLastSurvival())
	})
	t.Run("foreign entries count toward the bound", func(t *testing.T) {
		const bound = 8
		f := newContestPredicateFixture(t)
		SetContestScanMaxRecordsForTest(t, bound)
		for i := 0; i <= bound; i++ {
			f.putRelation(fmt.Sprintf("%s/%02d", contestF, i), "dependency-x", types.RelationType_RELATION_TYPE_REFORMULATES)
		}
		remains, incomplete := f.check(f.k.contestReasonRemainsSinceLastSurvival)
		require.True(t, remains, "with no reason found, the bound still means the reason remains")
		require.Len(t, incomplete, 1)
		require.Contains(t, incomplete[0]["reason"], "contest scan exceeds record bound")
	})
}

// contestEventsOf returns the attributes of every event of eventType on ctx.
func contestEventsOf(ctx sdk.Context, eventType string) []map[string]string {
	var out []map[string]string
	for _, event := range ctx.EventManager().Events() {
		if event.Type != eventType {
			continue
		}
		attrs := map[string]string{}
		for _, attr := range event.Attributes {
			attrs[attr.Key] = attr.Value
		}
		out = append(out, attrs)
	}
	return out
}

// requireLastTransition checks F's newest recorded transition.
func (f *contestPredicateFixture) requireLastTransition(prior, next types.FactStatus, cause, causeID string) {
	f.t.Helper()
	history := f.k.GetStatusHistory(f.ctx, contestF)
	require.NotEmpty(f.t, history)
	last := history[len(history)-1]
	require.Equal(f.t, []any{prior, next, cause, causeID}, []any{last.PriorStatus, last.NewStatus, last.CauseEventType, last.CauseId})
}

// contestAcceptedTargets (§5.2 row 2) on hand-built state. Counter-fact
// counter-k was created from counter-claim, whose relations name
// CONTRADICTS→F. F becomes CONTESTED, with cause
// contradiction_accepted/counter-claim and one contradiction_accepted event,
// only when F is VERIFIED or ACTIVE and the stored edge counter-k→F is
// CONTRADICTS: the edge R1 reads. Otherwise nothing is written or emitted.
// A stored edge that cannot be read is an error, and nothing is written.
func TestContestAcceptedTargets(t *testing.T) {
	counterClaim := &types.Claim{Id: "counter-claim", Relations: []*types.ClaimRelation{
		{Relation: types.RelationType_RELATION_TYPE_CONTRADICTS, TargetFactId: contestF},
	}}
	contest := func(f *contestPredicateFixture) (sdk.Context, error) {
		ctx := f.ctx.WithEventManager(sdk.NewEventManager())
		return ctx, f.k.contestAcceptedTargets(ctx, counterClaim, "counter-k")
	}
	contestable := map[types.FactStatus]bool{
		types.FactStatus_FACT_STATUS_VERIFIED: true,
		types.FactStatus_FACT_STATUS_ACTIVE:   true,
	}
	for _, status := range sortedFactStatuses() {
		t.Run(status.String(), func(t *testing.T) {
			f := newContestPredicateFixture(t)
			f.putFact(contestF, status)
			f.putFact("counter-k", types.FactStatus_FACT_STATUS_VERIFIED)
			f.putRelation("counter-k", contestF, types.RelationType_RELATION_TYPE_CONTRADICTS)
			before := f.snapshot(t)
			ctx, err := contest(f)
			require.NoError(t, err)
			if !contestable[status] {
				require.Equal(t, before, f.snapshot(t), "a %s target is left as it is", status)
				require.Empty(t, ctx.EventManager().Events())
				return
			}
			require.Equal(t, types.FactStatus_FACT_STATUS_CONTESTED, f.factF().Status)
			f.requireLastTransition(status, types.FactStatus_FACT_STATUS_CONTESTED, causeContradictionAccepted, "counter-claim")
			require.Equal(t, []map[string]string{{"fact_id": contestF, "counter_claim_id": "counter-claim", "counter_fact_id": "counter-k"}},
				contestEventsOf(ctx, "zerone.knowledge.contradiction_accepted"))
		})
	}
	for _, tc := range []struct {
		name   string
		inject func(*contestPredicateFixture)
		err    string
	}{
		{"stored edge is CITES", func(f *contestPredicateFixture) {
			f.putRelation("counter-k", contestF, types.RelationType_RELATION_TYPE_CITES)
		}, ""},
		{"no stored edge", func(*contestPredicateFixture) {}, ""},
		{"stored edge is the reverse direction", func(f *contestPredicateFixture) {
			f.putRelation(contestF, "counter-k", types.RelationType_RELATION_TYPE_CONTRADICTS)
		}, ""},
		{"unreadable stored edge", func(f *contestPredicateFixture) {
			f.ctx.KVStore(f.keys[0]).Set(types.FactRelationReverseKey(contestF, "counter-k"), []byte{0xff})
		}, "counter-fact edge"},
		{"stored edge names another source", func(f *contestPredicateFixture) {
			bz, err := proto.Marshal(&types.FactRelation{SourceFactId: "elsewhere", TargetFactId: contestF, Relation: types.RelationType_RELATION_TYPE_CONTRADICTS})
			require.NoError(f.t, err)
			f.ctx.KVStore(f.keys[0]).Set(types.FactRelationReverseKey(contestF, "counter-k"), bz)
		}, "counter-fact edge"},
		{"read error on stored edge", func(f *contestPredicateFixture) {
			f.kFault.op, f.kFault.prefix = "get", types.FactRelationReverseKey(contestF, "counter-k")
		}, "injected read failure"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newContestPredicateFixture(t)
			f.putFact(contestF, types.FactStatus_FACT_STATUS_VERIFIED)
			f.putFact("counter-k", types.FactStatus_FACT_STATUS_VERIFIED)
			tc.inject(f)
			before := f.snapshot(t)
			ctx, err := contest(f)
			*f.kFault = handoffFault{}
			if tc.err == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.err)
			}
			require.Equal(t, before, f.snapshot(t), "F stays VERIFIED and nothing is written")
			require.Empty(t, ctx.EventManager().Events())
		})
	}
}

// settleContradictionTargets (§5.2 row 4) reads F's last challenge_survived
// height. A dependency refuted before that survival no longer holds F, one
// refuted after it still does, and with no survival any refuted dependency
// holds F. A release records contradiction_reversed/<claim>.
func TestSettleContradictionTargetsSinceLastSurvival(t *testing.T) {
	claim := &types.Claim{Id: "contra-claim", Relations: []*types.ClaimRelation{
		{Relation: types.RelationType_RELATION_TYPE_CONTRADICTS, TargetFactId: contestF},
	}}
	for _, tc := range []struct {
		name     string
		fall     uint64
		survived bool
		released bool
	}{
		{"refuted before the last survival", 50, true, true},
		{"refuted after the last survival", 70, true, false},
		{"never survived", 50, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newContestPredicateFixture(t)
			f.disprove("dependency-d", tc.fall)
			f.putRelation(contestF, "dependency-d", types.RelationType_RELATION_TYPE_REQUIRES)
			if tc.survived {
				f.survive(55, 60)
			}
			before := f.snapshot(t)
			ctx := f.ctx.WithEventManager(sdk.NewEventManager())
			require.NoError(t, f.k.endContradiction(ctx, claim, true))
			require.Empty(t, contestEventsOf(ctx, "zerone.knowledge.contest_check_incomplete"))
			if !tc.released {
				require.Equal(t, before, f.snapshot(t), "R2 holds F")
				require.Empty(t, ctx.EventManager().Events())
				return
			}
			require.Equal(t, types.FactStatus_FACT_STATUS_VERIFIED, f.factF().Status)
			f.requireLastTransition(types.FactStatus_FACT_STATUS_CONTESTED, types.FactStatus_FACT_STATUS_VERIFIED, causeContradictionReversed, "contra-claim")
			require.Equal(t, []map[string]string{{"fact_id": contestF, "reverted_by_claim": "contra-claim"}},
				contestEventsOf(ctx, "zerone.knowledge.contradiction_reversed"))
		})
	}
}

// completeRound reads the claim-integrity marker before any write (plan
// §3.1). A malformed marker is an error, never knowledge 11: the round is not
// completed, nothing is written, and a CONTESTED target stays CONTESTED.
func TestCompleteRoundRefusesMalformedClaimIntegrityMarker(t *testing.T) {
	for _, verdict := range []types.Verdict{
		types.Verdict_VERDICT_ACCEPT,
		types.Verdict_VERDICT_REJECT,
		types.Verdict_VERDICT_MALFORMED,
		types.Verdict_VERDICT_INCONCLUSIVE,
	} {
		t.Run(verdict.String(), func(t *testing.T) {
			f := newContestPredicateFixture(t)
			claim := &types.Claim{
				Id: "contra-claim", Submitter: "contra-author", Status: types.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION,
				Relations: []*types.ClaimRelation{{Relation: types.RelationType_RELATION_TYPE_CONTRADICTS, TargetFactId: contestF}},
			}
			require.NoError(t, f.k.SetClaim(f.ctx, claim))
			f.ctx.KVStore(f.keys[0]).Set([]byte(ClaimIntegrityEnabledStoreKey), []byte{2})
			round := &types.VerificationRound{Id: "contra-round", ClaimId: claim.Id, Phase: types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION}
			want := proto.Clone(round)
			before := f.snapshot(t)
			err := f.k.completeRound(f.ctx, round, &VerificationResult{Verdict: verdict}, true)
			require.ErrorContains(t, err, "malformed claim integrity marker")
			require.True(t, proto.Equal(want, round), "the round is not completed")
			require.Equal(t, before, f.snapshot(t), "nothing is written")
			require.Equal(t, types.FactStatus_FACT_STATUS_CONTESTED, f.factF().Status)
		})
	}
}

// settleCounterFactTargets (§5.2 row 5, Task 13) on hand-built state. Counter-
// fact counter-k has just been disproven. F is released to VERIFIED, with
// cause counterfact_disproven/counter-k and a contradiction_reversed event
// naming counter-k, only when F is CONTESTED, the stored edge counter-k→F is
// CONTRADICTS and no contest reason remains. A conjecture is released to
// PROVISIONAL. Otherwise nothing is written or emitted.
func TestSettleCounterFactTargets(t *testing.T) {
	settle := func(f *contestPredicateFixture) (sdk.Context, error) {
		ctx := f.ctx.WithEventManager(sdk.NewEventManager())
		return ctx, f.k.settleCounterFactTargets(ctx, "counter-k")
	}
	base := func(t *testing.T) *contestPredicateFixture {
		f := newContestPredicateFixture(t)
		f.disprove("counter-k", 70)
		return f
	}
	requireReleased := func(t *testing.T, f *contestPredicateFixture, ctx sdk.Context, next types.FactStatus) {
		require.Equal(t, next, f.factF().Status)
		f.requireLastTransition(types.FactStatus_FACT_STATUS_CONTESTED, next, causeCounterfactDisproven, "counter-k")
		require.Equal(t, []map[string]string{{"fact_id": contestF, "reverted_by_fact": "counter-k"}},
			contestEventsOf(ctx, "zerone.knowledge.contradiction_reversed"))
		require.Len(t, ctx.EventManager().Events(), 1)
	}
	for _, status := range sortedFactStatuses() {
		t.Run("target "+status.String(), func(t *testing.T) {
			f := base(t)
			f.putFact(contestF, status)
			f.putRelation("counter-k", contestF, types.RelationType_RELATION_TYPE_CONTRADICTS)
			before := f.snapshot(t)
			ctx, err := settle(f)
			require.NoError(t, err)
			if status != types.FactStatus_FACT_STATUS_CONTESTED {
				require.Equal(t, before, f.snapshot(t), "a %s target is left as it is", status)
				require.Empty(t, ctx.EventManager().Events())
				return
			}
			requireReleased(t, f, ctx, types.FactStatus_FACT_STATUS_VERIFIED)
		})
	}
	for _, relation := range sortedRelationTypes() {
		t.Run("edge "+relation.String(), func(t *testing.T) {
			f := base(t)
			f.putRelation("counter-k", contestF, relation)
			before := f.snapshot(t)
			ctx, err := settle(f)
			require.NoError(t, err)
			if relation != types.RelationType_RELATION_TYPE_CONTRADICTS {
				require.Equal(t, before, f.snapshot(t), "only a stored CONTRADICTS edge names a target")
				require.Empty(t, ctx.EventManager().Events())
				return
			}
			requireReleased(t, f, ctx, types.FactStatus_FACT_STATUS_VERIFIED)
		})
	}
	t.Run("conjecture target", func(t *testing.T) {
		f := base(t)
		require.NoError(t, f.k.SetFactSkipTransition(f.ctx, &types.Fact{Id: contestF, ClaimId: contestF + "-claim", ClaimType: types.ClaimType_CLAIM_TYPE_CONJECTURE, Status: types.FactStatus_FACT_STATUS_CONTESTED}))
		f.putRelation("counter-k", contestF, types.RelationType_RELATION_TYPE_CONTRADICTS)
		ctx, err := settle(f)
		require.NoError(t, err)
		requireReleased(t, f, ctx, types.FactStatus_FACT_STATUS_PROVISIONAL)
	})
	for _, tc := range []struct {
		name   string
		inject func(*contestPredicateFixture)
	}{
		{"another counter-fact stands", func(f *contestPredicateFixture) {
			f.putFact("counter-k2", types.FactStatus_FACT_STATUS_VERIFIED)
			f.putRelation("counter-k2", contestF, types.RelationType_RELATION_TYPE_CONTRADICTS)
		}},
		{"a dependency fell after the last survival", func(f *contestPredicateFixture) {
			f.survive(55, 60)
			f.disprove("dependency-d", 65)
			f.putRelation(contestF, "dependency-d", types.RelationType_RELATION_TYPE_REQUIRES)
		}},
		{"the fallen counter-fact is also a dependency", func(f *contestPredicateFixture) {
			f.survive(55, 60)
			f.putRelation(contestF, "counter-k", types.RelationType_RELATION_TYPE_CITES)
		}},
	} {
		t.Run("held: "+tc.name, func(t *testing.T) {
			f := base(t)
			f.putRelation("counter-k", contestF, types.RelationType_RELATION_TYPE_CONTRADICTS)
			tc.inject(f)
			before := f.snapshot(t)
			ctx, err := settle(f)
			require.NoError(t, err)
			require.Equal(t, before, f.snapshot(t), "a contest reason remains")
			require.Empty(t, ctx.EventManager().Events())
		})
	}
	t.Run("missing target", func(t *testing.T) {
		f := base(t)
		f.putRelation("counter-k", "ghost-f", types.RelationType_RELATION_TYPE_CONTRADICTS)
		before := f.snapshot(t)
		ctx, err := settle(f)
		require.NoError(t, err)
		require.Equal(t, before, f.snapshot(t))
		require.Empty(t, ctx.EventManager().Events())
	})
}

// settleCounterFactTargets never releases on a check that cannot finish. If
// the fallen counter-fact's relations cannot be listed, no target is released
// and one contest_check_incomplete names the counter-fact. A target that
// cannot be read, or whose own check fails, is reported and left as it is,
// and the other targets still settle. A write error is returned.
func TestSettleCounterFactTargetsFailsTowardContested(t *testing.T) {
	putRaw := func(f *contestPredicateFixture, key, value []byte) { f.ctx.KVStore(f.keys[0]).Set(key, value) }
	for _, tc := range []struct {
		name     string
		inject   func(*contestPredicateFixture)
		named    string
		reason   string
		released bool // the second target, contest-g, still settles
	}{
		{"unreadable counter-fact relation", func(f *contestPredicateFixture) {
			putRaw(f, types.FactRelationKey("counter-k", "garbled"), []byte{0xff})
		}, "counter-k", "counter-fact relations", false},
		{"counter-fact relation stored under another key", func(f *contestPredicateFixture) {
			bz, err := proto.Marshal(&types.FactRelation{SourceFactId: "elsewhere", TargetFactId: contestF, Relation: types.RelationType_RELATION_TYPE_CONTRADICTS})
			require.NoError(f.t, err)
			putRaw(f, types.FactRelationKey("counter-k", "listed"), bz)
		}, "counter-k", "key/payload mismatch", false},
		{"counter-fact relations past the byte bound", func(f *contestPredicateFixture) {
			putRaw(f, types.FactRelationKey("counter-k", "oversized"), make([]byte, recordInventoryMaxBytes))
		}, "counter-k", "contest scan exceeds byte bound", false},
		{"unreadable target", func(f *contestPredicateFixture) {
			putRaw(f, types.FactKey(contestF), []byte{0xff})
		}, contestF, "target", true},
		{"read error on target", func(f *contestPredicateFixture) {
			f.kFault.op, f.kFault.prefix = "get", types.FactKey(contestF)
		}, contestF, "injected read failure", true},
		{"target check incomplete", func(f *contestPredicateFixture) {
			putRaw(f, types.FactRelationReverseKey(contestF, "garbled"), []byte{0xff})
		}, contestF, "incoming relation", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newContestPredicateFixture(t)
			f.disprove("counter-k", 70)
			f.putRelation("counter-k", contestF, types.RelationType_RELATION_TYPE_CONTRADICTS)
			f.putFact("contest-g", types.FactStatus_FACT_STATUS_CONTESTED)
			f.putRelation("counter-k", "contest-g", types.RelationType_RELATION_TYPE_CONTRADICTS)
			tc.inject(f)
			fBefore := f.ctx.KVStore(f.keys[0]).Get(types.FactKey(contestF))
			ctx := f.ctx.WithEventManager(sdk.NewEventManager())
			require.NoError(t, f.k.settleCounterFactTargets(ctx, "counter-k"))
			*f.kFault = handoffFault{}
			require.Equal(t, fBefore, f.ctx.KVStore(f.keys[0]).Get(types.FactKey(contestF)), "F is left as it is")
			incomplete := contestEventsOf(ctx, "zerone.knowledge.contest_check_incomplete")
			require.Len(t, incomplete, 1)
			require.Equal(t, tc.named, incomplete[0]["fact_id"])
			require.Contains(t, incomplete[0]["reason"], tc.reason)
			g, found, err := f.k.getFactChecked(f.ctx, "contest-g")
			require.NoError(t, err)
			require.True(t, found)
			if tc.released {
				require.Equal(t, types.FactStatus_FACT_STATUS_VERIFIED, g.Status, "the other target still settles")
				require.Equal(t, []map[string]string{{"fact_id": "contest-g", "reverted_by_fact": "counter-k"}},
					contestEventsOf(ctx, "zerone.knowledge.contradiction_reversed"))
			} else {
				require.Equal(t, types.FactStatus_FACT_STATUS_CONTESTED, g.Status, "no target is released")
				require.Empty(t, contestEventsOf(ctx, "zerone.knowledge.contradiction_reversed"))
			}
		})
	}
	t.Run("write error", func(t *testing.T) {
		f := newContestPredicateFixture(t)
		f.disprove("counter-k", 70)
		f.putRelation("counter-k", contestF, types.RelationType_RELATION_TYPE_CONTRADICTS)
		*f.kFault = handoffFault{op: "set", prefix: types.FactKey(contestF)}
		err := f.k.settleCounterFactTargets(f.ctx, "counter-k")
		*f.kFault = handoffFault{}
		require.ErrorContains(t, err, "injected set failure")
	})
}

// challengeEndStatus (§5.2 rows 7 and 8; review fixes SEC-1, F1, SEC-6) on
// hand-built state. F is CHALLENGED by challenge claim "challenge-c", opened
// at 50 from priorStatus. Dependency D fell at 50, except in the incomplete
// cases, where F depends on nothing.
func TestChallengeEndStatus(t *testing.T) {
	const challengeID = "challenge-c"
	challenge := &types.Claim{Id: challengeID, Submitter: "challenger", ProvisionalFactId: contestF, SubmittedAtBlock: 50}
	opened := func(t *testing.T, prior types.FactStatus) *contestPredicateFixture {
		f := newContestPredicateFixture(t)
		f.putFact(contestF, types.FactStatus_FACT_STATUS_CHALLENGED)
		require.NoError(t, f.k.SetClaim(f.ctx, challenge))
		f.putCausedTransition(prior, types.FactStatus_FACT_STATUS_CHALLENGED, causeChallengeOpened, challengeID, 50)
		return f
	}
	base := func(t *testing.T, prior types.FactStatus) *contestPredicateFixture {
		f := opened(t, prior)
		f.disprove("dependency-d", 50)
		f.putRelation(contestF, "dependency-d", types.RelationType_RELATION_TYPE_REQUIRES)
		return f
	}
	overBound := func(f *contestPredicateFixture) {
		for i := 0; i < 4; i++ {
			f.putRelation(fmt.Sprintf("citing-%d", i), contestF, types.RelationType_RELATION_TYPE_CITES)
		}
		SetContestScanMaxRecordsForTest(f.t, 3)
	}
	end := func(f *contestPredicateFixture, c *types.Claim, integrity, survived bool) (types.FactStatus, []map[string]string) {
		f.t.Helper()
		var status types.FactStatus
		_, incomplete := f.check(func(ctx context.Context, fact *types.Fact) bool {
			status = f.k.challengeEndStatus(ctx, fact, c, integrity, survived)
			return false
		})
		return status, incomplete
	}
	active, contested := types.FactStatus_FACT_STATUS_ACTIVE, types.FactStatus_FACT_STATUS_CONTESTED

	t.Run("a survival is judged from the height its challenge opened", func(t *testing.T) {
		for opened, want := range map[uint64]types.FactStatus{0: contested, 49: contested, 50: contested, 51: active} {
			f := base(t, types.FactStatus_FACT_STATUS_ACTIVE)
			c := proto.Clone(challenge).(*types.Claim)
			c.SubmittedAtBlock = opened
			status, incomplete := end(f, c, true, true)
			require.Equal(t, want, status, "opened at %d", opened)
			require.Empty(t, incomplete)
		}
	})
	t.Run("any other ending is judged since the last survival", func(t *testing.T) {
		f := base(t, types.FactStatus_FACT_STATUS_CONTESTED)
		status, _ := end(f, challenge, true, false)
		require.Equal(t, contested, status, "never survived")
		f.survive(51, 60)
		status, incomplete := end(f, challenge, true, false)
		require.Equal(t, active, status, "D fell before the survived challenge opened")
		require.Empty(t, incomplete)
	})
	t.Run("marker off and conjectures restore", func(t *testing.T) {
		f := base(t, types.FactStatus_FACT_STATUS_CONTESTED)
		for _, survived := range []bool{true, false} {
			status, incomplete := end(f, challenge, false, survived)
			require.Equal(t, active, status)
			require.Empty(t, incomplete)
		}
		require.NoError(t, f.k.SetFactSkipTransition(f.ctx, &types.Fact{Id: contestF, ClaimId: contestF + "-claim", ClaimType: types.ClaimType_CLAIM_TYPE_CONJECTURE, Status: types.FactStatus_FACT_STATUS_CHALLENGED}))
		status, incomplete := end(f, challenge, true, false)
		require.Equal(t, types.FactStatus_FACT_STATUS_PROVISIONAL, status)
		require.Empty(t, incomplete)
	})

	// R2 is read before R1, so a dependency refuted while the challenge was
	// open is found however many facts cite F: the check finishes and F is
	// CONTESTED, whatever its standing before (review G1, second pass).
	for _, prior := range []types.FactStatus{types.FactStatus_FACT_STATUS_VERIFIED, types.FactStatus_FACT_STATUS_ACTIVE, types.FactStatus_FACT_STATUS_CONTESTED} {
		for _, survived := range []bool{true, false} {
			t.Run(fmt.Sprintf("refuted dependency past the bound, %s before, survived %t", prior, survived), func(t *testing.T) {
				f := base(t, prior)
				overBound(f)
				status, incomplete := end(f, challenge, true, survived)
				require.Equal(t, contested, status)
				require.Empty(t, incomplete)
			})
		}
	}

	// An incomplete check changes nothing: the standing before the challenge
	// decides, and one event names F.
	for _, tc := range []struct {
		name   string
		prior  types.FactStatus
		setup  func(*contestPredicateFixture)
		want   types.FactStatus
		reason string
	}{
		{"VERIFIED before", types.FactStatus_FACT_STATUS_VERIFIED, nil, active, "contest scan exceeds record bound"},
		{"ACTIVE before", types.FactStatus_FACT_STATUS_ACTIVE, nil, active, "contest scan exceeds record bound"},
		{"CONTESTED before", types.FactStatus_FACT_STATUS_CONTESTED, nil, contested, "contest scan exceeds record bound"},
		{"CONTESTED at an earlier challenge only", types.FactStatus_FACT_STATUS_ACTIVE, func(f *contestPredicateFixture) {
			f.putCausedTransition(types.FactStatus_FACT_STATUS_CONTESTED, types.FactStatus_FACT_STATUS_CHALLENGED, causeChallengeOpened, "challenge-earlier", 20)
		}, active, "contest scan exceeds record bound"},
		{"no challenge_opened transition names the challenge", types.FactStatus_FACT_STATUS_ACTIVE, func(f *contestPredicateFixture) {
			f.ctx.KVStore(f.keys[0]).Delete(types.StatusTransitionKey(contestF, 1))
		}, contested, "status before the challenge: no challenge_opened transition names challenge-c"},
		{"unreadable history", types.FactStatus_FACT_STATUS_ACTIVE, func(f *contestPredicateFixture) {
			f.ctx.KVStore(f.keys[0]).Set(types.StatusTransitionKey(contestF, 1), []byte{0xff})
		}, contested, "status before the challenge"},
	} {
		for _, survived := range []bool{true, false} {
			t.Run(fmt.Sprintf("incomplete check, %s, survived %t", tc.name, survived), func(t *testing.T) {
				f := opened(t, tc.prior)
				if tc.setup != nil {
					tc.setup(f)
				}
				overBound(f)
				status, incomplete := end(f, challenge, true, survived)
				require.Equal(t, tc.want, status)
				require.Len(t, incomplete, 1)
				require.Equal(t, contestF, incomplete[0]["fact_id"])
				require.Contains(t, incomplete[0]["reason"], "contest scan exceeds record bound")
				require.Contains(t, incomplete[0]["reason"], tc.reason)
			})
		}
	}
}
