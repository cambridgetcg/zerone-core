package main

import (
	"bytes"
	"encoding/json"
	"testing"

	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/stretchr/testify/require"
)

// The fork compiler is consensus-key-only: it must preserve the source marker,
// not silently select new behavior just because DefaultGenesis now selects it.
func TestForkGenesisPreservesReformulationMarker(t *testing.T) {
	const field = "reformulation_vindication_ordering_enabled"
	for _, state := range []string{"absent", "off", "on"} {
		t.Run(state, func(t *testing.T) {
			f := newFixture(t)
			mutateAppState(t, &f, func(app map[string]any) {
				knowledge := app["knowledge"].(map[string]any)
				delete(knowledge, field)
				if state != "absent" {
					knowledge[field] = state == "on"
				}
			})
			output, _, err := compileGenesis(f.input, f.inputDigest, f.policy, f.policyFile, "did:zrn:reproducer-a")
			require.NoError(t, err)
			genesis, err := genutiltypes.AppGenesisFromReader(bytes.NewReader(output))
			require.NoError(t, err)
			var app map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(genesis.AppState, &app))
			var knowledge map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(app["knowledge"], &knowledge))
			value, present := knowledge[field]
			require.Equal(t, state != "absent", present)
			if present {
				require.Equal(t, state == "on", string(value) == "true")
			}
		})
	}
}
