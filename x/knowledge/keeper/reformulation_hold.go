package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// No hold key is written: retained ballots are the source of truth. A later
// vote can break a tie; invalid retained ballots need a separately governed
// resolution. Historical final verdicts, payments and graph edges are untouched.
type reformulationCandidate struct {
	verdict types.AugmentationVerdict
	weight  uint64
}

func reformulationCandidates(tally map[types.AugmentationVerdict]uint64) []reformulationCandidate {
	out := make([]reformulationCandidate, 0, len(tally))
	for verdict, weight := range tally {
		out = append(out, reformulationCandidate{verdict, weight})
	}
	return out
}

// selectReformulationWinner is order-independent. Exact positive top ties
// remain pending, even below the threshold; no enum gets a preference.
func selectReformulationWinner(candidates []reformulationCandidate, total, threshold uint64) (types.AugmentationVerdict, bool) {
	pending := types.AugmentationVerdict_AUGMENTATION_VERDICT_PENDING
	if total == 0 {
		return pending, false
	}
	winner, weight, tied := pending, uint64(0), false
	for _, candidate := range candidates {
		if candidate.weight > weight {
			winner, weight = candidate.verdict, candidate.weight
			tied = false // a unique higher weight clears a lower tie
		} else if candidate.weight > 0 && candidate.weight == weight {
			tied = true
		}
	}
	if tied {
		return pending, true
	}
	if safeMulDiv(weight, bps, total) < threshold {
		return pending, false
	}
	return winner, false
}

func reformulationStakeTally(aug *types.Augmentation) (map[types.AugmentationVerdict]uint64, uint64) {
	const panelCalibrationFloorBps uint64 = 200_000
	tally := make(map[types.AugmentationVerdict]uint64)
	var total uint64
	for i, vote := range aug.VerdictVotes {
		var stake, calibration uint64
		if i < len(aug.VerdictVoteStakes) {
			stake = aug.VerdictVoteStakes[i]
		}
		if i < len(aug.VerdictVoteCalibrationBps) {
			calibration = aug.VerdictVoteCalibrationBps[i]
		}
		if calibration < panelCalibrationFloorBps {
			calibration = panelCalibrationFloorBps
		}
		weight := safeMulDiv(stake, calibration, bps)
		tally[vote] += weight
		total += weight
	}
	return tally, total
}

// ReformulationHoldReason is the read-only prospective classification shared by
// the keeper census and CLI query. Callers must select the activation marker
// separately. It never finalizes or rewrites a record, nor infers escrow value.
func ReformulationHoldReason(aug *types.Augmentation, params *types.Params) string {
	if aug == nil || aug.Verdict != types.AugmentationVerdict_AUGMENTATION_VERDICT_PENDING {
		return ""
	}
	for _, vote := range aug.VerdictVotes {
		if vote < types.AugmentationVerdict_AUGMENTATION_VERDICT_EQUIVALENT || vote > types.AugmentationVerdict_AUGMENTATION_VERDICT_DRIFT {
			return "invalid_retained_vote"
		}
	}
	minVotes := params.ReformulationMinPanelVotes
	if minVotes == 0 {
		minVotes = 3
	}
	if uint64(len(aug.VerdictVoters)) < minVotes {
		return ""
	}
	tally, total := reformulationStakeTally(aug)
	_, tied := selectReformulationWinner(reformulationCandidates(tally), total, 0)
	if tied {
		return "tied_top_weight"
	}
	return ""
}

// ReformulationHold describes a pending record, not an escrow entitlement.
// Several records may refer to one bounty's pooled remaining escrow.
type ReformulationHold struct {
	AugmentationID string `json:"augmentation_id"`
	OriginalFactID string `json:"original_fact_id"`
	BountyID       string `json:"bounty_id"`
	Reason         string `json:"reason"`
}

// CensusReformulationHolds scans in store-key order and fails closed on unreadable
// records. The carrier must run this BEFORE activation, record the census and
// resolve/accept holds under its separate authority. No marker is required for
// this prospective read, and no state or events are changed by it.
func (k Keeper) CensusReformulationHolds(ctx context.Context) ([]ReformulationHold, error) {
	store := k.storeService.OpenKVStore(ctx)
	paramsBytes, err := store.Get(types.ParamsKey)
	if err != nil {
		return nil, err
	}
	params := types.DefaultParams()
	if paramsBytes != nil {
		if err := proto.Unmarshal(paramsBytes, &params); err != nil {
			return nil, fmt.Errorf("reformulation census: malformed params: %w", err)
		}
	}
	iter, err := store.Iterator(types.AugmentationKeyPrefix, prefixEndBytes(types.AugmentationKeyPrefix))
	if err != nil {
		return nil, err
	}
	closed := false
	defer func() {
		if !closed {
			_ = iter.Close()
		}
	}()
	holds := []ReformulationHold{}
	for ; iter.Valid(); iter.Next() {
		var aug types.Augmentation
		if err := proto.Unmarshal(iter.Value(), &aug); err != nil {
			return nil, fmt.Errorf("reformulation census: malformed augmentation: %w", err)
		}
		if aug.Id == "" || string(iter.Key()) != string(types.AugmentationKey(aug.Id)) {
			return nil, fmt.Errorf("reformulation census: augmentation key/id mismatch")
		}
		if reason := ReformulationHoldReason(&aug, &params); reason != "" {
			holds = append(holds, ReformulationHold{aug.Id, aug.OriginalFactId, aug.BountyId, reason})
		}
	}
	// Reuse the checked SDK-cache exhaustion/Close contract already used by
	// survival escrow; normal exhausted cache iterators report an error.
	closed = true
	if err := closeSurvivalIterator(iter); err != nil {
		return nil, err
	}
	return holds, nil
}

func emitReformulationHold(ctx context.Context, aug *types.Augmentation, reason string) {
	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(sdk.NewEvent(
		"zerone.knowledge.reformulation_held",
		sdk.NewAttribute("augmentation_id", aug.Id),
		sdk.NewAttribute("original_fact_id", aug.OriginalFactId),
		sdk.NewAttribute("bounty_id", aug.BountyId),
		sdk.NewAttribute("reason", reason),
	))
}

// Escrow is pooled per bounty: a different variant or a sponsor veto must not
// spend a held record's remaining backing. There is no per-variant reservation
// from which a safe partial release could be inferred.
func (k Keeper) requireReformulationBountyUnheld(ctx context.Context, bountyID string) error {
	enabled, err := k.ReformulationVindicationOrderingEnabled(ctx)
	if err != nil || !enabled {
		return err
	}
	hold, err := k.reformulationBountyHold(ctx, bountyID)
	if err != nil {
		return err
	}
	if hold != nil {
		return fmt.Errorf("bounty %s escrow held by augmentation %s: %s", bountyID, hold.AugmentationID, hold.Reason)
	}
	return nil
}

func (k Keeper) reformulationBountyHold(ctx context.Context, bountyID string) (*ReformulationHold, error) {
	holds, err := k.CensusReformulationHolds(ctx)
	if err != nil {
		return nil, err
	}
	for _, hold := range holds {
		if hold.BountyID == bountyID {
			return &hold, nil
		}
	}
	return nil, nil
}

func emitReformulationRefusal(ctx context.Context, operation, reason string) {
	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(sdk.NewEvent(
		"zerone.knowledge.reformulation_processing_refused",
		sdk.NewAttribute("operation", operation),
		sdk.NewAttribute("reason", reason),
	))
}
