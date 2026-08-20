package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/zerone-chain/zerone/tools/witness-v0/protocol"
)

func TestReadInputAcceptsBoundedRegularFileAndExplicitStdin(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "input.json")
	if err := os.WriteFile(path, []byte(`{"x":0}`), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readInput(path, strings.NewReader("ignored"))
	if err != nil || string(got) != `{"x":0}` {
		t.Fatalf("regular file: %q, %v", got, err)
	}
	got, err = readInput("-", strings.NewReader(`{"stdin":0}`))
	if err != nil || string(got) != `{"stdin":0}` {
		t.Fatalf("stdin: %q, %v", got, err)
	}
}

func TestReadInputRejectsSymlinkFIFODeviceDirectoryAndOversize(t *testing.T) {
	dir := t.TempDir()
	regular := filepath.Join(dir, "regular")
	if err := os.WriteFile(regular, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	symlink := filepath.Join(dir, "symlink")
	if err := os.Symlink(regular, symlink); err != nil {
		t.Fatal(err)
	}
	fifo := filepath.Join(dir, "fifo")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}
	oversize := filepath.Join(dir, "oversize")
	file, err := os.Create(oversize)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(protocol.MaxDocumentBytes + 1); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{symlink, fifo, "/dev/null", dir, oversize} {
		t.Run(filepath.Base(path), func(t *testing.T) {
			if _, err := readInput(path, strings.NewReader("")); err == nil {
				t.Fatalf("accepted unsafe path %q", path)
			}
		})
	}
	if _, err := readInput("-", bytes.NewReader(bytes.Repeat([]byte{'x'}, protocol.MaxDocumentBytes+1))); err == nil {
		t.Fatal("accepted oversized stdin")
	}
}

func TestCLIHelpAndSchemaHashesAreDeterministic(t *testing.T) {
	var first, second bytes.Buffer
	if err := run([]string{"schema-hashes"}, strings.NewReader(""), &first); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"schema-hashes"}, strings.NewReader(""), &second); err != nil {
		t.Fatal(err)
	}
	if first.String() != second.String() || !strings.Contains(first.String(), protocol.Protocol) {
		t.Fatal("schema-hashes output is not deterministic")
	}
	var help bytes.Buffer
	if err := run([]string{"help"}, strings.NewReader(""), &help); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(help.String(), "no network requests") {
		t.Fatal("help omits effect boundary")
	}
}

func TestCLIActivationAuditNeverUpgradesOfflineRecord(t *testing.T) {
	record, err := os.ReadFile(filepath.Join("testdata", "records", "settlement-root-0002-cross-batch-replay.json"))
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := run([]string{"activation-audit", "-"}, bytes.NewReader(record), &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), protocol.ActivationStatusNotConsensusAdmissible) ||
		!strings.Contains(output.String(), "PERMANENT_CROSS_BATCH_RECEIPT_NULLIFIERS_OR_PROOFS") {
		t.Fatalf("activation boundary missing from CLI output: %s", output.String())
	}
}
