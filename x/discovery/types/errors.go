package types

import "cosmossdk.io/errors"

var (
	ErrAgentNotFound     = errors.Register(ModuleName, 2, "agent profile not found")
	ErrAgentExists       = errors.Register(ModuleName, 3, "agent already registered")
	ErrInsufficientStake = errors.Register(ModuleName, 4, "insufficient registration stake")
	ErrMaxCapabilities   = errors.Register(ModuleName, 5, "max capabilities per agent exceeded")
	ErrAgentInactive     = errors.Register(ModuleName, 6, "agent profile is not active")
	ErrUnauthorized      = errors.Register(ModuleName, 7, "unauthorized")
	ErrInvalidAmount     = errors.Register(ModuleName, 8, "invalid amount")
)
