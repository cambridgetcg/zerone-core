package keeper_test

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

// These tests hold the consensus-repairs-v3 cited-reward escrow marker to the
// same module-side contract as the v1 qualification-semantics marker, on the
// same store fixture (qualification_semantics_activation_test.go): the key is
// pinned, native genesis selects the marker, the genesis field writes the
// marker and nothing else, genesis JSON carries it exactly with a file written
// before the field existing importing as off, a malformed marker is never
// exported as off, and the marker is never a raw state entry. The repair's own
// tests can build on it: citedRewardEscrowGenesisJSON(t, f, &off) turns the
// marker off and EnableCitedRewardEscrow turns it on.

const citedRewardEscrowGenesisField = "cited_reward_escrow_enabled"

func citedRewardEscrowStored(f *qualificationSemanticsFixture) []byte {
	return f.ctx.KVStore(f.key).Get(types.CitedRewardEscrowEnabledKey)
}

func citedRewardEscrowEnabled(t *testing.T, f *qualificationSemanticsFixture) bool {
	t.Helper()
	enabled, err := f.k.CitedRewardEscrowEnabled(f.ctx)
	require.NoError(t, err)
	return enabled
}

// citedRewardEscrowGenesisJSON is the module's default genesis JSON with the
// marker field set on, set off, or (nil) removed, as in a file written before
// the field existed. Every other field keeps its default.
func citedRewardEscrowGenesisJSON(t *testing.T, f *qualificationSemanticsFixture, field *bool) json.RawMessage {
	t.Helper()
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(f.am.DefaultGenesis(f.cdc), &fields))
	require.JSONEq(t, "true", string(fields[citedRewardEscrowGenesisField]), "native genesis selects the marker")
	if field == nil {
		delete(fields, citedRewardEscrowGenesisField)
	} else {
		fields[citedRewardEscrowGenesisField] = json.RawMessage(strconv.FormatBool(*field))
	}
	raw, err := json.Marshal(fields)
	require.NoError(t, err)
	return raw
}

// citedRewardEscrowExported reads the marker field of exported genesis JSON.
// Encoders that omit false fields leave it out, which reads as off.
func citedRewardEscrowExported(t *testing.T, raw json.RawMessage) bool {
	t.Helper()
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &fields))
	value, ok := fields[citedRewardEscrowGenesisField]
	if !ok {
		return false
	}
	var on bool
	require.NoError(t, json.Unmarshal(value, &on))
	return on
}

func TestCitedRewardEscrowGenesisFieldWritesOnlyTheMarker(t *testing.T) {
	enabled, disabled := true, false
	on, off := newQualificationSemanticsFixture(t), newQualificationSemanticsFixture(t)
	on.initGenesis(t, citedRewardEscrowGenesisJSON(t, on, &enabled))
	off.initGenesis(t, citedRewardEscrowGenesisJSON(t, off, &disabled))
	want := off.snapshot(t)
	want[string(types.CitedRewardEscrowEnabledKey)] = "\x01"
	require.Equal(t, want, on.snapshot(t), "the genesis field writes the marker and nothing else")
}

// A marker that is neither absent nor {1} is a fault, so ExportGenesis panics
// rather than export it as off.
func TestCitedRewardEscrowExportRefusesMalformedMarker(t *testing.T) {
	f := newQualificationSemanticsFixture(t)
	f.initGenesis(t, citedRewardEscrowGenesisJSON(t, f, nil))
	for _, bad := range [][]byte{{0}, {2}} {
		t.Run(fmt.Sprintf("%x", bad), func(t *testing.T) {
			f.ctx.KVStore(f.key).Set(types.CitedRewardEscrowEnabledKey, bad)
			require.Panics(t, func() { f.am.ExportGenesis(f.ctx, f.cdc) }, "a malformed marker is never exported as off")
		})
	}
}

func TestCitedRewardEscrowNativeGenesisSelectsMarker(t *testing.T) {
	require.Equal(t, []byte{0x91}, types.CitedRewardEscrowEnabledKey)
	require.NotEqual(t, types.QualificationSemanticsEnabledKey, types.CitedRewardEscrowEnabledKey)
	gs := types.DefaultGenesis()
	require.True(t, gs.CitedRewardEscrowEnabled, "a new chain starts with the repair selected")
	require.NoError(t, gs.Validate())
	gs.CitedRewardEscrowEnabled = false
	require.NoError(t, gs.Validate(), "off stays valid")

	f := newQualificationSemanticsFixture(t)
	f.initGenesis(t, f.am.DefaultGenesis(f.cdc))
	require.Equal(t, []byte{1}, citedRewardEscrowStored(f))
	require.True(t, citedRewardEscrowEnabled(t, f))
}

func TestCitedRewardEscrowGenesisRoundTripsMarker(t *testing.T) {
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
			source := newQualificationSemanticsFixture(t)
			source.initGenesis(t, citedRewardEscrowGenesisJSON(t, source, tc.field))
			require.Equal(t, tc.want, citedRewardEscrowEnabled(t, source))
			if !tc.want {
				require.Nil(t, citedRewardEscrowStored(source), "off writes no marker")
			}
			exported := source.am.ExportGenesis(source.ctx, source.cdc)
			require.Equal(t, tc.want, citedRewardEscrowExported(t, exported))

			imported := newQualificationSemanticsFixture(t)
			imported.initGenesis(t, exported)
			require.Equal(t, tc.want, citedRewardEscrowEnabled(t, imported))
			require.Equal(t, tc.want, citedRewardEscrowExported(t, imported.am.ExportGenesis(imported.ctx, imported.cdc)))
		})
	}
}

// The marker is typed genesis state: export never copies it into the raw
// state_entries dump, and validation refuses a raw entry that would set it
// around the typed field.
func TestCitedRewardEscrowMarkerIsNeverARawStateEntry(t *testing.T) {
	require.True(t, types.IsTypedGenesisStateKey(types.CitedRewardEscrowEnabledKey))
	require.False(t, types.IsAllowedGenesisStateKey(types.CitedRewardEscrowEnabledKey))

	gs := types.DefaultGenesis()
	gs.CitedRewardEscrowEnabled = false
	gs.StateEntries = []*types.GenesisStateEntry{{Key: types.CitedRewardEscrowEnabledKey, Value: []byte{1}}}
	require.ErrorContains(t, gs.Validate(), "forbidden or malformed key")

	f := newQualificationSemanticsFixture(t)
	require.ErrorContains(t, f.k.InitGenesis(f.ctx, gs), "forbidden or malformed key")
	require.Nil(t, citedRewardEscrowStored(f), "a refused import writes no marker")

	f.initGenesis(t, f.am.DefaultGenesis(f.cdc))
	exported := f.k.ExportGenesis(f.ctx)
	require.True(t, exported.CitedRewardEscrowEnabled)
	for _, entry := range exported.StateEntries {
		require.NotEqual(t, types.CitedRewardEscrowEnabledKey, entry.Key)
	}
}
