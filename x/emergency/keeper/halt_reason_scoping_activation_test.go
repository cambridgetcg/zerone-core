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

	emergencymodule "github.com/zerone-chain/zerone/x/emergency"
	"github.com/zerone-chain/zerone/x/emergency/keeper"
	"github.com/zerone-chain/zerone/x/emergency/types"
)

// These tests hold the consensus-repairs-v1 marker for zerone-core #125 to the
// shared activation contract: absent reads as off, only {1} reads as on, any
// other value is an error, a read charges no gas, EnableHaltReasonScoping is idempotent and
// writes only the marker, native genesis selects it, and genesis JSON carries
// it exactly, with a file written before the field existing importing as off.
// Every import and export goes through the module's own genesis JSON path.

const haltReasonScopingGenesisField = "halt_reason_scoping_enabled"

type haltReasonScopingFixture struct {
	ctx sdk.Context
	key storetypes.StoreKey
	k   keeper.Keeper
	am  emergencymodule.AppModule
	cdc codec.JSONCodec
}

func newHaltReasonScopingFixture(t *testing.T) *haltReasonScopingFixture {
	t.Helper()
	key := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	ms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	k := keeper.NewKeeper(runtime.NewKVStoreService(key), cdc, "authority", nil)
	ctx := sdk.NewContext(ms, cmtproto.Header{ChainID: "consensus-repairs-v1-marker", Height: 100}, false, log.NewNopLogger())
	return &haltReasonScopingFixture{ctx: ctx, key: key, k: k, am: emergencymodule.NewAppModule(cdc, k), cdc: cdc}
}

func (f *haltReasonScopingFixture) marker() []byte {
	return f.ctx.KVStore(f.key).Get(types.HaltReasonScopingEnabledKey)
}

func (f *haltReasonScopingFixture) enabled(t *testing.T) bool {
	t.Helper()
	enabled, err := f.k.HaltReasonScopingEnabled(f.ctx)
	require.NoError(t, err)
	return enabled
}

func (f *haltReasonScopingFixture) initGenesis(t *testing.T, raw json.RawMessage) {
	t.Helper()
	require.NoError(t, f.am.ValidateGenesis(f.cdc, nil, raw))
	f.am.InitGenesis(f.ctx, f.cdc, raw)
}

// genesisJSON is the module's default genesis JSON with the marker field set
// on, set off, or (nil) removed, as in a file written before the field existed.
func (f *haltReasonScopingFixture) genesisJSON(t *testing.T, field *bool) json.RawMessage {
	t.Helper()
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(f.am.DefaultGenesis(f.cdc), &fields))
	require.JSONEq(t, "true", string(fields[haltReasonScopingGenesisField]), "native genesis selects the marker")
	if field == nil {
		delete(fields, haltReasonScopingGenesisField)
	} else {
		fields[haltReasonScopingGenesisField] = json.RawMessage(strconv.FormatBool(*field))
	}
	raw, err := json.Marshal(fields)
	require.NoError(t, err)
	return raw
}

func (f *haltReasonScopingFixture) snapshot(t *testing.T) map[string]string {
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

// haltReasonScopingExported reads the marker field of exported genesis JSON. Encoders that
// omit false fields leave it out, which reads as off.
func haltReasonScopingExported(t *testing.T, raw json.RawMessage) bool {
	t.Helper()
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &fields))
	value, ok := fields[haltReasonScopingGenesisField]
	if !ok {
		return false
	}
	var on bool
	require.NoError(t, json.Unmarshal(value, &on))
	return on
}

func TestHaltReasonScopingMarkerReadsAreExact(t *testing.T) {
	require.Equal(t, []byte{0x14}, types.HaltReasonScopingEnabledKey)
	f := newHaltReasonScopingFixture(t)
	f.initGenesis(t, f.genesisJSON(t, nil))
	require.Nil(t, f.marker(), "a genesis file without the field writes no marker")
	require.False(t, f.enabled(t), "absent reads as off")
	require.NotPanics(t, func() { f.am.ExportGenesis(f.ctx, f.cdc) })

	kv := f.ctx.KVStore(f.key)
	for _, bad := range [][]byte{{0}, {2}, {1, 1}, {0xff}} {
		t.Run(fmt.Sprintf("%x", bad), func(t *testing.T) {
			kv.Set(types.HaltReasonScopingEnabledKey, bad)
			enabled, err := f.k.HaltReasonScopingEnabled(f.ctx)
			require.ErrorContains(t, err, "malformed")
			require.False(t, enabled, "a malformed marker is never read as on")
			require.Error(t, f.k.EnableHaltReasonScoping(f.ctx), "a malformed marker is never repaired")
			require.Equal(t, bad, f.marker())
			require.Panics(t, func() { f.am.ExportGenesis(f.ctx, f.cdc) }, "a malformed marker is never exported as off")
		})
	}

	kv.Set(types.HaltReasonScopingEnabledKey, []byte{1})
	require.True(t, f.enabled(t))
}

func TestHaltReasonScopingMarkerReadChargesNoGas(t *testing.T) {
	f := newHaltReasonScopingFixture(t)
	for _, present := range []bool{false, true} {
		if present {
			require.NoError(t, f.k.EnableHaltReasonScoping(f.ctx))
		}
		metered := f.ctx.WithGasMeter(storetypes.NewGasMeter(1_000_000))
		enabled, err := f.k.HaltReasonScopingEnabled(metered)
		require.NoError(t, err)
		require.Equal(t, present, enabled)
		require.Zero(t, metered.GasMeter().GasConsumed(), "present=%t: the marker read charged gas", present)

		// Control: the same read through the metered store is charged.
		metered.KVStore(f.key).Get(types.HaltReasonScopingEnabledKey)
		require.NotZero(t, metered.GasMeter().GasConsumed())
	}
}

func TestEnableHaltReasonScopingIsIdempotentAndWritesOnlyTheMarker(t *testing.T) {
	f := newHaltReasonScopingFixture(t)
	f.initGenesis(t, f.genesisJSON(t, nil))
	before := f.snapshot(t)
	require.NotContains(t, before, string(types.HaltReasonScopingEnabledKey))

	require.NoError(t, f.k.EnableHaltReasonScoping(f.ctx))
	after := f.snapshot(t)
	before[string(types.HaltReasonScopingEnabledKey)] = "\x01"
	require.Equal(t, before, after, "enabling writes the marker and nothing else")
	require.True(t, f.enabled(t))

	require.NoError(t, f.k.EnableHaltReasonScoping(f.ctx), "enabling an enabled marker is a no-op")
	require.Equal(t, after, f.snapshot(t))
}

func TestHaltReasonScopingNativeGenesisSelectsMarker(t *testing.T) {
	gs := types.DefaultGenesis()
	require.True(t, gs.HaltReasonScopingEnabled, "a new chain starts with the repair selected")
	require.NoError(t, gs.Validate())
	gs.HaltReasonScopingEnabled = false
	require.NoError(t, gs.Validate(), "off stays valid")

	f := newHaltReasonScopingFixture(t)
	f.initGenesis(t, f.am.DefaultGenesis(f.cdc))
	require.Equal(t, []byte{1}, f.marker())
	require.True(t, f.enabled(t))
}

func TestHaltReasonScopingGenesisRoundTripsMarker(t *testing.T) {
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
			source := newHaltReasonScopingFixture(t)
			source.initGenesis(t, source.genesisJSON(t, tc.field))
			require.Equal(t, tc.want, source.enabled(t))
			if !tc.want {
				require.Nil(t, source.marker(), "off writes no marker")
			}
			exported := source.am.ExportGenesis(source.ctx, source.cdc)
			require.Equal(t, tc.want, haltReasonScopingExported(t, exported))

			imported := newHaltReasonScopingFixture(t)
			imported.initGenesis(t, exported)
			require.Equal(t, tc.want, imported.enabled(t))
			require.Equal(t, tc.want, haltReasonScopingExported(t, imported.am.ExportGenesis(imported.ctx, imported.cdc)))
		})
	}
}
