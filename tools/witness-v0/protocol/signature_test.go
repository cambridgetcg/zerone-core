package protocol

import (
	"crypto/ed25519"
	"encoding/hex"
	"strings"
	"testing"

	"filippo.io/edwards25519"
)

func TestStrictEd25519AcceptsNormalSignature(t *testing.T) {
	key := testKey(1)
	message := []byte("strict-ed25519-known-message")
	signature := ed25519.Sign(key, message)
	if err := strictEd25519Verify(key.Public().(ed25519.PublicKey), message, signature); err != nil {
		t.Fatal(err)
	}
}

func TestStrictEd25519RejectsIdentitySmallMixedAndNoncanonicalPoints(t *testing.T) {
	identity := append([]byte{1}, make([]byte, 31)...)
	identitySignature := append(append([]byte(nil), identity...), make([]byte, 32)...)
	if err := strictEd25519Verify(identity, []byte("arbitrary"), identitySignature); err == nil || !strings.Contains(err.Error(), "identity") {
		t.Fatalf("identity forgery was not strictly rejected: %v", err)
	}

	lowOrder, _ := hex.DecodeString("26e8958fc2b227b045c3f489f2ef98f0d5dfac05d3c63339b13802886d53fc85")
	if err := validateStrictEd25519Point("low-order", lowOrder); err == nil || !strings.Contains(err.Error(), "small-order") {
		t.Fatalf("small-order point was not rejected: %v", err)
	}

	publicKey := testKey(1).Public().(ed25519.PublicKey)
	primePoint, _ := new(edwards25519.Point).SetBytes(publicKey)
	torsionPoint, _ := new(edwards25519.Point).SetBytes(lowOrder)
	mixed := new(edwards25519.Point).Add(primePoint, torsionPoint).Bytes()
	if err := validateStrictEd25519Point("mixed-order", mixed); err == nil || !strings.Contains(err.Error(), "prime-order subgroup") {
		t.Fatalf("mixed-order point was not rejected: %v", err)
	}

	noncanonical, _ := hex.DecodeString("eeffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7f")
	if err := validateStrictEd25519Point("noncanonical", noncanonical); err == nil || !strings.Contains(err.Error(), "noncanonical") {
		t.Fatalf("noncanonical point was not rejected: %v", err)
	}
}

func TestStrictEd25519RejectsNoncanonicalRAndS(t *testing.T) {
	key := testKey(1)
	message := []byte("strict-ed25519-components")
	signature := ed25519.Sign(key, message)
	noncanonicalR, _ := hex.DecodeString("eeffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7f")
	copy(signature[:32], noncanonicalR)
	if err := strictEd25519Verify(key.Public().(ed25519.PublicKey), message, signature); err == nil || !strings.Contains(err.Error(), "signature R encoding is noncanonical") {
		t.Fatalf("noncanonical R was not rejected: %v", err)
	}

	signature = ed25519.Sign(key, message)
	orderBigEndian := ed25519PrimeOrder.Bytes()
	orderLittleEndian := make([]byte, 32)
	for i := range orderBigEndian {
		orderLittleEndian[i] = orderBigEndian[len(orderBigEndian)-1-i]
	}
	copy(signature[32:], orderLittleEndian)
	if err := strictEd25519Verify(key.Public().(ed25519.PublicKey), message, signature); err == nil || !strings.Contains(err.Error(), "canonical Ed25519 scalar") {
		t.Fatalf("noncanonical S was not rejected: %v", err)
	}
}
