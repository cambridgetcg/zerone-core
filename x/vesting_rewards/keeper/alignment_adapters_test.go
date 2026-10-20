package keeper_test

import (
	"context"
	"math/big"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	alignment "github.com/zerone-chain/zerone/x/alignment"
	alignmentkeeper "github.com/zerone-chain/zerone/x/alignment/keeper"
	alignmenttypes "github.com/zerone-chain/zerone/x/alignment/types"
	stakingkeeper "github.com/zerone-chain/zerone/x/staking/keeper"
	stakingtypes "github.com/zerone-chain/zerone/x/staking/types"
	"github.com/zerone-chain/zerone/x/vesting_rewards/keeper"
	"github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

const alignmentSupplyTestFund = "alignment_supply_test_fund"

type alignmentSupplyHarness struct {
	ctx        sdk.Context
	cdc        codec.Codec
	bank       bankkeeper.BaseKeeper
	vesting    keeper.Keeper
	staking    stakingkeeper.Keeper
	alignment  alignmentkeeper.Keeper
	adapter    *keeper.AlignmentVestingRewardsAdapter
	vestingKey *storetypes.KVStoreKey
}

// Domain count is stubbed and knowledge/emergency are unwired. Supply, stake,
// both adapters, alignment observations and EndBlock use real keepers in memory.
type emptyAlignmentDomains struct{}

func (emptyAlignmentDomains) GetDomainCount(context.Context) uint64 { return 0 }

func newAlignmentSupplyHarness(t *testing.T) alignmentSupplyHarness {
	t.Helper()
	return newAlignmentSupplyHarnessWithActivation(t, true)
}

func newAlignmentSupplyHarnessWithActivation(t *testing.T, enabled bool) alignmentSupplyHarness {
	t.Helper()
	keys := storetypes.NewKVStoreKeys(types.StoreKey, alignmenttypes.StoreKey,
		stakingtypes.StoreKey, authtypes.StoreKey, banktypes.StoreKey)
	db := dbm.NewMemDB()
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	state := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	// Do not leave asynchronous IAVL pruning workers behind in repeated fixtures.
	state.SetIAVLSyncPruning(true)
	for _, key := range keys {
		state.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	}
	require.NoError(t, state.LoadLatestVersion())
	ctx := sdk.NewContext(state, cmtproto.Header{Height: 100}, false, log.NewNopLogger())
	registry := codectypes.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(registry)
	banktypes.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress("gov").String()
	perms := map[string][]string{
		alignmentSupplyTestFund: {authtypes.Minter, authtypes.Burner},
		stakingtypes.ModuleName: nil,
	}
	accounts := authkeeper.NewAccountKeeper(cdc, runtime.NewKVStoreService(keys[authtypes.StoreKey]),
		authtypes.ProtoBaseAccount, perms, addresscodec.NewBech32Codec("zrn"), "zrn", authority)
	accounts.InitGenesis(ctx, *authtypes.DefaultGenesisState())
	bank := bankkeeper.NewBaseKeeper(cdc, runtime.NewKVStoreService(keys[banktypes.StoreKey]),
		accounts, map[string]bool{}, authority, log.NewNopLogger())
	bank.InitGenesis(ctx, banktypes.DefaultGenesisState())
	vesting := keeper.NewKeeper(cdc, runtime.NewKVStoreService(keys[types.StoreKey]), bank, nil, authority)
	genesis := types.DefaultGenesis()
	genesis.AlignmentBankSupplyEnabled = false
	vesting.InitGenesis(ctx, genesis)
	if enabled {
		require.NoError(t, vesting.EnableAlignmentBankSupply(ctx))
	}
	staking := stakingkeeper.NewKeeper(cdc, keys[stakingtypes.StoreKey], accounts, bank, authority)
	staking.InitGenesis(ctx, stakingtypes.DefaultGenesisState())
	adapter := keeper.NewAlignmentVestingRewardsAdapter(vesting)
	align := alignmentkeeper.NewKeeper(runtime.NewKVStoreService(keys[alignmenttypes.StoreKey]),
		cdc, authority, nil, stakingkeeper.NewAlignmentStakingAdapter(staking), emptyAlignmentDomains{}, nil, adapter)
	align.InitGenesis(ctx, alignmenttypes.DefaultGenesis())
	return alignmentSupplyHarness{
		ctx: ctx, cdc: cdc, bank: bank, vesting: vesting, staking: staking,
		alignment: align, adapter: adapter, vestingKey: keys[types.StoreKey],
	}
}

func (h alignmentSupplyHarness) mint(t *testing.T, denom string, amount int64) {
	t.Helper()
	require.NoError(t, h.bank.MintCoins(h.ctx, alignmentSupplyTestFund,
		sdk.NewCoins(sdk.NewInt64Coin(denom, amount))))
}

func (h alignmentSupplyHarness) stake(t *testing.T, amount int64) {
	t.Helper()
	operator := sdk.AccAddress("alignment_operator__")
	require.NoError(t, h.bank.SendCoinsFromModuleToAccount(h.ctx, alignmentSupplyTestFund,
		operator, sdk.NewCoins(sdk.NewInt64Coin("uzrn", amount))))
	_, err := stakingkeeper.NewMsgServerImpl(h.staking).RegisterValidator(h.ctx, &stakingtypes.MsgRegisterValidator{
		Operator: operator.String(), ConsensusPubkey: "fixture-only", SelfDelegation: big.NewInt(amount).String(),
	})
	require.NoError(t, err)
	require.Equal(t, big.NewInt(amount), h.staking.GetTotalBondedStake(h.ctx))
}

func TestAlignmentSupply_UsesCurrentBankSupply(t *testing.T) {
	h := newAlignmentSupplyHarness(t)
	h.vesting.SetTotalMinted(h.ctx, big.NewInt(50_000_000_000))
	// A different denom must not become the denominator, even when uzrn is absent.
	h.mint(t, "uother", 7_000_000_000_000)
	require.Zero(t, h.adapter.GetTotalSupply(h.ctx).Sign())
	h.mint(t, "uzrn", 1_000_000_000_000)
	require.Equal(t, big.NewInt(1_000_000_000_000), h.adapter.GetTotalSupply(h.ctx))

	require.NoError(t, h.bank.BurnCoins(h.ctx, alignmentSupplyTestFund,
		sdk.NewCoins(sdk.NewInt64Coin("uzrn", 750_000_000_000))))
	require.Equal(t, big.NewInt(250_000_000_000), h.adapter.GetTotalSupply(h.ctx))
	require.NoError(t, h.bank.BurnCoins(h.ctx, alignmentSupplyTestFund,
		sdk.NewCoins(sdk.NewInt64Coin("uzrn", 250_000_000_000))))
	require.Zero(t, h.adapter.GetTotalSupply(h.ctx).Sign())
	require.Equal(t, big.NewInt(50_000_000_000), h.vesting.GetTotalMinted(h.ctx), "supply reads and burns must not rewrite the mint ledger")
}

func TestAlignmentSupply_ObservationsIgnoreMintLedger(t *testing.T) {
	for _, minted := range []int64{0, 50_000_000_000, 2_000_000_000_000} {
		t.Run(big.NewInt(minted).String(), func(t *testing.T) {
			h := newAlignmentSupplyHarness(t)
			h.mint(t, "uzrn", 1_000_000_000_000)
			h.stake(t, 100_000_000_000)
			h.vesting.SetTotalMinted(h.ctx, big.NewInt(minted))
			before, err := proto.Marshal(h.vesting.ExportGenesis(h.ctx))
			require.NoError(t, err)
			obs := h.alignment.ObserveAll(h.ctx)
			t.Logf("bank_supply=%s ledger=%d stake=%s staking_ratio=%d economic_stability=%d",
				h.bank.GetSupply(h.ctx, "uzrn").Amount, minted, h.staking.GetTotalBondedStake(h.ctx),
				obs.StakingRatio, obs.EconomicStability)
			require.Equal(t, uint64(100_000), obs.StakingRatio)
			require.Equal(t, uint64(100_000), obs.EconomicStability)
			require.Equal(t, alignmenttypes.CategoryCritical,
				h.alignment.CategorizeHealth(h.ctx, h.alignment.ComputeScores(h.ctx, obs).Composite))
			after, err := proto.Marshal(h.vesting.ExportGenesis(h.ctx))
			require.NoError(t, err)
			require.Equal(t, before, after, "observation is not a vesting state migration")

			// Burn outside MintWithCap: the next observation follows current supply,
			// not the unchanged cumulative ledger or a cached denominator.
			require.NoError(t, h.bank.BurnCoins(h.ctx, alignmentSupplyTestFund,
				sdk.NewCoins(sdk.NewInt64Coin("uzrn", 500_000_000_000))))
			obs = h.alignment.ObserveAll(h.ctx)
			require.Equal(t, uint64(200_000), obs.StakingRatio)
			require.Equal(t, uint64(200_000), obs.EconomicStability)
			require.Equal(t, big.NewInt(minted), h.vesting.GetTotalMinted(h.ctx))
		})
	}
}

func TestAlignmentSupply_EndBlockPreservesHistoryAndUpdatesPacing(t *testing.T) {
	h := newAlignmentSupplyHarnessWithActivation(t, false)
	h.mint(t, "uzrn", 1_000_000_000_000)
	h.stake(t, 100_000_000_000)
	// The marker-off adapter produces actual legacy zero-ledger readings.
	oldObs := h.alignment.ObserveAll(h.ctx.WithBlockHeight(99))
	require.Equal(t, alignmenttypes.NeutralBPS, oldObs.StakingRatio)
	require.Equal(t, alignmenttypes.NeutralBPS, oldObs.EconomicStability)
	oldScores := h.alignment.ComputeScores(h.ctx, oldObs)
	oldCategory := h.alignment.CategorizeHealth(h.ctx, oldScores.Composite)
	require.Equal(t, alignmenttypes.CategoryDegraded, oldCategory)
	oldHealth := h.alignment.BuildHealthIndex(oldScores, oldCategory, 0)
	h.alignment.SetObservation(h.ctx, oldObs)
	h.alignment.SetScores(h.ctx, oldScores)
	h.alignment.SetHealthIndex(h.ctx, oldHealth)
	h.alignment.SetState(h.ctx, &alignmenttypes.AlignmentState{
		Enabled: true, PreviousCategory: oldCategory, LastObservationHeight: 99, ObservationCount: 1,
	})
	creation, analysis := h.alignment.GetGlobalPacingMultiplier(h.ctx)
	require.Equal(t, uint64(750_000), creation)
	require.Equal(t, uint64(1_500_000), analysis)

	// Activation does not reset history or force an immediate observation.
	require.NoError(t, h.vesting.EnableAlignmentBankSupply(h.ctx))
	module := alignment.NewAppModule(h.cdc, h.alignment)
	require.NoError(t, module.EndBlock(h.ctx.WithBlockHeight(101)))
	require.Equal(t, oldCategory, h.alignment.GetState(h.ctx).PreviousCategory)
	require.NoError(t, module.EndBlock(h.ctx.WithBlockHeight(200)))
	obs, found := h.alignment.GetObservation(h.ctx, 200)
	require.True(t, found)
	require.Equal(t, uint64(100_000), obs.StakingRatio)
	require.Equal(t, uint64(100_000), obs.EconomicStability)
	health, found := h.alignment.GetHealthIndex(h.ctx, 200)
	require.True(t, found)
	require.Equal(t, alignmenttypes.CategoryCritical, health.Category)
	state := h.alignment.GetState(h.ctx)
	require.Equal(t, health.Category, state.PreviousCategory)
	require.Equal(t, uint64(2), state.ObservationCount)
	require.True(t, state.DegradedFrequencyActive)
	creation, analysis = h.alignment.GetGlobalPacingMultiplier(h.ctx)
	require.Equal(t, uint64(500_000), creation)
	require.Equal(t, uint64(2_000_000), analysis)
	t.Logf("retained_category=%s current_category=%s creation_pacing=%d analysis_pacing=%d",
		oldCategory, health.Category, creation, analysis)
	retainedObs, found := h.alignment.GetObservation(h.ctx, 99)
	require.True(t, found)
	require.True(t, proto.Equal(oldObs, retainedObs))
	retainedScores, found := h.alignment.GetScores(h.ctx, 99)
	require.True(t, found)
	require.True(t, proto.Equal(oldScores, retainedScores))
	retainedHealth, found := h.alignment.GetHealthIndex(h.ctx, 99)
	require.True(t, found)
	require.True(t, proto.Equal(oldHealth, retainedHealth))
	require.Zero(t, h.vesting.GetTotalMinted(h.ctx).Sign())
	require.Equal(t, "1000000000000", h.bank.GetSupply(h.ctx, "uzrn").Amount.String())
}

func TestAlignmentSupply_UnwiredBankDoesNotUseMintLedger(t *testing.T) {
	h := newAlignmentSupplyHarness(t)
	h.vesting.SetTotalMinted(h.ctx, big.NewInt(123))
	k := keeper.NewKeeper(h.cdc, runtime.NewKVStoreService(h.vestingKey), nil, nil, "authority")
	require.Equal(t, big.NewInt(123), k.GetTotalMinted(h.ctx))
	require.Zero(t, keeper.NewAlignmentVestingRewardsAdapter(k).GetTotalSupply(h.ctx).Sign())
}
