package keeper_test

import (
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

// These tests hold the consensus-repairs-v1 marker for zerone-core #118 to the
// module's side of the activation contract (internal/activation tests the reads
// and writes): the key is pinned, native genesis selects the marker, the
// genesis field writes the marker and nothing else, genesis JSON carries it
// exactly with a file written before the field existing importing as off, and
// a malformed marker is never exported as off. Every import and export goes
// through the module's own genesis JSON path. The repair's own tests can build
// on the fixture: genesisJSON(t, &off) turns the marker off and
// EnableReformulationVindicationOrdering turns it on.

const reformulationVindicationOrderingGenesisField = "reformulation_vindication_ordering_enabled"

type reformulationVindicationOrderingFixture struct {
	ctx sdk.Context
	key storetypes.StoreKey
	k   keeper.Keeper
	am  knowledgemodule.AppModule
	cdc codec.JSONCodec
}

func newReformulationVindicationOrderingFixture(t *testing.T) *reformulationVindicationOrderingFixture {
	t.Helper()
	f := newKnowledgeStoreFixture(t, "consensus-repairs-v1-marker")
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	return &reformulationVindicationOrderingFixture{ctx: f.ctx, key: f.key, k: f.k, am: knowledgemodule.NewAppModule(cdc, f.k), cdc: cdc}
}

func (f *reformulationVindicationOrderingFixture) marker() []byte {
	return f.ctx.KVStore(f.key).Get(types.ReformulationVindicationOrderingEnabledKey)
}

func (f *reformulationVindicationOrderingFixture) enabled(t *testing.T) bool {
	t.Helper()
	enabled, err := f.k.ReformulationVindicationOrderingEnabled(f.ctx)
	require.NoError(t, err)
	return enabled
}

func (f *reformulationVindicationOrderingFixture) initGenesis(t *testing.T, raw json.RawMessage) {
	t.Helper()
	require.NoError(t, f.am.ValidateGenesis(f.cdc, nil, raw))
	f.am.InitGenesis(f.ctx, f.cdc, raw)
}

// genesisJSON is the module's default genesis JSON with the marker field set
// on, set off, or (nil) removed, as in a file written before the field existed.
func (f *reformulationVindicationOrderingFixture) genesisJSON(t *testing.T, field *bool) json.RawMessage {
	t.Helper()
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(f.am.DefaultGenesis(f.cdc), &fields))
	require.JSONEq(t, "true", string(fields[reformulationVindicationOrderingGenesisField]), "native genesis selects the marker")
	if field == nil {
		delete(fields, reformulationVindicationOrderingGenesisField)
	} else {
		fields[reformulationVindicationOrderingGenesisField] = json.RawMessage(strconv.FormatBool(*field))
	}
	raw, err := json.Marshal(fields)
	require.NoError(t, err)
	return raw
}

func (f *reformulationVindicationOrderingFixture) snapshot(t *testing.T) map[string]string {
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

// reformulationVindicationOrderingExported reads the marker field of exported genesis JSON. Encoders that
// omit false fields leave it out, which reads as off.
func reformulationVindicationOrderingExported(t *testing.T, raw json.RawMessage) bool {
	t.Helper()
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &fields))
	value, ok := fields[reformulationVindicationOrderingGenesisField]
	if !ok {
		return false
	}
	var on bool
	require.NoError(t, json.Unmarshal(value, &on))
	return on
}

func TestReformulationVindicationOrderingGenesisFieldWritesOnlyTheMarker(t *testing.T) {
	enabled, disabled := true, false
	on, off := newReformulationVindicationOrderingFixture(t), newReformulationVindicationOrderingFixture(t)
	on.initGenesis(t, on.genesisJSON(t, &enabled))
	off.initGenesis(t, off.genesisJSON(t, &disabled))
	want := off.snapshot(t)
	want[string(types.ReformulationVindicationOrderingEnabledKey)] = "\x01"
	require.Equal(t, want, on.snapshot(t), "the genesis field writes the marker and nothing else")
}

// A marker that is neither absent nor {1} is a fault, so ExportGenesis panics
// rather than export it as off.
func TestReformulationVindicationOrderingExportRefusesMalformedMarker(t *testing.T) {
	f := newReformulationVindicationOrderingFixture(t)
	f.initGenesis(t, f.genesisJSON(t, nil))
	for _, bad := range [][]byte{{0}, {2}} {
		t.Run(fmt.Sprintf("%x", bad), func(t *testing.T) {
			f.ctx.KVStore(f.key).Set(types.ReformulationVindicationOrderingEnabledKey, bad)
			require.Panics(t, func() { f.am.ExportGenesis(f.ctx, f.cdc) }, "a malformed marker is never exported as off")
		})
	}
}

func TestReformulationVindicationOrderingNativeGenesisSelectsMarker(t *testing.T) {
	require.Equal(t, []byte{0x90}, types.ReformulationVindicationOrderingEnabledKey)
	gs := types.DefaultGenesis()
	require.True(t, gs.ReformulationVindicationOrderingEnabled, "a new chain starts with the repair selected")
	require.NoError(t, gs.Validate())
	gs.ReformulationVindicationOrderingEnabled = false
	require.NoError(t, gs.Validate(), "off stays valid")

	f := newReformulationVindicationOrderingFixture(t)
	f.initGenesis(t, f.am.DefaultGenesis(f.cdc))
	require.Equal(t, []byte{1}, f.marker())
	require.True(t, f.enabled(t))
}

func TestReformulationVindicationOrderingGenesisRoundTripsMarker(t *testing.T) {
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
			source := newReformulationVindicationOrderingFixture(t)
			source.initGenesis(t, source.genesisJSON(t, tc.field))
			require.Equal(t, tc.want, source.enabled(t))
			if !tc.want {
				require.Nil(t, source.marker(), "off writes no marker")
			}
			exported := source.am.ExportGenesis(source.ctx, source.cdc)
			require.Equal(t, tc.want, reformulationVindicationOrderingExported(t, exported))

			imported := newReformulationVindicationOrderingFixture(t)
			imported.initGenesis(t, exported)
			require.Equal(t, tc.want, imported.enabled(t))
			require.Equal(t, tc.want, reformulationVindicationOrderingExported(t, imported.am.ExportGenesis(imported.ctx, imported.cdc)))
		})
	}
}
