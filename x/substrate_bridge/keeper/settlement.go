package keeper

import (
	"context"
	"fmt"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

// SettleAttestation applies the reward formula and either:
//   - SETTLED (full reward, lineage propagation triggered)
//   - PARTIAL (reduced reward, lineage propagation on the partial)
//   - REJECTED (slash; attestation closed; no lineage)
//
// Eager: called synchronously when an attestation reaches READY or
// when BeginBlocker detects timeout or rejection-threshold trip.
func (k Keeper) SettleAttestation(ctx context.Context, attestationID string) error {
	att, found := k.GetAttestation(ctx, attestationID)
	if !found {
		return types.ErrAttestationNotFound
	}
	if att.Status != types.AttestationStatus_ATTESTATION_STATUS_READY &&
		att.Status != types.AttestationStatus_ATTESTATION_STATUS_PARTIAL &&
		att.Status != types.AttestationStatus_ATTESTATION_STATUS_AWAITING_RESOLUTION {
		return types.ErrAttestationWrongStatus
	}

	params := k.GetParams(ctx)

	totalCount := uint32(len(att.Link.CitedFacts)) + uint32(len(att.Link.PendingClaims))
	if totalCount == 0 {
		// Witness-only link: nothing to reward, but the bond was honest
		// collateral and returns whole.
		return k.finalizeSettle(ctx, att, sdkmath.ZeroInt(), types.AttestationStatus_ATTESTATION_STATUS_SETTLED)
	}

	// Rejection threshold check.
	pendingTotal := att.VerifiedCount + att.RejectedCount
	if pendingTotal > 0 {
		rejectionRatioBps := uint32(att.RejectedCount) * 10000 / pendingTotal
		if rejectionRatioBps >= params.PendingClaimRejectionThresholdBps {
			return k.settleRejected(ctx, att, fmt.Sprintf("rejection ratio %d bps >= threshold %d bps",
				rejectionRatioBps, params.PendingClaimRejectionThresholdBps))
		}
	}

	// Compute verified ratio.
	verifiedNumerator := uint32(att.VerifiedCount) + uint32(len(att.Link.CitedFacts))
	verifiedRatioBps := verifiedNumerator * 10000 / totalCount

	// Min-verified-ratio check.
	if verifiedRatioBps < params.MinVerifiedRatioForSettleBps {
		return k.settleRejected(ctx, att, fmt.Sprintf("verified ratio %d bps < min %d bps",
			verifiedRatioBps, params.MinVerifiedRatioForSettleBps))
	}

	// Compute reward.
	reward := k.computeReward(att, verifiedRatioBps, params)

	// Status: PARTIAL if any rejected; otherwise SETTLED.
	finalStatus := types.AttestationStatus_ATTESTATION_STATUS_SETTLED
	if att.RejectedCount > 0 {
		finalStatus = types.AttestationStatus_ATTESTATION_STATUS_PARTIAL
	}

	return k.finalizeSettle(ctx, att, reward, finalStatus)
}

// computeReward applies R = base + L × W × Q, with L scaled by verifiedRatio.
// Phase 0 simplification: Q is a fixed 0.5 (5000 bps) since x/knowledge doesn't
// yet expose per-round consensus_margin via the expected-keepers interface.
// Plan 1 (x/work) or a future task will refine Q with real consensus data.
func (k Keeper) computeReward(att *types.ExternalAttestation, verifiedRatioBps uint32, params types.Params) sdkmath.Int {
	base, _ := sdkmath.NewIntFromString(params.AttestationMinBondUzrn)
	// L proxy: base × verifiedRatio
	L := base.Mul(sdkmath.NewIntFromUint64(uint64(verifiedRatioBps))).Quo(sdkmath.NewInt(10000))
	// W proxy: sum of axis projections (Phase 0; future: per-axis weighted sum)
	wTotal := sdkmath.ZeroInt()
	if att.Link != nil && att.Link.RecursionWeight != nil {
		w := att.Link.RecursionWeight
		for _, v := range []uint64{w.AxisSubstrate, w.AxisVerification, w.AxisClassification, w.AxisAttribution, w.AxisTooling, w.AxisInterface} {
			wTotal = wTotal.Add(sdkmath.NewIntFromUint64(v))
		}
	}
	// Q: fixed 0.5 at Phase 0.
	Q := sdkmath.NewInt(5000)
	// R = base + L × W × Q / (10000^2) — keep units in uzrn.
	prod := L.Mul(wTotal).Mul(Q).Quo(sdkmath.NewInt(10000)).Quo(sdkmath.NewInt(10000))
	return base.Add(prod)
}

// settleRejected closes the attestation as REJECTED and burns the slashed
// bond (M1 fraud tier). Burning frees supply-cap headroom — slashed
// dishonesty becomes future emission room instead of dead weight in the
// module escrow. Atomic: state and coins move together or not at all.
func (k Keeper) settleRejected(ctx context.Context, att *types.ExternalAttestation, reason string) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	cacheCtx, writeCache := sdkCtx.CacheContext()

	att.Status = types.AttestationStatus_ATTESTATION_STATUS_REJECTED
	att.RejectionReason = reason
	att.SettledAtBlock = uint64(sdkCtx.BlockHeight())
	att.SlashUzrn = att.BondUzrn

	if k.bankKeeper != nil {
		if bond, ok := sdkmath.NewIntFromString(att.BondUzrn); ok && bond.IsPositive() {
			coins := sdk.NewCoins(sdk.NewCoin("uzrn", bond))
			if err := k.bankKeeper.BurnCoins(cacheCtx, types.ModuleName, coins); err != nil {
				return fmt.Errorf("burn slashed bond for %s: %w", att.AttestationId, err)
			}
		}
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		EventTypeExternalAttestationRejected,
		sdk.NewAttribute(AttrAttestationID, att.AttestationId),
		sdk.NewAttribute(AttrUsefulWorkCommitment, "UW"),
		sdk.NewAttribute(AttrMechanism, "M1"), // M1: full bond slash (fraud tier)
	))

	if err := k.WriteAttestation(cacheCtx, att); err != nil {
		return err
	}
	writeCache()
	return nil
}

// finalizeSettle closes a SETTLED or PARTIAL attestation. Two separate
// money movements, both atomic with the state write via a cache context
// (settlement runs from BeginBlocker and hooks, where a partial failure
// would otherwise persist a half-paid settle and retry into double-pays):
//
//  1. The escrowed bond returns whole to the submitter — it was honest
//     collateral, never payment.
//  2. The reward mints fresh through vesting_rewards.MintWithCap into the
//     audit bounty pool and pays out from there. Issuance follows
//     participation; rewards are never other submitters' escrowed bonds.
//     When the supply cap clips the mint, the attestation records what was
//     actually paid.
func (k Keeper) finalizeSettle(
	ctx context.Context,
	att *types.ExternalAttestation,
	reward sdkmath.Int,
	finalStatus types.AttestationStatus,
) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	cacheCtx, writeCache := sdkCtx.CacheContext()

	att.Status = finalStatus
	att.SettledAtBlock = uint64(sdkCtx.BlockHeight())

	submitterAddr, addrErr := sdk.AccAddressFromBech32(att.Submitter)
	if addrErr != nil {
		return fmt.Errorf("settle %s: bad submitter address: %w", att.AttestationId, addrErr)
	}

	// 1. Return the escrowed bond.
	if k.bankKeeper != nil {
		if bond, ok := sdkmath.NewIntFromString(att.BondUzrn); ok && bond.IsPositive() {
			coins := sdk.NewCoins(sdk.NewCoin("uzrn", bond))
			if err := k.bankKeeper.SendCoinsFromModuleToAccount(cacheCtx, types.ModuleName, submitterAddr, coins); err != nil {
				return fmt.Errorf("return bond for %s: %w", att.AttestationId, err)
			}
		}
	}

	// 2. Mint and pay the reward (cap-gated).
	paid := sdkmath.ZeroInt()
	if reward.GT(sdkmath.ZeroInt()) && k.vestingRewardsKeeper != nil && k.bankKeeper != nil {
		minted, err := k.vestingRewardsKeeper.MintWithCap(cacheCtx, types.AuditBountyPoolModuleName, reward.BigInt())
		if err != nil {
			return fmt.Errorf("mint reward for %s: %w", att.AttestationId, err)
		}
		paid = sdkmath.NewIntFromBigInt(minted)
		if paid.GT(sdkmath.ZeroInt()) {
			coins := sdk.NewCoins(sdk.NewCoin("uzrn", paid))
			if err := k.bankKeeper.SendCoinsFromModuleToAccount(cacheCtx, types.AuditBountyPoolModuleName, submitterAddr, coins); err != nil {
				return fmt.Errorf("pay reward for %s: %w", att.AttestationId, err)
			}
		}
	}
	att.RewardUzrn = paid.String()

	// Emit settlement event (SETTLED or PARTIAL; rejections go through
	// settleRejected).
	eventType := EventTypeExternalAttestationSettled
	mechanism := "M4" // M4: reward formula R = base + L × W × Q
	if finalStatus == types.AttestationStatus_ATTESTATION_STATUS_PARTIAL {
		eventType = EventTypeExternalAttestationPartial
		mechanism = "M1,M4" // M1 for slash, M4 for partial reward
	}
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		eventType,
		sdk.NewAttribute(AttrAttestationID, att.AttestationId),
		sdk.NewAttribute(AttrRewardUzrn, paid.String()),
		sdk.NewAttribute(AttrUsefulWorkCommitment, "UW"),
		sdk.NewAttribute(AttrMechanism, mechanism),
	))

	// Trigger lineage propagation on the amount actually paid.
	if paid.GT(sdkmath.ZeroInt()) {
		_ = k.PropagateLineage(cacheCtx, att.AttestationId, paid)
	}

	if err := k.WriteAttestation(cacheCtx, att); err != nil {
		return err
	}
	writeCache()
	return nil
}
