package keeper

import (
	"context"

	bvmtypes "github.com/zerone-chain/zerone/x/bvm/types"
)

// BVMKnowledgeAdapter wraps the knowledge Keeper to satisfy
// bvmtypes.KnowledgeKeeper interface.
type BVMKnowledgeAdapter struct {
	k Keeper
}

// NewBVMKnowledgeAdapter returns an adapter that bridges the knowledge keeper
// to the BVM module's expected interface.
func NewBVMKnowledgeAdapter(k Keeper) *BVMKnowledgeAdapter {
	return &BVMKnowledgeAdapter{k: k}
}

// Ensure compile-time interface compliance.
var _ bvmtypes.KnowledgeKeeper = (*BVMKnowledgeAdapter)(nil)

func (a *BVMKnowledgeAdapter) GetFactConfidence(ctx context.Context, factId string) (uint64, bool) {
	fact, found := a.k.GetFact(ctx, factId)
	if !found {
		return 0, false
	}
	return fact.Confidence, true
}
