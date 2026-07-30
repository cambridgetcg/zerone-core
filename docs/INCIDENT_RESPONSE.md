# Zerone Incident Response Protocol

> **The pipeline from "a bug is found on a live chain" to "the chain is fixed,
> documented, and auditable."** Use this incident-record reference with the
> canonical [Upgrade and Incident Operations
> runbook](UPGRADE_AND_INCIDENT_OPERATIONS.md), which controls all operational
> meanings of *halt*, *quarantine*, *resume*, and *recovery*. The
> [Upgrade Protocol](UPGRADE_PROTOCOL.md) covers the code-side migration
> recipe.

Incident response is recorded on-chain when consensus is available. Every
incident gets a structured `IncidentRecord` logging its severity, every
remediation attached to it, and the lineage linking each action to the
concrete mechanism (upgrade, param amendment, emergency ceremony, schema
change, documentation). If consensus is stalled or transaction admission
blocks incident messages, begin a signed external operations manifest and
anchor the record on-chain after safe admission is restored. The on-chain log
is an audit surface; it does not itself stop or restart consensus.

---

## Severity taxonomy

| Severity | Meaning | Default SLA (blocks @ 5s) | Typical remediation |
|---|---|---:|---|
| **P0** | Consensus break, hostile state transition, key compromise, or data loss. The chain cannot continue committing safely. | 720 (~1h) | Signer stop and/or transaction quarantine → attested named upgrade or explicit fork → authorized restart/reopen |
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
| `EMERGENCY_HALT` | `ceremony_id` | `x/emergency` `MsgProposeHalt` ceremony. On success it enables application transaction quarantine; block production and autonomous module processing can continue. |
| `EMERGENCY_RESUME` | `ceremony_id` | `x/emergency` `MsgProposeResume` ceremony. On success it reopens application transaction admission after explicit recovery authorization; it does not restart a stopped signer. |
| `STATE_CORRECTION` | `msg_type_url` | Authority-gated structured msg that patches specific records. Reserved for surgical use. |
| `SCHEMA_AMENDMENT` | `<schema>@v<N>` | `MsgAmendTokenizerSpec` / `MsgAmendTraceSchema` — version-tracked governance change. |
| `DOCUMENTATION` | `post_mortem_uri` | No on-chain state change. Records publication of a post-mortem. |

All remediations except documentation touch chain state through
**already-existing**, **already-tested** mechanisms. The incident log is the
coordination layer; it does not grant new powers, choose a canonical history,
or authorize a signer restart. Zerone does not provide a generic mechanism to
rewrite arbitrary finalized state. Committed-state repair must be a
deterministic forward upgrade or an explicitly authorized fork/re-genesis.

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

### P0 — unsafe consensus or state transition

**Preconditions:** consensus has stopped, conflicting state is suspected, a
signer may be compromised, or producing another block may cause harm. Report
the consensus, signer, transaction-admission, and release axes separately;
never infer one from another.

1. **Open a signed operations manifest and preserve evidence.** Record UTC
   time, last observed height, block hash, app hash, commit, signer state,
   binary digest, and independent RPC observations. Do not delete the
   consensus WAL or `priv_validator_state.json`.
2. **Contain the actual failure mode.**
   - Stop a suspect signer immediately. A chain-wide consensus stop requires
     enough voting power to prevent a commit and confirmation from independent
     observations.
   - On the current 1/1 networks, a consensus key that may have been copied
     cannot safely authorize its own in-chain replacement: every transition
     block would still depend on the suspect key. Preserve the signer and use
     the [one-validator key-replacement decision
     tree](UPGRADE_AND_INCIDENT_OPERATIONS.md#1141-one-validator-key-replacement);
     a possible sole-key compromise selects its signed fork/re-genesis branch.
   - Use edge controls for hostile public traffic.
   - Use an `x/emergency` halt ceremony only when peer-injected transactions
     must be rejected by application consensus logic. This is
     `CONSENSUS_QUARANTINE`; blocks, timers, rewards, PreBlock, BeginBlock, and
     EndBlock can continue.
3. **Open the on-chain incident when admission and consensus permit.**

   ```text
   MsgOpenIncident(
     authority, id="ZR-YYYY-NNNN",
     severity=P0, title, description, reporter,
     affected_modules
   )
   ```

   If it cannot be submitted safely, retain it in the manifest and anchor it
   after recovery.
4. **Record each containment action accurately.** Record
   `EMERGENCY_HALT` only for a successful transaction-quarantine ceremony;
   record an operator signer stop in the external manifest rather than
   describing it as that ceremony.
5. **Design and attest the recovery.** Follow the canonical runbook. Rehearse
   the exact release from the last common committed state, independently
   verify artifacts, reconcile supply and IBC, and obtain the required
   recovery authorization.
6. **Activate the recovery.**

   - If a safe chain can still process governance, schedule the attested
     forward upgrade with sufficient operational lead time. Do not improvise
     an unreviewed in-place state rewrite because the incident is urgent.
   - While application transaction quarantine is active, use standard SDK
     governance v1's `MsgSubmitProposal` with `expedited=true` and exactly one
     message: either
     `/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade` or
     `/cosmos.upgrade.v1beta1.MsgCancelUpgrade`. Votes, weighted votes, and
     deposits are admitted only for an existing proposal with that exact
     shape. Mixed batches, legacy content proposals, custom Zerone upgrade
     LIPs, authz wrappers, emergency messages inside governance, and unrelated
     expedited actions remain quarantined. See the canonical proposal shape
     in [UPGRADES.md](UPGRADES.md) and the
     [Cosmos SDK app-upgrade guide](https://docs.cosmos.network/sdk/v0.53/build/building-apps/app-upgrade).
     At each quarantined EndBlock, the app scans only the inactive/active
     deadline range that upstream governance would otherwise process at that
     block time. It permanently marks due non-recovery proposals `FAILED`,
     removes their queue entries, and refunds their deposits. This catches an
     ordinary proposal admitted before a halt and due in the halt block
     without making the halt block scan every future proposal. Future-deadline
     proposals remain frozen in their queues. Inventory and review them before
     resume; explicitly cancel an underlying scheduled upgrade through the
     exact recovery lane or account for the proposal in the recovery design.
     Resume does not erase retained proposals, and upstream governance may
     process them when they later become due. Monitor the bounded aggregate
     `zerone.gov.proposals_quarantined` event described in
     [EVENTS.md](EVENTS.md).
   - For an already scheduled H that has not committed, continue from H−1
     with a corrected binary for the agreed named upgrade. An authorized,
     coordinated `--unsafe-skip-upgrades H` is an exceptional H−1 fork
     choice, not a rollback.
   - Once H has committed, the migration is canonical. Repair forward with a
     new named upgrade, or use the explicit fork/re-genesis process. Never
     restart the old binary as if H had not happened.
7. **Restart signers and reopen admission independently.** Signer restart
   requires the signed recovery manifest and operator quorum. A successful
   `MsgProposeResume` only reopens application transaction admission and must
   bind to the independently anchored, verified `RECOVERY_READY` transition
   head. A resume cannot be proposed in the halt-finalization block. For each
   active halt and proposer, the first accepted attempt is recovery generation
   1; a retry is possible only after that ceremony fails, a later block begins,
   and the recovery-manifest digest changes. There is no automatic resume.
8. **Verify before reopening traffic.** Compare app hashes and commits,
   module versions, supply, module accounts, IBC state, artifact digests, and
   functional probes. Reopen IBC only after its separate reconciliation gate.
9. **Record the named upgrade, quarantine reopen, and documentation
   remediations; publish the post-mortem; resolve, then close after the
   monitoring window.**

The one-hour SLA is a response target, not authority to skip evidence,
artifact, quorum, or state-reconciliation gates. Practice both the planned
H−1/H handoff and hostile-event recovery on testnet.

### P1 — immediate fix via param amendment

**When:** bug is in a parameterized mechanism (e.g., verifier-panel consensus
threshold, TVW multiplier, augmentation expiry fee) and consensus can safely
continue. No consensus-signer stop is presumed.

1. **Open incident** at P1.
2. **Governance proposes `MsgUpdateParams`** with the corrective value.
3. **Apply** once passed.
4. **Record remediation** of type `PARAM_AMENDMENT` with `Params.<FieldName>=<newValue>`.
5. **Resolve** with post-mortem URI.

**Target: end-to-end in 4 hours.** Often closed same-day.

### P1 — expedited named upgrade

When a param amendment can't express the fix (e.g., logic bug, not a tunable):

1. **Open** at P1.
2. **Register a named upgrade** ahead of the normal release schedule.
3. **Standard SDK governance v1** submits an expedited
   `MsgSubmitProposal` containing exactly one
   `/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade` with the reviewed plan name,
   height, and `info`. The legacy `SoftwareUpgradeProposal` content type and
   custom Zerone upgrade LIPs are not executable authorities. Use exactly one
   `/cosmos.upgrade.v1beta1.MsgCancelUpgrade` instead when cancelling a
   scheduled plan before H.
4. **Validators** deploy the new binary before the upgrade height.
5. **Activate at H.** An old binary without the named handler stops before H
   commits, leaving H−1 as the last committed state. The staged new binary
   starts from H−1, runs the handler at H, and may commit H. If the running
   binary already has the handler there may be no visible process stop; no
   `x/emergency` quarantine is required.
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

**Important invariant:** before a `NAMED_UPGRADE` remediation can be credibly
recorded, the corresponding handler MUST pass its component tests and the
exact release MUST pass the canonical old/new binary H−1/H rehearsal.
`TestUpgrade_LineageParityWithHandlers` checks registration and advertisement;
`TestUpgrade_V3ToV4KnowledgeMigrationPipeline` or its analogue checks handler
behavior in a fixture. Neither alone proves the validator handoff.

---

## What this protocol does NOT handle

- **Discovery.** How a bug is found (monitoring, audit, user report, adversarial testing) is upstream. The incident log records what happened after discovery.
- **Validator-level coordination.** Consensus signer stop/restart is an
  off-chain operator action governed by the signed operations manifest.
  `x/emergency` ceremonies control application transaction admission only.
  The incident log references ceremony IDs; it does not drive either action.
- **Private incidents.** Security-sensitive incidents may need to be kept confidential until remediated. This protocol is public-by-default; a "sealed" flag deferring publication is a future extension (Wave 12+ consideration).
- **Automated remediation.** Every remediation is currently manual (human operator + governance authority). Auto-remediation for well-understood patterns is a future extension.
- **Cross-chain incidents.** IBC state issues require coordination between chains. This protocol handles single-chain incidents; cross-chain ones need both chains' incident logs to reference each other by id.

---

## The test guard

`tests/cross_stack/incident_response_test.go` exercises:

- **P0 record scenario end-to-end** — open → transaction-quarantine reference
  → upgrade reference → handler run → admission-reopen reference →
  documentation → resolve → close. This test does not prove a CometBFT
  consensus stop, old/new binary handoff, signer restart, or production
  recovery authorization.
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
- Any transaction-quarantine ceremony is proposed (open a P0 record when it is
  safe to submit; this is not automatic).

Not every bug needs an incident. A failing lint, a confusing error message, a doc typo — log these in the issue tracker and fix them in the next release without the ceremony. **Incidents are for bugs whose response is visible on-chain.**

---

— **Route B, Wave 11 · Incident Response Protocol** · 2026-04-23
