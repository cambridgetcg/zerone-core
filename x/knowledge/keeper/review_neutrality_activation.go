package keeper

import (
	"bytes"
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/zerone-chain/zerone/x/knowledge/types"
	"google.golang.org/protobuf/encoding/protowire"
)

// ReviewNeutralityEnabledStoreKey is separate from the migration-done markers:
// imported/native genesis preserves execution semantics without inventing an
// applied upgrade. Existing record/payout owners retain keys 0x81..0x83.
const ReviewNeutralityEnabledStoreKey = "\x84"

// ReviewNeutralityEnabled never treats an unreadable or malformed marker as
// legacy state. Callers must propagate the error before choosing semantics.
func (k Keeper) ReviewNeutralityEnabled(ctx context.Context) (bool, error) {
	bz, err := k.storeService.OpenKVStore(ctx).Get([]byte(ReviewNeutralityEnabledStoreKey))
	if err != nil {
		return false, fmt.Errorf("read review neutrality marker: %w", err)
	}
	if bz == nil {
		return false, nil
	}
	if !bytes.Equal(bz, []byte{1}) {
		return false, fmt.Errorf("malformed review neutrality marker")
	}
	return true, nil
}

// EnableReviewNeutrality selects current semantics without altering any claim,
// round, payment or historical record. The owning migrator performs validation
// and writes its separate completion marker in the same outer cache.
func (k Keeper) EnableReviewNeutrality(ctx context.Context) error {
	enabled, err := k.ReviewNeutralityEnabled(ctx)
	if err != nil || enabled {
		return err
	}
	if err := k.storeService.OpenKVStore(ctx).Set([]byte(ReviewNeutralityEnabledStoreKey), []byte{1}); err != nil {
		return fmt.Errorf("write review neutrality marker: %w", err)
	}
	return nil
}

// ClaimReviewPolicyVersion refuses unknown or unactivated policies. The claim's
// admission terms survive later rounds, upgrades, restarts and genesis import.
func (k Keeper) ClaimReviewPolicyVersion(ctx context.Context, claim *types.Claim) (uint32, error) {
	enabled, err := k.ReviewNeutralityEnabled(ctx)
	if err != nil {
		return 0, err
	}
	if claim == nil {
		return 0, fmt.Errorf("missing review-policy claim")
	}
	if err := types.ValidateReviewPolicyVersion(claim.ReviewPolicyVersion, enabled); err != nil {
		return 0, err
	}
	if claim.ReviewPolicyVersion == types.ReviewPolicyNeutral {
		records, err := k.RecordIntegrityEnabled(ctx)
		if err != nil {
			return 0, err
		}
		if !records {
			return 0, fmt.Errorf("neutral review requires record integrity")
		}
	}
	return claim.ReviewPolicyVersion, nil
}

// ValidateReviewNeutralityActivation inventories the exact predecessor records.
// Preseeded new policies, unknown fields and missing claim references are refused.
func (k Keeper) ValidateReviewNeutralityActivation(ctx context.Context) error {
	enabled, err := k.ReviewNeutralityEnabled(ctx)
	if err != nil {
		return err
	}
	if enabled {
		return fmt.Errorf("review neutrality requires unenabled predecessor")
	}
	records, err := k.RecordIntegrityEnabled(ctx)
	if err != nil {
		return err
	}
	if !records {
		return fmt.Errorf("review neutrality requires record-integrity predecessor")
	}
	// These fields do not exist in the c6fa predecessor schema. Even an
	// explicitly encoded zero is preseeded new-policy state, not old evidence.
	for _, spec := range []struct {
		prefix []byte
		field  protowire.Number
	}{
		{types.ClaimKeyPrefix, types.ClaimReviewPolicyField},
		{types.VerificationRoundKeyPrefix, types.RoundReviewPolicyField},
		{types.ContributionByModelKeyPrefix, 9},
	} {
		if err := k.requireAbsentPredecessorPolicyField(ctx, spec.prefix, spec.field); err != nil {
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
	gs := &types.GenesisState{PendingClaims: claims, RecordIntegrityEnabled: true}
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
	if _, err := k.GetAllContributionRecordsChecked(ctx); err != nil {
		return err
	}
	return k.ValidateKnowledgeHistoryState(ctx)
}

func (k Keeper) requireAbsentPredecessorPolicyField(ctx context.Context, prefix []byte, field protowire.Number) error {
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
			return fmt.Errorf("predecessor policy inventory exceeds bounds")
		}
		present, err := types.RawPolicyFieldPresent(it.Value(), field)
		if err != nil {
			_ = it.Close()
			return err
		}
		if present {
			_ = it.Close()
			return fmt.Errorf("predecessor contains new policy field %d", field)
		}
	}
	return closeSurvivalIterator(it)
}

// Validate existing retry reachability without rebuilding an index, changing
// any plan, or inferring that a missing entry was paid. A cursor is only a
// bounded traversal position; it may refer to a plan paid in an earlier batch.
func (k Keeper) validatePendingVerifierRewardIndex(ctx context.Context, rounds []*types.VerificationRound) error {
	expected := make(map[string]bool)
	for _, round := range rounds {
		if err := types.ValidateVerifierRewardSettlement(round); err != nil {
			return err
		}
		if plan := round.VerifierRewardSettlement; plan != nil && plan.PaidAtBlock == 0 {
			expected[string(pendingVerifierRewardKey(round.Id))] = true
		}
	}
	store := k.storeService.OpenKVStore(ctx)
	it, err := store.Iterator(pendingVerifierRewardPrefix, prefixEndBytes(pendingVerifierRewardPrefix))
	if err != nil {
		return err
	}
	count, size := 0, 0
	for ; it.Valid(); it.Next() {
		count++
		size += len(it.Key()) + len(it.Value())
		if count > recordInventoryMaxRecords || size > recordInventoryMaxBytes {
			_ = it.Close()
			return fmt.Errorf("pending verifier index exceeds inventory bounds")
		}
		key := string(it.Key())
		if !bytes.Equal(it.Value(), []byte{1}) || !expected[key] {
			_ = it.Close()
			return fmt.Errorf("pending verifier index does not match an unpaid primary plan")
		}
		delete(expected, key)
	}
	if err := closeSurvivalIterator(it); err != nil {
		return err
	}
	if len(expected) != 0 {
		return fmt.Errorf("pending verifier index omits %d unpaid primary plans", len(expected))
	}
	it, err = store.Iterator(pendingVerifierRewardCursor, prefixEndBytes(pendingVerifierRewardCursor))
	if err != nil {
		return err
	}
	for ; it.Valid(); it.Next() {
		key, cursor := it.Key(), it.Value()
		if !bytes.Equal(key, pendingVerifierRewardCursor) || len(cursor) <= len(pendingVerifierRewardPrefix) || len(cursor) > recordInventoryMaxBytes || !bytes.HasPrefix(cursor, pendingVerifierRewardPrefix) || !utf8.Valid(cursor[len(pendingVerifierRewardPrefix):]) {
			_ = it.Close()
			return fmt.Errorf("invalid pending verifier reward cursor")
		}
	}
	return closeSurvivalIterator(it)
}
