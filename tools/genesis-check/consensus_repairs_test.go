package main

import (
	"strings"
	"testing"

	"github.com/zerone-chain/zerone/app"
)

func markerGenesis() map[string]interface{} {
	state := make(map[string]interface{})
	for _, marker := range append(app.ConsensusRepairsV1GenesisSelections(), app.ConsensusRepairsV2GenesisSelections()...) {
		if state[marker.Module] == nil {
			state[marker.Module] = map[string]interface{}{}
		}
		state[marker.Module].(map[string]interface{})[marker.Name] = true
	}
	return map[string]interface{}{"app_state": state}
}

func TestConsensusRepairsGenesisMarkers(t *testing.T) {
	c := &checker{}
	checkConsensusRepairsMarkers(c, markerGenesis())
	if c.failed != 0 {
		t.Fatal("all fourteen exact true selections must pass")
	}
	for _, marker := range append(app.ConsensusRepairsV1GenesisSelections(), app.ConsensusRepairsV2GenesisSelections()...) {
		for _, invalid := range []string{"absent", "false", "null", "case alias", "camel alias", "null module"} {
			t.Run(marker.Module+"/"+invalid, func(t *testing.T) {
				g := markerGenesis()
				state := g["app_state"].(map[string]interface{})
				fields := state[marker.Module].(map[string]interface{})
				switch invalid {
				case "absent":
					delete(fields, marker.Name)
				case "false":
					fields[marker.Name] = false
				case "null":
					fields[marker.Name] = nil
				case "case alias":
					fields[strings.ToUpper(marker.Name)] = false
				case "camel alias":
					parts := strings.Split(marker.Name, "_")
					for i := 1; i < len(parts); i++ {
						parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
					}
					fields[strings.Join(parts, "")] = false
				case "null module":
					state[marker.Module] = nil
				}
				c := &checker{}
				checkConsensusRepairsMarkers(c, g)
				if c.failed == 0 {
					t.Fatal("invalid marker selection passed preflight")
				}
			})
		}
	}
}
