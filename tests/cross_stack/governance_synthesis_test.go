package cross_stack_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	govsynthkeeper "github.com/zerone-chain/zerone/x/governance_synthesis/keeper"
	govsynthtypes "github.com/zerone-chain/zerone/x/governance_synthesis/types"
	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// x/governance_synthesis is the third synthesizer module — same shape
// as x/training_provenance and x/trust_score, but at SYSTEM scope.
// Per-system signal: incidents, pauses, pending injections, privileged-
// action burst, cartel posture, alignment pacing.

// NORMAL: nothing happening — chain reports no stress.
func TestGovernanceSynthesis_NormalWhenIdle(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	qs := govsynthkeeper.NewQueryServerImpl(h.GovernanceSynthesisKeeper)
	resp, err := qs.SystemHealth(h.Ctx, &govsynthtypes.QuerySystemHealthRequest{})
	require.NoError(t, err)
	health := resp.Health
	require.NotNil(t, health)
	require.Equal(t, "NORMAL", health.StressLevel,
		"clean-state chain reports NORMAL")
	require.Equal(t, uint32(0), health.OpenIncidents)
	require.Equal(t, uint32(0), health.PausedModules)
	require.Equal(t, uint32(0), health.PendingFactInjections)
}

// CRITICAL via P0 incident open.
func TestGovernanceSynthesis_CriticalOnP0Incident(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	authority := h.KnowledgeKeeper.GetAuthority()

	_, err = ms.OpenIncident(h.Ctx, &knowledgetypes.MsgOpenIncident{
		Authority: authority, Id: "GOV-SYNTH-P0",
		Severity:        knowledgetypes.IncidentSeverity_INCIDENT_SEVERITY_P0,
		Title:           "test P0 for system health",
		AffectedModules: []string{"knowledge"},
	})
	require.NoError(t, err)

	qs := govsynthkeeper.NewQueryServerImpl(h.GovernanceSynthesisKeeper)
	resp, err := qs.SystemHealth(h.Ctx, &govsynthtypes.QuerySystemHealthRequest{})
	require.NoError(t, err)
	health := resp.Health
	require.GreaterOrEqual(t, health.P0Open, uint32(1))
	require.Equal(t, "CRITICAL", health.StressLevel,
		"any P0 incident open ⇒ CRITICAL")
}

// CRITICAL via paused module.
func TestGovernanceSynthesis_CriticalOnPausedModule(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	authority := h.KnowledgeKeeper.GetAuthority()

	_, err = ms.OpenIncident(h.Ctx, &knowledgetypes.MsgOpenIncident{
		Authority: authority, Id: "GOV-SYNTH-PAUSE",
		Severity:        knowledgetypes.IncidentSeverity_INCIDENT_SEVERITY_P1,
		Title:           "test pause",
		AffectedModules: []string{"knowledge"},
	})
	require.NoError(t, err)

	_, err = ms.PauseModule(h.Ctx, &knowledgetypes.MsgPauseModule{
		Authority:  authority,
		ModuleName: "knowledge",
		IncidentId: "GOV-SYNTH-PAUSE",
		Reason:     "test pause",
	})
	require.NoError(t, err)

	qs := govsynthkeeper.NewQueryServerImpl(h.GovernanceSynthesisKeeper)
	resp, err := qs.SystemHealth(h.Ctx, &govsynthtypes.QuerySystemHealthRequest{})
	require.NoError(t, err)
	health := resp.Health
	require.Equal(t, uint32(1), health.PausedModules)
	require.Equal(t, "CRITICAL", health.StressLevel,
		"any module pause ⇒ CRITICAL")
}

// ELEVATED via pending fact injection (guardian-veto window active).
// Demonstrates the synthesizer surfaces a multi-module signal: it sees
// the pending queue from x/knowledge and bumps the system-level state.
func TestGovernanceSynthesis_ElevatedOnPendingFactInjection(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	// Configure the guardian-veto window so MsgAddFact queues rather
	// than executing immediately.
	guardian := testAddr("gov_synth_guardian").String()
	params, err := h.KnowledgeKeeper.GetParams(h.Ctx)
	require.NoError(t, err)
	params.GuardianAddresses = []string{guardian}
	params.AddFactVetoWindowBlocks = 100
	require.NoError(t, h.KnowledgeKeeper.SetParams(h.Ctx, params))

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	_, err = ms.AddFact(h.Ctx, &knowledgetypes.MsgAddFact{
		Authority: h.KnowledgeKeeper.GetAuthority(),
		Content:   "gov synth: queued pending injection",
		Domain:    "sciences",
		Category:  "empirical",
		Confidence: 800_000,
	})
	require.NoError(t, err)

	qs := govsynthkeeper.NewQueryServerImpl(h.GovernanceSynthesisKeeper)
	resp, err := qs.SystemHealth(h.Ctx, &govsynthtypes.QuerySystemHealthRequest{})
	require.NoError(t, err)
	health := resp.Health
	require.GreaterOrEqual(t, health.PendingFactInjections, uint32(1))
	require.Contains(t, []string{"ELEVATED", "CRITICAL"}, health.StressLevel)
}

// Pacing multipliers from x/alignment surface in the health snapshot.
// Demonstrates the previously-under-consumed alignment signal flowing
// through the synthesizer to the governance-level view.
func TestGovernanceSynthesis_AlignmentPacingSurfaces(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	qs := govsynthkeeper.NewQueryServerImpl(h.GovernanceSynthesisKeeper)
	resp, err := qs.SystemHealth(h.Ctx, &govsynthtypes.QuerySystemHealthRequest{})
	require.NoError(t, err)
	health := resp.Health
	// Default pacing on a fresh chain is BPS = 1_000_000 (no throttle).
	// We assert non-zero — the field is populated, the wire is alive.
	require.Greater(t, health.CreationPacingBps, uint64(0),
		"creation pacing must surface — was previously under-consumed")
	require.Greater(t, health.AnalysisPacingBps, uint64(0))
}
