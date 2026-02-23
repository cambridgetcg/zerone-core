package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/autopoiesis/types"
)

// CollectAndAdapt is called from EndBlocker. It checks if an epoch boundary
// has been reached and, if so, collects cross-module metrics, computes SSI,
// and adjusts multipliers toward their targets.
func (k Keeper) CollectAndAdapt(ctx context.Context) {
	if !k.IsActive(ctx) {
		return
	}

	params := k.GetParams(ctx)
	if !params.Enabled {
		return
	}

	// Check emergency halt — skip adjustment during halts.
	if k.emergencyKeeper != nil && k.emergencyKeeper.IsHalted(ctx) {
		return
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockHeight := uint64(sdkCtx.BlockHeight())

	state := k.GetState(ctx)
	if state.LastEpochHeight == 0 {
		// First block after activation — set baseline.
		state.LastEpochHeight = blockHeight
		k.SetState(ctx, state)
		return
	}

	// Check epoch boundary.
	if blockHeight-state.LastEpochHeight < params.EpochLengthBlocks {
		return
	}

	// ---- Epoch boundary reached ----
	state.CurrentEpoch++
	state.LastEpochHeight = blockHeight

	// Collect metrics.
	stakingParticipation := k.collectStakingParticipation(ctx)
	verificationRate := k.collectVerificationRate(ctx)
	isHalted := k.emergencyKeeper != nil && k.emergencyKeeper.IsHalted(ctx)

	// Compute SSI.
	ssi := types.ComputeSSI(stakingParticipation, verificationRate, isHalted)
	category := types.ClassifySSI(ssi, params)
	k.SetSSI(ctx, ssi)

	// Adjust each multiplier.
	multipliers := k.GetAllMultipliers(ctx)
	var snapshotMultipliers []*types.MultiplierState
	for _, ms := range multipliers {
		if k.IsMultiplierFrozen(ctx, ms.Path) {
			ms.Frozen = true
			snapshotMultipliers = append(snapshotMultipliers, ms)
			continue
		}

		target := types.ComputeTarget(ssi, ms.Path)
		ms.TargetBps = target
		ms.CurrentBps = adjustMultiplier(ms.CurrentBps, target, params.MaxChangePerEpochBps, ms.MinBps, ms.MaxBps)
		ms.LastUpdated = blockHeight

		k.SetMultiplierState(ctx, ms)
		snapshotMultipliers = append(snapshotMultipliers, ms)
	}

	// Store snapshot.
	snapshot := &types.EpochSnapshot{
		Epoch:       state.CurrentEpoch,
		BlockHeight: blockHeight,
		Multipliers: snapshotMultipliers,
		SsiScore:    ssi,
		SsiCategory: category,
	}
	k.SetEpochSnapshot(ctx, snapshot)

	k.SetState(ctx, state)

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent("zerone.autopoiesis.epoch_processed",
			sdk.NewAttribute("epoch", fmt.Sprintf("%d", state.CurrentEpoch)),
			sdk.NewAttribute("ssi_score", fmt.Sprintf("%d", ssi)),
			sdk.NewAttribute("ssi_category", category),
			sdk.NewAttribute("block_height", fmt.Sprintf("%d", blockHeight)),
			sdk.NewAttribute("multiplier_count", fmt.Sprintf("%d", len(snapshotMultipliers))),
		),
	)

	k.Logger(ctx).Info("epoch processed",
		"epoch", state.CurrentEpoch,
		"ssi", ssi,
		"category", category,
		"block", blockHeight,
	)
}

// collectStakingParticipation computes staking participation as a BPS value.
// Returns the fraction of total supply that is bonded (simplified: uses
// bonded stake as a proportion of a reference supply).
func (k Keeper) collectStakingParticipation(ctx context.Context) uint64 {
	if k.stakingKeeper == nil {
		return types.BPSScale // assume healthy if no staking data
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	totalBonded := k.stakingKeeper.GetTotalBondedStake(sdkCtx)
	if totalBonded == nil || totalBonded.Sign() == 0 {
		return 0
	}

	activeCount := k.stakingKeeper.GetActiveValidatorCount(sdkCtx)
	if activeCount == 0 {
		return 0
	}

	// Heuristic: participation = min(1.0, activeValidators / 21) * BPSScale
	// This is simplified — in production, use bonded/total supply ratio.
	target := uint64(21) // target validator set size
	participation := uint64(activeCount) * types.BPSScale / target
	if participation > types.BPSScale {
		participation = types.BPSScale
	}
	return participation
}

// collectVerificationRate gets the knowledge verification rate.
func (k Keeper) collectVerificationRate(ctx context.Context) uint64 {
	if k.knowledgeKeeper == nil {
		return types.BPSScale // assume healthy if no knowledge data
	}
	rate := k.knowledgeKeeper.GetVerificationRate(ctx)
	if rate > types.BPSScale {
		rate = types.BPSScale
	}
	return rate
}

// adjustMultiplier moves current toward target by at most maxChange per step,
// clamped to [min, max].
func adjustMultiplier(current, target, maxChange, min, max uint64) uint64 {
	var result uint64

	if target > current {
		delta := target - current
		if delta > maxChange {
			delta = maxChange
		}
		result = current + delta
	} else if target < current {
		delta := current - target
		if delta > maxChange {
			delta = maxChange
		}
		result = current - delta
	} else {
		result = current
	}

	// Clamp to bounds.
	if result < min {
		result = min
	}
	if result > max {
		result = max
	}
	return result
}

// bigIntToBPS converts a big.Int ratio to BPS (used internally, exported for tests).
func bigIntToBPS(numerator, denominator *big.Int) uint64 {
	if denominator == nil || denominator.Sign() == 0 {
		return 0
	}
	// (numerator * BPSScale) / denominator
	scaled := new(big.Int).Mul(numerator, new(big.Int).SetUint64(types.BPSScale))
	result := new(big.Int).Div(scaled, denominator)
	if !result.IsUint64() {
		return types.BPSScale
	}
	bps := result.Uint64()
	if bps > types.BPSScale {
		bps = types.BPSScale
	}
	return bps
}
