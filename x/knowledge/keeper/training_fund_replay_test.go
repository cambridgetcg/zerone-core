package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// The training-fund replay guard. MsgClaimTrainingFundDisbursement mints up
// to 2,000 ZRN per message with no rate limit, no per-model cap and no
// cooldown; its ENTIRE replay guard was msg.Id, documented in the proto as
// "client-chosen". A payout key the payee picks is not a key — pick a fresh
// one and mint again, until the supply cap.
//
// The guard is now (model_id, disbursement epoch). These tests pin that the
// key is derived, not supplied.

const (
	tfPipelineID = "pipe-1"
	tfModelID    = "model-1"
)

// Computed lazily, not as package vars: package-level initialisation runs
// before the init() that installs the "zrn" bech32 prefix.
func tfOperator() string   { return makeValidBech32Addr("tf-operator") }
func tfDeployAddr() string { return makeValidBech32Addr("tf-deployment") }

func setupTrainingFundClaim(t *testing.T) (types.MsgServer, keeper.Keeper, sdk.Context) {
	t.Helper()
	k, ctx := setupKnowledgeTest(t)

	if err := k.SetTrainingPipeline(ctx, &types.TrainingPipeline{
		Id: tfPipelineID, OperatorAddress: tfOperator(), Status: "COMPLETED",
	}); err != nil {
		t.Fatalf("SetTrainingPipeline: %v", err)
	}
	if err := k.SetModelCard(ctx, &types.ModelCard{
		Id: tfModelID, PipelineId: tfPipelineID, DeploymentAddress: tfDeployAddr(), Active: true,
	}); err != nil {
		t.Fatalf("SetModelCard: %v", err)
	}
	// Calibration comfortably over the 50% floor.
	if err := k.SetAgentCalibration(ctx, &types.AgentCalibration{
		Address: tfDeployAddr(), CalibrationScoreBps: 900_000,
	}); err != nil {
		t.Fatalf("SetAgentCalibration: %v", err)
	}
	return keeper.NewMsgServerImpl(k), k, ctx
}

// THE FARM: same model, different caller-chosen ids, in the same epoch.
// Before the fix every one of these minted afresh.
func TestClaimTrainingFund_FreshIdCannotReplayWithinEpoch(t *testing.T) {
	ms, _, ctx := setupTrainingFundClaim(t)

	first, err := ms.ClaimTrainingFundDisbursement(ctx, &types.MsgClaimTrainingFundDisbursement{
		Claimant: tfOperator(), ModelId: tfModelID, Id: "whatever-i-like-1",
	})
	if err != nil {
		t.Fatalf("first claim failed: %v", err)
	}
	if first.TotalAmount == "" || first.TotalAmount == "0" {
		t.Fatalf("first claim minted nothing: %q", first.TotalAmount)
	}

	for _, id := range []string{"whatever-i-like-2", "whatever-i-like-3", ""} {
		if id == "" {
			continue // empty id is rejected by the arg check, not the replay guard
		}
		_, err := ms.ClaimTrainingFundDisbursement(ctx, &types.MsgClaimTrainingFundDisbursement{
			Claimant: tfOperator(), ModelId: tfModelID, Id: id,
		})
		if err == nil {
			t.Fatalf("replay with a fresh client-chosen id %q minted again — "+
				"the payout key is still caller-controlled", id)
		}
	}
}

// The bound is per model per epoch, so a later epoch may claim again. This is
// the intended behaviour, not a hole: it converts an unbounded faucet into a
// rate, which is what makes the lane finite.
func TestClaimTrainingFund_NextEpochMayClaimAgain(t *testing.T) {
	ms, k, ctx := setupTrainingFundClaim(t)

	if _, err := ms.ClaimTrainingFundDisbursement(ctx, &types.MsgClaimTrainingFundDisbursement{
		Claimant: tfOperator(), ModelId: tfModelID, Id: "a",
	}); err != nil {
		t.Fatalf("first claim failed: %v", err)
	}

	params, _ := k.GetParams(ctx)
	epochBlocks := params.FitnessEpochBlocks
	if epochBlocks == 0 {
		epochBlocks = 10_000
	}
	next := ctx.WithBlockHeight(ctx.BlockHeight() + int64(epochBlocks))

	if _, err := ms.ClaimTrainingFundDisbursement(next, &types.MsgClaimTrainingFundDisbursement{
		Claimant: tfOperator(), ModelId: tfModelID, Id: "b",
	}); err != nil {
		t.Fatalf("claim in the next epoch was refused: %v", err)
	}
}

// Disarming must not depend on a string staying non-empty. An unset base
// reward means the lane pays nothing — it must NOT silently fall back to the
// 1,000 ZRN default, or any migration that normalises the field re-arms the
// faucet with no code change and no governance vote.
func TestClaimTrainingFund_EmptyBaseRewardPaysNothing(t *testing.T) {
	ms, k, ctx := setupTrainingFundClaim(t)

	params, _ := k.GetParams(ctx)
	params.TrainingFundBaseReward = ""
	if err := k.SetParams(ctx, params); err != nil {
		t.Fatalf("SetParams: %v", err)
	}

	if _, err := ms.ClaimTrainingFundDisbursement(ctx, &types.MsgClaimTrainingFundDisbursement{
		Claimant: tfOperator(), ModelId: tfModelID, Id: "x",
	}); err == nil {
		t.Fatal("an empty TrainingFundBaseReward still minted — the implicit " +
			"1,000 ZRN default makes the param-only mitigation reversible")
	}
}
