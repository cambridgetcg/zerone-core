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

func TestAdapterQualificationMinimaJSONSpellings(t *testing.T) {
	for _, domain := range []string{"required_qualification_domain", "requiredQualificationDomain"} {
		for _, minimum := range []string{"min_qualification_status", "minQualificationStatus"} {
			for _, value := range []string{"QUALIFICATION_STATUS_ACTIVE", "QUALIFICATION_STATUS_DISTINGUISHED"} {
				t.Run(domain+"/"+minimum+"/"+value, func(t *testing.T) {
					g := makeGenesis(map[string]interface{}{"app_state.substrate_bridge.adapters": []interface{}{
						map[string]interface{}{"adapterId": "x", domain: "physics", minimum: value},
					}})
					c := &checker{}
					checkAdapterQualificationMinima(c, g)
					if (c.failed == 0) != (value == "QUALIFICATION_STATUS_ACTIVE") {
						t.Fatalf("value %s: failures=%d", value, c.failed)
					}
				})
			}
		}
	}
}

func TestAdapterQualificationMinimaRejectsDuplicateSpellings(t *testing.T) {
	for _, fields := range []map[string]interface{}{
		{"required_qualification_domain": "", "requiredQualificationDomain": "physics", "minQualificationStatus": "QUALIFICATION_STATUS_DISTINGUISHED"},
		{"required_qualification_domain": "physics", "requiredQualificationDomain": "physics"},
		{"requiredQualificationDomain": "physics", "min_qualification_status": "1", "minQualificationStatus": "1"},
		{"required_qualification_domain": "", "min_qualification_status": "1", "minQualificationStatus": "3"},
	} {
		g := makeGenesis(map[string]interface{}{"app_state.substrate_bridge.adapters": []interface{}{fields}})
		c := &checker{}
		checkAdapterQualificationMinima(c, g)
		if c.failed == 0 {
			t.Fatalf("duplicate spellings accepted: %v", fields)
		}
	}
}
