package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// A keeper quote is an observation, not a reservation of the admission price.
// This uses direct in-memory handlers and a tracking bank, not signed transport,
// actual transfers, or evidence of an earlier CI run's quote height or timing.
func TestSubmitClaimRejectsStaleReviewFeeAfterPacingChange(t *testing.T) {
	for _, tc := range []struct {
		name          string
		creationBps   uint64
		admissionFee  string
		wantAdmission bool
	}{
		{"unchanged_pacing_admits_quote", 1_000_000, "100000", true},
		{"lower_pacing_refuses_stale_quote", 500_000, "200000", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Reuse the synchronous-IAVL fixture without changing genesis flags,
			// fee parameters, cooldowns, or any runtime activation.
			k, ctx, bank, key := relationContractFixture(t, types.DefaultGenesis())
			pacing := &mockPacingKeeper{creationBps: 1_000_000, analysisBps: 1_000_000}
			k.SetPacingKeeper(pacing)
			server := keeper.NewMsgServerImpl(k)
			params, err := k.GetParams(ctx)
			require.NoError(t, err)
			quote := k.GetEffectiveMinReviewFee(ctx)
			require.Equal(t, "100000", quote)
			require.Equal(t, params.MinReviewFee, quote)
			msg := &types.MsgSubmitClaim{
				Submitter:   makeValidBech32Addr("fee-quote-author"),
				FactContent: "A bounded fixture observation tests review-fee admission.",
				Domain:      "physics",
				Category:    "empirical",
				Stake:       quote,
			}

			// Only the injected creation pacing changes after quoting. The
			// offered fee and fixture height remain fixed; no top-up or retry.
			pacing.creationBps = tc.creationBps
			require.Equal(t, tc.admissionFee, k.GetEffectiveMinReviewFee(ctx))
			currentParams, err := k.GetParams(ctx)
			require.NoError(t, err)
			require.Equal(t, params, currentParams)
			contentHash := keeper.ComputeClaimContentHash(msg.FactContent, msg.Domain)
			claimID := keeper.GenerateClaimID(msg.Submitter, contentHash, uint64(ctx.BlockHeight()))
			before := relationContractSnapshot(t, ctx, key)
			sendsBefore := len(bank.sendCalls)
			eventsBefore := len(ctx.EventManager().Events())
			require.Zero(t, k.GetLastClaimHeight(ctx, msg.Submitter))

			response, err := server.SubmitClaim(ctx, msg)
			t.Logf("fixture height=%d quoted_pacing=1000000 admission_pacing=%d offered=%suzrn minimum=%suzrn error=%v",
				ctx.BlockHeight(), tc.creationBps, msg.Stake, tc.admissionFee, err)
			if tc.wantAdmission {
				require.NoError(t, err)
				require.NotNil(t, response)
				require.Equal(t, claimID, response.ClaimId)
				claim, found := k.GetClaim(ctx, claimID)
				require.True(t, found)
				require.Equal(t, quote, claim.Stake)
				require.Equal(t, types.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION, claim.Status)
				round, found := k.GetRoundByClaimID(ctx, claimID)
				require.True(t, found)
				require.Equal(t, claim.VerificationRoundId, round.Id)
				require.Equal(t, uint64(ctx.BlockHeight()), k.GetLastClaimHeight(ctx, msg.Submitter))
				require.Greater(t, len(bank.sendCalls), sendsBefore)
				require.Equal(t, sendRecord{
					from: msg.Submitter, to: types.ModuleName,
					coins: sdk.NewCoins(sdk.NewInt64Coin("uzrn", 100000)),
				}, bank.sendCalls[sendsBefore], "tracking-bank call, not proof of an actual payment")
				return
			}

			// Nonfatal assertions let a disposable fee-check bypass expose both
			// the missing refusal and any admission effects in the same run.
			assert.ErrorContains(t, err, "review fee 100000uzrn below minimum 200000uzrn")
			assert.Nil(t, response, "a stale quote must not receive an admission response")
			_, claimFound := k.GetClaim(ctx, claimID)
			assert.False(t, claimFound, "stale quote must not store a claim")
			_, roundFound := k.GetRoundByClaimID(ctx, claimID)
			assert.False(t, roundFound, "stale quote must not create a verification round")
			_, indexed := k.GetClaimByContentHash(ctx, contentHash)
			assert.False(t, indexed, "stale quote must not reserve the content hash")
			assert.Zero(t, k.GetLastClaimHeight(ctx, msg.Submitter), "refusal must not consume the submitter cooldown")
			assert.Len(t, bank.sendCalls, sendsBefore, "refusal must precede bank calls")
			assert.Len(t, ctx.EventManager().Events(), eventsBefore, "refusal must not emit admission events")
			assert.Equal(t, before, relationContractSnapshot(t, ctx, key), "refusal must leave the knowledge store unchanged")
		})
	}
}
