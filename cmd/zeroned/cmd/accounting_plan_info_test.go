package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestAccountingPlanTupleRejectsUnboundSourceBeforeOpeningDB(t *testing.T) {
	validHash := strings.Repeat("ab", 32)
	for _, test := range []struct {
		name, chain, hash string
		height            int64
	}{
		{"missing chain", "", validHash, 1},
		{"space chain", " chain", validHash, 1},
		{"missing height", "chain", validHash, 0},
		{"negative height", "chain", validHash, -1},
		{"missing hash", "chain", "", 1},
		{"uppercase hash", "chain", strings.ToUpper(validHash), 1},
		{"short hash", "chain", "ab", 1},
		{"malformed hash", "chain", strings.Repeat("xy", 32), 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := validateAccountingPlanTuple(test.chain, test.height, test.hash); err == nil {
				t.Fatal("unbound tuple accepted")
			}
		})
	}
	if err := validateAccountingPlanTuple("chain", 1, validHash); err != nil {
		t.Fatal(err)
	}
	command := accountingPlanInfoCmd()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs(nil)
	// No application/server context is installed: missing evidence must be
	// rejected before even looking for a node database or producing a report.
	if err := command.Execute(); err == nil || !strings.Contains(err.Error(), "--expected-chain-id") {
		t.Fatalf("unexpected failure: %v", err)
	}
	if strings.Contains(output.String(), `"plan_info"`) {
		t.Fatal("failed preparation emitted plan evidence")
	}
}
