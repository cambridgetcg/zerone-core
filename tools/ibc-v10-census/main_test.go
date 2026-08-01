package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("forced write failure")
}

func TestRunExitThresholdsAndJSONEvidence(t *testing.T) {
	data := marshalFixture(t, cleanFixture())
	baseArgs := []string{
		"--input", "-",
		"--export-height", "123",
		"--app-hash", strings.Repeat("ab", 32),
		"--format", "json",
	}
	tests := []struct {
		name   string
		failOn string
		want   int
	}{
		{name: "errors only", failOn: "error", want: 1},
		{name: "warnings", failOn: "warning", want: 1},
		{name: "never", failOn: "never", want: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			args := append(append([]string{}, baseArgs...), "--fail-on", test.failOn)
			code := run(args, bytes.NewReader(data), &stdout, &stderr)
			if code != test.want {
				t.Fatalf("exit = %d, want %d; stderr=%s", code, test.want, stderr.String())
			}
			var report Report
			if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
				t.Fatalf("decode report: %v\n%s", err, stdout.String())
			}
			if report.Evidence.ExportHeight != "123" ||
				report.Evidence.AppHash != strings.Repeat("AB", 32) {
				t.Fatalf("evidence = %+v", report.Evidence)
			}
			if len(report.Evidence.InputSHA256) != 64 {
				t.Fatalf("input SHA-256 = %q", report.Evidence.InputSHA256)
			}
			if report.Complete || report.UpgradeReady {
				t.Fatalf("export-only report must remain incomplete: %+v", report)
			}
		})
	}
}

func TestRunUsageAndSchemaFailuresExitTwo(t *testing.T) {
	validArgs := []string{
		"--input", "-",
		"--export-height", "123",
		"--app-hash", strings.Repeat("AA", 32),
	}
	tests := []struct {
		name string
		args []string
		data []byte
		want string
	}{
		{
			name: "height required",
			args: []string{"--input", "-", "--app-hash", strings.Repeat("AA", 32)},
			data: marshalFixture(t, cleanFixture()),
			want: "--export-height is required",
		},
		{
			name: "canonical height",
			args: []string{"--input", "-", "--export-height", "001", "--app-hash", strings.Repeat("AA", 32)},
			data: marshalFixture(t, cleanFixture()),
			want: "canonical positive uint64",
		},
		{
			name: "app hash required",
			args: []string{"--input", "-", "--export-height", "123"},
			data: marshalFixture(t, cleanFixture()),
			want: "exactly 64 hexadecimal",
		},
		{
			name: "schema remains fatal with never threshold",
			args: append(append([]string{}, validArgs...), "--fail-on", "never"),
			data: []byte(`{"app_state":{}}`),
			want: "missing the auth module",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(test.args, bytes.NewReader(test.data), &stdout, &stderr)
			if code != 2 {
				t.Fatalf("exit = %d, want 2; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
			}
			if !strings.Contains(stderr.String(), test.want) {
				t.Fatalf("stderr = %q, want substring %q", stderr.String(), test.want)
			}
		})
	}
}

func TestRunAcceptsCompleteAppStateObject(t *testing.T) {
	fixture := cleanFixture()
	data, err := json.Marshal(appState(fixture))
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := run([]string{
		"--input", "-",
		"--export-height", "9",
		"--app-hash", strings.Repeat("0f", 32),
		"--fail-on", "error",
	}, bytes.NewReader(data), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit = %d, want mandatory old-DB refusal 1; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Zerone IBC-Go v10 preflight census") {
		t.Fatalf("unexpected text output: %s", stdout.String())
	}
}

func TestRunTextWriteFailureExitsTwo(t *testing.T) {
	var stderr bytes.Buffer
	code := run([]string{
		"--input", "-",
		"--export-height", "9",
		"--app-hash", strings.Repeat("0f", 32),
		"--fail-on", "never",
	}, bytes.NewReader(marshalFixture(t, cleanFixture())), failingWriter{}, &stderr)
	if code != 2 {
		t.Fatalf("exit = %d, want 2; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "write report: forced write failure") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
