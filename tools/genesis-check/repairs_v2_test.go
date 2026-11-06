package main

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zerone-chain/zerone/app"
)

func TestRunChecksV2MarkersAndRetainedFields(t *testing.T) {
	raw, err := os.ReadFile("testdata/good-genesis.json")
	require.NoError(t, err)
	for _, scenario := range []string{"good", "marker", "fx7-snake", "fx7-camel", "fx8-zero", "fx8-max", "fx8-alias", "fx8-orphan"} {
		t.Run(scenario, func(t *testing.T) {
			var g map[string]interface{}
			d := json.NewDecoder(bytes.NewReader(raw))
			d.UseNumber()
			require.NoError(t, d.Decode(&g))
			kg, _ := digMap(g, "app_state", "knowledge")
			gg, _ := digMap(g, "app_state", "zerone_gov")
			switch scenario {
			case "marker":
				m := app.ConsensusRepairsV2GenesisSelections()[0]
				fields, _ := digMap(g, "app_state", m.Module)
				delete(fields, m.Name)
			case "fx7-snake":
				kg["accounting_state"] = map[string]interface{}{"training_fund_non_escrow_balance": "0"}
				kg["training_fund_allocation"] = "1"
			case "fx7-camel":
				kg["accountingState"] = map[string]interface{}{"trainingFundNonEscrowBalance": "1"}
				kg["trainingFundAllocation"] = "1"
			case "fx8-zero":
				gg["next_research_spend_number"] = json.Number("0")
			case "fx8-max":
				gg["next_research_spend_number"] = json.Number("18446744073709551615")
			case "fx8-alias":
				gg["nextResearchSpendNumber"] = json.Number("0")
			case "fx8-orphan":
				gg["research_community_votes"] = []interface{}{map[string]interface{}{"proposal_id": 1, "voter": "absent", "vote": "yes"}}
			}
			c := &checker{profile: "testnet"}
			runChecks(c, g)
			if scenario == "good" {
				require.Zero(t, c.failed)
			} else {
				require.Positive(t, c.failed, "full command must run retained-inventory and v2 selection checks")
			}
		})
	}
}
