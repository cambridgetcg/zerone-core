package keeper

import (
	"context"
	"fmt"

	billingtypes "github.com/zerone-chain/zerone/x/billing/types"
)

// BillingKnowledgeAdapter wraps the knowledge Keeper to satisfy
// billingtypes.KnowledgeKeeper interface.
type BillingKnowledgeAdapter struct {
	k Keeper
}

// NewBillingKnowledgeAdapter returns an adapter that bridges the knowledge keeper
// to the billing module's expected interface.
func NewBillingKnowledgeAdapter(k Keeper) *BillingKnowledgeAdapter {
	return &BillingKnowledgeAdapter{k: k}
}

// Ensure compile-time interface compliance.
var _ billingtypes.KnowledgeKeeper = (*BillingKnowledgeAdapter)(nil)

func (a *BillingKnowledgeAdapter) GetFactConfidence(ctx context.Context, factId string) (uint64, bool) {
	fact, found := a.k.GetFact(ctx, factId)
	if !found {
		return 0, false
	}
	return fact.Confidence, true
}

func (a *BillingKnowledgeAdapter) GetFactCitationCount(ctx context.Context, factId string) (uint64, bool) {
	fact, found := a.k.GetFact(ctx, factId)
	if !found {
		return 0, false
	}
	return fact.CitationCount + fact.IncomingCitationCount, true
}

func (a *BillingKnowledgeAdapter) GetFactSubmitter(ctx context.Context, factId string) (string, bool) {
	fact, found := a.k.GetFact(ctx, factId)
	if !found {
		return "", false
	}
	return fact.Submitter, true
}

func (a *BillingKnowledgeAdapter) GetFactCreatedBlock(ctx context.Context, factId string) (uint64, bool) {
	fact, found := a.k.GetFact(ctx, factId)
	if !found {
		return 0, false
	}
	return fact.VerifiedAtBlock, true
}

func (a *BillingKnowledgeAdapter) IncrementCitationCount(ctx context.Context, factId string) error {
	fact, found := a.k.GetFact(ctx, factId)
	if !found {
		return fmt.Errorf("fact not found: %s", factId)
	}
	fact.CitationCount++
	return a.k.SetFact(ctx, fact)
}
