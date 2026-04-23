package keeper

import (
	"context"
	"fmt"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// ─── Route B Wave 8: the chain's heartbeat ───────────────────────────────
//
// ProcessRouteBLifecycle runs every block as part of BeginBlocker. It keeps
// the training infrastructure self-maintaining: expired bounty escrow
// returns automatically, vesting disbursements release when their window
// closes, and manifests that are superseded by newer finalized runs for
// the same pipeline get marked SUPERSEDED.
//
// Beauty at the system level: a correct system does the right thing
// without being reminded. These three processes, together, mean Route B
// progresses whether or not anyone happens to tx the right message.
//
// Cost: three prefix scans per block. Small today; if contention grows,
// move to time-indexed queues (block-height → expiry-set reverse index).

// ProcessRouteBLifecycle runs the three automatic lifecycle processes.
// Errors are logged rather than raised — consensus must not block on a
// transient state issue in a self-maintenance process.
func (k Keeper) ProcessRouteBLifecycle(ctx context.Context) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	height := uint64(sdkCtx.BlockHeight())
	k.Logger(ctx).Debug("route-b heartbeat", "height", height)

	// 1. Expire bounties whose expires_at_block has passed.
	k.expireBountiesBelow(ctx, height)

	// 2. Release vesting disbursements whose vesting_end_block has arrived.
	k.releaseMaturedVesting(ctx, height)

	// 3. Supersede older manifests when a newer FINALIZED exists for same pipeline.
	k.supersedeOlderManifests(ctx)
}

// ─── Bounty expiry ──────────────────────────────────────────────────────

// expireBountiesBelow returns escrow to sponsors for any active bounty
// whose expires_at_block has passed. Minus the kept-market-open fee.
func (k Keeper) expireBountiesBelow(ctx context.Context, height uint64) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	params, _ := k.GetParams(ctx)
	feeBps := params.AugmentationExpiryFeeBps
	if feeBps == 0 {
		feeBps = 30_000 // 3% default
	}

	var toExpire []*types.AugmentationBounty
	k.IterateAugmentationBounties(ctx, func(b *types.AugmentationBounty) bool {
		if b == nil || !b.Active {
			return false
		}
		if b.ExpiresAtBlock == 0 || b.ExpiresAtBlock > height {
			return false
		}
		toExpire = append(toExpire, b)
		return false
	})

	for _, b := range toExpire {
		remaining := k.GetEscrowLocked(ctx, b.Id)
		if remaining.IsPositive() {
			if err := k.ReturnEscrowToSponsor(ctx, b.Id, b.SponsorAddress, remaining, feeBps); err != nil {
				k.Logger(ctx).Error("bounty escrow return failed",
					"bounty_id", b.Id, "error", err.Error())
				continue
			}
		}
		b.Active = false
		if err := k.SetAugmentationBounty(ctx, b); err != nil {
			k.Logger(ctx).Error("bounty deactivation failed",
				"bounty_id", b.Id, "error", err.Error())
			continue
		}
		sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
			"zerone.knowledge.augmentation_bounty_expired",
			sdk.NewAttribute("bounty_id", b.Id),
			sdk.NewAttribute("sponsor", b.SponsorAddress),
			sdk.NewAttribute("refunded", remaining.String()),
			sdk.NewAttribute("fee_bps", fmt.Sprintf("%d", feeBps)),
		))
	}
}

// ─── Vesting release ────────────────────────────────────────────────────

// releaseMaturedVesting transfers the vesting portion of every disbursement
// whose vesting_end_block has arrived. Clawed-back disbursements are
// skipped (vesting_amount is preserved for audit).
func (k Keeper) releaseMaturedVesting(ctx context.Context, height uint64) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	var toRelease []*types.TrainingFundDisbursement
	k.IterateTrainingFundDisbursements(ctx, func(d *types.TrainingFundDisbursement) bool {
		if d == nil {
			return false
		}
		if d.VestingEndBlock == 0 || d.VestingEndBlock > height {
			return false
		}
		if d.VestingAmount == "" || d.VestingAmount == "0" {
			return false
		}
		if d.ClawedBackAtBlock > 0 {
			// Already clawed; vesting_amount kept for audit but not released.
			return false
		}
		toRelease = append(toRelease, d)
		return false
	})

	for _, d := range toRelease {
		amt, ok := sdkmath.NewIntFromString(d.VestingAmount)
		if !ok || amt.IsZero() {
			continue
		}
		claimantAddr, err := sdk.AccAddressFromBech32(d.Claimant)
		if err != nil {
			k.Logger(ctx).Error("vesting release failed: bad claimant",
				"disbursement_id", d.Id, "error", err.Error())
			continue
		}
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.TrainingFundModuleName, claimantAddr, sdk.NewCoins(sdk.NewCoin("uzrn", amt))); err != nil {
			k.Logger(ctx).Error("vesting release transfer failed",
				"disbursement_id", d.Id, "error", err.Error())
			continue
		}
		// Zero out the vesting amount so idempotency holds on future blocks.
		d.VestingAmount = "0"
		if err := k.SetTrainingFundDisbursement(ctx, d); err != nil {
			k.Logger(ctx).Error("vesting disbursement update failed",
				"disbursement_id", d.Id, "error", err.Error())
			continue
		}
		sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
			"zerone.knowledge.training_fund_vesting_released",
			sdk.NewAttribute("disbursement_id", d.Id),
			sdk.NewAttribute("model_id", d.ModelId),
			sdk.NewAttribute("claimant", d.Claimant),
			sdk.NewAttribute("amount", amt.String()),
		))
	}
}

// ─── Manifest supersession ──────────────────────────────────────────────

// supersedeOlderManifests scans FINALIZED + ATTESTED manifests per pipeline
// and marks older ones SUPERSEDED when a newer one exists. Keeps the
// "current bundle per pipeline" signal exact for trainers.
//
// Ordering: for each pipeline, the manifest with the greatest
// finalized_at_block is the canonical current; anything older is marked.
// ATTESTED manifests are never superseded by a mere FINALIZED peer —
// attestation is the binding commitment that a run actually happened.
func (k Keeper) supersedeOlderManifests(ctx context.Context) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Bucket by pipeline.
	byPipeline := make(map[string][]*types.TrainingManifest)
	k.IterateTrainingManifests(ctx, func(m *types.TrainingManifest) bool {
		if m == nil || m.PipelineId == "" {
			return false
		}
		if m.Status != types.ManifestStatus_MANIFEST_STATUS_FINALIZED &&
			m.Status != types.ManifestStatus_MANIFEST_STATUS_ATTESTED {
			return false
		}
		byPipeline[m.PipelineId] = append(byPipeline[m.PipelineId], m)
		return false
	})

	for _, group := range byPipeline {
		if len(group) < 2 {
			continue
		}
		// Find the latest: ATTESTED preferred over FINALIZED; then highest block.
		var latest *types.TrainingManifest
		for _, m := range group {
			if latest == nil || preferLater(m, latest) {
				latest = m
			}
		}
		for _, m := range group {
			if m == latest {
				continue
			}
			if m.Status == types.ManifestStatus_MANIFEST_STATUS_SUPERSEDED {
				continue
			}
			// ATTESTED manifests keep ATTESTED; only FINALIZED gets superseded.
			if m.Status == types.ManifestStatus_MANIFEST_STATUS_ATTESTED {
				continue
			}
			m.Status = types.ManifestStatus_MANIFEST_STATUS_SUPERSEDED
			if err := k.SetTrainingManifest(ctx, m); err != nil {
				k.Logger(ctx).Error("manifest supersession failed",
					"manifest_id", m.ManifestId, "error", err.Error())
				continue
			}
			sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
				"zerone.knowledge.training_manifest_superseded",
				sdk.NewAttribute("manifest_id", m.ManifestId),
				sdk.NewAttribute("pipeline_id", m.PipelineId),
				sdk.NewAttribute("superseded_by", latest.ManifestId),
			))
		}
	}
}

// preferLater returns true if a should replace b as "latest" for a pipeline.
// Rules: ATTESTED beats FINALIZED; among same-status, later finalized_at_block wins;
// ties broken deterministically by manifest_id lexicographic order.
func preferLater(a, b *types.TrainingManifest) bool {
	ar := statusRank(a.Status)
	br := statusRank(b.Status)
	if ar != br {
		return ar > br
	}
	if a.FinalizedAtBlock != b.FinalizedAtBlock {
		return a.FinalizedAtBlock > b.FinalizedAtBlock
	}
	return a.ManifestId > b.ManifestId
}

func statusRank(s types.ManifestStatus) int {
	switch s {
	case types.ManifestStatus_MANIFEST_STATUS_ATTESTED:
		return 2
	case types.ManifestStatus_MANIFEST_STATUS_FINALIZED:
		return 1
	}
	return 0
}
