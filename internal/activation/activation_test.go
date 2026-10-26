package activation_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	corestore "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/internal/activation"
)

var (
	markerKey   = []byte{0x7e}
	neighborKey = []byte{0x7f}
	errInjected = errors.New("injected store failure")
)

func newStore(t *testing.T) (sdk.Context, *storetypes.KVStoreKey) {
	t.Helper()
	key := storetypes.NewKVStoreKey("activation")
	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	ms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())
	return sdk.NewContext(ms, cmtproto.Header{Height: 1}, false, log.NewNopLogger()), key
}

// faultService fails Get or Set of the marker key with errInjected.
type faultService struct {
	corestore.KVStoreService
	failGet, failSet bool
}

func (s faultService) OpenKVStore(ctx context.Context) corestore.KVStore {
	return faultStore{KVStore: s.KVStoreService.OpenKVStore(ctx), service: s}
}

type faultStore struct {
	corestore.KVStore
	service faultService
}

func (s faultStore) Get(key []byte) ([]byte, error) {
	if s.service.failGet && bytes.Equal(key, markerKey) {
		return nil, errInjected
	}
	return s.KVStore.Get(key)
}

func (s faultStore) Set(key, value []byte) error {
	if s.service.failSet && bytes.Equal(key, markerKey) {
		return errInjected
	}
	return s.KVStore.Set(key, value)
}

func snapshot(t *testing.T, ctx sdk.Context, key storetypes.StoreKey) map[string]string {
	t.Helper()
	iter := ctx.KVStore(key).Iterator(nil, nil)
	defer iter.Close()
	entries := map[string]string{}
	for ; iter.Valid(); iter.Next() {
		entries[string(iter.Key())] = string(iter.Value())
	}
	require.NoError(t, iter.Error())
	return entries
}

// Both store shapes a keeper holds, a KVStoreService and a raw store key, read
// and write the marker identically.
var services = []struct {
	name string
	open func(*storetypes.KVStoreKey) corestore.KVStoreService
}{
	{"KVStoreService", func(key *storetypes.KVStoreKey) corestore.KVStoreService { return runtime.NewKVStoreService(key) }},
	{"StoreKeyService", func(key *storetypes.KVStoreKey) corestore.KVStoreService { return activation.StoreKeyService(key) }},
}

func TestReadAndEnable(t *testing.T) {
	for _, svc := range services {
		for _, tc := range []struct {
			name    string
			stored  []byte // nil: absent
			failGet bool
			want    bool
			wantErr string // "": no error
		}{
			{name: "absent"},
			{name: "on", stored: []byte{1}, want: true},
			{name: "zero", stored: []byte{0}, wantErr: "malformed test marker"},
			{name: "two", stored: []byte{2}, wantErr: "malformed test marker"},
			{name: "extra byte", stored: []byte{1, 1}, wantErr: "malformed test marker"},
			{name: "ff", stored: []byte{0xff}, wantErr: "malformed test marker"},
			{name: "empty", stored: []byte{}, wantErr: "malformed test marker"},
			{name: "get error", failGet: true, wantErr: "read test marker: injected store failure"},
		} {
			t.Run(svc.name+"/"+tc.name, func(t *testing.T) {
				ctx, key := newStore(t)
				ctx.KVStore(key).Set(neighborKey, []byte("neighbor"))
				if tc.stored != nil {
					ctx.KVStore(key).Set(markerKey, tc.stored)
				}
				var service corestore.KVStoreService = svc.open(key)
				if tc.failGet {
					service = faultService{KVStoreService: service, failGet: true}
				}

				metered := ctx.WithGasMeter(storetypes.NewGasMeter(1_000_000))
				got, err := activation.Read(metered, service, markerKey, "test")
				require.Zero(t, metered.GasMeter().GasConsumed(), "a marker read charges no gas")
				require.Equal(t, tc.want, got)
				before := snapshot(t, ctx, key)
				if tc.wantErr != "" {
					require.EqualError(t, err, tc.wantErr, "each module's marker text is unchanged")
					require.ErrorIs(t, err, activation.ErrMarkerFault)
					if tc.failGet {
						require.ErrorIs(t, err, errInjected)
					}
					require.False(t, got, "a fault never reads as on")
					require.EqualError(t, activation.Enable(ctx, service, markerKey, "test"), tc.wantErr, "a fault is never repaired")
					require.Equal(t, before, snapshot(t, ctx, key))
					return
				}
				require.NoError(t, err)

				// Control: the same read through the metered store is charged.
				metered.KVStore(key).Get(markerKey)
				require.NotZero(t, metered.GasMeter().GasConsumed())

				require.NoError(t, activation.Enable(ctx, service, markerKey, "test"))
				before[string(markerKey)] = "\x01"
				require.Equal(t, before, snapshot(t, ctx, key), "Enable writes the marker and nothing else")
				on, err := activation.Read(ctx, service, markerKey, "test")
				require.NoError(t, err)
				require.True(t, on)
				require.NoError(t, activation.Enable(ctx, service, markerKey, "test"), "Enable is idempotent")
				require.Equal(t, before, snapshot(t, ctx, key))
			})
		}
	}
}

func TestEnableWriteError(t *testing.T) {
	ctx, key := newStore(t)
	service := faultService{KVStoreService: runtime.NewKVStoreService(key), failSet: true}
	err := activation.Enable(ctx, service, markerKey, "test")
	require.EqualError(t, err, "write test marker: injected store failure")
	require.ErrorIs(t, err, activation.ErrMarkerFault)
	require.ErrorIs(t, err, errInjected)
	on, err := activation.Read(ctx, service, markerKey, "test")
	require.NoError(t, err)
	require.False(t, on)
	require.Empty(t, snapshot(t, ctx, key))
}
