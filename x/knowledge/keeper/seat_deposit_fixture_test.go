package keeper_test

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

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// seatBank is the tracking bank with real balances for the seat-deposit
// account only: a deposit needs the depositor's balance, a return needs the
// account's custody, and a return to an address in failReturnTo fails.
// Every other transfer keeps the tracking bank's record-only behaviour.
type seatBank struct {
	*trackingBankKeeper
	failReturnTo map[string]bool
	failAll      bool
	// failFrom fails every module-to-account transfer from these modules,
	// such as reviewer pay from the knowledge module.
	failFrom map[string]bool
}

func newSeatBank() *seatBank {
	return &seatBank{trackingBankKeeper: newTrackingBankKeeper(), failReturnTo: map[string]bool{}, failFrom: map[string]bool{}}
}

func (b *seatBank) SendCoinsFromAccountToModule(ctx context.Context, from sdk.AccAddress, module string, amt sdk.Coins) error {
	if module != types.SeatDepositModuleName {
		return b.trackingBankKeeper.SendCoinsFromAccountToModule(ctx, from, module, amt)
	}
	have := b.balances[from.String()]
	if !have.IsAllGTE(amt) {
		return fmt.Errorf("spendable balance %s is smaller than %s", have, amt)
	}
	b.balances[from.String()] = have.Sub(amt...)
	b.moduleBalances[module] = b.moduleBalances[module].Add(amt...)
	b.sendCalls = append(b.sendCalls, sendRecord{from.String(), module, amt})
	return nil
}

func (b *seatBank) SendCoinsFromModuleToAccount(ctx context.Context, module string, to sdk.AccAddress, amt sdk.Coins) error {
	if b.failFrom[module] {
		return fmt.Errorf("injected %s transfer failure", module)
	}
	if module != types.SeatDepositModuleName {
		return b.trackingBankKeeper.SendCoinsFromModuleToAccount(ctx, module, to, amt)
	}
	if b.failAll || b.failReturnTo[to.String()] {
		return fmt.Errorf("injected seat-deposit return failure")
	}
	have := b.moduleBalances[module]
	if !have.IsAllGTE(amt) {
		return fmt.Errorf("seat-deposit custody %s is smaller than %s", have, amt)
	}
	b.moduleBalances[module] = have.Sub(amt...)
	b.balances[to.String()] = b.balances[to.String()].Add(amt...)
	b.sendCalls = append(b.sendCalls, sendRecord{module, to.String(), amt})
	return nil
}

func (b *seatBank) custody() int64 {
	return b.moduleBalances[types.SeatDepositModuleName].AmountOf("uzrn").Int64()
}

func (b *seatBank) balance(address string) int64 {
	return b.balances[address].AmountOf("uzrn").Int64()
}

// returnsTo counts transfers out of the seat-deposit account to address.
func (b *seatBank) returnsTo(address string) int {
	n := 0
	for _, call := range b.sendCalls {
		if call.from == types.SeatDepositModuleName && call.to == address {
			n++
		}
	}
	return n
}

// seatAuth admits the accounts it lists.
type seatAuth struct{ admitted map[string]bool }

func (a *seatAuth) GetAccountType(_ context.Context, address string) (string, bool) {
	return "agent", a.admitted[address]
}

func (a *seatAuth) CheckReviewer(_ context.Context, address string) error {
	if !a.admitted[address] {
		return types.ErrReviewerNotAdmitted.Wrapf("%s is not registered with zerone_auth", address)
	}
	return nil
}

// seatAccounts serves the accounts it holds; the seat-deposit address is
// empty unless a test puts something there.
type seatAccounts struct{ accounts map[string]sdk.AccountI }

func (a *seatAccounts) GetAccount(_ context.Context, address sdk.AccAddress) sdk.AccountI {
	return a.accounts[address.String()]
}

type seatFixture struct {
	k        keeper.Keeper
	ctx      sdk.Context
	key      *storetypes.KVStoreKey
	bank     *seatBank
	auth     *seatAuth
	accounts *seatAccounts
	server   types.MsgServer
}

// newSeatFixture is native genesis with the seat-deposit marker on, unless
// mutate says otherwise, and short review windows (commit 2, reveal 2,
// aggregation 2 blocks) at height 100.
func newSeatFixture(t *testing.T, mutate func(*types.GenesisState)) *seatFixture {
	t.Helper()
	db := dbm.NewMemDB()
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	ms.SetIAVLSyncPruning(true)
	key := storetypes.NewKVStoreKey(types.StoreKey)
	ms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())
	ctx := sdk.NewContext(ms, cmtproto.Header{ChainID: "seat-deposit-test", Height: 100}, false, log.NewNopLogger())
	bank := newSeatBank()
	k := keeper.NewKeeper(runtime.NewKVStoreService(key), codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), "authority", bank, newTrackingStakingKeeper())
	auth := &seatAuth{admitted: map[string]bool{}}
	accounts := &seatAccounts{accounts: map[string]sdk.AccountI{}}
	k.SetZeroneAuthKeeper(auth)
	k.SetAccountKeeper(accounts)
	gs := types.DefaultGenesis()
	gs.SeatDepositEnabled = true
	gs.Params.CommitPhaseBlocks, gs.Params.RevealPhaseBlocks, gs.Params.AggregationPhaseBlocks = 2, 2, 2
	gs.Params.MinVerifiers, gs.Params.MinHeadcountAgreement = 3, 2
	if mutate != nil {
		mutate(gs)
	}
	require.NoError(t, k.InitGenesis(ctx, gs))
	return &seatFixture{k: k, ctx: ctx.WithEventManager(sdk.NewEventManager()), key: key, bank: bank, auth: auth, accounts: accounts, server: keeper.NewMsgServerImpl(k)}
}

// rewire rebuilds the message server after a keeper dependency changes.
func (f *seatFixture) rewire() { f.server = keeper.NewMsgServerImpl(f.k) }

func (f *seatFixture) height(h int64) { f.ctx = f.ctx.WithBlockHeight(h) }

// claim admits an ordinary claim and returns its round.
func (f *seatFixture) claim(t *testing.T, tag string) *types.VerificationRound {
	t.Helper()
	response, err := f.server.SubmitClaim(f.ctx, &types.MsgSubmitClaim{Submitter: makeValidBech32Addr("seat-author-" + tag), FactContent: "A reproducible synthetic observation " + tag + " with a retained test procedure.", Domain: "physics", Category: "empirical", Stake: "11000000"})
	require.NoError(t, err)
	claim, found := f.k.GetClaim(f.ctx, response.ClaimId)
	require.True(t, found)
	round, found := f.k.GetVerificationRound(f.ctx, claim.VerificationRoundId)
	require.True(t, found)
	return round
}

// reviewer is an admitted address holding balance uzrn.
func (f *seatFixture) reviewer(seed string, balance int64) string {
	address := makeValidBech32Addr(seed)
	f.auth.admitted[address] = true
	f.bank.balances[address] = sdk.NewCoins(sdk.NewInt64Coin("uzrn", balance))
	return address
}

var seatAttestation = &types.ReviewAttestation{Reason: "Synthetic seat-deposit review of the stated record only."}

func seatSalt(reviewer string) []byte { return []byte("seat-salt-" + reviewer[len(reviewer)-8:]) }

func (f *seatFixture) commit(t *testing.T, round *types.VerificationRound, reviewer, vote string) error {
	t.Helper()
	hash, err := types.ComputeReviewCommitmentV2(round.CommitmentChainId, round.Id, reviewer, vote, 800_000, seatSalt(reviewer), seatAttestation)
	require.NoError(t, err)
	_, err = f.server.SubmitCommitment(f.ctx, &types.MsgSubmitCommitment{Verifier: reviewer, RoundId: round.Id, CommitHash: hash})
	return err
}

func (f *seatFixture) reveal(t *testing.T, round *types.VerificationRound, reviewer, vote string) {
	t.Helper()
	_, err := f.server.SubmitReveal(f.ctx, &types.MsgSubmitReveal{Verifier: reviewer, RoundId: round.Id, Vote: vote, Confidence: 800_000, Salt: seatSalt(reviewer), Attestation: seatAttestation})
	require.NoError(t, err)
}

func (f *seatFixture) beginBlock(t *testing.T) {
	t.Helper()
	require.NoError(t, f.k.BeginBlocker(f.ctx))
}

func (f *seatFixture) round(t *testing.T, id string) *types.VerificationRound {
	t.Helper()
	round, found := f.k.GetVerificationRound(f.ctx, id)
	require.True(t, found)
	return round
}

func seatModuleAddress() sdk.AccAddress {
	return authtypes.NewModuleAddress(types.SeatDepositModuleName)
}
