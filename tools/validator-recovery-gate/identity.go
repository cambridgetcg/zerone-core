package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

const (
	zrnValoperHRP              = "zrnvaloper"
	zrnValoperAddressBytes     = 20
	cometAddressBytes          = 20
	bech32ChecksumConstant     = uint32(1)
	bech32mChecksumConstant    = uint32(0x2bc830a3)
	bech32ChecksumLength       = 6
	bech32MaximumEncodedLength = 90
	bech32Charset              = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"
)

// validateZRNValoper accepts only the canonical lowercase Bech32 encoding of
// exactly 20 address bytes with the zrnvaloper human-readable prefix.
func validateZRNValoper(name, value string) error {
	_, err := decodeZRNValoper(value)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return err
}

func decodeZRNValoper(value string) ([]byte, error) {
	hrp, payload, err := decodeBech32(value)
	if err != nil {
		return nil, fmt.Errorf("SDK operator address must be canonical lowercase Bech32: %w", err)
	}
	if hrp != zrnValoperHRP {
		return nil, fmt.Errorf("SDK operator address must use the %s prefix", zrnValoperHRP)
	}
	if len(payload) != zrnValoperAddressBytes {
		return nil, fmt.Errorf(
			"SDK operator address must encode exactly %d bytes",
			zrnValoperAddressBytes,
		)
	}
	canonical, err := encodeZRNValoper(payload)
	if err != nil {
		return nil, errors.New("SDK operator address canonical encoding failed")
	}
	if canonical != value {
		return nil, errors.New("SDK operator address is not canonical")
	}
	return payload, nil
}

func encodeZRNValoper(payload []byte) (string, error) {
	if len(payload) != zrnValoperAddressBytes {
		return "", fmt.Errorf(
			"SDK operator address payload must be exactly %d bytes",
			zrnValoperAddressBytes,
		)
	}
	return encodeBech32(zrnValoperHRP, payload)
}

// consensusAddressFromPublicKeyHex derives CometBFT's uppercase consensus
// address: SHA-256(raw Ed25519 public key) truncated to 20 bytes.
func consensusAddressFromPublicKeyHex(publicKeyHex string) (string, error) {
	publicKey, err := decodeCanonicalEd25519PublicKeyHex(
		"consensus public key",
		publicKeyHex,
	)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(publicKey)
	return strings.ToUpper(hex.EncodeToString(sum[:cometAddressBytes])), nil
}

func validateConsensusAddress(publicKeyHex, address string) error {
	if err := validateUpperHexAddress("consensus address", address); err != nil {
		return err
	}
	expected, err := consensusAddressFromPublicKeyHex(publicKeyHex)
	if err != nil {
		return err
	}
	if address != expected {
		return errors.New("consensus address does not match the consensus public key")
	}
	return nil
}

// nodeIDFromPublicKeyHex derives CometBFT's lowercase node ID:
// SHA-256(raw Ed25519 public key) truncated to 20 bytes.
func nodeIDFromPublicKeyHex(publicKeyHex string) (string, error) {
	publicKey, err := decodeCanonicalEd25519PublicKeyHex("node public key", publicKeyHex)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(publicKey)
	return hex.EncodeToString(sum[:cometAddressBytes]), nil
}

func validateNodeID(publicKeyHex, nodeID string) error {
	if err := validateLowerHexAddress("node ID", nodeID); err != nil {
		return err
	}
	expected, err := nodeIDFromPublicKeyHex(publicKeyHex)
	if err != nil {
		return err
	}
	if nodeID != expected {
		return errors.New("node ID does not match the node public key")
	}
	return nil
}

func decodeCanonicalEd25519PublicKeyHex(name, value string) ([]byte, error) {
	if len(value) != ed25519.PublicKeySize*2 {
		return nil, fmt.Errorf("%s must be a lowercase hexadecimal Ed25519 public key", name)
	}
	decoded, err := hex.DecodeString(value)
	if err != nil || hex.EncodeToString(decoded) != value {
		return nil, fmt.Errorf("%s must be a lowercase hexadecimal Ed25519 public key", name)
	}
	return decoded, nil
}

func validateUpperHexAddress(name, value string) error {
	if len(value) != cometAddressBytes*2 {
		return fmt.Errorf("%s must be 40 uppercase hexadecimal characters", name)
	}
	decoded, err := hex.DecodeString(value)
	if err != nil || strings.ToUpper(hex.EncodeToString(decoded)) != value {
		return fmt.Errorf("%s must be 40 uppercase hexadecimal characters", name)
	}
	return nil
}

func validateLowerHexAddress(name, value string) error {
	if len(value) != cometAddressBytes*2 {
		return fmt.Errorf("%s must be 40 lowercase hexadecimal characters", name)
	}
	decoded, err := hex.DecodeString(value)
	if err != nil || hex.EncodeToString(decoded) != value {
		return fmt.Errorf("%s must be 40 lowercase hexadecimal characters", name)
	}
	return nil
}

// encodeBech32 encodes an eight-bit payload with the original BIP-0173
// checksum. It never emits Bech32m.
func encodeBech32(hrp string, payload []byte) (string, error) {
	if err := validateBech32HRP(hrp); err != nil {
		return "", err
	}
	data, err := convertBits(payload, 8, 5, true)
	if err != nil {
		return "", errors.New("Bech32 payload conversion failed")
	}
	checksum := createBech32Checksum(hrp, data, bech32ChecksumConstant)
	if len(hrp)+1+len(data)+len(checksum) > bech32MaximumEncodedLength {
		return "", errors.New("Bech32 encoding exceeds 90 characters")
	}

	var encoded strings.Builder
	encoded.Grow(len(hrp) + 1 + len(data) + len(checksum))
	encoded.WriteString(hrp)
	encoded.WriteByte('1')
	for _, value := range data {
		encoded.WriteByte(bech32Charset[value])
	}
	for _, value := range checksum {
		encoded.WriteByte(bech32Charset[value])
	}
	return encoded.String(), nil
}

// decodeBech32 accepts only lowercase BIP-0173 Bech32. In particular, a valid
// Bech32m checksum is deliberately rejected.
func decodeBech32(value string) (string, []byte, error) {
	if len(value) < 1+1+bech32ChecksumLength ||
		len(value) > bech32MaximumEncodedLength {
		return "", nil, errors.New("Bech32 length is invalid")
	}
	for index := 0; index < len(value); index++ {
		character := value[index]
		if character < 33 || character > 126 {
			return "", nil, errors.New("Bech32 contains a non-printable ASCII character")
		}
		if character >= 'A' && character <= 'Z' {
			return "", nil, errors.New("Bech32 must be lowercase")
		}
	}

	separator := strings.LastIndexByte(value, '1')
	if separator < 1 || separator+1+bech32ChecksumLength > len(value) {
		return "", nil, errors.New("Bech32 separator position is invalid")
	}
	hrp := value[:separator]
	if err := validateBech32HRP(hrp); err != nil {
		return "", nil, err
	}

	encodedData := value[separator+1:]
	data := make([]byte, len(encodedData))
	for index := range encodedData {
		charsetIndex := strings.IndexByte(bech32Charset, encodedData[index])
		if charsetIndex < 0 {
			return "", nil, errors.New("Bech32 data contains a character outside its alphabet")
		}
		data[index] = byte(charsetIndex)
	}

	values := append(bech32HRPExpand(hrp), data...)
	checksum := bech32Polymod(values)
	if checksum == bech32mChecksumConstant {
		return "", nil, errors.New("Bech32m checksum is not permitted")
	}
	if checksum != bech32ChecksumConstant {
		return "", nil, errors.New("Bech32 checksum is invalid")
	}

	payload, err := convertBits(data[:len(data)-bech32ChecksumLength], 5, 8, false)
	if err != nil {
		return "", nil, errors.New("Bech32 payload has non-canonical padding")
	}
	canonical, err := encodeBech32(hrp, payload)
	if err != nil || canonical != value {
		return "", nil, errors.New("Bech32 encoding is not canonical")
	}
	return hrp, payload, nil
}

func validateBech32HRP(hrp string) error {
	if hrp == "" {
		return errors.New("Bech32 human-readable prefix is empty")
	}
	for index := 0; index < len(hrp); index++ {
		character := hrp[index]
		if character < 33 || character > 126 {
			return errors.New("Bech32 human-readable prefix contains invalid characters")
		}
		if character >= 'A' && character <= 'Z' {
			return errors.New("Bech32 human-readable prefix must be lowercase")
		}
	}
	return nil
}

func createBech32Checksum(hrp string, data []byte, constant uint32) []byte {
	values := make([]byte, 0, len(hrp)*2+1+len(data)+bech32ChecksumLength)
	values = append(values, bech32HRPExpand(hrp)...)
	values = append(values, data...)
	values = append(values, make([]byte, bech32ChecksumLength)...)
	polymod := bech32Polymod(values) ^ constant
	checksum := make([]byte, bech32ChecksumLength)
	for index := range checksum {
		shift := uint(5 * (bech32ChecksumLength - 1 - index))
		checksum[index] = byte((polymod >> shift) & 31)
	}
	return checksum
}

func bech32HRPExpand(hrp string) []byte {
	expanded := make([]byte, 0, len(hrp)*2+1)
	for index := 0; index < len(hrp); index++ {
		expanded = append(expanded, hrp[index]>>5)
	}
	expanded = append(expanded, 0)
	for index := 0; index < len(hrp); index++ {
		expanded = append(expanded, hrp[index]&31)
	}
	return expanded
}

func bech32Polymod(values []byte) uint32 {
	generators := [...]uint32{
		0x3b6a57b2,
		0x26508e6d,
		0x1ea119fa,
		0x3d4233dd,
		0x2a1462b3,
	}
	checksum := uint32(1)
	for _, value := range values {
		top := checksum >> 25
		checksum = (checksum&0x1ffffff)<<5 ^ uint32(value)
		for bit, generator := range generators {
			if (top>>bit)&1 != 0 {
				checksum ^= generator
			}
		}
	}
	return checksum
}

// convertBits performs the power-of-two base conversion specified by Bech32.
// Decoding is strict: excess or non-zero padding is rejected.
func convertBits(data []byte, fromBits, toBits uint, pad bool) ([]byte, error) {
	if fromBits == 0 || fromBits > 8 || toBits == 0 || toBits > 8 {
		return nil, errors.New("bit group size must be between one and eight")
	}

	var accumulator uint32
	var bitCount uint
	maxOutputValue := uint32(1<<toBits) - 1
	maxAccumulator := uint32(1<<(fromBits+toBits-1)) - 1
	converted := make([]byte, 0, (len(data)*int(fromBits)+int(toBits)-1)/int(toBits))

	for _, value := range data {
		if uint32(value)>>fromBits != 0 {
			return nil, errors.New("input value exceeds its declared bit group")
		}
		accumulator = ((accumulator << fromBits) | uint32(value)) & maxAccumulator
		bitCount += fromBits
		for bitCount >= toBits {
			bitCount -= toBits
			converted = append(
				converted,
				byte((accumulator>>bitCount)&maxOutputValue),
			)
		}
	}

	if pad {
		if bitCount > 0 {
			converted = append(
				converted,
				byte((accumulator<<(toBits-bitCount))&maxOutputValue),
			)
		}
	} else if bitCount >= fromBits ||
		((accumulator<<(toBits-bitCount))&maxOutputValue) != 0 {
		return nil, errors.New("input contains invalid padding")
	}
	return converted, nil
}
