package keeper

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/auth/types"
)

const (
	corruptionTestAddress1 = "zrn1m037n75vk2jhdr56y2ptzjjj02uljwnqwwzr7z"
	corruptionTestAddress2 = "zrn1ur4eyeuuhrkfpcyhykfjsasftv9hn33smszt58"
	corruptionTestDID1     = "did:zrn:d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a"
	corruptionTestDID2     = "did:zrn:3d4017c3e843895a92b70aa74d1b7ebc9c982ccf2ec4968cc0cd55f12af4660c"
)

func setupCorruptionTestKeeper(t *testing.T) (Keeper, sdk.Context) {
	t.Helper()
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	database := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(database, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, database)
	if err := stateStore.LoadLatestVersion(); err != nil {
		t.Fatal(err)
	}
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	keeper := NewKeeper(cdc, runtime.NewKVStoreService(storeKey), corruptionTestAddress2)
	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 1}, false, log.NewNopLogger())
	return keeper, ctx
}

func requirePanic(t *testing.T, operation func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected corrupt point read to panic")
		}
	}()
	operation()
}

func TestPointReadsDistinguishMissingFromCorruptState(t *testing.T) {
	keeper, ctx := setupCorruptionTestKeeper(t)
	if account, found := keeper.GetAccount(ctx, corruptionTestAddress1); found || account != nil {
		t.Fatal("missing account did not return (nil, false)")
	}
	if mapping, found := keeper.GetDIDMapping(ctx, corruptionTestDID1); found || mapping != nil {
		t.Fatal("missing DID mapping did not return (nil, false)")
	}

	kvStore := keeper.storeService.OpenKVStore(ctx)
	if err := kvStore.Set(types.AccountKey(corruptionTestAddress1), []byte{0xff}); err != nil {
		t.Fatal(err)
	}
	requirePanic(t, func() { _, _ = keeper.GetAccount(ctx, corruptionTestAddress1) })

	if err := kvStore.Set(types.DIDMappingKey(corruptionTestDID1), []byte{0xff}); err != nil {
		t.Fatal(err)
	}
	requirePanic(t, func() { _, _ = keeper.GetDIDMapping(ctx, corruptionTestDID1) })
}

func TestPointReadsRejectStoreKeyContentMismatch(t *testing.T) {
	keeper, ctx := setupCorruptionTestKeeper(t)
	kvStore := keeper.storeService.OpenKVStore(ctx)

	accountBytes, err := proto.Marshal(&types.Account{Address: corruptionTestAddress2})
	if err != nil {
		t.Fatal(err)
	}
	if err := kvStore.Set(types.AccountKey(corruptionTestAddress1), accountBytes); err != nil {
		t.Fatal(err)
	}
	requirePanic(t, func() { _, _ = keeper.GetAccount(ctx, corruptionTestAddress1) })

	mappingBytes, err := proto.Marshal(&types.DIDMapping{Did: corruptionTestDID2})
	if err != nil {
		t.Fatal(err)
	}
	if err := kvStore.Set(types.DIDMappingKey(corruptionTestDID1), mappingBytes); err != nil {
		t.Fatal(err)
	}
	requirePanic(t, func() { _, _ = keeper.GetDIDMapping(ctx, corruptionTestDID1) })
}
