package types

import "cosmossdk.io/errors"

var (
	ErrSubmissionNotFound       = errors.Register(ModuleName, 2, "research submission not found")
	ErrInvalidSubmission        = errors.Register(ModuleName, 3, "invalid research submission")
	ErrBountyNotFound           = errors.Register(ModuleName, 4, "bounty not found")
	ErrBountyAlreadyClaimed     = errors.Register(ModuleName, 5, "bounty already claimed")
	ErrInsufficientTreasury     = errors.Register(ModuleName, 6, "insufficient research treasury balance")
	ErrUnauthorized             = errors.Register(ModuleName, 7, "unauthorized")
	ErrInvalidResearchType      = errors.Register(ModuleName, 8, "invalid research type")
	ErrInsufficientStake        = errors.Register(ModuleName, 9, "insufficient stake for research submission")
	ErrResearchNotChallengeable = errors.Register(ModuleName, 10, "research is not in a challengeable status")
	ErrAlreadyReviewed          = errors.Register(ModuleName, 11, "reviewer has already reviewed this submission")
	ErrResearchNotUnderReview   = errors.Register(ModuleName, 12, "research is not under review")
	ErrInsufficientReviews      = errors.Register(ModuleName, 13, "insufficient reviews to resolve")
	ErrBountyNotOpen            = errors.Register(ModuleName, 14, "bounty is not open")
	ErrBountyNotClaimed         = errors.Register(ModuleName, 15, "bounty is not claimed")
	ErrBountyExpired            = errors.Register(ModuleName, 16, "bounty has expired")
	ErrInvalidReviewVerdict     = errors.Register(ModuleName, 17, "invalid review verdict")
	ErrDeadlineTooSoon          = errors.Register(ModuleName, 18, "deadline too soon")
	ErrExceedsMaxReward         = errors.Register(ModuleName, 19, "reward exceeds maximum bounty reward")
)
