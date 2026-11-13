package keeper_test

import (
	"fmt"
	"testing"

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
