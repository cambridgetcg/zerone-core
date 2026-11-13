package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

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
