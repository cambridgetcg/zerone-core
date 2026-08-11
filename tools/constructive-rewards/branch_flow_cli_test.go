package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/zerone-chain/zerone/tools/constructive-rewards/branchflow"
)

func TestBranchFlowCLITextMakesEffectBoundaryExplicit(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := run([]string{"-mode", "branch-flow"}, &stdout, &stderr); code != 0 {
		t.Fatalf("run code=%d stderr=%s", code, stderr.String())
	}
	for _, expected := range []string{
		"assurance=SHADOW_ONLY",
		"economic_effect=NONE",
		"moves_funds=false",
		"integration_ready=false",
		"funded_milestone=E5",
		"disposition=CONSUMED_ON_SUCCESSFUL_EVALUATION",
		"projected_paid_uzrn=82500000",
		"projected_commons_uzrn=17500000",
		"conservation=EXACT_BALANCED",
	} {
		if !strings.Contains(stdout.String(), expected) {
			t.Fatalf("output missing %q:\n%s", expected, stdout.String())
		}
	}
}

func TestBranchFlowCLIJSONIsExactShadowResult(t *testing.T) {
	requestJSON, err := json.Marshal(referenceBranchFlowRequest("100000000"))
	if err != nil {
		t.Fatalf("marshal CLI request: %v", err)
	}
	wantRequest, err := os.ReadFile("branchflow/testdata/reference_request.json")
	if err != nil {
		t.Fatalf("read golden request: %v", err)
	}
	if !bytes.Equal(requestJSON, bytes.TrimSpace(wantRequest)) {
		t.Fatalf("CLI request drifted from golden bytes:\nwant %s\n got %s", bytes.TrimSpace(wantRequest), requestJSON)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := run(
		[]string{"-mode", "branch-flow", "-format", "json", "-branch-envelope-uzrn", "100000000"},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf("run code=%d stderr=%s", code, stderr.String())
	}
	var result branchflow.Result
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode result: %v\n%s", err, stdout.String())
	}
	if result.EconomicEffect != branchflow.EconomicEffect || result.MovesFunds || result.IntegrationReady {
		t.Fatalf("effect fence drift: %+v", result)
	}
	if result.Policy.PolicyDigest != branchflow.ReferencePolicyDigest {
		t.Fatalf("policy digest=%q", result.Policy.PolicyDigest)
	}
	if result.ProjectedPaidUzrn != "82500000" || result.ProjectedCommonsUzrn != "17500000" {
		t.Fatalf("unexpected projection: paid=%s commons=%s", result.ProjectedPaidUzrn, result.ProjectedCommonsUzrn)
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, stdout.Bytes()); err != nil {
		t.Fatalf("compact CLI result: %v", err)
	}
	want, err := os.ReadFile("branchflow/testdata/reference_result.json")
	if err != nil {
		t.Fatalf("read golden result: %v", err)
	}
	if !bytes.Equal(compact.Bytes(), bytes.TrimSpace(want)) {
		t.Fatalf("CLI result drifted from golden bytes:\nwant %s\n got %s", bytes.TrimSpace(want), compact.Bytes())
	}
}

func TestBranchFlowCLIInvalidEnvelopeFailsClosed(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := run(
		[]string{"-mode", "branch-flow", "-branch-envelope-uzrn", "01"},
		&stdout,
		&stderr,
	); code != 2 {
		t.Fatalf("run code=%d, want 2", code)
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), branchflow.CodeInvalidAmount) {
		t.Fatalf("unexpected output stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestBranchFlowCLIRejectsIrrelevantFlags(t *testing.T) {
	for _, arguments := range [][]string{
		{"-mode", "branch-flow", "-budget", "1"},
		{"-mode", "branch-flow", "-alpha", "0.5"},
		{"-mode", "branch-flow", "-controller-cap", "0.01"},
		{"-mode", "report", "-branch-envelope-uzrn", "1"},
	} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		if code := run(arguments, &stdout, &stderr); code != 2 {
			t.Fatalf("run(%v) code=%d, want 2", arguments, code)
		}
		if stdout.Len() != 0 || !strings.Contains(stderr.String(), "valid") {
			t.Fatalf("run(%v) stdout=%q stderr=%q", arguments, stdout.String(), stderr.String())
		}
	}
}
