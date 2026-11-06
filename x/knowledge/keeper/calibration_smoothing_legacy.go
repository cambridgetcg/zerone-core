package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Frozen bodies from bc2a58fccb3e9bbb43647b0e51490fad034f34fd (also
// 6a56b621). Only producer names changed. The public historical formula stays
// here so RecomputeAllCalibrationScores and predecessor upgrade replay do not
// silently adopt v2's formula. These bodies never call the smoothed formula.
func (k Keeper) recordSubmissionOutcomeLegacy(
	ctx context.Context,
	submitter string,
	methodId string,
	verdict types.Verdict,
) error {
	neutral, err := k.ReviewNeutralityEnabled(ctx)
	if err != nil || neutral {
		return err
	}

	if submitter == "" {
		return nil
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	height := uint64(sdkCtx.BlockHeight())

	c := k.getOrInitCalibration(ctx, submitter)
	if c.FirstSubmissionBlock == 0 {
		c.FirstSubmissionBlock = height
	}
	c.LastSubmissionBlock = height
	c.TotalSubmissions++

	stats := ensureMethodStats(c, methodId)
	stats.Submissions++

	switch verdict {
	case types.Verdict_VERDICT_ACCEPT:
		c.Accepted++
		stats.Accepted++
	case types.Verdict_VERDICT_REJECT:
		c.Rejected++
		stats.Rejected++
	case types.Verdict_VERDICT_MALFORMED:
		c.Malformed++
	case types.Verdict_VERDICT_INCONCLUSIVE:
		c.Inconclusive++
	}

	c.CalibrationScoreBps = ComputeAgentCalibrationScore(c)
	c.LastUpdatedBlock = height
	_ = k.SetAgentCalibration(ctx, c)
	k.EmitCalibrationUpdated(ctx, c)
	return nil
}

func (k Keeper) recordCorroborationForSubmitterLegacy(
	ctx context.Context,
	submitter string,
	methodId string,
) error {
	neutral, err := k.ReviewNeutralityEnabled(ctx)
	if err != nil || neutral {
		return err
	}

	if submitter == "" {
		return nil
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	height := uint64(sdkCtx.BlockHeight())

	c := k.getOrInitCalibration(ctx, submitter)
	c.CorroborationsEarned++
	stats := ensureMethodStats(c, methodId)
	stats.CorroborationsEarned++

	c.CalibrationScoreBps = ComputeAgentCalibrationScore(c)
	c.LastUpdatedBlock = height
	_ = k.SetAgentCalibration(ctx, c)
	k.EmitCalibrationUpdated(ctx, c)
	return nil
}

func (k Keeper) recordDisprovalForSubmitterLegacy(
	ctx context.Context,
	submitter string,
	methodId string,
) error {
	neutral, err := k.ReviewNeutralityEnabled(ctx)
	if err != nil || neutral {
		return err
	}

	if submitter == "" {
		return nil
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	height := uint64(sdkCtx.BlockHeight())

	c := k.getOrInitCalibration(ctx, submitter)
	c.DisprovenCount++
	stats := ensureMethodStats(c, methodId)
	stats.Disproven++

	c.CalibrationScoreBps = ComputeAgentCalibrationScore(c)
	c.LastUpdatedBlock = height
	_ = k.SetAgentCalibration(ctx, c)
	k.EmitCalibrationUpdated(ctx, c)
	return nil
}

func (k Keeper) recordChallengeOutcomeLegacy(
	ctx context.Context,
	challenger string,
	succeeded bool,
) error {
	neutral, err := k.ReviewNeutralityEnabled(ctx)
	if err != nil || neutral {
		return err
	}

	if challenger == "" {
		return nil
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	height := uint64(sdkCtx.BlockHeight())

	c := k.getOrInitCalibration(ctx, challenger)
	c.ChallengesIssued++
	if succeeded {
		c.ChallengesSucceeded++
	} else {
		c.ChallengesFailed++
	}
	c.CalibrationScoreBps = ComputeAgentCalibrationScore(c)
	c.LastUpdatedBlock = height
	_ = k.SetAgentCalibration(ctx, c)
	k.EmitCalibrationUpdated(ctx, c)
	return nil
}

// ComputeAgentCalibrationScore is the historical, unsmoothed formula. New v2
// scoring explicitly uses ComputeSmoothedAgentCalibrationScore instead.
func ComputeAgentCalibrationScore(c *types.AgentCalibration) uint64 {
	if c == nil || c.TotalSubmissions == 0 {
		return 0
	}
	const bps uint64 = 1_000_000

	// Inconclusive outcomes are excluded from the denominator — the panel never
	// resolved them, so they are not evidence of (mis)calibration. The guard is
	// defensive against Inconclusive > TotalSubmissions (impossible in real data).
	var decisive uint64
	if c.Inconclusive < c.TotalSubmissions {
		decisive = c.TotalSubmissions - c.Inconclusive
	}
	if decisive == 0 {
		// Every submission was an honest, unresolved attempt: unproven, not
		// wrong. Strictly above a refuted-only record (which scores 0).
		return CalibrationReachingCreditBps
	}

	acceptanceBps := c.Accepted * bps / decisive

	var corrBonus uint64
	if c.Accepted > 0 {
		corrBonus = c.CorroborationsEarned * CalibrationCorroborationScaleBps / c.Accepted
		if corrBonus > CalibrationCorroborationBonusCapBps {
			corrBonus = CalibrationCorroborationBonusCapBps
		}
	}

	var disprovalPenalty uint64
	if c.Accepted > 0 {
		disprovalPenalty = c.DisprovenCount * bps / c.Accepted
	}

	// Compose — signed-ish math in uint64 space.
	score := acceptanceBps + corrBonus
	if disprovalPenalty >= score {
		return 0
	}
	score -= disprovalPenalty
	if score > bps {
		score = bps
	}
	return score
}
