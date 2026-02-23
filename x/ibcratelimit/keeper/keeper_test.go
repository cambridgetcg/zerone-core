package keeper_test

import (
	"math/big"
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

	"github.com/zerone-chain/zerone/x/ibcratelimit/keeper"
	"github.com/zerone-chain/zerone/x/ibcratelimit/types"
)

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("zrn", "zrnpub")
	config.SetBech32PrefixForValidator("zrnvaloper", "zrnvaloperpub")
	config.SetBech32PrefixForConsensusNode("zrnvalcons", "zrnvalconspub")
}

var testAuthority = sdk.AccAddress([]byte("authority-addr------")).String()

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context) {
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

	k := keeper.NewKeeper(runtime.NewKVStoreService(storeKey), cdc, testAuthority)
	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 100}, false, log.NewNopLogger())

	return k, ctx
}

func setupKeeperAtHeight(t *testing.T, height int64) (keeper.Keeper, sdk.Context) {
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

	k := keeper.NewKeeper(runtime.NewKVStoreService(storeKey), cdc, testAuthority)
	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: height}, false, log.NewNopLogger())

	return k, ctx
}

// ---------- 1. TestParamsCRUD ----------

func TestParamsCRUD(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Default params before any SetParams call
	got := k.GetParams(ctx)
	if got == nil {
		t.Fatal("expected non-nil default params")
	}
	if !got.Enabled {
		t.Fatal("expected default params to have Enabled=true")
	}

	// Set custom params
	k.SetParams(ctx, &types.Params{Enabled: false})
	got = k.GetParams(ctx)
	if got.Enabled {
		t.Fatal("expected Enabled=false after SetParams")
	}

	// Overwrite with Enabled=true
	k.SetParams(ctx, &types.Params{Enabled: true})
	got = k.GetParams(ctx)
	if !got.Enabled {
		t.Fatal("expected Enabled=true after second SetParams")
	}
}

// ---------- 2. TestRateLimitSetGetDelete ----------

func TestRateLimitSetGetDelete(t *testing.T) {
	k, ctx := setupKeeper(t)

	rl := &types.RateLimit{
		ChannelId:    "channel-0",
		Denom:        "uzrn",
		MaxSend:      "1000000",
		MaxRecv:      "2000000",
		WindowBlocks: 100,
		CurrentSend:  "0",
		CurrentRecv:  "0",
		WindowStart:  0,
	}

	// Set
	k.SetRateLimit(ctx, rl)

	// Get — found
	got, found := k.GetRateLimit(ctx, "channel-0", "uzrn")
	if !found {
		t.Fatal("expected rate limit to be found")
	}
	if got.ChannelId != "channel-0" {
		t.Fatalf("expected channel_id channel-0, got %s", got.ChannelId)
	}
	if got.Denom != "uzrn" {
		t.Fatalf("expected denom uzrn, got %s", got.Denom)
	}
	if got.MaxSend != "1000000" {
		t.Fatalf("expected max_send 1000000, got %s", got.MaxSend)
	}
	if got.MaxRecv != "2000000" {
		t.Fatalf("expected max_recv 2000000, got %s", got.MaxRecv)
	}
	if got.WindowBlocks != 100 {
		t.Fatalf("expected window_blocks 100, got %d", got.WindowBlocks)
	}

	// Delete
	k.DeleteRateLimit(ctx, "channel-0", "uzrn")

	// Get — not found
	_, found = k.GetRateLimit(ctx, "channel-0", "uzrn")
	if found {
		t.Fatal("expected rate limit to be not found after delete")
	}
}

// ---------- 3. TestGetAllRateLimits ----------

func TestGetAllRateLimits(t *testing.T) {
	k, ctx := setupKeeper(t)

	limits := []*types.RateLimit{
		{ChannelId: "channel-0", Denom: "uzrn", MaxSend: "1000", MaxRecv: "1000", WindowBlocks: 50, CurrentSend: "0", CurrentRecv: "0"},
		{ChannelId: "channel-0", Denom: "uatom", MaxSend: "2000", MaxRecv: "2000", WindowBlocks: 100, CurrentSend: "0", CurrentRecv: "0"},
		{ChannelId: "channel-1", Denom: "uzrn", MaxSend: "5000", MaxRecv: "5000", WindowBlocks: 200, CurrentSend: "0", CurrentRecv: "0"},
	}

	for _, rl := range limits {
		k.SetRateLimit(ctx, rl)
	}

	all := k.GetAllRateLimits(ctx)
	if len(all) != 3 {
		t.Fatalf("expected 3 rate limits, got %d", len(all))
	}

	// Build a lookup map by channelId+denom for order-independent checking
	lookup := make(map[string]*types.RateLimit)
	for _, rl := range all {
		lookup[rl.ChannelId+"/"+rl.Denom] = rl
	}

	for _, expected := range limits {
		key := expected.ChannelId + "/" + expected.Denom
		got, ok := lookup[key]
		if !ok {
			t.Fatalf("expected rate limit for %s not found in GetAllRateLimits", key)
		}
		if got.MaxSend != expected.MaxSend {
			t.Fatalf("rate limit %s: expected max_send %s, got %s", key, expected.MaxSend, got.MaxSend)
		}
	}
}

// ---------- 4. TestPacketFlowSetGetDelete ----------

func TestPacketFlowSetGetDelete(t *testing.T) {
	k, ctx := setupKeeper(t)

	flow := &types.PacketFlow{
		ChannelId: "channel-0",
		Sequence:  42,
		Denom:     "uzrn",
		Amount:    "500000",
	}

	// Set
	k.SetPacketFlow(ctx, flow)

	// Get — found
	got, found := k.GetPacketFlow(ctx, "channel-0", 42)
	if !found {
		t.Fatal("expected packet flow to be found")
	}
	if got.ChannelId != "channel-0" {
		t.Fatalf("expected channel_id channel-0, got %s", got.ChannelId)
	}
	if got.Sequence != 42 {
		t.Fatalf("expected sequence 42, got %d", got.Sequence)
	}
	if got.Denom != "uzrn" {
		t.Fatalf("expected denom uzrn, got %s", got.Denom)
	}
	if got.Amount != "500000" {
		t.Fatalf("expected amount 500000, got %s", got.Amount)
	}

	// Delete
	k.DeletePacketFlow(ctx, "channel-0", 42)

	// Get — not found
	_, found = k.GetPacketFlow(ctx, "channel-0", 42)
	if found {
		t.Fatal("expected packet flow to be not found after delete")
	}
}

// ---------- 5. TestWindowReset ----------

func TestWindowReset(t *testing.T) {
	// Create keeper with height 100, set rate limit with WindowStart=100, WindowBlocks=50
	k, ctx := setupKeeper(t) // height=100

	k.SetParams(ctx, &types.Params{Enabled: true})

	rl := &types.RateLimit{
		ChannelId:    "channel-0",
		Denom:        "uzrn",
		MaxSend:      "1000",
		MaxRecv:      "1000",
		WindowBlocks: 50,
		CurrentSend:  "500",
		CurrentRecv:  "300",
		WindowStart:  100,
	}
	k.SetRateLimit(ctx, rl)

	// At height 100, window is [100, 150). Height 100 < 100+50, so no reset.
	// At height 151, window should reset because 151 >= 100+50.
	ctx151 := sdk.NewContext(
		ctx.MultiStore(),
		cmtproto.Header{Height: 151},
		false,
		log.NewNopLogger(),
	)

	// Trigger window reset via CheckAndUpdateSendQuota with a small amount
	err := k.CheckAndUpdateSendQuota(ctx151, "channel-0", "uzrn", big.NewInt(1))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify the counters were reset
	got, found := k.GetRateLimit(ctx151, "channel-0", "uzrn")
	if !found {
		t.Fatal("expected rate limit to be found")
	}

	// After reset, CurrentSend should be "1" (reset to 0, then +1 from the quota check)
	if got.CurrentSend != "1" {
		t.Fatalf("expected current_send=1 after reset+send, got %s", got.CurrentSend)
	}
	// CurrentRecv should be "0" (reset)
	if got.CurrentRecv != "0" {
		t.Fatalf("expected current_recv=0 after reset, got %s", got.CurrentRecv)
	}
	// WindowStart should be updated to 151
	if got.WindowStart != 151 {
		t.Fatalf("expected window_start=151 after reset, got %d", got.WindowStart)
	}
}

// ---------- 6. TestCheckAndUpdateSendQuota ----------

func TestCheckAndUpdateSendQuota(t *testing.T) {
	k, ctx := setupKeeper(t)
	k.SetParams(ctx, &types.Params{Enabled: true})

	rl := &types.RateLimit{
		ChannelId:    "channel-0",
		Denom:        "uzrn",
		MaxSend:      "100",
		MaxRecv:      "100",
		WindowBlocks: 1000,
		CurrentSend:  "0",
		CurrentRecv:  "0",
		WindowStart:  100,
	}
	k.SetRateLimit(ctx, rl)

	// Under limit: send 50 out of 100
	err := k.CheckAndUpdateSendQuota(ctx, "channel-0", "uzrn", big.NewInt(50))
	if err != nil {
		t.Fatalf("expected no error for under limit, got %v", err)
	}

	// At limit: send 50 more to exactly reach 100
	err = k.CheckAndUpdateSendQuota(ctx, "channel-0", "uzrn", big.NewInt(50))
	if err != nil {
		t.Fatalf("expected no error for at limit, got %v", err)
	}

	// Verify exactly at 100
	got, _ := k.GetRateLimit(ctx, "channel-0", "uzrn")
	if got.CurrentSend != "100" {
		t.Fatalf("expected current_send=100, got %s", got.CurrentSend)
	}

	// Over limit: send 1 more
	err = k.CheckAndUpdateSendQuota(ctx, "channel-0", "uzrn", big.NewInt(1))
	if err == nil {
		t.Fatal("expected error for over limit, got nil")
	}

	// No rate limit configured — should pass through
	err = k.CheckAndUpdateSendQuota(ctx, "channel-99", "uzrn", big.NewInt(999999))
	if err != nil {
		t.Fatalf("expected no error for unconfigured channel, got %v", err)
	}
}

// ---------- 7. TestCheckAndUpdateRecvQuota ----------

func TestCheckAndUpdateRecvQuota(t *testing.T) {
	k, ctx := setupKeeper(t)
	k.SetParams(ctx, &types.Params{Enabled: true})

	rl := &types.RateLimit{
		ChannelId:    "channel-0",
		Denom:        "uzrn",
		MaxSend:      "100",
		MaxRecv:      "200",
		WindowBlocks: 1000,
		CurrentSend:  "0",
		CurrentRecv:  "0",
		WindowStart:  100,
	}
	k.SetRateLimit(ctx, rl)

	// Under limit: receive 100 out of 200
	err := k.CheckAndUpdateRecvQuota(ctx, "channel-0", "uzrn", big.NewInt(100))
	if err != nil {
		t.Fatalf("expected no error for under limit, got %v", err)
	}

	// At limit: receive 100 more to exactly reach 200
	err = k.CheckAndUpdateRecvQuota(ctx, "channel-0", "uzrn", big.NewInt(100))
	if err != nil {
		t.Fatalf("expected no error for at limit, got %v", err)
	}

	// Verify exactly at 200
	got, _ := k.GetRateLimit(ctx, "channel-0", "uzrn")
	if got.CurrentRecv != "200" {
		t.Fatalf("expected current_recv=200, got %s", got.CurrentRecv)
	}

	// Over limit: receive 1 more
	err = k.CheckAndUpdateRecvQuota(ctx, "channel-0", "uzrn", big.NewInt(1))
	if err == nil {
		t.Fatal("expected error for over limit, got nil")
	}

	// No rate limit configured — should pass through
	err = k.CheckAndUpdateRecvQuota(ctx, "channel-99", "uzrn", big.NewInt(999999))
	if err != nil {
		t.Fatalf("expected no error for unconfigured channel, got %v", err)
	}
}

// ---------- 8. TestReverseSendQuota ----------

func TestReverseSendQuota(t *testing.T) {
	k, ctx := setupKeeper(t)
	k.SetParams(ctx, &types.Params{Enabled: true})

	rl := &types.RateLimit{
		ChannelId:    "channel-0",
		Denom:        "uzrn",
		MaxSend:      "1000",
		MaxRecv:      "1000",
		WindowBlocks: 1000,
		CurrentSend:  "0",
		CurrentRecv:  "0",
		WindowStart:  100,
	}
	k.SetRateLimit(ctx, rl)

	// Send 500
	err := k.CheckAndUpdateSendQuota(ctx, "channel-0", "uzrn", big.NewInt(500))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Reverse 200
	k.ReverseSendQuota(ctx, "channel-0", "uzrn", big.NewInt(200))

	got, _ := k.GetRateLimit(ctx, "channel-0", "uzrn")
	if got.CurrentSend != "300" {
		t.Fatalf("expected current_send=300 after reverse, got %s", got.CurrentSend)
	}

	// Reverse more than current (should clamp to 0, not go negative)
	k.ReverseSendQuota(ctx, "channel-0", "uzrn", big.NewInt(999))

	got, _ = k.GetRateLimit(ctx, "channel-0", "uzrn")
	if got.CurrentSend != "0" {
		t.Fatalf("expected current_send=0 after over-reverse, got %s", got.CurrentSend)
	}

	// Reverse on a missing rate limit — should not panic
	k.ReverseSendQuota(ctx, "channel-99", "uzrn", big.NewInt(100))
}

// ---------- 9. TestMsgServerAddRateLimit ----------

func TestMsgServerAddRateLimit(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Success
	msg := &types.MsgAddRateLimit{
		Authority:    testAuthority,
		ChannelId:    "channel-0",
		Denom:        "uzrn",
		MaxSend:      "1000000",
		MaxRecv:      "2000000",
		WindowBlocks: 100,
	}

	_, err := ms.AddRateLimit(ctx, msg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify it was stored
	got, found := k.GetRateLimit(ctx, "channel-0", "uzrn")
	if !found {
		t.Fatal("expected rate limit to be found after AddRateLimit")
	}
	if got.MaxSend != "1000000" {
		t.Fatalf("expected max_send=1000000, got %s", got.MaxSend)
	}
	if got.CurrentSend != "0" {
		t.Fatalf("expected current_send=0, got %s", got.CurrentSend)
	}

	// Duplicate — should error
	_, err = ms.AddRateLimit(ctx, msg)
	if err == nil {
		t.Fatal("expected error for duplicate rate limit, got nil")
	}

	// Wrong authority — should error
	badMsg := &types.MsgAddRateLimit{
		Authority:    "zrn1wrongauthority",
		ChannelId:    "channel-1",
		Denom:        "uzrn",
		MaxSend:      "1000",
		MaxRecv:      "1000",
		WindowBlocks: 50,
	}
	_, err = ms.AddRateLimit(ctx, badMsg)
	if err == nil {
		t.Fatal("expected error for wrong authority, got nil")
	}
}

// ---------- 10. TestMsgServerRemoveRateLimit ----------

func TestMsgServerRemoveRateLimit(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Seed a rate limit
	k.SetRateLimit(ctx, &types.RateLimit{
		ChannelId:    "channel-0",
		Denom:        "uzrn",
		MaxSend:      "1000",
		MaxRecv:      "1000",
		WindowBlocks: 100,
		CurrentSend:  "0",
		CurrentRecv:  "0",
	})

	// Success
	_, err := ms.RemoveRateLimit(ctx, &types.MsgRemoveRateLimit{
		Authority: testAuthority,
		ChannelId: "channel-0",
		Denom:     "uzrn",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify deleted
	_, found := k.GetRateLimit(ctx, "channel-0", "uzrn")
	if found {
		t.Fatal("expected rate limit to be deleted")
	}

	// Not found — should error
	_, err = ms.RemoveRateLimit(ctx, &types.MsgRemoveRateLimit{
		Authority: testAuthority,
		ChannelId: "channel-0",
		Denom:     "uzrn",
	})
	if err == nil {
		t.Fatal("expected error for removing non-existent rate limit, got nil")
	}
}

// ---------- 11. TestMsgServerUpdateParams ----------

func TestMsgServerUpdateParams(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Success — disable
	_, err := ms.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: testAuthority,
		Params:    &types.Params{Enabled: false},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	got := k.GetParams(ctx)
	if got.Enabled {
		t.Fatal("expected Enabled=false after UpdateParams")
	}

	// Success — re-enable
	_, err = ms.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: testAuthority,
		Params:    &types.Params{Enabled: true},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	got = k.GetParams(ctx)
	if !got.Enabled {
		t.Fatal("expected Enabled=true after UpdateParams")
	}

	// Wrong authority — should error
	_, err = ms.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: "zrn1wrongauthority",
		Params:    &types.Params{Enabled: false},
	})
	if err == nil {
		t.Fatal("expected error for wrong authority, got nil")
	}
}

// ---------- 12. TestGenesisRoundTrip ----------

func TestGenesisRoundTrip(t *testing.T) {
	k, ctx := setupKeeper(t)

	genState := &types.GenesisState{
		Params: &types.Params{Enabled: true},
		RateLimits: []*types.RateLimit{
			{
				ChannelId:    "channel-0",
				Denom:        "uzrn",
				MaxSend:      "1000000",
				MaxRecv:      "2000000",
				WindowBlocks: 100,
				CurrentSend:  "500",
				CurrentRecv:  "300",
				WindowStart:  50,
			},
			{
				ChannelId:    "channel-1",
				Denom:        "uatom",
				MaxSend:      "5000000",
				MaxRecv:      "5000000",
				WindowBlocks: 200,
				CurrentSend:  "0",
				CurrentRecv:  "0",
				WindowStart:  0,
			},
		},
	}

	k.InitGenesis(ctx, genState)
	exported := k.ExportGenesis(ctx)

	// Verify params
	if exported.Params == nil {
		t.Fatal("expected non-nil params in exported genesis")
	}
	if exported.Params.Enabled != genState.Params.Enabled {
		t.Fatalf("expected Enabled=%v, got %v", genState.Params.Enabled, exported.Params.Enabled)
	}

	// Verify rate limits
	if len(exported.RateLimits) != len(genState.RateLimits) {
		t.Fatalf("expected %d rate limits, got %d", len(genState.RateLimits), len(exported.RateLimits))
	}

	// Build lookup for order-independent comparison
	lookup := make(map[string]*types.RateLimit)
	for _, rl := range exported.RateLimits {
		lookup[rl.ChannelId+"/"+rl.Denom] = rl
	}

	for _, expected := range genState.RateLimits {
		key := expected.ChannelId + "/" + expected.Denom
		got, ok := lookup[key]
		if !ok {
			t.Fatalf("exported genesis missing rate limit for %s", key)
		}
		if got.MaxSend != expected.MaxSend {
			t.Fatalf("%s: expected max_send=%s, got %s", key, expected.MaxSend, got.MaxSend)
		}
		if got.MaxRecv != expected.MaxRecv {
			t.Fatalf("%s: expected max_recv=%s, got %s", key, expected.MaxRecv, got.MaxRecv)
		}
		if got.WindowBlocks != expected.WindowBlocks {
			t.Fatalf("%s: expected window_blocks=%d, got %d", key, expected.WindowBlocks, got.WindowBlocks)
		}
		if got.CurrentSend != expected.CurrentSend {
			t.Fatalf("%s: expected current_send=%s, got %s", key, expected.CurrentSend, got.CurrentSend)
		}
		if got.CurrentRecv != expected.CurrentRecv {
			t.Fatalf("%s: expected current_recv=%s, got %s", key, expected.CurrentRecv, got.CurrentRecv)
		}
		if got.WindowStart != expected.WindowStart {
			t.Fatalf("%s: expected window_start=%d, got %d", key, expected.WindowStart, got.WindowStart)
		}
	}
}

// ---------- 13. TestTimeoutRefundsSendQuota ----------

func TestTimeoutRefundsSendQuota(t *testing.T) {
	k, ctx := setupKeeper(t)
	k.SetParams(ctx, &types.Params{Enabled: true})

	rl := &types.RateLimit{
		ChannelId:    "channel-0",
		Denom:        "uzrn",
		MaxSend:      "100",
		MaxRecv:      "100",
		WindowBlocks: 1000,
		CurrentSend:  "0",
		CurrentRecv:  "0",
		WindowStart:  100,
	}
	k.SetRateLimit(ctx, rl)

	// Send 80 out of 100
	err := k.CheckAndUpdateSendQuota(ctx, "channel-0", "uzrn", big.NewInt(80))
	if err != nil {
		t.Fatalf("send should succeed: %v", err)
	}

	// Sending 30 more would exceed limit (80+30=110 > 100)
	err = k.CheckAndUpdateSendQuota(ctx, "channel-0", "uzrn", big.NewInt(30))
	if err == nil {
		t.Fatal("expected error for exceeding limit")
	}

	// Simulate timeout: refund 50 from previous send
	k.ReverseSendQuota(ctx, "channel-0", "uzrn", big.NewInt(50))

	// Verify current_send decreased: 80 - 50 = 30
	got, _ := k.GetRateLimit(ctx, "channel-0", "uzrn")
	if got.CurrentSend != "30" {
		t.Fatalf("expected current_send=30 after timeout refund, got %s", got.CurrentSend)
	}

	// Now sending 30 more should succeed (30+30=60 <= 100)
	err = k.CheckAndUpdateSendQuota(ctx, "channel-0", "uzrn", big.NewInt(30))
	if err != nil {
		t.Fatalf("send should succeed after timeout refund: %v", err)
	}
}

// ---------- 14. TestErrorAckRefundsSendQuota ----------

func TestErrorAckRefundsSendQuota(t *testing.T) {
	k, ctx := setupKeeper(t)
	k.SetParams(ctx, &types.Params{Enabled: true})

	rl := &types.RateLimit{
		ChannelId:    "channel-0",
		Denom:        "uzrn",
		MaxSend:      "1000",
		MaxRecv:      "1000",
		WindowBlocks: 1000,
		CurrentSend:  "0",
		CurrentRecv:  "0",
		WindowStart:  100,
	}
	k.SetRateLimit(ctx, rl)

	// Store packet flow for error ack scenario
	k.SetPacketFlow(ctx, &types.PacketFlow{
		ChannelId: "channel-0",
		Sequence:  1,
		Denom:     "uzrn",
		Amount:    "500",
	})

	// Send 500
	err := k.CheckAndUpdateSendQuota(ctx, "channel-0", "uzrn", big.NewInt(500))
	if err != nil {
		t.Fatalf("send should succeed: %v", err)
	}

	// Simulate error ack: reverse the send
	k.ReverseSendQuota(ctx, "channel-0", "uzrn", big.NewInt(500))

	// Verify full refund
	got, _ := k.GetRateLimit(ctx, "channel-0", "uzrn")
	if got.CurrentSend != "0" {
		t.Fatalf("expected current_send=0 after error ack refund, got %s", got.CurrentSend)
	}

	// Clean up packet flow
	k.DeletePacketFlow(ctx, "channel-0", 1)
}

// ---------- 15. TestUnconfiguredChannelPassesThrough ----------

func TestUnconfiguredChannelPassesThrough(t *testing.T) {
	k, ctx := setupKeeper(t)
	k.SetParams(ctx, &types.Params{Enabled: true})

	// Only configure channel-0
	k.SetRateLimit(ctx, &types.RateLimit{
		ChannelId:    "channel-0",
		Denom:        "uzrn",
		MaxSend:      "100",
		MaxRecv:      "100",
		WindowBlocks: 1000,
		CurrentSend:  "0",
		CurrentRecv:  "0",
		WindowStart:  100,
	})

	// Sending on unconfigured channel should pass through
	err := k.CheckAndUpdateSendQuota(ctx, "channel-99", "uzrn", big.NewInt(999999999))
	if err != nil {
		t.Fatalf("expected unconfigured channel to pass through, got %v", err)
	}

	// Receiving on unconfigured channel should pass through
	err = k.CheckAndUpdateRecvQuota(ctx, "channel-99", "uzrn", big.NewInt(999999999))
	if err != nil {
		t.Fatalf("expected unconfigured channel to pass through, got %v", err)
	}

	// Unconfigured denom on configured channel should also pass through
	err = k.CheckAndUpdateSendQuota(ctx, "channel-0", "uatom", big.NewInt(999999999))
	if err != nil {
		t.Fatalf("expected unconfigured denom to pass through, got %v", err)
	}
}

// ---------- 16. TestDisabledParams ----------

func TestDisabledParams(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Set up a rate limit with max_send=100, max_recv=100
	rl := &types.RateLimit{
		ChannelId:    "channel-0",
		Denom:        "uzrn",
		MaxSend:      "100",
		MaxRecv:      "100",
		WindowBlocks: 1000,
		CurrentSend:  "0",
		CurrentRecv:  "0",
		WindowStart:  100,
	}
	k.SetRateLimit(ctx, rl)

	// Disable rate limiting
	k.SetParams(ctx, &types.Params{Enabled: false})

	// Send over limit — should still pass because disabled
	err := k.CheckAndUpdateSendQuota(ctx, "channel-0", "uzrn", big.NewInt(9999))
	if err != nil {
		t.Fatalf("expected no error when disabled, got %v", err)
	}

	// Recv over limit — should still pass because disabled
	err = k.CheckAndUpdateRecvQuota(ctx, "channel-0", "uzrn", big.NewInt(9999))
	if err != nil {
		t.Fatalf("expected no error when disabled, got %v", err)
	}

	// Verify the counters were NOT updated (disabled means early return)
	got, found := k.GetRateLimit(ctx, "channel-0", "uzrn")
	if !found {
		t.Fatal("expected rate limit to still exist")
	}
	if got.CurrentSend != "0" {
		t.Fatalf("expected current_send=0 when disabled (no update), got %s", got.CurrentSend)
	}
	if got.CurrentRecv != "0" {
		t.Fatalf("expected current_recv=0 when disabled (no update), got %s", got.CurrentRecv)
	}

	// Re-enable and verify the limit is enforced again
	k.SetParams(ctx, &types.Params{Enabled: true})

	err = k.CheckAndUpdateSendQuota(ctx, "channel-0", "uzrn", big.NewInt(101))
	if err == nil {
		t.Fatal("expected error when re-enabled and over limit, got nil")
	}

	err = k.CheckAndUpdateRecvQuota(ctx, "channel-0", "uzrn", big.NewInt(101))
	if err == nil {
		t.Fatal("expected error when re-enabled and over limit, got nil")
	}
}
