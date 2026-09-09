package keeper

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/gov/types"
)

// BeginBlocker processes automatic stage transitions and tally resolution.
func (k Keeper) BeginBlocker(ctx sdk.Context) {
	currentHeight := uint64(ctx.BlockHeight())
	params := k.GetParams(ctx)

	// 1. Auto-advance "review" LIPs to "last_call" after review_blocks.
	reviewLIPs := k.GetLIPsByStatus(ctx, types.StatusReview)
	for _, lip := range reviewLIPs {
		catCfg := types.GetCategoryConfig(params, lip.Category)
		reviewBlocks := uint64(0)
		if catCfg != nil {
			reviewBlocks = catCfg.ReviewBlocks
		}
		if reviewBlocks > ^uint64(0)-lip.ReviewStartedBlock {
			k.Logger(ctx).Error("refusing LIP transition with overflowing review end height",
				"lip_id", lip.Id,
			)
			continue
		}
		reviewEnd := lip.ReviewStartedBlock + reviewBlocks
		if currentHeight >= reviewEnd {
			lip.Stage = types.StatusLastCall
			lip.LastCallStartedBlock = currentHeight
			k.SetLIP(ctx, lip)

			ctx.EventManager().EmitEvent(
				sdk.NewEvent("zerone.gov.lip_stage_transition",
					sdk.NewAttribute("lip_id", lip.Id),
					sdk.NewAttribute("from_stage", types.StatusReview),
					sdk.NewAttribute("to_stage", types.StatusLastCall),
					sdk.NewAttribute("block_height", fmt.Sprintf("%d", currentHeight)),
				),
			)
		}
	}

	// 2. Auto-advance "last_call" LIPs to "voting" after discussion_period_blocks.
	lastCallLIPs := k.GetLIPsByStatus(ctx, types.StatusLastCall)
	for _, lip := range lastCallLIPs {
		if params.DiscussionPeriodBlocks > ^uint64(0)-lip.LastCallStartedBlock {
			k.Logger(ctx).Error("refusing LIP transition with overflowing discussion end height",
				"lip_id", lip.Id,
			)
			continue
		}
		transitionHeight := lip.LastCallStartedBlock + params.DiscussionPeriodBlocks
		if currentHeight >= transitionHeight {
			votingPeriod := k.getEffectiveVotingPeriod(ctx, lip, params)
			if votingPeriod > ^uint64(0)-currentHeight {
				k.Logger(ctx).Error("refusing LIP voting transition with overflowing end height",
					"lip_id", lip.Id,
				)
				continue
			}
			votingEnd := currentHeight + votingPeriod
			if err := k.validateUpgradePlanForVoting(ctx, lip, votingEnd); err != nil {
				if currentHeight == transitionHeight {
					ctx.EventManager().EmitEvent(
						sdk.NewEvent("zerone.gov.upgrade_voting_blocked",
							sdk.NewAttribute("lip_id", lip.Id),
							sdk.NewAttribute("reason", err.Error()),
							sdk.NewAttribute("prospective_voting_end_block", fmt.Sprintf("%d", votingEnd)),
						),
					)
				}
				continue
			}
			lip.Stage = types.StatusVoting
			lip.VotingEndBlock = votingEnd
			k.SetLIP(ctx, lip)

			ctx.EventManager().EmitEvent(
				sdk.NewEvent("zerone.gov.lip_stage_transition",
					sdk.NewAttribute("lip_id", lip.Id),
					sdk.NewAttribute("from_stage", types.StatusLastCall),
					sdk.NewAttribute("to_stage", types.StatusVoting),
					sdk.NewAttribute("block_height", fmt.Sprintf("%d", currentHeight)),
					sdk.NewAttribute("voting_end_block", fmt.Sprintf("%d", lip.VotingEndBlock)),
				),
			)
		}
	}

	// 3. Tally expired voting LIPs.
	votingLIPs := k.GetLIPsByStatus(ctx, types.StatusVoting)
	for _, lip := range votingLIPs {
		if lip.VotingEndBlock > 0 && currentHeight >= lip.VotingEndBlock {
			k.tallyAndResolve(ctx, lip, params)
		}
	}

	// 4. Process research spend proposal expiry.
	k.ProcessResearchSpendExpiry(ctx, currentHeight)

	// 5. Process seat election stage transitions (nominated→expired, discussion→voting).
	k.ProcessSeatElectionExpiry(ctx, currentHeight)

	// 6. Tally expired seat elections.
	k.TallySeatElections(ctx)

	// 7. Check community seat term expiry.
	k.CheckSeatTermExpiry(ctx)

	// 8. Check for long-vacant seats.
	k.CheckSeatVacancy(ctx)

	// 9. Process pending phase transitions (activation delay + condition recheck).
	k.BeginBlockPhaseTransition(ctx)
}

// tallyAndResolve records passage only after the complete immediate action commits.
// A text LIP is an advisory decision; a phase LIP approves a separately recorded
// activation delay. Neither is a claim that a delayed target action executed.
func (k Keeper) tallyAndResolve(ctx sdk.Context, lip *types.LIP, params *types.Params) {
	if !k.AccountingSafetyEnabled(ctx) {
		k.tallyAndResolveLegacy(ctx, lip, params)
		return
	}
	var quorumMet, passed bool
	if types.IsPhaseTransitionCategory(lip.Category) {
		quorumMet, passed = k.checkQuorumAndSupermajority(ctx, lip, params)
	} else {
		quorumMet, passed = k.checkQuorumAndSupport(ctx, lip, params)
	}

	var executionErr error
	if quorumMet && passed {
		cacheCtx, write := ctx.CacheContext()
		executionErr = k.executeApprovedLIP(cacheCtx, lip)
		if executionErr == nil {
			lip.Stage = types.StatusPassed
			lip.ExecutionError = ""
			k.SetLIP(cacheCtx, lip)
			write()
		}
	}
	if !quorumMet || !passed || executionErr != nil {
		lip.Stage = types.StatusFailed
		lip.ExecutionError = executionErrorCode(executionErr)
		k.SetLIP(ctx, lip)
		if executionErr != nil {
			ctx.EventManager().EmitEvent(
				sdk.NewEvent("zerone.gov.lip_execution_failed",
					sdk.NewAttribute("lip_id", lip.Id),
					sdk.NewAttribute("category", lip.Category),
					sdk.NewAttribute("reason", lip.ExecutionError),
				),
			)
			// Keep the existing category failure events for consumers. Successful
			// handler events live only in the discarded cache on this path.
			if lip.Category == types.CategoryUpgrade {
				ctx.EventManager().EmitEvent(sdk.NewEvent("zerone.gov.upgrade_schedule_failed",
					sdk.NewAttribute("lip_id", lip.Id),
					sdk.NewAttribute("reason", lip.ExecutionError)))
			}
			if lip.Category == types.CategoryParameter {
				failureEvent := sdk.NewEvent("zerone.gov.param_change_failed",
					sdk.NewAttribute("lip_id", lip.Id),
					sdk.NewAttribute("reason", lip.ExecutionError),
				)
				if pe, ok := executionErr.(*paramExecutionError); ok {
					failureEvent = failureEvent.AppendAttributes(sdk.NewAttribute("module", pe.module), sdk.NewAttribute("key", pe.key))
				}
				ctx.EventManager().EmitEvent(failureEvent)
			}
			k.Logger(ctx).Error("approved LIP execution failed; target state discarded",
				"lip_id", lip.Id, "error", executionErr)
		}
		if types.IsPhaseTransitionCategory(lip.Category) {
			// A corrupt active LIP must not overwrite an already approved or
			// terminal target record. Only its still-unapproved metadata fails.
			if meta, found := k.GetPhaseTransitionMeta(ctx, lip.Id); found &&
				meta.LipID == lip.Id && meta.Stage == types.PhaseTransitionStagePending &&
				meta.ActivationBlock == 0 && meta.IsRollback == (lip.Category == types.CategoryPhaseRollback) {
				k.HandlePhaseTransitionFail(ctx, lip.Id)
			}
		}
	}

	// Legacy aggregate escrow stays unchanged: this store does not identify
	// individual contributors, so resolving a LIP cannot infer a refund owner.
	ctx.EventManager().EmitEvent(
		sdk.NewEvent("zerone.gov.lip_tallied",
			sdk.NewAttribute("lip_id", lip.Id),
			sdk.NewAttribute("outcome", lip.Stage),
			sdk.NewAttribute("yes_stake", lip.YesStake),
			sdk.NewAttribute("no_stake", lip.NoStake),
			sdk.NewAttribute("abstain_stake", lip.AbstainStake),
			sdk.NewAttribute("unique_voters", fmt.Sprintf("%d", lip.UniqueVoters)),
			sdk.NewAttribute("quorum_met", fmt.Sprintf("%t", quorumMet)),
		),
	)
}

// executeApprovedLIP runs only inside the caller's cached context. It grants no
// new dispatch authority and never replays an already-terminal historical LIP.
func (k Keeper) executeApprovedLIP(ctx sdk.Context, lip *types.LIP) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			// Panic values may contain process addresses or other nondeterministic
			// details. Keep the consensus-visible failure reason fixed.
			err = lipExecutionFailure("execution_panicked", nil)
		}
	}()

	switch lip.Category {
	case types.CategoryText:
		return nil // Advisory approval has no target mutation.
	case types.CategoryParameter:
		return k.executeParamChanges(ctx, lip)
	case types.CategoryUpgrade:
		// SDK governance remains the sole executable software-upgrade authority.
		_, err := k.scheduleApprovedUpgrade(ctx, lip)
		if err != nil {
			return lipExecutionFailure("custom_upgrade_authority_retired", err)
		}
		return nil
	case types.CategoryPhaseTransition, types.CategoryPhaseRollback:
		if err := k.HandlePhaseTransitionPass(ctx, lip.Id); err != nil {
			return lipExecutionFailure("phase_approval_failed", err)
		}
		return nil
	case types.CategoryCreedAmendment:
		pin, found := k.GetCreedAmendmentPin(ctx, lip.Id)
		if !found {
			return lipExecutionFailure("creed_payload_missing", nil)
		}
		ck := k.GetCreedKeeper()
		if ck == nil {
			return lipExecutionFailure("creed_keeper_missing", nil)
		}
		if err := ck.AnchorPinFromBytes(ctx, lip.Id, pin.CanonicalHash, pin.CommitmentsJSON); err != nil {
			return lipExecutionFailure("creed_anchor_failed", err)
		}
		ctx.EventManager().EmitEvent(sdk.NewEvent("zerone.gov.creed_amendment_anchored",
			sdk.NewAttribute("lip_id", lip.Id),
			sdk.NewAttribute("canonical_hash", fmt.Sprintf("%x", pin.CanonicalHash)),
			sdk.NewAttribute("creed_commitment", "10,19")))
		return nil
	case types.CategoryAdapterRegistration:
		return lipExecutionFailure("adapter_dispatch_unimplemented", nil)
	case types.CategoryResearchSpend, types.CategorySeatElection:
		return lipExecutionFailure("category_dispatch_unimplemented", nil)
	default:
		return lipExecutionFailure("category_unsupported", nil)
	}
}

// checkQuorumAndSupport checks quorum and support thresholds on 1,000,000 BPS scale.
func (k Keeper) checkQuorumAndSupport(ctx sdk.Context, lip *types.LIP, params *types.Params) (quorumMet bool, passed bool) {
	yesBig, _ := new(big.Int).SetString(lip.YesStake, 10)
	if yesBig == nil {
		yesBig = big.NewInt(0)
	}
	noBig, _ := new(big.Int).SetString(lip.NoStake, 10)
	if noBig == nil {
		noBig = big.NewInt(0)
	}
	abstainBig, _ := new(big.Int).SetString(lip.AbstainStake, 10)
	if abstainBig == nil {
		abstainBig = big.NewInt(0)
	}

	totalVoted := new(big.Int).Add(yesBig, noBig)
	totalVoted.Add(totalVoted, abstainBig)

	// Get total bonded stake.
	totalBonded := big.NewInt(0)
	if k.stakingKeeper != nil {
		bondedStr, err := k.stakingKeeper.GetTotalBondedStake(ctx)
		if err == nil {
			if tb, ok := new(big.Int).SetString(bondedStr, 10); ok {
				totalBonded = tb
			}
		}
	}

	// Quorum check: (totalVoted * 1_000_000) / totalBonded >= quorumThresholdBps
	if totalBonded.Sign() > 0 {
		actualBps := new(big.Int).Mul(totalVoted, big.NewInt(int64(types.BPSScale)))
		actualBps.Div(actualBps, totalBonded)
		quorumMet = actualBps.Uint64() >= params.QuorumThresholdBps
	}

	// Support check: (yesStake * 1_000_000) / (yesStake + noStake) >= supportThresholdBps
	yesNoTotal := new(big.Int).Add(yesBig, noBig)
	if yesNoTotal.Sign() > 0 {
		supportBps := new(big.Int).Mul(yesBig, big.NewInt(int64(types.BPSScale)))
		supportBps.Div(supportBps, yesNoTotal)
		passed = quorumMet && supportBps.Uint64() >= params.SupportThresholdBps
	}

	return quorumMet, passed
}

// checkQuorumAndSupermajority checks quorum and a 66.7% supermajority threshold
// for phase transition proposals.
func (k Keeper) checkQuorumAndSupermajority(ctx sdk.Context, lip *types.LIP, params *types.Params) (quorumMet bool, passed bool) {
	yesBig, _ := new(big.Int).SetString(lip.YesStake, 10)
	if yesBig == nil {
		yesBig = big.NewInt(0)
	}
	noBig, _ := new(big.Int).SetString(lip.NoStake, 10)
	if noBig == nil {
		noBig = big.NewInt(0)
	}
	abstainBig, _ := new(big.Int).SetString(lip.AbstainStake, 10)
	if abstainBig == nil {
		abstainBig = big.NewInt(0)
	}

	totalVoted := new(big.Int).Add(yesBig, noBig)
	totalVoted.Add(totalVoted, abstainBig)

	// Get total bonded stake.
	totalBonded := big.NewInt(0)
	if k.stakingKeeper != nil {
		bondedStr, err := k.stakingKeeper.GetTotalBondedStake(ctx)
		if err == nil {
			if tb, ok := new(big.Int).SetString(bondedStr, 10); ok {
				totalBonded = tb
			}
		}
	}

	// Quorum check: same as standard (33.4%).
	if totalBonded.Sign() > 0 {
		actualBps := new(big.Int).Mul(totalVoted, big.NewInt(int64(types.BPSScale)))
		actualBps.Div(actualBps, totalBonded)
		quorumMet = actualBps.Uint64() >= params.QuorumThresholdBps
	}

	// Supermajority: 66.7% of non-abstain votes.
	yesNoTotal := new(big.Int).Add(yesBig, noBig)
	if yesNoTotal.Sign() > 0 {
		supportBps := new(big.Int).Mul(yesBig, big.NewInt(int64(types.BPSScale)))
		supportBps.Div(supportBps, yesNoTotal)
		passed = quorumMet && supportBps.Uint64() >= types.TransitionSupermajorityBps
	}

	return quorumMet, passed
}

// paramExecutionError retains deterministic failure coordinates outside the
// discarded execution cache without forwarding its misleading success events.
type paramExecutionError struct {
	module string
	key    string
	err    error
}

func (e *paramExecutionError) Error() string {
	return fmt.Sprintf("parameter %s.%s: %v", e.module, e.key, e.err)
}

// executeParamChanges stops at the first failure. The caller commits the entire
// bundle and its events only when every handler succeeds.
func (k Keeper) executeParamChanges(ctx sdk.Context, lip *types.LIP) error {
	if len(lip.ParamChanges) == 0 {
		return lipExecutionFailure("parameter_changes_missing", nil)
	}
	router := k.GetParamRouter()
	for _, pc := range lip.ParamChanges {
		if pc == nil || pc.Module == "" || pc.Key == "" {
			return lipExecutionFailure("parameter_change_malformed", nil)
		}
		if router == nil {
			return &paramExecutionError{module: pc.Module, key: pc.Key, err: fmt.Errorf("param router not set")}
		}
		if err := router.ApplyParamChange(ctx, pc.Module, pc.Key, pc.Value); err != nil {
			return &paramExecutionError{module: pc.Module, key: pc.Key, err: err}
		}
		ctx.EventManager().EmitEvent(sdk.NewEvent("zerone.gov.param_change_applied",
			sdk.NewAttribute("lip_id", lip.Id),
			sdk.NewAttribute("module", pc.Module),
			sdk.NewAttribute("key", pc.Key),
			sdk.NewAttribute("value", pc.Value)))
	}
	return nil
}
