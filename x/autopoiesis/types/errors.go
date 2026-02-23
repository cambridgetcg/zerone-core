package types

import "cosmossdk.io/errors"

var (
	ErrUnauthorized      = errors.Register(ModuleName, 2, "unauthorized: sender is not governance authority")
	ErrInvalidParams     = errors.Register(ModuleName, 3, "invalid module parameters")
	ErrNotActivated      = errors.Register(ModuleName, 4, "autopoiesis module is not activated")
	ErrMultiplierFrozen  = errors.Register(ModuleName, 5, "multiplier is frozen by governance")
	ErrInvalidPath       = errors.Register(ModuleName, 6, "invalid multiplier path")
	ErrInvalidValue      = errors.Register(ModuleName, 7, "invalid multiplier value")
)
