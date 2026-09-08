package keeper_test

import (
	"fmt"
	"math"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func feedbackTemperatureDomain(t *testing.T) (keeper.Keeper, sdk.Context, string) {
	t.Helper()
	k, ctx := setupKnowledgeTest(t)
	p, err := k.GetParams(ctx)
	require.NoError(t, err)
	p.FitnessEpochBlocks = 10
	require.NoError(t, k.SetParams(ctx, p))
	domain := "temperature-feedback"
	require.NoError(t, k.SetDomain(ctx, &types.Domain{Name: domain}))
	require.NoError(t, k.SetDomainEpistemicState(ctx, &types.DomainEpistemicState{Domain: domain, Temperature: 500000}))
	return k, ctx, domain
}

func requireFeedbackTemperature(t *testing.T, k keeper.Keeper, ctx sdk.Context, domain string, temperature, streak, updateHeight uint64) {
	t.Helper()
	state, found, err := k.GetDomainEpistemicState(ctx, domain)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, temperature, state.Temperature)
	require.Equal(t, streak, state.ConformityStreak)
	require.Equal(t, updateHeight, state.LastTemperatureUpdate, "update height is not the diversity epoch or its last block")
}

func TestFeedbackTemperatureClosedEpochBoundaries(t *testing.T) {
	k, ctx, domain := feedbackTemperatureDomain(t)
	for _, height := range []int64{9, 19} {
		require.NoError(t, k.RecordRoundDiversity(ctx.WithBlockHeight(height), fmt.Sprintf("unanimous-%d", height), domain, 4, 0))
	}
	// Epoch 2 has no data; epoch 3 is healthy. Neither is conformity evidence.
	require.NoError(t, k.RecordRoundDiversity(ctx.WithBlockHeight(39), "healthy", domain, 2, 2))
	for _, step := range []struct {
		height                            int64
		temperature, streak, updateHeight uint64
	}{
		{0, 500000, 0, 0},
		{9, 500000, 0, 0},
		{10, 495000, 1, 10},
		{11, 495000, 1, 10},
		{19, 495000, 1, 10},
		{20, 485025, 2, 20},
		{21, 485025, 2, 20},
		{29, 485025, 2, 20},
		{30, 485100, 0, 30},
		{31, 485100, 0, 30},
		{39, 485100, 0, 30},
		{40, 485175, 0, 40},
		{41, 485175, 0, 40},
	} {
		require.NoError(t, k.BeginBlocker(ctx.WithBlockHeight(step.height)), "height %d", step.height)
		requireFeedbackTemperature(t, k, ctx, domain, step.temperature, step.streak, step.updateHeight)
	}
}

func TestFeedbackTemperatureIgnoresBoundaryRoundUntilClosed(t *testing.T) {
	k, ctx, domain := feedbackTemperatureDomain(t)
	require.NoError(t, k.RecordRoundDiversity(ctx.WithBlockHeight(9), "healthy-before", domain, 2, 2))
	claim := &types.Claim{Id: "temperature-boundary", FactContent: "Boundary round fixture", Domain: domain, Submitter: "boundary-submitter", Stake: "1000000", Status: types.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION}
	require.NoError(t, k.SetClaim(ctx, claim))
	round := earlyAggRound("temperature-round", claim.Id, types.VerificationPhase_VERIFICATION_PHASE_REVEAL, 4, 4)
	round.StartedAtBlock, round.CommitDeadline, round.RevealDeadline, round.AggregationDeadline = 1, 9, 10, 11
	for _, commit := range round.Commits {
		commit.CommittedAtBlock = 8
	}
	for _, reveal := range round.Reveals {
		reveal.RevealedAtBlock = 9
	}
	require.NoError(t, k.SetVerificationRound(ctx, round))
	require.NoError(t, k.BeginBlocker(ctx.WithBlockHeight(10)))
	completed, found := k.GetVerificationRound(ctx, round.Id)
	require.True(t, found)
	require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_COMPLETE, completed.Phase)
	require.EqualValues(t, 10, completed.VerdictBlock)
	diversity, found, err := k.GetRoundDiversity(ctx, round.Id)
	require.NoError(t, err)
	require.True(t, found)
	require.EqualValues(t, 1, diversity.Epoch)
	require.Zero(t, diversity.Entropy)
	requireFeedbackTemperature(t, k, ctx, domain, 500000, 0, 10)
	// Even an already materialized OPEN summary must not become cooling evidence.
	require.NoError(t, k.AggregateDomainDiversity(ctx, domain, 1))
	require.NoError(t, k.UpdateEpistemicTemperature(ctx.WithBlockHeight(10), domain))
	requireFeedbackTemperature(t, k, ctx, domain, 500000, 0, 10)
	require.NoError(t, k.BeginBlocker(ctx.WithBlockHeight(11)))
	requireFeedbackTemperature(t, k, ctx, domain, 500000, 0, 10)
	require.NoError(t, k.BeginBlocker(ctx.WithBlockHeight(20)))
	requireFeedbackTemperature(t, k, ctx, domain, 495000, 1, 20)
	require.NoError(t, k.BeginBlocker(ctx.WithBlockHeight(21)))
	requireFeedbackTemperature(t, k, ctx, domain, 495000, 1, 20)
}

func TestFeedbackTemperatureCoolingKeepsCurrentHeightHeating(t *testing.T) {
	k, ctx, domain := feedbackTemperatureDomain(t)
	require.NoError(t, k.RecordRoundDiversity(ctx.WithBlockHeight(9), "cool-before", domain, 4, 0))
	require.NoError(t, k.RecordRoundDiversity(ctx.WithBlockHeight(19), "cool-after", domain, 4, 0))
	for i, height := range []uint64{9, 10, 11} {
		id := string(rune('a' + i))
		makeTestFact(t, k, ctx, id, "vindication fixture", domain, "general", "zrn1submitter1", 700000)
		require.NoError(t, k.SetVindicationRecord(ctx, id, types.VindicationRecord{FactId: id, Verifier: "v", VindicatedAt: height}))
	}
	require.NoError(t, k.BeginBlocker(ctx.WithBlockHeight(9)))
	requireFeedbackTemperature(t, k, ctx, domain, 500000, 0, 0)
	require.NoError(t, k.BeginBlocker(ctx.WithBlockHeight(10)))
	requireFeedbackTemperature(t, k, ctx, domain, 695000, 1, 10)
	state, _, err := k.GetDomainEpistemicState(ctx, domain)
	require.NoError(t, err)
	require.EqualValues(t, 2, state.VindicationCount, "heating includes H but not H+1")
	require.NoError(t, k.BeginBlocker(ctx.WithBlockHeight(11)))
	requireFeedbackTemperature(t, k, ctx, domain, 695000, 1, 10)
	require.NoError(t, k.BeginBlocker(ctx.WithBlockHeight(20)))
	requireFeedbackTemperature(t, k, ctx, domain, 784025, 2, 20)
	state, _, err = k.GetDomainEpistemicState(ctx, domain)
	require.NoError(t, err)
	require.EqualValues(t, 3, state.VindicationCount, "prior vindications must not heat twice")
}

func TestFeedbackTemperatureEpochZeroHasNoClosedDiversity(t *testing.T) {
	for _, height := range []int64{0, 9} {
		t.Run(fmt.Sprintf("height_%d", height), func(t *testing.T) {
			k, ctx, domain := feedbackTemperatureDomain(t)
			require.NoError(t, k.SetDomainEpistemicState(ctx, &types.DomainEpistemicState{Domain: domain, Temperature: 800000, ConformityStreak: 4}))
			// Both the open epoch and an underflow sentinel carry conformity data.
			for _, epoch := range []uint64{0, math.MaxUint64} {
				require.NoError(t, k.SetDomainDiversity(ctx, domain, epoch, keeper.DomainDiversityRecord{Domain: domain, Epoch: epoch, RoundCount: 1}))
			}
			require.NoError(t, k.BeginBlocker(ctx.WithBlockHeight(height)))
			requireFeedbackTemperature(t, k, ctx, domain, 800000, 4, 0)
			require.NoError(t, k.UpdateEpistemicTemperature(ctx.WithBlockHeight(height), domain))
			requireFeedbackTemperature(t, k, ctx, domain, 798500, 0, uint64(height))
		})
	}
}
