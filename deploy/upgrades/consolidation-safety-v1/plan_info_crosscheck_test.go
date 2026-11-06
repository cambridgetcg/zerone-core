//go:build zerone_h1_crosscheck

// Cross-check for h1-proposal.py's canonical plan.Info port (review finding H1-F2).
//
// Never part of a build. Copy this file into the app/ directory of a FRESH
// build-source export (provenance.py --dry-run writes one under src/) and run:
//
//	PLAN_INFO_CORPUS=/abs/path/plan-info-corpus.json \
//	  go test -tags zerone_h1_crosscheck -count=1 -run TestPlanInfoCrossCheck ./app
//
// It feeds every corpus case to the REAL validateCanonicalConsolidationPlanInfo
// at 65c19cd8 and fails if any verdict differs from the one recorded in the
// corpus. test_plan_info.py checks the Python port against the same verdicts.
// Set PLAN_INFO_VERDICTS to also write the Go verdicts to a new file.
package app

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"
)

func TestPlanInfoCrossCheck(t *testing.T) {
	raw, err := os.ReadFile(os.Getenv("PLAN_INFO_CORPUS"))
	if err != nil {
		t.Fatal(err)
	}
	var corpus struct {
		Cases []struct {
			Info string `json:"info_b64"`
			OK   bool   `json:"ok"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &corpus); err != nil {
		t.Fatal(err)
	}
	if len(corpus.Cases) == 0 {
		t.Fatal("empty corpus")
	}
	type verdict struct {
		Info  string `json:"info_b64"`
		OK    bool   `json:"ok"`
		Error string `json:"go_error"`
	}
	verdicts := make([]verdict, 0, len(corpus.Cases))
	for i, item := range corpus.Cases {
		info, err := base64.StdEncoding.DecodeString(item.Info)
		if err != nil {
			t.Fatal(err)
		}
		got := verdict{Info: item.Info, OK: true}
		if err := validateCanonicalConsolidationPlanInfo(string(info)); err != nil {
			got.OK, got.Error = false, err.Error()
		}
		if got.OK != item.OK {
			t.Errorf("case %d: Go verdict %t differs from recorded %t (%s)", i, got.OK, item.OK, got.Error)
		}
		verdicts = append(verdicts, got)
	}
	if path := os.Getenv("PLAN_INFO_VERDICTS"); path != "" {
		body, _ := json.MarshalIndent(verdicts, "", " ")
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
