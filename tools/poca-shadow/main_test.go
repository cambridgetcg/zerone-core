package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/zerone-chain/zerone/tools/poca-shadow/evaluate"
)

func TestRunPublishedPartialExample(t *testing.T) {
	profilePath, evidencePath := publishedExamples(t)
	var stdout, stderr bytes.Buffer
	err := run([]string{
		"--profile", profilePath,
		"--evidence", evidencePath,
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run: %v (stderr: %s)", err, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("successful run wrote stderr: %s", stderr.String())
	}

	var certificate evaluate.Certificate
	decoder := json.NewDecoder(&stdout)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&certificate); err != nil {
		t.Fatalf("decode certificate: %v\n%s", err, stdout.String())
	}
	if certificate.CrownStatus != "BLOCKED" {
		t.Fatalf("published partial fixture must keep crown blocked, got %q", certificate.CrownStatus)
	}
	if certificate.AttainedTier != "E2_CONFORMANT" {
		t.Fatalf("published partial fixture tier: want E2_CONFORMANT, got %q", certificate.AttainedTier)
	}
	if certificate.Reward.EconomicEffect != "NONE" || certificate.Reward.AmountUzrn != "0" {
		t.Fatalf("published partial fixture emitted reward: %#v", certificate.Reward)
	}
}

func TestPublishedKnownAnswerVectors(t *testing.T) {
	profilePath, evidencePath := publishedExamples(t)
	profileBytes, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatal(err)
	}
	evidenceBytes, err := os.ReadFile(evidencePath)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := evaluate.ParseProfile(profileBytes)
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := evaluate.ParseEvidence(evidenceBytes)
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := evaluate.Evaluate(profile, evidence)
	if err != nil {
		t.Fatal(err)
	}
	statement, err := evaluate.EvaluateInToto(profile, evidence)
	if err != nil {
		t.Fatal(err)
	}
	certificateJSON, err := json.Marshal(certificate)
	if err != nil {
		t.Fatal(err)
	}
	statementJSON, err := json.Marshal(statement)
	if err != nil {
		t.Fatal(err)
	}

	assertEqual := func(name, got, want string) {
		t.Helper()
		if got != want {
			t.Fatalf("%s vector drift: want %s, got %s", name, want, got)
		}
	}
	assertEqual("profile digest", certificate.Profile.Digest, "sha256:1ca9318cca35d231636b48d8a11e498bebb64b4eeaf3aa0b64051e3138214c8b")
	assertEqual("evidence digest", certificate.EvidenceBundleDigest, "sha256:1a71481fc5a0d0789563a14a07c95a8af10de252f8bc88e5ca72f1328125663f")
	assertEqual("claim id", certificate.ClaimID, "sha256:e9f7f0b62637b9f62b1360239c8c68c6e8c19cbf176da0fe2ef8a28e4e88824f")
	assertEqual("certificate JSON sha256", fmt.Sprintf("%x", sha256.Sum256(certificateJSON)), "d4d73eacd41586a4d3d6e6d8b42a235dc16634866b557f3aa27c4e69ddb9ca65")
	assertEqual("in-toto JSON sha256", fmt.Sprintf("%x", sha256.Sum256(statementJSON)), "d4bae2b6268d680a3c9a03658bb751cac48bb0b4a77b05a411ee096339d4c3b5")
}

func TestRunRequireCrownFailsClosed(t *testing.T) {
	profilePath, evidencePath := publishedExamples(t)
	certificate := evaluatePublishedExamples(t)
	var stdout, stderr bytes.Buffer
	err := run([]string{
		"--profile", profilePath,
		"--evidence", evidencePath,
		"--require-crown",
		"--expect-profile-digest", certificate.Profile.Digest,
	}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "crown") || !strings.Contains(err.Error(), "BLOCKED") {
		t.Fatalf("expected blocked-crown error, got %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("failed crown gate wrote %d output bytes", stdout.Len())
	}

	stdout.Reset()
	stderr.Reset()
	err = run([]string{
		"--profile", profilePath,
		"--evidence", evidencePath,
		"--require-crown",
	}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "--expect-profile-digest") {
		t.Fatalf("expected unpinned crown-gate rejection, got %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	err = run([]string{
		"--profile", profilePath,
		"--evidence", evidencePath,
		"--expect-profile-digest", "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
	}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "profile digest mismatch") {
		t.Fatalf("expected profile-digest mismatch, got %v", err)
	}
}

func TestRunEmitsUnsignedInTotoProjection(t *testing.T) {
	profilePath, evidencePath := publishedExamples(t)
	var stdout, stderr bytes.Buffer
	err := run([]string{
		"--profile", profilePath,
		"--evidence", evidencePath,
		"--format", "in-toto",
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run: %v (stderr: %s)", err, stderr.String())
	}
	var statement evaluate.InTotoStatement
	if err := json.Unmarshal(stdout.Bytes(), &statement); err != nil {
		t.Fatalf("decode statement: %v\n%s", err, stdout.String())
	}
	if statement.Type != evaluate.InTotoStatementV1 ||
		statement.PredicateType != evaluate.PredicateTypeV0 ||
		statement.Predicate.Reward.AmountUzrn != "0" {
		t.Fatalf("unexpected in-toto projection: %#v", statement)
	}
}

func TestRunRejectsMissingAndPositionalInputs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run(nil, &stdout, &stderr); err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatalf("expected required-input error, got %v", err)
	}
	if err := run([]string{"profile.json"}, &stdout, &stderr); err == nil || !strings.Contains(err.Error(), "positional") {
		t.Fatalf("expected positional-input error, got %v", err)
	}
	if err := run([]string{"--profile", "x", "--evidence", "y", "--format", "xml"}, &stdout, &stderr); err == nil || !strings.Contains(err.Error(), "--format") {
		t.Fatalf("expected output-format error, got %v", err)
	}
}

func TestReadBoundedRegularFileRejectsSymlinkAndOversize(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.json")
	if err := os.WriteFile(target, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.json")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := readBoundedRegularFile(link); err == nil || !strings.Contains(err.Error(), "regular file") {
		t.Fatalf("expected symlink rejection, got %v", err)
	}

	oversized := filepath.Join(dir, "oversized.json")
	if err := os.WriteFile(oversized, make([]byte, maxInputBytes+1), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readBoundedRegularFile(oversized); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected oversize rejection, got %v", err)
	}
}

func publishedExamples(t *testing.T) (string, string) {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	return filepath.Join(root, "docs", "examples", "poca", "slsa-build-l2-v0.profile.json"),
		filepath.Join(root, "docs", "examples", "poca", "zerone-release-partial-v0.evidence.json")
}

func evaluatePublishedExamples(t *testing.T) evaluate.Certificate {
	t.Helper()
	profilePath, evidencePath := publishedExamples(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{
		"--profile", profilePath,
		"--evidence", evidencePath,
	}, &stdout, &stderr); err != nil {
		t.Fatalf("evaluate published examples: %v (stderr: %s)", err, stderr.String())
	}
	var certificate evaluate.Certificate
	if err := json.Unmarshal(stdout.Bytes(), &certificate); err != nil {
		t.Fatalf("decode published certificate: %v", err)
	}
	return certificate
}
