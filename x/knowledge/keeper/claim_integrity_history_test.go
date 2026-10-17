package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Claim integrity, Task 7 (P6): each challenge-driven status change records
// what caused it and the ID of the challenge claim that caused it (plan §5.2).
// With the marker off the inferred strings of knowledge 11 are kept, as
// TestV11_TransitionCausesInferred also pins.
//
// The scenario uses only challenges, reviewers who are never parties, and a
// fact with no CONTRADICTS edge, so later v12 behaviour (Tasks 8-14) does not
// change what it records.

type challengeCauseScenario struct {
	claimG, factG, claimH, factH  string
	starved, survived, disproving string // challenge claim IDs against G
	counterFact                   string // fact created by accepting `disproving`
	conjClaim, conjFact           string
	provisional                   string // malformed provisional challenge of the conjecture
}

func runChallengeCauseScenario(t *testing.T, f *v11Fixture) challengeCauseScenario {
	t.Helper()
	reviewers := v11Reviewers()
	var s challengeCauseScenario
	s.claimG = f.mustSubmitClaim(t, v11Addr("v12-p6-author"), "Synthetic v12 probe P6: fact G, whose history must name each cause exactly.", "physics", nil)
	s.factG = f.accept(t, s.claimG, reviewers...)
	s.claimH = f.mustSubmitClaim(t, v11Addr("v12-p6-dependent"), "Synthetic v12 probe P6: fact H, which requires fact G to hold.", "physics",
		func(m *types.MsgSubmitClaim) {
			m.Relations = []*types.ClaimRelation{{Relation: types.RelationType_RELATION_TYPE_REQUIRES, TargetFactId: s.factG}}
		})
	s.factH = f.accept(t, s.claimH, reviewers...)

	s.starved = f.challenge(t, v11Addr("v12-p6-challenger-1"), s.factG)
	require.Equal(t, types.FactStatus_FACT_STATUS_CHALLENGED, f.fact(t, s.factG).Status)
	f.starve(t, s.starved)
	require.Equal(t, types.FactStatus_FACT_STATUS_ACTIVE, f.fact(t, s.factG).Status)

	s.survived = f.challenge(t, v11Addr("v12-p6-challenger-2"), s.factG)
	require.Equal(t, types.Verdict_VERDICT_REJECT, f.decide(t, s.survived, "reject", reviewers...).Verdict)
	require.Equal(t, types.FactStatus_FACT_STATUS_ACTIVE, f.fact(t, s.factG).Status)

	s.disproving = f.challenge(t, v11Addr("v12-p6-challenger-3"), s.factG)
	s.counterFact = f.accept(t, s.disproving, reviewers...)
	require.Equal(t, types.FactStatus_FACT_STATUS_DISPROVEN, f.fact(t, s.factG).Status)
	require.Equal(t, types.FactStatus_FACT_STATUS_CONTESTED, f.fact(t, s.factH).Status)

	// MsgChallengeProvisionalFact opens the same kind of round on a conjecture;
	// a MALFORMED verdict restores it like a starved round does.
	conjecture, err := f.server.PostConjecture(f.ctx, &types.MsgPostConjecture{
		Proposer: v11Addr("v12-p6-proposer"), Domain: "physics", Category: "empirical", Stake: v11ReviewFee,
		Statement:              "Synthetic v12 probe P6: conjecture Q, open until a counterexample is shown.",
		FalsificationPredicate: "Exhibit one synthetic counterexample to conjecture Q in this fixture.",
	})
	require.NoError(t, err)
	s.conjClaim = conjecture.ClaimId
	s.conjFact = f.accept(t, s.conjClaim, reviewers...)
	require.Equal(t, types.FactStatus_FACT_STATUS_PROVISIONAL, f.fact(t, s.conjFact).Status)
	params, err := f.k.GetParams(f.ctx)
	require.NoError(t, err)
	opened, err := f.server.ChallengeProvisionalFact(f.ctx, &types.MsgChallengeProvisionalFact{
		Challenger: v11Addr("v12-p6-refuter"), FactId: s.conjFact, ClaimId: s.conjClaim, Stake: params.MinChallengeStake,
		Reason: "Synthetic v12 provisional challenge that the panel finds malformed.",
	})
	require.NoError(t, err)
	round, found := f.k.GetVerificationRound(f.ctx, opened.ChallengeId)
	require.True(t, found)
	s.provisional = round.ClaimId
	require.Equal(t, types.FactStatus_FACT_STATUS_CHALLENGED, f.fact(t, s.conjFact).Status)
	require.Equal(t, types.Verdict_VERDICT_MALFORMED, f.decide(t, s.provisional, "malformed", reviewers...).Verdict)
	require.Equal(t, types.FactStatus_FACT_STATUS_PROVISIONAL, f.fact(t, s.conjFact).Status)

	f.requireNoLiveRounds(t)
	return s
}

const (
	causeUnspecified = types.FactStatus_FACT_STATUS_UNSPECIFIED
	causeProvisional = types.FactStatus_FACT_STATUS_PROVISIONAL
	causeVerified    = types.FactStatus_FACT_STATUS_VERIFIED
	causeContested   = types.FactStatus_FACT_STATUS_CONTESTED
	causeChallenged  = types.FactStatus_FACT_STATUS_CHALLENGED
	causeActive      = types.FactStatus_FACT_STATUS_ACTIVE
	causeDisproven   = types.FactStatus_FACT_STATUS_DISPROVEN
)

// requireDependantsUntouched: the descendant keeps the exact cascade cause and
// the counter-fact its verification, in both selections. A cause attached for
// G must never leak into another fact's history.
func requireDependantsUntouched(t *testing.T, f *v11Fixture, s challengeCauseScenario) {
	t.Helper()
	require.Equal(t, []v11Transition{
		{causeUnspecified, causeVerified, "verification", s.claimH},
		{causeVerified, causeContested, "cascade", s.factG},
	}, f.transitions(s.factH))
	require.Equal(t, []v11Transition{
		{causeUnspecified, causeVerified, "verification", s.disproving},
	}, f.transitions(s.counterFact))
}

// conjectureChallengeTransitions checks the conjecture's creation row and
// returns the rows after it. With the marker on, creation records
// verification/<claim id> (plan §5.2 row 1), as a VERIFIED fact's creation
// already does by inference. With it off, knowledge 11 infers "unknown" with
// no ID for PROVISIONAL, and that is kept byte for byte.
func conjectureChallengeTransitions(t *testing.T, f *v11Fixture, s challengeCauseScenario, created v11Transition) []v11Transition {
	t.Helper()
	all := f.transitions(s.conjFact)
	require.Len(t, all, 3)
	require.Equal(t, created, all[0], "conjecture creation")
	return all[1:]
}

func TestClaimIntegrity_ChallengeTransitionsRecordExactCauses(t *testing.T) {
	f := setupClaimIntegrityTest(t)
	require.True(t, f.claimIntegrityEnabled(t))
	s := runChallengeCauseScenario(t, f)

	require.Equal(t, []v11Transition{
		{causeUnspecified, causeVerified, "verification", s.claimG},
		{causeVerified, causeChallenged, "challenge_opened", s.starved},
		{causeChallenged, causeActive, "challenge_inconclusive", s.starved},
		{causeActive, causeChallenged, "challenge_opened", s.survived},
		{causeChallenged, causeActive, "challenge_survived", s.survived},
		{causeActive, causeChallenged, "challenge_opened", s.disproving},
		{causeChallenged, causeDisproven, "challenge_disproven", s.disproving},
	}, f.transitions(s.factG))
	require.Equal(t, []v11Transition{
		{causeProvisional, causeChallenged, "challenge_opened", s.provisional},
		{causeChallenged, causeProvisional, "challenge_inconclusive", s.provisional},
	}, conjectureChallengeTransitions(t, f, s, v11Transition{causeUnspecified, causeProvisional, "verification", s.conjClaim}))
	requireDependantsUntouched(t, f, s)
}

// The marker-off twin: the same messages and verdicts keep knowledge-11's
// inferred strings, with no cause ID.
func TestClaimIntegrity_ChallengeTransitionsMarkerOffKeepInferredCauses(t *testing.T) {
	f := setupV11SemanticsTest(t)
	require.False(t, f.claimIntegrityEnabled(t))
	s := runChallengeCauseScenario(t, f)

	require.Equal(t, []v11Transition{
		{causeUnspecified, causeVerified, "verification", s.claimG},
		{causeVerified, causeChallenged, "unknown", ""},
		{causeChallenged, causeActive, "unknown", ""},
		{causeActive, causeChallenged, "unknown", ""},
		{causeChallenged, causeActive, "unknown", ""},
		{causeActive, causeChallenged, "unknown", ""},
		{causeChallenged, causeDisproven, "challenge_disproven", ""},
	}, f.transitions(s.factG))
	require.Equal(t, []v11Transition{
		{causeProvisional, causeChallenged, "unknown", ""},
		{causeChallenged, causeProvisional, "unknown", ""},
	}, conjectureChallengeTransitions(t, f, s, v11Transition{causeUnspecified, causeProvisional, "unknown", ""}))
	requireDependantsUntouched(t, f, s)
}

func TestClaimIntegrity_TransitionCausesSurviveExportImport(t *testing.T) {
	source := setupClaimIntegrityTest(t)
	s := runChallengeCauseScenario(t, source)

	exported := source.k.ExportGenesis(source.ctx)
	require.NoError(t, exported.Validate())
	require.True(t, exported.ClaimIntegrityEnabled)
	imported := newClaimIntegrityGenesisFixture(t, "claim-integrity-history-import", exported)

	for _, id := range []string{s.factG, s.factH, s.counterFact, s.conjFact} {
		require.NotEmpty(t, source.transitions(id))
		require.Equal(t, source.transitions(id), imported.transitions(id), "fact %s", id)
	}
	require.Equal(t, "challenge_disproven", imported.transitions(s.factG)[6].cause)
	require.Equal(t, s.disproving, imported.transitions(s.factG)[6].causeID)
	reexported := imported.k.ExportGenesis(imported.ctx)
	require.Len(t, reexported.StatusTransitions, len(exported.StatusTransitions))
	for i := range exported.StatusTransitions {
		require.True(t, proto.Equal(exported.StatusTransitions[i], reexported.StatusTransitions[i]), "transition %d", i)
	}
	require.Len(t, reexported.StatusTransitionCounters, len(exported.StatusTransitionCounters))
	for i := range exported.StatusTransitionCounters {
		require.True(t, proto.Equal(exported.StatusTransitionCounters[i], reexported.StatusTransitionCounters[i]), "counter %d", i)
	}
}
