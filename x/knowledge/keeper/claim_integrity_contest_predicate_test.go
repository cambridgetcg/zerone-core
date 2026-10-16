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

func (f *contestPredicateFixture) survive(height uint64) {
	f.t.Helper()
	f.putTransition(contestF, types.FactStatus_FACT_STATUS_CHALLENGED, types.FactStatus_FACT_STATUS_ACTIVE, causeChallengeSurvived, height)
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

// R1: an incoming CONTRADICTS edge whose source stands (VERIFIED, ACTIVE,
// CONTESTED, CHALLENGED or AT_RISK). Other incoming types, and F's own
// outgoing CONTRADICTS edges, are not reasons.
func TestContestReasonStandingCounterFact(t *testing.T) {
	standing := map[types.FactStatus]bool{
		types.FactStatus_FACT_STATUS_VERIFIED:   true,
		types.FactStatus_FACT_STATUS_ACTIVE:     true,
		types.FactStatus_FACT_STATUS_CONTESTED:  true,
		types.FactStatus_FACT_STATUS_CHALLENGED: true,
		types.FactStatus_FACT_STATUS_AT_RISK:    true,
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
// unless F survived a direct challenge at or after D's latest transition to
// DISPROVEN. A D with no recorded fall is cleared by any survival.
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
			for survivedAt, remains := range map[uint64]bool{0: true, 49: true, 50: false, 51: false} {
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
		require.False(t, f.remainsAt(70))
	})
	t.Run("one uncleared dependency is enough", func(t *testing.T) {
		f := newContestPredicateFixture(t)
		f.disprove("dependency-a", 20)
		f.disprove("dependency-b", 80)
		f.putRelation(contestF, "dependency-a", types.RelationType_RELATION_TYPE_REQUIRES)
		f.putRelation(contestF, "dependency-b", types.RelationType_RELATION_TYPE_SUPPORTS)
		require.True(t, f.remainsAt(50))
		require.False(t, f.remainsAt(80))
	})
}

// Without a supplied height, survivedAt is the height of F's latest
// challenge_survived transition. An inconclusive challenge or a knowledge-11
// survival (recorded as "unknown") clears nothing.
func TestContestReasonSinceLastSurvival(t *testing.T) {
	for _, tc := range []struct {
		fall    uint64
		remains bool
	}{{55, false}, {60, false}, {65, true}, {85, true}} {
		t.Run(fmt.Sprintf("fall at %d", tc.fall), func(t *testing.T) {
			f := newContestPredicateFixture(t)
			f.survive(40)
			f.survive(60)
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
			f.survive(60)
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
