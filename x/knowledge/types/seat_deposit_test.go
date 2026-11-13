package types_test

import (
	"bytes"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// seatRound is an open policy-1, scheme-2 round with seat terms and n seats,
// each holding the round's deposit.
func seatRound(t *testing.T, n int, deposit string) *types.VerificationRound {
	t.Helper()
	r := &types.VerificationRound{Id: "seat-round", ClaimId: "seat-claim", Phase: types.VerificationPhase_VERIFICATION_PHASE_REVEAL,
		StartedAtBlock: 10, CommitDeadline: 20, RevealDeadline: 30, AggregationDeadline: 40,
		CommitmentScheme: types.CommitmentSchemeReviewV2, CommitmentChainId: "seat-test", ReviewPolicyVersion: types.ReviewPolicyNeutral,
		SeatTerms: &types.ReviewSeatTerms{CountedSeatDeposit: deposit}}
	for i := 0; i < n; i++ {
		address := sdk.AccAddress(bytes.Repeat([]byte{byte(i + 90)}, 20)).String()
		r.SelectedVerifiers = append(r.SelectedVerifiers, address)
		r.Commits = append(r.Commits, &types.CommitEntry{Verifier: address, CommitHash: bytes.Repeat([]byte{byte(i + 1)}, 32), CommittedAtBlock: 11, SeatDeposit: deposit})
	}
	require.NoError(t, types.ValidateVerificationRoundRecord(r, true))
	return r
}

func completeSeatRound(t *testing.T, r *types.VerificationRound, height uint64) {
	t.Helper()
	r.Phase = types.VerificationPhase_VERIFICATION_PHASE_COMPLETE
	r.Verdict = types.Verdict_VERDICT_INCONCLUSIVE
	r.VerdictBlock = height
	if r.SeatDepositReturns == nil {
		r.SeatDepositReturns = types.BuildSeatDepositReturns(r, height)
	}
	require.NoError(t, types.ValidateVerificationRoundRecord(r, true))
}

func TestSeatDepositDecodedRoundRules(t *testing.T) {
	cases := map[string]func(r *types.VerificationRound){
		"deposit without terms": func(r *types.VerificationRound) { r.SeatTerms = nil },
		"plan without terms": func(r *types.VerificationRound) {
			r.SeatTerms = nil
			r.Commits[0].SeatDeposit = ""
			r.Commits[1].SeatDeposit = ""
			r.SeatDepositReturns = &types.SeatDepositReturnSettlement{CreatedAtBlock: 40}
		},
		"zero deposit":                  func(r *types.VerificationRound) { setDeposit(r, "0") },
		"empty deposit":                 func(r *types.VerificationRound) { setDeposit(r, "") },
		"leading zero":                  func(r *types.VerificationRound) { setDeposit(r, "0100") },
		"signed":                        func(r *types.VerificationRound) { setDeposit(r, "+100") },
		"seat without deposit":          func(r *types.VerificationRound) { r.Commits[1].SeatDeposit = "" },
		"seat with another deposit":     func(r *types.VerificationRound) { r.Commits[1].SeatDeposit = "101" },
		"policy 0":                      func(r *types.VerificationRound) { r.ReviewPolicyVersion = types.ReviewPolicyLegacy },
		"expired":                       func(r *types.VerificationRound) { r.Phase = types.VerificationPhase_VERIFICATION_PHASE_EXPIRED },
		"open plan before the deadline": func(r *types.VerificationRound) { r.SeatDepositReturns = types.BuildSeatDepositReturns(r, 39) },
		"empty plan": func(r *types.VerificationRound) {
			r.SeatDepositReturns = &types.SeatDepositReturnSettlement{CreatedAtBlock: 40}
		},
		"plan missing a seat": func(r *types.VerificationRound) {
			r.SeatDepositReturns = types.BuildSeatDepositReturns(r, 40)
			r.SeatDepositReturns.Returns = r.SeatDepositReturns.Returns[:1]
		},
		"plan to another address": func(r *types.VerificationRound) {
			r.SeatDepositReturns = types.BuildSeatDepositReturns(r, 40)
			r.SeatDepositReturns.Returns[0].Recipient = r.Commits[1].Verifier
		},
		"plan out of order": func(r *types.VerificationRound) {
			r.SeatDepositReturns = types.BuildSeatDepositReturns(r, 40)
			p := r.SeatDepositReturns.Returns
			p[0], p[1] = p[1], p[0]
		},
		"plan for less": func(r *types.VerificationRound) {
			r.SeatDepositReturns = types.BuildSeatDepositReturns(r, 40)
			r.SeatDepositReturns.Returns[0].Amount = "99"
		},
		"paid before frozen": func(r *types.VerificationRound) {
			r.SeatDepositReturns = types.BuildSeatDepositReturns(r, 40)
			r.SeatDepositReturns.Returns[0].PaidAtBlock = 39
		},
		"completed plan from another height": func(r *types.VerificationRound) {
			r.Phase = types.VerificationPhase_VERIFICATION_PHASE_COMPLETE
			r.VerdictBlock = 30
			r.SeatDepositReturns = types.BuildSeatDepositReturns(r, 31)
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			r := seatRound(t, 2, "100")
			mutate(r)
			require.Error(t, types.ValidateVerificationRoundRecord(r, true))
		})
	}

	t.Run("completed without a plan", func(t *testing.T) {
		r := seatRound(t, 2, "100")
		r.Phase = types.VerificationPhase_VERIFICATION_PHASE_COMPLETE
		r.VerdictBlock = 30
		// Completion validates the round in memory before it freezes the
		// plan; what is stored or imported must hold it.
		require.NoError(t, types.ValidateVerificationRoundRecord(r, true))
		require.ErrorContains(t, types.ValidateStoredSeatDeposits(r), "must hold its seat deposit returns")
		require.Error(t, types.ValidateSeatDepositUpdate(nil, r))
	})

	t.Run("valid shapes", func(t *testing.T) {
		require.NoError(t, types.ValidateVerificationRoundRecord(seatRound(t, 0, "1"), true), "terms with no seat")
		r := seatRound(t, 2, "18446744073709551615")
		r.SeatDepositReturns = types.BuildSeatDepositReturns(r, 40)
		require.NoError(t, types.ValidateVerificationRoundRecord(r, true), "an open round frozen at its deadline")
		completeSeatRound(t, r, 45)
		require.NoError(t, types.ValidateVerificationRoundRecord(r, true), "a later completion keeps the deadline plan")
		r = seatRound(t, 2, "100")
		completeSeatRound(t, r, 30)
		r.SeatDepositReturns.Returns[1].PaidAtBlock = 31
		require.NoError(t, types.ValidateVerificationRoundRecord(r, true), "a partly paid plan")
		empty := seatRound(t, 0, "100")
		empty.Phase = types.VerificationPhase_VERIFICATION_PHASE_COMPLETE
		empty.VerdictBlock = 30
		require.NoError(t, types.ValidateVerificationRoundRecord(empty, true), "a completed round with no seat needs no plan")
	})
}

func setDeposit(r *types.VerificationRound, deposit string) {
	r.SeatTerms.CountedSeatDeposit = deposit
	for _, c := range r.Commits {
		c.SeatDeposit = deposit
	}
}

func TestSeatDepositUpdateRule(t *testing.T) {
	before := seatRound(t, 2, "100")
	completeSeatRound(t, before, 30)

	t.Run("unpaid to paid, once", func(t *testing.T) {
		after := proto.Clone(before).(*types.VerificationRound)
		after.SeatDepositReturns.Returns[0].PaidAtBlock = 31
		require.NoError(t, types.ValidateSeatDepositUpdate(before, after))
		again := proto.Clone(after).(*types.VerificationRound)
		again.SeatDepositReturns.Returns[0].PaidAtBlock = 32
		require.ErrorContains(t, types.ValidateSeatDepositUpdate(after, again), "paid seat deposit return")
		back := proto.Clone(after).(*types.VerificationRound)
		back.SeatDepositReturns.Returns[0].PaidAtBlock = 0
		require.ErrorContains(t, types.ValidateSeatDepositUpdate(after, back), "paid seat deposit return")
	})
	t.Run("plan cannot be removed or repriced", func(t *testing.T) {
		removed := proto.Clone(before).(*types.VerificationRound)
		removed.SeatDepositReturns = nil
		require.Error(t, types.ValidateSeatDepositUpdate(before, removed))
		moved := proto.Clone(before).(*types.VerificationRound)
		moved.SeatDepositReturns.CreatedAtBlock = 29
		require.Error(t, types.ValidateSeatDepositUpdate(before, moved))
	})
	t.Run("terms cannot appear, change or disappear", func(t *testing.T) {
		open := seatRound(t, 0, "100")
		changed := proto.Clone(open).(*types.VerificationRound)
		changed.SeatTerms.CountedSeatDeposit = "200"
		require.ErrorContains(t, types.ValidateSeatDepositUpdate(open, changed), "immutable")
		gone := proto.Clone(open).(*types.VerificationRound)
		gone.SeatTerms = nil
		require.ErrorContains(t, types.ValidateSeatDepositUpdate(open, gone), "immutable")
		require.ErrorContains(t, types.ValidateSeatDepositUpdate(gone, open), "immutable")
		require.NoError(t, types.ValidateSeatDepositUpdate(nil, open), "a new round records its terms")
	})
}

func TestSeatDepositsHeld(t *testing.T) {
	r := seatRound(t, 3, "18446744073709551615")
	held, err := types.SeatDepositsHeld(r)
	require.NoError(t, err)
	require.Equal(t, "55340232221128654845", held.String(), "three seats of the uint64 maximum do not overflow")
	completeSeatRound(t, r, 30)
	r.SeatDepositReturns.Returns[1].PaidAtBlock = 31
	held, err = types.SeatDepositsHeld(r)
	require.NoError(t, err)
	require.Equal(t, "36893488147419103230", held.String(), "paid returns are no longer held")
	none, err := types.SeatDepositsHeld(&types.VerificationRound{Id: "r", ClaimId: "c"})
	require.NoError(t, err)
	require.True(t, none.IsZero())
	require.Equal(t, 2, types.PendingSeatDepositReturns(r))
}

func TestSeatDepositGenesisPresenceAndPredecessors(t *testing.T) {
	gs := types.DefaultGenesis()
	require.True(t, gs.SeatDepositEnabled, "native genesis selects the seat deposit")
	require.NoError(t, types.ValidateSeatDepositGenesis(gs))
	gs.SeatDepositEnabled = false
	require.NoError(t, types.ValidateSeatDepositGenesis(gs))
	gs.ActiveRounds = []*types.VerificationRound{seatRound(t, 1, "100")}
	require.ErrorContains(t, types.ValidateSeatDepositGenesis(gs), "without seat_deposit_enabled")
	gs.SeatDepositEnabled = true
	require.NoError(t, types.ValidateSeatDepositGenesis(gs))
	for name, off := range map[string]func(*types.GenesisState){
		"parameter":         func(g *types.GenesisState) { g.ReviewerBalanceParamEnabled = false },
		"review neutrality": func(g *types.GenesisState) { g.ReviewNeutralityEnabled = false },
		"record integrity":  func(g *types.GenesisState) { g.RecordIntegrityEnabled = false },
	} {
		t.Run(name, func(t *testing.T) {
			g := proto.Clone(gs).(*types.GenesisState)
			off(g)
			require.ErrorContains(t, types.ValidateSeatDepositGenesis(g), "requires")
			require.NoError(t, types.ValidateSeatDepositPresence(g), "the structural check has no predecessor rule")
		})
	}
	for name, round := range map[string]*types.VerificationRound{
		"seat deposit only": {Id: "r", ClaimId: "c", Commits: []*types.CommitEntry{{Verifier: "v", SeatDeposit: "1"}}},
		"plan only":         {Id: "r", ClaimId: "c", SeatDepositReturns: &types.SeatDepositReturnSettlement{}},
	} {
		t.Run("marker off refuses "+name, func(t *testing.T) {
			g := types.DefaultGenesis()
			g.SeatDepositEnabled = false
			g.CompletedRounds = []*types.VerificationRound{round}
			require.ErrorContains(t, types.ValidateSeatDepositGenesis(g), "without seat_deposit_enabled")
		})
	}
}

// encodeRound is the deterministic store encoding of a round.
func encodeRound(t *testing.T, r *types.VerificationRound) []byte {
	t.Helper()
	bz, err := proto.MarshalOptions{Deterministic: true}.Marshal(r)
	require.NoError(t, err)
	return bz
}

func appendMessage(b []byte, num protowire.Number, value []byte) []byte {
	b = protowire.AppendTag(b, num, protowire.BytesType)
	return protowire.AppendBytes(b, value)
}

func TestSeatDepositRawEncodingRules(t *testing.T) {
	r := seatRound(t, 2, "100")
	r.SeatDepositReturns = types.BuildSeatDepositReturns(r, 40)
	valid := encodeRound(t, r)
	require.NoError(t, types.ValidateRawRoundSeatDeposit(valid))
	legacy := encodeRound(t, &types.VerificationRound{Id: "r", ClaimId: "c", Commits: []*types.CommitEntry{{Verifier: "v", CommitHash: []byte{1}}}})
	require.NoError(t, types.ValidateRawRoundSeatDeposit(legacy), "a record without the fields passes")

	termsBytes, err := proto.Marshal(r.SeatTerms)
	require.NoError(t, err)
	planBytes, err := proto.Marshal(r.SeatDepositReturns)
	require.NoError(t, err)
	retBytes, err := proto.Marshal(r.SeatDepositReturns.Returns[0])
	require.NoError(t, err)
	commitBytes, err := proto.Marshal(r.Commits[0])
	require.NoError(t, err)

	// Each case starts from its own copy, so no two cases share bytes.
	L := func() []byte { return bytes.Clone(legacy) }
	cases := map[string][]byte{
		"duplicate terms":          appendMessage(bytes.Clone(valid), 18, termsBytes),
		"duplicate plan":           appendMessage(bytes.Clone(valid), 19, planBytes),
		"empty terms":              appendMessage(L(), 18, nil),
		"empty plan":               appendMessage(L(), 19, nil),
		"terms as varint":          protowire.AppendVarint(protowire.AppendTag(L(), 18, protowire.VarintType), 1),
		"terms unknown field 2":    appendMessage(L(), 18, appendMessage(bytes.Clone(termsBytes), 2, []byte("20000"))),
		"terms unknown field 3":    appendMessage(L(), 18, protowire.AppendVarint(protowire.AppendTag(bytes.Clone(termsBytes), 3, protowire.VarintType), 32)),
		"terms duplicate amount":   appendMessage(L(), 18, appendMessage(bytes.Clone(termsBytes), 1, []byte("100"))),
		"terms noncanonical tag":   appendMessage(L(), 18, append([]byte{0x8a, 0x00, 3}, []byte("100")...)),
		"plan unknown field":       appendMessage(L(), 19, appendMessage(bytes.Clone(planBytes), 3, nil)),
		"plan duplicate height":    appendMessage(L(), 19, protowire.AppendVarint(protowire.AppendTag(bytes.Clone(planBytes), 1, protowire.VarintType), 40)),
		"plan noncanonical height": appendMessage(L(), 19, []byte{0x08, 0xa8, 0x80, 0x00}),
		"return unknown field":     appendMessage(L(), 19, appendMessage(nil, 2, appendMessage(bytes.Clone(retBytes), 4, nil))),
		"return duplicate amount":  appendMessage(L(), 19, appendMessage(nil, 2, appendMessage(bytes.Clone(retBytes), 2, []byte("1")))),
		"empty return":             appendMessage(L(), 19, appendMessage(nil, 2, nil)),
		"seat duplicate deposit":   appendMessage(L(), 6, appendMessage(bytes.Clone(commitBytes), 4, []byte("100"))),
		"seat empty deposit":       appendMessage(L(), 6, appendMessage(nil, 4, nil)),
		"seat deposit as varint":   appendMessage(L(), 6, protowire.AppendVarint(protowire.AppendTag(nil, 4, protowire.VarintType), 100)),
		"noncanonical length":      append(append(L(), 0x92, 0x01, 0x83, 0x00), termsBytes[:3]...),
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			require.Error(t, types.ValidateRawRoundSeatDeposit(data))
		})
	}
}

// Step C will add the uncounted-seat deposit and limit as fields of
// ReviewSeatTerms. Until its own marker exists this binary refuses them, raw
// and decoded, rather than reading a round it does not understand.
func TestSeatDepositRefusesUnknownSeatTermsFields(t *testing.T) {
	r := seatRound(t, 1, "100")
	termsBytes, err := proto.Marshal(r.SeatTerms)
	require.NoError(t, err)
	r.SeatTerms = nil
	r.Commits[0].SeatDeposit = ""
	base := encodeRound(t, r)
	withFuture := appendMessage(bytes.Clone(base), 18, protowire.AppendVarint(protowire.AppendTag(appendMessage(bytes.Clone(termsBytes), 2, []byte("20000")), 3, protowire.VarintType), 32))
	require.Error(t, types.ValidateRawRoundSeatDeposit(withFuture))
	decoded := new(types.VerificationRound)
	require.NoError(t, proto.Unmarshal(withFuture, decoded))
	decoded.Commits[0].SeatDeposit = "100"
	require.ErrorContains(t, types.ValidateRoundSeatDeposits(decoded), "unsupported seat terms fields")
}
