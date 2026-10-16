package keeper_test

import (
	"context"
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Claim integrity, Task 12 (P3b): a contradiction contests its target only
// once a panel accepts it (plan §5.2 rows 2 and 4, §5.3). With the marker on:
//   - admitting a contradiction, through MsgSubmitContradiction or a
//     MsgSubmitClaim carrying CONTRADICTS, changes no fact;
//   - accepting one moves each VERIFIED or ACTIVE target to CONTESTED, with
//     cause contradiction_accepted/<counter-claim ID>;
//   - when one ends without acceptance (REJECT, MALFORMED, INCONCLUSIVE or a
//     starved panel), a CONTESTED target returns to VERIFIED (PROVISIONAL for
//     a conjecture), with cause contradiction_reversed/<claim ID>, only when
//     no contest reason remains: no standing accepted counter-fact (R1) and
//     no refuted dependency since its last survived challenge (R2). A check
//     that cannot finish keeps the target CONTESTED and says so in an event.
// With the marker off, knowledge 11 flips at admission and restores whenever
// no other contradiction is live; the marker-off twins pin that.
//
// Reviewers and challengers are never parties to the rounds they decide, so
// Task 14 does not change these tests. Task 13 adds paths that settle a
// CONTESTED target (a challenge on it, a refuted counter-fact); no test here
// challenges a CONTESTED fact or a counter-fact.

const (
	csUnspecified = types.FactStatus_FACT_STATUS_UNSPECIFIED
	csProvisional = types.FactStatus_FACT_STATUS_PROVISIONAL
	csVerified    = types.FactStatus_FACT_STATUS_VERIFIED
	csContested   = types.FactStatus_FACT_STATUS_CONTESTED
	csDisproven   = types.FactStatus_FACT_STATUS_DISPROVEN

	csAcceptedEvent   = "zerone.knowledge.contradiction_accepted"
	csReversedEvent   = "zerone.knowledge.contradiction_reversed"
	csIncompleteEvent = "zerone.knowledge.contest_check_incomplete"
)

func (f *v11Fixture) freshEvents() {
	f.ctx = f.ctx.WithEventManager(sdk.NewEventManager())
}

func (f *v11Fixture) nextBlock(t *testing.T) {
	t.Helper()
	f.advanceTo(t, uint64(f.ctx.BlockHeight())+1)
}

func csContradicts(factID string) func(*types.MsgSubmitClaim) {
	return func(m *types.MsgSubmitClaim) {
		m.Relations = []*types.ClaimRelation{{Relation: types.RelationType_RELATION_TYPE_CONTRADICTS, TargetFactId: factID}}
	}
}

func csReversed(factID, claimID string) []map[string]string {
	return []map[string]string{{"fact_id": factID, "reverted_by_claim": claimID}}
}

// contestEndings are the three ways a claim ends without acceptance.
var contestEndings = []struct {
	name string
	end  func(*testing.T, *v11Fixture, string)
}{
	{"rejected", func(t *testing.T, f *v11Fixture, id string) {
		require.Equal(t, types.Verdict_VERDICT_REJECT, f.decide(t, id, "reject", v11Reviewers()...).Verdict)
	}},
	{"malformed", func(t *testing.T, f *v11Fixture, id string) {
		require.Equal(t, types.Verdict_VERDICT_MALFORMED, f.decide(t, id, "malformed", v11Reviewers()...).Verdict)
	}},
	{"starved", func(t *testing.T, f *v11Fixture, id string) { f.starve(t, id) }},
}

// contestActivations select claim integrity on a knowledge-11 store: in place,
// as Migrate11to12 does at the upgrade height, or by exporting the store and
// importing the file with the flag set, at the same chain ID and height.
var contestActivations = []struct {
	name     string
	activate func(*testing.T, *v11Fixture) *v11Fixture
}{
	{"upgraded in place", func(t *testing.T, f *v11Fixture) *v11Fixture {
		require.NoError(t, f.k.ValidateClaimIntegrityActivation(f.ctx))
		require.NoError(t, f.k.EnableClaimIntegrity(f.ctx))
		require.True(t, f.claimIntegrityEnabled(t))
		return f
	}},
	{"imported", func(t *testing.T, f *v11Fixture) *v11Fixture {
		gs := f.k.ExportGenesis(f.ctx)
		require.False(t, gs.ClaimIntegrityEnabled)
		gs.ClaimIntegrityEnabled = true
		g := newClaimIntegrityGenesisFixture(t, f.ctx.ChainID(), gs)
		g.advanceTo(t, uint64(f.ctx.BlockHeight()))
		require.True(t, g.claimIntegrityEnabled(t))
		return g
	}},
}

// P3 replica (plan §7 #6, #7, #9). Marker on: A's admission leaves F
// VERIFIED; A's acceptance contests F and names A; B, at the minimum stake,
// starves and F stays CONTESTED with no contradiction_reversed event, because
// the counter-fact K still stands. Marker off: knowledge 11 contests F at
// admission and B's starvation launders F back to VERIFIED while K stands.
func TestClaimIntegrity_ContradictionContestsOnlyWhenAccepted(t *testing.T) {
	for _, sel := range dedupSelections {
		t.Run(sel.name, func(t *testing.T) {
			f := sel.setup(t)
			require.Equal(t, sel.integrity, f.claimIntegrityEnabled(t))
			claimF, factF, minimum := f.contraTarget(t, "v12-cs-author", "Synthetic v12 probe P3: compound Q reduced tumour volume in mice by 30 percent.")
			created := v11Transition{csUnspecified, csVerified, "verification", claimF}
			admissionFlip := v11Transition{csVerified, csContested, "cascade", ""}

			a := f.mustContradict(t, v11Addr("v12-cs-jones"), factF, "Synthetic v12 probe P3: a replication found compound Q did not reduce tumour volume.", minimum)
			if sel.integrity {
				require.Equal(t, csVerified, f.fact(t, factF).Status, "admitting a contradiction changes no fact")
				require.Equal(t, []v11Transition{created}, f.transitions(factF))
			} else {
				require.Equal(t, csContested, f.fact(t, factF).Status, "knowledge 11 contests F at admission")
				require.Equal(t, []v11Transition{created, admissionFlip}, f.transitions(factF))
			}

			f.freshEvents()
			factK := f.accept(t, a, v11Reviewers()...)
			require.Equal(t, csVerified, f.fact(t, factK).Status, "the counter-fact stands (asymmetric, plan §9 D2)")
			require.Equal(t, csContested, f.fact(t, factF).Status)
			if sel.integrity {
				require.Equal(t, []v11Transition{created, {csVerified, csContested, "contradiction_accepted", a}}, f.transitions(factF))
				require.Equal(t, []map[string]string{{"fact_id": factF, "counter_claim_id": a, "counter_fact_id": factK}},
					v11Event(f.ctx, csAcceptedEvent))
			} else {
				require.Equal(t, []v11Transition{created, admissionFlip}, f.transitions(factF))
				require.Empty(t, v11Event(f.ctx, csAcceptedEvent))
			}
			history := f.transitions(factF)

			f.nextBlock(t)
			b := f.mustContradict(t, v11Addr("v12-cs-griefer"), factF, "Synthetic v12 probe P3: a contradiction at the minimum stake, left without reviewers.", minimum)
			require.Equal(t, csContested, f.fact(t, factF).Status)
			f.freshEvents()
			f.starve(t, b)
			if sel.integrity {
				require.Equal(t, csContested, f.fact(t, factF).Status, "a starved contradiction cannot release F while K stands")
				require.Empty(t, v11Event(f.ctx, csReversedEvent))
				require.Equal(t, history, f.transitions(factF))
			} else {
				require.Equal(t, csVerified, f.fact(t, factF).Status, "knowledge 11 launders F while K stands")
				require.Equal(t, csReversed(factF, b), v11Event(f.ctx, csReversedEvent))
				require.Equal(t, append(history, v11Transition{csContested, csVerified, "verification", claimF}), f.transitions(factF))
			}
			require.Equal(t, csVerified, f.fact(t, factK).Status)
			f.requireNoLiveRounds(t)
		})
	}
}

// A MsgSubmitClaim carrying CONTRADICTS does not contest its target at
// admission; its acceptance does, and names it. The accepted claim's fact K
// carries the CONTRADICTS edge to F. Marker off:
// TestSubmitClaim_ContradictionAutoContests (knowledge 11) still holds.
func TestClaimIntegrity_RelationContradictionContestsOnlyWhenAccepted(t *testing.T) {
	for _, sel := range dedupSelections {
		t.Run(sel.name, func(t *testing.T) {
			f := sel.setup(t)
			claimF, factF, _ := f.contraTarget(t, "v12-cs-rel-author", "Synthetic v12 probe P3: fact F, which a claim relation contradicts.")
			created := v11Transition{csUnspecified, csVerified, "verification", claimF}
			admissionFlip := v11Transition{csVerified, csContested, "cascade", ""}

			c := f.mustSubmitClaim(t, v11Addr("v12-cs-rel-counter"), "Synthetic v12 probe P3: a finding that contradicts fact F through a claim relation.", dedupDomain, csContradicts(factF))
			if sel.integrity {
				require.Equal(t, csVerified, f.fact(t, factF).Status, "admitting a CONTRADICTS relation changes no fact")
				require.Equal(t, []v11Transition{created}, f.transitions(factF))
			} else {
				require.Equal(t, csContested, f.fact(t, factF).Status)
				require.Equal(t, []v11Transition{created, admissionFlip}, f.transitions(factF))
			}

			f.freshEvents()
			factK := f.accept(t, c, v11Reviewers()...)
			require.Equal(t, csContested, f.fact(t, factF).Status)
			require.Equal(t, csVerified, f.fact(t, factK).Status)
			incoming, err := f.k.GetIncomingRelations(f.ctx, factF)
			require.NoError(t, err)
			require.Len(t, incoming, 1)
			require.Equal(t, factK, incoming[0].SourceFactId)
			require.Equal(t, types.RelationType_RELATION_TYPE_CONTRADICTS, incoming[0].Relation)
			if sel.integrity {
				require.Equal(t, []v11Transition{created, {csVerified, csContested, "contradiction_accepted", c}}, f.transitions(factF))
				require.Equal(t, []map[string]string{{"fact_id": factF, "counter_claim_id": c, "counter_fact_id": factK}},
					v11Event(f.ctx, csAcceptedEvent))
			} else {
				require.Equal(t, []v11Transition{created, admissionFlip}, f.transitions(factF))
				require.Empty(t, v11Event(f.ctx, csAcceptedEvent))
			}
			f.requireNoLiveRounds(t)
		})
	}
}

// A claim can name the same target twice. The relation store keeps one edge
// per (source, target) pair and the last relation written wins, so a claim
// with [CONTRADICTS F, CITES F] stores K→F as CITES. With the marker on,
// acceptance contests F only when the stored edge is CONTRADICTS, the edge R1
// reads. Otherwise F would be CONTESTED with no edge to hold it, and a later
// starved contradiction would release it while K stands. Marker off:
// knowledge 11 flips F at admission in either order, and a starved
// contradiction launders it.
func TestClaimIntegrity_AcceptedContestFollowsStoredEdge(t *testing.T) {
	for _, order := range []struct {
		name   string
		first  types.RelationType
		second types.RelationType
	}{
		{"CONTRADICTS then CITES", types.RelationType_RELATION_TYPE_CONTRADICTS, types.RelationType_RELATION_TYPE_CITES},
		{"CITES then CONTRADICTS", types.RelationType_RELATION_TYPE_CITES, types.RelationType_RELATION_TYPE_CONTRADICTS},
	} {
		for _, sel := range dedupSelections {
			t.Run(order.name+"/"+sel.name, func(t *testing.T) {
				f := sel.setup(t)
				claimF, factF, minimum := f.contraTarget(t, "v12-cs-dup-author", "Synthetic v12 probe P3: fact F, which one claim names twice.")
				created := v11Transition{csUnspecified, csVerified, "verification", claimF}
				admissionFlip := v11Transition{csVerified, csContested, "cascade", ""}

				c := f.mustSubmitClaim(t, v11Addr("v12-cs-dup-counter"), "Synthetic v12 probe P3: a finding that both contradicts and cites fact F.", dedupDomain,
					func(m *types.MsgSubmitClaim) {
						m.Relations = []*types.ClaimRelation{{Relation: order.first, TargetFactId: factF}, {Relation: order.second, TargetFactId: factF}}
					})
				if sel.integrity {
					require.Equal(t, csVerified, f.fact(t, factF).Status, "admission changes no fact")
					require.Equal(t, []v11Transition{created}, f.transitions(factF))
				} else {
					require.Equal(t, csContested, f.fact(t, factF).Status, "knowledge 11 contests F at admission")
					require.Equal(t, []v11Transition{created, admissionFlip}, f.transitions(factF))
				}

				f.freshEvents()
				factK := f.accept(t, c, v11Reviewers()...)
				incoming, err := f.k.GetIncomingRelations(f.ctx, factF)
				require.NoError(t, err)
				require.Len(t, incoming, 1, "one edge per (source, target) pair")
				require.Equal(t, factK, incoming[0].SourceFactId)
				require.Equal(t, order.second, incoming[0].Relation, "the last relation written wins")
				contested := sel.integrity && order.second == types.RelationType_RELATION_TYPE_CONTRADICTS
				switch {
				case !sel.integrity:
					require.Equal(t, csContested, f.fact(t, factF).Status)
					require.Equal(t, []v11Transition{created, admissionFlip}, f.transitions(factF))
					require.Empty(t, v11Event(f.ctx, csAcceptedEvent))
				case contested:
					require.Equal(t, csContested, f.fact(t, factF).Status, "the stored CONTRADICTS edge contests F")
					require.Equal(t, []v11Transition{created, {csVerified, csContested, "contradiction_accepted", c}}, f.transitions(factF))
					require.Equal(t, []map[string]string{{"fact_id": factF, "counter_claim_id": c, "counter_fact_id": factK}},
						v11Event(f.ctx, csAcceptedEvent))
				default:
					require.Equal(t, csVerified, f.fact(t, factF).Status, "no stored CONTRADICTS edge, so acceptance contests nothing")
					require.Equal(t, []v11Transition{created}, f.transitions(factF))
					require.Empty(t, v11Event(f.ctx, csAcceptedEvent))
				}
				history := f.transitions(factF)

				f.nextBlock(t)
				b := f.mustContradict(t, v11Addr("v12-cs-dup-griefer"), factF, "Synthetic v12 probe P3: a contradiction of the twice-named fact F, left without reviewers.", minimum)
				f.freshEvents()
				f.starve(t, b)
				switch {
				case !sel.integrity:
					require.Equal(t, csVerified, f.fact(t, factF).Status, "knowledge 11 launders F")
					require.Equal(t, csReversed(factF, b), v11Event(f.ctx, csReversedEvent))
					require.Equal(t, append(history, v11Transition{csContested, csVerified, "verification", claimF}), f.transitions(factF))
				case contested:
					require.Equal(t, csContested, f.fact(t, factF).Status, "R1: K's stored CONTRADICTS edge still holds F")
					require.Empty(t, v11Event(f.ctx, csReversedEvent))
					require.Equal(t, history, f.transitions(factF))
				default:
					require.Equal(t, csVerified, f.fact(t, factF).Status)
					require.Empty(t, v11Event(f.ctx, csReversedEvent))
					require.Equal(t, history, f.transitions(factF))
				}
				require.Equal(t, csVerified, f.fact(t, factK).Status)
				f.requireNoLiveRounds(t)
			})
		}
	}
}

// With the marker on, a contradiction that ends without acceptance, by either
// admission path and in each of the three non-accepted endings, leaves its
// VERIFIED target exactly as it was: no transition and no event.
func TestClaimIntegrity_UnacceptedContradictionChangesNothing(t *testing.T) {
	for _, ending := range contestEndings {
		for _, via := range []string{"MsgSubmitContradiction", "MsgSubmitClaim"} {
			t.Run(ending.name+"/"+via, func(t *testing.T) {
				f := setupClaimIntegrityTest(t)
				claimF, factF, minimum := f.contraTarget(t, "v12-cs-un-author", "Synthetic v12 probe P3: fact F, disputed by a claim that is never accepted.")
				const counter = "Synthetic v12 probe P3: a dispute of fact F that no panel accepts."
				var c string
				if via == "MsgSubmitContradiction" {
					c = f.mustContradict(t, v11Addr("v12-cs-un-counter"), factF, counter, minimum)
				} else {
					c = f.mustSubmitClaim(t, v11Addr("v12-cs-un-counter"), counter, dedupDomain, csContradicts(factF))
				}
				f.freshEvents()
				ending.end(t, f, c)
				require.Equal(t, csVerified, f.fact(t, factF).Status)
				require.Equal(t, []v11Transition{{csUnspecified, csVerified, "verification", claimF}}, f.transitions(factF))
				require.Empty(t, v11Event(f.ctx, csReversedEvent))
				require.Empty(t, v11Event(f.ctx, csAcceptedEvent))
				f.requireNoLiveRounds(t)
			})
		}
	}
}

// Plan §3.6: a fact CONTESTED by a knowledge-11 admission flip is released by
// the same predicate when that contradiction ends, after an in-place upgrade
// or an import. If an accepted counter-fact K stands (the P3 state before
// activation), R1 keeps the fact CONTESTED instead, in every ending.
func TestClaimIntegrity_PreActivationContestSettlesThroughPredicate(t *testing.T) {
	for _, standing := range []bool{false, true} {
		for _, activation := range contestActivations {
			for _, ending := range contestEndings {
				t.Run(fmt.Sprintf("counter-fact standing=%t/%s/%s", standing, activation.name, ending.name), func(t *testing.T) {
					f := setupV11SemanticsTest(t)
					_, factF, _ := f.contraTarget(t, "v12-pa-author", "Synthetic v12 probe P3: fact F, contested before claim integrity existed.")
					if standing {
						a, err := f.contradict(v11Addr("v12-pa-jones"), factF, "Synthetic v12 probe P3: an accepted finding that contradicts fact F.", dedupDomain, v11ReviewFee)
						require.NoError(t, err)
						f.accept(t, a, v11Reviewers()...)
						f.nextBlock(t)
					}
					c1, err := f.contradict(v11Addr("v12-pa-pending"), factF, "Synthetic v12 probe P3: a contradiction of fact F still pending at activation.", dedupDomain, v11ReviewFee)
					require.NoError(t, err)
					require.Equal(t, csContested, f.fact(t, factF).Status)
					history := f.transitions(factF)
					require.Equal(t, v11Transition{csVerified, csContested, "cascade", ""}, history[len(history)-1], "the knowledge-11 admission flip")

					g := activation.activate(t, f)
					require.Equal(t, history, g.transitions(factF))
					g.freshEvents()
					ending.end(t, g, c1)
					if standing {
						require.Equal(t, csContested, g.fact(t, factF).Status, "R1: the accepted counter-fact still stands")
						require.Empty(t, v11Event(g.ctx, csReversedEvent))
						require.Equal(t, history, g.transitions(factF))
					} else {
						require.Equal(t, csVerified, g.fact(t, factF).Status, "no contest reason remains")
						require.Equal(t, csReversed(factF, c1), v11Event(g.ctx, csReversedEvent))
						require.Equal(t, append(history, v11Transition{csContested, csVerified, "contradiction_reversed", c1}), g.transitions(factF))
					}
					require.Empty(t, v11Event(g.ctx, csIncompleteEvent))
					g.requireNoLiveRounds(t)
				})
			}
		}
	}
}

// A fact contested by the cascade (a dependency was refuted) stays CONTESTED
// when a contradiction of it starves: R2 holds, since it has survived no
// challenge since. This settles an inferred defect (plan header): with the
// marker off, knowledge 11 launders the cascade-contested fact to VERIFIED.
func TestClaimIntegrity_CascadeContestedTargetStaysContested(t *testing.T) {
	for _, sel := range dedupSelections {
		t.Run(sel.name, func(t *testing.T) {
			f := sel.setup(t)
			_, factD, _ := f.contraTarget(t, "v12-cc-dependency", "Synthetic v12 probe P3: fact D, on which fact F depends.")
			claimF := f.mustSubmitClaim(t, v11Addr("v12-cc-author"), "Synthetic v12 probe P3: fact F, which requires fact D to hold.", dedupDomain,
				func(m *types.MsgSubmitClaim) {
					m.Relations = []*types.ClaimRelation{{Relation: types.RelationType_RELATION_TYPE_REQUIRES, TargetFactId: factD}}
				})
			factF := f.accept(t, claimF, v11Reviewers()...)
			f.accept(t, f.challenge(t, v11Addr("v12-cc-challenger"), factD), v11Reviewers()...)
			require.Equal(t, csDisproven, f.fact(t, factD).Status)
			cascaded := []v11Transition{
				{csUnspecified, csVerified, "verification", claimF},
				{csVerified, csContested, "cascade", factD},
			}
			require.Equal(t, cascaded, f.transitions(factF))

			params, err := f.k.GetParams(f.ctx)
			require.NoError(t, err)
			minimum := keeper.EffectiveMinChallengeStake(params, f.fact(t, factF).Confidence).String()
			b := f.mustContradict(t, v11Addr("v12-cc-griefer"), factF, "Synthetic v12 probe P3: a contradiction of the cascade-contested fact F that starves.", minimum)
			f.freshEvents()
			f.starve(t, b)
			if sel.integrity {
				require.Equal(t, csContested, f.fact(t, factF).Status, "R2: the refuted dependency still stands against F")
				require.Empty(t, v11Event(f.ctx, csReversedEvent))
				require.Equal(t, cascaded, f.transitions(factF))
			} else {
				require.Equal(t, csVerified, f.fact(t, factF).Status, "knowledge 11 launders a cascade-contested fact")
				require.Equal(t, csReversed(factF, b), v11Event(f.ctx, csReversedEvent))
				require.Equal(t, append(cascaded, v11Transition{csContested, csVerified, "verification", claimF}), f.transitions(factF))
			}
			f.requireNoLiveRounds(t)
		})
	}
}

// A conjecture cannot be contradicted, but a record that predates that gate
// can leave one CONTESTED with a pending contradiction naming it. When that
// contradiction ends, the conjecture returns to PROVISIONAL, never VERIFIED.
// The fact the contradiction first named is untouched.
func TestClaimIntegrity_ConjectureShapedTargetReturnsToProvisional(t *testing.T) {
	f := setupClaimIntegrityTest(t)
	conjecture, err := f.server.PostConjecture(f.ctx, &types.MsgPostConjecture{
		Proposer: v11Addr("v12-cj-proposer"), Domain: dedupDomain, Category: "empirical", Stake: v11ReviewFee,
		Statement:              "Synthetic v12 probe P3: conjecture Q, open until a counterexample is shown.",
		FalsificationPredicate: "Exhibit one synthetic counterexample to conjecture Q in this fixture.",
	})
	require.NoError(t, err)
	conjFact := f.accept(t, conjecture.ClaimId, v11Reviewers()...)
	require.Equal(t, csProvisional, f.fact(t, conjFact).Status)
	claimX, factX, minimum := f.contraTarget(t, "v12-cj-author", "Synthetic v12 probe P3: fact X, which the contradiction first names.")
	c := f.mustContradict(t, v11Addr("v12-cj-legacy"), factX, "Synthetic v12 probe P3: a contradiction later pointed at conjecture Q.", minimum)

	// Model the legacy record: the pending contradiction names the conjecture,
	// and the conjecture is CONTESTED.
	claim := f.claim(t, c)
	claim.Relations[0].TargetFactId = conjFact
	require.NoError(t, f.k.SetClaim(f.ctx, claim))
	fact := f.fact(t, conjFact)
	fact.Status = csContested
	require.NoError(t, f.k.SetFact(f.ctx, fact))
	history := f.transitions(conjFact)

	f.freshEvents()
	f.starve(t, c)
	require.Equal(t, csProvisional, f.fact(t, conjFact).Status, "a conjecture is restored to PROVISIONAL, never VERIFIED")
	require.Equal(t, append(history, v11Transition{csContested, csProvisional, "contradiction_reversed", c}), f.transitions(conjFact))
	require.Equal(t, csReversed(conjFact, c), v11Event(f.ctx, csReversedEvent))
	require.Equal(t, csVerified, f.fact(t, factX).Status)
	require.Equal(t, []v11Transition{{csUnspecified, csVerified, "verification", claimX}}, f.transitions(factX))
	f.requireNoLiveRounds(t)
}

// Plan §5.3 failure handling: a relation record that cannot be read, or a
// relation scan past the 100k-record bound, means the contest reason
// remains. F, contested by a knowledge-11 flip with no real reason left,
// stays CONTESTED and contest_check_incomplete names it and the cause.
// Without the fault the same state releases F
// (TestClaimIntegrity_PreActivationContestSettlesThroughPredicate).
func TestClaimIntegrity_IncompleteContestCheckKeepsTargetContested(t *testing.T) {
	for _, fault := range []struct {
		name   string
		inject func(*testing.T, *v11Fixture, string)
		reason string
	}{
		{"unreadable incoming relation", func(t *testing.T, f *v11Fixture, factF string) {
			f.ctx.KVStore(f.key).Set(types.FactRelationReverseKey(factF, "unreadable-source"), []byte{0xff})
		}, "incoming relation"},
		{"relation scan bound", func(t *testing.T, f *v11Fixture, factF string) {
			kv := f.ctx.KVStore(f.key)
			for i := 0; i <= 100_000; i++ {
				source := fmt.Sprintf("bound-source-%06d", i)
				rel := &types.FactRelation{SourceFactId: source, TargetFactId: factF, Relation: types.RelationType_RELATION_TYPE_SUPPORTS}
				bz, err := proto.Marshal(rel)
				require.NoError(t, err)
				kv.Set(types.FactRelationReverseKey(factF, source), bz)
			}
		}, "contest scan exceeds record bound"},
	} {
		t.Run(fault.name, func(t *testing.T) {
			f := setupV11SemanticsTest(t)
			_, factF, _ := f.contraTarget(t, "v12-ic-author", "Synthetic v12 probe P3: fact F, whose contest check cannot finish.")
			c1, err := f.contradict(v11Addr("v12-ic-pending"), factF, "Synthetic v12 probe P3: a contradiction of fact F pending at activation.", dedupDomain, v11ReviewFee)
			require.NoError(t, err)
			require.Equal(t, csContested, f.fact(t, factF).Status)
			contestActivations[0].activate(t, f)
			fault.inject(t, f, factF)
			history := f.transitions(factF)

			f.freshEvents()
			f.starve(t, c1)
			require.Equal(t, csContested, f.fact(t, factF).Status, "an incomplete check never restores")
			require.Equal(t, history, f.transitions(factF))
			require.Empty(t, v11Event(f.ctx, csReversedEvent))
			incomplete := v11Event(f.ctx, csIncompleteEvent)
			require.Len(t, incomplete, 1)
			require.Equal(t, factF, incomplete[0]["fact_id"])
			require.Contains(t, incomplete[0]["reason"], fault.reason)
			f.requireNoLiveRounds(t)
		})
	}
}

// A round admitted under the legacy review policy can still be live once
// claim integrity is on. When it reaches its aggregation deadline with too
// few reveals (the phases.go expiry branch; the round stays EXPIRED), its
// contradiction targets settle through the same predicate: released with no
// reason, held by a standing counter-fact. A malformed marker is never read
// as knowledge 11: the targets are left as they are and BeginBlock does not
// halt. As in TestSubmitContradiction_StarvedDoesNotLockFact, the advance
// skips the reveal deadline, so no aggregation runs first.
func TestClaimIntegrity_LegacyPolicyStarvedContradictionSettles(t *testing.T) {
	for _, tc := range []struct {
		name            string
		standing        bool
		malformedMarker bool
	}{
		{"counter-fact standing=false", false, false},
		{"counter-fact standing=true", true, false},
		{"malformed marker", false, true},
	} {
		standing := tc.standing
		t.Run(tc.name, func(t *testing.T) {
			gs := types.DefaultGenesis()
			gs.RecordIntegrityEnabled = true
			gs.ReviewNeutralityEnabled = false
			gs.ClaimRecordsEnabled = false
			gs.FundSettlementEnabled = false
			gs.ClaimIntegrityEnabled = false
			f := newClaimIntegrityGenesisFixture(t, "claim-integrity-legacy-policy", gs)
			factF := makeTestFact(t, f.k, f.ctx, "v12-lp-fact-f", "Synthetic v12 probe P3: fact F under the legacy review policy.", dedupDomain, "empirical", v11Addr("v12-lp-author"), 800_000).Id
			if standing {
				factK := makeTestFact(t, f.k, f.ctx, "v12-lp-fact-k", "Synthetic v12 probe P3: counter-fact K, which contradicts fact F.", dedupDomain, "empirical", v11Addr("v12-lp-jones"), 800_000).Id
				require.NoError(t, f.k.SetFactRelation(f.ctx, &types.FactRelation{SourceFactId: factK, TargetFactId: factF, Relation: types.RelationType_RELATION_TYPE_CONTRADICTS}))
			}
			c, err := f.contradict(v11Addr("v12-lp-contra"), factF, "Synthetic v12 probe P3: a legacy-policy contradiction of fact F.", dedupDomain, v11ReviewFee)
			require.NoError(t, err)
			require.Zero(t, f.claim(t, c).ReviewPolicyVersion)
			require.Equal(t, csContested, f.fact(t, factF).Status)
			history := f.transitions(factF)

			for _, enable := range []func(context.Context) error{f.k.EnableReviewNeutrality, f.k.EnableClaimRecords, f.k.EnableFundSettlement, f.k.EnableClaimIntegrity} {
				require.NoError(t, enable(f.ctx))
			}
			round := f.roundOf(t, c)
			require.Zero(t, round.ReviewPolicyVersion)
			f.advanceTo(t, round.AggregationDeadline)
			if tc.malformedMarker {
				f.ctx.KVStore(f.key).Set([]byte(keeper.ClaimIntegrityEnabledStoreKey), []byte{2})
			}
			f.freshEvents()
			f.beginBlock(t)
			require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_EXPIRED, f.roundOf(t, c).Phase)
			if tc.malformedMarker {
				// Without the fault this state releases F (standing=false).
				require.Equal(t, csContested, f.fact(t, factF).Status, "a malformed marker is never read as knowledge 11")
				require.Empty(t, v11Event(f.ctx, csReversedEvent))
				require.Equal(t, history, f.transitions(factF))
				return
			}
			require.Equal(t, types.ClaimStatus_CLAIM_STATUS_INSUFFICIENT, f.claim(t, c).Status)
			if standing {
				require.Equal(t, csContested, f.fact(t, factF).Status)
				require.Empty(t, v11Event(f.ctx, csReversedEvent))
				require.Equal(t, history, f.transitions(factF))
			} else {
				require.Equal(t, csVerified, f.fact(t, factF).Status)
				require.Equal(t, csReversed(factF, c), v11Event(f.ctx, csReversedEvent))
				require.Equal(t, append(history, v11Transition{csContested, csVerified, "contradiction_reversed", c}), f.transitions(factF))
			}
		})
	}
}
