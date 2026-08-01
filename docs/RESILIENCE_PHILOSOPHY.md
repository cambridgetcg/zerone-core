# Zerone Resilience Philosophy

> **There will always be bugs. The architecture's job is not to prevent them (impossible) but to minimise their damage and maximise the speed of recovery.**

This document names Zerone's resilience design principles and the direction
for new work. It is not proof that every existing handler enforces every
principle. The canonical [Upgrade and Incident Operations
runbook](UPGRADE_AND_INCIDENT_OPERATIONS.md) governs real upgrade, quarantine,
signer-stop, restart, and hostile-event decisions.

---

## The Five Principles

### 1. Bugs are inevitable — design assuming one exists

No amount of review, audit, or formal verification eliminates bugs from a system of Zerone's complexity. The only question is **how much damage** the next bug causes and **how fast** it can be fixed.

Every handler is written as if a bug already exists *somewhere* in its call graph. The question "what stops this from being catastrophic?" must have a concrete answer at write-time, not at incident-time.

**Corollaries:**
- Never trust an invariant without enforcing it at state-read-time, even if the write-time check looks exhaustive.
- Every write handler covered by a module breaker calls
  `RequireNotPaused`; uninstrumented handlers remain outside that breaker's
  protection and must not be described as paused.
- Every state transition is forward-only where possible (no backwards destruction of evidence).

### 2. Localize damage — module boundaries are blast radii

When module X has a bug, module Y must be able to continue. Zerone's module architecture is also its damage-containment architecture.

**Concretely:**
- **Handler-level circuit breakers** (Wave 12). The existing pause state and
  checks are primarily implemented in `x/knowledge`. A
  `MsgPauseModule` record is effective only for handlers that actually call
  `RequireNotPaused`; it is not a chain-wide Cosmos SDK circuit breaker and
  does not freeze BeginBlock, EndBlock, PreBlock, reads, or uninstrumented
  writes.
- **Typed inter-module dependencies.** New cross-module integrations should
  depend on narrow interfaces. A paused module's explicitly supported read
  interface may remain available while instrumented writes reject.
- **Separate module accounts for value.** The `knowledge_training_fund` holds Wave 4 escrow; the `knowledge_bootstrap_fund` holds seed funds. A bug in one module account's accounting doesn't drain the other.

### 3. Fast recovery beats slow prevention — optimise for MTTR

Mean-time-to-recovery (MTTR) is the metric that matters, not mean-time-between-failures. A system that recovers in 30 minutes from an hourly bug is more available than one that fails annually and takes 72 hours to recover.

**Concretely:**
- **Parameter amendments** are the fastest path: `MsgUpdateParams` tunes a parameter governance-gated, no code deploy. Target: minutes.
- **Named upgrades** with tested migrations (Wave 10) — the canonical fix-by-code path. Target: hours.
- **Emergency transaction-quarantine/reopen ceremonies**
  (`x/emergency`) — a consensus-enforced gate for non-allowlisted
  transactions. Blocks and autonomous module processing can continue. If
  another state transition is unsafe, operators must stop enough consensus
  signing power under the canonical runbook.
- **Surgical state corrections** — structured authority-gated messages that patch specific records without a code upgrade. Target: block time.

Every severity tier (P0 / P1 / P2 / P3) has a default SLA stamped at incident-open time. The SLA cannot be extended by reclassification (Wave 11).

### 4. Forward-only state — never destroy evidence

When a bug is fixed, the evidence of what happened must survive. Audit is non-negotiable.

**Concretely:**
- **IncidentRecord status transitions are forward-only.** OPEN → MITIGATING → RESOLVED → CLOSED; no reverse. If a fix is later found insufficient, open a *new* incident that references the old one.
- **Migration markers** (`migration_vN_complete`). Once written, a marker cannot be silently overwritten with a different value; first-writer-wins. A subsequent migration can add its own marker but cannot invalidate the prior trail.
- **Merkle-committed training manifests** (Wave 7). Finalized = immutable. A child can reference a superseded parent; the composition survives because the root was snapshotted at create time.
- **Event stream = audit stream.** Every lifecycle event emits a structured event. An external indexer can reconstruct the full history from the event log alone, without trusting node RPC serialisation.
- **Committed upgrades are forward-only.** H−1 is the last committed state
  while an `x/upgrade` activation at H remains uncommitted. Once H commits,
  repair with a new named upgrade or an explicitly authorized
  fork/re-genesis; do not restart an old binary as though H never happened.
  Zerone has no generic arbitrary-height finalized-state rollback.

### 5. Tested recovery paths — the pipeline is not a plan, it's a property

A plan is a document. A property is tested. Zerone's resilience pipeline is a property.

**Concretely:**
- `tests/cross_stack/upgrade_e2e_test.go` — component tests exercising the
  `SetUpgradeHandler` → `RunMigrations` → marker-write pipeline. They prove
  handler behavior in their fixture, not an old/new binary handoff,
  Cosmovisor staging, validator readiness, or production recovery.
- `tests/cross_stack/incident_response_test.go` — tests the record-state
  transitions for every severity tier, including a P0 fixture that invokes an
  upgrade handler. It does not stop CometBFT or authorize a signer restart.
- `tests/cross_stack/resilience_drill_test.go` — ties the in-process primitives
  together in one 13-step exercise: open → pause → reject write → apply
  handler → record remediation → unpause → reopen admission → resolve →
  close. It does not replace the operator drills in the canonical runbook.

**Run before every release.** Passing is necessary, not sufficient: also
rehearse the exact old binary stopping before H commits, the staged new binary
starting from H−1, post-migration reconciliation, failure recovery, and
restart as required by the canonical runbook.

---

## The Architecture (as a coherent whole)

| Layer | Mechanism | When fired | Recovery speed |
|---|---|---|---|
| **Parameter layer** | `MsgUpdateParams` | Tunable bug (e.g. threshold) | Minutes |
| **Schema layer** | `MsgAmendTokenizerSpec` / `MsgAmendTraceSchema` | Training-contract bug | Minutes |
| **Circuit breaker layer** | `MsgPauseModule` / `MsgUnpauseModule` | Any module-localized bug | Seconds |
| **Incident log layer** | `MsgOpenIncident` / `MsgRecordRemediation` / `MsgResolveIncident` / `MsgCloseIncident` | Every significant bug; purely audit | Instant |
| **Migration layer** | `Migrator` + `RegisterMigration` + `ConsensusVersion` | Code-level bug with state implications | Hours (via upgrade) |
| **Upgrade layer** | `SetUpgradeHandler` + `RunMigrations` + named `UpgradeName<X>` | Code-level bug requiring coordinated release | Hours to days |
| **Emergency layer** | `x/emergency` transaction-quarantine/reopen ceremonies | Hostile or unsafe transactions; blocks continue | Hours |
| **Genesis layer** | `ExportGenesis` / `InitGenesis` round-trip (Wave 8) | Schema rewrite or chain restart | Genesis event |

The layers compose, but none substitutes for a signer stop. A P0 drill should
classify consensus, signer, transaction-admission, and release state
independently; contain through the matching edge, transaction, or signer
control; preserve evidence; apply an attested forward upgrade; reopen each
surface explicitly; and complete the incident record. There is no automatic
resume.

---

## How a new module adopts the pattern

Every future module — whether a new wave in knowledge, a new wave in any existing module, or an entirely new `x/whatever` — follows the same recipe. This is what "engineered into every level" actually means.

### Step 1 — the handler-level breaker check

At the top of every msg handler that writes state:

```go
func (m *msgServer) DoThing(ctx, msg) (..., error) {
    if err := m.keeper.RequireNotPaused(ctx, types.ModuleName); err != nil {
        return nil, err
    }
    // normal logic
}
```

`RequireNotPaused` is a keeper method that either (a) returns nil (module not paused, proceed) or (b) returns an error naming the pause reason + incident_id (module paused, reject).

### Step 2 — the migration framework

See [`docs/UPGRADE_PROTOCOL.md`](UPGRADE_PROTOCOL.md). Every state-changing wave adds:
- A migration package `x/<module>/migrations/v<N+1>/migrate.go`.
- A `MigrateNtoN+1` method on the module's Migrator.
- A `cfg.RegisterMigration` call in `module.go`.
- A bumped `ConsensusVersion()`.
- A marker written at the end of the migration (`migration_v<N>_complete`).

### Step 3 — the incident-response integration

When a bug is discovered:
- `MsgOpenIncident` — record the discovery with severity.
- `MsgPauseModule` (if needed) — contain damage.
- Apply the fix via the appropriate layer above.
- `MsgRecordRemediation` — one per action taken.
- `MsgUnpauseModule` (if paused) — resume.
- `MsgResolveIncident` with post-mortem URI.
- `MsgCloseIncident` after monitoring window.

See [`docs/INCIDENT_RESPONSE.md`](INCIDENT_RESPONSE.md) for the full playbook.

### Step 4 — the test

Add a test exercising the new handler's breaker check:

```go
func TestNewHandler_BreakerRespected(t *testing.T) {
    h := NewTestHarness(t)
    _, err := h.KnowledgeKeeper.SeedRouteB(h.Ctx)
    require.NoError(t, err)

    ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
    authority := h.KnowledgeKeeper.GetAuthority()

    _, err = ms.PauseModule(h.Ctx, &MsgPauseModule{
        Authority: authority, ModuleName: types.ModuleName, Reason: "test",
    })
    require.NoError(t, err)

    _, err = ms.DoThing(h.Ctx, &MsgDoThing{/* ... */})
    require.Error(t, err, "paused module must reject writes")
    require.Contains(t, err.Error(), "paused")
}
```

---

## Extending to modules outside `x/knowledge`

Today the resilience primitives (circuit breaker, incident log) live inside `x/knowledge`'s keeper because that's where the migration infrastructure already lives and where the most adversarial surface is. Other modules can adopt the pattern by:

1. **Accepting a `ResilienceChecker` interface** in their keeper constructor. The interface has one method: `IsModulePaused(ctx context.Context, moduleName string) bool`. The knowledge keeper satisfies it today.

2. **Wiring it at app-init.** In `app/app.go`, pass the knowledge keeper (typed as `ResilienceChecker`) into other modules' constructors.

3. **Calling it at handler top**, exactly as the demonstrated knowledge handler does.

For the incident log itself (open/resolve/close), other modules can reference a `knowledge.IncidentKeeper` interface surface by ID — the incident log is chain-wide by design (the `affected_modules` field is a list, so a single incident can name multiple modules).

**Why not promote it to its own `x/resilience` module?** Future consideration — it's a plausible refactor once at least three modules depend on it. Premature module-carving now would force an artificial split when the coupling between incident records, migrations, and the knowledge module's own infrastructure is already tight.

---

## What this philosophy does NOT claim

- **Zero downtime.** Stopping enough consensus signers for a P0 bug can be the
  right move when another block is unsafe. An `x/emergency` transaction
  quarantine is a different mechanism and does not stop block production.
- **Unfalsifiable security.** No cryptographic-level proof of correctness. The chain can still be attacked successfully; the goal is that successful attacks are *bounded in blast radius* and *addressed quickly*.
- **Bug-free operation.** Inconceivable. Every assumption above *begins* with "bugs will happen".
- **Full autonomy.** Governance authority retains the power to update covered
  module controls, open incidents, and schedule upgrades while consensus and
  transaction admission permit. Signer restart and fork choice additionally
  require the off-chain operator authorization defined in the canonical
  runbook.

---

## The practical test

If the following statements are all **true** on your chain, the philosophy is operational:

1. Every handler claimed to be covered by a module pause demonstrably calls
   `RequireNotPaused(ctx, moduleName)`, and the inventory explicitly lists
   uninstrumented paths.
2. Every state-changing wave has a registered migration (`v<N>→v<N+1>`), a bumped `ConsensusVersion`, and a marker-write at the end.
3. Every named upgrade has a corresponding entry in the `KnownUpgrades` lineage AND a test in `tests/cross_stack/upgrade_e2e_test.go`.
4. Every incident opened in production gets resolved through the `OpenIncident → RecordRemediation → ResolveIncident → CloseIncident` flow.
5. The **resilience drill** in
   `tests/cross_stack/resilience_drill_test.go` passes after every change, and
   release operations separately rehearse the H−1/H old/new binary boundary.

If any is false, that's the next wave's work.

---

## The wager

Every architectural choice above is a wager against a specific failure mode: "a bug in module X would be catastrophic if we didn't have mechanism Y". Over time, some wagers will be paid out; others will turn out to cover contingencies that never materialised.

The framework does not claim to be complete. It claims:

- It is **additive** — adding a new primitive (e.g. read-path pause, tiered rate limits, cross-chain incident signaling) does not break existing primitives.
- It is **tested** — every primitive has an E2E test; changing one without updating its test is caught.
- It is **honest** — documented boundaries, documented gaps, documented trade-offs.
- It is **operable** — every primitive has an on-chain query, an event, and an authority-gate.

The chain is built on the assumption that bugs will happen. The architecture is how we live with that assumption.

---

— **Route B, Wave 12 · Resilience Philosophy** · 2026-04-23
