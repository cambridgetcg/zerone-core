package keeper_test

import (
	"context"
	"fmt"
	"testing"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/compute_pool/keeper"
	"github.com/zerone-chain/zerone/x/compute_pool/types"
)

// -----------------------------------------------------------------------
// Mock BankKeeper
// -----------------------------------------------------------------------

type mockBankKeeper struct {
	balances map[string]sdkmath.Int
}

func newMockBankKeeper() *mockBankKeeper {
	return &mockBankKeeper{
		balances: make(map[string]sdkmath.Int),
	}
}

func (m *mockBankKeeper) setBalance(addr sdk.AccAddress, denom string, amount sdkmath.Int) {
	m.balances[addr.String()+"/"+denom] = amount
}

func (m *mockBankKeeper) SendCoinsFromAccountToModule(_ context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	for _, coin := range amt {
		key := senderAddr.String() + "/" + coin.Denom
		bal, ok := m.balances[key]
		if !ok {
			bal = sdkmath.ZeroInt()
		}
		if bal.LT(coin.Amount) {
			return fmt.Errorf("insufficient balance for send to module")
		}
		m.balances[key] = bal.Sub(coin.Amount)
		modKey := recipientModule + "/" + coin.Denom
		modBal, ok := m.balances[modKey]
		if !ok {
			modBal = sdkmath.ZeroInt()
		}
		m.balances[modKey] = modBal.Add(coin.Amount)
	}
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	for _, coin := range amt {
		modKey := senderModule + "/" + coin.Denom
		modBal, ok := m.balances[modKey]
		if !ok {
			modBal = sdkmath.ZeroInt()
		}
		if modBal.LT(coin.Amount) {
			return fmt.Errorf("insufficient module balance")
		}
		m.balances[modKey] = modBal.Sub(coin.Amount)

		key := recipientAddr.String() + "/" + coin.Denom
		bal, ok := m.balances[key]
		if !ok {
			bal = sdkmath.ZeroInt()
		}
		m.balances[key] = bal.Add(coin.Amount)
	}
	return nil
}

// -----------------------------------------------------------------------
// Test Setup
// -----------------------------------------------------------------------

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("zrn", "zrnpub")
	config.SetBech32PrefixForValidator("zrnvaloper", "zrnvaloperpub")
	config.SetBech32PrefixForConsensusNode("zrnvalcons", "zrnvalconspub")
}

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context, *mockBankKeeper) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		t.Fatalf("failed to load latest version: %v", err)
	}

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	bk := newMockBankKeeper()

	authority := sdk.AccAddress([]byte("authority-addr------")).String()
	k := keeper.NewKeeper(runtime.NewKVStoreService(storeKey), cdc, authority, bk)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 100}, false, log.NewNopLogger())

	return k, ctx, bk
}

// setupKeeperAtHeight creates a keeper with a context at the specified block height.
func setupKeeperAtHeight(t *testing.T, height int64) (keeper.Keeper, sdk.Context, *mockBankKeeper) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		t.Fatalf("failed to load latest version: %v", err)
	}

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	bk := newMockBankKeeper()

	authority := sdk.AccAddress([]byte("authority-addr------")).String()
	k := keeper.NewKeeper(runtime.NewKVStoreService(storeKey), cdc, authority, bk)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: height}, false, log.NewNopLogger())

	return k, ctx, bk
}

func testAddr(i int) string {
	addr := sdk.AccAddress([]byte(fmt.Sprintf("test-addr-%010d", i)))
	return addr.String()
}

// -----------------------------------------------------------------------
// 1. TestSetGetParams - params persistence
// -----------------------------------------------------------------------

func TestSetGetParams(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Default params should be returned when none are set explicitly.
	params := k.GetParams(ctx)
	if params.MinProviderStake != "10000000" {
		t.Errorf("expected MinProviderStake 10000000, got %s", params.MinProviderStake)
	}
	if params.HeartbeatIntervalBlocks != 100 {
		t.Errorf("expected HeartbeatIntervalBlocks 100, got %d", params.HeartbeatIntervalBlocks)
	}
	if params.MaxPricePerCu != "1000000" {
		t.Errorf("expected MaxPricePerCu 1000000, got %s", params.MaxPricePerCu)
	}
	if params.ProviderUnbondingBlocks != 10000 {
		t.Errorf("expected ProviderUnbondingBlocks 10000, got %d", params.ProviderUnbondingBlocks)
	}
	if params.PriceChangeDelayBlocks != 500 {
		t.Errorf("expected PriceChangeDelayBlocks 500, got %d", params.PriceChangeDelayBlocks)
	}
	if params.ComputePoolShareBps != 100000 {
		t.Errorf("expected ComputePoolShareBps 100000, got %d", params.ComputePoolShareBps)
	}

	// Set custom params and verify round-trip.
	custom := types.DefaultParams()
	custom.MinProviderStake = "50000000"
	custom.HeartbeatIntervalBlocks = 200
	custom.MaxPricePerCu = "5000000"
	custom.ProviderUnbondingBlocks = 20000
	custom.PriceChangeDelayBlocks = 1000
	k.SetParams(ctx, custom)

	got := k.GetParams(ctx)
	if got.MinProviderStake != "50000000" {
		t.Errorf("expected MinProviderStake 50000000, got %s", got.MinProviderStake)
	}
	if got.HeartbeatIntervalBlocks != 200 {
		t.Errorf("expected HeartbeatIntervalBlocks 200, got %d", got.HeartbeatIntervalBlocks)
	}
	if got.MaxPricePerCu != "5000000" {
		t.Errorf("expected MaxPricePerCu 5000000, got %s", got.MaxPricePerCu)
	}
	if got.ProviderUnbondingBlocks != 20000 {
		t.Errorf("expected ProviderUnbondingBlocks 20000, got %d", got.ProviderUnbondingBlocks)
	}
	if got.PriceChangeDelayBlocks != 1000 {
		t.Errorf("expected PriceChangeDelayBlocks 1000, got %d", got.PriceChangeDelayBlocks)
	}
}

// -----------------------------------------------------------------------
// 2. TestProviderCRUD - set/get/delete/iterate providers
// -----------------------------------------------------------------------

func TestProviderCRUD(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	addr := testAddr(1)

	// Get should return not found for non-existent provider.
	_, found := k.GetProvider(ctx, addr)
	if found {
		t.Error("expected provider not found")
	}

	// Set a provider and verify retrieval.
	provider := &types.ComputeProvider{
		Address:       addr,
		ServiceType:   "inference",
		Endpoint:      "https://compute.example.com",
		PricePerCu:    "500000",
		Stake:         "100000000",
		Status:        "active",
		RegisteredAt:  100,
		TasksServed:   0,
		TasksFailed:   0,
		AvgLatencyMs:  0,
		UptimeBps:     1000000,
		LastHeartbeat: 100,
	}
	k.SetProvider(ctx, provider)

	got, found := k.GetProvider(ctx, addr)
	if !found {
		t.Fatal("expected provider to be found")
	}
	if got.Stake != "100000000" {
		t.Errorf("expected stake 100000000, got %s", got.Stake)
	}
	if got.ServiceType != "inference" {
		t.Errorf("expected service type inference, got %s", got.ServiceType)
	}
	if got.Status != "active" {
		t.Errorf("expected status active, got %s", got.Status)
	}
	if got.Endpoint != "https://compute.example.com" {
		t.Errorf("expected endpoint https://compute.example.com, got %s", got.Endpoint)
	}
	if got.PricePerCu != "500000" {
		t.Errorf("expected price_per_cu 500000, got %s", got.PricePerCu)
	}

	// Delete the provider.
	k.DeleteProvider(ctx, addr)
	_, found = k.GetProvider(ctx, addr)
	if found {
		t.Error("expected provider to be deleted")
	}
}

func TestGetAllProviders(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	for i := 0; i < 5; i++ {
		k.SetProvider(ctx, &types.ComputeProvider{
			Address:     testAddr(i),
			ServiceType: "inference",
			PricePerCu:  "500000",
			Stake:       "100000000",
			Status:      "active",
		})
	}
	all := k.GetAllProviders(ctx)
	if len(all) != 5 {
		t.Errorf("expected 5 providers, got %d", len(all))
	}
}

func TestIterateProviders_EarlyStop(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	for i := 0; i < 5; i++ {
		k.SetProvider(ctx, &types.ComputeProvider{
			Address:     testAddr(i),
			ServiceType: "inference",
			PricePerCu:  "500000",
			Stake:       "100000000",
			Status:      "active",
		})
	}

	count := 0
	k.IterateProviders(ctx, func(p *types.ComputeProvider) bool {
		count++
		return count >= 2 // stop after 2
	})
	if count != 2 {
		t.Errorf("expected iteration to stop at 2, got %d", count)
	}
}

// -----------------------------------------------------------------------
// 3. TestCreditCRUD - set/get/iterate credits
// -----------------------------------------------------------------------

func TestCreditCRUD(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	addr := testAddr(1)

	// Get should return not found for non-existent credit.
	_, found := k.GetCredit(ctx, addr)
	if found {
		t.Error("expected credit not found")
	}

	// Set a credit and verify retrieval.
	credit := &types.ComputeCredit{
		ValidatorAddr: addr,
		Balance:       5000,
		EarnedTotal:   5000,
		RedeemedTotal: 0,
	}
	k.SetCredit(ctx, credit)

	got, found := k.GetCredit(ctx, addr)
	if !found {
		t.Fatal("expected credit to be found")
	}
	if got.Balance != 5000 {
		t.Errorf("expected balance 5000, got %d", got.Balance)
	}
	if got.EarnedTotal != 5000 {
		t.Errorf("expected earned_total 5000, got %d", got.EarnedTotal)
	}
	if got.RedeemedTotal != 0 {
		t.Errorf("expected redeemed_total 0, got %d", got.RedeemedTotal)
	}

	// Update the credit.
	got.Balance = 3000
	got.RedeemedTotal = 2000
	k.SetCredit(ctx, got)

	updated, _ := k.GetCredit(ctx, addr)
	if updated.Balance != 3000 {
		t.Errorf("expected updated balance 3000, got %d", updated.Balance)
	}
	if updated.RedeemedTotal != 2000 {
		t.Errorf("expected updated redeemed_total 2000, got %d", updated.RedeemedTotal)
	}
}

func TestGetAllCredits(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	for i := 0; i < 3; i++ {
		k.SetCredit(ctx, &types.ComputeCredit{
			ValidatorAddr: testAddr(i),
			Balance:       uint64(1000 * (i + 1)),
			EarnedTotal:   uint64(1000 * (i + 1)),
		})
	}
	all := k.GetAllCredits(ctx)
	if len(all) != 3 {
		t.Errorf("expected 3 credits, got %d", len(all))
	}
}

// -----------------------------------------------------------------------
// 4. TestRegisterProvider - register with stake, service type validation
// -----------------------------------------------------------------------

func TestRegisterProvider(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	addr := testAddr(1)
	accAddr, _ := sdk.AccAddressFromBech32(addr)
	bk.setBalance(accAddr, "uzrn", sdkmath.NewInt(200000000))

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.RegisterProvider(ctx, &types.MsgRegisterProvider{
		Sender:      addr,
		ServiceType: "inference",
		Endpoint:    "https://compute.example.com",
		PricePerCu:  "500000",
		Stake:       "100000000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	provider, found := k.GetProvider(ctx, addr)
	if !found {
		t.Fatal("expected provider to be found")
	}
	if provider.Status != "active" {
		t.Errorf("expected status active, got %s", provider.Status)
	}
	if provider.Stake != "100000000" {
		t.Errorf("expected stake 100000000, got %s", provider.Stake)
	}
	if provider.ServiceType != "inference" {
		t.Errorf("expected service type inference, got %s", provider.ServiceType)
	}
	if provider.Endpoint != "https://compute.example.com" {
		t.Errorf("expected endpoint https://compute.example.com, got %s", provider.Endpoint)
	}
	if provider.PricePerCu != "500000" {
		t.Errorf("expected price_per_cu 500000, got %s", provider.PricePerCu)
	}
	if provider.RegisteredAt != 100 {
		t.Errorf("expected registered_at 100, got %d", provider.RegisteredAt)
	}
	if provider.UptimeBps != 1000000 {
		t.Errorf("expected uptime_bps 1000000, got %d", provider.UptimeBps)
	}
	if provider.LastHeartbeat != 100 {
		t.Errorf("expected last_heartbeat 100, got %d", provider.LastHeartbeat)
	}

	// Verify stake was deducted from account balance.
	remaining := bk.balances[accAddr.String()+"/uzrn"]
	expected := sdkmath.NewInt(100000000) // 200M - 100M
	if !remaining.Equal(expected) {
		t.Errorf("expected remaining balance %s, got %s", expected, remaining)
	}

	// Verify stake moved to module account.
	moduleBal := bk.balances[types.ModuleName+"/uzrn"]
	expectedMod := sdkmath.NewInt(100000000)
	if !moduleBal.Equal(expectedMod) {
		t.Errorf("expected module balance %s, got %s", expectedMod, moduleBal)
	}
}

func TestRegisterProvider_InvalidServiceType(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	addr := testAddr(1)
	accAddr, _ := sdk.AccAddressFromBech32(addr)
	bk.setBalance(accAddr, "uzrn", sdkmath.NewInt(200000000))

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.RegisterProvider(ctx, &types.MsgRegisterProvider{
		Sender:      addr,
		ServiceType: "invalid_type",
		Endpoint:    "https://compute.example.com",
		PricePerCu:  "500000",
		Stake:       "100000000",
	})
	if err == nil {
		t.Error("expected invalid service type error")
	}
}

func TestRegisterProvider_ValidServiceTypes(t *testing.T) {
	validTypes := []string{"inference", "verification", "storage"}

	for i, svcType := range validTypes {
		k, ctx, bk := setupKeeper(t)
		addr := testAddr(i + 10)
		accAddr, _ := sdk.AccAddressFromBech32(addr)
		bk.setBalance(accAddr, "uzrn", sdkmath.NewInt(200000000))

		srv := keeper.NewMsgServerImpl(k)
		_, err := srv.RegisterProvider(ctx, &types.MsgRegisterProvider{
			Sender:      addr,
			ServiceType: svcType,
			Endpoint:    "https://compute.example.com",
			PricePerCu:  "500000",
			Stake:       "100000000",
		})
		if err != nil {
			t.Errorf("expected service type %q to be valid, got error: %v", svcType, err)
		}
	}
}

func TestRegisterProvider_Duplicate(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	addr := testAddr(1)
	accAddr, _ := sdk.AccAddressFromBech32(addr)
	bk.setBalance(accAddr, "uzrn", sdkmath.NewInt(500000000))

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.RegisterProvider(ctx, &types.MsgRegisterProvider{
		Sender:      addr,
		ServiceType: "inference",
		Endpoint:    "https://compute.example.com",
		PricePerCu:  "500000",
		Stake:       "100000000",
	})
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	_, err = srv.RegisterProvider(ctx, &types.MsgRegisterProvider{
		Sender:      addr,
		ServiceType: "verification",
		Endpoint:    "https://compute2.example.com",
		PricePerCu:  "500000",
		Stake:       "100000000",
	})
	if err == nil {
		t.Error("expected duplicate provider error")
	}
}

func TestRegisterProvider_PriceExceedsCeiling(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	addr := testAddr(1)
	accAddr, _ := sdk.AccAddressFromBech32(addr)
	bk.setBalance(accAddr, "uzrn", sdkmath.NewInt(200000000))

	srv := keeper.NewMsgServerImpl(k)
	// MaxPricePerCu default is 1000000; setting 2000000 exceeds it.
	_, err := srv.RegisterProvider(ctx, &types.MsgRegisterProvider{
		Sender:      addr,
		ServiceType: "inference",
		Endpoint:    "https://compute.example.com",
		PricePerCu:  "2000000",
		Stake:       "100000000",
	})
	if err == nil {
		t.Error("expected price exceeds ceiling error")
	}
}

// -----------------------------------------------------------------------
// 5. TestRegisterProviderInsufficientStake - below minimum
// -----------------------------------------------------------------------

func TestRegisterProviderInsufficientStake(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	addr := testAddr(1)
	accAddr, _ := sdk.AccAddressFromBech32(addr)
	bk.setBalance(accAddr, "uzrn", sdkmath.NewInt(200000000))

	srv := keeper.NewMsgServerImpl(k)
	// MinProviderStake default is 10000000; 1000 is below.
	_, err := srv.RegisterProvider(ctx, &types.MsgRegisterProvider{
		Sender:      addr,
		ServiceType: "inference",
		Endpoint:    "https://compute.example.com",
		PricePerCu:  "500000",
		Stake:       "1000",
	})
	if err == nil {
		t.Error("expected insufficient stake error")
	}
}

func TestRegisterProviderInsufficientBalance(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	addr := testAddr(1)
	accAddr, _ := sdk.AccAddressFromBech32(addr)
	bk.setBalance(accAddr, "uzrn", sdkmath.NewInt(1000000)) // only 1 ZRN, need 10 ZRN

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.RegisterProvider(ctx, &types.MsgRegisterProvider{
		Sender:      addr,
		ServiceType: "inference",
		Endpoint:    "https://compute.example.com",
		PricePerCu:  "500000",
		Stake:       "10000000",
	})
	if err == nil {
		t.Error("expected insufficient balance error")
	}
}

// -----------------------------------------------------------------------
// 6. TestHeartbeat - update last_heartbeat
// -----------------------------------------------------------------------

func TestHeartbeat(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	addr := testAddr(1)
	accAddr, _ := sdk.AccAddressFromBech32(addr)
	bk.setBalance(accAddr, "uzrn", sdkmath.NewInt(200000000))

	srv := keeper.NewMsgServerImpl(k)

	// Register provider at block 100.
	_, err := srv.RegisterProvider(ctx, &types.MsgRegisterProvider{
		Sender:      addr,
		ServiceType: "inference",
		Endpoint:    "https://compute.example.com",
		PricePerCu:  "500000",
		Stake:       "100000000",
	})
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	provider, _ := k.GetProvider(ctx, addr)
	if provider.LastHeartbeat != 100 {
		t.Errorf("expected last_heartbeat 100 after registration, got %d", provider.LastHeartbeat)
	}

	// Advance to block 150 and send heartbeat.
	ctx = ctx.WithBlockHeight(150)
	_, err = srv.Heartbeat(ctx, &types.MsgHeartbeat{Sender: addr})
	if err != nil {
		t.Fatalf("heartbeat failed: %v", err)
	}

	provider, _ = k.GetProvider(ctx, addr)
	if provider.LastHeartbeat != 150 {
		t.Errorf("expected last_heartbeat 150 after heartbeat, got %d", provider.LastHeartbeat)
	}
	if provider.Status != "active" {
		t.Errorf("expected status active after heartbeat, got %s", provider.Status)
	}
}

func TestHeartbeat_UnknownProvider(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.Heartbeat(ctx, &types.MsgHeartbeat{Sender: testAddr(99)})
	if err == nil {
		t.Error("expected provider not found error")
	}
}

func TestHeartbeat_ReactivatesJailedProvider(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	addr := testAddr(1)

	// Directly insert a jailed provider.
	k.SetProvider(ctx, &types.ComputeProvider{
		Address:       addr,
		ServiceType:   "inference",
		PricePerCu:    "500000",
		Stake:         "100000000",
		Status:        "jailed",
		LastHeartbeat: 50,
	})

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.Heartbeat(ctx, &types.MsgHeartbeat{Sender: addr})
	if err != nil {
		t.Fatalf("heartbeat failed: %v", err)
	}

	provider, _ := k.GetProvider(ctx, addr)
	if provider.Status != "active" {
		t.Errorf("expected jailed provider to be reactivated, got status %s", provider.Status)
	}
	if provider.LastHeartbeat != 100 {
		t.Errorf("expected last_heartbeat 100, got %d", provider.LastHeartbeat)
	}
}

func TestHeartbeat_UnbondingProviderRejected(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	addr := testAddr(1)

	// Directly insert an unbonding provider.
	k.SetProvider(ctx, &types.ComputeProvider{
		Address:       addr,
		ServiceType:   "inference",
		PricePerCu:    "500000",
		Stake:         "100000000",
		Status:        "unbonding",
		UnbondingAt:   90,
		LastHeartbeat: 50,
	})

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.Heartbeat(ctx, &types.MsgHeartbeat{Sender: addr})
	if err == nil {
		t.Error("expected error for heartbeat on unbonding provider")
	}
}

// -----------------------------------------------------------------------
// 7. TestJailOnMissedHeartbeat - BeginBlocker jails inactive providers
// -----------------------------------------------------------------------

func TestJailOnMissedHeartbeat(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	addr := testAddr(1)

	// Default HeartbeatIntervalBlocks = 100.
	// Provider last heartbeat at block 100. At block 201, it should be jailed
	// because 100 + 100 < 201.
	k.SetProvider(ctx, &types.ComputeProvider{
		Address:       addr,
		ServiceType:   "inference",
		PricePerCu:    "500000",
		Stake:         "100000000",
		Status:        "active",
		LastHeartbeat: 100,
	})

	// Advance to block 201 (past heartbeat window).
	ctx = ctx.WithBlockHeight(201)
	err := k.BeginBlocker(ctx)
	if err != nil {
		t.Fatalf("BeginBlocker failed: %v", err)
	}

	provider, found := k.GetProvider(ctx, addr)
	if !found {
		t.Fatal("expected provider to still exist")
	}
	if provider.Status != "jailed" {
		t.Errorf("expected status jailed, got %s", provider.Status)
	}
}

func TestJailOnMissedHeartbeat_ActiveProviderNotJailedWithinWindow(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	addr := testAddr(1)

	// HeartbeatIntervalBlocks = 100. Provider at block 100.
	// At block 199, 100 + 100 = 200 >= 199, so NOT jailed.
	k.SetProvider(ctx, &types.ComputeProvider{
		Address:       addr,
		ServiceType:   "inference",
		PricePerCu:    "500000",
		Stake:         "100000000",
		Status:        "active",
		LastHeartbeat: 100,
	})

	ctx = ctx.WithBlockHeight(199)
	err := k.BeginBlocker(ctx)
	if err != nil {
		t.Fatalf("BeginBlocker failed: %v", err)
	}

	provider, _ := k.GetProvider(ctx, addr)
	if provider.Status != "active" {
		t.Errorf("expected status active (within heartbeat window), got %s", provider.Status)
	}
}

// -----------------------------------------------------------------------
// 8. TestUpdatePrice - pending price with delay
// -----------------------------------------------------------------------

func TestUpdatePrice(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	addr := testAddr(1)
	accAddr, _ := sdk.AccAddressFromBech32(addr)
	bk.setBalance(accAddr, "uzrn", sdkmath.NewInt(200000000))

	srv := keeper.NewMsgServerImpl(k)

	// Register provider.
	_, err := srv.RegisterProvider(ctx, &types.MsgRegisterProvider{
		Sender:      addr,
		ServiceType: "inference",
		Endpoint:    "https://compute.example.com",
		PricePerCu:  "500000",
		Stake:       "100000000",
	})
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	// Update price.
	_, err = srv.UpdatePrice(ctx, &types.MsgUpdatePrice{
		Sender:   addr,
		NewPrice: "600000",
	})
	if err != nil {
		t.Fatalf("update price failed: %v", err)
	}

	provider, _ := k.GetProvider(ctx, addr)

	// Price should not change immediately.
	if provider.PricePerCu != "500000" {
		t.Errorf("expected price to remain 500000 until delay, got %s", provider.PricePerCu)
	}

	// Pending price should be set.
	if provider.PendingPrice != "600000" {
		t.Errorf("expected pending_price 600000, got %s", provider.PendingPrice)
	}

	// PriceChangeAt = current block (100) + PriceChangeDelayBlocks (500) = 600.
	if provider.PriceChangeAt != 600 {
		t.Errorf("expected price_change_at 600, got %d", provider.PriceChangeAt)
	}
}

func TestUpdatePrice_UnknownProvider(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.UpdatePrice(ctx, &types.MsgUpdatePrice{
		Sender:   testAddr(99),
		NewPrice: "600000",
	})
	if err == nil {
		t.Error("expected provider not found error")
	}
}

func TestUpdatePrice_ExceedsCeiling(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	addr := testAddr(1)
	k.SetProvider(ctx, &types.ComputeProvider{
		Address:       addr,
		ServiceType:   "inference",
		PricePerCu:    "500000",
		Stake:         "100000000",
		Status:        "active",
		LastHeartbeat: 100,
	})

	srv := keeper.NewMsgServerImpl(k)
	// MaxPricePerCu default = 1000000; 2000000 exceeds.
	_, err := srv.UpdatePrice(ctx, &types.MsgUpdatePrice{
		Sender:   addr,
		NewPrice: "2000000",
	})
	if err == nil {
		t.Error("expected price exceeds ceiling error")
	}
}

func TestUpdatePrice_InactiveProvider(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	addr := testAddr(1)
	k.SetProvider(ctx, &types.ComputeProvider{
		Address:       addr,
		ServiceType:   "inference",
		PricePerCu:    "500000",
		Stake:         "100000000",
		Status:        "jailed",
		LastHeartbeat: 50,
	})

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.UpdatePrice(ctx, &types.MsgUpdatePrice{
		Sender:   addr,
		NewPrice: "600000",
	})
	if err == nil {
		t.Error("expected provider not active error")
	}
}

// -----------------------------------------------------------------------
// 9. TestApplyPendingPrice - BeginBlocker applies price after delay
// -----------------------------------------------------------------------

func TestApplyPendingPrice(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	addr := testAddr(1)

	// HeartbeatIntervalBlocks default = 100.
	// Provider with pending price effective at block 150 and recent heartbeat.
	// LastHeartbeat=100 keeps provider alive through block 200 (100+100).
	k.SetProvider(ctx, &types.ComputeProvider{
		Address:       addr,
		ServiceType:   "inference",
		PricePerCu:    "500000",
		Stake:         "100000000",
		Status:        "active",
		LastHeartbeat: 100,
		PendingPrice:  "700000",
		PriceChangeAt: 150,
	})

	// At block 149, price should NOT yet be applied (PriceChangeAt=150 > 149).
	ctx = ctx.WithBlockHeight(149)
	err := k.BeginBlocker(ctx)
	if err != nil {
		t.Fatalf("BeginBlocker failed: %v", err)
	}

	provider, _ := k.GetProvider(ctx, addr)
	if provider.PricePerCu != "500000" {
		t.Errorf("expected price still 500000 at block 149, got %s", provider.PricePerCu)
	}
	if provider.PendingPrice != "700000" {
		t.Errorf("expected pending_price still set at block 149, got %s", provider.PendingPrice)
	}

	// At block 150, price should be applied (PriceChangeAt=150 <= 150).
	// Provider is still alive: LastHeartbeat(100) + HeartbeatInterval(100) = 200 >= 150.
	ctx = ctx.WithBlockHeight(150)
	err = k.BeginBlocker(ctx)
	if err != nil {
		t.Fatalf("BeginBlocker failed: %v", err)
	}

	provider, _ = k.GetProvider(ctx, addr)
	if provider.PricePerCu != "700000" {
		t.Errorf("expected price 700000 at block 150, got %s", provider.PricePerCu)
	}
	if provider.PendingPrice != "" {
		t.Errorf("expected pending_price cleared, got %s", provider.PendingPrice)
	}
	if provider.PriceChangeAt != 0 {
		t.Errorf("expected price_change_at 0, got %d", provider.PriceChangeAt)
	}
}

func TestApplyPendingPrice_JailedBeforeApply(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	addr := testAddr(1)

	// Provider has a pending price at block 600, but last heartbeat is old
	// so it will be jailed before price is applied.
	k.SetProvider(ctx, &types.ComputeProvider{
		Address:       addr,
		ServiceType:   "inference",
		PricePerCu:    "500000",
		Stake:         "100000000",
		Status:        "active",
		LastHeartbeat: 100, // old heartbeat
		PendingPrice:  "700000",
		PriceChangeAt: 600,
	})

	// At block 201, heartbeat window (100+100=200 < 201) is exceeded, so provider gets jailed.
	ctx = ctx.WithBlockHeight(201)
	err := k.BeginBlocker(ctx)
	if err != nil {
		t.Fatalf("BeginBlocker failed: %v", err)
	}

	provider, _ := k.GetProvider(ctx, addr)
	if provider.Status != "jailed" {
		t.Errorf("expected provider to be jailed, got %s", provider.Status)
	}
	// Price should NOT have been applied because provider was jailed.
	if provider.PricePerCu != "500000" {
		t.Errorf("expected price unchanged (jailed before apply), got %s", provider.PricePerCu)
	}
}

// -----------------------------------------------------------------------
// 10. TestUnregisterAndUnbonding - initiate unbonding, complete after delay
// -----------------------------------------------------------------------

func TestUnregisterAndUnbonding(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	addr := testAddr(1)
	accAddr, _ := sdk.AccAddressFromBech32(addr)
	bk.setBalance(accAddr, "uzrn", sdkmath.NewInt(200000000))

	srv := keeper.NewMsgServerImpl(k)

	// Register provider.
	_, err := srv.RegisterProvider(ctx, &types.MsgRegisterProvider{
		Sender:      addr,
		ServiceType: "inference",
		Endpoint:    "https://compute.example.com",
		PricePerCu:  "500000",
		Stake:       "100000000",
	})
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	// Unregister provider to start unbonding.
	_, err = srv.UnregisterProvider(ctx, &types.MsgUnregisterProvider{Sender: addr})
	if err != nil {
		t.Fatalf("unregister failed: %v", err)
	}

	provider, found := k.GetProvider(ctx, addr)
	if !found {
		t.Fatal("expected provider to still exist during unbonding")
	}
	if provider.Status != "unbonding" {
		t.Errorf("expected status unbonding, got %s", provider.Status)
	}
	if provider.UnbondingAt != 100 {
		t.Errorf("expected unbonding_at 100, got %d", provider.UnbondingAt)
	}

	// ProviderUnbondingBlocks default = 10000.
	// At block 10099, unbonding NOT completed (100 + 10000 = 10100 > 10099).
	ctx = ctx.WithBlockHeight(10099)
	err = k.BeginBlocker(ctx)
	if err != nil {
		t.Fatalf("BeginBlocker failed: %v", err)
	}

	_, found = k.GetProvider(ctx, addr)
	if !found {
		t.Error("expected provider to still exist before unbonding completes")
	}

	// At block 10100, unbonding completes (100 + 10000 = 10100 <= 10100).
	ctx = ctx.WithBlockHeight(10100)
	err = k.BeginBlocker(ctx)
	if err != nil {
		t.Fatalf("BeginBlocker failed: %v", err)
	}

	_, found = k.GetProvider(ctx, addr)
	if found {
		t.Error("expected provider to be removed after unbonding completes")
	}

	// Verify stake refunded to provider.
	finalBal := bk.balances[accAddr.String()+"/uzrn"]
	expectedBal := sdkmath.NewInt(200000000) // 100M remaining + 100M refunded
	if !finalBal.Equal(expectedBal) {
		t.Errorf("expected refunded balance %s, got %s", expectedBal, finalBal)
	}
}

func TestUnregisterProvider_NotFound(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.UnregisterProvider(ctx, &types.MsgUnregisterProvider{Sender: testAddr(99)})
	if err == nil {
		t.Error("expected provider not found error")
	}
}

func TestUnregisterProvider_AlreadyUnbonding(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	addr := testAddr(1)
	k.SetProvider(ctx, &types.ComputeProvider{
		Address:     addr,
		ServiceType: "inference",
		PricePerCu:  "500000",
		Stake:       "100000000",
		Status:      "unbonding",
		UnbondingAt: 90,
	})

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.UnregisterProvider(ctx, &types.MsgUnregisterProvider{Sender: addr})
	if err == nil {
		t.Error("expected provider already unbonding error")
	}
}

// -----------------------------------------------------------------------
// 11. TestRedeemCredits - credit deduction and uzrn payout
// -----------------------------------------------------------------------

func TestRedeemCredits(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	addr := testAddr(1)
	accAddr, _ := sdk.AccAddressFromBech32(addr)

	// Set up credit for the address.
	k.SetCredit(ctx, &types.ComputeCredit{
		ValidatorAddr: addr,
		Balance:       5000,
		EarnedTotal:   5000,
		RedeemedTotal: 0,
	})

	// Fund the module account so it can pay out.
	bk.balances[types.ModuleName+"/uzrn"] = sdkmath.NewInt(10000)

	srv := keeper.NewMsgServerImpl(k)
	resp, err := srv.RedeemCredits(ctx, &types.MsgRedeemCredits{
		Sender: addr,
		Amount: 3000,
	})
	if err != nil {
		t.Fatalf("redeem credits failed: %v", err)
	}

	if resp.RedeemedUzrn != "3000" {
		t.Errorf("expected redeemed_uzrn 3000, got %s", resp.RedeemedUzrn)
	}

	// Verify credit balance was deducted.
	credit, _ := k.GetCredit(ctx, addr)
	if credit.Balance != 2000 {
		t.Errorf("expected credit balance 2000, got %d", credit.Balance)
	}
	if credit.RedeemedTotal != 3000 {
		t.Errorf("expected redeemed_total 3000, got %d", credit.RedeemedTotal)
	}

	// Verify uzrn was sent to the account.
	accBal := bk.balances[accAddr.String()+"/uzrn"]
	if !accBal.Equal(sdkmath.NewInt(3000)) {
		t.Errorf("expected account balance 3000 uzrn, got %s", accBal)
	}

	// Verify module balance was deducted.
	moduleBal := bk.balances[types.ModuleName+"/uzrn"]
	if !moduleBal.Equal(sdkmath.NewInt(7000)) {
		t.Errorf("expected module balance 7000, got %s", moduleBal)
	}
}

func TestRedeemCredits_FullBalance(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	addr := testAddr(1)

	k.SetCredit(ctx, &types.ComputeCredit{
		ValidatorAddr: addr,
		Balance:       5000,
		EarnedTotal:   5000,
		RedeemedTotal: 0,
	})

	bk.balances[types.ModuleName+"/uzrn"] = sdkmath.NewInt(10000)

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.RedeemCredits(ctx, &types.MsgRedeemCredits{
		Sender: addr,
		Amount: 5000,
	})
	if err != nil {
		t.Fatalf("redeem credits failed: %v", err)
	}

	credit, _ := k.GetCredit(ctx, addr)
	if credit.Balance != 0 {
		t.Errorf("expected credit balance 0 after full redeem, got %d", credit.Balance)
	}
	if credit.RedeemedTotal != 5000 {
		t.Errorf("expected redeemed_total 5000, got %d", credit.RedeemedTotal)
	}
}

// -----------------------------------------------------------------------
// 12. TestRedeemInsufficientCredits - balance check
// -----------------------------------------------------------------------

func TestRedeemInsufficientCredits(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	addr := testAddr(1)

	k.SetCredit(ctx, &types.ComputeCredit{
		ValidatorAddr: addr,
		Balance:       1000,
		EarnedTotal:   1000,
		RedeemedTotal: 0,
	})

	bk.balances[types.ModuleName+"/uzrn"] = sdkmath.NewInt(10000)

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.RedeemCredits(ctx, &types.MsgRedeemCredits{
		Sender: addr,
		Amount: 5000,
	})
	if err == nil {
		t.Error("expected insufficient credits error")
	}

	// Verify credit was not modified.
	credit, _ := k.GetCredit(ctx, addr)
	if credit.Balance != 1000 {
		t.Errorf("expected credit balance unchanged at 1000, got %d", credit.Balance)
	}
}

func TestRedeemCredits_NoCreditsExist(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.RedeemCredits(ctx, &types.MsgRedeemCredits{
		Sender: testAddr(99),
		Amount: 1000,
	})
	if err == nil {
		t.Error("expected insufficient credits error for non-existent credit")
	}
}

func TestRedeemCredits_ZeroAmount(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	addr := testAddr(1)
	k.SetCredit(ctx, &types.ComputeCredit{
		ValidatorAddr: addr,
		Balance:       5000,
		EarnedTotal:   5000,
	})

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.RedeemCredits(ctx, &types.MsgRedeemCredits{
		Sender: addr,
		Amount: 0,
	})
	if err == nil {
		t.Error("expected error for zero amount redeem")
	}
}

func TestRedeemCredits_InsufficientModuleBalance(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	addr := testAddr(1)

	k.SetCredit(ctx, &types.ComputeCredit{
		ValidatorAddr: addr,
		Balance:       5000,
		EarnedTotal:   5000,
		RedeemedTotal: 0,
	})

	// Module has insufficient balance to pay out.
	bk.balances[types.ModuleName+"/uzrn"] = sdkmath.NewInt(100)

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.RedeemCredits(ctx, &types.MsgRedeemCredits{
		Sender: addr,
		Amount: 5000,
	})
	if err == nil {
		t.Error("expected error when module has insufficient balance")
	}
}

// -----------------------------------------------------------------------
// Additional: UpdateParams governance
// -----------------------------------------------------------------------

func TestUpdateParams(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	authority := k.GetAuthority()

	srv := keeper.NewMsgServerImpl(k)
	newParams := types.DefaultParams()
	newParams.MinProviderStake = "50000000"
	newParams.HeartbeatIntervalBlocks = 200

	_, err := srv.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: authority,
		Params:    newParams,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := k.GetParams(ctx)
	if got.MinProviderStake != "50000000" {
		t.Errorf("expected MinProviderStake 50000000, got %s", got.MinProviderStake)
	}
	if got.HeartbeatIntervalBlocks != 200 {
		t.Errorf("expected HeartbeatIntervalBlocks 200, got %d", got.HeartbeatIntervalBlocks)
	}
}

func TestUpdateParamsUnauthorized(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: testAddr(99),
		Params:    types.DefaultParams(),
	})
	if err == nil {
		t.Error("expected unauthorized error")
	}
}

func TestUpdateParamsNilParams(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	authority := k.GetAuthority()

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: authority,
		Params:    nil,
	})
	if err == nil {
		t.Error("expected error for nil params")
	}
}

func TestUpdateParamsInvalidParams(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	authority := k.GetAuthority()

	srv := keeper.NewMsgServerImpl(k)
	invalidParams := types.DefaultParams()
	invalidParams.HeartbeatIntervalBlocks = 0 // invalid

	_, err := srv.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: authority,
		Params:    invalidParams,
	})
	if err == nil {
		t.Error("expected validation error for invalid params")
	}
}

// -----------------------------------------------------------------------
// Additional: Genesis import/export
// -----------------------------------------------------------------------

func TestGenesis(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	addr1 := testAddr(1)
	addr2 := testAddr(2)
	k.SetProvider(ctx, &types.ComputeProvider{
		Address: addr1, ServiceType: "inference", PricePerCu: "500000",
		Stake: "100000000", Status: "active",
	})
	k.SetProvider(ctx, &types.ComputeProvider{
		Address: addr2, ServiceType: "verification", PricePerCu: "600000",
		Stake: "200000000", Status: "active",
	})

	k.SetCredit(ctx, &types.ComputeCredit{
		ValidatorAddr: addr1, Balance: 5000, EarnedTotal: 5000,
	})

	genState := k.ExportGenesis(ctx)
	if len(genState.Providers) != 2 {
		t.Errorf("expected 2 providers in export, got %d", len(genState.Providers))
	}
	if len(genState.Credits) != 1 {
		t.Errorf("expected 1 credit in export, got %d", len(genState.Credits))
	}
	if genState.Params == nil {
		t.Fatal("expected non-nil params in export")
	}

	// Import into a fresh keeper.
	k2, ctx2, _ := setupKeeper(t)
	k2.InitGenesis(ctx2, genState)

	got := k2.ExportGenesis(ctx2)
	if len(got.Providers) != 2 {
		t.Errorf("expected 2 providers after import, got %d", len(got.Providers))
	}
	if len(got.Credits) != 1 {
		t.Errorf("expected 1 credit after import, got %d", len(got.Credits))
	}
}

func TestGenesisValidation(t *testing.T) {
	valid := types.DefaultGenesis()
	if err := valid.Validate(); err != nil {
		t.Errorf("unexpected error on valid genesis: %v", err)
	}

	nilParams := &types.GenesisState{Params: nil}
	if err := nilParams.Validate(); err == nil {
		t.Error("expected error for nil params genesis")
	}

	invalidParams := &types.GenesisState{
		Params: &types.Params{
			HeartbeatIntervalBlocks:  0,
			TargetUtilizationLowBps:  300000,
			TargetUtilizationHighBps: 800000,
		},
	}
	if err := invalidParams.Validate(); err == nil {
		t.Error("expected validation error for invalid params")
	}
}

// -----------------------------------------------------------------------
// Additional: BeginBlocker with multiple providers
// -----------------------------------------------------------------------

func TestBeginBlocker_MixedProviderStates(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	activeAddr := testAddr(1)
	jailableAddr := testAddr(2)
	unbondingAddr := testAddr(3)
	pendingPriceAddr := testAddr(4)

	// Active provider with recent heartbeat (should stay active).
	k.SetProvider(ctx, &types.ComputeProvider{
		Address: activeAddr, ServiceType: "inference", PricePerCu: "500000",
		Stake: "100000000", Status: "active", LastHeartbeat: 100,
	})

	// Provider with old heartbeat (should be jailed).
	k.SetProvider(ctx, &types.ComputeProvider{
		Address: jailableAddr, ServiceType: "inference", PricePerCu: "500000",
		Stake: "100000000", Status: "active", LastHeartbeat: 10,
	})

	// Unbonding provider ready for removal.
	unbondingAccAddr, _ := sdk.AccAddressFromBech32(unbondingAddr)
	bk.balances[types.ModuleName+"/uzrn"] = sdkmath.NewInt(100000000) // fund module for refund
	k.SetProvider(ctx, &types.ComputeProvider{
		Address: unbondingAddr, ServiceType: "storage", PricePerCu: "500000",
		Stake: "100000000", Status: "unbonding", UnbondingAt: 1,
	})

	// Active provider with pending price ready to apply.
	k.SetProvider(ctx, &types.ComputeProvider{
		Address: pendingPriceAddr, ServiceType: "verification", PricePerCu: "500000",
		Stake: "100000000", Status: "active", LastHeartbeat: 100,
		PendingPrice: "800000", PriceChangeAt: 150,
	})

	// Run BeginBlocker at block 150.
	ctx = ctx.WithBlockHeight(150)
	err := k.BeginBlocker(ctx)
	if err != nil {
		t.Fatalf("BeginBlocker failed: %v", err)
	}

	// Active provider should remain active.
	p, _ := k.GetProvider(ctx, activeAddr)
	if p.Status != "active" {
		t.Errorf("expected activeAddr to remain active, got %s", p.Status)
	}

	// Jailable provider should be jailed.
	p, _ = k.GetProvider(ctx, jailableAddr)
	if p.Status != "jailed" {
		t.Errorf("expected jailableAddr to be jailed, got %s", p.Status)
	}

	// Unbonding provider should be removed (1 + 10000 = 10001 <= 150? No, 10001 > 150).
	// Actually 1 + 10000 = 10001 > 150, so it should still exist.
	_, found := k.GetProvider(ctx, unbondingAddr)
	if !found {
		t.Error("expected unbondingAddr to still exist (unbonding not complete)")
	}

	// Pending price provider should have price applied (PriceChangeAt=150 <= 150).
	p, _ = k.GetProvider(ctx, pendingPriceAddr)
	if p.PricePerCu != "800000" {
		t.Errorf("expected pendingPriceAddr price updated to 800000, got %s", p.PricePerCu)
	}
	if p.PendingPrice != "" {
		t.Errorf("expected pending_price cleared, got %s", p.PendingPrice)
	}

	// Now advance far enough for unbonding to complete.
	ctx = ctx.WithBlockHeight(10002)
	err = k.BeginBlocker(ctx)
	if err != nil {
		t.Fatalf("BeginBlocker failed: %v", err)
	}

	_, found = k.GetProvider(ctx, unbondingAddr)
	if found {
		t.Error("expected unbondingAddr to be removed after unbonding completes")
	}

	// Verify refund.
	refundedBal := bk.balances[unbondingAccAddr.String()+"/uzrn"]
	if !refundedBal.Equal(sdkmath.NewInt(100000000)) {
		t.Errorf("expected refunded 100000000, got %s", refundedBal)
	}
}
