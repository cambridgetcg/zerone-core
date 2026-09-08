package types

import "cosmossdk.io/errors"

var (
	ErrUnauthorized       = errors.Register(ModuleName, 2, "unauthorized")
	ErrAdmissionClosed    = errors.Register(ModuleName, 3, "new schedule admission is closed")
	ErrScheduleNotFound   = errors.Register(ModuleName, 4, "schedule not found")
	ErrScheduleNotActive  = errors.Register(ModuleName, 5, "schedule is not active")
	ErrRevisionConflict   = errors.Register(ModuleName, 6, "schedule revision conflict")
	ErrExecutionConflict  = errors.Register(ModuleName, 7, "schedule execution count conflict")
	ErrInvalidSchedule    = errors.Register(ModuleName, 8, "invalid schedule")
	ErrScheduleLimit      = errors.Register(ModuleName, 9, "active schedule limit reached")
	ErrInsufficientEscrow = errors.Register(ModuleName, 10, "insufficient schedule escrow")
	ErrReceiptNotFound    = errors.Register(ModuleName, 11, "execution receipt not found")
	ErrBlockedRecipient   = errors.Register(ModuleName, 12, "recipient cannot receive module transfers")
	ErrEscrowInvariant    = errors.Register(ModuleName, 13, "schedule escrow invariant violated")
)
