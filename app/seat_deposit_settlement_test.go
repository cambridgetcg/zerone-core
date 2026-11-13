package app

import (
	"testing"

	"cosmossdk.io/core/header"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// Reviewer seat deposit (the reviewer-entry design draft, pull request #190,
// step B) through the application's actual SDK bank, account and zerone_auth
// keepers and the knowledge message handlers. Reviewers here are synthetic
// accounts funded by test issuance; they show how the code behaves, not that
// any outside reviewer has come in. These tests do not run the ante or count
// gas.

const seatTrialChainID = "seat-deposit-bank-test"

// seatTrial is one review round admitted through a real handler, on native
// genesis with the seat-deposit marker on or off.
type seatTrial struct {
	app   *ZeroneApp
	ctx   sdk.Context
	claim *knowledgetypes.Claim
	round *knowledgetypes.VerificationRound
	payer sdk.AccAddress
}

// newSeatTrial admits a claim through route ("ordinary", "conjecture",
// "contradiction", "explicit" or "provisional") on a chain whose seat-deposit
// marker is seat and whose min_reviewer_balance is figure (written straight to
// the params store, as governance would). The windows are two blocks each.
func newSeatTrial(t *testing.T, route string, seat bool, figure string) *seatTrial {
	t.Helper()
	app, ctx, _ := newAccountingAuthorityFixture(t)
	ctx = ctx.WithBlockHeight(100).WithChainID(seatTrialChainID).WithHeaderInfo(header.Info{Height: 100, ChainID: seatTrialChainID})
	if seat {
		require.NoError(t, app.KnowledgeKeeper.EnableSeatDeposit(ctx))
	}
	params, err := app.KnowledgeKeeper.GetParams(ctx)
	require.NoError(t, err)
	params.MinReviewFee, params.MinChallengeStake = "101", "101"
	params.ChallengeConfidenceScalingBps = 0
	params.MinVerifiers, params.MinHeadcountAgreement, params.ConfidenceThreshold = 3, 2, 600000
	params.CommitPhaseBlocks, params.RevealPhaseBlocks, params.AggregationPhaseBlocks = 2, 2, 2
	params.VerificationReward = "0"
	params.MinReviewerBalance = figure
	require.NoError(t, app.KnowledgeKeeper.SetParams(ctx, params))
	tr := &seatTrial{app: app, ctx: ctx, payer: settlementAddress(150)}
	tr.fund(t, tr.payer, 10_000)
	target := &knowledgetypes.Fact{Id: "seat-test-target", Domain: "physics", Category: "empirical", Content: "A scoped synthetic target observation.", Submitter: settlementAddress(151).String(), Status: knowledgetypes.FactStatus_FACT_STATUS_ACTIVE}
	if route == "provisional" {
		target.Status = knowledgetypes.FactStatus_FACT_STATUS_PROVISIONAL
	}
	if route != "ordinary" && route != "conjecture" {
		require.NoError(t, app.KnowledgeKeeper.SetFact(ctx, target))
	}
	server := knowledgekeeper.NewMsgServerImpl(app.KnowledgeKeeper)
	var claimID, roundID string
	switch route {
	case "ordinary":
		response, err := server.SubmitClaim(ctx, &knowledgetypes.MsgSubmitClaim{Submitter: tr.payer.String(), FactContent: "This synthetic local computation has a retained review fee.", Domain: "physics", Category: "empirical", Stake: "101", ClaimType: knowledgetypes.ClaimType_CLAIM_TYPE_ASSERTION})
		require.NoError(t, err)
		claimID = response.ClaimId
	case "conjecture":
		response, err := server.PostConjecture(ctx, &knowledgetypes.MsgPostConjecture{Proposer: tr.payer.String(), Statement: "A synthetic model predicts a repeated observation outside its sampled range.", FalsificationPredicate: "One controlled observation outside the range violates that prediction.", Domain: "physics", Category: "empirical", Stake: "101", ReasoningTrace: "Synthetic well-posedness fixture; acceptance does not establish truth."})
		require.NoError(t, err)
		claimID = response.ClaimId
	case "contradiction":
		response, err := server.SubmitContradiction(ctx, &knowledgetypes.MsgSubmitContradiction{Submitter: tr.payer.String(), FactId: target.Id, CounterClaim: "This synthetic counter-observation contradicts the target.", Category: "empirical", Stake: "101", Reason: "A scoped synthetic counterexample."})
		require.NoError(t, err)
		claimID = response.CounterFactId
	case "explicit":
		response, err := server.ChallengeFact(ctx, &knowledgetypes.MsgChallengeFact{Challenger: tr.payer.String(), FactId: target.Id, Stake: "101", Reason: "A scoped synthetic refutation."})
		require.NoError(t, err)
		roundID = response.RoundId
	case "provisional":
		response, err := server.ChallengeProvisionalFact(ctx, &knowledgetypes.MsgChallengeProvisionalFact{Challenger: tr.payer.String(), FactId: target.Id, Stake: "101", Reason: "A scoped synthetic provisional refutation.", CounterClaim: "The synthetic predicted observation was absent."})
		require.NoError(t, err)
		roundID = response.ChallengeId
	default:
		t.Fatalf("unknown seat trial route %s", route)
	}
	if roundID != "" {
		round, found := app.KnowledgeKeeper.GetVerificationRound(ctx, roundID)
		require.True(t, found)
		claimID = round.ClaimId
	}
	claim, found := app.KnowledgeKeeper.GetClaim(ctx, claimID)
	require.True(t, found)
	tr.claim = claim
	tr.round, found = app.KnowledgeKeeper.GetVerificationRound(ctx, claim.VerificationRoundId)
	require.True(t, found)
	require.Equal(t, knowledgetypes.ReviewPolicyNeutral, tr.round.ReviewPolicyVersion)
	require.Equal(t, knowledgetypes.CommitmentSchemeReviewV2, tr.round.CommitmentScheme)
	return tr
}

func (tr *seatTrial) fund(t *testing.T, address sdk.AccAddress, amount int64) {
	t.Helper()
	coins := sdk.NewCoins(sdk.NewInt64Coin("uzrn", amount))
	require.NoError(t, tr.app.BankKeeper.MintCoins(tr.ctx, knowledgetypes.BootstrapFundModuleName, coins))
	require.NoError(t, tr.app.BankKeeper.SendCoinsFromModuleToAccount(tr.ctx, knowledgetypes.BootstrapFundModuleName, address, coins))
}

func (tr *seatTrial) balance(address sdk.AccAddress) int64 {
	return tr.app.BankKeeper.GetBalance(tr.ctx, address, "uzrn").Amount.Int64()
}

func (tr *seatTrial) custody() int64 {
	return tr.balance(authtypes.NewModuleAddress(knowledgetypes.SeatDepositModuleName))
}

var seatTrialRoutes = []string{"ordinary", "conjecture", "contradiction", "explicit", "provisional"}

// D1: every admission route records the round's seat terms while the marker
// is on, copied from min_reviewer_balance as it stood at creation; an empty
// parameter locks the built-in 100 ZRN. With the marker off no route records
// terms.
func TestSeatDepositTermsAtEveryCreationSite(t *testing.T) {
	for _, route := range seatTrialRoutes {
		for _, tc := range []struct {
			name   string
			seat   bool
			figure string
			want   string
		}{
			{"on/raised-figure", true, "150000000", "150000000"},
			{"on/empty-locks-100-ZRN", true, "", "100000000"},
			{"off", false, "", ""},
		} {
			t.Run(route+"/"+tc.name, func(t *testing.T) {
				tr := newSeatTrial(t, route, tc.seat, tc.figure)
				if tc.want == "" {
					require.Nil(t, tr.round.SeatTerms)
					return
				}
				require.NotNil(t, tr.round.SeatTerms)
				require.Equal(t, tc.want, tr.round.SeatTerms.CountedSeatDeposit)
				// A later parameter change does not alter the open round.
				params, err := tr.app.KnowledgeKeeper.GetParams(tr.ctx)
				require.NoError(t, err)
				params.MinReviewerBalance = "999000000"
				require.NoError(t, tr.app.KnowledgeKeeper.SetParams(tr.ctx, params))
				round, found := tr.app.KnowledgeKeeper.GetVerificationRound(tr.ctx, tr.round.Id)
				require.True(t, found)
				require.Equal(t, tc.want, round.SeatTerms.CountedSeatDeposit)
				round.SeatTerms.CountedSeatDeposit = "999000000"
				require.ErrorContains(t, tr.app.KnowledgeKeeper.SetVerificationRound(tr.ctx, round), "seat terms are immutable")
			})
		}
	}
}

// D1: the marker on with its parameter predecessor off fails closed at round
// creation instead of locking a figure governance cannot set.
func TestSeatDepositTermsRequireTheParameterMarker(t *testing.T) {
	app, ctx, _ := newAccountingAuthorityFixture(t)
	ctx = ctx.WithBlockHeight(100).WithChainID(seatTrialChainID).WithHeaderInfo(header.Info{Height: 100, ChainID: seatTrialChainID})
	require.NoError(t, app.KnowledgeKeeper.EnableSeatDeposit(ctx))
	ctx.KVStore(app.keys[knowledgetypes.StoreKey]).Delete(knowledgetypes.ReviewerBalanceParamEnabledKey)
	payer := settlementAddress(152)
	coins := sdk.NewCoins(sdk.NewInt64Coin("uzrn", 100_000_000))
	require.NoError(t, app.BankKeeper.MintCoins(ctx, knowledgetypes.BootstrapFundModuleName, coins))
	require.NoError(t, app.BankKeeper.SendCoinsFromModuleToAccount(ctx, knowledgetypes.BootstrapFundModuleName, payer, coins))
	params, err := app.KnowledgeKeeper.GetParams(ctx)
	require.NoError(t, err)
	_, err = knowledgekeeper.NewMsgServerImpl(app.KnowledgeKeeper).SubmitClaim(ctx, &knowledgetypes.MsgSubmitClaim{Submitter: payer.String(), FactContent: "This synthetic local computation has a retained review fee.", Domain: "physics", Category: "empirical", Stake: params.MinReviewFee, ClaimType: knowledgetypes.ClaimType_CLAIM_TYPE_ASSERTION})
	require.ErrorContains(t, err, "seat deposits require the reviewer-balance-param activation")
}
