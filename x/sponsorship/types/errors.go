package types

import "cosmossdk.io/errors"

var (
	// ErrUnauthorized is raised when the caller is not the bounty's sponsor.
	ErrUnauthorized = errors.Register(ModuleName, 2, "unauthorized")

	// ErrBountyNotFound is raised when a bounty id has no record.
	ErrBountyNotFound = errors.Register(ModuleName, 3, "bounty order not found")

	// ErrBountyNotActive is raised when the bounty's status is not ACTIVE.
	// Commitment 1 (methodology over statement): a fulfilled or canceled
	// bounty cannot be revived by editorial decree.
	ErrBountyNotActive = errors.Register(ModuleName, 4, "bounty order not active")

	// ErrBountyExpired is raised when currentBlock >= bounty.end_block.
	// Sponsor's commitment had a deadline; the chain honors that deadline
	// without exception.
	ErrBountyExpired = errors.Register(ModuleName, 5, "bounty order expired")

	// ErrAlreadyFulfilled is raised when (bounty_id, fact_id) has an
	// existing fulfillment. Each fact qualifies once per bounty.
	ErrAlreadyFulfilled = errors.Register(ModuleName, 6, "fact already fulfilled this bounty")

	// ErrFactNotEligible is raised when the fact does not meet the bounty's
	// criteria. The chain refuses to pay for unverified, off-domain, or
	// out-of-window facts (commitment 8: panel weights skill, not sponsor's
	// preference; the chain decides what counts).
	ErrFactNotEligible = errors.Register(ModuleName, 7, "fact not eligible for this bounty")

	// ErrInvalidConfig is raised for malformed bounty configuration.
	ErrInvalidConfig = errors.Register(ModuleName, 8, "invalid bounty configuration")

	// ErrInsufficientEscrow is raised when the sponsor lacks the funds to
	// back the bounty they're trying to create.
	ErrInsufficientEscrow = errors.Register(ModuleName, 9, "insufficient escrow balance")

	// ErrWorkContractRequired keeps v1 orders refund-only after the v2
	// activation boundary.
	ErrWorkContractRequired = errors.Register(ModuleName, 10, "bound work contract required")

	ErrSettlementReplay = errors.Register(ModuleName, 11, "sponsorship settlement already consumed")

	ErrEscrowInvariant = errors.Register(ModuleName, 12, "escrow liability invariant violated")

	ErrFactNotMature = errors.Register(ModuleName, 13, "fact has not completed challenge survival")

	ErrBountyNotCancelable = errors.Register(ModuleName, 14, "bound bounty cannot be canceled before its deadline")

	ErrAgentEconomyDisabled = errors.Register(ModuleName, 15, "agent economy is not activated")
)
