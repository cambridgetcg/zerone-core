package keeper_test

import (
	"context"
	"crypto/sha256"
	"testing"

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

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("zrn", "zrnpub")
	config.SetBech32PrefixForValidator("zrnvaloper", "zrnvaloperpub")
	config.SetBech32PrefixForConsensusNode("zrnvalcons", "zrnvalconspub")
}

// ---------- Mock StakingKeeper ----------

type mockStakingKeeper struct {
	tiers map[string]uint32
}

func newMockStakingKeeper() *mockStakingKeeper {
	return &mockStakingKeeper{tiers: make(map[string]uint32)}
}

func (m *mockStakingKeeper) GetValidatorTier(_ context.Context, addr string) (uint32, error) {
	return m.tiers[addr], nil
}

// ---------- Mock AuthKeeper ----------

type mockAuthKeeper struct {
	registrationBlocks map[string]uint64
}

func newMockAuthKeeper() *mockAuthKeeper {
	return &mockAuthKeeper{registrationBlocks: make(map[string]uint64)}
}

func (m *mockAuthKeeper) GetRegistrationBlock(_ context.Context, addr string) (uint64, error) {
	return m.registrationBlocks[addr], nil
}

// ---------- Mock BankKeeper ----------

type mockBankKeeper struct {
	moduleBalances map[string]map[string]int64
	balances       map[string]map[string]int64
}

func newMockBankKeeper() *mockBankKeeper {
	return &mockBankKeeper{
		moduleBalances: make(map[string]map[string]int64),
		balances:       make(map[string]map[string]int64),
	}
}

func (m *mockBankKeeper) setModuleBalance(moduleName, denom string, amount int64) {
	if m.moduleBalances[moduleName] == nil {
		m.moduleBalances[moduleName] = make(map[string]int64)
	}
	m.moduleBalances[moduleName][denom] = amount
}

func (m *mockBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	to := recipientAddr.String()
	for _, coin := range amt {
		if m.moduleBalances[senderModule] == nil {
			m.moduleBalances[senderModule] = make(map[string]int64)
		}
		if m.balances[to] == nil {
			m.balances[to] = make(map[string]int64)
		}
		m.moduleBalances[senderModule][coin.Denom] -= coin.Amount.Int64()
		m.balances[to][coin.Denom] += coin.Amount.Int64()
	}
	return nil
}

// ---------- Test Addresses ----------

func testAddr(name string) string {
	h := sha256.Sum256([]byte("test_seed:" + name))
	return sdk.AccAddress(h[:20]).String()
}

// ---------- Test Setup ----------

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context, *mockStakingKeeper, *mockAuthKeeper, *mockBankKeeper) {
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

	mockSK := newMockStakingKeeper()
	mockAK := newMockAuthKeeper()
	mockBK := newMockBankKeeper()

	storeService := runtime.NewKVStoreService(storeKey)
	k := keeper.NewKeeper(storeService, cdc, "zrn1authority", mockSK, mockAK, mockBK)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 1000, ChainID: "zerone-test-1"}, false, log.NewNopLogger())

	return k, ctx, mockSK, mockAK, mockBK
}

func setupMsgServer(t *testing.T) (types.MsgServer, keeper.Keeper, sdk.Context, *mockStakingKeeper, *mockAuthKeeper, *mockBankKeeper) {
	t.Helper()
	k, ctx, sk, ak, bk := setupKeeper(t)
	return keeper.NewMsgServerImpl(k), k, ctx, sk, ak, bk
}

// ---------- Test 1: Create Pot (authority-gated) ----------

func TestCreatePot(t *testing.T) {
	msgSrv, k, ctx, _, _, _ := setupMsgServer(t)

	resp, err := msgSrv.CreatePot(ctx, &types.MsgCreatePot{
		Authority:   "zrn1authority",
		Name:        "Test Pot",
		TotalAmount: "10000000",
		Schedule: &types.VestingSchedule{
			StartBlock:   500,
			EndBlock:     2000,
			CliffBlocks:  100,
			PeriodBlocks: 10,
		},
		Eligibility: &types.EligibilityCriteria{
			MinStakingTier: 1,
		},
	})
	if err != nil {
		t.Fatalf("CreatePot failed: %v", err)
	}
	if resp.PotId == "" {
		t.Fatal("expected non-empty pot ID")
	}

	pot, found := k.GetPot(ctx, resp.PotId)
	if !found {
		t.Fatal("pot not found after creation")
	}
	if pot.Name != "Test Pot" {
		t.Errorf("expected name 'Test Pot', got %s", pot.Name)
	}
	if pot.TotalAmount != "10000000" {
		t.Errorf("expected total_amount 10000000, got %s", pot.TotalAmount)
	}
	if pot.ClaimedAmount != "0" {
		t.Errorf("expected claimed_amount '0', got %s", pot.ClaimedAmount)
	}
	if pot.Status != types.PotStatus_POT_STATUS_ACTIVE {
		t.Errorf("expected ACTIVE status, got %s", pot.Status.String())
	}
}

// ---------- Test 2: Create pot unauthorized ----------

func TestCreatePotUnauthorized(t *testing.T) {
	msgSrv, _, ctx, _, _, _ := setupMsgServer(t)

	_, err := msgSrv.CreatePot(ctx, &types.MsgCreatePot{
		Authority:   testAddr("not-authority"),
		Name:        "Bad Pot",
		TotalAmount: "10000000",
		Schedule: &types.VestingSchedule{
			StartBlock: 500,
			EndBlock:   2000,
		},
	})
	if err == nil {
		t.Fatal("expected error for unauthorized CreatePot")
	}
}

// ---------- Test 3: Max active pots limit ----------

func TestMaxActivePotsLimit(t *testing.T) {
	msgSrv, k, ctx, _, _, _ := setupMsgServer(t)

	// Set max to 2
	k.SetParams(ctx, &types.Params{
		MaxPotsActive:  2,
		MinClaimAmount: "1000",
	})

	for i := 0; i < 2; i++ {
		_, err := msgSrv.CreatePot(ctx, &types.MsgCreatePot{
			Authority:   "zrn1authority",
			Name:        "Pot",
			TotalAmount: "10000000",
			Schedule: &types.VestingSchedule{
				StartBlock: 500,
				EndBlock:   2000,
			},
		})
		if err != nil {
			t.Fatalf("CreatePot %d failed: %v", i, err)
		}
	}

	// 3rd should fail
	_, err := msgSrv.CreatePot(ctx, &types.MsgCreatePot{
		Authority:   "zrn1authority",
		Name:        "Overflow Pot",
		TotalAmount: "10000000",
		Schedule: &types.VestingSchedule{
			StartBlock: 500,
			EndBlock:   2000,
		},
	})
	if err == nil {
		t.Fatal("expected error for exceeding max active pots")
	}
}

// ---------- Test 4: Claim with vesting ----------

func TestClaimWithVesting(t *testing.T) {
	msgSrv, k, ctx, sk, ak, bk := setupMsgServer(t)

	claimant := testAddr("claimant1")
	sk.tiers[claimant] = 2
	ak.registrationBlocks[claimant] = 10 // registered early
	bk.setModuleBalance(types.ModuleName, "uzrn", 10_000_000)

	// Create pot: start=500, end=2000, cliff=100, total=10000000
	resp, _ := msgSrv.CreatePot(ctx, &types.MsgCreatePot{
		Authority:   "zrn1authority",
		Name:        "Vesting Pot",
		TotalAmount: "10000000",
		Schedule: &types.VestingSchedule{
			StartBlock:   500,
			EndBlock:     2000,
			CliffBlocks:  100,
			PeriodBlocks: 10,
		},
		Eligibility: &types.EligibilityCriteria{
			MinStakingTier:     1,
			MinRegistrationAge: 100,
		},
	})

	// At block 1000: elapsed=500, totalDuration=1500
	// vested = 10000000 * 500 / 1500 = 3333333
	claimResp, err := msgSrv.Claim(ctx, &types.MsgClaim{
		Claimant: claimant,
		PotId:    resp.PotId,
	})
	if err != nil {
		t.Fatalf("Claim failed: %v", err)
	}
	if claimResp.Amount != "3333333" {
		t.Errorf("expected claimable 3333333, got %s", claimResp.Amount)
	}

	// Verify claim recorded
	claim, found := k.GetClaim(ctx, resp.PotId, claimant)
	if !found {
		t.Fatal("claim not found")
	}
	if claim.Amount != "3333333" {
		t.Errorf("expected claim amount 3333333, got %s", claim.Amount)
	}

	// Verify pot updated
	pot, _ := k.GetPot(ctx, resp.PotId)
	if pot.ClaimedAmount != "3333333" {
		t.Errorf("expected pot claimed_amount 3333333, got %s", pot.ClaimedAmount)
	}

	// Verify bank transfer
	if bk.balances[claimant]["uzrn"] != 3333333 {
		t.Errorf("expected claimant balance 3333333, got %d", bk.balances[claimant]["uzrn"])
	}
}

// ---------- Test 5: Claim before cliff fails ----------

func TestClaimBeforeCliff(t *testing.T) {
	msgSrv, _, ctx, sk, ak, bk := setupMsgServer(t)

	claimant := testAddr("claimant1")
	sk.tiers[claimant] = 2
	ak.registrationBlocks[claimant] = 10
	bk.setModuleBalance(types.ModuleName, "uzrn", 10_000_000)

	// Create pot: start=500, cliff=600 blocks → cliff ends at 1100
	resp, _ := msgSrv.CreatePot(ctx, &types.MsgCreatePot{
		Authority:   "zrn1authority",
		Name:        "Cliff Pot",
		TotalAmount: "10000000",
		Schedule: &types.VestingSchedule{
			StartBlock:  500,
			EndBlock:    2000,
			CliffBlocks: 600,
		},
		Eligibility: &types.EligibilityCriteria{
			MinStakingTier:     1,
			MinRegistrationAge: 100,
		},
	})

	// At block 1000: cliff at 500+600=1100, current=1000 < 1100 → nothing vested
	_, err := msgSrv.Claim(ctx, &types.MsgClaim{
		Claimant: claimant,
		PotId:    resp.PotId,
	})
	if err == nil {
		t.Fatal("expected error for claim before cliff")
	}
}

// ---------- Test 6: Ineligible claimant (whitelist + tier) ----------

func TestClaimIneligible(t *testing.T) {
	msgSrv, _, ctx, sk, ak, bk := setupMsgServer(t)

	claimant := testAddr("claimant1")
	outsider := testAddr("outsider")
	sk.tiers[claimant] = 2
	sk.tiers[outsider] = 0 // low tier
	ak.registrationBlocks[claimant] = 10
	ak.registrationBlocks[outsider] = 10
	bk.setModuleBalance(types.ModuleName, "uzrn", 10_000_000)

	// Pot with whitelist
	resp, _ := msgSrv.CreatePot(ctx, &types.MsgCreatePot{
		Authority:   "zrn1authority",
		Name:        "Whitelist Pot",
		TotalAmount: "10000000",
		Schedule: &types.VestingSchedule{
			StartBlock: 500,
			EndBlock:   2000,
		},
		Eligibility: &types.EligibilityCriteria{
			Whitelist: []string{claimant},
		},
	})

	// Outsider (not in whitelist) should fail
	_, err := msgSrv.Claim(ctx, &types.MsgClaim{
		Claimant: outsider,
		PotId:    resp.PotId,
	})
	if err == nil {
		t.Fatal("expected error for non-whitelisted claimant")
	}

	// Pot with tier requirement
	resp2, _ := msgSrv.CreatePot(ctx, &types.MsgCreatePot{
		Authority:   "zrn1authority",
		Name:        "Tier Pot",
		TotalAmount: "10000000",
		Schedule: &types.VestingSchedule{
			StartBlock: 500,
			EndBlock:   2000,
		},
		Eligibility: &types.EligibilityCriteria{
			MinStakingTier: 2,
		},
	})

	// Low-tier outsider should fail
	_, err = msgSrv.Claim(ctx, &types.MsgClaim{
		Claimant: outsider,
		PotId:    resp2.PotId,
	})
	if err == nil {
		t.Fatal("expected error for low-tier claimant")
	}
}

// ---------- Test 7: Double claim rejected ----------

func TestDoubleClaim(t *testing.T) {
	msgSrv, _, ctx, sk, ak, bk := setupMsgServer(t)

	claimant := testAddr("claimant1")
	sk.tiers[claimant] = 2
	ak.registrationBlocks[claimant] = 10
	bk.setModuleBalance(types.ModuleName, "uzrn", 10_000_000)

	resp, _ := msgSrv.CreatePot(ctx, &types.MsgCreatePot{
		Authority:   "zrn1authority",
		Name:        "Pot",
		TotalAmount: "10000000",
		Schedule: &types.VestingSchedule{
			StartBlock:   500,
			EndBlock:     2000,
			CliffBlocks:  0,
			PeriodBlocks: 10,
		},
		Eligibility: &types.EligibilityCriteria{
			MinStakingTier:     1,
			MinRegistrationAge: 100,
		},
	})

	// First claim succeeds
	_, err := msgSrv.Claim(ctx, &types.MsgClaim{
		Claimant: claimant,
		PotId:    resp.PotId,
	})
	if err != nil {
		t.Fatalf("first Claim failed: %v", err)
	}

	// Second claim should fail
	_, err = msgSrv.Claim(ctx, &types.MsgClaim{
		Claimant: claimant,
		PotId:    resp.PotId,
	})
	if err == nil {
		t.Fatal("expected error for double claim")
	}
}

// ---------- Test 8: Genesis round-trip ----------

func TestGenesisRoundTrip(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	genState := &types.GenesisState{
		Params: &types.Params{
			MaxPotsActive:  5,
			MinClaimAmount: "5000",
		},
		Pots: []*types.ClaimingPot{
			{
				Id:          "gen-pot-1",
				Name:        "Genesis Pot",
				TotalAmount: "50000000",
				ClaimedAmount: "10000000",
				Schedule: &types.VestingSchedule{
					StartBlock: 100,
					EndBlock:   5000,
					CliffBlocks: 200,
				},
				Eligibility: &types.EligibilityCriteria{
					MinStakingTier: 1,
				},
				CreatedAtBlock: 100,
				Status:         types.PotStatus_POT_STATUS_ACTIVE,
			},
		},
		Claims: []*types.Claim{
			{
				PotId:     "gen-pot-1",
				Claimant:  testAddr("claimant1"),
				Amount:    "10000000",
				ClaimedAt: 500,
			},
		},
	}

	k.InitGenesis(ctx, genState)

	exported := k.ExportGenesis(ctx)

	// Check params
	if exported.Params.MaxPotsActive != 5 {
		t.Errorf("expected max_pots_active 5, got %d", exported.Params.MaxPotsActive)
	}
	if exported.Params.MinClaimAmount != "5000" {
		t.Errorf("expected min_claim_amount '5000', got %s", exported.Params.MinClaimAmount)
	}

	// Check pots
	if len(exported.Pots) != 1 {
		t.Fatalf("expected 1 pot, got %d", len(exported.Pots))
	}
	if exported.Pots[0].Id != "gen-pot-1" {
		t.Errorf("expected pot ID gen-pot-1, got %s", exported.Pots[0].Id)
	}
	if exported.Pots[0].ClaimedAmount != "10000000" {
		t.Errorf("expected claimed_amount 10000000, got %s", exported.Pots[0].ClaimedAmount)
	}

	// Check claims
	if len(exported.Claims) != 1 {
		t.Fatalf("expected 1 claim, got %d", len(exported.Claims))
	}
	if exported.Claims[0].Amount != "10000000" {
		t.Errorf("expected claim amount 10000000, got %s", exported.Claims[0].Amount)
	}
}

// ---------- Test: Pot expiry (BeginBlocker) ----------

func TestPotExpiry(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	// Create an active pot that should expire
	k.SetPot(ctx, &types.ClaimingPot{
		Id:          "expiring-pot",
		Name:        "Expiring",
		TotalAmount: "1000000",
		ClaimedAmount: "0",
		Schedule: &types.VestingSchedule{
			StartBlock: 100,
			EndBlock:   500,
		},
		Status: types.PotStatus_POT_STATUS_ACTIVE,
	})

	// Not yet expired (block 400)
	k.ProcessPotExpiry(ctx, 400)
	pot, _ := k.GetPot(ctx, "expiring-pot")
	if pot.Status != types.PotStatus_POT_STATUS_ACTIVE {
		t.Errorf("expected ACTIVE at block 400, got %s", pot.Status.String())
	}

	// Process at block 500 → should expire
	k.ProcessPotExpiry(ctx, 500)
	pot, _ = k.GetPot(ctx, "expiring-pot")
	if pot.Status != types.PotStatus_POT_STATUS_EXPIRED {
		t.Errorf("expected EXPIRED at block 500, got %s", pot.Status.String())
	}
}

// ---------- Test: Default genesis validates ----------

func TestDefaultGenesisValidation(t *testing.T) {
	gs := types.DefaultGenesis()
	if err := gs.Validate(); err != nil {
		t.Errorf("default genesis should be valid: %v", err)
	}
}

// ---------- Test: UpdatePotParams ----------

func TestUpdatePotParams(t *testing.T) {
	msgSrv, k, ctx, _, _, _ := setupMsgServer(t)

	_, err := msgSrv.UpdatePotParams(ctx, &types.MsgUpdatePotParams{
		Authority: "zrn1authority",
		Params: &types.Params{
			MaxPotsActive:  20,
			MinClaimAmount: "5000",
		},
	})
	if err != nil {
		t.Fatalf("UpdatePotParams failed: %v", err)
	}

	params := k.GetParams(ctx)
	if params.MaxPotsActive != 20 {
		t.Errorf("expected max_pots_active 20, got %d", params.MaxPotsActive)
	}

	// Unauthorized
	_, err = msgSrv.UpdatePotParams(ctx, &types.MsgUpdatePotParams{
		Authority: testAddr("not-authority"),
		Params:    types.DefaultParams(),
	})
	if err == nil {
		t.Fatal("expected error for unauthorized UpdatePotParams")
	}
}

// ---------- Test: Claim from inactive pot fails ----------

func TestClaimFromInactivePot(t *testing.T) {
	msgSrv, k, ctx, sk, ak, bk := setupMsgServer(t)

	claimant := testAddr("claimant1")
	sk.tiers[claimant] = 2
	ak.registrationBlocks[claimant] = 10
	bk.setModuleBalance(types.ModuleName, "uzrn", 10_000_000)

	// Create and expire a pot
	k.SetPot(ctx, &types.ClaimingPot{
		Id:          "expired-pot",
		Name:        "Expired",
		TotalAmount: "10000000",
		ClaimedAmount: "0",
		Schedule: &types.VestingSchedule{
			StartBlock: 500,
			EndBlock:   2000,
		},
		Status: types.PotStatus_POT_STATUS_EXPIRED,
	})

	_, err := msgSrv.Claim(ctx, &types.MsgClaim{
		Claimant: claimant,
		PotId:    "expired-pot",
	})
	if err == nil {
		t.Fatal("expected error for claim from expired pot")
	}
}

// ---------- Test: CalculateClaimable math ----------

func TestCalculateClaimable(t *testing.T) {
	// Before start
	pot := &types.ClaimingPot{
		TotalAmount:   "10000000",
		ClaimedAmount: "0",
		Schedule: &types.VestingSchedule{
			StartBlock:  1000,
			EndBlock:    2000,
			CliffBlocks: 100,
		},
	}

	result := keeper.CalculateClaimable(pot, 500) // before start
	if result.Int64() != 0 {
		t.Errorf("before start: expected 0, got %d", result.Int64())
	}

	result = keeper.CalculateClaimable(pot, 1050) // before cliff (start+cliff = 1100)
	if result.Int64() != 0 {
		t.Errorf("before cliff: expected 0, got %d", result.Int64())
	}

	// At midpoint: elapsed=500, duration=1000 → vested = 10000000*500/1000 = 5000000
	result = keeper.CalculateClaimable(pot, 1500)
	if result.Int64() != 5000000 {
		t.Errorf("at midpoint: expected 5000000, got %d", result.Int64())
	}

	// After end: fully vested
	result = keeper.CalculateClaimable(pot, 3000)
	if result.Int64() != 10000000 {
		t.Errorf("after end: expected 10000000, got %d", result.Int64())
	}

	// With some already claimed
	pot.ClaimedAmount = "3000000"
	result = keeper.CalculateClaimable(pot, 1500) // vested=5000000 - claimed=3000000 = 2000000
	if result.Int64() != 2000000 {
		t.Errorf("with claimed: expected 2000000, got %d", result.Int64())
	}

	// After end with some claimed
	result = keeper.CalculateClaimable(pot, 3000) // total=10000000 - claimed=3000000 = 7000000
	if result.Int64() != 7000000 {
		t.Errorf("after end with claimed: expected 7000000, got %d", result.Int64())
	}
}
