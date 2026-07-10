package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Calibration score formula constants. These are intentionally conservative
// under Phase 5 — the score rewards consistency over volume and penalizes
// post-acceptance disprovals, matching the training-loop intuition that a
// model's reliability is measured by how often its outputs survive.
const (
	// Max corroboration bonus contribution (BPS). 10% added to a perfect
	// acceptance-rate score if the submitter's facts earn corroborations.
	CalibrationCorroborationBonusCapBps uint64 = 100_000
	// Per-corroboration-per-acceptance scaling into the bonus. Tuned so that
	// 1 corroboration per accepted fact approaches the cap.
	CalibrationCorroborationScaleBps uint64 = 100_000
	// Compassion (docs/COMPASSION.md, commitment C2 — "error is not deceit"):
	// the floor a submitter scores when EVERY one of their submissions was
	// INCONCLUSIVE — an honest attempt the panel never resolved. There is no
	// calibration evidence either way, so the record is "unproven", not "wrong":
	// strictly above a submitter whose decisive claims were refuted (which scores
	// 0), and far below any verified fact. The chain tells trying apart from
	// being-wrong. Deliberately tiny — it sits well under the training-fund
	// disbursement floor, so it unlocks no reward; it only orders an honest
	// unresolved attempt above a refuted one.
	CalibrationReachingCreditBps uint64 = 1_000
)

// ─── CRUD ────────────────────────────────────────────────────────────────

// SetAgentCalibration stores the calibration record.
func (k Keeper) SetAgentCalibration(ctx context.Context, c *types.AgentCalibration) error {
	if c == nil || c.Address == "" {
		return nil
	}
	store := k.storeService.OpenKVStore(ctx)
	bz, err := marshalOpts.Marshal(c)
	if err != nil {
		return err
	}
	return store.Set(types.AgentCalibrationKey(c.Address), bz)
}

// GetAgentCalibration fetches a calibration record, returning a zero-valued
// struct if absent (so callers can treat "never submitted" as "empty stats").
func (k Keeper) GetAgentCalibration(ctx context.Context, addr string) (*types.AgentCalibration, bool) {
	if addr == "" {
		return nil, false
	}
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.AgentCalibrationKey(addr))
	if err != nil || bz == nil {
		return nil, false
	}
	var c types.AgentCalibration
	if err := proto.Unmarshal(bz, &c); err != nil {
		return nil, false
	}
	return &c, true
}

// IterateAgentCalibrations yields every calibration record.
func (k Keeper) IterateAgentCalibrations(ctx context.Context, cb func(*types.AgentCalibration) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.AgentCalibrationKeyPrefix, prefixEndBytes(types.AgentCalibrationKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var c types.AgentCalibration
		if err := proto.Unmarshal(iter.Value(), &c); err != nil {
			continue
		}
		if cb(&c) {
			return
		}
	}
}

// getOrInit returns the stored calibration or a fresh record bound to addr.
// Callers should call SetAgentCalibration after mutation.
func (k Keeper) getOrInitCalibration(ctx context.Context, addr string) *types.AgentCalibration {
	if c, found := k.GetAgentCalibration(ctx, addr); found {
		return c
	}
	accountType := ""
	if k.zeroneAuthKeeper != nil {
		accountType = k.getAccountType(ctx, addr)
	}
	return &types.AgentCalibration{
		Address:     addr,
		AccountType: accountType,
		PerMethod:   map[string]*types.AgentMethodStats{},
	}
}

// ensureMethodStats returns the mutable per-method slot for a calibration,
// creating it if needed.
func ensureMethodStats(c *types.AgentCalibration, methodId string) *types.AgentMethodStats {
	if c.PerMethod == nil {
		c.PerMethod = map[string]*types.AgentMethodStats{}
	}
	stats, ok := c.PerMethod[methodId]
	if !ok {
		stats = &types.AgentMethodStats{}
		c.PerMethod[methodId] = stats
	}
	return stats
}

// ─── Update sites ────────────────────────────────────────────────────────

// RecordSubmissionOutcome is called from CompleteRound after a verdict is
// determined. Updates the submitter's lifetime + per-method stats.
func (k Keeper) RecordSubmissionOutcome(
	ctx context.Context,
	submitter string,
	methodId string,
	verdict types.Verdict,
) {
	if submitter == "" {
		return
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
}

// RecordCorroborationForSubmitter is called from handleChallengeSurvival
// (a challenge was rejected → the cited fact survived).
func (k Keeper) RecordCorroborationForSubmitter(
	ctx context.Context,
	submitter string,
	methodId string,
) {
	if submitter == "" {
		return
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
}

// RecordDisprovalForSubmitter is called from cascadeFalsification / direct
// disproof — the submitter's previously-accepted fact was invalidated.
func (k Keeper) RecordDisprovalForSubmitter(
	ctx context.Context,
	submitter string,
	methodId string,
) {
	if submitter == "" {
		return
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
}

// RecordChallengeOutcome tracks the challenger's side of the ledger. A
// challenge is "successful" when the target fact is marked DISPROVEN.
func (k Keeper) RecordChallengeOutcome(
	ctx context.Context,
	challenger string,
	succeeded bool,
) {
	if challenger == "" {
		return
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
}

// ─── Score computation ──────────────────────────────────────────────────

// ComputeAgentCalibrationScore returns a BPS score in [0, BPS] summarising a
// submitter's track record. Formula:
//
//	decisive        = total_submissions − inconclusive     (outcomes the panel judged)
//	acceptance_rate = accepted / decisive                  (reaching credit if no decisive)
//	corr_bonus      = min(cap, corroborations × scale / accepted)
//	disproval_pen   = disproven × BPS / accepted
//	score           = acceptance_rate + corr_bonus - disproval_pen
//	                  clamped to [0, BPS]
//
// Compassion (docs/COMPASSION.md, commitment C2 — "error is not deceit"): an
// INCONCLUSIVE outcome is the chain failing to reach a verdict, not the agent
// failing to be right. It is an honest attempt the panel could not resolve, so
// it is EXCLUDED from the denominator — it neither helps nor hurts the
// acceptance rate. Only decisive outcomes (accepted / rejected / malformed,
// where the panel reached a content judgement) measure calibration. A submitter
// whose every attempt was inconclusive has no calibration evidence: they score a
// tiny "reaching credit" (unproven, not wrong) that sits strictly above a
// submitter whose decisive claims were refuted (which scores 0). The change is
// monotonic — excluding inconclusive can only raise or hold a score, never lower
// it; a record with no inconclusive outcomes is scored identically to before.
//
// This score is NOT cosmetic. A training-fund disbursement gates on it
// (msg_server_training_v4.go — a floor, then a linear scale up to 2× base),
// x/trust_score reads it as submission accuracy, and the structured corpus
// export denormalises it for training weighting. The compassion change only ever
// raises the score of submitters with inconclusive history, so its economic
// effect is bounded to "stop under-paying honest unresolved attempts", and all
// minting remains cap-gated by MintWithCap.
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

// RecomputeAllCalibrationScores re-derives CalibrationScoreBps for every stored
// calibration record under the current formula, in one deterministic pass (store
// key order). Used by the compassion-calibration-v1 upgrade to refresh stored
// scores when the formula changes (the inconclusive-excluding rule), so stored
// state matches the live computation on every node from the upgrade height.
// Idempotent — running it twice yields identical scores. Returns the count
// refreshed. Only CalibrationScoreBps is touched; counts and per-method stats
// are untouched.
func (k Keeper) RecomputeAllCalibrationScores(ctx context.Context) (int, error) {
	var recs []*types.AgentCalibration
	k.IterateAgentCalibrations(ctx, func(c *types.AgentCalibration) bool {
		recs = append(recs, c)
		return false
	})
	for _, c := range recs {
		c.CalibrationScoreBps = ComputeAgentCalibrationScore(c)
		if err := k.SetAgentCalibration(ctx, c); err != nil {
			return 0, err
		}
	}
	return len(recs), nil
}

// EmitCalibrationUpdated emits an event for off-chain observers (training
// pipelines, dashboards) to track calibration changes without polling.
func (k Keeper) EmitCalibrationUpdated(ctx context.Context, c *types.AgentCalibration) {
	if c == nil {
		return
	}
	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(sdk.NewEvent(
		"zerone.knowledge.agent_calibration_updated",
		sdk.NewAttribute("address", c.Address),
		sdk.NewAttribute("account_type", c.AccountType),
		sdk.NewAttribute("total_submissions", fmt.Sprintf("%d", c.TotalSubmissions)),
		sdk.NewAttribute("accepted", fmt.Sprintf("%d", c.Accepted)),
		sdk.NewAttribute("disproven_count", fmt.Sprintf("%d", c.DisprovenCount)),
		sdk.NewAttribute("corroborations_earned", fmt.Sprintf("%d", c.CorroborationsEarned)),
		sdk.NewAttribute("calibration_score_bps", fmt.Sprintf("%d", c.CalibrationScoreBps)),
	))
}
