package keeper_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Reviewer seat deposit (the reviewer-entry design draft, pull request #190,
// step B) at keeper level. The app-level tests in
// app/seat_deposit_settlement_test.go drive the same rules through the real
// SDK bank.

// D1, D7: only a review-policy-1, scheme-2 round records seat terms. A
// policy-0 round created while the seat-deposit marker is on gets none and
// keeps the balance check.
func TestSeatDepositTermsOnlyForPolicy1Scheme2(t *testing.T) {
	for _, recordIntegrity := range []bool{false, true} {
		t.Run(map[bool]string{false: "scheme-0", true: "scheme-2"}[recordIntegrity], func(t *testing.T) {
			k, ctx, _ := setupKnowledgeTestWithBank(t)
			if recordIntegrity {
				require.NoError(t, k.EnableRecordIntegrity(ctx))
			}
			require.NoError(t, k.EnableSeatDeposit(ctx))
			ctx = ctx.WithChainID("seat-policy-0-test")
			_, err := keeper.NewMsgServerImpl(k).SubmitClaim(ctx, &types.MsgSubmitClaim{Submitter: makeValidBech32Addr("seat-policy-0-author"), FactContent: "A reproducible observation with a retained test procedure.", Domain: "physics", Category: "empirical", Stake: "11000000"})
			require.NoError(t, err)
			var rounds []*types.VerificationRound
			k.IterateActiveRounds(ctx, func(r *types.VerificationRound) bool { rounds = append(rounds, r); return false })
			require.Len(t, rounds, 1)
			require.Equal(t, types.ReviewPolicyLegacy, rounds[0].ReviewPolicyVersion)
			require.Nil(t, rounds[0].SeatTerms)
		})
	}
}

// Genesis carrying any seat-deposit field with the marker off, or the marker
// without its predecessors, is refused by validation and by InitGenesis
// before either import path writes anything.
func TestSeatDepositGenesisRefusedBeforeAnyWrite(t *testing.T) {
	seatRound := &types.VerificationRound{Id: "seat-genesis-round", ClaimId: "seat-genesis-claim", Phase: types.VerificationPhase_VERIFICATION_PHASE_COMMIT,
		StartedAtBlock: 1, CommitDeadline: 3, RevealDeadline: 5, AggregationDeadline: 7,
		CommitmentScheme: types.CommitmentSchemeReviewV2, CommitmentChainId: "seat-genesis", ReviewPolicyVersion: types.ReviewPolicyNeutral,
		SeatTerms: &types.ReviewSeatTerms{CountedSeatDeposit: "100000000"}}
	for name, mutate := range map[string]func(*types.GenesisState){
		"terms with the marker off": func(gs *types.GenesisState) { gs.ActiveRounds = []*types.VerificationRound{seatRound} },
		"marker without the parameter marker": func(gs *types.GenesisState) {
			gs.SeatDepositEnabled = true
			gs.ReviewerBalanceParamEnabled = false
		},
	} {
		for _, accounting := range []bool{false, true} {
			t.Run(name+map[bool]string{false: "", true: "/accounting"}[accounting], func(t *testing.T) {
				gs := types.DefaultGenesis()
				if accounting {
					gs.TrainingFundAllocation = "0"
					gs.AccountingState = &types.KnowledgeGenesisAccounting{TrainingFundNonEscrowBalance: "0"}
				}
				mutate(gs)
				require.Error(t, gs.Validate())
				f := newKnowledgeStoreFixture(t, "seat-genesis")
				require.ErrorContains(t, f.k.InitGenesis(f.ctx, gs), "seat_deposit_enabled")
				it := f.ctx.KVStore(f.key).Iterator(nil, nil)
				defer it.Close()
				require.False(t, it.Valid(), "a refused genesis writes nothing")
			})
		}
	}
}

// D2, D3 at keeper level: without zerone_auth or the account keeper wired,
// a round with seat terms fails closed; with them, a deposit moves into the
// seat-deposit account and is recorded with the seat. A refused commitment
// moves and stores nothing.
func TestSeatDepositHandlerFailsClosedWithoutItsKeepers(t *testing.T) {
	f := newSeatFixture(t, nil)
	round := f.claim(t, "fail-closed")
	require.NotNil(t, round.SeatTerms)
	reviewer := f.reviewer("seat-fail-closed-rv", 200_000_000)

	withoutAuth := f.k
	withoutAuth.SetZeroneAuthKeeper(nil)
	err := f.commitThrough(t, keeper.NewMsgServerImpl(withoutAuth), round, reviewer)
	require.ErrorIs(t, err, types.ErrReviewerNotAdmitted)
	require.ErrorContains(t, err, "zerone_auth is unavailable")

	withoutAccounts := f.k
	withoutAccounts.SetAccountKeeper(nil)
	require.ErrorContains(t, f.commitThrough(t, keeper.NewMsgServerImpl(withoutAccounts), round, reviewer), "require the account keeper")

	delete(f.auth.admitted, reviewer)
	require.ErrorIs(t, f.commit(t, round, reviewer, "accept"), types.ErrReviewerNotAdmitted)
	require.Empty(t, f.round(t, round.Id).Commits)
	require.Equal(t, int64(200_000_000), f.bank.balance(reviewer))
	require.Zero(t, f.bank.custody())

	f.auth.admitted[reviewer] = true
	require.NoError(t, f.commit(t, round, reviewer, "accept"))
	require.Equal(t, int64(100_000_000), f.bank.balance(reviewer))
	require.Equal(t, int64(100_000_000), f.bank.custody())
	stored := f.round(t, round.Id)
	require.Len(t, stored.Commits, 1)
	require.Equal(t, "100000000", stored.Commits[0].SeatDeposit)
}

func (f *seatFixture) commitThrough(t *testing.T, server types.MsgServer, round *types.VerificationRound, reviewer string) error {
	t.Helper()
	hash, err := types.ComputeReviewCommitmentV2(round.CommitmentChainId, round.Id, reviewer, "accept", 800_000, seatSalt(reviewer), seatAttestation)
	require.NoError(t, err)
	_, err = server.SubmitCommitment(f.ctx, &types.MsgSubmitCommitment{Verifier: reviewer, RoundId: round.Id, CommitHash: hash})
	return err
}

func (f *seatFixture) updateFigure(t *testing.T, value string) error {
	t.Helper()
	params, err := f.k.GetParams(f.ctx)
	require.NoError(t, err)
	next := proto.Clone(params).(*types.Params)
	next.MinReviewerBalance = value
	_, err = f.server.UpdateParams(f.ctx, &types.MsgUpdateParams{Authority: "authority", Params: next})
	return err
}

func (f *seatFixture) storedFigure(t *testing.T) string {
	t.Helper()
	params, err := f.k.GetParams(f.ctx)
	require.NoError(t, err)
	return params.MinReviewerBalance
}

// A3: while the seat-deposit marker is off, a figure below 100 ZRN is refused
// with the sentence 3a844c48 used, at genesis validation, at InitGenesis and
// when MsgUpdateParams executes, and nothing is stored.
func TestSeatDepositFloorHoldsWhileDepositsOff(t *testing.T) {
	for _, value := range []string{"99999999", "20000", "1"} {
		t.Run(value, func(t *testing.T) {
			f := newSeatFixture(t, func(gs *types.GenesisState) { gs.SeatDepositEnabled = false })
			err := f.updateFigure(t, value)
			require.ErrorContains(t, err, "invalid params: min_reviewer_balance "+value+" "+reviewerFloorRefusal)
			require.Empty(t, f.storedFigure(t))

			gs := types.DefaultGenesis()
			gs.Params.MinReviewerBalance = value
			require.ErrorContains(t, gs.Validate(), reviewerFloorRefusal)
			fresh := newKnowledgeStoreFixture(t, "seat-floor")
			require.ErrorContains(t, fresh.k.InitGenesis(fresh.ctx, gs), reviewerFloorRefusal)
		})
	}
}

// A2, A3: with the seat-deposit marker on, any canonical positive figure is
// accepted at genesis and by MsgUpdateParams, including ones below 100 ZRN;
// zero and malformed values are still refused, and empty still means 100 ZRN.
func TestSeatDepositLowerFigureAcceptedWhenDepositsOn(t *testing.T) {
	f := newSeatFixture(t, nil)
	for _, value := range []string{"1", "20000", "99999999", "100000000", ""} {
		require.NoError(t, f.updateFigure(t, value), value)
		require.Equal(t, value, f.storedFigure(t))
	}
	for _, value := range []string{"0", "020000", "-1", " 20000", "18446744073709551616"} {
		require.Error(t, f.updateFigure(t, value), value)
		require.Empty(t, f.storedFigure(t), "a refused update stores nothing")
	}
	gs := types.DefaultGenesis()
	gs.SeatDepositEnabled = true
	gs.Params.MinReviewerBalance = "20000"
	require.NoError(t, gs.Validate())
	fresh := newKnowledgeStoreFixture(t, "seat-floor-on")
	require.NoError(t, fresh.k.InitGenesis(fresh.ctx, gs))
	gs.ReviewerBalanceParamEnabled = false
	require.Error(t, gs.Validate(), "a configured figure still needs the parameter marker")
}

// A5, D7: a round without seat terms keeps the balance check, compared with
// the larger of the figure and 100 ZRN, after governance lowers the figure
// with the seat-deposit marker on. Here a policy-1 round created before the
// marker refuses a reviewer holding 99,999,999 uzrn and admits one holding
// 100,000,000, and moves nothing; a round created after the marker locks the
// lowered figure.
func TestSeatDepositRoundWithoutSeatTermsIgnoresLowerFigure(t *testing.T) {
	f := newSeatFixture(t, func(gs *types.GenesisState) { gs.SeatDepositEnabled = false })
	before := f.claim(t, "pre-activation")
	require.Nil(t, before.SeatTerms)
	require.NoError(t, f.k.EnableSeatDeposit(f.ctx))
	require.NoError(t, f.updateFigure(t, "1"))

	short := f.reviewer("seat-a5-short", 99_999_999)
	err := f.commit(t, before, short, "accept")
	require.ErrorContains(t, err, "verifier does not meet minimum balance requirement")
	enough := f.reviewer("seat-a5-enough", 100_000_000)
	require.NoError(t, f.commit(t, before, enough, "accept"))
	stored := f.round(t, before.Id)
	require.Len(t, stored.Commits, 1)
	require.Empty(t, stored.Commits[0].SeatDeposit)
	require.Zero(t, f.bank.custody())
	require.Equal(t, int64(100_000_000), f.bank.balance(enough), "a balance check moves nothing")

	after := f.claim(t, "post-activation")
	require.Equal(t, "1", after.SeatTerms.CountedSeatDeposit)
	tiny := f.reviewer("seat-a5-tiny", 1)
	require.NoError(t, f.commit(t, after, tiny, "accept"))
	require.Zero(t, f.bank.balance(tiny))
	require.Equal(t, int64(1), f.bank.custody())
}

// D7 for policy-0 rounds: with both markers on and the figure lowered, a
// legacy round still needs 100 ZRN of balance.
func TestSeatDepositPolicyZeroRoundKeepsBalanceCheck(t *testing.T) {
	f := newReviewerBalanceFixture(t, true, "")
	require.NoError(t, f.k.EnableSeatDeposit(f.ctx))
	require.NoError(t, updateReviewerBalance(t, f, "1"))
	f.requireCommitFigure(t, 100_000_000)
	for _, call := range f.bank.sendCalls {
		require.NotEqual(t, types.SeatDepositModuleName, call.to, fmt.Sprintf("no seat deposit in a policy-0 round: %v", call))
	}
}

// seatPanel seats reviewers in round, each committing to its vote; a vote of
// "" commits to accept and never reveals. It moves to the reveal window,
// reveals, and leaves the fixture at the reveal deadline before BeginBlock.
func (f *seatFixture) seatPanel(t *testing.T, round *types.VerificationRound, tag string, votes ...string) []string {
	t.Helper()
	reviewers := make([]string, len(votes))
	for i, vote := range votes {
		if vote == "" {
			vote = "accept"
		}
		reviewers[i] = f.reviewer(fmt.Sprintf("seat-%s-%d", tag, i), 100_000_000)
		require.NoError(t, f.commit(t, round, reviewers[i], vote))
	}
	f.height(int64(round.CommitDeadline))
	f.beginBlock(t)
	for i, vote := range votes {
		if vote != "" {
			f.reveal(t, round, reviewers[i], vote)
		}
	}
	f.height(int64(round.RevealDeadline))
	return reviewers
}

func eventsOf(ctx sdk.Context, kind string) []sdk.Event {
	var out []sdk.Event
	for _, event := range ctx.EventManager().Events() {
		if event.Type == kind {
			out = append(out, event)
		}
	}
	return out
}

func eventAttribute(event sdk.Event, key string) string {
	for _, attr := range event.Attributes {
		if attr.Key == key {
			return attr.Value
		}
	}
	return ""
}

// D4, D5: completion freezes one return per seat, in commit order, to the
// address that locked it, for the amount it locked, whatever the verdict and
// vote and whether the seat revealed.
func TestSeatDepositReturnsFrozenAtCompletion(t *testing.T) {
	f := newSeatFixture(t, nil)
	round := f.claim(t, "frozen")
	reviewers := f.seatPanel(t, round, "frozen", "accept", "reject", "accept", "")
	f.ctx = f.ctx.WithEventManager(sdk.NewEventManager())
	f.beginBlock(t)
	done := f.round(t, round.Id)
	require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_COMPLETE, done.Phase)
	plan := done.SeatDepositReturns
	require.NotNil(t, plan)
	require.Equal(t, done.VerdictBlock, plan.CreatedAtBlock)
	require.Len(t, plan.Returns, len(reviewers))
	for i, ret := range plan.Returns {
		require.Equal(t, reviewers[i], ret.Recipient)
		require.Equal(t, "100000000", ret.Amount)
	}
	accrued := eventsOf(f.ctx, "zerone.knowledge.seat_deposit_returns_accrued")
	require.Len(t, accrued, 1)
	require.Equal(t, "completion", eventAttribute(accrued[0], "trigger"))
	require.Equal(t, "4", eventAttribute(accrued[0], "seats"))
	require.Equal(t, "400000000", eventAttribute(accrued[0], "total_uzrn"))
	held, err := types.SeatDepositsHeld(done)
	require.NoError(t, err)
	require.Equal(t, f.bank.custody(), held.Int64())
}

// completeWithPanel seats a panel, completes the round at its reveal deadline
// and returns the completed round and the reviewers.
func (f *seatFixture) completeWithPanel(t *testing.T, tag string, votes ...string) (*types.VerificationRound, []string) {
	t.Helper()
	round := f.claim(t, tag)
	reviewers := f.seatPanel(t, round, tag, votes...)
	f.beginBlock(t)
	done := f.round(t, round.Id)
	require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_COMPLETE, done.Phase)
	require.NotNil(t, done.SeatDepositReturns)
	return done, reviewers
}

func (f *seatFixture) nextBlock(t *testing.T) {
	t.Helper()
	f.height(f.ctx.BlockHeight() + 1)
	f.ctx = f.ctx.WithEventManager(sdk.NewEventManager())
	f.beginBlock(t)
}

func pendingIndexed(f *seatFixture, roundID string) bool {
	return f.ctx.KVStore(f.key).Has(append([]byte{0x82}, roundID...))
}

// D4, D5: the pending queue pays every return once, in the block after
// completion, from the seat-deposit account; the round leaves the queue; a
// later block pays nothing again. A round whose only pending obligation is
// its returns is indexed and processed without error.
func TestSeatDepositReturnsPaidOnceThroughTheQueue(t *testing.T) {
	f := newSeatFixture(t, nil)
	done, reviewers := f.completeWithPanel(t, "queue", "accept", "reject", "accept", "")
	require.NotNil(t, done.VerifierRewardSettlement)
	require.NotZero(t, done.VerifierRewardSettlement.PaidAtBlock, "reviewer pay settles at completion, as before")
	require.True(t, pendingIndexed(f, done.Id), "returns alone keep the round in the queue")
	require.Equal(t, int64(400_000_000), f.bank.custody())

	f.nextBlock(t)
	paid := f.round(t, done.Id)
	for i, ret := range paid.SeatDepositReturns.Returns {
		require.Equal(t, uint64(f.ctx.BlockHeight()), ret.PaidAtBlock)
		require.Equal(t, int64(100_000_000), f.bank.balance(reviewers[i]), "every seat gets its deposit back, whatever it voted or whether it revealed")
		require.Equal(t, 1, f.bank.returnsTo(reviewers[i]))
	}
	require.Zero(t, f.bank.custody())
	require.False(t, pendingIndexed(f, done.Id))
	require.Len(t, eventsOf(f.ctx, "zerone.knowledge.seat_deposit_returned"), len(reviewers))

	f.nextBlock(t)
	require.True(t, proto.Equal(paid, f.round(t, done.Id)))
	for _, reviewer := range reviewers {
		require.Equal(t, 1, f.bank.returnsTo(reviewer), "nothing is paid twice")
	}
	require.Empty(t, eventsOf(f.ctx, "zerone.knowledge.seat_deposit_returned"))
}

// D5: a failed return stays pending and the retry repeats nothing already
// paid.
func TestSeatDepositFailedReturnRepeatsNothing(t *testing.T) {
	f := newSeatFixture(t, nil)
	done, reviewers := f.completeWithPanel(t, "retry", "accept", "accept", "accept")
	f.bank.failReturnTo[reviewers[1]] = true
	f.nextBlock(t)
	partly := f.round(t, done.Id)
	require.NotZero(t, partly.SeatDepositReturns.Returns[0].PaidAtBlock)
	require.Zero(t, partly.SeatDepositReturns.Returns[1].PaidAtBlock)
	require.NotZero(t, partly.SeatDepositReturns.Returns[2].PaidAtBlock)
	require.Equal(t, int64(100_000_000), f.bank.custody())
	require.True(t, pendingIndexed(f, done.Id))
	firstPaid := partly.SeatDepositReturns.Returns[0].PaidAtBlock

	delete(f.bank.failReturnTo, reviewers[1])
	f.nextBlock(t)
	settled := f.round(t, done.Id)
	require.Equal(t, firstPaid, settled.SeatDepositReturns.Returns[0].PaidAtBlock, "a paid record never moves")
	require.Equal(t, uint64(f.ctx.BlockHeight()), settled.SeatDepositReturns.Returns[1].PaidAtBlock)
	for _, reviewer := range reviewers {
		require.Equal(t, 1, f.bank.returnsTo(reviewer))
		require.Equal(t, int64(100_000_000), f.bank.balance(reviewer))
	}
	require.Zero(t, f.bank.custody())
	require.False(t, pendingIndexed(f, done.Id))
}

// D4: returns are independent of reviewer pay and refunds, in each
// direction. Reviewer pay that cannot be paid does not hold the returns, and
// returns that cannot be paid do not hold reviewer pay.
func TestSeatDepositReturnsIndependentOfReviewerPay(t *testing.T) {
	t.Run("reviewer pay fails", func(t *testing.T) {
		f := newSeatFixture(t, nil)
		f.bank.failFrom[types.ModuleName] = true
		done, reviewers := f.completeWithPanel(t, "pay-fails", "accept", "accept", "accept")
		require.Zero(t, done.VerifierRewardSettlement.PaidAtBlock)
		f.nextBlock(t)
		round := f.round(t, done.Id)
		require.Zero(t, round.VerifierRewardSettlement.PaidAtBlock, "reviewer pay stays pending")
		for i, ret := range round.SeatDepositReturns.Returns {
			require.NotZero(t, ret.PaidAtBlock)
			require.Equal(t, int64(100_000_000), f.bank.balance(reviewers[i]))
		}
		require.True(t, pendingIndexed(f, done.Id), "the round stays queued for its reviewer pay")
		delete(f.bank.failFrom, types.ModuleName)
		f.nextBlock(t)
		round = f.round(t, done.Id)
		require.NotZero(t, round.VerifierRewardSettlement.PaidAtBlock)
		require.False(t, pendingIndexed(f, done.Id))
		for _, reviewer := range reviewers {
			require.Equal(t, 1, f.bank.returnsTo(reviewer))
		}
	})
	t.Run("returns fail", func(t *testing.T) {
		f := newSeatFixture(t, nil)
		f.bank.failAll = true
		done, _ := f.completeWithPanel(t, "returns-fail", "accept", "accept", "accept")
		require.NotZero(t, done.VerifierRewardSettlement.PaidAtBlock, "reviewer pay settles at completion")
		f.nextBlock(t)
		round := f.round(t, done.Id)
		for _, ret := range round.SeatDepositReturns.Returns {
			require.Zero(t, ret.PaidAtBlock)
		}
		require.Equal(t, int64(300_000_000), f.bank.custody())
		require.True(t, pendingIndexed(f, done.Id))
	})
}

// A round with seat terms and its deposit history is retained, even for an
// unfunded claim (the seat deposit does not require fund settlement).
func TestSeatDepositRoundCannotBeDeleted(t *testing.T) {
	f := newSeatFixture(t, func(gs *types.GenesisState) {
		gs.ClaimIntegrityEnabled = false
		gs.FundSettlementEnabled = false
	})
	round := f.claim(t, "delete")
	claim, found := f.k.GetClaim(f.ctx, round.ClaimId)
	require.True(t, found)
	require.Nil(t, claim.FundingTerms)
	require.ErrorContains(t, f.k.DeleteVerificationRound(f.ctx, round.Id), "a round with seat terms and its deposit history must be retained")
	f.round(t, round.Id)
}
