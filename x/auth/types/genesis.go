package types

import (
	"bytes"
	"crypto/ed25519"
	"fmt"
	"strings"
)

const legacyExportChainID = "zerone-1"

// DefaultParams returns default module parameters.
func DefaultParams() Params {
	return Params{
		KeyRotationCooldown: 111,
		MaxMetadataLength:   1024,
		RequireDid:          false,
	}
}

// DefaultGenesis returns default genesis state.
func DefaultGenesis() *GenesisState {
	params := DefaultParams()
	return &GenesisState{
		Params:           &params,
		Accounts:         []*Account{},
		DidMappings:      []*DIDMapping{},
		LastKeyRotations: []*KeyRotationRecord{},
	}
}

// Validate validates genesis state for initialization. It is intentionally
// strict for every chain ID: AppModuleBasic.ValidateGenesis is not given a
// trusted chain ID, so historical export compatibility must never enter this
// path.
func (gs *GenesisState) Validate() error {
	return gs.validate(false)
}

// ValidateForExport validates state read from an already-running chain.
//
// The historical zerone-1 RegisterAccount implementation accepted 32- or
// 64-character, case-insensitive DID suffixes (and its canonical constructor
// used the first 32 identity-key hex characters) and allowed callers to omit
// operational_key_hash. Its export must preserve those committed values because
// a later binary cannot prove the original registration's possession evidence
// or safely invent provenance. The exceptions are deliberately limited to the
// exact historical chain and grammar; all other fields and all other chain IDs
// use current validation. This method is not an import validator.
func (gs *GenesisState) ValidateForExport(chainID string) error {
	return gs.validate(chainID == legacyExportChainID)
}

func (gs *GenesisState) validate(allowLegacyZerone1State bool) error {
	if gs == nil {
		return fmt.Errorf("genesis state cannot be nil")
	}
	if gs.Params == nil {
		return fmt.Errorf("genesis params cannot be nil")
	}
	if err := gs.Params.Validate(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}

	accountsByAddress := make(map[string]*Account, len(gs.Accounts))
	seenDIDs := make(map[string]struct{}, len(gs.Accounts))
	seenIdentityKeys := make(map[string]struct{}, len(gs.Accounts))
	for index, acc := range gs.Accounts {
		if acc == nil {
			return fmt.Errorf("account %d cannot be nil", index)
		}
		if err := ValidateCanonicalAccountAddress(acc.Address); err != nil {
			return fmt.Errorf("invalid account %d address: %w", index, err)
		}
		if _, exists := accountsByAddress[acc.Address]; exists {
			return fmt.Errorf("duplicate address: %s", acc.Address)
		}
		if _, exists := seenDIDs[acc.Did]; exists {
			return fmt.Errorf("duplicate DID: %s", acc.Did)
		}
		if err := validateGenesisDIDDerivation(acc.Did, acc.PublicKey, allowLegacyZerone1State); err != nil {
			return fmt.Errorf("invalid DID or identity key for account %s: %w", acc.Address, err)
		}
		if _, exists := seenIdentityKeys[acc.PublicKey]; exists {
			return fmt.Errorf("duplicate identity public key: %s", acc.PublicKey)
		}
		identityKey, err := DecodeEd25519PublicKeyHex(acc.PublicKey)
		if err != nil {
			return fmt.Errorf("invalid identity key for account %s: %w", acc.Address, err)
		}
		operationalKey, err := DecodeEd25519PublicKeyHex(acc.OperationalPublicKey)
		if err != nil {
			return fmt.Errorf("invalid operational key for account %s: %w", acc.Address, err)
		}
		expectedHash, err := OperationalKeyHash(operationalKey)
		if err != nil {
			return fmt.Errorf("invalid operational key for account %s: %w", acc.Address, err)
		}
		if acc.OperationalKeyHash != expectedHash &&
			!(allowLegacyZerone1State &&
				acc.OperationalKeyVersion == 1 &&
				acc.OperationalKeyHash == "") {
			return fmt.Errorf("operational key hash mismatch for account %s", acc.Address)
		}
		if acc.OperationalKeyVersion == 0 {
			return fmt.Errorf("operational key version must be positive for account %s", acc.Address)
		}
		if acc.OperationalKeyVersion == 1 && !bytes.Equal(identityKey, operationalKey) {
			return fmt.Errorf("version-1 operational key must equal identity key for account %s", acc.Address)
		}
		if err := ValidateAccountType(acc.AccountType); err != nil {
			return fmt.Errorf("invalid account type for %s: %w", acc.Address, err)
		}
		if acc.ReputationScore > 1_000_000 {
			return fmt.Errorf("reputation score exceeds 1000000 for account %s", acc.Address)
		}
		if acc.CreatedAtBlock == 0 {
			return fmt.Errorf("created_at_block must be positive for account %s", acc.Address)
		}
		if acc.LastActiveBlock < acc.CreatedAtBlock {
			return fmt.Errorf("last_active_block precedes created_at_block for account %s", acc.Address)
		}
		if acc.Flags == nil {
			return fmt.Errorf("flags cannot be nil for account %s", acc.Address)
		}
		if !acc.Flags.Frozen && acc.Flags.FreezeReason != "" {
			return fmt.Errorf("unfrozen account %s cannot retain a freeze reason", acc.Address)
		}
		if (acc.AccountType == "contract" || acc.AccountType == "system") &&
			(acc.Flags.CanSubmitClaims || acc.Flags.CanChallenge) {
			return fmt.Errorf("%s account %s cannot submit claims or challenges", acc.AccountType, acc.Address)
		}
		if uint64(len(acc.Metadata)) > uint64(gs.Params.MaxMetadataLength) {
			return fmt.Errorf("metadata exceeds maximum length for account %s", acc.Address)
		}

		accountsByAddress[acc.Address] = acc
		seenDIDs[acc.Did] = struct{}{}
		seenIdentityKeys[acc.PublicKey] = struct{}{}
	}

	mappedAddresses := make(map[string]struct{}, len(gs.DidMappings))
	mappedDIDs := make(map[string]struct{}, len(gs.DidMappings))
	for index, mapping := range gs.DidMappings {
		if mapping == nil {
			return fmt.Errorf("DID mapping %d cannot be nil", index)
		}
		if err := ValidateCanonicalAccountAddress(mapping.Bech32); err != nil {
			return fmt.Errorf("invalid address in DID mapping %d: %w", index, err)
		}
		if err := validateGenesisDIDDerivation(mapping.Did, mapping.PubKey, allowLegacyZerone1State); err != nil {
			return fmt.Errorf("invalid DID mapping %d: %w", index, err)
		}
		if _, exists := mappedDIDs[mapping.Did]; exists {
			return fmt.Errorf("duplicate DID mapping: %s", mapping.Did)
		}
		if _, exists := mappedAddresses[mapping.Bech32]; exists {
			return fmt.Errorf("duplicate DID mapping address: %s", mapping.Bech32)
		}
		account, exists := accountsByAddress[mapping.Bech32]
		if !exists {
			return fmt.Errorf("DID mapping %s points to missing account %s", mapping.Did, mapping.Bech32)
		}
		if mapping.Did != account.Did || mapping.PubKey != account.PublicKey {
			return fmt.Errorf("DID mapping %s does not exactly match account %s", mapping.Did, mapping.Bech32)
		}
		mappedDIDs[mapping.Did] = struct{}{}
		mappedAddresses[mapping.Bech32] = struct{}{}
	}
	if len(mappedAddresses) != len(accountsByAddress) {
		return fmt.Errorf("every account must have exactly one DID mapping")
	}

	rotationsByAddress := make(map[string]uint64, len(gs.LastKeyRotations))
	for index, rotation := range gs.LastKeyRotations {
		if rotation == nil {
			return fmt.Errorf("key rotation record %d cannot be nil", index)
		}
		if err := ValidateCanonicalAccountAddress(rotation.Address); err != nil {
			return fmt.Errorf("invalid key rotation record %d address: %w", index, err)
		}
		if rotation.Height == 0 {
			return fmt.Errorf("key rotation height must be positive for account %s", rotation.Address)
		}
		if _, exists := rotationsByAddress[rotation.Address]; exists {
			return fmt.Errorf("duplicate key rotation record for account %s", rotation.Address)
		}
		account, exists := accountsByAddress[rotation.Address]
		if !exists {
			return fmt.Errorf("key rotation record points to missing account %s", rotation.Address)
		}
		if rotation.Height < account.CreatedAtBlock || rotation.Height > account.LastActiveBlock {
			return fmt.Errorf("key rotation height is outside account activity bounds for %s", rotation.Address)
		}
		rotationsByAddress[rotation.Address] = rotation.Height
	}
	for address, account := range accountsByAddress {
		_, hasRotation := rotationsByAddress[address]
		if account.OperationalKeyVersion == 1 && hasRotation {
			return fmt.Errorf("version-1 account %s cannot have a key rotation record", address)
		}
		if account.OperationalKeyVersion > 1 && !hasRotation {
			return fmt.Errorf("rotated account %s is missing its key rotation record", address)
		}
	}

	return nil
}

func validateGenesisDIDDerivation(did, publicKey string, allowLegacyZerone1State bool) error {
	if allowLegacyZerone1State {
		// The historical validator required this exact lowercase prefix, accepted
		// 32- or 64-character hex suffixes case-insensitively, and compared them
		// case-insensitively with the key (or its first 32 characters). Keep the
		// key itself on today's strict Ed25519 validation; only that old DID
		// spelling and derivation grammar is grandfathered.
		if _, err := DecodeEd25519PublicKeyHex(publicKey); err == nil && strings.HasPrefix(did, "did:zrn:") {
			suffix := strings.TrimPrefix(did, "did:zrn:")
			switch len(suffix) {
			case ed25519.PublicKeySize:
				if strings.EqualFold(suffix, publicKey[:ed25519.PublicKeySize]) {
					return nil
				}
			case ed25519.PublicKeySize * 2:
				if strings.EqualFold(suffix, publicKey) {
					return nil
				}
			}
		}
	}
	return ValidateDIDDerivation(did, publicKey)
}
