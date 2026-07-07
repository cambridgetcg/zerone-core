package keeper_test

// Drill: prove ProcessPotExpiry is O(active pots), not O(all pots ever)
// (commit de48ab2). Seeds stores with wildly different TOTAL pot counts but
// controlled ACTIVE counts, then times steady-state ProcessPotExpiry calls.
//
// PASS criteria:
//  1. FLAT in total: 100,000 total pots (1000x of 100) with the same 10
//     active pots must not cost meaningfully more per call.
//  2. SCALES with active: 10,000 active (1000x of 10) at the same 100,000
//     total must cost much more per call.

import (
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/zerone-chain/zerone/x/claiming_pot/keeper"
	"github.com/zerone-chain/zerone/x/claiming_pot/types"
)

// setupScaleKeeper mirrors setupKeeperFull but takes testing.TB and also
// returns the CommitMultiStore so seeded state can be committed — on a real
// node, pots from prior blocks are COMMITTED state, and IAVL's iterator over
// uncommitted writes merges the whole dirty set (an O(total) test-harness
// artifact that BeginBlock never pays).
func setupScaleKeeper(tb testing.TB) (keeper.Keeper, sdk.Context, storetypes.CommitMultiStore) {
	tb.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		tb.Fatalf("failed to load latest version: %v", err)
	}

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	mockBK := newMockBankKeeper()
	storeService := runtime.NewKVStoreService(storeKey)
	k := keeper.NewKeeper(storeService, cdc, "zrn1authority", newMockStakingKeeper(), newMockAuthKeeper(), mockBK, newMockVestingRewardsKeeper(mockBK))

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 1000, ChainID: "zerone-test-1"}, false, log.NewNopLogger())
	return k, ctx, stateStore
}

// seedPots writes `total` pots: the first `active` are ACTIVE with a far
// future EndBlock (never expire during measurement — steady-state scan),
// the rest are DEPLETED (terminal, kept in state by design).
func seedPots(k keeper.Keeper, ctx sdk.Context, total, active int) {
	for i := 0; i < total; i++ {
		status := types.PotStatus_POT_STATUS_DEPLETED
		if i < active {
			status = types.PotStatus_POT_STATUS_ACTIVE
		}
		k.SetPot(ctx, &types.ClaimingPot{
			Id:            fmt.Sprintf("scale-pot-%07d", i),
			TotalAmount:   "1000000",
			ClaimedAmount: "0",
			Schedule: &types.VestingSchedule{
				StartBlock: 1,
				EndBlock:   10_000_000, // never expires at currentBlock=1000
			},
			Status: status,
		})
	}
}

// timeExpiry returns the average wall time of one ProcessPotExpiry call.
func timeExpiry(k keeper.Keeper, ctx sdk.Context, iters int) time.Duration {
	// Warm-up (cache effects, one-time IAVL fast-node work).
	k.ProcessPotExpiry(ctx, 1000)
	start := time.Now()
	for i := 0; i < iters; i++ {
		k.ProcessPotExpiry(ctx, 1000)
	}
	return time.Since(start) / time.Duration(iters)
}

func TestProcessPotExpiry_ScalesWithActiveNotTotal(t *testing.T) {
	const iters = 200

	// Scenario A: 100 total, 10 active (90% terminal).
	kA, ctxA, cmsA := setupScaleKeeper(t)
	seedPots(kA, ctxA, 100, 10)
	cmsA.Commit() // pots from prior blocks are committed state on a real node
	if got := kA.CountActivePots(ctxA); got != 10 {
		t.Fatalf("scenario A: want 10 active, got %d", got)
	}
	perCallA := timeExpiry(kA, ctxA, iters)

	// Scenario B: 100,000 total (1000x), still 10 active.
	kB, ctxB, cmsB := setupScaleKeeper(t)
	seedPots(kB, ctxB, 100_000, 10)
	cmsB.Commit()
	if got := kB.CountActivePots(ctxB); got != 10 {
		t.Fatalf("scenario B: want 10 active, got %d", got)
	}
	perCallB := timeExpiry(kB, ctxB, iters)

	// Scenario C (control): 100,000 total, 10,000 active (1000x active of A/B).
	kC, ctxC, cmsC := setupScaleKeeper(t)
	seedPots(kC, ctxC, 100_000, 10_000)
	cmsC.Commit()
	if got := kC.CountActivePots(ctxC); got != 10_000 {
		t.Fatalf("scenario C: want 10000 active, got %d", got)
	}
	perCallC := timeExpiry(kC, ctxC, 20)

	flatRatio := float64(perCallB) / float64(perCallA)
	activeRatio := float64(perCallC) / float64(perCallB)

	t.Logf("A:     100 total /     10 active → %v per ProcessPotExpiry call", perCallA)
	t.Logf("B: 100,000 total /     10 active → %v per ProcessPotExpiry call (%.2fx of A at 1000x total)", perCallB, flatRatio)
	t.Logf("C: 100,000 total / 10,000 active → %v per ProcessPotExpiry call (%.1fx of B at 1000x active)", perCallC, activeRatio)

	// (1) Flat in total-pot count: 1000x more terminal pots must not blow up
	// the per-block cost. Allow generous slack for IAVL O(log n) seeks/noise.
	if flatRatio > 5.0 {
		t.Errorf("NOT O(active): 1000x total pots made ProcessPotExpiry %.2fx slower (want <= 5x)", flatRatio)
	}

	// (2) Cost tracks the ACTIVE count: 1000x more active pots at identical
	// total must cost far more, proving active count is the real driver.
	if activeRatio < 20.0 {
		t.Errorf("cost did not track active count: 1000x active only %.2fx slower (want >= 20x)", activeRatio)
	}
}
