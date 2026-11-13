package keeper

import (
	"context"
	"fmt"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// seatInternalBank holds real balances for reviewers and the seat-deposit
// account; every other transfer succeeds without moving anything, as the
// claim fee legs need no checking here.
type seatInternalBank struct {
	balances    map[string]int64
	custody     int64
	failReturns bool
	returns     map[string]int
	// transfers counts every module-to-account transfer attempted that
	// succeeded, of any module: reviewer pay and seat-deposit returns.
	transfers int
}

func newSeatInternalBank() *seatInternalBank {
	return &seatInternalBank{balances: map[string]int64{}, returns: map[string]int{}}
}

func (b *seatInternalBank) SendCoins(context.Context, sdk.AccAddress, sdk.AccAddress, sdk.Coins) error {
	return nil
}

func (b *seatInternalBank) SendCoinsFromModuleToAccount(_ context.Context, module string, to sdk.AccAddress, amt sdk.Coins) error {
	if module != types.SeatDepositModuleName {
		b.transfers++
		return nil
	}
	amount := amt.AmountOf("uzrn").Int64()
	if b.failReturns {
		return fmt.Errorf("injected return failure")
	}
	if b.custody < amount {
		return fmt.Errorf("seat-deposit custody %d is below %d", b.custody, amount)
	}
	b.custody -= amount
	b.balances[to.String()] += amount
	b.returns[to.String()]++
	b.transfers++
	return nil
}

func (b *seatInternalBank) SendCoinsFromAccountToModule(_ context.Context, from sdk.AccAddress, module string, amt sdk.Coins) error {
	if module != types.SeatDepositModuleName {
		return nil
	}
	amount := amt.AmountOf("uzrn").Int64()
	if b.balances[from.String()] < amount {
		return fmt.Errorf("spendable balance is smaller than %d", amount)
	}
	b.balances[from.String()] -= amount
	b.custody += amount
	return nil
}

func (b *seatInternalBank) SendCoinsFromModuleToModule(context.Context, string, string, sdk.Coins) error {
	return nil
}

func (b *seatInternalBank) MintCoins(context.Context, string, sdk.Coins) error { return nil }

func (b *seatInternalBank) GetBalance(_ context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	if addr.Equals(authtypes.NewModuleAddress(types.SeatDepositModuleName)) {
		return sdk.NewInt64Coin(denom, b.custody)
	}
	return sdk.NewInt64Coin(denom, b.balances[addr.String()])
}

type seatInternalAuth struct{}

func (seatInternalAuth) GetAccountType(context.Context, string) (string, bool) { return "agent", true }
func (seatInternalAuth) CheckReviewer(context.Context, string) error           { return nil }

type seatInternalAccounts struct{}

func (seatInternalAccounts) GetAccount(context.Context, sdk.AccAddress) sdk.AccountI { return nil }

type seatInternalFixture struct {
	k      Keeper
	ctx    sdk.Context
	key    *storetypes.KVStoreKey
	bank   *seatInternalBank
	fault  *handoffFault
	server types.MsgServer
}

// newSeatInternalFixture is native genesis with the seat-deposit marker on
// and two-block windows, on a store whose writes can be made to fail.
func newSeatInternalFixture(t *testing.T) *seatInternalFixture {
	t.Helper()
	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	key := storetypes.NewKVStoreKey(types.StoreKey)
	ms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())
	ctx := sdk.NewContext(ms, cmtproto.Header{ChainID: "seat-internal", Height: 100}, false, log.NewNopLogger())
	fault := new(handoffFault)
	bank := newSeatInternalBank()
	k := NewKeeper(handoffFaultService{runtime.NewKVStoreService(key), fault}, codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), "authority", bank, nil)
	k.SetZeroneAuthKeeper(seatInternalAuth{})
	k.SetAccountKeeper(seatInternalAccounts{})
	gs := types.DefaultGenesis()
	gs.SeatDepositEnabled = true
	gs.Params.CommitPhaseBlocks, gs.Params.RevealPhaseBlocks, gs.Params.AggregationPhaseBlocks = 2, 2, 2
	gs.Params.MinVerifiers, gs.Params.MinHeadcountAgreement = 3, 2
	require.NoError(t, k.InitGenesis(ctx, gs))
	return &seatInternalFixture{k: k, ctx: ctx.WithEventManager(sdk.NewEventManager()), key: key, bank: bank, fault: fault, server: NewMsgServerImpl(k)}
}

func seatInternalAddr(seed string) string {
	b := make([]byte, 20)
	copy(b, seed)
	return sdk.AccAddress(b).String()
}

// acceptPanel admits a claim and seats three reviewers who commit and reveal
// accept, leaving the fixture at the reveal deadline before BeginBlock.
func (f *seatInternalFixture) acceptPanel(t *testing.T, tag string) (*types.VerificationRound, []string) {
	t.Helper()
	response, err := f.server.SubmitClaim(f.ctx, &types.MsgSubmitClaim{Submitter: seatInternalAddr("author-" + tag), FactContent: "A reproducible synthetic observation " + tag + " with a retained test procedure.", Domain: "physics", Category: "empirical", Stake: "11000000"})
	require.NoError(t, err)
	claim, found := f.k.GetClaim(f.ctx, response.ClaimId)
	require.True(t, found)
	round, found := f.k.GetVerificationRound(f.ctx, claim.VerificationRoundId)
	require.True(t, found)
	require.NotNil(t, round.SeatTerms)
	attestation := &types.ReviewAttestation{Reason: "Synthetic seat-deposit review of the stated record only."}
	var reviewers []string
	for i := 0; i < 3; i++ {
		reviewer := seatInternalAddr(fmt.Sprintf("rv-%s-%d", tag, i))
		f.bank.balances[reviewer] = 100_000_000
		salt := []byte(fmt.Sprintf("seat-internal-salt-%d", i))
		hash, err := types.ComputeReviewCommitmentV2(round.CommitmentChainId, round.Id, reviewer, "accept", 800_000, salt, attestation)
		require.NoError(t, err)
		_, err = f.server.SubmitCommitment(f.ctx, &types.MsgSubmitCommitment{Verifier: reviewer, RoundId: round.Id, CommitHash: hash})
		require.NoError(t, err)
		reviewers = append(reviewers, reviewer)
	}
	require.Equal(t, int64(300_000_000), f.bank.custody)
	f.ctx = f.ctx.WithBlockHeight(int64(round.CommitDeadline))
	require.NoError(t, f.k.BeginBlocker(f.ctx))
	for i, reviewer := range reviewers {
		_, err := f.server.SubmitReveal(f.ctx, &types.MsgSubmitReveal{Verifier: reviewer, RoundId: round.Id, Vote: "accept", Confidence: 800_000, Salt: []byte(fmt.Sprintf("seat-internal-salt-%d", i)), Attestation: attestation})
		require.NoError(t, err)
	}
	f.ctx = f.ctx.WithBlockHeight(int64(round.RevealDeadline))
	return round, reviewers
}

func (f *seatInternalFixture) stored(t *testing.T, id string) *types.VerificationRound {
	t.Helper()
	round, err := f.k.getVerificationRoundChecked(f.ctx, id)
	require.NoError(t, err)
	return round
}

func countSeatEvents(ctx sdk.Context, kind, trigger string) int {
	n := 0
	for _, event := range ctx.EventManager().Events() {
		if event.Type != kind {
			continue
		}
		if trigger == "" {
			n++
			continue
		}
		for _, attr := range event.Attributes {
			if attr.Key == "trigger" && attr.Value == trigger {
				n++
			}
		}
	}
	return n
}

// D4: a round whose completion keeps failing does not hold deposits. Here an
// accepted claim cannot complete while fact writes fail (a write only
// completion performs): the round stays in aggregation, and from its
// aggregation deadline its returns are frozen anyway. When completion later
// succeeds it keeps that plan and accrues nothing again.
func TestSeatDepositReturnedFromAggregationDeadline(t *testing.T) {
	f := newSeatInternalFixture(t)
	round, reviewers := f.acceptPanel(t, "deadline")
	*f.fault = handoffFault{op: "set", prefix: types.FactKeyPrefix}

	require.NoError(t, f.k.BeginBlocker(f.ctx))
	stuck := f.stored(t, round.Id)
	require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION, stuck.Phase, "completion failed")
	require.Nil(t, stuck.SeatDepositReturns, "nothing is frozen before the deadline")

	f.ctx = f.ctx.WithBlockHeight(int64(round.AggregationDeadline)).WithEventManager(sdk.NewEventManager())
	require.NoError(t, f.k.BeginBlocker(f.ctx))
	frozen := f.stored(t, round.Id)
	require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION, frozen.Phase)
	require.NotNil(t, frozen.SeatDepositReturns)
	require.Equal(t, round.AggregationDeadline, frozen.SeatDepositReturns.CreatedAtBlock)
	require.Len(t, frozen.SeatDepositReturns.Returns, len(reviewers))
	require.Equal(t, 1, countSeatEvents(f.ctx, "zerone.knowledge.seat_deposit_returns_accrued", "aggregation_deadline"))

	f.ctx = f.ctx.WithBlockHeight(int64(round.AggregationDeadline) + 1).WithEventManager(sdk.NewEventManager())
	require.NoError(t, f.k.BeginBlocker(f.ctx))
	require.Zero(t, countSeatEvents(f.ctx, "zerone.knowledge.seat_deposit_returns_accrued", ""), "a frozen plan is never frozen again")
	// The pending queue paid every return on the block after the deadline,
	// while the round itself still cannot complete.
	paid := f.stored(t, round.Id)
	require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION, paid.Phase)
	for _, ret := range paid.SeatDepositReturns.Returns {
		require.Equal(t, round.AggregationDeadline+1, ret.PaidAtBlock)
	}
	for _, reviewer := range reviewers {
		require.Equal(t, int64(100_000_000), f.bank.balances[reviewer])
		require.Equal(t, 1, f.bank.returns[reviewer])
	}
	require.Zero(t, f.bank.custody)
	require.Equal(t, len(reviewers), countSeatEvents(f.ctx, "zerone.knowledge.seat_deposit_returned", ""))

	*f.fault = handoffFault{}
	f.ctx = f.ctx.WithBlockHeight(int64(round.AggregationDeadline) + 2).WithEventManager(sdk.NewEventManager())
	require.NoError(t, f.k.BeginBlocker(f.ctx))
	done := f.stored(t, round.Id)
	require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_COMPLETE, done.Phase)
	require.Equal(t, types.Verdict_VERDICT_ACCEPT, done.Verdict)
	require.Greater(t, done.VerdictBlock, done.SeatDepositReturns.CreatedAtBlock)
	require.Equal(t, round.AggregationDeadline, done.SeatDepositReturns.CreatedAtBlock, "late completion keeps the deadline plan")
	require.Zero(t, countSeatEvents(f.ctx, "zerone.knowledge.seat_deposit_returns_accrued", ""))
	require.True(t, proto.Equal(paid.SeatDepositReturns, done.SeatDepositReturns), "late completion repays nothing")
	for _, reviewer := range reviewers {
		require.Equal(t, 1, f.bank.returns[reviewer])
	}
}

// seedSettledSeatRound stores a completed policy-1 round with seats seats,
// each revealed and paid 1 uzrn of reviewer pay that is still unpaid, and
// each seat's 1 uzrn deposit return still unpaid; with seats 0 it stores a
// round with one unpaid reviewer payment and no seat terms. The claim is
// unfunded, so the round owes no refund.
func (f *seatInternalFixture) seedSettledSeatRound(t *testing.T, id string, seats int) {
	t.Helper()
	claim := &types.Claim{Id: "claim-" + id, Submitter: seatInternalAddr("seed-submitter"), Stake: "11000000", FactContent: "A seeded synthetic claim " + id + ".", ReviewPolicyVersion: types.ReviewPolicyNeutral, VerificationRoundId: id, Status: types.ClaimStatus_CLAIM_STATUS_INSUFFICIENT}
	require.NoError(t, f.k.SetClaim(f.ctx, claim))
	round := &types.VerificationRound{Id: id, ClaimId: claim.Id, Phase: types.VerificationPhase_VERIFICATION_PHASE_COMPLETE,
		StartedAtBlock: 10, CommitDeadline: 12, RevealDeadline: 14, AggregationDeadline: 16, Verdict: types.Verdict_VERDICT_INCONCLUSIVE, VerdictBlock: 14,
		CommitmentScheme: types.CommitmentSchemeReviewV2, CommitmentChainId: "seat-internal", ReviewPolicyVersion: types.ReviewPolicyNeutral,
		VerifierRewardSettlement: &types.VerifierRewardSettlement{CreatedAtBlock: 14, WithheldTotal: "0"}}
	n := seats
	if seats > 0 {
		round.SeatTerms = &types.ReviewSeatTerms{CountedSeatDeposit: "1"}
	} else {
		n = 1
	}
	attestation := &types.ReviewAttestation{Reason: "Seeded synthetic review."}
	for i := 0; i < n; i++ {
		reviewer := sdk.AccAddress([]byte(fmt.Sprintf("%-14s%06d", id[:min(len(id), 14)], i))).String()
		salt := []byte(fmt.Sprintf("seeded-salt-%06d", i))
		hash, err := types.ComputeReviewCommitmentV2(round.CommitmentChainId, round.Id, reviewer, "accept", 800_000, salt, attestation)
		require.NoError(t, err)
		commit := &types.CommitEntry{Verifier: reviewer, CommitHash: hash, CommittedAtBlock: 11}
		if seats > 0 {
			commit.SeatDeposit = "1"
		}
		round.Commits = append(round.Commits, commit)
		round.SelectedVerifiers = append(round.SelectedVerifiers, reviewer)
		round.Reveals = append(round.Reveals, &types.RevealEntry{Verifier: reviewer, Vote: "accept", Salt: salt, Confidence: 800_000, Attestation: attestation, RevealedAtBlock: 13})
		round.VerifierRewardSettlement.Payments = append(round.VerifierRewardSettlement.Payments, &types.VerifierRewardPayment{Verifier: reviewer, Amount: "1", Withheld: "0"})
	}
	if seats > 0 {
		round.SeatDepositReturns = types.BuildSeatDepositReturns(round, 14)
		f.bank.custody += int64(seats)
	}
	require.NoError(t, f.k.SetVerificationRound(f.ctx, round))
}

func (f *seatInternalFixture) unpaidLegs(t *testing.T, id string) int {
	t.Helper()
	return pendingObligationTransferCount(f.stored(t, id))
}

// D5: a full round's plan (512 reviewer payments and 512 returns, 1,024
// transfers, above the 513-transfer budget) does not stall the queue. It is
// paid alone in a block of its own when it comes first, and the rounds
// queued before and after it are paid too.
func TestSeatDepositFullRoundPlanDoesNotStallQueue(t *testing.T) {
	f := newSeatInternalFixture(t)
	f.seedSettledSeatRound(t, "queue-a-before", 0)
	f.seedSettledSeatRound(t, "queue-m-full", types.CommitSeatHardCap)
	f.seedSettledSeatRound(t, "queue-z-after", 0)
	require.Equal(t, 2*types.CommitSeatHardCap, f.unpaidLegs(t, "queue-m-full"))
	execution, err := f.k.selectRoundsExecution(f.ctx)
	require.NoError(t, err)
	require.True(t, execution.seatDeposit)

	var perBlock []int
	for block := 0; block < 4; block++ {
		f.ctx = f.ctx.WithBlockHeight(f.ctx.BlockHeight() + 1)
		before := f.bank.transfers
		require.NoError(t, execution.ProcessPendingVerifierRewards(f.ctx))
		perBlock = append(perBlock, f.bank.transfers-before)
	}
	// Block 1 pays the round before and stops at the full round; block 2
	// pays the full round alone; block 3 pays the round after; block 4 has
	// nothing left.
	require.Equal(t, []int{1, 2 * types.CommitSeatHardCap, 1, 0}, perBlock)
	for _, id := range []string{"queue-a-before", "queue-m-full", "queue-z-after"} {
		require.Zero(t, f.unpaidLegs(t, id), id)
	}
	require.Zero(t, f.bank.custody)
}

// The companion: the 3a844c48 queue body cannot carry returns. It counts and
// pays only reviewer pay and refunds, so it pays the full round's 512
// reviewer payments and none of its returns; the round stays indexed for
// them, and on the next block the legacy consistency check refuses a queued
// round with no reviewer pay or refund pending. Both begin-block bodies
// return that error, so a chain running the legacy body with returns queued
// would fail its block. This is why the queue dispatches to a new body while
// seat deposits are on.
func TestSeatDepositLegacyQueueCannotCarryReturns(t *testing.T) {
	f := newSeatInternalFixture(t)
	f.seedSettledSeatRound(t, "queue-m-full", types.CommitSeatHardCap)
	f.ctx = f.ctx.WithBlockHeight(f.ctx.BlockHeight() + 1)
	require.NoError(t, f.k.processPendingVerifierRewardsLegacy(f.ctx))
	require.Equal(t, types.CommitSeatHardCap, f.bank.transfers, "reviewer pay only")
	require.Equal(t, types.CommitSeatHardCap, f.unpaidLegs(t, "queue-m-full"), "every return still pending")
	f.ctx = f.ctx.WithBlockHeight(f.ctx.BlockHeight() + 1)
	require.ErrorContains(t, f.k.processPendingVerifierRewardsLegacy(f.ctx), "pending verifier reward index does not match primary round")
}
