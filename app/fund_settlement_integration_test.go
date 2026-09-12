package app

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"testing"

	"cosmossdk.io/core/header"
	corestore "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/runtime"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	"google.golang.org/protobuf/proto"
)

// Synthetic participants and test issuance exercise the actual SDK bank and
// message handlers. These tests do not assert independent reviewers or ante/gas.
type fundingBankTrial struct {
	app       *ZeroneApp
	db        dbm.DB
	ctx       sdk.Context
	claim     *knowledgetypes.Claim
	round     *knowledgetypes.VerificationRound
	payer     sdk.AccAddress
	reviewers []sdk.AccAddress
	before    map[string]int64
}

func fundingBalanceSnapshot(app *ZeroneApp, ctx sdk.Context, payer sdk.AccAddress, reviewers []sdk.AccAddress) map[string]int64 {
	balances := map[string]int64{"payer": app.BankKeeper.GetBalance(ctx, payer, "uzrn").Amount.Int64()}
	for _, name := range []string{knowledgetypes.ModuleName, "protocol_treasury", "development_fund", "research_fund"} {
		balances[name] = app.BankKeeper.GetBalance(ctx, authtypes.NewModuleAddress(name), "uzrn").Amount.Int64()
	}
	for i, reviewer := range reviewers {
		balances[fmt.Sprintf("reviewer-%d", i)] = app.BankKeeper.GetBalance(ctx, reviewer, "uzrn").Amount.Int64()
	}
	return balances
}

func newFundingBankTrial(t *testing.T, route string) *fundingBankTrial {
	t.Helper()
	app, ctx, db := newAccountingAuthorityFixture(t)
	ctx = ctx.WithBlockHeight(100).WithChainID("fund-settlement-bank-test").WithHeaderInfo(header.Info{Height: 100, ChainID: "fund-settlement-bank-test"})
	require.NoError(t, app.KnowledgeKeeper.EnableClaimRecords(ctx))
	require.NoError(t, app.KnowledgeKeeper.EnableFundSettlement(ctx))
	params, err := app.KnowledgeKeeper.GetParams(ctx)
	require.NoError(t, err)
	params.MinReviewFee, params.MinChallengeStake = "101", "101"
	params.ChallengeConfidenceScalingBps = 0
	params.MinVerifiers, params.MinHeadcountAgreement, params.ConfidenceThreshold = 3, 2, 600000
	params.CommitPhaseBlocks, params.RevealPhaseBlocks, params.AggregationPhaseBlocks = 2, 2, 2
	params.VerificationReward = "0"
	require.NoError(t, app.KnowledgeKeeper.SetParams(ctx, params))
	f := &fundingBankTrial{app: app, db: db, ctx: ctx, payer: settlementAddress(220), reviewers: []sdk.AccAddress{settlementAddress(221), settlementAddress(222), settlementAddress(223)}}
	for i, address := range append([]sdk.AccAddress{f.payer}, f.reviewers...) {
		amount := int64(100000000)
		if i == 0 {
			amount = 10000
		}
		coins := sdk.NewCoins(sdk.NewInt64Coin("uzrn", amount))
		require.NoError(t, app.BankKeeper.MintCoins(ctx, knowledgetypes.BootstrapFundModuleName, coins))
		require.NoError(t, app.BankKeeper.SendCoinsFromModuleToAccount(ctx, knowledgetypes.BootstrapFundModuleName, address, coins))
	}
	f.before = fundingBalanceSnapshot(app, ctx, f.payer, f.reviewers)
	target := &knowledgetypes.Fact{Id: "funding-test-target", Domain: "physics", Category: "empirical", Content: "A scoped synthetic target observation.", Submitter: settlementAddress(224).String(), Status: knowledgetypes.FactStatus_FACT_STATUS_ACTIVE}
	if route != "ordinary" && route != "conjecture" {
		if route == "provisional" {
			target.Status = knowledgetypes.FactStatus_FACT_STATUS_PROVISIONAL
		}
		require.NoError(t, app.KnowledgeKeeper.SetFact(ctx, target))
	}
	server := knowledgekeeper.NewMsgServerImpl(app.KnowledgeKeeper)
	var claimID, roundID string
	switch route {
	case "ordinary":
		response, err := server.SubmitClaim(ctx, &knowledgetypes.MsgSubmitClaim{Submitter: f.payer.String(), FactContent: "This synthetic local computation has a retained review fee.", Domain: "physics", Category: "empirical", Stake: "101", ClaimType: knowledgetypes.ClaimType_CLAIM_TYPE_ASSERTION})
		require.NoError(t, err)
		claimID = response.ClaimId
	case "conjecture":
		response, err := server.PostConjecture(ctx, &knowledgetypes.MsgPostConjecture{Proposer: f.payer.String(), Statement: "A synthetic model predicts a repeated observation outside its sampled range.", FalsificationPredicate: "One controlled observation outside the range violates that prediction.", Domain: "physics", Category: "empirical", Stake: "101", ReasoningTrace: "Synthetic well-posedness fixture; acceptance does not establish truth."})
		require.NoError(t, err)
		claimID = response.ClaimId
	case "contradiction":
		response, err := server.SubmitContradiction(ctx, &knowledgetypes.MsgSubmitContradiction{Submitter: f.payer.String(), FactId: target.Id, CounterClaim: "This synthetic counter-observation contradicts the target.", Category: "empirical", Stake: "101", Reason: "A scoped synthetic counterexample."})
		require.NoError(t, err)
		claimID = response.CounterFactId
	case "explicit":
		response, err := server.ChallengeFact(ctx, &knowledgetypes.MsgChallengeFact{Challenger: f.payer.String(), FactId: target.Id, Stake: "101", Reason: "A scoped synthetic refutation."})
		require.NoError(t, err)
		roundID = response.RoundId
	case "provisional":
		response, err := server.ChallengeProvisionalFact(ctx, &knowledgetypes.MsgChallengeProvisionalFact{Challenger: f.payer.String(), FactId: target.Id, Stake: "101", Reason: "A scoped synthetic provisional refutation.", CounterClaim: "The synthetic predicted observation was absent."})
		require.NoError(t, err)
		roundID = response.ChallengeId
	default:
		t.Fatalf("unknown funding trial route %s", route)
	}
	if roundID != "" {
		round, found := app.KnowledgeKeeper.GetVerificationRound(ctx, roundID)
		require.True(t, found)
		claimID = round.ClaimId
	}
	claim, found := app.KnowledgeKeeper.GetClaim(ctx, claimID)
	require.True(t, found)
	f.claim = claim
	f.round, found = app.KnowledgeKeeper.GetVerificationRound(ctx, claim.VerificationRoundId)
	require.True(t, found)
	require.NotNil(t, claim.FundingTerms)
	require.Equal(t, "101", claim.FundingTerms.PaidAmount)
	require.Equal(t, "55", claim.FundingTerms.ReviewBudget)
	require.Equal(t, f.before["payer"]-101, app.BankKeeper.GetBalance(ctx, f.payer, "uzrn").Amount.Int64())
	if route == "ordinary" || route == "conjecture" {
		require.Equal(t, knowledgetypes.ClaimFundingKind_CLAIM_FUNDING_KIND_REVIEW_FEE, claim.FundingTerms.Kind)
		require.Equal(t, "46", claim.FundingTerms.RetainedFee)
	} else {
		require.Equal(t, knowledgetypes.ClaimFundingKind_CLAIM_FUNDING_KIND_CHALLENGE_DEPOSIT, claim.FundingTerms.Kind)
		require.Equal(t, "46", claim.FundingTerms.RefundableAmount)
	}
	if route == "contradiction" {
		require.Empty(t, claim.ProvisionalFactId, "financial repair must not relabel the scientific target route")
	}
	return f
}

func (f *fundingBankTrial) height(height int64) {
	f.ctx = f.ctx.WithBlockHeight(height).WithHeaderInfo(header.Info{Height: height, ChainID: f.ctx.ChainID()})
}

func (f *fundingBankTrial) reviews(t *testing.T, votes []string) {
	t.Helper()
	server := knowledgekeeper.NewMsgServerImpl(f.app.KnowledgeKeeper)
	att := &knowledgetypes.ReviewAttestation{Reason: "Synthetic operator-owned review of a stated test fixture.", Scope: "No independent scientific assurance is claimed."}
	for i, vote := range votes {
		if vote == "" {
			vote = "accept"
		}
		salt := bytes.Repeat([]byte{byte(i + 1)}, 16)
		hash, err := knowledgetypes.ComputeReviewCommitmentV2(f.round.CommitmentChainId, f.round.Id, f.reviewers[i].String(), vote, 800000, salt, att)
		require.NoError(t, err)
		_, err = server.SubmitCommitment(f.ctx, &knowledgetypes.MsgSubmitCommitment{Verifier: f.reviewers[i].String(), RoundId: f.round.Id, CommitHash: hash})
		require.NoError(t, err)
	}
	f.height(102)
	require.NoError(t, f.app.KnowledgeKeeper.AdvanceRoundPhases(f.ctx))
	for i, vote := range votes {
		if vote == "" {
			continue
		}
		_, err := server.SubmitReveal(f.ctx, &knowledgetypes.MsgSubmitReveal{Verifier: f.reviewers[i].String(), RoundId: f.round.Id, Vote: vote, Confidence: 800000, Salt: bytes.Repeat([]byte{byte(i + 1)}, 16), Attestation: att})
		require.NoError(t, err)
	}
}

func (f *fundingBankTrial) complete(t *testing.T) *knowledgetypes.VerificationRound {
	t.Helper()
	f.height(106)
	require.NoError(t, f.app.KnowledgeKeeper.AdvanceRoundPhases(f.ctx))
	round, found := f.app.KnowledgeKeeper.GetVerificationRound(f.ctx, f.round.Id)
	require.True(t, found)
	require.Equal(t, knowledgetypes.VerificationPhase_VERIFICATION_PHASE_COMPLETE, round.Phase)
	return round
}

func TestFundSettlementActualBankAllRoutesAndTerminalOutcomes(t *testing.T) {
	for _, route := range []string{"ordinary", "conjecture", "contradiction", "explicit", "provisional"} {
		for _, tc := range []struct {
			name    string
			votes   []string
			verdict knowledgetypes.Verdict
		}{
			{"accept", []string{"accept", "accept", "accept"}, knowledgetypes.Verdict_VERDICT_ACCEPT},
			{"reject-with-dissent", []string{"reject", "reject", "accept"}, knowledgetypes.Verdict_VERDICT_REJECT},
			{"malformed", []string{"malformed", "malformed", "accept"}, knowledgetypes.Verdict_VERDICT_MALFORMED},
			{"inconclusive", []string{"accept", "reject", "malformed"}, knowledgetypes.Verdict_VERDICT_INCONCLUSIVE},
			{"under-quorum", []string{"accept", "reject", ""}, knowledgetypes.Verdict_VERDICT_INCONCLUSIVE},
			{"zero-review", nil, knowledgetypes.Verdict_VERDICT_INCONCLUSIVE},
		} {
			t.Run(route+"/"+tc.name, func(t *testing.T) {
				f := newFundingBankTrial(t, route)
				f.reviews(t, tc.votes)
				supply := f.app.BankKeeper.GetSupply(f.ctx, "uzrn")
				round := f.complete(t)
				require.Equal(t, tc.verdict, round.Verdict)
				valid := 0
				for _, vote := range tc.votes {
					if vote != "" {
						valid++
					}
				}
				refund := int64(0)
				if route != "ordinary" && route != "conjecture" {
					refund = 46
				}
				if valid == 0 {
					refund += 55
					require.Nil(t, round.VerifierRewardSettlement)
				} else {
					require.NotNil(t, round.VerifierRewardSettlement)
					require.NotZero(t, round.VerifierRewardSettlement.PaidAtBlock)
					require.Len(t, round.VerifierRewardSettlement.Payments, valid)
					require.Equal(t, "0", round.VerifierRewardSettlement.WithheldTotal)
				}
				if refund == 0 {
					require.Nil(t, round.ClaimRefundSettlement)
				} else {
					require.NotNil(t, round.ClaimRefundSettlement)
					require.Equal(t, fmt.Sprint(refund), round.ClaimRefundSettlement.Amount)
					require.NotZero(t, round.ClaimRefundSettlement.PaidAtBlock)
				}
				after := fundingBalanceSnapshot(f.app, f.ctx, f.payer, f.reviewers)
				require.Equal(t, f.before["payer"]-101+refund, after["payer"])
				var reviewerTotal int64
				for i := range f.reviewers {
					delta := after[fmt.Sprintf("reviewer-%d", i)] - f.before[fmt.Sprintf("reviewer-%d", i)]
					if i >= len(tc.votes) || tc.votes[i] == "" {
						require.Zero(t, delta)
					} else {
						expected := int64(55 / valid)
						if i == 0 {
							expected += int64(55 % valid)
						}
						require.Equal(t, expected, delta)
					}
					reviewerTotal += delta
				}
				if valid != 0 {
					require.Equal(t, int64(55), reviewerTotal)
				}
				require.Equal(t, f.before[knowledgetypes.ModuleName], after[knowledgetypes.ModuleName], "no unassigned new claim balance remains")
				if route == "ordinary" || route == "conjecture" {
					require.Equal(t, int64(22), after["protocol_treasury"]-f.before["protocol_treasury"])
					require.Equal(t, int64(19), after["development_fund"]-f.before["development_fund"])
					require.Equal(t, int64(5), after["research_fund"]-f.before["research_fund"])
				} else {
					for _, name := range []string{"protocol_treasury", "development_fund", "research_fund"} {
						require.Equal(t, f.before[name], after[name])
					}
				}
				require.Equal(t, supply, f.app.BankKeeper.GetSupply(f.ctx, "uzrn"))
				if route == "conjecture" && tc.verdict == knowledgetypes.Verdict_VERDICT_ACCEPT {
					fact, found := f.app.KnowledgeKeeper.GetFact(f.ctx, knowledgekeeper.GenerateFactID(f.claim.Id, round.VerdictBlock))
					require.True(t, found)
					// The financial outcome does not promote an admitted
					// question into established scientific evidence.
					require.Equal(t, knowledgetypes.ClaimType_CLAIM_TYPE_CONJECTURE, fact.ClaimType)
					require.Equal(t, knowledgetypes.FactStatus_FACT_STATUS_PROVISIONAL, fact.Status)
					require.Zero(t, fact.Confidence)
				}
				f.height(107)
				require.NoError(t, f.app.KnowledgeKeeper.ProcessPendingVerifierRewards(f.ctx))
				require.NoError(t, f.app.KnowledgeKeeper.TrySettleVerifierRewards(f.ctx, round.Id))
				require.Equal(t, after, fundingBalanceSnapshot(f.app, f.ctx, f.payer, f.reviewers), "retry cannot pay twice")
				stored, _ := f.app.KnowledgeKeeper.GetVerificationRound(f.ctx, round.Id)
				require.True(t, proto.Equal(round, stored))
			})
		}
	}
}

func TestFundSettlementActualBankRefundFailureRollsBackReviewPayments(t *testing.T) {
	f := newFundingBankTrial(t, "contradiction")
	f.reviews(t, []string{"accept", "reject", "malformed"})
	// Leave enough for all 55 review units and only 45 of the 46-unit bond.
	// This is an explicit test balance movement, not a production seizure.
	require.NoError(t, f.app.BankKeeper.SendCoinsFromModuleToModule(f.ctx, knowledgetypes.ModuleName, knowledgetypes.BootstrapFundModuleName, sdk.NewCoins(sdk.NewInt64Coin("uzrn", 1))))
	before := fundingBalanceSnapshot(f.app, f.ctx, f.payer, f.reviewers)
	f.ctx = f.ctx.WithEventManager(sdk.NewEventManager())
	round := f.complete(t)
	require.Equal(t, knowledgetypes.Verdict_VERDICT_INCONCLUSIVE, round.Verdict)
	require.Zero(t, round.VerifierRewardSettlement.PaidAtBlock)
	require.Zero(t, round.ClaimRefundSettlement.PaidAtBlock)
	require.Equal(t, before, fundingBalanceSnapshot(f.app, f.ctx, f.payer, f.reviewers), "failed final refund discards every earlier reviewer transfer")
	for _, event := range f.ctx.EventManager().Events() {
		require.NotEqual(t, "zerone.knowledge.verifier_rewarded", event.Type)
		require.NotEqual(t, "zerone.knowledge.claim_refunded", event.Type)
	}
	frozen := proto.Clone(round).(*knowledgetypes.VerificationRound)
	require.NoError(t, f.app.BankKeeper.SendCoinsFromModuleToModule(f.ctx, knowledgetypes.BootstrapFundModuleName, knowledgetypes.ModuleName, sdk.NewCoins(sdk.NewInt64Coin("uzrn", 1))))
	supply := f.app.BankKeeper.GetSupply(f.ctx, "uzrn")
	f.height(107)
	require.NoError(t, f.app.KnowledgeKeeper.ProcessPendingVerifierRewards(f.ctx))
	paid, _ := f.app.KnowledgeKeeper.GetVerificationRound(f.ctx, round.Id)
	frozen.VerifierRewardSettlement.PaidAtBlock, frozen.ClaimRefundSettlement.PaidAtBlock = 107, 107
	require.True(t, proto.Equal(frozen, paid))
	require.Equal(t, int64(9945), f.app.BankKeeper.GetBalance(f.ctx, f.payer, "uzrn").Amount.Int64())
	require.Equal(t, supply, f.app.BankKeeper.GetSupply(f.ctx, "uzrn"))
	after := fundingBalanceSnapshot(f.app, f.ctx, f.payer, f.reviewers)
	require.NoError(t, f.app.KnowledgeKeeper.ProcessPendingVerifierRewards(f.ctx))
	require.Equal(t, after, fundingBalanceSnapshot(f.app, f.ctx, f.payer, f.reviewers))
}

func TestFundSettlementRefusesSecondRoundAndFundingRewrite(t *testing.T) {
	f := newFundingBankTrial(t, "ordinary")
	before := claimRecordsStores(t, f.app, f.ctx)
	_, err := f.app.KnowledgeKeeper.CreateVerificationRound(f.ctx, f.claim)
	require.ErrorContains(t, err, "sole verification round")
	require.Equal(t, before, claimRecordsStores(t, f.app, f.ctx))
	require.Error(t, f.app.KnowledgeKeeper.DeleteClaim(f.ctx, f.claim.Id))
	require.Error(t, f.app.KnowledgeKeeper.DeleteVerificationRound(f.ctx, f.round.Id))
	require.Equal(t, before, claimRecordsStores(t, f.app, f.ctx))
	for _, change := range []func(*knowledgetypes.Claim){
		func(c *knowledgetypes.Claim) { c.Submitter = f.reviewers[0].String() },
		func(c *knowledgetypes.Claim) { c.VerificationRoundId = "another-round" },
		func(c *knowledgetypes.Claim) { c.FundingTerms = nil },
	} {
		claim := proto.Clone(f.claim).(*knowledgetypes.Claim)
		change(claim)
		require.Error(t, f.app.KnowledgeKeeper.SetClaim(f.ctx, claim))
		require.Equal(t, before, claimRecordsStores(t, f.app, f.ctx))
	}
}

func TestFundSettlementRefundOnlyQueueSurvivesReopenAndGenesis(t *testing.T) {
	f := newFundingBankTrial(t, "ordinary")
	f.reviews(t, nil)
	// One missing unit is enough to retain the complete 55-unit refund.
	require.NoError(t, f.app.BankKeeper.SendCoinsFromModuleToModule(f.ctx, knowledgetypes.ModuleName, knowledgetypes.BootstrapFundModuleName, sdk.NewCoins(sdk.NewInt64Coin("uzrn", 1))))
	round := f.complete(t)
	require.Nil(t, round.VerifierRewardSettlement)
	require.Equal(t, "55", round.ClaimRefundSettlement.Amount)
	require.Zero(t, round.ClaimRefundSettlement.PaidAtBlock)
	before := fundingBalanceSnapshot(f.app, f.ctx, f.payer, f.reviewers)
	exported := f.app.KnowledgeKeeper.ExportGenesis(f.ctx)
	encoded := f.app.appCodec.MustMarshalJSON(exported)
	var decoded knowledgetypes.GenesisState
	f.app.appCodec.MustUnmarshalJSON(encoded, &decoded)
	imported, ictx, _ := newAccountingAuthorityFixture(t)
	require.NoError(t, imported.KnowledgeKeeper.InitGenesis(ictx, &decoded))
	retained, found := imported.KnowledgeKeeper.GetVerificationRound(ictx, round.Id)
	require.True(t, found)
	require.True(t, proto.Equal(round, retained), "JSON import preserves an unpaid refund without inventing reviewer work")
	// This import check establishes record/index preservation, not a whole
	// ledger balance migration. The same-DB reopen below retains real balances.
	index := append([]byte{0x82}, []byte(round.Id)...)
	require.Equal(t, []byte{1}, ictx.KVStore(imported.keys[knowledgetypes.StoreKey]).Get(index))
	f.app.CommitMultiStore().Commit()
	f.app = NewZeroneApp(log.NewNopLogger(), f.db, nil, true, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()))
	f.ctx = f.app.NewUncachedContext(false, cmtproto.Header{ChainID: "fund-settlement-bank-test", Height: 107}).WithHeaderInfo(header.Info{Height: 107, ChainID: "fund-settlement-bank-test"})
	require.Equal(t, before, fundingBalanceSnapshot(f.app, f.ctx, f.payer, f.reviewers))
	require.NoError(t, f.app.KnowledgeKeeper.ProcessPendingVerifierRewards(f.ctx))
	retained, _ = f.app.KnowledgeKeeper.GetVerificationRound(f.ctx, round.Id)
	require.True(t, proto.Equal(round, retained), "unfunded replay retains exact instructions")
	require.NoError(t, f.app.BankKeeper.SendCoinsFromModuleToModule(f.ctx, knowledgetypes.BootstrapFundModuleName, knowledgetypes.ModuleName, sdk.NewCoins(sdk.NewInt64Coin("uzrn", 1))))
	supply := f.app.BankKeeper.GetSupply(f.ctx, "uzrn")
	require.NoError(t, f.app.KnowledgeKeeper.ProcessPendingVerifierRewards(f.ctx))
	retained, _ = f.app.KnowledgeKeeper.GetVerificationRound(f.ctx, round.Id)
	require.Equal(t, uint64(107), retained.ClaimRefundSettlement.PaidAtBlock)
	require.Equal(t, int64(9954), f.app.BankKeeper.GetBalance(f.ctx, f.payer, "uzrn").Amount.Int64())
	require.Equal(t, supply, f.app.BankKeeper.GetSupply(f.ctx, "uzrn"))
	require.Nil(t, f.ctx.KVStore(f.app.keys[knowledgetypes.StoreKey]).Get(index))
	after := fundingBalanceSnapshot(f.app, f.ctx, f.payer, f.reviewers)
	f.app.CommitMultiStore().Commit()
	f.app = NewZeroneApp(log.NewNopLogger(), f.db, nil, true, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()))
	f.ctx = f.app.NewUncachedContext(false, cmtproto.Header{ChainID: "fund-settlement-bank-test", Height: 108}).WithHeaderInfo(header.Info{Height: 108, ChainID: "fund-settlement-bank-test"})
	require.NoError(t, f.app.KnowledgeKeeper.ProcessPendingVerifierRewards(f.ctx))
	require.Equal(t, after, fundingBalanceSnapshot(f.app, f.ctx, f.payer, f.reviewers), "a paid refund cannot replay after reopen")
}

func TestFundSettlementInvalidAndLateRevealsEarnNoPayment(t *testing.T) {
	for _, scenario := range []string{"invalid preimage", "late reveal"} {
		t.Run(scenario, func(t *testing.T) {
			f := newFundingBankTrial(t, "contradiction")
			f.reviews(t, []string{""}) // one actual commitment, no valid reveal
			if scenario == "late reveal" {
				f.height(104)
			}
			salt := bytes.Repeat([]byte{1}, 16)
			if scenario == "invalid preimage" {
				salt[0] = 2
			}
			before, found := f.app.KnowledgeKeeper.GetVerificationRound(f.ctx, f.round.Id)
			require.True(t, found)
			_, err := knowledgekeeper.NewMsgServerImpl(f.app.KnowledgeKeeper).SubmitReveal(f.ctx, &knowledgetypes.MsgSubmitReveal{Verifier: f.reviewers[0].String(), RoundId: f.round.Id, Vote: "accept", Confidence: 800000, Salt: salt,
				Attestation: &knowledgetypes.ReviewAttestation{Reason: "Synthetic operator-owned review of a stated test fixture.", Scope: "No independent scientific assurance is claimed."}})
			require.Error(t, err)
			after, _ := f.app.KnowledgeKeeper.GetVerificationRound(f.ctx, f.round.Id)
			require.True(t, proto.Equal(before, after))
			round := f.complete(t)
			require.Nil(t, round.VerifierRewardSettlement)
			require.Empty(t, round.Reveals)
			require.Equal(t, "101", round.ClaimRefundSettlement.Amount)
			require.Equal(t, f.before["payer"], f.app.BankKeeper.GetBalance(f.ctx, f.payer, "uzrn").Amount.Int64())
			require.Equal(t, f.before["reviewer-0"], f.app.BankKeeper.GetBalance(f.ctx, f.reviewers[0], "uzrn").Amount.Int64())
		})
	}
}

func TestFundSettlementMaxReviewBatchIncludesRefundAndDefersNextRound(t *testing.T) {
	f := newFundingBankTrial(t, "contradiction")
	round := proto.Clone(f.round).(*knowledgetypes.VerificationRound)
	// This bound test seeds cryptographically coherent retained records directly;
	// admission and actual signed tuple checks are covered by the handler tests.
	for i := 0; i < knowledgetypes.CommitSeatHardCap; i++ {
		digest := sha256.Sum256([]byte(fmt.Sprintf("synthetic-funding-seat-%d", i)))
		address := sdk.AccAddress(digest[:20]).String()
		salt := bytes.Repeat([]byte{1}, 16)
		att := &knowledgetypes.ReviewAttestation{Reason: "Synthetic transfer-budget fixture."}
		hash, err := knowledgetypes.ComputeReviewCommitmentV2(round.CommitmentChainId, round.Id, address, "reject", 800000, salt, att)
		require.NoError(t, err)
		round.Commits = append(round.Commits, &knowledgetypes.CommitEntry{Verifier: address, CommitHash: hash, CommittedAtBlock: 100})
		round.SelectedVerifiers = append(round.SelectedVerifiers, address)
		round.Reveals = append(round.Reveals, &knowledgetypes.RevealEntry{Verifier: address, Vote: "reject", Confidence: 800000, Salt: salt, Attestation: att, RevealedAtBlock: 102})
	}
	round.Phase = knowledgetypes.VerificationPhase_VERIFICATION_PHASE_COMPLETE
	round.Verdict, round.VerdictBlock = knowledgetypes.Verdict_VERDICT_REJECT, 106
	round.VerifierRewardSettlement = &knowledgetypes.VerifierRewardSettlement{CreatedAtBlock: 106, WithheldTotal: "0"}
	for i, reveal := range round.Reveals {
		amount := "0"
		if i == 0 {
			amount = "55"
		}
		round.VerifierRewardSettlement.Payments = append(round.VerifierRewardSettlement.Payments, &knowledgetypes.VerifierRewardPayment{Verifier: reveal.Verifier, Amount: amount, Withheld: "0"})
	}
	round.ClaimRefundSettlement = &knowledgetypes.ClaimRefundSettlement{Recipient: f.payer.String(), Amount: "46", CreatedAtBlock: 106}
	require.NoError(t, f.app.KnowledgeKeeper.SetVerificationRound(f.ctx, round))
	secondClaim := proto.Clone(f.claim).(*knowledgetypes.Claim)
	secondClaim.Id, secondClaim.VerificationRoundId = "zz-funding-budget-claim", ""
	require.NoError(t, f.app.KnowledgeKeeper.SetClaim(f.ctx, secondClaim))
	second := &knowledgetypes.VerificationRound{Id: "zz-funding-budget-round", ClaimId: secondClaim.Id, Phase: knowledgetypes.VerificationPhase_VERIFICATION_PHASE_COMPLETE,
		StartedAtBlock: 100, CommitDeadline: 102, RevealDeadline: 104, AggregationDeadline: 106, CommitmentScheme: 2, CommitmentChainId: f.ctx.ChainID(), ReviewPolicyVersion: 1,
		Verdict: knowledgetypes.Verdict_VERDICT_INCONCLUSIVE, VerdictBlock: 106,
		ClaimRefundSettlement: &knowledgetypes.ClaimRefundSettlement{Recipient: f.payer.String(), Amount: "101", CreatedAtBlock: 106}}
	require.NoError(t, f.app.KnowledgeKeeper.SetVerificationRound(f.ctx, second))
	secondClaim.VerificationRoundId = second.Id
	require.NoError(t, f.app.KnowledgeKeeper.SetClaim(f.ctx, secondClaim))
	fundSettlementFixture(t, f.app, f.ctx, 101) // explicit fixture funding for the second seeded claim
	supply := f.app.BankKeeper.GetSupply(f.ctx, "uzrn")
	f.height(106)
	require.NoError(t, f.app.KnowledgeKeeper.ProcessPendingVerifierRewards(f.ctx))
	firstPaid, _ := f.app.KnowledgeKeeper.GetVerificationRound(f.ctx, round.Id)
	require.Equal(t, uint64(106), firstPaid.VerifierRewardSettlement.PaidAtBlock)
	require.Equal(t, uint64(106), firstPaid.ClaimRefundSettlement.PaidAtBlock)
	deferred, _ := f.app.KnowledgeKeeper.GetVerificationRound(f.ctx, second.Id)
	require.Zero(t, deferred.ClaimRefundSettlement.PaidAtBlock, "512 review instructions plus one refund consume the existing batch budget")
	require.NoError(t, f.app.KnowledgeKeeper.ProcessPendingVerifierRewards(f.ctx))
	deferred, _ = f.app.KnowledgeKeeper.GetVerificationRound(f.ctx, second.Id)
	require.Equal(t, uint64(106), deferred.ClaimRefundSettlement.PaidAtBlock)
	require.Equal(t, supply, f.app.BankKeeper.GetSupply(f.ctx, "uzrn"))
	before := fundingBalanceSnapshot(f.app, f.ctx, f.payer, f.reviewers)
	require.NoError(t, f.app.KnowledgeKeeper.ProcessPendingVerifierRewards(f.ctx))
	require.Equal(t, before, fundingBalanceSnapshot(f.app, f.ctx, f.payer, f.reviewers))
}

func TestFundSettlementBankPaymentsRollBackWhenPaidRecordWriteFails(t *testing.T) {
	f := newFundingBankTrial(t, "contradiction")
	f.reviews(t, []string{"accept", "reject", "malformed"})
	require.NoError(t, f.app.BankKeeper.SendCoinsFromModuleToModule(f.ctx, knowledgetypes.ModuleName, knowledgetypes.BootstrapFundModuleName, sdk.NewCoins(sdk.NewInt64Coin("uzrn", 1))))
	round := f.complete(t)
	require.Zero(t, round.ClaimRefundSettlement.PaidAtBlock)
	require.NoError(t, f.app.BankKeeper.SendCoinsFromModuleToModule(f.ctx, knowledgetypes.BootstrapFundModuleName, knowledgetypes.ModuleName, sdk.NewCoins(sdk.NewInt64Coin("uzrn", 1))))
	before := claimRecordsStores(t, f.app, f.ctx)
	f.ctx = f.ctx.WithEventManager(sdk.NewEventManager())
	// All bank transfers use the real keeper. Only the final primary-round
	// write is faulted, after its nested cache has seen the mutated record.
	service := claimRecordFaultService{KVStoreService: runtime.NewKVStoreService(f.app.keys[knowledgetypes.StoreKey]), op: "set", prefix: knowledgetypes.RoundKey(round.Id), after: true}
	faulted := knowledgekeeper.NewKeeper(service, f.app.appCodec, f.app.KnowledgeKeeper.GetAuthority(), f.app.BankKeeper, nil)
	require.ErrorContains(t, faulted.TrySettleVerifierRewards(f.ctx, round.Id), "injected")
	require.Equal(t, before, claimRecordsStores(t, f.app, f.ctx), "bank, account, plan and queue bytes all roll back")
	require.Empty(t, f.ctx.EventManager().Events())
	require.NoError(t, f.app.KnowledgeKeeper.TrySettleVerifierRewards(f.ctx, round.Id))
	paid, _ := f.app.KnowledgeKeeper.GetVerificationRound(f.ctx, round.Id)
	require.Equal(t, paid.ClaimRefundSettlement.PaidAtBlock, paid.VerifierRewardSettlement.PaidAtBlock)
	require.NotZero(t, paid.ClaimRefundSettlement.PaidAtBlock)
}

func TestFundSettlementConjectureSplitAndStorageFailuresRollBack(t *testing.T) {
	for _, failure := range []string{"final split transfer", "round write after fee split"} {
		t.Run(failure, func(t *testing.T) {
			f := newFundingBankTrial(t, "ordinary")
			f.height(1000) // explicit fixture progression beyond the first author's cooldown
			f.ctx = f.ctx.WithEventManager(sdk.NewEventManager())
			before := claimRecordsStores(t, f.app, f.ctx)
			var service corestore.KVStoreService = runtime.NewKVStoreService(f.app.keys[knowledgetypes.StoreKey])
			var bank knowledgetypes.BankKeeper = f.app.BankKeeper
			if failure == "final split transfer" {
				bank = researchFailureBank{bank}
			} else {
				service = claimRecordFaultService{KVStoreService: service, op: "set", prefix: knowledgetypes.VerificationRoundKeyPrefix, after: true}
			}
			k := knowledgekeeper.NewKeeper(service, f.app.appCodec, f.app.KnowledgeKeeper.GetAuthority(), bank, nil)
			_, err := knowledgekeeper.NewMsgServerImpl(k).PostConjecture(f.ctx, &knowledgetypes.MsgPostConjecture{Proposer: f.payer.String(), Statement: "A distinct synthetic conjecture proposes an unobserved test outcome.", FalsificationPredicate: "One controlled counter-observation would refute the proposed outcome.", Domain: "physics", Category: "empirical", Stake: "101"})
			require.ErrorContains(t, err, "injected")
			require.Equal(t, before, claimRecordsStores(t, f.app, f.ctx), "fees, claim, timing, round and indexes roll back as one admission")
			require.Empty(t, f.ctx.EventManager().Events())
		})
	}
}
