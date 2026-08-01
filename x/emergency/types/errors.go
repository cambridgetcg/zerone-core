package types

import "cosmossdk.io/errors"

var (
	ErrNotGuardian                       = errors.Register(ModuleName, 2, "sender is not a Guardian-tier validator")
	ErrCeremonyActive                    = errors.Register(ModuleName, 3, "another ceremony is already active")
	ErrDuplicateVote                     = errors.Register(ModuleName, 4, "duplicate vote from this guardian")
	ErrPrevoteRequired                   = errors.Register(ModuleName, 5, "must prevote yes before precommit")
	ErrProposalLimitExceeded             = errors.Register(ModuleName, 6, "emergency proposal limit exceeded")
	ErrCooldownActive                    = errors.Register(ModuleName, 7, "emergency cooldown period still active")
	ErrStatusConflict                    = errors.Register(ModuleName, 8, "operation conflicts with current emergency status")
	ErrRevertDepthExceeded               = errors.Register(ModuleName, 9, "revert depth exceeds maximum allowed")
	ErrNoCeremony                        = errors.Register(ModuleName, 10, "no active ceremony found")
	ErrQuorumImpossible                  = errors.Register(ModuleName, 11, "quorum impossible: too many no votes")
	ErrInvalidPhase                      = errors.Register(ModuleName, 12, "ceremony is not in the expected phase")
	ErrCeremonyTimedOut                  = errors.Register(ModuleName, 13, "ceremony has timed out")
	ErrHaltRequired                      = errors.Register(ModuleName, 14, "chain must be halted for this operation")
	ErrInsufficientGuardians             = errors.Register(ModuleName, 15, "insufficient total guardian stake")
	ErrChainHalted                       = errors.Register(ModuleName, 16, "application transaction quarantine is active: only emergency coordination and approved recovery transactions are accepted")
	ErrUnsafeRevertDisabled              = errors.Register(ModuleName, 17, "height-only state revert is disabled: use a hash-bound forward recovery or an explicit social fork")
	ErrCeremonyIDExists                  = errors.Register(ModuleName, 18, "emergency ceremony id already exists")
	ErrUnsafeEmergencyBatch              = errors.Register(ModuleName, 19, "emergency status-transition messages cannot be batched with ordinary or governance messages")
	ErrUnauthenticatedEmergencyExecution = errors.Register(
		ModuleName,
		20,
		"emergency coordination messages require a direct authenticated transaction",
	)
)
