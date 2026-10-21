package main

import (
	"bytes"
	"encoding/json"
	"testing"

	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/stretchr/testify/require"
	zeroneapp "github.com/zerone-chain/zerone/app"
)

func TestForkGenesisPreservesV2SelectionsWithoutActivation(t *testing.T) {
	for _, m := range zeroneapp.ConsensusRepairsV2GenesisSelections() {
		for _, missing := range []bool{false, true} {
			f := newFixture(t)
			mutateAppState(t, &f, func(state map[string]any) {
				fields := state[m.Module].(map[string]any)
				if missing {
					delete(fields, m.Name)
				} else {
					fields[m.Name] = false
				}
			})
			output, _, err := compileGenesis(f.input, f.inputDigest, f.policy, f.policyFile, "did:zrn:reproducer-a")
			require.NoError(t, err)
			genesis, err := genutiltypes.AppGenesisFromReader(bytes.NewReader(output))
			require.NoError(t, err)
			var state zeroneapp.GenesisState
			require.NoError(t, json.Unmarshal(genesis.AppState, &state))
			var fields map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(state[m.Module], &fields))
			if missing {
				require.NotContains(t, fields, m.Name)
			} else {
				require.JSONEq(t, "false", string(fields[m.Name]))
			}
			require.Error(t, zeroneapp.ValidateConsensusRepairsGenesisSelection(state), "forking keys cannot select consensus semantics")
		}
	}
}
