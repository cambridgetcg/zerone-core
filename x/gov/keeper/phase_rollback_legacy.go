package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/gov/types"
)

// validatePhaseRollbackLegacy is the verbatim body from 77521a0d.
func (k Keeper) validatePhaseRollbackLegacy(ctx sdk.Context) error {
	state := k.GetResearchFundGovernanceState(ctx)

	if state.CurrentPhase <= types.ResearchFundPhase_RESEARCH_FUND_PHASE_GENESIS_PAIR {
		return types.ErrCannotRollbackGenesis
	}

	// Check there's no existing active phase transition.
	if k.HasActivePhaseTransitionLIP(ctx) {
		return types.ErrPendingTransitionExists
	}

	// Check gridlock: 3+ consecutive expired proposals.
	gridlocked := k.CountConsecutiveExpiredProposals(ctx) >= types.RollbackGridlockThreshold

	// Check emergency halt for research fund.
	haltTriggered := false
	if k.emergencyKeeper != nil {
		haltTriggered = k.emergencyKeeper.CountHaltsForReason(ctx, "research_fund") > 0
	}

	if !gridlocked && !haltTriggered {
		return types.ErrNoRollbackJustification
	}

	return nil
}
