package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/gov/types"
)

// beginBlockPhaseTransitionLegacy is the verbatim body from 77521a0d.
func (k Keeper) beginBlockPhaseTransitionLegacy(ctx sdk.Context) {
	currentHeight := uint64(ctx.BlockHeight())

	k.IteratePhaseTransitionMeta(ctx, func(meta *types.PhaseTransitionProposal) bool {
		if meta.Stage != types.PhaseTransitionStagePending {
			return false
		}

		if meta.ActivationBlock == 0 || currentHeight < meta.ActivationBlock {
			return false
		}

		// Ready to activate.
		if meta.IsRollback {
			k.ExecuteRollback(ctx)
			meta.Stage = types.PhaseTransitionStageActivated
			k.SetPhaseTransitionMeta(ctx, meta)
		} else {
			// Re-verify conditions at activation time.
			_, stillMet := k.CheckPhaseExitConditions(ctx)
			if !stillMet {
				k.CancelPhaseTransition(ctx, meta, "exit conditions no longer met")
				return false
			}
			k.AdvancePhase(ctx, meta.TargetPhase)
			meta.Stage = types.PhaseTransitionStageActivated
			k.SetPhaseTransitionMeta(ctx, meta)
		}

		return false
	})
}
