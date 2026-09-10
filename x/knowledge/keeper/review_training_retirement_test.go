package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func TestReviewNeutralityKeepsOwnerAttributionWithoutComputedValue(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)
	require.NoError(t, k.SetFact(ctx, &types.Fact{Id: "fact", Status: types.FactStatus_FACT_STATUS_VERIFIED, CorroborationCount: 100, SubmitterCalibrationSnapshotBps: 900_000}))
	old := &types.ContributionRecord{ModelId: "old-model", FactIds: []string{"fact"}, AttributedBy: "owner", ComputedTvw: 777, TotalWeight: 101, PerFactCalibrationBps: []uint64{900_000}}
	require.NoError(t, k.SetContributionRecord(ctx, old))
	require.NoError(t, k.EnableReviewNeutrality(ctx))
	server := keeper.NewMsgServerImpl(k)
	for _, declared := range []uint64{0, 7} {
		model := "new-model"
		if declared > 0 {
			model = "declared-weight"
		}
		require.NoError(t, k.SetModelCard(ctx, &types.ModelCard{Id: model, OwnerAddress: "owner"}))
		_, err := server.AttributeContributions(ctx, &types.MsgAttributeContributions{ModelId: model, Owner: "other", FactIds: []string{"fact"}})
		require.ErrorContains(t, err, "only the model owner")
		response, err := server.AttributeContributions(ctx, &types.MsgAttributeContributions{ModelId: model, Owner: "owner", FactIds: []string{"fact"}, TotalWeight: declared})
		require.NoError(t, err)
		require.EqualValues(t, 1, response.Recorded)
		record, found := k.GetContributionRecord(ctx, model)
		require.True(t, found)
		require.Equal(t, []string{"fact"}, record.FactIds)
		require.Equal(t, "owner", record.AttributedBy)
		require.EqualValues(t, 1, record.AttributionPolicyVersion)
		require.Equal(t, declared, record.TotalWeight)
		require.Zero(t, record.ComputedTvw)
		require.Empty(t, record.PerFactCalibrationBps)
		require.Contains(t, k.GetModelsThatUsedFact(ctx, "fact"), model)
	}
	after, found := k.GetContributionRecord(ctx, "old-model")
	require.True(t, found)
	require.True(t, proto.Equal(old, after), "retirement does not rewrite historical valuation records")
	_, err := server.ClaimTrainingFundDisbursement(ctx, &types.MsgClaimTrainingFundDisbursement{Id: "payment", ModelId: "new-model", Claimant: "owner"})
	require.ErrorContains(t, err, "disbursement disabled", "owner declarations cannot reopen disabled training payments")
}

func TestReviewNeutralityPreservesUnscoredManifestsAndOldFinalization(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)
	require.NoError(t, k.SetFact(ctx, &types.Fact{Id: "fact", Status: types.FactStatus_FACT_STATUS_VERIFIED, SubmitterCalibrationSnapshotBps: 900_000}))
	require.NoError(t, k.SetTrainingPipeline(ctx, &types.TrainingPipeline{Id: "pipeline", OperatorAddress: "owner"}))
	server := keeper.NewMsgServerImpl(k)
	_, err := server.CreateTrainingManifest(ctx, &types.MsgCreateTrainingManifest{Id: "old-scored", PipelineId: "pipeline", Creator: "owner", CorpusSelector: &types.CorpusSelector{MinSubmitterCalibrationBps: 800_000}})
	require.NoError(t, err)
	old, found := k.GetTrainingManifest(ctx, "old-scored")
	require.True(t, found)
	require.NoError(t, k.EnableReviewNeutrality(ctx))
	_, err = server.CreateTrainingManifest(ctx, &types.MsgCreateTrainingManifest{Id: "new-scored", PipelineId: "pipeline", Creator: "owner", CorpusSelector: &types.CorpusSelector{MinSubmitterCalibrationBps: 1}})
	require.ErrorContains(t, err, "retired submitter calibration")
	_, found = k.GetTrainingManifest(ctx, "new-scored")
	require.False(t, found)
	_, err = server.CreateTrainingManifest(ctx, &types.MsgCreateTrainingManifest{Id: "new-unscored", PipelineId: "pipeline", Creator: "owner"})
	require.NoError(t, err)
	_, err = server.FinalizeTrainingManifest(ctx, &types.MsgFinalizeTrainingManifest{ManifestId: "old-scored", Creator: "owner"})
	require.NoError(t, err)
	after, found := k.GetTrainingManifest(ctx, "old-scored")
	require.True(t, found)
	require.Equal(t, old.IncludedFactIds, after.IncludedFactIds)
	require.True(t, proto.Equal(old.CorpusSelector, after.CorpusSelector))
	require.Equal(t, types.ManifestStatus_MANIFEST_STATUS_FINALIZED, after.Status)
}

func TestReviewNeutralityClosesNewAugmentationsPreservesOldBallotsAndEscrow(t *testing.T) {
	k, ctx, bank, _ := setupKnowledgeTestFull(t)
	require.NoError(t, k.SetFact(ctx, &types.Fact{Id: "fact", Status: types.FactStatus_FACT_STATUS_VERIFIED, Domain: "physics"}))
	qualification := newMockDomainQualificationKeeper()
	for _, voter := range []string{"v1", "v2", "v3"} {
		qualification.qualify(voter, "physics")
	}
	k.SetDomainQualificationKeeper(qualification)
	server := keeper.NewMsgServerImpl(k)
	sponsor := sdk.AccAddress([]byte("sponsor-address-0001")).String()
	submitter := sdk.AccAddress([]byte("submitter-address-01")).String()
	for _, id := range []string{"pay-bounty", "expire-bounty"} {
		_, err := server.CreateAugmentationBounty(ctx, &types.MsgCreateAugmentationBounty{Id: id, Sponsor: sponsor, TargetFactId: "fact", RewardPerVariant: 1_000, MaxVariants: 1, ExpiresAtBlock: 200})
		require.NoError(t, err)
	}
	_, err := server.SubmitAugmentation(ctx, &types.MsgSubmitAugmentation{Id: "pending", BountyId: "pay-bounty", OriginalFactId: "fact", VariantContent: "variant", Submitter: submitter})
	require.NoError(t, err)
	_, err = server.VoteOnAugmentation(ctx, &types.MsgVoteOnAugmentation{AugmentationId: "pending", Verifier: "v1", Vote: types.AugmentationVerdict_AUGMENTATION_VERDICT_EQUIVALENT})
	require.NoError(t, err)
	before, found := k.GetAugmentation(ctx, "pending")
	require.True(t, found)
	require.NoError(t, k.EnableReviewNeutrality(ctx))
	bankCalls := len(bank.sendCalls)
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	_, err = server.CreateAugmentationBounty(ctx, &types.MsgCreateAugmentationBounty{Id: "new", Sponsor: sponsor, TargetFactId: "fact", RewardPerVariant: 1_000, MaxVariants: 1})
	require.ErrorContains(t, err, "retired")
	_, err = server.SubmitAugmentation(ctx, &types.MsgSubmitAugmentation{Id: "new", OriginalFactId: "fact", VariantContent: "new variant", Submitter: submitter})
	require.ErrorContains(t, err, "retired")
	require.Len(t, bank.sendCalls, bankCalls)
	require.Empty(t, ctx.EventManager().Events())
	for _, voter := range []string{"v2", "v3"} {
		_, err = server.VoteOnAugmentation(ctx, &types.MsgVoteOnAugmentation{AugmentationId: "pending", Verifier: voter, Vote: types.AugmentationVerdict_AUGMENTATION_VERDICT_EQUIVALENT})
		require.NoError(t, err)
	}
	after, found := k.GetAugmentation(ctx, "pending")
	require.True(t, found)
	require.Equal(t, before.VerdictVoters, after.VerdictVoters[:1])
	require.Equal(t, before.VerdictVoteStakes, after.VerdictVoteStakes[:1])
	require.Equal(t, before.VerdictVoteCalibrationBps, after.VerdictVoteCalibrationBps[:1])
	require.True(t, after.Accepted)
	require.Equal(t, "1000", after.PayoutAmount)
	require.Equal(t, "500", k.GetEscrowLocked(ctx, "pay-bounty").String())
	require.Empty(t, qualification.outcomes, "legacy financial finalization must not restart global score feedback")
	require.Equal(t, submitter, bank.sendCalls[bankCalls].to)
	require.Equal(t, "1000uzrn", bank.sendCalls[bankCalls].coins.String())
	k.ProcessRouteBLifecycle(ctx.WithBlockHeight(201))
	require.True(t, k.GetEscrowLocked(ctx, "expire-bounty").IsZero())
	last := bank.sendCalls[len(bank.sendCalls)-1]
	require.Equal(t, sponsor, last.to)
	require.Equal(t, "1455uzrn", last.coins.String(), "legacy expiry fee/refund is preserved")
}
