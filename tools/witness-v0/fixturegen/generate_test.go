package fixturegen

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckedInCorpusHasZeroGeneratorDrift(t *testing.T) {
	if err := CheckDir(filepath.Join("..", "testdata")); err != nil {
		t.Fatal(err)
	}
}

func TestGeneratorRequiresExplicitDestinationAndCheckDoesNotWrite(t *testing.T) {
	for _, path := range []string{"", ".", string(filepath.Separator)} {
		if _, err := WriteDir(path); err == nil {
			t.Fatalf("WriteDir(%q) did not refuse broad/implicit destination", path)
		}
	}
	dir := t.TempDir()
	sentinel := filepath.Join(dir, "sentinel")
	if err := os.WriteFile(sentinel, []byte("unchanged"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := CheckDir(dir); err == nil {
		t.Fatal("empty/non-corpus directory unexpectedly matched")
	}
	contents, err := os.ReadFile(sentinel)
	if err != nil || string(contents) != "unchanged" {
		t.Fatalf("CheckDir wrote storage: %q, %v", contents, err)
	}
}
