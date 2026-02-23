package types

import "cosmossdk.io/errors"

var (
	ErrContractNotFound     = errors.Register(ModuleName, 2, "contract not found")
	ErrExecutionFailed      = errors.Register(ModuleName, 3, "contract execution failed")
	ErrOutOfGas             = errors.Register(ModuleName, 4, "contract execution out of gas")
	ErrInvalidBytecode      = errors.Register(ModuleName, 5, "invalid contract bytecode")
	ErrStaticCallViolation  = errors.Register(ModuleName, 6, "state modification in static call context")
	ErrCodeNotFound         = errors.Register(ModuleName, 7, "contract code not found")
	ErrContractExists       = errors.Register(ModuleName, 8, "contract already exists at address")
	ErrScheduleNotFound     = errors.Register(ModuleName, 9, "schedule not found")
	ErrScheduleNotOwner     = errors.Register(ModuleName, 10, "sender is not the schedule owner")
	ErrBytecodeTooBig       = errors.Register(ModuleName, 11, "bytecode exceeds maximum size")
	ErrStateEntryLimit      = errors.Register(ModuleName, 12, "contract state entry limit exceeded")
	ErrNotContractCreator   = errors.Register(ModuleName, 13, "sender is not the contract creator")
	ErrScheduleLimitReached = errors.Register(ModuleName, 14, "contract schedule limit reached")
	ErrScheduleInPast       = errors.Register(ModuleName, 15, "schedule execution block is in the past")
)
