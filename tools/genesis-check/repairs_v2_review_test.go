package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	gov "github.com/zerone-chain/zerone/x/gov/types"
	knowledge "github.com/zerone-chain/zerone/x/knowledge/types"
)

func repairsV2ReviewGenesis(t *testing.T) map[string]interface{} {
	t.Helper()
	raw, err := os.ReadFile("testdata/good-genesis.json")
	require.NoError(t, err)
	var g map[string]interface{}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	require.NoError(t, d.Decode(&g))
	return g
}

func TestFX8RetainedAliasesAllSixKeys(t *testing.T) {
	names := []string{"lips", "research_spend_proposals", "next_research_spend_number", "research_community_votes", "distinct_voters", "phase_transitions"}
	require.Equal(t, names, gov.RetainedGenesisFields())
	for _, name := range names {
		for _, alias := range []string{strings.ToUpper(name), strings.Replace(name, "s", "ſ", 1)} {
			t.Run(alias, func(t *testing.T) {
				require.NotEqual(t, name, alias, "the Unicode probe must actually change the key")
				g := repairsV2ReviewGenesis(t)
				gg, _ := digMap(g, "app_state", "zerone_gov")
				gg[alias] = nil
				c := &checker{}
				checkRepairsV2RetainedGenesis(c, g)
				require.Equal(t, 1, c.failed, "pin the retained checker independently of marker selection")
				full := &checker{profile: "testnet"}
				runChecks(full, g)
				require.Positive(t, full.failed)
			})
		}
	}
}

func TestFX8CounterHeadroom(t *testing.T) {
	for _, tc := range []struct {
		value   string
		allowed bool
	}{
		{"1", true}, {"9223372036854775808", true}, {"9223372036854775809", false}, {"18446744073709551614", false},
	} {
		t.Run(tc.value, func(t *testing.T) {
			g := repairsV2ReviewGenesis(t)
			gg, _ := digMap(g, "app_state", "zerone_gov")
			gg["next_research_spend_number"] = json.Number(tc.value)
			c := &checker{}
			checkRepairsV2RetainedGenesis(c, g)
			require.Equal(t, tc.allowed, c.failed == 0)
		})
	}
}

func TestFX7DuplicateTrainingFundBalanceRefused(t *testing.T) {
	for _, duplicate := range []bool{false, true} {
		g := repairsV2ReviewGenesis(t)
		kg, _ := digMap(g, "app_state", "knowledge")
		kg["accounting_state"] = map[string]interface{}{"training_fund_non_escrow_balance": "0"}
		kg["training_fund_allocation"] = "0"
		bank, _ := digMap(g, "app_state", "bank")
		entry := map[string]interface{}{"address": authtypes.NewModuleAddress(knowledge.TrainingFundModuleName).String(), "coins": []interface{}{}}
		bank["balances"] = append(bank["balances"].([]interface{}), entry)
		if duplicate {
			bank["balances"] = append(bank["balances"].([]interface{}), entry)
		}
		c := &checker{}
		checkRepairsV2RetainedGenesis(c, g)
		require.Equal(t, duplicate, c.failed > 0, "identical zero balances expose a missing duplicate guard")
	}
}
