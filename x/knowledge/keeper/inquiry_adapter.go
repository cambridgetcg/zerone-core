package keeper

import (
	"context"

	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	inquirytypes "github.com/zerone-chain/zerone/x/inquiry/types"
)

// InquiryKnowledgeAdapter wraps a knowledge Keeper to expose the
// narrow read x/inquiry needs:
//   - ClaimSubmitter — owner of a claim, for refusing cross-author
//     answer attachments.
//   - AcceptedFactForClaim — the fact id (and ok) iff a claim has
//     produced an accepted fact in the corpus.
//
// The implementation iterates facts to find one whose ClaimId
// matches. This is O(N) where N is the total fact count. For
// testnet that's tolerable; mainnet should add a claim_id → fact_id
// index. Captured in commitment 16's "what would break it": a fact
// lookup that became too expensive to run in BeginBlocker would
// silently disable auto-resolution.
type InquiryKnowledgeAdapter struct {
	k Keeper
}

func NewInquiryKnowledgeAdapter(k Keeper) *InquiryKnowledgeAdapter {
	return &InquiryKnowledgeAdapter{k: k}
}

// ClaimSubmitter returns the bech32 owner of a claim. ok=false if
// the claim does not exist.
func (a *InquiryKnowledgeAdapter) ClaimSubmitter(ctx context.Context, claimID string) (string, bool) {
	claim, ok := a.k.GetClaim(ctx, claimID)
	if !ok || claim == nil {
		return "", false
	}
	return claim.Submitter, true
}

// AcceptedFactForClaim returns the fact id iff a fact has been
// created from this claim and that fact is in an accepted status
// (VERIFIED or ACTIVE — not DISPROVEN, REVOKED, EXPIRED, etc).
//
// Walks all facts; short-circuits on the first match. The fact must
// be in an accepted lifecycle state — a DISPROVEN fact does not
// constitute a successful answer to an inquiry.
func (a *InquiryKnowledgeAdapter) AcceptedFactForClaim(ctx context.Context, claimID string) (string, bool) {
	if claimID == "" {
		return "", false
	}
	var factID string
	a.k.IterateFacts(ctx, func(f *knowledgetypes.Fact) bool {
		if f == nil || f.ClaimId != claimID {
			return false
		}
		switch f.Status {
		case knowledgetypes.FactStatus_FACT_STATUS_VERIFIED,
			knowledgetypes.FactStatus_FACT_STATUS_ACTIVE:
			factID = f.Id
			return true // stop iteration
		default:
			return false
		}
	})
	if factID == "" {
		return "", false
	}
	return factID, true
}

var _ inquirytypes.KnowledgeKeeper = (*InquiryKnowledgeAdapter)(nil)
