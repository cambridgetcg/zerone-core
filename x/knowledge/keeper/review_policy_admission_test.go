package keeper_test

import (
	"bytes"
	"context"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func TestNeutralReviewEveryAdmissionFreezesClaimAndRoundPolicy(t *testing.T) {
	for _, kind := range []string{"ordinary", "challenge", "provisional-challenge", "contradiction", "conjecture"} {
		t.Run(kind, func(t *testing.T) {
			k, ctx, _ := setupKnowledgeTestWithBank(t)
			require.NoError(t, k.EnableRecordIntegrity(ctx))
			require.NoError(t, k.EnableReviewNeutrality(ctx))
			ctx = ctx.WithChainID("neutral-admission-test")
			ms := keeper.NewMsgServerImpl(k)
			author := makeValidBech32Addr("same-author-and-challenger")
			target := &types.Fact{Id: "target", Submitter: author, Domain: "physics", Category: "empirical", Confidence: 800000, Status: types.FactStatus_FACT_STATUS_VERIFIED}
			if kind == "provisional-challenge" {
				target.Status = types.FactStatus_FACT_STATUS_PROVISIONAL
				target.ClaimType = types.ClaimType_CLAIM_TYPE_CONJECTURE
			}
			require.NoError(t, k.SetFact(ctx, target))
			var err error
			switch kind {
			case "ordinary":
				_, err = ms.SubmitClaim(ctx, &types.MsgSubmitClaim{Submitter: author, FactContent: "A reproducible observation with a retained test procedure.", Domain: "physics", Category: "empirical", Stake: "11000000"})
			case "challenge":
				_, err = ms.ChallengeFact(ctx, &types.MsgChallengeFact{Challenger: author, FactId: target.Id, Stake: "11000000", Reason: "The repeated measurement differs under the specified condition."})
			case "provisional-challenge":
				_, err = ms.ChallengeProvisionalFact(ctx, &types.MsgChallengeProvisionalFact{Challenger: author, FactId: target.Id, Stake: "11000000", Reason: "The proposed falsification condition holds in this example.", CounterClaim: "The recorded counterexample contradicts the conjecture."})
			case "contradiction":
				_, err = ms.SubmitContradiction(ctx, &types.MsgSubmitContradiction{Submitter: author, FactId: target.Id, Stake: "11000000", CounterClaim: "A counterexample to the recorded assertion is reproducible.", Category: "empirical"})
			case "conjecture":
				_, err = ms.PostConjecture(ctx, &types.MsgPostConjecture{Proposer: author, Statement: "The conjecture describes an untested response under pressure.", FalsificationPredicate: "A reproducible counterexample under the specified pressure.", Domain: "physics", Category: "empirical", Stake: "11000000"})
			}
			require.NoError(t, err)
			var claims []*types.Claim
			k.IterateClaims(ctx, func(claim *types.Claim) bool { claims = append(claims, claim); return false })
			require.Len(t, claims, 1)
			require.Equal(t, uint32(1), claims[0].ReviewPolicyVersion)
			round, found := k.GetRoundByClaimID(ctx, claims[0].Id)
			require.True(t, found)
			require.Equal(t, uint32(1), round.ReviewPolicyVersion)
			require.Equal(t, uint32(2), round.CommitmentScheme)
			require.Equal(t, author, claims[0].Submitter)
		})
	}
}

func TestNeutralReviewFundedBootstrapCannotSponsorNewClaim(t *testing.T) {
	k, ctx, bank := setupKnowledgeTestWithBank(t)
	require.NoError(t, k.EnableRecordIntegrity(ctx))
	require.NoError(t, k.EnableReviewNeutrality(ctx))
	ctx = ctx.WithChainID("neutral-bootstrap-test")
	author := makeValidBech32Addr("fresh-sponsored-author")
	before := k.GetBootstrapFundBalance(ctx)
	require.True(t, before.IsPositive())
	sends := len(bank.sendCalls)
	events := len(ctx.EventManager().Events())
	_, err := keeper.NewMsgServerImpl(k).SubmitClaim(ctx, &types.MsgSubmitClaim{Submitter: author, FactContent: "A sponsored claim cannot turn a related review panel into a free withdrawal.", Domain: "physics", Category: "empirical", Stake: "11000000", Sponsored: true})
	require.ErrorContains(t, err, "automatic bootstrap sponsorship is retired")
	require.Equal(t, before, k.GetBootstrapFundBalance(ctx))
	require.Len(t, bank.sendCalls, sends)
	require.Len(t, ctx.EventManager().Events(), events)
	require.Zero(t, k.GetBootstrapClaimCount(ctx, author))
	claims := 0
	k.IterateClaims(ctx, func(*types.Claim) bool { claims++; return false })
	require.Zero(t, claims)
}

// Implements only the methods a legacy qualification lookup may call. A neutral
// round must remain open to this admitted/funded reviewer even when the old
// agreement-derived qualification service denies eligibility.
type neutralRejectQualification struct{ calls int }

func (q *neutralRejectQualification) IsQualified(context.Context, string, string) (bool, error) {
	q.calls++
	return false, nil
}
func (q *neutralRejectQualification) GetQualifiedValidators(context.Context, string) ([]string, error) {
	q.calls++
	return []string{"one", "two", "three", "four", "five", "six"}, nil
}
func (q *neutralRejectQualification) GetQualificationWeight(context.Context, string, string) (uint64, error) {
	q.calls++
	return 0, nil
}
func (q *neutralRejectQualification) RecordVerificationOutcome(context.Context, string, string, bool) error {
	q.calls++
	return nil
}

func TestNeutralReviewCommitRetainsBalanceGateWithoutOldQualification(t *testing.T) {
	k, ctx, bank := setupKnowledgeTestWithBank(t)
	require.NoError(t, k.EnableRecordIntegrity(ctx))
	require.NoError(t, k.EnableReviewNeutrality(ctx))
	ctx = ctx.WithChainID("neutral-admission-test")
	author := makeValidBech32Addr("neutral-reviewer")
	claim := &types.Claim{Id: "neutral-review-claim", Submitter: author, Domain: "physics", Category: "empirical", ReviewPolicyVersion: 1}
	require.NoError(t, k.SetClaim(ctx, claim))
	round, err := k.CreateVerificationRound(ctx, claim)
	require.NoError(t, err)
	qualification := new(neutralRejectQualification)
	k.SetDomainQualificationKeeper(qualification)
	ms := keeper.NewMsgServerImpl(k)
	msg := &types.MsgSubmitCommitment{Verifier: author, RoundId: round.Id, CommitHash: bytes.Repeat([]byte{0xaa}, 32)}
	_, err = ms.SubmitCommitment(ctx, msg)
	require.ErrorContains(t, err, "minimum balance")
	bank.balances[author] = sdk.NewCoins(sdk.NewInt64Coin("uzrn", 100_000_000))
	_, err = ms.SubmitCommitment(ctx, msg)
	require.NoError(t, err)
	require.Zero(t, qualification.calls)
}

func TestNeutralReviewChallengeAmountBoundPrecedesFunding(t *testing.T) {
	for _, kind := range []string{"challenge", "provisional", "contradiction"} {
		t.Run(kind, func(t *testing.T) {
			k, ctx, bank := setupKnowledgeTestWithBank(t)
			require.NoError(t, k.EnableRecordIntegrity(ctx))
			require.NoError(t, k.EnableReviewNeutrality(ctx))
			ctx = ctx.WithChainID("neutral-amount-bound")
			target := &types.Fact{Id: "amount-bound-target", Domain: "physics", Category: "empirical", Status: types.FactStatus_FACT_STATUS_VERIFIED}
			if kind == "provisional" {
				target.Status = types.FactStatus_FACT_STATUS_PROVISIONAL
			}
			require.NoError(t, k.SetFact(ctx, target))
			sends := len(bank.sendCalls)
			events := len(ctx.EventManager().Events())
			author := makeValidBech32Addr("large-collateral")
			const amount = "18446744073709551616"
			ms := keeper.NewMsgServerImpl(k)
			var err error
			switch kind {
			case "challenge":
				_, err = ms.ChallengeFact(ctx, &types.MsgChallengeFact{Challenger: author, FactId: target.Id, Stake: amount, Reason: "The bounded funding test carries a valid declared reason."})
			case "provisional":
				_, err = ms.ChallengeProvisionalFact(ctx, &types.MsgChallengeProvisionalFact{Challenger: author, FactId: target.Id, Stake: amount, Reason: "The bounded funding test carries a valid declared reason."})
			case "contradiction":
				_, err = ms.SubmitContradiction(ctx, &types.MsgSubmitContradiction{Submitter: author, FactId: target.Id, Stake: amount})
			}
			require.ErrorContains(t, err, "uint64 accounting range")
			require.Len(t, bank.sendCalls, sends)
			require.Len(t, ctx.EventManager().Events(), events)
			after, found := k.GetFact(ctx, target.Id)
			require.True(t, found)
			require.Equal(t, target.Status, after.Status)
		})
	}
}
