package types

import "cosmossdk.io/errors"

var (
	// Tool management
	ErrToolNotFound      = errors.Register(ModuleName, 2, "tool not found")
	ErrToolAlreadyExists = errors.Register(ModuleName, 3, "tool already exists")
	ErrNotDeployer       = errors.Register(ModuleName, 4, "not the tool deployer")
	ErrSharesNotSumTo100 = errors.Register(ModuleName, 5, "contributor shares must sum to 1,000,000 bps")
	ErrTooManyContributors = errors.Register(ModuleName, 6, "too many contributors")
	ErrContributorNotFound = errors.Register(ModuleName, 7, "contributor not found")
	ErrContributorExists   = errors.Register(ModuleName, 8, "contributor already exists")

	// Share lock & dependency
	ErrSharesLocked           = errors.Register(ModuleName, 9, "shares are locked")
	ErrDependencyCycle        = errors.Register(ModuleName, 10, "dependency cycle detected")
	ErrTooManyDependencies    = errors.Register(ModuleName, 11, "too many dependencies")
	ErrDependencyDepthExceeded = errors.Register(ModuleName, 12, "dependency depth exceeded")

	// Tool status
	ErrToolRetired    = errors.Register(ModuleName, 13, "tool is retired")
	ErrToolDeprecated = errors.Register(ModuleName, 14, "tool is deprecated")

	// Transaction/state
	ErrInsufficientBalance = errors.Register(ModuleName, 15, "insufficient balance")
	ErrFeeTooHigh          = errors.Register(ModuleName, 16, "fee exceeds max_fee")
	ErrNotRegisteredAgent  = errors.Register(ModuleName, 17, "caller is not a registered agent")
	ErrInvalidContractOwner = errors.Register(ModuleName, 18, "contract owner does not match deployer")
	ErrPendingContributorship = errors.Register(ModuleName, 19, "contributorship already pending")

	// Dependency & trust
	ErrNoPendingContributorship = errors.Register(ModuleName, 20, "no pending contributorship")
	ErrInvalidToolType          = errors.Register(ModuleName, 21, "invalid tool type")
	ErrInsufficientStake        = errors.Register(ModuleName, 22, "insufficient tool stake")
	ErrDependencyNotFound       = errors.Register(ModuleName, 23, "dependency not found")
	ErrIneligibleDependency     = errors.Register(ModuleName, 24, "dependency tool has insufficient trust")

	// Validation
	ErrUnauthorized     = errors.Register(ModuleName, 25, "unauthorized")
	ErrInvalidCategory  = errors.Register(ModuleName, 26, "invalid tool category")
	ErrFreeTierExhausted = errors.Register(ModuleName, 27, "free tier calls exhausted for this epoch")
	ErrInvalidLicense    = errors.Register(ModuleName, 28, "invalid license type")
	ErrInvalidStatus     = errors.Register(ModuleName, 29, "invalid tool status")
	ErrInvalidRole       = errors.Register(ModuleName, 30, "invalid contributor role")
	ErrInvalidParams     = errors.Register(ModuleName, 31, "invalid params")
)
