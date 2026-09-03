package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	dbm "github.com/cosmos/cosmos-db"
)

type commandTestDB struct {
	closeErr error
}

func (*commandTestDB) Get([]byte) ([]byte, error)                           { return nil, nil }
func (*commandTestDB) Iterator([]byte, []byte) (dbm.Iterator, error)        { return nil, nil }
func (*commandTestDB) ReverseIterator([]byte, []byte) (dbm.Iterator, error) { return nil, nil }
func (db *commandTestDB) Close() error                                      { return db.closeErr }

func validCommandArgs() []string {
	return []string{
		"--home", "/tmp/zerone-census-copy",
		"--backend", "goleveldb",
		"--chain-id", "zerone-2",
		"--expected-height", "42",
		"--expected-app-hash", "0000000000000000000000000000000000000000000000000000000000000000",
		"--source-commit", "1111111111111111111111111111111111111111",
		"--copied-db",
	}
}

func TestRunPassAndFailExitSemantics(t *testing.T) {
	tests := []struct {
		name       string
		passed     bool
		wantCode   int
		wantStderr bool
	}{
		{name: "pass", passed: true, wantCode: exitOK},
		{name: "reconciliation failure still emits evidence", passed: false, wantCode: exitReconcile, wantStderr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			opened := false
			open := func(home, backend string) (openedPhysicalDB, error) {
				opened = true
				if home != "/tmp/zerone-census-copy" || backend != "goleveldb" {
					t.Fatalf("unexpected opener arguments: %q %q", home, backend)
				}
				return &commandTestDB{}, nil
			}
			census := func(_ physicalDB, options censusOptions) ([]byte, bool, error) {
				if options.ChainID != "zerone-2" || options.Height != 42 {
					t.Fatalf("unexpected census options: %+v", options)
				}
				return []byte(`{"schema":"test"}`), test.passed, nil
			}
			if got := run(validCommandArgs(), &stdout, &stderr, open, census); got != test.wantCode {
				t.Fatalf("exit code = %d, want %d; stderr=%q", got, test.wantCode, stderr.String())
			}
			if !opened {
				t.Fatal("database was not opened")
			}
			if got := stdout.String(); got != "{\"schema\":\"test\"}\n" {
				t.Fatalf("stdout = %q", got)
			}
			if test.wantStderr != (stderr.Len() > 0) {
				t.Fatalf("stderr = %q", stderr.String())
			}
		})
	}
}

func TestRunRejectsUnsafeOrInvalidEvidenceBeforeOpen(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "copy attestation missing", args: validCommandArgs()[:len(validCommandArgs())-1]},
		{name: "noncanonical height", args: replaceCommandArg(validCommandArgs(), "--expected-height", "042")},
		{name: "uppercase app hash", args: replaceCommandArg(validCommandArgs(), "--expected-app-hash", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")},
		{name: "uppercase commit", args: replaceCommandArg(validCommandArgs(), "--source-commit", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")},
		{name: "unsafe chain id", args: replaceCommandArg(validCommandArgs(), "--chain-id", "../zerone-2")},
		{name: "positional argument", args: append(validCommandArgs(), "extra")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			open := func(_, _ string) (openedPhysicalDB, error) {
				t.Fatal("invalid arguments reached database opener")
				return nil, nil
			}
			code := run(test.args, &stdout, &stderr, open, func(physicalDB, censusOptions) ([]byte, bool, error) {
				t.Fatal("invalid arguments reached census")
				return nil, false, nil
			})
			if code != exitOperational {
				t.Fatalf("exit code = %d", code)
			}
			if stdout.Len() != 0 || stderr.Len() == 0 {
				t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
			}
		})
	}
}

func TestRunSuppressesUnverifiedReportOnAuditOrCloseError(t *testing.T) {
	tests := []struct {
		name     string
		auditErr error
		closeErr error
	}{
		{name: "audit", auditErr: errors.New("proof failed")},
		{name: "close", closeErr: errors.New("close failed")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(
				validCommandArgs(),
				&stdout,
				&stderr,
				func(_, _ string) (openedPhysicalDB, error) { return &commandTestDB{closeErr: test.closeErr}, nil },
				func(physicalDB, censusOptions) ([]byte, bool, error) {
					return []byte(`{"unsafe":true}`), true, test.auditErr
				},
			)
			if code != exitOperational || stdout.Len() != 0 || stderr.Len() == 0 {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
		})
	}
}

func TestRunPublishesAtomicOutputWithoutOverwriting(t *testing.T) {
	sourceHome := t.TempDir()
	outputPath := filepath.Join(t.TempDir(), "census.json")
	args := replaceCommandArg(validCommandArgs(), "--home", sourceHome)
	args = append(args, "--output", outputPath)
	var stdout, stderr bytes.Buffer
	open := func(_, _ string) (openedPhysicalDB, error) { return &commandTestDB{}, nil }
	census := func(physicalDB, censusOptions) ([]byte, bool, error) {
		return []byte(`{"schema":"test"}`), false, nil
	}

	code := run(args, &stdout, &stderr, open, census)
	if code != exitReconcile {
		t.Fatalf("exit code = %d, want %d; stderr=%q", code, exitReconcile, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("atomic output mode wrote stdout: %q", stdout.String())
	}
	contents, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "{\"schema\":\"test\"}\n" {
		t.Fatalf("report contents = %q", contents)
	}

	stderr.Reset()
	if code := run(args, &stdout, &stderr, open, census); code != exitOperational {
		t.Fatalf("overwrite exit code = %d, want %d", code, exitOperational)
	}
	contentsAfter, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(contents, contentsAfter) {
		t.Fatal("existing evidence report was overwritten")
	}
}

func TestRunRejectsAtomicOutputInsideCopiedHome(t *testing.T) {
	sourceHome := t.TempDir()
	outputPath := filepath.Join(sourceHome, "report.json")
	args := replaceCommandArg(validCommandArgs(), "--home", sourceHome)
	args = append(args, "--output", outputPath)
	var stdout, stderr bytes.Buffer
	code := run(
		args,
		&stdout,
		&stderr,
		func(_, _ string) (openedPhysicalDB, error) { return &commandTestDB{}, nil },
		func(physicalDB, censusOptions) ([]byte, bool, error) { return []byte(`{"schema":"test"}`), true, nil },
	)
	if code != exitOperational || stderr.Len() == 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	if _, err := os.Stat(outputPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("source-home output exists or returned unexpected error: %v", err)
	}
}

type failingPrefixWriter struct {
	written bool
}

func (writer *failingPrefixWriter) Write(value []byte) (int, error) {
	if writer.written {
		return 0, errors.New("output failed")
	}
	writer.written = true
	if len(value) == 0 {
		return 0, errors.New("output failed")
	}
	return 1, errors.New("output failed after prefix")
}

func TestRunTreatsPartialStdoutWriteAsOperationalFailure(t *testing.T) {
	stdout := &failingPrefixWriter{}
	var stderr bytes.Buffer
	code := run(
		validCommandArgs(),
		stdout,
		&stderr,
		func(_, _ string) (openedPhysicalDB, error) { return &commandTestDB{}, nil },
		func(physicalDB, censusOptions) ([]byte, bool, error) { return []byte(`{"schema":"test"}`), true, nil },
	)
	if code != exitOperational || stderr.Len() == 0 || !stdout.written {
		t.Fatalf("code=%d stderr=%q wrote=%v", code, stderr.String(), stdout.written)
	}
}

func replaceCommandArg(args []string, flagName, value string) []string {
	result := append([]string(nil), args...)
	for index := range result {
		if result[index] == flagName && index+1 < len(result) {
			result[index+1] = value
			return result
		}
	}
	panic("flag not found: " + flagName)
}
