package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/qualification/types"
)

// BeginBlocker processes qualification expiry, probation promotion, endorsement expiry,
// stake unlocking, and (Wave 16) accuracy-based decay.
func (k Keeper) BeginBlocker(ctx context.Context) error {
	neutral, err := k.reviewNeutralityEnabled(ctx)
	if err != nil {
		return err
	}
	singleRefund, err := k.SingleRefundEnabled(ctx)
	if err != nil {
		return err
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentBlock := uint64(sdkCtx.BlockHeight())
	params := k.GetParams(ctx)

	// Wave 16: accuracy-based decay scan. Skill is current, not
	// historical. Run periodically (not every block) to bound cost.
	// Iterates qualifications and transitions state when accuracy
	// crosses thresholds — closing the feedback loop the per-domain
	// panel writes into via RecordVerificationOutcome.
	if !neutral && params.DecayCheckIntervalBlocks > 0 && currentBlock > 0 &&
		currentBlock%params.DecayCheckIntervalBlocks == 0 {
		if err := k.RunAccuracyDecay(ctx, currentBlock, params); err != nil {
			return err
		}
	}

	k.IterateQualifications(ctx, func(q *types.DomainQualification) bool {
		switch q.Status {
		case types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE:
			// Check expiry.
			if q.ExpiresAt > 0 && currentBlock >= q.ExpiresAt {
				k.expireQualification(ctx, q, currentBlock, params, singleRefund)
			}

		case types.QualificationStatus_QUALIFICATION_STATUS_PROBATIONARY:
			// Check if probation period ended → promote to active or expire.
			if !neutral && q.ProbationUntil > 0 && currentBlock >= q.ProbationUntil {
				k.promoteProbationary(ctx, q, currentBlock, params)
			}
			// Also check overall expiry.
			if q.ExpiresAt > 0 && currentBlock >= q.ExpiresAt {
				k.expireQualification(ctx, q, currentBlock, params, singleRefund)
			}
		}
		return false
	})

	// Expire endorsements.
	k.IterateEndorsements(ctx, func(e *types.QualificationEndorsement) bool {
		if e.ExpiresAt > 0 && currentBlock >= e.ExpiresAt {
			// Decrement endorsement count on qualification.
			q, found := k.GetQualification(ctx, e.QualificationValidator, e.QualificationDomain)
			if found && q.EndorsementCount > 0 {
				q.EndorsementCount--
				k.SetQualification(ctx, q)
			}
			k.DeleteEndorsement(ctx, e)
		}
		return false
	})

	return nil
}

// expireQualification handles qualification expiry.
func (k Keeper) expireQualification(ctx context.Context, q *types.DomainQualification, currentBlock uint64, params *types.Params, singleRefund bool) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if !singleRefund {
		// Preserve the predecessor's writes, uncached refund and events until
		// the separately authorized carrier enables single-refund semantics.
		q.Status = types.QualificationStatus_QUALIFICATION_STATUS_EXPIRED
		k.SetQualification(ctx, q)
		if q.Pathway == types.QualificationPathway_QUALIFICATION_PATHWAY_STAKE && q.StakedAmount != "" {
			if currentBlock >= q.GrantedAt+params.StakeLockPeriod {
				_ = k.unlockStake(ctx, q.Validator, q.StakedAmount, false)
			}
		}
		sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
			"zerone.qualification.qualification_expired",
			sdk.NewAttribute("validator", q.Validator),
			sdk.NewAttribute("domain", q.Domain),
		))
		return
	}
	refundCtx, write := sdkCtx.CacheContext()

	q.Status = types.QualificationStatus_QUALIFICATION_STATUS_EXPIRED
	pendingReason := ""

	// Commit the refund and cleared liability together. A failed bank call may
	// already have written state or events; discard its cache, not the claim.
	if q.Pathway == types.QualificationPathway_QUALIFICATION_PATHWAY_STAKE && q.StakedAmount != "" {
		if currentBlock < q.GrantedAt+params.StakeLockPeriod {
			pendingReason = "stake_locked"
		} else if err := k.unlockStake(refundCtx, q.Validator, q.StakedAmount, true); err != nil {
			pendingReason = "transfer_failed"
			k.Logger(ctx).Error("qualification stake refund pending", "validator", q.Validator, "domain", q.Domain, "error", err)
		} else {
			q.StakedAmount = ""
		}
	}

	if pendingReason != "" {
		// Reuse the existing non-eligible, withdrawable SUSPENDED state for an
		// unpaid expiry. Retain the amount and deadline; withdrawal retries after
		// the lock ends. Do not retry from BeginBlock or mark it EXPIRED: legacy
		// EXPIRED records with an amount have an ambiguous refund history.
		q.Status = types.QualificationStatus_QUALIFICATION_STATUS_SUSPENDED
		q.ProbationUntil = 0
		k.SetQualification(ctx, q)
		sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
			"zerone.qualification.stake_refund_pending",
			sdk.NewAttribute("validator", q.Validator),
			sdk.NewAttribute("domain", q.Domain),
			sdk.NewAttribute("reason", pendingReason),
		))
	} else {
		k.SetQualification(refundCtx, q)
		write()
	}

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.qualification.qualification_expired",
			sdk.NewAttribute("validator", q.Validator),
			sdk.NewAttribute("domain", q.Domain),
		),
	)
}

// promoteProbationary promotes a probationary qualification to active if metrics meet threshold.
func (k Keeper) promoteProbationary(ctx context.Context, q *types.DomainQualification, _ uint64, params *types.Params) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Check if metrics are acceptable for promotion.
	if q.Metrics != nil && q.Metrics.TotalVerifications >= params.MinVerifications {
		accuracyBps := uint64(0)
		if q.Metrics.TotalVerifications > 0 {
			accuracyBps = (q.Metrics.CorrectVerifications * 1000000) / q.Metrics.TotalVerifications
		}
		if accuracyBps >= params.MinAccuracyBps {
			q.Status = types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE
			q.ProbationUntil = 0
			k.SetQualification(ctx, q)

			sdkCtx.EventManager().EmitEvent(
				sdk.NewEvent(
					"zerone.qualification.qualification_promoted",
					sdk.NewAttribute("validator", q.Validator),
					sdk.NewAttribute("domain", q.Domain),
				),
			)
			return
		}
	}

	// Failed to meet criteria → suspend.
	q.Status = types.QualificationStatus_QUALIFICATION_STATUS_SUSPENDED
	q.ProbationUntil = 0
	k.SetQualification(ctx, q)

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.qualification.qualification_suspended",
			sdk.NewAttribute("validator", q.Validator),
			sdk.NewAttribute("domain", q.Domain),
			sdk.NewAttribute("reason", "failed_probation"),
		),
	)
}

// unlockStake returns staked tokens to the validator.
func (k Keeper) unlockStake(ctx context.Context, validator string, amount string, singleRefund bool) error {
	if k.bankKeeper == nil {
		if !singleRefund {
			return nil // Historical nil-adapter behavior, before activation only.
		}
		return fmt.Errorf("bank keeper is not configured")
	}
	amt := new(big.Int)
	if _, ok := amt.SetString(amount, 10); !ok || amt.Sign() <= 0 {
		return fmt.Errorf("invalid stake amount: %s", amount)
	}
	recipientAddr, err := sdk.AccAddressFromBech32(validator)
	if err != nil {
		return fmt.Errorf("invalid validator address: %w", err)
	}
	coins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(amt)))
	return k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, recipientAddr, coins)
}
