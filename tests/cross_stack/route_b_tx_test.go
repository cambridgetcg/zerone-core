package cross_stack_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// TestRouteB_TxPipelineRegistration exercises the full tx-gated pipeline
// lifecycle: register → reject unauthorized update → legitimate update →
// register model card → unauthorized update rejected → legitimate update
// → retire.
func TestRouteB_TxPipelineAndModelCardTxFlow(t *testing.T) {
	h := NewTestHarness(t)
	require.NoError(t, h.KnowledgeKeeper.SeedDefaultMethodologies(h.Ctx))
	require.NoError(t, h.KnowledgeKeeper.SeedDefaultTokenizerSpec(h.Ctx))

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)

	operator := "zerone1pipelineoperator0000000000000001"
	otherAddr := "zerone1otherparty000000000000000000001"
	owner := "zerone1modelownerroutebtx0000000000001"
	deploymentAddr := "zerone1deployedmodelagenttx000000aaaaa"

	// ─── 1. Register pipeline ─────────────────────────────────────────
	_, err := ms.RegisterTrainingPipeline(h.Ctx, &knowledgetypes.MsgRegisterTrainingPipeline{
		Operator:              operator,
		Id:                    "pipe-tx-1",
		CorpusSnapshotHeight:  1,
		TokenizerVersion:      1,
		MethodologySetVersion: 1,
		RecipeHash:            "sha256:deadbeef",
		Description:           "tx-driven pipeline",
		CorpusFilter:          `{"min_tier":"GOLD"}`,
	})
	require.NoError(t, err)

	// Duplicate registration rejected.
	_, err = ms.RegisterTrainingPipeline(h.Ctx, &knowledgetypes.MsgRegisterTrainingPipeline{
		Operator: operator,
		Id:       "pipe-tx-1",
	})
	require.Error(t, err)

	// Unknown tokenizer version rejected.
	_, err = ms.RegisterTrainingPipeline(h.Ctx, &knowledgetypes.MsgRegisterTrainingPipeline{
		Operator:         operator,
		Id:               "pipe-tx-bad-tokenizer",
		TokenizerVersion: 99,
	})
	require.Error(t, err)

	// ─── 2. Update pipeline — only operator may ───────────────────────
	_, err = ms.UpdateTrainingPipeline(h.Ctx, &knowledgetypes.MsgUpdateTrainingPipeline{
		Operator:  otherAddr,
		Id:        "pipe-tx-1",
		NewStatus: "running",
	})
	require.Error(t, err, "non-operator must not update pipeline")
	require.Contains(t, err.Error(), "only the declaring operator")

	_, err = ms.UpdateTrainingPipeline(h.Ctx, &knowledgetypes.MsgUpdateTrainingPipeline{
		Operator:         operator,
		Id:               "pipe-tx-1",
		NewStatus:        "completed",
		CompletedAtBlock: 100,
	})
	require.NoError(t, err)

	pipeline, ok := h.KnowledgeKeeper.GetTrainingPipeline(h.Ctx, "pipe-tx-1")
	require.True(t, ok)
	require.Equal(t, "completed", pipeline.Status)
	require.Equal(t, uint64(100), pipeline.CompletedAtBlock)

	// ─── 3. Register model card tied to the pipeline ──────────────────
	_, err = ms.RegisterModelCard(h.Ctx, &knowledgetypes.MsgRegisterModelCard{
		Owner:                    owner,
		Id:                       "model-tx-1",
		Name:                     "ZERONE-native-v0.1",
		PipelineId:               "pipe-tx-1",
		DeploymentAddress:        deploymentAddr,
		ParameterCount:           7,
		Route:                    "from_scratch",
		EvalAcceptanceRateBps:    650_000,
		EvalCorroborationRateBps: 300_000,
		EvalSampleSize:           500,
	})
	require.NoError(t, err)

	// Invalid route rejected.
	_, err = ms.RegisterModelCard(h.Ctx, &knowledgetypes.MsgRegisterModelCard{
		Owner:      owner,
		Id:         "model-tx-bad-route",
		PipelineId: "pipe-tx-1",
		Route:      "magic",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid route")

	// Reference to unknown pipeline rejected.
	_, err = ms.RegisterModelCard(h.Ctx, &knowledgetypes.MsgRegisterModelCard{
		Owner:      owner,
		Id:         "model-tx-orphan",
		PipelineId: "does-not-exist",
		Route:      "from_scratch",
	})
	require.Error(t, err)

	// ─── 4. Update card — only owner may ──────────────────────────────
	_, err = ms.UpdateModelCard(h.Ctx, &knowledgetypes.MsgUpdateModelCard{
		Owner:                 otherAddr,
		Id:                    "model-tx-1",
		EvalAcceptanceRateBps: 900_000,
	})
	require.Error(t, err, "non-owner must not update model card")

	_, err = ms.UpdateModelCard(h.Ctx, &knowledgetypes.MsgUpdateModelCard{
		Owner:                 owner,
		Id:                    "model-tx-1",
		EvalAcceptanceRateBps: 900_000,
	})
	require.NoError(t, err)

	card, ok := h.KnowledgeKeeper.GetModelCard(h.Ctx, "model-tx-1")
	require.True(t, ok)
	require.Equal(t, uint64(900_000), card.EvalAcceptanceRateBps)
	require.True(t, card.Active)

	// ─── 5. Retire card — only owner; after retirement, update fails ─
	_, err = ms.RetireModelCard(h.Ctx, &knowledgetypes.MsgRetireModelCard{
		Owner:  otherAddr,
		Id:     "model-tx-1",
		Reason: "unauthorized",
	})
	require.Error(t, err, "non-owner must not retire model card")

	_, err = ms.RetireModelCard(h.Ctx, &knowledgetypes.MsgRetireModelCard{
		Owner:  owner,
		Id:     "model-tx-1",
		Reason: "superseded by v0.2",
	})
	require.NoError(t, err)

	retiredCard, _ := h.KnowledgeKeeper.GetModelCard(h.Ctx, "model-tx-1")
	require.False(t, retiredCard.Active)
	require.Greater(t, retiredCard.RetiredAtBlock, uint64(0))
	require.Equal(t, "superseded by v0.2", retiredCard.RetiredReason)

	// Double-retire rejected.
	_, err = ms.RetireModelCard(h.Ctx, &knowledgetypes.MsgRetireModelCard{
		Owner: owner, Id: "model-tx-1", Reason: "again",
	})
	require.Error(t, err)

	// Update after retirement rejected.
	_, err = ms.UpdateModelCard(h.Ctx, &knowledgetypes.MsgUpdateModelCard{
		Owner: owner, Id: "model-tx-1", EvalAcceptanceRateBps: 100,
	})
	require.Error(t, err)
}
