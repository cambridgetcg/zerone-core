package keeper

import (
	"context"

	alignmenttypes "github.com/zerone-chain/zerone/x/alignment/types"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// AlignmentKnowledgeAdapter wraps the knowledge Keeper to satisfy
// alignmenttypes.KnowledgeKeeper interface.
type AlignmentKnowledgeAdapter struct {
	k Keeper
}

// NewAlignmentKnowledgeAdapter returns an adapter for the alignment module.
func NewAlignmentKnowledgeAdapter(k Keeper) *AlignmentKnowledgeAdapter {
	return &AlignmentKnowledgeAdapter{k: k}
}

// Ensure compile-time interface compliance.
var _ alignmenttypes.KnowledgeKeeper = (*AlignmentKnowledgeAdapter)(nil)

// GetVerificationRate computes accepted / terminal claims in BPS.
func (a *AlignmentKnowledgeAdapter) GetVerificationRate(ctx context.Context) uint64 {
	var accepted, terminal uint64
	a.k.IterateClaims(ctx, func(claim *types.Claim) bool {
		switch claim.Status {
		case types.ClaimStatus_CLAIM_STATUS_ACCEPTED:
			accepted++
			terminal++
		case types.ClaimStatus_CLAIM_STATUS_REJECTED,
			types.ClaimStatus_CLAIM_STATUS_MALFORMED,
			types.ClaimStatus_CLAIM_STATUS_INSUFFICIENT:
			terminal++
		}
		return false
	})
	if terminal == 0 {
		return 500_000 // NeutralBPS — no data yet
	}
	rate := accepted * 1_000_000 / terminal
	if rate > 1_000_000 {
		return 1_000_000
	}
	return rate
}

// GetSurvivedChallengeRate computes survived / (survived + disproven) facts in BPS.
// This is the survival-gate coupling signal: a fact that stood an adversarial
// challenge (CorroborationCount > 0) is quality expensive to fake, while a
// FACT_STATUS_DISPROVEN fact lost its challenge. Facts never challenged are
// excluded, so the rate measures the truth quality of the knowledge base under
// adversarial pressure — not acceptance volume. Neutral (500_000) until at least
// one challenge has resolved. Unlike accept-rate, rewarding this cannot be gamed
// by rubber-stamping: reward flows to survival, and rejecting a false claim
// (which then falls to DISPROVEN on challenge) raises the rate rather than
// lowering it.
func (a *AlignmentKnowledgeAdapter) GetSurvivedChallengeRate(ctx context.Context) uint64 {
	var survived, disproven uint64
	a.k.IterateFacts(ctx, func(fact *types.Fact) bool {
		switch {
		case fact.Status == types.FactStatus_FACT_STATUS_DISPROVEN:
			disproven++
		case fact.CorroborationCount > 0:
			survived++
		}
		return false
	})
	challenged := survived + disproven
	if challenged == 0 {
		return 500_000 // NeutralBPS — no challenges resolved yet
	}
	rate := survived * 1_000_000 / challenged
	if rate > 1_000_000 {
		return 1_000_000
	}
	return rate
}

// GetTotalFacts counts all accepted claims (facts).
func (a *AlignmentKnowledgeAdapter) GetTotalFacts(ctx context.Context) uint64 {
	var count uint64
	a.k.IterateClaims(ctx, func(claim *types.Claim) bool {
		if claim.Status == types.ClaimStatus_CLAIM_STATUS_ACCEPTED {
			count++
		}
		return false
	})
	return count
}

// GetConsensusDiversity returns the global consensus diversity score in BPS.
func (a *AlignmentKnowledgeAdapter) GetConsensusDiversity(ctx context.Context) uint64 {
	return a.k.GetGlobalConsensusDiversity(ctx)
}

// GetPendingVerificationRatio returns pending claims / active facts in BPS (R31-1).
func (a *AlignmentKnowledgeAdapter) GetPendingVerificationRatio(ctx context.Context) uint64 {
	return a.k.GetPendingVerificationRatio(ctx)
}

// GetVerificationHealth returns verification health metrics for the alignment sensor (R31-2).
func (a *AlignmentKnowledgeAdapter) GetVerificationHealth(ctx context.Context) (uint64, uint64, uint64) {
	return a.k.GetVerificationHealth(ctx)
}
