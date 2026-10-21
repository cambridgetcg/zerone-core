package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// beginBlockInconclusiveAccountingLegacy preserves the BeginBlocker body verbatim
// from 6a56b621082064483af1ced7718462766d9ae756. AdvanceRoundPhases,
// performAggregation, AggregateVerificationResult and CompleteRound retain their
// original bodies too; the activated path does not modify those helpers.
func (k Keeper) beginBlockInconclusiveAccountingLegacy(ctx context.Context) error {
	// This marker is read once, before ANY begin-block write. Its reader
	// wraps activation.ErrMarkerFault; ordinary historical errors below keep
	// their existing handling rather than becoming new consensus halts.
	reformulationOrdering, err := k.ReformulationVindicationOrderingEnabled(ctx)
	if err != nil {
		return err
	}
	// Retry a fixed number of frozen verifier obligations independently of
	// scientific round processing. Failed bank sends retain the whole plan.
	if err := k.ProcessPendingVerifierRewards(ctx); err != nil {
		return err
	}
	// Route B Wave 8: the heartbeat. Self-maintenance of the training
	// infrastructure — bounty expiry / vesting release / manifest
	// supersession. Runs first so a round-advance error cannot silently
	// starve Route B lifecycle progression. Non-fatal: errors logged inside.
	k.processRouteBLifecycle(ctx, reformulationOrdering)

	if err := k.AdvanceRoundPhases(ctx); err != nil {
		return err
	}

	// Survival-gate: issue escrowed submitter rewards for facts whose challenge
	// window closed while still VERIFIED (survived unchallenged).
	k.SweepSurvivedRewards(ctx)

	// Check if we're at a fitness epoch boundary
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	height := uint64(sdkCtx.BlockHeight())
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil // non-fatal: don't block consensus for param read failure
	}
	// Prune expired vindication entries every 1000 blocks
	if height > 0 && height%1000 == 0 && params.VindicationRefundEnabled {
		k.pruneExpiredVindications(ctx, height, params.VindicationWindowBlocks, reformulationOrdering)
	}

	if params.FitnessEpochBlocks > 0 && height > 0 && height%params.FitnessEpochBlocks == 0 {
		epoch := height / params.FitnessEpochBlocks

		// Order matters:
		// 1. Update fitness scores (current usage data)
		if err := k.UpdateAllFitnessScores(ctx); err != nil {
			k.Logger(ctx).Error("fitness update failed", "error", err)
		}
		// 2. Process competition (uses fitness to rank niches)
		if err := k.ProcessCompetition(ctx, epoch); err != nil {
			k.Logger(ctx).Error("competition processing failed", "epoch", epoch, "error", err)
		}
		// 3. Process symbiosis (adjusts fitness based on relationships)
		k.ProcessSymbiosis(ctx, params)
		// 4. Process metabolism (uses fitness + competition tax to drain/replenish energy)
		if err := k.ProcessMetabolism(ctx, epoch); err != nil {
			k.Logger(ctx).Error("metabolism processing failed", "epoch", epoch, "error", err)
		}
		// 5. Process agent demand bounties
		if err := k.ProcessDemandBounties(ctx, epoch); err != nil {
			k.Logger(ctx).Error("demand bounty processing failed", "epoch", epoch, "error", err)
		}
		// 6. Clean up expired bounties
		k.ProcessExpiredBounties(ctx)
		// 7. Clear query receipts (bound receipt storage to one epoch)
		k.ClearQueryReceipts(ctx)
		// 8. Aggregate diversity metrics and check conformity alerts (R28-2)
		if err := k.ProcessDiversity(ctx, epoch); err != nil {
			k.Logger(ctx).Error("diversity processing failed", "epoch", epoch, "error", err)
		}
		// 9. Update epistemic temperature for all domains (R29-2)
		k.IterateDomains(ctx, func(domain *types.Domain) bool {
			if dErr := k.UpdateEpistemicTemperature(ctx, domain.Name); dErr != nil {
				k.Logger(ctx).Error("epistemic temperature update failed", "domain", domain.Name, "error", dErr)
			}
			return false
		})
		// 10. Decay domain role elasticity records (R29-3)
		if params.RoleElasticityDecayEpochs > 0 && epoch%params.RoleElasticityDecayEpochs == 0 {
			if err := k.DecayRoleRecords(ctx); err != nil {
				return err
			}
		}
	}

	// Advance fact confidence at ConfidenceGrowthEpoch intervals (R29-2)
	if params.ConfidenceGrowthEpoch > 0 && height > 0 && height%params.ConfidenceGrowthEpoch == 0 {
		if err := k.AdvanceConfidence(ctx); err != nil {
			k.Logger(ctx).Error("confidence growth failed", "error", err)
		}
	}

	// Wave 15: stress-test invitation heartbeat. Chain-driven demand for
	// probes on idle high-confidence facts. Bounded per block by BOTH
	// ProbeInvitationBatchSize (invitations) and probeScanMaxExaminedPerBlock
	// (facts read), resuming from a persisted cursor. The batch size alone
	// did NOT bound this — see InviteIdleFactsForProbing.
	if params.ProbeInvitationIdleThresholdBlocks > 0 {
		if err := k.InviteIdleFactsForProbing(ctx, height, params); err != nil {
			return err
		}
	}

	// Wave 15: probe bounty pool mint. Per-block issuance into the pool
	// that funds successful-probe bonuses, capped at ProbeBountyMaxPoolSize
	// so minting throttles naturally.
	if params.ProbeBountyMintPerBlock != "" && params.ProbeBountyMintPerBlock != "0" {
		if err := k.MintToProbeBountyPool(ctx, params); err != nil {
			return err
		}
	}

	// Wave 16: materialize pending fact injections whose guardian-veto
	// window has expired without a veto. Until materialization the fact
	// does not exist in state — guardians can still cancel.
	k.MaterializeMaturedFactInjections(ctx, height)

	return nil
}
