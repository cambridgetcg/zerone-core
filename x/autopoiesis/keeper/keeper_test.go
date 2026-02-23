package keeper_test

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"
	"time"

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

	"github.com/zerone-chain/zerone/x/autopoiesis/keeper"
	"github.com/zerone-chain/zerone/x/autopoiesis/types"
)

// --- Mock Keepers ---

type mockStakingKeeper struct {
	totalBonded    *big.Int
	activeValCount int
}

func (m *mockStakingKeeper) GetTotalBondedStake(_ sdk.Context) *big.Int {
	if m.totalBonded == nil {
		return big.NewInt(0)
	}
	return new(big.Int).Set(m.totalBonded)
}

func (m *mockStakingKeeper) GetActiveValidatorCount(_ sdk.Context) int {
	return m.activeValCount
}

type mockKnowledgeKeeper struct {
	verificationRate uint64
}

func (m *mockKnowledgeKeeper) GetVerificationRate(_ context.Context) uint64 {
	return m.verificationRate
}

type mockEmergencyKeeper struct {
	halted bool
}

func (m *mockEmergencyKeeper) IsHalted(_ context.Context) bool {
	return m.halted
}

// --- Test Setup ---

func setupKeeper(t *testing.T) (keeper.Keeper, *mockStakingKeeper, *mockKnowledgeKeeper, *mockEmergencyKeeper, sdk.Context) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		t.Fatalf("failed to load latest version: %v", err)
	}

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 1}, false, log.NewNopLogger()).
		WithBlockTime(time.Now())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	mockSK := &mockStakingKeeper{
		totalBonded:    big.NewInt(1_000_000_000_000),
		activeValCount: 21,
	}
	mockKK := &mockKnowledgeKeeper{verificationRate: types.BPSScale}
	mockEK := &mockEmergencyKeeper{halted: false}

	k := keeper.NewKeeper(runtime.NewKVStoreService(storeKey), cdc, "authority", mockSK)
	k.SetKnowledgeKeeper(mockKK)
	k.SetEmergencyKeeper(mockEK)

	// Init with default genesis.
	genState := types.DefaultGenesis()
	genState.Activated = true
	k.InitGenesis(ctx, genState)

	return k, mockSK, mockKK, mockEK, ctx
}

// advanceBlocks returns a new context with block height advanced by n.
func advanceBlocks(ctx sdk.Context, n int64) sdk.Context {
	return ctx.WithBlockHeight(ctx.BlockHeight() + n)
}

// --- Tests ---

// 1. Epoch boundary triggers multiplier adjustment.
func TestEpochBoundaryTriggersAdjustment(t *testing.T) {
	k, mockSK, mockKK, _, ctx := setupKeeper(t)

	// Set lower-than-perfect signals to cause SSI < 1M.
	mockSK.activeValCount = 10 // ~47.6% of target 21
	mockKK.verificationRate = 500_000 // 50%

	// First call at block 1 — sets baseline.
	k.CollectAndAdapt(ctx)

	// Advance past epoch boundary (default 100 blocks).
	ctx = advanceBlocks(ctx, 101)
	k.CollectAndAdapt(ctx)

	// Check that multipliers changed from 1.0x.
	rewardsVal, err := k.GetMultiplier(ctx, "rewards.block")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rewardsVal == types.BPSScale {
		t.Error("expected rewards.block to change from 1.0x after epoch with degraded signals")
	}

	// Verify snapshot was stored.
	snap, found := k.GetEpochSnapshot(ctx, 1)
	if !found {
		t.Fatal("expected epoch 1 snapshot to exist")
	}
	if snap.SsiScore == 0 {
		t.Error("expected non-zero SSI score in snapshot")
	}
}

// 2. SSI computation from cross-module signals.
func TestSSIComputation(t *testing.T) {
	tests := []struct {
		name       string
		staking    uint64
		verif      uint64
		halted     bool
		wantMin    uint64
		wantMax    uint64
		wantCat    string
	}{
		{
			name:    "all healthy",
			staking: 1_000_000, verif: 1_000_000, halted: false,
			wantMin: 900_000, wantMax: 1_000_000, wantCat: types.SSIThriving,
		},
		{
			name:    "zero staking and verification",
			staking: 0, verif: 0, halted: false,
			wantMin: 100_000, wantMax: 300_000, wantCat: types.SSICritical,
		},
		{
			name:    "emergency halted",
			staking: 1_000_000, verif: 1_000_000, halted: true,
			wantMin: 700_000, wantMax: 900_000, wantCat: types.SSIThriving,
		},
		{
			name:    "half staking half verif",
			staking: 500_000, verif: 500_000, halted: false,
			wantMin: 500_000, wantMax: 700_000, wantCat: types.SSIHealthy,
		},
	}

	params := types.DefaultParams()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ssi := types.ComputeSSI(tc.staking, tc.verif, tc.halted)
			if ssi < tc.wantMin || ssi > tc.wantMax {
				t.Errorf("SSI=%d, expected in [%d, %d]", ssi, tc.wantMin, tc.wantMax)
			}
			cat := types.ClassifySSI(ssi, &params)
			if cat != tc.wantCat {
				t.Errorf("category=%s, expected %s (ssi=%d)", cat, tc.wantCat, ssi)
			}
		})
	}
}

// 3. Multiplier respects min/max bounds and max-change-per-epoch.
func TestMultiplierBoundsAndMaxChange(t *testing.T) {
	k, mockSK, mockKK, _, ctx := setupKeeper(t)

	// Set very low signals to push slashing.severity high.
	mockSK.activeValCount = 0
	mockKK.verificationRate = 0

	// Set baseline.
	k.CollectAndAdapt(ctx)

	// Run one epoch.
	ctx = advanceBlocks(ctx, 101)
	k.CollectAndAdapt(ctx)

	ms, found := k.GetMultiplierState(ctx, "slashing.severity")
	if !found {
		t.Fatal("slashing.severity not found")
	}

	params := k.GetParams(ctx)

	// Change should be at most MaxChangePerEpochBps from initial 1.0x.
	delta := int64(ms.CurrentBps) - int64(types.BPSScale)
	if delta < 0 {
		delta = -delta
	}
	if uint64(delta) > params.MaxChangePerEpochBps {
		t.Errorf("delta=%d exceeds max_change_per_epoch=%d", delta, params.MaxChangePerEpochBps)
	}

	// Current must be within bounds.
	if ms.CurrentBps < ms.MinBps || ms.CurrentBps > ms.MaxBps {
		t.Errorf("current_bps=%d outside [%d, %d]", ms.CurrentBps, ms.MinBps, ms.MaxBps)
	}
}

// 4. Frozen multiplier skips adjustment.
func TestFrozenMultiplierSkips(t *testing.T) {
	k, mockSK, mockKK, _, ctx := setupKeeper(t)

	// Degrade signals.
	mockSK.activeValCount = 5
	mockKK.verificationRate = 300_000

	// Freeze rewards.block.
	k.SetMultiplierFrozen(ctx, "rewards.block", true)

	// Get initial value.
	initialMs, _ := k.GetMultiplierState(ctx, "rewards.block")
	initialBps := initialMs.CurrentBps

	// Set baseline + run epoch.
	k.CollectAndAdapt(ctx)
	ctx = advanceBlocks(ctx, 101)
	k.CollectAndAdapt(ctx)

	// Frozen multiplier should not change.
	ms, _ := k.GetMultiplierState(ctx, "rewards.block")
	if ms.CurrentBps != initialBps {
		t.Errorf("frozen multiplier changed: was %d, now %d", initialBps, ms.CurrentBps)
	}

	// But non-frozen multiplier should change.
	slashMs, _ := k.GetMultiplierState(ctx, "slashing.severity")
	if slashMs.CurrentBps == types.BPSScale {
		t.Error("expected non-frozen slashing.severity to change")
	}
}

// 5. Override sets value directly.
func TestOverrideMultiplier(t *testing.T) {
	k, _, _, _, ctx := setupKeeper(t)

	msgServer := keeper.NewMsgServerImpl(k)

	_, err := msgServer.OverrideMultiplier(ctx, &types.MsgOverrideMultiplier{
		Authority: "authority",
		Path:      "rewards.block",
		Value:     750_000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ms, found := k.GetMultiplierState(ctx, "rewards.block")
	if !found {
		t.Fatal("rewards.block not found after override")
	}
	if ms.CurrentBps != 750_000 {
		t.Errorf("expected 750000, got %d", ms.CurrentBps)
	}
	if ms.TargetBps != 750_000 {
		t.Errorf("expected target 750000, got %d", ms.TargetBps)
	}
}

// 6. Emergency halt disables auto-adjustment.
func TestEmergencyHaltDisables(t *testing.T) {
	k, _, _, mockEK, ctx := setupKeeper(t)

	// Set baseline.
	k.CollectAndAdapt(ctx)

	// Halt the chain.
	mockEK.halted = true

	// Advance past epoch.
	ctx = advanceBlocks(ctx, 101)
	k.CollectAndAdapt(ctx)

	// Multipliers should remain at 1.0x (no adjustment during halt).
	rewardsVal, err := k.GetMultiplier(ctx, "rewards.block")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rewardsVal != types.BPSScale {
		t.Errorf("expected 1.0x during halt, got %d", rewardsVal)
	}
}

// 7. Genesis import/export round-trip.
func TestGenesisRoundTrip(t *testing.T) {
	k, _, _, _, ctx := setupKeeper(t)

	// Set baseline + run an epoch to generate some state.
	k.CollectAndAdapt(ctx)
	ctx = advanceBlocks(ctx, 101)
	k.CollectAndAdapt(ctx)

	// Export.
	exported := k.ExportGenesis(ctx)

	// Marshal/unmarshal (simulates file round-trip).
	bz, err := json.Marshal(exported)
	if err != nil {
		t.Fatalf("failed to marshal genesis: %v", err)
	}
	var imported types.GenesisState
	if err := json.Unmarshal(bz, &imported); err != nil {
		t.Fatalf("failed to unmarshal genesis: %v", err)
	}

	// Validate imported state.
	if err := imported.Validate(); err != nil {
		t.Fatalf("imported genesis invalid: %v", err)
	}

	// Set up a fresh keeper and import.
	storeKey := storetypes.NewKVStoreKey(types.StoreKey + "2")
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		t.Fatalf("failed to load: %v", err)
	}
	ctx2 := sdk.NewContext(stateStore, cmtproto.Header{Height: 200}, false, log.NewNopLogger()).
		WithBlockTime(time.Now())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	k2 := keeper.NewKeeper(runtime.NewKVStoreService(storeKey), cdc, "authority", nil)
	k2.InitGenesis(ctx2, &imported)

	// Verify round-tripped state matches.
	reExported := k2.ExportGenesis(ctx2)

	if exported.Activated != reExported.Activated {
		t.Errorf("activated mismatch: %v vs %v", exported.Activated, reExported.Activated)
	}
	if len(exported.Multipliers) != len(reExported.Multipliers) {
		t.Errorf("multiplier count mismatch: %d vs %d", len(exported.Multipliers), len(reExported.Multipliers))
	}
	for i, m := range exported.Multipliers {
		if i < len(reExported.Multipliers) {
			m2 := reExported.Multipliers[i]
			if m.Path != m2.Path || m.CurrentBps != m2.CurrentBps {
				t.Errorf("multiplier[%d] mismatch: {%s,%d} vs {%s,%d}", i, m.Path, m.CurrentBps, m2.Path, m2.CurrentBps)
			}
		}
	}
	if len(exported.Snapshots) != len(reExported.Snapshots) {
		t.Errorf("snapshot count mismatch: %d vs %d", len(exported.Snapshots), len(reExported.Snapshots))
	}
}
