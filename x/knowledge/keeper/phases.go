package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// BeginBlocker runs knowledge module begin-block logic.
// Advances verification round phases by deadline and triggers fitness epoch updates.
func (k Keeper) BeginBlocker(ctx context.Context) error {
	// Route B Wave 8: the heartbeat. Self-maintenance of the training
	// infrastructure — bounty expiry / vesting release / manifest
	// supersession. Runs first so a round-advance error cannot silently
	// starve Route B lifecycle progression. Non-fatal: errors logged inside.
	k.ProcessRouteBLifecycle(ctx)

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
		k.PruneExpiredVindications(ctx, height, params.VindicationWindowBlocks)
	}

	if params.FitnessEpochBlocks > 0 && height > 0 && height%params.FitnessEpochBlocks == 0 {
		epoch := height / params.FitnessEpochBlocks
		// Diversity consumes the epoch that just COMPLETED at this boundary:
		// rounds finalized during it carry epoch (height/E)-1
		// (RecordRoundDiversity labels a round with height/E at finalization).
		// `epoch` itself names the epoch BEGINNING at this height — aggregating
		// it here found only rounds finalizing at exactly the boundary block,
		// so the completed epoch's unanimity was invisible and the conformity
		// streak reset every epoch (off-by-one fixed 2026-08-03). At the first
		// boundary (H = E) the completed epoch is 0, covering heights 0..E-1.
		// The other consumers below (competition, metabolism, demand bounties,
		// role-elasticity pacing) use `epoch` as a monotone label, not as a
		// window over past rounds, and keep the current epoch.
		completedEpoch := uint64(0)
		if epoch > 0 {
			completedEpoch = epoch - 1
		}

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
		if err := k.ProcessDiversity(ctx, completedEpoch); err != nil {
			k.Logger(ctx).Error("diversity processing failed", "epoch", completedEpoch, "error", err)
		}
		// 9. Update epistemic temperature for all domains (R29-2) — consuming
		// the same completed epoch step 8 just aggregated.
		k.IterateDomains(ctx, func(domain *types.Domain) bool {
			if dErr := k.UpdateEpistemicTemperature(ctx, domain.Name, completedEpoch); dErr != nil {
				k.Logger(ctx).Error("epistemic temperature update failed", "domain", domain.Name, "error", dErr)
			}
			return false
		})
		// 10. Decay domain role elasticity records (R29-3)
		if params.RoleElasticityDecayEpochs > 0 && epoch%params.RoleElasticityDecayEpochs == 0 {
			k.DecayRoleRecords(ctx)
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
		k.InviteIdleFactsForProbing(ctx, height, params)
	}

	// Wave 15: probe bounty pool mint. Per-block issuance into the pool
	// that funds successful-probe bonuses, capped at ProbeBountyMaxPoolSize
	// so minting throttles naturally.
	if params.ProbeBountyMintPerBlock != "" && params.ProbeBountyMintPerBlock != "0" {
		k.MintToProbeBountyPool(ctx, params)
	}

	// Wave 16: materialize pending fact injections whose guardian-veto
	// window has expired without a veto. Until materialization the fact
	// does not exist in state — guardians can still cancel.
	k.MaterializeMaturedFactInjections(ctx, height)

	return nil
}

// AdvanceRoundPhases iterates all active rounds and transitions phases by
// deadline — or early, once a round's reveal set is closed (see GetExpectedPhase).
func (k Keeper) AdvanceRoundPhases(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	height := uint64(sdkCtx.BlockHeight())

	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}

	// Collect rounds to process (avoid modifying store during iteration)
	var roundsToProcess []*types.VerificationRound
	k.IterateActiveRounds(ctx, func(round *types.VerificationRound) bool {
		roundsToProcess = append(roundsToProcess, round)
		return false
	})

	for _, round := range roundsToProcess {
		expectedPhase := GetExpectedPhase(round, height, params)

		if expectedPhase == round.Phase {
			continue // no transition needed
		}

		switch expectedPhase {
		case types.VerificationPhase_VERIFICATION_PHASE_REVEAL:
			// COMMIT → REVEAL transition
			round.Phase = types.VerificationPhase_VERIFICATION_PHASE_REVEAL
			if err := k.SetVerificationRound(ctx, round); err != nil {
				k.Logger(ctx).Error("failed to transition round to REVEAL", "round_id", round.Id, "error", err)
			}
			sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
				"zerone.knowledge.round_phase_changed",
				sdk.NewAttribute("round_id", round.Id),
				sdk.NewAttribute("phase", "REVEAL"),
			))

		case types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION:
			// Early advance only (the deadline path needs no extra gate): the
			// closed commit set must clear the same quorum threshold
			// aggregation itself reads THIS block — the override-inclusive
			// effective minimum, not raw MinVerifiers. Otherwise a
			// fast-revealing panel finalizes under a threshold read at a
			// height it chose, evading an active capture-challenge override
			// (review fix). A held round loses nothing: by the reveal
			// deadline the override either still binds (INCONCLUSIVE, as
			// under fixed deadlines) or has expired (decisive, as under
			// fixed deadlines).
			if height < round.RevealDeadline &&
				uint64(len(round.Commits)) < k.effectiveMinVerifiersForRound(ctx, round, params) {
				continue
			}
			// REVEAL → AGGREGATION transition
			round.Phase = types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION
			if err := k.SetVerificationRound(ctx, round); err != nil {
				k.Logger(ctx).Error("failed to transition round to AGGREGATION", "round_id", round.Id, "error", err)
				continue
			}
			sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
				"zerone.knowledge.round_phase_changed",
				sdk.NewAttribute("round_id", round.Id),
				sdk.NewAttribute("phase", "AGGREGATION"),
				sdk.NewAttribute("reveal_count", fmt.Sprintf("%d", len(round.Reveals))),
			))
			// Perform aggregation immediately
			if err := k.performAggregation(ctx, round); err != nil {
				k.Logger(ctx).Error("aggregation failed", "round_id", round.Id, "error", err)
			}

		case types.VerificationPhase_VERIFICATION_PHASE_EXPIRED:
			// Round has expired — check if we can still aggregate
			if uint64(len(round.Reveals)) >= params.MinVerifiers {
				// Enough reveals — aggregate
				round.Phase = types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION
				if err := k.SetVerificationRound(ctx, round); err != nil {
					continue
				}
				if err := k.performAggregation(ctx, round); err != nil {
					k.Logger(ctx).Error("late aggregation failed", "round_id", round.Id, "error", err)
				}
			} else {
				// Insufficient reveals — mark as expired
				round.Phase = types.VerificationPhase_VERIFICATION_PHASE_EXPIRED
				round.Verdict = types.Verdict_VERDICT_INCONCLUSIVE
				if err := k.SetVerificationRound(ctx, round); err != nil {
					continue
				}
				// Review fee is non-refundable — mark claim as insufficient
				claim, found := k.GetClaim(ctx, round.ClaimId)
				if found {
					claim.Status = types.ClaimStatus_CLAIM_STATUS_INSUFFICIENT
					_ = k.SetClaim(ctx, claim)
					// A starved round (the chain failed to seat a panel) must
					// not silently erase the attempt or become a griefing
					// weapon. Perform the same three side effects the
					// INCONCLUSIVE arm of CompleteRound performs — and ONLY
					// those (no invitation bonus, no diversity/metric indexing),
					// leaving the round EXPIRED rather than COMPLETE.
					//
					// (1) C2 / COMPASSION.md: record the attempt in the
					//     calibration ledger (otherwise done only in CompleteRound),
					//     so a being whose ordinary claims all starve still gets
					//     credited. Conjectures remain exempt here exactly as they
					//     are in CompleteRound: asking a well-posed question cannot
					//     create or change truth-calibration standing merely because
					//     the chain failed to seat a panel.
					if claim.ClaimType != types.ClaimType_CLAIM_TYPE_CONJECTURE {
						k.RecordSubmissionOutcome(ctx, claim.Submitter, ResolveMethodId(claim.MethodId), types.Verdict_VERDICT_INCONCLUSIVE)
					}
					// (2) Attack fix: a starved CONTRADICTS claim must not leave
					//     its target fact locked CONTESTED forever (0.1-ZRN grief).
					k.reverseContradictionsFromClaim(ctx, claim)
					// (3) Attack fix: a starved CHALLENGE must not leave its
					//     target fact locked CHALLENGED forever — restore it with
					//     no survival credit, since no panel actually judged it.
					k.restoreChallengedFactOnInconclusive(ctx, claim)
					// K-alpha: this is the one terminal close outside
					// CompleteRound — and the default fate of any claim that
					// cannot attract MinVerifiers reveals, i.e. the most
					// common INCONCLUSIVE route on a thin-verifier chain.
					// The pending pair opened at submission settles here too
					// (K-2; review fix: this path previously left every
					// starved claim's pair open forever). Emission cannot
					// error; O(1) per already-iterated round.
					emitKarmaEdgeState(sdkCtx, "pending_settle", karmaEdgeStateOrdinal,
						claim.Submitter, "", claim.Id, claim.Domain,
						sdk.NewAttribute("verdict", types.Verdict_VERDICT_INCONCLUSIVE.String()))
				}
				sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
					"zerone.knowledge.round_expired",
					sdk.NewAttribute("round_id", round.Id),
					sdk.NewAttribute("reveals", fmt.Sprintf("%d", len(round.Reveals))),
				))
			}
		}
	}

	return nil
}

// effectiveMinVerifiersForRound mirrors the quorum read that
// AggregateVerificationResult performs (confidence.go): domain claims use the
// domain-adjusted, capture-override-inclusive minimum; domainless claims (or
// rounds whose claim no longer resolves) use the raw params minimum. Reading
// it at the advance height keeps the early-aggregation gate and the
// aggregation quorum on the same block, so a panel cannot pick which
// threshold snapshot applies by timing its reveals.
func (k Keeper) effectiveMinVerifiersForRound(ctx context.Context, round *types.VerificationRound, params *types.Params) uint64 {
	if claim, found := k.GetClaim(ctx, round.ClaimId); found && claim.Domain != "" {
		return uint64(k.GetEffectiveMinVerifiers(ctx, claim.Domain))
	}
	return params.MinVerifiers
}

// GetExpectedPhase is a pure function that maps a block height to the expected phase.
//
// Two refinements over raw deadline arithmetic (K-alpha):
//
//   - Early aggregation: inside the reveal window, once every committer has
//     revealed and the commit set meets MinVerifiers, no further reveal can
//     arrive (a reveal requires a matching commit and the commit window is
//     closed), so waiting out the reveal deadline adds only latency. The
//     commit window itself is never shortened. The MinVerifiers read here is
//     a cheap pure precondition; AdvanceRoundPhases re-checks the
//     override-inclusive effective minimum at the advance height before
//     acting, so an active capture-challenge override binds early advances
//     exactly as it binds deadline aggregation. Residual, printed rather
//     than papered: aggregation height still moves with the last reveal, so
//     state read at aggregation (per-verifier stake snapshots, an override
//     filed only after the final reveal landed) is read earlier than the
//     fixed reveal deadline would have read it — the objection window for
//     post-reveal capture challenges narrows to the gap between the last
//     reveal and the advance block.
//   - Phase monotonicity: after an early advance the stored phase leads the
//     deadline arithmetic. Without flooring at the stored phase, a failed
//     aggregation — which leaves the round standing in AGGREGATION — would be
//     regressed to REVEAL next block and reprocessed forever. A round never
//     moves backwards: the result is max(deadline-expected, stored) in
//     lifecycle order (the VerificationPhase enum is declared in that order).
func GetExpectedPhase(round *types.VerificationRound, height uint64, params *types.Params) types.VerificationPhase {
	if round.Phase == types.VerificationPhase_VERIFICATION_PHASE_COMPLETE ||
		round.Phase == types.VerificationPhase_VERIFICATION_PHASE_EXPIRED {
		return round.Phase
	}

	expected := types.VerificationPhase_VERIFICATION_PHASE_COMMIT
	switch {
	case height >= round.AggregationDeadline:
		expected = types.VerificationPhase_VERIFICATION_PHASE_EXPIRED
	case height >= round.RevealDeadline:
		expected = types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION
	case height >= round.CommitDeadline:
		expected = types.VerificationPhase_VERIFICATION_PHASE_REVEAL
	}

	// Early aggregation: the reveal set is closed (both counts are consensus
	// state), so the verdict-bearing transition needs no further waiting.
	if expected == types.VerificationPhase_VERIFICATION_PHASE_REVEAL &&
		len(round.Reveals) == len(round.Commits) &&
		uint64(len(round.Commits)) >= params.MinVerifiers {
		expected = types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION
	}

	// Phase monotonicity: never regress behind the stored phase.
	if expected < round.Phase {
		return round.Phase
	}
	return expected
}

// performAggregation aggregates votes and completes the round.
func (k Keeper) performAggregation(ctx context.Context, round *types.VerificationRound) error {
	result, err := k.AggregateVerificationResult(ctx, round)
	if err != nil {
		return err
	}
	return k.CompleteRound(ctx, round, result)
}
