package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestForkGenesisRefusesResearchSpendInventory(t *testing.T) {
	for _, scenario := range []string{"discussion", "voting", "executed", "community-vote", "alias"} {
		t.Run(scenario, func(t *testing.T) {
			f := newFixture(t)
			mutateAppState(t, &f, func(state map[string]any) {
				gg := state["zerone_gov"].(map[string]any)
				switch scenario {
				case "community-vote":
					gg["research_community_votes"] = []any{map[string]any{"proposal_id": 1, "voter": "retained", "vote": "yes"}}
				case "alias":
					gg["RESEARCH_SPEND_PROPOSALS"] = []any{}
				default:
					gg["research_spend_proposals"] = []any{map[string]any{"proposal_id": 1, "stage": scenario}}
				}
			})
			_, _, err := compileGenesis(f.input, f.inputDigest, f.policy, f.policyFile, "did:zrn:reproducer-a")
			if scenario == "alias" {
				require.ErrorContains(t, err, "FX8 alias")
			} else {
				require.ErrorContains(t, err, "research-spend proposal/vote")
			}
		})
	}
}
