package simulation_test

import (
	"encoding/csv"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

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

	// Blank import sets bech32 prefix to "zrn".
	_ "github.com/zerone-chain/zerone/app"

	vestingkeeper "github.com/zerone-chain/zerone/x/vesting_rewards/keeper"
	vestingtypes "github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

// ============================================================================
// R9-5 — Economic Simulation: 1000-Block Run
//
// Simulates 1000 blocks of chain operation, verifying token conservation,
// full research routing, zero identity payouts, zero transaction-presence
// issuance, and validator compensation from real fees.
// ============================================================================

const (
	simBlocks           = 1000
	simBlocksPerEpoch   = 100 // epoch-invariant interval
	simValidatorCount   = 4
	simAgentCount       = 10
	simInitialFactCount = 50
	simToolCount        = 5
	simSeed             = 42 // deterministic random seed
)

func TestEconomicSimulation(t *testing.T) {
	start := time.Now()
	t.Logf("seed: %d", simSeed)

	// Module accounts participating in the simulation.
	moduleNames := []string{
		vestingtypes.ModuleName,
		vestingtypes.ResearchFundModuleName,
		vestingtypes.KnowledgeModuleName,
		vestingtypes.DevelopmentFundModuleName,
		authtypes.FeeCollectorName,
		"staking",
		"toolbox",
	}

	// ---- Wire up keepers ----
	bank := newSimBankKeeper(moduleNames)
	sk := &simStakingKeeper{activeCount: simValidatorCount}

	vestingStoreKey := storetypes.NewKVStoreKey(vestingtypes.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(vestingStoreKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		t.Fatalf("failed to load store: %v", err)
	}

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	vk := vestingkeeper.NewKeeper(cdc, runtime.NewKVStoreService(vestingStoreKey), bank, sk, "authority")

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 1}, false, log.NewNopLogger())

	// Init vesting genesis with simulation-tuned params.
	gs := vestingtypes.DefaultGenesis()
	gs.Params.BlocksPerRewardEpoch = simBlocksPerEpoch
	vk.InitGenesis(ctx, gs)

	// ---- Seed state ----
	validators := seedValidators(t, bank)
	agents := seedAgents(t, bank)
	tools := seedTools(agents)

	initialSupply := bank.GetSupply(nil, "uzrn").Amount
	t.Logf("initial supply: %s uzrn (%.2f ZRN)", initialSupply, uzrnToZrn(initialSupply))

	state := &SimState{
		bank:                bank,
		vestingKeeper:       vk,
		ctx:                 ctx,
		validators:          validators,
		agents:              agents,
		tools:               tools,
		moduleNames:         moduleNames,
		currentBlockReward:  sdkmath.ZeroInt(),
		totalMinted:         sdkmath.ZeroInt(),
		totalFeesCharged:    sdkmath.ZeroInt(),
		totalFeeResearch:    sdkmath.ZeroInt(),
		totalFeeDevelopment: sdkmath.ZeroInt(),
		totalValidatorFees:  sdkmath.ZeroInt(),
		initialSupply:       initialSupply,
		factsAdded:          simInitialFactCount,
		toolRevenue:         sdkmath.ZeroInt(),
	}

	// ---- Activity generator ----
	agentAddrs := make([]sdk.AccAddress, len(agents))
	for i, a := range agents {
		agentAddrs[i] = a.addr
	}
	validatorAddrs := make([]sdk.AccAddress, len(validators))
	for i, v := range validators {
		validatorAddrs[i] = v.addr
	}
	gen := NewActivityGenerator(simSeed, agentAddrs, validatorAddrs, simToolCount)

	perBlock := PerBlockInvariants()
	epochInv := EpochInvariants()
	finalInv := FinalInvariants()

	// ---- CSV output ----
	if err := os.MkdirAll("/tmp/zerone_sim", 0o755); err != nil {
		t.Fatal(err)
	}
	csvFile, err := os.Create("/tmp/zerone_sim/block_results.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer csvFile.Close()
	csvW := csv.NewWriter(csvFile)
	defer csvW.Flush()
	_ = csvW.Write([]string{
		"block", "epoch", "automatic_block_reward", "automatic_total_minted", "fee_development_routed",
		"supply", "research_fund", "knowledge_pool", "compute_pool",
		"activity_count", "facts_total",
	})

	// ========================================================================
	// BLOCK LOOP
	// ========================================================================
	for height := int64(1); height <= simBlocks; height++ {
		state.currentHeight = height
		state.currentEpoch = int((height - 1) / simBlocksPerEpoch)

		ctx = ctx.WithBlockHeight(height)
		state.ctx = ctx

		// Generate and execute random activity.
		activities := gen.GenerateBlock(height)
		hasTransactions := len(activities) > 0

		for _, act := range activities {
			executeActivity(state, act)
		}

		// ---- Prove transaction presence cannot trigger issuance ----
		producer := validators[int(height)%len(validators)]
		dist, err := vk.DistributeBlockReward(
			ctx,
			producer.addr.String(),
			sk.activeCount,
			hasTransactions,
		)
		if err != nil {
			t.Fatalf("block %d: retired reward compatibility path failed: %v", height, err)
		}
		if dist.TotalMinted != "0" || dist.ProducerReward != "0" || dist.ResearchShare != "0" ||
			dist.DevelopmentAmount != "0" || dist.ProtocolShare != "0" ||
			(dist.FounderShare != "" && dist.FounderShare != "0") {
			t.Fatalf("block %d: transaction presence produced automatic value: %+v", height, dist)
		}
		state.currentBlockReward = sdkmath.ZeroInt()

		// ---- Route accumulated fees ----
		feesToRoute := bank.moduleBalance(authtypes.FeeCollectorName, "uzrn")
		researchBefore := bank.moduleBalance(vestingtypes.ResearchFundModuleName, "uzrn")
		developmentBefore := bank.moduleBalance(vestingtypes.DevelopmentFundModuleName, "uzrn")
		if err := vk.RouteFees(ctx); err != nil {
			t.Fatalf("block %d: RouteFees failed: %v", height, err)
		}

		researchDelta := bank.moduleBalance(vestingtypes.ResearchFundModuleName, "uzrn").Sub(researchBefore)
		developmentDelta := bank.moduleBalance(vestingtypes.DevelopmentFundModuleName, "uzrn").Sub(developmentBefore)
		split := vk.GetRevenueSplit(ctx)
		expectedResearch := feesToRoute.MulRaw(int64(split.ResearchBps)).QuoRaw(1_000_000)
		expectedDevelopment := feesToRoute.MulRaw(int64(split.DevelopmentBps)).QuoRaw(1_000_000)
		if !researchDelta.Equal(expectedResearch) || !developmentDelta.Equal(expectedDevelopment) {
			t.Fatalf("block %d: real fee route mismatch: fees=%s research=%s/%s development=%s/%s",
				height, feesToRoute, researchDelta, expectedResearch, developmentDelta, expectedDevelopment)
		}
		state.totalFeeResearch = state.totalFeeResearch.Add(researchDelta)
		state.totalFeeDevelopment = state.totalFeeDevelopment.Add(developmentDelta)

		// Simulate x/distribution's same-block sweep of the remainder. This keeps
		// prior validator fees from being routed through the research/development
		// split again on subsequent blocks.
		validatorFees := bank.moduleBalance(authtypes.FeeCollectorName, "uzrn")
		if validatorFees.IsPositive() {
			coins := sdk.NewCoins(sdk.NewCoin("uzrn", validatorFees))
			if err := bank.SendCoinsFromModuleToAccount(ctx, authtypes.FeeCollectorName, producer.addr, coins); err != nil {
				t.Fatalf("block %d: validator fee sweep failed: %v", height, err)
			}
			producer.totalEarned = producer.totalEarned.Add(validatorFees)
			producer.feeEarned = producer.feeEarned.Add(validatorFees)
			state.totalValidatorFees = state.totalValidatorFees.Add(validatorFees)
		}

		// ---- Per-block invariants ----
		if err := runInvariants(state, perBlock); err != nil {
			t.Fatal(err)
		}

		// ---- Epoch invariants ----
		if height%simBlocksPerEpoch == 0 {
			if err := runInvariants(state, epochInv); err != nil {
				t.Fatal(err)
			}
			t.Logf("epoch %d (block %d): supply=%s, automatic_mint=%s, fee_development=%s, facts=%d",
				state.currentEpoch, height,
				bank.GetSupply(nil, "uzrn").Amount,
				state.totalMinted, state.totalFeeDevelopment, state.factsAdded)
		}

		// ---- CSV row ----
		_ = csvW.Write([]string{
			fmt.Sprintf("%d", height),
			fmt.Sprintf("%d", state.currentEpoch),
			state.currentBlockReward.String(),
			state.totalMinted.String(),
			state.totalFeeDevelopment.String(),
			bank.GetSupply(nil, "uzrn").Amount.String(),
			bank.moduleBalance("research_fund", "uzrn").String(),
			bank.moduleBalance("knowledge", "uzrn").String(),
			bank.moduleBalance("compute_pool", "uzrn").String(),
			fmt.Sprintf("%d", len(activities)),
			fmt.Sprintf("%d", state.factsAdded),
		})
	}

	// ========================================================================
	// FINAL INVARIANTS
	// ========================================================================
	if err := runInvariants(state, finalInv); err != nil {
		t.Fatal(err)
	}

	// ========================================================================
	// SUMMARY
	// ========================================================================
	elapsed := time.Since(start)
	supply := bank.GetSupply(nil, "uzrn").Amount

	sep := strings.Repeat("=", 90)
	t.Log(sep)
	t.Log("ZERONE — ECONOMIC SIMULATION SUMMARY (1000 blocks)")
	t.Log(sep)
	t.Logf("Duration:        %v", elapsed)
	t.Logf("Initial supply:  %s uzrn (%.2f ZRN)", state.initialSupply, uzrnToZrn(state.initialSupply))
	t.Logf("Automatic mint:  %s uzrn (%.2f ZRN)", state.totalMinted, uzrnToZrn(state.totalMinted))
	t.Logf("Fees charged:    %s uzrn", state.totalFeesCharged)
	t.Logf("Fee research:    %s uzrn", state.totalFeeResearch)
	t.Logf("Fee development: %s uzrn", state.totalFeeDevelopment)
	t.Logf("Validator fees:  %s uzrn", state.totalValidatorFees)
	t.Logf("Final supply:    %s uzrn (%.2f ZRN)", supply, uzrnToZrn(supply))
	t.Logf("Research fund:   %s uzrn", bank.moduleBalance("research_fund", "uzrn"))
	t.Logf("Knowledge pool:  %s uzrn", bank.moduleBalance("knowledge", "uzrn"))
	t.Logf("Compute pool:    %s uzrn", bank.moduleBalance("compute_pool", "uzrn"))
	t.Logf("Dev fund:        %s uzrn", bank.moduleBalance("development_fund", "uzrn"))
	t.Logf("Facts total:     %d (initial: %d, added: %d)", state.factsAdded, simInitialFactCount, state.factsAdded-simInitialFactCount)
	t.Logf("Tool revenue:    %s uzrn", state.toolRevenue)

	t.Log("")
	t.Log("VALIDATORS:")
	for _, v := range validators {
		bal := bank.GetBalance(nil, v.addr, "uzrn").Amount
		t.Logf("  Tier %d: balance=%s, earned=%s, fee_earned=%s, staked=%s",
			v.tier, bal, v.totalEarned, v.feeEarned, v.staked)
	}

	t.Log("")
	t.Log("AGENTS (top 5):")
	for i, a := range agents {
		if i >= 5 {
			break
		}
		bal := bank.GetBalance(nil, a.addr, "uzrn").Amount
		t.Logf("  %s: balance=%s uzrn (%.4f ZRN)", a.name, bal, uzrnToZrn(bal))
	}

	t.Log(sep)
	t.Logf("CSV written to /tmp/zerone_sim/block_results.csv")

	if elapsed > 120*time.Second {
		t.Errorf("simulation exceeded 120s: %v", elapsed)
	}
}

// ============================================================================
// SEEDING
// ============================================================================

func seedValidators(t *testing.T, bank *simBankKeeper) []*simValidator {
	t.Helper()
	validators := []*simValidator{
		{addr: sdk.AccAddress("val_apprentice______"), tier: 0, staked: sdkmath.NewInt(111_000)},
		{addr: sdk.AccAddress("val_verified________"), tier: 1, staked: sdkmath.NewInt(1_110_000)},
		{addr: sdk.AccAddress("val_scholar_________"), tier: 2, staked: sdkmath.NewInt(1_111_000_000)},
		{addr: sdk.AccAddress("val_guardian________0"), tier: 3, staked: sdkmath.NewInt(11_111_000_000)},
	}
	for _, v := range validators {
		v.totalEarned = sdkmath.ZeroInt()
		v.feeEarned = sdkmath.ZeroInt()
		liquid := v.staked.Quo(sdkmath.NewInt(10))
		total := v.staked.Add(liquid)
		coins := sdk.NewCoins(sdk.NewCoin("uzrn", total))
		if err := bank.MintCoins(nil, vestingtypes.ModuleName, coins); err != nil {
			t.Fatal(err)
		}
		if err := bank.SendCoinsFromModuleToAccount(nil, vestingtypes.ModuleName, v.addr, coins); err != nil {
			t.Fatal(err)
		}
	}
	return validators
}

func seedAgents(t *testing.T, bank *simBankKeeper) []*simAgent {
	t.Helper()
	defs := []struct {
		name    string
		balance int64
	}{
		{"whale", 100_000_000},
		{"orca", 50_000_000},
		{"dolphin", 20_000_000},
		{"turtle_1", 10_000_000},
		{"turtle_2", 5_000_000},
		{"fish_1", 1_000_000},
		{"fish_2", 500_000},
		{"shrimp_1", 100_000},
		{"shrimp_2", 10_000},
		{"plankton", 1_000},
	}
	agents := make([]*simAgent, len(defs))
	for i, def := range defs {
		addr := sdk.AccAddress(fmt.Sprintf("agent_%02d_%-10s", i, def.name))
		agents[i] = &simAgent{addr: addr, name: def.name}
		if def.balance > 0 {
			coins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewInt(def.balance)))
			if err := bank.MintCoins(nil, vestingtypes.ModuleName, coins); err != nil {
				t.Fatal(err)
			}
			if err := bank.SendCoinsFromModuleToAccount(nil, vestingtypes.ModuleName, addr, coins); err != nil {
				t.Fatal(err)
			}
		}
	}
	return agents
}

func seedTools(agents []*simAgent) []*simTool {
	tools := make([]*simTool, simToolCount)
	for i := 0; i < simToolCount; i++ {
		tools[i] = &simTool{id: i, creator: agents[i%len(agents)].addr}
	}
	return tools
}

// ============================================================================
// ACTIVITY EXECUTION
// ============================================================================

func executeActivity(s *SimState, act SimActivity) {
	bank := s.bank

	// Deduct gas from actor → fee_collector.
	if act.Gas.IsPositive() {
		gasCoins := sdk.NewCoins(sdk.NewCoin("uzrn", act.Gas))
		if err := bank.SendCoinsFromAccountToModule(nil, act.Actor, authtypes.FeeCollectorName, gasCoins); err != nil {
			return // insufficient funds — skip
		}
		s.totalFeesCharged = s.totalFeesCharged.Add(act.Gas)
	}

	switch act.Type {
	case ActivityKnowledgeClaim:
		if act.Amount.IsPositive() {
			stakeCoins := sdk.NewCoins(sdk.NewCoin("uzrn", act.Amount))
			if err := bank.SendCoinsFromAccountToModule(nil, act.Actor, "knowledge", stakeCoins); err != nil {
				return
			}
		}
		s.factsAdded++

	case ActivityVerification:
		tier := validatorTier(s, act.Actor)
		rewardBase := act.Amount
		switch tier {
		case 0:
			rewardBase = rewardBase.Quo(sdkmath.NewInt(10))
		case 1:
			rewardBase = rewardBase.Quo(sdkmath.NewInt(2))
		case 3:
			rewardBase = rewardBase.Mul(sdkmath.NewInt(2))
		}
		if bank.moduleBalance("knowledge", "uzrn").GTE(rewardBase) {
			rewardCoins := sdk.NewCoins(sdk.NewCoin("uzrn", rewardBase))
			if err := bank.SendCoinsFromModuleToAccount(nil, "knowledge", act.Actor, rewardCoins); err == nil {
				for _, v := range s.validators {
					if v.addr.Equals(act.Actor) {
						v.totalEarned = v.totalEarned.Add(rewardBase)
						break
					}
				}
			}
		}

	case ActivityToolCall:
		if act.Amount.IsPositive() {
			feeCoins := sdk.NewCoins(sdk.NewCoin("uzrn", act.Amount))
			if err := bank.SendCoinsFromAccountToModule(nil, act.Actor, "toolbox", feeCoins); err != nil {
				return
			}
			creatorShare := act.Amount.Mul(sdkmath.NewInt(550_000)).Quo(sdkmath.NewInt(1_000_000))
			if creatorShare.IsPositive() {
				tool := s.tools[act.ToolID%len(s.tools)]
				creatorCoins := sdk.NewCoins(sdk.NewCoin("uzrn", creatorShare))
				_ = bank.SendCoinsFromModuleToAccount(nil, "toolbox", tool.creator, creatorCoins)
				s.toolRevenue = s.toolRevenue.Add(creatorShare)
			}
		}

	case ActivityTransfer:
		if act.Amount.IsPositive() && act.Target != nil {
			coins := sdk.NewCoins(sdk.NewCoin("uzrn", act.Amount))
			_ = bank.SendCoins(nil, act.Actor, act.Target, coins)
		}

	case ActivityDelegation:
		if act.Amount.IsPositive() {
			coins := sdk.NewCoins(sdk.NewCoin("uzrn", act.Amount))
			if err := bank.SendCoinsFromAccountToModule(nil, act.Actor, "staking", coins); err != nil {
				return
			}
			if act.Target != nil {
				for _, v := range s.validators {
					if v.addr.Equals(act.Target) {
						v.staked = v.staked.Add(act.Amount)
						break
					}
				}
			}
		}

	case ActivityGovernance:
		// Gas-only.

	case ActivityResearch:
		if act.Amount.IsPositive() {
			coins := sdk.NewCoins(sdk.NewCoin("uzrn", act.Amount))
			_ = bank.SendCoinsFromAccountToModule(nil, act.Actor, "research_fund", coins)
		}
	}
}

func validatorTier(s *SimState, addr sdk.AccAddress) int {
	for _, v := range s.validators {
		if v.addr.Equals(addr) {
			return v.tier
		}
	}
	return -1
}

func uzrnToZrn(amt sdkmath.Int) float64 {
	f := new(big.Float).SetInt(amt.BigInt())
	f.Quo(f, new(big.Float).SetInt64(1_000_000))
	result, _ := f.Float64()
	return result
}
