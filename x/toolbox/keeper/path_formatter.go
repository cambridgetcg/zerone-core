package keeper

import (
	"fmt"

	"github.com/zerone-chain/zerone/x/toolbox/types"
)

// FormatPath is a pure function that converts a PurposeAnalysis into a FormattedPath
// with concrete steps and timeline. No keeper or state access.
func FormatPath(analysis *types.PurposeAnalysis, agentAddr string, blockHeight uint64) *types.FormattedPath {
	if analysis == nil || analysis.PrimaryPurpose == nil {
		return &types.FormattedPath{
			CurrentState:    "Unknown",
			Destination:     "Explore diverse tasks to discover your unique purpose",
			EstimatedEpochs: 0,
		}
	}

	// Current state from overall confidence.
	currentState := types.ConfidenceLabel(analysis.OverallConfidence)

	// Build steps from growth recommendations.
	var steps []*types.PathStep
	var totalEpochs uint64

	for i, rec := range analysis.GrowthPath {
		steps = append(steps, &types.PathStep{
			StepNumber:   uint32(i + 1),
			Action:       fmt.Sprintf("Develop %s", rec.Capability),
			Description:  rec.Rationale,
			TargetMetric: fmt.Sprintf("Reach %s tier", rec.TargetTier),
		})
		totalEpochs += rec.EstimatedEpochs
	}

	return &types.FormattedPath{
		CurrentState:    currentState,
		Steps:           steps,
		Destination:     analysis.PrimaryPurpose.Statement,
		EstimatedEpochs: totalEpochs,
	}
}
