package types

import (
	"fmt"
	"math"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"
)

// SettlementAmount parses a canonical uzrn amount. Settlement plans freeze the
// existing uint64 fee-pool arithmetic; they cannot introduce new issuance.
func SettlementAmount(value string) (uint64, error) {
	n, err := strconv.ParseUint(value, 10, 64)
	if err != nil || strconv.FormatUint(n, 10) != value {
		return 0, fmt.Errorf("noncanonical verifier settlement amount %q", value)
	}
	return n, nil
}

// ValidateVerifierRewardSettlement validates retained payment instructions,
// independently of current balances and current reward parameters. Nil means
// no recorded plan, including historical rounds whose payment is unknown.
func ValidateVerifierRewardSettlement(round *VerificationRound) error {
	if round == nil {
		return fmt.Errorf("nil verification round")
	}
	plan := round.VerifierRewardSettlement
	if plan == nil {
		return nil
	}
	if round.Id == "" || round.ClaimId == "" || round.Phase != VerificationPhase_VERIFICATION_PHASE_COMPLETE {
		return fmt.Errorf("verifier settlement requires an identified completed round")
	}
	if plan.CreatedAtBlock == 0 || plan.CreatedAtBlock != round.VerdictBlock ||
		(plan.PaidAtBlock != 0 && plan.PaidAtBlock < plan.CreatedAtBlock) {
		return fmt.Errorf("invalid verifier settlement heights")
	}
	if len(plan.Payments) == 0 || len(plan.Payments) > CommitSeatHardCap {
		return fmt.Errorf("invalid verifier settlement payment count")
	}
	withheldTotal, err := SettlementAmount(plan.WithheldTotal)
	if err != nil {
		return err
	}
	seen := make(map[string]bool, len(plan.Payments))
	var actualWithheld, total uint64
	for _, payment := range plan.Payments {
		if payment == nil || seen[payment.Verifier] {
			return fmt.Errorf("nil or duplicate verifier settlement payment")
		}
		addr, err := sdk.AccAddressFromBech32(payment.Verifier)
		if err != nil || addr.String() != payment.Verifier {
			return fmt.Errorf("invalid canonical verifier settlement recipient")
		}
		seen[payment.Verifier] = true
		amount, err := SettlementAmount(payment.Amount)
		if err != nil {
			return err
		}
		withheld, err := SettlementAmount(payment.Withheld)
		if err != nil {
			return err
		}
		if amount > math.MaxUint64-withheld || total > math.MaxUint64-amount-withheld || actualWithheld > math.MaxUint64-withheld {
			return fmt.Errorf("verifier settlement amount overflow")
		}
		total += amount + withheld
		actualWithheld += withheld
	}
	if total == 0 || actualWithheld != withheldTotal {
		return fmt.Errorf("verifier settlement does not conserve its frozen pool")
	}
	return nil
}

// ValidateVerifierRewardSettlementUpdate allows only the unpaid -> paid transition
// for an existing plan. Retries cannot recalculate shares or erase history.
func ValidateVerifierRewardSettlementUpdate(before, after *VerificationRound) error {
	if err := ValidateVerifierRewardSettlement(after); err != nil {
		return err
	}
	if before == nil || before.VerifierRewardSettlement == nil {
		return nil
	}
	if after.VerifierRewardSettlement == nil {
		return fmt.Errorf("cannot remove verifier settlement")
	}
	old := proto.Clone(before.VerifierRewardSettlement).(*VerifierRewardSettlement)
	next := proto.Clone(after.VerifierRewardSettlement).(*VerifierRewardSettlement)
	if old.PaidAtBlock != 0 && old.PaidAtBlock != next.PaidAtBlock {
		return fmt.Errorf("cannot change completed verifier settlement")
	}
	old.PaidAtBlock, next.PaidAtBlock = 0, 0
	if !proto.Equal(old, next) {
		return fmt.Errorf("cannot change frozen verifier settlement plan")
	}
	return nil
}
