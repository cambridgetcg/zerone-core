package keeper_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/gov/keeper"
	"github.com/zerone-chain/zerone/x/gov/types"
)

type statefulParamRouter func(context.Context, string, string, string) error

func (f statefulParamRouter) ApplyParamChange(ctx context.Context, module, key, value string) error {
	return f(ctx, module, key, value)
}

func approvedLIP(k keeper.Keeper, ctx sdk.Context, category string) *types.LIP {
	lip := &types.LIP{Id: "LIP-1", Proposer: testAddr("alice"), Category: category,
		Stage: types.StatusVoting, StakedAmount: "1000000", YesStake: "500000",
		NoStake: "0", AbstainStake: "0", UniqueVoters: 1, VotingEndBlock: uint64(ctx.BlockHeight())}
	k.SetLIP(ctx, lip)
	return lip
}

func TestAtomicLIPExecution_CommitsAllOrDiscardsTargetWritesAndEvents(t *testing.T) {
	for _, mode := range []string{"success", "error", "panic"} {
		t.Run(mode, func(t *testing.T) {
			k, ctx, _ := setupWithStaking(t, "1000000")
			require.NoError(t, k.EnableAccountingSafety(ctx))
			before := k.GetParams(ctx)
			calls := 0
			k.SetParamRouter(statefulParamRouter(func(goCtx context.Context, _, key, _ string) error {
				calls++
				executionCtx := sdk.UnwrapSDKContext(goCtx)
				params := k.GetParams(executionCtx)
				params.VotingPeriodBlocks += 10
				k.SetParams(executionCtx, params)
				executionCtx.EventManager().EmitEvent(sdk.NewEvent("target_parameter_written"))
				if key == "second" {
					if mode == "error" {
						return fmt.Errorf("diagnostic error after writing")
					}
					if mode == "panic" {
						panic("diagnostic panic after writing")
					}
				}
				return nil
			}))
			lip := approvedLIP(k, ctx, types.CategoryParameter)
			lip.ParamChanges = []*types.ParamChange{
				{Module: "target", Key: "first", Value: "1"},
				{Module: "target", Key: "second", Value: "2"},
				{Module: "target", Key: "third", Value: "3"},
			}
			k.SetLIP(ctx, lip)
			k.BeginBlocker(ctx)
			resolved, found := k.GetLIP(ctx, lip.Id)
			require.True(t, found)
			require.Equal(t, lip.StakedAmount, resolved.StakedAmount, "resolution cannot infer an escrow recipient")
			applied, target, failures := 0, 0, 0
			for _, event := range ctx.EventManager().Events() {
				switch event.Type {
				case "zerone.gov.param_change_applied":
					applied++
				case "target_parameter_written":
					target++
				case "zerone.gov.lip_execution_failed":
					failures++
				case "zerone.gov.lip_tallied":
					for _, attr := range event.Attributes {
						if attr.Key == "outcome" {
							require.Equal(t, resolved.Stage, attr.Value)
						}
					}
				}
			}
			if mode == "success" {
				require.Equal(t, types.StatusPassed, resolved.Stage)
				require.Empty(t, resolved.ExecutionError)
				require.Equal(t, before.VotingPeriodBlocks+30, k.GetParams(ctx).VotingPeriodBlocks)
				require.Equal(t, 3, calls)
				require.Equal(t, 3, applied)
				require.Equal(t, 3, target)
				require.Zero(t, failures)
			} else {
				require.Equal(t, types.StatusFailed, resolved.Stage)
				expectedCode := "parameter_dispatch_failed"
				if mode == "panic" {
					expectedCode = "execution_panicked"
				}
				require.Equal(t, expectedCode, resolved.ExecutionError)
				require.True(t, proto.Equal(before, k.GetParams(ctx)), "all target writes must roll back, including the failing handler's")
				require.Equal(t, 2, calls, "later handlers must never run")
				require.Zero(t, applied, "discard success events from partial execution")
				require.Zero(t, target, "discard the failed handler's own events")
				require.Equal(t, 1, failures)
			}
			// A later BeginBlock cannot retry terminal execution or replay side effects.
			k.BeginBlocker(ctx.WithBlockHeight(ctx.BlockHeight() + 1))
			if mode == "success" {
				require.Equal(t, 3, calls)
			} else {
				require.Equal(t, 2, calls)
			}

			// Failure reason, stake and terminal decision survive ordinary genesis
			// serialization. The full app enables the exported profile after import.
			export := k.ExportGenesis(ctx)
			require.True(t, export.AccountingSafetyEnabled)
			encoded, err := json.Marshal(export)
			require.NoError(t, err)
			var imported types.GenesisState
			require.NoError(t, json.Unmarshal(encoded, &imported))
			other, otherCtx, _ := setupWithStaking(t, "1000000")
			other.InitGenesis(otherCtx, &imported)
			require.False(t, other.AccountingSafetyEnabled(otherCtx), "module init alone cannot activate the profile")
			require.NoError(t, other.EnableAccountingSafety(otherCtx))
			other.BeginBlocker(otherCtx.WithBlockHeight(ctx.BlockHeight() + 2))
			roundTrip, found := other.GetLIP(otherCtx, lip.Id)
			require.True(t, found)
			require.True(t, proto.Equal(resolved, roundTrip))
		})
	}
}

func TestAtomicLIPExecution_MissingAndUnimplementedDispatchFailClosed(t *testing.T) {
	cases := []struct {
		name, category, code string
		changes              []*types.ParamChange
	}{
		{"missing parameter payload", types.CategoryParameter, "parameter_changes_missing", nil},
		{"missing parameter router", types.CategoryParameter, "parameter_dispatch_failed", []*types.ParamChange{{Module: "target", Key: "field", Value: "1"}}},
		{"nil parameter change", types.CategoryParameter, "parameter_change_malformed", []*types.ParamChange{nil}},
		{"missing creed payload", types.CategoryCreedAmendment, "creed_payload_missing", nil},
		{"unimplemented adapter", types.CategoryAdapterRegistration, "adapter_dispatch_unimplemented", nil},
		{"generic research spend", types.CategoryResearchSpend, "category_dispatch_unimplemented", nil},
		{"generic seat election", types.CategorySeatElection, "category_dispatch_unimplemented", nil},
		{"unknown category", "unrecognized", "category_unsupported", nil},
		{"missing phase metadata", types.CategoryPhaseTransition, "phase_approval_failed", nil},
		{"retired custom upgrade", types.CategoryUpgrade, "custom_upgrade_authority_retired", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			k, ctx, _ := setupWithStaking(t, "1000000")
			require.NoError(t, k.EnableAccountingSafety(ctx))
			lip := approvedLIP(k, ctx, tc.category)
			lip.ParamChanges = tc.changes
			k.SetLIP(ctx, lip)
			k.BeginBlocker(ctx)
			got, _ := k.GetLIP(ctx, lip.Id)
			require.Equal(t, types.StatusFailed, got.Stage)
			require.Equal(t, tc.code, got.ExecutionError)
			require.Equal(t, lip.StakedAmount, got.StakedAmount)
		})
	}
}

func TestAtomicLIPExecution_TextIsAdvisoryAndPhaseIsPending(t *testing.T) {
	k, ctx, _ := setupWithStaking(t, "1000000")
	require.NoError(t, k.EnableAccountingSafety(ctx))
	text := approvedLIP(k, ctx, types.CategoryText)
	k.BeginBlocker(ctx)
	got, _ := k.GetLIP(ctx, text.Id)
	require.Equal(t, types.StatusPassed, got.Stage)
	require.Empty(t, got.ExecutionError)

	phase := approvedLIP(k, ctx, types.CategoryPhaseTransition)
	original := k.GetResearchFundGovernanceState(ctx)
	k.SetPhaseTransitionMeta(ctx, &types.PhaseTransitionProposal{
		LipID: phase.Id, TargetPhase: original.CurrentPhase + 1, Stage: types.PhaseTransitionStagePending,
	})
	k.BeginBlocker(ctx)
	got, _ = k.GetLIP(ctx, phase.Id)
	require.Equal(t, types.StatusPassed, got.Stage)
	meta, found := k.GetPhaseTransitionMeta(ctx, phase.Id)
	require.True(t, found)
	require.Equal(t, types.PhaseTransitionStagePending, meta.Stage)
	require.Equal(t, uint64(ctx.BlockHeight())+types.TransitionActivationDelay, meta.ActivationBlock)
	require.Equal(t, original.CurrentPhase, k.GetResearchFundGovernanceState(ctx).CurrentPhase)
}

func TestAccountingSafety_PreservesHistoricalTerminalLIPsAndLegacyBranch(t *testing.T) {
	k, ctx, _ := setupWithStaking(t, "1000000")
	require.False(t, k.AccountingSafetyEnabled(ctx))
	require.Error(t, k.ValidateAccountingSafety(ctx))
	lip := approvedLIP(k, ctx, types.CategoryAdapterRegistration)
	k.BeginBlocker(ctx)
	old, _ := k.GetLIP(ctx, lip.Id)
	require.Equal(t, types.StatusPassed, old.Stage, "unmarked state retains pre-upgrade consensus semantics")
	oldBytes, err := json.Marshal(old)
	require.NoError(t, err)
	require.NoError(t, k.EnableAccountingSafety(ctx))
	require.NoError(t, k.EnableAccountingSafety(ctx), "activation is idempotent")
	require.NoError(t, k.ValidateAccountingSafety(ctx))
	k.BeginBlocker(ctx.WithBlockHeight(ctx.BlockHeight() + 1))
	preserved, _ := k.GetLIP(ctx, lip.Id)
	preservedBytes, err := json.Marshal(preserved)
	require.NoError(t, err)
	require.Equal(t, oldBytes, preservedBytes, "historical passed-but-inert record is not rewritten or replayed")
}

type statefulCreedKeeper struct {
	anchor func(context.Context, string, []byte, []byte) error
}

func (c statefulCreedKeeper) AnchorPinFromBytes(ctx context.Context, lipID string, hash, payload []byte) error {
	return c.anchor(ctx, lipID, hash, payload)
}
func (statefulCreedKeeper) IsActiveCouncilMember(context.Context, string) bool { return false }

func TestAtomicLIPExecution_CreedRequiresActualSuccessfulAnchoring(t *testing.T) {
	for _, mode := range []string{"missing keeper", "error", "success"} {
		t.Run(mode, func(t *testing.T) {
			k, ctx, _ := setupWithStaking(t, "1000000")
			require.NoError(t, k.EnableAccountingSafety(ctx))
			lip := approvedLIP(k, ctx, types.CategoryCreedAmendment)
			k.SetCreedAmendmentPin(ctx, lip.Id, &keeper.CreedAmendmentPin{CanonicalHash: []byte("hash"), CommitmentsJSON: []byte("payload")})
			original := k.GetNextLIPNumber(ctx)
			if mode != "missing keeper" {
				k.SetCreedKeeper(statefulCreedKeeper{anchor: func(goCtx context.Context, id string, hash, payload []byte) error {
					require.Equal(t, lip.Id, id)
					require.Equal(t, []byte("hash"), hash)
					require.Equal(t, []byte("payload"), payload)
					targetCtx := sdk.UnwrapSDKContext(goCtx)
					k.SetNextLIPNumber(targetCtx, 9000)
					targetCtx.EventManager().EmitEvent(sdk.NewEvent("target_creed_written"))
					if mode == "error" {
						return fmt.Errorf("rejected after write")
					}
					return nil
				}})
			}
			k.BeginBlocker(ctx)
			got, _ := k.GetLIP(ctx, lip.Id)
			anchored := 0
			for _, event := range ctx.EventManager().Events() {
				if event.Type == "zerone.gov.creed_amendment_anchored" || event.Type == "target_creed_written" {
					anchored++
				}
			}
			if mode == "success" {
				require.Equal(t, types.StatusPassed, got.Stage)
				require.Equal(t, uint64(9000), k.GetNextLIPNumber(ctx))
				require.Equal(t, 2, anchored)
			} else {
				require.Equal(t, types.StatusFailed, got.Stage)
				code := "creed_anchor_failed"
				if mode == "missing keeper" {
					code = "creed_keeper_missing"
				}
				require.Equal(t, code, got.ExecutionError)
				require.Equal(t, original, k.GetNextLIPNumber(ctx))
				require.Zero(t, anchored)
			}
		})
	}
}

func TestAtomicLIPExecution_DoesNotOverwriteTerminalPhaseTarget(t *testing.T) {
	k, ctx, _ := setupWithStaking(t, "1000000")
	require.NoError(t, k.EnableAccountingSafety(ctx))
	lip := approvedLIP(k, ctx, types.CategoryPhaseTransition)
	prior := &types.PhaseTransitionProposal{LipID: lip.Id, TargetPhase: types.ResearchFundPhase_RESEARCH_FUND_PHASE_OBSERVER,
		Stage: types.PhaseTransitionStageActivated, ActivationBlock: 50}
	k.SetPhaseTransitionMeta(ctx, prior)
	k.BeginBlocker(ctx)
	got, _ := k.GetLIP(ctx, lip.Id)
	require.Equal(t, types.StatusFailed, got.Stage)
	require.Equal(t, "phase_approval_failed", got.ExecutionError)
	preserved, found := k.GetPhaseTransitionMeta(ctx, lip.Id)
	require.True(t, found)
	require.Equal(t, prior, preserved)
}
