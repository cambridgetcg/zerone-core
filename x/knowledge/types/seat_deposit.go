package types

import (
	"fmt"
	"strconv"

	"cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	"google.golang.org/protobuf/proto"
)

// Reviewer seat deposit (the reviewer-entry design draft,
// docs/specs/knowledge-reviewer-entry-v1.md in pull request #190, step B,
// rules D1 to D7). A round created while the seat-deposit marker is on, under
// review policy 1 with commitment scheme 2, records seat terms: the counted
// seat deposit, copied from min_reviewer_balance at creation. Each commitment
// in such a round moves that deposit from its signer into the
// SeatDepositModuleName account and records it with the seat. Every seat's
// deposit is returned in full to the address that locked it, whatever the
// verdict, the seat's vote or whether it revealed: nothing is forfeited. The
// returns are frozen once, at completion or from the aggregation deadline,
// and the pending payment queue pays them.

// SeatDepositModuleName is the module account that holds seat deposits and
// nothing else. It has no permissions, and it is on the app's blocked list
// (every maccPerms entry but gov is), so nothing else can pay into it.
const SeatDepositModuleName = "knowledge_seat_deposits"

var (
	// ErrSeatDepositRequired refuses a seat in a round with seat terms that
	// would be stored without its deposit being locked.
	ErrSeatDepositRequired = errors.Register(ModuleName, 83, "seat deposit required")
	// ErrReviewerNotAdmitted refuses a commitment, in a round with seat terms,
	// from a signer that zerone_auth does not admit to review.
	ErrReviewerNotAdmitted = errors.Register(ModuleName, 84, "reviewer not admitted")
)

// RoundCarriesSeatDepositFields reports whether any seat-deposit field is
// present: terms, a returns plan, or a deposit on any seat.
func RoundCarriesSeatDepositFields(round *VerificationRound) bool {
	if round == nil {
		return false
	}
	if round.SeatTerms != nil || round.SeatDepositReturns != nil {
		return true
	}
	for _, commit := range round.Commits {
		if commit != nil && commit.SeatDeposit != "" {
			return true
		}
	}
	return false
}

// RoundSeatTermsAllowed reports whether a round's admission terms admit seat
// terms: review policy 1 with commitment scheme 2 (D1). Step C would widen
// this to its own policy.
func RoundSeatTermsAllowed(round *VerificationRound) bool {
	return round != nil && round.ReviewPolicyVersion == ReviewPolicyNeutral && round.CommitmentScheme == CommitmentSchemeReviewV2
}

// SeatDepositAmount parses a canonical positive uzrn decimal.
func SeatDepositAmount(value string) (uint64, error) {
	amount, err := SettlementAmount(value)
	if err != nil {
		return 0, fmt.Errorf("seat deposit must be a canonical uzrn decimal: %w", err)
	}
	if amount == 0 {
		return 0, fmt.Errorf("seat deposit must be positive")
	}
	return amount, nil
}

// ValidateRoundSeatDeposits checks a round's seat terms, its seats' deposits
// and its returns plan, independently of the marker, which a pure function
// cannot read. A round without terms carries no deposit and no plan. A round
// with terms is a policy-1, scheme-2 round that never expires; every seat
// carries exactly the terms' deposit; and a plan, when present, holds one
// return per seat in commit order, to the seat's own address, for the seat's
// own deposit. Completion validates the round in memory before it freezes
// the plan, so the rule that a completed round holds its plan is
// ValidateStoredSeatDeposits', which applies to what is stored or imported.
func ValidateRoundSeatDeposits(round *VerificationRound) error {
	if round == nil {
		return fmt.Errorf("nil verification round")
	}
	terms := round.SeatTerms
	if terms == nil {
		if round.SeatDepositReturns != nil {
			return fmt.Errorf("seat deposit returns require seat terms")
		}
		for _, commit := range round.Commits {
			if commit != nil && commit.SeatDeposit != "" {
				return fmt.Errorf("a seat deposit requires seat terms")
			}
		}
		return nil
	}
	if len(terms.ProtoReflect().GetUnknown()) != 0 {
		return fmt.Errorf("unsupported seat terms fields")
	}
	if !RoundSeatTermsAllowed(round) {
		return fmt.Errorf("seat terms require review policy 1 and commitment scheme 2")
	}
	if round.Phase == VerificationPhase_VERIFICATION_PHASE_EXPIRED {
		// Policy-1 rounds never expire; an expired round would strand its
		// deposits outside every return path.
		return fmt.Errorf("a round with seat terms cannot be expired")
	}
	if _, err := SeatDepositAmount(terms.CountedSeatDeposit); err != nil {
		return fmt.Errorf("seat terms: %w", err)
	}
	for _, commit := range round.Commits {
		if commit == nil || commit.SeatDeposit != terms.CountedSeatDeposit {
			return fmt.Errorf("every seat in a round with seat terms must hold the round's deposit")
		}
	}
	plan := round.SeatDepositReturns
	if plan == nil {
		return nil
	}
	if len(plan.ProtoReflect().GetUnknown()) != 0 {
		return fmt.Errorf("unsupported seat deposit returns fields")
	}
	if len(plan.Returns) == 0 || len(plan.Returns) != len(round.Commits) {
		return fmt.Errorf("seat deposit returns must hold one return per seat")
	}
	if plan.CreatedAtBlock == 0 {
		return fmt.Errorf("seat deposit returns require a creation height")
	}
	atDeadline := plan.CreatedAtBlock >= round.AggregationDeadline
	if round.Phase == VerificationPhase_VERIFICATION_PHASE_COMPLETE {
		if plan.CreatedAtBlock != round.VerdictBlock && !(atDeadline && plan.CreatedAtBlock < round.VerdictBlock) {
			return fmt.Errorf("seat deposit returns are frozen at completion or from the aggregation deadline")
		}
	} else if !atDeadline {
		return fmt.Errorf("an open round holds seat deposit returns only from its aggregation deadline")
	}
	for i, ret := range plan.Returns {
		commit := round.Commits[i]
		if ret == nil || len(ret.ProtoReflect().GetUnknown()) != 0 {
			return fmt.Errorf("invalid seat deposit return")
		}
		if ret.Recipient != commit.Verifier || ret.Amount != commit.SeatDeposit {
			return fmt.Errorf("a seat deposit return must repay its own seat in commit order")
		}
		if ret.PaidAtBlock != 0 && ret.PaidAtBlock < plan.CreatedAtBlock {
			return fmt.Errorf("invalid seat deposit return payment height")
		}
	}
	return nil
}

// ValidateStoredSeatDeposits is ValidateRoundSeatDeposits for a record that is
// stored or imported: in addition, a completed round with seats holds its
// returns plan, so no completed round keeps a deposit outside every return.
func ValidateStoredSeatDeposits(round *VerificationRound) error {
	if err := ValidateRoundSeatDeposits(round); err != nil {
		return err
	}
	if round.SeatTerms != nil && round.SeatDepositReturns == nil && round.Phase == VerificationPhase_VERIFICATION_PHASE_COMPLETE && len(round.Commits) > 0 {
		return fmt.Errorf("a completed round with seats must hold its seat deposit returns")
	}
	return nil
}

// ValidateSeatDepositUpdate allows only what a round's life does to its seat
// deposits: terms are fixed when the round is first stored and never appear,
// change or disappear afterwards; a returns plan, once frozen, keeps its
// height, recipients and amounts, and each return moves only from unpaid to
// paid, never back and never to another height. Seats themselves are
// append-only under the scheme-2 rule in SetVerificationRound.
func ValidateSeatDepositUpdate(before, after *VerificationRound) error {
	if err := ValidateStoredSeatDeposits(after); err != nil {
		return err
	}
	if before == nil {
		return nil
	}
	if !proto.Equal(before.SeatTerms, after.SeatTerms) {
		return fmt.Errorf("seat terms are immutable")
	}
	old := before.SeatDepositReturns
	if old == nil {
		return nil
	}
	next := after.SeatDepositReturns
	if next == nil {
		return fmt.Errorf("cannot remove seat deposit returns")
	}
	if old.CreatedAtBlock != next.CreatedAtBlock || len(old.Returns) != len(next.Returns) {
		return fmt.Errorf("cannot change frozen seat deposit returns")
	}
	for i, ret := range old.Returns {
		updated := next.Returns[i]
		if ret.Recipient != updated.Recipient || ret.Amount != updated.Amount {
			return fmt.Errorf("cannot change frozen seat deposit returns")
		}
		if ret.PaidAtBlock != 0 && ret.PaidAtBlock != updated.PaidAtBlock {
			return fmt.Errorf("cannot change a paid seat deposit return")
		}
	}
	return nil
}

// BuildSeatDepositReturns freezes one return per seat, in commit order, at
// height. It returns nil for a round without terms or without seats.
func BuildSeatDepositReturns(round *VerificationRound, height uint64) *SeatDepositReturnSettlement {
	if round == nil || round.SeatTerms == nil || len(round.Commits) == 0 {
		return nil
	}
	plan := &SeatDepositReturnSettlement{CreatedAtBlock: height}
	for _, commit := range round.Commits {
		plan.Returns = append(plan.Returns, &SeatDepositReturn{Recipient: commit.Verifier, Amount: commit.SeatDeposit})
	}
	return plan
}

// SeatDepositReturnTotal sums a plan's amounts, in uzrn.
func SeatDepositReturnTotal(plan *SeatDepositReturnSettlement) (sdkmath.Int, error) {
	total := sdkmath.ZeroInt()
	if plan == nil {
		return total, nil
	}
	for _, ret := range plan.Returns {
		amount, err := SeatDepositAmount(ret.GetAmount())
		if err != nil {
			return sdkmath.Int{}, err
		}
		total = total.Add(sdkmath.NewIntFromUint64(amount))
	}
	return total, nil
}

// SeatDepositsHeld is what the seat-deposit account owes for one round: every
// seat's deposit before the returns are frozen, and the unpaid returns after.
// It is zero for a round without terms. Sums use sdkmath.Int, since 512 seats
// of a uint64 figure overflow uint64.
func SeatDepositsHeld(round *VerificationRound) (sdkmath.Int, error) {
	held := sdkmath.ZeroInt()
	if round == nil || round.SeatTerms == nil {
		return held, nil
	}
	if plan := round.SeatDepositReturns; plan != nil {
		for _, ret := range plan.Returns {
			if ret.GetPaidAtBlock() != 0 {
				continue
			}
			amount, err := SeatDepositAmount(ret.GetAmount())
			if err != nil {
				return sdkmath.Int{}, err
			}
			held = held.Add(sdkmath.NewIntFromUint64(amount))
		}
		return held, nil
	}
	for _, commit := range round.Commits {
		amount, err := SeatDepositAmount(commit.GetSeatDeposit())
		if err != nil {
			return sdkmath.Int{}, err
		}
		held = held.Add(sdkmath.NewIntFromUint64(amount))
	}
	return held, nil
}

// PendingSeatDepositReturns counts a round's frozen, unpaid returns.
func PendingSeatDepositReturns(round *VerificationRound) int {
	if round == nil || round.SeatDepositReturns == nil {
		return 0
	}
	pending := 0
	for _, ret := range round.SeatDepositReturns.Returns {
		if ret != nil && ret.PaidAtBlock == 0 {
			pending++
		}
	}
	return pending
}

// ValidateSeatDepositGenesis checks the seat-deposit state a genesis carries,
// before any write. With the marker off, no round carries terms, a plan or a
// seat deposit: a file written for a chain that has none imports exactly as
// before, and nothing configures a rule that would be silently ignored. With
// the marker on, the marker's predecessors must be selected too (the design's
// section 7: the reviewer-balance parameter, review neutrality and record
// integrity), and every round passes ValidateRoundSeatDeposits. The custody
// balance is checked against the bank by the keeper, which can read it.
func ValidateSeatDepositGenesis(gs *GenesisState) error {
	if gs == nil {
		return nil
	}
	if err := ValidateSeatDepositPresence(gs); err != nil {
		return err
	}
	if !gs.SeatDepositEnabled {
		return nil
	}
	if !gs.ReviewerBalanceParamEnabled || !gs.ReviewNeutralityEnabled || !gs.RecordIntegrityEnabled {
		return fmt.Errorf("seat_deposit_enabled requires reviewer_balance_param_enabled, review_neutrality_enabled and record_integrity_enabled")
	}
	return nil
}

// ValidateSeatDepositPresence is the structural part of
// ValidateSeatDepositGenesis: with the marker off no round carries a
// seat-deposit field, and every round's seat deposits are consistent. It does
// not apply the predecessor rule, so an export can check it whatever other
// markers a test or chain selects.
func ValidateSeatDepositPresence(gs *GenesisState) error {
	if gs == nil {
		return nil
	}
	for _, group := range [][]*VerificationRound{gs.ActiveRounds, gs.CompletedRounds} {
		for _, round := range group {
			if round == nil {
				continue
			}
			if !gs.SeatDepositEnabled && RoundCarriesSeatDepositFields(round) {
				return fmt.Errorf("round %s carries seat-deposit fields without seat_deposit_enabled", round.Id)
			}
			if err := ValidateStoredSeatDeposits(round); err != nil {
				return fmt.Errorf("round %s: %w", round.Id, err)
			}
		}
	}
	return nil
}

// SeatDepositsHeldInGenesis sums SeatDepositsHeld over every round a genesis
// carries.
func SeatDepositsHeldInGenesis(gs *GenesisState) (sdkmath.Int, error) {
	held := sdkmath.ZeroInt()
	if gs == nil {
		return held, nil
	}
	for _, group := range [][]*VerificationRound{gs.ActiveRounds, gs.CompletedRounds} {
		for _, round := range group {
			amount, err := SeatDepositsHeld(round)
			if err != nil {
				return sdkmath.Int{}, err
			}
			held = held.Add(amount)
		}
	}
	return held, nil
}

// FormatSeatDeposit is the canonical decimal of a deposit in uzrn.
func FormatSeatDeposit(amount uint64) string { return strconv.FormatUint(amount, 10) }
