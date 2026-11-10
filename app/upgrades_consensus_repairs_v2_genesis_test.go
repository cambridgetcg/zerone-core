package app

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/stretchr/testify/require"
	gov "github.com/zerone-chain/zerone/x/gov/types"
)

func TestConsensusRepairsV2FX8GenesisAliasesRefusedAtInitChain(t *testing.T) {
	names := []string{"lips", "research_spend_proposals", "next_research_spend_number", "research_community_votes", "distinct_voters", "phase_transitions"}
	require.Equal(t, names, gov.RetainedGenesisFields(), "pin the six-field contract independently of production")
	for _, name := range names {
		for _, alias := range []string{strings.ToUpper(name), consensusRepairsCamelCase(name), strings.Replace(name, "s", "ſ", 1)} {
			if alias == name {
				continue // lips has no separate camelCase spelling.
			}
			t.Run(name+"/"+alias, func(t *testing.T) {
				a := NewZeroneApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()))
				g := sdk053IBC10GenesisWithValidator(t, a)
				var fields map[string]json.RawMessage
				require.NoError(t, json.Unmarshal(g["zerone_gov"], &fields))
				value := json.RawMessage(`[]`)
				if name == "next_research_spend_number" {
					value = json.RawMessage(`2`)
				}
				fields[name] = value
				raw, err := json.Marshal(fields)
				require.NoError(t, err)
				// Canonical first and alias last reproduces encoding/json's
				// last-key-wins bypass, including Unicode simple folding.
				g["zerone_gov"] = json.RawMessage(fmt.Sprintf(`%s,%q:null}`, raw[:len(raw)-1], alias))
				require.ErrorContains(t, ValidateConsensusRepairsGenesisSelection(g), "FX8 alias")
				encoded, err := json.Marshal(g)
				require.NoError(t, err)
				ctx := a.NewUncachedContext(false, cmtproto.Header{})
				before := claimIntegrityAllStores(t, a, ctx)
				_, err = a.InitChainer(ctx, &abci.RequestInitChain{AppStateBytes: encoded})
				require.ErrorContains(t, err, "FX8 alias")
				require.Equal(t, before, claimIntegrityAllStores(t, a, ctx), "selection fails before module import")
			})
		}
	}
}
