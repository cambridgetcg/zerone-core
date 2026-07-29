package keeper

import (
	"testing"

	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	bridgetypes "github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

func setupGenesisExportKeeper(t *testing.T) (Keeper, sdk.Context) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(bridgetypes.StoreKey)
	db := dbm.NewMemDB()
	cms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	cms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, cms.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	bridgetypes.RegisterInterfaces(registry)
	k := NewKeeper(codec.NewProtoCodec(registry), storeKey, "authority-addr", nil, nil, nil, nil, nil)
	ctx := sdk.NewContext(cms, cmtproto.Header{Height: 1}, false, log.NewNopLogger())
	require.NoError(t, k.SetParams(ctx, bridgetypes.DefaultParams()))

	return k, ctx
}

func TestExportGenesisSkipsOnlyExactAppIAVLInitSentinel(t *testing.T) {
	t.Run("exact sentinel is infrastructure", func(t *testing.T) {
		k, ctx := setupGenesisExportKeeper(t)
		ctx.KVStore(k.storeKey).Set([]byte(appIAVLInitSentinelKey), []byte{0x01})

		var exportedKeys [][]byte
		require.NotPanics(t, func() {
			for _, entry := range k.ExportGenesis(ctx).StateEntries {
				exportedKeys = append(exportedKeys, entry.Key)
			}
		})
		require.NotContains(t, exportedKeys, []byte(appIAVLInitSentinelKey))
	})

	t.Run("nearby unknown key still fails loud", func(t *testing.T) {
		k, ctx := setupGenesisExportKeeper(t)
		ctx.KVStore(k.storeKey).Set([]byte(appIAVLInitSentinelKey+"_extra"), []byte{0x01})

		require.PanicsWithValue(
			t,
			"substrate_bridge export has unhandled state key 5f6961766c5f696e69745f6578747261",
			func() { k.ExportGenesis(ctx) },
		)
	})
}
