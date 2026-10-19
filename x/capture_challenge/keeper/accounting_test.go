package keeper_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	corestore "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

	capturechallenge "github.com/zerone-chain/zerone/x/capture_challenge"
	"github.com/zerone-chain/zerone/x/capture_challenge/keeper"
	"github.com/zerone-chain/zerone/x/capture_challenge/types"
)

// Transfers and balances are real SDK bank/auth state. Only the returned-error
// seam is synthetic; a post-transfer failure proves the caller discards writes
// and events, not merely that a mock declined a send.
type challengeAccountingBank struct {
	bankkeeper.BaseKeeper
	failRefund    string
	failRecipient string
	refundCalls   int
}

func (b *challengeAccountingBank) SendCoinsFromModuleToAccount(ctx context.Context, module string, addr sdk.AccAddress, coins sdk.Coins) error {
	fail := b.failRecipient == "" || b.failRecipient == addr.String()
	if module == types.ModuleName {
		b.refundCalls++
		if fail && b.failRefund == "before" {
			return errors.New("injected before refund")
		}
	}
	if err := b.BaseKeeper.SendCoinsFromModuleToAccount(ctx, module, addr, coins); err != nil {
		return err
	}
	if module == types.ModuleName && fail && b.failRefund == "after" {
		return errors.New("injected after refund")
	}
	return nil
}

// The pinned SDK adapter normally returns nil after its underlying void Set.
// These faults test our dependency contract, not a production disk-error claim.
type challengeWriteFault struct {
	corestore.KVStoreService
	key  []byte
	mode string
}

func (f *challengeWriteFault) OpenKVStore(ctx context.Context) corestore.KVStore {
	return challengeFaultStore{KVStore: f.KVStoreService.OpenKVStore(ctx), fault: f}
}

type challengeFaultStore struct {
	corestore.KVStore
	fault *challengeWriteFault
}

func (s challengeFaultStore) Set(key, value []byte) error {
	match := bytes.Equal(key, s.fault.key)
	if match && s.fault.mode == "before" {
		return errors.New("injected before expiry record")
	}
	if err := s.KVStore.Set(key, value); err != nil {
		return err
	}
	if match && s.fault.mode == "after" {
		return errors.New("injected after expiry record")
	}
	return nil
}

type challengeAccountingHarness struct {
	k          keeper.Keeper
	ctx        sdk.Context
	bank       *challengeAccountingBank
	auth       authkeeper.AccountKeeper
	module     capturechallenge.AppModule
	keys       map[string]*storetypes.KVStoreKey
	writeFault *challengeWriteFault
}

func newChallengeAccountingHarness(t *testing.T) *challengeAccountingHarness {
	t.Helper()
	keys := storetypes.NewKVStoreKeys(types.StoreKey, authtypes.StoreKey, banktypes.StoreKey)
	db := dbm.NewMemDB()
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	state := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	state.SetIAVLSyncPruning(true)
	for _, name := range []string{types.StoreKey, authtypes.StoreKey, banktypes.StoreKey} {
		state.MountStoreWithDB(keys[name], storetypes.StoreTypeIAVL, db)
	}
	require.NoError(t, state.LoadLatestVersion())
	ctx := sdk.NewContext(state.CacheMultiStore(), cmtproto.Header{Height: 100}, false, log.NewNopLogger())
	registry := codectypes.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(registry)
	banktypes.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress("gov").String()
	perms := map[string][]string{
		types.ModuleName:        nil,
		"development_fund":      nil,
		"challenge_test_issuer": {authtypes.Minter},
		"challenge_test_sink":   nil,
	}
	ak := authkeeper.NewAccountKeeper(cdc, runtime.NewKVStoreService(keys[authtypes.StoreKey]), authtypes.ProtoBaseAccount, perms, addresscodec.NewBech32Codec("zrn"), "zrn", authority)
	ak.InitGenesis(ctx, *authtypes.DefaultGenesisState())
	bk := bankkeeper.NewBaseKeeper(cdc, runtime.NewKVStoreService(keys[banktypes.StoreKey]), ak, map[string]bool{}, authority, log.NewNopLogger())
	bk.InitGenesis(ctx, banktypes.DefaultGenesisState())
	for _, name := range []string{types.ModuleName, "development_fund", "challenge_test_issuer", "challenge_test_sink"} {
		ak.GetModuleAccount(ctx, name)
	}
	bank := &challengeAccountingBank{BaseKeeper: bk}
	fault := &challengeWriteFault{KVStoreService: runtime.NewKVStoreService(keys[types.StoreKey])}
	k := keeper.NewKeeper(fault, cdc, authority, bank)
	k.InitGenesis(ctx, types.DefaultGenesis())
	return &challengeAccountingHarness{k: k, ctx: ctx, bank: bank, auth: ak, module: capturechallenge.NewAppModule(cdc, k), keys: keys, writeFault: fault}
}

func (h *challengeAccountingHarness) fund(t *testing.T, addr sdk.AccAddress, amount int64) {
	t.Helper()
	coins := sdk.NewCoins(sdk.NewInt64Coin("uzrn", amount))
	require.NoError(t, h.bank.MintCoins(h.ctx, "challenge_test_issuer", coins))
	require.NoError(t, h.bank.BaseKeeper.SendCoinsFromModuleToAccount(h.ctx, "challenge_test_issuer", addr, coins))
}

func (h *challengeAccountingHarness) submit(t *testing.T, who int, domain string) string {
	t.Helper()
	addr := sdk.MustAccAddressFromBech32(testAddr(who))
	h.fund(t, addr, 10_000_000)
	resp, err := keeper.NewMsgServerImpl(h.k).SubmitChallenge(h.ctx, &types.MsgSubmitChallenge{
		Challenger: addr.String(), Domain: domain, Stake: "10000000",
		AccusedValidators: []string{testAddr(99)}, Reason: "synthetic accounting fixture",
	})
	require.NoError(t, err)
	return resp.ChallengeId
}

func (h *challengeAccountingHarness) advance(t *testing.T, height uint64) {
	t.Helper()
	h.ctx = h.ctx.WithBlockHeight(int64(height))
	require.NoError(t, h.module.BeginBlock(h.ctx))
}

func (h *challengeAccountingHarness) snapshot(t *testing.T) map[string]map[string]string {
	t.Helper()
	result := make(map[string]map[string]string)
	for name, key := range h.keys {
		result[name] = make(map[string]string)
		it := h.ctx.KVStore(key).Iterator(nil, nil)
		for ; it.Valid(); it.Next() {
			result[name][string(it.Key())] = string(it.Value())
		}
		require.NoError(t, it.Close())
	}
	return result
}

func (h *challengeAccountingHarness) challenge(t *testing.T, id string) *types.CaptureChallenge {
	t.Helper()
	ch, found := h.k.GetChallenge(h.ctx, id)
	require.True(t, found)
	return ch
}

func (h *challengeAccountingHarness) resolve(t *testing.T, id string, outcome types.ChallengeOutcome) {
	t.Helper()
	// SDK governance owns the outer transaction cache in production.
	ctx, write := h.ctx.CacheContext()
	_, err := keeper.NewMsgServerImpl(h.k).ResolveChallenge(ctx, &types.MsgResolveChallenge{
		Authority: h.k.GetAuthority(), ChallengeId: id, Outcome: outcome, Reason: "synthetic resolution",
	})
	require.NoError(t, err)
	write()
}

func (h *challengeAccountingHarness) moduleBalance(name string) string {
	return h.bank.GetBalance(h.ctx, authtypes.NewModuleAddress(name), "uzrn").Amount.String()
}

func (h *challengeAccountingHarness) assertBacking(t *testing.T) {
	t.Helper()
	pools, escrows := sdkmath.ZeroInt(), sdkmath.ZeroInt()
	for _, pool := range h.k.GetAllBountyPools(h.ctx) {
		amount, ok := sdkmath.NewIntFromString(pool.Balance)
		require.True(t, ok)
		pools = pools.Add(amount)
	}
	for _, ch := range h.k.GetAllChallenges(h.ctx) {
		switch ch.Status {
		case types.ChallengeStatus_CHALLENGE_STATUS_OPEN, types.ChallengeStatus_CHALLENGE_STATUS_EVIDENCE, types.ChallengeStatus_CHALLENGE_STATUS_UNDER_REVIEW:
			amount, ok := sdkmath.NewIntFromString(ch.Stake)
			require.True(t, ok)
			escrows = escrows.Add(amount)
		}
	}
	t.Logf("bank=%s pools=%s escrows=%s development=%s", h.moduleBalance(types.ModuleName), pools, escrows, h.moduleBalance("development_fund"))
	require.Equal(t, pools.Add(escrows).String(), h.moduleBalance(types.ModuleName), "pool claims must be backed separately from refundable escrows")
	h.assertSupply(t)
}

func (h *challengeAccountingHarness) assertSupply(t *testing.T) {
	t.Helper()
	total := sdkmath.ZeroInt()
	h.bank.IterateAllBalances(h.ctx, func(_ sdk.AccAddress, coin sdk.Coin) bool {
		require.Equal(t, "uzrn", coin.Denom)
		require.False(t, coin.Amount.IsNegative())
		total = total.Add(coin.Amount)
		return false
	})
	require.Equal(t, total.String(), h.bank.GetSupply(h.ctx, "uzrn").Amount.String())
}

func TestAccountingRejectedThenRewardKeepsEscrowsBacked(t *testing.T) {
	for _, funding := range []int64{0, 100_000_000} {
		for _, outcome := range []types.ChallengeOutcome{types.ChallengeOutcome_CHALLENGE_OUTCOME_UPHELD, types.ChallengeOutcome_CHALLENGE_OUTCOME_PARTIAL} {
			t.Run(fmt.Sprintf("funding_%d/%s", funding, outcome), func(t *testing.T) {
				h := newChallengeAccountingHarness(t)
				if funding > 0 {
					funder := sdk.MustAccAddressFromBech32(testAddr(10))
					h.fund(t, funder, funding)
					_, err := keeper.NewMsgServerImpl(h.k).FundBountyPool(h.ctx, &types.MsgFundBountyPool{Sender: funder.String(), Domain: "math", Amount: fmt.Sprint(funding)})
					require.NoError(t, err)
				}
				rejected := h.submit(t, 1, "math")
				h.advance(t, h.challenge(t, rejected).EvidenceDeadline)
				h.resolve(t, rejected, types.ChallengeOutcome_CHALLENGE_OUTCOME_REJECTED)
				require.Equal(t, "10000000", h.moduleBalance("development_fund"))
				h.assertBacking(t)

				// The first domain pause has elapsed; keep defaults, including 10 ZRN minimum.
				winner := h.submit(t, 2, "math")
				innocent := h.submit(t, 3, "physics")
				h.advance(t, h.challenge(t, winner).EvidenceDeadline)
				h.resolve(t, winner, outcome)
				reward := funding / 10
				returned := int64(10_000_000)
				if outcome == types.ChallengeOutcome_CHALLENGE_OUTCOME_PARTIAL {
					reward /= 2
					returned /= 2
				}
				require.Equal(t, fmt.Sprint(reward), h.challenge(t, winner).Resolution.RewardAmount)
				require.Equal(t, fmt.Sprint(returned+reward), h.bank.GetBalance(h.ctx, sdk.MustAccAddressFromBech32(testAddr(2)), "uzrn").Amount.String())
				h.assertBacking(t)
				h.advance(t, h.challenge(t, innocent).ReviewDeadline)
				require.Equal(t, types.ChallengeStatus_CHALLENGE_STATUS_EXPIRED, h.challenge(t, innocent).Status)
				require.Equal(t, "10000000", h.bank.GetBalance(h.ctx, sdk.MustAccAddressFromBech32(testAddr(3)), "uzrn").Amount.String())
				h.assertBacking(t)
				// With all escrows settled, the bank balance is exactly the pool's backing.
				require.Equal(t, fmt.Sprint(funding-reward), h.moduleBalance(types.ModuleName))
			})
		}
	}
}

func TestAccountingExpiryRefundRetriesRealBank(t *testing.T) {
	for _, failure := range []string{"insufficient_funds", "before", "after", "write_before", "write_after"} {
		t.Run(failure, func(t *testing.T) {
			h := newChallengeAccountingHarness(t)
			id := h.submit(t, 1, "math")
			ch := h.challenge(t, id)
			h.advance(t, ch.EvidenceDeadline)
			coins := sdk.NewCoins(sdk.NewInt64Coin("uzrn", 10_000_000))
			if failure == "insufficient_funds" {
				// Model existing deficient custody without destroying bank supply.
				require.NoError(t, h.bank.SendCoinsFromModuleToModule(h.ctx, types.ModuleName, "challenge_test_sink", coins))
			} else if strings.HasPrefix(failure, "write_") {
				h.writeFault.key = append(append([]byte{}, types.ChallengeKeyPrefix...), []byte(id)...)
				h.writeFault.mode = strings.TrimPrefix(failure, "write_")
			} else {
				h.bank.failRefund = failure
			}
			before := h.snapshot(t)
			eventsBefore := len(h.ctx.EventManager().Events())
			h.advance(t, ch.ReviewDeadline)
			require.Equal(t, before, h.snapshot(t), "failed refund must discard bank, auth and challenge writes")
			require.Equal(t, types.ChallengeStatus_CHALLENGE_STATUS_UNDER_REVIEW, h.challenge(t, id).Status, "overdue UNDER_REVIEW is refund-pending")
			require.Equal(t, "10000000", h.challenge(t, id).Stake)
			require.Equal(t, "0", h.bank.GetBalance(h.ctx, sdk.MustAccAddressFromBech32(testAddr(1)), "uzrn").Amount.String())
			require.Equal(t, eventsBefore, len(h.ctx.EventManager().Events()), "failed refund events must not escape its cache")
			require.Equal(t, 1, h.bank.refundCalls)
			h.assertSupply(t)
			for _, outcome := range []types.ChallengeOutcome{types.ChallengeOutcome_CHALLENGE_OUTCOME_UPHELD, types.ChallengeOutcome_CHALLENGE_OUTCOME_REJECTED, types.ChallengeOutcome_CHALLENGE_OUTCOME_PARTIAL} {
				_, err := keeper.NewMsgServerImpl(h.k).ResolveChallenge(h.ctx, &types.MsgResolveChallenge{Authority: h.k.GetAuthority(), ChallengeId: id, Outcome: outcome})
				require.ErrorIs(t, err, types.ErrChallengeNotOpen, "pending refund is no longer resolvable")
			}
			require.Equal(t, 1, h.bank.refundCalls, "overdue resolution must not attempt transfers")
			require.Equal(t, before, h.snapshot(t))
			// Failure is retained across more than one block, not merely one attempt.
			h.advance(t, ch.ReviewDeadline+1)
			require.Equal(t, before, h.snapshot(t))
			require.Equal(t, 2, h.bank.refundCalls)
			if failure == "insufficient_funds" {
				require.NoError(t, h.bank.SendCoinsFromModuleToModule(h.ctx, "challenge_test_sink", types.ModuleName, coins))
			} else {
				h.bank.failRefund = ""
				h.writeFault.mode = ""
			}
			h.advance(t, ch.ReviewDeadline+2)
			require.Equal(t, types.ChallengeStatus_CHALLENGE_STATUS_EXPIRED, h.challenge(t, id).Status)
			require.Equal(t, "10000000", h.bank.GetBalance(h.ctx, sdk.MustAccAddressFromBech32(testAddr(1)), "uzrn").Amount.String())
			require.Equal(t, 3, h.bank.refundCalls)
			h.assertBacking(t)
			h.advance(t, ch.ReviewDeadline+3)
			require.Equal(t, 3, h.bank.refundCalls, "terminal expiry must not refund twice")
			h.assertBacking(t)
		})
	}
}

func TestAccountingExpiryFailureDoesNotBlockOtherChallenges(t *testing.T) {
	h := newChallengeAccountingHarness(t)
	pending := h.submit(t, 1, "math")
	paid := h.submit(t, 2, "physics")
	ch := h.challenge(t, pending)
	h.advance(t, ch.EvidenceDeadline)
	h.bank.failRefund = "after"
	h.bank.failRecipient = testAddr(1)
	h.advance(t, ch.ReviewDeadline)
	require.Equal(t, types.ChallengeStatus_CHALLENGE_STATUS_UNDER_REVIEW, h.challenge(t, pending).Status)
	require.Equal(t, types.ChallengeStatus_CHALLENGE_STATUS_EXPIRED, h.challenge(t, paid).Status)
	require.Equal(t, "0", h.bank.GetBalance(h.ctx, sdk.MustAccAddressFromBech32(testAddr(1)), "uzrn").Amount.String())
	require.Equal(t, "10000000", h.bank.GetBalance(h.ctx, sdk.MustAccAddressFromBech32(testAddr(2)), "uzrn").Amount.String())
	h.assertBacking(t)
	active, err := keeper.NewQueryServerImpl(h.k).ActiveChallenges(h.ctx, &types.QueryActiveChallengesRequest{})
	require.NoError(t, err)
	require.Len(t, active.Challenges, 1)
	require.Equal(t, pending, active.Challenges[0].Id)
	h.bank.failRefund = ""
	h.advance(t, ch.ReviewDeadline+1)
	h.assertBacking(t)
}

func TestAccountingPendingRefundSurvivesGenesisRoundTrip(t *testing.T) {
	h := newChallengeAccountingHarness(t)
	id := h.submit(t, 1, "math")
	ch := h.challenge(t, id)
	h.advance(t, ch.EvidenceDeadline)
	h.bank.failRefund = "before"
	h.advance(t, ch.ReviewDeadline)
	genesis := h.k.ExportGenesis(h.ctx)
	require.NoError(t, genesis.Validate())
	target := newChallengeAccountingHarness(t)
	target.auth.InitGenesis(target.ctx, *h.auth.ExportGenesis(h.ctx))
	target.bank.InitGenesis(target.ctx, h.bank.ExportGenesis(h.ctx))
	target.k.InitGenesis(target.ctx, genesis)
	require.Equal(t, types.ChallengeStatus_CHALLENGE_STATUS_UNDER_REVIEW, target.challenge(t, id).Status)
	require.Equal(t, ch.ReviewDeadline, target.challenge(t, id).ReviewDeadline)
	target.assertBacking(t)
	target.advance(t, ch.ReviewDeadline+1)
	require.Equal(t, types.ChallengeStatus_CHALLENGE_STATUS_EXPIRED, target.challenge(t, id).Status)
	require.Equal(t, "10000000", target.bank.GetBalance(target.ctx, sdk.MustAccAddressFromBech32(testAddr(1)), "uzrn").Amount.String())
	target.advance(t, ch.ReviewDeadline+2)
	require.Equal(t, 1, target.bank.refundCalls)
	target.assertBacking(t)
}

func TestAccountingExpiryZeroStakeAndLegacyTerminalRecords(t *testing.T) {
	h := newChallengeAccountingHarness(t)
	// Pre-fix EXPIRED does not identify whether payment succeeded. Never replay
	// those records automatically, even if the module now has coins available.
	h.fund(t, authtypes.NewModuleAddress(types.ModuleName), 20_000_000)
	for _, status := range []types.ChallengeStatus{types.ChallengeStatus_CHALLENGE_STATUS_EXPIRED, types.ChallengeStatus_CHALLENGE_STATUS_RESOLVED} {
		h.k.SetChallenge(h.ctx, &types.CaptureChallenge{Id: status.String(), Challenger: testAddr(1), Stake: "10000000", Status: status, ReviewDeadline: 100})
	}
	require.NoError(t, h.k.AutoSubmitChallenge(h.ctx, "auto", 0, 0, "synthetic automatic challenge"))
	id := keeper.GenerateChallengeID(authtypes.NewModuleAddress(types.ModuleName).String(), "auto", h.ctx.BlockHeight())
	ch := h.challenge(t, id)
	require.Equal(t, "0", ch.Stake)
	h.advance(t, ch.ReviewDeadline)
	require.Equal(t, types.ChallengeStatus_CHALLENGE_STATUS_EXPIRED, h.challenge(t, id).Status)
	h.advance(t, ch.ReviewDeadline+1)
	require.Zero(t, h.bank.refundCalls)
	require.Equal(t, "20000000", h.moduleBalance(types.ModuleName))
	h.assertSupply(t)
}

func TestAccountingExpiryMalformedRecordStaysPending(t *testing.T) {
	for _, tc := range []struct{ name, address, stake string }{
		{"address", "invalid-address", "10000000"},
		{"amount", testAddr(1), "not-an-amount"},
		{"negative", testAddr(1), "-1"},
		{"overflow", testAddr(1), strings.Repeat("9", 100)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newChallengeAccountingHarness(t)
			h.k.SetChallenge(h.ctx, &types.CaptureChallenge{Id: "malformed", Challenger: tc.address, Stake: tc.stake, Status: types.ChallengeStatus_CHALLENGE_STATUS_UNDER_REVIEW, ReviewDeadline: 100})
			before := h.snapshot(t)
			h.advance(t, 100)
			h.advance(t, 101)
			require.Equal(t, before, h.snapshot(t))
			require.Zero(t, h.bank.refundCalls)
		})
	}
}

func TestAccountingResolutionDeadlineBoundary(t *testing.T) {
	for _, outcome := range []types.ChallengeOutcome{types.ChallengeOutcome_CHALLENGE_OUTCOME_UPHELD, types.ChallengeOutcome_CHALLENGE_OUTCOME_REJECTED, types.ChallengeOutcome_CHALLENGE_OUTCOME_PARTIAL} {
		t.Run(outcome.String(), func(t *testing.T) {
			h := newChallengeAccountingHarness(t)
			id := h.submit(t, 1, "math")
			ch := h.challenge(t, id)
			h.advance(t, ch.EvidenceDeadline)
			before := h.snapshot(t)
			// Refuse an overdue record even before its first expiry attempt.
			h.ctx = h.ctx.WithBlockHeight(int64(ch.ReviewDeadline))
			_, err := keeper.NewMsgServerImpl(h.k).ResolveChallenge(h.ctx, &types.MsgResolveChallenge{Authority: h.k.GetAuthority(), ChallengeId: id, Outcome: outcome})
			require.ErrorIs(t, err, types.ErrChallengeNotOpen)
			require.Equal(t, before, h.snapshot(t))
			require.Zero(t, h.bank.refundCalls)
			// Last review block remains valid; this fixture intentionally rewinds
			// only its context height after the refused call made no state change.
			h.ctx = h.ctx.WithBlockHeight(int64(ch.ReviewDeadline - 1))
			h.resolve(t, id, outcome)
			h.assertBacking(t)
		})
	}
}

func TestAccountingPendingRefundDoesNotSuppressAutomaticChallenge(t *testing.T) {
	h := newChallengeAccountingHarness(t)
	id := h.submit(t, 1, "math")
	ch := h.challenge(t, id)
	h.advance(t, ch.EvidenceDeadline)
	h.ctx = h.ctx.WithBlockHeight(int64(ch.ReviewDeadline - 1))
	require.NoError(t, h.k.AutoSubmitChallenge(h.ctx, "math", 0, 0, "synthetic signal"))
	require.Len(t, h.k.GetChallengesByDomain(h.ctx, "math"), 1, "live adjudication still deduplicates automatic challenges")
	h.bank.failRefund = "before"
	h.advance(t, ch.ReviewDeadline)
	require.Equal(t, types.ChallengeStatus_CHALLENGE_STATUS_UNDER_REVIEW, h.challenge(t, id).Status)
	require.NoError(t, h.k.AutoSubmitChallenge(h.ctx, "math", 0, 0, "synthetic signal"))
	require.Len(t, h.k.GetChallengesByDomain(h.ctx, "math"), 2, "a pending financial exit must not suppress new adjudication")
	autoID := keeper.GenerateChallengeID(authtypes.NewModuleAddress(types.ModuleName).String(), "math", h.ctx.BlockHeight())
	require.Equal(t, "0", h.challenge(t, autoID).Stake)
	h.ctx = h.ctx.WithBlockHeight(int64(ch.ReviewDeadline + 1))
	require.NoError(t, h.k.AutoSubmitChallenge(h.ctx, "math", 0, 0, "repeated signal"))
	require.Len(t, h.k.GetChallengesByDomain(h.ctx, "math"), 2, "the new automatic challenge still deduplicates")
	h.bank.failRefund = ""
	h.advance(t, ch.ReviewDeadline+2)
	require.Equal(t, types.ChallengeStatus_CHALLENGE_STATUS_EXPIRED, h.challenge(t, id).Status)
	require.Equal(t, types.ChallengeStatus_CHALLENGE_STATUS_UNDER_REVIEW, h.challenge(t, autoID).Status)
	h.assertBacking(t)
}
