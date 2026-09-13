package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"testing"

	"cosmossdk.io/core/header"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// These tests use the application's actual SDK bank/account keepers and
// multistore caches. Funding is explicit test issuance, not earned science.
func settlementFixture(t *testing.T) (*ZeroneApp, sdk.Context) {
	t.Helper()
	app, ctx, _ := newAccountingAuthorityFixture(t)
	ctx = ctx.WithBlockHeight(100).WithHeaderInfo(header.Info{Height: 100, ChainID: ctx.ChainID()})
	require.NoError(t, app.KnowledgeKeeper.EnableRecordIntegrity(ctx))
	params, err := app.KnowledgeKeeper.GetParams(ctx)
	require.NoError(t, err)
	params.IndependenceRewardStrengthBps = 0
	require.NoError(t, app.KnowledgeKeeper.SetParams(ctx, params))
	return app, ctx
}

func fundSettlementFixture(t *testing.T, app *ZeroneApp, ctx sdk.Context, amount int64) {
	t.Helper()
	coins := sdk.NewCoins(sdk.NewInt64Coin("uzrn", amount))
	require.NoError(t, app.BankKeeper.MintCoins(ctx, knowledgetypes.BootstrapFundModuleName, coins))
	require.NoError(t, app.BankKeeper.SendCoinsFromModuleToModule(ctx, knowledgetypes.BootstrapFundModuleName, knowledgetypes.ModuleName, coins))
}

func settlementAddress(n byte) sdk.AccAddress { return sdk.AccAddress(bytes.Repeat([]byte{n}, 20)) }

func TestRecordIntegrityVerifierPaymentFailurePreservesVerdictAndObligation(t *testing.T) {
	app, ctx := settlementFixture(t)
	a, b := settlementAddress(61), settlementAddress(62)
	claim := &knowledgetypes.Claim{Id: "payment-claim", Submitter: settlementAddress(60).String(), Stake: "100", FactContent: "A test assertion whose review is rejected.", ClaimType: knowledgetypes.ClaimType_CLAIM_TYPE_ASSERTION}
	round := &knowledgetypes.VerificationRound{Id: "payment-round", ClaimId: claim.Id, StartedAtBlock: 90, Phase: knowledgetypes.VerificationPhase_VERIFICATION_PHASE_AGGREGATION}
	require.NoError(t, app.KnowledgeKeeper.SetClaim(ctx, claim))
	require.NoError(t, app.KnowledgeKeeper.SetVerificationRound(ctx, round))
	// The first 28uzrn transfer could succeed, but the complete 55uzrn batch
	// cannot. A partial payment must roll back through the actual bank store.
	fundSettlementFixture(t, app, ctx, 40)
	result := &knowledgekeeper.VerificationResult{Verdict: knowledgetypes.Verdict_VERDICT_REJECT, Rewards: []knowledgekeeper.VerifierReward{{Verifier: a.String()}, {Verifier: b.String()}}}
	require.NoError(t, app.KnowledgeKeeper.CompleteRound(ctx, round, result))
	stored, found := app.KnowledgeKeeper.GetVerificationRound(ctx, round.Id)
	require.True(t, found)
	require.Equal(t, knowledgetypes.VerificationPhase_VERIFICATION_PHASE_COMPLETE, stored.Phase)
	require.Equal(t, knowledgetypes.Verdict_VERDICT_REJECT, stored.Verdict)
	require.NotNil(t, stored.VerifierRewardSettlement)
	require.Zero(t, stored.VerifierRewardSettlement.PaidAtBlock)
	require.Equal(t, "28", stored.VerifierRewardSettlement.Payments[0].Amount)
	require.Equal(t, "27", stored.VerifierRewardSettlement.Payments[1].Amount)
	require.True(t, app.BankKeeper.GetBalance(ctx, a, "uzrn").IsZero())
	require.True(t, app.BankKeeper.GetBalance(ctx, b, "uzrn").IsZero())
	for _, event := range ctx.EventManager().Events() {
		require.NotEqual(t, "zerone.knowledge.verifier_rewarded", event.Type)
	}
	frozen := proto.Clone(stored.VerifierRewardSettlement).(*knowledgetypes.VerifierRewardSettlement)
	// Later policy changes cannot reprice an already accrued obligation.
	params, err := app.KnowledgeKeeper.GetParams(ctx)
	require.NoError(t, err)
	params.IndependenceRewardStrengthBps = 900000
	require.NoError(t, app.KnowledgeKeeper.SetParams(ctx, params))
	fundSettlementFixture(t, app, ctx, 15)
	beforeSupply := app.BankKeeper.GetSupply(ctx, "uzrn")
	ctx = ctx.WithBlockHeight(101).WithHeaderInfo(header.Info{Height: 101, ChainID: ctx.ChainID()})
	require.NoError(t, app.KnowledgeKeeper.ProcessPendingVerifierRewards(ctx))
	stored, found = app.KnowledgeKeeper.GetVerificationRound(ctx, round.Id)
	require.True(t, found)
	require.Equal(t, uint64(101), stored.VerifierRewardSettlement.PaidAtBlock)
	frozen.PaidAtBlock = 101
	require.True(t, proto.Equal(frozen, stored.VerifierRewardSettlement))
	require.Equal(t, "28", app.BankKeeper.GetBalance(ctx, a, "uzrn").Amount.String())
	require.Equal(t, "27", app.BankKeeper.GetBalance(ctx, b, "uzrn").Amount.String())
	require.Equal(t, beforeSupply, app.BankKeeper.GetSupply(ctx, "uzrn"))
	paidEvents := 0
	for _, event := range ctx.EventManager().Events() {
		if event.Type == "zerone.knowledge.verifier_rewarded" {
			paidEvents++
		}
	}
	require.Equal(t, 2, paidEvents)
	beforeRound := proto.Clone(stored).(*knowledgetypes.VerificationRound)
	require.NoError(t, app.KnowledgeKeeper.CompleteRound(ctx, round, result))
	require.NoError(t, app.KnowledgeKeeper.ProcessPendingVerifierRewards(ctx))
	stored, _ = app.KnowledgeKeeper.GetVerificationRound(ctx, round.Id)
	require.True(t, proto.Equal(beforeRound, stored))
	require.Equal(t, "28", app.BankKeeper.GetBalance(ctx, a, "uzrn").Amount.String())
	require.Equal(t, "27", app.BankKeeper.GetBalance(ctx, b, "uzrn").Amount.String())
	stored.VerifierRewardSettlement.Payments[0].Amount = "29"
	require.ErrorContains(t, app.KnowledgeKeeper.SetVerificationRound(ctx, stored), "frozen")
}

func TestRecordIntegrityWithholdingFailureRollsBackEveryBankLeg(t *testing.T) {
	app, ctx := settlementFixture(t)
	a, b := settlementAddress(63), settlementAddress(64)
	round := &knowledgetypes.VerificationRound{Id: "withheld-round", ClaimId: "withheld-claim", Phase: knowledgetypes.VerificationPhase_VERIFICATION_PHASE_COMPLETE, VerdictBlock: 100,
		VerifierRewardSettlement: &knowledgetypes.VerifierRewardSettlement{CreatedAtBlock: 100, WithheldTotal: "5", Payments: []*knowledgetypes.VerifierRewardPayment{
			{Verifier: a.String(), Amount: "10", Withheld: "3"}, {Verifier: b.String(), Amount: "10", Withheld: "2"},
		}},
	}
	require.NoError(t, app.KnowledgeKeeper.SetVerificationRound(ctx, round))
	fundSettlementFixture(t, app, ctx, 24)
	beforeEvents := len(ctx.EventManager().Events())
	require.ErrorContains(t, app.KnowledgeKeeper.TrySettleVerifierRewards(ctx, round.Id), "withholding remains unpaid")
	require.True(t, app.BankKeeper.GetBalance(ctx, a, "uzrn").IsZero())
	require.True(t, app.BankKeeper.GetBalance(ctx, b, "uzrn").IsZero())
	require.Len(t, ctx.EventManager().Events(), beforeEvents)
	stored, _ := app.KnowledgeKeeper.GetVerificationRound(ctx, round.Id)
	require.Zero(t, stored.VerifierRewardSettlement.PaidAtBlock)
	fundSettlementFixture(t, app, ctx, 1)
	require.NoError(t, app.KnowledgeKeeper.TrySettleVerifierRewards(ctx, round.Id))
	require.Equal(t, "10", app.BankKeeper.GetBalance(ctx, a, "uzrn").Amount.String())
	require.Equal(t, "10", app.BankKeeper.GetBalance(ctx, b, "uzrn").Amount.String())
}

func TestRecordIntegrityPendingRewardCursorDoesNotStarveLaterRound(t *testing.T) {
	app, ctx := settlementFixture(t)
	for i := 0; i <= knowledgekeeper.PendingVerifierRewardBatchSize; i++ {
		amount := "100"
		if i == knowledgekeeper.PendingVerifierRewardBatchSize {
			amount = "1"
		}
		round := &knowledgetypes.VerificationRound{Id: fmt.Sprintf("cursor-%02d", i), ClaimId: fmt.Sprintf("cursor-claim-%02d", i), Phase: knowledgetypes.VerificationPhase_VERIFICATION_PHASE_COMPLETE, VerdictBlock: 100,
			VerifierRewardSettlement: &knowledgetypes.VerifierRewardSettlement{CreatedAtBlock: 100, WithheldTotal: "0", Payments: []*knowledgetypes.VerifierRewardPayment{{Verifier: settlementAddress(byte(80 + i)).String(), Amount: amount, Withheld: "0"}}},
		}
		require.NoError(t, app.KnowledgeKeeper.SetVerificationRound(ctx, round))
	}
	fundSettlementFixture(t, app, ctx, 1)
	require.NoError(t, app.KnowledgeKeeper.ProcessPendingVerifierRewards(ctx))
	last, _ := app.KnowledgeKeeper.GetVerificationRound(ctx, fmt.Sprintf("cursor-%02d", knowledgekeeper.PendingVerifierRewardBatchSize))
	require.Zero(t, last.VerifierRewardSettlement.PaidAtBlock, "first batch is bounded")
	require.NoError(t, app.KnowledgeKeeper.ProcessPendingVerifierRewards(ctx))
	last, _ = app.KnowledgeKeeper.GetVerificationRound(ctx, last.Id)
	require.Equal(t, uint64(100), last.VerifierRewardSettlement.PaidAtBlock)
	first, _ := app.KnowledgeKeeper.GetVerificationRound(ctx, "cursor-00")
	require.Zero(t, first.VerifierRewardSettlement.PaidAtBlock)
}

func TestRecordIntegrityLegacyCompletedRoundDoesNotInventPayment(t *testing.T) {
	app, ctx := settlementFixture(t)
	round := &knowledgetypes.VerificationRound{Id: "legacy-round", ClaimId: "legacy-claim", Phase: knowledgetypes.VerificationPhase_VERIFICATION_PHASE_COMPLETE, Verdict: knowledgetypes.Verdict_VERDICT_ACCEPT, VerdictBlock: 5}
	require.NoError(t, app.KnowledgeKeeper.SetVerificationRound(ctx, round))
	beforeSupply := app.BankKeeper.GetSupply(ctx, "uzrn")
	require.NoError(t, app.KnowledgeKeeper.TrySettleVerifierRewards(ctx, round.Id))
	require.NoError(t, app.KnowledgeKeeper.RebuildPendingVerifierRewardIndex(ctx))
	require.NoError(t, app.KnowledgeKeeper.ProcessPendingVerifierRewards(ctx))
	after, _ := app.KnowledgeKeeper.GetVerificationRound(ctx, round.Id)
	require.Nil(t, after.VerifierRewardSettlement)
	require.True(t, proto.Equal(round, after))
	require.Equal(t, beforeSupply, app.BankKeeper.GetSupply(ctx, "uzrn"))
}

func TestRecordIntegrityRewardPlanRejectsCorruptIndependence(t *testing.T) {
	for _, raw := range []string{`{`, `{"total_votes":1,"minority_votes":2}`, `{"total_votes":0,"minority_votes":1}`} {
		t.Run(raw, func(t *testing.T) {
			app, ctx := settlementFixture(t)
			params, err := app.KnowledgeKeeper.GetParams(ctx)
			require.NoError(t, err)
			params.IndependenceRewardStrengthBps = 500000
			require.NoError(t, app.KnowledgeKeeper.SetParams(ctx, params))
			verifier := settlementAddress(65)
			claim := &knowledgetypes.Claim{Id: "corrupt-policy-claim", Submitter: settlementAddress(66).String(), Stake: "100", ClaimType: knowledgetypes.ClaimType_CLAIM_TYPE_ASSERTION}
			round := &knowledgetypes.VerificationRound{Id: "corrupt-policy-round", ClaimId: claim.Id, StartedAtBlock: 90, Phase: knowledgetypes.VerificationPhase_VERIFICATION_PHASE_AGGREGATION}
			require.NoError(t, app.KnowledgeKeeper.SetClaim(ctx, claim))
			require.NoError(t, app.KnowledgeKeeper.SetVerificationRound(ctx, round))
			fundSettlementFixture(t, app, ctx, 55)
			ctx.KVStore(app.keys[knowledgetypes.StoreKey]).Set(knowledgetypes.ValidatorIndependenceKey(verifier.String()), []byte(raw))
			beforeEvents := len(ctx.EventManager().Events())
			result := &knowledgekeeper.VerificationResult{Verdict: knowledgetypes.Verdict_VERDICT_REJECT, Rewards: []knowledgekeeper.VerifierReward{{Verifier: verifier.String()}}}
			require.ErrorContains(t, app.KnowledgeKeeper.CompleteRound(ctx, round, result), "independence")
			stored, found := app.KnowledgeKeeper.GetVerificationRound(ctx, round.Id)
			require.True(t, found)
			require.Equal(t, knowledgetypes.VerificationPhase_VERIFICATION_PHASE_AGGREGATION, stored.Phase)
			require.Nil(t, stored.VerifierRewardSettlement)
			require.True(t, app.BankKeeper.GetBalance(ctx, verifier, "uzrn").IsZero())
			require.Len(t, ctx.EventManager().Events(), beforeEvents)
			// Repairing the source counters permits the same completion. The
			// established formula with 50% conformity and 50% strength pays
			// floor(55 * .75) = 41 and explicitly withholds the remaining 14.
			require.NoError(t, app.KnowledgeKeeper.SetValidatorIndependence(ctx, verifier.String(), knowledgekeeper.ValidatorIndependenceRecord{Validator: verifier.String(), TotalVotes: 2, MinorityVotes: 1}))
			require.NoError(t, app.KnowledgeKeeper.CompleteRound(ctx, round, result))
			stored, _ = app.KnowledgeKeeper.GetVerificationRound(ctx, round.Id)
			require.Equal(t, "41", stored.VerifierRewardSettlement.Payments[0].Amount)
			require.Equal(t, "14", stored.VerifierRewardSettlement.WithheldTotal)
			require.Equal(t, "41", app.BankKeeper.GetBalance(ctx, verifier, "uzrn").Amount.String())
		})
	}
}

type researchFailureBank struct{ knowledgetypes.BankKeeper }

func (b researchFailureBank) SendCoinsFromModuleToModule(ctx context.Context, from, to string, coins sdk.Coins) error {
	if to == "research_fund" {
		return fmt.Errorf("injected research transfer failure")
	}
	return b.BankKeeper.SendCoinsFromModuleToModule(ctx, from, to, coins)
}

func TestRecordIntegritySubmissionSplitFailureRollsBackFeeAndSponsorship(t *testing.T) {
	for _, sponsored := range []bool{false, true} {
		t.Run(fmt.Sprintf("sponsored=%t", sponsored), func(t *testing.T) {
			app, ctx := settlementFixture(t)
			if sponsored {
				// Explicit predecessor fixture: sponsorship is retired for
				// newly admitted neutral-policy claims. This case tests the
				// existing sponsored fee batch's rollback, not new admission.
				ctx.KVStore(app.keys[knowledgetypes.StoreKey]).Delete([]byte(knowledgekeeper.FundSettlementEnabledStoreKey))
				ctx.KVStore(app.keys[knowledgetypes.StoreKey]).Delete([]byte(knowledgekeeper.ReviewNeutralityEnabledStoreKey))
			}
			submitter := settlementAddress(111)
			coins := sdk.NewCoins(sdk.NewInt64Coin("uzrn", 2200000))
			require.NoError(t, app.BankKeeper.MintCoins(ctx, knowledgetypes.BootstrapFundModuleName, coins))
			if !sponsored {
				require.NoError(t, app.BankKeeper.SendCoinsFromModuleToAccount(ctx, knowledgetypes.BootstrapFundModuleName, submitter, coins))
			}
			// Wrap only the final bank leg; the earlier fee collection and
			// treasury/development legs use the actual SDK bank store.
			k := knowledgekeeper.NewKeeper(runtime.NewKVStoreService(app.keys[knowledgetypes.StoreKey]), app.appCodec, app.KnowledgeKeeper.GetAuthority(), researchFailureBank{app.BankKeeper}, nil)
			server := knowledgekeeper.NewMsgServerImpl(k)
			beforeSender := app.BankKeeper.GetBalance(ctx, submitter, "uzrn")
			beforeSupply := app.BankKeeper.GetSupply(ctx, "uzrn")
			beforeEvents := len(ctx.EventManager().Events())
			beforeCount := k.GetBootstrapClaimCount(ctx, submitter.String())
			beforeEpoch := k.GetBootstrapEpochCount(ctx, k.CurrentEpoch(ctx))
			statement := "A recorded contribution with a failing fee split in a local test."
			_, err := server.SubmitClaim(ctx, &knowledgetypes.MsgSubmitClaim{Submitter: submitter.String(), FactContent: statement, Domain: "mathematics", Category: "formal", Stake: "2200000", Sponsored: sponsored})
			require.ErrorContains(t, err, "injected research transfer failure")
			require.Equal(t, beforeSender, app.BankKeeper.GetBalance(ctx, submitter, "uzrn"))
			require.Equal(t, beforeSupply, app.BankKeeper.GetSupply(ctx, "uzrn"))
			require.Equal(t, beforeCount, k.GetBootstrapClaimCount(ctx, submitter.String()))
			require.Equal(t, beforeEpoch, k.GetBootstrapEpochCount(ctx, k.CurrentEpoch(ctx)))
			require.Len(t, ctx.EventManager().Events(), beforeEvents)
			_, found := k.GetClaimByContentHash(ctx, knowledgekeeper.ComputeClaimContentHash(statement, "mathematics"))
			require.False(t, found)
		})
	}
}

func TestRecordIntegrityRetryBudgetStopsBeforeAnotherLargeBatch(t *testing.T) {
	app, ctx := settlementFixture(t)
	for i, count := range []int{2, knowledgetypes.CommitSeatHardCap} {
		payments := make([]*knowledgetypes.VerifierRewardPayment, 0, count)
		for j := 0; j < count; j++ {
			digest := sha256.Sum256([]byte(fmt.Sprintf("reviewer-%d-%d", i, j)))
			amount := "0"
			if j == 0 {
				amount = "1"
			}
			payments = append(payments, &knowledgetypes.VerifierRewardPayment{Verifier: sdk.AccAddress(digest[:20]).String(), Amount: amount, Withheld: "0"})
		}
		round := &knowledgetypes.VerificationRound{Id: fmt.Sprintf("budget-%d", i), ClaimId: fmt.Sprintf("budget-claim-%d", i), Phase: knowledgetypes.VerificationPhase_VERIFICATION_PHASE_COMPLETE, VerdictBlock: 100,
			VerifierRewardSettlement: &knowledgetypes.VerifierRewardSettlement{CreatedAtBlock: 100, WithheldTotal: "0", Payments: payments},
		}
		require.NoError(t, app.KnowledgeKeeper.SetVerificationRound(ctx, round))
	}
	fundSettlementFixture(t, app, ctx, 2)
	require.NoError(t, app.KnowledgeKeeper.ProcessPendingVerifierRewards(ctx))
	first, _ := app.KnowledgeKeeper.GetVerificationRound(ctx, "budget-0")
	second, _ := app.KnowledgeKeeper.GetVerificationRound(ctx, "budget-1")
	require.NotZero(t, first.VerifierRewardSettlement.PaidAtBlock)
	require.Zero(t, second.VerifierRewardSettlement.PaidAtBlock)
	require.NoError(t, app.KnowledgeKeeper.ProcessPendingVerifierRewards(ctx))
	second, _ = app.KnowledgeKeeper.GetVerificationRound(ctx, "budget-1")
	require.NotZero(t, second.VerifierRewardSettlement.PaidAtBlock)
}
