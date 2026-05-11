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
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	totalCount := uint32(len(att.Link.CitedFacts)) + uint32(len(att.Link.PendingClaims))
	if totalCount == 0 {
		// Nothing to settle; close as SETTLED with base only.
		return k.finalizeSettle(ctx, att, sdkmath.ZeroInt(), types.AttestationStatus_ATTESTATION_STATUS_SETTLED)
	}

	// Rejection threshold check.
	pendingTotal := att.VerifiedCount + att.RejectedCount
	if pendingTotal > 0 {
		rejectionRatioBps := uint32(att.RejectedCount) * 10000 / pendingTotal
		if rejectionRatioBps >= params.PendingClaimRejectionThresholdBps {
			att.Status = types.AttestationStatus_ATTESTATION_STATUS_REJECTED
			att.RejectionReason = fmt.Sprintf("rejection ratio %d bps >= threshold %d bps",
				rejectionRatioBps, params.PendingClaimRejectionThresholdBps)
			att.SettledAtBlock = uint64(sdkCtx.BlockHeight())
			// Full bond slash (M1 fraud tier).
			att.SlashUzrn = att.BondUzrn
			return k.WriteAttestation(ctx, att)
		}
	}

	// Compute verified ratio.
	verifiedNumerator := uint32(att.VerifiedCount) + uint32(len(att.Link.CitedFacts))
	verifiedRatioBps := verifiedNumerator * 10000 / totalCount

	// Min-verified-ratio check.
	if verifiedRatioBps < params.MinVerifiedRatioForSettleBps {
		att.Status = types.AttestationStatus_ATTESTATION_STATUS_REJECTED
		att.RejectionReason = fmt.Sprintf("verified ratio %d bps < min %d bps", verifiedRatioBps, params.MinVerifiedRatioForSettleBps)
		att.SettledAtBlock = uint64(sdkCtx.BlockHeight())
		att.SlashUzrn = att.BondUzrn
		return k.WriteAttestation(ctx, att)
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

func (k Keeper) finalizeSettle(
	ctx context.Context,
	att *types.ExternalAttestation,
	reward sdkmath.Int,
	finalStatus types.AttestationStatus,
) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	att.Status = finalStatus
	att.SettledAtBlock = uint64(sdkCtx.BlockHeight())
	att.RewardUzrn = reward.String()

	// Pay the submitter (release bond + reward).
	if k.bankKeeper != nil && reward.GT(sdkmath.ZeroInt()) {
		submitterAddr, err := sdk.AccAddressFromBech32(att.Submitter)
		if err == nil {
			coins := sdk.NewCoins(sdk.NewCoin("uzrn", reward))
			_ = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, submitterAddr, coins)
		}
	}

	// Trigger lineage propagation if this is a paid settle (not REJECTED).
	if finalStatus != types.AttestationStatus_ATTESTATION_STATUS_REJECTED && reward.GT(sdkmath.ZeroInt()) {
		_ = k.PropagateLineage(ctx, att.AttestationId, reward)
	}

	return k.WriteAttestation(ctx, att)
}
