package types

import "cosmossdk.io/errors"

var (
	ErrReputationNotFound       = errors.Register(ModuleName, 2, "reputation not found")
	ErrInvalidValidator         = errors.Register(ModuleName, 3, "invalid validator address")
	ErrInvalidDomain            = errors.Register(ModuleName, 4, "invalid domain")
	ErrInsufficientVerifications = errors.Register(ModuleName, 5, "insufficient verifications for score")
	ErrMetricsNotFound          = errors.Register(ModuleName, 6, "capture metrics not found")
	ErrMismatchedArrayLengths   = errors.Register(ModuleName, 7, "mismatched array lengths")
	ErrInvalidStratum           = errors.Register(ModuleName, 8, "invalid stratum")
)
