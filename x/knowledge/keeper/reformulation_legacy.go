package keeper

import (
	"context"
	"fmt"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Legacy bodies copied verbatim (only function names changed) from
// 77521a0daf14420836483ae9d8ea3fbcc1ae0d40, training_economics.go and
// route_b_heartbeat.go. In particular the vote tally is NOT shared with the
// repaired selector or hold census. Preserve historical gas, errors and effects.
func (k Keeper) recordAugmentationVoteLegacy(ctx context.Context, augID, verifier string, vote types.AugmentationVerdict) (bool, types.AugmentationVerdict, error) {
	if _, err := k.ReviewNeutralityEnabled(ctx); err != nil {
		return false, types.AugmentationVerdict_AUGMENTATION_VERDICT_PENDING, err
	}
	aug, ok := k.GetAugmentation(ctx, augID)
	if !ok {
		return false, types.AugmentationVerdict_AUGMENTATION_VERDICT_PENDING, fmt.Errorf("augmentation %s not found", augID)
	}
	if aug.Verdict != types.AugmentationVerdict_AUGMENTATION_VERDICT_PENDING {
		return false, aug.Verdict, fmt.Errorf("augmentation verdict is already final")
	}
	if vote == types.AugmentationVerdict_AUGMENTATION_VERDICT_PENDING {
		return false, aug.Verdict, fmt.Errorf("pending is not a valid vote")
	}

	// Guard: sponsor and submitter may not vote.
	if aug.Submitter == verifier {
		return false, aug.Verdict, fmt.Errorf("submitter may not vote on their own variant")
	}
	if aug.BountyId != "" {
		if bounty, ok := k.GetAugmentationBounty(ctx, aug.BountyId); ok && bounty.SponsorAddress == verifier {
			return false, aug.Verdict, fmt.Errorf("sponsor may not vote as verifier")
		}
	}

	// Dedup: one vote per verifier.
	for _, v := range aug.VerdictVoters {
		if v == verifier {
			return false, aug.Verdict, fmt.Errorf("verifier already voted")
		}
	}

	// Snapshot the verifier's effective stake at vote time. Non-validator
	// addresses return stake=0 (and nil error for not-found via the keeper
	// contract); zero-stake votes are still recorded for audit but carry
	// no weight in the consensus tally.
	var stakeAtVote uint64
	if k.stakingKeeper != nil {
		if s, err := k.stakingKeeper.GetEffectiveStake(ctx, verifier); err == nil {
			stakeAtVote = s
		}
	}

	// Snapshot the verifier's skill signal at vote time. Wave 15c: when
	// the target fact has a domain, use DOMAIN-SPECIFIC qualification as
	// the primary panel weight — a physics-domain augmentation should
	// be adjudicated by physics-qualified voters, and cross-domain
	// credentials earn no credit regardless of how strong they are
	// globally. Voters unqualified in the target domain are floored at
	// the tally's panelCalibrationFloorBps, preserving liveness (they
	// still vote, still carry some weight, but cannot dominate).
	//
	// Global calibration is used only as a fallback when:
	//   - the target fact has no domain (legacy / migration records), OR
	//   - no DomainQualificationKeeper is wired (genesis / early bootstrap).
	var calibrationAtVote uint64
	domain := ""
	if aug.OriginalFactId != "" {
		if origFact, ok := k.GetFact(ctx, aug.OriginalFactId); ok && origFact != nil {
			domain = origFact.Domain
		}
	}
	if domain != "" && k.domainQualificationKeeper != nil {
		// Domain path: ONLY domain qualification counts. Unqualified
		// voters get weight 0, which the tally floors to 20%.
		if w, err := k.domainQualificationKeeper.GetQualificationWeight(ctx, verifier, domain); err == nil && w > 0 {
			// x/qualification reports weight in the 1..100 range; scale
			// to BPS (0..1_000_000). Cap at BPS in case governance ever
			// raises the qualification scale past 100.
			scaled := w * 10_000
			if scaled > bps {
				scaled = bps
			}
			calibrationAtVote = scaled
		}
	} else {
		// Fallback path: no domain on target fact, or no qualification
		// adapter wired. Use global calibration.
		if cal, ok := k.GetAgentCalibration(ctx, verifier); ok && cal != nil {
			calibrationAtVote = cal.CalibrationScoreBps
		}
	}

	aug.VerdictVoters = append(aug.VerdictVoters, verifier)
	aug.VerdictVotes = append(aug.VerdictVotes, vote)
	aug.VerdictVoteStakes = append(aug.VerdictVoteStakes, stakeAtVote)
	aug.VerdictVoteCalibrationBps = append(aug.VerdictVoteCalibrationBps, calibrationAtVote)
	if err := k.SetAugmentation(ctx, aug); err != nil {
		return false, aug.Verdict, err
	}

	// Consensus check.
	params, err := k.GetParams(ctx)
	if err != nil {
		return false, aug.Verdict, err
	}
	minVotes := params.ReformulationMinPanelVotes
	if minVotes == 0 {
		minVotes = 3
	}
	if uint64(len(aug.VerdictVoters)) < minVotes {
		return false, aug.Verdict, nil
	}
	consensusBps := params.ReformulationConsensusBps
	if consensusBps == 0 {
		consensusBps = 666_000
	}

	// Reputation-weighted tally (Wave 15). Each voter's effective panel
	// weight is stake × max(floor, calibration) / BPS. Stake alone isn't
	// enough to dominate — a verifier who has shown they can't tell
	// truth from falsehood carries reduced weight regardless of bond.
	// The calibration floor prevents lockup in the early chain state
	// before anyone has built up a record; it also means NO stake-
	// bearing verifier ever drops to zero weight (stake × floor > 0).
	const panelCalibrationFloorBps uint64 = 200_000 // 20% — bare minimum signal credibility
	stakeTally := make(map[types.AugmentationVerdict]uint64)
	var totalStake uint64
	for i, v := range aug.VerdictVotes {
		stake := uint64(0)
		if i < len(aug.VerdictVoteStakes) {
			stake = aug.VerdictVoteStakes[i]
		}
		cal := uint64(0)
		if i < len(aug.VerdictVoteCalibrationBps) {
			cal = aug.VerdictVoteCalibrationBps[i]
		}
		if cal < panelCalibrationFloorBps {
			cal = panelCalibrationFloorBps
		}
		// Effective weight = stake × calibration / BPS. A 100M stake at
		// 50% calibration weighs the same as a 50M stake at 100%.
		effectiveWeight := safeMulDiv(stake, cal, bps)
		stakeTally[v] += effectiveWeight
		totalStake += effectiveWeight
	}
	if totalStake == 0 {
		// No stake-bearing votes yet; wait for more.
		return false, aug.Verdict, nil
	}
	var winner types.AugmentationVerdict
	var winnerStake uint64
	for v, s := range stakeTally {
		if s*bps/totalStake >= consensusBps && s > winnerStake {
			winner = v
			winnerStake = s
		}
	}
	if winner == types.AugmentationVerdict_AUGMENTATION_VERDICT_PENDING {
		return false, aug.Verdict, nil
	}
	// Finalize.
	return true, winner, nil
}

func (k Keeper) releaseAugmentationPayoutLegacy(ctx context.Context, bountyID, recipient string, amount sdkmath.Int) error {
	if amount.IsZero() {
		return nil
	}
	recipAddr, err := sdk.AccAddressFromBech32(recipient)
	if err != nil {
		return fmt.Errorf("invalid recipient address: %w", err)
	}
	coins := sdk.NewCoins(sdk.NewCoin("uzrn", amount))
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.TrainingFundModuleName, recipAddr, coins); err != nil {
		return err
	}
	return k.reduceEscrowLocked(ctx, bountyID, amount)
}

func (k Keeper) forfeitAugmentationDepositLegacy(ctx context.Context, bountyID string, amount sdkmath.Int) error {
	if amount.IsZero() {
		return nil
	}
	coins := sdk.NewCoins(sdk.NewCoin("uzrn", amount))
	// Send from training fund → module-level deposit; reuse research fund
	// when the vesting_rewards keeper is present.
	if k.vestingRewardsKeeper != nil {
		if err := k.vestingRewardsKeeper.DepositToResearchFund(ctx, types.TrainingFundModuleName, coins); err != nil {
			return err
		}
	} else {
		// Fallback: keep in training fund as accumulated reserve.
	}
	return k.reduceEscrowLocked(ctx, bountyID, amount)
}

func (k Keeper) expireBountiesBelowLegacy(ctx context.Context, height uint64) {
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
