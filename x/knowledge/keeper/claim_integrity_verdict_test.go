package keeper_test

import (
	"fmt"
	"math/big"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Claim integrity, Task 13 (P3c): participants decide the target, and its
// status follows the latest finding (plan §5.2 rows 5-8). With the marker on:
//   - MsgChallengeFact also accepts a CONTESTED fact, at the same
//     confidence-scaled minimum stake;
//   - a challenge that ends without refuting its target (REJECT, MALFORMED,
//     INCONCLUSIVE or a starved panel) returns it to ACTIVE (PROVISIONAL for a
//     conjecture), except that an established fact returns to CONTESTED while
//     a contest reason remains. A survival (REJECT) clears dependencies
//     refuted at or before its own height; any other ending clears nothing,
//     so reasons are judged since the fact's last survived challenge. A
//     standing counter-fact is never cleared by a survival;
//   - when a counter-fact K is disproven, each CONTESTED target of its stored
//     CONTRADICTS edges returns to VERIFIED, with cause counterfact_disproven/K,
//     once no contest reason remains.
// With the marker off, a CONTESTED fact cannot be challenged and a falling
// counter-fact re-evaluates nothing; the marker-off twins pin that.
//
// Reviewers and challengers are never parties to the rounds they decide, so
// Task 14 does not change these tests.

const (
	vdActive     = types.FactStatus_FACT_STATUS_ACTIVE
	vdChallenged = types.FactStatus_FACT_STATUS_CHALLENGED

	vdRestoredEvent = "zerone.knowledge.challenge_inconclusive_restored"
)

func vdNotChallengeable(factID string, status types.FactStatus, integrity bool) string {
	accepted := "verified or active"
	if integrity {
		accepted = "verified, active or contested"
	}
	return fmt.Sprintf("fact %s is not in a challengeable state (status: %s) (commitment 4: only %s facts accept probes; degraded facts are already moving)",
		factID, status, accepted)
}

// vdMinimum is the stake that challenging factID costs at its confidence.
func (f *v11Fixture) vdMinimum(t *testing.T, factID string) *big.Int {
	t.Helper()
	params, err := f.k.GetParams(f.ctx)
	require.NoError(t, err)
	return keeper.EffectiveMinChallengeStake(params, f.fact(t, factID).Confidence)
}

// vdChallengeAt opens a MsgChallengeFact and returns its challenge claim ID.
func (f *v11Fixture) vdChallengeAt(challenger, factID, stake string) (string, error) {
	response, err := f.server.ChallengeFact(f.ctx, &types.MsgChallengeFact{
		Challenger: challenger, FactId: factID, Stake: stake, Reason: "Synthetic v12 probe P3c: a challenge that a panel decides.",
	})
	if err != nil {
		return "", err
	}
	round, found := f.k.GetVerificationRound(f.ctx, response.RoundId)
	if !found {
		return "", fmt.Errorf("round %s missing", response.RoundId)
	}
	return round.ClaimId, nil
}

// requireChallengeRefused requires an exact refusal that moves no coins and
// writes nothing.
func (f *v11Fixture) requireChallengeRefused(t *testing.T, challenger, factID, stake, want string) {
	t.Helper()
	before, _ := f.storeDigest(t)
	sends := len(f.bank.sendCalls)
	_, err := f.vdChallengeAt(challenger, factID, stake)
	require.EqualError(t, err, want)
	require.Len(t, f.bank.sendCalls, sends, "a refusal moves no coins")
	after, _ := f.storeDigest(t)
	require.Equal(t, before, after, "a refusal writes nothing")
}

// vdContested accepts fact F and then a contradiction A of it, so counter-fact
// K stands with a stored CONTRADICTS edge to F and F is CONTESTED. With the
// marker off F was already contested when A was admitted.
func (f *v11Fixture) vdContested(t *testing.T, tag string) (claimF, factF, a, factK string) {
	t.Helper()
	claimF, factF, minimum := f.contraTarget(t, "v12-vd-"+tag+"-f", "Synthetic v12 probe P3c ("+tag+"): fact F, which counter-fact K contradicts.")
	a = f.mustContradict(t, v11Addr("v12-vd-"+tag+"-k"), factF, "Synthetic v12 probe P3c ("+tag+"): accepted finding K, which contradicts fact F.", minimum)
	factK = f.accept(t, a, v11Reviewers()...)
	require.Equal(t, csContested, f.fact(t, factF).Status)
	require.Equal(t, csVerified, f.fact(t, factK).Status)
	return claimF, factF, a, factK
}

// vdDisprove refutes factID through a real challenge panel and returns the
// challenge claim ID.
func (f *v11Fixture) vdDisprove(t *testing.T, challenger, factID string) string {
	t.Helper()
	c := f.challenge(t, challenger, factID)
	f.accept(t, c, v11Reviewers()...)
	require.Equal(t, csDisproven, f.fact(t, factID).Status)
	return c
}

func vdCounterReversed(factID, counterFactID string) []map[string]string {
	return []map[string]string{{"fact_id": factID, "reverted_by_fact": counterFactID}}
}

// Plan §7 #11. With the marker on, MsgChallengeFact accepts a VERIFIED, ACTIVE
// or CONTESTED fact and refuses every other status; a CONTESTED target costs
// the same confidence-scaled minimum. With the marker off a CONTESTED fact is
// refused exactly as knowledge 11 refuses it. Every refusal moves no coins
// and writes nothing.
func TestClaimIntegrity_ChallengeFactAdmitsContestedTarget(t *testing.T) {
	var statuses []int
	for value := range types.FactStatus_name {
		statuses = append(statuses, int(value))
	}
	sort.Ints(statuses)
	require.Len(t, statuses, 13)

	for _, sel := range dedupSelections {
		t.Run(sel.name, func(t *testing.T) {
			f := sel.setup(t)
			_, factF, _, _ := f.vdContested(t, "gate")
			history := f.transitions(factF)
			minimum := f.vdMinimum(t, factF)
			challenger := v11Addr("v12-vd-gate-ch")
			if !sel.integrity {
				f.requireChallengeRefused(t, challenger, factF, minimum.String(), vdNotChallengeable(factF, csContested, false))
				require.Equal(t, history, f.transitions(factF))
			} else {
				below := new(big.Int).Sub(minimum, big.NewInt(1)).String()
				f.requireChallengeRefused(t, challenger, factF, below,
					fmt.Sprintf("challenge stake %s below effective minimum %s (fact confidence %d) (commitment 4: probe cost scales with confidence — high-confidence facts must be the cheapest to test relative to what is at stake)",
						below, minimum, f.fact(t, factF).Confidence))
				sends := len(f.bank.sendCalls)
				c, err := f.vdChallengeAt(challenger, factF, minimum.String())
				require.NoError(t, err)
				require.Equal(t, minimum.String()+"uzrn", f.bank.sendCalls[sends].coins.String(), "the stake is locked")
				require.Equal(t, vdChallenged, f.fact(t, factF).Status)
				require.Equal(t, append(history, v11Transition{csContested, vdChallenged, "challenge_opened", c}), f.transitions(factF))
				f.starve(t, c)
			}

			_, factG, _ := f.contraTarget(t, "v12-vd-gate-g", "Synthetic v12 probe P3c: an accepted fact whose status is varied in a cache.")
			stake := f.vdMinimum(t, factG).String()
			for _, value := range statuses {
				status := types.FactStatus(value)
				f.inCache(func() {
					fact := f.fact(t, factG)
					fact.Status = status
					require.NoError(t, f.k.SetFact(f.ctx, fact))
					admitted := status == csVerified || status == vdActive || (sel.integrity && status == csContested)
					if !admitted {
						f.requireChallengeRefused(t, v11Addr("v12-vd-gate-gc"), factG, stake, vdNotChallengeable(factG, status, sel.integrity))
						return
					}
					_, err := f.vdChallengeAt(v11Addr("v12-vd-gate-gc"), factG, stake)
					require.NoError(t, err, "status %s", status)
					require.Equal(t, vdChallenged, f.fact(t, factG).Status)
				})
			}
			f.requireNoLiveRounds(t)
		})
	}
}

// With F CONTESTED while counter-fact K stands, a challenge of F is decided
// by its panel: ACCEPT makes F DISPROVEN, and every other ending returns F to
// CONTESTED, because a survival never clears a standing counter-fact (R1).
// A starved challenge launders nothing. K is untouched throughout.
func TestClaimIntegrity_ContestedFactChallengeWhileCounterFactStands(t *testing.T) {
	type ending struct {
		name  string
		end   func(*testing.T, *v11Fixture, string)
		next  types.FactStatus
		cause string
	}
	endings := []ending{{"accepted", func(t *testing.T, f *v11Fixture, id string) { f.accept(t, id, v11Reviewers()...) }, csDisproven, "challenge_disproven"}}
	for _, e := range contestEndings {
		cause := "challenge_inconclusive"
		if e.name == "rejected" {
			cause = "challenge_survived"
		}
		endings = append(endings, ending{e.name, e.end, csContested, cause})
	}
	for _, e := range endings {
		t.Run(e.name, func(t *testing.T) {
			f := setupClaimIntegrityTest(t)
			_, factF, _, factK := f.vdContested(t, "cs")
			history := f.transitions(factF)
			c := f.challenge(t, v11Addr("v12-vd-cs-ch"), factF)
			require.Equal(t, vdChallenged, f.fact(t, factF).Status)
			f.freshEvents()
			e.end(t, f, c)
			require.Equal(t, e.next, f.fact(t, factF).Status)
			require.Equal(t, append(history,
				v11Transition{csContested, vdChallenged, "challenge_opened", c},
				v11Transition{vdChallenged, e.next, e.cause, c}), f.transitions(factF))
			if e.next == csContested && e.cause == "challenge_inconclusive" {
				require.Equal(t, []map[string]string{{"fact_id": factF, "challenge_claim_id": c}}, v11Event(f.ctx, vdRestoredEvent))
			}
			require.Empty(t, v11Event(f.ctx, csReversedEvent))
			require.Empty(t, v11Event(f.ctx, csIncompleteEvent))
			require.Equal(t, csVerified, f.fact(t, factK).Status, "a challenge of F never touches K")
			f.requireNoLiveRounds(t)
		})
	}
}

// A fact contested by the cascade (dependency D1 refuted at h1) is decided by
// challenges of it. A starved challenge leaves it CONTESTED; a REJECT at
// h2 > h1 makes it ACTIVE, and a later starved challenge does not re-contest
// it on account of D1. A dependency refuted after that survival (D2) contests
// it again through the cascade, a starved challenge keeps it CONTESTED, and
// only a new survival clears D2. With the marker off a CONTESTED fact cannot
// be challenged.
func TestClaimIntegrity_SurvivalClearsOnlyDependenciesRefutedBeforeIt(t *testing.T) {
	for _, sel := range dedupSelections {
		t.Run(sel.name, func(t *testing.T) {
			f := sel.setup(t)
			_, factD1, _ := f.contraTarget(t, "v12-vd-sv-d1", "Synthetic v12 probe P3c: fact D1, on which fact F depends.")
			_, factD2, _ := f.contraTarget(t, "v12-vd-sv-d2", "Synthetic v12 probe P3c: fact D2, on which fact F also depends.")
			claimF := f.mustSubmitClaim(t, v11Addr("v12-vd-sv-f"), "Synthetic v12 probe P3c: fact F, which requires facts D1 and D2 to hold.", dedupDomain,
				func(m *types.MsgSubmitClaim) {
					m.Relations = []*types.ClaimRelation{
						{Relation: types.RelationType_RELATION_TYPE_REQUIRES, TargetFactId: factD1},
						{Relation: types.RelationType_RELATION_TYPE_REQUIRES, TargetFactId: factD2},
					}
				})
			factF := f.accept(t, claimF, v11Reviewers()...)
			f.vdDisprove(t, v11Addr("v12-vd-sv-cd1"), factD1)
			want := []v11Transition{
				{csUnspecified, csVerified, "verification", claimF},
				{csVerified, csContested, "cascade", factD1},
			}
			require.Equal(t, want, f.transitions(factF))
			if !sel.integrity {
				f.requireChallengeRefused(t, v11Addr("v12-vd-sv-ch0"), factF, f.vdMinimum(t, factF).String(), vdNotChallengeable(factF, csContested, false))
				return
			}

			step := func(challenger string, end func(*testing.T, *v11Fixture, string), next types.FactStatus, cause string) {
				t.Helper()
				prior := f.fact(t, factF).Status
				c := f.challenge(t, v11Addr(challenger), factF)
				end(t, f, c)
				require.Equal(t, next, f.fact(t, factF).Status, "after %s", cause)
				want = append(want, v11Transition{prior, vdChallenged, "challenge_opened", c}, v11Transition{vdChallenged, next, cause, c})
				require.Equal(t, want, f.transitions(factF))
			}
			starve := func(t *testing.T, f *v11Fixture, id string) { f.starve(t, id) }
			reject := contestEndings[0].end
			f.freshEvents()
			step("v12-vd-sv-ch1", starve, csContested, "challenge_inconclusive")
			step("v12-vd-sv-ch2", reject, vdActive, "challenge_survived")
			step("v12-vd-sv-ch3", starve, vdActive, "challenge_inconclusive")

			f.vdDisprove(t, v11Addr("v12-vd-sv-cd2"), factD2)
			want = append(want, v11Transition{vdActive, csContested, "cascade", factD2})
			require.Equal(t, want, f.transitions(factF))
			step("v12-vd-sv-ch4", starve, csContested, "challenge_inconclusive")
			step("v12-vd-sv-ch5", reject, vdActive, "challenge_survived")
			require.Empty(t, v11Event(f.ctx, csIncompleteEvent))
			f.requireNoLiveRounds(t)
		})
	}
}

// Plan §7 #12. Challenging counter-fact K with ACCEPT makes K DISPROVEN and
// releases F to VERIFIED, with cause counterfact_disproven/K and a
// contradiction_reversed event naming K. With the marker off nothing
// re-evaluates F: it stays CONTESTED with an unchanged history.
func TestClaimIntegrity_DisprovenCounterFactReleasesTarget(t *testing.T) {
	for _, sel := range dedupSelections {
		t.Run(sel.name, func(t *testing.T) {
			f := sel.setup(t)
			_, factF, _, factK := f.vdContested(t, "rk")
			history := f.transitions(factF)
			ck := f.challenge(t, v11Addr("v12-vd-rk-ch"), factK)
			f.freshEvents()
			f.accept(t, ck, v11Reviewers()...)
			require.Equal(t, csDisproven, f.fact(t, factK).Status)
			kHistory := f.transitions(factK)
			if sel.integrity {
				require.Equal(t, v11Transition{vdChallenged, csDisproven, "challenge_disproven", ck}, kHistory[len(kHistory)-1])
				require.Equal(t, csVerified, f.fact(t, factF).Status, "K fell and no contest reason remains")
				require.Equal(t, append(history, v11Transition{csContested, csVerified, "counterfact_disproven", factK}), f.transitions(factF))
				require.Equal(t, vdCounterReversed(factF, factK), v11Event(f.ctx, csReversedEvent))
			} else {
				require.Equal(t, v11Transition{vdChallenged, csDisproven, "challenge_disproven", ""}, kHistory[len(kHistory)-1])
				require.Equal(t, csContested, f.fact(t, factF).Status, "knowledge 11 never re-evaluates F")
				require.Equal(t, history, f.transitions(factF))
				require.Empty(t, v11Event(f.ctx, csReversedEvent))
			}
			require.Empty(t, v11Event(f.ctx, csIncompleteEvent))
			f.requireNoLiveRounds(t)
		})
	}
}

// With two standing counter-facts, disproving one leaves F CONTESTED (R1 on
// the other) and records nothing for F; disproving the second releases it.
func TestClaimIntegrity_DisprovingOneOfTwoCounterFactsKeepsTargetContested(t *testing.T) {
	for _, sel := range dedupSelections {
		t.Run(sel.name, func(t *testing.T) {
			f := sel.setup(t)
			_, factF, minimum := f.contraTarget(t, "v12-vd-two-f", "Synthetic v12 probe P3c: fact F, which two accepted findings contradict.")
			a1 := f.mustContradict(t, v11Addr("v12-vd-two-k1"), factF, "Synthetic v12 probe P3c: first accepted finding K1, which contradicts fact F.", minimum)
			factK1 := f.accept(t, a1, v11Reviewers()...)
			a2 := f.mustContradict(t, v11Addr("v12-vd-two-k2"), factF, "Synthetic v12 probe P3c: second accepted finding K2, which contradicts fact F.", minimum)
			factK2 := f.accept(t, a2, v11Reviewers()...)
			require.Equal(t, csContested, f.fact(t, factF).Status)
			history := f.transitions(factF)

			f.freshEvents()
			f.vdDisprove(t, v11Addr("v12-vd-two-c1"), factK1)
			require.Equal(t, csContested, f.fact(t, factF).Status, "K2 still stands")
			require.Equal(t, history, f.transitions(factF))
			require.Empty(t, v11Event(f.ctx, csReversedEvent))

			f.freshEvents()
			f.vdDisprove(t, v11Addr("v12-vd-two-c2"), factK2)
			if sel.integrity {
				require.Equal(t, csVerified, f.fact(t, factF).Status)
				require.Equal(t, append(history, v11Transition{csContested, csVerified, "counterfact_disproven", factK2}), f.transitions(factF))
				require.Equal(t, vdCounterReversed(factF, factK2), v11Event(f.ctx, csReversedEvent))
			} else {
				require.Equal(t, csContested, f.fact(t, factF).Status)
				require.Equal(t, history, f.transitions(factF))
				require.Empty(t, v11Event(f.ctx, csReversedEvent))
			}
			require.Empty(t, v11Event(f.ctx, csIncompleteEvent))
			f.requireNoLiveRounds(t)
		})
	}
}

// F is contested by an accepted counter-fact K, and its dependency D is then
// refuted; the cascade skips F because it is already CONTESTED. When K falls,
// R2 still holds F, because F has survived no challenge since D fell. A
// challenge of F that it survives then clears both.
func TestClaimIntegrity_DisprovenCounterFactLeavesRefutedDependencyHolding(t *testing.T) {
	f := setupClaimIntegrityTest(t)
	_, factD, _ := f.contraTarget(t, "v12-vd-rd-d", "Synthetic v12 probe P3c: fact D, on which fact F depends.")
	claimF := f.mustSubmitClaim(t, v11Addr("v12-vd-rd-f"), "Synthetic v12 probe P3c: fact F, which requires fact D and which K contradicts.", dedupDomain,
		func(m *types.MsgSubmitClaim) {
			m.Relations = []*types.ClaimRelation{{Relation: types.RelationType_RELATION_TYPE_REQUIRES, TargetFactId: factD}}
		})
	factF := f.accept(t, claimF, v11Reviewers()...)
	a := f.mustContradict(t, v11Addr("v12-vd-rd-k"), factF, "Synthetic v12 probe P3c: accepted finding K, which contradicts fact F.", f.vdMinimum(t, factF).String())
	factK := f.accept(t, a, v11Reviewers()...)
	want := []v11Transition{
		{csUnspecified, csVerified, "verification", claimF},
		{csVerified, csContested, "contradiction_accepted", a},
	}
	require.Equal(t, want, f.transitions(factF))

	f.vdDisprove(t, v11Addr("v12-vd-rd-cd"), factD)
	require.Equal(t, want, f.transitions(factF), "the cascade skips a fact that is already CONTESTED")
	f.freshEvents()
	f.vdDisprove(t, v11Addr("v12-vd-rd-ck"), factK)
	require.Equal(t, csContested, f.fact(t, factF).Status, "R2: D fell and F has survived no challenge since")
	require.Equal(t, want, f.transitions(factF))
	require.Empty(t, v11Event(f.ctx, csReversedEvent))
	require.Empty(t, v11Event(f.ctx, csIncompleteEvent))

	c := f.challenge(t, v11Addr("v12-vd-rd-ch"), factF)
	require.Equal(t, types.Verdict_VERDICT_REJECT, f.decide(t, c, "reject", v11Reviewers()...).Verdict)
	require.Equal(t, vdActive, f.fact(t, factF).Status)
	require.Equal(t, append(want,
		v11Transition{csContested, vdChallenged, "challenge_opened", c},
		v11Transition{vdChallenged, vdActive, "challenge_survived", c}), f.transitions(factF))
	f.requireNoLiveRounds(t)
}

// Plan §3.6 and §5.6: a fact laundered before activation (VERIFIED while the
// accepted counter-fact K stands, probe P3) is left as it is, and either side
// can be challenged. When a challenge of it ends without refuting it, R1
// returns it to CONTESTED, after an in-place upgrade or an import. With the
// marker off it returns to ACTIVE, as knowledge 11 does.
func TestClaimIntegrity_LaunderedFactChallengeFollowsStandingCounterFact(t *testing.T) {
	activations := append([]struct {
		name     string
		activate func(*testing.T, *v11Fixture) *v11Fixture
	}{{"marker off", func(_ *testing.T, f *v11Fixture) *v11Fixture { return f }}}, contestActivations...)
	for _, activation := range activations {
		for _, ending := range contestEndings {
			t.Run(activation.name+"/"+ending.name, func(t *testing.T) {
				f := setupV11SemanticsTest(t)
				_, factF, _, factK := f.vdContested(t, "ld")
				f.nextBlock(t)
				b := f.mustContradict(t, v11Addr("v12-vd-ld-b"), factF, "Synthetic v12 probe P3c: a contradiction of fact F that starves and launders it.", f.vdMinimum(t, factF).String())
				f.starve(t, b)
				require.Equal(t, csVerified, f.fact(t, factF).Status, "knowledge 11 laundered F while K stands")

				g := activation.activate(t, f)
				history := g.transitions(factF)
				c := g.challenge(t, v11Addr("v12-vd-ld-ch"), factF)
				g.freshEvents()
				ending.end(t, g, c)
				cause := "challenge_inconclusive"
				if ending.name == "rejected" {
					cause = "challenge_survived"
				}
				next := csContested
				if activation.name == "marker off" {
					next, cause, c = vdActive, "unknown", ""
				}
				require.Equal(t, next, g.fact(t, factF).Status)
				opened := v11Transition{csVerified, vdChallenged, "challenge_opened", c}
				if activation.name == "marker off" {
					opened = v11Transition{csVerified, vdChallenged, "unknown", ""}
				}
				require.Equal(t, append(history, opened, v11Transition{vdChallenged, next, cause, c}), g.transitions(factF))
				require.Equal(t, csVerified, g.fact(t, factK).Status)
				require.Empty(t, v11Event(g.ctx, csIncompleteEvent))
				g.requireNoLiveRounds(t)
			})
		}
	}
}

// A conjecture is exempt from contest reasons when a challenge of it ends:
// it returns to PROVISIONAL, never ACTIVE and never CONTESTED, even while a
// counter-fact stands. A record that predates the contradiction gate can
// leave one CONTESTED; MsgChallengeFact accepts it at the unscaled minimum.
// When the counter-fact falls, the conjecture is released to PROVISIONAL.
func TestClaimIntegrity_ConjectureChallengeEndsProvisional(t *testing.T) {
	for _, ending := range contestEndings {
		t.Run(ending.name, func(t *testing.T) {
			f := setupClaimIntegrityTest(t)
			conjecture, err := f.server.PostConjecture(f.ctx, &types.MsgPostConjecture{
				Proposer: v11Addr("v12-vd-cj-proposer"), Domain: dedupDomain, Category: "empirical", Stake: v11ReviewFee,
				Statement:              "Synthetic v12 probe P3c: conjecture Q, open until a counterexample is shown.",
				FalsificationPredicate: "Exhibit one synthetic counterexample to conjecture Q in this fixture.",
			})
			require.NoError(t, err)
			conjFact := f.accept(t, conjecture.ClaimId, v11Reviewers()...)
			_, factK, _ := f.contraTarget(t, "v12-vd-cj-k", "Synthetic v12 probe P3c: fact K, which a legacy record lets contradict conjecture Q.")
			// Model the legacy record: K contradicts the conjecture, which is CONTESTED.
			require.NoError(t, f.k.SetFactRelation(f.ctx, &types.FactRelation{SourceFactId: factK, TargetFactId: conjFact, Relation: types.RelationType_RELATION_TYPE_CONTRADICTS}))
			fact := f.fact(t, conjFact)
			fact.Status = csContested
			require.NoError(t, f.k.SetFact(f.ctx, fact))
			history := f.transitions(conjFact)

			c := f.challenge(t, v11Addr("v12-vd-cj-ch"), conjFact)
			ending.end(t, f, c)
			cause := "challenge_inconclusive"
			if ending.name == "rejected" {
				cause = "challenge_survived"
			}
			require.Equal(t, csProvisional, f.fact(t, conjFact).Status, "a conjecture is restored to PROVISIONAL while K stands")
			want := append(history,
				v11Transition{csContested, vdChallenged, "challenge_opened", c},
				v11Transition{vdChallenged, csProvisional, cause, c})
			require.Equal(t, want, f.transitions(conjFact))

			fact = f.fact(t, conjFact)
			fact.Status = csContested
			require.NoError(t, f.k.SetFact(f.ctx, fact))
			want = f.transitions(conjFact)
			f.freshEvents()
			f.vdDisprove(t, v11Addr("v12-vd-cj-ck"), factK)
			require.Equal(t, csProvisional, f.fact(t, conjFact).Status, "released to PROVISIONAL, never VERIFIED")
			require.Equal(t, append(want, v11Transition{csContested, csProvisional, "counterfact_disproven", factK}), f.transitions(conjFact))
			require.Equal(t, vdCounterReversed(conjFact, factK), v11Event(f.ctx, csReversedEvent))
			f.requireNoLiveRounds(t)
		})
	}
}

// Plan §5.3 failure handling on the new paths: a check that cannot finish
// never releases. A challenged fact whose check fails returns to CONTESTED
// however its challenge ends without refuting it; a target whose check fails
// stays CONTESTED when its counter-fact falls; and if the fallen
// counter-fact's own relations cannot be listed, no target is released and
// the event names the counter-fact. The disproval itself stands.
func TestClaimIntegrity_IncompleteCheckOnVerdictPathsFailsTowardContested(t *testing.T) {
	unreadableIncoming := func(f *v11Fixture, factID string) {
		f.ctx.KVStore(f.key).Set(types.FactRelationReverseKey(factID, "unreadable-source"), []byte{0xff})
	}
	for _, ending := range contestEndings {
		t.Run("challenge of F "+ending.name, func(t *testing.T) {
			f := setupClaimIntegrityTest(t)
			_, factF, _ := f.contraTarget(t, "v12-vd-ic-f", "Synthetic v12 probe P3c: fact F, whose contest check cannot finish.")
			unreadableIncoming(f, factF)
			history := f.transitions(factF)
			c := f.challenge(t, v11Addr("v12-vd-ic-ch"), factF)
			f.freshEvents()
			ending.end(t, f, c)
			cause := "challenge_inconclusive"
			if ending.name == "rejected" {
				cause = "challenge_survived"
			}
			require.Equal(t, csContested, f.fact(t, factF).Status, "an incomplete check never restores")
			require.Equal(t, append(history,
				v11Transition{csVerified, vdChallenged, "challenge_opened", c},
				v11Transition{vdChallenged, csContested, cause, c}), f.transitions(factF))
			incomplete := v11Event(f.ctx, csIncompleteEvent)
			require.Len(t, incomplete, 1)
			require.Equal(t, factF, incomplete[0]["fact_id"])
			require.Contains(t, incomplete[0]["reason"], "incoming relation")
			f.requireNoLiveRounds(t)
		})
	}
	for _, tc := range []struct {
		name   string
		inject func(f *v11Fixture, factF, factK string)
		named  func(factF, factK string) string
		reason string
	}{
		{"target check incomplete", func(f *v11Fixture, factF, _ string) { unreadableIncoming(f, factF) },
			func(factF, _ string) string { return factF }, "incoming relation"},
		{"counter-fact relations unreadable", func(f *v11Fixture, _, factK string) {
			f.ctx.KVStore(f.key).Set(types.FactRelationKey(factK, "unreadable-target"), []byte{0xff})
		}, func(_, factK string) string { return factK }, "counter-fact relations"},
	} {
		t.Run("counter-fact falls/"+tc.name, func(t *testing.T) {
			f := setupClaimIntegrityTest(t)
			_, factF, _, factK := f.vdContested(t, "if")
			tc.inject(f, factF, factK)
			history := f.transitions(factF)
			f.freshEvents()
			f.vdDisprove(t, v11Addr("v12-vd-if-ck"), factK)
			require.Equal(t, csContested, f.fact(t, factF).Status, "an incomplete check never releases")
			require.Equal(t, history, f.transitions(factF))
			require.Empty(t, v11Event(f.ctx, csReversedEvent))
			incomplete := v11Event(f.ctx, csIncompleteEvent)
			require.Len(t, incomplete, 1)
			require.Equal(t, tc.named(factF, factK), incomplete[0]["fact_id"])
			require.Contains(t, incomplete[0]["reason"], tc.reason)
			f.requireNoLiveRounds(t)
		})
	}
}
