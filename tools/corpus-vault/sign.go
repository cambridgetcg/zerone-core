package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
)

// LoadPrivateKey reads an ed25519 private key from a PEM file. The
// PEM block must contain the 64-byte raw private key (seed + pub).
//
// To generate one:
//
//	corpus-vault genkey --out operator.pem
//
// The matching public key is exposed via Pubkey().
func LoadPrivateKey(path string) (ed25519.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read key file: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("pem decode: no block found in %s", path)
	}
	if block.Type != "ED25519 PRIVATE KEY" {
		return nil, fmt.Errorf("unexpected pem type %q (want ED25519 PRIVATE KEY)", block.Type)
	}
	if len(block.Bytes) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("private key has wrong length %d (want %d)", len(block.Bytes), ed25519.PrivateKeySize)
	}
	return ed25519.PrivateKey(block.Bytes), nil
}

// GenerateKeyPair creates a new ed25519 keypair and writes the private
// key as PEM to privPath. Returns the matching public key.
func GenerateKeyPair(privPath string) (ed25519.PublicKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "ED25519 PRIVATE KEY",
		Bytes: priv,
	})
	if err := os.WriteFile(privPath, pemBytes, 0o600); err != nil {
		return nil, fmt.Errorf("write key file: %w", err)
	}
	return pub, nil
}

// EncodePubkey returns the operator-pubkey string format used on-chain
// in MsgRegisterVault. The format follows the convention documented
// in PROTOCOL.md: "ed25519:<hex>".
func EncodePubkey(pub ed25519.PublicKey) string {
	return "ed25519:" + hex.EncodeToString(pub)
}

// SignManifest produces the hex-encoded signature over the canonical
// JSON of the manifest body. This is what populates SignedManifest.Signature.
func SignManifest(priv ed25519.PrivateKey, body ManifestBody) (string, error) {
	bz, err := CanonicalJSON(body)
	if err != nil {
		return "", err
	}
	sig := ed25519.Sign(priv, bz)
	return hex.EncodeToString(sig), nil
}

// VerifyManifest verifies a signed manifest against an ed25519 public
// key. Returns nil on success.
func VerifyManifest(pub ed25519.PublicKey, m SignedManifest) error {
	bz, err := CanonicalJSON(m.ManifestBody)
	if err != nil {
		return err
	}
	sig, err := hex.DecodeString(m.Signature)
	if err != nil {
		return fmt.Errorf("decode signature hex: %w", err)
	}
	if !ed25519.Verify(pub, bz, sig) {
		return fmt.Errorf("signature verification failed")
	}
	return nil
}

// ParseClientPubkey decodes a client public key from one of two formats:
//   - "ed25519:<hex>" (matches the on-chain operator-pubkey convention)
//   - "<base64>" (legacy / convenience)
func ParseClientPubkey(s string) (ed25519.PublicKey, error) {
	if len(s) > 8 && s[:8] == "ed25519:" {
		raw, err := hex.DecodeString(s[8:])
		if err != nil {
			return nil, fmt.Errorf("decode ed25519 hex: %w", err)
		}
		if len(raw) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("ed25519 pubkey wrong length %d", len(raw))
		}
		return ed25519.PublicKey(raw), nil
	}
	raw, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("decode base64: %w", err)
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("ed25519 pubkey wrong length %d", len(raw))
	}
	return ed25519.PublicKey(raw), nil
}
