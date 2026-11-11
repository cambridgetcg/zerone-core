package types

import (
	"fmt"
	"strconv"
)

// BuiltInMinReviewerBalanceUzrn is the bank balance, in uzrn, that
// MsgSubmitCommitment has always required of a reviewer: 100 ZRN. With the
// reviewer-balance-param marker off it is the only figure. With the marker on
// it is the figure an empty Params.MinReviewerBalance stands for, and the floor
// of every figure: no parameter can set or apply less.
//
// It is a balance check at commit time, read through bank GetBalance: nothing
// is locked, bonded or escrowed, and the same coins can pass the check for one
// account and then move to another. In a review-neutral round every valid
// reveal counts as one, so a holder of the figure can seat as many accounts as
// a verdict needs, for fees alone. So the figure is a parameter that can be
// raised, and lowering it below 100 ZRN waits for a locked seat deposit (the
// reviewer-entry design draft, docs/specs/knowledge-reviewer-entry-v1.md in
// pull request #190, rules A3 and A5), which this source does not have. The
// figure sets no vote weight, bond or reputation, and it is not Sybil
// resistance (docs/specs/knowledge-review-neutrality-v1.md: "this release does
// not claim Sybil resistance").
const BuiltInMinReviewerBalanceUzrn uint64 = 100_000_000

// ParseMinReviewerBalance returns the figure, in uzrn, that a
// Params.MinReviewerBalance value names. Empty means
// BuiltInMinReviewerBalanceUzrn. Anything else must be a canonical positive
// base-10 integer within uint64: no sign, no leading zero, no whitespace, and
// not zero. It checks the form only: ValidateMinReviewerBalance adds the
// floor, and EffectiveMinReviewerBalance is what a commitment is compared with.
//
// There is no upper bound below the uint64 limit. A figure above every
// account's balance closes MsgSubmitCommitment to everyone; the vote-extension
// commit path (app/abci.go) never reads it. That is within governance's
// existing power over knowledge params, and nothing here prevents it.
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

// ValidateMinReviewerBalance refuses a Params.MinReviewerBalance value that is
// malformed (ParseMinReviewerBalance) or names a figure below
// BuiltInMinReviewerBalanceUzrn. Params.Validate calls it, so it holds at
// genesis validation and when MsgUpdateParams executes, and
// ValidateReviewerBalanceGenesis calls it for InitGenesis. A figure at or above
// the built-in minimum passes.
//
// A lower figure is safe only for a seat that locks it, and this source locks
// nothing, so the floor always applies here. A source that adds a locked seat
// deposit decides where a lower figure may apply; until then nothing may store
// one.
func ValidateMinReviewerBalance(value string) error {
	amount, err := ParseMinReviewerBalance(value)
	if err != nil {
		return err
	}
	if amount < BuiltInMinReviewerBalanceUzrn {
		return fmt.Errorf("min_reviewer_balance %s is below the built-in minimum of %d uzrn: a lower figure needs a locked seat deposit, which this source does not have", value, BuiltInMinReviewerBalanceUzrn)
	}
	return nil
}

// EffectiveMinReviewerBalance returns the figure, in uzrn, that
// MsgSubmitCommitment compares a reviewer's balance with while the marker is
// on: the larger of the stored value and BuiltInMinReviewerBalanceUzrn. The
// balance is checked and not locked, so a stored figure below the built-in
// minimum can never lower the check, even if one reached the store without
// validation (for example state written by an older draft of this branch). A
// malformed value is an error, never a figure.
func EffectiveMinReviewerBalance(value string) (uint64, error) {
	amount, err := ParseMinReviewerBalance(value)
	if err != nil {
		return 0, err
	}
	return max(amount, BuiltInMinReviewerBalanceUzrn), nil
}

// ValidateReviewerBalanceGenesis checks the reviewer minimum balance a genesis
// carries. The value must be well formed and at or above the built-in minimum
// (ValidateMinReviewerBalance), and a non-empty value needs
// reviewer_balance_param_enabled: with the marker off, execution uses the
// built-in figure and never reads the field, so a genesis may not configure a
// control that would be silently ignored.
func ValidateReviewerBalanceGenesis(gs *GenesisState) error {
	if gs == nil || gs.Params == nil || gs.Params.MinReviewerBalance == "" {
		return nil
	}
	if err := ValidateMinReviewerBalance(gs.Params.MinReviewerBalance); err != nil {
		return err
	}
	if !gs.ReviewerBalanceParamEnabled {
		return fmt.Errorf("min_reviewer_balance requires reviewer_balance_param_enabled")
	}
	return nil
}
