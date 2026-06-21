package types

import "cosmossdk.io/errors"

var (
	// Module-level sentinel errors. Codes start at 2 to match x/tokens convention
	// (code 1 is reserved by cosmossdk.io/errors for internal use).

	ErrNotConfigured       = errors.Register(ModuleName, 2, "contagion module is not configured for ZO yet")
	ErrAlreadyConfigured   = errors.Register(ModuleName, 3, "contagion module is already configured; formula is immutable")
	ErrUnauthorized        = errors.Register(ModuleName, 4, "sender is not the contagion authority")
	ErrReserveDepleted     = errors.Register(ModuleName, 5, "contagion reserve is depleted; sneeze is dormant")
	ErrReserveInsufficient = errors.Register(ModuleName, 6, "reserve cannot fund a full sneeze (need 154 ZO); partial rewards are not fired")
	ErrAlreadyInfected     = errors.Register(ModuleName, 7, "recipient is already infected; no sneeze")
	ErrTokenIdMismatch     = errors.Register(ModuleName, 8, "token id does not match the configured ZO token id")
	ErrInvalidAmount       = errors.Register(ModuleName, 9, "invalid amount: must be a positive integer")
	ErrInvalidReserve      = errors.Register(ModuleName, 10, "invalid reserve: must be a positive integer")
	ErrInvalidReward       = errors.Register(ModuleName, 11, "invalid reward: must be a positive integer")
	ErrInvalidAuthority    = errors.Register(ModuleName, 12, "invalid authority address")
	ErrInvalidAddress      = errors.Register(ModuleName, 13, "invalid address")
	ErrDecimalsMismatch    = errors.Register(ModuleName, 14, "decimals do not match the configured ZO token")
	ErrSelfSneeze          = errors.Register(ModuleName, 15, "sender and recipient cannot be the same")
	ErrTokenNotFound       = errors.Register(ModuleName, 16, "ZO token not found in tokens module")
)
