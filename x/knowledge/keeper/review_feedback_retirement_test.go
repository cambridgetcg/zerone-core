package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/zerone-chain/zerone/x/knowledge/types"
	"google.golang.org/protobuf/proto"
)

func TestReviewNeutralityPreservesHistoricalScoreRecords(t *testing.T) {
	f := setupHandoff(t)
	require.NoError(t, f.k.RecordSubmissionOutcome(f.ctx, "submitter", "method", types.Verdict_VERDICT_ACCEPT))
	old, found := f.k.GetAgentCalibration(f.ctx, "submitter")
	require.True(t, found)
	require.EqualValues(t, 1, old.Accepted, "legacy producers still operate before activation")
	require.NoError(t, f.k.SetDomainRoleRecord(f.ctx, &types.DomainRoleRecord{Domain: "physics", AgentCorrectCalls: 101, HumanIncorrectCalls: 99}))
	require.NoError(t, f.k.EnableReviewNeutrality(f.ctx))
	f.ctx = f.ctx.WithEventManager(sdk.NewEventManager())
	before := f.snapshot(t)
	for _, verdict := range []types.Verdict{types.Verdict_VERDICT_ACCEPT, types.Verdict_VERDICT_REJECT, types.Verdict_VERDICT_MALFORMED, types.Verdict_VERDICT_INCONCLUSIVE} {
		require.NoError(t, f.k.RecordSubmissionOutcome(f.ctx, "submitter", "method", verdict))
	}
	require.NoError(t, f.k.RecordSubmissionOutcome(f.ctx, "new-submitter", "method", types.Verdict_VERDICT_ACCEPT))
	require.NoError(t, f.k.RecordCorroborationForSubmitter(f.ctx, "submitter", "method"))
	require.NoError(t, f.k.RecordDisprovalForSubmitter(f.ctx, "submitter", "method"))
	require.NoError(t, f.k.RecordChallengeOutcome(f.ctx, "submitter", true))
	require.NoError(t, f.k.RecordChallengeOutcome(f.ctx, "submitter", false))
	require.NoError(t, f.k.RecordVindicationRoleImpact(f.ctx, &types.VerificationRound{}, "physics"))
	require.NoError(t, f.k.RecordChallengeRoleImpact(f.ctx, "fact", "physics", true))
	require.NoError(t, f.k.RecordChallengeRoleImpact(f.ctx, "fact", "physics", false))
	require.NoError(t, f.k.DecayRoleRecords(f.ctx))
	require.Equal(t, before, f.snapshot(t), "all primary records and indexes remain byte-identical")
	require.Empty(t, f.ctx.EventManager().Events())
	_, found = f.k.GetAgentCalibration(f.ctx, "new-submitter")
	require.False(t, found)
}

func TestContributionPolicyValidationBeforeWriteAndGenesisImport(t *testing.T) {
	for _, invalid := range []*types.ContributionRecord{
		{ModelId: "model", AttributionPolicyVersion: 2},
		{ModelId: "model", AttributionPolicyVersion: 1, ComputedTvw: 1},
		{ModelId: "model", AttributionPolicyVersion: 1, PerFactCalibrationBps: []uint64{0}},
	} {
		f := setupHandoff(t)
		old := &types.ContributionRecord{ModelId: "model", FactIds: []string{"fact"}, ComputedTvw: 999, PerFactCalibrationBps: []uint64{500_000}}
		require.NoError(t, f.k.SetContributionRecord(f.ctx, old))
		before := f.snapshot(t)
		require.Error(t, f.k.SetContributionRecord(f.ctx, invalid))
		require.Equal(t, before, f.snapshot(t))
		gs := types.DefaultGenesis()
		gs.ContributionRecords = []*types.ContributionRecord{invalid}
		require.Error(t, f.k.InitGenesis(f.ctx, gs))
		after, found := f.k.GetContributionRecord(f.ctx, "model")
		require.True(t, found)
		require.True(t, proto.Equal(old, after))
	}
}

func TestReviewNeutralityRetiredProducersRefusePolicyFailure(t *testing.T) {
	for _, mode := range []string{"malformed", "read-error"} {
		t.Run(mode, func(t *testing.T) {
			f := setupHandoff(t)
			if mode == "malformed" {
				f.ctx.KVStore(f.keys[0]).Set([]byte(ReviewNeutralityEnabledStoreKey), []byte{2})
			} else {
				*f.kFault = handoffFault{"get", []byte(ReviewNeutralityEnabledStoreKey), false}
			}
			before := f.snapshot(t)
			require.Error(t, f.k.RecordSubmissionOutcome(f.ctx, "submitter", "method", types.Verdict_VERDICT_ACCEPT))
			require.Error(t, f.k.RecordCorroborationForSubmitter(f.ctx, "submitter", "method"))
			require.Error(t, f.k.RecordDisprovalForSubmitter(f.ctx, "submitter", "method"))
			require.Error(t, f.k.RecordChallengeOutcome(f.ctx, "submitter", true))
			require.Error(t, f.k.RecordVindicationRoleImpact(f.ctx, nil, "physics"))
			require.Error(t, f.k.RecordChallengeRoleImpact(f.ctx, "fact", "physics", true))
			require.Error(t, f.k.DecayRoleRecords(f.ctx))
			server := NewMsgServerImpl(f.k)
			_, err := server.AttributeContributions(f.ctx, &types.MsgAttributeContributions{ModelId: "model"})
			require.ErrorContains(t, err, "review neutrality marker")
			_, err = server.CreateAugmentationBounty(f.ctx, &types.MsgCreateAugmentationBounty{Id: "bounty"})
			require.ErrorContains(t, err, "review neutrality marker")
			_, err = server.SubmitAugmentation(f.ctx, &types.MsgSubmitAugmentation{Id: "augmentation"})
			require.ErrorContains(t, err, "review neutrality marker")
			_, err = server.CreateTrainingManifest(f.ctx, &types.MsgCreateTrainingManifest{Id: "manifest"})
			require.ErrorContains(t, err, "review neutrality marker")
			_, _, err = f.k.RecordAugmentationVote(f.ctx, "augmentation", "voter", types.AugmentationVerdict_AUGMENTATION_VERDICT_EQUIVALENT)
			require.ErrorContains(t, err, "review neutrality marker")
			require.ErrorContains(t, f.k.ApplyFinalizedAugmentationVerdict(f.ctx, "augmentation", types.AugmentationVerdict_AUGMENTATION_VERDICT_EQUIVALENT), "review neutrality marker")
			require.Equal(t, before, f.snapshot(t))
			require.Empty(t, f.ctx.EventManager().Events())
		})
	}
}
