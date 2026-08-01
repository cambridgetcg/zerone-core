package main

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"
)

func TestZRNValoperKnownVector(t *testing.T) {
	payload := make([]byte, zrnValoperAddressBytes)
	for index := range payload {
		payload[index] = byte(index)
	}

	const expected = "zrnvaloper1qqqsyqcyq5rqwzqfpg9scrgwpugpzysnarg74n"
	encoded, err := encodeZRNValoper(payload)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if encoded != expected {
		t.Fatalf("encoding = %q, want %q", encoded, expected)
	}
	if err := validateZRNValoper("operator", expected); err != nil {
		t.Fatalf("validate: %v", err)
	}
	decoded, err := decodeZRNValoper(expected)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !bytes.Equal(decoded, payload) {
		t.Fatalf("decoded payload = %x, want %x", decoded, payload)
	}
}

func TestZRNValoperRejectsNonCanonicalEncodings(t *testing.T) {
	payload := bytes.Repeat([]byte{0x42}, zrnValoperAddressBytes)
	valid, err := encodeZRNValoper(payload)
	if err != nil {
		t.Fatal(err)
	}
	wrongPrefix, err := encodeBech32("zrn", payload)
	if err != nil {
		t.Fatal(err)
	}
	shortPayload, err := encodeBech32(
		zrnValoperHRP,
		payload[:zrnValoperAddressBytes-1],
	)
	if err != nil {
		t.Fatal(err)
	}
	longPayload, err := encodeBech32(
		zrnValoperHRP,
		append(append([]byte{}, payload...), 0),
	)
	if err != nil {
		t.Fatal(err)
	}
	bech32m := encodeWithChecksumConstant(
		t,
		zrnValoperHRP,
		payload,
		bech32mChecksumConstant,
	)
	badChecksum := valid[:len(valid)-1] + replacementBech32Character(valid[len(valid)-1])
	invalidAlphabet := valid[:len(valid)-1] + "i"

	tests := []struct {
		name  string
		value string
	}{
		{name: "uppercase", value: strings.ToUpper(valid)},
		{name: "mixed case", value: strings.ToUpper(valid[:1]) + valid[1:]},
		{name: "wrong prefix", value: wrongPrefix},
		{name: "short payload", value: shortPayload},
		{name: "long payload", value: longPayload},
		{name: "bech32m", value: bech32m},
		{name: "bad checksum", value: badChecksum},
		{name: "invalid alphabet", value: invalidAlphabet},
		{name: "missing separator", value: strings.Replace(valid, "1", "q", 1)},
		{name: "non ASCII", value: valid[:len(valid)-1] + "é"},
		{name: "too long", value: valid + strings.Repeat("q", 90)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := validateZRNValoper("operator", test.value); err == nil {
				t.Fatalf("validateZRNValoper(%q) succeeded", test.value)
			}
		})
	}
}

func TestZRNValoperRejectsNonCanonicalPadding(t *testing.T) {
	payload := bytes.Repeat([]byte{0x24}, zrnValoperAddressBytes)
	fiveBit, err := convertBits(payload, 8, 5, true)
	if err != nil {
		t.Fatal(err)
	}
	fiveBit = append(fiveBit, 0)
	checksum := createBech32Checksum(
		zrnValoperHRP,
		fiveBit,
		bech32ChecksumConstant,
	)

	var encoded strings.Builder
	encoded.WriteString(zrnValoperHRP)
	encoded.WriteByte('1')
	for _, value := range append(fiveBit, checksum...) {
		encoded.WriteByte(bech32Charset[value])
	}
	if err := validateZRNValoper("operator", encoded.String()); err == nil {
		t.Fatal("non-canonical excess padding was accepted")
	}
}

func TestConvertBitsRoundTripAndFailures(t *testing.T) {
	source := []byte{0x00, 0x01, 0x7f, 0x80, 0xfe, 0xff}
	fiveBit, err := convertBits(source, 8, 5, true)
	if err != nil {
		t.Fatalf("8-to-5 conversion: %v", err)
	}
	roundTrip, err := convertBits(fiveBit, 5, 8, false)
	if err != nil {
		t.Fatalf("5-to-8 conversion: %v", err)
	}
	if !bytes.Equal(roundTrip, source) {
		t.Fatalf("round trip = %x, want %x", roundTrip, source)
	}

	tests := []struct {
		name     string
		data     []byte
		fromBits uint
		toBits   uint
		pad      bool
	}{
		{name: "zero source width", data: []byte{0}, fromBits: 0, toBits: 5, pad: true},
		{name: "oversized target width", data: []byte{0}, fromBits: 5, toBits: 9, pad: true},
		{name: "value outside group", data: []byte{32}, fromBits: 5, toBits: 8, pad: false},
		{name: "excess padding", data: []byte{0}, fromBits: 5, toBits: 8, pad: false},
		{name: "nonzero padding", data: []byte{1}, fromBits: 5, toBits: 8, pad: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := convertBits(
				test.data,
				test.fromBits,
				test.toBits,
				test.pad,
			); err == nil {
				t.Fatal("conversion unexpectedly succeeded")
			}
		})
	}
}

func TestConsensusAndNodeIdentityKnownVector(t *testing.T) {
	publicKey := make([]byte, 32)
	for index := range publicKey {
		publicKey[index] = byte(index)
	}
	publicKeyHex := hex.EncodeToString(publicKey)

	const expectedLower = "630dcd2966c4336691125448bbb25b4ff412a49c"
	consensusAddress, err := consensusAddressFromPublicKeyHex(publicKeyHex)
	if err != nil {
		t.Fatalf("derive consensus address: %v", err)
	}
	if consensusAddress != strings.ToUpper(expectedLower) {
		t.Fatalf(
			"consensus address = %q, want %q",
			consensusAddress,
			strings.ToUpper(expectedLower),
		)
	}
	if err := validateConsensusAddress(publicKeyHex, consensusAddress); err != nil {
		t.Fatalf("validate consensus address: %v", err)
	}

	nodeID, err := nodeIDFromPublicKeyHex(publicKeyHex)
	if err != nil {
		t.Fatalf("derive node ID: %v", err)
	}
	if nodeID != expectedLower {
		t.Fatalf("node ID = %q, want %q", nodeID, expectedLower)
	}
	if err := validateNodeID(publicKeyHex, nodeID); err != nil {
		t.Fatalf("validate node ID: %v", err)
	}
}

func TestConsensusAndNodeIdentityRejectMalformedOrMismatchedValues(t *testing.T) {
	publicKey := make([]byte, 32)
	for index := range publicKey {
		publicKey[index] = byte(31 - index)
	}
	publicKeyHex := hex.EncodeToString(publicKey)
	consensusAddress, err := consensusAddressFromPublicKeyHex(publicKeyHex)
	if err != nil {
		t.Fatal(err)
	}
	nodeID, err := nodeIDFromPublicKeyHex(publicKeyHex)
	if err != nil {
		t.Fatal(err)
	}

	badPublicKeys := []string{
		publicKeyHex[:len(publicKeyHex)-2],
		strings.ToUpper(publicKeyHex),
		strings.Repeat("z", 64),
	}
	for _, badPublicKey := range badPublicKeys {
		if _, err := consensusAddressFromPublicKeyHex(badPublicKey); err == nil {
			t.Fatalf("consensus derivation accepted public key %q", badPublicKey)
		}
		if _, err := nodeIDFromPublicKeyHex(badPublicKey); err == nil {
			t.Fatalf("node derivation accepted public key %q", badPublicKey)
		}
	}

	consensusCases := []string{
		strings.ToLower(consensusAddress),
		consensusAddress[:len(consensusAddress)-1],
		strings.Repeat("G", 40),
		"0000000000000000000000000000000000000000",
	}
	for _, candidate := range consensusCases {
		if err := validateConsensusAddress(publicKeyHex, candidate); err == nil {
			t.Fatalf("consensus address %q was accepted", candidate)
		}
	}

	nodeCases := []string{
		strings.ToUpper(nodeID),
		nodeID[:len(nodeID)-1],
		strings.Repeat("g", 40),
		"0000000000000000000000000000000000000000",
	}
	for _, candidate := range nodeCases {
		if err := validateNodeID(publicKeyHex, candidate); err == nil {
			t.Fatalf("node ID %q was accepted", candidate)
		}
	}
}

func TestEncodeZRNValoperRequiresTwentyBytes(t *testing.T) {
	for _, size := range []int{0, zrnValoperAddressBytes - 1, zrnValoperAddressBytes + 1} {
		if _, err := encodeZRNValoper(make([]byte, size)); err == nil {
			t.Fatalf("encoding %d bytes succeeded", size)
		}
	}
}

func encodeWithChecksumConstant(
	t *testing.T,
	hrp string,
	payload []byte,
	constant uint32,
) string {
	t.Helper()
	data, err := convertBits(payload, 8, 5, true)
	if err != nil {
		t.Fatal(err)
	}
	checksum := createBech32Checksum(hrp, data, constant)

	var encoded strings.Builder
	encoded.WriteString(hrp)
	encoded.WriteByte('1')
	for _, value := range append(data, checksum...) {
		encoded.WriteByte(bech32Charset[value])
	}
	return encoded.String()
}

func replacementBech32Character(value byte) string {
	if value == 'q' {
		return "p"
	}
	return "q"
}
