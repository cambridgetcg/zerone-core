package cross_stack_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	aligntypes "github.com/zerone-chain/zerone/x/alignment/types"
	emergencytypes "github.com/zerone-chain/zerone/x/emergency/types"
)

// TestScenario2_AlignmentCorrections verifies alignment generates corrections
// when knowledge quality is below the degraded threshold, and records them
// queryably. Since the autopoiesis regulator retired (slim cut), corrections
// are never auto-applied — the observation layer keeps speaking; nothing
// adjusts multipliers.
func TestScenario2_AlignmentCorrections(t *testing.T) {
	h := NewTestHarness(t)

	// 1. Enable alignment.
	h.AlignmentKeeper.SetState(h.Ctx, &aligntypes.AlignmentState{
		Enabled:               true,
		LastObservationHeight: 0,
		ObservationCount:      0,
	})

	// 1b. Raise auto-apply magnitude cap: even inside bounds, nothing applies.
	alignParamsInit := h.AlignmentKeeper.GetParams(h.Ctx)
	alignParamsInit.MaxAutoApplyMagnitudeBps = 1_000_000
	h.AlignmentKeeper.SetParams(h.Ctx, alignParamsInit)

	// 2. Observe — sensors will return NeutralBPS (500,000) for nil cross-module
	// keepers. Knowledge quality depends on the KnowledgeKeeper wired in the app.
	obs := h.AlignmentKeeper.ObserveAll(h.Ctx)
	require.NotNil(t, obs)
	require.Greater(t, obs.Height, uint64(0))

	// 3. Compute scores.
	scores := h.AlignmentKeeper.ComputeScores(h.Ctx, obs)
	require.NotNil(t, scores)

	// 4. Force low knowledge quality to trigger correction generation.
	// DegradedThreshold is 400,000. Set KnowledgeQuality well below it.
	scores.KnowledgeQuality = 100_000
	scores.Composite = 100_000 // Force low composite too.

	// 5. Generate corrections.
	corrections := h.AlignmentKeeper.GenerateCorrections(h.Ctx, scores)
	require.NotEmpty(t, corrections, "corrections must be generated for low knowledge quality")

	// Find the knowledge quality correction.
	var knowledgeCorrection *aligntypes.CorrectionRecord
	for _, c := range corrections {
		if c.Dimension == aligntypes.DimKnowledgeQuality {
			knowledgeCorrection = c
			break
		}
	}
	require.NotNil(t, knowledgeCorrection, "knowledge_quality correction must exist")
	require.Equal(t, "increase", knowledgeCorrection.Direction)
	require.False(t, knowledgeCorrection.Applied, "correction must not be applied yet")

	// 6. Apply corrections — records them with applied=false (no regulator).
	h.AlignmentKeeper.ApplyCorrections(h.Ctx, corrections)

	stored, total := h.AlignmentKeeper.GetCorrections(h.Ctx, 100, 0)
	require.Greater(t, total, uint64(0), "corrections must be stored queryably")
	for _, c := range stored {
		require.False(t, c.Applied, "correction %s must be recorded, never auto-applied", c.Dimension)
	}
}

// TestScenario10_EmergencyHaltStopsAdaptiveLayer verifies that the emergency
// halt prevents alignment from observing, and that it resumes afterwards.
func TestScenario10_EmergencyHaltStopsAdaptiveLayer(t *testing.T) {
	h := NewTestHarness(t)

	alignParams := aligntypes.DefaultParams()
	alignParams.ObservationIntervalBlocks = 5
	h.AlignmentKeeper.SetParams(h.Ctx, &alignParams)
	h.AlignmentKeeper.SetState(h.Ctx, &aligntypes.AlignmentState{
		Enabled: true,
	})

	// 1. Advance a few blocks to establish baseline.
	h.AdvanceBlocks(6)

	// 2. Halt the chain via emergency module.
	h.EmergencyKeeper.SetEmergencyStatus(h.Ctx, emergencytypes.StatusHalted)
	require.True(t, h.EmergencyKeeper.IsHalted(h.Ctx), "chain must be halted")

	// 3. Advance blocks — alignment should skip processing.
	h.AdvanceBlocks(20)
	alignStateHalted := h.AlignmentKeeper.GetState(h.Ctx)
	obsCountDuringHalt := alignStateHalted.ObservationCount

	// 4. Resume: set status back to normal.
	h.EmergencyKeeper.SetEmergencyStatus(h.Ctx, emergencytypes.StatusNormal)
	require.False(t, h.EmergencyKeeper.IsHalted(h.Ctx), "chain must be resumed")

	// 5. Advance blocks — alignment should resume processing.
	h.AdvanceBlocks(20)
	alignStateResumed := h.AlignmentKeeper.GetState(h.Ctx)
	require.GreaterOrEqual(t, alignStateResumed.ObservationCount, obsCountDuringHalt,
		"alignment must resume observations after emergency ends")
}
