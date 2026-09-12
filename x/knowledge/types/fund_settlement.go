package types

import (
	"fmt"
	"strconv"

	"google.golang.org/protobuf/proto"
)

const ClaimFundingPolicyV1 uint32 = 1

// ClaimReviewBudget preserves floor(paid * 55 / 100) without multiplication
// overflow. The remaining integer dust belongs to the explicit remainder.
func ClaimReviewBudget(paid uint64) uint64 { return paid/100*55 + paid%100*55/100 }

// ValidateClaimFundingTerms checks prospective instructions, not historical
// custody or current solvency. Nil terms preserve unknown predecessor funding.
func ValidateClaimFundingTerms(claim *Claim, enabled bool) error {
	if claim == nil {
		return fmt.Errorf("nil funded claim")
	}
	terms := claim.FundingTerms
	if terms == nil {
		return nil
	}
	if !enabled || terms.PolicyVersion != ClaimFundingPolicyV1 || claim.ReviewPolicyVersion != ReviewPolicyNeutral {
		return fmt.Errorf("claim funding requires activated policy 1 neutral review")
	}
	if len(terms.ProtoReflect().GetUnknown()) != 0 {
		return fmt.Errorf("unknown claim funding fields")
	}
	if err := ValidateRecordText("funded claim ID", claim.Id, MaxCommitmentContextBytes, true); err != nil {
		return err
	}
	if _, err := CanonicalReviewAddress(claim.Submitter); err != nil {
		return fmt.Errorf("invalid funded claim payer: %w", err)
	}
	paid, err := SettlementAmount(terms.PaidAmount)
	if err != nil || paid == 0 || claim.Stake != terms.PaidAmount {
		return fmt.Errorf("claim funding paid amount must be positive canonical stake")
	}
	review, err := SettlementAmount(terms.ReviewBudget)
	if err != nil || review != ClaimReviewBudget(paid) {
		return fmt.Errorf("claim funding review budget must be the fixed 55 percent allocation")
	}
	refundable, err := SettlementAmount(terms.RefundableAmount)
	if err != nil {
		return err
	}
	retained, err := SettlementAmount(terms.RetainedFee)
	if err != nil {
		return err
	}
	switch terms.Kind {
	case ClaimFundingKind_CLAIM_FUNDING_KIND_REVIEW_FEE:
		if refundable != 0 || retained != paid-review {
			return fmt.Errorf("ordinary funding must retain the non-review fee without a bond")
		}
	case ClaimFundingKind_CLAIM_FUNDING_KIND_CHALLENGE_DEPOSIT:
		if retained != 0 || refundable != paid-review {
			return fmt.Errorf("challenge funding must preserve the full non-review bond")
		}
	default:
		return fmt.Errorf("unsupported claim funding kind %d", terms.Kind)
	}
	return nil
}

// ValidateClaimFundingUpdate forbids repricing, payer changes and a second
// selected round. Historical absence cannot be rewritten into new funding.
func ValidateClaimFundingUpdate(before, after *Claim) error {
	if err := ValidateClaimFundingTerms(after, true); err != nil {
		return err
	}
	if before == nil {
		return nil
	}
	if !proto.Equal(before.FundingTerms, after.FundingTerms) {
		return fmt.Errorf("cannot change frozen claim funding terms")
	}
	if before.FundingTerms != nil && (before.Id != after.Id || before.Submitter != after.Submitter || before.Stake != after.Stake ||
		(before.VerificationRoundId != "" && before.VerificationRoundId != after.VerificationRoundId)) {
		return fmt.Errorf("cannot change funded claim payer, stake or selected round")
	}
	return nil
}

func fundedTerminalVerdict(verdict Verdict) bool {
	return verdict == Verdict_VERDICT_ACCEPT || verdict == Verdict_VERDICT_REJECT ||
		verdict == Verdict_VERDICT_MALFORMED || verdict == Verdict_VERDICT_INCONCLUSIVE
}

// ValidateClaimRefundSettlement validates an explicit positive instruction.
// Its economic entitlement additionally requires ValidateClaimFundingRound.
func ValidateClaimRefundSettlement(round *VerificationRound) error {
	if round == nil {
		return fmt.Errorf("nil refund round")
	}
	plan := round.ClaimRefundSettlement
	if plan == nil {
		return nil
	}
	if len(plan.ProtoReflect().GetUnknown()) != 0 {
		return fmt.Errorf("unknown claim refund fields")
	}
	if round.Id == "" || round.ClaimId == "" || round.Phase != VerificationPhase_VERIFICATION_PHASE_COMPLETE || !fundedTerminalVerdict(round.Verdict) {
		return fmt.Errorf("claim refund requires an identified completed verdict")
	}
	if plan.CreatedAtBlock == 0 || plan.CreatedAtBlock != round.VerdictBlock ||
		(plan.PaidAtBlock != 0 && plan.PaidAtBlock < plan.CreatedAtBlock) {
		return fmt.Errorf("invalid claim refund heights")
	}
	if _, err := CanonicalReviewAddress(plan.Recipient); err != nil {
		return fmt.Errorf("invalid claim refund recipient: %w", err)
	}
	amount, err := SettlementAmount(plan.Amount)
	if err != nil || amount == 0 {
		return fmt.Errorf("claim refund must be a positive canonical amount")
	}
	return nil
}

func ValidateClaimRefundSettlementUpdate(before, after *VerificationRound) error {
	if err := ValidateClaimRefundSettlement(after); err != nil {
		return err
	}
	if before == nil || before.ClaimRefundSettlement == nil {
		return nil
	}
	if after.ClaimRefundSettlement == nil {
		return fmt.Errorf("cannot remove frozen claim refund")
	}
	old := proto.Clone(before.ClaimRefundSettlement).(*ClaimRefundSettlement)
	next := proto.Clone(after.ClaimRefundSettlement).(*ClaimRefundSettlement)
	if old.PaidAtBlock != 0 && old.PaidAtBlock != next.PaidAtBlock {
		return fmt.Errorf("cannot change completed claim refund")
	}
	old.PaidAtBlock, next.PaidAtBlock = 0, 0
	if !proto.Equal(old, next) {
		return fmt.Errorf("cannot change frozen claim refund plan")
	}
	return nil
}

// ValidateClaimFundingRound joins the payer's frozen terms to the actual
// authenticated reviews. A completed funded round must account for every
// available unit, including an unused review budget, without penalizing dissent.
// The selected pointer may be empty during initial runtime round creation;
// genesis additionally requires that pointer and exactly one retained round.
func ValidateClaimFundingRound(claim *Claim, round *VerificationRound) error {
	if round == nil {
		return fmt.Errorf("nil funded verification round")
	}
	if err := ValidateClaimRefundSettlement(round); err != nil {
		return err
	}
	if claim == nil || claim.FundingTerms == nil {
		if round.ClaimRefundSettlement != nil {
			return fmt.Errorf("claim refund has no funded parent")
		}
		return nil
	}
	if err := ValidateClaimFundingTerms(claim, true); err != nil {
		return err
	}
	if round.ClaimId != claim.Id || (claim.VerificationRoundId != "" && claim.VerificationRoundId != round.Id) || round.ReviewPolicyVersion != ReviewPolicyNeutral || round.CommitmentScheme != CommitmentSchemeReviewV2 {
		return fmt.Errorf("funded claim and selected neutral round must match")
	}
	if err := ValidateVerificationRoundRecord(round, true); err != nil {
		return err
	}
	if round.Phase != VerificationPhase_VERIFICATION_PHASE_COMPLETE {
		if round.Phase == VerificationPhase_VERIFICATION_PHASE_EXPIRED || round.VerifierRewardSettlement != nil || round.ClaimRefundSettlement != nil {
			return fmt.Errorf("funded round must finalize its obligations through a completed verdict")
		}
		return nil
	}
	if !fundedTerminalVerdict(round.Verdict) || round.VerdictBlock == 0 {
		return fmt.Errorf("funded round requires a terminal verdict and height")
	}
	review, _ := SettlementAmount(claim.FundingTerms.ReviewBudget)
	refund, _ := SettlementAmount(claim.FundingTerms.RefundableAmount)
	n := uint64(len(round.Reveals))
	plan := round.VerifierRewardSettlement
	if n == 0 || review == 0 {
		if plan != nil {
			return fmt.Errorf("funded round has no payable review work")
		}
		if n == 0 {
			refund += review // components were checked against one uint64 paid amount
		}
	} else {
		if plan == nil || uint64(len(plan.Payments)) != n || plan.WithheldTotal != "0" {
			return fmt.Errorf("funded round must preserve every eligible review payment")
		}
		for i, payment := range plan.Payments {
			amount := review / n
			if i == 0 {
				amount += review % n
			}
			if payment.Verifier != round.Reveals[i].Verifier || payment.Amount != strconv.FormatUint(amount, 10) || payment.Withheld != "0" || len(payment.ProtoReflect().GetUnknown()) != 0 {
				return fmt.Errorf("funded review payment differs from fixed allocation")
			}
		}
		if len(plan.ProtoReflect().GetUnknown()) != 0 {
			return fmt.Errorf("unknown funded reviewer settlement fields")
		}
	}
	rp := round.ClaimRefundSettlement
	if refund == 0 {
		if rp != nil {
			return fmt.Errorf("funded round has no refundable amount")
		}
	} else if rp == nil || rp.Recipient != claim.Submitter || rp.Amount != strconv.FormatUint(refund, 10) {
		return fmt.Errorf("funded round must preserve the exact payer refund")
	}
	if plan != nil && rp != nil && plan.PaidAtBlock != rp.PaidAtBlock {
		return fmt.Errorf("review payments and refund must settle atomically")
	}
	return nil
}
