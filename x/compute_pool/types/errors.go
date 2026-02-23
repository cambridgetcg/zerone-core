package types

import "cosmossdk.io/errors"

var (
	ErrProviderNotFound    = errors.Register(ModuleName, 2, "provider not found")
	ErrProviderExists      = errors.Register(ModuleName, 3, "provider already exists")
	ErrProviderNotActive   = errors.Register(ModuleName, 4, "provider not active")
	ErrInsufficientStake   = errors.Register(ModuleName, 5, "insufficient stake")
	ErrInvalidServiceType  = errors.Register(ModuleName, 6, "invalid service type")
	ErrInvalidAmount       = errors.Register(ModuleName, 7, "invalid amount")
	ErrUnauthorized        = errors.Register(ModuleName, 8, "unauthorized")
	ErrInsufficientCredits = errors.Register(ModuleName, 9, "insufficient credits")
	ErrHeartbeatTooEarly   = errors.Register(ModuleName, 10, "heartbeat too early")
	ErrPriceExceedsCeiling = errors.Register(ModuleName, 11, "price exceeds ceiling")
	ErrProviderUnbonding   = errors.Register(ModuleName, 12, "provider is unbonding")
)
