package keeper_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"math"
	"math/big"
	"strings"
	"testing"

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

	"github.com/zerone-chain/zerone/x/staking/keeper"
	"github.com/zerone-chain/zerone/x/staking/types"
)

// Real SDK bank/auth stores participate in CacheContext. The optional fault
// happens AFTER the real bank transfer to prove rollback across both stores.
type accountingBank struct {
	bankkeeper.BaseKeeper
	failAt, calls int
}

func (b *accountingBank) SendCoinsFromModuleToAccount(ctx context.Context, module string, addr sdk.AccAddress, coins sdk.Coins) error {
	if err := b.BaseKeeper.SendCoinsFromModuleToAccount(ctx, module, addr, coins); err != nil {
		return err
	}
	b.calls++
	if b.failAt > 0 && b.calls == b.failAt {
		return errors.New("injected post-transfer failure")
	}
	return nil
}

type accountingHarness struct {
	k    keeper.Keeper
	ctx  sdk.Context
	bank *accountingBank
	key  *storetypes.KVStoreKey
}

func newAccountingHarness(t *testing.T, enable bool) *accountingHarness {
	t.Helper()
	keys := storetypes.NewKVStoreKeys(types.StoreKey, authtypes.StoreKey, banktypes.StoreKey)
	db := dbm.NewMemDB()
	state := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	for _, key := range keys {
		state.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	}
	require.NoError(t, state.LoadLatestVersion())
	ctx := sdk.NewContext(state, cmtproto.Header{Height: 100}, false, log.NewNopLogger())
	registry := codectypes.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(registry)
	banktypes.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress("gov").String()
	perms := map[string][]string{types.ModuleName: nil, "accounting_test_fund": {authtypes.Minter}, "development_fund": nil, "vindication_escrow": nil}
	ak := authkeeper.NewAccountKeeper(cdc, runtime.NewKVStoreService(keys[authtypes.StoreKey]), authtypes.ProtoBaseAccount, perms, addresscodec.NewBech32Codec("zrn"), "zrn", authority)
	ak.InitGenesis(ctx, *authtypes.DefaultGenesisState())
	bk := bankkeeper.NewBaseKeeper(cdc, runtime.NewKVStoreService(keys[banktypes.StoreKey]), ak, map[string]bool{}, authority, log.NewNopLogger())
	bk.InitGenesis(ctx, banktypes.DefaultGenesisState())
	for name := range perms {
		ak.GetModuleAccount(ctx, name)
	}
	b := &accountingBank{BaseKeeper: bk}
	k := keeper.NewKeeper(cdc, keys[types.StoreKey], ak, b, authority)
	k.InitGenesis(ctx, types.DefaultGenesisState())
	p := k.GetParams(ctx)
	p.UnbondingPeriod = 10
	p.RedelegationCooldownBlocks = 1
	k.SetParams(ctx, p)
	h := &accountingHarness{k: k, ctx: ctx, bank: b, key: keys[types.StoreKey]}
	if enable {
		require.NoError(t, k.EnableAccountingSafety(ctx))
	}
	return h
}
func (h *accountingHarness) fund(t *testing.T, addr string, amount int64) {
	t.Helper()
	coins := sdk.NewCoins(sdk.NewInt64Coin("uzrn", amount))
	address, err := sdk.AccAddressFromBech32(addr)
	require.NoError(t, err)
	require.NoError(t, h.bank.MintCoins(h.ctx, "accounting_test_fund", coins))
	require.NoError(t, h.bank.BaseKeeper.SendCoinsFromModuleToAccount(h.ctx, "accounting_test_fund", address, coins))
}
func (h *accountingHarness) register(t *testing.T, seed string) string {
	t.Helper()
	addr := testAddr(seed)
	h.fund(t, addr, 10_000_000)
	_, err := keeper.NewMsgServerImpl(h.k).RegisterValidator(h.ctx, &types.MsgRegisterValidator{Operator: addr, ConsensusPubkey: "legacy-description-only", SelfDelegation: "1000000", Did: "did:test:" + seed})
	require.NoError(t, err)
	h.assertConserved(t)
	return addr
}
func (h *accountingHarness) balance(addr string) sdkmath.Int {
	a, _ := sdk.AccAddressFromBech32(addr)
	return h.bank.GetBalance(h.ctx, a, "uzrn").Amount
}
func (h *accountingHarness) snapshot() map[string]string {
	result := map[string]string{}
	it := h.ctx.KVStore(h.key).Iterator(nil, nil)
	defer it.Close()
	for ; it.Valid(); it.Next() {
		result[string(it.Key())] = string(it.Value())
	}
	return result
}
func (h *accountingHarness) assertConserved(t *testing.T) {
	t.Helper()
	liabilities := new(big.Int)
	self, delegated := map[string]*big.Int{}, map[string]*big.Int{}
	h.k.IterateDelegations(h.ctx, func(d *types.Delegation) bool {
		amount, ok := new(big.Int).SetString(d.Amount, 10)
		require.True(t, ok)
		require.Positive(t, amount.Sign())
		liabilities.Add(liabilities, amount)
		dest := delegated
		if d.DelegatorAddress == d.ValidatorAddress {
			dest = self
		}
		if dest[d.ValidatorAddress] == nil {
			dest[d.ValidatorAddress] = new(big.Int)
		}
		dest[d.ValidatorAddress].Add(dest[d.ValidatorAddress], amount)
		return false
	})
	h.k.IterateUnbondings(h.ctx, func(u *types.UnbondingEntry) bool {
		if u.Status == "pending" {
			amount, ok := new(big.Int).SetString(u.Amount, 10)
			require.True(t, ok)
			liabilities.Add(liabilities, amount)
		}
		return false
	})
	require.Equal(t, liabilities.String(), h.bank.GetBalance(h.ctx, authtypes.NewModuleAddress(types.ModuleName), "uzrn").Amount.String())
	h.k.IterateValidators(h.ctx, func(v *types.Validator) bool {
		s, d := self[v.OperatorAddress], delegated[v.OperatorAddress]
		if s == nil {
			s = new(big.Int)
		}
		if d == nil {
			d = new(big.Int)
		}
		require.Equal(t, s.String(), v.SelfDelegation)
		require.Equal(t, d.String(), v.DelegatedStake)
		require.Equal(t, new(big.Int).Add(s, d).String(), v.TotalStake)
		return false
	})
	require.NoError(t, h.k.ValidateAccountingSafety(h.ctx))
}

func TestAccountingSafetyRealBankLifecycle(t *testing.T) {
	h := newAccountingHarness(t, true)
	a := h.register(t, "acct-a")
	b := h.register(t, "acct-b")
	d := testAddr("acct-delegator")
	h.fund(t, d, 1_000_000)
	ms := keeper.NewMsgServerImpl(h.k)
	for _, msg := range []*types.MsgDelegate{{Delegator: a, Validator: a, Amount: "200000"}, {Delegator: d, Validator: a, Amount: "300000"}} {
		_, err := ms.Delegate(h.ctx, msg)
		require.NoError(t, err)
		h.assertConserved(t)
	}
	// One owner crosses self -> external and external -> self using one claim ledger.
	_, err := ms.Redelegate(h.ctx, &types.MsgRedelegate{Delegator: a, SrcValidator: a, DstValidator: b, Amount: "100000"})
	require.NoError(t, err)
	h.assertConserved(t)
	h.ctx = h.ctx.WithBlockHeight(102)
	_, err = ms.Redelegate(h.ctx, &types.MsgRedelegate{Delegator: a, SrcValidator: b, DstValidator: a, Amount: "50000"})
	require.NoError(t, err)
	h.assertConserved(t)
	_, err = ms.UpdateValidatorStake(h.ctx, &types.MsgUpdateValidatorStake{Operator: a, Amount: "50000", Increase: true})
	require.NoError(t, err)
	h.assertConserved(t)
	_, err = ms.Undelegate(h.ctx, &types.MsgUndelegate{Delegator: a, Validator: a, Amount: "200000"})
	require.NoError(t, err)
	h.assertConserved(t)
	_, err = ms.UpdateValidatorStake(h.ctx, &types.MsgUpdateValidatorStake{Operator: a, Amount: "100000", Increase: false})
	require.NoError(t, err)
	h.assertConserved(t)
	_, err = ms.Undelegate(h.ctx, &types.MsgUndelegate{Delegator: d, Validator: a, Amount: "300000"})
	require.NoError(t, err)
	h.assertConserved(t)
	beforeA, beforeD := h.balance(a), h.balance(d)
	h.ctx = h.ctx.WithBlockHeight(112)
	h.k.BeginBlocker(h.ctx)
	require.Equal(t, beforeA.AddRaw(300000), h.balance(a))
	require.Equal(t, beforeD.AddRaw(300000), h.balance(d))
	h.assertConserved(t)
	state := h.snapshot()
	h.k.BeginBlocker(h.ctx)
	require.Equal(t, state, h.snapshot())
	require.Equal(t, beforeA.AddRaw(300000), h.balance(a))
	gs := h.k.ExportGenesis(h.ctx)
	require.True(t, gs.AccountingSafetyEnabled)
	require.Len(t, gs.RedelegationCooldowns, 1)
	require.Equal(t, uint64(102), gs.RedelegationCooldowns[0].Height)
	other := newAccountingHarness(t, false)
	other.k.InitGenesis(other.ctx, gs)
	coins := h.bank.GetAllBalances(h.ctx, authtypes.NewModuleAddress(types.ModuleName))
	require.NoError(t, other.bank.MintCoins(other.ctx, "accounting_test_fund", coins))
	require.NoError(t, other.bank.SendCoinsFromModuleToModule(other.ctx, "accounting_test_fund", types.ModuleName, coins))
	require.False(t, other.k.AccountingSafetyEnabled(other.ctx), "genesis declaration is not custody validation")
	require.NoError(t, other.k.EnableAccountingSafety(other.ctx))
	require.Equal(t, gs, other.k.ExportGenesis(other.ctx))
	other.assertConserved(t)
}

func TestAccountingSafetyNoAggregateOnlySecondWithdrawal(t *testing.T) {
	h := newAccountingHarness(t, true)
	a := h.register(t, "acct-double")
	ms := keeper.NewMsgServerImpl(h.k)
	_, err := ms.Undelegate(h.ctx, &types.MsgUndelegate{Delegator: a, Validator: a, Amount: "1000000"})
	require.NoError(t, err)
	h.assertConserved(t)
	before := h.snapshot()
	_, err = ms.UpdateValidatorStake(h.ctx, &types.MsgUpdateValidatorStake{Operator: a, Amount: "1000000"})
	require.Error(t, err)
	require.Equal(t, before, h.snapshot())
	// Even a plausible-looking historical aggregate cannot create a claimant.
	v, _ := h.k.GetValidator(h.ctx, a)
	v.SelfDelegation = "1000000"
	v.TotalStake = "1000000"
	bz, _ := json.Marshal(v)
	h.ctx.KVStore(h.key).Set(types.ValidatorKey(a), bz)
	before = h.snapshot()
	_, err = ms.UpdateValidatorStake(h.ctx, &types.MsgUpdateValidatorStake{Operator: a, Amount: "1000000"})
	require.Error(t, err)
	require.Equal(t, before, h.snapshot())
	h.ctx = h.ctx.WithBlockHeight(110)
	balance := h.balance(a)
	h.k.BeginBlocker(h.ctx)
	require.Equal(t, balance, h.balance(a))
	require.Equal(t, before, h.snapshot())
}

func TestAccountingSafetySlashesPreserveEveryClaim(t *testing.T) {
	h := newAccountingHarness(t, true)
	a := h.register(t, "acct-no-slash")
	d := testAddr("acct-no-slash-d")
	h.fund(t, d, 500000)
	_, err := keeper.NewMsgServerImpl(h.k).Delegate(h.ctx, &types.MsgDelegate{Delegator: d, Validator: a, Amount: "500000"})
	require.NoError(t, err)
	before := h.snapshot()
	balance := h.bank.GetAllBalances(h.ctx, authtypes.NewModuleAddress(types.ModuleName))
	h.k.SlashValidator(h.ctx, a, big.NewInt(2_000_000), "epistemic_disagreement")
	actual := h.k.SlashValidatorToModule(h.ctx, a, big.NewInt(2_000_000), "vindication_escrow", "epistemic_disagreement")
	require.Zero(t, actual.Sign())
	require.Equal(t, before, h.snapshot())
	require.Equal(t, balance, h.bank.GetAllBalances(h.ctx, authtypes.NewModuleAddress(types.ModuleName)))
	for _, module := range []string{"vindication_escrow", "development_fund"} {
		require.True(t, h.bank.GetAllBalances(h.ctx, authtypes.NewModuleAddress(module)).IsZero())
	}
	h.assertConserved(t)
	events := h.ctx.EventManager().Events()
	require.Equal(t, "zerone.staking.monetary_slash_not_applied", events[len(events)-1].Type)
}

func TestAccountingSafetyPayoutBatchRollsBackBankFailure(t *testing.T) {
	h := newAccountingHarness(t, true)
	a := h.register(t, "acct-batch-a")
	b := h.register(t, "acct-batch-b")
	ms := keeper.NewMsgServerImpl(h.k)
	for _, a := range []string{a, b} {
		_, err := ms.Undelegate(h.ctx, &types.MsgUndelegate{Delegator: a, Validator: a, Amount: "100000"})
		require.NoError(t, err)
	}
	before := h.snapshot()
	aBalance, bBalance := h.balance(a), h.balance(b)
	h.ctx = h.ctx.WithBlockHeight(110)
	h.bank.failAt = 2
	h.bank.calls = 0
	h.k.BeginBlocker(h.ctx)
	require.Equal(t, before, h.snapshot())
	require.Equal(t, aBalance, h.balance(a))
	require.Equal(t, bBalance, h.balance(b))
	h.assertConserved(t)
	h.bank.failAt = 0
	h.k.BeginBlocker(h.ctx)
	require.Equal(t, aBalance.AddRaw(100000), h.balance(a))
	require.Equal(t, bBalance.AddRaw(100000), h.balance(b))
	h.assertConserved(t)
	h.k.BeginBlocker(h.ctx)
	require.Equal(t, aBalance.AddRaw(100000), h.balance(a))
	require.Equal(t, bBalance.AddRaw(100000), h.balance(b))
}

func TestAccountingSafetyActivationRejectsUnresolvedHistory(t *testing.T) {
	cases := map[string]func(*accountingHarness, string){
		"missing reverse": func(h *accountingHarness, a string) {
			h.ctx.KVStore(h.key).Delete(types.ValidatorDelegationIndexKey(a, a))
		},
		"orphan reverse": func(h *accountingHarness, a string) {
			h.ctx.KVStore(h.key).Set(types.ValidatorDelegationIndexKey(a, testAddr("orphan")), []byte{1})
		},
		"orphan DID": func(h *accountingHarness, a string) {
			h.ctx.KVStore(h.key).Set(types.ValidatorByDIDKey("did:orphan"), []byte(a))
		},
		"malformed claim": func(h *accountingHarness, a string) { h.ctx.KVStore(h.key).Set(types.DelegationKey(a, a), []byte("{")) },
		"duplicate JSON": func(h *accountingHarness, a string) {
			bz := h.ctx.KVStore(h.key).Get(types.DelegationKey(a, a))
			bz = bytes.Replace(bz, []byte(`"amount":"1000000"`), []byte(`"amount":"1","amount":"1000000"`), 1)
			h.ctx.KVStore(h.key).Set(types.DelegationKey(a, a), bz)
		},
		"case-aliased JSON": func(h *accountingHarness, a string) {
			bz := h.ctx.KVStore(h.key).Get(types.DelegationKey(a, a))
			bz = bytes.Replace(bz, []byte(`"amount":"1000000"`), []byte(`"Amount":"1","amount":"1000000"`), 1)
			h.ctx.KVStore(h.key).Set(types.DelegationKey(a, a), bz)
		},
		"invalid UTF8": func(h *accountingHarness, a string) {
			bz := h.ctx.KVStore(h.key).Get(types.ValidatorKey(a))
			bz = append(bz[:len(bz)-1], []byte(",\"moniker\":\"\xff\"}")...)
			h.ctx.KVStore(h.key).Set(types.ValidatorKey(a), bz)
		},
		"nested field alias": func(h *accountingHarness, _ string) {
			bz := h.ctx.KVStore(h.key).Get(types.ParamsKey)
			bz = bytes.Replace(bz, []byte(`"min_stake"`), []byte(`"Min_Stake"`), 1)
			h.ctx.KVStore(h.key).Set(types.ParamsKey, bz)
		},
		"hostile nesting": func(h *accountingHarness, _ string) {
			bz := []byte(`{"tier_configs":` + strings.Repeat("[", 10000) + `0` + strings.Repeat("]", 10000) + `}`)
			h.ctx.KVStore(h.key).Set(types.ParamsKey, bz)
		},
		"deficit": func(h *accountingHarness, a string) {
			addr, _ := sdk.AccAddressFromBech32(a)
			require.NoError(t, h.bank.BaseKeeper.SendCoinsFromModuleToAccount(h.ctx, types.ModuleName, addr, sdk.NewCoins(sdk.NewInt64Coin("uzrn", 1))))
		},
		"surplus": func(h *accountingHarness, a string) {
			addr, _ := sdk.AccAddressFromBech32(a)
			require.NoError(t, h.bank.SendCoinsFromAccountToModule(h.ctx, addr, types.ModuleName, sdk.NewCoins(sdk.NewInt64Coin("uzrn", 1))))
		},
		"bad sentinel": func(h *accountingHarness, _ string) { h.ctx.KVStore(h.key).Set([]byte("_iavl_init"), []byte{2}) },
	}
	for name, corrupt := range cases {
		t.Run(name, func(t *testing.T) {
			h := newAccountingHarness(t, true)
			a := h.register(t, "acct-activation")
			h.ctx.KVStore(h.key).Delete(types.AccountingSafetyKey)
			corrupt(h, a)
			before := h.snapshot()
			require.Error(t, h.k.EnableAccountingSafety(h.ctx))
			require.False(t, h.k.AccountingSafetyEnabled(h.ctx))
			require.Equal(t, before, h.snapshot())
		})
	}
}

func TestAccountingSafetyPayoutRetainsUnresolvedClaims(t *testing.T) {
	for _, kind := range []string{"deficit", "surplus", "malformed"} {
		t.Run(kind, func(t *testing.T) {
			h := newAccountingHarness(t, true)
			a := h.register(t, "acct-held-payout")
			ms := keeper.NewMsgServerImpl(h.k)
			_, err := ms.Undelegate(h.ctx, &types.MsgUndelegate{Delegator: a, Validator: a, Amount: "100000"})
			require.NoError(t, err)
			addr, _ := sdk.AccAddressFromBech32(a)
			one := sdk.NewCoins(sdk.NewInt64Coin("uzrn", 1))
			switch kind {
			case "deficit":
				require.NoError(t, h.bank.BaseKeeper.SendCoinsFromModuleToAccount(h.ctx, types.ModuleName, addr, one))
			case "surplus":
				require.NoError(t, h.bank.SendCoinsFromAccountToModule(h.ctx, addr, types.ModuleName, one))
			case "malformed":
				h.ctx.KVStore(h.key).Set(types.DelegationKey(a, a), []byte(`{"amount":"900000","Amount":"1"}`))
			}
			before, balance := h.snapshot(), h.balance(a)
			h.ctx = h.ctx.WithBlockHeight(110)
			h.k.BeginBlocker(h.ctx)
			require.Equal(t, before, h.snapshot())
			require.Equal(t, balance, h.balance(a))
			require.Error(t, h.k.ValidateAccountingSafety(h.ctx))
		})
	}
}

func TestAccountingSafetyMetadataAdmissionCannotPoisonPayoutAudit(t *testing.T) {
	h := newAccountingHarness(t, true)
	a := h.register(t, "acct-metadata")
	ms := keeper.NewMsgServerImpl(h.k)
	before := h.snapshot()
	p := h.k.GetParams(h.ctx)
	p.TierConfigs[0].AllowedCategories = make([]string, 40000)
	for i := range p.TierConfigs[0].AllowedCategories {
		p.TierConfigs[0].AllowedCategories[i] = "x"
	}
	_, err := ms.UpdateParams(h.ctx, &types.MsgUpdateParams{Authority: h.k.GetAuthority(), Params: p})
	require.Error(t, err)
	require.Equal(t, before, h.snapshot())
	_, err = ms.RegisterValidator(h.ctx, &types.MsgRegisterValidator{Operator: testAddr("oversized-key"), ConsensusPubkey: strings.Repeat("x", 5000), SelfDelegation: "1000000"})
	require.Error(t, err)
	require.Equal(t, before, h.snapshot())
	_, err = ms.Undelegate(h.ctx, &types.MsgUndelegate{Delegator: a, Validator: a, Amount: "1000000"})
	require.NoError(t, err)
	h.ctx = h.ctx.WithBlockHeight(110)
	h.k.BeginBlocker(h.ctx)
	h.assertConserved(t)
}

func TestAccountingSafetyAdmissionRollbackAndOverflow(t *testing.T) {
	h := newAccountingHarness(t, true)
	a := h.register(t, "acct-failure")
	ms := keeper.NewMsgServerImpl(h.k)
	before := h.snapshot()
	_, err := ms.Delegate(h.ctx, &types.MsgDelegate{Delegator: testAddr("unfunded"), Validator: a, Amount: "100"})
	require.Error(t, err)
	require.Equal(t, before, h.snapshot())
	_, err = ms.Redelegate(h.ctx, &types.MsgRedelegate{Delegator: a, SrcValidator: a, DstValidator: a, Amount: "1"})
	require.ErrorIs(t, err, types.ErrSameValidator)
	require.Equal(t, before, h.snapshot())
	h.ctx = h.ctx.WithBlockHeight(math.MaxInt64)
	_, err = ms.Undelegate(h.ctx, &types.MsgUndelegate{Delegator: a, Validator: a, Amount: "1"})
	require.Error(t, err)
	require.Equal(t, before, h.snapshot())
	h.ctx = h.ctx.WithBlockHeight(100)
	h.ctx.KVStore(h.key).Set(types.UnbondingSeqKey, types.Uint64ToBytes(math.MaxUint64))
	before = h.snapshot()
	_, err = ms.Undelegate(h.ctx, &types.MsgUndelegate{Delegator: a, Validator: a, Amount: "1"})
	require.Error(t, err)
	require.Equal(t, before, h.snapshot())
}

func TestAccountingSafetyFullExitAtCapacity(t *testing.T) {
	h := newAccountingHarness(t, true)
	a := h.register(t, "acct-capacity")
	store := h.ctx.KVStore(h.key)
	store.Set([]byte("_iavl_init"), []byte{1})
	count := len(h.snapshot())
	for i := count; i < keeper.MaxAccountingStoreEntries; i++ {
		addr := make([]byte, 20)
		binary.BigEndian.PutUint64(addr[12:], uint64(i+1))
		store.Set(types.RedelegationCooldownKey(sdk.AccAddress(addr).String()), types.Uint64ToBytes(1))
	}
	require.NoError(t, h.k.ValidateAccountingSafety(h.ctx))
	before := h.snapshot()
	ms := keeper.NewMsgServerImpl(h.k)
	_, err := ms.Undelegate(h.ctx, &types.MsgUndelegate{Delegator: a, Validator: a, Amount: "1"})
	require.Error(t, err)
	require.Equal(t, before, h.snapshot(), "partial admission at capacity is atomic")
	_, err = ms.Undelegate(h.ctx, &types.MsgUndelegate{Delegator: a, Validator: a, Amount: "1000000"})
	require.NoError(t, err, "full exit frees claim/index keys before adding withdrawal")
	balance := h.balance(a)
	h.ctx = h.ctx.WithBlockHeight(110)
	h.k.BeginBlocker(h.ctx)
	require.Equal(t, balance.AddRaw(1000000), h.balance(a))
	h.assertConserved(t)
}
