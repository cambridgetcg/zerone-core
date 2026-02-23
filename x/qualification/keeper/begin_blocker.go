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
// and stake unlocking.
func (k Keeper) BeginBlocker(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentBlock := uint64(sdkCtx.BlockHeight())
	params := k.GetParams(ctx)

	k.IterateQualifications(ctx, func(q *types.DomainQualification) bool {
		switch q.Status {
		case types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE:
			// Check expiry.
			if q.ExpiresAt > 0 && currentBlock >= q.ExpiresAt {
				k.expireQualification(ctx, q, currentBlock, params)
			}

		case types.QualificationStatus_QUALIFICATION_STATUS_PROBATIONARY:
			// Check if probation period ended → promote to active or expire.
			if q.ProbationUntil > 0 && currentBlock >= q.ProbationUntil {
				k.promoteProbationary(ctx, q, currentBlock, params)
			}
			// Also check overall expiry.
			if q.ExpiresAt > 0 && currentBlock >= q.ExpiresAt {
				k.expireQualification(ctx, q, currentBlock, params)
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
func (k Keeper) expireQualification(ctx context.Context, q *types.DomainQualification, currentBlock uint64, params *types.Params) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	q.Status = types.QualificationStatus_QUALIFICATION_STATUS_EXPIRED
	k.SetQualification(ctx, q)

	// Unlock stake for stake-pathway qualifications.
	if q.Pathway == types.QualificationPathway_QUALIFICATION_PATHWAY_STAKE && q.StakedAmount != "" {
		if currentBlock >= q.GrantedAt+params.StakeLockPeriod {
			_ = k.unlockStake(ctx, q.Validator, q.StakedAmount)
		}
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
func (k Keeper) unlockStake(ctx context.Context, validator string, amount string) error {
	if k.bankKeeper == nil {
		return nil
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
