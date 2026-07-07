package integration_test

import (
	"context"
	"math/big"
	"testing"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	// Blank import triggers init() which sets bech32 prefixes to "zrn"/"zrnpub".
	_ "github.com/zerone-chain/zerone/app"

	vestingkeeper "github.com/zerone-chain/zerone/x/vesting_rewards/keeper"
	vestingtypes "github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

// ---------- Shared Mock Bank Keeper ----------

type mockBankKeeper struct {
	mintedCoins   sdk.Coins
	burnedCoins   sdk.Coins
	sentToAccount map[string]sdk.Coins // recipientAddr -> coins
	sentToModule  map[string]sdk.Coins // moduleName -> coins
	sentFromAcct  map[string]sdk.Coins // senderAddr -> coins debited via SendCoinsFromAccountToModule
	p2pSent       map[string]sdk.Coins // "from->to" -> coins
	supply        map[string]sdkmath.Int
	balances      map[string]sdk.Coins // addr -> coins (for GetAllBalances)
	mintErr       error
}

func newMockBankKeeper() *mockBankKeeper {
	return &mockBankKeeper{
		sentToAccount: make(map[string]sdk.Coins),
		sentToModule:  make(map[string]sdk.Coins),
		sentFromAcct:  make(map[string]sdk.Coins),
		p2pSent:       make(map[string]sdk.Coins),
		supply:        make(map[string]sdkmath.Int),
		balances:      make(map[string]sdk.Coins),
	}
}

func (m *mockBankKeeper) MintCoins(_ context.Context, _ string, amounts sdk.Coins) error {
	if m.mintErr != nil {
		return m.mintErr
	}
	m.mintedCoins = m.mintedCoins.Add(amounts...)
	for _, coin := range amounts {
		cur, ok := m.supply[coin.Denom]
		if !ok {
			cur = sdkmath.ZeroInt()
		}
		m.supply[coin.Denom] = cur.Add(coin.Amount)
	}
	return nil
}

func (m *mockBankKeeper) BurnCoins(_ context.Context, _ string, amounts sdk.Coins) error {
	m.burnedCoins = m.burnedCoins.Add(amounts...)
	for _, coin := range amounts {
		cur, ok := m.supply[coin.Denom]
		if !ok {
			cur = sdkmath.ZeroInt()
		}
		m.supply[coin.Denom] = cur.Sub(coin.Amount)
	}
	return nil
}

func (m *mockBankKeeper) GetSupply(_ context.Context, denom string) sdk.Coin {
	if amt, ok := m.supply[denom]; ok {
		return sdk.NewCoin(denom, amt)
	}
	return sdk.NewCoin(denom, sdkmath.ZeroInt())
}

func (m *mockBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, _ string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	key := recipientAddr.String()
	m.sentToAccount[key] = m.sentToAccount[key].Add(amt...)
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToModule(_ context.Context, _ string, recipientModule string, amt sdk.Coins) error {
	m.sentToModule[recipientModule] = m.sentToModule[recipientModule].Add(amt...)
	return nil
}

func (m *mockBankKeeper) GetAllBalances(_ context.Context, addr sdk.AccAddress) sdk.Coins {
	if coins, ok := m.balances[addr.String()]; ok {
		return coins
	}
	return sdk.Coins{}
}

func (m *mockBankKeeper) GetBalance(_ context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	if coins, ok := m.balances[addr.String()]; ok {
		return sdk.NewCoin(denom, coins.AmountOf(denom))
	}
	return sdk.NewCoin(denom, sdkmath.ZeroInt())
}

func (m *mockBankKeeper) SendCoinsFromAccountToModule(_ context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	key := senderAddr.String()
	m.sentFromAcct[key] = m.sentFromAcct[key].Add(amt...)
	m.sentToModule[recipientModule] = m.sentToModule[recipientModule].Add(amt...)
	return nil
}

func (m *mockBankKeeper) SendCoins(_ context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error {
	key := fromAddr.String() + "->" + toAddr.String()
	m.p2pSent[key] = m.p2pSent[key].Add(amt...)
	m.sentToAccount[toAddr.String()] = m.sentToAccount[toAddr.String()].Add(amt...)
	return nil
}

// totalSentToModule returns the total uzrn sent to a module account.
func (m *mockBankKeeper) totalSentToModule(module string) sdkmath.Int {
	if coins, ok := m.sentToModule[module]; ok {
		return coins.AmountOf("uzrn")
	}
	return sdkmath.ZeroInt()
}

// totalSentToAddr returns the total uzrn sent to an address.
func (m *mockBankKeeper) totalSentToAddr(addr string) sdkmath.Int {
	if coins, ok := m.sentToAccount[addr]; ok {
		return coins.AmountOf("uzrn")
	}
	return sdkmath.ZeroInt()
}

// totalMinted returns the total uzrn minted.
func (m *mockBankKeeper) totalMinted() sdkmath.Int {
	return m.mintedCoins.AmountOf("uzrn")
}

// totalBurned returns the total uzrn burned.
func (m *mockBankKeeper) totalBurned() sdkmath.Int {
	return m.burnedCoins.AmountOf("uzrn")
}

// ---------- Mock Staking Keeper ----------

type mockStakingKeeper struct {
	activeCount uint32
}

func (m *mockStakingKeeper) GetActiveValidatorCount(_ context.Context) uint32 {
	return m.activeCount
}

func (m *mockStakingKeeper) GetValidatorByConsAddr(_ context.Context, _ sdk.ConsAddress) (stakingtypes.Validator, error) {
	return stakingtypes.Validator{}, stakingtypes.ErrNoValidatorFound
}

// ---------- Test Harness ----------

type revenueTestHarness struct {
	bk            *mockBankKeeper
	sk            *mockStakingKeeper
	vestingKeeper vestingkeeper.Keeper
	ctx           sdk.Context
	founderAddr   sdk.AccAddress
	producerAddr  sdk.AccAddress
	submitterAddr sdk.AccAddress
}

func setupRevenueHarness(t *testing.T) *revenueTestHarness {
	t.Helper()

	bk := newMockBankKeeper()
	sk := &mockStakingKeeper{activeCount: 22}

	// Addresses
	founderAddr := sdk.AccAddress("founder_____________")
	producerAddr := sdk.AccAddress("producer____________")
	submitterAddr := sdk.AccAddress("submitter___________")

	// --- Set up vesting_rewards keeper ---
	vestingStoreKey := storetypes.NewKVStoreKey(vestingtypes.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(vestingStoreKey, storetypes.StoreTypeIAVL, db)

	if err := stateStore.LoadLatestVersion(); err != nil {
		t.Fatalf("failed to load latest version: %v", err)
	}

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	// Vesting keeper with founder address configured
	vk := vestingkeeper.NewKeeper(cdc, runtime.NewKVStoreService(vestingStoreKey), bk, sk, "authority")
	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 1000}, false, log.NewNopLogger())

	// Init vesting genesis with founder configured
	gs := vestingtypes.DefaultGenesis()
	gs.Params.FounderAddress = founderAddr.String()
	vk.InitGenesis(ctx, gs)

	return &revenueTestHarness{
		bk:            bk,
		sk:            sk,
		vestingKeeper: vk,
		ctx:           ctx,
		founderAddr:   founderAddr,
		producerAddr:  producerAddr,
		submitterAddr: submitterAddr,
	}
}

// testApplyDecay is a local copy of the unexported applyDecay from
// x/vesting_rewards/keeper/rewards.go:267. It computes:
// amount * (decayBps/1000000)^epochs using integer exponentiation by squaring.
func testApplyDecay(amount *big.Int, decayBps uint64, epochs uint64) *big.Int {
	if epochs == 0 {
		return new(big.Int).Set(amount)
	}

	denom := big.NewInt(1000000)
	base := big.NewInt(int64(decayBps))
	exp := epochs

	result := new(big.Int).Set(denom) // start at 1.0
	for exp > 0 {
		if exp%2 == 1 {
			result.Mul(result, base)
			result.Div(result, denom)
		}
		base.Mul(base, base)
		base.Div(base, denom)
		exp /= 2
	}

	out := new(big.Int).Mul(amount, result)
	out.Div(out, denom)
	return out
}
