package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zerone-chain/zerone/app"
)

func TestRunChecksGoodGenesisAndMissingMarker(t *testing.T) {
	raw, err := os.ReadFile("testdata/good-genesis.json")
	require.NoError(t, err)
	for _, missing := range []bool{false, true} {
		name := "good fixture"
		if missing {
			name = "missing marker"
		}
		t.Run(name, func(t *testing.T) {
			// Decode a fresh copy in the same number-preserving mode as main.
			var genesis map[string]interface{}
			decoder := json.NewDecoder(bytes.NewReader(raw))
			decoder.UseNumber()
			require.NoError(t, decoder.Decode(&genesis))
			if missing {
				marker := app.ConsensusRepairsV1GenesisSelections()[0]
				fields, ok := digMap(genesis, "app_state", marker.Module)
				require.True(t, ok)
				require.Equal(t, true, fields[marker.Name])
				delete(fields, marker.Name)
			}
			c := &checker{profile: "testnet"}
			runChecks(c, genesis)
			if missing {
				require.Equal(t, 1, c.failed, "the full command check list must reject a missing marker")
			} else {
				require.Zero(t, c.failed)
				require.Positive(t, c.passed)
			}
		})
	}
}

func TestAdapterQualificationFailureNamesEitherIDSpelling(t *testing.T) {
	for _, field := range []string{"adapter_id", "adapterId"} {
		t.Run(field, func(t *testing.T) {
			g := makeGenesis(map[string]interface{}{"app_state.substrate_bridge.adapters": []interface{}{
				map[string]interface{}{field: "named-adapter", "requiredQualificationDomain": "physics", "minQualificationStatus": "QUALIFICATION_STATUS_DISTINGUISHED"},
			}})
			// Capture the actual CLI diagnostic, not just the field lookup helper.
			out, err := os.CreateTemp(t.TempDir(), "diagnostic")
			require.NoError(t, err)
			defer out.Close()
			stdout := os.Stdout
			os.Stdout = out
			defer func() { os.Stdout = stdout }()
			c := &checker{}
			checkAdapterQualificationMinima(c, g)
			os.Stdout = stdout
			require.Equal(t, 1, c.failed)
			raw, err := os.ReadFile(out.Name())
			require.NoError(t, err)
			require.Contains(t, string(raw), "adapter[0] named-adapter has unsupported minimum")
			require.False(t, strings.Contains(string(raw), "<nil>"))
		})
	}
}
