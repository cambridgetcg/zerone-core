package cross_stack_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	qualificationtypes "github.com/zerone-chain/zerone/x/qualification/types"
)

// Per-domain panel voting (Wave 15c). The augmentation verifier panel
// should be weighted by DOMAIN-SPECIFIC qualification, not global
// calibration. A physics-domain fact should be adjudicated by those
// who have proven competence in physics — cross-domain expertise
// earns no credit in domain-specific panels.
//
// This drill demonstrates the shift: a validator with high global
// calibration but NO qualification in the target domain carries only
// the floor weight, while a validator with moderate stake but strong
// in-domain qualification dominates.

func TestDomainPanel_DomainQualifiedVotersDominateGloballyCalibrated(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	sponsor := testAddr("domain_panel_sponsor")
	require.NoError(t, h.FundAccount(sponsor, sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewInt(100_000_000)))))
	submitter := testAddr("domain_panel_sub").String()

	// Seed target fact in "mathematics" domain.
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
		Id:         "F-MATH-PANEL",
		Content:    "mathematical claim under audit",
		Domain:     "mathematics",
		Status:     knowledgetypes.FactStatus_FACT_STATUS_ACTIVE,
		Submitter:  sponsor.String(),
		MethodId:   knowledgetypes.MethodologyFormal,
		Confidence: 900_000,
	}))
	_, err = ms.CreateAugmentationBounty(h.Ctx, &knowledgetypes.MsgCreateAugmentationBounty{
		Sponsor: sponsor.String(), Id: "b-math-panel", TargetFactId: "F-MATH-PANEL",
		RewardPerVariant: 1_000_000, MaxVariants: 1,
	})
	require.NoError(t, err)
	_, err = ms.SubmitAugmentation(h.Ctx, &knowledgetypes.MsgSubmitAugmentation{
		Submitter: submitter, Id: "aug-math-panel", BountyId: "b-math-panel",
		OriginalFactId: "F-MATH-PANEL", VariantContent: "paraphrased mathematics variant",
	})
	require.NoError(t, err)

	// "Polymath" — large stake, high GLOBAL calibration, but qualified
	// in biology (not mathematics). In domain-specific panel voting,
	// their cross-domain credentials don't count — they fall back to
	// the floor because they have no qualification in mathematics.
	polymath := testAddr("domain_panel_polymath").String()
	h.BondTestValidator(polymath, 200_000_000)
	require.NoError(t, h.KnowledgeKeeper.SetAgentCalibration(h.Ctx, &knowledgetypes.AgentCalibration{
		Address: polymath, CalibrationScoreBps: 1_000_000,
		Accepted: 100, TotalSubmissions: 100,
	}))
	h.SetDomainQualification(polymath, "biology", 100) // wrong domain

	// Two "mathematicians" — moderate stake, moderate global calibration,
	// strong qualification in mathematics. They dominate the math panel.
	mathVoters := []string{
		testAddr("domain_panel_math1").String(),
		testAddr("domain_panel_math2").String(),
	}
	for _, v := range mathVoters {
		h.BondTestValidator(v, 50_000_000)
		require.NoError(t, h.KnowledgeKeeper.SetAgentCalibration(h.Ctx, &knowledgetypes.AgentCalibration{
			Address: v, CalibrationScoreBps: 500_000, // moderate global
			Accepted: 50, TotalSubmissions: 100,
		}))
		h.SetDomainQualification(v, "mathematics", 90) // strong in-domain
	}

	// Polymath votes DRIFT; mathematicians vote EQUIVALENT.
	_, err = ms.VoteOnAugmentation(h.Ctx, &knowledgetypes.MsgVoteOnAugmentation{
		Verifier: polymath, AugmentationId: "aug-math-panel",
		Vote: knowledgetypes.AugmentationVerdict_AUGMENTATION_VERDICT_DRIFT,
	})
	require.NoError(t, err)
	for _, v := range mathVoters {
		_, err := ms.VoteOnAugmentation(h.Ctx, &knowledgetypes.MsgVoteOnAugmentation{
			Verifier: v, AugmentationId: "aug-math-panel",
			Vote: knowledgetypes.AugmentationVerdict_AUGMENTATION_VERDICT_EQUIVALENT,
		})
		require.NoError(t, err)
	}

	// With per-domain weighting:
	//   Polymath: stake 200M × floor 0.2 = 40M (no math qualification)
	//   Math1:    stake 50M × qualification 0.9 = 45M
	//   Math2:    stake 50M × qualification 0.9 = 45M
	//   Total: 130M; EQUIVALENT share: 90/130 = 69.2% → clears 66.6%.
	// Without per-domain weighting (raw stake): polymath at 200M would
	// dominate the 100M total from mathematicians.
	aug, _ := h.KnowledgeKeeper.GetAugmentation(h.Ctx, "aug-math-panel")
	require.Equal(t, knowledgetypes.AugmentationVerdict_AUGMENTATION_VERDICT_EQUIVALENT, aug.Verdict,
		"domain-qualified mathematicians dominate the math panel despite smaller aggregate stake")

	// Calibration snapshots on the record reflect the DOMAIN-SPECIFIC
	// weights. Polymath has NO math qualification, so their recorded
	// weight is 0 — the tally floors them at 20% at consensus time.
	// Global calibration is explicitly NOT consulted when the target
	// has a domain: domain specialization is mandatory in this path.
	require.Len(t, aug.VerdictVoteCalibrationBps, 3)
	require.Equal(t, uint64(0), aug.VerdictVoteCalibrationBps[0],
		"polymath unqualified in math → 0 recorded; floored to 20% at tally (not global calibration)")
	require.Equal(t, uint64(900_000), aug.VerdictVoteCalibrationBps[1],
		"math1 recorded domain qualification (90 × 10_000)")
	require.Equal(t, uint64(900_000), aug.VerdictVoteCalibrationBps[2])
}

// Feedback loop: augmentation verdicts must update x/qualification
// metrics. Without this, the panel is set-and-forget — a voter who
// got qualified once never has their record updated by how they vote.
// With it, the per-domain panel has a training signal: consistent
// correct voters grow in weight; persistent dissenters against
// consensus see their accuracy erode.
func TestDomainPanel_VerdictFeedbackLoopUpdatesQualification(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	sponsor := testAddr("domain_fbk_sponsor")
	require.NoError(t, h.FundAccount(sponsor, sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewInt(100_000_000)))))
	submitter := testAddr("domain_fbk_sub").String()

	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
		Id: "F-FBK", Domain: "mathematics",
		Status:     knowledgetypes.FactStatus_FACT_STATUS_ACTIVE,
		Submitter:  sponsor.String(),
		MethodId:   knowledgetypes.MethodologyFormal,
		Confidence: 900_000,
		Content:    "feedback target",
	}))
	_, err = ms.CreateAugmentationBounty(h.Ctx, &knowledgetypes.MsgCreateAugmentationBounty{
		Sponsor: sponsor.String(), Id: "b-fbk", TargetFactId: "F-FBK",
		RewardPerVariant: 1_000_000, MaxVariants: 1,
	})
	require.NoError(t, err)
	_, err = ms.SubmitAugmentation(h.Ctx, &knowledgetypes.MsgSubmitAugmentation{
		Submitter: submitter, Id: "aug-fbk", BountyId: "b-fbk",
		OriginalFactId: "F-FBK", VariantContent: "variant for feedback test",
	})
	require.NoError(t, err)

	// One dissenter + two majority = 3 voters (MinPanelVotes default).
	// Dissenter votes first (so they don't see "already final"); the
	// consensus check fires on the 3rd vote (2nd majority vote), where
	// 32M EQUIVALENT / 48M total = 66.7% just clears the 66.6% bar.
	majority := []string{
		testAddr("domain_fbk_ok1").String(),
		testAddr("domain_fbk_ok2").String(),
	}
	dissenter := testAddr("domain_fbk_no").String()

	for _, v := range majority {
		h.BondTestValidator(v, 20_000_000)
		h.SetDomainQualification(v, "mathematics", 80)
	}
	h.BondTestValidator(dissenter, 20_000_000)
	h.SetDomainQualification(dissenter, "mathematics", 80)

	// Dissenter votes FIRST (DRIFT), then majority votes EQUIVALENT.
	// Otherwise the 3 majority votes hit consensus and finalize before
	// the dissenter gets a chance to weigh in.
	_, err = ms.VoteOnAugmentation(h.Ctx, &knowledgetypes.MsgVoteOnAugmentation{
		Verifier: dissenter, AugmentationId: "aug-fbk",
		Vote: knowledgetypes.AugmentationVerdict_AUGMENTATION_VERDICT_DRIFT,
	})
	require.NoError(t, err)
	for _, v := range majority {
		_, err := ms.VoteOnAugmentation(h.Ctx, &knowledgetypes.MsgVoteOnAugmentation{
			Verifier: v, AugmentationId: "aug-fbk",
			Vote: knowledgetypes.AugmentationVerdict_AUGMENTATION_VERDICT_EQUIVALENT,
		})
		require.NoError(t, err)
	}

	// Majority voters should show TotalVerifications=1, CorrectVerifications=1.
	// Dissenter should show TotalVerifications=1, CorrectVerifications=0.
	for _, v := range majority {
		q, found := h.QualificationKeeper.GetQualification(h.Ctx, v, "mathematics")
		require.True(t, found, "majority voter %s should still have qualification", v)
		require.NotNil(t, q.Metrics)
		require.Equal(t, uint64(1), q.Metrics.TotalVerifications,
			"majority voter %s should have 1 recorded verification", v)
		require.Equal(t, uint64(1), q.Metrics.CorrectVerifications,
			"majority voter %s should have 1 correct verification", v)
	}
	dq, found := h.QualificationKeeper.GetQualification(h.Ctx, dissenter, "mathematics")
	require.True(t, found)
	require.NotNil(t, dq.Metrics)
	require.Equal(t, uint64(1), dq.Metrics.TotalVerifications,
		"dissenter still participated and should be recorded")
	require.Equal(t, uint64(0), dq.Metrics.CorrectVerifications,
		"dissenter voted against consensus — 0 correct")
}

// Wave 16 qualification decay (Phase B). The feedback loop now
// closes: a voter whose AccuracyBps drops below the probation
// threshold is demoted ACTIVE → PROBATIONARY. If accuracy keeps
// falling, PROBATIONARY → SUSPENDED. If accuracy recovers above the
// recovery threshold, PROBATIONARY → ACTIVE. Skill is current, not
// historical; the chain runs an ongoing competency assessment, not
// a one-time qualification exam.
func TestDomainPanel_QualificationDecaysOnLowAccuracy(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	// Shrink the decay-check interval so the test doesn't have to
	// advance 10K blocks to trigger the BeginBlocker scan.
	qParams := h.QualificationKeeper.GetParams(h.Ctx)
	qParams.DecayCheckIntervalBlocks = 5
	qParams.DecayMinSamples = 10
	h.QualificationKeeper.SetParams(h.Ctx, qParams)

	// Seed an ACTIVE math qualification with low accuracy: 30 verifications,
	// 12 correct → 40% accuracy. Below the 60% probation threshold.
	bad := testAddr("decay_low_accuracy").String()
	h.SetDomainQualification(bad, "mathematics", 80)
	q, _ := h.QualificationKeeper.GetQualification(h.Ctx, bad, "mathematics")
	q.Metrics = &qualificationtypes.QualificationMetrics{
		TotalVerifications:   30,
		CorrectVerifications: 12,
		AccuracyBps:          400_000,
	}
	h.QualificationKeeper.SetQualification(h.Ctx, q)

	// Advance past the next decay check.
	h.AdvanceBlocks(10)

	q, _ = h.QualificationKeeper.GetQualification(h.Ctx, bad, "mathematics")
	require.Equal(t, qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_PROBATIONARY, q.Status,
		"low-accuracy ACTIVE must decay to PROBATIONARY")
	require.Greater(t, q.ProbationUntil, uint64(0), "probation deadline must be set")
}

// Recovery: a voter whose accuracy improves while probationary climbs
// back to ACTIVE. The feedback loop is bidirectional.
func TestDomainPanel_QualificationRecoversFromProbation(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	qParams := h.QualificationKeeper.GetParams(h.Ctx)
	qParams.DecayCheckIntervalBlocks = 5
	qParams.DecayMinSamples = 10
	h.QualificationKeeper.SetParams(h.Ctx, qParams)

	// Seed a PROBATIONARY qualification with HIGH accuracy: 50 verifications,
	// 40 correct → 80% accuracy. Above the 75% recovery threshold.
	improver := testAddr("decay_recovers").String()
	h.SetDomainQualification(improver, "mathematics", 80)
	q, _ := h.QualificationKeeper.GetQualification(h.Ctx, improver, "mathematics")
	q.Status = qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_PROBATIONARY
	q.ProbationUntil = uint64(h.Height()) + 1_000_000 // far in the future, not auto-promoted
	q.Metrics = &qualificationtypes.QualificationMetrics{
		TotalVerifications:   50,
		CorrectVerifications: 40,
		AccuracyBps:          800_000,
	}
	h.QualificationKeeper.SetQualification(h.Ctx, q)

	h.AdvanceBlocks(10)

	q, _ = h.QualificationKeeper.GetQualification(h.Ctx, improver, "mathematics")
	require.Equal(t, qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_ACTIVE, q.Status,
		"high-accuracy PROBATIONARY must recover to ACTIVE")
	require.Equal(t, uint64(0), q.ProbationUntil, "probation deadline cleared on recovery")
}

// Suspension: PROBATIONARY voter whose accuracy falls further below
// the suspension threshold loses status entirely. They must re-qualify
// to vote effectively again.
func TestDomainPanel_QualificationSuspendsOnContinuedFailure(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	qParams := h.QualificationKeeper.GetParams(h.Ctx)
	qParams.DecayCheckIntervalBlocks = 5
	qParams.DecayMinSamples = 10
	h.QualificationKeeper.SetParams(h.Ctx, qParams)

	// PROBATIONARY voter with very low accuracy: 50 verifications, 15 correct
	// → 30% accuracy. Below the 40% suspension threshold.
	terrible := testAddr("decay_suspends").String()
	h.SetDomainQualification(terrible, "mathematics", 80)
	q, _ := h.QualificationKeeper.GetQualification(h.Ctx, terrible, "mathematics")
	q.Status = qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_PROBATIONARY
	q.ProbationUntil = uint64(h.Height()) + 1_000_000
	q.Metrics = &qualificationtypes.QualificationMetrics{
		TotalVerifications:   50,
		CorrectVerifications: 15,
		AccuracyBps:          300_000,
	}
	h.QualificationKeeper.SetQualification(h.Ctx, q)

	h.AdvanceBlocks(10)

	q, _ = h.QualificationKeeper.GetQualification(h.Ctx, terrible, "mathematics")
	require.Equal(t, qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_SUSPENDED, q.Status,
		"sustained low accuracy in PROBATIONARY must suspend")
}

// Sample-size guard: a qualification with very few verifications is
// NOT decayed even if accuracy looks bad. Early-life noise must not
// drive transitions; the loop only fires once enough signal exists.
func TestDomainPanel_QualificationDoesNotDecayBelowMinSamples(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	qParams := h.QualificationKeeper.GetParams(h.Ctx)
	qParams.DecayCheckIntervalBlocks = 5
	qParams.DecayMinSamples = 20
	h.QualificationKeeper.SetParams(h.Ctx, qParams)

	// Only 5 verifications, 0 correct — looks terrible, but below the
	// minimum sample threshold. Status stays ACTIVE.
	rookie := testAddr("decay_rookie").String()
	h.SetDomainQualification(rookie, "mathematics", 80)
	q, _ := h.QualificationKeeper.GetQualification(h.Ctx, rookie, "mathematics")
	q.Metrics = &qualificationtypes.QualificationMetrics{
		TotalVerifications:   5,
		CorrectVerifications: 0,
		AccuracyBps:          0,
	}
	h.QualificationKeeper.SetQualification(h.Ctx, q)

	h.AdvanceBlocks(10)

	q, _ = h.QualificationKeeper.GetQualification(h.Ctx, rookie, "mathematics")
	require.Equal(t, qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_ACTIVE, q.Status,
		"insufficient samples → no decay; early-life noise must not drive transitions")
}

// Negative path: a validator qualified in the RIGHT domain but with
// very low qualification weight (e.g., on probation or new) still gets
// recorded at the qualification level, not falsely boosted by global
// calibration. Domain qualification is the PRIMARY signal for panel
// weight; global calibration is a fallback only.
func TestDomainPanel_InDomainLowWeightNotInflatedByGlobalCalibration(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	sponsor := testAddr("domain_panel_sponsor_2")
	require.NoError(t, h.FundAccount(sponsor, sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewInt(100_000_000)))))
	submitter := testAddr("domain_panel_sub_2").String()

	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
		Id:         "F-MATH-PROBATION",
		Domain:     "mathematics",
		Status:     knowledgetypes.FactStatus_FACT_STATUS_ACTIVE,
		Submitter:  sponsor.String(),
		MethodId:   knowledgetypes.MethodologyFormal,
		Confidence: 900_000,
		Content:    "probation voter test",
	}))
	_, err = ms.CreateAugmentationBounty(h.Ctx, &knowledgetypes.MsgCreateAugmentationBounty{
		Sponsor: sponsor.String(), Id: "b-math-prob", TargetFactId: "F-MATH-PROBATION",
		RewardPerVariant: 1_000_000, MaxVariants: 1,
	})
	require.NoError(t, err)
	_, err = ms.SubmitAugmentation(h.Ctx, &knowledgetypes.MsgSubmitAugmentation{
		Submitter: submitter, Id: "aug-prob", BountyId: "b-math-prob",
		OriginalFactId: "F-MATH-PROBATION", VariantContent: "variant",
	})
	require.NoError(t, err)

	// Probation voter: high global calibration (perhaps from other domains)
	// but only weight=20 in mathematics (barely qualified).
	probation := testAddr("domain_panel_probation").String()
	h.BondTestValidator(probation, 50_000_000)
	require.NoError(t, h.KnowledgeKeeper.SetAgentCalibration(h.Ctx, &knowledgetypes.AgentCalibration{
		Address: probation, CalibrationScoreBps: 900_000,
		Accepted: 90, TotalSubmissions: 100,
	}))
	h.SetDomainQualification(probation, "mathematics", 20)

	_, err = ms.VoteOnAugmentation(h.Ctx, &knowledgetypes.MsgVoteOnAugmentation{
		Verifier: probation, AugmentationId: "aug-prob",
		Vote: knowledgetypes.AugmentationVerdict_AUGMENTATION_VERDICT_EQUIVALENT,
	})
	require.NoError(t, err)

	aug, _ := h.KnowledgeKeeper.GetAugmentation(h.Ctx, "aug-prob")
	require.Len(t, aug.VerdictVoteCalibrationBps, 1)
	require.Equal(t, uint64(200_000), aug.VerdictVoteCalibrationBps[0],
		"probation voter (weight 20 → 200_000 BPS) recorded at domain level, NOT global 900_000")
}
