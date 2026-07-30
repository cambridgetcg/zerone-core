package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/zerone-chain/zerone/tools/constructive-receipts/bridge"
)

func TestRunPublishedRefusalFixture(t *testing.T) {
	paths := publishedPaths(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{
		"--request", paths.request,
		"--tree", paths.tree,
		"--profile", paths.profile,
		"--evidence", paths.evidence,
	}, &stdout, &stderr); err != nil {
		t.Fatalf("run: %v (stderr: %s)", err, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("successful run wrote stderr: %s", stderr.String())
	}

	var got bridge.Receipt
	decoder := json.NewDecoder(&stdout)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&got); err != nil {
		t.Fatalf("decode output: %v\n%s", err, stdout.String())
	}
	var want bridge.Receipt
	fixture, err := os.ReadFile(paths.receipt)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(fixture, &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CLI fixture drift\nwant: %#v\ngot:  %#v", want, got)
	}
}

func TestRunRejectsMissingAndPositionalInputs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run(nil, &stdout, &stderr); err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatalf("expected required-input error, got %v", err)
	}
	if err := run([]string{"request.json"}, &stdout, &stderr); err == nil || !strings.Contains(err.Error(), "positional") {
		t.Fatalf("expected positional-input error, got %v", err)
	}
}

func TestRunRejectsUnknownPoCAFields(t *testing.T) {
	paths := publishedPaths(t)
	profile, err := os.ReadFile(paths.profile)
	if err != nil {
		t.Fatal(err)
	}
	mutated := strings.Replace(string(profile), `"title":`, `"caller_certificate":{},"title":`, 1)
	mutatedPath := filepath.Join(t.TempDir(), "profile.json")
	if err := os.WriteFile(mutatedPath, []byte(mutated), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	err = run([]string{
		"--request", paths.request,
		"--tree", paths.tree,
		"--profile", mutatedPath,
		"--evidence", paths.evidence,
	}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected strict PoCA profile rejection, got %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("failed evaluation wrote %d output bytes", stdout.Len())
	}
}

func TestReadBoundedRegularFileRejectsSymlinkAndOversize(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "target.json")
	if err := os.WriteFile(target, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(directory, "link.json")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := readBoundedRegularFile(link); err == nil || !strings.Contains(err.Error(), "regular file") {
		t.Fatalf("expected symlink rejection, got %v", err)
	}

	oversized := filepath.Join(directory, "oversized.json")
	if err := os.WriteFile(oversized, make([]byte, maxInputBytes+1), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readBoundedRegularFile(oversized); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected oversize rejection, got %v", err)
	}
}

type fixturePaths struct {
	request  string
	tree     string
	profile  string
	evidence string
	receipt  string
}

func publishedPaths(t *testing.T) fixturePaths {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	return fixturePaths{
		request:  filepath.Join(root, "docs", "examples", "constructive-receipts", "zerone-release-partial-v0.request.json"),
		tree:     filepath.Join(root, "dashboard", "public", "standards", "constructive-intelligence-tree.v1.json"),
		profile:  filepath.Join(root, "docs", "examples", "poca", "slsa-build-l2-v0.profile.json"),
		evidence: filepath.Join(root, "docs", "examples", "poca", "zerone-release-partial-v0.evidence.json"),
		receipt:  filepath.Join(root, "docs", "examples", "constructive-receipts", "zerone-release-partial-v0.receipt.json"),
	}
}
