package types

import "cosmossdk.io/errors"

var (
	ErrChannelNotFound       = errors.Register(ModuleName, 2, "channel not found")
	ErrChannelAlreadyClosed  = errors.Register(ModuleName, 3, "channel already closed")
	ErrInvalidStateUpdate    = errors.Register(ModuleName, 4, "invalid state update")
	ErrDisputePeriodActive   = errors.Register(ModuleName, 5, "dispute period still active")
	ErrInsufficientDeposit   = errors.Register(ModuleName, 6, "deposit below minimum")
	ErrChannelExpired        = errors.Register(ModuleName, 7, "channel expired")
	ErrNotChannelParty       = errors.Register(ModuleName, 8, "not a channel party")
	ErrInvalidNonce          = errors.Register(ModuleName, 9, "nonce must increase monotonically")
	ErrDisputeAlreadyActive  = errors.Register(ModuleName, 10, "dispute already active for channel")
	ErrNoActiveDispute       = errors.Register(ModuleName, 11, "no active dispute")
	ErrChannelDurationExceeded = errors.Register(ModuleName, 12, "channel duration exceeds maximum")
	ErrMaxChannelsExceeded   = errors.Register(ModuleName, 13, "max channels per pair exceeded")
	ErrInvalidSignature      = errors.Register(ModuleName, 14, "signature verification failed")
	ErrSpentExceedsDeposit   = errors.Register(ModuleName, 15, "spent exceeds deposited amount")
	ErrChannelNotOpen        = errors.Register(ModuleName, 16, "channel not open")
	ErrNotExpired            = errors.Register(ModuleName, 17, "channel not yet expired")
)
