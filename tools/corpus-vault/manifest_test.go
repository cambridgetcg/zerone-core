package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCanonicalJSON_StableOrder(t *testing.T) {
	body := ManifestBody{
		ManifestID: "v1#1",
		VaultID:    "v1",
		Items: []ManifestItem{
			{ID: "b.txt", SizeBytes: 1, Hash: "bb", URL: "/item/b.txt"},
			{ID: "a.txt", SizeBytes: 1, Hash: "aa", URL: "/item/a.txt"},
		},
	}
	bz1, err := CanonicalJSON(body)
	if err != nil {
		t.Fatal(err)
	}
	// Reorder items in input; canonical output should be identical.
	body.Items[0], body.Items[1] = body.Items[1], body.Items[0]
	bz2, err := CanonicalJSON(body)
	if err != nil {
		t.Fatal(err)
	}
	if string(bz1) != string(bz2) {
		t.Fatalf("canonical JSON not stable across input order:\n%s\nvs\n%s", bz1, bz2)
	}
	// And the bytes should sort items by ID (a before b).
	if !strings.Contains(string(bz1), `"id":"a.txt"`) {
		t.Fatalf("expected a.txt to appear in canonical output")
	}
	idxA := strings.Index(string(bz1), `"id":"a.txt"`)
	idxB := strings.Index(string(bz1), `"id":"b.txt"`)
	if idxA >= idxB {
		t.Fatalf("expected a.txt to come before b.txt in canonical output")
	}
}

func TestContentHash_Determinism(t *testing.T) {
	body := ManifestBody{
		ManifestID: "v1#1",
		VaultID:    "v1",
		Items: []ManifestItem{
			{ID: "a", SizeBytes: 1, Hash: "aa", URL: "/item/a"},
		},
	}
	h1, err := ContentHash(body)
	if err != nil {
		t.Fatal(err)
	}
	h2, err := ContentHash(body)
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Fatalf("content hash non-deterministic: %s vs %s", h1, h2)
	}
	// And it should equal the SHA-256 of the canonical bytes.
	bz, _ := CanonicalJSON(body)
	want := sha256.Sum256(bz)
	if h1 != hex.EncodeToString(want[:]) {
		t.Fatalf("content hash mismatch: %s vs %s", h1, hex.EncodeToString(want[:]))
	}
}

func TestBuildManifestFromDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "alpha.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "beta.txt"), []byte("world"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Dotfile should be skipped.
	if err := os.WriteFile(filepath.Join(dir, ".hidden"), []byte("nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	body, err := BuildManifestFromDir("v1#1", "v1", dir, "/item/v1#1/")
	if err != nil {
		t.Fatal(err)
	}
	if body.ManifestID != "v1#1" || body.VaultID != "v1" {
		t.Fatalf("manifest header wrong: %+v", body)
	}
	if len(body.Items) != 2 {
		t.Fatalf("expected 2 items (.hidden skipped), got %d", len(body.Items))
	}
	for _, it := range body.Items {
		if !strings.HasPrefix(it.URL, "/item/v1#1/") {
			t.Errorf("URL prefix wrong: %s", it.URL)
		}
		if it.Hash == "" || it.SizeBytes == 0 {
			t.Errorf("item not populated: %+v", it)
		}
	}
}

func TestBuildManifestFromDir_EmptyFails(t *testing.T) {
	dir := t.TempDir()
	if _, err := BuildManifestFromDir("v1#1", "v1", dir, "/item/"); err == nil {
		t.Fatalf("expected empty dir to fail")
	}
}
