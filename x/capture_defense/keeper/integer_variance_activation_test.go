package keeper_test

import (
	"encoding/json"
	"fmt"
	"strconv"
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
	"github.com/stretchr/testify/require"

	capturedefensemodule "github.com/zerone-chain/zerone/x/capture_defense"
	"github.com/zerone-chain/zerone/x/capture_defense/keeper"
	"github.com/zerone-chain/zerone/x/capture_defense/types"
)

// These tests hold the consensus-repairs-v1 marker for zerone-core #119 to the
// module's side of the activation contract (internal/activation tests the reads
// and writes): the key is pinned, native genesis selects the marker, the
// genesis field writes the marker and nothing else, genesis JSON carries it
// exactly with a file written before the field existing importing as off, and
// a malformed marker is never exported as off. Every import and export goes
// through the module's own genesis JSON path. The repair's own tests can build
// on the fixture: genesisJSON(t, &off) turns the marker off and
// EnableIntegerVariance turns it on.

const integerVarianceGenesisField = "integer_variance_enabled"

type integerVarianceFixture struct {
	ctx sdk.Context
	key storetypes.StoreKey
	k   keeper.Keeper
	am  capturedefensemodule.AppModule
	cdc codec.JSONCodec
}

func newIntegerVarianceFixture(t *testing.T) *integerVarianceFixture {
	t.Helper()
	key := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	ms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	k := keeper.NewKeeper(runtime.NewKVStoreService(key), cdc, "authority")
	ctx := sdk.NewContext(ms, cmtproto.Header{ChainID: "consensus-repairs-v1-marker", Height: 100}, false, log.NewNopLogger())
	return &integerVarianceFixture{ctx: ctx, key: key, k: k, am: capturedefensemodule.NewAppModule(cdc, k), cdc: cdc}
}

func (f *integerVarianceFixture) marker() []byte {
	return f.ctx.KVStore(f.key).Get(types.IntegerVarianceEnabledKey)
}

func (f *integerVarianceFixture) enabled(t *testing.T) bool {
	t.Helper()
	enabled, err := f.k.IntegerVarianceEnabled(f.ctx)
	require.NoError(t, err)
	return enabled
}

func (f *integerVarianceFixture) initGenesis(t *testing.T, raw json.RawMessage) {
	t.Helper()
	require.NoError(t, f.am.ValidateGenesis(f.cdc, nil, raw))
	f.am.InitGenesis(f.ctx, f.cdc, raw)
}

// genesisJSON is the module's default genesis JSON with the marker field set
// on, set off, or (nil) removed, as in a file written before the field existed.
func (f *integerVarianceFixture) genesisJSON(t *testing.T, field *bool) json.RawMessage {
	t.Helper()
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(f.am.DefaultGenesis(f.cdc), &fields))
	require.JSONEq(t, "true", string(fields[integerVarianceGenesisField]), "native genesis selects the marker")
	if field == nil {
		delete(fields, integerVarianceGenesisField)
	} else {
		fields[integerVarianceGenesisField] = json.RawMessage(strconv.FormatBool(*field))
	}
	raw, err := json.Marshal(fields)
	require.NoError(t, err)
	return raw
}

func (f *integerVarianceFixture) snapshot(t *testing.T) map[string]string {
	t.Helper()
	iter := f.ctx.KVStore(f.key).Iterator(nil, nil)
	defer iter.Close()
	entries := map[string]string{}
	for ; iter.Valid(); iter.Next() {
		entries[string(iter.Key())] = string(iter.Value())
	}
	require.NoError(t, iter.Error())
	return entries
}

// integerVarianceExported reads the marker field of exported genesis JSON. Encoders that
// omit false fields leave it out, which reads as off.
func integerVarianceExported(t *testing.T, raw json.RawMessage) bool {
	t.Helper()
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &fields))
	value, ok := fields[integerVarianceGenesisField]
	if !ok {
		return false
	}
	var on bool
	require.NoError(t, json.Unmarshal(value, &on))
	return on
}

func TestIntegerVarianceGenesisFieldWritesOnlyTheMarker(t *testing.T) {
	enabled, disabled := true, false
	on, off := newIntegerVarianceFixture(t), newIntegerVarianceFixture(t)
	on.initGenesis(t, on.genesisJSON(t, &enabled))
	off.initGenesis(t, off.genesisJSON(t, &disabled))
	want := off.snapshot(t)
	want[string(types.IntegerVarianceEnabledKey)] = "\x01"
	require.Equal(t, want, on.snapshot(t), "the genesis field writes the marker and nothing else")
}

// A marker that is neither absent nor {1} is a fault, so ExportGenesis panics
// rather than export it as off.
func TestIntegerVarianceExportRefusesMalformedMarker(t *testing.T) {
	f := newIntegerVarianceFixture(t)
	f.initGenesis(t, f.genesisJSON(t, nil))
	for _, bad := range [][]byte{{0}, {2}} {
		t.Run(fmt.Sprintf("%x", bad), func(t *testing.T) {
			f.ctx.KVStore(f.key).Set(types.IntegerVarianceEnabledKey, bad)
			require.Panics(t, func() { f.am.ExportGenesis(f.ctx, f.cdc) }, "a malformed marker is never exported as off")
		})
	}
}

func TestIntegerVarianceNativeGenesisSelectsMarker(t *testing.T) {
	require.Equal(t, []byte{0x08}, types.IntegerVarianceEnabledKey)
	gs := types.DefaultGenesis()
	require.True(t, gs.IntegerVarianceEnabled, "a new chain starts with the repair selected")
	require.NoError(t, gs.Validate())
	gs.IntegerVarianceEnabled = false
	require.NoError(t, gs.Validate(), "off stays valid")

	f := newIntegerVarianceFixture(t)
	f.initGenesis(t, f.am.DefaultGenesis(f.cdc))
	require.Equal(t, []byte{1}, f.marker())
	require.True(t, f.enabled(t))
}

func TestIntegerVarianceGenesisRoundTripsMarker(t *testing.T) {
	on, off := true, false
	for _, tc := range []struct {
		name  string
		field *bool
		want  bool
	}{
		{"on", &on, true},
		{"off", &off, false},
		{"file without the field", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := newIntegerVarianceFixture(t)
			source.initGenesis(t, source.genesisJSON(t, tc.field))
			require.Equal(t, tc.want, source.enabled(t))
			if !tc.want {
				require.Nil(t, source.marker(), "off writes no marker")
			}
			exported := source.am.ExportGenesis(source.ctx, source.cdc)
			require.Equal(t, tc.want, integerVarianceExported(t, exported))

			imported := newIntegerVarianceFixture(t)
			imported.initGenesis(t, exported)
			require.Equal(t, tc.want, imported.enabled(t))
			require.Equal(t, tc.want, integerVarianceExported(t, imported.am.ExportGenesis(imported.ctx, imported.cdc)))
		})
	}
}
