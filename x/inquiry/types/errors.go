package types

import "cosmossdk.io/errors"

var (
	ErrInquiryNotFound       = errors.Register(ModuleName, 2, "inquiry not found")
	ErrSubmissionsDisabled   = errors.Register(ModuleName, 3, "inquiry submissions are disabled")
	ErrEmptyQuestion         = errors.Register(ModuleName, 4, "question cannot be empty")
	ErrTextTooLong           = errors.Register(ModuleName, 5, "text exceeds max length")
	ErrBountyTooLow          = errors.Register(ModuleName, 6, "bounty below min_bounty")
	ErrInvalidBounty         = errors.Register(ModuleName, 7, "invalid bounty amount")
	ErrInvalidExpiry         = errors.Register(ModuleName, 8, "invalid expiry window")
	ErrInquiryNotOpen        = errors.Register(ModuleName, 9, "inquiry is not open for new answers")
	ErrInquiryAlreadyResolved = errors.Register(ModuleName, 10, "inquiry already resolved")
	ErrClaimNotFound         = errors.Register(ModuleName, 11, "claim not found")
	ErrClaimNotOwned         = errors.Register(ModuleName, 12, "claim does not belong to answerer")
	ErrClaimAlreadyLinked    = errors.Register(ModuleName, 13, "claim is already linked to an inquiry")
	ErrTooManyAnswers        = errors.Register(ModuleName, 14, "inquiry has reached max_answers_per_inquiry")
	ErrNotAsker              = errors.Register(ModuleName, 15, "caller is not the inquiry's asker")
	ErrAnswersInFlight       = errors.Register(ModuleName, 16, "cannot cancel inquiry with answers in flight")
	ErrInvalidAuthority      = errors.Register(ModuleName, 17, "invalid authority")
	ErrInvalidDomain         = errors.Register(ModuleName, 18, "invalid domain")
)
