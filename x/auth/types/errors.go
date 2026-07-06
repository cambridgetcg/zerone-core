package types

import (
	"cosmossdk.io/errors"
)

// Error codes 4, 5, 11, 13, 16 (sessions), 19-24 (social recovery), and
// 26-28 (bootstrap auto-claim) were retired in the 2026-07 slim cut and
// must not be reused.
var (
	ErrAccountNotFound         = errors.Register(ModuleName, 2, "account not found")
	ErrDIDNotFound             = errors.Register(ModuleName, 3, "DID not found")
	ErrUnauthorizedKey         = errors.Register(ModuleName, 6, "unauthorized key operation")
	ErrKeyRotationCooldown     = errors.Register(ModuleName, 7, "key rotation cooldown active")
	ErrAccountFrozen           = errors.Register(ModuleName, 8, "account is frozen")
	ErrInvalidDID              = errors.Register(ModuleName, 9, "invalid DID format")
	ErrDuplicateDID            = errors.Register(ModuleName, 10, "DID already registered")
	ErrInvalidKeyType          = errors.Register(ModuleName, 12, "invalid key type")
	ErrAccountAlreadyExists    = errors.Register(ModuleName, 14, "account already exists")
	ErrInvalidPublicKey        = errors.Register(ModuleName, 15, "invalid public key")
	ErrDIDResolutionFailed     = errors.Register(ModuleName, 17, "DID resolution failed")
	ErrUnauthorized            = errors.Register(ModuleName, 18, "unauthorized")
	ErrAccountNotFrozen        = errors.Register(ModuleName, 25, "account is not frozen")
	ErrDIDDerivationMismatch   = errors.Register(ModuleName, 29, "DID does not derive from public key")
	ErrAccountCapabilityDenied = errors.Register(ModuleName, 30, "account capability denied")
)
