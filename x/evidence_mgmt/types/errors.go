package types

import "cosmossdk.io/errors"

var (
	ErrEvidenceNotFound        = errors.Register(ModuleName, 2, "evidence not found")
	ErrInvalidEvidence         = errors.Register(ModuleName, 3, "invalid evidence")
	ErrDuplicateEvidence       = errors.Register(ModuleName, 4, "duplicate evidence")
	ErrInsufficientVerifierTier = errors.Register(ModuleName, 5, "verifier tier below minimum")
	ErrChallengeWindowClosed   = errors.Register(ModuleName, 6, "challenge window closed")
	ErrAlreadyVerified         = errors.Register(ModuleName, 7, "evidence already verified by this verifier")
	ErrSelfVerification        = errors.Register(ModuleName, 8, "submitter cannot verify own evidence")
	ErrInvalidParams           = errors.Register(ModuleName, 9, "invalid parameters")
	ErrUnauthorized            = errors.Register(ModuleName, 10, "unauthorized")
	ErrNotCustodian            = errors.Register(ModuleName, 11, "sender is not the current custodian")
)
