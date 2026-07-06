package types

import (
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Validate validates module parameters.
func (p *Params) Validate() error {
	if p.MaxMetadataLength == 0 {
		return fmt.Errorf("max_metadata_length must be > 0")
	}
	return nil
}

// ValidateDID validates DID format: did:zrn:{hex}
// Accepts 32-char (canonical) or 64-char (full pubkey) hex suffixes.
func ValidateDID(did string) error {
	if !strings.HasPrefix(did, "did:zrn:") {
		return fmt.Errorf("DID must start with 'did:zrn:'")
	}
	suffix := strings.TrimPrefix(did, "did:zrn:")
	if len(suffix) != 32 && len(suffix) != 64 {
		return fmt.Errorf("DID suffix must be 32 or 64 hex characters, got %d", len(suffix))
	}
	for _, c := range suffix {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return fmt.Errorf("DID suffix contains non-hex character: %c", c)
		}
	}
	return nil
}

// PublicKeyToDID derives the canonical DID from a hex-encoded Ed25519 public key.
// Format: did:zrn:{first 32 hex chars of pubkey}
func PublicKeyToDID(pubKeyHex string) string {
	if len(pubKeyHex) < 32 {
		return ""
	}
	return "did:zrn:" + strings.ToLower(pubKeyHex[:32])
}

// ValidateDIDDerivation checks that a DID correctly derives from the given public key.
func ValidateDIDDerivation(did string, pubKeyHex string) error {
	if len(pubKeyHex) != 64 {
		return fmt.Errorf("public key must be 64 hex characters, got %d", len(pubKeyHex))
	}
	suffix := strings.TrimPrefix(did, "did:zrn:")
	switch len(suffix) {
	case 32:
		expected := strings.ToLower(pubKeyHex[:32])
		if strings.ToLower(suffix) != expected {
			return fmt.Errorf("DID does not derive from public key: expected did:zrn:%s, got %s", expected, did)
		}
	case 64:
		if !strings.EqualFold(suffix, pubKeyHex) {
			return fmt.Errorf("DID does not match full public key: expected did:zrn:%s, got %s", strings.ToLower(pubKeyHex), did)
		}
	default:
		return fmt.Errorf("DID suffix must be 32 or 64 hex characters, got %d", len(suffix))
	}
	return nil
}

// sdk.Msg interface implementations for proto-generated types.

func (msg *MsgRotateKey) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{sender}
}

func (msg *MsgRotateKey) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if len(msg.NewOperationalKey) == 0 {
		return fmt.Errorf("new_operational_key cannot be empty")
	}
	if len(msg.AuthorizationSignature) == 0 {
		return fmt.Errorf("authorization_signature cannot be empty")
	}
	return nil
}

func (msg *MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

func (msg *MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if msg.Params != nil {
		return msg.Params.Validate()
	}
	return nil
}

func (msg *MsgRegisterAccount) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{sender}
}

func (msg *MsgRegisterAccount) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if err := ValidateDID(msg.Did); err != nil {
		return fmt.Errorf("invalid DID: %w", err)
	}
	if msg.PublicKey == "" {
		return fmt.Errorf("public_key cannot be empty")
	}
	if len(msg.PublicKey) != 64 {
		return fmt.Errorf("public_key must be 64 hex characters (32 bytes Ed25519)")
	}
	if err := ValidateDIDDerivation(msg.Did, msg.PublicKey); err != nil {
		return fmt.Errorf("DID derivation mismatch: %w", err)
	}
	validTypes := map[string]bool{"agent": true, "human": true, "contract": true, "system": true}
	if !validTypes[msg.AccountType] {
		return fmt.Errorf("account_type must be agent, human, contract, or system")
	}
	return nil
}

func (msg *MsgFreezeAccount) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{sender}
}

func (msg *MsgFreezeAccount) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.Address); err != nil {
		return fmt.Errorf("invalid target address: %w", err)
	}
	return nil
}

func (msg *MsgUnfreezeAccount) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{sender}
}

func (msg *MsgUnfreezeAccount) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.Address); err != nil {
		return fmt.Errorf("invalid target address: %w", err)
	}
	return nil
}

