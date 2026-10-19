package keeper_test

import (
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

	"github.com/zerone-chain/zerone/x/vesting_rewards/keeper"
	"github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

// setupFeeRoutingFixture funds the fee collector in real SDK auth/bank stores.
// The returned module list covers every account holding the synthetic issuance.
func setupFeeRoutingFixture(t *testing.T, genesis *types.GenesisState, fees sdk.Coins) (keeper.Keeper, sdk.Context, bankkeeper.BaseKeeper, []string) {
	t.Helper()
	const fundingModule = "fee_routing_test_fund"
	keys := storetypes.NewKVStoreKeys(types.StoreKey, authtypes.StoreKey, banktypes.StoreKey)
	db := dbm.NewMemDB()
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	state := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	state.SetIAVLSyncPruning(true)
	for _, name := range []string{types.StoreKey, authtypes.StoreKey, banktypes.StoreKey} {
		state.MountStoreWithDB(keys[name], storetypes.StoreTypeIAVL, db)
	}
	require.NoError(t, state.LoadLatestVersion())
	ctx := sdk.NewContext(state, cmtproto.Header{Height: 100}, false, log.NewNopLogger())
	registry := codectypes.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(registry)
	banktypes.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress("gov").String()
	perms := map[string][]string{
		fundingModule: {authtypes.Minter}, authtypes.FeeCollectorName: nil,
		types.ModuleName: nil, types.ResearchFundModuleName: nil, types.DevelopmentFundModuleName: nil,
	}
	ak := authkeeper.NewAccountKeeper(cdc, runtime.NewKVStoreService(keys[authtypes.StoreKey]),
		authtypes.ProtoBaseAccount, perms, addresscodec.NewBech32Codec("zrn"), "zrn", authority)
	ak.InitGenesis(ctx, *authtypes.DefaultGenesisState())
	bank := bankkeeper.NewBaseKeeper(cdc, runtime.NewKVStoreService(keys[banktypes.StoreKey]), ak, map[string]bool{}, authority, log.NewNopLogger())
	bank.InitGenesis(ctx, banktypes.DefaultGenesisState())
	modules := []string{fundingModule, authtypes.FeeCollectorName, types.ModuleName, types.ResearchFundModuleName, types.DevelopmentFundModuleName}
	for _, module := range modules {
		require.NotNil(t, ak.GetModuleAccount(ctx, module))
	}
	k := keeper.NewKeeper(cdc, runtime.NewKVStoreService(keys[types.StoreKey]), bank, nil, authority)
	k.InitGenesis(ctx, genesis)
	require.NoError(t, bank.MintCoins(ctx, fundingModule, fees), "synthetic test issuance")
	require.NoError(t, bank.SendCoinsFromModuleToModule(ctx, fundingModule, authtypes.FeeCollectorName, fees))
	return k, ctx.WithEventManager(sdk.NewEventManager()), bank, modules
}
