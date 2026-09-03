package types

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Validate validates module parameters.
func (p *Params) Validate() error {
	if p == nil {
		return fmt.Errorf("params cannot be nil")
	}
	if p.MaxMetadataLength == 0 {
		return fmt.Errorf("max_metadata_length must be > 0")
	}
	return nil
}

// ValidateCanonicalAccountAddress validates the one canonical address form
// accepted by zerone_auth state and authorization encodings.
func ValidateCanonicalAccountAddress(address string) error {
	parsed, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return fmt.Errorf("invalid account address: %w", err)
	}
	if len(parsed) != cosmosAccountAddressBytes {
		return fmt.Errorf("account address must decode to %d bytes, got %d", cosmosAccountAddressBytes, len(parsed))
	}
	if parsed.String() != address {
		return fmt.Errorf("account address must use canonical lowercase Bech32 encoding")
	}
	return nil
}

// ValidateDID validates the sole canonical DID form used by zerone-2:
// did:zrn:{the full 32-byte identity public key as lowercase hex}.
func ValidateDID(did string) error {
	if !strings.HasPrefix(did, "did:zrn:") {
		return fmt.Errorf("DID must start with 'did:zrn:'")
	}
	suffix := strings.TrimPrefix(did, "did:zrn:")
	if len(suffix) != ed25519.PublicKeySize*2 {
		return fmt.Errorf("DID suffix must be %d lowercase hex characters, got %d", ed25519.PublicKeySize*2, len(suffix))
	}
	decoded, err := hex.DecodeString(suffix)
	if err != nil || hex.EncodeToString(decoded) != suffix {
		return fmt.Errorf("DID suffix must be lowercase hex")
	}
	return nil
}

// PublicKeyToDID derives the canonical DID from a hex-encoded Ed25519 public key.
// Format: did:zrn:{full 64-lower-hex public key}.
func PublicKeyToDID(pubKeyHex string) string {
	if _, err := DecodeEd25519PublicKeyHex(pubKeyHex); err != nil {
		return ""
	}
	return "did:zrn:" + pubKeyHex
}

// ValidateDIDDerivation checks that a DID correctly derives from the given public key.
func ValidateDIDDerivation(did string, pubKeyHex string) error {
	if err := ValidateDID(did); err != nil {
		return err
	}
	if _, err := DecodeEd25519PublicKeyHex(pubKeyHex); err != nil {
		return err
	}
	expected := "did:zrn:" + pubKeyHex
	if did != expected {
		return fmt.Errorf("DID does not derive from public key: expected %s, got %s", expected, did)
	}
	return nil
}

// DecodeEd25519PublicKeyHex decodes the canonical lowercase hex form and
// applies the strict curve/subgroup validation used by all auth entry points.
func DecodeEd25519PublicKeyHex(value string) ([]byte, error) {
	if len(value) != ed25519.PublicKeySize*2 {
		return nil, fmt.Errorf("public key must be %d lowercase hex characters", ed25519.PublicKeySize*2)
	}
	decoded, err := hex.DecodeString(value)
	if err != nil || hex.EncodeToString(decoded) != value {
		return nil, fmt.Errorf("public key must be lowercase hex")
	}
	if err := ValidateEd25519PublicKey(decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

// ValidateAccountType validates the closed account-type vocabulary.
func ValidateAccountType(accountType string) error {
	switch accountType {
	case "agent", "human", "contract", "system":
		return nil
	default:
		return fmt.Errorf("account_type must be agent, human, contract, or system")
	}
}

// sdk.Msg interface implementations for proto-generated types.

func (msg *MsgRotateKey) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{sender}
}

func (msg *MsgRotateKey) ValidateBasic() error {
	if err := ValidateCanonicalAccountAddress(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if err := ValidateEd25519PublicKey(msg.NewOperationalKey); err != nil {
		return fmt.Errorf("invalid new_operational_key: %w", err)
	}
	if len(msg.AuthorizationSignature) != ed25519.SignatureSize {
		return fmt.Errorf("authorization_signature must be %d bytes", ed25519.SignatureSize)
	}
	if len(msg.NewKeyConfirmationSignature) != ed25519.SignatureSize {
		return fmt.Errorf("new_key_confirmation_signature must be %d bytes", ed25519.SignatureSize)
	}
	if msg.AuthorizationExpiresAtUnix <= 0 {
		return fmt.Errorf("authorization_expires_at_unix must be positive")
	}
	return nil
}

func (msg *MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

func (msg *MsgUpdateParams) ValidateBasic() error {
	if err := ValidateCanonicalAccountAddress(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if msg.Params == nil {
		return fmt.Errorf("params cannot be nil")
	}
	return msg.Params.Validate()
}

func (msg *MsgRegisterAccount) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{sender}
}

func (msg *MsgRegisterAccount) ValidateBasic() error {
	if err := ValidateCanonicalAccountAddress(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if err := ValidateDIDDerivation(msg.Did, msg.PublicKey); err != nil {
		return fmt.Errorf("DID derivation mismatch: %w", err)
	}
	publicKey, err := DecodeEd25519PublicKeyHex(msg.PublicKey)
	if err != nil {
		return fmt.Errorf("invalid public_key: %w", err)
	}
	if msg.OperationalKeyHash != "" {
		expected, err := OperationalKeyHash(publicKey)
		if err != nil {
			return err
		}
		if msg.OperationalKeyHash != expected {
			return fmt.Errorf("%w: operational_key_hash must be the lowercase SHA-256 of public_key", ErrInvalidPublicKey)
		}
	}
	if err := ValidateAccountType(msg.AccountType); err != nil {
		return err
	}
	if len(msg.IdentityProofSignature) != ed25519.SignatureSize {
		return fmt.Errorf("identity_proof_signature must be %d bytes", ed25519.SignatureSize)
	}
	return nil
}

func (msg *MsgFreezeAccount) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{sender}
}

func (msg *MsgFreezeAccount) ValidateBasic() error {
	if err := ValidateCanonicalAccountAddress(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if err := ValidateCanonicalAccountAddress(msg.Address); err != nil {
		return fmt.Errorf("invalid target address: %w", err)
	}
	return nil
}

func (msg *MsgUnfreezeAccount) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{sender}
}

func (msg *MsgUnfreezeAccount) ValidateBasic() error {
	if err := ValidateCanonicalAccountAddress(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if err := ValidateCanonicalAccountAddress(msg.Address); err != nil {
		return fmt.Errorf("invalid target address: %w", err)
	}
	return nil
}
