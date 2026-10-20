package main

import (
	"encoding/json"
	"testing"
)

func TestAdapterQualificationMinima(t *testing.T) {
	for _, tc := range []struct {
		minimum interface{}
		valid   bool
	}{
		{"QUALIFICATION_STATUS_UNSPECIFIED", true}, {"QUALIFICATION_STATUS_PROBATIONARY", true}, {"QUALIFICATION_STATUS_ACTIVE", true},
		{json.Number("0"), true}, {json.Number("1"), true}, {json.Number("2"), true},
		{"QUALIFICATION_STATUS_DISTINGUISHED", false}, {json.Number("3"), false}, {json.Number("99"), false}, {json.Number("1.5"), false}, {"UNKNOWN", false}, {nil, false}, {true, false},
	} {
		g := makeGenesis(map[string]interface{}{"app_state.substrate_bridge.adapters": []interface{}{map[string]interface{}{"adapter_id": "test", "required_qualification_domain": "physics", "min_qualification_status": tc.minimum}}})
		c := &checker{}
		checkAdapterQualificationMinima(c, g)
		if (c.failed == 0) != tc.valid {
			t.Errorf("minimum %v: failures=%d", tc.minimum, c.failed)
		}
	}
	c := &checker{}
	checkAdapterQualificationMinima(c, makeGenesis(map[string]interface{}{"app_state.substrate_bridge.adapters": []interface{}{map[string]interface{}{"adapter_id": "open", "required_qualification_domain": "", "min_qualification_status": "UNKNOWN"}}}))
	if c.failed != 0 {
		t.Fatal("open adapters do not impose a qualification floor")
	}
}
