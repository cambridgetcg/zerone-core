package keeper

import (
	"bytes"
	"context"
	"fmt"
	"strconv"

	"github.com/zerone-chain/zerone/x/knowledge/types"
	"google.golang.org/protobuf/encoding/protowire"
)

// FundSettlementEnabledStoreKey selects prospective financial terms. Historical
// claims without those terms never acquire an inferred refund entitlement.
const FundSettlementEnabledStoreKey = "\x86"

func (k Keeper) FundSettlementEnabled(ctx context.Context) (bool, error) {
	bz, err := k.storeService.OpenKVStore(ctx).Get([]byte(FundSettlementEnabledStoreKey))
	if err != nil {
		return false, fmt.Errorf("read fund settlement marker: %w", err)
	}
	if bz == nil {
		return false, nil
	}
	if !bytes.Equal(bz, []byte{1}) {
		return false, fmt.Errorf("malformed fund settlement marker")
	}
	return true, nil
}

func (k Keeper) EnableFundSettlement(ctx context.Context) error {
	enabled, err := k.FundSettlementEnabled(ctx)
	if err != nil || enabled {
		return err
	}
	claims, err := k.ClaimRecordsEnabled(ctx)
	if err != nil || !claims {
		return fmt.Errorf("fund settlement requires claim records: %v", err)
	}
	neutral, err := k.ReviewNeutralityEnabled(ctx)
	if err != nil || !neutral {
		return fmt.Errorf("fund settlement requires review neutrality: %v", err)
	}
	records, err := k.RecordIntegrityEnabled(ctx)
	if err != nil || !records {
		return fmt.Errorf("fund settlement requires record integrity: %v", err)
	}
	return k.storeService.OpenKVStore(ctx).Set([]byte(FundSettlementEnabledStoreKey), []byte{1})
}

// ValidateFundSettlementActivation inventories the exact predecessor without
// changing records, retry scheduling, bank balances or historical terms.
func (k Keeper) ValidateFundSettlementActivation(ctx context.Context) error {
	enabled, err := k.FundSettlementEnabled(ctx)
	if err != nil || enabled {
		return fmt.Errorf("fund settlement requires unenabled predecessor: %v", err)
	}
	claimsEnabled, err := k.ClaimRecordsEnabled(ctx)
	if err != nil || !claimsEnabled {
		return fmt.Errorf("fund settlement requires claim-record predecessor: %v", err)
	}
	for _, spec := range []struct {
		prefix []byte
		field  protowire.Number
	}{{types.ClaimKeyPrefix, types.ClaimFundingTermsField}, {types.VerificationRoundKeyPrefix, types.RoundClaimRefundField}} {
		if err := k.requireAbsentPredecessorFundingField(ctx, spec.prefix, spec.field); err != nil {
			return err
		}
	}
	claims, err := k.GetAllClaimsChecked(ctx)
	if err != nil {
		return err
	}
	rounds, err := k.GetAllVerificationRoundsChecked(ctx)
	if err != nil {
		return err
	}
	gs := &types.GenesisState{PendingClaims: claims, RecordIntegrityEnabled: true, ReviewNeutralityEnabled: true, ClaimRecordsEnabled: true}
	for _, round := range rounds {
		if round.Phase == types.VerificationPhase_VERIFICATION_PHASE_COMPLETE || round.Phase == types.VerificationPhase_VERIFICATION_PHASE_EXPIRED {
			gs.CompletedRounds = append(gs.CompletedRounds, round)
		} else {
			gs.ActiveRounds = append(gs.ActiveRounds, round)
		}
	}
	if err := types.ValidateGenesisRounds(gs); err != nil {
		return err
	}
	if err := k.validatePendingVerifierRewardIndex(ctx, rounds); err != nil {
		return err
	}
	return k.ValidateKnowledgeHistoryState(ctx)
}

// setClaimFundingTerms is used only at admission, inside the message's bank and
// state cache. Kind records the actual payment route rather than inferring it
// from a scientific relation or from ProvisionalFactId.
func (k Keeper) setClaimFundingTerms(ctx context.Context, claim *types.Claim, kind types.ClaimFundingKind) error {
	enabled, err := k.FundSettlementEnabled(ctx)
	if err != nil || !enabled {
		return err
	}
	if k.bankKeeper == nil {
		return fmt.Errorf("funded claim requires bank keeper")
	}
	paid, err := types.SettlementAmount(claim.Stake)
	if err != nil || paid == 0 {
		return fmt.Errorf("funded claim requires a canonical positive payment: %v", err)
	}
	pool := verifierPoolFromFee(paid)
	terms := &types.ClaimFundingTerms{
		PolicyVersion: 1, Kind: kind, PaidAmount: claim.Stake,
		ReviewBudget: strconv.FormatUint(pool, 10), RefundableAmount: "0", RetainedFee: "0",
	}
	switch kind {
	case types.ClaimFundingKind_CLAIM_FUNDING_KIND_REVIEW_FEE:
		terms.RetainedFee = strconv.FormatUint(paid-pool, 10)
	case types.ClaimFundingKind_CLAIM_FUNDING_KIND_CHALLENGE_DEPOSIT:
		terms.RefundableAmount = strconv.FormatUint(paid-pool, 10)
	default:
		return fmt.Errorf("unsupported claim funding kind")
	}
	claim.FundingTerms = terms
	return types.ValidateClaimFundingTerms(claim, true)
}

func (k Keeper) validateClaimFunding(ctx context.Context, claim *types.Claim) error {
	enabled, err := k.FundSettlementEnabled(ctx)
	if err != nil {
		return err
	}
	return types.ValidateClaimFundingTerms(claim, enabled)
}

// buildClaimRefundSettlement freezes the unused review budget and refundable
// deposit. Its amount is independent of the panel's scientific verdict.
func (k Keeper) buildClaimRefundSettlement(ctx context.Context, claim *types.Claim, round *types.VerificationRound, result *VerificationResult) error {
	if err := k.validateClaimFunding(ctx, claim); err != nil {
		return err
	}
	if claim.FundingTerms == nil {
		return nil
	}
	refund, _ := types.SettlementAmount(claim.FundingTerms.RefundableAmount)
	if len(result.Rewards) == 0 {
		pool, _ := types.SettlementAmount(claim.FundingTerms.ReviewBudget)
		refund += pool // Funding-term validation proves this cannot overflow.
	}
	if refund > 0 {
		round.ClaimRefundSettlement = &types.ClaimRefundSettlement{
			Recipient: claim.Submitter, Amount: strconv.FormatUint(refund, 10), CreatedAtBlock: round.VerdictBlock,
		}
	}
	return types.ValidateClaimFundingRound(claim, round)
}

func roundHasPendingFunds(round *types.VerificationRound) bool {
	return (round.VerifierRewardSettlement != nil && round.VerifierRewardSettlement.PaidAtBlock == 0) ||
		(round.ClaimRefundSettlement != nil && round.ClaimRefundSettlement.PaidAtBlock == 0)
}

func pendingFundTransferCount(round *types.VerificationRound) int {
	legs := 0
	if plan := round.VerifierRewardSettlement; plan != nil && plan.PaidAtBlock == 0 {
		legs = len(plan.Payments)
		if plan.WithheldTotal != "0" {
			legs++
		}
	}
	if plan := round.ClaimRefundSettlement; plan != nil && plan.PaidAtBlock == 0 {
		legs++
	}
	return legs
}

// Inspect newly introduced singular messages before protobuf decoding can merge
// duplicates or discard invalid wire encodings.
func validateRawClaimRecord(data []byte) error {
	if err := types.ValidateRawPolicyField(data, types.ClaimReviewPolicyField); err != nil {
		return err
	}
	return types.ValidateRawClaimFunding(data)
}

func validateRawRoundRecord(data []byte) error {
	if err := types.ValidateRawPolicyField(data, types.RoundReviewPolicyField); err != nil {
		return err
	}
	return types.ValidateRawRoundRefund(data)
}

func (k Keeper) validateRoundFunding(ctx context.Context, round *types.VerificationRound) error {
	enabled, err := k.FundSettlementEnabled(ctx)
	if err != nil {
		return err
	}
	if !enabled {
		if round.ClaimRefundSettlement != nil {
			return fmt.Errorf("claim refund requires fund settlement activation")
		}
		return nil
	}
	claim, err := k.getClaimRecordChecked(ctx, round.ClaimId)
	if err != nil {
		return err
	}
	if claim == nil {
		if round.ClaimRefundSettlement != nil {
			return fmt.Errorf("claim refund requires its funded parent")
		}
		return nil // Preserve a frozen historical reviewer plan without inferring a refund.
	}
	if err := types.ValidateClaimFundingTerms(claim, true); err != nil {
		return err
	}
	return types.ValidateClaimFundingRound(claim, round)
}

func (k Keeper) requireAbsentPredecessorFundingField(ctx context.Context, prefix []byte, field protowire.Number) error {
	it, err := k.storeService.OpenKVStore(ctx).Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return err
	}
	count, size := 0, 0
	for ; it.Valid(); it.Next() {
		count++
		size += len(it.Key()) + len(it.Value())
		if count > recordInventoryMaxRecords || size > recordInventoryMaxBytes {
			_ = it.Close()
			return fmt.Errorf("predecessor funding inventory exceeds bounds")
		}
		present, err := types.RawFundSettlementFieldPresent(it.Value(), field)
		if err != nil {
			_ = it.Close()
			return err
		}
		if present {
			_ = it.Close()
			return fmt.Errorf("predecessor contains new funding field %d", field)
		}
	}
	return closeSurvivalIterator(it)
}
