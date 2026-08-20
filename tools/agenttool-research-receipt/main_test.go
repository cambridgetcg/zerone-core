package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

func TestRunRequiresOnlyNamedInputs(t *testing.T) {
	for name, arguments := range map[string][]string{
		"missing inputs":   nil,
		"positional input": {"settlement.json"},
	} {
		t.Run(name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			err := run(arguments, &stdout, &stderr)
			if err == nil {
				t.Fatal("run unexpectedly succeeded")
			}
			if stdout.Len() != 0 {
				t.Fatalf("refusal wrote output: %q", stdout.String())
			}
		})
	}
}

func TestReadBoundedRegularFileRefusesNonRegularAndOversizedInputs(t *testing.T) {
	temporary := t.TempDir()
	regularPath := filepath.Join(temporary, "input.json")
	if err := os.WriteFile(regularPath, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := readBoundedRegularFile(temporary, 10); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("directory must be refused, got %v", err)
	}

	symlinkPath := filepath.Join(temporary, "input-link.json")
	if err := os.Symlink(regularPath, symlinkPath); err != nil {
		t.Fatal(err)
	}
	if _, err := readBoundedRegularFile(symlinkPath, 10); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("final-component symlink must be refused, got %v", err)
	}

	if _, err := readBoundedRegularFile(regularPath, 1); err == nil || !strings.Contains(err.Error(), "exceeds 1-byte limit") {
		t.Fatalf("oversized file must be refused, got %v", err)
	}

	fifoPath := filepath.Join(temporary, "input.fifo")
	if err := unix.Mkfifo(fifoPath, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readBoundedRegularFile(fifoPath, 10); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("FIFO must be refused without blocking, got %v", err)
	}
}

func TestReadBoundedRegularFileRefusesPathSwapAfterOpen(t *testing.T) {
	temporary := t.TempDir()
	inputPath := filepath.Join(temporary, "input.json")
	originalPath := filepath.Join(temporary, "original.json")
	replacementPath := filepath.Join(temporary, "replacement.json")
	if err := os.WriteFile(inputPath, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(replacementPath, []byte("replaced"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := readBoundedRegularFileWithHook(inputPath, 64, func() error {
		if err := os.Rename(inputPath, originalPath); err != nil {
			return err
		}
		return os.Rename(replacementPath, inputPath)
	})
	if err == nil || !strings.Contains(err.Error(), "changed while") {
		t.Fatalf("post-open pathname swap must be refused, got %v", err)
	}
}

func TestReadBoundedRegularFileAcceptsStableIntermediateSymlink(t *testing.T) {
	temporary := t.TempDir()
	realDirectory := filepath.Join(temporary, "real")
	if err := os.Mkdir(realDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	inputPath := filepath.Join(realDirectory, "input.json")
	if err := os.WriteFile(inputPath, []byte("stable"), 0o600); err != nil {
		t.Fatal(err)
	}
	linkedDirectory := filepath.Join(temporary, "linked")
	if err := os.Symlink(realDirectory, linkedDirectory); err != nil {
		t.Fatal(err)
	}

	data, err := readBoundedRegularFile(filepath.Join(linkedDirectory, "input.json"), 64)
	if err != nil {
		t.Fatalf("stable intermediate symlink should remain an explicit-path input: %v", err)
	}
	if string(data) != "stable" {
		t.Fatalf("read through stable intermediate symlink = %q", data)
	}
}

func TestReadBoundedRegularFileRefusesDescriptorMutationDuringRead(t *testing.T) {
	temporary := t.TempDir()
	inputPath := filepath.Join(temporary, "input.json")
	if err := os.WriteFile(inputPath, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := readBoundedRegularFileWithHook(inputPath, 64, func() error {
		return os.WriteFile(inputPath, []byte("mutated-longer"), 0o600)
	})
	if err == nil || !strings.Contains(err.Error(), "changed while it was being read") {
		t.Fatalf("post-open descriptor mutation must be refused, got %v", err)
	}
}
