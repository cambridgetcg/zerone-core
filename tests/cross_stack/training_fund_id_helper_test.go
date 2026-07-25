package cross_stack_test

import (
	"fmt"
	"testing"
)

// A training-fund disbursement is identified by (model_id, disbursement
// epoch), not by the client-chosen msg.Id — see
// x/knowledge/keeper/msg_server_training_v4.go. These helpers rebuild that
// key so tests address records the way the chain does.

func disbursementEpochBlocks(h *TestHarness) uint64 {
	params, _ := h.KnowledgeKeeper.GetParams(h.Ctx)
	if params == nil || params.FitnessEpochBlocks == 0 {
		return 10_000
	}
	return params.FitnessEpochBlocks
}

// derivedDisbursementIDAt builds the id for a claim made at `height`.
func derivedDisbursementIDAt(h *TestHarness, modelID string, height int64) string {
	return fmt.Sprintf("%s/%d", modelID, uint64(height)/disbursementEpochBlocks(h))
}

// derivedDisbursementID builds the id for a claim made at the harness's
// current height.
func derivedDisbursementID(t *testing.T, h *TestHarness, modelID string) string {
	t.Helper()
	return derivedDisbursementIDAt(h, modelID, h.Height())
}
