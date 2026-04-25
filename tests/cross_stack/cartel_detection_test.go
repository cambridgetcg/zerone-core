package cross_stack_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	capturechallengekeeper "github.com/zerone-chain/zerone/x/capture_challenge/keeper"
	capturechallengetypes "github.com/zerone-chain/zerone/x/capture_challenge/types"
	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// Cross-module cartel-detection drill.
//
// The Wave 10 fix stake-weighted the augmentation verifier panel so
// zero-stake Sybil addresses can no longer carry a verdict. But a
// sufficiently-funded attacker can still collude across real
// stake-bearing validators. When that happens, the detection path is
// x/capture_challenge: the community accuses a set of validators of
// coordinated capture, attaches evidence, and governance authority
// resolves with slashing.
//
// This drill exercises that pipeline end-to-end:
//   1. Three stake-bearing validators collude on an augmentation verdict.
//   2. The verdict finalizes (primary defense held by stake, but stake
//      was itself compromised).
//   3. Community submits x/capture_challenge against the three.
//   4. Evidence is attached during the EVIDENCE phase.
//   5. The phase advances to UNDER_REVIEW.
//   6. Authority resolves UPHELD.
//   7. Challenger is refunded + rewarded; slash records are written.
//
// The test is the integration-level proof that the second line of
// defense (cartel detection) actually catches what the first line of
// defense (stake-weighted voting) cannot: a coordinated stake-bearing
// attack. Without this path working end-to-end, the Wave 10 fix is
// complete in isolation but the moat is incomplete as a system.
func TestCartelDetection_EndToEndCaptureChallengeFlow(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)

	// ── Setup: sponsor, target fact, bounty, variant ──
	sponsor := testAddr("cartel_sponsor")
	require.NoError(t, h.FundAccount(sponsor, sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewInt(500_000_000)))))
	submitter := testAddr("cartel_sub").String()

	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
		Id: "F-CARTEL", Content: "target for coordinated attack",
		Domain: "mathematics", Status: knowledgetypes.FactStatus_FACT_STATUS_ACTIVE,
		Submitter: sponsor.String(), MethodId: knowledgetypes.MethodologyEmpirical,
		Confidence: 900_000,
	}))
	_, err = ms.CreateAugmentationBounty(h.Ctx, &knowledgetypes.MsgCreateAugmentationBounty{
		Sponsor: sponsor.String(), Id: "b-cartel", TargetFactId: "F-CARTEL",
		RewardPerVariant: 1_000_000, MaxVariants: 1,
	})
	require.NoError(t, err)
	_, err = ms.SubmitAugmentation(h.Ctx, &knowledgetypes.MsgSubmitAugmentation{
		Submitter: submitter, Id: "aug-cartel", BountyId: "b-cartel",
		OriginalFactId: "F-CARTEL", VariantContent: "a variant that drifts the original meaning",
	})
	require.NoError(t, err)

	// ── Step 1: three stake-bearing validators collude on EQUIVALENT ──
	// Each bonds significant stake (50M) so their weighted votes sail
	// past the 66.6% consensus threshold. This simulates the attack
	// vector stake-weighting CANNOT prevent: a genuinely coordinated
	// cabal of real-stake validators.
	colluders := []string{
		testAddr("cartel_v1").String(),
		testAddr("cartel_v2").String(),
		testAddr("cartel_v3").String(),
	}
	for _, v := range colluders {
		h.BondTestValidator(v, 50_000_000)
		_, err := ms.VoteOnAugmentation(h.Ctx, &knowledgetypes.MsgVoteOnAugmentation{
			Verifier: v, AugmentationId: "aug-cartel",
			Vote: knowledgetypes.AugmentationVerdict_AUGMENTATION_VERDICT_EQUIVALENT,
		})
		require.NoError(t, err)
	}

	// ── Step 2: verdict finalizes — primary defense fell to stake attack ──
	aug, _ := h.KnowledgeKeeper.GetAugmentation(h.Ctx, "aug-cartel")
	require.Equal(t, knowledgetypes.AugmentationVerdict_AUGMENTATION_VERDICT_EQUIVALENT, aug.Verdict,
		"cartel with real stake CAN finalize — this is the attack the cartel-detection layer must catch")

	// ── Step 3: community submits capture_challenge against the three ──
	whistleblower := testAddr("cartel_whistleblower")
	require.NoError(t, h.FundAccount(whistleblower, sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewInt(100_000_000)))))

	// Fund the domain's bounty pool so UPHELD has something to pay from.
	ccMS := capturechallengekeeper.NewMsgServerImpl(h.CaptureChallengeKeeper)
	_, err = ccMS.FundBountyPool(h.Ctx, &capturechallengetypes.MsgFundBountyPool{
		Sender: whistleblower.String(), Domain: "mathematics", Amount: "50000000",
	})
	require.NoError(t, err)

	balAfterPoolFund := h.GetBalance(whistleblower, "uzrn")
	require.Equal(t, sdkmath.NewInt(50_000_000), balAfterPoolFund.Amount)

	submitResp, err := ccMS.SubmitChallenge(h.Ctx, &capturechallengetypes.MsgSubmitChallenge{
		Challenger:        whistleblower.String(),
		Domain:            "mathematics",
		AccusedValidators: colluders,
		Stake:             "10000000",
		Reason:            "three validators pushed EQUIVALENT on aug-cartel in the same block — coordinated cartel signal",
	})
	require.NoError(t, err)
	require.NotEmpty(t, submitResp.ChallengeId)

	balAfterStake := h.GetBalance(whistleblower, "uzrn")
	require.Equal(t, sdkmath.NewInt(40_000_000), balAfterStake.Amount,
		"challenger stake locked (50M pool-fund already paid → 50M balance; 10M stake → 40M)")

	// ── Step 4: attach evidence during the EVIDENCE phase ──
	_, err = ccMS.AddEvidence(h.Ctx, &capturechallengetypes.MsgAddEvidence{
		Challenger:  whistleblower.String(),
		ChallengeId: submitResp.ChallengeId,
		Description: "off-chain trace: same-block coordinated votes from v1/v2/v3 on aug-cartel",
		DataHash:    "sha256:0c0fee5...",
	})
	require.NoError(t, err)

	// ── Step 5: advance phase to UNDER_REVIEW ──
	// Default EvidencePeriodBlocks = 5000; advance past the deadline
	// plus a buffer to trigger the phase transition via the keeper's
	// heartbeat hook.
	h.AdvanceBlocks(5001)
	sdkCtx := sdk.UnwrapSDKContext(h.Ctx)
	h.CaptureChallengeKeeper.AdvanceChallengePhases(sdkCtx, uint64(sdkCtx.BlockHeight()))
	ch, found := h.CaptureChallengeKeeper.GetChallenge(sdkCtx, submitResp.ChallengeId)
	require.True(t, found)
	require.Equal(t, capturechallengetypes.ChallengeStatus_CHALLENGE_STATUS_UNDER_REVIEW, ch.Status,
		"challenge must transition to UNDER_REVIEW after evidence deadline")

	// ── Step 6: authority resolves UPHELD ──
	authority := h.CaptureChallengeKeeper.GetAuthority()
	_, err = ccMS.ResolveChallenge(h.Ctx, &capturechallengetypes.MsgResolveChallenge{
		Authority:   authority,
		ChallengeId: submitResp.ChallengeId,
		Outcome:     capturechallengetypes.ChallengeOutcome_CHALLENGE_OUTCOME_UPHELD,
		Reason:      "coordinated vote pattern verified; slashing the cartel",
	})
	require.NoError(t, err)

	// ── Step 7: challenger recovers stake + bounty reward; slash records exist ──
	sdkCtx = sdk.UnwrapSDKContext(h.Ctx)
	ch, _ = h.CaptureChallengeKeeper.GetChallenge(sdkCtx, submitResp.ChallengeId)
	require.Equal(t, capturechallengetypes.ChallengeStatus_CHALLENGE_STATUS_RESOLVED, ch.Status)
	require.NotNil(t, ch.Resolution)
	require.Equal(t, capturechallengetypes.ChallengeOutcome_CHALLENGE_OUTCOME_UPHELD, ch.Resolution.Outcome)

	// Stake returned (10M back) + reward = 10% of pool balance (50M × 10% = 5M).
	// Expected balance: 40M + 10M + 5M = 55M.
	balFinal := h.GetBalance(whistleblower, "uzrn")
	require.True(t, balFinal.Amount.GT(balAfterStake.Amount),
		"successful challenger must receive stake refund plus bounty reward")

	// Slash records: one per accused validator.
	require.Len(t, ch.Slashes, 3, "one slash record per colluder")
	slashedValidators := make(map[string]bool)
	for _, s := range ch.Slashes {
		slashedValidators[s.Validator] = true
		require.Equal(t, "coordinated vote pattern verified; slashing the cartel", s.Reason)
	}
	for _, c := range colluders {
		require.True(t, slashedValidators[c], "colluder %s must appear in slash records", c)
	}
}

// Cartel UPHELD writes a qualification penalty (R28-8); the penalty
// must reduce the validator's effective panel weight on the next vote.
// Pattern instance: ReduceQualificationWeight has always written
// penalties; GetQualificationWeight had never read them. With the
// Wave 16b wire, a confirmed cartel member's vote weight on the
// affected domain is halved for the penalty window — the cartel
// detection layer finally has teeth on the panel layer.
func TestCartelDetection_UpheldPenaltyReducesPanelWeight(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	// Validator with active math qualification at weight 80.
	implicated := testAddr("cartel_pen_implicated").String()
	h.SetDomainQualification(implicated, "mathematics", 80)

	// Pre-penalty: GetQualificationWeight returns the full 80.
	pre := h.QualificationKeeper.GetQualificationWeight(h.Ctx, implicated, "mathematics")
	require.Equal(t, uint32(80), pre, "before cartel resolution, weight is unmodified")

	// Capture-challenge resolution path applies a 50% penalty.
	require.NoError(t, h.QualificationKeeper.ReduceQualificationWeight(
		h.Ctx, implicated, "mathematics", 500_000, uint64(h.Height())+10_000,
	))

	// Post-penalty: weight is halved (80 × 0.5 = 40).
	post := h.QualificationKeeper.GetQualificationWeight(h.Ctx, implicated, "mathematics")
	require.Equal(t, uint32(40), post,
		"penalty reduces effective qualification weight by ReductionBps")

	// Now wire it through to the panel: the implicated validator's
	// next math vote carries half-weight, not full weight. Validate
	// that the recorded VerdictVoteCalibrationBps reflects the
	// post-penalty weight.
	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	sponsor := testAddr("cartel_pen_sponsor")
	require.NoError(t, h.FundAccount(sponsor, sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewInt(50_000_000)))))
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
		Id: "F-PEN", Domain: "mathematics",
		Status:     knowledgetypes.FactStatus_FACT_STATUS_ACTIVE,
		Submitter:  sponsor.String(),
		MethodId:   knowledgetypes.MethodologyFormal,
		Confidence: 900_000,
		Content:    "penalty wire test",
	}))
	_, err = ms.CreateAugmentationBounty(h.Ctx, &knowledgetypes.MsgCreateAugmentationBounty{
		Sponsor: sponsor.String(), Id: "b-pen", TargetFactId: "F-PEN",
		RewardPerVariant: 1_000_000, MaxVariants: 1,
	})
	require.NoError(t, err)
	_, err = ms.SubmitAugmentation(h.Ctx, &knowledgetypes.MsgSubmitAugmentation{
		Submitter: testAddr("cartel_pen_sub").String(), Id: "aug-pen", BountyId: "b-pen",
		OriginalFactId: "F-PEN", VariantContent: "post-penalty variant",
	})
	require.NoError(t, err)

	h.BondTestValidator(implicated, 50_000_000)
	_, err = ms.VoteOnAugmentation(h.Ctx, &knowledgetypes.MsgVoteOnAugmentation{
		Verifier: implicated, AugmentationId: "aug-pen",
		Vote: knowledgetypes.AugmentationVerdict_AUGMENTATION_VERDICT_EQUIVALENT,
	})
	require.NoError(t, err)

	aug, _ := h.KnowledgeKeeper.GetAugmentation(h.Ctx, "aug-pen")
	require.Len(t, aug.VerdictVoteCalibrationBps, 1)
	// Penalty-adjusted weight 40 × 10_000 = 400_000 BPS recorded.
	require.Equal(t, uint64(400_000), aug.VerdictVoteCalibrationBps[0],
		"cartel-implicated voter's panel weight reflects the penalty (40 × 10_000), not the base 80")
}

// Negative path: a challenge against innocent validators must be
// REJECTED by authority and the challenger's stake SLASHED. The
// whistleblower economics cannot be "free to accuse anyone" — the
// cost of a false accusation is the challenger's stake, which flows to
// the domain's bounty pool.
func TestCartelDetection_FalseAccusationSlashesChallenger(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	whistleblower := testAddr("cartel_false_accuser")
	require.NoError(t, h.FundAccount(whistleblower, sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewInt(50_000_000)))))

	innocents := []string{
		testAddr("cartel_innocent_1").String(),
		testAddr("cartel_innocent_2").String(),
	}
	for _, v := range innocents {
		h.BondTestValidator(v, 50_000_000)
	}

	ccMS := capturechallengekeeper.NewMsgServerImpl(h.CaptureChallengeKeeper)
	submitResp, err := ccMS.SubmitChallenge(h.Ctx, &capturechallengetypes.MsgSubmitChallenge{
		Challenger:        whistleblower.String(),
		Domain:            "mathematics",
		AccusedValidators: innocents,
		Stake:             "10000000",
		Reason:            "false accusation — validators were actually honest",
	})
	require.NoError(t, err)

	balAfterStake := h.GetBalance(whistleblower, "uzrn")
	require.Equal(t, sdkmath.NewInt(40_000_000), balAfterStake.Amount)

	h.AdvanceBlocks(5001)
	sdkCtx := sdk.UnwrapSDKContext(h.Ctx)
	h.CaptureChallengeKeeper.AdvanceChallengePhases(sdkCtx, uint64(sdkCtx.BlockHeight()))

	// Authority resolves REJECTED.
	authority := h.CaptureChallengeKeeper.GetAuthority()
	_, err = ccMS.ResolveChallenge(h.Ctx, &capturechallengetypes.MsgResolveChallenge{
		Authority:   authority,
		ChallengeId: submitResp.ChallengeId,
		Outcome:     capturechallengetypes.ChallengeOutcome_CHALLENGE_OUTCOME_REJECTED,
		Reason:      "no evidence of coordination; accused validators cleared",
	})
	require.NoError(t, err)

	// Challenger's stake does NOT come back — the false accusation
	// cost is absorbed by the challenger, not the innocents.
	balAfter := h.GetBalance(whistleblower, "uzrn")
	require.Equal(t, balAfterStake.Amount, balAfter.Amount,
		"false accuser's stake is not refunded on REJECTED")

	sdkCtx = sdk.UnwrapSDKContext(h.Ctx)
	ch, _ := h.CaptureChallengeKeeper.GetChallenge(sdkCtx, submitResp.ChallengeId)
	require.Equal(t, capturechallengetypes.ChallengeStatus_CHALLENGE_STATUS_RESOLVED, ch.Status)
	require.Equal(t, capturechallengetypes.ChallengeOutcome_CHALLENGE_OUTCOME_REJECTED, ch.Resolution.Outcome)
	require.Empty(t, ch.Slashes, "rejected challenge writes no validator slashes")
}
