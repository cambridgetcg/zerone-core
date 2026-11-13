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
}

func newSeatInternalBank() *seatInternalBank {
	return &seatInternalBank{balances: map[string]int64{}, returns: map[string]int{}}
}

func (b *seatInternalBank) SendCoins(context.Context, sdk.AccAddress, sdk.AccAddress, sdk.Coins) error {
	return nil
}

func (b *seatInternalBank) SendCoinsFromModuleToAccount(_ context.Context, module string, to sdk.AccAddress, amt sdk.Coins) error {
	if module != types.SeatDepositModuleName {
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

	*f.fault = handoffFault{}
	f.ctx = f.ctx.WithBlockHeight(int64(round.AggregationDeadline) + 2).WithEventManager(sdk.NewEventManager())
	require.NoError(t, f.k.BeginBlocker(f.ctx))
	done := f.stored(t, round.Id)
	require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_COMPLETE, done.Phase)
	require.Equal(t, types.Verdict_VERDICT_ACCEPT, done.Verdict)
	require.Greater(t, done.VerdictBlock, done.SeatDepositReturns.CreatedAtBlock)
	require.Equal(t, round.AggregationDeadline, done.SeatDepositReturns.CreatedAtBlock, "late completion keeps the deadline plan")
	require.Zero(t, countSeatEvents(f.ctx, "zerone.knowledge.seat_deposit_returns_accrued", ""))
}
