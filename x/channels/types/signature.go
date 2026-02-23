package types

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	PackedSignatureSize = 96 // 32-byte pubkey + 64-byte signature
	Ed25519PubKeySize   = 32
	Ed25519SigSize      = 64
)

// ChannelSigningPayload builds a canonical signing payload for channel operations.
// Format: SHA-256("zerone.channels.v1:{op}" | "|" | chainId | "|" | channelId | "|" | BE(nonce) | "|" | spent)
func ChannelSigningPayload(operation, chainId, channelId string, nonce uint64, spent string) []byte {
	h := sha256.New()
	h.Write([]byte("zerone.channels.v1:" + operation))
	h.Write([]byte("|"))
	h.Write([]byte(chainId))
	h.Write([]byte("|"))
	h.Write([]byte(channelId))
	h.Write([]byte("|"))
	nonceBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBytes, nonce)
	h.Write(nonceBytes)
	h.Write([]byte("|"))
	h.Write([]byte(spent))
	return h.Sum(nil)
}

// PackSignature packs an Ed25519 public key and signature into a single byte slice.
func PackSignature(pubkey ed25519.PublicKey, sig []byte) []byte {
	packed := make([]byte, PackedSignatureSize)
	copy(packed[:Ed25519PubKeySize], pubkey)
	copy(packed[Ed25519PubKeySize:], sig)
	return packed
}

// VerifyPackedSignature verifies a packed [pubkey||sig] against a canonical payload
// and checks the pubkey derives to the expected bech32 address.
func VerifyPackedSignature(payload, packed []byte, expectedAddr string) error {
	if len(packed) != PackedSignatureSize {
		return fmt.Errorf("packed signature must be %d bytes, got %d", PackedSignatureSize, len(packed))
	}

	pubkey := ed25519.PublicKey(packed[:Ed25519PubKeySize])
	sig := packed[Ed25519PubKeySize:]

	// Derive address: SHA-256(pubkey)[:20] -> bech32
	addrHash := sha256.Sum256(pubkey)
	derivedAddr := sdk.AccAddress(addrHash[:20])
	if derivedAddr.String() != expectedAddr {
		return fmt.Errorf("pubkey does not match expected address: derived %s, expected %s", derivedAddr.String(), expectedAddr)
	}

	if !ed25519.Verify(pubkey, payload, sig) {
		return ErrInvalidSignature
	}
	return nil
}
