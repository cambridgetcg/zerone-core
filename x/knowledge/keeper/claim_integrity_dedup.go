package keeper

import (
	"context"
	"fmt"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Status-aware duplicate checks (knowledge v12, plan §5.4; P1).
//
// A dedup index entry (0x09 content hash, 0x34 canonical hash) names the
// latest admitted record with that hash. With claim integrity on it blocks
// admission only while that record is live or standing. Nothing is deleted or
// rewritten: a terminal claim or a fallen fact simply stops blocking, and the
// next admission's first save (SetClaim) takes the entry. Errors are returned
// to the tx, so an unreadable entry never admits a possible duplicate.

// dedupBlocker names the record that blocks an admission and its status.
type dedupBlocker struct {
	id, status string
}

func (b *dedupBlocker) String() string {
	return fmt.Sprintf("%s (status %s)", b.id, b.status)
}

// blockingDuplicate returns the record under an index key that still blocks
// admission, or nil when the key is absent or its record no longer blocks.
func (k Keeper) blockingDuplicate(ctx context.Context, key []byte) (*dedupBlocker, error) {
	bz, err := k.storeService.OpenKVStore(ctx).Get(key)
	if err != nil {
		return nil, fmt.Errorf("read dedup index entry: %w", err)
	}
	if bz == nil {
		return nil, nil
	}
	id := string(bz)
	blocks, status, err := k.dedupEntryBlocks(ctx, id)
	if err != nil || !blocks {
		return nil, err
	}
	return &dedupBlocker{id: id, status: status}, nil
}

// dedupEntryBlocks implements the §5.4 table for the record an entry names,
// resolving a fact first and then a claim. It also returns the status that
// decided, for the refusal text. An entry naming nothing does not block; an
// ACCEPTED claim whose round or fact is missing blocks (fails closed).
func (k Keeper) dedupEntryBlocks(ctx context.Context, id string) (bool, string, error) {
	if id == "" {
		return false, "", fmt.Errorf("dedup index entry is empty")
	}
	fact, found, err := k.getFactChecked(ctx, id)
	if err != nil {
		return false, "", fmt.Errorf("dedup index entry %s: read fact: %w", id, err)
	}
	if found {
		return factBlocksDedup(fact.Status), fact.Status.String(), nil
	}
	claim, err := k.getClaimRecordChecked(ctx, id)
	if err != nil {
		return false, "", fmt.Errorf("dedup index entry %s: read claim: %w", id, err)
	}
	if claim == nil {
		return false, "", nil
	}
	status := claim.Status.String()
	switch claim.Status {
	case types.ClaimStatus_CLAIM_STATUS_REJECTED,
		types.ClaimStatus_CLAIM_STATUS_MALFORMED,
		types.ClaimStatus_CLAIM_STATUS_INSUFFICIENT,
		types.ClaimStatus_CLAIM_STATUS_EXPIRED:
		return false, status, nil
	case types.ClaimStatus_CLAIM_STATUS_ACCEPTED:
	default:
		// Live, UNSPECIFIED and any unknown status block.
		return true, status, nil
	}

	// The accepted claim's fact has the ID createFactFromClaim gave it at the
	// verdict height.
	roundID := claim.VerificationRoundId
	if roundID == "" {
		return true, status + "; no round", nil
	}
	roundBytes, err := k.storeService.OpenKVStore(ctx).Get(types.RoundKey(roundID))
	if err != nil {
		return false, "", fmt.Errorf("dedup index entry %s: read round %s: %w", id, roundID, err)
	}
	if roundBytes == nil {
		return true, status + "; round " + roundID + " missing", nil
	}
	round, err := k.getVerificationRoundChecked(ctx, roundID)
	if err != nil {
		return false, "", fmt.Errorf("dedup index entry %s: read round %s: %w", id, roundID, err)
	}
	factID := GenerateFactID(claim.Id, round.VerdictBlock)
	fact, found, err = k.getFactChecked(ctx, factID)
	if err != nil {
		return false, "", fmt.Errorf("dedup index entry %s: read fact %s: %w", id, factID, err)
	}
	if !found {
		return true, status + "; fact " + factID + " missing", nil
	}
	return factBlocksDedup(fact.Status), status + "; fact " + factID + " " + fact.Status.String(), nil
}

// factBlocksDedup: a fallen fact releases its hashes; a standing fact,
// UNSPECIFIED and any unknown status block.
func factBlocksDedup(status types.FactStatus) bool {
	switch status {
	case types.FactStatus_FACT_STATUS_SUPERSEDED,
		types.FactStatus_FACT_STATUS_EXPIRED,
		types.FactStatus_FACT_STATUS_DISPROVEN,
		types.FactStatus_FACT_STATUS_REVOKED,
		types.FactStatus_FACT_STATUS_PRUNED:
		return false
	default:
		return true
	}
}

// acceptedFactMayTakeCanonicalEntry reports whether an accepted claim's fact
// may take the 0x34 entry. With claim integrity on it may only when the entry
// is absent or still names the claim itself, so a late acceptance never takes
// the entry from a newer admitted record. With it off it always may.
func (k Keeper) acceptedFactMayTakeCanonicalEntry(ctx context.Context, hash, claimID string) (bool, error) {
	integrity, err := k.ClaimIntegrityEnabled(ctx)
	if err != nil {
		return false, err
	}
	if !integrity {
		return true, nil
	}
	bz, err := k.storeService.OpenKVStore(ctx).Get(types.CanonicalHashKey(hash))
	if err != nil {
		return false, fmt.Errorf("read canonical index entry: %w", err)
	}
	return bz == nil || string(bz) == claimID, nil
}
