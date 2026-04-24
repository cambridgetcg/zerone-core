package cross_stack_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// Moat-integrity tests. Each pins a gate that protects the "chain-attested
// model is the most trustworthy model" thesis. Failure of any one of these
// means an unverified input can reach the training substrate without going
// through the full trust chain.

// ConfidenceThreshold is the gate deciding whether a verification round
// accepts a claim. Governance setting it to zero would accept every claim
// regardless of verifier consensus — a single-parameter bypass of the
// entire verifier panel. Params.Validate must reject it.
func TestMoat_ConfidenceThresholdFloor(t *testing.T) {
	p := knowledgetypes.DefaultParams()
	require.NoError(t, p.Validate())

	p.ConfidenceThreshold = 0
	err := p.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "confidence_threshold must be > 0")

	// QuorumThreshold floor is the sibling gate — a single verifier
	// could otherwise carry a round. Guard both together.
	p = knowledgetypes.DefaultParams()
	p.QuorumThreshold = 0
	err = p.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "quorum_threshold must be > 0")
}

// A fact in PENDING, PROVISIONAL, AT_RISK, REVOKED, EXPIRED, PRUNED, or
// SUPERSEDED status must earn zero TVW. These are the statuses that either
// never passed verification or have been invalidated; paying them would
// mean the attribution-revenue pipeline rewards unverified training data.
func TestMoat_TVWStatusGate(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	qs := knowledgekeeper.NewQueryServerImpl(h.KnowledgeKeeper)
	submitter := testAddr("moat_tvw_sub").String()

	base := func(id string, status knowledgetypes.FactStatus) *knowledgetypes.Fact {
		return &knowledgetypes.Fact{
			Id: id, Content: "moat test", Domain: "sciences",
			Status:                          status,
			Submitter:                       submitter,
			MethodId:                        knowledgetypes.MethodologyEmpirical,
			Confidence:                      900_000,
			CorroborationCount:              3,
			SubmitterCalibrationSnapshotBps: 800_000,
			AxiomDistance:                   2,
		}
	}

	eligible := []knowledgetypes.FactStatus{
		knowledgetypes.FactStatus_FACT_STATUS_VERIFIED,
		knowledgetypes.FactStatus_FACT_STATUS_ACTIVE,
		knowledgetypes.FactStatus_FACT_STATUS_CONTESTED,
		knowledgetypes.FactStatus_FACT_STATUS_CHALLENGED,
	}
	ineligible := []knowledgetypes.FactStatus{
		knowledgetypes.FactStatus_FACT_STATUS_UNSPECIFIED,
		knowledgetypes.FactStatus_FACT_STATUS_PENDING,
		knowledgetypes.FactStatus_FACT_STATUS_PROVISIONAL,
		knowledgetypes.FactStatus_FACT_STATUS_SUPERSEDED,
		knowledgetypes.FactStatus_FACT_STATUS_EXPIRED,
		knowledgetypes.FactStatus_FACT_STATUS_REVOKED,
		knowledgetypes.FactStatus_FACT_STATUS_AT_RISK,
		knowledgetypes.FactStatus_FACT_STATUS_PRUNED,
	}

	for _, s := range eligible {
		id := "F-OK-" + s.String()
		require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, base(id, s)))
		resp, err := qs.TrainingValueWeight(h.Ctx, &knowledgetypes.QueryTrainingValueWeightRequest{FactId: id})
		require.NoError(t, err)
		require.Greater(t, resp.TvwBps, uint64(0),
			"status %s must earn positive TVW", s.String())
		require.False(t, resp.StatusIneligible,
			"status %s must not be flagged ineligible", s.String())
	}

	for _, s := range ineligible {
		id := "F-NOPE-" + s.String()
		require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, base(id, s)))
		resp, err := qs.TrainingValueWeight(h.Ctx, &knowledgetypes.QueryTrainingValueWeightRequest{FactId: id})
		require.NoError(t, err)
		require.Equal(t, uint64(0), resp.TvwBps,
			"status %s must earn zero TVW", s.String())
		require.True(t, resp.StatusIneligible,
			"status %s must be flagged ineligible", s.String())
	}
}

// Self-reported ModelCard eval metrics are BPS-scaled in [0, 1_000_000].
// A value above 1M is nonsensical; a downstream trust-dashboard consuming
// it would render malformed model comparisons. Handler must bound.
func TestMoat_ModelCardEvalRangeBound(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	owner := testAddr("moat_eval_owner").String()
	require.NoError(t, h.KnowledgeKeeper.SetTrainingPipeline(h.Ctx, &knowledgetypes.TrainingPipeline{
		Id: "pipe-moat-eval", OperatorAddress: owner, TokenizerVersion: 1,
	}))

	_, err = ms.RegisterModelCard(h.Ctx, &knowledgetypes.MsgRegisterModelCard{
		Owner: owner, Id: "m-moat-bad-eval", PipelineId: "pipe-moat-eval",
		Route: "from_scratch", EvalAcceptanceRateBps: 2_000_000,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "eval_acceptance_rate_bps")

	_, err = ms.RegisterModelCard(h.Ctx, &knowledgetypes.MsgRegisterModelCard{
		Owner: owner, Id: "m-moat-bad-corr", PipelineId: "pipe-moat-eval",
		Route: "from_scratch", EvalCorroborationRateBps: 9_999_999,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "eval_corroboration_rate_bps")

	// Register a legitimate card first to test the update path.
	_, err = ms.RegisterModelCard(h.Ctx, &knowledgetypes.MsgRegisterModelCard{
		Owner: owner, Id: "m-moat-good", PipelineId: "pipe-moat-eval",
		Route: "from_scratch", EvalAcceptanceRateBps: 500_000,
	})
	require.NoError(t, err)

	_, err = ms.UpdateModelCard(h.Ctx, &knowledgetypes.MsgUpdateModelCard{
		Owner: owner, Id: "m-moat-good",
		EvalAcceptanceRateBps: 1_500_000,
	})
	require.Error(t, err, "out-of-range update must also reject")
}

// MsgAddFact bypasses the verifier-panel trust chain by design (genesis
// seeding and authority-gated corrections need it). Every call must emit
// a PrivilegedAction log entry so compromised-authority abuse is
// queryable via the uniform admin-action surface, not buried in the
// general block event stream.
func TestMoat_AddFactWritesPrivilegedLog(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	qs := knowledgekeeper.NewQueryServerImpl(h.KnowledgeKeeper)
	authority := h.KnowledgeKeeper.GetAuthority()

	resp, err := ms.AddFact(h.Ctx, &knowledgetypes.MsgAddFact{
		Authority:  authority,
		Content:    "moat: authority-injected fact",
		Domain:     "sciences",
		Category:   "empirical",
		Confidence: 950_000,
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.FactId)

	logs, err := qs.PrivilegedActions(h.Ctx, &knowledgetypes.QueryPrivilegedActionsRequest{
		Type: knowledgetypes.PrivilegedActionType_PRIVILEGED_ACTION_TYPE_FACT_AUTHORITY_INJECT,
	})
	require.NoError(t, err)
	require.Len(t, logs.Actions, 1,
		"authority-gated AddFact must emit a PrivilegedAction so compromise is queryable")
	require.Equal(t, resp.FactId, logs.Actions[0].Target)
	require.Equal(t, authority, logs.Actions[0].Invoker)
}
