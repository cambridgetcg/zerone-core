package app

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	vrfcrypto "github.com/zerone-chain/zerone/x/knowledge/crypto"
)

// writePrivValKey writes a CometBFT-format priv_validator_key.json carrying priv.
func writePrivValKey(t *testing.T, dir string, priv ed25519.PrivateKey) string {
	t.Helper()
	doc := map[string]any{
		"address": "0000000000000000000000000000000000000000",
		"pub_key": map[string]any{
			"type":  "tendermint/PubKeyEd25519",
			"value": base64.StdEncoding.EncodeToString(priv[32:]),
		},
		"priv_key": map[string]any{
			"type":  "tendermint/PrivKeyEd25519",
			"value": base64.StdEncoding.EncodeToString(priv),
		},
	}
	bz, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "priv_validator_key.json")
	if err := os.WriteFile(p, bz, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// The parser must round-trip a real CometBFT key AND the loaded bytes must be
// usable by the VRF the vote-extension handler actually calls.
func TestLoadConsensusPrivKey_RoundTripAndUsable(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	p := writePrivValKey(t, t.TempDir(), priv)

	got, err := loadConsensusPrivKey(p)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got) != 64 {
		t.Fatalf("key length = %d, want 64", len(got))
	}
	if !bytes.Equal(got, priv) {
		t.Fatal("loaded key does not match written key")
	}

	seed := vrfcrypto.GenerateVRFSeed("claim-1", 100, nil)
	out, proof, err := vrfcrypto.GenerateVRF(seed, got)
	if err != nil {
		t.Fatalf("loaded key rejected by VRF: %v", err)
	}
	if len(out) == 0 || len(proof) == 0 {
		t.Fatal("VRF produced empty output/proof from loaded key")
	}
}

// A missing or malformed key must return an error, never os.Exit or panic —
// this is what keeps a mis-set validator-address from aborting node startup.
func TestLoadConsensusPrivKey_FailsSafe(t *testing.T) {
	if _, err := loadConsensusPrivKey(filepath.Join(t.TempDir(), "absent.json")); err == nil {
		t.Fatal("expected error for missing file")
	}

	dir := t.TempDir()
	p := filepath.Join(dir, "priv_validator_key.json")

	if err := os.WriteFile(p, []byte(`{"priv_key":{"value":""}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadConsensusPrivKey(p); err == nil {
		t.Fatal("expected error for empty priv_key")
	}

	if err := os.WriteFile(p, []byte(`not json`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadConsensusPrivKey(p); err == nil {
		t.Fatal("expected error for malformed json")
	}

	badLen := base64.StdEncoding.EncodeToString([]byte("tooshort"))
	if err := os.WriteFile(p, []byte(`{"priv_key":{"value":"`+badLen+`"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadConsensusPrivKey(p); err == nil {
		t.Fatal("expected error for wrong key length")
	}
}
