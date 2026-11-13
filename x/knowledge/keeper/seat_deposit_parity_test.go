package keeper_test

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"
	"testing"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Marker-off parity (the reviewer-entry design draft, pull request #190,
// section 9): with the seat-deposit marker off, every message result, stored
// byte, event and gas figure equals 3a844c48's. seatParityDigest runs one
// scenario through the message handlers and BeginBlocker: a policy-1 claim,
// a commitment refused for its balance, three admitted commitments, reveals,
// completion and settlement, the next block's queue, and parameter updates
// that raise the figure, lower it below 100 ZRN and leave it alone. It hashes
// the whole knowledge store, every event with its attributes, each step's gas
// and each step's error text.
//
// This file compiles unchanged at 3a844c48, using only what that commit's
// test package has. The digests below were recorded by running it there, on a
// scratch export of that commit, with the reviewer-balance-param marker on and
// off; this branch must produce the same.
var seatParityDigests = map[bool]string{
	false: "93815f4dd29824b102e19c3ddf765324bc2406ee6b783310a02c47723fb6655f",
	true:  "1a0e739e181d892302dac54fd16de29f38de63a3211dfb7f523ecc75960c6a1b",
}

func seatParityDigest(t *testing.T, reviewerBalanceParam bool) string {
	t.Helper()
	gs := types.DefaultGenesis()
	gs.ReviewerBalanceParamEnabled = reviewerBalanceParam
	gs.Params.CommitPhaseBlocks, gs.Params.RevealPhaseBlocks, gs.Params.AggregationPhaseBlocks = 2, 2, 2
	gs.Params.MinVerifiers, gs.Params.MinHeadcountAgreement = 3, 2
	f := newClaimIntegrityGenesisFixture(t, "seat-parity", gs)
	h := sha256.New()
	write := func(parts ...string) {
		var n [binary.MaxVarintLen64]byte
		for _, part := range parts {
			h.Write(n[:binary.PutUvarint(n[:], uint64(len(part)))])
			h.Write([]byte(part))
		}
	}
	step := func(name string, run func(ctx sdk.Context) error) {
		meter := storetypes.NewGasMeter(1 << 40)
		ctx := f.ctx.WithGasMeter(meter).WithEventManager(sdk.NewEventManager())
		err := run(ctx)
		errText := ""
		if err != nil {
			errText = err.Error()
		}
		write("step", name, fmt.Sprint(meter.GasConsumed()), errText)
		for _, event := range ctx.EventManager().Events() {
			write("event", event.Type)
			for _, attr := range event.Attributes {
				write(attr.Key, attr.Value)
			}
		}
	}
	at := func(height int64) { f.ctx = f.ctx.WithBlockHeight(height) }

	author := makeValidBech32Addr("parity-author")
	var claimID string
	step("submit-claim", func(ctx sdk.Context) error {
		response, err := f.server.SubmitClaim(ctx, &types.MsgSubmitClaim{Submitter: author, FactContent: "A reproducible synthetic parity observation with a retained test procedure.", Domain: "physics", Category: "empirical", Stake: "11000000"})
		if err == nil {
			claimID = response.ClaimId
		}
		return err
	})
	claim, found := f.k.GetClaim(f.ctx, claimID)
	require.True(t, found)
	round, found := f.k.GetVerificationRound(f.ctx, claim.VerificationRoundId)
	require.True(t, found)
	require.Equal(t, types.ReviewPolicyNeutral, round.ReviewPolicyVersion)

	attestation := &types.ReviewAttestation{Reason: "Synthetic parity review of the stated record only."}
	salt := func(i int) []byte { return []byte(fmt.Sprintf("parity-review-salt-%02d", i)) }
	reviewers := []string{makeValidBech32Addr("parity-short"), makeValidBech32Addr("parity-rv-1"), makeValidBech32Addr("parity-rv-2"), makeValidBech32Addr("parity-rv-3")}
	votes := []string{"accept", "accept", "reject", "accept"}
	for i, reviewer := range reviewers {
		balance := int64(100_000_000)
		if i == 0 {
			balance = 99_999_999
		}
		f.bank.balances[reviewer] = sdk.NewCoins(sdk.NewInt64Coin("uzrn", balance))
		hash, err := types.ComputeReviewCommitmentV2(round.CommitmentChainId, round.Id, reviewer, votes[i], 800_000, salt(i), attestation)
		require.NoError(t, err)
		step(fmt.Sprintf("commit-%d", i), func(ctx sdk.Context) error {
			_, err := f.server.SubmitCommitment(ctx, &types.MsgSubmitCommitment{Verifier: reviewer, RoundId: round.Id, CommitHash: hash})
			return err
		})
	}
	at(int64(round.CommitDeadline))
	step("begin-block-reveal", func(ctx sdk.Context) error { return f.k.BeginBlocker(ctx) })
	for i := 1; i < len(reviewers); i++ {
		step(fmt.Sprintf("reveal-%d", i), func(ctx sdk.Context) error {
			_, err := f.server.SubmitReveal(ctx, &types.MsgSubmitReveal{Verifier: reviewers[i], RoundId: round.Id, Vote: votes[i], Confidence: 800_000, Salt: salt(i), Attestation: attestation})
			return err
		})
	}
	at(int64(round.CommitDeadline) + 1)
	step("begin-block-complete", func(ctx sdk.Context) error { return f.k.BeginBlocker(ctx) })
	done, found := f.k.GetVerificationRound(f.ctx, round.Id)
	require.True(t, found)
	require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_COMPLETE, done.Phase, "the scenario completes its round")
	require.Len(t, done.Commits, 3, "the short reviewer was refused")
	at(int64(round.CommitDeadline) + 2)
	step("begin-block-queue", func(ctx sdk.Context) error { return f.k.BeginBlocker(ctx) })

	for _, figure := range []string{"150000000", "99999999", "1", "0", ""} {
		step("update-params-"+figure, func(ctx sdk.Context) error {
			params, err := f.k.GetParams(ctx)
			require.NoError(t, err)
			next := proto.Clone(params).(*types.Params)
			next.MinReviewerBalance = figure
			_, err = f.server.UpdateParams(ctx, &types.MsgUpdateParams{Authority: "authority", Params: next})
			return err
		})
	}

	var entries []string
	it := f.ctx.KVStore(f.key).Iterator(nil, nil)
	for ; it.Valid(); it.Next() {
		entries = append(entries, string(it.Key())+"\x00"+string(it.Value()))
	}
	require.NoError(t, it.Close())
	sort.Strings(entries)
	write("store", fmt.Sprint(len(entries)))
	for _, entry := range entries {
		write(entry)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func TestSeatDepositMarkerOffMatchesBase(t *testing.T) {
	for _, param := range []bool{false, true} {
		t.Run(fmt.Sprintf("reviewer-balance-param=%t", param), func(t *testing.T) {
			digest := seatParityDigest(t, param)
			t.Logf("seat parity digest (reviewer-balance-param=%t): %s", param, digest)
			require.Equal(t, digest, seatParityDigest(t, param), "the scenario is deterministic")
			require.Equal(t, seatParityDigests[param], digest)
		})
	}
}
