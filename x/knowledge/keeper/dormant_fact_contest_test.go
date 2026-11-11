package keeper_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// ─── Dormant-fact contest (consensus-repairs-v3, marker 0x95) ───────────────
//
// A fact that gains no energy goes AT_RISK, EXPIRED and PRUNED. With the
// marker on it stays open to challenge and contradiction, a PRUNED
// conjecture stays an open question, and a contest of it does not move it:
// while the contest is open and after it ends without refuting the fact,
// the fact is what it would have been without the contest. These tests run
// the real handlers, review panels and metabolism on real stores, in both
// marker states. Every dormant status here is reached through real
// metabolism epochs, so the fact's at-risk stamp and the domain counters are
// metabolism's own.

const (
	dfcChallenged = types.FactStatus_FACT_STATUS_CHALLENGED
	dfcDisproven  = types.FactStatus_FACT_STATUS_DISPROVEN
	dfcAtRisk     = types.FactStatus_FACT_STATUS_AT_RISK
	dfcExpired    = types.FactStatus_FACT_STATUS_EXPIRED
	dfcPruned     = types.FactStatus_FACT_STATUS_PRUNED
)

var dormantStatuses = []types.FactStatus{dfcAtRisk, dfcExpired, dfcPruned}

// dormantFixture is DefaultGenesis, the native policy-1 chain (review
// neutrality, record integrity and claim integrity on), with the dormant-fact
// contest selected or not. epoch is the last metabolism epoch run.
type dormantFixture struct {
	*v11Fixture
	epoch uint64
}

func newDormantFixture(t *testing.T, dormant bool) *dormantFixture {
	t.Helper()
	gs := types.DefaultGenesis()
	gs.BootstrapFundAllocation = "22222000000"
	gs.DormantFactContestEnabled = dormant
	f := newClaimIntegrityGenesisFixture(t, "dormant-contest-test", gs)
	enabled, err := f.k.DormantFactContestEnabled(f.ctx)
	require.NoError(t, err)
	require.Equal(t, dormant, enabled)
	return &dormantFixture{v11Fixture: f}
}

// assertedFact accepts an assertion in dedupDomain through a real panel.
func (f *dormantFixture) assertedFact(t *testing.T, tag string) (claimID, factID string) {
	t.Helper()
	claimID, factID, _ = f.contraTarget(t, "dfc-a-"+tag, dormantAssertionText(tag))
	return claimID, factID
}

func dormantAssertionText(tag string) string {
	return "Synthetic dormant-fact contest: an accepted assertion that nobody cites (" + tag + ")."
}

// conjectureFact accepts a conjecture in dedupDomain through a real panel.
func (f *dormantFixture) conjectureFact(t *testing.T, tag string) (claimID, factID string) {
	t.Helper()
	claimID, err := f.postConjectureIn(v11Addr("dfc-q-"+tag), dedupDomain, "Synthetic dormant-fact contest: a conjecture that nobody answers ("+tag+").")
	require.NoError(t, err)
	factID = f.accept(t, claimID)
	require.True(t, keeper.IsConjecture(f.fact(t, factID)))
	return claimID, factID
}

// decayTo runs real metabolism epochs until factID reaches status.
func (f *dormantFixture) decayTo(t *testing.T, factID string, status types.FactStatus) {
	t.Helper()
	f.metabolizeUntil(t, factID, &f.epoch, status)
}

// metabolize runs one more real metabolism epoch.
func (f *dormantFixture) metabolize(t *testing.T) {
	t.Helper()
	f.epoch++
	require.NoError(t, f.k.ProcessMetabolism(f.ctx, f.epoch))
}

// challengeProvisional opens a MsgChallengeProvisionalFact at the minimum
// stake and returns its challenge claim ID.
func (f *dormantFixture) challengeProvisional(challenger, claimID, factID string) (string, error) {
	params, err := f.k.GetParams(f.ctx)
	if err != nil {
		return "", err
	}
	response, err := f.server.ChallengeProvisionalFact(f.ctx, &types.MsgChallengeProvisionalFact{
		Challenger: challenger, FactId: factID, ClaimId: claimID, Stake: params.MinChallengeStake,
		Reason: "Synthetic dormant-fact contest: a refutation attempt that a panel decides.",
	})
	if err != nil {
		return "", err
	}
	round, found := f.k.GetVerificationRound(f.ctx, response.ChallengeId)
	if !found {
		return "", fmt.Errorf("round %s missing", response.ChallengeId)
	}
	return round.ClaimId, nil
}

// patronize funds factID for a long duration, enough to lift an AT_RISK fact
// back above the active threshold at once.
func (f *dormantFixture) patronize(t *testing.T, patron, factID string) {
	t.Helper()
	f.bank.balances[patron] = sdk.NewCoins(sdk.NewInt64Coin("uzrn", 1_000_000))
	_, err := f.server.PatronizeFact(f.ctx, &types.MsgPatronizeFact{
		Patron: patron, FactId: factID, Amount: "1", DurationBlocks: 100_000_000,
	})
	require.NoError(t, err)
}

// dormantState is everything rule 5 says a contest must leave as it was: the
// fact record (status, energy, at-risk stamp, corroboration, every other
// field), the domain's pressure counters, the training grade and the
// training value.
type dormantState struct {
	fact  *types.Fact
	stats keeper.DomainStats
	tier  types.TrainingQualityTier
	tvw   uint64
}

func (f *dormantFixture) state(t *testing.T, factID string) dormantState {
	t.Helper()
	fact := f.fact(t, factID)
	stats, _ := f.k.GetDomainStats(f.ctx, fact.Domain)
	tier, _ := keeper.ClassifyTrainingQuality(fact)
	tvw, err := f.k.ComputeTrainingValueWeightChecked(f.ctx, factID)
	require.NoError(t, err)
	return dormantState{fact: fact, stats: *stats, tier: tier, tvw: tvw.Final}
}

func requireSameState(t *testing.T, want, got dormantState) {
	t.Helper()
	require.True(t, proto.Equal(want.fact, got.fact), "fact record\nwant %s\ngot  %s", prototext.Format(want.fact), prototext.Format(got.fact))
	require.Equal(t, want.stats, got.stats, "domain counters")
	require.Equal(t, want.tier, got.tier, "training grade")
	require.Equal(t, want.tvw, got.tvw, "training value")
}

// requireUntouched requires that factID is exactly as before and that no
// status transition was recorded for it since.
func (f *dormantFixture) requireUntouched(t *testing.T, factID string, before dormantState, history []v11Transition) {
	t.Helper()
	requireSameState(t, before, f.state(t, factID))
	require.Equal(t, history, f.transitions(factID), "no status transition")
}

// citedEdges lists the "cited" karma edges whose reference is factID.
func (f *dormantFixture) citedEdges(factID string) []map[string]string {
	var out []map[string]string
	for _, edge := range karmaEdgesOfKind(f.ctx.EventManager().Events(), "cited") {
		if edge["ref_id"] == factID {
			out = append(out, edge)
		}
	}
	return out
}

// dormantEnding ends the round of a challenge or contradiction claim through
// a real panel: accepted, rejected, malformed, or inconclusive because no
// panel was seated by the reveal deadline.
type dormantEnding struct {
	name     string
	accepted bool
	end      func(*testing.T, *dormantFixture, string)
}

func dormantDecide(vote string, verdict types.Verdict) func(*testing.T, *dormantFixture, string) {
	return func(t *testing.T, f *dormantFixture, claimID string) {
		t.Helper()
		require.Equal(t, verdict, f.decide(t, claimID, vote, v11Reviewers()...).Verdict)
	}
}

var dormantEndings = []dormantEnding{
	{"accepted", true, dormantDecide("accept", types.Verdict_VERDICT_ACCEPT)},
	{"rejected", false, dormantDecide("reject", types.Verdict_VERDICT_REJECT)},
	{"malformed", false, dormantDecide("malformed", types.Verdict_VERDICT_MALFORMED)},
	{"inconclusive", false, func(t *testing.T, f *dormantFixture, claimID string) { t.Helper(); f.starve(t, claimID) }},
}

// Each of the three messages, for each dormant status, in both marker states.
// Admission is checked in a cache, so the fact keeps decaying in one line.
func TestDormantFactContest_Admission(t *testing.T) {
	for _, dormant := range []bool{false, true} {
		t.Run(fmt.Sprintf("dormant=%t", dormant), func(t *testing.T) {
			f := newDormantFixture(t, dormant)
			_, factF := f.assertedFact(t, "adm")
			g := newDormantFixture(t, dormant)
			claimQ, factQ := g.conjectureFact(t, "adm")

			for _, status := range dormantStatuses {
				f.decayTo(t, factF, status)
				stake := f.vdMinimum(t, factF).String()

				// MsgChallengeFact: off, a dormant fact is refused as today; on,
				// it is admitted and not moved.
				f.inCache(func() {
					if !dormant {
						f.requireChallengeRefused(t, v11Addr("dfc-adm-ch"), factF, stake, vdNotChallengeable(factF, status, true))
						return
					}
					before, history := f.state(t, factF), f.transitions(factF)
					_, err := f.vdChallengeAt(v11Addr("dfc-adm-ch"), factF, stake)
					require.NoError(t, err, "status %s", status)
					f.requireUntouched(t, factF, before, history)
				})

				// MsgSubmitContradiction: AT_RISK already stands; EXPIRED and
				// PRUNED stand only with the marker on. Admission changes no fact.
				f.inCache(func() {
					counter := "Synthetic dormant-fact contest: a counter-observation against a fact in status " + status.String() + "."
					if !dormant && status != dfcAtRisk {
						f.requireContradictionRefused(t, v11Addr("dfc-adm-ct"), factF, counter, stake, contraNotStanding(factF, status))
						return
					}
					before, history := f.state(t, factF), f.transitions(factF)
					f.mustContradict(t, v11Addr("dfc-adm-ct"), factF, counter, stake)
					f.requireUntouched(t, factF, before, history)
				})

				// MsgChallengeProvisionalFact: a lapsed conjecture keeps its own
				// door. AT_RISK and EXPIRED were already open and move to
				// CHALLENGED with the marker off; PRUNED was resolved. With the
				// marker on all three are open and none is moved.
				g.decayTo(t, factQ, status)
				g.inCache(func() {
					before, history := g.state(t, factQ), g.transitions(factQ)
					digest, _ := g.storeDigest(t)
					_, err := g.challengeProvisional(v11Addr("dfc-adm-pc"), claimQ, factQ)
					switch {
					case !dormant && status == dfcPruned:
						require.EqualError(t, err, fmt.Sprintf("fact %s is already resolved (status: %s) — nothing left to refute", factQ, status))
						after, _ := g.storeDigest(t)
						require.Equal(t, digest, after, "a refusal writes nothing")
					case !dormant:
						require.NoError(t, err, "status %s", status)
						require.Equal(t, dfcChallenged, g.fact(t, factQ).Status)
					default:
						require.NoError(t, err, "status %s", status)
						g.requireUntouched(t, factQ, before, history)
					}
				})

				// MsgChallengeFact never admits a conjecture, in either state.
				g.inCache(func() {
					want := vdNotChallengeable(factQ, status, true)
					if dormant {
						want = fmt.Sprintf("fact %s is a conjecture (status: %s); refute it with `tx knowledge challenge-provisional`", factQ, status)
					}
					g.requireChallengeRefused(t, v11Addr("dfc-adm-qc"), factQ, g.vdMinimum(t, factQ).String(), want)
				})
			}
		})
	}
}

// A challenge of a dormant fact, for each status and each ending. With the
// marker off it is refused and nothing changes. With it on, the fact is
// untouched while the round is open (status, energy, at-risk stamp,
// training grade and value, domain counters, history), an accepted challenge
// disproves it as today, and every other ending leaves it untouched.
func TestDormantFactContest_ChallengeEndings(t *testing.T) {
	for _, status := range dormantStatuses {
		for _, ending := range dormantEndings {
			t.Run(status.String()+"/"+ending.name, func(t *testing.T) {
				for _, dormant := range []bool{false, true} {
					f := newDormantFixture(t, dormant)
					_, factF := f.assertedFact(t, "end")
					f.decayTo(t, factF, status)
					before, history := f.state(t, factF), f.transitions(factF)
					stake := f.vdMinimum(t, factF).String()
					if !dormant {
						f.requireChallengeRefused(t, v11Addr("dfc-end-ch"), factF, stake, vdNotChallengeable(factF, status, true))
						requireSameState(t, before, f.state(t, factF))
						continue
					}
					c, err := f.vdChallengeAt(v11Addr("dfc-end-ch"), factF, stake)
					require.NoError(t, err)
					f.requireUntouched(t, factF, before, history)
					ending.end(t, f, c)
					if !ending.accepted {
						f.requireUntouched(t, factF, before, history)
						require.Empty(t, v11Event(f.ctx, "zerone.knowledge.contest_check_incomplete"))
						continue
					}
					after := f.state(t, factF)
					require.Equal(t, dfcDisproven, after.fact.Status)
					require.Equal(t, before.fact.Energy, after.fact.Energy)
					require.Equal(t, before.fact.AtRiskSinceEpoch, after.fact.AtRiskSinceEpoch)
					require.Equal(t, before.fact.CorroborationCount, after.fact.CorroborationCount)
					require.Equal(t, append(history, v11Transition{status, dfcDisproven, "challenge_disproven", c}), f.transitions(factF))
					// As today: the disproven fact moves no counter, and the
					// accepted challenge claim became a new fact in its domain.
					require.Equal(t, before.stats.AtRiskCount, after.stats.AtRiskCount)
					require.Equal(t, before.stats.ActiveCount+1, after.stats.ActiveCount)
				}
			})
		}
	}
}

// A contradiction of a dormant fact, for each status and each ending. Its
// admission never changes the target, and no ending does either: an accepted
// contradiction contests only a VERIFIED or ACTIVE target, and the other
// endings release only a CONTESTED one. With the marker on an accepted
// contradiction also feeds a dormant target no citation energy and gives its
// author no "cited" edge; with it off an AT_RISK target earns both, as today.
// With the marker off, EXPIRED and PRUNED are refused at admission.
func TestDormantFactContest_ContradictionEndings(t *testing.T) {
	for _, status := range dormantStatuses {
		for _, ending := range dormantEndings {
			t.Run(status.String()+"/"+ending.name, func(t *testing.T) {
				for _, dormant := range []bool{false, true} {
					f := newDormantFixture(t, dormant)
					_, factF := f.assertedFact(t, "ct")
					f.decayTo(t, factF, status)
					before, history := f.state(t, factF), f.transitions(factF)
					stake := f.vdMinimum(t, factF).String()
					counter := "Synthetic dormant-fact contest: a counter-observation that a panel decides."
					if !dormant && status != dfcAtRisk {
						f.requireContradictionRefused(t, v11Addr("dfc-ct-c"), factF, counter, stake, contraNotStanding(factF, status))
						requireSameState(t, before, f.state(t, factF))
						continue
					}
					c := f.mustContradict(t, v11Addr("dfc-ct-c"), factF, counter, stake)
					f.requireUntouched(t, factF, before, history)
					ending.end(t, f, c)
					after := f.state(t, factF)
					require.True(t, proto.Equal(before.fact, after.fact), "the target is never moved\nwant %s\ngot  %s",
						prototext.Format(before.fact), prototext.Format(after.fact))
					require.Equal(t, history, f.transitions(factF))
					require.Equal(t, before.tier, after.tier)
					require.Equal(t, before.tvw, after.tvw)
					require.Equal(t, before.stats.AtRiskCount, after.stats.AtRiskCount)
					if !ending.accepted {
						require.Equal(t, before.stats, after.stats)
						require.Zero(t, f.k.GetNewCitationsThisEpoch(f.ctx, factF))
						continue
					}
					// The counter-fact is a new fact in the same domain.
					require.Equal(t, before.stats.ActiveCount+1, after.stats.ActiveCount)
					if dormant {
						require.Zero(t, f.k.GetNewCitationsThisEpoch(f.ctx, factF), "a contest feeds a dormant fact no citation energy")
						require.Empty(t, f.citedEdges(factF), "nor its author a cited edge")
					} else {
						require.Equal(t, uint64(1), f.k.GetNewCitationsThisEpoch(f.ctx, factF), "marker off: as today")
						require.Len(t, f.citedEdges(factF), 1, "marker off: as today")
					}
				}
			})
		}
	}
}

// A refutation attempt on a lapsed conjecture, for each status and each
// ending. With the marker off a PRUNED conjecture is refused, and an AT_RISK
// or EXPIRED one is moved to CHALLENGED and, if it survives, comes back
// PROVISIONAL with its at-risk stamp cleared, as today. With the marker on it
// is never moved. An accepted refutation disproves it in both states.
func TestDormantFactContest_LapsedConjectureEndings(t *testing.T) {
	for _, status := range dormantStatuses {
		for _, ending := range dormantEndings {
			t.Run(status.String()+"/"+ending.name, func(t *testing.T) {
				for _, dormant := range []bool{false, true} {
					f := newDormantFixture(t, dormant)
					claimQ, factQ := f.conjectureFact(t, "lc")
					f.decayTo(t, factQ, status)
					before, history := f.state(t, factQ), f.transitions(factQ)
					c, err := f.challengeProvisional(v11Addr("dfc-lc-pc"), claimQ, factQ)
					if !dormant && status == dfcPruned {
						require.EqualError(t, err, fmt.Sprintf("fact %s is already resolved (status: %s) — nothing left to refute", factQ, status))
						requireSameState(t, before, f.state(t, factQ))
						continue
					}
					require.NoError(t, err)
					if dormant {
						f.requireUntouched(t, factQ, before, history)
					}
					ending.end(t, f, c)
					after := f.state(t, factQ)
					switch {
					case ending.accepted:
						require.Equal(t, dfcDisproven, after.fact.Status)
						require.Equal(t, before.fact.Energy, after.fact.Energy)
						require.Equal(t, before.fact.AtRiskSinceEpoch, after.fact.AtRiskSinceEpoch)
					case dormant:
						f.requireUntouched(t, factQ, before, history)
					default:
						require.Equal(t, types.FactStatus_FACT_STATUS_PROVISIONAL, after.fact.Status, "marker off: restored as today")
						require.Zero(t, after.fact.AtRiskSinceEpoch, "marker off: the at-risk stamp is cleared, as today")
						require.Equal(t, before.fact.Energy, after.fact.Energy)
						require.Equal(t, before.stats, after.stats, "today's restore moves no counter either")
					}
				}
			})
		}
	}
}

// A conjecture that metabolism PRUNED. With the marker on it is listed as an
// open question and can be refuted; while its refutation round is open it
// stays PRUNED (so the listing does not flag it as under challenge), and a
// refutation resolves it. With the marker off it is resolved and unlisted,
// and the refutation door is shut.
func TestDormantFactContest_PrunedConjectureStaysOpen(t *testing.T) {
	for _, dormant := range []bool{false, true} {
		t.Run(fmt.Sprintf("dormant=%t", dormant), func(t *testing.T) {
			f := newDormantFixture(t, dormant)
			claimQ, factQ := f.conjectureFact(t, "open")
			f.decayTo(t, factQ, dfcPruned)
			query := keeper.NewQueryServerImpl(f.k)

			listed := func() *types.OpenQuestion {
				response, err := query.OpenQuestions(f.ctx, &types.QueryOpenQuestionsRequest{Domain: dedupDomain})
				require.NoError(t, err)
				for _, q := range response.Questions {
					if q.FactId == factQ {
						return q
					}
				}
				return nil
			}
			require.Equal(t, dormant, listed() != nil, "a pruned conjecture is an open question only with the marker on")

			c, err := f.challengeProvisional(v11Addr("dfc-open-pc"), claimQ, factQ)
			if !dormant {
				require.EqualError(t, err, fmt.Sprintf("fact %s is already resolved (status: %s) — nothing left to refute", factQ, dfcPruned))
				return
			}
			require.NoError(t, err)
			open := listed()
			require.NotNil(t, open, "under challenge it stays listed")
			require.Equal(t, dfcPruned, f.fact(t, factQ).Status)
			require.False(t, open.UnderChallenge, "it is not held CHALLENGED, so the listing does not flag it")
			f.accept(t, c)
			require.Equal(t, dfcDisproven, f.fact(t, factQ).Status)
			require.Nil(t, listed(), "a refuted conjecture is resolved")
		})
	}
}

// decideByBlock is decide driven by BeginBlocker, the production entry,
// instead of AdvanceRoundPhases. Heights stay below the first fitness epoch.
func (f *dormantFixture) decideByBlock(t *testing.T, claimID, vote string) *types.VerificationRound {
	t.Helper()
	round := f.roundOf(t, claimID)
	attestation := &types.ReviewAttestation{Reason: "Synthetic dormant-fact contest review of the stated record only."}
	for i, reviewer := range v11Reviewers() {
		f.bank.balances[reviewer] = sdk.NewCoins(sdk.NewInt64Coin("uzrn", 100_000_000))
		salt := []byte(fmt.Sprintf("dfc-e2e-salt-%s-%02d", claimID[:8], i))
		hash, err := types.ComputeReviewCommitmentV2(round.CommitmentChainId, round.Id, reviewer, vote, v11Confidence, salt, attestation)
		require.NoError(t, err)
		_, err = f.server.SubmitCommitment(f.ctx, &types.MsgSubmitCommitment{Verifier: reviewer, RoundId: round.Id, CommitHash: hash})
		require.NoError(t, err)
	}
	f.advanceTo(t, round.CommitDeadline)
	require.NoError(t, f.k.BeginBlocker(f.ctx))
	for i, reviewer := range v11Reviewers() {
		_, err := f.server.SubmitReveal(f.ctx, &types.MsgSubmitReveal{
			Verifier: reviewer, RoundId: round.Id, Vote: vote, Confidence: v11Confidence,
			Salt: []byte(fmt.Sprintf("dfc-e2e-salt-%s-%02d", claimID[:8], i)), Attestation: attestation,
		})
		require.NoError(t, err)
	}
	f.advanceTo(t, round.CommitDeadline+1)
	require.NoError(t, f.k.BeginBlocker(f.ctx))
	done := f.roundOf(t, claimID)
	require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_COMPLETE, done.Phase)
	params, err := f.k.GetParams(f.ctx)
	require.NoError(t, err)
	require.Less(t, uint64(f.ctx.BlockHeight()), params.FitnessEpochBlocks, "no fitness epoch ran inside the contest")
	return done
}

// End to end, with the marker on: a fact metabolism PRUNED survives a
// challenge decided through BeginBlocker. While the challenge is open and
// after it, the fact is untouched and its text stays free for a twin; later
// epochs leave it alone; and a second challenge then disproves it.
func TestDormantFactContest_PrunedFactEndToEnd(t *testing.T) {
	f := newDormantFixture(t, true)
	_, factF := f.assertedFact(t, "e2e")
	text := dormantAssertionText("e2e")
	f.decayTo(t, factF, dfcPruned)
	before, history := f.state(t, factF), f.transitions(factF)

	twinAdmitted := func() {
		f.inCache(func() {
			_, err := f.submitClaim(v11Addr("dfc-e2e-twin"), text, dedupDomain, nil)
			require.NoError(t, err, "a pruned fact's text is free for a twin")
		})
	}
	twinAdmitted()

	// It survives. The open challenge buys it nothing.
	survive, err := f.vdChallengeAt(v11Addr("dfc-e2e-ch1"), factF, f.vdMinimum(t, factF).String())
	require.NoError(t, err)
	f.requireUntouched(t, factF, before, history)
	twinAdmitted()
	require.Equal(t, types.Verdict_VERDICT_REJECT, f.decideByBlock(t, survive, "reject").Verdict)
	f.requireUntouched(t, factF, before, history)
	require.Equal(t, []map[string]string(nil), v11Event(f.ctx, "zerone.knowledge.contest_check_incomplete"))
	twinAdmitted()

	// Later epochs leave a PRUNED fact alone: it is not metabolised, counted
	// or revived.
	for i := 0; i < 3; i++ {
		f.metabolize(t)
		f.requireUntouched(t, factF, before, history)
	}

	// It is still open to falsification, and this time it falls.
	refute, err := f.vdChallengeAt(v11Addr("dfc-e2e-ch2"), factF, f.vdMinimum(t, factF).String())
	require.NoError(t, err)
	require.Equal(t, types.Verdict_VERDICT_ACCEPT, f.decideByBlock(t, refute, "accept").Verdict)
	after := f.state(t, factF)
	require.Equal(t, dfcDisproven, after.fact.Status)
	require.Equal(t, before.fact.Energy, after.fact.Energy)
	require.Equal(t, before.stats.AtRiskCount, after.stats.AtRiskCount)
	require.NotEmpty(t, v11Event(f.ctx, "zerone.knowledge.fact_disproven"))
	f.metabolize(t)
	require.Equal(t, dfcDisproven, f.fact(t, factF).Status)
}

// A contest that does not refute an AT_RISK fact leaves it where an
// uncontested twin is, at every epoch, including the epochs that fall inside
// the contest. Two chains with the same history run the same metabolism
// epochs; one also runs the contests of the schedule (each with some
// epochs inside it, chained, or with a patron paying while it is open). The
// two facts have the same status, energy, at-risk stamp, training grade and
// value, and the domains the same counters, until both are PRUNED (or, with
// a patron, for a fixed span).
func TestDormantFactContest_EndsWhereUncontested(t *testing.T) {
	type schedule struct {
		name    string
		inside  []int // metabolism epochs inside each contest, in order
		ending  dormantEnding
		patron  bool
		horizon int // epochs to compare after the last contest; 0 runs to PRUNED
	}
	var schedules []schedule
	for _, ending := range dormantEndings[1:] {
		for _, inside := range []int{0, 1, 5} {
			schedules = append(schedules, schedule{name: fmt.Sprintf("%s/%d-epochs-inside", ending.name, inside), inside: []int{inside}, ending: ending})
		}
	}
	schedules = append(schedules,
		schedule{name: "inconclusive/chained", inside: []int{1, 1, 1}, ending: dormantEndings[3]},
		schedule{name: "rejected/patron-pays-inside", inside: []int{1}, ending: dormantEndings[1], patron: true, horizon: 8},
	)
	for _, s := range schedules {
		t.Run(s.name, func(t *testing.T) {
			contested, quiet := newDormantFixture(t, true), newDormantFixture(t, true)
			_, factA := contested.assertedFact(t, "twin")
			_, factB := quiet.assertedFact(t, "twin")
			require.Equal(t, factA, factB)
			contested.decayTo(t, factA, dfcAtRisk)
			quiet.decayTo(t, factB, dfcAtRisk)
			require.Equal(t, contested.epoch, quiet.epoch)

			compare := func(when string) {
				t.Helper()
				a, b := contested.state(t, factA), quiet.state(t, factB)
				require.Equal(t, b.fact.Status, a.fact.Status, "%s, epoch %d", when, contested.epoch)
				require.Equal(t, b.fact.Energy, a.fact.Energy, "%s, epoch %d", when, contested.epoch)
				require.Equal(t, b.fact.AtRiskSinceEpoch, a.fact.AtRiskSinceEpoch, "%s, epoch %d", when, contested.epoch)
				require.Equal(t, b.tier, a.tier, "%s, epoch %d", when, contested.epoch)
				require.Equal(t, b.tvw, a.tvw, "%s, epoch %d", when, contested.epoch)
				require.Equal(t, b.stats, a.stats, "%s, epoch %d", when, contested.epoch)
			}
			step := func(when string) {
				t.Helper()
				quiet.advanceTo(t, uint64(contested.ctx.BlockHeight()))
				contested.metabolize(t)
				quiet.metabolize(t)
				compare(when)
			}

			for i, inside := range s.inside {
				c, err := contested.vdChallengeAt(v11Addr(fmt.Sprintf("dfc-twin-ch%d", i)), factA, contested.vdMinimum(t, factA).String())
				require.NoError(t, err)
				compare("admitted")
				for e := 0; e < inside; e++ {
					step("inside the contest")
				}
				if s.patron {
					quiet.advanceTo(t, uint64(contested.ctx.BlockHeight()))
					contested.patronize(t, v11Addr("dfc-twin-patron"), factA)
					quiet.patronize(t, v11Addr("dfc-twin-patron"), factB)
					compare("patron paid inside the contest")
					require.Equal(t, types.FactStatus_FACT_STATUS_ACTIVE, contested.fact(t, factA).Status)
				}
				s.ending.end(t, contested, c)
				compare("ended")
			}

			if s.horizon > 0 {
				for i := 0; i < s.horizon; i++ {
					step("after")
				}
				return
			}
			sawExpired := false
			for i := 0; i < 100; i++ {
				sawExpired = sawExpired || contested.fact(t, factA).Status == dfcExpired
				if contested.fact(t, factA).Status == dfcPruned {
					require.True(t, sawExpired, "it passed through EXPIRED on metabolism's own clock")
					return
				}
				step("after")
			}
			t.Fatal("the facts never reached PRUNED")
		})
	}
}

// A counter-fact that metabolism PRUNED no longer holds its target (R1), and
// a challenge of it does not make it hold again. F is CONTESTED by K; K decays
// to PRUNED. With the marker on K is challenged; while that challenge is
// open a challenge of F is rejected, and F returns to ACTIVE exactly as on
// the marker-off chain, where K's challenge is refused. F stays ACTIVE after
// K's challenge ends.
func TestDormantFactContest_PrunedCounterFactHoldsNothing(t *testing.T) {
	for _, dormant := range []bool{false, true} {
		t.Run(fmt.Sprintf("dormant=%t", dormant), func(t *testing.T) {
			f := newDormantFixture(t, dormant)
			_, factF, _, factK := f.vdContested(t, "r1")
			f.decayTo(t, factK, dfcPruned)
			require.Equal(t, csContested, f.fact(t, factF).Status, "metabolism does not touch a CONTESTED fact")
			beforeK, historyK := f.state(t, factK), f.transitions(factK)

			stakeK := f.vdMinimum(t, factK).String()
			var cK string
			if dormant {
				var err error
				cK, err = f.vdChallengeAt(v11Addr("dfc-r1-chk"), factK, stakeK)
				require.NoError(t, err)
				f.requireUntouched(t, factK, beforeK, historyK)
			} else {
				f.requireChallengeRefused(t, v11Addr("dfc-r1-chk"), factK, stakeK, vdNotChallengeable(factK, dfcPruned, true))
			}

			cF, err := f.vdChallengeAt(v11Addr("dfc-r1-chf"), factF, f.vdMinimum(t, factF).String())
			require.NoError(t, err)
			require.Equal(t, types.Verdict_VERDICT_REJECT, f.decide(t, cF, "reject", v11Reviewers()...).Verdict)
			require.Equal(t, types.FactStatus_FACT_STATUS_ACTIVE, f.fact(t, factF).Status, "a pruned counter-fact holds nothing")

			if dormant {
				f.starve(t, cK)
				f.requireUntouched(t, factK, beforeK, historyK)
				require.Equal(t, types.FactStatus_FACT_STATUS_ACTIVE, f.fact(t, factF).Status)
			}
		})
	}
}

// decideTogether seats a panel on each round at once, commits and reveals its
// vote, and aggregates them on one block. A claim with an empty vote gets no
// panel and stays open.
func (f *dormantFixture) decideTogether(t *testing.T, votes map[string]string) {
	t.Helper()
	attestation := &types.ReviewAttestation{Reason: "Synthetic dormant-fact contest review of the stated record only."}
	deadline := uint64(0)
	for claimID, vote := range votes {
		round := f.roundOf(t, claimID)
		require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_COMMIT, round.Phase)
		if deadline != 0 {
			require.Equal(t, deadline, round.CommitDeadline, "the rounds were opened together")
		}
		deadline = round.CommitDeadline
		if vote == "" {
			continue
		}
		for i, reviewer := range v11Reviewers() {
			f.bank.balances[reviewer] = sdk.NewCoins(sdk.NewInt64Coin("uzrn", 100_000_000))
			salt := []byte(fmt.Sprintf("dfc-tog-salt-%s-%02d", claimID[:8], i))
			hash, err := types.ComputeReviewCommitmentV2(round.CommitmentChainId, round.Id, reviewer, vote, v11Confidence, salt, attestation)
			require.NoError(t, err)
			_, err = f.server.SubmitCommitment(f.ctx, &types.MsgSubmitCommitment{Verifier: reviewer, RoundId: round.Id, CommitHash: hash})
			require.NoError(t, err)
		}
	}
	f.advanceTo(t, deadline)
	f.beginBlock(t)
	for claimID, vote := range votes {
		if vote == "" {
			continue
		}
		round := f.roundOf(t, claimID)
		for i, reviewer := range v11Reviewers() {
			_, err := f.server.SubmitReveal(f.ctx, &types.MsgSubmitReveal{
				Verifier: reviewer, RoundId: round.Id, Vote: vote, Confidence: v11Confidence,
				Salt: []byte(fmt.Sprintf("dfc-tog-salt-%s-%02d", claimID[:8], i)), Attestation: attestation,
			})
			require.NoError(t, err)
		}
	}
	f.advanceTo(t, deadline+1)
	f.beginBlock(t)
	for claimID, vote := range votes {
		phase := f.roundOf(t, claimID).Phase
		if vote == "" {
			require.NotEqual(t, types.VerificationPhase_VERIFICATION_PHASE_COMPLETE, phase, "an unstaffed round stays open")
		} else {
			require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_COMPLETE, phase)
		}
	}
}

// A dormant fact is not held CHALLENGED, so two challenges of it can be open
// at once. The first that refutes it disproves it; a second accepted one, or
// a rejected one, leaves the disproven fact, its history and its descendants
// as they are, and fact_disproven is emitted once.
func TestDormantFactContest_ConcurrentChallenges(t *testing.T) {
	for _, second := range []string{"accept", "reject"} {
		t.Run("second="+second, func(t *testing.T) {
			f := newDormantFixture(t, true)
			_, factF := f.assertedFact(t, "conc")
			f.decayTo(t, factF, dfcPruned)
			before, history := f.state(t, factF), f.transitions(factF)
			stake := f.vdMinimum(t, factF).String()
			c1, err := f.vdChallengeAt(v11Addr("dfc-conc-ch1"), factF, stake)
			require.NoError(t, err)
			c2, err := f.vdChallengeAt(v11Addr("dfc-conc-ch2"), factF, stake)
			require.NoError(t, err)
			f.requireUntouched(t, factF, before, history)

			f.freshEvents()
			f.decideTogether(t, map[string]string{c1: "accept", c2: second})
			require.Equal(t, dfcDisproven, f.fact(t, factF).Status)
			require.Len(t, f.transitions(factF), len(history)+1, "one disproval")
			disproven := v11Event(f.ctx, "zerone.knowledge.fact_disproven")
			require.Len(t, disproven, 1)
			require.Equal(t, factF, disproven[0]["fact_id"])
			require.Empty(t, v11Event(f.ctx, "zerone.knowledge.contest_check_incomplete"))
		})
	}
}

// A dormant challenge D holds nothing. If its AT_RISK target recovers through
// patronage while D is open and a second challenge S then holds it
// CHALLENGED, in the same block, D's end leaves S's hold alone (the
// transition names S, not D), and S's own end restores the fact even while D
// is still open: the record, not the active-round scan, decides who holds it.
func TestDormantFactContest_DormantChallengeHoldsNothing(t *testing.T) {
	type ending struct {
		name  string
		votes func(d, s string) map[string]string
		after func(*testing.T, *dormantFixture, string, string, string)
	}
	for _, e := range []ending{
		{"dormant-rejected-first", func(d, s string) map[string]string { return map[string]string{d: "reject", s: ""} },
			func(t *testing.T, f *dormantFixture, factF, d, s string) {
				require.Equal(t, dfcChallenged, f.fact(t, factF).Status, "S still holds it")
				f.starve(t, s)
				require.Equal(t, types.FactStatus_FACT_STATUS_ACTIVE, f.fact(t, factF).Status)
			}},
		{"held-malformed-first", func(d, s string) map[string]string { return map[string]string{d: "", s: "malformed"} },
			func(t *testing.T, f *dormantFixture, factF, d, s string) {
				require.Equal(t, types.FactStatus_FACT_STATUS_ACTIVE, f.fact(t, factF).Status, "D's open round does not keep S's target held")
				f.starve(t, d)
				require.Equal(t, types.FactStatus_FACT_STATUS_ACTIVE, f.fact(t, factF).Status)
			}},
	} {
		t.Run(e.name, func(t *testing.T) {
			f := newDormantFixture(t, true)
			_, factF := f.assertedFact(t, "hold")
			f.decayTo(t, factF, dfcAtRisk)
			d, err := f.vdChallengeAt(v11Addr("dfc-hold-d"), factF, f.vdMinimum(t, factF).String())
			require.NoError(t, err)
			require.Equal(t, dfcAtRisk, f.fact(t, factF).Status)
			f.patronize(t, v11Addr("dfc-hold-patron"), factF)
			require.Equal(t, types.FactStatus_FACT_STATUS_ACTIVE, f.fact(t, factF).Status)
			s, err := f.vdChallengeAt(v11Addr("dfc-hold-s"), factF, f.vdMinimum(t, factF).String())
			require.NoError(t, err)
			require.Equal(t, dfcChallenged, f.fact(t, factF).Status)
			require.Equal(t, f.claim(t, d).SubmittedAtBlock, f.claim(t, s).SubmittedAtBlock, "one block: only the cause tells them apart")

			f.decideTogether(t, e.votes(d, s))
			e.after(t, f, factF, d, s)
			require.Empty(t, v11Event(f.ctx, "zerone.knowledge.contest_check_incomplete"))
		})
	}
}

// ─── The legacy review policy (review neutrality off) ───────────────────────

// legacyDormantFixture is a chain without review neutrality, record
// integrity or claim integrity: every round is policy 0, a survival credits
// energy and corroboration, and an accepted challenge earns a
// confidence-scaled bonus. A stub vesting keeper makes the submitter escrow
// reachable.
type legacyDormantFixture struct {
	k      keeper.Keeper
	ctx    sdk.Context
	server types.MsgServer
	epoch  uint64
}

func newLegacyDormantFixture(t *testing.T, dormant bool) *legacyDormantFixture {
	t.Helper()
	gs := types.DefaultGenesis()
	gs.ReviewNeutralityEnabled = false
	gs.ClaimRecordsEnabled = false
	gs.FundSettlementEnabled = false
	gs.ClaimIntegrityEnabled = false
	gs.RecordIntegrityEnabled = false
	gs.ReformulationVindicationOrderingEnabled = false
	gs.BootstrapFundAllocation = "22222000000"
	gs.DormantFactContestEnabled = dormant
	k, ctx, _ := setupKnowledgeTestWithBankGenesis(t, gs)
	k.SetVestingRewardsKeeper(stubVestingKeeper{})
	return &legacyDormantFixture{k: k, ctx: ctx.WithEventManager(sdk.NewEventManager()), server: keeper.NewMsgServerImpl(k)}
}

// acceptedFact completes a policy-0 round for an assertion with ACCEPT.
func (f *legacyDormantFixture) acceptedFact(t *testing.T, id string) string {
	t.Helper()
	claim := &types.Claim{
		Id: id, FactContent: "Synthetic dormant-fact contest: a legacy-policy assertion (" + id + ").",
		Domain: "physics", Category: "empirical", Submitter: makeValidBech32Addr("dfc-legacy-author"),
		Stake: "1000000", Status: types.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION, SubmittedAtBlock: uint64(f.ctx.BlockHeight()),
	}
	require.NoError(t, f.k.SetClaim(f.ctx, claim))
	round := makeRoundInPhase("r-"+id, id, types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION, 80)
	require.NoError(t, f.k.SetVerificationRound(f.ctx, round))
	require.NoError(t, f.k.CompleteRound(f.ctx, round, &keeper.VerificationResult{Verdict: types.Verdict_VERDICT_ACCEPT, Confidence: 800_000}))
	factID := keeper.GenerateFactID(id, uint64(f.ctx.BlockHeight()))
	_, found := f.k.GetFact(f.ctx, factID)
	require.True(t, found)
	return factID
}

func (f *legacyDormantFixture) decayTo(t *testing.T, factID string, status types.FactStatus) {
	t.Helper()
	for i := 0; i < 1000; i++ {
		fact, _ := f.k.GetFact(f.ctx, factID)
		if fact.Status == status {
			return
		}
		f.epoch++
		require.NoError(t, f.k.ProcessMetabolism(f.ctx, f.epoch))
	}
	t.Fatalf("metabolism did not reach %s", status)
}

func (f *legacyDormantFixture) state(t *testing.T, factID string) dormantState {
	t.Helper()
	fact, found := f.k.GetFact(f.ctx, factID)
	require.True(t, found)
	stats, _ := f.k.GetDomainStats(f.ctx, fact.Domain)
	tier, _ := keeper.ClassifyTrainingQuality(fact)
	tvw, err := f.k.ComputeTrainingValueWeightChecked(f.ctx, factID)
	require.NoError(t, err)
	return dormantState{fact: fact, stats: *stats, tier: tier, tvw: tvw.Final}
}

// challenge opens a MsgChallengeFact at the minimum stake and returns its
// round.
func (f *legacyDormantFixture) challenge(t *testing.T, factID string) (*types.VerificationRound, error) {
	t.Helper()
	fact, _ := f.k.GetFact(f.ctx, factID)
	params, err := f.k.GetParams(f.ctx)
	require.NoError(t, err)
	response, err := f.server.ChallengeFact(f.ctx, &types.MsgChallengeFact{
		Challenger: makeValidBech32Addr("dfc-legacy-ch"), FactId: factID,
		Stake:  keeper.EffectiveMinChallengeStake(params, fact.Confidence).String(),
		Reason: "Synthetic dormant-fact contest: a legacy-policy challenge.",
	})
	if err != nil {
		return nil, err
	}
	round, found := f.k.GetVerificationRound(f.ctx, response.RoundId)
	require.True(t, found)
	require.Equal(t, types.ReviewPolicyLegacy, round.ReviewPolicyVersion)
	return round, nil
}

// challengeProvisional opens a MsgChallengeProvisionalFact at the minimum
// stake and returns its round.
func (f *legacyDormantFixture) challengeProvisional(t *testing.T, factID string) (*types.VerificationRound, error) {
	t.Helper()
	params, err := f.k.GetParams(f.ctx)
	require.NoError(t, err)
	response, err := f.server.ChallengeProvisionalFact(f.ctx, &types.MsgChallengeProvisionalFact{
		Challenger: makeValidBech32Addr("dfc-legacy-pc"), FactId: factID, Stake: params.MinChallengeStake,
		Reason: "Synthetic dormant-fact contest: a legacy-policy refutation attempt.",
	})
	if err != nil {
		return nil, err
	}
	round, found := f.k.GetVerificationRound(f.ctx, response.ChallengeId)
	require.True(t, found)
	require.Equal(t, types.ReviewPolicyLegacy, round.ReviewPolicyVersion)
	return round, nil
}

// On the legacy policy a MALFORMED or INCONCLUSIVE aggregated verdict
// restores no challenged fact (completeRound is pinned), and an accepted
// challenge earns a confidence-scaled bonus. A dormant target held
// CHALLENGED could be held for good; one not held could pay that bonus to
// every concurrent challenger. So the challenge doors apply the contest only
// to a review-neutral claim: on this chain, with the marker on or off, a
// dormant non-conjecture is refused, a PRUNED conjecture is resolved and
// unlisted, and an AT_RISK or EXPIRED conjecture is held CHALLENGED and ends
// as today. Contradictions apply the contest on this policy too.
func TestDormantFactContest_LegacyPolicyChallengesAsToday(t *testing.T) {
	complete := func(t *testing.T, f *legacyDormantFixture, round *types.VerificationRound, verdict types.Verdict) {
		t.Helper()
		require.NoError(t, f.k.CompleteRound(f.ctx, round, &keeper.VerificationResult{Verdict: verdict, Confidence: 800_000}))
	}
	for _, status := range dormantStatuses {
		t.Run(status.String(), func(t *testing.T) {
			type outcome struct {
				refusal string
				after   *types.Fact
			}
			outcomes := map[bool]map[string]outcome{}
			for _, dormant := range []bool{false, true} {
				outcomes[dormant] = map[string]outcome{}

				// A dormant assertion: refused in both states, with today's text.
				f := newLegacyDormantFixture(t, dormant)
				factF := f.acceptedFact(t, "legacy")
				f.decayTo(t, factF, status)
				before := f.state(t, factF)
				_, err := f.challenge(t, factF)
				require.EqualError(t, err, vdNotChallengeable(factF, status, false))
				requireSameState(t, before, f.state(t, factF))

				// A lapsed conjecture.
				for _, verdict := range []types.Verdict{types.Verdict_VERDICT_REJECT, types.Verdict_VERDICT_MALFORMED} {
					g := newLegacyDormantFixture(t, dormant)
					q := acceptConjecture(t, g.k, g.ctx, "dfc-legacy-q", "Synthetic dormant-fact contest: a legacy-policy conjecture.")
					g.decayTo(t, q.Id, status)
					round, err := g.challengeProvisional(t, q.Id)
					if status == dfcPruned {
						require.EqualError(t, err, fmt.Sprintf("fact %s is already resolved (status: %s) — nothing left to refute", q.Id, status))
						questions, _, _, _ := g.k.OpenQuestionsForDomain(g.ctx, q.Domain, 0)
						require.Empty(t, questions, "unlisted where its door is shut")
						outcomes[dormant][verdict.String()] = outcome{refusal: err.Error()}
						continue
					}
					require.NoError(t, err)
					after, _ := g.k.GetFact(g.ctx, q.Id)
					require.Equal(t, dfcChallenged, after.Status, "held CHALLENGED, as today")
					complete(t, g, round, verdict)
					after, _ = g.k.GetFact(g.ctx, q.Id)
					outcomes[dormant][verdict.String()] = outcome{after: after}
				}
			}
			// The marker changes none of it.
			for verdict, off := range outcomes[false] {
				on := outcomes[true][verdict]
				require.Equal(t, off.refusal, on.refusal, verdict)
				require.True(t, proto.Equal(off.after, on.after), "%s\noff %s\non  %s", verdict, prototext.Format(off.after), prototext.Format(on.after))
			}
			if status != dfcPruned {
				require.Equal(t, types.FactStatus_FACT_STATUS_PROVISIONAL, outcomes[true][types.Verdict_VERDICT_REJECT.String()].after.Status)
				require.Equal(t, dfcChallenged, outcomes[true][types.Verdict_VERDICT_MALFORMED.String()].after.Status)
			}
		})
	}
}

// ─── Who holds a challenged fact ─────────────────────────────────────────────

// The end of a challenge restores and credits a CHALLENGED fact only when
// this challenge holds it: the fact's latest status transition is the move
// to CHALLENGED at the height the challenge was submitted, naming it when a
// cause is recorded. Here the record is fabricated on a legacy chain. A
// record of another admission (another height, or another cause) is not
// this challenge's hold: the fact stays CHALLENGED, nothing is credited and
// nothing is reported. A latest transition that is not a move to CHALLENGED
// cannot say who holds the fact: it stays CHALLENGED and
// contest_check_incomplete says why. A record that matches restores it as
// today (a policy-0 survival). With the marker off every case restores as
// today.
func TestDormantFactContest_ChallengeEndReadsWhoHoldsTheFact(t *testing.T) {
	const claimID = "dfc-hold-challenge"
	type record func(*testing.T, *legacyDormantFixture, *types.Fact) (submitted int64)
	moved := func(cause string, submitted int64) record {
		return func(t *testing.T, f *legacyDormantFixture, fact *types.Fact) int64 {
			t.Helper()
			if cause != "" {
				require.NoError(t, f.k.RecordStatusTransition(f.ctx, &types.StatusTransition{
					FactId: fact.Id, PriorStatus: fact.Status, NewStatus: dfcChallenged,
					BlockHeight: uint64(f.ctx.BlockHeight()), CauseEventType: "challenge_opened", CauseId: cause,
				}))
				fact.Status = dfcChallenged
				require.NoError(t, f.k.SetFactSkipTransition(f.ctx, fact))
				return submitted
			}
			fact.Status = dfcChallenged
			require.NoError(t, f.k.SetFact(f.ctx, fact))
			return submitted
		}
	}
	unrecorded := func(t *testing.T, f *legacyDormantFixture, fact *types.Fact) int64 {
		t.Helper()
		fact.Status = dfcChallenged
		require.NoError(t, f.k.SetFactSkipTransition(f.ctx, fact))
		return 0
	}
	for _, tc := range []struct {
		name     string
		dormant  bool
		record   record
		restored bool
		reported bool
	}{
		{"marker off, another height", false, moved("", -1), true, false},
		{"marker off, another cause", false, moved("dfc-other-challenge", 0), true, false},
		{"marker on, another height", true, moved("", -1), false, false},
		{"marker on, another cause", true, moved("dfc-other-challenge", 0), false, false},
		{"marker on, latest transition not CHALLENGED", true, unrecorded, false, true},
		{"marker on, this cause", true, moved(claimID, 0), true, false},
		{"marker on, this height", true, moved("", 0), true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newLegacyDormantFixture(t, tc.dormant)
			factF := f.acceptedFact(t, "hold")
			before := f.state(t, factF)

			fact, _ := f.k.GetFact(f.ctx, factF)
			submitted := tc.record(t, f, fact)
			claim := &types.Claim{
				Id: claimID, FactContent: "Synthetic challenge of " + factF, Domain: fact.Domain,
				Category: fact.Category, Submitter: makeValidBech32Addr("dfc-hold-ch"), Stake: "11000000",
				Status: types.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION, ProvisionalFactId: factF,
				SubmittedAtBlock: uint64(f.ctx.BlockHeight() + submitted),
			}
			require.NoError(t, f.k.SetClaim(f.ctx, claim))
			round := makeRoundInPhase("r-hold", claim.Id, types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION, 80)
			require.NoError(t, f.k.SetVerificationRound(f.ctx, round))
			require.NoError(t, f.k.CompleteRound(f.ctx, round, &keeper.VerificationResult{Verdict: types.Verdict_VERDICT_REJECT, Confidence: 800_000}))

			after := f.state(t, factF)
			incomplete := v11Event(f.ctx, "zerone.knowledge.contest_check_incomplete")
			if tc.restored {
				params, err := f.k.GetParams(f.ctx)
				require.NoError(t, err)
				require.Equal(t, types.FactStatus_FACT_STATUS_ACTIVE, after.fact.Status)
				require.Equal(t, before.fact.Energy+params.MetabolismEnergyChallengeSurvival, after.fact.Energy, "survival energy, as today")
				require.Empty(t, incomplete)
				return
			}
			require.Equal(t, dfcChallenged, after.fact.Status)
			require.Equal(t, before.fact.Energy, after.fact.Energy, "nothing is credited")
			require.Equal(t, before.fact.CorroborationCount, after.fact.CorroborationCount)
			require.Empty(t, v11Event(f.ctx, "zerone.knowledge.corroboration_incremented"))
			if !tc.reported {
				require.Empty(t, incomplete)
				return
			}
			require.Len(t, incomplete, 1)
			require.Equal(t, factF, incomplete[0]["fact_id"])
			require.Contains(t, incomplete[0]["reason"], "admission record of challenge "+claimID)
		})
	}
}
