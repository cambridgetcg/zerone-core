package types_test

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/auth/types"
)

const rotationTestAddress = "zrn1m037n75vk2jhdr56y2ptzjjj02uljwnqwwzr7z"

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("zrn", "zrnpub")
}

func TestKeyRotationAuthorizationSignBytesKnownVector(t *testing.T) {
	newKey := make([]byte, ed25519.PublicKeySize)
	for index := range newKey {
		newKey[index] = byte(index)
	}
	got, err := types.KeyRotationAuthorizationSignBytes(
		"zerone-test-1", rotationTestAddress, 7, newKey, 1788436800,
	)
	if err != nil {
		t.Fatal(err)
	}
	const expectedHex = "7a65726f6e652e617574682f726f746174652d6b65792f7631000000000d7a65726f6e652d746573742d310000002a7a726e316d3033376e3735766b326a6864723536793270747a6a6a6a3032756c6a776e7177777a72377a00000007000000006a996140000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
	if hex.EncodeToString(got) != expectedHex {
		t.Fatalf("rotation authorization encoding drifted:\nwant %s\n got %x", expectedHex, got)
	}
}

func TestKeyRotationAuthorizationSignBytesRejectsAmbiguousInputs(t *testing.T) {
	validKey := make([]byte, ed25519.PublicKeySize)
	tests := map[string]struct {
		chainID string
		sender  string
		version uint32
		key     []byte
		expiry  int64
	}{
		"empty chain ID":        {"", rotationTestAddress, 1, validKey, 1},
		"padded chain ID":       {" zerone-test-1", rotationTestAddress, 1, validKey, 1},
		"invalid sender":        {"zerone-test-1", "invalid", 1, validKey, 1},
		"noncanonical sender":   {"zerone-test-1", strings.ToUpper(rotationTestAddress), 1, validKey, 1},
		"zero version":          {"zerone-test-1", rotationTestAddress, 0, validKey, 1},
		"wrong key length":      {"zerone-test-1", rotationTestAddress, 1, validKey[:31], 1},
		"nonpositive timestamp": {"zerone-test-1", rotationTestAddress, 1, validKey, 0},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := types.KeyRotationAuthorizationSignBytes(
				test.chainID, test.sender, test.version, test.key, test.expiry,
			); err == nil {
				t.Fatal("expected invalid input rejection")
			}
		})
	}
}

func TestKeyRotationAcceptanceUsesIndependentDomain(t *testing.T) {
	newKey := make([]byte, ed25519.PublicKeySize)
	for index := range newKey {
		newKey[index] = byte(index)
	}
	authorization, err := types.KeyRotationAuthorizationSignBytes(
		"zerone-test-1", rotationTestAddress, 7, newKey, 1788436800,
	)
	if err != nil {
		t.Fatal(err)
	}
	acceptance, err := types.KeyRotationAcceptanceSignBytes(
		"zerone-test-1", rotationTestAddress, 7, newKey, 1788436800,
	)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(authorization, acceptance) {
		t.Fatal("current-key authorization and new-key acceptance bytes share a domain")
	}
	const expectedHex = "7a65726f6e652e617574682f6163636570742d6b65792f7631000000000d7a65726f6e652d746573742d310000002a7a726e316d3033376e3735766b326a6864723536793270747a6a6a6a3032756c6a776e7177777a72377a00000007000000006a996140000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
	if hex.EncodeToString(acceptance) != expectedHex {
		t.Fatalf("rotation acceptance encoding drifted:\nwant %s\n got %x", expectedHex, acceptance)
	}
}

func TestOperationalKeyHashKnownVector(t *testing.T) {
	hash, err := types.OperationalKeyHash(make([]byte, ed25519.PublicKeySize))
	if err != nil {
		t.Fatal(err)
	}
	if hash != "66687aadf862bd776c8fc18b8e9f8e20089714856ee233b3902a591d0d5f2925" {
		t.Fatalf("unexpected hash: %s", hash)
	}
	if hash != strings.ToLower(hash) {
		t.Fatal("operational key hash is not lowercase")
	}
}
