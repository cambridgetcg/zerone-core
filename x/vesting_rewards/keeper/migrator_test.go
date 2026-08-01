package keeper

import (
	"fmt"
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

func setupMigratorKeeper(t *testing.T) (Keeper, sdk.Context) {
	t.Helper()
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())
	cod := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	keeper := NewKeeper(cod, runtime.NewKVStoreService(storeKey), nil, nil, "authority")
	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 222}, false, log.NewNopLogger())
	return keeper, ctx
}

func seedRawParams(t *testing.T, keeper Keeper, ctx sdk.Context, params *types.Params) {
	t.Helper()
	bz, err := proto.Marshal(params)
	require.NoError(t, err)
	require.NoError(t, keeper.storeService.OpenKVStore(ctx).Set(types.ParamsKey, bz))
}

func TestMigrate1to2ClearsEveryLegacyFounderShapeAndPreservesParams(t *testing.T) {
	for _, founderAddress := range []string{"", "zrn1legacyfounderaddress"} {
		t.Run(map[bool]string{true: "empty-address", false: "configured-address"}[founderAddress == ""], func(t *testing.T) {
			keeper, ctx := setupMigratorKeeper(t)
			legacy := types.DefaultParams()
			legacy.FounderShareBps = 70_000
			legacy.FounderAddress = founderAddress
			legacy.BlockReward = "1234567"
			legacy.FloorReward = "765432"
			legacy.EmptyBlockRewardRate = 500
			legacy.GovernanceActivationHeight = 987_654
			legacy.KnowledgeCouplingTargetBps = 654_321
			expected := proto.Clone(legacy).(*types.Params)
			expected.FounderShareBps = 0
			expected.FounderAddress = ""
			expected.BlockReward = "0"
			expected.FloorReward = "0"
			expected.EmptyBlockRewardRate = 0
			seedRawParams(t, keeper, ctx, legacy)
			legacyDistribution := &types.BlockRewardDistribution{
				BlockHeight:       221,
				ProducerReward:    "550000",
				ResearchShare:     "30969",
				TotalMinted:       "1000000",
				ValidatorCount:    22,
				FundBalanceAfter:  "123456789",
				FounderShare:      "2331",
				DevelopmentAmount: "196700",
				ProtocolShare:     "220000",
			}
			keeper.SetBlockRewardDistribution(ctx, legacyDistribution)
			observedBefore := keeper.GetParams(ctx)
			require.True(t, proto.Equal(legacy, observedBefore), "pre-migration reads must preserve historical bytes")
			routing, err := keeper.DistributeRevenue(
				ctx,
				types.SourceVerification,
				"1000000",
				"zrn1legacyexecutionmustignorefounderfields",
				"legacy-state-fixture",
			)
			require.NoError(t, err)
			require.Equal(t, "0", routing.FounderShare, "v2 execution must ignore legacy founder bytes")
			require.Equal(t, fmt.Sprint(legacy.RevenueSplit.ResearchBps), routing.ResearchShare)
			require.ErrorIs(
				t,
				keeper.SetParams(ctx, types.DefaultParams()),
				types.ErrFounderMigrationRequired,
				"ordinary parameter writes must not impersonate the named migration",
			)
			stillLegacy := keeper.GetParams(ctx)
			require.True(t, proto.Equal(legacy, stillLegacy), "rejected write changed historical bytes")

			migrator := NewMigrator(keeper)
			require.NoError(t, migrator.Migrate1to2(ctx))
			stored, err := keeper.getStoredParams(ctx)
			require.NoError(t, err)
			require.True(t, proto.Equal(expected, stored), "migration changed unrelated params\nwant: %+v\n got: %+v", expected, stored)
			preservedDistribution, found := keeper.GetBlockRewardDistribution(ctx, legacyDistribution.BlockHeight)
			require.True(t, found)
			require.True(t, proto.Equal(legacyDistribution, preservedDistribution), "migration rewrote a historical distribution")

			// The migration is deterministic and idempotent.
			require.NoError(t, migrator.Migrate1to2(ctx))
			storedAgain, err := keeper.getStoredParams(ctx)
			require.NoError(t, err)
			require.True(t, proto.Equal(expected, storedAgain))
		})
	}
}

func TestMigrate1to2FailsClosedOnMissingOrCorruptParams(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		keeper, ctx := setupMigratorKeeper(t)
		err := NewMigrator(keeper).Migrate1to2(ctx)
		require.ErrorContains(t, err, "params are missing")
	})

	t.Run("corrupt", func(t *testing.T) {
		keeper, ctx := setupMigratorKeeper(t)
		require.NoError(t, keeper.storeService.OpenKVStore(ctx).Set(types.ParamsKey, []byte{0xff, 0x01}))
		err := NewMigrator(keeper).Migrate1to2(ctx)
		require.ErrorContains(t, err, "unmarshal params")
	})
}

func TestSetParamsEnforcesFounderZeroAtStorageBoundary(t *testing.T) {
	keeper, ctx := setupMigratorKeeper(t)
	params := types.DefaultParams()
	require.NoError(t, keeper.SetParams(ctx, params))

	params.FounderShareBps = 1
	require.ErrorIs(t, keeper.SetParams(ctx, params), types.ErrFounderShareRenounced)
	params.FounderShareBps = 0
	params.FounderAddress = "zrn1recipient"
	require.ErrorIs(t, keeper.SetParams(ctx, params), types.ErrFounderShareRenounced)
	params.FounderAddress = ""
	params.BlockReward = "1"
	require.ErrorIs(t, keeper.SetParams(ctx, params), types.ErrAutomaticRewardRetired)

	stored, err := keeper.getStoredParams(ctx)
	require.NoError(t, err)
	require.Zero(t, stored.FounderShareBps)
	require.Empty(t, stored.FounderAddress)
}

func TestRetirementErrorCodesAreUniqueAndStable(t *testing.T) {
	require.Equal(t, uint32(21), types.ErrFounderShareRenounced.ABCICode())
	require.Equal(t, uint32(22), types.ErrAutomaticRewardRetired.ABCICode())
	require.Equal(t, uint32(23), types.ErrFounderMigrationRequired.ABCICode())
	require.NotEqual(t, types.ErrFounderShareRenounced.ABCICode(), types.ErrAutomaticRewardRetired.ABCICode())
	require.NotEqual(t, types.ErrFounderShareRenounced.ABCICode(), types.ErrFounderMigrationRequired.ABCICode())
	require.NotEqual(t, types.ErrAutomaticRewardRetired.ABCICode(), types.ErrFounderMigrationRequired.ABCICode())
	require.Same(t, types.ErrFounderShareRenounced, types.ErrFounderShareRetired)
}
