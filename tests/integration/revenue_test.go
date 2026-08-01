package integration_test

import (
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

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	vestingkeeper "github.com/zerone-chain/zerone/x/vesting_rewards/keeper"
	vestingtypes "github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

// ---------- Test 1: Complete Revenue Flow Map ----------

func TestCompleteRevenueMap(t *testing.T) {
	h := setupRevenueHarness(t)

	// Revenue earned by independently witnessed work still uses the transparent
	// four-way router. It is not minted merely because a transaction appeared.
	routing, err := h.vestingKeeper.DistributeRevenue(
		h.ctx,
		vestingtypes.SourceVerification,
		"1000000",
		h.producerAddr.String(),
		"fact-1",
	)
	if err != nil {
		t.Fatalf("revenue routing failed: %v", err)
	}

	want := map[string]string{
		"contributor":  "550000",
		"protocol":     "220000",
		"research":     "33300",
		"development":  "196700",
		"founder":      "0",
		"verification": "66000",
	}
	got := map[string]string{
		"contributor":  routing.ContributorShare,
		"protocol":     routing.ProtocolShare,
		"research":     routing.ResearchShare,
		"development":  routing.DevelopmentAmount,
		"founder":      routing.FounderShare,
		"verification": routing.VerificationPool,
	}
	for part, expected := range want {
		if got[part] != expected {
			t.Errorf("%s share: got %s, want %s", part, got[part], expected)
		}
	}

	// The compatibility block-reward entry point is inert under v2 params even
	// when the proposer supplies a transaction-bearing block.
	dist, err := h.vestingKeeper.DistributeBlockReward(h.ctx, h.producerAddr.String(), 22, true)
	if err != nil {
		t.Fatalf("retired block reward path returned an error: %v", err)
	}
	if dist.TotalMinted != "0" || dist.ProducerReward != "0" || dist.ResearchShare != "0" ||
		dist.DevelopmentAmount != "0" || dist.ProtocolShare != "0" {
		t.Fatalf("transaction presence produced a non-zero distribution: %+v", dist)
	}
	if !h.bk.totalMinted().IsZero() {
		t.Fatalf("transaction presence minted %s uzrn", h.bk.totalMinted())
	}
}

// ---------- Test 2: Full Research Routing Across Sources ----------

func TestResearchDepositsRemainWholeAcrossSources(t *testing.T) {
	h := setupRevenueHarness(t)
	depositAmount := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewInt(1000000)))

	expectedResearch := sdkmath.NewInt(1000000)

	sources := []string{
		"vesting_rewards",
		"knowledge",
	}

	for _, source := range sources {
		t.Run(source, func(t *testing.T) {
			bk := newMockBankKeeper()
			vestingStoreKey := storetypes.NewKVStoreKey(vestingtypes.StoreKey)
			db := dbm.NewMemDB()
			stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
			stateStore.MountStoreWithDB(vestingStoreKey, storetypes.StoreTypeIAVL, db)
			if err := stateStore.LoadLatestVersion(); err != nil {
				t.Fatalf("failed to load: %v", err)
			}
			registry := codectypes.NewInterfaceRegistry()
			cdc := codec.NewProtoCodec(registry)
			vk := vestingkeeper.NewKeeper(cdc, runtime.NewKVStoreService(vestingStoreKey), bk, nil, "authority")
			ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 1000}, false, log.NewNopLogger())

			gs := vestingtypes.DefaultGenesis()
			vk.InitGenesis(ctx, gs)

			err := vk.DepositToResearchFund(ctx, source, depositAmount)
			if err != nil {
				t.Fatalf("DepositToResearchFund from %s failed: %v", source, err)
			}

			founderGot := bk.totalSentToAddr(h.founderAddr.String())
			if !founderGot.IsZero() {
				t.Errorf("retired founder account got %s from %s", founderGot, source)
			}

			researchGot := bk.totalSentToModule("research_fund")
			if !researchGot.Equal(expectedResearch) {
				t.Errorf("research_fund got %s from %s, want %s", researchGot, source, expectedResearch)
			}

			if !researchGot.Equal(depositAmount.AmountOf("uzrn")) {
				t.Errorf("deposit conservation failed for %s: routed %s, input %s", source, researchGot, depositAmount)
			}
		})
	}
}

// ---------- Test 3: Research Escrow Has One Beneficiary ----------

func TestResearchEscrowHasOneBeneficiary(t *testing.T) {
	h := setupRevenueHarness(t)
	deposit := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewInt(1_000_000)))

	if err := h.vestingKeeper.DepositToResearchFund(h.ctx, "knowledge", deposit); err != nil {
		t.Fatalf("research deposit failed: %v", err)
	}
	if got := h.bk.totalSentToModule(vestingtypes.ModuleName); !got.Equal(sdkmath.NewInt(1_000_000)) {
		t.Errorf("vesting escrow received %s, want 1000000", got)
	}
	if got := h.bk.totalSentToModule(vestingtypes.ResearchFundModuleName); !got.Equal(sdkmath.NewInt(1_000_000)) {
		t.Errorf("research fund received %s, want 1000000", got)
	}
	if got := h.bk.totalSentToAddr(h.founderAddr.String()); !got.IsZero() {
		t.Errorf("retired founder account received %s", got)
	}
	if got := h.bk.totalMinted(); !got.IsZero() {
		t.Errorf("moving existing research revenue unexpectedly minted %s", got)
	}
}

// ---------- Test 5: Dead Accounts Have Zero Balance ----------

func TestDeadAccountsRemoved(t *testing.T) {
	h := setupRevenueHarness(t)

	deadAccounts := []string{
		"treasury_research",
		"treasury_foundation",
		"treasury_community",
		"treasury_developers",
		"treasury_reserve",
	}

	deposit := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewInt(1_000_000)))
	dist, err := h.vestingKeeper.DistributeBlockReward(h.ctx, h.producerAddr.String(), 22, true)
	if err != nil {
		t.Fatalf("retired reward path failed: %v", err)
	}
	if dist.TotalMinted != "0" {
		t.Fatalf("transaction presence minted %s before research deposit", dist.TotalMinted)
	}
	if err := h.vestingKeeper.DepositToResearchFund(h.ctx, vestingtypes.ModuleName, deposit); err != nil {
		t.Fatalf("research deposit failed: %v", err)
	}

	for _, name := range deadAccounts {
		sent := h.bk.totalSentToModule(name)
		if sent.IsPositive() {
			t.Errorf("dead account %q received %s uzrn — should be zero", name, sent)
		}
	}

	if sent := h.bk.totalSentToModule(vestingtypes.ResearchFundModuleName); !sent.IsPositive() {
		t.Errorf("active research fund received zero")
	}
	if sent := h.bk.totalSentToModule("knowledge"); !sent.IsZero() {
		t.Errorf("retired transaction-presence reward sent %s to knowledge", sent)
	}
}

// ---------- Test 6: Full Ledger Balance ----------

func TestTransactionPresenceDoesNotChangeLedgerSupply(t *testing.T) {
	h := setupRevenueHarness(t)

	for i := 0; i < 5; i++ {
		h.ctx = h.ctx.WithBlockHeight(int64(1000 + i))
		dist, err := h.vestingKeeper.DistributeBlockReward(h.ctx, h.producerAddr.String(), uint32(i+1), i%2 == 0)
		if err != nil {
			t.Fatalf("block %d retired reward path failed: %v", 1000+i, err)
		}
		if dist.TotalMinted != "0" {
			t.Errorf("block %d minted %s from transaction presence", 1000+i, dist.TotalMinted)
		}
	}

	totalMinted := h.bk.totalMinted()
	totalBurned := h.bk.totalBurned()
	expectedSupply := totalMinted.Sub(totalBurned)
	actualSupply := h.bk.GetSupply(h.ctx, "uzrn").Amount

	if !actualSupply.Equal(expectedSupply) {
		t.Errorf("supply mismatch: actual %s != minted(%s) - burned(%s) = %s",
			actualSupply, totalMinted, totalBurned, expectedSupply)
	}

	if !totalMinted.IsZero() || !totalBurned.IsZero() || !actualSupply.IsZero() {
		t.Errorf("retired automatic issuance changed ledger: minted=%s burned=%s supply=%s", totalMinted, totalBurned, actualSupply)
	}
	if len(h.bk.sentToAccount) != 0 || len(h.bk.sentToModule) != 0 {
		t.Errorf("retired automatic issuance moved funds: accounts=%v modules=%v", h.bk.sentToAccount, h.bk.sentToModule)
	}
}

// ---------- Test 7: DepositToResearchFund with No Founder ----------

func TestDepositToResearchFund_NoFounder(t *testing.T) {
	bk := newMockBankKeeper()

	vestingStoreKey := storetypes.NewKVStoreKey(vestingtypes.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(vestingStoreKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		t.Fatalf("failed to load: %v", err)
	}
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	vk := vestingkeeper.NewKeeper(cdc, runtime.NewKVStoreService(vestingStoreKey), bk, nil, "authority")
	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 1000}, false, log.NewNopLogger())

	gs := vestingtypes.DefaultGenesis()
	gs.Params.FounderAddress = ""
	vk.InitGenesis(ctx, gs)

	deposit := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewInt(1000000)))
	err := vk.DepositToResearchFund(ctx, "knowledge", deposit)
	if err != nil {
		t.Fatalf("deposit failed: %v", err)
	}

	researchGot := bk.totalSentToModule("research_fund")
	if !researchGot.Equal(sdkmath.NewInt(1000000)) {
		t.Errorf("research_fund got %s, want 1000000 (no founder)", researchGot)
	}

	for addr, coins := range bk.sentToAccount {
		if coins.AmountOf("uzrn").IsPositive() {
			t.Errorf("unexpected transfer to account %s: %s (should be zero with no founder)", addr, coins)
		}
	}
}

// ---------- Test 9: Automatic Reward Compatibility Surface Is Inert ----------

func TestAutomaticRewardCompatibilitySurfaceIsInert(t *testing.T) {
	h := setupRevenueHarness(t)
	params := h.vestingKeeper.GetParams(h.ctx)
	if params.BlockReward != "0" || params.FloorReward != "0" || params.EmptyBlockRewardRate != 0 {
		t.Fatalf("automatic reward fields are not neutral: block=%q floor=%q empty_rate=%d",
			params.BlockReward, params.FloorReward, params.EmptyBlockRewardRate)
	}

	for _, epoch := range []uint64{0, 1, 10, 850, 1_000_000} {
		if got := h.vestingKeeper.GetEpochBlockRewardPool(h.ctx, epoch); got != 0 {
			t.Errorf("epoch %d advertises a retired reward pool of %d", epoch, got)
		}
	}
}

// ---------- Test 10: Real Fee Revenue Still Flows ----------

func TestRealFeeRevenueStillFlows(t *testing.T) {
	h := setupRevenueHarness(t)

	// Route an existing research receipt. Unlike transaction-presence rewards,
	// this transfers already-owned value and cannot alter supply.
	amount := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewInt(33300)))
	if err := h.vestingKeeper.DepositToResearchFund(h.ctx, vestingtypes.ModuleName, amount); err != nil {
		t.Fatalf("research fee deposit failed: %v", err)
	}
	if got := h.bk.totalSentToModule(vestingtypes.ResearchFundModuleName); !got.Equal(sdkmath.NewInt(33300)) {
		t.Errorf("research fund received %s, want 33300", got)
	}
	if got := h.bk.totalSentToAddr(h.founderAddr.String()); !got.IsZero() {
		t.Errorf("retired founder account received %s from real fees", got)
	}
	if got := h.bk.totalMinted(); !got.IsZero() {
		t.Errorf("routing real fees minted %s", got)
	}
}
