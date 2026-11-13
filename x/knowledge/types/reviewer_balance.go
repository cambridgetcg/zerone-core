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
// In a round without seat terms it is a balance check at commit time, read
// through bank GetBalance: nothing is locked, bonded or escrowed, and the same
// coins can pass the check for one account and then move to another. In a
// review-neutral round every valid reveal counts as one, so a holder of the
// figure can seat as many accounts as a verdict needs, for fees alone. So
// there the check never falls below 100 ZRN (rule A5 of the reviewer-entry
// design draft, docs/specs/knowledge-reviewer-entry-v1.md in pull request
// #190). In a round with seat terms (step B of that draft, behind the
// seat-deposit marker) the figure is instead an amount each seat locks until
// the round returns it, and only there may a lower figure apply (A3, D1). The
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

// ValidateMinReviewerBalance is ValidateMinReviewerBalanceFor with the
// seat-deposit marker off: a malformed value, or a figure below
// BuiltInMinReviewerBalanceUzrn, is refused. Params.Validate calls it.
func ValidateMinReviewerBalance(value string) error {
	return ValidateMinReviewerBalanceFor(value, false)
}

// ValidateMinReviewerBalanceFor refuses a Params.MinReviewerBalance value that
// is malformed (ParseMinReviewerBalance) and, unless seatDeposit is set, one
// that names a figure below BuiltInMinReviewerBalanceUzrn (rule A3).
// Params.ValidateWithSeatDeposit calls it, so it holds at genesis validation
// and when MsgUpdateParams executes, and ValidateReviewerBalanceGenesis calls
// it for InitGenesis; each passes the seat-deposit marker as it stands, or as
// the file being imported sets it.
//
// A lower figure is safe only for a seat that locks it. With the seat-deposit
// marker on, every new review-policy-1 round locks the figure as each seat's
// deposit, and every round without seat terms still compares a balance with
// the larger of the figure and 100 ZRN (EffectiveMinReviewerBalance, A5), so
// any canonical positive figure is accepted. Zero and malformed values are
// refused either way: nothing may read "empty" or "0" as no gate (A2).
//
// The refusal sentence below is kept word for word as 3a844c48 wrote it, so a
// chain with the marker off records the same failure reason (governance
// stores it in the proposal). Its last clause describes the marker-off case:
// a locked seat deposit exists in this source only behind the marker.
func ValidateMinReviewerBalanceFor(value string, seatDeposit bool) error {
	amount, err := ParseMinReviewerBalance(value)
	if err != nil {
		return err
	}
	if !seatDeposit && amount < BuiltInMinReviewerBalanceUzrn {
		return fmt.Errorf("min_reviewer_balance %s is below the built-in minimum of %d uzrn: a lower figure needs a locked seat deposit, which this source does not have", value, BuiltInMinReviewerBalanceUzrn)
	}
	return nil
}

// EffectiveMinReviewerBalance returns the figure, in uzrn, that
// MsgSubmitCommitment compares a reviewer's balance with while the
// reviewer-balance-param marker is on, in a round without seat terms: the
// larger of the stored value and BuiltInMinReviewerBalanceUzrn. There the
// balance is checked and not locked, so a stored figure below the built-in
// minimum, which the seat-deposit marker allows governance to set, can never
// lower the check (A5, D7). A round with seat terms locks its recorded deposit
// instead and never calls this. A malformed value is an error, never a figure.
func EffectiveMinReviewerBalance(value string) (uint64, error) {
	amount, err := ParseMinReviewerBalance(value)
	if err != nil {
		return 0, err
	}
	return max(amount, BuiltInMinReviewerBalanceUzrn), nil
}

// ValidateReviewerBalanceGenesis checks the reviewer minimum balance a genesis
// carries. The value must be well formed and, unless the file selects the
// seat-deposit marker, at or above the built-in minimum
// (ValidateMinReviewerBalanceFor), and a non-empty value needs
// reviewer_balance_param_enabled: with the marker off, execution uses the
// built-in figure and never reads the field, so a genesis may not configure a
// control that would be silently ignored.
func ValidateReviewerBalanceGenesis(gs *GenesisState) error {
	if gs == nil || gs.Params == nil || gs.Params.MinReviewerBalance == "" {
		return nil
	}
	if err := ValidateMinReviewerBalanceFor(gs.Params.MinReviewerBalance, gs.SeatDepositEnabled); err != nil {
		return err
	}
	if !gs.ReviewerBalanceParamEnabled {
		return fmt.Errorf("min_reviewer_balance requires reviewer_balance_param_enabled")
	}
	return nil
}
