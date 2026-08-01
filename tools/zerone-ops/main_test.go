package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTestTrustPolicy(t *testing.T, transition Transition) (string, string, TrustPolicy) {
	t.Helper()
	policy := testPolicyForTransition(transition)
	document, err := json.Marshal(policy)
	if err != nil {
		t.Fatalf("marshal trust policy: %v", err)
	}
	path := filepath.Join(t.TempDir(), "trust-policy.json")
	if err := os.WriteFile(path, document, 0o600); err != nil {
		t.Fatalf("write trust policy: %v", err)
	}
	return path, mustTestPolicySHA256(t, policy), policy
}

func TestRunVerifyFromStdin(t *testing.T) {
	transition := mustSeal(t, incidentTransition(1, StateRunning, StateAssessing, ""))
	document := canonicalDocument(t, transition)
	policyPath, policySHA256, _ := writeTestTrustPolicy(t, transition)
	var stdout, stderr bytes.Buffer

	exitCode := run(
		[]string{
			"verify",
			"--trust-policy", policyPath,
			"--trust-policy-sha256", policySHA256,
			"--chain-id", "zerone-1",
			"--incident-id", "ZR-2026-0001",
			"--head-sha256", transition.TransitionSHA256,
			"-",
		},
		bytes.NewReader(document),
		&stdout,
		&stderr,
	)
	if exitCode != 0 {
		t.Fatalf("verify exit=%d stderr=%s", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "VALID transitions=1 lane=incident") {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestRunVerifyInvalidReturnsOne(t *testing.T) {
	transition := mustSeal(t, incidentTransition(1, StateRunning, StateAssessing, ""))
	document := append(canonicalDocument(t, transition), '\n')
	policyPath, policySHA256, _ := writeTestTrustPolicy(t, transition)
	var stdout, stderr bytes.Buffer

	exitCode := run(
		[]string{
			"verify",
			"--trust-policy", policyPath,
			"--trust-policy-sha256", policySHA256,
			"--head-sha256", transition.TransitionSHA256,
			"-",
		},
		bytes.NewReader(document),
		&stdout,
		&stderr,
	)
	if exitCode != 1 {
		t.Fatalf("verify exit=%d, want 1; stderr=%s", exitCode, stderr.String())
	}
	if !strings.Contains(stderr.String(), "not canonical") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestRunVerifyRequiresExternallyPinnedHead(t *testing.T) {
	transition := mustSeal(t, incidentTransition(1, StateRunning, StateAssessing, ""))
	policyPath, policySHA256, _ := writeTestTrustPolicy(t, transition)
	var stdout, stderr bytes.Buffer

	exitCode := run(
		[]string{
			"verify",
			"--trust-policy", policyPath,
			"--trust-policy-sha256", policySHA256,
			"-",
		},
		bytes.NewReader(canonicalDocument(t, transition)),
		&stdout,
		&stderr,
	)
	if exitCode != 2 ||
		!strings.Contains(stderr.String(), "--head-sha256 is required") {
		t.Fatalf("verify exit=%d stderr=%s", exitCode, stderr.String())
	}
}

func TestRunSealEmitsVerifiableCanonicalBytes(t *testing.T) {
	draft := incidentTransition(1, StateRunning, StateAssessing, "")
	draft.Approvals = []Approval{
		signApproval(t, draft, "operations-approver", "did:zrn:approver-1", 1, "0"),
	}
	document := canonicalDocument(t, draft)
	policyPath, policySHA256, policy := writeTestTrustPolicy(t, draft)
	var sealedOutput, stderr bytes.Buffer

	exitCode := run(
		[]string{
			"seal",
			"--trust-policy", policyPath,
			"--trust-policy-sha256", policySHA256,
			"--input", "-",
		},
		bytes.NewReader(document),
		&sealedOutput,
		&stderr,
	)
	if exitCode != 0 {
		t.Fatalf("seal exit=%d stderr=%s", exitCode, stderr.String())
	}
	if bytes.HasSuffix(sealedOutput.Bytes(), []byte("\n")) {
		t.Fatal("seal output must not append a noncanonical newline")
	}
	if _, err := verifyDocuments([][]byte{sealedOutput.Bytes()}, VerifyOptions{
		TrustPolicy:       &policy,
		TrustPolicySHA256: policySHA256,
	}); err != nil {
		t.Fatalf("sealed output did not verify: %v", err)
	}
}

func TestRunVerifyRequiresExternallyPinnedTrustPolicy(t *testing.T) {
	transition := mustSeal(t, incidentTransition(1, StateRunning, StateAssessing, ""))
	var stdout, stderr bytes.Buffer
	exitCode := run(
		[]string{"verify", "-"},
		bytes.NewReader(canonicalDocument(t, transition)),
		&stdout,
		&stderr,
	)
	if exitCode != 2 || !strings.Contains(stderr.String(), "--trust-policy is required") {
		t.Fatalf("verify exit=%d stderr=%s", exitCode, stderr.String())
	}
}
