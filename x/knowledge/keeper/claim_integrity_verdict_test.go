package keeper_test

import (
	"fmt"
	"math/big"
	"sort"
	"strings"
	"testing"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
//     a contest reason remains. A survival (REJECT) clears only dependencies
//     refuted before its challenge opened (review fix SEC-1); any other ending
//     clears nothing, so reasons are judged since the fact's last survived
//     challenge. A standing counter-fact is never cleared by a survival;
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
// however the challenge ends without refuting it, whatever its standing
// before the challenge (review G1R, which reverts G1's SEC-6 fallback). The
// CONTESTED F here depends on a refuted D; R2 is read before the unreadable
// incoming edge, so unless F survives a challenge opened after D fell, D is
// found and the check finishes. A target whose check fails stays CONTESTED
// when its counter-fact falls; and if the fallen counter-fact's own relations
// cannot be listed, no target is released and the event names the
// counter-fact. The disproval itself stands.
func TestClaimIntegrity_IncompleteCheckOnVerdictPathsNeverReleases(t *testing.T) {
	unreadableIncoming := func(f *v11Fixture, factID string) {
		f.ctx.KVStore(f.key).Set(types.FactRelationReverseKey(factID, "unreadable-source"), []byte{0xff})
	}
	for _, before := range []types.FactStatus{csVerified, csContested} {
		for _, ending := range contestEndings {
			t.Run("challenge of "+before.String()+" F "+ending.name, func(t *testing.T) {
				f := setupClaimIntegrityTest(t)
				var factF string
				if before == csContested {
					var factD string
					_, factF, factD = f.vdDependent(t, "ic")
					f.vdDisprove(t, v11Addr("v12-vd-ic-cd"), factD)
					f.nextBlock(t)
				} else {
					_, factF, _ = f.contraTarget(t, "v12-vd-ic-f", "Synthetic v12 probe P3c: fact F, whose contest check cannot finish.")
				}
				unreadableIncoming(f, factF)
				history := f.transitions(factF)
				c := f.challenge(t, v11Addr("v12-vd-ic-ch"), factF)
				f.freshEvents()
				ending.end(t, f, c)
				cause := "challenge_inconclusive"
				if ending.name == "rejected" {
					cause = "challenge_survived"
				}
				require.Equal(t, csContested, f.fact(t, factF).Status, "an incomplete check never releases")
				require.Equal(t, append(history,
					v11Transition{before, vdChallenged, "challenge_opened", c},
					v11Transition{vdChallenged, csContested, cause, c}), f.transitions(factF))
				incomplete := v11Event(f.ctx, csIncompleteEvent)
				if before == csContested && ending.name != "rejected" {
					require.Empty(t, incomplete, "refuted D is found first and was never cleared")
					f.requireNoLiveRounds(t)
					return
				}
				require.Len(t, incomplete, 1)
				require.Equal(t, factF, incomplete[0]["fact_id"])
				require.Contains(t, incomplete[0]["reason"], "incoming relation")
				f.requireNoLiveRounds(t)
			})
		}
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

// ─── Review fixes, group G1: the contest predicate at a challenge's end ──────

var vdAttestation = &types.ReviewAttestation{Reason: "Synthetic v12 review of the stated record only."}

func vdSalt(i int) []byte { return []byte(fmt.Sprintf("v12-vd-review-salt-%02d", i)) }

// vdCommit seats the three reviewers on claimID's round, each committing vote.
// Unlike decide it stops there, so two rounds can be interleaved.
func (f *v11Fixture) vdCommit(t *testing.T, claimID, vote string) {
	t.Helper()
	round := f.roundOf(t, claimID)
	require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_COMMIT, round.Phase)
	for i, reviewer := range v11Reviewers() {
		f.bank.balances[reviewer] = sdk.NewCoins(sdk.NewInt64Coin("uzrn", 100_000_000))
		hash, err := types.ComputeReviewCommitmentV2(round.CommitmentChainId, round.Id, reviewer, vote, v11Confidence, vdSalt(i), vdAttestation)
		require.NoError(t, err)
		_, err = f.server.SubmitCommitment(f.ctx, &types.MsgSubmitCommitment{Verifier: reviewer, RoundId: round.Id, CommitHash: hash})
		require.NoError(t, err)
	}
}

// vdReveal reveals the votes vdCommit committed.
func (f *v11Fixture) vdReveal(t *testing.T, claimID, vote string) {
	t.Helper()
	round := f.roundOf(t, claimID)
	require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_REVEAL, round.Phase)
	for i, reviewer := range v11Reviewers() {
		_, err := f.server.SubmitReveal(f.ctx, &types.MsgSubmitReveal{
			Verifier: reviewer, RoundId: round.Id, Vote: vote, Confidence: v11Confidence, Salt: vdSalt(i), Attestation: vdAttestation,
		})
		require.NoError(t, err)
	}
}

// vdDependent accepts fact D and then fact F, which REQUIRES D.
func (f *v11Fixture) vdDependent(t *testing.T, tag string) (claimF, factF, factD string) {
	t.Helper()
	_, factD, _ = f.contraTarget(t, "v12-vd-"+tag+"-d", "Synthetic v12 review G1 ("+tag+"): fact D, on which fact F depends.")
	claimF = f.mustSubmitClaim(t, v11Addr("v12-vd-"+tag+"-f"), "Synthetic v12 review G1 ("+tag+"): fact F, which requires fact D to hold.", dedupDomain,
		func(m *types.MsgSubmitClaim) {
			m.Relations = []*types.ClaimRelation{{Relation: types.RelationType_RELATION_TYPE_REQUIRES, TargetFactId: factD}}
		})
	factF = f.accept(t, claimF, v11Reviewers()...)
	return claimF, factF, factD
}

// SEC-1, the shield challenge. F REQUIRES D. F is challenged, and D is
// refuted while that challenge is open; the cascade skips F because it is
// CHALLENGED. F then survives. A panel judges the evidence of the challenge
// it was given, so its survival clears only refutations from before the
// challenge opened: F returns to CONTESTED. A later challenge, opened after D
// fell, that F survives clears D. With the marker off F returns to ACTIVE, as
// knowledge 11 does.
func TestClaimIntegrity_SurvivalClearsNothingRefutedWhileTheChallengeWasOpen(t *testing.T) {
	for _, sel := range dedupSelections {
		t.Run(sel.name, func(t *testing.T) {
			f := sel.setup(t)
			claimF, factF, factD := f.vdDependent(t, "sh")
			cd := f.challenge(t, v11Addr("v12-vd-sh-cd"), factD)
			f.nextBlock(t)
			opened := f.ctx.BlockHeight()
			cf := f.challenge(t, v11Addr("v12-vd-sh-cf"), factF)
			f.vdCommit(t, cd, "accept")
			f.vdCommit(t, cf, "reject")
			rd, rf := f.roundOf(t, cd), f.roundOf(t, cf)
			require.Equal(t, rd.CommitDeadline+1, rf.CommitDeadline, "D's round runs one block ahead of F's")

			f.advanceTo(t, rd.CommitDeadline)
			f.beginBlock(t)
			f.vdReveal(t, cd, "accept")
			f.advanceTo(t, rf.CommitDeadline)
			f.beginBlock(t)
			require.Equal(t, csDisproven, f.fact(t, factD).Status)
			refuted := f.ctx.BlockHeight()
			require.Greater(t, refuted, opened, "D fell while F's challenge was open")
			require.Equal(t, vdChallenged, f.fact(t, factF).Status, "the cascade skips a CHALLENGED fact")

			f.vdReveal(t, cf, "reject")
			f.advanceTo(t, rf.CommitDeadline+1)
			f.freshEvents()
			f.beginBlock(t)
			require.Equal(t, types.Verdict_VERDICT_REJECT, f.roundOf(t, cf).Verdict)
			require.Greater(t, f.ctx.BlockHeight(), refuted, "F survived after D fell")
			want := []v11Transition{{csUnspecified, csVerified, "verification", claimF}}
			if !sel.integrity {
				require.Equal(t, vdActive, f.fact(t, factF).Status, "knowledge 11 restores F")
				require.Equal(t, append(want, v11Transition{csVerified, vdChallenged, "unknown", ""}, v11Transition{vdChallenged, vdActive, "unknown", ""}), f.transitions(factF))
				f.requireNoLiveRounds(t)
				return
			}
			require.Equal(t, csContested, f.fact(t, factF).Status, "a survival never clears a refutation from after its challenge opened")
			want = append(want, v11Transition{csVerified, vdChallenged, "challenge_opened", cf}, v11Transition{vdChallenged, csContested, "challenge_survived", cf})
			require.Equal(t, want, f.transitions(factF))
			require.Empty(t, v11Event(f.ctx, csIncompleteEvent))

			// Starving a later challenge clears nothing; surviving one opened
			// after D fell clears D.
			f.nextBlock(t)
			starved := f.challenge(t, v11Addr("v12-vd-sh-c2"), factF)
			f.starve(t, starved)
			require.Equal(t, csContested, f.fact(t, factF).Status)
			f.nextBlock(t)
			c3 := f.challenge(t, v11Addr("v12-vd-sh-c3"), factF)
			require.Equal(t, types.Verdict_VERDICT_REJECT, f.decide(t, c3, "reject", v11Reviewers()...).Verdict)
			require.Equal(t, vdActive, f.fact(t, factF).Status)
			want = append(want,
				v11Transition{csContested, vdChallenged, "challenge_opened", starved}, v11Transition{vdChallenged, csContested, "challenge_inconclusive", starved},
				v11Transition{csContested, vdChallenged, "challenge_opened", c3}, v11Transition{vdChallenged, vdActive, "challenge_survived", c3})
			require.Equal(t, want, f.transitions(factF))
			require.Empty(t, v11Event(f.ctx, csIncompleteEvent))
			f.requireNoLiveRounds(t)
		})
	}
}

// vdShieldChallenge runs the SEC-1 shield challenge on F, which REQUIRES D. D
// is challenged, and F one block later; D is refuted while F is CHALLENGED, so
// the cascade skips F, and F's challenge then ends as ending says (reject,
// malformed or starve). With the marker on, R2 finds D and F returns to
// CONTESTED with a complete check. With it off F returns to ACTIVE, as
// knowledge 11 does.
func (f *v11Fixture) vdShieldChallenge(t *testing.T, integrity bool, tag, ending, claimF, factF, factD string) {
	t.Helper()
	cd := f.challenge(t, v11Addr("v12-vd-"+tag+"-cd"), factD)
	f.nextBlock(t)
	cf := f.challenge(t, v11Addr("v12-vd-"+tag+"-cf"), factF)
	f.vdCommit(t, cd, "accept")
	if ending != "starve" {
		f.vdCommit(t, cf, ending)
	}
	rd, rf := f.roundOf(t, cd), f.roundOf(t, cf)
	require.Equal(t, rd.CommitDeadline+1, rf.CommitDeadline, "D's round runs one block ahead of F's")
	f.advanceTo(t, rd.CommitDeadline)
	f.beginBlock(t)
	f.vdReveal(t, cd, "accept")
	f.advanceTo(t, rf.CommitDeadline)
	f.beginBlock(t)
	require.Equal(t, csDisproven, f.fact(t, factD).Status)
	require.Equal(t, vdChallenged, f.fact(t, factF).Status, "the cascade skips a CHALLENGED fact")

	f.freshEvents()
	cause := "challenge_inconclusive"
	if ending == "starve" {
		f.starve(t, cf)
	} else {
		f.vdReveal(t, cf, ending)
		f.advanceTo(t, rf.CommitDeadline+1)
		f.beginBlock(t)
		verdict := types.Verdict_VERDICT_MALFORMED
		if ending == "reject" {
			verdict, cause = types.Verdict_VERDICT_REJECT, "challenge_survived"
		}
		require.Equal(t, verdict, f.roundOf(t, cf).Verdict)
	}
	want := []v11Transition{{csUnspecified, csVerified, "verification", claimF}}
	if !integrity {
		require.Equal(t, vdActive, f.fact(t, factF).Status, "knowledge 11 restores F")
		require.Equal(t, append(want, v11Transition{csVerified, vdChallenged, "unknown", ""}, v11Transition{vdChallenged, vdActive, "unknown", ""}), f.transitions(factF))
		f.requireNoLiveRounds(t)
		return
	}
	require.Equal(t, csContested, f.fact(t, factF).Status, "R2 finds the refuted dependency")
	require.Equal(t, append(want,
		v11Transition{csVerified, vdChallenged, "challenge_opened", cf},
		v11Transition{vdChallenged, csContested, cause, cf}), f.transitions(factF))
	require.Empty(t, v11Event(f.ctx, csIncompleteEvent), "the check finished")
	f.requireNoLiveRounds(t)
}

// SEC-1 on a heavily cited fact (review G1, second pass). F REQUIRES D, and
// more facts cite F than one contest check may read (the bound is lowered to 8
// records). D is refuted while a challenge of F is open, so the cascade skips
// F. However that challenge ends without refuting F, R2 is read before R1 and
// finds D before the bound is reached: F returns to CONTESTED, and the check
// is complete. With the marker off F returns to ACTIVE, as knowledge 11 does.
func TestClaimIntegrity_ShieldChallengeOfHeavilyCitedFactLaundersNothing(t *testing.T) {
	const bound = 8
	for _, sel := range dedupSelections {
		for _, ending := range []string{"reject", "malformed", "starve"} {
			t.Run(sel.name+"/"+ending, func(t *testing.T) {
				f := sel.setup(t)
				keeper.SetContestScanMaxRecordsForTest(t, bound)
				claimF, factF, factD := f.vdDependent(t, "hub")
				for i := 0; i <= bound; i++ {
					require.NoError(t, f.k.SetFactRelation(f.ctx, &types.FactRelation{
						SourceFactId: fmt.Sprintf("v12-vd-hub-citing-%02d", i), TargetFactId: factF, Relation: types.RelationType_RELATION_TYPE_CITES,
					}))
				}
				f.vdShieldChallenge(t, sel.integrity, "hub", ending, claimF, factF, factD)
			})
		}
	}
}

// vdBeforeActivation runs fn with the claim-integrity marker lifted, as if fn
// ran before activation, and then restores the marker. With the marker off it
// only runs fn.
func (f *v11Fixture) vdBeforeActivation(t *testing.T, fn func()) {
	t.Helper()
	kv := f.ctx.KVStore(f.key)
	key := []byte(keeper.ClaimIntegrityEnabledStoreKey)
	marker := kv.Get(key)
	if marker != nil {
		kv.Delete(key)
		defer kv.Set(key, marker)
	}
	fn()
}

// vdPassAugmentation stores augmentation augID of original and applies a
// passing (EQUIVALENT) verdict through the real writer.
func (f *v11Fixture) vdPassAugmentation(t *testing.T, augID, original string) {
	t.Helper()
	require.NoError(t, f.k.SetAugmentation(f.ctx, &types.Augmentation{
		Id: augID, OriginalFactId: original, VariantContent: "Synthetic v12 review G1: a variant of fact D.",
	}))
	require.NoError(t, f.k.ApplyFinalizedAugmentationVerdict(f.ctx, augID, types.AugmentationVerdict_AUGMENTATION_VERDICT_EQUIVALENT))
}

// SEC-1 behind an augmentation edge (review G1, third pass). F REQUIRES D.
// Augmentation IDs are chosen by their submitter, and a passing verdict
// (ApplyFinalizedAugmentationVerdict) wrote a REFORMULATES edge whose source
// is the augmentation's ID. An augmentation named "<F>/0" therefore put its
// edge under F's forward-index prefix, next to F's own edges. With the marker
// on the writer no longer writes such an edge (review G1R), so the verdict
// runs before activation, where the edge is still written: the closed set of
// pre-activation edges. R2 reads F's dependencies from F's own claim and
// never reads that entry, so the shield challenge leaves F CONTESTED and the
// check complete. With the marker off F returns to ACTIVE, as knowledge 11
// does.
func TestClaimIntegrity_ShieldChallengeBehindAugmentationEdgeLaundersNothing(t *testing.T) {
	for _, sel := range dedupSelections {
		for _, ending := range []string{"reject", "malformed", "starve"} {
			t.Run(sel.name+"/"+ending, func(t *testing.T) {
				f := sel.setup(t)
				claimF, factF, factD := f.vdDependent(t, "aug")
				augID := factF + "/0"
				f.vdBeforeActivation(t, func() { f.vdPassAugmentation(t, augID, factD) })
				outgoing, err := f.k.GetFactRelations(f.ctx, factF)
				require.NoError(t, err)
				var listed [][3]string
				for _, rel := range outgoing {
					listed = append(listed, [3]string{rel.SourceFactId, rel.TargetFactId, rel.Relation.String()})
				}
				require.ElementsMatch(t, [][3]string{
					{factF, factD, types.RelationType_RELATION_TYPE_REQUIRES.String()},
					{augID, factD, types.RelationType_RELATION_TYPE_REFORMULATES.String()},
				}, listed, "the augmentation's edge lies under F's prefix")
				f.vdShieldChallenge(t, sel.integrity, "aug", ending, claimF, factF, factD)
			})
		}
	}
}

// The byte-bound shield (review G1R). F REQUIRES D. Before activation, passing
// augmentations whose long IDs extend F's ID put their REFORMULATES edges
// under F's forward-index prefix, together more bytes than one contest check
// may read (the byte bound is lowered here to 64 KiB; on chain, about 343 IDs
// of 96 KiB fill 64 MiB). An R2 that scanned that prefix stopped at the bound
// before D, and G1's SEC-6 fallback then returned the shielded F to ACTIVE.
// R2 now reads F's dependencies by point lookups from F's own claim, so the
// inflated prefix is never read: however the shield challenge ends without
// refuting F, F returns to CONTESTED and the check is complete. With the
// marker off F returns to ACTIVE, as knowledge 11 does.
func TestClaimIntegrity_ShieldChallengeBehindLongAugmentationIDsLaundersNothing(t *testing.T) {
	const byteBound, idPad = 64 << 10, 8 << 10
	pad := strings.Repeat("x", idPad)
	for _, sel := range dedupSelections {
		for _, ending := range []string{"reject", "malformed", "starve"} {
			t.Run(sel.name+"/"+ending, func(t *testing.T) {
				f := sel.setup(t)
				keeper.SetContestScanMaxBytesForTest(t, byteBound)
				claimF, factF, factD := f.vdDependent(t, "bb")
				f.vdBeforeActivation(t, func() {
					for i := 0; i*2*idPad <= byteBound; i++ {
						f.vdPassAugmentation(t, fmt.Sprintf("%s/%02d%s", factF, i, pad), factD)
					}
				})
				prefix := types.FactRelationsBySourcePrefix(factF)
				it := f.ctx.KVStore(f.key).Iterator(prefix, storetypes.PrefixEndBytes(prefix))
				size := 0
				for ; it.Valid(); it.Next() {
					size += len(it.Key()) + len(it.Value())
				}
				require.NoError(t, it.Close())
				require.Greater(t, size, byteBound, "F's forward prefix alone exceeds the byte bound")
				f.vdShieldChallenge(t, sel.integrity, "bb", ending, claimF, factF, factD)
			})
		}
	}
}

// F1 and CI-REVIEW-2, the same-block tie. F's challenge and D's challenge end
// in the same BeginBlock, F's round first: F survives, and only then is D
// refuted and the cascade contests F. Heights cannot order two events of one
// block, so a refutation at the height a survival is judged from counts
// against it. A starved challenge of F afterwards leaves F CONTESTED. With the
// marker off F is CONTESTED and cannot be challenged.
func TestClaimIntegrity_SameBlockRefutationAfterSurvivalIsNotCleared(t *testing.T) {
	for _, sel := range dedupSelections {
		t.Run(sel.name, func(t *testing.T) {
			f := sel.setup(t)
			claimF, factF, factD := f.vdDependent(t, "sb")
			height := uint64(f.ctx.BlockHeight())
			cfAddr := v11Addr("v12-vd-sb-cf")
			roundF := keeper.GenerateRoundID(keeper.GenerateClaimID(cfAddr, factF, height), height)
			// Rounds complete in round-ID order. Pick D's challenger so that
			// F's round completes first.
			cdAddr := ""
			for i := 0; i < 32 && cdAddr == ""; i++ {
				candidate := v11Addr(fmt.Sprintf("v12-vd-sb-cd%02d", i))
				if keeper.GenerateRoundID(keeper.GenerateClaimID(candidate, factD, height), height) > roundF {
					cdAddr = candidate
				}
			}
			require.NotEmpty(t, cdAddr)
			cf := f.challenge(t, cfAddr, factF)
			cd := f.challenge(t, cdAddr, factD)
			require.Equal(t, roundF, f.roundOf(t, cf).Id)
			f.vdCommit(t, cf, "reject")
			f.vdCommit(t, cd, "accept")
			round := f.roundOf(t, cf)
			require.Equal(t, round.CommitDeadline, f.roundOf(t, cd).CommitDeadline)
			f.advanceTo(t, round.CommitDeadline)
			f.beginBlock(t)
			f.vdReveal(t, cf, "reject")
			f.vdReveal(t, cd, "accept")
			f.advanceTo(t, round.CommitDeadline+1)
			f.beginBlock(t)
			require.Equal(t, types.Verdict_VERDICT_REJECT, f.roundOf(t, cf).Verdict)
			require.Equal(t, csDisproven, f.fact(t, factD).Status)
			require.Equal(t, csContested, f.fact(t, factF).Status, "D fell after F survived, in the same block")
			opened, survived := "unknown", "unknown"
			openedID, survivedID := "", ""
			if sel.integrity {
				opened, survived, openedID, survivedID = "challenge_opened", "challenge_survived", cf, cf
			}
			want := []v11Transition{
				{csUnspecified, csVerified, "verification", claimF},
				{csVerified, vdChallenged, opened, openedID},
				{vdChallenged, vdActive, survived, survivedID},
				{vdActive, csContested, "cascade", factD},
			}
			require.Equal(t, want, f.transitions(factF))
			if !sel.integrity {
				f.requireChallengeRefused(t, v11Addr("v12-vd-sb-c2"), factF, f.vdMinimum(t, factF).String(), vdNotChallengeable(factF, csContested, false))
				f.requireNoLiveRounds(t)
				return
			}

			f.nextBlock(t)
			starved := f.challenge(t, v11Addr("v12-vd-sb-c2"), factF)
			f.freshEvents()
			f.starve(t, starved)
			require.Equal(t, csContested, f.fact(t, factF).Status, "a starved challenge launders nothing")
			require.Equal(t, append(want,
				v11Transition{csContested, vdChallenged, "challenge_opened", starved},
				v11Transition{vdChallenged, csContested, "challenge_inconclusive", starved}), f.transitions(factF))
			require.Empty(t, v11Event(f.ctx, csIncompleteEvent))
			f.requireNoLiveRounds(t)
		})
	}
}

// SEC-5 and decision (b): an EXPIRED counter-fact can be revived by patronage
// or metabolism, so it has not fallen and still holds F (R1). Neither a
// contradiction ending nor any non-refuting ending of a challenge of F
// releases F while K is EXPIRED. With the marker off, knowledge 11 restores F
// when the contradiction starves, whatever K's status.
func TestClaimIntegrity_ExpiredCounterFactStillHoldsTarget(t *testing.T) {
	expire := func(t *testing.T, f *v11Fixture, factID string) {
		t.Helper()
		k := f.fact(t, factID)
		k.Status = types.FactStatus_FACT_STATUS_EXPIRED
		require.NoError(t, f.k.SetFact(f.ctx, k))
	}
	for _, sel := range dedupSelections {
		t.Run(sel.name+"/contradiction starves", func(t *testing.T) {
			f := sel.setup(t)
			_, factF, _, factK := f.vdContested(t, "ex")
			expire(t, f, factK)
			history := f.transitions(factF)
			f.nextBlock(t)
			b := f.mustContradict(t, v11Addr("v12-vd-ex-b"), factF, "Synthetic v12 review SEC-5: a contradiction of fact F that starves while K is expired.", f.vdMinimum(t, factF).String())
			f.freshEvents()
			f.starve(t, b)
			if !sel.integrity {
				require.Equal(t, csVerified, f.fact(t, factF).Status, "knowledge 11 restores F")
				return
			}
			require.Equal(t, csContested, f.fact(t, factF).Status, "an EXPIRED counter-fact still stands")
			require.Equal(t, history, f.transitions(factF))
			require.Empty(t, v11Event(f.ctx, csReversedEvent))
			require.Empty(t, v11Event(f.ctx, csIncompleteEvent))
			f.requireNoLiveRounds(t)
		})
	}
	for _, ending := range contestEndings {
		t.Run("marker on/challenge "+ending.name, func(t *testing.T) {
			f := setupClaimIntegrityTest(t)
			_, factF, _, factK := f.vdContested(t, "ex")
			expire(t, f, factK)
			history := f.transitions(factF)
			c := f.challenge(t, v11Addr("v12-vd-ex-ch"), factF)
			ending.end(t, f, c)
			cause := "challenge_inconclusive"
			if ending.name == "rejected" {
				cause = "challenge_survived"
			}
			require.Equal(t, csContested, f.fact(t, factF).Status, "an EXPIRED counter-fact still stands")
			require.Equal(t, append(history,
				v11Transition{csContested, vdChallenged, "challenge_opened", c},
				v11Transition{vdChallenged, csContested, cause, c}), f.transitions(factF))
			f.requireNoLiveRounds(t)
		})
	}
}

// The disclosed limitation that replaced SEC-6 (review G1R). A heavily cited
// fact makes every contest check that finds no reason in F's dependencies
// reach the scan bound (lowered here to 8 records). An incomplete check at a
// challenge's end never releases: F ends CONTESTED whatever its standing
// before the challenge, and contest_check_incomplete names it once. So one
// starved, refunded challenge leaves a very heavily cited fact CONTESTED
// (griefing, disclosed); G1's fallback, which returned it to ACTIVE, let any
// attacker-inflatable read launder a shielded fact. In the contested-before
// cases F depends on a refuted D: only a survival of a challenge opened after
// D fell clears D and so reaches the bound; any other ending finds D first
// and finishes. With the marker off nothing is checked and F returns to
// ACTIVE.
func TestClaimIntegrity_IncompleteCheckAtChallengeEndLeavesFactContested(t *testing.T) {
	const bound = 8
	cite := func(t *testing.T, f *v11Fixture, factID string) {
		t.Helper()
		for i := 0; i <= bound; i++ {
			require.NoError(t, f.k.SetFactRelation(f.ctx, &types.FactRelation{
				SourceFactId: fmt.Sprintf("v12-vd-citing-%02d", i), TargetFactId: factID, Relation: types.RelationType_RELATION_TYPE_CITES,
			}))
		}
	}
	for _, sel := range dedupSelections {
		for _, ending := range contestEndings {
			t.Run(sel.name+"/not contested before/"+ending.name, func(t *testing.T) {
				f := sel.setup(t)
				keeper.SetContestScanMaxRecordsForTest(t, bound)
				_, factF, _ := f.contraTarget(t, "v12-vd-cb-f", "Synthetic v12 review SEC-6: fact F, which many facts cite.")
				cite(t, f, factF)
				history := f.transitions(factF)
				c := f.challenge(t, v11Addr("v12-vd-cb-ch"), factF)
				f.freshEvents()
				ending.end(t, f, c)
				if !sel.integrity {
					require.Equal(t, vdActive, f.fact(t, factF).Status, "knowledge 11 restores F")
					require.Empty(t, v11Event(f.ctx, csIncompleteEvent))
					return
				}
				require.Equal(t, csContested, f.fact(t, factF).Status, "an incomplete check never releases")
				cause := "challenge_inconclusive"
				if ending.name == "rejected" {
					cause = "challenge_survived"
				}
				require.Equal(t, append(history,
					v11Transition{csVerified, vdChallenged, "challenge_opened", c},
					v11Transition{vdChallenged, csContested, cause, c}), f.transitions(factF))
				incomplete := v11Event(f.ctx, csIncompleteEvent)
				require.Len(t, incomplete, 1)
				require.Equal(t, factF, incomplete[0]["fact_id"])
				require.Contains(t, incomplete[0]["reason"], "contest scan exceeds record bound")
				f.requireNoLiveRounds(t)
			})
		}
	}
	for _, ending := range contestEndings {
		t.Run("marker on/contested before/"+ending.name, func(t *testing.T) {
			f := setupClaimIntegrityTest(t)
			keeper.SetContestScanMaxRecordsForTest(t, bound)
			_, factF, factD := f.vdDependent(t, "cc")
			cite(t, f, factF)
			f.vdDisprove(t, v11Addr("v12-vd-cc-cd"), factD)
			require.Equal(t, csContested, f.fact(t, factF).Status)
			history := f.transitions(factF)
			f.nextBlock(t)
			c := f.challenge(t, v11Addr("v12-vd-cc-ch"), factF)
			f.freshEvents()
			ending.end(t, f, c)
			cause := "challenge_inconclusive"
			if ending.name == "rejected" {
				cause = "challenge_survived"
			}
			require.Equal(t, csContested, f.fact(t, factF).Status, "an incomplete check never releases")
			require.Equal(t, append(history,
				v11Transition{csContested, vdChallenged, "challenge_opened", c},
				v11Transition{vdChallenged, csContested, cause, c}), f.transitions(factF))
			incomplete := v11Event(f.ctx, csIncompleteEvent)
			if ending.name != "rejected" {
				require.Empty(t, incomplete, "refuted D is found before the bound and was never cleared")
				f.requireNoLiveRounds(t)
				return
			}
			require.Len(t, incomplete, 1)
			require.Equal(t, factF, incomplete[0]["fact_id"])
			require.Contains(t, incomplete[0]["reason"], "contest scan exceeds record bound")
			f.requireNoLiveRounds(t)
		})
	}
}
