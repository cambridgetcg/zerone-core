package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Reviewer seat deposit, second pass (review finding: marker-off refusals and
// the raw-encoding wiring were untested). Seat-deposit state that reached a
// store some other way than through the marker, such as store damage or a
// marker removed under it, is refused wherever it would act, and malformed
// encodings of the new fields are refused before decoding can merge them.

// Section 7: with the marker off, a stored round that carries seat terms takes
// no commitment, and its completion cannot freeze returns: the handler
// refuses before anything moves, and completion fails closed, leaving the
// round, its seats and the custody as they were.
func TestSeatDepositMarkerOffRefusesStoredSeatTerms(t *testing.T) {
	t.Run("commitment", func(t *testing.T) {
		f := newSeatInternalFixture(t)
		response, err := f.server.SubmitClaim(f.ctx, &types.MsgSubmitClaim{Submitter: seatInternalAddr("author-off"), FactContent: "A reproducible synthetic observation off with a retained test procedure.", Domain: "physics", Category: "empirical", Stake: "11000000"})
		require.NoError(t, err)
		claim, found := f.k.GetClaim(f.ctx, response.ClaimId)
		require.True(t, found)
		round := f.stored(t, claim.VerificationRoundId)
		require.NotNil(t, round.SeatTerms)
		f.ctx.KVStore(f.key).Delete(types.SeatDepositEnabledKey)

		reviewer := seatInternalAddr("rv-off")
		f.bank.balances[reviewer] = 100_000_000
		hash, err := types.ComputeReviewCommitmentV2(round.CommitmentChainId, round.Id, reviewer, "accept", 800_000, []byte("seat-off-salt-000"), seatAttestationInternal)
		require.NoError(t, err)
		before := storeSnapshotOf(t, f.ctx, f.key)
		_, err = f.server.SubmitCommitment(f.ctx, &types.MsgSubmitCommitment{Verifier: reviewer, RoundId: round.Id, CommitHash: hash})
		require.ErrorContains(t, err, "has seat terms but the seat-deposit activation is off")
		require.Equal(t, before, storeSnapshotOf(t, f.ctx, f.key), "nothing is stored")
		require.Equal(t, int64(100_000_000), f.bank.balances[reviewer], "nothing moves")
		require.Zero(t, f.bank.custody)
	})
	t.Run("completion", func(t *testing.T) {
		f := newSeatInternalFixture(t)
		round, _ := f.acceptPanel(t, "off")
		f.ctx.KVStore(f.key).Delete(types.SeatDepositEnabledKey)
		f.ctx = f.ctx.WithEventManager(sdk.NewEventManager())
		require.NoError(t, f.k.BeginBlocker(f.ctx), "a failed completion is retried, not a failed block")
		stuck := f.stored(t, round.Id)
		require.NotEqual(t, types.VerificationPhase_VERIFICATION_PHASE_COMPLETE, stuck.Phase, "completion refused")
		require.Nil(t, stuck.SeatDepositReturns)
		require.Len(t, stuck.Commits, 3)
		require.Equal(t, int64(300_000_000), f.bank.custody)
		require.Zero(t, countSeatEvents(f.ctx, "zerone.knowledge.seat_deposit_returns_accrued", ""))
		_, err := roundsExecution{Keeper: f.k}.freezeSeatDepositReturns(stuck, stuck.RevealDeadline)
		require.ErrorContains(t, err, "has seat terms but the seat-deposit activation is off")
	})
}

var seatAttestationInternal = &types.ReviewAttestation{Reason: "Synthetic seat-deposit review of the stated record only."}

// Section 7: with record integrity off, the legacy setter refuses a round that
// carries any seat-deposit field (terms, a returns plan or a seat's deposit),
// even in a legacy-scheme round it would otherwise store.
func TestSeatDepositLegacySetterRefusesSeatFields(t *testing.T) {
	gs := types.DefaultGenesis()
	gs.ReviewNeutralityEnabled, gs.ClaimRecordsEnabled, gs.FundSettlementEnabled, gs.ClaimIntegrityEnabled = false, false, false, false
	gs.RecordIntegrityEnabled, gs.SeatDepositEnabled = false, false
	f := setupRelationGenesis(t, gs)
	legacy := func() *types.VerificationRound {
		return &types.VerificationRound{Id: "legacy-seat-round", ClaimId: "legacy-seat-claim", Phase: types.VerificationPhase_VERIFICATION_PHASE_COMMIT,
			StartedAtBlock: 100, CommitDeadline: 102, RevealDeadline: 104, AggregationDeadline: 106,
			Commits: []*types.CommitEntry{{Verifier: seatInternalAddr("legacy-rv"), CommitHash: make([]byte, 32), CommittedAtBlock: 100}}}
	}
	require.NoError(t, f.k.SetVerificationRound(f.ctx, legacy()), "the legacy round itself is stored")
	for name, mutate := range map[string]func(*types.VerificationRound){
		"terms":        func(r *types.VerificationRound) { r.SeatTerms = &types.ReviewSeatTerms{CountedSeatDeposit: "1"} },
		"seat deposit": func(r *types.VerificationRound) { r.Commits[0].SeatDeposit = "1" },
		"returns plan": func(r *types.VerificationRound) {
			r.SeatDepositReturns = &types.SeatDepositReturnSettlement{CreatedAtBlock: 104}
		},
	} {
		t.Run(name, func(t *testing.T) {
			round := legacy()
			mutate(round)
			require.True(t, types.RoundCarriesSeatDepositFields(round))
			require.ErrorContains(t, f.k.SetVerificationRound(f.ctx, round), "new round fields require record-integrity activation")
			stored, found := f.k.GetVerificationRound(f.ctx, round.Id)
			require.True(t, found)
			require.False(t, types.RoundCarriesSeatDepositFields(stored))
		})
	}
}

// Section 7, raw encodings: a stored round whose seat terms appear twice, or
// one of whose commitments carries its seat deposit twice, decodes without
// error (protobuf merges or keeps the last), so the raw walk is what refuses
// it. Every keeper read of a round runs that walk: the checked getter, the
// unchecked getter (which reports the round absent), the full scan behind
// custody and import inventories, the pending-index rebuild, and the setter's
// read of the prior record.
func TestSeatDepositRawEncodingRefusedOnEveryRead(t *testing.T) {
	f := newSeatInternalFixture(t)
	round, _ := f.acceptPanel(t, "raw")
	stored := f.stored(t, round.Id)
	valid := f.ctx.KVStore(f.key).Get(types.RoundKey(round.Id))
	require.NotNil(t, valid)

	terms, err := proto.Marshal(stored.SeatTerms)
	require.NoError(t, err)
	duplicateTerms := protowire.AppendBytes(protowire.AppendTag(append([]byte(nil), valid...), types.RoundSeatTermsField, protowire.BytesType), terms)

	withoutCommits := proto.Clone(stored).(*types.VerificationRound)
	withoutCommits.Commits = nil
	duplicateDeposit, err := proto.Marshal(withoutCommits)
	require.NoError(t, err)
	for i, commit := range stored.Commits {
		encoded, err := proto.Marshal(commit)
		require.NoError(t, err)
		if i == 0 {
			encoded = protowire.AppendString(protowire.AppendTag(encoded, types.CommitSeatDepositField, protowire.BytesType), commit.SeatDeposit)
		}
		duplicateDeposit = protowire.AppendBytes(protowire.AppendTag(duplicateDeposit, types.RoundCommitsField, protowire.BytesType), encoded)
	}

	for name, tc := range map[string]struct {
		data []byte
		want string
	}{
		"terms twice":        {duplicateTerms, "duplicate seat deposit field 18"},
		"seat deposit twice": {duplicateDeposit, "duplicate commitment seat deposit"},
	} {
		t.Run(name, func(t *testing.T) {
			var decoded types.VerificationRound
			require.NoError(t, proto.Unmarshal(tc.data, &decoded), "protobuf alone accepts the encoding")
			cache, _ := f.ctx.CacheContext()
			cache.KVStore(f.key).Set(types.RoundKey(round.Id), tc.data)

			_, err := f.k.getVerificationRoundChecked(cache, round.Id)
			require.ErrorContains(t, err, tc.want, "checked getter")
			_, found := f.k.GetVerificationRound(cache, round.Id)
			require.False(t, found, "unchecked getter")
			_, err = f.k.GetAllVerificationRoundsChecked(cache)
			require.ErrorContains(t, err, tc.want, "full scan")
			_, _, err = f.k.SeatDepositCustody(cache)
			require.ErrorContains(t, err, tc.want, "custody")
			require.ErrorContains(t, f.k.RebuildPendingVerifierRewardIndex(cache), tc.want, "pending-index rebuild")
			require.ErrorContains(t, f.k.SetVerificationRound(cache, stored), tc.want, "setter's prior record")
		})
	}
}
