# Substrate Bridge (`x/substrate_bridge`) — Design Spec

**Status:** Brainstormed and ready for implementation planning.

**Inception:** 2026-05-10.

**Position in the architecture:** Tier-1 foundation for external recursive work modules. Sits between the Phase-1 `x/work` primitive (work-class registry + lifecycle) and external work classes (`x/translation`, `x/curriculum`, `x/hypothesis_market`, etc.). Every external work class registers with `x/substrate_bridge` for adapter, substrate-link, and lineage primitives.

**Doctrinal alignment:** Operationalizes UW + M2 (substrate-link mandate) + M3 (class-specific verification) + M5 (recursion-weight axes) + M6 (lineage propagates and recurses) for external work. Does not amend `docs/USEFUL_WORK.md`.

---

## 1. Module identity

**Name:** `x/substrate_bridge`

**Tagline:** *The chain absorbs external compute through this gate.*

**Module-level position-layer declaration** (for `x/substrate_bridge/doc.go`):

> `x/substrate_bridge` is the one place external work meets ZERONE substrate. Three responsibilities are unified here because they share the same lifecycle and the same on-chain state graph: (a) **adapter framework** — typed converters from external sources to internal attestations, (b) **substrate-link compiler** — deterministic provenance from external content to ToK fact-IDs (existing or pending), (c) **cross-class lineage propagator** — bidirectional royalty flow as work compounds across classes. The module is the load-bearing edge between ZERONE and everywhere else; every external work class registers with it.

### File layout

```
x/substrate_bridge/
├── doc.go                                      # position-layer declaration
├── module.go                                   # AppModule wiring
├── types/
│   ├── adapter.go                              # AdapterRegistration proto wrapper, AdapterStatus
│   ├── substrate_link.go                       # SubstrateLink proto wrapper, validation helpers
│   ├── lineage.go                              # LineageEdge proto wrapper, propagation rules
│   ├── attestation.go                          # ExternalAttestation + AttestationStatus state machine
│   ├── keys.go                                 # store-key prefixes (0x80-0x8F reserved)
│   ├── errors.go                               # typed errors for refusal-layer messages
│   └── codec.go
├── keeper/
│   ├── keeper.go                               # core keeper struct, store key, codec
│   ├── adapter_registry.go                     # gov-gated adapter registration / suspension / tombstone
│   ├── substrate_link.go                       # link compilation verification, storage, hash check
│   ├── attestation.go                          # submit, transition, settle
│   ├── pending_fact_index.go                   # reverse index pending_claim_id → []attestation_id
│   ├── lineage.go                              # forward + backward walks, depth/decay caps
│   ├── settlement.go                           # eager settle on resolution; partial/rejected paths
│   ├── msg_server.go                           # MsgRegisterAdapter (gov), MsgSuspendAdapter (gov),
│   │                                           # MsgTombstoneAdapter (gov), MsgSubmitExternalAttestation
│   ├── grpc_query.go                           # query adapter registry, link, lineage, attestation status
│   ├── begin_block.go                          # scan AWAITING_RESOLUTION for resolution + timeout
│   └── events.go                               # voice-layer event constants
├── client/cli/
│   ├── query.go
│   └── tx.go
```

### Cross-module integration

- `x/work` (Phase 1 primitive) calls into `x/substrate_bridge` when an external work class is submitted; substrate-link compilation happens at commit phase; settlement is held by `x/substrate_bridge` until pending facts resolve.
- `x/knowledge` notifies `x/substrate_bridge` on every `claim_resolved` event so the pending-fact index can resolve waiting links. Implemented via direct keeper-call hook in `x/knowledge`'s `CompleteRound`.
- `x/creed` registers the doctrine for the `CategoryAdapterRegistration` LIP class (mirrors M3 governance flow).
- `x/qualification` is queried at attestation submit to enforce per-adapter qualification requirements.

### Out of scope (deferred to Tier 2 / Tier 3)

- Off-chain compute (running models, fetching URLs, replicating training) → Tier 2 `x/offchain_workers`
- External-event resolution / oracle consensus → Tier 2 `x/event_resolution`
- Cross-chain message export (other chains depending on ZERONE truth) → Tier 3 `x/interchain_knowledge`
- Work-class lifecycle (commit/reveal/verify/settle phases) → `x/work` Phase 1
- The actual external work classes (`x/translation`, `x/curriculum`, etc.) — each gets its own brainstorm/spec cycle

---

## 2. Permissive substrate-link semantics

### Link shape — two sections, one hash

```protobuf
message SubstrateLink {
  // Existing ToK nodes this external content cites.
  // Every fact_id MUST exist in x/knowledge at commit time (verified by
  // x/substrate_bridge at MsgSubmitExternalAttestation).
  repeated FactCitation cited_facts = 1;

  // New claims this external content introduces. Auto-submitted to
  // x/knowledge as Claims at commit; attestation held in
  // AWAITING_RESOLUTION until they verify. M2 satisfied: every pending
  // claim becomes a real on-chain claim with full provenance.
  repeated PendingClaim pending_claims = 2;

  // Per-axis recursion-weight projection (M5). Compiler output.
  AxisProjection recursion_weight = 3;

  // Adapter that produced this link.
  string adapter_id = 4;

  // External source typed reference.
  ExternalSource source = 5;

  // sha256 of deterministic canonical form. Required for re-derivability.
  bytes link_hash = 6;
}

message FactCitation {
  string fact_id = 1;
  CitationType citation_type = 2;
  string citation_context = 3;  // optional excerpt for audit
}

message PendingClaim {
  // Same shape as x/knowledge.Claim; auto-submitted at commit phase.
  string claim_content = 1;
  string proposed_fact_id = 2;  // optional; chain assigns if empty
  string domain = 3;
  repeated ClaimRelation relations = 4;
  string methodology_id = 5;
}

message AxisProjection {
  uint64 axis_substrate      = 1;
  uint64 axis_verification   = 2;
  uint64 axis_classification = 3;
  uint64 axis_attribution    = 4;
  uint64 axis_tooling        = 5;
  uint64 axis_interface      = 6;
}

message ExternalSource {
  string adapter_id     = 1;
  string source_id      = 2;  // e.g. Wikipedia article ID
  string source_url     = 3;  // optional, for audit
  bytes  content_hash   = 4;
  uint64 fetched_at_block = 5;
}
```

### Attestation state machine

```
                    MsgSubmitExternalAttestation
                              │
                              ▼
                       SUBMITTED  ← link compiled, bond locked, qualified
                              │
                              ▼  commit-phase OK
                       COMMITTED  ← pending claims auto-submitted to x/knowledge
                              │
                              ▼
                AWAITING_RESOLUTION
                      │   │      │
       all verified   │   │      │ timeout reached
       (or ≥ floor)   │   │      │ > rejection threshold
                      ▼   ▼      ▼
                  READY  PARTIAL  REJECTED
                    │      │        │
                    ▼      ▼        ▼
                 SETTLED  SETTLED  SLASHED
              (full reward)(partial)(bond gone)
```

**Settlement is eager** (at the BeginBlocker that detects the resolution event), not lazy. This keeps the system simple and predictable for submitters.

### Reward formula under partial settlement

```
verified_count    = number of pending_claims that resolved VERIFIED
total_count       = len(pending_claims) + len(cited_facts)
verified_ratio    = (verified_count + len(cited_facts)) / total_count

L (substrate-link weight) = base_L × verified_ratio
W (recursion-weight)      = AxisProjection recomputed against verified subset only
Q (verification-quality)  = average consensus_margin across verified claims

R = base + L × W × Q
```

A 90%-good translator earns ~90% of the recursion-weighted reward + flat base.
A 30%-good translator may trip the rejection threshold and lose their bond entirely.

### Anti-spam parameters (gov-tunable)

| Parameter | Purpose | Default |
|---|---|---|
| `max_pending_claims_per_attestation` | Cap bulk size | 100,000 |
| `per_pending_claim_bond_uzrn` | Per-claim spam cost | 222 uzrn |
| `attestation_min_bond_uzrn` | Floor regardless of size | 222,000 uzrn |
| `max_pending_window_blocks` | Timeout for resolution | ~6 months (≈6.2M blocks at 2.5s) |
| `pending_claim_rejection_threshold_bps` | Whole-attestation reject threshold | 5000 (50%) |
| `min_verified_ratio_for_settle_bps` | Minimum verified to allow settle | 1000 (10%) |

### Idempotency / deduplication

Pending claim canonicalized as `sha256(domain || methodology_id || canonical_content)`. If a claim with the same hash already exists in `x/knowledge` (verified or pending), the attestation's `pending_claim` is upgraded to a `cited_fact` pointing at the existing claim. Prevents bulk translators from racing on identical content.

---

## 3. Adapter framework + gov-gated registration

### Adapter operator model — no separate role

An "adapter" is a recipe (binary hash + axis bounds + bond requirements), not a service. The chain doesn't run adapter binaries; the chain registers what a binary's output must look like. Anyone who runs the registered binary AND submits the resulting attestation is the submitter and earns via the UW formula. No canonical operator; no operator compensation channel; no centralization risk.

### AdapterRegistration proto

```protobuf
message AdapterRegistration {
  string adapter_id  = 1;          // canonical, gov-approved (e.g. "wikipedia-en-v1")
  string source_type = 2;          // "wikipedia" | "arxiv" | "ibc_packet" | "iot_mqtt" | ...
  string version     = 3;          // semver

  // Determinism guarantee. Validators run this binary and re-derive
  // the SubstrateLink; if their re-derivation doesn't match, refuse.
  // Binary fetched by hash from a registered distribution channel
  // (IPFS CID, GitHub release, gov-approved registry).
  bytes compiler_binary_hash = 4;

  // Per-axis recursion-weight bounds. Submitter's claims must fall
  // within these. M5 enforcement at the doctrinal boundary.
  AxisBounds axis_bounds = 5;

  // Bond requirements applied per-attestation and per-pending-claim.
  string min_attestation_bond_uzrn = 6;
  string min_per_claim_bond_uzrn   = 7;

  // Slash gradient by failure mode (mirrors M1's graduated slashing).
  SlashGradient slash_gradient = 8;

  // Submitters must hold this x/qualification domain at min status.
  string required_qualification_domain = 9;
  QualificationStatus min_qualification_status = 10;

  // Work classes allowed to use this adapter. Empty = any.
  repeated string allowed_class_ids = 11;

  // Lifecycle.
  AdapterStatus status              = 12;  // ACTIVE | SUSPENDED | TOMBSTONED
  string registered_via_lip_id      = 13;
  uint64 registered_at_block        = 14;
  uint64 tombstoned_at_block        = 15;  // 0 if not tombstoned
}

enum AdapterStatus {
  ADAPTER_STATUS_UNSPECIFIED  = 0;
  ADAPTER_STATUS_ACTIVE       = 1;
  ADAPTER_STATUS_SUSPENDED    = 2;
  ADAPTER_STATUS_TOMBSTONED   = 3;
}

message AxisBounds {
  uint64 axis_substrate_max      = 1;
  uint64 axis_verification_max   = 2;
  uint64 axis_classification_max = 3;
  uint64 axis_attribution_max    = 4;
  uint64 axis_tooling_max        = 5;
  uint64 axis_interface_max      = 6;
}

message SlashGradient {
  uint32 compiler_drift_bps = 1;  // adapter binary hash mismatch — full slash
  uint32 axis_overflow_bps  = 2;  // axis claim exceeds bounds — pro-rata
  uint32 fraud_bps          = 3;  // >threshold rejected — full slash + close
}
```

### Lifecycle

```
                  Author drafts adapter spec
                            │
                            ▼
       MsgSubmitProposal (CategoryAdapterRegistration LIP class)
                            │
                            ▼
                Standard gov voting + Creed Council quorum
                            │
                            ▼
                    ACTIVE
                       │       │
                       │       └─→ Gov vote (fast incident)
                       │              │
                       │              ▼
                       │         SUSPENDED ←─→ ACTIVE
                       │              │
                       │              ▼ (gov)
                       │         TOMBSTONED (permanent)
                       │
                       └─→ Gov vote (planned retirement)
                                      │
                                      ▼
                                 TOMBSTONED
```

- **REGISTER**: `CategoryAdapterRegistration` LIP class. Gov + Creed Council quorum. Bond posted by proposer; refunded on pass, slashed on bad-faith submission.
- **SUSPEND** (fast): incident response. New attestations refused; in-flight settle. Can be restored.
- **TOMBSTONE** (slow): permanent retirement. Forward-only (commitment 10); adapter_id never reused.

### Qualification integration

Adapter registration specifies the `x/qualification` domain submitters must hold. Mirrors how PoT validators specialize by domain, applied to external work submitters. Without qualification at min status, the chain refuses the attestation with refusal-layer message: *"Submission refused — submitter lacks `<domain>` qualification at status `<min>` (UW + M3)"*.

Connects external work to accuracy-decay: a submitter whose attestations get rejected at scale loses qualification status, reducing future earning power.

### What the on-chain framework needs

```go
// AdapterRegistry surface
func (k Keeper) RegisterAdapter(ctx, reg *AdapterRegistration) error  // gov-authority only
func (k Keeper) SuspendAdapter(ctx, adapterID, reason string) error    // gov-authority only
func (k Keeper) TombstoneAdapter(ctx, adapterID string) error          // gov-authority only
func (k Keeper) GetAdapter(ctx, adapterID string) (*AdapterRegistration, bool)
func (k Keeper) IterateAdapters(ctx, cb func(*AdapterRegistration) bool)

// AttestationVerify (called from msg_server's SubmitExternalAttestation)
func (k Keeper) VerifyAttestationConforms(ctx, att *ExternalAttestation) error
```

Verification checks: (1) adapter exists + ACTIVE, (2) work class allowed, (3) submitter qualified, (4) axis projection within bounds, (5) bond sufficient, (6) link hash valid. Typed errors at each failure (refusal layer).

---

## 4. Cross-class lineage propagation (M6 generalized)

### LineageEdge as a first-class record

```protobuf
message LineageEdge {
  string upstream_attestation_id   = 1;
  string downstream_attestation_id = 2;
  string upstream_class_id         = 3;
  string downstream_class_id       = 4;

  CitationType citation_type       = 5;
  uint32 contribution_share_bps    = 6;  // submitter-claimed share within budget
  uint32 depth_from_downstream     = 7;  // 1 for direct cite; +1 per propagation hop
  uint64 created_at_block          = 8;
  bytes  settlement_payment_uzrn   = 9;  // total paid via this edge to date
}

enum CitationType {
  CITATION_TYPE_UNSPECIFIED = 0;
  CITATION_TYPE_CITES       = 1;  // 1× base weight
  CITATION_TYPE_SUPPORTS    = 2;  // 2× base weight
  CITATION_TYPE_EXTENDS     = 3;  // 3× base weight
  CITATION_TYPE_REFINES     = 4;  // 3× base weight
  CITATION_TYPE_GENERALIZES = 5;  // 4× base weight
}
```

### Storage shape

```
0x80 | edge_id                                          → LineageEdge
0x81 | upstream_attestation_id   | edge_id              → 1 (forward index)
0x82 | downstream_attestation_id | edge_id              → 1 (backward index)
0x83 | attestation_id                                   → cumulative lineage uzrn earned
```

Forward walk: scan `0x81` filtered by upstream_id. Backward walk: scan `0x82` filtered by downstream_id.

### Propagation

```
Downstream attestation D settles with reward R
              │
              ▼
    lineage_budget = R × LINEAGE_SHARE_BPS  (default 30%)
              │
              ▼
  For each upstream cite U_i:
    share_i = lineage_budget
            × contribution_share_bps_i
            × citation_type_weight_i
            (clamped to remaining budget)
    Pay share_i to U_i's submitter.
    Write LineageEdge(U_i, D, depth=1).
              │
              ▼
  Recursive upstream propagation on each U_i:
    propagated_share = share_i × DECAY_BPS_PER_HOP  (default 30%)
    Apply same distribution logic to U_i's own upstream cites.
              │
              ▼
  Halt when depth ≥ max_propagation_depth (default 5)
       OR propagated_share < min_propagation_uzrn (default 1000 uzrn).
```

**Lineage paid from downstream reward, not new mint.** No inflation pressure. M4 formula unchanged.

### Cycle prevention via timestamp DAG

```
Invariant: LineageEdge(U, D) is valid only if U.created_at_block < D.created_at_block.
```

Enforced at edge creation in settlement. O(1) per edge. No expensive cycle-detection walk.

### Recursion amplification — revenue-stream semantics

The doctrine's "load-bearing facts compound in value as their downstream work amplifies the chain" is realized as:

- **Static**: an attestation's recursion-weight `W` is computed at its own settlement; never retroactively adjusted.
- **Dynamic**: an attestation's TOTAL LIFETIME EARNINGS grow as downstream work proliferates. The `LineageRoyaltyAccumulator` at `0x83` increases monotonically.

A fact originally earning `base + L × W × Q = 222 uzrn` can end up earning 222M uzrn over time if 100K downstream attestations cite it. Settled `W` stays; load-bearing value (cumulative revenue stream) grows. Queryable.

### Anti-spam / anti-gaming

- **Citation-type weights bounded by proto enum**: malicious adapters can't fabricate weight types.
- **`contribution_share_bps` sums to 10000 per attestation**: enforced at compile time.
- **`min_propagation_uzrn` floor**: prevents dust-attack lineage chains.
- **Self-citation allowed but `contribution_share_bps` hard-capped at 5000 (50%)** when `upstream.submitter == downstream.submitter`. Legitimate self-builds permitted; self-funneling prevented.

---

## 5. Integration with existing modules

### `x/work` (Phase 1) coordination

Phase 1's `x/work` primitive owns the work-class registry and the four-phase lifecycle. `x/substrate_bridge` registers as a *consumer* of `x/work`'s lifecycle hooks:

- **commit phase**: `x/work` calls `x/substrate_bridge.PrepareExternalAttestation` to validate adapter + qualification + bond + link hash, then auto-submit pending claims.
- **reveal phase**: no `x/substrate_bridge` involvement; external work has nothing to reveal beyond what was committed.
- **verify phase**: typically delegated to `x/knowledge` PoT verification of pending claims (external work's "verification" is its pending claims passing).
- **settle phase**: `x/work` calls `x/substrate_bridge.SettleExternalAttestation` which applies the partial-settlement formula and triggers lineage propagation.

### `x/knowledge` hook

`x/knowledge.CompleteRound` gets one new line: after writing the round result, notify `x/substrate_bridge` via direct keeper-call:

```go
k.substrateBridgeKeeper.OnClaimResolved(ctx, claimID, verdict)
```

This is what triggers AWAITING_RESOLUTION → READY transitions.

### `x/creed` integration

Plan-0 adapter LIP registration uses `x/creed`'s LIP class machinery (the same machinery that gates creed amendments). Adds:

- New `CategoryAdapterRegistration` enum value in `x/creed/types`
- Quorum requirements: same as `CategoryCreedAmendment` (full Creed Council quorum)

### `x/qualification` query

At `MsgSubmitExternalAttestation`, the keeper calls:

```go
qual := k.qualificationKeeper.GetDomainQualification(ctx, submitter, adapter.RequiredQualificationDomain)
if qual.Status < adapter.MinQualificationStatus {
    return ErrInsufficientQualification
}
```

No new state in `x/qualification`; reuses the existing query surface.

### Voice-layer events introduced

```
external_attestation_submitted    - adapter_id, work_class_id, attestation_id, bond_uzrn, useful_work_commitment="UW", mechanism="M1,M2,M3"
external_attestation_committed     - attestation_id, pending_claim_count, mechanism="M2,M3"
external_attestation_settled       - attestation_id, reward_uzrn, verified_ratio_bps, mechanism="M4"
external_attestation_rejected      - attestation_id, rejection_reason, slash_uzrn, mechanism="M1"
external_attestation_partial       - attestation_id, reward_uzrn, verified_ratio_bps, mechanism="M1,M4"
adapter_registered                 - adapter_id, lip_id, useful_work_commitment="UW", mechanism="M3"
adapter_suspended                  - adapter_id, reason
adapter_tombstoned                 - adapter_id
lineage_edge_created               - upstream, downstream, citation_type, contribution_share_bps, mechanism="M6"
lineage_royalty_paid               - to_attestation, from_attestation, uzrn, depth, mechanism="M6"
```

### Refusal-layer messages

Every refusal cites UW + the violated mechanism. Examples:

- *"Submission refused — adapter `<id>` not in ACTIVE status (UW + M3)"*
- *"Submission refused — submitter lacks qualification `<domain>` at `<min_status>` (UW + M3)"*
- *"Submission refused — substrate-link missing required cited_fact_id `<id>` (UW + M2)"*
- *"Submission refused — axis_substrate exceeds adapter bound (UW + M5)"*
- *"Submission refused — pending_claim count `<n>` exceeds max_pending_claims_per_attestation (UW + M2)"*
- *"Edge creation refused — upstream attestation created after downstream (cycle prevention, UW + M6)"*

---

## 6. Five-layer enforcement plan

Same discipline as the other USEFUL_WORK bindings.

### Test layer

- `tests/cross_stack/substrate_bridge_test.go` — integration tests for: register adapter → submit attestation → resolve → settle → propagate lineage.
- Per-module tests in `x/substrate_bridge/keeper/*_test.go` for each keeper function.

### Position layer

- `x/substrate_bridge/doc.go` declares UW + which mechanisms it implements (M2, M3, M5, M6 partially; M1 via bond and slash; M4 via partial-settlement formula; M7 via fraud detection paths).

### Voice layer

- Events with `useful_work_commitment="UW"` attribute (listed in §5).
- Every `lineage_royalty_paid` event tagged `mechanism="M6"`.
- Every adapter lifecycle event tagged `mechanism="M3"`.

### Refusal layer

- Typed errors in `x/substrate_bridge/types/errors.go`, each with a doctrine-citing message (listed in §5).

### Graph layer

- The module's `doc.go` cross-references UW + M1–M7 it implements.
- The module's tests cross-reference the doctrinal requirement they verify.

---

## 7. Out of scope (deferred)

- **Tier 2 — `x/offchain_workers`**: TEE attestation for heavy off-chain compute. Adapters that fit on-chain (Wikipedia text, IBC packet parsing) don't need this. Adapters that require ML inference (e.g., LLM-based fact extraction) do.
- **Tier 2 — `x/event_resolution`**: time-locked oracles, multi-source consensus, dispute windows. Needed by hypothesis-market and oracle-attestation adapters; not by translation / curriculum.
- **Tier 3 — `x/interchain_knowledge`**: IBC packet types for exporting verified facts to other chains.
- **Specific work classes** — `x/translation`, `x/curriculum`, `x/hypothesis_market`, `x/oracle_attestation`, etc. Each gets its own brainstorm/spec cycle and registers with this module.

---

## 8. Open questions for the implementation plan

These are not doctrinal commitments; they are implementation choices for the Phase-1 plan:

- **Bond settlement timing**: refund per-claim bonds on each pending-claim verification, or wait for the whole attestation to settle?
- **Adapter binary distribution**: gov-approved registry vs. IPFS CIDs vs. GitHub releases — what's the canonical channel?
- **Lineage budget percentage**: 30% per hop default — is this the right starting point, or should the curve be parameterized by gov from day 1?
- **Self-citation cap**: 50% hard cap — should it vary by class, or by upstream-downstream relation?
- **Pending-fact deduplication strategy**: canonical hash collapse vs. allow competing claims for PoT to resolve — and how to handle near-duplicates (paraphrases)?
- **Genesis state**: does `x/substrate_bridge` ship with any default adapters at genesis, or does the chain start empty and require LIPs for every adapter?

---

## 9. What this is not

- **Not a marketplace overlay.** External work registers via gov-gated adapters and pays via M4. Trust-deliberate; not a free-for-all.
- **Not a Phase-1 prerequisite**. `x/work` Phase 1 can ship before `x/substrate_bridge`; internal-only work classes don't need this module.
- **Not a doctrine amendment.** Doesn't change UW or any of M1–M7. Operationalizes M2/M3/M5/M6 for external work, mirrors M1's slash gradient, mirrors M4's reward formula.
- **Not complete.** Tier 2 and Tier 3 follow when their consumer modules are designed. The three sub-systems (adapter, link, lineage) ship together because they're deeply interlinked; later infra modules ship separately.

---

## 10. The discipline

Before merging a change that touches `x/substrate_bridge` or any registered adapter:

1. Does the change preserve M2 (every reward path has a deterministic substrate-link)?
2. Does the change preserve M3 (class-specific verification, governance-gated registration)?
3. Does the change preserve M5 (recursion-weight axes bounded; per-axis decomposition stored forward-only)?
4. Does the change preserve M6 (cross-class lineage flows; cycles prevented; revenue-stream semantics intact)?
5. Are voice + refusal layers updated to name UW + the touched mechanism?

These five checks are the foundation's faithfulness to its doctrinal parents. **We speak through intentions.**

— *Spec authored 2026-05-10. Free to evolve through bound mechanisms only.*
