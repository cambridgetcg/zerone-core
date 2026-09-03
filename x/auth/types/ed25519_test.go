package types_test

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"strings"
	"testing"

	"filippo.io/edwards25519"

	"github.com/zerone-chain/zerone/x/auth/types"
)

func TestStrictEd25519AcceptsGeneratedKeysAndSignatures(t *testing.T) {
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x42}, ed25519.SeedSize))
	publicKey := privateKey.Public().(ed25519.PublicKey)
	message := []byte("zerone auth strict Ed25519")
	signature := ed25519.Sign(privateKey, message)

	if err := types.ValidateEd25519PublicKey(publicKey); err != nil {
		t.Fatalf("generated public key rejected: %v", err)
	}
	if err := types.VerifyEd25519Signature(publicKey, message, signature); err != nil {
		t.Fatalf("generated signature rejected: %v", err)
	}
	if err := types.VerifyEd25519Signature(publicKey, append(message, '!'), signature); err == nil {
		t.Fatal("signature accepted for a different message")
	}
}

func TestStrictEd25519RejectsIdentitySmallMixedAndNoncanonicalPoints(t *testing.T) {
	identity := append([]byte{1}, make([]byte, 31)...)
	if err := types.ValidateEd25519PublicKey(identity); err == nil || !strings.Contains(err.Error(), "small-order") {
		t.Fatalf("identity point was not rejected as small order: %v", err)
	}

	lowOrder, err := hex.DecodeString("26e8958fc2b227b045c3f489f2ef98f0d5dfac05d3c63339b13802886d53fc85")
	if err != nil {
		t.Fatal(err)
	}
	if err := types.ValidateEd25519PublicKey(lowOrder); err == nil || !strings.Contains(err.Error(), "small-order") {
		t.Fatalf("small-order point was not rejected: %v", err)
	}

	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x43}, ed25519.SeedSize))
	primePoint, err := new(edwards25519.Point).SetBytes(privateKey.Public().(ed25519.PublicKey))
	if err != nil {
		t.Fatal(err)
	}
	torsionPoint, err := new(edwards25519.Point).SetBytes(lowOrder)
	if err != nil {
		t.Fatal(err)
	}
	mixed := new(edwards25519.Point).Add(primePoint, torsionPoint).Bytes()
	if err := types.ValidateEd25519PublicKey(mixed); err == nil || !strings.Contains(err.Error(), "prime-order subgroup") {
		t.Fatalf("mixed-order point was not rejected: %v", err)
	}

	noncanonical, err := hex.DecodeString("eeffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7f")
	if err != nil {
		t.Fatal(err)
	}
	if err := types.ValidateEd25519PublicKey(noncanonical); err == nil || !strings.Contains(err.Error(), "noncanonical") {
		t.Fatalf("noncanonical point was not rejected: %v", err)
	}

	for _, malformed := range [][]byte{nil, make([]byte, 31), make([]byte, 33)} {
		if err := types.ValidateEd25519PublicKey(malformed); err == nil {
			t.Fatalf("public key length %d was accepted", len(malformed))
		}
	}
}

func TestStrictEd25519RejectsMalformedSignatureComponents(t *testing.T) {
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x44}, ed25519.SeedSize))
	publicKey := privateKey.Public().(ed25519.PublicKey)
	message := []byte("strict signature components")

	for _, malformed := range [][]byte{nil, make([]byte, 63), make([]byte, 65)} {
		if err := types.VerifyEd25519Signature(publicKey, message, malformed); err == nil {
			t.Fatalf("signature length %d was accepted", len(malformed))
		}
	}

	signature := ed25519.Sign(privateKey, message)
	noncanonicalR, err := hex.DecodeString("eeffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7f")
	if err != nil {
		t.Fatal(err)
	}
	copy(signature[:32], noncanonicalR)
	if err := types.VerifyEd25519Signature(publicKey, message, signature); err == nil || !strings.Contains(err.Error(), "signature R encoding is noncanonical") {
		t.Fatalf("noncanonical R was not rejected: %v", err)
	}

	// l, the subgroup order, encoded little endian. RFC 8032 requires S < l.
	signature = ed25519.Sign(privateKey, message)
	noncanonicalS, err := hex.DecodeString("edd3f55c1a631258d69cf7a2def9de14" + "00000000000000000000000000000010")
	if err != nil {
		t.Fatal(err)
	}
	copy(signature[32:], noncanonicalS)
	if err := types.VerifyEd25519Signature(publicKey, message, signature); err == nil || !strings.Contains(err.Error(), "canonical Ed25519 scalar") {
		t.Fatalf("noncanonical S was not rejected: %v", err)
	}
}
