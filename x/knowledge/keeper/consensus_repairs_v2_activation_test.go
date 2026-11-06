package keeper_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	knowledgemodule "github.com/zerone-chain/zerone/x/knowledge"
	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// These tests hold each consensus-repairs-v2 knowledge marker to the module's
// side of the activation contract (internal/activation tests the reads and
// writes): the key is pinned, native genesis selects the marker, the genesis
// field writes that marker and nothing else, genesis JSON carries it exactly
// with a file written before the field existing importing as off, and a
// malformed marker is never exported as off. Every import and export goes
// through the module's own genesis JSON path. A repair's own tests can build
// on the fixture: genesisJSON(t, &off) turns its marker off and its Enable
// turns it on.

// knowledgeV2Marker binds one consensus-repairs-v2 marker to its pinned key,
// genesis field and keeper functions.
type knowledgeV2Marker struct {
	field   string
	pinned  byte
	key     []byte
	enabled func(keeper.Keeper, context.Context) (bool, error)
	enable  func(keeper.Keeper, context.Context) error
	genesis func(*types.GenesisState) *bool
}

var knowledgeV2Markers = []knowledgeV2Marker{
	{
		field: "inconclusive_round_accounting_enabled", pinned: 0x91, key: types.InconclusiveRoundAccountingEnabledKey,
		enabled: keeper.Keeper.InconclusiveRoundAccountingEnabled, enable: keeper.Keeper.EnableInconclusiveRoundAccounting,
		genesis: func(gs *types.GenesisState) *bool { return &gs.InconclusiveRoundAccountingEnabled },
	},
	{
		field: "calibration_smoothing_enabled", pinned: 0x92, key: types.CalibrationSmoothingEnabledKey,
		enabled: keeper.Keeper.CalibrationSmoothingEnabled, enable: keeper.Keeper.EnableCalibrationSmoothing,
		genesis: func(gs *types.GenesisState) *bool { return &gs.CalibrationSmoothingEnabled },
	},
	{
		field: "zero_confidence_floor_enabled", pinned: 0x93, key: types.ZeroConfidenceFloorEnabledKey,
		enabled: keeper.Keeper.ZeroConfidenceFloorEnabled, enable: keeper.Keeper.EnableZeroConfidenceFloor,
		genesis: func(gs *types.GenesisState) *bool { return &gs.ZeroConfidenceFloorEnabled },
	},
}

type knowledgeV2MarkerFixture struct {
	marker knowledgeV2Marker
	ctx    sdk.Context
	key    storetypes.StoreKey
	k      keeper.Keeper
	am     knowledgemodule.AppModule
	cdc    codec.JSONCodec
}

func newKnowledgeV2MarkerFixture(t *testing.T, marker knowledgeV2Marker) *knowledgeV2MarkerFixture {
	t.Helper()
	f := newKnowledgeStoreFixture(t, "consensus-repairs-v2-marker")
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	return &knowledgeV2MarkerFixture{marker: marker, ctx: f.ctx, key: f.key, k: f.k, am: knowledgemodule.NewAppModule(cdc, f.k), cdc: cdc}
}

func (f *knowledgeV2MarkerFixture) stored() []byte {
	return f.ctx.KVStore(f.key).Get(f.marker.key)
}

func (f *knowledgeV2MarkerFixture) enabled(t *testing.T) bool {
	t.Helper()
	enabled, err := f.marker.enabled(f.k, f.ctx)
	require.NoError(t, err)
	return enabled
}

func (f *knowledgeV2MarkerFixture) initGenesis(t *testing.T, raw json.RawMessage) {
	t.Helper()
	require.NoError(t, f.am.ValidateGenesis(f.cdc, nil, raw))
	f.am.InitGenesis(f.ctx, f.cdc, raw)
}

// genesisJSON is the module's default genesis JSON with this marker's field
// set on, set off, or (nil) removed, as in a file written before the field
// existed. Every other field keeps its default.
func (f *knowledgeV2MarkerFixture) genesisJSON(t *testing.T, field *bool) json.RawMessage {
	t.Helper()
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(f.am.DefaultGenesis(f.cdc), &fields))
	require.JSONEq(t, "true", string(fields[f.marker.field]), "native genesis selects the marker")
	if field == nil {
		delete(fields, f.marker.field)
	} else {
		fields[f.marker.field] = json.RawMessage(strconv.FormatBool(*field))
	}
	raw, err := json.Marshal(fields)
	require.NoError(t, err)
	return raw
}

func (f *knowledgeV2MarkerFixture) snapshot(t *testing.T) map[string]string {
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

// exported reads this marker's field of exported genesis JSON. Encoders that
// omit false fields leave it out, which reads as off.
func (f *knowledgeV2MarkerFixture) exported(t *testing.T, raw json.RawMessage) bool {
	t.Helper()
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &fields))
	value, ok := fields[f.marker.field]
	if !ok {
		return false
	}
	var on bool
	require.NoError(t, json.Unmarshal(value, &on))
	return on
}

func TestKnowledgeV2MarkersNativeGenesisSelectsMarker(t *testing.T) {
	for _, m := range knowledgeV2Markers {
		t.Run(m.field, func(t *testing.T) {
			require.Equal(t, []byte{m.pinned}, m.key)
			gs := types.DefaultGenesis()
			require.True(t, *m.genesis(gs), "a new chain starts with the repair selected")
			require.NoError(t, gs.Validate())
			*m.genesis(gs) = false
			require.NoError(t, gs.Validate(), "off stays valid")

			f := newKnowledgeV2MarkerFixture(t, m)
			f.initGenesis(t, f.am.DefaultGenesis(f.cdc))
			require.Equal(t, []byte{1}, f.stored())
			require.True(t, f.enabled(t))
		})
	}
}

func TestKnowledgeV2MarkersGenesisFieldWritesOnlyTheMarker(t *testing.T) {
	enabled, disabled := true, false
	for _, m := range knowledgeV2Markers {
		t.Run(m.field, func(t *testing.T) {
			on, off := newKnowledgeV2MarkerFixture(t, m), newKnowledgeV2MarkerFixture(t, m)
			on.initGenesis(t, on.genesisJSON(t, &enabled))
			off.initGenesis(t, off.genesisJSON(t, &disabled))
			want := off.snapshot(t)
			want[string(m.key)] = "\x01"
			require.Equal(t, want, on.snapshot(t), "the genesis field writes the marker and nothing else")
		})
	}
}

// A marker that is neither absent nor {1} is a fault, so ExportGenesis panics
// rather than export it as off.
func TestKnowledgeV2MarkersExportRefusesMalformedMarker(t *testing.T) {
	for _, m := range knowledgeV2Markers {
		t.Run(m.field, func(t *testing.T) {
			f := newKnowledgeV2MarkerFixture(t, m)
			f.initGenesis(t, f.genesisJSON(t, nil))
			for _, bad := range [][]byte{{0}, {2}} {
				t.Run(fmt.Sprintf("%x", bad), func(t *testing.T) {
					f.ctx.KVStore(f.key).Set(m.key, bad)
					require.Panics(t, func() { f.am.ExportGenesis(f.ctx, f.cdc) }, "a malformed marker is never exported as off")
				})
			}
		})
	}
}

func TestKnowledgeV2MarkersGenesisRoundTripsMarker(t *testing.T) {
	on, off := true, false
	for _, m := range knowledgeV2Markers {
		for _, tc := range []struct {
			name  string
			field *bool
			want  bool
		}{
			{"on", &on, true},
			{"off", &off, false},
			{"file without the field", nil, false},
		} {
			t.Run(m.field+"/"+tc.name, func(t *testing.T) {
				source := newKnowledgeV2MarkerFixture(t, m)
				source.initGenesis(t, source.genesisJSON(t, tc.field))
				require.Equal(t, tc.want, source.enabled(t))
				if !tc.want {
					require.Nil(t, source.stored(), "off writes no marker")
				}
				exported := source.am.ExportGenesis(source.ctx, source.cdc)
				require.Equal(t, tc.want, source.exported(t, exported))

				imported := newKnowledgeV2MarkerFixture(t, m)
				imported.initGenesis(t, exported)
				require.Equal(t, tc.want, imported.enabled(t))
				require.Equal(t, tc.want, imported.exported(t, imported.am.ExportGenesis(imported.ctx, imported.cdc)))
			})
		}
	}
}
