package keeper

// Frozen pre-accounting-authority-v1 behavior. It is retained only so an old
// state cannot adopt consensus changes before the explicit activation marker.
// New native genesis and the separately owned migration enable the repaired
// execution path; historical terminal records are never replayed.

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/zerone-chain/zerone/x/gov/types"
)

// tallyAndResolve tallies votes and sets the LIP to passed or failed.
func (k Keeper) tallyAndResolveLegacy(ctx sdk.Context, lip *types.LIP, params *types.Params) {
	// Phase transition categories use supermajority (66.7%), others use standard (50%).
	var quorumMet, passed bool
	if types.IsPhaseTransitionCategory(lip.Category) {
		quorumMet, passed = k.checkQuorumAndSupermajority(ctx, lip, params)
	} else {
		quorumMet, passed = k.checkQuorumAndSupport(ctx, lip, params)
	}

	var scheduledPlan *types.UpgradePlan
	var scheduleErr error
	if quorumMet && passed && lip.Category == types.CategoryUpgrade {
		scheduledPlan, scheduleErr = k.scheduleApprovedUpgrade(ctx, lip)
	}

	if quorumMet && passed && scheduleErr == nil {
		lip.Stage = types.StatusPassed
		k.SetLIP(ctx, lip)

		// Category-specific post-pass handling.
		switch lip.Category {
		case types.CategoryParameter:
			if len(lip.ParamChanges) > 0 {
				k.executeParamChangesLegacy(ctx, lip)
			}
		case types.CategoryUpgrade:
			ctx.EventManager().EmitEvent(
				sdk.NewEvent("zerone.gov.upgrade_scheduled",
					sdk.NewAttribute("lip_id", lip.Id),
					sdk.NewAttribute("upgrade_name", scheduledPlan.Name),
					sdk.NewAttribute("height", fmt.Sprintf("%d", scheduledPlan.Height)),
				),
			)
			k.Logger(ctx).Info("software upgrade scheduled via LIP governance",
				"lip_id", lip.Id,
				"upgrade_name", scheduledPlan.Name,
				"height", scheduledPlan.Height,
			)
		case types.CategoryPhaseTransition, types.CategoryPhaseRollback:
			// Phase transitions don't execute immediately — enter activation delay.
			k.handlePhaseTransitionPassLegacy(ctx, lip.Id)
		case types.CategoryCreedAmendment:
			// Commitment 19 (the creed is governance-gated): on
			// pass, ship the attached pin payload to x/creed via
			// AnchorPinFromBytes. The LIP id is recorded as the
			// source so the post-launch audit trail names the LIP
			// that authorized every creed amendment.
			if pin, found := k.GetCreedAmendmentPin(ctx, lip.Id); found {
				if ck := k.GetCreedKeeper(); ck != nil {
					if err := ck.AnchorPinFromBytes(ctx, lip.Id, pin.CanonicalHash, pin.CommitmentsJSON); err != nil {
						k.Logger(ctx).Error("failed to anchor creed amendment from passed LIP",
							"lip_id", lip.Id,
							"error", err,
						)
					} else {
						ctx.EventManager().EmitEvent(
							sdk.NewEvent("zerone.gov.creed_amendment_anchored",
								sdk.NewAttribute("lip_id", lip.Id),
								sdk.NewAttribute("canonical_hash", fmt.Sprintf("%x", pin.CanonicalHash)),
								sdk.NewAttribute("creed_commitment", "10,19"),
							),
						)
						k.Logger(ctx).Info("creed amendment anchored via LIP governance",
							"lip_id", lip.Id,
						)
					}
				}
			}
		case types.CategoryAdapterRegistration:
			// Commitment 20 (issuance follows participation): on pass,
			// dispatch to x/substrate_bridge.WriteAdapterFromGov with
			// the adapter spec that was attached to the LIP body.
			//
			// TODO(Phase-1): retrieve the adapter payload attached to
			// this LIP (analogous to GetCreedAmendmentPin for
			// CategoryCreedAmendment), then call:
			//
			//   sbk := k.GetSubstrateBridgeKeeper()
			//   if sbk != nil {
			//       adapterBytes := <retrieve from LIP attachment store>
			//       if err := sbk.WriteAdapterFromGov(ctx, lip.Id, adapterBytes); err != nil {
			//           k.Logger(ctx).Error(...)
			//       }
			//   }
			//
			// The attachment mechanism (MsgAttachAdapterRegistration +
			// SetAdapterRegistrationPayload/GetAdapterRegistrationPayload)
			// mirrors the creed-amendment pin pattern and will be wired
			// in the follow-up plan task when the generic LIP-dispatch
			// mechanism stabilises. Until then, the governance weight
			// and vocabulary are established: a passed LIP of this class
			// is recorded on-chain at the correct quorum bar.
			k.Logger(ctx).Info("adapter_registration LIP passed; dispatch to substrate_bridge pending Phase-1 wiring",
				"lip_id", lip.Id,
			)
			ctx.EventManager().EmitEvent(
				sdk.NewEvent("zerone.gov.adapter_registration_lip_passed",
					sdk.NewAttribute("lip_id", lip.Id),
					sdk.NewAttribute("creed_commitment", "20"),
					sdk.NewAttribute("dispatch_status", "pending_phase1_wiring"),
				),
			)
		}
	} else {
		lip.Stage = types.StatusFailed
		k.SetLIP(ctx, lip)

		if scheduleErr != nil {
			ctx.EventManager().EmitEvent(
				sdk.NewEvent("zerone.gov.upgrade_schedule_failed",
					sdk.NewAttribute("lip_id", lip.Id),
					sdk.NewAttribute("reason", scheduleErr.Error()),
				),
			)
			k.Logger(ctx).Error("approved upgrade LIP failed closed because scheduling failed",
				"lip_id", lip.Id,
				"error", scheduleErr,
			)
		}

		// Notify metadata of failure for phase transition categories.
		if types.IsPhaseTransitionCategory(lip.Category) {
			k.HandlePhaseTransitionFail(ctx, lip.Id)
		}
	}

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

// executeParamChanges applies parameter changes from a passed LIP.
func (k Keeper) executeParamChangesLegacy(ctx sdk.Context, lip *types.LIP) {
	logger := k.Logger(ctx)
	router := k.GetParamRouter()

	for _, pc := range lip.ParamChanges {
		if router == nil {
			logger.Error("param router not set, skipping param change",
				"lip_id", lip.Id, "module", pc.Module, "key", pc.Key,
			)
			ctx.EventManager().EmitEvent(
				sdk.NewEvent("zerone.gov.param_change_failed",
					sdk.NewAttribute("lip_id", lip.Id),
					sdk.NewAttribute("module", pc.Module),
					sdk.NewAttribute("key", pc.Key),
					sdk.NewAttribute("reason", "param router not set"),
				),
			)
			continue
		}

		if err := router.ApplyParamChange(ctx, pc.Module, pc.Key, pc.Value); err != nil {
			logger.Error("param change failed",
				"lip_id", lip.Id, "module", pc.Module, "key", pc.Key, "error", err,
			)
			ctx.EventManager().EmitEvent(
				sdk.NewEvent("zerone.gov.param_change_failed",
					sdk.NewAttribute("lip_id", lip.Id),
					sdk.NewAttribute("module", pc.Module),
					sdk.NewAttribute("key", pc.Key),
					sdk.NewAttribute("reason", err.Error()),
				),
			)
		} else {
			logger.Info("param change applied",
				"lip_id", lip.Id, "module", pc.Module, "key", pc.Key, "value", pc.Value,
			)
			ctx.EventManager().EmitEvent(
				sdk.NewEvent("zerone.gov.param_change_applied",
					sdk.NewAttribute("lip_id", lip.Id),
					sdk.NewAttribute("module", pc.Module),
					sdk.NewAttribute("key", pc.Key),
					sdk.NewAttribute("value", pc.Value),
				),
			)
		}
	}
}
func (k Keeper) handlePhaseTransitionPassLegacy(ctx sdk.Context, lipID string) {
	meta, found := k.GetPhaseTransitionMeta(ctx, lipID)
	if !found {
		return
	}

	currentHeight := uint64(ctx.BlockHeight())
	meta.ActivationBlock = currentHeight + types.TransitionActivationDelay
	// Stage remains pending_activation; BeginBlocker will execute it.
	k.SetPhaseTransitionMeta(ctx, meta)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.gov.phase_transition_passed",
			sdk.NewAttribute("lip_id", lipID),
			sdk.NewAttribute("target_phase", fmt.Sprint(meta.TargetPhase)),
			sdk.NewAttribute("activation_block", fmt.Sprintf("%d", meta.ActivationBlock)),
			sdk.NewAttribute("is_rollback", fmt.Sprintf("%t", meta.IsRollback)),
		),
	)
}
