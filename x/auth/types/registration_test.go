package types_test

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/zerone-chain/zerone/x/auth/types"
)

const secondRegistrationTestAddress = "zrn1ur4eyeuuhrkfpcyhykfjsasftv9hn33smszt58"

func TestAccountRegistrationProofSignBytesKnownVector(t *testing.T) {
	publicKey, err := hex.DecodeString("d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a")
	if err != nil {
		t.Fatal(err)
	}
	did := "did:zrn:" + hex.EncodeToString(publicKey)
	got, err := types.AccountRegistrationProofSignBytes(
		"zerone-test-1",
		rotationTestAddress,
		did,
		publicKey,
		"agent",
		`{"name":"Sophia"}`,
	)
	if err != nil {
		t.Fatal(err)
	}
	const expectedHex = "7a65726f6e652e617574682f72656769737465722d6163636f756e742f7631000000000d7a65726f6e652d746573742d310000002a7a726e316d3033376e3735766b326a6864723536793270747a6a6a6a3032756c6a776e7177777a72377a000000486469643a7a726e3a64373561393830313832623130616237643534626665643363393634303733613065653137326633646161363233323561663032316136386637303735313161d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a000000056167656e74000000117b226e616d65223a22536f70686961227d"
	if hex.EncodeToString(got) != expectedHex {
		t.Fatalf("registration proof encoding drifted:\nwant %s\n got %x", expectedHex, got)
	}
}

func TestAccountRegistrationProofBindsEveryMutableField(t *testing.T) {
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x62}, ed25519.SeedSize))
	publicKey := privateKey.Public().(ed25519.PublicKey)
	did := "did:zrn:" + hex.EncodeToString(publicKey)
	original, err := types.AccountRegistrationProofSignBytes(
		"zerone-test-1", rotationTestAddress, did, publicKey, "agent", `{"v":1}`,
	)
	if err != nil {
		t.Fatal(err)
	}
	signature := ed25519.Sign(privateKey, original)
	if err := types.VerifyEd25519Signature(publicKey, original, signature); err != nil {
		t.Fatal(err)
	}

	mutations := []struct {
		name        string
		chainID     string
		sender      string
		accountType string
		metadata    string
	}{
		{name: "chain ID", chainID: "zerone-other-1", sender: rotationTestAddress, accountType: "agent", metadata: `{"v":1}`},
		{name: "sender", chainID: "zerone-test-1", sender: secondRegistrationTestAddress, accountType: "agent", metadata: `{"v":1}`},
		{name: "account type", chainID: "zerone-test-1", sender: rotationTestAddress, accountType: "human", metadata: `{"v":1}`},
		{name: "metadata", chainID: "zerone-test-1", sender: rotationTestAddress, accountType: "agent", metadata: `{"v":2}`},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			changed, err := types.AccountRegistrationProofSignBytes(
				mutation.chainID,
				mutation.sender,
				did,
				publicKey,
				mutation.accountType,
				mutation.metadata,
			)
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Equal(original, changed) {
				t.Fatal("field mutation did not change registration proof bytes")
			}
			if err := types.VerifyEd25519Signature(publicKey, changed, signature); err == nil {
				t.Fatal("original signature accepted mutated registration")
			}
		})
	}
}

func TestAccountRegistrationProofRejectsNoncanonicalInputs(t *testing.T) {
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x63}, ed25519.SeedSize))
	publicKey := privateKey.Public().(ed25519.PublicKey)
	did := "did:zrn:" + hex.EncodeToString(publicKey)
	tests := []struct {
		name        string
		chainID     string
		sender      string
		did         string
		publicKey   []byte
		accountType string
	}{
		{name: "empty chain ID", sender: rotationTestAddress, did: did, publicKey: publicKey, accountType: "agent"},
		{name: "padded chain ID", chainID: " zerone-test-1", sender: rotationTestAddress, did: did, publicKey: publicKey, accountType: "agent"},
		{name: "uppercase address", chainID: "zerone-test-1", sender: strings.ToUpper(rotationTestAddress), did: did, publicKey: publicKey, accountType: "agent"},
		{name: "truncated DID", chainID: "zerone-test-1", sender: rotationTestAddress, did: did[:len(did)-2], publicKey: publicKey, accountType: "agent"},
		{name: "wrong key length", chainID: "zerone-test-1", sender: rotationTestAddress, did: did, publicKey: publicKey[:31], accountType: "agent"},
		{name: "invalid account type", chainID: "zerone-test-1", sender: rotationTestAddress, did: did, publicKey: publicKey, accountType: "administrator"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := types.AccountRegistrationProofSignBytes(
				test.chainID, test.sender, test.did, test.publicKey, test.accountType, "",
			); err == nil {
				t.Fatal("expected invalid registration proof input to be rejected")
			}
		})
	}
}
