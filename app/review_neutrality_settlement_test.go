package app

import (
	"fmt"
	"testing"

	"cosmossdk.io/core/header"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	"google.golang.org/protobuf/proto"
)

// Actual application bank, commitment handler and retained review records:
// valid dissent is payable work; admission balances are not task collateral.
func TestNeutralReviewActualBankAdmissionAndAtomicFeePool(t *testing.T) {
	for _, tc := range []struct {
		name    string
		votes   []string
		verdict knowledgetypes.Verdict
	}{
		{"dissent", []string{"reject", "reject", "accept"}, knowledgetypes.Verdict_VERDICT_REJECT},
		{"malformed", []string{"malformed", "malformed", "reject"}, knowledgetypes.Verdict_VERDICT_MALFORMED},
		{"inconclusive", []string{"accept", "reject", "malformed"}, knowledgetypes.Verdict_VERDICT_INCONCLUSIVE},
		{"under-quorum", []string{"accept", "reject", ""}, knowledgetypes.Verdict_VERDICT_INCONCLUSIVE},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := settlementFixture(t)
			ctx = ctx.WithChainID("neutral-bank-test").WithHeaderInfo(header.Info{Height: 100, ChainID: "neutral-bank-test"})
			require.NoError(t, app.KnowledgeKeeper.EnableReviewNeutrality(ctx))
			params, err := app.KnowledgeKeeper.GetParams(ctx)
			require.NoError(t, err)
			params.MinVerifiers, params.MinHeadcountAgreement = 3, 2
			params.CommitPhaseBlocks, params.RevealPhaseBlocks, params.AggregationPhaseBlocks = 2, 2, 2
			params.ConfidenceThreshold = 600000
			params.IndependenceRewardStrengthBps = 990000
			params.VerificationReward = "0"
			require.NoError(t, app.KnowledgeKeeper.SetParams(ctx, params))
			claim := &knowledgetypes.Claim{Id: "neutral-bank-claim", Submitter: settlementAddress(170).String(), Stake: "101", FactContent: "A local review fixture.", ClaimType: knowledgetypes.ClaimType_CLAIM_TYPE_ASSERTION, Domain: "physics", ReviewPolicyVersion: knowledgetypes.ReviewPolicyNeutral}
			require.NoError(t, app.KnowledgeKeeper.SetClaim(ctx, claim))
			round, err := app.KnowledgeKeeper.CreateVerificationRound(ctx, claim)
			require.NoError(t, err)
			require.Equal(t, knowledgetypes.ReviewPolicyNeutral, round.ReviewPolicyVersion)
			server := knowledgekeeper.NewMsgServerImpl(app.KnowledgeKeeper)
			att := &knowledgetypes.ReviewAttestation{Reason: "I checked the fixture and retained my own assessment."}
			reviewers := []sdk.AccAddress{settlementAddress(171), settlementAddress(172), settlementAddress(173)}
			for i, addr := range reviewers {
				vote := tc.votes[i]
				if vote == "" {
					vote = "accept"
				}
				salt := []byte(fmt.Sprintf("retained-long-salt-%d", i))
				hash, err := knowledgetypes.ComputeReviewCommitmentV2(round.CommitmentChainId, round.Id, addr.String(), vote, 800000, salt, att)
				require.NoError(t, err)
				_, err = server.SubmitCommitment(ctx, &knowledgetypes.MsgSubmitCommitment{Verifier: addr.String(), RoundId: round.Id, CommitHash: hash})
				require.ErrorContains(t, err, "minimum balance")
				fund := sdk.NewCoins(sdk.NewInt64Coin("uzrn", 100000000))
				require.NoError(t, app.BankKeeper.MintCoins(ctx, knowledgetypes.BootstrapFundModuleName, fund))
				require.NoError(t, app.BankKeeper.SendCoinsFromModuleToAccount(ctx, knowledgetypes.BootstrapFundModuleName, addr, fund))
				before := app.BankKeeper.GetBalance(ctx, addr, "uzrn")
				_, err = server.SubmitCommitment(ctx, &knowledgetypes.MsgSubmitCommitment{Verifier: addr.String(), RoundId: round.Id, CommitHash: hash})
				require.NoError(t, err, "no qualification or custom validator registration is required")
				require.Equal(t, before, app.BankKeeper.GetBalance(ctx, addr, "uzrn"), "commit admission locks no funds")
			}
			ctx = ctx.WithBlockHeight(102).WithHeaderInfo(header.Info{Height: 102, ChainID: ctx.ChainID()})
			require.NoError(t, app.KnowledgeKeeper.AdvanceRoundPhases(ctx))
			valid := 0
			for i, vote := range tc.votes {
				if vote == "" {
					continue
				}
				valid++
				_, err = server.SubmitReveal(ctx, &knowledgetypes.MsgSubmitReveal{Verifier: reviewers[i].String(), RoundId: round.Id, Vote: vote, Confidence: 800000, Salt: []byte(fmt.Sprintf("retained-long-salt-%d", i)), Attestation: att})
				require.NoError(t, err)
			}
			fundSettlementFixture(t, app, ctx, 40)
			ctx = ctx.WithBlockHeight(106).WithHeaderInfo(header.Info{Height: 106, ChainID: ctx.ChainID()})
			beforeSupply := app.BankKeeper.GetSupply(ctx, "uzrn")
			require.NoError(t, app.KnowledgeKeeper.AdvanceRoundPhases(ctx))
			stored, found := app.KnowledgeKeeper.GetVerificationRound(ctx, round.Id)
			require.True(t, found)
			require.Equal(t, tc.verdict, stored.Verdict)
			require.Equal(t, knowledgetypes.VerificationPhase_VERIFICATION_PHASE_COMPLETE, stored.Phase)
			plan := stored.VerifierRewardSettlement
			require.NotNil(t, plan)
			require.Len(t, plan.Payments, valid)
			require.Zero(t, plan.PaidAtBlock)
			require.Equal(t, "0", plan.WithheldTotal)
			for _, addr := range reviewers {
				require.Equal(t, "100000000", app.BankKeeper.GetBalance(ctx, addr, "uzrn").Amount.String())
			}
			require.Equal(t, beforeSupply, app.BankKeeper.GetSupply(ctx, "uzrn"))
			frozen := proto.Clone(plan).(*knowledgetypes.VerifierRewardSettlement)
			fundSettlementFixture(t, app, ctx, 15)
			beforeSupply = app.BankKeeper.GetSupply(ctx, "uzrn")
			ctx = ctx.WithBlockHeight(107).WithHeaderInfo(header.Info{Height: 107, ChainID: ctx.ChainID()})
			require.NoError(t, app.KnowledgeKeeper.ProcessPendingVerifierRewards(ctx))
			stored, _ = app.KnowledgeKeeper.GetVerificationRound(ctx, round.Id)
			frozen.PaidAtBlock = 107
			require.True(t, proto.Equal(frozen, stored.VerifierRewardSettlement))
			var paid int64
			for i, addr := range reviewers {
				delta := app.BankKeeper.GetBalance(ctx, addr, "uzrn").Amount.Int64() - 100000000
				if tc.votes[i] == "" {
					require.Zero(t, delta)
				} else {
					require.Positive(t, delta)
				}
				paid += delta
			}
			require.Equal(t, int64(55), paid)
			require.Equal(t, beforeSupply, app.BankKeeper.GetSupply(ctx, "uzrn"))
			balancesBeforeRetry := make([]sdk.Coin, len(reviewers))
			for i, addr := range reviewers {
				balancesBeforeRetry[i] = app.BankKeeper.GetBalance(ctx, addr, "uzrn")
			}
			require.NoError(t, app.KnowledgeKeeper.CompleteRound(ctx, stored, &knowledgekeeper.VerificationResult{Verdict: tc.verdict}))
			require.NoError(t, app.KnowledgeKeeper.ProcessPendingVerifierRewards(ctx))
			after, _ := app.KnowledgeKeeper.GetVerificationRound(ctx, round.Id)
			require.True(t, proto.Equal(stored, after))
			require.Equal(t, beforeSupply, app.BankKeeper.GetSupply(ctx, "uzrn"))
			for i, addr := range reviewers {
				require.Equal(t, balancesBeforeRetry[i], app.BankKeeper.GetBalance(ctx, addr, "uzrn"))
			}
		})
	}
}

func TestNeutralReviewPaysExactAccruedLegacyPlanAfterActivation(t *testing.T) {
	app, ctx := settlementFixture(t)
	ctx.KVStore(app.keys[knowledgetypes.StoreKey]).Delete([]byte(knowledgekeeper.FundSettlementEnabledStoreKey))
	ctx.KVStore(app.keys[knowledgetypes.StoreKey]).Delete([]byte(knowledgekeeper.ReviewNeutralityEnabledStoreKey))
	recipient := settlementAddress(181)
	round := &knowledgetypes.VerificationRound{Id: "accrued-before-neutrality", ClaimId: "legacy-unavailable-claim", Phase: knowledgetypes.VerificationPhase_VERIFICATION_PHASE_COMPLETE, Verdict: knowledgetypes.Verdict_VERDICT_REJECT, VerdictBlock: 100, VerifierRewardSettlement: &knowledgetypes.VerifierRewardSettlement{CreatedAtBlock: 100, WithheldTotal: "1", Payments: []*knowledgetypes.VerifierRewardPayment{{Verifier: recipient.String(), Amount: "13", Withheld: "1"}}}}
	require.NoError(t, app.KnowledgeKeeper.SetVerificationRound(ctx, round))
	frozen := proto.Clone(round.VerifierRewardSettlement).(*knowledgetypes.VerifierRewardSettlement)
	require.NoError(t, app.KnowledgeKeeper.EnableReviewNeutrality(ctx))
	fundSettlementFixture(t, app, ctx, 14)
	beforeSupply := app.BankKeeper.GetSupply(ctx, "uzrn")
	require.NoError(t, app.KnowledgeKeeper.ProcessPendingVerifierRewards(ctx))
	stored, found := app.KnowledgeKeeper.GetVerificationRound(ctx, round.Id)
	require.True(t, found)
	require.Zero(t, stored.ReviewPolicyVersion)
	frozen.PaidAtBlock = 100
	require.True(t, proto.Equal(frozen, stored.VerifierRewardSettlement))
	require.Equal(t, "13", app.BankKeeper.GetBalance(ctx, recipient, "uzrn").Amount.String())
	require.Equal(t, beforeSupply, app.BankKeeper.GetSupply(ctx, "uzrn"))
	require.NoError(t, app.KnowledgeKeeper.ProcessPendingVerifierRewards(ctx))
	require.Equal(t, "13", app.BankKeeper.GetBalance(ctx, recipient, "uzrn").Amount.String())
}
