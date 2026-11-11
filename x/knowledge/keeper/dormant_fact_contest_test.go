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
// conjecture stays an open question, and a contest that does not refute the
// fact leaves it exactly as it was. These tests run the real handlers,
// review panels and metabolism on real stores, in both marker states. Every
// dormant status here is reached through real metabolism epochs, so the
// fact's at-risk stamp and the domain counters are metabolism's own.

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

// requireRestoredFrom requires that the contest left factID exactly as before
// and that its last recorded transition is the restore from CHALLENGED.
func (f *dormantFixture) requireRestoredFrom(t *testing.T, factID string, before dormantState) {
	t.Helper()
	requireSameState(t, before, f.state(t, factID))
	history := f.transitions(factID)
	require.NotEmpty(t, history)
	last := history[len(history)-1]
	require.Equal(t, dfcChallenged, last.prior)
	require.Equal(t, before.fact.Status, last.next)
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
				// it is admitted and moves to CHALLENGED like any target.
				f.inCache(func() {
					if !dormant {
						f.requireChallengeRefused(t, v11Addr("dfc-adm-ch"), factF, stake, vdNotChallengeable(factF, status, true))
						return
					}
					_, err := f.vdChallengeAt(v11Addr("dfc-adm-ch"), factF, stake)
					require.NoError(t, err, "status %s", status)
					require.Equal(t, dfcChallenged, f.fact(t, factF).Status)
				})

				// MsgSubmitContradiction: AT_RISK already stands; EXPIRED and
				// PRUNED stand only with the marker on. Admission changes no fact.
				f.inCache(func() {
					counter := "Synthetic dormant-fact contest: a counter-observation against a fact in status " + status.String() + "."
					if !dormant && status != dfcAtRisk {
						f.requireContradictionRefused(t, v11Addr("dfc-adm-ct"), factF, counter, stake, contraNotStanding(factF, status))
						return
					}
					f.mustContradict(t, v11Addr("dfc-adm-ct"), factF, counter, stake)
					require.Equal(t, status, f.fact(t, factF).Status, "admitting a contradiction changes no fact")
				})

				// MsgChallengeProvisionalFact: a lapsed conjecture keeps its own
				// door. AT_RISK and EXPIRED were already open; PRUNED was resolved
				// and is open only with the marker on.
				g.decayTo(t, factQ, status)
				g.inCache(func() {
					before, _ := g.storeDigest(t)
					_, err := g.challengeProvisional(v11Addr("dfc-adm-pc"), claimQ, factQ)
					if !dormant && status == dfcPruned {
						require.EqualError(t, err, fmt.Sprintf("fact %s is already resolved (status: %s) — nothing left to refute", factQ, status))
						after, _ := g.storeDigest(t)
						require.Equal(t, before, after, "a refusal writes nothing")
						return
					}
					require.NoError(t, err, "status %s", status)
					require.Equal(t, dfcChallenged, g.fact(t, factQ).Status)
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
// marker off it is refused and nothing changes. With it on, an accepted
// challenge disproves the fact as today; every other ending leaves the fact
// record, the domain counters, the training grade and the training value
// exactly as they were.
func TestDormantFactContest_ChallengeEndings(t *testing.T) {
	for _, status := range dormantStatuses {
		for _, ending := range dormantEndings {
			t.Run(status.String()+"/"+ending.name, func(t *testing.T) {
				for _, dormant := range []bool{false, true} {
					f := newDormantFixture(t, dormant)
					_, factF := f.assertedFact(t, "end")
					f.decayTo(t, factF, status)
					before := f.state(t, factF)
					stake := f.vdMinimum(t, factF).String()
					if !dormant {
						f.requireChallengeRefused(t, v11Addr("dfc-end-ch"), factF, stake, vdNotChallengeable(factF, status, true))
						requireSameState(t, before, f.state(t, factF))
						continue
					}
					c, err := f.vdChallengeAt(v11Addr("dfc-end-ch"), factF, stake)
					require.NoError(t, err)
					require.Equal(t, dfcChallenged, f.fact(t, factF).Status)
					ending.end(t, f, c)
					if !ending.accepted {
						f.requireRestoredFrom(t, factF, before)
						continue
					}
					after := f.state(t, factF)
					require.Equal(t, dfcDisproven, after.fact.Status)
					require.Equal(t, before.fact.Energy, after.fact.Energy)
					require.Equal(t, before.fact.AtRiskSinceEpoch, after.fact.AtRiskSinceEpoch)
					require.Equal(t, before.fact.CorroborationCount, after.fact.CorroborationCount)
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
// endings release only a CONTESTED one. With the marker off, EXPIRED and
// PRUNED are refused at admission.
func TestDormantFactContest_ContradictionEndings(t *testing.T) {
	for _, status := range dormantStatuses {
		for _, ending := range dormantEndings {
			t.Run(status.String()+"/"+ending.name, func(t *testing.T) {
				for _, dormant := range []bool{false, true} {
					f := newDormantFixture(t, dormant)
					_, factF := f.assertedFact(t, "ct")
					f.decayTo(t, factF, status)
					before := f.state(t, factF)
					stake := f.vdMinimum(t, factF).String()
					counter := "Synthetic dormant-fact contest: a counter-observation that a panel decides."
					if !dormant && status != dfcAtRisk {
						f.requireContradictionRefused(t, v11Addr("dfc-ct-c"), factF, counter, stake, contraNotStanding(factF, status))
						requireSameState(t, before, f.state(t, factF))
						continue
					}
					c := f.mustContradict(t, v11Addr("dfc-ct-c"), factF, counter, stake)
					requireSameState(t, before, f.state(t, factF))
					ending.end(t, f, c)
					after := f.state(t, factF)
					require.True(t, proto.Equal(before.fact, after.fact), "the target is never moved\nwant %s\ngot  %s",
						prototext.Format(before.fact), prototext.Format(after.fact))
					require.Equal(t, before.tier, after.tier)
					require.Equal(t, before.tvw, after.tvw)
					require.Equal(t, before.stats.AtRiskCount, after.stats.AtRiskCount)
					if ending.accepted {
						// The counter-fact is a new fact in the same domain.
						require.Equal(t, before.stats.ActiveCount+1, after.stats.ActiveCount)
					} else {
						require.Equal(t, before.stats, after.stats)
					}
				}
			})
		}
	}
}

// A refutation attempt on a lapsed conjecture, for each status and each
// ending. With the marker off a PRUNED conjecture is refused, and an AT_RISK
// or EXPIRED one that survives comes back PROVISIONAL with its at-risk stamp
// cleared, as today. With the marker on it comes back exactly as it was. An
// accepted refutation disproves it in both states.
func TestDormantFactContest_LapsedConjectureEndings(t *testing.T) {
	for _, status := range dormantStatuses {
		for _, ending := range dormantEndings {
			t.Run(status.String()+"/"+ending.name, func(t *testing.T) {
				for _, dormant := range []bool{false, true} {
					f := newDormantFixture(t, dormant)
					claimQ, factQ := f.conjectureFact(t, "lc")
					f.decayTo(t, factQ, status)
					before := f.state(t, factQ)
					c, err := f.challengeProvisional(v11Addr("dfc-lc-pc"), claimQ, factQ)
					if !dormant && status == dfcPruned {
						require.EqualError(t, err, fmt.Sprintf("fact %s is already resolved (status: %s) — nothing left to refute", factQ, status))
						requireSameState(t, before, f.state(t, factQ))
						continue
					}
					require.NoError(t, err)
					ending.end(t, f, c)
					after := f.state(t, factQ)
					switch {
					case ending.accepted:
						require.Equal(t, dfcDisproven, after.fact.Status)
						require.Equal(t, before.fact.Energy, after.fact.Energy)
						require.Equal(t, before.fact.AtRiskSinceEpoch, after.fact.AtRiskSinceEpoch)
					case dormant:
						f.requireRestoredFrom(t, factQ, before)
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
// open question and can be refuted; a refutation resolves it. With the marker
// off it is resolved and unlisted, and the refutation door is shut.
func TestDormantFactContest_PrunedConjectureStaysOpen(t *testing.T) {
	for _, dormant := range []bool{false, true} {
		t.Run(fmt.Sprintf("dormant=%t", dormant), func(t *testing.T) {
			f := newDormantFixture(t, dormant)
			claimQ, factQ := f.conjectureFact(t, "open")
			f.decayTo(t, factQ, dfcPruned)
			query := keeper.NewQueryServerImpl(f.k)

			listed := func() bool {
				response, err := query.OpenQuestions(f.ctx, &types.QueryOpenQuestionsRequest{Domain: dedupDomain})
				require.NoError(t, err)
				for _, q := range response.Questions {
					if q.FactId == factQ {
						return true
					}
				}
				return false
			}
			require.Equal(t, dormant, listed(), "a pruned conjecture is an open question only with the marker on")

			c, err := f.challengeProvisional(v11Addr("dfc-open-pc"), claimQ, factQ)
			if !dormant {
				require.EqualError(t, err, fmt.Sprintf("fact %s is already resolved (status: %s) — nothing left to refute", factQ, dfcPruned))
				return
			}
			require.NoError(t, err)
			require.True(t, listed(), "under challenge it stays listed")
			f.accept(t, c)
			require.Equal(t, dfcDisproven, f.fact(t, factQ).Status)
			require.False(t, listed(), "a refuted conjecture is resolved")
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
// challenge decided through BeginBlocker, stays PRUNED through later epochs
// with its counters untouched, keeps its text free for a twin, and a second
// challenge then disproves it.
func TestDormantFactContest_PrunedFactEndToEnd(t *testing.T) {
	f := newDormantFixture(t, true)
	_, factF := f.assertedFact(t, "e2e")
	text := dormantAssertionText("e2e")
	f.decayTo(t, factF, dfcPruned)
	before := f.state(t, factF)

	twinAdmitted := func() {
		f.inCache(func() {
			_, err := f.submitClaim(v11Addr("dfc-e2e-twin"), text, dedupDomain, nil)
			require.NoError(t, err, "a pruned fact's text is free for a twin")
		})
	}
	twinAdmitted()

	// It survives.
	survive, err := f.vdChallengeAt(v11Addr("dfc-e2e-ch1"), factF, f.vdMinimum(t, factF).String())
	require.NoError(t, err)
	require.Equal(t, types.Verdict_VERDICT_REJECT, f.decideByBlock(t, survive, "reject").Verdict)
	f.requireRestoredFrom(t, factF, before)
	require.Equal(t, []map[string]string(nil), v11Event(f.ctx, "zerone.knowledge.contest_check_incomplete"))
	twinAdmitted()

	// Later epochs leave a restored PRUNED fact alone: it is not metabolised,
	// counted or revived.
	for i := 0; i < 3; i++ {
		f.epoch++
		require.NoError(t, f.k.ProcessMetabolism(f.ctx, f.epoch))
		requireSameState(t, before, f.state(t, factF))
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
	f.epoch++
	require.NoError(t, f.k.ProcessMetabolism(f.ctx, f.epoch))
	require.Equal(t, dfcDisproven, f.fact(t, factF).Status)
}

// An AT_RISK fact that survives a contest ends where an uncontested twin
// ends. Two chains with the same history run the same metabolism epochs; one
// also runs a challenge that a panel rejects or that starves, inside one
// epoch. At every later epoch, until both are PRUNED, the two facts have the
// same status, energy and at-risk stamp, and the domains the same counters.
func TestDormantFactContest_AtRiskTwinEndsWhereUncontested(t *testing.T) {
	for _, ending := range dormantEndings[1:] {
		t.Run(ending.name, func(t *testing.T) {
			contested, quiet := newDormantFixture(t, true), newDormantFixture(t, true)
			_, factA := contested.assertedFact(t, "twin")
			_, factB := quiet.assertedFact(t, "twin")
			require.Equal(t, factA, factB)
			contested.decayTo(t, factA, dfcAtRisk)
			quiet.decayTo(t, factB, dfcAtRisk)
			require.Equal(t, contested.epoch, quiet.epoch)

			c, err := contested.vdChallengeAt(v11Addr("dfc-twin-ch"), factA, contested.vdMinimum(t, factA).String())
			require.NoError(t, err)
			ending.end(t, contested, c)
			quiet.advanceTo(t, uint64(contested.ctx.BlockHeight()))

			sawExpired := false
			for i := 0; i < 100; i++ {
				a, b := contested.state(t, factA), quiet.state(t, factB)
				require.Equal(t, b.fact.Status, a.fact.Status, "epoch %d", contested.epoch)
				require.Equal(t, b.fact.Energy, a.fact.Energy, "epoch %d", contested.epoch)
				require.Equal(t, b.fact.AtRiskSinceEpoch, a.fact.AtRiskSinceEpoch, "epoch %d", contested.epoch)
				require.Equal(t, b.stats, a.stats, "epoch %d", contested.epoch)
				sawExpired = sawExpired || a.fact.Status == dfcExpired
				if a.fact.Status == dfcPruned {
					require.True(t, sawExpired, "it passed through EXPIRED on metabolism's own clock")
					return
				}
				contested.epoch++
				quiet.epoch++
				require.NoError(t, contested.k.ProcessMetabolism(contested.ctx, contested.epoch))
				require.NoError(t, quiet.k.ProcessMetabolism(quiet.ctx, quiet.epoch))
			}
			t.Fatal("the facts never reached PRUNED")
		})
	}
}

// ─── The legacy review policy (review neutrality off) ───────────────────────

// legacyDormantFixture is a chain without review neutrality, record
// integrity or claim integrity: every round is policy 0, a survival credits
// energy and corroboration, and an accepted fact escrows its submitter
// reward. A stub vesting keeper makes that escrow reachable.
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

// With review neutrality off, a survival credits energy and corroboration
// and hands the submitter's escrowed reward to vesting. A dormant target
// gets none of that: it comes back as it was, its pending reward is left to
// the sweep, and the sweep cancels it as it would have without the contest.
// The policy-0 starved path (an expired round never aggregated) restores it
// the same way, and an accepted challenge disproves it. A MALFORMED or
// INCONCLUSIVE aggregated verdict restores no challenged fact on this chain,
// dormant or not, and still does not: completeRound's body is pinned to its
// foundation (TestQ2LegacyBodiesPinnedToFoundation). With the marker off the
// challenge is refused.
func TestDormantFactContest_LegacyReviewPolicy(t *testing.T) {
	type ending struct {
		name string
		end  func(*testing.T, *legacyDormantFixture, *types.VerificationRound)
	}
	complete := func(verdict types.Verdict) func(*testing.T, *legacyDormantFixture, *types.VerificationRound) {
		return func(t *testing.T, f *legacyDormantFixture, round *types.VerificationRound) {
			t.Helper()
			require.NoError(t, f.k.CompleteRound(f.ctx, round, &keeper.VerificationResult{Verdict: verdict, Confidence: 800_000}))
		}
	}
	for _, status := range dormantStatuses {
		for _, e := range []ending{
			{"accepted", complete(types.Verdict_VERDICT_ACCEPT)},
			{"rejected", complete(types.Verdict_VERDICT_REJECT)},
			{"aggregated-inconclusive", complete(types.Verdict_VERDICT_INCONCLUSIVE)},
			{"aggregated-malformed", complete(types.Verdict_VERDICT_MALFORMED)},
			{"expired-unaggregated", func(t *testing.T, f *legacyDormantFixture, round *types.VerificationRound) {
				t.Helper()
				// No block ran between the reveal and aggregation deadlines, so
				// the round reaches the policy-0 EXPIRED arm unaggregated.
				f.ctx = f.ctx.WithBlockHeight(int64(round.AggregationDeadline))
				require.NoError(t, f.k.AdvanceRoundPhases(f.ctx))
				done, _ := f.k.GetVerificationRound(f.ctx, round.Id)
				require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_EXPIRED, done.Phase)
			}},
		} {
			t.Run(status.String()+"/"+e.name, func(t *testing.T) {
				for _, dormant := range []bool{false, true} {
					f := newLegacyDormantFixture(t, dormant)
					factF := f.acceptedFact(t, "legacy")
					pending, found := f.k.GetSurvivalPendingReward(f.ctx, factF)
					require.True(t, found, "the accepted fact escrowed its submitter reward")
					f.decayTo(t, factF, status)
					before := f.state(t, factF)

					round, err := f.challenge(t, factF)
					if !dormant {
						require.EqualError(t, err, vdNotChallengeable(factF, status, false))
						requireSameState(t, before, f.state(t, factF))
						continue
					}
					require.NoError(t, err)
					e.end(t, f, round)
					after := f.state(t, factF)

					switch e.name {
					case "accepted":
						require.Equal(t, dfcDisproven, after.fact.Status)
						require.Equal(t, before.fact.Energy, after.fact.Energy)
						require.Equal(t, before.fact.AtRiskSinceEpoch, after.fact.AtRiskSinceEpoch)
						return
					case "aggregated-inconclusive", "aggregated-malformed":
						require.Equal(t, dfcChallenged, after.fact.Status, "no policy-0 aggregated non-decision restores a challenged fact")
						require.Equal(t, before.fact.Energy, after.fact.Energy)
						require.Equal(t, before.fact.AtRiskSinceEpoch, after.fact.AtRiskSinceEpoch)
						require.Equal(t, before.stats, after.stats)
						return
					}

					requireSameState(t, before, after)
					if e.name == "rejected" {
						events := v11Event(f.ctx, "zerone.knowledge.corroboration_incremented")
						require.Len(t, events, 1)
						require.Equal(t, "false", events[0]["counter_incremented"], "a dormant survival moves no corroboration")
						require.Empty(t, v11Event(f.ctx, "zerone.knowledge.survival_reward_released"))
					}
					// The pending reward waits for the sweep, which pays only a
					// VERIFIED or ACTIVE fact: this one is cancelled, as it would
					// have been without the contest.
					_, stillPending := f.k.GetSurvivalPendingReward(f.ctx, factF)
					require.True(t, stillPending)
					f.ctx = f.ctx.WithBlockHeight(int64(pending.Deadline)).WithEventManager(sdk.NewEventManager())
					f.k.SweepSurvivedRewards(f.ctx)
					_, stillPending = f.k.GetSurvivalPendingReward(f.ctx, factF)
					require.False(t, stillPending)
					require.Len(t, v11Event(f.ctx, "zerone.knowledge.survival_reward_cancelled"), 1)
					require.Empty(t, v11Event(f.ctx, "zerone.knowledge.survival_reward_released"))
				}
			})
		}
	}
}

// The end of a challenge learns what it found from the transition its
// admission recorded. When that record does not match the challenge (here a
// fabricated claim submitted one block before the CHALLENGED write), the
// marker-on chain cannot tell whether the fact was dormant, so it holds the
// fact CHALLENGED, credits nothing and says why. With the marker off the
// fact is restored as today. A record that matches restores exactly.
func TestDormantFactContest_UnmatchedAdmissionRecordHoldsTheFact(t *testing.T) {
	for _, tc := range []struct {
		name      string
		dormant   bool
		submitted int64 // relative to the CHALLENGED write
		want      types.FactStatus
	}{
		{"marker off, unmatched", false, -1, types.FactStatus_FACT_STATUS_ACTIVE},
		{"marker on, unmatched", true, -1, dfcChallenged},
		{"marker on, matched", true, 0, dfcAtRisk},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newLegacyDormantFixture(t, tc.dormant)
			factF := f.acceptedFact(t, "unmatched")
			f.decayTo(t, factF, dfcAtRisk)
			before := f.state(t, factF)

			fact, _ := f.k.GetFact(f.ctx, factF)
			fact.Status = dfcChallenged
			require.NoError(t, f.k.SetFact(f.ctx, fact))
			claim := &types.Claim{
				Id: "dfc-unmatched-challenge", FactContent: "Synthetic challenge of " + factF, Domain: fact.Domain,
				Category: fact.Category, Submitter: makeValidBech32Addr("dfc-unmatched-ch"), Stake: "11000000",
				Status: types.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION, ProvisionalFactId: factF,
				SubmittedAtBlock: uint64(f.ctx.BlockHeight() + tc.submitted),
			}
			require.NoError(t, f.k.SetClaim(f.ctx, claim))
			round := makeRoundInPhase("r-unmatched", claim.Id, types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION, 80)
			require.NoError(t, f.k.SetVerificationRound(f.ctx, round))
			require.NoError(t, f.k.CompleteRound(f.ctx, round, &keeper.VerificationResult{Verdict: types.Verdict_VERDICT_REJECT, Confidence: 800_000}))

			after := f.state(t, factF)
			require.Equal(t, tc.want, after.fact.Status)
			incomplete := v11Event(f.ctx, "zerone.knowledge.contest_check_incomplete")
			switch tc.want {
			case dfcChallenged:
				require.Equal(t, before.fact.Energy, after.fact.Energy, "nothing is credited")
				require.Equal(t, before.fact.AtRiskSinceEpoch, after.fact.AtRiskSinceEpoch)
				require.Equal(t, before.fact.CorroborationCount, after.fact.CorroborationCount)
				require.Len(t, incomplete, 1)
				require.Equal(t, factF, incomplete[0]["fact_id"])
				require.Contains(t, incomplete[0]["reason"], "admission record of challenge dfc-unmatched-challenge")
			case dfcAtRisk:
				requireSameState(t, before, after)
				require.Empty(t, incomplete)
			default:
				params, err := f.k.GetParams(f.ctx)
				require.NoError(t, err)
				require.Zero(t, after.fact.AtRiskSinceEpoch)
				require.Equal(t, before.fact.Energy+params.MetabolismEnergyChallengeSurvival, after.fact.Energy, "marker off: survival energy, as today")
				require.Empty(t, incomplete)
			}
		})
	}
}
