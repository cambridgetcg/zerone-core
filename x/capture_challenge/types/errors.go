package types

import "cosmossdk.io/errors"

var (
	ErrChallengeNotFound    = errors.Register(ModuleName, 2, "challenge not found")
	ErrInsufficientStake    = errors.Register(ModuleName, 3, "insufficient challenge stake")
	ErrChallengeNotOpen     = errors.Register(ModuleName, 4, "challenge is not open")
	ErrEvidenceDeadlinePassed = errors.Register(ModuleName, 5, "evidence deadline has passed")
	ErrNotChallenger        = errors.Register(ModuleName, 6, "only the challenger can add evidence")
	ErrInvalidOutcome       = errors.Register(ModuleName, 7, "invalid challenge outcome")
	ErrBountyPoolNotFound   = errors.Register(ModuleName, 8, "bounty pool not found")
	ErrDomainPaused         = errors.Register(ModuleName, 9, "domain is currently paused")
	ErrUnauthorized         = errors.Register(ModuleName, 10, "unauthorized")
)
