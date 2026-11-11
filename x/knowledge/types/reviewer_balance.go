package types

import (
	"fmt"
	"strconv"
)

// BuiltInMinReviewerBalanceUzrn is the bank balance, in uzrn, that
// MsgSubmitCommitment has always required of a reviewer: 100 ZRN. With the
// reviewer-balance-param marker off it is the only figure. With the marker on
// it is the figure an empty Params.MinReviewerBalance stands for.
//
// It is a balance check at commit time, read through bank GetBalance: nothing
// is locked, bonded or escrowed, and the same coins can pass the check for one
// account and then move to another. The figure sets no vote weight, bond or
// reputation, and it is not Sybil resistance (docs/specs/
// knowledge-review-neutrality-v1.md: "this release does not claim Sybil
// resistance"). A lower figure lowers the cost of holding many reviewer seats.
const BuiltInMinReviewerBalanceUzrn uint64 = 100_000_000

// ParseMinReviewerBalance returns the reviewer minimum balance, in uzrn, that
// a Params.MinReviewerBalance value stands for. Empty means
// BuiltInMinReviewerBalanceUzrn. Anything else must be a canonical positive
// base-10 integer within uint64: no sign, no leading zero, no whitespace, and
// not zero.
func ParseMinReviewerBalance(value string) (uint64, error) {
	if value == "" {
		return BuiltInMinReviewerBalanceUzrn, nil
	}
	amount, err := strconv.ParseUint(value, 10, 64)
	if err != nil || amount == 0 || strconv.FormatUint(amount, 10) != value {
		return 0, fmt.Errorf("min_reviewer_balance must be empty or a canonical positive base-10 integer within uint64, got %q", value)
	}
	return amount, nil
}

// ValidateReviewerBalanceGenesis checks the reviewer minimum balance a genesis
// carries. The value must parse, and a non-empty value needs
// reviewer_balance_param_enabled: with the marker off, execution uses the
// built-in figure and never reads the field, so a genesis may not configure a
// control that would be silently ignored.
func ValidateReviewerBalanceGenesis(gs *GenesisState) error {
	if gs == nil || gs.Params == nil || gs.Params.MinReviewerBalance == "" {
		return nil
	}
	if _, err := ParseMinReviewerBalance(gs.Params.MinReviewerBalance); err != nil {
		return err
	}
	if !gs.ReviewerBalanceParamEnabled {
		return fmt.Errorf("min_reviewer_balance requires reviewer_balance_param_enabled")
	}
	return nil
}
