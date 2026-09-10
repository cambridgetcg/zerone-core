package keeper

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/zerone-chain/zerone/x/knowledge/types"
	"google.golang.org/protobuf/proto"
)

// A nondecisive review must not undo another open challenge. This reads the
// existing active-round index, with the same explicit 100k-record/64MiB safety
// ceilings as the record inventory. Exhaustion refuses this restoration; it
// does not reject new claim admission or create a new control index.
func (k Keeper) hasOtherActiveChallenge(ctx context.Context, excludeRoundID, targetFactID string) (found bool, err error) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.ActiveRoundIndexPrefix, prefixEndBytes(types.ActiveRoundIndexPrefix))
	if err != nil {
		return false, err
	}
	defer func() { err = errors.Join(err, closeSurvivalIterator(iter)) }()
	records, totalBytes := 0, 0
	charge := func(size int) error {
		if size > recordInventoryMaxBytes-totalBytes {
			return fmt.Errorf("active challenge scan exceeds byte bound")
		}
		totalBytes += size
		return nil
	}
	for ; iter.Valid(); iter.Next() {
		records++
		if records > recordInventoryMaxRecords {
			return false, fmt.Errorf("active challenge scan exceeds record bound")
		}
		key := iter.Key()
		if err := charge(len(key) + len(iter.Value())); err != nil {
			return false, err
		}
		if len(key) <= len(types.ActiveRoundIndexPrefix) || !bytes.Equal(iter.Value(), []byte{1}) {
			return false, fmt.Errorf("malformed active review index")
		}
		id := string(key[len(types.ActiveRoundIndexPrefix):])
		roundBytes, err := store.Get(types.RoundKey(id))
		if err != nil {
			return false, err
		}
		if err := charge(len(roundBytes)); err != nil {
			return false, err
		}
		if roundBytes == nil {
			return false, fmt.Errorf("active review round is missing")
		}
		if err := types.ValidateRawPolicyField(roundBytes, types.RoundReviewPolicyField); err != nil {
			return false, err
		}
		var round types.VerificationRound
		if err := proto.Unmarshal(roundBytes, &round); err != nil {
			return false, err
		}
		if round.Id != id || hasUnknownRecordFields(round.ProtoReflect()) {
			return false, fmt.Errorf("active review identity or fields invalid")
		}
		if round.Id == excludeRoundID {
			continue
		}
		if round.Phase == types.VerificationPhase_VERIFICATION_PHASE_COMPLETE || round.Phase == types.VerificationPhase_VERIFICATION_PHASE_EXPIRED {
			return false, fmt.Errorf("terminal round remains in active review index")
		}
		if err := types.ValidateVerificationRoundRecord(&round, true); err != nil {
			return false, err
		}
		claimBytes, err := store.Get(types.ClaimKey(round.ClaimId))
		if err != nil {
			return false, err
		}
		if err := charge(len(claimBytes)); err != nil {
			return false, err
		}
		if claimBytes == nil {
			return false, fmt.Errorf("active review claim is missing")
		}
		if err := types.ValidateRawPolicyField(claimBytes, types.ClaimReviewPolicyField); err != nil {
			return false, err
		}
		var claim types.Claim
		if err := proto.Unmarshal(claimBytes, &claim); err != nil {
			return false, err
		}
		if claim.Id != round.ClaimId || hasUnknownRecordFields(claim.ProtoReflect()) {
			return false, fmt.Errorf("active claim identity or fields invalid")
		}
		if _, err := k.ClaimReviewPolicyVersion(ctx, &claim); err != nil {
			return false, err
		}
		if claim.ReviewPolicyVersion != round.ReviewPolicyVersion {
			return false, fmt.Errorf("active review policy differs from claim")
		}
		if claim.ProvisionalFactId == targetFactID {
			return true, nil
		}
	}
	return false, nil
}
