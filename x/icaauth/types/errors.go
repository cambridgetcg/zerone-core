package types

import "cosmossdk.io/errors"

var (
	ErrMaxAccountsReached   = errors.Register(ModuleName, 2, "maximum remote accounts reached for owner")
	ErrAlreadyRegistered    = errors.Register(ModuleName, 3, "interchain account already registered for this connection")
	ErrNotRegistered        = errors.Register(ModuleName, 4, "interchain account not registered")
	ErrMsgTypeNotAllowed    = errors.Register(ModuleName, 5, "message type not in allowlist")
	ErrRegistrationCooldown = errors.Register(ModuleName, 6, "registration cooldown not elapsed")
	ErrMaxMessagesExceeded  = errors.Register(ModuleName, 7, "maximum messages per transaction exceeded")
)
