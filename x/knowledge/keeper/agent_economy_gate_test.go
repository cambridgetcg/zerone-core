package keeper_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func seedAgentEconomyGateMarkers(
	t *testing.T,
	k keeper.Keeper,
	ctx context.Context,
	markers map[string]string,
) {
	t.Helper()
	for key, value := range markers {
		require.NoError(t, k.WriteMigrationMarker(ctx, key, value))
	}
}

func TestAgentEconomyGate_SubmitComputationalClaimRequiresExactLineage(t *testing.T) {
	tests := []struct {
		name       string
		markers    map[string]string
		wantDetail string
	}{
		{
			name:       "marker absent",
			wantDetail: "agent economy is not activated",
		},
		{
			name: "marker value is not exact",
			markers: map[string]string{
				types.AgentEconomyUpgradeMarker: "wrong-release",
			},
			wantDetail: "has value \"wrong-release\"",
		},
		{
			name: "upgrade and native lineages conflict",
			markers: map[string]string{
				types.AgentEconomyUpgradeMarker: types.AgentEconomyActivationValue,
				types.AgentEconomyNativeMarker:  types.AgentEconomyActivationValue,
			},
			wantDetail: "conflicting agent-economy lineages",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			k, ctx, bank := setupKnowledgeTestWithBank(t)
			seedAgentEconomyGateMarkers(t, k, ctx, test.markers)
			server := keeper.NewMsgServerImpl(k)
			bankCallsBefore := len(bank.sendCalls)

			response, err := server.SubmitClaim(
				ctx,
				computationalSubmitMsg(
					makeValidBech32Addr("sealed-compute-claim"),
					"5",
					"6",
				),
			)

			require.Nil(t, response)
			require.ErrorIs(t, err, types.ErrAgentEconomyDisabled)
			require.ErrorContains(t, err, test.wantDetail)
			require.Len(t, bank.sendCalls, bankCallsBefore,
				"a lineage refusal must happen before any fee movement")
			claimCount := 0
			k.IterateClaims(ctx, func(*types.Claim) bool {
				claimCount++
				return false
			})
			require.Zero(t, claimCount,
				"a lineage refusal must not create a claim or verification round")
		})
	}
}

func TestAgentEconomyGate_CompleteComputationalRoundRequiresExactLineage(t *testing.T) {
	tests := []struct {
		name       string
		markers    map[string]string
		wantDetail string
	}{
		{
			name:       "marker absent",
			wantDetail: "agent economy is not activated",
		},
		{
			name: "marker value is not exact",
			markers: map[string]string{
				types.AgentEconomyNativeMarker: "wrong-release",
			},
			wantDetail: "has value \"wrong-release\"",
		},
		{
			name: "upgrade and native lineages conflict",
			markers: map[string]string{
				types.AgentEconomyUpgradeMarker: types.AgentEconomyActivationValue,
				types.AgentEconomyNativeMarker:  types.AgentEconomyActivationValue,
			},
			wantDetail: "conflicting agent-economy lineages",
		},
	}

	for _, test := range tests {
		for _, commitmentCase := range []struct {
			name       string
			commitment *types.ComputationalCommitment
		}{
			{name: "bound", commitment: &types.ComputationalCommitment{}},
			{name: "legacy unbound"},
		} {
			t.Run(test.name+"/"+commitmentCase.name, func(t *testing.T) {
				k, ctx, bank := setupKnowledgeTestWithBank(t)
				seedAgentEconomyGateMarkers(t, k, ctx, test.markers)
				claim := &types.Claim{
					Id:                      "sealed-computational-claim",
					FactContent:             "A computational claim waiting for the reviewed release",
					Domain:                  "physics",
					Category:                "computational",
					Submitter:               makeValidBech32Addr("sealed-compute-worker"),
					Status:                  types.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION,
					ClaimType:               types.ClaimType_CLAIM_TYPE_COMPUTATIONAL,
					ComputationalCommitment: commitmentCase.commitment,
				}
				round := makeRoundInPhase(
					"sealed-computational-round",
					claim.Id,
					types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION,
					uint64(ctx.BlockHeight()-1),
				)
				require.NoError(t, k.SetClaim(ctx, claim))
				require.NoError(t, k.SetVerificationRound(ctx, round))
				bankCallsBefore := len(bank.sendCalls)
				eventsBefore := len(ctx.EventManager().Events())
				factsBefore := 0
				k.IterateFacts(ctx, func(*types.Fact) bool {
					factsBefore++
					return false
				})

				err := k.CompleteRound(ctx, round, &keeper.VerificationResult{
					Verdict:    types.Verdict_VERDICT_ACCEPT,
					Confidence: 900_000,
				})

				require.ErrorIs(t, err, types.ErrAgentEconomyDisabled)
				require.ErrorContains(t, err, test.wantDetail)
				require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION, round.Phase)
				require.Equal(t, types.Verdict_VERDICT_UNSPECIFIED, round.Verdict)
				storedRound, found := k.GetVerificationRound(ctx, round.Id)
				require.True(t, found)
				require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION, storedRound.Phase)
				storedClaim, found := k.GetClaim(ctx, claim.Id)
				require.True(t, found)
				require.Equal(t, types.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION, storedClaim.Status)
				require.Len(t, bank.sendCalls, bankCallsBefore)
				require.Len(t, ctx.EventManager().Events(), eventsBefore)
				factCount := 0
				k.IterateFacts(ctx, func(*types.Fact) bool {
					factCount++
					return false
				})
				require.Equal(t, factsBefore, factCount,
					"a sealed acceptance must not materialize a fact")
			})
		}
	}
}

func TestAgentEconomyGate_PreservesNonComputationalClaimLifecycle(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)
	active, err := k.AgentEconomyActivated(ctx)
	require.NoError(t, err)
	require.False(t, active, "fixture must have no agent-economy lineage")

	server := keeper.NewMsgServerImpl(k)
	response, err := server.SubmitClaim(ctx, &types.MsgSubmitClaim{
		Submitter:   makeValidBech32Addr("ordinary-unsealed-claim"),
		FactContent: "An ordinary empirical claim remains open before agent economy activation",
		Domain:      "physics",
		Category:    "empirical",
		Stake:       "1000000",
		ClaimType:   types.ClaimType_CLAIM_TYPE_ASSERTION,
	})
	require.NoError(t, err)
	claim, found := k.GetClaim(ctx, response.ClaimId)
	require.True(t, found)
	round, found := k.GetVerificationRound(ctx, claim.VerificationRoundId)
	require.True(t, found)

	require.NoError(t, k.CompleteRound(ctx, round, &keeper.VerificationResult{
		Verdict:    types.Verdict_VERDICT_ACCEPT,
		Confidence: 900_000,
	}))
	claim, found = k.GetClaim(ctx, response.ClaimId)
	require.True(t, found)
	require.Equal(t, types.ClaimStatus_CLAIM_STATUS_ACCEPTED, claim.Status)
}
