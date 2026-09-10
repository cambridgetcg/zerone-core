package keeper

import (
	"bytes"
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/zerone-chain/zerone/x/knowledge/types"
	"google.golang.org/protobuf/proto"
)

// Nil embedded dependencies panic if the neutral policy consults retired
// stake, role, qualification or agreement-feedback machinery.
type forbiddenReviewStake struct{ types.StakingKeeper }
type forbiddenReviewRole struct{ types.ZeroneAuthKeeper }
type forbiddenReviewQualification struct {
	types.DomainQualificationKeeper
}
type forbiddenReviewCapture struct{ types.CaptureDefenseKeeper }

func neutralReviewFixture(t *testing.T, votes []string) (handoffFixture, *types.Claim, *types.VerificationRound) {
	t.Helper()
	f := setupHandoff(t)
	f.ctx = f.ctx.WithChainID("neutral-review-test")
	require.NoError(t, f.k.EnableRecordIntegrity(f.ctx))
	require.NoError(t, f.k.EnableReviewNeutrality(f.ctx))
	params := types.DefaultParams()
	params.MinVerifiers, params.MinHeadcountAgreement = 3, 2
	params.ConfidenceThreshold, params.MaxConfidence = 600000, 950000
	params.IndependenceRewardStrengthBps = 990000
	params.VerificationReward = "0" // No unrelated nominal-reward eligibility gate.
	require.NoError(t, f.k.SetParams(f.ctx, &params))
	claim := &types.Claim{Id: "neutral-claim", Submitter: sdk.AccAddress(bytes.Repeat([]byte{90}, 20)).String(), Stake: "101", Domain: "physics", FactContent: "A scoped fixture observation.", Category: "empirical", ClaimType: types.ClaimType_CLAIM_TYPE_ASSERTION, ReviewPolicyVersion: types.ReviewPolicyNeutral, PartnershipId: "legacy-role-input"}
	require.NoError(t, f.k.SetClaim(f.ctx, claim))
	round := &types.VerificationRound{Id: "neutral-round", ClaimId: claim.Id, StartedAtBlock: 80, CommitDeadline: 85, RevealDeadline: 90, AggregationDeadline: 95, Phase: types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION, CommitmentScheme: types.CommitmentSchemeReviewV2, CommitmentChainId: f.ctx.ChainID(), ReviewPolicyVersion: types.ReviewPolicyNeutral}
	for i, vote := range votes {
		v := sdk.AccAddress(bytes.Repeat([]byte{byte(i + 1)}, 20)).String()
		att := &types.ReviewAttestation{Reason: fmt.Sprintf("Declared check %d", i)}
		salt := []byte(fmt.Sprintf("salt-with-sixteen-bytes-%d", i))
		committedVote := vote
		if vote == "" {
			committedVote = "accept"
		}
		hash, err := types.ComputeReviewCommitmentV2(round.CommitmentChainId, round.Id, v, committedVote, 800000, salt, att)
		require.NoError(t, err)
		round.Commits = append(round.Commits, &types.CommitEntry{Verifier: v, CommitHash: hash, CommittedAtBlock: 82})
		round.SelectedVerifiers = append(round.SelectedVerifiers, v)
		if vote != "" {
			round.Reveals = append(round.Reveals, &types.RevealEntry{Verifier: v, Vote: vote, Confidence: 800000, Salt: salt, Attestation: att, RevealedAtBlock: 87})
		}
	}
	require.NoError(t, f.k.SetVerificationRound(f.ctx, round))
	f.k.stakingKeeper = forbiddenReviewStake{}
	f.k.zeroneAuthKeeper = forbiddenReviewRole{}
	f.k.domainQualificationKeeper = forbiddenReviewQualification{}
	f.k.captureDefenseKeeper = forbiddenReviewCapture{}
	return f, claim, round
}

func TestNeutralReviewCountsDeclaredAccountsAndRewardsEveryReveal(t *testing.T) {
	for _, tc := range []struct {
		name       string
		votes      []string
		verdict    types.Verdict
		confidence uint64
	}{
		{"accept-with-dissent", []string{"accept", "accept", "reject", ""}, types.Verdict_VERDICT_ACCEPT, 666666},
		{"reject-with-dissent", []string{"reject", "reject", "accept"}, types.Verdict_VERDICT_REJECT, 666666},
		{"malformed-with-dissent", []string{"malformed", "malformed", "reject"}, types.Verdict_VERDICT_MALFORMED, 666666},
		{"inconclusive", []string{"accept", "reject", "malformed"}, types.Verdict_VERDICT_INCONCLUSIVE, 0},
		{"below-quorum", []string{"accept", "reject", ""}, types.Verdict_VERDICT_INCONCLUSIVE, 0},
		{"no-reviews", []string{""}, types.Verdict_VERDICT_INCONCLUSIVE, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, _, round := neutralReviewFixture(t, tc.votes)
			// Old capture thresholds cannot reweight this declared panel.
			require.NoError(t, f.k.IncreaseVerificationThreshold(f.ctx, "physics", 50, 1000))
			result, err := f.k.AggregateVerificationResult(f.ctx, round)
			require.NoError(t, err)
			require.Equal(t, tc.verdict, result.Verdict)
			require.Equal(t, tc.confidence, result.Confidence)
			require.Len(t, result.Rewards, len(round.Reveals))
			require.Empty(t, result.Slashes)
			for i, reveal := range round.Reveals {
				require.Equal(t, reveal.Verifier, result.Rewards[i].Verifier)
			}
		})
	}
}

func TestNeutralReviewCompletesWithoutScoreOrNominalRewardWrites(t *testing.T) {
	f, claim, round := neutralReviewFixture(t, []string{"accept", "accept", "reject", ""})
	// Caller-supplied recipients/slashes/verdict are not authoritative.
	err := f.k.CompleteRound(f.ctx, round, &VerificationResult{Verdict: types.Verdict_VERDICT_REJECT, Rewards: []VerifierReward{{Verifier: claim.Submitter}}, Slashes: []VerifierSlash{{Verifier: claim.Submitter, SlashBps: 1000000}}})
	require.NoError(t, err)
	after, found := f.k.GetVerificationRound(f.ctx, round.Id)
	require.True(t, found)
	require.Equal(t, types.Verdict_VERDICT_ACCEPT, after.Verdict)
	require.Len(t, after.VerifierRewardSettlement.Payments, 3)
	require.Equal(t, "19", after.VerifierRewardSettlement.Payments[0].Amount)
	require.Equal(t, "18", after.VerifierRewardSettlement.Payments[1].Amount)
	require.Equal(t, "0", after.VerifierRewardSettlement.WithheldTotal)
	require.Zero(t, after.VerifierRewardSettlement.PaidAtBlock, "nil bank leaves an explicit unchanged obligation")
	fact, found := f.k.GetFact(f.ctx, GenerateFactID(claim.Id, uint64(f.ctx.BlockHeight())))
	require.True(t, found)
	require.Equal(t, uint64(666666), fact.Confidence, "no role or partnership multiplier")
	require.Zero(t, fact.SubmitterCalibrationSnapshotBps)
	_, hasPending := f.k.GetSurvivalPendingReward(f.ctx, fact.Id)
	require.False(t, hasPending)
	_, cal := f.k.GetAgentCalibration(f.ctx, claim.Submitter)
	require.False(t, cal)
	for _, reveal := range round.Reveals {
		_, exists, err := f.k.GetValidatorIndependence(f.ctx, reveal.Verifier)
		require.NoError(t, err)
		require.False(t, exists)
	}
	verifyEvents := 0
	for _, event := range f.ctx.EventManager().Events() {
		if event.Type != "zerone.karma.edge" {
			continue
		}
		attrs := map[string]string{}
		for _, attr := range event.Attributes {
			attrs[attr.Key] = attr.Value
		}
		if attrs["kind"] == "verify" {
			verifyEvents++
			require.Equal(t, "valid_review", attrs["assessment_basis"])
		}
	}
	require.Equal(t, 3, verifyEvents)
	frozen := proto.Clone(after)
	require.NoError(t, f.k.CompleteRound(f.ctx, after, &VerificationResult{Verdict: types.Verdict_VERDICT_ACCEPT}))
	after, _ = f.k.GetVerificationRound(f.ctx, round.Id)
	require.True(t, proto.Equal(frozen, after))
}

func TestNeutralReviewCorruptionAndFailedCompletionRetainRetry(t *testing.T) {
	f, _, round := neutralReviewFixture(t, []string{"accept", "reject"})
	before := f.snapshot(t)
	*f.kFault = handoffFault{"get", types.ClaimKeyPrefix, false}
	_, err := f.k.AggregateVerificationResult(f.ctx, round)
	require.Error(t, err)
	require.Equal(t, before, f.snapshot(t))
	*f.kFault = handoffFault{"set", types.VerificationRoundKeyPrefix, true}
	require.Error(t, f.k.CompleteRound(f.ctx, round, &VerificationResult{}))
	require.Equal(t, before, f.snapshot(t))
	*f.kFault = handoffFault{}
	// Even at the terminal deadline a below-quorum panel receives its pool.
	require.NoError(t, f.k.AdvanceRoundPhases(f.ctx))
	after, found := f.k.GetVerificationRound(f.ctx, round.Id)
	require.True(t, found)
	require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_COMPLETE, after.Phase)
	require.Equal(t, types.Verdict_VERDICT_INCONCLUSIVE, after.Verdict)
	require.Len(t, after.VerifierRewardSettlement.Payments, 2)
	require.Equal(t, "28", after.VerifierRewardSettlement.Payments[0].Amount)
	require.Equal(t, "27", after.VerifierRewardSettlement.Payments[1].Amount)
}

func TestNeutralReviewCannotCompleteAnOpenPanel(t *testing.T) {
	f, _, round := neutralReviewFixture(t, []string{"accept", "reject", ""})
	f.ctx = f.ctx.WithBlockHeight(88)
	before := f.snapshot(t)
	require.ErrorContains(t, f.k.CompleteRound(f.ctx, round, &VerificationResult{}), "panel is not closed")
	require.Equal(t, before, f.snapshot(t))
}

func TestNeutralReviewBirthPressureKeepsPopulationCapacityWithoutCaptureScores(t *testing.T) {
	f, _, _ := neutralReviewFixture(t, nil)
	params, err := f.k.GetParams(f.ctx)
	require.NoError(t, err)
	params.DomainBaseCapacity = 100
	params.UnderpopulationBirthBonusBps = 500000
	require.NoError(t, f.k.SetParams(f.ctx, params))
	f.k.SetDomainStats(f.ctx, &DomainStats{Domain: "physics", ActiveCount: 50})
	energy, err := f.k.applyNeutralReviewBirthPressure(f.ctx, "physics", 100)
	require.NoError(t, err)
	require.Equal(t, uint64(125), energy)
	f.k.SetDomainStats(f.ctx, &DomainStats{Domain: "physics", ActiveCount: 100})
	energy, err = f.k.applyNeutralReviewBirthPressure(f.ctx, "physics", 100)
	require.NoError(t, err)
	require.Equal(t, uint64(100), energy)
	*f.kFault = handoffFault{"get", types.DomainStatsKey("physics"), false}
	_, err = f.k.applyNeutralReviewBirthPressure(f.ctx, "physics", 100)
	require.Error(t, err)
}

func TestNeutralReviewLegacyFinancialRoundCannotProduceNewAgreementScores(t *testing.T) {
	f, _, neutral := neutralReviewFixture(t, nil)
	claim := &types.Claim{Id: "legacy-inflight-claim", Submitter: "legacy-author", Stake: "100", Domain: "physics"}
	require.NoError(t, f.k.SetClaim(f.ctx, claim))
	round := &types.VerificationRound{Id: "legacy-inflight-round", ClaimId: claim.Id, Phase: types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION, Reveals: []*types.RevealEntry{{Verifier: neutral.Id, Vote: "accept"}}}
	require.NoError(t, f.k.SetVerificationRound(f.ctx, round))
	// A nonagreeing legacy voter must not trigger fresh capture/reputation
	// feedback after global retirement, even though financial policy stays 0.
	require.NoError(t, f.k.CompleteRound(f.ctx, round, &VerificationResult{Verdict: types.Verdict_VERDICT_INCONCLUSIVE}))
	stored, found := f.k.GetVerificationRound(f.ctx, round.Id)
	require.True(t, found)
	require.Zero(t, stored.ReviewPolicyVersion)
	_, found = f.k.GetAgentCalibration(f.ctx, claim.Submitter)
	require.False(t, found)
	_, found, err := f.k.GetValidatorIndependence(f.ctx, neutral.Id)
	require.NoError(t, err)
	require.False(t, found)
}
