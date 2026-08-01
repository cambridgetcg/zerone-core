package keeper

import (
	"bytes"
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
	"github.com/stretchr/testify/require"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

func setupStrictMigrationKeeper(t *testing.T) (Keeper, sdk.Context) {
	t.Helper()
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	k := NewKeeper(
		codec.NewProtoCodec(registry),
		runtime.NewKVStoreService(storeKey),
		nil,
		nil,
		"authority",
	)
	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 1000}, false, log.NewNopLogger())
	return k, ctx
}

func writeRawMigrationParams(t *testing.T, k Keeper, ctx sdk.Context, params *types.Params) []byte {
	t.Helper()
	bz, err := proto.Marshal(params)
	require.NoError(t, err)
	require.NoError(t, k.storeService.OpenKVStore(ctx).Set(types.ParamsKey, bz))
	return bz
}

func readRawMigrationParams(t *testing.T, k Keeper, ctx sdk.Context) []byte {
	t.Helper()
	bz, err := k.storeService.OpenKVStore(ctx).Get(types.ParamsKey)
	require.NoError(t, err)
	return bz
}

func legacyEconomicParams() *types.Params {
	params := types.DefaultParams()
	params.FounderShareBps = 70_000
	params.FounderAddress = "zrn1legacyfounder00000000000000000000000"
	params.BlockReward = "10000000"
	params.FloorReward = "100000"
	params.EmptyBlockRewardRate = 500
	return params
}

func TestMigrate1to2StrictReadFailsOnMissingParams(t *testing.T) {
	k, ctx := setupStrictMigrationKeeper(t)

	err := NewMigrator(k).Migrate1to2(ctx)
	require.ErrorContains(t, err, "vesting_rewards params are missing")
	require.Nil(t, readRawMigrationParams(t, k, ctx))
}

func TestInitGenesisRejectsMissingParams(t *testing.T) {
	k, ctx := setupStrictMigrationKeeper(t)
	require.PanicsWithValue(t,
		"invalid vesting_rewards genesis params: params must not be nil",
		func() {
			k.InitGenesis(ctx, &types.GenesisState{CategoryConfigs: types.DefaultCategoryConfigs()})
		},
	)
	require.Nil(t, readRawMigrationParams(t, k, ctx))
}

func TestMigrate1to2StrictReadFailsOnCorruptParamsWithoutMutation(t *testing.T) {
	k, ctx := setupStrictMigrationKeeper(t)
	corrupt := []byte{0xff, 0x01, 0x80}
	require.NoError(t, k.storeService.OpenKVStore(ctx).Set(types.ParamsKey, corrupt))

	err := NewMigrator(k).Migrate1to2(ctx)
	require.ErrorContains(t, err, "unmarshal params")
	require.True(t, bytes.Equal(corrupt, readRawMigrationParams(t, k, ctx)))
}

func TestOrdinarySetParamsCannotImpersonateMigration(t *testing.T) {
	k, ctx := setupStrictMigrationKeeper(t)
	legacy := legacyEconomicParams()
	legacyBytes := writeRawMigrationParams(t, k, ctx, legacy)

	proposed := proto.Clone(legacy).(*types.Params)
	proposed.FounderShareBps = 0
	proposed.FounderAddress = ""
	proposed.BlockReward = "0"
	proposed.FloorReward = "0"
	proposed.EmptyBlockRewardRate = 0

	err := k.SetParams(ctx, proposed)
	require.ErrorIs(t, err, types.ErrEconomicNeutralityMigrationRequired)
	require.True(t, bytes.Equal(legacyBytes, readRawMigrationParams(t, k, ctx)))
}

func TestMigrate1to2IsIdempotentAndRestoresOrdinaryWrites(t *testing.T) {
	k, ctx := setupStrictMigrationKeeper(t)
	writeRawMigrationParams(t, k, ctx, legacyEconomicParams())
	migrator := NewMigrator(k)

	require.NoError(t, migrator.Migrate1to2(ctx))
	first := append([]byte(nil), readRawMigrationParams(t, k, ctx)...)
	require.NoError(t, migrator.Migrate1to2(ctx))
	require.True(t, bytes.Equal(first, readRawMigrationParams(t, k, ctx)))

	updated, err := k.getStoredParams(ctx)
	require.NoError(t, err)
	updated.ReleasedClawbackRate++
	require.NoError(t, k.SetParams(ctx, updated))
	require.Equal(t, uint64(3301), k.GetParams(ctx).ReleasedClawbackRate)
}

func TestRuntimeQueriesFailClosedOnMissingOrCorruptParams(t *testing.T) {
	for _, fixture := range []struct {
		name string
		seed []byte
	}{
		{name: "missing"},
		{name: "corrupt", seed: []byte{0xff, 0x01, 0x80}},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			k, ctx := setupStrictMigrationKeeper(t)
			if fixture.seed != nil {
				require.NoError(t, k.storeService.OpenKVStore(ctx).Set(types.ParamsKey, fixture.seed))
			}
			queries := NewQueryServerImpl(k)
			_, err := queries.Params(ctx, &types.QueryParamsRequest{})
			require.Error(t, err)
			_, err = queries.FounderShareStatus(ctx, &types.QueryFounderShareStatusRequest{})
			require.Error(t, err)
			_, err = queries.SupplyCouplingAudit(ctx, &types.QuerySupplyCouplingAuditRequest{})
			require.Error(t, err)
		})
	}
}
