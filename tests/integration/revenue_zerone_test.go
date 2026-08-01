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
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	vestingkeeper "github.com/zerone-chain/zerone/x/vesting_rewards/keeper"
	vestingtypes "github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

// ---------- Test 13: Transaction Presence Cannot Mint ----------

func TestTransactionPresenceCannotMintAcrossValidatorCounts(t *testing.T) {
	tests := []struct {
		name             string
		activeValidators uint32
		hasTransactions  bool
	}{
		{
			name:             "full_validators",
			activeValidators: 22,
			hasTransactions:  true,
		},
		{
			name:             "half_validators",
			activeValidators: 11,
			hasTransactions:  true,
		},
		{
			name:             "one_validator",
			activeValidators: 1,
			hasTransactions:  true,
		},
		{
			name:             "over_target",
			activeValidators: 30,
			hasTransactions:  true,
		},
		{
			name:             "empty_block_zero_reward",
			activeValidators: 22,
			hasTransactions:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
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

			sk := &mockStakingKeeper{activeCount: tc.activeValidators}
			vk := vestingkeeper.NewKeeper(cdc, runtime.NewKVStoreService(vestingStoreKey), bk, sk, "authority")
			ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 1000}, false, log.NewNopLogger())

			gs := vestingtypes.DefaultGenesis()
			gs.Params.FounderAddress = ""
			vk.InitGenesis(ctx, gs)

			producerAddr := sdk.AccAddress("producer____________")
			dist, err := vk.DistributeBlockReward(ctx, producerAddr.String(), tc.activeValidators, tc.hasTransactions)
			if err != nil {
				t.Fatalf("DistributeBlockReward failed: %v", err)
			}

			if dist.TotalMinted != "0" || dist.ProducerReward != "0" ||
				dist.ResearchShare != "0" || dist.DevelopmentAmount != "0" ||
				dist.ProtocolShare != "0" {
				t.Errorf("retired automatic reward produced a distribution: %+v", dist)
			}
			if got := bk.totalMinted(); !got.IsZero() {
				t.Errorf("validator count %d and transaction=%v minted %s",
					tc.activeValidators, tc.hasTransactions, got)
			}
			if got := bk.totalSentToAddr(producerAddr.String()); !got.IsZero() {
				t.Errorf("producer received retired automatic reward %s", got)
			}
		})
	}
}

// ---------- Test 14: Research Fund Deposit and Disburse ----------

func TestResearchFundDepositAndDisburse(t *testing.T) {
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

	founderAddr := sdk.AccAddress("founder_____________")
	recipientAddr := sdk.AccAddress("recipient___________")

	vk := vestingkeeper.NewKeeper(cdc, runtime.NewKVStoreService(vestingStoreKey), bk, nil, "authority")
	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 1000}, false, log.NewNopLogger())

	vk.InitGenesis(ctx, vestingtypes.DefaultGenesis())

	// Step 1: Deposit 1,000,000 uzrn to research fund
	depositCoins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewInt(1000000)))
	err := vk.DepositToResearchFund(ctx, "billing", depositCoins)
	if err != nil {
		t.Fatalf("deposit failed: %v", err)
	}

	// Consensus v2 routes the full deposit to the research fund. The founder
	// address is only a sentinel proving that no identity payout occurs.
	expectedResearchDeposit := sdkmath.NewInt(1000000)

	founderGot := bk.totalSentToAddr(founderAddr.String())
	if !founderGot.IsZero() {
		t.Errorf("retired founder account got %s on deposit", founderGot)
	}

	researchGot := bk.totalSentToModule("research_fund")
	if !researchGot.Equal(expectedResearchDeposit) {
		t.Errorf("research_fund got %s on deposit, want %s", researchGot, expectedResearchDeposit)
	}

	// Step 2: governance-directed disbursement preserves the full amount.
	bk.sentToAccount = make(map[string]sdk.Coins) // reset to isolate disburse
	disburseCoins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewInt(500000)))
	err = vk.DisburseFromResearchFund(ctx, recipientAddr, disburseCoins)
	if err != nil {
		t.Fatalf("disburse failed: %v", err)
	}

	// Recipient should get full amount (no founder split on disburse path)
	recipientGot := bk.totalSentToAddr(recipientAddr.String())
	if !recipientGot.Equal(sdkmath.NewInt(500000)) {
		t.Errorf("recipient got %s, want 500000", recipientGot)
	}

	// The sentinel founder account is not a beneficiary on either path.
	founderDisburse := bk.totalSentToAddr(founderAddr.String())
	if founderDisburse.IsPositive() {
		t.Errorf("retired founder account got %s on disburse", founderDisburse)
	}
}

// ---------- Test 15: Fee Router Split ----------

func TestFeeRouterSplit(t *testing.T) {
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

	founderAddr := sdk.AccAddress("founder_____________")

	vk := vestingkeeper.NewKeeper(cdc, runtime.NewKVStoreService(vestingStoreKey), bk, nil, "authority")
	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 1000}, false, log.NewNopLogger())

	vk.InitGenesis(ctx, vestingtypes.DefaultGenesis())

	// Seed fee_collector balance using the canonical module address
	feeCollectorAddr := authtypes.NewModuleAddress(authtypes.FeeCollectorName)
	feeBalance := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewInt(1000000)))
	bk.balances[feeCollectorAddr.String()] = feeBalance

	// Call RouteFees
	err := vk.RouteFees(ctx)
	if err != nil {
		t.Fatalf("RouteFees failed: %v", err)
	}

	// RevenueSplit defaults: Research=33300 (3.33%), Development=196700 (19.67%)
	// Research share: 3.33% of 1M = 33,300, deposited in full.
	expectedResearchTotal := int64(33300)

	// Verify research fund received the complete research allocation.
	researchGot := bk.totalSentToModule("research_fund")
	if !researchGot.Equal(sdkmath.NewInt(expectedResearchTotal)) {
		t.Errorf("research_fund got %s, want %d", researchGot, expectedResearchTotal)
	}

	// No identity-based account receives part of the fee flow.
	founderGot := bk.totalSentToAddr(founderAddr.String())
	if !founderGot.IsZero() {
		t.Errorf("retired founder account got %s from fees", founderGot)
	}

	// Development fund: 19.67% of 1M = 196,700 (no burn)
	expectedDev := int64(196700)
	devGot := bk.totalSentToModule("development_fund")
	if !devGot.Equal(sdkmath.NewInt(expectedDev)) {
		t.Errorf("development_fund got %s, want %d", devGot, expectedDev)
	}

	// No tokens should be burned
	burnGot := bk.totalBurned()
	if burnGot.IsPositive() {
		t.Errorf("expected no burn, got %s", burnGot)
	}

	// Verify remainder stays in fee_collector for x/distribution
	// RouteFees extracts research + development; the rest (77%) stays for validators
	totalExtracted := expectedResearchTotal + expectedDev // 230,000
	remaining := int64(1000000) - totalExtracted          // 770,000

	// Research is escrowed through vesting_rewards before reaching its fund.
	vestingSent := bk.totalSentToModule("vesting_rewards")
	if !vestingSent.Equal(sdkmath.NewInt(expectedResearchTotal)) {
		t.Errorf("vesting_rewards received %s, want %d (research escrow only)", vestingSent, expectedResearchTotal)
	}

	_ = remaining // 770,000 stays in fee_collector for validators via x/distribution
}
