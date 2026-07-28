package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunRejectsPositionalArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"bundle.json"}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "positional arguments") {
		t.Fatalf("expected positional-argument error, got %v", err)
	}
}

func TestRunFailsClosedWhenRequiredPolicyIsMissing(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run(nil, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "bundle path must be non-empty") {
		t.Fatalf("expected missing-policy error, got %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("failed verification wrote %d bytes to stdout", stdout.Len())
	}
}
