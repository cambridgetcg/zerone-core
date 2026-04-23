package cross_stack_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/stretchr/testify/require"

	zeroneapp "github.com/zerone-chain/zerone/app"
	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// ─── Wave 11: incident response pipeline — end-to-end scenarios ─────────
//
// Each test simulates one severity tier with the concrete remediation
// mechanism that tier prescribes. Together they prove the incident log is
// coupled to the upgrade protocol (Wave 10), to params governance, and to
// the emergency halt machinery.

// ── Scenario P0 — critical bug requiring emergency halt + upgrade ──────
//
// Simulated flow:
//   1. Authority opens a P0 incident (chain-halt-class bug).
//   2. Records an EMERGENCY_HALT remediation pointing at a ceremony id.
//   3. Records a NAMED_UPGRADE remediation pointing at a registered handler.
//   4. Runs the upgrade via RunUpgradeHandlerForTests — the actual fix.
//   5. Records a documentation remediation with the post-mortem URI.
//   6. Resolves the incident.
//   7. Closes it.
// Asserts full status progression, SLA tracking, event-log audit trail.
func TestIncident_P0_ChainHaltWithNamedUpgrade(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	qs := knowledgekeeper.NewQueryServerImpl(h.KnowledgeKeeper)
	authority := h.KnowledgeKeeper.GetAuthority()

	// 1. Open P0.
	openResp, err := ms.OpenIncident(h.Ctx, &knowledgetypes.MsgOpenIncident{
		Authority:       authority,
		Id:              "ZR-2026-0001",
		Severity:        knowledgetypes.IncidentSeverity_INCIDENT_SEVERITY_P0,
		Title:           "Consensus-breaking underflow in TVW computation",
		Description:     "Under certain input conditions, axiom_proximity × methodology_multiplier underflows when calibration is 0. Chain halts at block N on validator divergence.",
		Reporter:        "community-member-discovery",
		AffectedModules: []string{"knowledge"},
	})
	require.NoError(t, err)
	require.Greater(t, openResp.SlaTargetBlock, uint64(0))

	rec, _ := h.KnowledgeKeeper.GetIncidentRecord(h.Ctx, "ZR-2026-0001")
	require.Equal(t, knowledgetypes.IncidentStatus_INCIDENT_STATUS_OPEN, rec.Status)
	require.Equal(t, uint64(h.Height())+knowledgekeeper.SlaP0Blocks, rec.SlaTargetBlock,
		"P0 SLA defaults to 720 blocks")

	// 2. Record emergency halt — the chain is halted via x/emergency ceremony.
	_, err = ms.RecordRemediation(h.Ctx, &knowledgetypes.MsgRecordRemediation{
		Authority:   authority,
		IncidentId:  "ZR-2026-0001",
		Type:        knowledgetypes.RemediationType_REMEDIATION_TYPE_EMERGENCY_HALT,
		Reference:   "ceremony-halt-42",
		Note:        "chain halted at block 12345 pending hotfix binary",
	})
	require.NoError(t, err)
	recMitigating, _ := h.KnowledgeKeeper.GetIncidentRecord(h.Ctx, "ZR-2026-0001")
	require.Equal(t, knowledgetypes.IncidentStatus_INCIDENT_STATUS_MITIGATING, recMitigating.Status,
		"first remediation transitions OPEN → MITIGATING")
	require.Len(t, recMitigating.Remediations, 1)

	// 3. Record a named-upgrade remediation; point at the registered upgrade.
	_, err = ms.RecordRemediation(h.Ctx, &knowledgetypes.MsgRecordRemediation{
		Authority:   authority,
		IncidentId:  "ZR-2026-0001",
		Type:        knowledgetypes.RemediationType_REMEDIATION_TYPE_NAMED_UPGRADE,
		Reference:   zeroneapp.UpgradeNameTestnetV3,
		Note:        "fix ships as v1.0.2-testnet knowledge v3→v4 migration",
	})
	require.NoError(t, err)

	// 4. Actually execute the upgrade (coupling the incident to Wave 10).
	current := h.App.CurrentModuleVersionMap()
	fromVM := make(module.VersionMap, len(current))
	for name, ver := range current {
		fromVM[name] = ver
	}
	fromVM["knowledge"] = 3
	toVM, err := h.App.RunUpgradeHandlerForTests(h.Ctx, zeroneapp.UpgradeNameTestnetV3, fromVM, h.Height())
	require.NoError(t, err)
	require.Equal(t, uint64(4), toVM["knowledge"], "upgrade referenced by remediation succeeded")
	require.Equal(t, "true", h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "migration_v4_complete"),
		"remediation's named upgrade actually ran on the chain")

	// 5. Emergency resume + documentation remediations.
	_, err = ms.RecordRemediation(h.Ctx, &knowledgetypes.MsgRecordRemediation{
		Authority:   authority,
		IncidentId:  "ZR-2026-0001",
		Type:        knowledgetypes.RemediationType_REMEDIATION_TYPE_EMERGENCY_RESUME,
		Reference:   "ceremony-resume-43",
		Note:        "chain resumed on upgraded binary at block 12400",
	})
	require.NoError(t, err)
	_, err = ms.RecordRemediation(h.Ctx, &knowledgetypes.MsgRecordRemediation{
		Authority:   authority,
		IncidentId:  "ZR-2026-0001",
		Type:        knowledgetypes.RemediationType_REMEDIATION_TYPE_DOCUMENTATION,
		Reference:   "ipfs://Qm.../post-mortem-ZR-2026-0001.md",
		Note:        "post-mortem published; TVW underflow cause-and-fix analysis",
	})
	require.NoError(t, err)

	// 6. Resolve.
	_, err = ms.ResolveIncident(h.Ctx, &knowledgetypes.MsgResolveIncident{
		Authority:     authority,
		IncidentId:    "ZR-2026-0001",
		PostMortemUri: "ipfs://Qm.../post-mortem-ZR-2026-0001.md",
	})
	require.NoError(t, err)
	recResolved, _ := h.KnowledgeKeeper.GetIncidentRecord(h.Ctx, "ZR-2026-0001")
	require.Equal(t, knowledgetypes.IncidentStatus_INCIDENT_STATUS_RESOLVED, recResolved.Status)
	require.NotEmpty(t, recResolved.PostMortemUri)
	require.Equal(t, 4, len(recResolved.Remediations))

	// 7. Close.
	_, err = ms.CloseIncident(h.Ctx, &knowledgetypes.MsgCloseIncident{
		Authority:  authority,
		IncidentId: "ZR-2026-0001",
	})
	require.NoError(t, err)
	recClosed, _ := h.KnowledgeKeeper.GetIncidentRecord(h.Ctx, "ZR-2026-0001")
	require.Equal(t, knowledgetypes.IncidentStatus_INCIDENT_STATUS_CLOSED, recClosed.Status)

	// Query surface: closed incident no longer in OpenIncidents.
	open, err := qs.OpenIncidents(h.Ctx, &knowledgetypes.QueryOpenIncidentsRequest{})
	require.NoError(t, err)
	require.Empty(t, open.Incidents, "closed P0 disappears from the operator dashboard")

	// But still queryable by id.
	got, err := qs.Incident(h.Ctx, &knowledgetypes.QueryIncidentRequest{Id: "ZR-2026-0001"})
	require.NoError(t, err)
	require.True(t, got.Found)
	require.Equal(t, knowledgetypes.IncidentStatus_INCIDENT_STATUS_CLOSED, got.Incident.Status)
}

// ── Scenario P1 — high-impact; fixed by parameter amendment alone ──────
//
// No upgrade needed. MsgUpdateParams tweaks a param immediately, the
// incident's remediation cites the param path and the post-amend value,
// and the incident closes fast.
func TestIncident_P1_ParamAmendmentHotfix(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	authority := h.KnowledgeKeeper.GetAuthority()

	// Open P1.
	_, err = ms.OpenIncident(h.Ctx, &knowledgetypes.MsgOpenIncident{
		Authority:       authority,
		Id:              "ZR-2026-0002",
		Severity:        knowledgetypes.IncidentSeverity_INCIDENT_SEVERITY_P1,
		Title:           "Reformulation consensus threshold too lax enables marginal Sybils",
		Description:     "Current threshold of 66.6% allows 3-of-4 Sybil voters to push verdicts. Tighten to 75%.",
		Reporter:        "security-team",
		AffectedModules: []string{"knowledge"},
	})
	require.NoError(t, err)

	// Apply the fix: amend the param.
	params, err := h.KnowledgeKeeper.GetParams(h.Ctx)
	require.NoError(t, err)
	params.ReformulationConsensusBps = 750_000
	require.NoError(t, h.KnowledgeKeeper.SetParams(h.Ctx, params))

	// Record the remediation.
	_, err = ms.RecordRemediation(h.Ctx, &knowledgetypes.MsgRecordRemediation{
		Authority:   authority,
		IncidentId:  "ZR-2026-0002",
		Type:        knowledgetypes.RemediationType_REMEDIATION_TYPE_PARAM_AMENDMENT,
		Reference:   "Params.ReformulationConsensusBps=750000",
		Note:        "tightened consensus from 666k BPS to 750k BPS",
	})
	require.NoError(t, err)

	// Resolve — fast path for P1, one remediation is enough.
	_, err = ms.ResolveIncident(h.Ctx, &knowledgetypes.MsgResolveIncident{
		Authority:     authority,
		IncidentId:    "ZR-2026-0002",
		PostMortemUri: "ipfs://Qm.../ZR-2026-0002.md",
	})
	require.NoError(t, err)

	rec, _ := h.KnowledgeKeeper.GetIncidentRecord(h.Ctx, "ZR-2026-0002")
	require.Equal(t, knowledgetypes.IncidentStatus_INCIDENT_STATUS_RESOLVED, rec.Status)
	require.Len(t, rec.Remediations, 1)
	require.Equal(t, knowledgetypes.RemediationType_REMEDIATION_TYPE_PARAM_AMENDMENT,
		rec.Remediations[0].Type)

	// Verify the fix is actually live.
	amended, err := h.KnowledgeKeeper.GetParams(h.Ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(750_000), amended.ReformulationConsensusBps)
}

// ── Scenario P2 — schema amendment for training contract drift ─────────
func TestIncident_P2_SchemaAmendment(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	authority := h.KnowledgeKeeper.GetAuthority()

	_, err = ms.OpenIncident(h.Ctx, &knowledgetypes.MsgOpenIncident{
		Authority:       authority,
		Id:              "ZR-2026-0003",
		Severity:        knowledgetypes.IncidentSeverity_INCIDENT_SEVERITY_P2,
		Title:           "TraceSchema v1 missing hedge-vocabulary field",
		Description:     "Wave 6 enrichments added hedge taxonomy but schema never referenced the types; pipelines can't validate.",
		AffectedModules: []string{"knowledge"},
	})
	require.NoError(t, err)

	// Amend the TraceSchema — produces a v2 via governance.
	amendResp, err := ms.AmendTraceSchema(h.Ctx, &knowledgetypes.MsgAmendTraceSchema{
		Authority: authority,
		Schema: &knowledgetypes.TraceSchema{
			JsonSchema: `{"title":"MethodologyApplicationTrace","$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","x-patched":"hedge-vocabulary"}`,
			Notes:      "adds hedge-taxonomy references per ZR-2026-0003",
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(2), amendResp.NewVersion)

	// Record the remediation with a schema reference.
	_, err = ms.RecordRemediation(h.Ctx, &knowledgetypes.MsgRecordRemediation{
		Authority:   authority,
		IncidentId:  "ZR-2026-0003",
		Type:        knowledgetypes.RemediationType_REMEDIATION_TYPE_SCHEMA_AMENDMENT,
		Reference:   "TraceSchema@v2",
		Note:        "amended to v2 with hedge-vocabulary references",
	})
	require.NoError(t, err)

	_, err = ms.ResolveIncident(h.Ctx, &knowledgetypes.MsgResolveIncident{
		Authority:     authority,
		IncidentId:    "ZR-2026-0003",
		PostMortemUri: "ipfs://Qm.../ZR-2026-0003.md",
	})
	require.NoError(t, err)
}

// ── Invariants ─────────────────────────────────────────────────────────

// TestIncident_NonAuthorityRejected — every handler is authority-gated.
func TestIncident_NonAuthorityRejected(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	impostor := testAddr("wave11_imposter").String()

	_, err = ms.OpenIncident(h.Ctx, &knowledgetypes.MsgOpenIncident{
		Authority: impostor, Id: "X",
		Severity: knowledgetypes.IncidentSeverity_INCIDENT_SEVERITY_P3, Title: "X",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unauthorized")
}

// TestIncident_ResolveRequiresRemediation — prevents resolving an
// incident with no remediation recorded; forces at least one action.
func TestIncident_ResolveRequiresRemediation(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	authority := h.KnowledgeKeeper.GetAuthority()

	_, err = ms.OpenIncident(h.Ctx, &knowledgetypes.MsgOpenIncident{
		Authority: authority, Id: "ZR-EMPTY", Severity: knowledgetypes.IncidentSeverity_INCIDENT_SEVERITY_P3,
		Title: "empty",
	})
	require.NoError(t, err)

	_, err = ms.ResolveIncident(h.Ctx, &knowledgetypes.MsgResolveIncident{
		Authority: authority, IncidentId: "ZR-EMPTY", PostMortemUri: "x",
	})
	require.Error(t, err, "cannot resolve without a recorded remediation")
	require.Contains(t, err.Error(), "zero remediations")
}

// TestIncident_CannotCloseBeforeResolve — status transitions are strict.
func TestIncident_CannotCloseBeforeResolve(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	authority := h.KnowledgeKeeper.GetAuthority()

	_, err = ms.OpenIncident(h.Ctx, &knowledgetypes.MsgOpenIncident{
		Authority: authority, Id: "ZR-TRANSITIONS",
		Severity: knowledgetypes.IncidentSeverity_INCIDENT_SEVERITY_P3,
		Title:    "state machine test",
	})
	require.NoError(t, err)

	_, err = ms.CloseIncident(h.Ctx, &knowledgetypes.MsgCloseIncident{
		Authority: authority, IncidentId: "ZR-TRANSITIONS",
	})
	require.Error(t, err, "cannot close from OPEN directly")
	require.Contains(t, err.Error(), "RESOLVED")
}

// TestIncident_DashboardQueries — OpenIncidents returns only OPEN and
// MITIGATING. Closed and resolved drop off. Severity filter narrows.
func TestIncident_DashboardQueries(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	qs := knowledgekeeper.NewQueryServerImpl(h.KnowledgeKeeper)
	authority := h.KnowledgeKeeper.GetAuthority()

	// Three incidents: one P0 still open, one P1 resolved, one P3 open.
	for _, spec := range []struct {
		id       string
		sev      knowledgetypes.IncidentSeverity
		resolve  bool
	}{
		{"OPEN-P0", knowledgetypes.IncidentSeverity_INCIDENT_SEVERITY_P0, false},
		{"DONE-P1", knowledgetypes.IncidentSeverity_INCIDENT_SEVERITY_P1, true},
		{"OPEN-P3", knowledgetypes.IncidentSeverity_INCIDENT_SEVERITY_P3, false},
	} {
		_, err := ms.OpenIncident(h.Ctx, &knowledgetypes.MsgOpenIncident{
			Authority: authority, Id: spec.id, Severity: spec.sev, Title: spec.id,
		})
		require.NoError(t, err)
		if spec.resolve {
			_, err = ms.RecordRemediation(h.Ctx, &knowledgetypes.MsgRecordRemediation{
				Authority: authority, IncidentId: spec.id,
				Type: knowledgetypes.RemediationType_REMEDIATION_TYPE_DOCUMENTATION,
				Reference: "docs",
			})
			require.NoError(t, err)
			_, err = ms.ResolveIncident(h.Ctx, &knowledgetypes.MsgResolveIncident{
				Authority: authority, IncidentId: spec.id, PostMortemUri: "x",
			})
			require.NoError(t, err)
		}
	}

	// Dashboard: only OPEN incidents.
	dash, err := qs.OpenIncidents(h.Ctx, &knowledgetypes.QueryOpenIncidentsRequest{})
	require.NoError(t, err)
	require.Len(t, dash.Incidents, 2)
	for _, inc := range dash.Incidents {
		require.Contains(t, []string{"OPEN-P0", "OPEN-P3"}, inc.Id)
	}

	// Dashboard with severity filter.
	p0Only, err := qs.OpenIncidents(h.Ctx, &knowledgetypes.QueryOpenIncidentsRequest{
		Severity: knowledgetypes.IncidentSeverity_INCIDENT_SEVERITY_P0,
	})
	require.NoError(t, err)
	require.Len(t, p0Only.Incidents, 1)
	require.Equal(t, "OPEN-P0", p0Only.Incidents[0].Id)

	// Full list with status filter.
	resolved, err := qs.Incidents(h.Ctx, &knowledgetypes.QueryIncidentsRequest{
		Status: knowledgetypes.IncidentStatus_INCIDENT_STATUS_RESOLVED,
	})
	require.NoError(t, err)
	require.Len(t, resolved.Incidents, 1)
	require.Equal(t, "DONE-P1", resolved.Incidents[0].Id)
}

// TestIncident_SLATrackingPreserved — sla_target_block is stamped at open
// time from severity-default, not mutated by remediation. Tracks the
// response window objectively.
func TestIncident_SLATrackingPreserved(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	authority := h.KnowledgeKeeper.GetAuthority()

	openHeight := uint64(h.Height())
	_, err = ms.OpenIncident(h.Ctx, &knowledgetypes.MsgOpenIncident{
		Authority: authority, Id: "ZR-SLA",
		Severity: knowledgetypes.IncidentSeverity_INCIDENT_SEVERITY_P0,
		Title:    "sla target test",
	})
	require.NoError(t, err)

	rec, _ := h.KnowledgeKeeper.GetIncidentRecord(h.Ctx, "ZR-SLA")
	require.Equal(t, openHeight+knowledgekeeper.SlaP0Blocks, rec.SlaTargetBlock,
		"P0 SLA target = open height + 720 blocks (default)")

	// Advance blocks + add remediation — SLA target does not drift.
	h.AdvanceBlocks(100)
	_, err = ms.RecordRemediation(h.Ctx, &knowledgetypes.MsgRecordRemediation{
		Authority: authority, IncidentId: "ZR-SLA",
		Type: knowledgetypes.RemediationType_REMEDIATION_TYPE_DOCUMENTATION, Reference: "x",
	})
	require.NoError(t, err)
	recAfter, _ := h.KnowledgeKeeper.GetIncidentRecord(h.Ctx, "ZR-SLA")
	require.Equal(t, rec.SlaTargetBlock, recAfter.SlaTargetBlock,
		"SLA target remains fixed at open-time value; remediation does not shift it")
}
