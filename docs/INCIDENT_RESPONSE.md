# Zerone Incident Response Protocol

> **The pipeline from "a bug is found on a live chain" to "the chain is fixed, documented, and auditable."** Coupled tightly with the [Upgrade Protocol](UPGRADE_PROTOCOL.md) — this document is what you follow when something breaks in production.

Incident response is engineered on-chain. Every incident gets a structured `IncidentRecord` logging its severity, every remediation attached to it, and the lineage linking each action to the concrete mechanism (upgrade, param amendment, emergency ceremony, schema change, documentation). The state transitions are strict; authority-gating is pervasive; the SLA is stamped at open-time and cannot drift.

---

## Severity taxonomy

| Severity | Meaning | Default SLA (blocks @ 5s) | Typical remediation |
|---|---|---:|---|
| **P0** | Consensus break / chain halt / data loss. Chain cannot continue producing blocks safely. | 720 (~1h) | `EMERGENCY_HALT` → `NAMED_UPGRADE` → `EMERGENCY_RESUME` |
| **P1** | High-impact bug requiring immediate fix. Chain functions but a mechanism is broken. | 2,880 (~4h) | `PARAM_AMENDMENT` (fast) or `NAMED_UPGRADE` (emergency release) |
| **P2** | Correctness or architectural bug; scheduled fix acceptable. | 120,960 (~7d) | `NAMED_UPGRADE` on next release / `SCHEMA_AMENDMENT` |
| **P3** | Low-impact; documentation or next-release fix. | 518,400 (~30d) | `DOCUMENTATION` / `STATE_CORRECTION` for isolated cases |

The SLA window is frozen at open time. Reclassifying severity later does not change the measured SLA for that incident — this prevents the playbook from being gamed ("downgrade to buy more time").

---

## The five remediation types

| Type | Reference field | Mechanism |
|---|---|---|
| `PARAM_AMENDMENT` | `Params.<FieldName>=<value>` | `MsgUpdateParams` (governance authority). Fastest path — no code deploy. |
| `NAMED_UPGRADE` | `upgrade_name` | `UpgradeKeeper.SetUpgradeHandler` + `ModuleManager.RunMigrations`. See [UPGRADE_PROTOCOL.md](UPGRADE_PROTOCOL.md). |
| `EMERGENCY_HALT` | `ceremony_id` | `x/emergency` module's `MsgProposeHalt` + validator-voted ceremony. Stops block production. |
| `EMERGENCY_RESUME` | `ceremony_id` | `x/emergency` module's `MsgProposeResume`. Resumes production after hotfix binary rolls. |
| `STATE_CORRECTION` | `msg_type_url` | Authority-gated structured msg that patches specific records. Reserved for surgical use. |
| `SCHEMA_AMENDMENT` | `<schema>@v<N>` | `MsgAmendTokenizerSpec` / `MsgAmendTraceSchema` — version-tracked governance change. |
| `DOCUMENTATION` | `post_mortem_uri` | No on-chain state change. Records publication of a post-mortem. |

All remediations except documentation touch chain state through **already-existing**, **already-tested** mechanisms. The incident log is the coordination layer; it does not grant new powers, it only names which power was used and why.

---

## The state machine

```
OPEN ──record remediation──▶ MITIGATING ──resolve──▶ RESOLVED ──close──▶ CLOSED
 │                                 │                      │                │
 └─ triage only ─────────────┴─ fixes applied ──┴─ monitoring ──┴─ post-mortem archived
```

- **OPEN → MITIGATING**: automatic on the first `RecordRemediation` call.
- **MITIGATING → RESOLVED**: requires at least one remediation and a `post_mortem_uri`.
- **RESOLVED → CLOSED**: after monitoring window; purely archival.
- **No backward transitions.** An incident never reverts status. If a fix is later found insufficient, open a **new** incident referencing the original in its description.

Strict forward-only transitions mean the audit trail always tells one coherent story per incident.

---

## Response playbook

### P0 — chain halt

**Preconditions:** block production has stopped OR will stop imminently.

1. **Halt the chain.** Validators run `x/emergency.MsgProposeHalt`; quorum votes; chain halts at next ceremony-stamped height.
2. **Open the incident.**
   ```
   MsgOpenIncident(
     authority, id="ZR-YYYY-NNNN",
     severity=P0, title, description, reporter,
     affected_modules
   )
   ```
3. **Record the halt remediation.**
   ```
   MsgRecordRemediation(type=EMERGENCY_HALT, reference="ceremony-halt-42", ...)
   ```
4. **Develop the fix binary.** Standard SDLC, but fast.
5. **Register the named upgrade** following [UPGRADE_PROTOCOL.md](UPGRADE_PROTOCOL.md) §1-5. Make sure `TestUpgrade_*` passes against the new binary before proposing.
6. **Record the upgrade remediation.**
   ```
   MsgRecordRemediation(type=NAMED_UPGRADE, reference="v1.0.2-hotfix", ...)
   ```
7. **Governance votes on the upgrade** and the resume ceremony. Binary is deployed on all validators.
8. **Resume the chain.** `MsgProposeResume`; chain resumes on new binary; upgrade handler fires.
9. **Record the resume remediation.** 
10. **Publish the post-mortem** (IPFS preferred).
11. **Record documentation remediation** with `post_mortem_uri`.
12. **Resolve** the incident with the `post_mortem_uri`.
13. **Close** after the monitoring window.

**Target: end-to-end in 1 hour.** On testnet, practice this flow quarterly.

### P1 — immediate fix via param amendment

**When:** bug is in a parameterized mechanism (e.g., verifier-panel consensus threshold, TVW multiplier, augmentation expiry fee). No chain halt needed.

1. **Open incident** at P1.
2. **Governance proposes `MsgUpdateParams`** with the corrective value.
3. **Apply** once passed.
4. **Record remediation** of type `PARAM_AMENDMENT` with `Params.<FieldName>=<newValue>`.
5. **Resolve** with post-mortem URI.

**Target: end-to-end in 4 hours.** Often closed same-day.

### P1 — emergency upgrade (no halt)

When a param amendment can't express the fix (e.g., logic bug, not a tunable):

1. **Open** at P1.
2. **Register a named upgrade** ahead of the normal release schedule.
3. **Governance** proposes `SoftwareUpgradeProposal` with a near-term height.
4. **Validators** deploy the new binary before the upgrade height.
5. **Upgrade fires automatically** at the proposed height. No halt.
6. **Record remediation** of type `NAMED_UPGRADE`.
7. **Resolve** + close.

### P2 — scheduled upgrade

1. **Open** at P2.
2. **Queue fix** for the next scheduled release.
3. **Ship upgrade** through normal release cadence.
4. **Record remediation** of type `NAMED_UPGRADE` or `SCHEMA_AMENDMENT`.
5. **Resolve** + close.

### P3 — documentation / next-release

1. **Open** at P3.
2. Typically a misleading doc, a suboptimal default, or a cosmetic bug.
3. **Record** a `DOCUMENTATION` remediation or queue for next release.
4. **Resolve** + close.

---

## Observability

### Operator dashboard

Primary query: `OpenIncidents` — returns every `OPEN` or `MITIGATING` incident, optionally filtered by severity. Wire this to an alerting system; fire pages on any `P0` remaining in the queue past its SLA target.

```
gRPC → zerone.knowledge.v1.Query/OpenIncidents
     ↓
Returns IncidentRecord[]: [id, severity, status, title, sla_target_block, remediations…]
```

### Full history

`Incidents` returns all incidents (filterable by severity/status) for auditing, for post-mortem aggregation, and for governance reporting.

### Per-incident detail

`Incident(id)` returns one record with its full remediation lineage, linking back to upgrades, params, ceremonies, and schemas.

---

## Event stream

Every step emits a structured event. An external indexer can reconstruct the full incident lifecycle from the event log alone — the critical property for audit and regulatory transparency:

- `zerone.knowledge.incident_opened`
- `zerone.knowledge.incident_remediation_recorded`
- `zerone.knowledge.incident_resolved`
- `zerone.knowledge.incident_closed`

See [EVENTS.md](EVENTS.md) for the attribute schema.

---

## Coupling with the upgrade protocol

The incident record **references** the named upgrade that fixed the bug. The upgrade itself remains a first-class object — tested via `TestUpgrade_*` in `tests/cross_stack/upgrade_e2e_test.go`, registered in `app/upgrades.go`, documented in `UPGRADE_PROTOCOL.md`. The incident log's job is to name WHY the upgrade was needed, not to replace the upgrade itself.

**Important invariant:** before a `NAMED_UPGRADE` remediation can be credibly recorded, the corresponding upgrade handler MUST be tested end-to-end. The test `TestUpgrade_LineageParityWithHandlers` ensures the upgrade is registered and advertised; the operator should also run `TestUpgrade_V3ToV4KnowledgeMigrationPipeline` or its analogue against the specific upgrade name before deploying.

---

## What this protocol does NOT handle

- **Discovery.** How a bug is found (monitoring, audit, user report, adversarial testing) is upstream. The incident log records what happened after discovery.
- **Validator-level coordination.** Quorum-gated halt and resume ceremonies live in `x/emergency`. The incident log references their ceremony IDs; it doesn't drive the vote.
- **Private incidents.** Security-sensitive incidents may need to be kept confidential until remediated. This protocol is public-by-default; a "sealed" flag deferring publication is a future extension (Wave 12+ consideration).
- **Automated remediation.** Every remediation is currently manual (human operator + governance authority). Auto-remediation for well-understood patterns is a future extension.
- **Cross-chain incidents.** IBC state issues require coordination between chains. This protocol handles single-chain incidents; cross-chain ones need both chains' incident logs to reference each other by id.

---

## The test guard

`tests/cross_stack/incident_response_test.go` exercises:

- **P0 scenario end-to-end** — open → halt → upgrade reference → actual upgrade run → resume → documentation → resolve → close.
- **P1 scenario** — param amendment hotfix verifying the amendment is actually live.
- **P2 scenario** — schema amendment via `MsgAmendTraceSchema`.
- **Non-authority rejection** — every handler.
- **Resolve requires remediation** — cannot close an incident with zero actions recorded.
- **Cannot close before resolve** — strict state transitions.
- **Dashboard queries** — `OpenIncidents` correctly excludes resolved/closed; severity filter narrows.
- **SLA tracking preserved** — target block frozen at open-time; remediation doesn't shift it.

Run before every release. If this passes, the mechanism works; if a new response pattern is needed, add its test here.

---

## When to open an incident (the threshold)

Any of:

- A mechanism is producing incorrect outputs (confidence-scoring bug, migration regression, economic invariant violation).
- A param's value causes demonstrable misalignment (e.g., Sybil vulnerability because threshold is too low).
- A fact's corroboration or calibration history is provably incorrect because of a prior bug.
- A training manifest's Merkle commitment can be shown invalid.
- An attestation cannot be reconciled with its bound manifest.
- Any halt ceremony is proposed (automatically log the P0).

Not every bug needs an incident. A failing lint, a confusing error message, a doc typo — log these in the issue tracker and fix them in the next release without the ceremony. **Incidents are for bugs whose response is visible on-chain.**

---

— **Route B, Wave 11 · Incident Response Protocol** · 2026-04-23
