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

	qualificationmodule "github.com/zerone-chain/zerone/x/qualification"
	"github.com/zerone-chain/zerone/x/qualification/keeper"
	"github.com/zerone-chain/zerone/x/qualification/types"
)

// These tests hold the consensus-repairs-v1 marker for zerone-core #120 to the
// module's side of the activation contract (internal/activation tests the reads
// and writes): the key is pinned, native genesis selects the marker, the
// genesis field writes the marker and nothing else, genesis JSON carries it
// exactly with a file written before the field existing importing as off, and
// a malformed marker is never exported as off. Every import and export goes
// through the module's own genesis JSON path. The repair's own tests can build
// on the fixture: genesisJSON(t, &off) turns the marker off and
// EnableSingleRefund turns it on.

const singleRefundGenesisField = "single_refund_enabled"

type singleRefundFixture struct {
	ctx sdk.Context
	key storetypes.StoreKey
	k   keeper.Keeper
	am  qualificationmodule.AppModule
	cdc codec.JSONCodec
}

func newSingleRefundFixture(t *testing.T) *singleRefundFixture {
	t.Helper()
	key := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	ms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	k := keeper.NewKeeper(runtime.NewKVStoreService(key), cdc, "authority", nil)
	ctx := sdk.NewContext(ms, cmtproto.Header{ChainID: "consensus-repairs-v1-marker", Height: 100}, false, log.NewNopLogger())
	return &singleRefundFixture{ctx: ctx, key: key, k: k, am: qualificationmodule.NewAppModule(cdc, k), cdc: cdc}
}

func (f *singleRefundFixture) marker() []byte {
	return f.ctx.KVStore(f.key).Get(types.SingleRefundEnabledKey)
}

func (f *singleRefundFixture) enabled(t *testing.T) bool {
	t.Helper()
	enabled, err := f.k.SingleRefundEnabled(f.ctx)
	require.NoError(t, err)
	return enabled
}

func (f *singleRefundFixture) initGenesis(t *testing.T, raw json.RawMessage) {
	t.Helper()
	require.NoError(t, f.am.ValidateGenesis(f.cdc, nil, raw))
	f.am.InitGenesis(f.ctx, f.cdc, raw)
}

// genesisJSON is the module's default genesis JSON with the marker field set
// on, set off, or (nil) removed, as in a file written before the field existed.
func (f *singleRefundFixture) genesisJSON(t *testing.T, field *bool) json.RawMessage {
	t.Helper()
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(f.am.DefaultGenesis(f.cdc), &fields))
	require.JSONEq(t, "true", string(fields[singleRefundGenesisField]), "native genesis selects the marker")
	if field == nil {
		delete(fields, singleRefundGenesisField)
	} else {
		fields[singleRefundGenesisField] = json.RawMessage(strconv.FormatBool(*field))
	}
	raw, err := json.Marshal(fields)
	require.NoError(t, err)
	return raw
}

func (f *singleRefundFixture) snapshot(t *testing.T) map[string]string {
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

// singleRefundExported reads the marker field of exported genesis JSON. Encoders that
// omit false fields leave it out, which reads as off.
func singleRefundExported(t *testing.T, raw json.RawMessage) bool {
	t.Helper()
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &fields))
	value, ok := fields[singleRefundGenesisField]
	if !ok {
		return false
	}
	var on bool
	require.NoError(t, json.Unmarshal(value, &on))
	return on
}

func TestSingleRefundGenesisFieldWritesOnlyTheMarker(t *testing.T) {
	enabled, disabled := true, false
	on, off := newSingleRefundFixture(t), newSingleRefundFixture(t)
	on.initGenesis(t, on.genesisJSON(t, &enabled))
	off.initGenesis(t, off.genesisJSON(t, &disabled))
	want := off.snapshot(t)
	want[string(types.SingleRefundEnabledKey)] = "\x01"
	require.Equal(t, want, on.snapshot(t), "the genesis field writes the marker and nothing else")
}

// A marker that is neither absent nor {1} is a fault, so ExportGenesis panics
// rather than export it as off.
func TestSingleRefundExportRefusesMalformedMarker(t *testing.T) {
	f := newSingleRefundFixture(t)
	f.initGenesis(t, f.genesisJSON(t, nil))
	for _, bad := range [][]byte{{0}, {2}} {
		t.Run(fmt.Sprintf("%x", bad), func(t *testing.T) {
			f.ctx.KVStore(f.key).Set(types.SingleRefundEnabledKey, bad)
			require.Panics(t, func() { f.am.ExportGenesis(f.ctx, f.cdc) }, "a malformed marker is never exported as off")
		})
	}
}

func TestSingleRefundNativeGenesisSelectsMarker(t *testing.T) {
	require.Equal(t, []byte{0x40}, types.SingleRefundEnabledKey)
	gs := types.DefaultGenesis()
	require.True(t, gs.SingleRefundEnabled, "a new chain starts with the repair selected")
	require.NoError(t, gs.Validate())
	gs.SingleRefundEnabled = false
	require.NoError(t, gs.Validate(), "off stays valid")

	f := newSingleRefundFixture(t)
	f.initGenesis(t, f.am.DefaultGenesis(f.cdc))
	require.Equal(t, []byte{1}, f.marker())
	require.True(t, f.enabled(t))
}

func TestSingleRefundGenesisRoundTripsMarker(t *testing.T) {
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
			source := newSingleRefundFixture(t)
			source.initGenesis(t, source.genesisJSON(t, tc.field))
			require.Equal(t, tc.want, source.enabled(t))
			if !tc.want {
				require.Nil(t, source.marker(), "off writes no marker")
			}
			exported := source.am.ExportGenesis(source.ctx, source.cdc)
			require.Equal(t, tc.want, singleRefundExported(t, exported))

			imported := newSingleRefundFixture(t)
			imported.initGenesis(t, exported)
			require.Equal(t, tc.want, imported.enabled(t))
			require.Equal(t, tc.want, singleRefundExported(t, imported.am.ExportGenesis(imported.ctx, imported.cdc)))
		})
	}
}
