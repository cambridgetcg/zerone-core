package types

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

const (
	// KeyRotationAuthorizationDomain separates operational-key proofs from
	// every other Ed25519 signature accepted by Zerone.
	KeyRotationAuthorizationDomain = "zerone.auth/rotate-key/v1"
	// KeyRotationAcceptanceDomain separates proof by the proposed new key from
	// authorization by the current key, while binding the same transition.
	KeyRotationAcceptanceDomain = "zerone.auth/accept-key/v1"

	// KeyRotationAuthorizationMaxTTL bounds how far the signed expiry may be in
	// the future relative to consensus block time. Without a signed-at value or
	// unpredictable chain challenge, it does not prove when a signature was made.
	// The transaction remains independently subject to Cosmos sequence and
	// timeout-height rules.
	KeyRotationAuthorizationMaxTTL = 10 * time.Minute
)

// KeyRotationAuthorizationSignBytes returns the exact bytes that the current
// operational Ed25519 key must sign. The current key version is a single-use
// nonce, chainID prevents cross-chain replay, and expiresAtUnix is checked
// against consensus block time by the message server.
//
// Encoding (all integers big endian):
//
//	domain || 0x00 || u32(len(chain_id)) || chain_id ||
//	u32(len(sender)) || sender || u32(current_key_version) ||
//	i64(authorization_expires_at_unix) || new_key[32]
func KeyRotationAuthorizationSignBytes(
	chainID string,
	sender string,
	currentKeyVersion uint32,
	newOperationalKey []byte,
	expiresAtUnix int64,
) ([]byte, error) {
	return keyRotationSignBytes(
		KeyRotationAuthorizationDomain,
		chainID,
		sender,
		currentKeyVersion,
		newOperationalKey,
		expiresAtUnix,
	)
}

// KeyRotationAcceptanceSignBytes returns the exact bytes the proposed new
// operational key must sign to confirm possession and acceptance of the same
// transition authorized by the current key.
func KeyRotationAcceptanceSignBytes(
	chainID string,
	sender string,
	currentKeyVersion uint32,
	newOperationalKey []byte,
	expiresAtUnix int64,
) ([]byte, error) {
	return keyRotationSignBytes(
		KeyRotationAcceptanceDomain,
		chainID,
		sender,
		currentKeyVersion,
		newOperationalKey,
		expiresAtUnix,
	)
}

func keyRotationSignBytes(
	domainName string,
	chainID string,
	sender string,
	currentKeyVersion uint32,
	newOperationalKey []byte,
	expiresAtUnix int64,
) ([]byte, error) {
	if chainID == "" || chainID != strings.TrimSpace(chainID) {
		return nil, fmt.Errorf("chain ID must be non-empty without surrounding whitespace")
	}
	if err := ValidateCanonicalAccountAddress(sender); err != nil {
		return nil, fmt.Errorf("invalid sender address: %w", err)
	}
	if currentKeyVersion == 0 {
		return nil, fmt.Errorf("current key version must be positive")
	}
	if len(newOperationalKey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("new operational key must be %d bytes", ed25519.PublicKeySize)
	}
	if expiresAtUnix <= 0 {
		return nil, fmt.Errorf("authorization expiry must be a positive Unix timestamp")
	}

	domain := []byte(domainName)
	chain := []byte(chainID)
	address := []byte(sender)
	if uint64(len(chain)) > uint64(^uint32(0)) || uint64(len(address)) > uint64(^uint32(0)) {
		return nil, fmt.Errorf("chain ID or sender is too long")
	}

	capacity := len(domain) + 1 + 4 + len(chain) + 4 + len(address) + 4 + 8 + ed25519.PublicKeySize
	result := make([]byte, 0, capacity)
	result = append(result, domain...)
	result = append(result, 0)
	result = appendUint32(result, uint32(len(chain)))
	result = append(result, chain...)
	result = appendUint32(result, uint32(len(address)))
	result = append(result, address...)
	result = appendUint32(result, currentKeyVersion)
	result = binary.BigEndian.AppendUint64(result, uint64(expiresAtUnix))
	result = append(result, newOperationalKey...)
	return result, nil
}

// OperationalKeyHash returns the canonical lowercase SHA-256 commitment for
// a 32-byte operational public key.
func OperationalKeyHash(key []byte) (string, error) {
	if len(key) != ed25519.PublicKeySize {
		return "", fmt.Errorf("operational key must be %d bytes", ed25519.PublicKeySize)
	}
	digest := sha256.Sum256(key)
	return hex.EncodeToString(digest[:]), nil
}

func appendUint32(dst []byte, value uint32) []byte {
	return binary.BigEndian.AppendUint32(dst, value)
}
