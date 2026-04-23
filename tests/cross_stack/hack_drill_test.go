package cross_stack_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// ─── Wave 13: recursive hack drill ──────────────────────────────────────
//
// Each iteration simulates an external attack and runs the full response
// pipeline end-to-end. Between iterations, discovered friction is fixed
// in the main code and the drill is re-run. Convergence target: each
// iteration produces only cosmetic differences from the prior run.
//
// The drills are integration tests, not unit tests — they exercise every
// primitive (circuit breaker, incident log, surgical correction,
// upgrade, merkle re-verification, audit trail) together.

// ─── Iteration 1: manifest corruption attack ────────────────────────────
//
// ATTACK:
//   An attacker gains RPC-write access (e.g. via a compromised validator
//   key or a node-level exploit) and rewrites a FINALIZED manifest's
//   Merkle root directly in the knowledge module's KV store. A trainer
//   downloading the bundle would now see an invalid commitment.
//
// EXPECTED RESPONSE:
//   1. Monitor observes merkle_root_valid=false on bundle query.
//   2. Incident opened (P1 — not a chain halt, but integrity-critical).
//   3. Module paused (contain further corruption).
//   4. Authority applies surgical correction — recompute + rewrite root.
//   5. Bundle re-verifies.
//   6. Unpause, record remediations, resolve, close.
//
// EXPECTED ITER-1 GAP:
//   There is no authority-gated "recompute manifest root" msg. The
//   Wave 7 design deliberately made finalized manifests immutable —
//   which means an ATTACK that corrupts state CANNOT be corrected by
//   legitimate authority via a structured message. The only path is a
//   code upgrade that runs a migration, OR direct keeper access.
//   This test documents that gap; iter-2 adds MsgCorrectManifest.
func TestHackDrill_Iter1_ManifestCorruption(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	qs := knowledgekeeper.NewQueryServerImpl(h.KnowledgeKeeper)
	authority := h.KnowledgeKeeper.GetAuthority()

	operator := testAddr("hack1_op").String()

	// Setup: pipeline + fact + manifest (legitimately created, legitimately finalized).
	require.NoError(t, h.KnowledgeKeeper.SetTrainingPipeline(h.Ctx, &knowledgetypes.TrainingPipeline{
		Id: "pipe-hack1", OperatorAddress: operator, TokenizerVersion: 1,
	}))
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
		Id: "F-HACK1", Content: "legit", Domain: "sciences",
		Status: knowledgetypes.FactStatus_FACT_STATUS_ACTIVE, Submitter: operator,
		MethodId: knowledgetypes.MethodologyEmpirical, Confidence: 900_000, CorroborationCount: 3,
	}))
	_, err = ms.CreateTrainingManifest(h.Ctx, &knowledgetypes.MsgCreateTrainingManifest{
		Creator: operator, Id: "m-hack1", PipelineId: "pipe-hack1",
		CorpusSelector: &knowledgetypes.CorpusSelector{MethodId: knowledgetypes.MethodologyEmpirical, MinCorroboration: 3},
	})
	require.NoError(t, err)
	finResp, err := ms.FinalizeTrainingManifest(h.Ctx, &knowledgetypes.MsgFinalizeTrainingManifest{
		Creator: operator, ManifestId: "m-hack1",
	})
	require.NoError(t, err)
	trueRoot := finResp.MerkleRoot
	require.NotEmpty(t, trueRoot)

	// ── ATTACK: rewrite the manifest's Merkle root to something wrong ──
	// This represents a hacker with RPC-write access directly mutating
	// state outside the msg-server path.
	corrupted, _ := h.KnowledgeKeeper.GetTrainingManifest(h.Ctx, "m-hack1")
	corrupted.MerkleRoot = "deadbeef" + trueRoot[8:]
	require.NoError(t, h.KnowledgeKeeper.SetTrainingManifest(h.Ctx, corrupted))

	// ── DETECTION ──
	bundle, err := qs.TrainingManifestBundle(h.Ctx, &knowledgetypes.QueryTrainingManifestBundleRequest{Id: "m-hack1"})
	require.NoError(t, err)
	require.False(t, bundle.MerkleRootValid,
		"the tamper is detectable — this is how the incident is discovered")
	require.NotEqual(t, bundle.DerivedMerkleRoot, bundle.Manifest.MerkleRoot)

	// ── RESPONSE: incident + pause + correction attempt ──
	_, err = ms.OpenIncident(h.Ctx, &knowledgetypes.MsgOpenIncident{
		Authority: authority, Id: "HACK-001", Severity: knowledgetypes.IncidentSeverity_INCIDENT_SEVERITY_P1,
		Title: "Manifest m-hack1 Merkle root corrupted — source unknown",
		Description: "bundle.merkle_root_valid=false on downstream verification. External consumers would reject. Suspected RPC-write exploit.",
		AffectedModules: []string{"knowledge"},
	})
	require.NoError(t, err)

	_, err = ms.PauseModule(h.Ctx, &knowledgetypes.MsgPauseModule{
		Authority: authority, ModuleName: "knowledge",
		Reason: "HACK-001: stop further manifest writes while investigation proceeds",
		IncidentId: "HACK-001",
	})
	require.NoError(t, err)

	// ── ITER-1 GAP: no authority-gated way to recompute + rewrite ──
	// The surgical correction path is NOT available in iter 1. The only
	// way to fix the manifest is via a code upgrade's migration or
	// direct keeper access (which is what the hacker used — and shouldn't
	// be part of a trust-minimised response).
	//
	// Iter 2 adds MsgCorrectManifest. For iter 1 we document and simulate
	// the direct keeper call to close the loop so the rest of the drill
	// can run; the simulation stands in for the upgrade path.
	corrected, _ := h.KnowledgeKeeper.GetTrainingManifest(h.Ctx, "m-hack1")
	corrected.MerkleRoot = trueRoot
	require.NoError(t, h.KnowledgeKeeper.SetTrainingManifest(h.Ctx, corrected))
	// (In iter-2 this becomes a MsgCorrectManifest call, not a direct keeper hit.)

	// ── VERIFY fix ──
	fixedBundle, err := qs.TrainingManifestBundle(h.Ctx, &knowledgetypes.QueryTrainingManifestBundleRequest{Id: "m-hack1"})
	require.NoError(t, err)
	require.True(t, fixedBundle.MerkleRootValid, "manifest re-verifies post-correction")

	// ── RESUME + REMEDIATION TRAIL ──
	_, err = ms.RecordRemediation(h.Ctx, &knowledgetypes.MsgRecordRemediation{
		Authority: authority, IncidentId: "HACK-001",
		Type: knowledgetypes.RemediationType_REMEDIATION_TYPE_STATE_CORRECTION,
		Reference: "direct-keeper-write (ITER-1 GAP: no surgical msg exists)",
		Note: "manifest root recomputed from canonical IDs + rewritten",
	})
	require.NoError(t, err)

	_, err = ms.UnpauseModule(h.Ctx, &knowledgetypes.MsgUnpauseModule{
		Authority: authority, ModuleName: "knowledge",
	})
	require.NoError(t, err)

	_, err = ms.ResolveIncident(h.Ctx, &knowledgetypes.MsgResolveIncident{
		Authority: authority, IncidentId: "HACK-001", PostMortemUri: "ipfs://Qm.../HACK-001.md",
	})
	require.NoError(t, err)
	_, err = ms.CloseIncident(h.Ctx, &knowledgetypes.MsgCloseIncident{
		Authority: authority, IncidentId: "HACK-001",
	})
	require.NoError(t, err)

	// ── ITER-1 FRICTION POINTS (recorded) ──
	// 1. [CRITICAL] No authority-gated surgical correction msg for manifests.
	//    Recovery required direct keeper access — not trust-minimised.
	// 2. [MINOR] Incident record's STATE_CORRECTION remediation reference
	//    string is unstructured — just free-form text. An indexer couldn't
	//    programmatically correlate the correction with the corrupted
	//    object. Iter 3+ may tighten.
	// 3. [MINOR] Incident description had to include the merkle root diff
	//    manually. Could auto-capture "delta" context on OpenIncident.
}

// ─── Iteration 2: same attack, surgical correction available ────────────
//
// ATTACK (identical to iter 1).
//
// RESPONSE (improved):
//   Iteration 2 adds MsgCorrectManifestMerkleRoot. The correction is now
//   a structured, authority-gated, incident-bound msg — NOT a direct
//   keeper write. An external auditor can verify the chain recovered
//   purely from public state + event stream.
//
// EXPECTED NEW FRICTION:
//   - Still needs a way to detect SLA-breached incidents programmatically
//     (eyeballing the dashboard is operator-dependent).
//   - The correction is for one manifest at a time; a mass-corruption
//     attack would require N calls. For now acceptable; Wave 14+ can
//     consider batching.
func TestHackDrill_Iter2_ManifestCorruptionWithSurgicalMsg(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	qs := knowledgekeeper.NewQueryServerImpl(h.KnowledgeKeeper)
	authority := h.KnowledgeKeeper.GetAuthority()

	operator := testAddr("hack2_op").String()
	require.NoError(t, h.KnowledgeKeeper.SetTrainingPipeline(h.Ctx, &knowledgetypes.TrainingPipeline{
		Id: "pipe-hack2", OperatorAddress: operator, TokenizerVersion: 1,
	}))
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
		Id: "F-HACK2", Content: "legit", Domain: "sciences",
		Status: knowledgetypes.FactStatus_FACT_STATUS_ACTIVE, Submitter: operator,
		MethodId: knowledgetypes.MethodologyEmpirical, Confidence: 900_000, CorroborationCount: 3,
	}))
	_, err = ms.CreateTrainingManifest(h.Ctx, &knowledgetypes.MsgCreateTrainingManifest{
		Creator: operator, Id: "m-hack2", PipelineId: "pipe-hack2",
		CorpusSelector: &knowledgetypes.CorpusSelector{MethodId: knowledgetypes.MethodologyEmpirical, MinCorroboration: 3},
	})
	require.NoError(t, err)
	finResp, err := ms.FinalizeTrainingManifest(h.Ctx, &knowledgetypes.MsgFinalizeTrainingManifest{
		Creator: operator, ManifestId: "m-hack2",
	})
	require.NoError(t, err)
	trueRoot := finResp.MerkleRoot

	// ── ATTACK: rewrite Merkle root via direct keeper (simulated exploit) ──
	corrupted, _ := h.KnowledgeKeeper.GetTrainingManifest(h.Ctx, "m-hack2")
	corrupted.MerkleRoot = "deadbeef" + trueRoot[8:]
	require.NoError(t, h.KnowledgeKeeper.SetTrainingManifest(h.Ctx, corrupted))

	// ── DETECTION ──
	bundle, _ := qs.TrainingManifestBundle(h.Ctx, &knowledgetypes.QueryTrainingManifestBundleRequest{Id: "m-hack2"})
	require.False(t, bundle.MerkleRootValid)

	// ── RESPONSE ──
	_, err = ms.OpenIncident(h.Ctx, &knowledgetypes.MsgOpenIncident{
		Authority: authority, Id: "HACK-002",
		Severity: knowledgetypes.IncidentSeverity_INCIDENT_SEVERITY_P1,
		Title: "m-hack2 merkle root tampered",
		AffectedModules: []string{"knowledge"},
	})
	require.NoError(t, err)
	_, err = ms.PauseModule(h.Ctx, &knowledgetypes.MsgPauseModule{
		Authority: authority, ModuleName: "knowledge",
		Reason: "HACK-002", IncidentId: "HACK-002",
	})
	require.NoError(t, err)

	// ── ITER-2: SURGICAL CORRECTION VIA STRUCTURED MSG ──
	correctResp, err := ms.CorrectManifestMerkleRoot(h.Ctx, &knowledgetypes.MsgCorrectManifestMerkleRoot{
		Authority:  authority,
		ManifestId: "m-hack2",
		IncidentId: "HACK-002",
		// Optional expected-root assertion — operator has the true root
		// from the original Finalize response, can defensively assert it.
		ExpectedRecomputedRoot: trueRoot,
		Note:                   "recompute from canonical IDs post-exploit",
	})
	require.NoError(t, err)
	require.True(t, correctResp.WasCorrupted, "handler detected the tamper")
	require.Equal(t, trueRoot, correctResp.RecomputedRoot)
	require.NotEqual(t, correctResp.PriorRoot, correctResp.RecomputedRoot)

	// ── VERIFY fix ──
	fixedBundle, _ := qs.TrainingManifestBundle(h.Ctx, &knowledgetypes.QueryTrainingManifestBundleRequest{Id: "m-hack2"})
	require.True(t, fixedBundle.MerkleRootValid, "bundle re-verifies post-correction")

	// ── STRUCTURED REMEDIATION REFERENCE ──
	// Iter 2 improvement: remediation reference is structured now.
	_, err = ms.RecordRemediation(h.Ctx, &knowledgetypes.MsgRecordRemediation{
		Authority: authority, IncidentId: "HACK-002",
		Type:      knowledgetypes.RemediationType_REMEDIATION_TYPE_STATE_CORRECTION,
		Reference: "CorrectManifestMerkleRoot:m-hack2",
		Note:      "recomputed root via surgical msg",
	})
	require.NoError(t, err)

	_, err = ms.UnpauseModule(h.Ctx, &knowledgetypes.MsgUnpauseModule{
		Authority: authority, ModuleName: "knowledge",
	})
	require.NoError(t, err)

	_, err = ms.ResolveIncident(h.Ctx, &knowledgetypes.MsgResolveIncident{
		Authority: authority, IncidentId: "HACK-002",
		PostMortemUri: "ipfs://Qm.../HACK-002.md",
	})
	require.NoError(t, err)

	// ── Idempotency check: running the correction on a CLEAN manifest is a no-op ──
	// A second correction attempt (with the incident now RESOLVED) must reject
	// (open-incident invariant). Verifies the audit-trail binding is enforced.
	_, err = ms.CorrectManifestMerkleRoot(h.Ctx, &knowledgetypes.MsgCorrectManifestMerkleRoot{
		Authority: authority, ManifestId: "m-hack2", IncidentId: "HACK-002",
	})
	require.Error(t, err, "cannot run correction against a resolved incident")
	require.Contains(t, err.Error(), "not open")
}

// ─── Iteration 2 negative cases — the correction handler's safety props ──
func TestHackDrill_Iter2_CorrectionSafety(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	authority := h.KnowledgeKeeper.GetAuthority()

	// Setup a clean manifest.
	operator := testAddr("hack2safe_op").String()
	require.NoError(t, h.KnowledgeKeeper.SetTrainingPipeline(h.Ctx, &knowledgetypes.TrainingPipeline{
		Id: "pipe-safe", OperatorAddress: operator, TokenizerVersion: 1,
	}))
	_, err = ms.CreateTrainingManifest(h.Ctx, &knowledgetypes.MsgCreateTrainingManifest{
		Creator: operator, Id: "m-safe", PipelineId: "pipe-safe",
		CorpusSelector: &knowledgetypes.CorpusSelector{},
	})
	require.NoError(t, err)
	_, err = ms.FinalizeTrainingManifest(h.Ctx, &knowledgetypes.MsgFinalizeTrainingManifest{
		Creator: operator, ManifestId: "m-safe",
	})
	require.NoError(t, err)

	// SAFETY 1: authority gate.
	_, err = ms.CorrectManifestMerkleRoot(h.Ctx, &knowledgetypes.MsgCorrectManifestMerkleRoot{
		Authority: testAddr("not_auth").String(), ManifestId: "m-safe", IncidentId: "X",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unauthorized")

	// SAFETY 2: incident_id required.
	_, err = ms.CorrectManifestMerkleRoot(h.Ctx, &knowledgetypes.MsgCorrectManifestMerkleRoot{
		Authority: authority, ManifestId: "m-safe",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "incident_id required")

	// SAFETY 3: unknown incident rejected.
	_, err = ms.CorrectManifestMerkleRoot(h.Ctx, &knowledgetypes.MsgCorrectManifestMerkleRoot{
		Authority: authority, ManifestId: "m-safe", IncidentId: "NONEXISTENT",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")

	// SAFETY 4: running against CLEAN manifest is a no-op (was_corrupted=false).
	_, err = ms.OpenIncident(h.Ctx, &knowledgetypes.MsgOpenIncident{
		Authority: authority, Id: "INC-NOOP",
		Severity: knowledgetypes.IncidentSeverity_INCIDENT_SEVERITY_P3,
		Title: "noop test",
	})
	require.NoError(t, err)
	resp, err := ms.CorrectManifestMerkleRoot(h.Ctx, &knowledgetypes.MsgCorrectManifestMerkleRoot{
		Authority: authority, ManifestId: "m-safe", IncidentId: "INC-NOOP",
	})
	require.NoError(t, err, "no-op correction succeeds without mutating state")
	require.False(t, resp.WasCorrupted, "clean manifest reports was_corrupted=false")
	require.Equal(t, resp.PriorRoot, resp.RecomputedRoot)

	// SAFETY 5: expected_root mismatch aborts.
	_, err = ms.CorrectManifestMerkleRoot(h.Ctx, &knowledgetypes.MsgCorrectManifestMerkleRoot{
		Authority: authority, ManifestId: "m-safe", IncidentId: "INC-NOOP",
		ExpectedRecomputedRoot: "definitely-not-right",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "expected_recomputed_root mismatch")
}

// ─── Iteration 3: SLA dashboard + second attack scenario ───────────────
//
// ATTACK:
//   A compromised model owner inflates their ContributionRecord.fact_ids
//   with facts they never trained on. This inflates computed_tvw and
//   would extract higher revenue share when the fund distributes. The
//   existing MsgChallengeContribution requires a 5 ZRN bond from any
//   interested party — but the attacker hopes nobody will pay to
//   challenge for small-value inflations.
//
// RESPONSE:
//   - Security monitor notices statistical anomaly (sudden 10x jump in
//     a model's computed_tvw with no corresponding training activity).
//   - P1 incident opened; SLA target 2,880 blocks.
//   - Simulate: operator DOES NOT respond within SLA (pathological case).
//   - Advance blocks past SLA target.
//   - The new SlaBreachedIncidents query surfaces it — the ops team gets
//     paged.
//   - Authority then applies a remediation (in this case, resolving via
//     ChallengeContribution + ResolveContributionChallenge; the attacker
//     forfeits the over-reported portion via the challenge mechanism).
//   - Incident resolves; SLA-breach flag stamped in audit.
//
// EXPECTED FRICTION: the incident record doesn't currently carry a
// persistent "was_sla_breached" flag — only the instantaneous query
// reveals it. For audit completeness, the ResolveIncident event already
// emits sla_met; the record itself could also stamp it. Minor; deferred.
func TestHackDrill_Iter3_SLABreachSurfacing(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	qs := knowledgekeeper.NewQueryServerImpl(h.KnowledgeKeeper)
	authority := h.KnowledgeKeeper.GetAuthority()

	// Open a P3 incident (30-day default SLA ~518k blocks — too long for
	// this test, so use a custom short SLA window).
	_, err = ms.OpenIncident(h.Ctx, &knowledgetypes.MsgOpenIncident{
		Authority: authority, Id: "SLA-BREACH-001",
		Severity: knowledgetypes.IncidentSeverity_INCIDENT_SEVERITY_P3,
		Title:    "Attribution over-report suspected",
		Description: "Statistical anomaly in model-X TVW; operator hasn't acknowledged yet",
		AffectedModules: []string{"knowledge"},
		SlaWindowBlocks: 5, // custom short window for test
	})
	require.NoError(t, err)

	// Pre-breach: query returns empty (SLA not yet missed).
	preBreach, err := qs.SlaBreachedIncidents(h.Ctx, &knowledgetypes.QuerySlaBreachedIncidentsRequest{})
	require.NoError(t, err)
	require.Empty(t, preBreach.Incidents, "SLA not yet breached")

	// Advance past the SLA target.
	h.AdvanceBlocks(10)

	// Post-breach: the SLA-dashboard surfaces it.
	postBreach, err := qs.SlaBreachedIncidents(h.Ctx, &knowledgetypes.QuerySlaBreachedIncidentsRequest{})
	require.NoError(t, err)
	require.Len(t, postBreach.Incidents, 1, "SLA-breached incident appears on the dashboard")
	require.Equal(t, "SLA-BREACH-001", postBreach.Incidents[0].Id)
	require.Greater(t, postBreach.CurrentBlockHeight, postBreach.Incidents[0].SlaTargetBlock,
		"query reports current vs target so alerts can compute late-by-N-blocks")

	// Apply a remediation late — the incident can still be resolved, but the
	// SLA breach is visible in the audit (via the resolve event's sla_met=false).
	_, err = ms.RecordRemediation(h.Ctx, &knowledgetypes.MsgRecordRemediation{
		Authority: authority, IncidentId: "SLA-BREACH-001",
		Type: knowledgetypes.RemediationType_REMEDIATION_TYPE_DOCUMENTATION,
		Reference: "late-response",
	})
	require.NoError(t, err)
	_, err = ms.ResolveIncident(h.Ctx, &knowledgetypes.MsgResolveIncident{
		Authority: authority, IncidentId: "SLA-BREACH-001",
		PostMortemUri: "ipfs://Qm.../SLA-BREACH-001.md",
	})
	require.NoError(t, err)

	// Post-resolve: SLA dashboard is empty again.
	postResolve, err := qs.SlaBreachedIncidents(h.Ctx, &knowledgetypes.QuerySlaBreachedIncidentsRequest{})
	require.NoError(t, err)
	require.Empty(t, postResolve.Incidents, "resolved incidents drop off SLA dashboard")
}

// ─── Iteration 3b: second hack scenario — attribution over-report ──────
//
// ATTACK:
//   Model owner lists FACT-UNUSED in their ContributionRecord. They never
//   trained on it. This inflates computed_tvw and will claim unearned
//   revenue when distributed.
//
// RESPONSE via existing mechanisms (no new code needed — iter 3b is the
// convergence check that iter 2 fixed enough):
//   1. Anyone posts MsgChallengeContribution with 5 ZRN bond.
//   2. Authority resolves the challenge — "uphold".
//   3. Challenger receives bond × 2; model owner's over-report recorded.
//   4. Parallel to challenge: incident log records the governance action.
//   5. Resolve.
//
// The chain recovered without a single new msg. This is convergence:
// the existing primitives covered the case.
func TestHackDrill_Iter3b_AttributionOverReport(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	authority := h.KnowledgeKeeper.GetAuthority()

	ownerAddr := testAddr("hack3_owner")
	owner := ownerAddr.String()
	challengerAddr := testAddr("hack3_challenger")
	challenger := challengerAddr.String()
	require.NoError(t, h.FundAccount(challengerAddr, sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewInt(20_000_000)))))

	require.NoError(t, h.KnowledgeKeeper.SetTrainingPipeline(h.Ctx, &knowledgetypes.TrainingPipeline{
		Id: "pipe-hack3b", OperatorAddress: owner, TokenizerVersion: 1,
	}))
	require.NoError(t, h.KnowledgeKeeper.SetModelCard(h.Ctx, &knowledgetypes.ModelCard{
		Id: "model-hack3b", PipelineId: "pipe-hack3b", OwnerAddress: owner,
		Route: "from_scratch", Active: true,
	}))
	// Legitimate fact.
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
		Id: "F-REAL-3B", Content: "real", Domain: "sciences",
		Status: knowledgetypes.FactStatus_FACT_STATUS_ACTIVE, Submitter: owner,
		MethodId: knowledgetypes.MethodologyEmpirical, Confidence: 900_000,
	}))
	// Fact that the owner falsely claims to have used.
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
		Id: "F-UNUSED", Content: "never used in training", Domain: "sciences",
		Status: knowledgetypes.FactStatus_FACT_STATUS_ACTIVE, Submitter: owner,
		MethodId: knowledgetypes.MethodologyEmpirical, Confidence: 900_000,
		CorroborationCount: 10, // high weight → attractive to falsely claim
	}))

	// ── ATTACK ──
	_, err = ms.AttributeContributions(h.Ctx, &knowledgetypes.MsgAttributeContributions{
		Owner: owner, ModelId: "model-hack3b",
		FactIds: []string{"F-REAL-3B", "F-UNUSED"}, // F-UNUSED is the over-report
	})
	require.NoError(t, err)
	rec, _ := h.KnowledgeKeeper.GetContributionRecord(h.Ctx, "model-hack3b")
	preRaidTVW := rec.ComputedTvw
	require.Greater(t, preRaidTVW, uint64(0),
		"over-reported fact inflated the TVW; attack succeeded at attribution time")

	// ── RESPONSE ──
	// 1. Incident opened (audit trail of governance action).
	_, err = ms.OpenIncident(h.Ctx, &knowledgetypes.MsgOpenIncident{
		Authority: authority, Id: "HACK-ATTRIB-001",
		Severity: knowledgetypes.IncidentSeverity_INCIDENT_SEVERITY_P2,
		Title: "Suspected attribution over-reporting on model-hack3b",
		AffectedModules: []string{"knowledge"},
	})
	require.NoError(t, err)

	// 2. Challenger files challenge (5 ZRN bond).
	_, err = ms.ChallengeContribution(h.Ctx, &knowledgetypes.MsgChallengeContribution{
		Challenger: challenger, ModelId: "model-hack3b",
		DisputedFactId: "F-UNUSED", DisputeType: "fraudulent",
		Evidence: "verification prompt + response shows F-UNUSED wasn't in training",
		Id: "chal-hack3b",
	})
	require.NoError(t, err)

	// 3. Authority resolves the challenge — upheld (attacker lied).
	resp, err := ms.ResolveContributionChallenge(h.Ctx, &knowledgetypes.MsgResolveContributionChallenge{
		Resolver: authority, ChallengeId: "chal-hack3b", Uphold: true,
		Note: "confirmed: F-UNUSED never in model-hack3b's training corpus",
	})
	require.NoError(t, err)
	require.Equal(t, "10000000", resp.PayoutToWinner, "challenger collects bond × 2")

	// 4. Record remediation on incident.
	_, err = ms.RecordRemediation(h.Ctx, &knowledgetypes.MsgRecordRemediation{
		Authority: authority, IncidentId: "HACK-ATTRIB-001",
		Type:      knowledgetypes.RemediationType_REMEDIATION_TYPE_STATE_CORRECTION,
		Reference: "ContributionChallenge:chal-hack3b",
		Note:      "challenge upheld; attacker bond-slash & contributor paid",
	})
	require.NoError(t, err)

	// 5. Resolve incident.
	_, err = ms.ResolveIncident(h.Ctx, &knowledgetypes.MsgResolveIncident{
		Authority: authority, IncidentId: "HACK-ATTRIB-001",
		PostMortemUri: "ipfs://Qm.../HACK-ATTRIB-001.md",
	})
	require.NoError(t, err)

	// Challenger is whole plus reward: started with 20M, paid 5M bond, got 10M back = 25M.
	finalBal := h.GetBalance(challengerAddr, "uzrn")
	require.Equal(t, sdkmath.NewInt(25_000_000), finalBal.Amount,
		"economic invariant: challenger earned net 5M; attacker implicitly penalised")

	// ── CONVERGENCE ──
	// This drill required no new msg handlers. The existing Wave 4 challenge
	// mechanism + Wave 11 incident log + Wave 12 pause + Wave 13 correction
	// covered the scenario. If iter 4 reveals no new gaps, we've converged.
}

// ─── Iteration 4: convergence check — novel third attack ────────────────
//
// ATTACK:
//   A compromised sponsor opens a bounty for 100M uzrn escrow. Three
//   Sybil verifier addresses (known gap from Wave 9) rapidly push a
//   DRIFT variant through as EQUIVALENT. The payout fires; the sponsor
//   drains their own escrow back to themselves via the compromised
//   submitter account. Net: chain's integrity is hit; training fund
//   accounting is unaffected (the escrow was the sponsor's money to
//   begin with) BUT the attacker now has a falsely-accepted variant in
//   the training corpus — poisoning downstream training runs.
//
// RESPONSE:
//   1. Monitor notices anomaly (rapid verdict finalization, unusual
//      vote timing).
//   2. Incident opened (P1 — integrity issue).
//   3. Module paused.
//   4. Authority uses CorrectManifestMerkleRoot on any manifest that
//      already pulled the drift variant into its included set (there
//      are none in this test; production would enumerate via
//      TrainingManifests query).
//   5. Unpause.
//   6. Resolve + close.
//
// CONVERGENCE CRITERION:
//   The test runs to completion using only primitives introduced in
//   Waves 1–13. No new msg handlers, no new query surface, no new
//   proto types required. If this holds, the incident-response
//   pipeline has converged for the current attack class.
func TestHackDrill_Iter4_VerifierSybilDriftPoisoning(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
	require.NoError(t, err)

	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	qs := knowledgekeeper.NewQueryServerImpl(h.KnowledgeKeeper)
	authority := h.KnowledgeKeeper.GetAuthority()

	sponsorAddr := testAddr("hack4_sponsor")
	sponsor := sponsorAddr.String()
	submitter := testAddr("hack4_sub").String()
	require.NoError(t, h.FundAccount(sponsorAddr, sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewInt(500_000_000)))))

	// Setup: target fact + bounty + drift variant.
	require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
		Id: "F-HACK4", Content: "target", Domain: "sciences",
		Status: knowledgetypes.FactStatus_FACT_STATUS_ACTIVE, Submitter: sponsor,
		MethodId: knowledgetypes.MethodologyEmpirical, Confidence: 900_000,
	}))
	_, err = ms.CreateAugmentationBounty(h.Ctx, &knowledgetypes.MsgCreateAugmentationBounty{
		Sponsor: sponsor, Id: "b-hack4", TargetFactId: "F-HACK4",
		RewardPerVariant: 10_000_000, MaxVariants: 1,
	})
	require.NoError(t, err)
	_, err = ms.SubmitAugmentation(h.Ctx, &knowledgetypes.MsgSubmitAugmentation{
		Submitter: submitter, Id: "aug-hack4", BountyId: "b-hack4",
		OriginalFactId: "F-HACK4",
		VariantContent: "semantically-changed variant (drift) — attacker pushes as EQUIVALENT",
	})
	require.NoError(t, err)

	// ── ATTACK: Sybil consensus pushes DRIFT through as EQUIVALENT ──
	for _, v := range []string{"hack4_sybil1", "hack4_sybil2", "hack4_sybil3"} {
		_, err := ms.VoteOnAugmentation(h.Ctx, &knowledgetypes.MsgVoteOnAugmentation{
			Verifier: testAddr(v).String(), AugmentationId: "aug-hack4",
			Vote: knowledgetypes.AugmentationVerdict_AUGMENTATION_VERDICT_EQUIVALENT,
		})
		require.NoError(t, err, "Sybil succeeds pre-Wave-10 (documented gap)")
	}
	// Attack succeeded.
	aug, _ := h.KnowledgeKeeper.GetAugmentation(h.Ctx, "aug-hack4")
	require.Equal(t, knowledgetypes.AugmentationVerdict_AUGMENTATION_VERDICT_EQUIVALENT, aug.Verdict)
	require.True(t, aug.Accepted)

	// ── RESPONSE ──
	_, err = ms.OpenIncident(h.Ctx, &knowledgetypes.MsgOpenIncident{
		Authority: authority, Id: "HACK-SYBIL-001",
		Severity: knowledgetypes.IncidentSeverity_INCIDENT_SEVERITY_P1,
		Title:    "Sybil consensus poisoned aug-hack4 (DRIFT variant accepted as EQUIVALENT)",
		Description: "Verifier panel gap: 3 addresses, same-actor probable. Contaminated variant should not propagate to training manifests.",
		AffectedModules: []string{"knowledge"},
	})
	require.NoError(t, err)

	// Pause the module.
	_, err = ms.PauseModule(h.Ctx, &knowledgetypes.MsgPauseModule{
		Authority: authority, ModuleName: "knowledge",
		Reason:     "HACK-SYBIL-001: stop further Sybil exploitation",
		IncidentId: "HACK-SYBIL-001",
	})
	require.NoError(t, err)

	// While paused, downstream writes are blocked — so no manifest can be
	// created that would include this poisoned variant.
	operator := testAddr("hack4_op").String()
	require.NoError(t, h.KnowledgeKeeper.SetTrainingPipeline(h.Ctx, &knowledgetypes.TrainingPipeline{
		Id: "pipe-hack4", OperatorAddress: operator, TokenizerVersion: 1,
	}))
	_, err = ms.CreateTrainingManifest(h.Ctx, &knowledgetypes.MsgCreateTrainingManifest{
		Creator: operator, Id: "m-poisoned", PipelineId: "pipe-hack4",
		CorpusSelector: &knowledgetypes.CorpusSelector{IncludeContrastivePairs: true},
	})
	require.Error(t, err, "breaker blocks manifest creation → poisoned variant can't propagate")

	// Remediation: record the pause itself, then the future Wave that will
	// deliver stake-weighted Sybil resistance — for now the remediation is
	// documentation.
	_, err = ms.RecordRemediation(h.Ctx, &knowledgetypes.MsgRecordRemediation{
		Authority: authority, IncidentId: "HACK-SYBIL-001",
		Type:      knowledgetypes.RemediationType_REMEDIATION_TYPE_STATE_CORRECTION,
		Reference: "ModulePause:knowledge",
	})
	require.NoError(t, err)
	_, err = ms.RecordRemediation(h.Ctx, &knowledgetypes.MsgRecordRemediation{
		Authority: authority, IncidentId: "HACK-SYBIL-001",
		Type:      knowledgetypes.RemediationType_REMEDIATION_TYPE_DOCUMENTATION,
		Reference: "route_b_waves_remaining/stake-weighted-verifier-panel",
		Note:      "Sybil-resistance Wave scheduled; this incident documents the exposure window",
	})
	require.NoError(t, err)

	// Unpause.
	_, err = ms.UnpauseModule(h.Ctx, &knowledgetypes.MsgUnpauseModule{
		Authority: authority, ModuleName: "knowledge",
	})
	require.NoError(t, err)

	// Resolve + close.
	_, err = ms.ResolveIncident(h.Ctx, &knowledgetypes.MsgResolveIncident{
		Authority: authority, IncidentId: "HACK-SYBIL-001",
		PostMortemUri: "ipfs://Qm.../HACK-SYBIL-001.md",
	})
	require.NoError(t, err)
	_, err = ms.CloseIncident(h.Ctx, &knowledgetypes.MsgCloseIncident{
		Authority: authority, IncidentId: "HACK-SYBIL-001",
	})
	require.NoError(t, err)

	// ── CONVERGENCE ASSERTION ──
	// Every handler called in this test was already present before iter 4.
	// No new msg handler added. No new query surface required. No new
	// proto types. The response pipeline covered a novel attack scenario
	// with the primitives that already existed.
	//
	// Verifiable clean state: zero open incidents, zero paused modules.
	open, err := qs.OpenIncidents(h.Ctx, &knowledgetypes.QueryOpenIncidentsRequest{})
	require.NoError(t, err)
	require.Empty(t, open.Incidents)
	paused, err := qs.PausedModules(h.Ctx, &knowledgetypes.QueryPausedModulesRequest{})
	require.NoError(t, err)
	require.Empty(t, paused.Paused)
	breached, err := qs.SlaBreachedIncidents(h.Ctx, &knowledgetypes.QuerySlaBreachedIncidentsRequest{})
	require.NoError(t, err)
	require.Empty(t, breached.Incidents)
}
