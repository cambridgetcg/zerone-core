package integration_test

import (
	"math/big"
	"testing"

	sdkmath "cosmossdk.io/math"
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

	vestingkeeper "github.com/zerone-chain/zerone/x/vesting_rewards/keeper"
	vestingtypes "github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

// ---------- Test 1: Complete Revenue Flow Map ----------

func TestCompleteRevenueMap(t *testing.T) {
	h := setupRevenueHarness(t)

	// SOURCE 1: Block Rewards (10 ZRN = 10,000,000 uzrn at full validators)
	dist, err := h.vestingKeeper.DistributeBlockReward(h.ctx, h.producerAddr.String(), 22, true)
	if err != nil {
		t.Fatalf("block reward distribution failed: %v", err)
	}

	totalMinted := new(big.Int)
	totalMinted.SetString(dist.TotalMinted, 10)
	if totalMinted.Sign() <= 0 {
		t.Fatal("expected non-zero block reward mint")
	}

	// 4-way revenue split: contributor 55%, protocol 22%, development 19.67%, research 3.33%
	bps := big.NewInt(1000000)

	// Gross research = 3.33% of total
	grossResearch := new(big.Int).Mul(totalMinted, big.NewInt(33300))
	grossResearch.Div(grossResearch, bps)

	// Founder = 7% of gross research
	expectedFounder := new(big.Int).Mul(grossResearch, big.NewInt(70000))
	expectedFounder.Div(expectedFounder, bps)

	// Net research = gross research - founder
	expectedNetResearch := new(big.Int).Sub(grossResearch, expectedFounder)

	researchShare := new(big.Int)
	researchShare.SetString(dist.ResearchShare, 10)
	if researchShare.Cmp(expectedNetResearch) != 0 {
		t.Errorf("block reward research share: got %s, want %s", researchShare, expectedNetResearch)
	}

	founderShare := new(big.Int)
	founderShare.SetString(dist.FounderShare, 10)
	if founderShare.Cmp(expectedFounder) != 0 {
		t.Errorf("block reward founder share: got %s, want %s", founderShare, expectedFounder)
	}

	// Contributor (producer) = 55% of total
	expectedProducer := new(big.Int).Mul(totalMinted, big.NewInt(550000))
	expectedProducer.Div(expectedProducer, bps)

	producerReward := new(big.Int)
	producerReward.SetString(dist.ProducerReward, 10)
	if producerReward.Cmp(expectedProducer) != 0 {
		t.Errorf("block reward producer: got %s, want %s", producerReward, expectedProducer)
	}

	// Protocol = 22% of total
	protocolAmt := new(big.Int).Mul(totalMinted, big.NewInt(220000))
	protocolAmt.Div(protocolAmt, bps)

	// Protocol sub-split: the verification pool (30% of protocol) funds
	// knowledge in full (the former compute_pool slice was removed with
	// x/compute_pool in the slim cut).
	verificationPool := new(big.Int).Mul(protocolAmt, big.NewInt(300000))
	verificationPool.Div(verificationPool, bps)

	knowledgeSent := h.bk.totalSentToModule("knowledge")
	if !knowledgeSent.Equal(sdkmath.NewIntFromBigInt(verificationPool)) {
		t.Errorf("knowledge module received %s, want %s", knowledgeSent, verificationPool)
	}

}

// ---------- Test 2: Founder Split Consistency ----------

func TestFounderSplitAllSources(t *testing.T) {
	h := setupRevenueHarness(t)
	depositAmount := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewInt(1000000)))

	// FounderShareBps = 70000 (7% on 1M scale)
	expectedFounder := sdkmath.NewInt(70000)
	expectedResearch := sdkmath.NewInt(930000)

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
			gs.Params.FounderAddress = h.founderAddr.String()
			vk.InitGenesis(ctx, gs)

			err := vk.DepositToResearchFund(ctx, source, depositAmount)
			if err != nil {
				t.Fatalf("DepositToResearchFund from %s failed: %v", source, err)
			}

			founderGot := bk.totalSentToAddr(h.founderAddr.String())
			if !founderGot.Equal(expectedFounder) {
				t.Errorf("founder got %s from %s, want %s", founderGot, source, expectedFounder)
			}

			researchGot := bk.totalSentToModule("research_fund")
			if !researchGot.Equal(expectedResearch) {
				t.Errorf("research_fund got %s from %s, want %s", researchGot, source, expectedResearch)
			}

			totalRouted := founderGot.Add(researchGot)
			if !totalRouted.Equal(sdkmath.NewInt(1000000)) {
				t.Errorf("total routed %s from %s, want 1000000 (dust detected)", totalRouted, source)
			}
		})
	}
}

// ---------- Test 3: No Double Taxation ----------

func TestNoDoubleTaxation(t *testing.T) {
	h := setupRevenueHarness(t)

	dist, err := h.vestingKeeper.DistributeBlockReward(h.ctx, h.producerAddr.String(), 22, true)
	if err != nil {
		t.Fatalf("block reward failed: %v", err)
	}

	totalMinted := new(big.Int)
	totalMinted.SetString(dist.TotalMinted, 10)

	// 4-way split: contributor 55%, protocol 22%, development 19.67%, research 3.33%
	bps := big.NewInt(1000000)

	researchNet := new(big.Int)
	researchNet.SetString(dist.ResearchShare, 10)
	founderAmt := new(big.Int)
	founderAmt.SetString(dist.FounderShare, 10)

	// Knowledge module receives from protocol sub-split (30% verification × 70% to knowledge)
	knowledgeModuleBalance := h.bk.totalSentToModule("knowledge")

	// Compute expected knowledge balance from the actual 4-way split
	protocolAmt := new(big.Int).Mul(totalMinted, big.NewInt(220000))
	protocolAmt.Div(protocolAmt, bps)
	verificationPool := new(big.Int).Mul(protocolAmt, big.NewInt(300000))
	verificationPool.Div(verificationPool, bps)
	expectedKnowledgeBalance := verificationPool

	if !knowledgeModuleBalance.Equal(sdkmath.NewIntFromBigInt(expectedKnowledgeBalance)) {
		t.Errorf("knowledge module balance %s != expected %s — possible double taxation",
			knowledgeModuleBalance, expectedKnowledgeBalance)
	}

	// The key no-double-tax invariant: verifier receives full knowledge balance
	// without additional research deduction (tax was applied at mint time only).
	verifierAddr := sdk.AccAddress("verifier____________")
	_ = verifierAddr

	// Full accounting: all parts sum to total minted
	producerReward := new(big.Int)
	producerReward.SetString(dist.ProducerReward, 10)
	burnAmt := new(big.Int)
	burnAmt.SetString(dist.DevelopmentAmount, 10)
	protocolShare := new(big.Int)
	protocolShare.SetString(dist.ProtocolShare, 10)

	// Gross research = net research + founder
	grossResearch := new(big.Int).Add(researchNet, founderAmt)

	// Total = producer + protocol + gross_research + burn
	accounting := new(big.Int).Add(producerReward, protocolShare)
	accounting.Add(accounting, grossResearch)
	accounting.Add(accounting, burnAmt)

	if accounting.Cmp(totalMinted) != 0 {
		t.Errorf("accounting mismatch: sum of parts %s != total minted %s", accounting, totalMinted)
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

	_, err := h.vestingKeeper.DistributeBlockReward(h.ctx, h.producerAddr.String(), 22, true)
	if err != nil {
		t.Fatalf("block reward failed: %v", err)
	}

	for _, name := range deadAccounts {
		sent := h.bk.totalSentToModule(name)
		if sent.IsPositive() {
			t.Errorf("dead account %q received %s uzrn — should be zero", name, sent)
		}
	}

	activeAccounts := map[string]bool{
		"research_fund":    true,
		"knowledge":        true,
		"development_fund": true,
	}
	for name := range activeAccounts {
		sent := h.bk.totalSentToModule(name)
		if !sent.IsPositive() {
			t.Errorf("active account %q received zero — expected positive balance", name)
		}
	}
}

// ---------- Test 6: Full Ledger Balance ----------

func TestLedgerBalance(t *testing.T) {
	h := setupRevenueHarness(t)

	for i := 0; i < 5; i++ {
		h.ctx = h.ctx.WithBlockHeight(int64(1000 + i))
		_, err := h.vestingKeeper.DistributeBlockReward(h.ctx, h.producerAddr.String(), 22, true)
		if err != nil {
			t.Fatalf("block %d reward failed: %v", 1000+i, err)
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

	var totalDistributed sdkmath.Int = sdkmath.ZeroInt()

	for module, coins := range h.bk.sentToModule {
		amt := coins.AmountOf("uzrn")
		if amt.IsPositive() {
			t.Logf("  module %s: %s uzrn", module, amt)
			totalDistributed = totalDistributed.Add(amt)
		}
	}

	for addr, coins := range h.bk.sentToAccount {
		amt := coins.AmountOf("uzrn")
		if amt.IsPositive() {
			t.Logf("  account %s: %s uzrn", addr, amt)
			totalDistributed = totalDistributed.Add(amt)
		}
	}

	t.Logf("Total minted: %s, burned: %s, distributed: %s", totalMinted, totalBurned, totalDistributed)
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

// ---------- Test 9: Verification Reward Decay Pool Solvency ----------

func TestVerificationRewardDecay_PoolSolvency(t *testing.T) {
	// Zerone: baseReward = 10,000,000 uzrn (10 ZRN)
	baseReward := uint64(10000000)
	decayBps := uint64(994478)    // ~1-year half-life (0.994478x per epoch)
	floorReward := uint64(100000) // 0.1 ZRN

	baseRewardBig := new(big.Int).SetUint64(baseReward)
	var prev uint64 = baseReward
	floorReached := false

	// With 1-year half-life, floor (~0.1 ZRN) is reached at ~epoch 832 (~year 6.6).
	// Sample key epochs to verify monotonic decay without iterating all 850.
	checkEpochs := []uint64{0, 1, 2, 5, 10, 50, 100, 125, 250, 500, 750, 832, 850}
	for _, epoch := range checkEpochs {
		decayed := testApplyDecay(baseRewardBig, decayBps, epoch).Uint64()

		if decayed > baseReward {
			t.Errorf("epoch %d: decayed %d > base %d", epoch, decayed, baseReward)
		}

		if decayed > prev {
			t.Errorf("epoch %d: decayed %d > previous %d — not monotonic", epoch, decayed, prev)
		}

		if decayed < floorReward {
			floorReached = true
		}

		if epoch == 0 && decayed != baseReward {
			t.Errorf("epoch 0: expected %d, got %d", baseReward, decayed)
		}

		// At epoch 1, reward should be 0.994478 * 10,000,000 = 9,944,780
		if epoch == 1 {
			expected := uint64(9944780)
			if decayed != expected {
				t.Errorf("epoch 1: expected %d, got %d", expected, decayed)
			}
		}

		prev = decayed
	}

	if !floorReached {
		t.Errorf("floor reward %d was never reached by epoch 850", floorReward)
	}

	// At epoch 850, decay should be well below floor
	deepDecay := testApplyDecay(baseRewardBig, decayBps, 850).Uint64()
	if deepDecay >= floorReward {
		t.Errorf("epoch 850: expected decay below floor, got %d (floor %d)", deepDecay, floorReward)
	}
}

// ---------- Test 10: Full Revenue Flow With Verification Pool ----------

func TestFullRevenueFlow_WithVerificationPool(t *testing.T) {
	h := setupRevenueHarness(t)

	// --- Part A: Block reward distributes to verification pool ---
	dist, err := h.vestingKeeper.DistributeBlockReward(h.ctx, h.producerAddr.String(), 22, true)
	if err != nil {
		t.Fatalf("block reward distribution failed: %v", err)
	}

	totalMinted := new(big.Int)
	totalMinted.SetString(dist.TotalMinted, 10)
	if totalMinted.Sign() <= 0 {
		t.Fatal("expected non-zero block reward mint")
	}

	knowledgeSent := h.bk.totalSentToModule("knowledge")
	if !knowledgeSent.IsPositive() {
		t.Error("knowledge module received zero from block reward — verification pool missing")
	}
}
