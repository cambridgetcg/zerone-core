package types

import "cosmossdk.io/errors"

var (
	ErrPotNotFound         = errors.Register(ModuleName, 2, "pot not found")
	ErrPotNotActive        = errors.Register(ModuleName, 3, "pot is not active")
	ErrIneligible          = errors.Register(ModuleName, 4, "claimant is not eligible")
	ErrAlreadyClaimed      = errors.Register(ModuleName, 5, "already claimed from this pot")
	ErrInsufficientPotFunds = errors.Register(ModuleName, 6, "insufficient pot funds")
	ErrCliffNotReached     = errors.Register(ModuleName, 7, "cliff period not reached")
	ErrBelowMinClaim       = errors.Register(ModuleName, 8, "claim amount below minimum")
	ErrMaxPotsReached      = errors.Register(ModuleName, 9, "maximum active pots reached")
	ErrInvalidConfig       = errors.Register(ModuleName, 10, "invalid pot configuration")
	ErrUnauthorized        = errors.Register(ModuleName, 11, "unauthorized")
)
