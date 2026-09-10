package cross_stack_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	qualificationtypes "github.com/zerone-chain/zerone/x/qualification/types"
)

func TestReviewNeutralityNativeAttributionAndFeedbackRetirement(t *testing.T) {
	h := NewTestHarness(t)
	enabled, err := h.KnowledgeKeeper.ReviewNeutralityEnabled(h.Ctx)
	require.NoError(t, err)
	require.True(t, enabled, "the real native application defaults to the new policy")
	ownerAddr := testAddr("neutral-owner")
	owner := ownerAddr.String()
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{Id: "used-fact", Domain: "physics", Status: knowledgetypes.FactStatus_FACT_STATUS_VERIFIED, Submitter: owner, Confidence: 800_000, CorroborationCount: 99, SubmitterCalibrationSnapshotBps: 999_999}))
	require.NoError(t, h.KnowledgeKeeper.SetModelCard(h.Ctx, &knowledgetypes.ModelCard{Id: "declared-model", OwnerAddress: owner}))
	legacy := &knowledgetypes.ContributionRecord{ModelId: "archived-model", AttributedBy: owner, FactIds: []string{"used-fact"}, ComputedTvw: 999, PerFactCalibrationBps: []uint64{999_999}}
	require.NoError(t, h.KnowledgeKeeper.SetContributionRecord(h.Ctx, legacy))
	q := &qualificationtypes.DomainQualification{Validator: owner, Domain: "physics", Status: qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_ACTIVE, Weight: 90, Metrics: &qualificationtypes.QualificationMetrics{TotalVerifications: 50, CorrectVerifications: 1, AccuracyBps: 20_000}}
	h.QualificationKeeper.SetQualification(h.Ctx, q)
	supply := h.BankKeeper.GetSupply(h.Ctx, "uzrn")
	balance := h.BankKeeper.GetBalance(h.Ctx, ownerAddr, "uzrn")
	h.Ctx = h.Ctx.WithEventManager(sdk.NewEventManager())
	require.NoError(t, h.QualificationKeeper.RecordVerificationOutcome(h.Ctx, owner, "physics", true))
	require.NoError(t, h.KnowledgeKeeper.RecordSubmissionOutcome(h.Ctx, owner, "method", knowledgetypes.Verdict_VERDICT_ACCEPT))
	require.NoError(t, h.QualificationKeeper.BeginBlocker(h.Ctx))
	afterQ, found := h.QualificationKeeper.GetQualification(h.Ctx, owner, "physics")
	require.True(t, found)
	require.True(t, proto.Equal(q, afterQ), "the application-wired policy freezes qualification feedback")
	_, found = h.KnowledgeKeeper.GetAgentCalibration(h.Ctx, owner)
	require.False(t, found)
	require.Empty(t, h.Ctx.EventManager().Events())
	_, err = knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper).AttributeContributions(h.Ctx, &knowledgetypes.MsgAttributeContributions{Owner: owner, ModelId: "declared-model", FactIds: []string{"used-fact"}, TotalWeight: 7})
	require.NoError(t, err)
	declared, found := h.KnowledgeKeeper.GetContributionRecord(h.Ctx, "declared-model")
	require.True(t, found)
	require.EqualValues(t, 1, declared.AttributionPolicyVersion)
	require.Equal(t, []string{"used-fact"}, declared.FactIds)
	require.EqualValues(t, 7, declared.TotalWeight)
	require.Zero(t, declared.ComputedTvw)
	require.Empty(t, declared.PerFactCalibrationBps)
	require.Equal(t, supply, h.BankKeeper.GetSupply(h.Ctx, "uzrn"))
	require.Equal(t, balance, h.BankKeeper.GetBalance(h.Ctx, ownerAddr, "uzrn"))
	// This checks the contribution slice's transport, not whole-chain identity.
	genesis := h.KnowledgeKeeper.ExportGenesis(h.Ctx)
	require.True(t, genesis.ReviewNeutralityEnabled)
	h2 := NewTestHarness(t)
	require.NoError(t, h2.KnowledgeKeeper.InitGenesis(h2.Ctx, genesis))
	for _, record := range []*knowledgetypes.ContributionRecord{legacy, declared} {
		restored, exists := h2.KnowledgeKeeper.GetContributionRecord(h2.Ctx, record.ModelId)
		require.True(t, exists)
		require.True(t, proto.Equal(record, restored))
		require.Contains(t, h2.KnowledgeKeeper.GetModelsThatUsedFact(h2.Ctx, "used-fact"), record.ModelId)
	}
}
