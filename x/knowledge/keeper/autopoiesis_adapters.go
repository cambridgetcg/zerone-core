package keeper

import (
	"context"

	apkeeper "github.com/zerone-chain/zerone/x/autopoiesis/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// AutopoiesisKnowledgeAdapter wraps the autopoiesis Keeper to satisfy
// knowledgetypes.AutopoiesisKeeper (returns uint64, error).
type AutopoiesisKnowledgeAdapter struct {
	k apkeeper.Keeper
}

// NewAutopoiesisKnowledgeAdapter returns an adapter for the knowledge module.
func NewAutopoiesisKnowledgeAdapter(k apkeeper.Keeper) *AutopoiesisKnowledgeAdapter {
	return &AutopoiesisKnowledgeAdapter{k: k}
}

// Compile-time interface check.
var _ knowledgetypes.AutopoiesisKeeper = (*AutopoiesisKnowledgeAdapter)(nil)

// GetMultiplier satisfies knowledgetypes.AutopoiesisKeeper.
func (a *AutopoiesisKnowledgeAdapter) GetMultiplier(ctx context.Context, path string) (uint64, error) {
	return a.k.GetMultiplier(ctx, path)
}
