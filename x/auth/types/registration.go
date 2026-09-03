package types

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	// AccountRegistrationProofDomain separates identity-key registration proofs
	// from every other Ed25519 signature accepted by Zerone.
	AccountRegistrationProofDomain = "zerone.auth/register-account/v1"
)

// AccountRegistrationProofSignBytes returns the exact bytes the identity key
// must sign to authorize its binding to a transaction account.
//
// Encoding (all lengths are unsigned 32-bit big endian):
//
//	domain || 0x00 || u32(len(chain_id)) || chain_id ||
//	u32(len(sender)) || sender || u32(len(did)) || did ||
//	identity_public_key[32] || u32(len(account_type)) || account_type ||
//	u32(len(metadata)) || metadata
//
// operational_key_hash is intentionally absent: it is a deterministic SHA-256
// commitment derived by the handler from identity_public_key.
func AccountRegistrationProofSignBytes(
	chainID string,
	sender string,
	did string,
	identityPublicKey []byte,
	accountType string,
	metadata string,
) ([]byte, error) {
	if chainID == "" || chainID != strings.TrimSpace(chainID) {
		return nil, fmt.Errorf("chain ID must be non-empty without surrounding whitespace")
	}
	if err := ValidateCanonicalAccountAddress(sender); err != nil {
		return nil, fmt.Errorf("invalid sender address: %w", err)
	}
	if len(identityPublicKey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("identity public key must be %d bytes", ed25519.PublicKeySize)
	}
	if err := ValidateDIDDerivation(did, hex.EncodeToString(identityPublicKey)); err != nil {
		return nil, fmt.Errorf("invalid DID derivation: %w", err)
	}
	if err := ValidateAccountType(accountType); err != nil {
		return nil, err
	}

	fields := []string{chainID, sender, did, accountType, metadata}
	for _, field := range fields {
		if uint64(len(field)) > uint64(^uint32(0)) {
			return nil, fmt.Errorf("registration proof field exceeds uint32 length")
		}
	}

	result := make([]byte, 0,
		len(AccountRegistrationProofDomain)+1+
			5*4+len(chainID)+len(sender)+len(did)+len(identityPublicKey)+len(accountType)+len(metadata),
	)
	result = append(result, AccountRegistrationProofDomain...)
	result = append(result, 0)
	result = appendLengthPrefixedString(result, chainID)
	result = appendLengthPrefixedString(result, sender)
	result = appendLengthPrefixedString(result, did)
	result = append(result, identityPublicKey...)
	result = appendLengthPrefixedString(result, accountType)
	result = appendLengthPrefixedString(result, metadata)
	return result, nil
}

func appendLengthPrefixedString(dst []byte, value string) []byte {
	dst = appendUint32(dst, uint32(len(value)))
	return append(dst, value...)
}
