package keeper

import (
	"context"

	"github.com/zerone-chain/zerone/x/knowledge/types"
	vestingrewardstypes "github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

// VestingRewardsKnowledgeAdapter wraps the knowledge Keeper to satisfy
// vesting_rewards/types.KnowledgeKeeper. This allows block rewards to be
// coupled to verification throughput (T9 / thesis claim 1).
type VestingRewardsKnowledgeAdapter struct {
	alignmentAdapter *AlignmentKnowledgeAdapter
	keeper           Keeper
}

// NewVestingRewardsKnowledgeAdapter returns an adapter for vesting_rewards.
func NewVestingRewardsKnowledgeAdapter(k Keeper) *VestingRewardsKnowledgeAdapter {
	return &VestingRewardsKnowledgeAdapter{
		alignmentAdapter: NewAlignmentKnowledgeAdapter(k),
		keeper:           k,
	}
}

// Ensure compile-time interface compliance.
var _ vestingrewardstypes.KnowledgeKeeper = (*VestingRewardsKnowledgeAdapter)(nil)

// GetVerificationRate delegates to the shared accepted-over-terminal calculation
// (legacy accept-rate; retained for the audit query, no longer couples emission).
func (a *VestingRewardsKnowledgeAdapter) GetVerificationRate(ctx context.Context) uint64 {
	return a.alignmentAdapter.GetVerificationRate(ctx)
}

// GetSurvivedChallengeRate delegates to the survival-gate calculation —
// survived/(survived+disproven) facts. This is what block emission couples to.
func (a *VestingRewardsKnowledgeAdapter) GetSurvivedChallengeRate(ctx context.Context) uint64 {
	return a.alignmentAdapter.GetSurvivedChallengeRate(ctx)
}

// IsFactDisproven reports whether the PoT layer has adjudicated this fact
// false. It is the single predicate standing between a vesting schedule and
// clawback: falsification is something the chain concludes, never something a
// caller asserts. An empty or unknown fact id returns false, so a clawback
// cannot be authorised by naming a fact that does not exist.
func (a *VestingRewardsKnowledgeAdapter) IsFactDisproven(ctx context.Context, factID string) bool {
	if factID == "" {
		return false
	}
	fact, found := a.keeper.GetFact(ctx, factID)
	if !found || fact == nil {
		return false
	}
	return fact.Status == types.FactStatus_FACT_STATUS_DISPROVEN
}
