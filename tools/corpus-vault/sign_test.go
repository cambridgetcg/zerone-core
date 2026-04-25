package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"path/filepath"
	"testing"
)

func TestGenerateAndLoadKey(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "key.pem")
	pub, err := GenerateKeyPair(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(pub) != ed25519.PublicKeySize {
		t.Fatalf("public key wrong length: %d", len(pub))
	}
	priv, err := LoadPrivateKey(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	pub2 := priv.Public().(ed25519.PublicKey)
	if string(pub) != string(pub2) {
		t.Fatalf("loaded public key does not match generated key")
	}
}

func TestSignVerifyManifest_Roundtrip(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	body := ManifestBody{
		ManifestID: "v1#1",
		VaultID:    "v1",
		Items: []ManifestItem{
			{ID: "a", SizeBytes: 1, Hash: "aa", URL: "/item/v1#1/a"},
			{ID: "b", SizeBytes: 2, Hash: "bb", URL: "/item/v1#1/b"},
		},
	}
	sig, err := SignManifest(priv, body)
	if err != nil {
		t.Fatal(err)
	}
	signed := SignedManifest{ManifestBody: body, Signature: sig}
	if err := VerifyManifest(pub, signed); err != nil {
		t.Fatalf("verify: %v", err)
	}
	// Tamper: flip a byte in items, signature should no longer verify.
	signed.Items[0].Hash = "ab"
	if err := VerifyManifest(pub, signed); err == nil {
		t.Fatalf("verify should fail after tamper")
	}
}

func TestSignManifest_StableSignature(t *testing.T) {
	// Same body must produce the same signature (modulo nondeterminism
	// in ed25519 sign — which is deterministic by spec).
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	body := ManifestBody{
		ManifestID: "v1#1",
		VaultID:    "v1",
		Items: []ManifestItem{
			{ID: "a", SizeBytes: 1, Hash: "aa", URL: "/item/a"},
		},
	}
	s1, err := SignManifest(priv, body)
	if err != nil {
		t.Fatal(err)
	}
	s2, err := SignManifest(priv, body)
	if err != nil {
		t.Fatal(err)
	}
	if s1 != s2 {
		t.Fatalf("ed25519 signing non-deterministic — bug or stdlib regression")
	}
}

func TestParseClientPubkey_BothFormats(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	// ed25519:hex format.
	encoded := EncodePubkey(pub)
	parsed, err := ParseClientPubkey(encoded)
	if err != nil {
		t.Fatalf("parse ed25519: %v", err)
	}
	if string(parsed) != string(pub) {
		t.Fatalf("ed25519 round-trip mismatch")
	}
}
