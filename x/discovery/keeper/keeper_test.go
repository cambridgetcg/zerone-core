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

	"github.com/zerone-chain/zerone/x/discovery/keeper"
	"github.com/zerone-chain/zerone/x/discovery/types"
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

func testAddr(i int) string {
	addr := sdk.AccAddress([]byte(fmt.Sprintf("test-addr-%010d", i)))
	return addr.String()
}

// -----------------------------------------------------------------------
// 1. TestSetGetParams
// -----------------------------------------------------------------------

func TestSetGetParams(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Default params should be returned when nothing is set.
	params := k.GetParams(ctx)
	if params.MinRegistrationStake != "1000000" {
		t.Errorf("expected MinRegistrationStake 1000000, got %s", params.MinRegistrationStake)
	}
	if params.MaxCapabilitiesPerAgent != 20 {
		t.Errorf("expected MaxCapabilitiesPerAgent 20, got %d", params.MaxCapabilitiesPerAgent)
	}
	if params.ProfileExpiryBlocks != 100000 {
		t.Errorf("expected ProfileExpiryBlocks 100000, got %d", params.ProfileExpiryBlocks)
	}

	// Set custom params, then read them back.
	custom := types.DefaultParams()
	custom.MinRegistrationStake = "5000000"
	custom.MaxCapabilitiesPerAgent = 10
	custom.ProfileExpiryBlocks = 50000
	k.SetParams(ctx, custom)

	got := k.GetParams(ctx)
	if got.MinRegistrationStake != "5000000" {
		t.Errorf("expected MinRegistrationStake 5000000, got %s", got.MinRegistrationStake)
	}
	if got.MaxCapabilitiesPerAgent != 10 {
		t.Errorf("expected MaxCapabilitiesPerAgent 10, got %d", got.MaxCapabilitiesPerAgent)
	}
	if got.ProfileExpiryBlocks != 50000 {
		t.Errorf("expected ProfileExpiryBlocks 50000, got %d", got.ProfileExpiryBlocks)
	}
}

// -----------------------------------------------------------------------
// 2. TestProfileCRUD
// -----------------------------------------------------------------------

func TestProfileCRUD(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	addr := testAddr(1)

	// Get on missing profile should return not-found.
	_, found := k.GetProfile(ctx, addr)
	if found {
		t.Error("expected profile not found")
	}

	// Set a profile.
	profile := &types.AgentProfile{
		Address:     addr,
		DisplayName: "Alice",
		Capabilities: []*types.AgentCapability{
			{CapabilityType: "inference", Domains: []string{"math"}},
		},
		Domains:           []string{"mathematics", "physics"},
		Status:            "active",
		ReputationScore:   500000,
		RegisteredAtBlock: 100,
		LastActiveBlock:   100,
		Stake:             "1000000",
		Description:       "Agent Alice",
		Metadata:          `{"version":"1"}`,
	}
	k.SetProfile(ctx, profile)

	// Get should succeed.
	got, found := k.GetProfile(ctx, addr)
	if !found {
		t.Fatal("expected profile to be found")
	}
	if got.DisplayName != "Alice" {
		t.Errorf("expected display_name Alice, got %s", got.DisplayName)
	}
	if got.Stake != "1000000" {
		t.Errorf("expected stake 1000000, got %s", got.Stake)
	}
	if len(got.Domains) != 2 {
		t.Errorf("expected 2 domains, got %d", len(got.Domains))
	}
	if len(got.Capabilities) != 1 {
		t.Errorf("expected 1 capability, got %d", len(got.Capabilities))
	}

	// Delete profile.
	k.DeleteProfile(ctx, profile)
	_, found = k.GetProfile(ctx, addr)
	if found {
		t.Error("expected profile to be deleted")
	}

	// IterateProfiles / GetAllProfiles with multiple entries.
	for i := 0; i < 5; i++ {
		k.SetProfile(ctx, &types.AgentProfile{
			Address: testAddr(100 + i),
			Domains: []string{"test"},
			Status:  "active",
			Stake:   "1000000",
		})
	}
	all := k.GetAllProfiles(ctx)
	if len(all) != 5 {
		t.Errorf("expected 5 profiles, got %d", len(all))
	}

	// Iterate with early stop.
	count := 0
	k.IterateProfiles(ctx, func(p *types.AgentProfile) bool {
		count++
		return count >= 3
	})
	if count != 3 {
		t.Errorf("expected iterate to stop at 3, stopped at %d", count)
	}
}

// -----------------------------------------------------------------------
// 3. TestRegisterProfile
// -----------------------------------------------------------------------

func TestRegisterProfile(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	addr := testAddr(1)
	accAddr, _ := sdk.AccAddressFromBech32(addr)
	bk.setBalance(accAddr, "uzrn", sdkmath.NewInt(200000000))

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.RegisterProfile(ctx, &types.MsgRegisterProfile{
		Sender:      addr,
		DisplayName: "TestAgent",
		Capabilities: []*types.AgentCapability{
			{CapabilityType: "inference", Domains: []string{"mathematics"}},
			{CapabilityType: "verification", Domains: []string{"physics"}},
		},
		Domains:     []string{"mathematics", "physics"},
		Stake:       "10000000",
		Description: "A test agent",
		Metadata:    `{"version":"1"}`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify profile was stored correctly.
	profile, found := k.GetProfile(ctx, addr)
	if !found {
		t.Fatal("expected profile to be found")
	}
	if profile.Status != "active" {
		t.Errorf("expected status active, got %s", profile.Status)
	}
	if profile.ReputationScore != 500000 {
		t.Errorf("expected reputation 500000, got %d", profile.ReputationScore)
	}
	if profile.Stake != "10000000" {
		t.Errorf("expected stake 10000000, got %s", profile.Stake)
	}
	if profile.DisplayName != "TestAgent" {
		t.Errorf("expected display_name TestAgent, got %s", profile.DisplayName)
	}
	if profile.RegisteredAtBlock != 100 {
		t.Errorf("expected registered_at_block 100, got %d", profile.RegisteredAtBlock)
	}
	if profile.LastActiveBlock != 100 {
		t.Errorf("expected last_active_block 100, got %d", profile.LastActiveBlock)
	}
	if len(profile.Capabilities) != 2 {
		t.Errorf("expected 2 capabilities, got %d", len(profile.Capabilities))
	}

	// Verify domain index works.
	mathProfiles := k.GetProfilesByDomain(ctx, "mathematics")
	if len(mathProfiles) != 1 {
		t.Errorf("expected 1 math profile, got %d", len(mathProfiles))
	}
	physicsProfiles := k.GetProfilesByDomain(ctx, "physics")
	if len(physicsProfiles) != 1 {
		t.Errorf("expected 1 physics profile, got %d", len(physicsProfiles))
	}

	// Verify capability index works.
	inferenceProfiles := k.GetProfilesByCapability(ctx, "inference")
	if len(inferenceProfiles) != 1 {
		t.Errorf("expected 1 inference profile, got %d", len(inferenceProfiles))
	}

	// Verify stake was deducted from the sender.
	remainingBal := bk.balances[accAddr.String()+"/uzrn"]
	expected := sdkmath.NewInt(190000000)
	if !remainingBal.Equal(expected) {
		t.Errorf("expected sender balance %s, got %s", expected, remainingBal)
	}

	// Verify module account received the stake.
	moduleBal := bk.balances[types.ModuleName+"/uzrn"]
	expectedModule := sdkmath.NewInt(10000000)
	if !moduleBal.Equal(expectedModule) {
		t.Errorf("expected module balance %s, got %s", expectedModule, moduleBal)
	}
}

// -----------------------------------------------------------------------
// 4. TestRegisterProfileInsufficientStake
// -----------------------------------------------------------------------

func TestRegisterProfileInsufficientStake(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	addr := testAddr(1)
	accAddr, _ := sdk.AccAddressFromBech32(addr)
	bk.setBalance(accAddr, "uzrn", sdkmath.NewInt(200000000))

	srv := keeper.NewMsgServerImpl(k)

	// Default MinRegistrationStake is "1000000", try with less.
	_, err := srv.RegisterProfile(ctx, &types.MsgRegisterProfile{
		Sender:  addr,
		Domains: []string{"mathematics"},
		Stake:   "500",
	})
	if err == nil {
		t.Error("expected insufficient stake error")
	}

	// Verify the profile was NOT created.
	_, found := k.GetProfile(ctx, addr)
	if found {
		t.Error("expected profile not to be created with insufficient stake")
	}
}

// -----------------------------------------------------------------------
// 5. TestRegisterProfileAlreadyExists
// -----------------------------------------------------------------------

func TestRegisterProfileAlreadyExists(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	addr := testAddr(1)
	accAddr, _ := sdk.AccAddressFromBech32(addr)
	bk.setBalance(accAddr, "uzrn", sdkmath.NewInt(500000000))

	srv := keeper.NewMsgServerImpl(k)

	// First registration succeeds.
	_, err := srv.RegisterProfile(ctx, &types.MsgRegisterProfile{
		Sender:      addr,
		DisplayName: "Agent1",
		Domains:     []string{"mathematics"},
		Stake:       "1000000",
	})
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	// Second registration with same address should fail.
	_, err = srv.RegisterProfile(ctx, &types.MsgRegisterProfile{
		Sender:      addr,
		DisplayName: "Agent1Duplicate",
		Domains:     []string{"physics"},
		Stake:       "1000000",
	})
	if err == nil {
		t.Error("expected duplicate agent error")
	}

	// Original profile should be unchanged.
	profile, found := k.GetProfile(ctx, addr)
	if !found {
		t.Fatal("expected original profile to still exist")
	}
	if profile.DisplayName != "Agent1" {
		t.Errorf("expected original display_name Agent1, got %s", profile.DisplayName)
	}
}

// -----------------------------------------------------------------------
// 6. TestUpdateProfile
// -----------------------------------------------------------------------

func TestUpdateProfile(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	addr := testAddr(1)
	accAddr, _ := sdk.AccAddressFromBech32(addr)
	bk.setBalance(accAddr, "uzrn", sdkmath.NewInt(200000000))

	srv := keeper.NewMsgServerImpl(k)

	// Register first.
	_, err := srv.RegisterProfile(ctx, &types.MsgRegisterProfile{
		Sender:      addr,
		DisplayName: "OriginalName",
		Domains:     []string{"mathematics"},
		Stake:       "1000000",
		Description: "Original description",
		Metadata:    `{"v":"1"}`,
	})
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	// Update display_name only (other fields empty => unchanged).
	_, err = srv.UpdateProfile(ctx, &types.MsgUpdateProfile{
		Sender:      addr,
		DisplayName: "UpdatedName",
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	profile, _ := k.GetProfile(ctx, addr)
	if profile.DisplayName != "UpdatedName" {
		t.Errorf("expected display_name UpdatedName, got %s", profile.DisplayName)
	}
	if profile.Description != "Original description" {
		t.Errorf("expected description unchanged, got %s", profile.Description)
	}
	if profile.Metadata != `{"v":"1"}` {
		t.Errorf("expected metadata unchanged, got %s", profile.Metadata)
	}

	// Update description and metadata.
	_, err = srv.UpdateProfile(ctx, &types.MsgUpdateProfile{
		Sender:      addr,
		Description: "Updated description",
		Metadata:    `{"v":"2"}`,
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	profile, _ = k.GetProfile(ctx, addr)
	if profile.DisplayName != "UpdatedName" {
		t.Errorf("expected display_name UpdatedName (unchanged), got %s", profile.DisplayName)
	}
	if profile.Description != "Updated description" {
		t.Errorf("expected Updated description, got %s", profile.Description)
	}
	if profile.Metadata != `{"v":"2"}` {
		t.Errorf("expected metadata v2, got %s", profile.Metadata)
	}

	// Update for non-existent profile should fail.
	_, err = srv.UpdateProfile(ctx, &types.MsgUpdateProfile{
		Sender:      testAddr(99),
		DisplayName: "Ghost",
	})
	if err == nil {
		t.Error("expected error updating non-existent profile")
	}
}

// -----------------------------------------------------------------------
// 7. TestHeartbeat
// -----------------------------------------------------------------------

func TestHeartbeat(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	addr := testAddr(1)
	accAddr, _ := sdk.AccAddressFromBech32(addr)
	bk.setBalance(accAddr, "uzrn", sdkmath.NewInt(200000000))

	srv := keeper.NewMsgServerImpl(k)

	// Register at block 100.
	_, err := srv.RegisterProfile(ctx, &types.MsgRegisterProfile{
		Sender:  addr,
		Domains: []string{"mathematics"},
		Stake:   "1000000",
	})
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	profile, _ := k.GetProfile(ctx, addr)
	if profile.LastActiveBlock != 100 {
		t.Errorf("expected last_active_block 100, got %d", profile.LastActiveBlock)
	}

	// Advance to block 200 and send heartbeat.
	ctx200 := ctx.WithBlockHeight(200)
	_, err = srv.Heartbeat(ctx200, &types.MsgHeartbeat{
		Sender: addr,
	})
	if err != nil {
		t.Fatalf("heartbeat failed: %v", err)
	}

	profile, _ = k.GetProfile(ctx200, addr)
	if profile.LastActiveBlock != 200 {
		t.Errorf("expected last_active_block 200, got %d", profile.LastActiveBlock)
	}

	// Manually expire the profile, then heartbeat to reactivate.
	profile.Status = "expired"
	k.SetProfile(ctx200, profile)

	ctx300 := ctx.WithBlockHeight(300)
	_, err = srv.Heartbeat(ctx300, &types.MsgHeartbeat{
		Sender: addr,
	})
	if err != nil {
		t.Fatalf("heartbeat reactivation failed: %v", err)
	}

	profile, _ = k.GetProfile(ctx300, addr)
	if profile.Status != "active" {
		t.Errorf("expected status active after heartbeat reactivation, got %s", profile.Status)
	}
	if profile.LastActiveBlock != 300 {
		t.Errorf("expected last_active_block 300, got %d", profile.LastActiveBlock)
	}

	// Heartbeat for non-existent profile should fail.
	_, err = srv.Heartbeat(ctx300, &types.MsgHeartbeat{
		Sender: testAddr(99),
	})
	if err == nil {
		t.Error("expected error for heartbeat on non-existent profile")
	}
}

// -----------------------------------------------------------------------
// 8. TestDeregisterProfile
// -----------------------------------------------------------------------

func TestDeregisterProfile(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	addr := testAddr(1)
	accAddr, _ := sdk.AccAddressFromBech32(addr)
	bk.setBalance(accAddr, "uzrn", sdkmath.NewInt(200000000))

	srv := keeper.NewMsgServerImpl(k)

	// Register.
	_, err := srv.RegisterProfile(ctx, &types.MsgRegisterProfile{
		Sender:  addr,
		Domains: []string{"mathematics", "physics"},
		Capabilities: []*types.AgentCapability{
			{CapabilityType: "inference"},
		},
		Stake: "10000000",
	})
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	// Verify pre-deregister state.
	balBefore := bk.balances[accAddr.String()+"/uzrn"]
	expectedBefore := sdkmath.NewInt(190000000)
	if !balBefore.Equal(expectedBefore) {
		t.Errorf("expected sender balance %s before deregister, got %s", expectedBefore, balBefore)
	}

	// Deregister.
	resp, err := srv.DeregisterProfile(ctx, &types.MsgDeregisterProfile{
		Sender: addr,
	})
	if err != nil {
		t.Fatalf("deregistration failed: %v", err)
	}
	if resp.RefundedAmount != "10000000" {
		t.Errorf("expected refunded amount 10000000, got %s", resp.RefundedAmount)
	}

	// Verify profile is deleted.
	_, found := k.GetProfile(ctx, addr)
	if found {
		t.Error("expected profile to be deleted")
	}

	// Verify domain index was cleaned.
	mathProfiles := k.GetProfilesByDomain(ctx, "mathematics")
	if len(mathProfiles) != 0 {
		t.Errorf("expected 0 math profiles after deregister, got %d", len(mathProfiles))
	}
	physicsProfiles := k.GetProfilesByDomain(ctx, "physics")
	if len(physicsProfiles) != 0 {
		t.Errorf("expected 0 physics profiles after deregister, got %d", len(physicsProfiles))
	}

	// Verify capability index was cleaned.
	inferenceProfiles := k.GetProfilesByCapability(ctx, "inference")
	if len(inferenceProfiles) != 0 {
		t.Errorf("expected 0 inference profiles after deregister, got %d", len(inferenceProfiles))
	}

	// Verify stake was refunded.
	balAfter := bk.balances[accAddr.String()+"/uzrn"]
	expectedAfter := sdkmath.NewInt(200000000)
	if !balAfter.Equal(expectedAfter) {
		t.Errorf("expected sender balance %s after refund, got %s", expectedAfter, balAfter)
	}

	// Module account should be empty.
	moduleBal, ok := bk.balances[types.ModuleName+"/uzrn"]
	if ok && moduleBal.IsPositive() {
		t.Errorf("expected module balance 0 after deregister, got %s", moduleBal)
	}

	// Deregister non-existent profile should fail.
	_, err = srv.DeregisterProfile(ctx, &types.MsgDeregisterProfile{
		Sender: testAddr(99),
	})
	if err == nil {
		t.Error("expected error deregistering non-existent profile")
	}
}

// -----------------------------------------------------------------------
// 9. TestExpireStaleProfiles
// -----------------------------------------------------------------------

func TestExpireStaleProfiles(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Set a short expiry for testing purposes.
	params := types.DefaultParams()
	params.ProfileExpiryBlocks = 500
	k.SetParams(ctx, params)

	// Create 3 profiles at block 100 (current block).
	for i := 0; i < 3; i++ {
		k.SetProfile(ctx, &types.AgentProfile{
			Address:         testAddr(i),
			Domains:         []string{"test"},
			Status:          "active",
			Stake:           "1000000",
			LastActiveBlock: 100,
		})
	}

	// At block 200 (not a multiple of 100 boundary-hit yet), run BeginBlocker.
	// block 200 is a multiple of 100, but 200 < 100 + 500, so no expiry.
	ctx200 := ctx.WithBlockHeight(200)
	err := k.BeginBlocker(ctx200)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All profiles should still be active.
	all := k.GetAllProfiles(ctx200)
	for _, p := range all {
		if p.Status != "active" {
			t.Errorf("expected active at block 200, got %s for %s", p.Status, p.Address)
		}
	}

	// At block 700 (700 % 100 == 0 and 100 + 500 < 700), profiles should expire.
	ctx700 := ctx.WithBlockHeight(700)
	err = k.BeginBlocker(ctx700)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	all = k.GetAllProfiles(ctx700)
	for _, p := range all {
		if p.Status != "expired" {
			t.Errorf("expected expired at block 700, got %s for %s", p.Status, p.Address)
		}
	}

	// A profile with recent activity should NOT be expired.
	k.SetProfile(ctx, &types.AgentProfile{
		Address:         testAddr(10),
		Domains:         []string{"fresh"},
		Status:          "active",
		Stake:           "1000000",
		LastActiveBlock: 650, // 650 + 500 = 1150 > 700
	})

	// Re-run at block 700 to check the fresh profile.
	err = k.BeginBlocker(ctx700)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fresh, found := k.GetProfile(ctx700, testAddr(10))
	if !found {
		t.Fatal("expected fresh profile to exist")
	}
	if fresh.Status != "active" {
		t.Errorf("expected fresh profile to be active, got %s", fresh.Status)
	}

	// BeginBlocker at a non-100 block should be a no-op.
	ctx701 := ctx.WithBlockHeight(701)
	// Manually re-activate a profile to check that no expiry happens.
	p := all[0]
	p.Status = "active"
	k.SetProfile(ctx701, p)

	err = k.BeginBlocker(ctx701)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	check, _ := k.GetProfile(ctx701, p.Address)
	if check.Status != "active" {
		t.Errorf("expected no expiry at non-100 block, but profile was %s", check.Status)
	}
}

// -----------------------------------------------------------------------
// 10. TestSearchByDomain
// -----------------------------------------------------------------------

func TestSearchByDomain(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Create agents in various domains.
	k.SetProfile(ctx, &types.AgentProfile{
		Address: testAddr(1),
		Domains: []string{"mathematics"},
		Status:  "active",
		Stake:   "1000000",
	})
	k.SetProfile(ctx, &types.AgentProfile{
		Address: testAddr(2),
		Domains: []string{"mathematics", "physics"},
		Status:  "active",
		Stake:   "1000000",
	})
	k.SetProfile(ctx, &types.AgentProfile{
		Address: testAddr(3),
		Domains: []string{"physics"},
		Status:  "active",
		Stake:   "1000000",
	})
	// Inactive profile should NOT appear in domain search.
	k.SetProfile(ctx, &types.AgentProfile{
		Address: testAddr(4),
		Domains: []string{"mathematics"},
		Status:  "expired",
		Stake:   "1000000",
	})

	// Search by domain "mathematics".
	results := k.SearchProfiles(ctx, "mathematics", "", 0)
	if len(results) != 2 {
		t.Errorf("expected 2 mathematics profiles, got %d", len(results))
	}

	// Search by domain "physics".
	results = k.SearchProfiles(ctx, "physics", "", 0)
	if len(results) != 2 {
		t.Errorf("expected 2 physics profiles, got %d", len(results))
	}

	// Search by non-existent domain.
	results = k.SearchProfiles(ctx, "chemistry", "", 0)
	if len(results) != 0 {
		t.Errorf("expected 0 chemistry profiles, got %d", len(results))
	}
}

// -----------------------------------------------------------------------
// 11. TestSearchByCapability
// -----------------------------------------------------------------------

func TestSearchByCapability(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Create agents with various capabilities.
	k.SetProfile(ctx, &types.AgentProfile{
		Address: testAddr(1),
		Domains: []string{"math"},
		Capabilities: []*types.AgentCapability{
			{CapabilityType: "inference"},
			{CapabilityType: "reasoning"},
		},
		Status: "active",
		Stake:  "1000000",
	})
	k.SetProfile(ctx, &types.AgentProfile{
		Address: testAddr(2),
		Domains: []string{"physics"},
		Capabilities: []*types.AgentCapability{
			{CapabilityType: "inference"},
		},
		Status: "active",
		Stake:  "1000000",
	})
	k.SetProfile(ctx, &types.AgentProfile{
		Address: testAddr(3),
		Domains: []string{"biology"},
		Capabilities: []*types.AgentCapability{
			{CapabilityType: "verification"},
		},
		Status: "active",
		Stake:  "1000000",
	})
	// Inactive profile should be filtered out.
	k.SetProfile(ctx, &types.AgentProfile{
		Address: testAddr(4),
		Domains: []string{"math"},
		Capabilities: []*types.AgentCapability{
			{CapabilityType: "inference"},
		},
		Status: "expired",
		Stake:  "1000000",
	})

	// Search by capability "inference".
	results := k.SearchProfiles(ctx, "", "inference", 0)
	if len(results) != 2 {
		t.Errorf("expected 2 inference profiles, got %d", len(results))
	}

	// Search by capability "reasoning".
	results = k.SearchProfiles(ctx, "", "reasoning", 0)
	if len(results) != 1 {
		t.Errorf("expected 1 reasoning profile, got %d", len(results))
	}

	// Search by capability "verification".
	results = k.SearchProfiles(ctx, "", "verification", 0)
	if len(results) != 1 {
		t.Errorf("expected 1 verification profile, got %d", len(results))
	}

	// Combined: domain "math" + capability "inference".
	results = k.SearchProfiles(ctx, "math", "inference", 0)
	if len(results) != 1 {
		t.Errorf("expected 1 math+inference profile, got %d", len(results))
	}

	// Combined: domain "physics" + capability "reasoning" (no match).
	results = k.SearchProfiles(ctx, "physics", "reasoning", 0)
	if len(results) != 0 {
		t.Errorf("expected 0 physics+reasoning profiles, got %d", len(results))
	}
}

// -----------------------------------------------------------------------
// 12. TestSearchWithMinReputation
// -----------------------------------------------------------------------

func TestSearchWithMinReputation(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Create agents with varying reputations.
	k.SetProfile(ctx, &types.AgentProfile{
		Address:         testAddr(1),
		Domains:         []string{"math"},
		Capabilities:    []*types.AgentCapability{{CapabilityType: "inference"}},
		Status:          "active",
		ReputationScore: 800000, // high
		Stake:           "1000000",
	})
	k.SetProfile(ctx, &types.AgentProfile{
		Address:         testAddr(2),
		Domains:         []string{"math"},
		Capabilities:    []*types.AgentCapability{{CapabilityType: "inference"}},
		Status:          "active",
		ReputationScore: 500000, // medium
		Stake:           "1000000",
	})
	k.SetProfile(ctx, &types.AgentProfile{
		Address:         testAddr(3),
		Domains:         []string{"math"},
		Capabilities:    []*types.AgentCapability{{CapabilityType: "inference"}},
		Status:          "active",
		ReputationScore: 200000, // low
		Stake:           "1000000",
	})

	// No min reputation: all active profiles returned.
	results := k.SearchProfiles(ctx, "math", "", 0)
	if len(results) != 3 {
		t.Errorf("expected 3 profiles with no min reputation, got %d", len(results))
	}

	// Min reputation 400000: only 2 profiles pass.
	results = k.SearchProfiles(ctx, "math", "", 400000)
	if len(results) != 2 {
		t.Errorf("expected 2 profiles with min reputation 400000, got %d", len(results))
	}

	// Min reputation 700000: only 1 profile passes.
	results = k.SearchProfiles(ctx, "math", "", 700000)
	if len(results) != 1 {
		t.Errorf("expected 1 profile with min reputation 700000, got %d", len(results))
	}

	// Min reputation 900000: none pass.
	results = k.SearchProfiles(ctx, "math", "", 900000)
	if len(results) != 0 {
		t.Errorf("expected 0 profiles with min reputation 900000, got %d", len(results))
	}

	// Capability filter + reputation filter combined.
	results = k.SearchProfiles(ctx, "", "inference", 500000)
	if len(results) != 2 {
		t.Errorf("expected 2 inference profiles with min reputation 500000, got %d", len(results))
	}

	// All active profiles with reputation filter (no domain/capability filter).
	results = k.SearchProfiles(ctx, "", "", 500000)
	if len(results) != 2 {
		t.Errorf("expected 2 profiles with min reputation 500000 (no index filter), got %d", len(results))
	}
}

// -----------------------------------------------------------------------
// Additional: TestMaxCapabilitiesValidation
// -----------------------------------------------------------------------

func TestMaxCapabilitiesValidation(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	// Set max capabilities to 2.
	params := types.DefaultParams()
	params.MaxCapabilitiesPerAgent = 2
	k.SetParams(ctx, params)

	addr := testAddr(1)
	accAddr, _ := sdk.AccAddressFromBech32(addr)
	bk.setBalance(accAddr, "uzrn", sdkmath.NewInt(200000000))

	srv := keeper.NewMsgServerImpl(k)

	// Try to register with 3 capabilities (exceeds max of 2).
	_, err := srv.RegisterProfile(ctx, &types.MsgRegisterProfile{
		Sender:  addr,
		Domains: []string{"math"},
		Capabilities: []*types.AgentCapability{
			{CapabilityType: "inference"},
			{CapabilityType: "reasoning"},
			{CapabilityType: "verification"},
		},
		Stake: "1000000",
	})
	if err == nil {
		t.Error("expected max capabilities error")
	}

	// With 2 capabilities should succeed.
	_, err = srv.RegisterProfile(ctx, &types.MsgRegisterProfile{
		Sender:  addr,
		Domains: []string{"math"},
		Capabilities: []*types.AgentCapability{
			{CapabilityType: "inference"},
			{CapabilityType: "reasoning"},
		},
		Stake: "1000000",
	})
	if err != nil {
		t.Fatalf("registration with 2 capabilities should succeed: %v", err)
	}
}

// -----------------------------------------------------------------------
// Additional: TestGenesisExportImport
// -----------------------------------------------------------------------

func TestGenesisExportImport(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Set custom params.
	params := types.DefaultParams()
	params.MinRegistrationStake = "5000000"
	params.MaxCapabilitiesPerAgent = 15
	k.SetParams(ctx, params)

	// Create profiles.
	k.SetProfile(ctx, &types.AgentProfile{
		Address:         testAddr(1),
		DisplayName:     "Agent1",
		Domains:         []string{"math", "physics"},
		Capabilities:    []*types.AgentCapability{{CapabilityType: "inference"}},
		Status:          "active",
		ReputationScore: 700000,
		Stake:           "5000000",
	})
	k.SetProfile(ctx, &types.AgentProfile{
		Address:         testAddr(2),
		DisplayName:     "Agent2",
		Domains:         []string{"biology"},
		Status:          "expired",
		ReputationScore: 300000,
		Stake:           "5000000",
	})

	// Export genesis.
	genState := k.ExportGenesis(ctx)
	if len(genState.Profiles) != 2 {
		t.Errorf("expected 2 profiles in genesis export, got %d", len(genState.Profiles))
	}
	if genState.Params.MinRegistrationStake != "5000000" {
		t.Errorf("expected exported MinRegistrationStake 5000000, got %s", genState.Params.MinRegistrationStake)
	}

	// Import into a fresh keeper.
	k2, ctx2, _ := setupKeeper(t)
	k2.InitGenesis(ctx2, genState)

	// Verify params.
	gotParams := k2.GetParams(ctx2)
	if gotParams.MinRegistrationStake != "5000000" {
		t.Errorf("expected imported MinRegistrationStake 5000000, got %s", gotParams.MinRegistrationStake)
	}

	// Verify profiles.
	allProfiles := k2.GetAllProfiles(ctx2)
	if len(allProfiles) != 2 {
		t.Errorf("expected 2 profiles after import, got %d", len(allProfiles))
	}

	// Verify domain indexes were rebuilt.
	mathProfiles := k2.GetProfilesByDomain(ctx2, "math")
	if len(mathProfiles) != 1 {
		t.Errorf("expected 1 math profile after import, got %d", len(mathProfiles))
	}
}

// -----------------------------------------------------------------------
// Additional: TestUpdateParamsGovernance
// -----------------------------------------------------------------------

func TestUpdateParamsGovernance(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	authority := k.GetAuthority()

	srv := keeper.NewMsgServerImpl(k)
	newParams := types.DefaultParams()
	newParams.MinRegistrationStake = "10000000"
	newParams.MaxCapabilitiesPerAgent = 50
	newParams.ProfileExpiryBlocks = 200000

	// Authority can update params.
	_, err := srv.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: authority,
		Params:    newParams,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := k.GetParams(ctx)
	if got.MinRegistrationStake != "10000000" {
		t.Errorf("expected 10000000, got %s", got.MinRegistrationStake)
	}
	if got.MaxCapabilitiesPerAgent != 50 {
		t.Errorf("expected 50, got %d", got.MaxCapabilitiesPerAgent)
	}
	if got.ProfileExpiryBlocks != 200000 {
		t.Errorf("expected 200000, got %d", got.ProfileExpiryBlocks)
	}

	// Non-authority should be rejected.
	_, err = srv.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: testAddr(99),
		Params:    types.DefaultParams(),
	})
	if err == nil {
		t.Error("expected unauthorized error")
	}

	// Nil params should be rejected.
	_, err = srv.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: authority,
		Params:    nil,
	})
	if err == nil {
		t.Error("expected nil params error")
	}

	// Invalid params should be rejected.
	invalidParams := types.DefaultParams()
	invalidParams.MaxCapabilitiesPerAgent = 0 // invalid
	_, err = srv.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: authority,
		Params:    invalidParams,
	})
	if err == nil {
		t.Error("expected invalid params error")
	}
}
