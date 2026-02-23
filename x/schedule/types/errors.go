package types

import "cosmossdk.io/errors"

var (
	ErrProcessNotFound       = errors.Register(ModuleName, 2, "process not found")
	ErrInvalidCondition      = errors.Register(ModuleName, 3, "invalid condition")
	ErrMaxSchedulesExceeded  = errors.Register(ModuleName, 4, "max schedules exceeded")
	ErrInsufficientFee       = errors.Register(ModuleName, 5, "insufficient fee")
	ErrProcessNotActive      = errors.Register(ModuleName, 6, "process not active")
	ErrProcessNotPaused      = errors.Register(ModuleName, 7, "process not paused")
	ErrUnauthorized          = errors.Register(ModuleName, 8, "unauthorized")
	ErrInvalidInterval       = errors.Register(ModuleName, 9, "invalid interval")
	ErrCompoundDepthExceeded = errors.Register(ModuleName, 10, "compound depth exceeded")
	ErrInvalidAmount         = errors.Register(ModuleName, 11, "invalid amount")
	ErrProcessCompleted      = errors.Register(ModuleName, 12, "process completed")
)
