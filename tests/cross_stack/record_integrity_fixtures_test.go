package cross_stack_test

import (
	"github.com/stretchr/testify/require"
	kt "github.com/zerone-chain/zerone/x/knowledge/types"
	"google.golang.org/protobuf/proto"
	"testing"
)

// Direct aggregation fixtures supply a synthetic result, not signed reviews.
// Persist their unfinished round so native completion reads authoritative state.
// Existing rounds (including real transaction-path rounds) are never replaced.
func storeUnfinalizedFixtureRound(t *testing.T, h *TestHarness, round *kt.VerificationRound) {
	t.Helper()
	if _, found := h.KnowledgeKeeper.GetVerificationRound(h.Ctx, round.Id); found {
		return
	}
	staged := proto.Clone(round).(*kt.VerificationRound)
	staged.Phase = kt.VerificationPhase_VERIFICATION_PHASE_AGGREGATION
	staged.Verdict = kt.Verdict_VERDICT_UNSPECIFIED
	staged.VerdictBlock = 0
	require.NoError(t, h.KnowledgeKeeper.SetVerificationRound(h.Ctx, staged))
}
