package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// This narrowly scoped receiver closes the old handler's call tree without
// altering its body or charging another marker read for each fact. It is never
// installed as a module keeper or server.
type calibrationMsgLegacy struct{ keeper calibrationKeeperLegacy }
type calibrationKeeperLegacy struct{ Keeper }

func (k calibrationKeeperLegacy) ComputeTrainingValueWeight(ctx context.Context, factID string) TrainingValueWeightBreakdown {
	return k.computeTrainingValueWeightLegacy(ctx, factID)
}

// AttributeContributions is verbatim from
// bc2a58fccb3e9bbb43647b0e51490fad034f34fd:msg_server_training_v3.go.
func (m calibrationMsgLegacy) AttributeContributions(ctx context.Context, msg *types.MsgAttributeContributions) (*types.MsgAttributeContributionsResponse, error) {
	neutral, err := m.keeper.ReviewNeutralityEnabled(ctx)
	if err != nil {
		return nil, err
	}
	if msg == nil || msg.ModelId == "" {
		return nil, fmt.Errorf("model_id required")
	}
	card, ok := m.keeper.GetModelCard(ctx, msg.ModelId)
	if !ok {
		return nil, fmt.Errorf("model card %s not found", msg.ModelId)
	}
	if card.OwnerAddress != msg.Owner {
		return nil, fmt.Errorf("only the model owner may attribute contributions")
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	height := uint64(sdkCtx.BlockHeight())

	// Is-ought wall: partition ids. Normative commitments are rejected.
	facts, rejected := m.keeper.FilterIsOughtIds(ctx, msg.FactIds)

	// Wave 3 total_weight (raw, for audit); Wave 4 computed TVW (Popper-weighted).
	var totalWeight uint64
	var computedTVW uint64
	var perFactCal []uint64
	var attributionPolicy uint32
	if neutral {
		// This is the owner's declaration of use, not a protocol valuation.
		// Zero computed weight and absent calibration snapshots are intentional.
		totalWeight = msg.TotalWeight
		attributionPolicy = 1
	} else {
		perFactCal = make([]uint64, 0, len(facts))
		for _, f := range facts {
			if fact, ok := m.keeper.GetFact(ctx, f); ok {
				totalWeight += fact.CorroborationCount + 1
				perFactCal = append(perFactCal, fact.SubmitterCalibrationSnapshotBps)
			} else {
				totalWeight += 1
				perFactCal = append(perFactCal, 0)
			}
			tvw := m.keeper.ComputeTrainingValueWeight(ctx, f)
			computedTVW += tvw.Final
		}
		if msg.TotalWeight != 0 {
			totalWeight = msg.TotalWeight // legacy raw override
		}
	}

	record := &types.ContributionRecord{
		ModelId:                  msg.ModelId,
		FactIds:                  facts,
		AttributedBy:             msg.Owner,
		AttributedAtBlock:        height,
		TotalWeight:              totalWeight,
		ComputedTvw:              computedTVW,
		RejectedCommitmentCount:  uint32(len(rejected)),
		PerFactCalibrationBps:    perFactCal,
		AttributionPolicyVersion: attributionPolicy,
	}
	if err := m.keeper.SetContributionRecord(ctx, record); err != nil {
		return nil, err
	}
	event := sdk.NewEvent(
		"zerone.knowledge.contributions_attributed",
		sdk.NewAttribute("model_id", msg.ModelId),
		sdk.NewAttribute("attributed_by", msg.Owner),
		sdk.NewAttribute("fact_count", fmt.Sprintf("%d", len(facts))),
		sdk.NewAttribute("total_weight", fmt.Sprintf("%d", totalWeight)),
		sdk.NewAttribute("computed_tvw", fmt.Sprintf("%d", computedTVW)),
		sdk.NewAttribute("rejected_commitments", fmt.Sprintf("%d", len(rejected))),
	)
	if neutral {
		event = event.AppendAttributes(sdk.NewAttribute("attribution_policy_version", "1"))
	}
	sdkCtx.EventManager().EmitEvent(event)
	return &types.MsgAttributeContributionsResponse{Recorded: uint32(len(facts))}, nil
}
