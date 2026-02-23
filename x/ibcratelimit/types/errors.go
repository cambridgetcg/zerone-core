package types

import "cosmossdk.io/errors"

var (
	ErrSendRateExceeded  = errors.Register(ModuleName, 2, "send rate limit exceeded")
	ErrRecvRateExceeded  = errors.Register(ModuleName, 3, "receive rate limit exceeded")
	ErrRateLimitNotFound = errors.Register(ModuleName, 4, "rate limit not found")
	ErrInvalidRateLimit  = errors.Register(ModuleName, 5, "invalid rate limit")
)
