package cross_stack_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// TestRouteB_Wave6_StepLevelReasoning verifies a JSON reasoning_trace
// parses into structured ReasoningSteps with inference type, predecessor
// refs, and dependency graph. PRM-aligned (Lightman 2023).
func TestRouteB_Wave6_StepLevelReasoning(t *testing.T) {
	h := NewTestHarness(t)
	require.NoError(t, h.KnowledgeKeeper.SeedDefaultMethodologies(h.Ctx))
	require.NoError(t, h.KnowledgeKeeper.SeedDefaultTokenizerSpec(h.Ctx))
	require.NoError(t, h.KnowledgeKeeper.SeedDefaultTraceSchema(h.Ctx))

	qs := knowledgekeeper.NewQueryServerImpl(h.KnowledgeKeeper)

	// Structured reasoning trace — a toy proof.
	reasoning := `[
		{"step":1,"inference":"observation","content":"Given a>0 and b>0."},
		{"step":2,"inference":"definition","content":"Define f(x)=x^2.","depends_on":[1]},
		{"step":3,"inference":"deduction","content":"f is monotone on [0,∞).","depends_on":[2],"supports":["AXIOM-MONOTONE"]},
		{"step":4,"inference":"conclusion","content":"Therefore f(a)<f(b) iff a<b.","depends_on":[3]}
	]`

	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
		Id: "F-STEPS", Content: "monotone of x^2 on positives", Domain: "math",
		Confidence: 900_000, Status: knowledgetypes.FactStatus_FACT_STATUS_ACTIVE,
		Submitter: testAddr("wave6_steps").String(),
		MethodId:  knowledgetypes.MethodologyFormal,
		ReasoningTrace: reasoning,
		CorroborationCount: 2,
	}))

	resp, err := qs.MethodologyApplicationTrace(h.Ctx, &knowledgetypes.QueryMethodologyApplicationTraceRequest{
		FactId: "F-STEPS",
	})
	require.NoError(t, err)
	require.True(t, resp.Found)

	steps := resp.Trace.ReasoningSteps
	require.Len(t, steps, 4, "structured trace must yield 4 ReasoningStep rows")

	require.Equal(t, knowledgetypes.StepInference_STEP_INFERENCE_OBSERVATION, steps[0].StepInference)
	require.Equal(t, knowledgetypes.StepInference_STEP_INFERENCE_DEFINITION, steps[1].StepInference)
	require.Equal(t, knowledgetypes.StepInference_STEP_INFERENCE_DEDUCTION, steps[2].StepInference)
	require.Equal(t, knowledgetypes.StepInference_STEP_INFERENCE_CONCLUSION, steps[3].StepInference)
	require.Equal(t, []uint32{1}, steps[1].DependsOnSteps,
		"step dependencies preserved for PRM training")
	require.Equal(t, []string{"AXIOM-MONOTONE"}, steps[2].PredecessorFactIds,
		"step predecessor fact refs preserved")
	for _, s := range steps {
		require.Equal(t, knowledgetypes.StepVerdict_STEP_VERDICT_UNEXAMINED, s.Verdict,
			"default step verdict is UNEXAMINED; panel may override later")
	}
}

// TestRouteB_Wave6_StepLevelReasoningPlainTextFallback — unstructured
// reasoning still produces steps via paragraph boundaries.
func TestRouteB_Wave6_StepLevelReasoningPlainTextFallback(t *testing.T) {
	h := NewTestHarness(t)
	require.NoError(t, h.KnowledgeKeeper.SeedDefaultMethodologies(h.Ctx))
	require.NoError(t, h.KnowledgeKeeper.SeedDefaultTokenizerSpec(h.Ctx))
	require.NoError(t, h.KnowledgeKeeper.SeedDefaultTraceSchema(h.Ctx))

	qs := knowledgekeeper.NewQueryServerImpl(h.KnowledgeKeeper)

	trace := "First, I observed the temperature rise by 3°C.\n\nSecond, I checked the calibration of the thermometer.\n\nThird, I concluded the rise is real, not instrumental."

	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
		Id: "F-PLAIN", Content: "temperature rose 3°C, real not instrumental", Domain: "sciences",
		Confidence: 800_000, Status: knowledgetypes.FactStatus_FACT_STATUS_ACTIVE,
		Submitter: testAddr("wave6_plain").String(),
		MethodId:  knowledgetypes.MethodologyEmpirical,
		ReasoningTrace: trace,
	}))

	resp, err := qs.MethodologyApplicationTrace(h.Ctx, &knowledgetypes.QueryMethodologyApplicationTraceRequest{
		FactId: "F-PLAIN",
	})
	require.NoError(t, err)
	require.Len(t, resp.Trace.ReasoningSteps, 3,
		"plain text reasoning splits on blank lines — legacy facts aren't starved")
	for _, s := range resp.Trace.ReasoningSteps {
		require.Equal(t, knowledgetypes.StepInference_STEP_INFERENCE_UNSPECIFIED, s.StepInference,
			"plain-text fallback assigns UNSPECIFIED inference per step")
	}
}

// TestRouteB_Wave6_DriftDiagnosisAttachment — DRIFT variants surface a
// DriftDiagnosis record in the unified trace (heuristic for now; panel can
// overwrite via future handler).
func TestRouteB_Wave6_DriftDiagnosisAttachment(t *testing.T) {
	h := NewTestHarness(t)
	require.NoError(t, h.KnowledgeKeeper.SeedDefaultMethodologies(h.Ctx))
	require.NoError(t, h.KnowledgeKeeper.SeedDefaultTokenizerSpec(h.Ctx))
	require.NoError(t, h.KnowledgeKeeper.SeedDefaultTraceSchema(h.Ctx))

	qs := knowledgekeeper.NewQueryServerImpl(h.KnowledgeKeeper)

	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
		Id: "F-DRIFT-PARENT", Content: "all swans observed so far are white", Domain: "sciences",
		Confidence: 900_000, Status: knowledgetypes.FactStatus_FACT_STATUS_ACTIVE,
		Submitter: testAddr("wave6_drift").String(),
		MethodId:  knowledgetypes.MethodologyEmpirical,
	}))

	// DRIFT: variant overgeneralises ("all" → "must be"). Modal shift.
	require.NoError(t, h.KnowledgeKeeper.SetAugmentation(h.Ctx, &knowledgetypes.Augmentation{
		Id: "aug-modal-drift", OriginalFactId: "F-DRIFT-PARENT",
		VariantContent:        "swans must be white",
		VariantReasoningTrace: `[{"step":1,"content":"all observed swans are white"},{"step":2,"inference":"induction","content":"therefore swans are white","depends_on":[1]}]`,
		Submitter: testAddr("wave6_drifter").String(),
		Verdict:   knowledgetypes.AugmentationVerdict_AUGMENTATION_VERDICT_DRIFT,
	}))

	resp, err := qs.MethodologyApplicationTrace(h.Ctx, &knowledgetypes.QueryMethodologyApplicationTraceRequest{
		FactId: "F-DRIFT-PARENT",
	})
	require.NoError(t, err)
	require.Len(t, resp.Trace.DriftExamples, 1)
	drift := resp.Trace.DriftExamples[0]
	require.NotNil(t, drift.Diagnosis, "drift variant carries a diagnosis record")
	require.Len(t, drift.DrifterSteps, 2,
		"drifter's own structured reasoning is surfaced for fine-grained meaning-preservation training")
	require.Equal(t, knowledgetypes.StepInference_STEP_INFERENCE_INDUCTION, drift.DrifterSteps[1].StepInference)
}

// TestRouteB_Wave6_MethodologyChoice — submitter encoded considered /
// abandoned methods as a JSON prefix in reasoning_trace; the trace exposes
// a MethodologyChoice record for selection-training.
func TestRouteB_Wave6_MethodologyChoice(t *testing.T) {
	h := NewTestHarness(t)
	require.NoError(t, h.KnowledgeKeeper.SeedDefaultMethodologies(h.Ctx))
	require.NoError(t, h.KnowledgeKeeper.SeedDefaultTokenizerSpec(h.Ctx))
	require.NoError(t, h.KnowledgeKeeper.SeedDefaultTraceSchema(h.Ctx))

	qs := knowledgekeeper.NewQueryServerImpl(h.KnowledgeKeeper)

	trace := `{"considered":["M-FORMAL","M-EMPIRICAL"],"rationale":"Direct measurement available; formal derivation would require assumptions I cannot verify","abandoned":["M-LEGACY"],"abandon_reason":"legacy method does not record methodology"}
[{"step":1,"inference":"observation","content":"I measured the voltage drop."}]`

	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
		Id: "F-CHOICE", Content: "voltage drop across resistor is 3.2V", Domain: "sciences",
		Confidence: 900_000, Status: knowledgetypes.FactStatus_FACT_STATUS_ACTIVE,
		Submitter: testAddr("wave6_choice").String(),
		MethodId:  knowledgetypes.MethodologyEmpirical,
		ReasoningTrace: trace,
	}))

	resp, err := qs.MethodologyApplicationTrace(h.Ctx, &knowledgetypes.QueryMethodologyApplicationTraceRequest{
		FactId: "F-CHOICE",
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Trace.MethodologyChoice)
	ch := resp.Trace.MethodologyChoice
	require.Equal(t, knowledgetypes.MethodologyEmpirical, ch.ChosenMethodId)
	require.ElementsMatch(t, []string{"M-FORMAL", "M-EMPIRICAL"}, ch.ConsideredMethods)
	require.Contains(t, ch.Rationale, "Direct measurement")
	require.ElementsMatch(t, []string{"M-LEGACY"}, ch.AbandonedMethods)
	require.Contains(t, ch.AbandonmentReason, "does not record methodology")
}

// TestRouteB_Wave6_MethodologyChoiceFallback — absent explicit rationale,
// a minimal MethodologyChoice still names the chosen method so models
// always see the SELECT signal.
func TestRouteB_Wave6_MethodologyChoiceFallback(t *testing.T) {
	h := NewTestHarness(t)
	require.NoError(t, h.KnowledgeKeeper.SeedDefaultMethodologies(h.Ctx))
	require.NoError(t, h.KnowledgeKeeper.SeedDefaultTokenizerSpec(h.Ctx))
	require.NoError(t, h.KnowledgeKeeper.SeedDefaultTraceSchema(h.Ctx))

	qs := knowledgekeeper.NewQueryServerImpl(h.KnowledgeKeeper)

	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
		Id: "F-CHOICE-MIN", Content: "plain fact", Domain: "sciences",
		Status: knowledgetypes.FactStatus_FACT_STATUS_ACTIVE,
		Submitter: testAddr("wave6_min").String(),
		MethodId:  knowledgetypes.MethodologyEmpirical,
	}))

	resp, err := qs.MethodologyApplicationTrace(h.Ctx, &knowledgetypes.QueryMethodologyApplicationTraceRequest{
		FactId: "F-CHOICE-MIN",
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Trace.MethodologyChoice)
	require.Equal(t, knowledgetypes.MethodologyEmpirical, resp.Trace.MethodologyChoice.ChosenMethodId)
	require.Empty(t, resp.Trace.MethodologyChoice.ConsideredMethods,
		"no considered-methods when submitter didn't record them")
}

// TestRouteB_Wave6_BeliefRevisionChain — a fact with corroborations and an
// incoming contradiction yields a monotone-trending belief-revision chain
// ending at current confidence.
func TestRouteB_Wave6_BeliefRevisionChain(t *testing.T) {
	h := NewTestHarness(t)
	require.NoError(t, h.KnowledgeKeeper.SeedDefaultMethodologies(h.Ctx))
	require.NoError(t, h.KnowledgeKeeper.SeedDefaultTokenizerSpec(h.Ctx))
	require.NoError(t, h.KnowledgeKeeper.SeedDefaultTraceSchema(h.Ctx))

	qs := knowledgekeeper.NewQueryServerImpl(h.KnowledgeKeeper)

	// A seasoned fact: submitted, corroborated 4 times, survived 1 contradiction.
	f := &knowledgetypes.Fact{
		Id: "F-REVISIONS", Content: "water boils at 100°C at 1 atm", Domain: "sciences",
		Confidence: 950_000, Status: knowledgetypes.FactStatus_FACT_STATUS_ACTIVE,
		Submitter: testAddr("wave6_rev").String(),
		MethodId:  knowledgetypes.MethodologyEmpirical,
		CorroborationCount: 4,
		SubmittedAtBlock:      10,
		LastCorroboratedBlock: 200,
		LastVerifiedBlock:     210,
	}
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, f))

	// An incoming CONTRADICTS edge from a (failed) counter-claim.
	failedCounter := &knowledgetypes.Fact{
		Id: "F-FAILED-COUNTER", Content: "water boils at 99°C at 1 atm", Domain: "sciences",
		Status: knowledgetypes.FactStatus_FACT_STATUS_DISPROVEN, // the challenger lost
		Submitter: testAddr("wave6_contender").String(),
		MethodId:  knowledgetypes.MethodologyEmpirical,
	}
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, failedCounter))
	require.NoError(t, h.KnowledgeKeeper.SetFactRelation(h.Ctx, &knowledgetypes.FactRelation{
		SourceFactId: failedCounter.Id, TargetFactId: f.Id,
		Relation: knowledgetypes.RelationType_RELATION_TYPE_CONTRADICTS,
		CreatedAtBlock: 150,
	}))

	resp, err := qs.MethodologyApplicationTrace(h.Ctx, &knowledgetypes.QueryMethodologyApplicationTraceRequest{
		FactId: f.Id,
	})
	require.NoError(t, err)
	revs := resp.Trace.BeliefRevisions
	require.GreaterOrEqual(t, len(revs), 6,
		"at least: 1 initial + 4 corroborations + 1 contradiction + 1 reconcile")

	// First row is always the initial prior.
	require.Equal(t, knowledgetypes.RevisionReason_REVISION_REASON_RESUBMISSION, revs[0].Reason)
	require.Equal(t, uint64(0), revs[0].PriorConfidenceBps)

	// Corroboration rows appear.
	var sawCorro, sawContra bool
	for _, r := range revs {
		switch r.Reason {
		case knowledgetypes.RevisionReason_REVISION_REASON_CORROBORATION:
			sawCorro = true
		case knowledgetypes.RevisionReason_REVISION_REASON_CONTRADICTION:
			sawContra = true
			require.Contains(t, r.EvidenceFactIds, failedCounter.Id,
				"contradiction revision cites the challenger fact as evidence")
		}
	}
	require.True(t, sawCorro, "CORROBORATION reason appears in the chain")
	require.True(t, sawContra, "CONTRADICTION reason appears in the chain")

	// Final row equals the fact's current confidence.
	last := revs[len(revs)-1]
	require.Equal(t, f.Confidence, last.PosteriorConfidenceBps,
		"chain closes by reconciling to current chain state")
}

// TestRouteB_Wave6_DialecticTreeFromChallenges — flat challenges become a
// DialecticNode tree with speakers, roles, and a verdict leaf.
func TestRouteB_Wave6_DialecticTreeFromChallenges(t *testing.T) {
	h := NewTestHarness(t)
	require.NoError(t, h.KnowledgeKeeper.SeedDefaultMethodologies(h.Ctx))
	require.NoError(t, h.KnowledgeKeeper.SeedDefaultTokenizerSpec(h.Ctx))
	require.NoError(t, h.KnowledgeKeeper.SeedDefaultTraceSchema(h.Ctx))

	qs := knowledgekeeper.NewQueryServerImpl(h.KnowledgeKeeper)

	// Seed a domain for the challenge claim to hang on.
	require.NoError(t, h.KnowledgeKeeper.SetDomain(h.Ctx, &knowledgetypes.Domain{
		Name: "sciences", Status: knowledgetypes.DomainStatus_DOMAIN_STATUS_ACTIVE,
	}))

	defenderAddr := testAddr("wave6_defender").String()
	challengerAddr := testAddr("wave6_challenger").String()

	f := &knowledgetypes.Fact{
		Id: "F-DEBATE", Content: "P", Domain: "sciences",
		Confidence: 900_000, Status: knowledgetypes.FactStatus_FACT_STATUS_ACTIVE,
		Submitter: defenderAddr, MethodId: knowledgetypes.MethodologyEmpirical,
	}
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, f))

	// Challenge claim with a verdict and a rebuttal text.
	round := &knowledgetypes.VerificationRound{
		Id: "r-debate", Phase: knowledgetypes.VerificationPhase_VERIFICATION_PHASE_COMPLETE,
		Verdict: knowledgetypes.Verdict_VERDICT_REJECT, // challenge rejected → fact survived
		VerdictBlock: 500,
	}
	require.NoError(t, h.KnowledgeKeeper.SetVerificationRound(h.Ctx, round))

	ch := &knowledgetypes.Claim{
		Id: "c-debate-1", Submitter: challengerAddr,
		FactContent: "not P", Domain: "sciences", Category: "empirical",
		MethodId: knowledgetypes.MethodologyEmpirical,
		ProvisionalFactId: f.Id,
		VerificationRoundId: round.Id,
		ArgumentText: "I challenge P because of evidence E.",
		RebuttalText: "E is spurious because of counter-evidence F.",
	}
	require.NoError(t, h.KnowledgeKeeper.SetClaim(h.Ctx, ch))

	resp, err := qs.MethodologyApplicationTrace(h.Ctx, &knowledgetypes.QueryMethodologyApplicationTraceRequest{
		FactId: f.Id,
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Trace.DialecticTree, "dialectic tree must be populated from challenges")

	root := resp.Trace.DialecticTree[0]
	require.Equal(t, knowledgetypes.DialecticRole_DIALECTIC_ROLE_CHALLENGE, root.Role)
	require.Equal(t, challengerAddr, root.Speaker)
	require.Contains(t, root.ArgumentText, "I challenge P")

	require.GreaterOrEqual(t, len(root.Children), 2,
		"tree must include rebuttal and verdict leaves")

	var sawRebuttal, sawVerdict bool
	for _, ch := range root.Children {
		switch ch.Role {
		case knowledgetypes.DialecticRole_DIALECTIC_ROLE_REBUTTAL:
			sawRebuttal = true
			require.Equal(t, defenderAddr, ch.Speaker)
		case knowledgetypes.DialecticRole_DIALECTIC_ROLE_VERDICT:
			sawVerdict = true
			require.Equal(t, knowledgetypes.StepVerdict_STEP_VERDICT_SOUND, ch.NodeVerdict,
				"challenge rejected → fact survived → SOUND verdict at the leaf")
		}
	}
	require.True(t, sawRebuttal)
	require.True(t, sawVerdict)
}
