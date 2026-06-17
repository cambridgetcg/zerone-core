# ToK Cascade Bundling Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bind TC4 of `docs/TOK_SUBSTRATE.md` ("the graph carries its disprovals") by delivering cascade-bundling on top of Plan 1: a `CascadeReplay` selector, persistent `StatusTransition` and `CascadeEvent` records, a V2 snapshot root that commits to the disproval-graph alongside the support-graph, and two invariant tests that prevent disproven facts from ever being curated out of the substrate.

**Architecture:** Plan 1 ships the *support-graph* — descendants/ancestors via `SUPPORTS|REQUIRES|REFINES|GENERALIZES|CITES` edges, deliberately filtering out `CONTRADICTS|SUPERSEDES` so the support cone stays clean. Plan 2 layers the *disproval-graph* on top: a parallel selector (`CascadeReplay`) that walks `CONTRADICTS|SUPERSEDES` edges from a disproven root, materialises the cascade events that fired when the disproof landed, and ships vindication records + supersession chains for every node in the bundle. The two graphs are bundled together in one `ToKBundle` whose snapshot root commits to both — a trainer that asks for `RootedSubtree` gets the support-graph; a trainer that asks for `CascadeReplay` gets the disproval-graph; both are pinned by re-derivable Merkle roots.

The state extension is what makes TC4 binding rather than rhetorical. `Fact.status` today is a current value — disproven facts have `Status=DISPROVEN` but no on-chain timeline. Plan 2 adds a forward-only `StatusTransition` log per fact (one entry per status change) and a `CascadeEvent` log per disproof (one entry per cascaded descendant). Events become records; records become bundle entries; bundles get pinned. The voice the chain already speaks (`falsification_cascade`, `fact_disproven`) becomes the state the chain stores.

**Tech Stack:** Cosmos SDK v0.50, Go 1.23, protobuf v3, cosmossdk.io modules. New code stays in `x/knowledge/` (no new module). Reuses Plan 1's `ToKSelector` union, `ComputeToKSnapshotRoot` Merkle pattern, and `AssembleToKBundle` orchestration shape — extends them rather than duplicating.

**Spec:** `docs/TOK_SUBSTRATE.md` (TC4, lines 59–67).

**Plan series:**
- Plan 1: ToK Foundation (TC1, TC2, TC3, TC5) — *shipped*
- Plan 2: **TC4 Cascade Bundling** — *this doc*
- Plan 3: ToKManifestTrust (training_provenance extension)
- Plan 4: TC6 Lineage Royalties
- Plan 5: Doctrine Closure (meta-test + TRAINING_ON_TOK.md + README/Truth-Paper)

---

## File Structure

**New files:**
- `proto/zerone/knowledge/v1/tok_cascade.proto` — `CascadeReplaySelector`, `CascadeEvent`, `StatusTransition`, extended `ToKBundle` fields
- `x/knowledge/keeper/status_transitions.go` — `RecordStatusTransition`, `GetStatusHistory`, `IterateStatusTransitions`
- `x/knowledge/keeper/cascade_events.go` — `RecordCascadeEvent`, `GetCascadeEventsForDisproof`, `IterateCascadeEvents`
- `x/knowledge/keeper/tok_cascade.go` — `GatherCascade`, V2 snapshot root extension, `AssembleToKBundleV2`
- `x/knowledge/keeper/tok_cascade_test.go` — unit tests for cascade gather + bundle V2 + status-transition store
- `tests/cross_stack/tok_substrate_tc4_test.go` — TC4 invariants (cascade-replay returns disprovals; non-cascade selectors do not prune DISPROVEN)

**Modified files:**
- `proto/zerone/knowledge/v1/tok.proto` — promote reserved tag 4 (`cascade_replay`) to active variant
- `proto/zerone/knowledge/v1/query.proto` — extend `ToKBundle` with cascade fields; bump `ToKCapabilities.tok_doctrine_version`
- `x/knowledge/keeper/tok_selector.go` — add `ValidateAndCapToKSelector` branch for `CascadeReplay`
- `x/knowledge/keeper/tok_bundle.go` — `SelectToKIds` dispatches `CascadeReplay`; root computation upgrades to V2 when cascade fields present
- `x/knowledge/keeper/tok_serialise.go` — emit `kind:"cascade_event"`, `kind:"transition"`, `kind:"vindication"` JSONL lines
- `x/knowledge/keeper/grpc_query.go` — `BundleToK` handler accepts `CascadeReplay`; `RouteBCapabilities` advertises new selector
- `x/knowledge/keeper/state.go` — `SetFact` records a `StatusTransition` when status changes
- `x/knowledge/keeper/vindication.go` — `cascadeFalsification` writes `CascadeEvent` records (parallel to event emission)
- `x/knowledge/keeper/events.go` — add `EventTypeCascadeReplayed`, `EventTypeCascadeCompleted` constants
- `x/knowledge/client/cli/query_tok.go` — `cascade-replay` subcommand
- `x/knowledge/types/keys.go` — `StatusTransitionKeyPrefix=0x7C`, `StatusTransitionSeqKeyPrefix=0x7D`, `CascadeEventKeyPrefix=0x7E`, `CascadeEventByDescendantPrefix=0x7F`
- `x/knowledge/doc.go` — append TC4 to position layer
- `docs/TOK_SUBSTRATE.md` — TC4 section: align doctrine event names with shipped names (`falsification_cascade` is the per-descendant event; add `cascade_completed` and `cascade_replayed` to the named set)
- `docs/EVENTS.md` — document `cascade_replayed`, `cascade_completed` events with `tok_commitment="TC4"` attributes

---

## Pre-Tasks: Read Before Starting

Skim these in this order to absorb the patterns this plan extends:

- `docs/plans/2026-05-09-tok-foundation-plan.md` — Plan 1, shipped. This plan is the layered extension; familiarity with the Selector → Compute → Assemble pipeline is assumed.
- `docs/TOK_SUBSTRATE.md:59–67` — TC4 spec. Each clause names a code expression that this plan must hit.
- `x/knowledge/keeper/tok_bundle.go` — Plan 1's orchestration. Plan 2 wraps `AssembleToKBundle` in a V2 dispatch, not a rewrite.
- `x/knowledge/keeper/tok_selector.go:60–96` — `ComputeToKSnapshotRoot`. Plan 2 extends the domain set (TOK_NODES, TOK_EDGES, TOK_CASCADE, TOK_TRANSITIONS, TOK_VINDICATIONS) under a new root tag (`TOK_ROOT_V2`).
- `x/knowledge/keeper/vindication.go:197–298` — `handleChallengeDisproven` + `cascadeFalsification`. The cascade machinery is built; Plan 2 makes it forward-queryable by promoting events to records.
- `x/knowledge/types/vindication.go` — `VindicationRecord` shape. Plan 2 ships these as bundle entries (read-only — no schema change).
- `x/knowledge/keeper/training_trace_v5.go:405–426` — `collectSupersessionChain`. The 8-deep walk is reused by `GatherCascade`.
- `tests/cross_stack/tok_cascade_test.go` — `TestToK_FalsificationCascade`. Already proves the cascade behaviour. TC4 invariant tests will check that this behaviour is *bundle-visible*, not just chain-visible.
- `x/knowledge/types/keys.go:120–210` — store key prefix register. Plan 2 reserves 0x7C, 0x7D, 0x7E, 0x7F.
- `CLAUDE.md` — *Proto-Go Consistency Rule* applies: add fields to `.proto` first; never edit `*.pb.go`.

---

## Doctrinal alignment note

`docs/TOK_SUBSTRATE.md` TC4 currently names the cascade-event vocabulary as `descendant_status_flipped` and `cascade_completed`. The chain in fact emits `falsification_cascade` (per descendant; shipped in `vindication.go:291`) and has no `cascade_completed`. Plan 2 reconciles these:

- `falsification_cascade` is the canonical *per-descendant* event. It is in production and indexers depend on it. **This name stays.**
- `cascade_completed` is added as a new *aggregate* event, fired once per disproof after all descendants are processed (with attributes: `disproven_fact_id`, `descendant_count`, `vindication_count`, `tok_commitment="TC4"`).
- `cascade_replayed` is added for **bundle extraction** (fired when `BundleToK` returns a `CascadeReplay` bundle, with attributes: `disproven_fact_id`, `node_count`, `cascade_event_count`, `tok_commitment="TC4"`).

Task 19 updates the doctrine doc to name the events the chain actually emits. Doctrine and code both move toward each other — doctrine drops the never-shipped name, code gains the genuinely-aggregate event.

---

## Tasks

### Task 1: Define cascade proto types

**Files:**
- Create: `proto/zerone/knowledge/v1/tok_cascade.proto`

- [ ] **Step 1: Create the proto file**

```proto
syntax = "proto3";
package zerone.knowledge.v1;

option go_package = "github.com/zerone-chain/zerone/x/knowledge/types";

import "zerone/knowledge/v1/types.proto";

// CascadeReplaySelector walks the disproval-graph from a DISPROVEN root.
// Returns the disproven fact + first-hop cascaded descendants + every
// CascadeEvent the chain recorded when the disproof landed. Optionally
// includes vindication records and supersession-chain pointers.
//
// TC4 (the graph carries its disprovals) is the doctrinal binding.
// Plan 1's RootedSubtree/AncestorCone deliberately filter out
// CONTRADICTS|SUPERSEDES edges; CascadeReplay deliberately includes them.
// The two selectors compose: a trainer that wants both the support-graph
// and the disproval-graph for a fact issues two BundleToK calls.
message CascadeReplaySelector {
  string disproven_fact_id    = 1;  // must reference a fact with Status=DISPROVEN
  uint32 max_depth            = 2;  // cascade depth; 0 = chain default 1; cap 3
  bool   include_vindications = 3;  // ship VindicationRecord entries
  bool   include_supersessions = 4; // walk SUPERSEDES chain (8-deep cap)
  bool   include_status_history = 5; // ship StatusTransition timeline per node
}

// CascadeEvent is the persistent record of a single descendant status flip
// during a falsification cascade. The chain emits a `falsification_cascade`
// voice event when this fires; the record is what makes TC4 bundle-able.
//
// One CascadeEvent is written per descendant per disproof. Forward-only
// (commitment 10): records are append-only, never amended in place.
message CascadeEvent {
  uint64 seq                  = 1;  // monotonic per disproven_fact_id
  string disproven_fact_id    = 2;  // root of the cascade
  string descendant_fact_id   = 3;  // direct descendant whose status flipped
  string challenge_claim_id   = 4;  // challenge that triggered the cascade
  string edge_relation        = 5;  // SUPPORTS|REQUIRES|REFINES|GENERALIZES|CITES
  FactStatus prior_status     = 6;  // VERIFIED|ACTIVE|AT_RISK
  FactStatus new_status       = 7;  // CONTESTED (current cascade always)
  uint64 block_height         = 8;  // height the cascade fired
}

// StatusTransition is the persistent record of a single status change on
// a fact. Forward-only: once written, never modified. The full history of
// a fact's status is reconstructable by iterating these records by factID.
//
// TC4: "Fact.status is preserved in graph manifests with full transition
// history." This is the record that makes "full transition history" real.
message StatusTransition {
  uint64 seq                = 1;  // monotonic per fact_id
  string fact_id            = 2;
  FactStatus prior_status   = 3;  // status before this transition
  FactStatus new_status     = 4;  // status after this transition
  uint64 block_height       = 5;
  string cause_event_type   = 6;  // "verification" | "challenge_disproven" | "cascade" | "supersession"
  string cause_id           = 7;  // round_id | challenge_claim_id | disproven_fact_id | superseding_fact_id
}
```

- [ ] **Step 2: Run proto-check**

Run: `make proto-check`
Expected: PASS — file is well-formed proto3.

- [ ] **Step 3: Commit**

```bash
git add proto/zerone/knowledge/v1/tok_cascade.proto
git commit -m "proto(knowledge): define CascadeReplay selector + CascadeEvent + StatusTransition (TC4)"
```

---

### Task 2: Promote reserved CascadeReplay tag in tok.proto and extend ToKBundle

**Files:**
- Modify: `proto/zerone/knowledge/v1/tok.proto`
- Modify: `proto/zerone/knowledge/v1/query.proto`

- [ ] **Step 1: Activate reserved tag 4 in `ToKSelector`**

In `tok.proto`, change:

```proto
oneof variant {
  RootedSubtreeSelector rooted_subtree = 1;
  AncestorConeSelector  ancestor_cone  = 2;
  FrontierSelector      frontier       = 3;
  // Reserved: 4 = cascade_replay (Plan 2), 5 = fork_and_decide (Plan 4).
}
```

To:

```proto
oneof variant {
  RootedSubtreeSelector rooted_subtree = 1;
  AncestorConeSelector  ancestor_cone  = 2;
  FrontierSelector      frontier       = 3;
  CascadeReplaySelector cascade_replay = 4;  // Plan 2 (TC4)
  // Reserved: 5 = fork_and_decide (Plan 4).
}
```

Add the import at the top:

```proto
import "zerone/knowledge/v1/tok_cascade.proto";
```

- [ ] **Step 2: Extend `ToKBundle` with cascade fields**

Append to the existing `ToKBundle` message:

```proto
message ToKBundle {
  // … existing fields 1-8 …

  // ─── TC4: the graph carries its disprovals ──────────────────────────
  // Populated only when selector is CascadeReplay or when other selectors
  // touch DISPROVEN nodes whose history was requested via
  // CascadeReplaySelector.include_status_history. Empty for pure Plan 1
  // bundles.
  repeated CascadeEvent      cascade_events     = 9;
  repeated VindicationRecord vindications       = 10;
  repeated string            supersession_chain = 11;  // factIDs, root → leaf
  repeated StatusTransition  status_history     = 12;  // sorted by (fact_id, seq)

  // V2 root: when cascade fields are non-empty, snapshot_root commits
  // to all of (nodes, edges, cascade_events, vindications, transitions).
  // V1 root semantics (Plan 1) preserved when cascade fields are empty.
  // Provenance.tok_root_version distinguishes "v1" / "v2".
}
```

Note: `VindicationRecord` already exists in `x/knowledge/types/vindication.go` as a Go struct (JSON-serialised). For proto compatibility, add a parallel proto message in `tok_cascade.proto`:

```proto
message VindicationRecord {
  string verifier      = 1;
  string fact_id       = 2;
  string refund_amount = 3;
  string bonus_amount  = 4;
  uint64 vindicated_at = 5;
  string disproven_by  = 6;
  string round_id      = 7;
}
```

The keeper writes/reads JSON for backward compatibility but converts to/from this proto for bundle assembly.

- [ ] **Step 3: Bump `ToKCapabilities.tok_doctrine_version`**

In `query.proto`, the existing field's value will be advertised by the Go-side handler (Task 13). No proto change needed; Go-side string changes.

- [ ] **Step 4: Run codegen**

Run: `make proto-gen`
Expected: regenerated `*.pb.go` files. No errors.

- [ ] **Step 5: Verify build**

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 6: Commit**

```bash
git add proto/zerone/knowledge/v1/tok.proto proto/zerone/knowledge/v1/tok_cascade.proto proto/zerone/knowledge/v1/query.proto x/knowledge/types/*.pb.go
git commit -m "proto(knowledge): activate cascade_replay variant + extend ToKBundle (TC4)"
```

---

### Task 3: Reserve key prefixes for new state

**Files:**
- Modify: `x/knowledge/types/keys.go`

- [ ] **Step 1: Add prefix declarations**

Find the last-declared prefix block (currently `PendingFactInjectionByExecuteKeyPrefix = []byte{0x7B}`). Append:

```go
// ─── ToK Cascade Bundling (Plan 2 / TC4) ─────────────────────────────────
StatusTransitionKeyPrefix      = []byte{0x7C} // 0x7C | factID | be64(seq) → StatusTransition (proto)
StatusTransitionSeqKeyPrefix   = []byte{0x7D} // 0x7D | factID → uvarint next-seq counter
CascadeEventKeyPrefix          = []byte{0x7E} // 0x7E | disprovenFactID | be64(seq) → CascadeEvent (proto)
CascadeEventByDescendantPrefix = []byte{0x7F} // 0x7F | descendantFactID | disprovenFactID → 1 (reverse index)
```

- [ ] **Step 2: Add key constructors**

```go
// StatusTransitionKey returns the store key for a single transition.
func StatusTransitionKey(factID string, seq uint64) []byte {
    key := append([]byte{}, StatusTransitionKeyPrefix...)
    key = append(key, []byte(factID)...)
    key = append(key, 0x00)  // separator
    key = binary.BigEndian.AppendUint64(key, seq)
    return key
}

// StatusTransitionPrefixForFact returns the prefix for iterating all
// transitions for a single fact (sorted by seq).
func StatusTransitionPrefixForFact(factID string) []byte {
    key := append([]byte{}, StatusTransitionKeyPrefix...)
    key = append(key, []byte(factID)...)
    key = append(key, 0x00)
    return key
}

// StatusTransitionSeqKey returns the next-seq counter for a fact.
func StatusTransitionSeqKey(factID string) []byte {
    return append(append([]byte{}, StatusTransitionSeqKeyPrefix...), []byte(factID)...)
}

// CascadeEventKey returns the store key for a single cascade event.
func CascadeEventKey(disprovenFactID string, seq uint64) []byte {
    key := append([]byte{}, CascadeEventKeyPrefix...)
    key = append(key, []byte(disprovenFactID)...)
    key = append(key, 0x00)
    key = binary.BigEndian.AppendUint64(key, seq)
    return key
}

// CascadeEventPrefixForDisproof returns the prefix for iterating all
// cascade events for a single disproof.
func CascadeEventPrefixForDisproof(disprovenFactID string) []byte {
    key := append([]byte{}, CascadeEventKeyPrefix...)
    key = append(key, []byte(disprovenFactID)...)
    key = append(key, 0x00)
    return key
}

// CascadeEventByDescendantKey returns the reverse-index key.
func CascadeEventByDescendantKey(descendantFactID, disprovenFactID string) []byte {
    key := append([]byte{}, CascadeEventByDescendantPrefix...)
    key = append(key, []byte(descendantFactID)...)
    key = append(key, 0x00)
    key = append(key, []byte(disprovenFactID)...)
    return key
}
```

- [ ] **Step 3: Verify build**

Run: `go build ./x/knowledge/types/...`
Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add x/knowledge/types/keys.go
git commit -m "feat(knowledge): reserve key prefixes 0x7C-0x7F for cascade bundling (TC4)"
```

---

### Task 4: Implement StatusTransition store

**Files:**
- Create: `x/knowledge/keeper/status_transitions.go`
- Create: `x/knowledge/keeper/status_transitions_test.go` (or extend `tok_cascade_test.go` later)

- [ ] **Step 1: Failing test**

```go
package keeper_test

import (
    "testing"

    "github.com/stretchr/testify/require"
    "github.com/zerone-chain/zerone/x/knowledge/types"
)

func TestRecordStatusTransition_AppendsForwardOnly(t *testing.T) {
    k, ctx, _, _ := setupKnowledgeTestFull(t)

    require.NoError(t, k.RecordStatusTransition(ctx, &types.StatusTransition{
        FactId:         "fact-a",
        PriorStatus:    types.FactStatus_FACT_STATUS_ACTIVE,
        NewStatus:      types.FactStatus_FACT_STATUS_VERIFIED,
        BlockHeight:    100,
        CauseEventType: "verification",
        CauseId:        "round-1",
    }))

    require.NoError(t, k.RecordStatusTransition(ctx, &types.StatusTransition{
        FactId:         "fact-a",
        PriorStatus:    types.FactStatus_FACT_STATUS_VERIFIED,
        NewStatus:      types.FactStatus_FACT_STATUS_DISPROVEN,
        BlockHeight:    200,
        CauseEventType: "challenge_disproven",
        CauseId:        "challenge-7",
    }))

    history := k.GetStatusHistory(ctx, "fact-a")
    require.Len(t, history, 2)
    require.Equal(t, uint64(1), history[0].Seq, "first transition seq=1")
    require.Equal(t, uint64(2), history[1].Seq, "second transition seq=2")
    require.Equal(t, types.FactStatus_FACT_STATUS_VERIFIED, history[0].NewStatus)
    require.Equal(t, types.FactStatus_FACT_STATUS_DISPROVEN, history[1].NewStatus)
}

func TestGetStatusHistory_EmptyForUntouchedFact(t *testing.T) {
    k, ctx, _, _ := setupKnowledgeTestFull(t)
    history := k.GetStatusHistory(ctx, "never-existed")
    require.Empty(t, history)
}
```

- [ ] **Step 2: Run, verify FAIL**

Run: `go test ./x/knowledge/keeper/ -run TestRecordStatusTransition -v`
Expected: FAIL — `RecordStatusTransition` undefined.

- [ ] **Step 3: Implement**

```go
// Package keeper, file: status_transitions.go
package keeper

import (
    "context"
    "encoding/binary"
    "fmt"

    storetypes "cosmossdk.io/store/types"
    sdk "github.com/cosmos/cosmos-sdk/types"

    "github.com/zerone-chain/zerone/x/knowledge/types"
)

// RecordStatusTransition appends a forward-only StatusTransition entry for
// the given fact. The seq is auto-allocated from a per-fact counter.
//
// TC4: "Fact.status is preserved in graph manifests with full transition
// history." This is the function that makes "full transition history" real.
// Commitment 10: forward-only audit; transitions never modify in place.
func (k Keeper) RecordStatusTransition(ctx context.Context, t *types.StatusTransition) error {
    if t == nil || t.FactId == "" {
        return fmt.Errorf("status transition requires fact_id")
    }
    if t.PriorStatus == t.NewStatus {
        // No-op: same status. Don't write a transition for an unchanged status
        // — this keeps the history meaningful.
        return nil
    }

    sdkCtx := sdk.UnwrapSDKContext(ctx)
    store := sdkCtx.KVStore(k.storeKey)

    // Allocate next seq.
    seqKey := types.StatusTransitionSeqKey(t.FactId)
    var nextSeq uint64
    if buf := store.Get(seqKey); buf != nil {
        nextSeq, _ = binary.Uvarint(buf)
    }
    nextSeq++
    t.Seq = nextSeq

    // Persist seq counter.
    seqBuf := make([]byte, binary.MaxVarintLen64)
    n := binary.PutUvarint(seqBuf, nextSeq)
    store.Set(seqKey, seqBuf[:n])

    // Persist transition.
    bz, err := k.cdc.Marshal(t)
    if err != nil {
        return fmt.Errorf("marshal status transition: %w", err)
    }
    store.Set(types.StatusTransitionKey(t.FactId, nextSeq), bz)
    return nil
}

// GetStatusHistory returns all status transitions for a fact, sorted by seq.
// Empty slice for facts with no recorded transitions (e.g. genesis facts
// that never changed status post-genesis).
func (k Keeper) GetStatusHistory(ctx context.Context, factID string) []*types.StatusTransition {
    sdkCtx := sdk.UnwrapSDKContext(ctx)
    store := sdkCtx.KVStore(k.storeKey)
    prefix := types.StatusTransitionPrefixForFact(factID)
    iter := storetypes.KVStorePrefixIterator(store, prefix)
    defer iter.Close()

    var out []*types.StatusTransition
    for ; iter.Valid(); iter.Next() {
        t := &types.StatusTransition{}
        if err := k.cdc.Unmarshal(iter.Value(), t); err != nil {
            continue
        }
        out = append(out, t)
    }
    return out
}

// IterateStatusTransitions calls f for every transition in the store.
// Used by genesis export and audit tooling.
func (k Keeper) IterateStatusTransitions(ctx context.Context, f func(*types.StatusTransition) bool) {
    sdkCtx := sdk.UnwrapSDKContext(ctx)
    store := sdkCtx.KVStore(k.storeKey)
    iter := storetypes.KVStorePrefixIterator(store, types.StatusTransitionKeyPrefix)
    defer iter.Close()
    for ; iter.Valid(); iter.Next() {
        t := &types.StatusTransition{}
        if err := k.cdc.Unmarshal(iter.Value(), t); err != nil {
            continue
        }
        if f(t) {
            return
        }
    }
}
```

- [ ] **Step 4: Run test to verify PASS**

Run: `go test ./x/knowledge/keeper/ -run TestRecordStatusTransition -v`
Expected: PASS (2 tests).

- [ ] **Step 5: Commit**

```bash
git add x/knowledge/keeper/status_transitions.go x/knowledge/keeper/status_transitions_test.go
git commit -m "feat(knowledge): StatusTransition store — forward-only per-fact history (TC4)"
```

---

### Task 5: Hook SetFact to record status transitions

**Files:**
- Modify: `x/knowledge/keeper/state.go`

The cleanest hook: `SetFact` is the single write path for facts. Detect status change by comparing against the prior persisted value.

- [ ] **Step 1: Failing test**

```go
func TestSetFact_RecordsStatusTransition(t *testing.T) {
    k, ctx, _, _ := setupKnowledgeTestFull(t)

    // Initial set: ACTIVE → VERIFIED.
    require.NoError(t, k.SetFact(ctx, &types.Fact{
        Id: "fact-x", Domain: "physics",
        Status: types.FactStatus_FACT_STATUS_ACTIVE,
    }))
    require.NoError(t, k.SetFact(ctx, &types.Fact{
        Id: "fact-x", Domain: "physics",
        Status: types.FactStatus_FACT_STATUS_VERIFIED,
    }))

    history := k.GetStatusHistory(ctx, "fact-x")
    require.Len(t, history, 1, "exactly one transition: ACTIVE → VERIFIED")
    require.Equal(t, types.FactStatus_FACT_STATUS_ACTIVE, history[0].PriorStatus)
    require.Equal(t, types.FactStatus_FACT_STATUS_VERIFIED, history[0].NewStatus)
}

func TestSetFact_NoTransitionOnUnchangedStatus(t *testing.T) {
    k, ctx, _, _ := setupKnowledgeTestFull(t)

    require.NoError(t, k.SetFact(ctx, &types.Fact{
        Id: "fact-y", Domain: "math",
        Status: types.FactStatus_FACT_STATUS_VERIFIED,
    }))
    // Re-write same status with different unrelated field.
    require.NoError(t, k.SetFact(ctx, &types.Fact{
        Id: "fact-y", Domain: "math",
        Status:     types.FactStatus_FACT_STATUS_VERIFIED,
        Confidence: 750_000,
    }))

    history := k.GetStatusHistory(ctx, "fact-y")
    require.Empty(t, history, "no transitions written when status unchanged")
}

func TestSetFact_FirstWriteRecordsFromUnspecified(t *testing.T) {
    k, ctx, _, _ := setupKnowledgeTestFull(t)

    require.NoError(t, k.SetFact(ctx, &types.Fact{
        Id: "fact-genesis", Domain: "math",
        Status: types.FactStatus_FACT_STATUS_VERIFIED,
    }))
    history := k.GetStatusHistory(ctx, "fact-genesis")
    require.Len(t, history, 1, "first write records initial transition")
    require.Equal(t, types.FactStatus_FACT_STATUS_UNSPECIFIED, history[0].PriorStatus)
    require.Equal(t, types.FactStatus_FACT_STATUS_VERIFIED, history[0].NewStatus)
}
```

- [ ] **Step 2: Run, verify FAIL**

- [ ] **Step 3: Modify SetFact**

In `state.go:53` (current `SetFact` body), insert before the actual write:

```go
func (k Keeper) SetFact(ctx context.Context, fact *types.Fact) error {
    if fact == nil || fact.Id == "" {
        return fmt.Errorf("fact requires id")
    }

    // TC4: record status transition forward-only when status differs from prior.
    var priorStatus types.FactStatus
    if existing, ok := k.GetFact(ctx, fact.Id); ok {
        priorStatus = existing.Status
    }
    if priorStatus != fact.Status {
        sdkCtx := sdk.UnwrapSDKContext(ctx)
        cause, causeID := inferStatusTransitionCause(sdkCtx, fact)
        _ = k.RecordStatusTransition(ctx, &types.StatusTransition{
            FactId:         fact.Id,
            PriorStatus:    priorStatus,
            NewStatus:      fact.Status,
            BlockHeight:    uint64(sdkCtx.BlockHeight()),
            CauseEventType: cause,
            CauseId:        causeID,
        })
    }

    // … existing SetFact write logic unchanged …
}

// inferStatusTransitionCause looks at the SDK context to infer what action
// triggered the status change. Best-effort; defaults to "unknown" if no
// signal can be read.
//
// The caller of SetFact often has rich context (round_id, challenge_id) that
// SetFact does not. As a future refinement, callers can stash the cause in
// the SDK context via a typed key. For now: best-effort string inspection.
func inferStatusTransitionCause(ctx sdk.Context, fact *types.Fact) (string, string) {
    switch fact.Status {
    case types.FactStatus_FACT_STATUS_DISPROVEN:
        return "challenge_disproven", "" // caller (handleChallengeDisproven) has the challenge_claim_id but not in ctx
    case types.FactStatus_FACT_STATUS_CONTESTED:
        return "cascade", ""
    case types.FactStatus_FACT_STATUS_SUPERSEDED:
        return "supersession", ""
    case types.FactStatus_FACT_STATUS_VERIFIED:
        return "verification", fact.ClaimId
    default:
        return "unknown", ""
    }
}
```

- [ ] **Step 4: Refine cause attribution**

The `inferStatusTransitionCause` heuristic is best-effort. For sharper attribution, add an optional `RecordStatusTransitionWithCause` helper that callers (`handleChallengeDisproven`, `cascadeFalsification`) call directly *before* `SetFact`, then `SetFact` skips the auto-record:

```go
// In state.go:
func (k Keeper) SetFactSkipTransition(ctx context.Context, fact *types.Fact) error {
    // Same as SetFact but skips the auto status-transition record. Use
    // when the caller has already written a precise StatusTransition with
    // full cause attribution.
    // …
}
```

This is the cleaner approach. Refactor `handleChallengeDisproven` and `cascadeFalsification` to write the precise transition first, then call `SetFactSkipTransition`. (Done in Task 7.)

- [ ] **Step 5: Run tests to verify PASS**

Run: `go test ./x/knowledge/keeper/ -run TestSetFact_Records -v`
Expected: PASS (3 tests).

- [ ] **Step 6: Run the broader test suite**

Run: `go test ./x/knowledge/... -timeout 300s`
Expected: PASS or known-skipped. Watch for any test that asserts on transition-counts in unrelated areas — the SetFact hook now writes transitions everywhere.

- [ ] **Step 7: Commit**

```bash
git add x/knowledge/keeper/state.go x/knowledge/keeper/status_transitions_test.go
git commit -m "feat(knowledge): SetFact records StatusTransition on status change (TC4)"
```

---

### Task 6: Implement CascadeEvent store

**Files:**
- Create: `x/knowledge/keeper/cascade_events.go`

- [ ] **Step 1: Failing test**

```go
func TestRecordCascadeEvent_AppendsForwardOnly(t *testing.T) {
    k, ctx, _, _ := setupKnowledgeTestFull(t)

    require.NoError(t, k.RecordCascadeEvent(ctx, &types.CascadeEvent{
        DisprovenFactId:   "axiom-d",
        DescendantFactId:  "child-1",
        ChallengeClaimId:  "challenge-7",
        EdgeRelation:      "SUPPORTS",
        PriorStatus:       types.FactStatus_FACT_STATUS_VERIFIED,
        NewStatus:         types.FactStatus_FACT_STATUS_CONTESTED,
        BlockHeight:       200,
    }))
    require.NoError(t, k.RecordCascadeEvent(ctx, &types.CascadeEvent{
        DisprovenFactId:   "axiom-d",
        DescendantFactId:  "child-2",
        ChallengeClaimId:  "challenge-7",
        EdgeRelation:      "REQUIRES",
        PriorStatus:       types.FactStatus_FACT_STATUS_VERIFIED,
        NewStatus:         types.FactStatus_FACT_STATUS_CONTESTED,
        BlockHeight:       200,
    }))

    events := k.GetCascadeEventsForDisproof(ctx, "axiom-d")
    require.Len(t, events, 2)
    require.Equal(t, uint64(1), events[0].Seq)
    require.Equal(t, uint64(2), events[1].Seq)
    require.Equal(t, "child-1", events[0].DescendantFactId)
    require.Equal(t, "child-2", events[1].DescendantFactId)
}

func TestGetCascadeEventsForDisproof_EmptyForUntouchedFact(t *testing.T) {
    k, ctx, _, _ := setupKnowledgeTestFull(t)
    events := k.GetCascadeEventsForDisproof(ctx, "never-disproven")
    require.Empty(t, events)
}

func TestCascadeEvent_ReverseIndexFindable(t *testing.T) {
    k, ctx, _, _ := setupKnowledgeTestFull(t)

    require.NoError(t, k.RecordCascadeEvent(ctx, &types.CascadeEvent{
        DisprovenFactId:  "axiom-a",
        DescendantFactId: "leaf-x",
        EdgeRelation:     "SUPPORTS",
    }))
    require.NoError(t, k.RecordCascadeEvent(ctx, &types.CascadeEvent{
        DisprovenFactId:  "axiom-b",
        DescendantFactId: "leaf-x",
        EdgeRelation:     "CITES",
    }))

    // leaf-x was hit by two disproofs; reverse index lets us find both.
    disproofs := k.GetDisproofsAffectingDescendant(ctx, "leaf-x")
    require.Len(t, disproofs, 2)
    require.Contains(t, disproofs, "axiom-a")
    require.Contains(t, disproofs, "axiom-b")
}
```

- [ ] **Step 2: Run, verify FAIL**

- [ ] **Step 3: Implement**

```go
// Package keeper, file: cascade_events.go
package keeper

import (
    "context"
    "encoding/binary"
    "fmt"

    storetypes "cosmossdk.io/store/types"
    sdk "github.com/cosmos/cosmos-sdk/types"

    "github.com/zerone-chain/zerone/x/knowledge/types"
)

// RecordCascadeEvent appends a CascadeEvent to the per-disproof log and
// updates the reverse index.
//
// TC4: cascade events are bundled with the substrate. The chain emits a
// `falsification_cascade` voice event when this fires; the record is what
// makes TC4 bundle-able. Commitment 10: forward-only.
func (k Keeper) RecordCascadeEvent(ctx context.Context, ev *types.CascadeEvent) error {
    if ev == nil || ev.DisprovenFactId == "" || ev.DescendantFactId == "" {
        return fmt.Errorf("cascade event requires disproven_fact_id and descendant_fact_id")
    }

    sdkCtx := sdk.UnwrapSDKContext(ctx)
    store := sdkCtx.KVStore(k.storeKey)

    // Allocate seq from a counter on the disproof root. We piggyback on the
    // CascadeEvent prefix iteration: count existing entries + 1. (Cheaper
    // than a separate counter for a write path that fires at most ~depth
    // times per disproof.)
    iter := storetypes.KVStorePrefixIterator(store, types.CascadeEventPrefixForDisproof(ev.DisprovenFactId))
    var lastSeq uint64
    for ; iter.Valid(); iter.Next() {
        existing := &types.CascadeEvent{}
        if err := k.cdc.Unmarshal(iter.Value(), existing); err == nil {
            if existing.Seq > lastSeq {
                lastSeq = existing.Seq
            }
        }
    }
    iter.Close()
    ev.Seq = lastSeq + 1

    // Persist cascade event.
    bz, err := k.cdc.Marshal(ev)
    if err != nil {
        return fmt.Errorf("marshal cascade event: %w", err)
    }
    store.Set(types.CascadeEventKey(ev.DisprovenFactId, ev.Seq), bz)

    // Persist reverse-index marker.
    store.Set(types.CascadeEventByDescendantKey(ev.DescendantFactId, ev.DisprovenFactId), []byte{0x01})
    return nil
}

// GetCascadeEventsForDisproof returns all cascade events for a single
// disproof, sorted by seq.
func (k Keeper) GetCascadeEventsForDisproof(ctx context.Context, disprovenFactID string) []*types.CascadeEvent {
    sdkCtx := sdk.UnwrapSDKContext(ctx)
    store := sdkCtx.KVStore(k.storeKey)
    iter := storetypes.KVStorePrefixIterator(store, types.CascadeEventPrefixForDisproof(disprovenFactID))
    defer iter.Close()

    var out []*types.CascadeEvent
    for ; iter.Valid(); iter.Next() {
        ev := &types.CascadeEvent{}
        if err := k.cdc.Unmarshal(iter.Value(), ev); err == nil {
            out = append(out, ev)
        }
    }
    return out
}

// GetDisproofsAffectingDescendant returns the disproven_fact_ids of every
// disproof that cascaded onto the given descendant. Used by training-data
// auditors to surface "this fact was hit by N disproofs."
func (k Keeper) GetDisproofsAffectingDescendant(ctx context.Context, descendantFactID string) []string {
    sdkCtx := sdk.UnwrapSDKContext(ctx)
    store := sdkCtx.KVStore(k.storeKey)
    prefix := append(append([]byte{}, types.CascadeEventByDescendantPrefix...), []byte(descendantFactID)...)
    prefix = append(prefix, 0x00)
    iter := storetypes.KVStorePrefixIterator(store, prefix)
    defer iter.Close()

    var out []string
    for ; iter.Valid(); iter.Next() {
        // Key suffix after the prefix is the disproven_fact_id bytes.
        key := iter.Key()
        out = append(out, string(key[len(prefix):]))
    }
    return out
}

// IterateCascadeEvents iterates every cascade event. Used by genesis export.
func (k Keeper) IterateCascadeEvents(ctx context.Context, f func(*types.CascadeEvent) bool) {
    sdkCtx := sdk.UnwrapSDKContext(ctx)
    store := sdkCtx.KVStore(k.storeKey)
    iter := storetypes.KVStorePrefixIterator(store, types.CascadeEventKeyPrefix)
    defer iter.Close()
    for ; iter.Valid(); iter.Next() {
        ev := &types.CascadeEvent{}
        if err := k.cdc.Unmarshal(iter.Value(), ev); err != nil {
            continue
        }
        if f(ev) {
            return
        }
    }
}
```

- [ ] **Step 4: Run tests to verify PASS**

Run: `go test ./x/knowledge/keeper/ -run TestRecordCascadeEvent -v`
Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add x/knowledge/keeper/cascade_events.go
git commit -m "feat(knowledge): CascadeEvent store + reverse index by descendant (TC4)"
```

---

### Task 7: Hook cascadeFalsification to write CascadeEvent records

**Files:**
- Modify: `x/knowledge/keeper/vindication.go`

- [ ] **Step 1: Failing test (cross-cutting)**

```go
func TestCascadeFalsification_WritesCascadeEventRecords(t *testing.T) {
    h := NewTestHarness(t)
    domain := "test_cascade_record_domain"
    require.NoError(t, h.KnowledgeKeeper.SetDomain(h.Ctx, &knowledgetypes.Domain{
        Name: domain, Status: knowledgetypes.DomainStatus_DOMAIN_STATUS_ACTIVE,
    }))

    axiom := &knowledgetypes.Fact{
        Id: "test-axiom", Domain: domain,
        Status: knowledgetypes.FactStatus_FACT_STATUS_VERIFIED,
    }
    require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, axiom))

    factB := submitAndAcceptChainedClaim(t, h, domain, "depends on axiom",
        []*knowledgetypes.ClaimRelation{{
            TargetFactId: axiom.Id,
            Relation:     knowledgetypes.RelationType_RELATION_TYPE_REQUIRES,
            Inference:    knowledgetypes.InferenceType_INFERENCE_TYPE_DEDUCTIVE,
            InferenceStrengthBps: 1_000_000,
        }}, "factB-rec")

    // Disprove axiom (driven by harness path).
    challengeClaim := &knowledgetypes.Claim{
        Id: "challenge-rec", Submitter: "challenger", Domain: domain,
        FactContent:       "axiom is wrong",
        Category:          "empirical",
        Status:            knowledgetypes.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION,
        Stake:             "11000000",
        ProvisionalFactId: axiom.Id,
        Relations: []*knowledgetypes.ClaimRelation{{
            TargetFactId: axiom.Id,
            Relation:     knowledgetypes.RelationType_RELATION_TYPE_CONTRADICTS,
        }},
    }
    require.NoError(t, h.KnowledgeKeeper.SetClaim(h.Ctx, challengeClaim))
    round := &knowledgetypes.VerificationRound{
        Id: "round-rec", ClaimId: challengeClaim.Id,
        Phase: knowledgetypes.VerificationPhase_VERIFICATION_PHASE_COMPLETE,
    }
    require.NoError(t, h.KnowledgeKeeper.CompleteRound(h.Ctx, round, &keeper.VerificationResult{
        Verdict: knowledgetypes.Verdict_VERDICT_ACCEPT, Confidence: 900_000, AcceptCount: 3,
    }))

    // CascadeEvent record must exist for factB.
    events := h.KnowledgeKeeper.GetCascadeEventsForDisproof(h.Ctx, axiom.Id)
    require.Len(t, events, 1)
    require.Equal(t, factB.Id, events[0].DescendantFactId)
    require.Equal(t, "RELATION_TYPE_REQUIRES", events[0].EdgeRelation)
    require.Equal(t, knowledgetypes.FactStatus_FACT_STATUS_VERIFIED, events[0].PriorStatus)
    require.Equal(t, knowledgetypes.FactStatus_FACT_STATUS_CONTESTED, events[0].NewStatus)
}
```

- [ ] **Step 2: Run, verify FAIL** (events not yet written by cascade)

- [ ] **Step 3: Modify `cascadeFalsification`**

In `vindication.go:252`, inside the descendant-walk loop, add the record write parallel to the event emission. Replace:

```go
descendant.Status = types.FactStatus_FACT_STATUS_CONTESTED
_ = k.SetFact(ctx, descendant)
affectedCount++
sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
    "zerone.knowledge.falsification_cascade",
    …
))
```

With:

```go
priorStatus := descendant.Status
descendant.Status = types.FactStatus_FACT_STATUS_CONTESTED

// Write the precise StatusTransition with full cause attribution
// before SetFact (which would otherwise auto-record with imprecise cause).
_ = k.RecordStatusTransition(ctx, &types.StatusTransition{
    FactId:         descendant.Id,
    PriorStatus:    priorStatus,
    NewStatus:      types.FactStatus_FACT_STATUS_CONTESTED,
    BlockHeight:    uint64(sdkCtx.BlockHeight()),
    CauseEventType: "cascade",
    CauseId:        disprovenFactId,
})
_ = k.SetFactSkipTransition(ctx, descendant)

// Persist the cascade event for TC4 bundling.
_ = k.RecordCascadeEvent(ctx, &types.CascadeEvent{
    DisprovenFactId:  disprovenFactId,
    DescendantFactId: descendant.Id,
    ChallengeClaimId: challengeClaimId,
    EdgeRelation:     rel.Relation.String(),
    PriorStatus:      priorStatus,
    NewStatus:        types.FactStatus_FACT_STATUS_CONTESTED,
    BlockHeight:      uint64(sdkCtx.BlockHeight()),
})

affectedCount++
sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
    "zerone.knowledge.falsification_cascade",
    sdk.NewAttribute("descendant_fact_id", descendant.Id),
    sdk.NewAttribute("disproven_fact_id", disprovenFactId),
    sdk.NewAttribute("challenge_claim_id", challengeClaimId),
    sdk.NewAttribute("edge_relation", rel.Relation.String()),
    sdk.NewAttribute("creed_commitment", "3"),
    sdk.NewAttribute("tok_commitment", "TC4"),
))
```

- [ ] **Step 4: Add `cascade_completed` aggregate event**

After the descendant loop (where `if affectedCount > 0` block already exists), emit the aggregate:

```go
sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
    EventTypeCascadeCompleted,
    sdk.NewAttribute("disproven_fact_id", disprovenFactId),
    sdk.NewAttribute("challenge_claim_id", challengeClaimId),
    sdk.NewAttribute("descendant_count", fmt.Sprintf("%d", affectedCount)),
    sdk.NewAttribute("tok_commitment", "TC4"),
    sdk.NewAttribute("creed_commitment", "3"),
))
```

- [ ] **Step 5: Run tests to verify PASS**

Run: `go test ./x/knowledge/... -run TestCascadeFalsification -v`
Expected: PASS.

Also run the existing cross-stack cascade test to verify no regression:
Run: `go test ./tests/cross_stack/ -run TestToK_FalsificationCascade -v`
Expected: still PASS.

- [ ] **Step 6: Commit**

```bash
git add x/knowledge/keeper/vindication.go x/knowledge/keeper/events.go
git commit -m "feat(knowledge): cascadeFalsification persists CascadeEvent + StatusTransition + cascade_completed (TC4)"
```

---

### Task 8: CascadeReplay selector validation

**Files:**
- Modify: `x/knowledge/keeper/tok_selector.go`

- [ ] **Step 1: Failing test**

```go
func TestValidateToKSelector_CascadeReplay_RequiresDisprovenFactId(t *testing.T) {
    sel := &types.ToKSelector{Variant: &types.ToKSelector_CascadeReplay{
        CascadeReplay: &types.CascadeReplaySelector{},
    }}
    err := keeper.ValidateToKSelector(sel)
    require.Error(t, err)
    require.Contains(t, err.Error(), "disproven_fact_id")
}

func TestValidateToKSelector_CascadeReplay_CapsMaxDepth(t *testing.T) {
    sel := &types.ToKSelector{Variant: &types.ToKSelector_CascadeReplay{
        CascadeReplay: &types.CascadeReplaySelector{
            DisprovenFactId: "fact-1", MaxDepth: 100,
        },
    }}
    capped, err := keeper.ValidateAndCapToKSelector(sel)
    require.NoError(t, err)
    require.Equal(t, uint32(3), capped.GetCascadeReplay().MaxDepth, "cascade depth caps at 3")
}

func TestValidateToKSelector_CascadeReplay_ZeroDepthDefaults(t *testing.T) {
    sel := &types.ToKSelector{Variant: &types.ToKSelector_CascadeReplay{
        CascadeReplay: &types.CascadeReplaySelector{
            DisprovenFactId: "fact-1", MaxDepth: 0,
        },
    }}
    capped, err := keeper.ValidateAndCapToKSelector(sel)
    require.NoError(t, err)
    require.Equal(t, uint32(1), capped.GetCascadeReplay().MaxDepth, "zero-depth defaults to first-hop only")
}
```

- [ ] **Step 2: Run, verify FAIL**

- [ ] **Step 3: Add the cascade depth cap constant + extend validation**

In `tok_selector.go`, near the existing caps:

```go
const (
    ToKMaxDepthCap        uint32 = 32
    ToKMaxPathsCap        uint32 = 256
    ToKFrontierLimit      uint32 = 1024
    ToKFrontierCap        uint32 = 8192

    // ToKCascadeDepthDefault: first-hop only, matching the chain's
    // existing cascadeFalsification scope.
    ToKCascadeDepthDefault uint32 = 1
    // ToKCascadeDepthCap: hard ceiling on cascade-replay depth. Higher
    // values would re-introduce runaway invalidation that the chain
    // explicitly avoids in its falsification cascade. The bundle layer
    // can request transitive walks for training purposes, but capped.
    ToKCascadeDepthCap     uint32 = 3
)
```

Extend `ValidateToKSelector`:

```go
case *types.ToKSelector_CascadeReplay:
    if v.CascadeReplay == nil || v.CascadeReplay.DisprovenFactId == "" {
        return fmt.Errorf("cascade_replay.disproven_fact_id required (TC4: cascade-replay walks the disproval-graph from a DISPROVEN root)")
    }
```

Extend `ValidateAndCapToKSelector`:

```go
case *types.ToKSelector_CascadeReplay:
    cp := protov2.Clone(v.CascadeReplay).(*types.CascadeReplaySelector)
    if cp.MaxDepth == 0 {
        cp.MaxDepth = ToKCascadeDepthDefault
    } else if cp.MaxDepth > ToKCascadeDepthCap {
        cp.MaxDepth = ToKCascadeDepthCap
    }
    out.Variant = &types.ToKSelector_CascadeReplay{CascadeReplay: cp}
```

- [ ] **Step 4: Run tests to verify PASS**

Run: `go test ./x/knowledge/keeper/ -run TestValidateToKSelector_CascadeReplay -v`
Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add x/knowledge/keeper/tok_selector.go x/knowledge/keeper/tok_bundle_test.go
git commit -m "feat(knowledge): CascadeReplay selector validation + depth cap (TC4)"
```

---

### Task 9: Implement GatherCascade

**Files:**
- Create: `x/knowledge/keeper/tok_cascade.go`

`GatherCascade` reads the existing `CascadeEvent` store (Task 6) and the descendant relation graph. Unlike `GatherRootedSubtree`, it includes `CONTRADICTS` and the supersession-walk via `SUPERSEDES`.

- [ ] **Step 1: Failing test**

```go
func TestGatherCascade_ReturnsDisproofPlusCascadedDescendants(t *testing.T) {
    k, ctx, _, _ := setupKnowledgeTestFull(t)

    // Pre-seed: disproven axiom + 2 cascaded descendants.
    require.NoError(t, k.SetFact(ctx, &types.Fact{
        Id: "disproven-x", Domain: "physics",
        Status: types.FactStatus_FACT_STATUS_DISPROVEN, VerifiedAtBlock: 100,
    }))
    require.NoError(t, k.SetFact(ctx, &types.Fact{
        Id: "child-1", Domain: "physics",
        Status: types.FactStatus_FACT_STATUS_CONTESTED, VerifiedAtBlock: 100,
    }))
    require.NoError(t, k.SetFact(ctx, &types.Fact{
        Id: "child-2", Domain: "physics",
        Status: types.FactStatus_FACT_STATUS_CONTESTED, VerifiedAtBlock: 100,
    }))

    // Pre-seed: cascade events.
    require.NoError(t, k.RecordCascadeEvent(ctx, &types.CascadeEvent{
        DisprovenFactId: "disproven-x", DescendantFactId: "child-1",
        EdgeRelation: "RELATION_TYPE_SUPPORTS", BlockHeight: 200,
    }))
    require.NoError(t, k.RecordCascadeEvent(ctx, &types.CascadeEvent{
        DisprovenFactId: "disproven-x", DescendantFactId: "child-2",
        EdgeRelation: "RELATION_TYPE_REQUIRES", BlockHeight: 200,
    }))

    sel := &types.CascadeReplaySelector{
        DisprovenFactId: "disproven-x", MaxDepth: 1,
    }
    nodeIDs, edges, cascadeEvents, _, _, err := k.GatherCascade(ctx, sel)
    require.NoError(t, err)
    require.Equal(t, []string{"child-1", "child-2", "disproven-x"}, nodeIDs)
    require.Len(t, cascadeEvents, 2)
    // Edges include the CONTRADICTS that flipped the axiom (if recorded
    // as a relation) and the SUPPORTS/REQUIRES edges that cascaded.
    require.NotEmpty(t, edges)
}

func TestGatherCascade_RejectsNonDisprovenRoot(t *testing.T) {
    k, ctx, _, _ := setupKnowledgeTestFull(t)

    require.NoError(t, k.SetFact(ctx, &types.Fact{
        Id: "still-verified", Domain: "physics",
        Status: types.FactStatus_FACT_STATUS_VERIFIED,
    }))

    sel := &types.CascadeReplaySelector{
        DisprovenFactId: "still-verified", MaxDepth: 1,
    }
    _, _, _, _, _, err := k.GatherCascade(ctx, sel)
    require.Error(t, err, "TC4: cascade replay must reject non-DISPROVEN roots")
    require.Contains(t, err.Error(), "DISPROVEN")
}
```

- [ ] **Step 2: Run, verify FAIL**

- [ ] **Step 3: Implement**

```go
// Package keeper, file: tok_cascade.go
package keeper

import (
    "context"
    "fmt"
    "sort"

    "github.com/zerone-chain/zerone/x/knowledge/types"
)

// ErrToKCascadeNotDisproven is the typed error for "selector root is not DISPROVEN".
var ErrToKCascadeNotDisproven = fmt.Errorf("cascade-replay root is not DISPROVEN")

// GatherCascade walks the disproval-graph from a DISPROVEN root. Returns:
//   - nodeIDs: disproven root + cascaded descendants (sorted)
//   - edges:   CONTRADICTS + first-hop support edges to descendants (sorted)
//   - cascadeEvents: every recorded CascadeEvent for this disproof
//   - vindications:  every VindicationRecord for this disproof's facts
//                    (only populated if sel.IncludeVindications)
//   - supersessionChain: SUPERSEDES walk (only if sel.IncludeSupersessions)
//
// TC4: the graph carries its disprovals. The disproval-graph is the parallel
// of Plan 1's support-graph. Plan 1's selectors filter CONTRADICTS|SUPERSEDES;
// this selector includes them.
func (k Keeper) GatherCascade(
    ctx context.Context,
    sel *types.CascadeReplaySelector,
) (
    nodeIDs []string,
    edges []*types.ToKEdge,
    cascadeEvents []*types.CascadeEvent,
    vindications []*types.VindicationRecord,
    supersessionChain []string,
    err error,
) {
    root, found := k.GetFact(ctx, sel.DisprovenFactId)
    if !found {
        return nil, nil, nil, nil, nil, fmt.Errorf("%w: %s", ErrToKRootFactNotFound, sel.DisprovenFactId)
    }
    if root.Status != types.FactStatus_FACT_STATUS_DISPROVEN {
        return nil, nil, nil, nil, nil, fmt.Errorf("%w (TC4): root %s has status %s, not DISPROVEN",
            ErrToKCascadeNotDisproven, sel.DisprovenFactId, root.Status)
    }

    // Collect cascade events for this disproof.
    cascadeEvents = k.GetCascadeEventsForDisproof(ctx, sel.DisprovenFactId)

    // Build node set: root + every descendant in cascade events.
    visited := map[string]bool{root.Id: true}
    for _, ev := range cascadeEvents {
        visited[ev.DescendantFactId] = true
    }

    // For depth > 1, walk transitively from each cascaded descendant.
    if sel.MaxDepth > 1 {
        for _, ev := range cascadeEvents {
            err := k.gatherCascadeRecursive(ctx, ev.DescendantFactId, 1, sel.MaxDepth, visited)
            if err != nil {
                return nil, nil, nil, nil, nil, err
            }
        }
    }

    // Materialise node ID list.
    for id := range visited {
        nodeIDs = append(nodeIDs, id)
    }
    sort.Strings(nodeIDs)

    // Build edge set:
    //   (a) CONTRADICTS edges into the disproven root
    //   (b) support-bearing edges from descendants → root (these are the
    //       "this fact requires the now-disproven root" relations that
    //       triggered the cascade in the first place)
    edgeSet := map[string]*types.ToKEdge{}
    rootIncoming, _ := k.GetIncomingRelations(ctx, root.Id)
    for _, rel := range rootIncoming {
        // Include CONTRADICTS to the root (the challenge edges) and any
        // support edges from descendants we already counted.
        if rel.Relation == types.RelationType_RELATION_TYPE_CONTRADICTS ||
            visited[rel.SourceFactId] {
            edgeKey := rel.SourceFactId + "->" + rel.TargetFactId + "|" + rel.Relation.String()
            edgeSet[edgeKey] = &types.ToKEdge{
                FromFactId: rel.SourceFactId,
                ToFactId:   rel.TargetFactId,
                Relation:   rel.Relation.String(),
                Inference:  rel.Inference.String(),
            }
        }
    }

    // For depth > 1 traversal, also include edges among visited descendants.
    if sel.MaxDepth > 1 {
        for nodeID := range visited {
            if nodeID == root.Id {
                continue
            }
            outgoing, _ := k.GetFactRelations(ctx, nodeID)
            for _, rel := range outgoing {
                if !visited[rel.TargetFactId] {
                    continue
                }
                edgeKey := rel.SourceFactId + "->" + rel.TargetFactId + "|" + rel.Relation.String()
                if _, ok := edgeSet[edgeKey]; !ok {
                    edgeSet[edgeKey] = &types.ToKEdge{
                        FromFactId: rel.SourceFactId,
                        ToFactId:   rel.TargetFactId,
                        Relation:   rel.Relation.String(),
                        Inference:  rel.Inference.String(),
                    }
                }
            }
        }
    }

    for _, e := range edgeSet {
        edges = append(edges, e)
    }
    sortToKEdges(edges)

    // Optional: vindication records.
    if sel.IncludeVindications {
        // Pull vindication records for the disproven fact (its slashed
        // minority voters whose unpopular vote turned out right).
        records := k.GetVindicationRecordsForFact(ctx, root.Id)
        for i := range records {
            r := records[i] // local copy
            vindications = append(vindications, &types.VindicationRecord{
                Verifier:     r.Verifier,
                FactId:       r.FactId,
                RefundAmount: r.RefundAmount,
                BonusAmount:  r.BonusAmount,
                VindicatedAt: r.VindicatedAt,
                DisprovenBy:  r.DisprovenBy,
                RoundId:      r.RoundId,
            })
        }
    }

    // Optional: supersession chain.
    if sel.IncludeSupersessions {
        supersessionChain = k.collectSupersessionChain(ctx, root.Id)
    }

    return nodeIDs, edges, cascadeEvents, vindications, supersessionChain, nil
}

func (k Keeper) gatherCascadeRecursive(
    ctx context.Context,
    factID string,
    depth, maxDepth uint32,
    visited map[string]bool,
) error {
    if depth >= maxDepth {
        return nil
    }
    incoming, err := k.GetIncomingRelations(ctx, factID)
    if err != nil {
        return err
    }
    for _, rel := range incoming {
        // Walk descendants via support-bearing relations only — same filter
        // as Plan 1's support-graph walk. The disproval cascade *itself*
        // already happened (or didn't) at level 1; deeper levels are the
        // trainer asking "show me the indirect blast radius."
        switch rel.Relation {
        case types.RelationType_RELATION_TYPE_SUPPORTS,
            types.RelationType_RELATION_TYPE_REQUIRES,
            types.RelationType_RELATION_TYPE_REFINES,
            types.RelationType_RELATION_TYPE_GENERALIZES,
            types.RelationType_RELATION_TYPE_CITES:
        default:
            continue
        }
        if !visited[rel.SourceFactId] {
            visited[rel.SourceFactId] = true
            if err := k.gatherCascadeRecursive(ctx, rel.SourceFactId, depth+1, maxDepth, visited); err != nil {
                return err
            }
        }
    }
    return nil
}
```

- [ ] **Step 4: Run tests to verify PASS**

Run: `go test ./x/knowledge/keeper/ -run TestGatherCascade -v`
Expected: PASS (2 tests).

- [ ] **Step 5: Commit**

```bash
git add x/knowledge/keeper/tok_cascade.go x/knowledge/keeper/tok_cascade_test.go
git commit -m "feat(knowledge): GatherCascade — disproval-graph walk for ToK (TC4)"
```

---

### Task 10: Extend ComputeToKSnapshotRoot to V2

**Files:**
- Modify: `x/knowledge/keeper/tok_bundle.go`

The V2 root commits to (nodes, edges, cascade_events, vindications, status_transitions). When cascade fields are empty, V2 produces the SAME 32-byte output as V1 — empty-domain hashes are well-defined and deterministic. **No backward-compat break:** existing Plan 1 bundles re-derive identically under V2 if their cascade fields stay empty.

Wait — that's wrong. V1's root structure is `sha256("TOK_ROOT" || nodesH || edgesH)`; V2's is `sha256("TOK_ROOT_V2" || nodesH || edgesH || cascadeH || vindH || transH)`. Even with empty extras, the domain tag differs and the byte concatenation differs — the roots differ.

The correct approach: **V1 stays as-is; V2 is a parallel function**. The bundle's `Provenance.tok_root_version` tells re-derivers which to use.

- [ ] **Step 1: Failing test**

```go
func TestComputeToKSnapshotRootV2_DistinctFromV1(t *testing.T) {
    nodeIDs := []string{"a", "b"}
    edges := []*types.ToKEdge{{FromFactId: "b", ToFactId: "a", Relation: "SUPPORTS"}}

    v1 := keeper.ComputeToKSnapshotRoot(nodeIDs, edges)
    v2 := keeper.ComputeToKSnapshotRootV2(nodeIDs, edges, nil, nil, nil)

    require.NotEqual(t, v1, v2, "V1 and V2 roots must differ even with empty cascade fields")
    require.Len(t, v2, 32)
}

func TestComputeToKSnapshotRootV2_DomainSeparated(t *testing.T) {
    cascadeEvent := []*types.CascadeEvent{{
        DisprovenFactId: "x", DescendantFactId: "y", EdgeRelation: "SUPPORTS",
    }}

    rWithCascade := keeper.ComputeToKSnapshotRootV2([]string{"x", "y"}, nil, cascadeEvent, nil, nil)
    rWithoutCascade := keeper.ComputeToKSnapshotRootV2([]string{"x", "y"}, nil, nil, nil, nil)

    require.NotEqual(t, rWithCascade, rWithoutCascade, "cascade events must affect root")
}

func TestComputeToKSnapshotRootV2_Deterministic(t *testing.T) {
    nodeIDs := []string{"a", "b"}
    cascade := []*types.CascadeEvent{{
        DisprovenFactId: "a", DescendantFactId: "b", EdgeRelation: "SUPPORTS", BlockHeight: 100,
    }}
    r1 := keeper.ComputeToKSnapshotRootV2(nodeIDs, nil, cascade, nil, nil)
    r2 := keeper.ComputeToKSnapshotRootV2(nodeIDs, nil, cascade, nil, nil)
    require.Equal(t, r1, r2)
}
```

- [ ] **Step 2: Run, verify FAIL**

- [ ] **Step 3: Implement V2 root**

Append to `tok_bundle.go`:

```go
const (
    tokDomainCascade       = "TOK_CASCADE"
    tokDomainVindications  = "TOK_VINDICATIONS"
    tokDomainTransitions   = "TOK_TRANSITIONS"
    tokDomainRootV2        = "TOK_ROOT_V2"
)

// ComputeToKSnapshotRootV2 returns a 32-byte Merkle commitment over the full
// (nodes, edges, cascade_events, vindications, transitions) bundle. Each
// component is independently domain-tagged; the V2 root tag distinguishes
// V2 bundles from V1 bundles even when cascade fields are empty.
//
// TC4: the disproval-graph is bundled with the support-graph under one root.
// A trainer who has the IDs + cascade event canon can re-derive the root
// without trusting the RPC.
func ComputeToKSnapshotRootV2(
    nodeIDs []string,
    edges []*types.ToKEdge,
    cascadeEvents []*types.CascadeEvent,
    vindications []*types.VindicationRecord,
    transitions []*types.StatusTransition,
) []byte {
    sortedNodes := append([]string{}, nodeIDs...)
    sort.Strings(sortedNodes)
    sortedEdges := append([]*types.ToKEdge{}, edges...)
    sortToKEdges(sortedEdges)
    sortedCascade := sortCascadeEvents(cascadeEvents)
    sortedVind := sortVindications(vindications)
    sortedTrans := sortStatusTransitions(transitions)

    nodesH := tokDomainHash(tokDomainNodes, func(h interface{ Write([]byte) (int, error) }) {
        for _, id := range sortedNodes {
            writeLenString(h, id)
        }
    })
    edgesH := tokDomainHash(tokDomainEdges, func(h interface{ Write([]byte) (int, error) }) {
        for _, e := range sortedEdges {
            writeLenString(h, e.FromFactId)
            writeLenString(h, e.ToFactId)
            writeLenString(h, e.Relation)
            writeLenString(h, e.Inference)
        }
    })
    cascadeH := tokDomainHash(tokDomainCascade, func(h interface{ Write([]byte) (int, error) }) {
        for _, ev := range sortedCascade {
            writeLenString(h, ev.DisprovenFactId)
            writeLenString(h, ev.DescendantFactId)
            writeLenString(h, ev.ChallengeClaimId)
            writeLenString(h, ev.EdgeRelation)
            writeLenUint64(h, ev.Seq)
            writeLenUint64(h, ev.BlockHeight)
            writeLenUint64(h, uint64(ev.PriorStatus))
            writeLenUint64(h, uint64(ev.NewStatus))
        }
    })
    vindH := tokDomainHash(tokDomainVindications, func(h interface{ Write([]byte) (int, error) }) {
        for _, v := range sortedVind {
            writeLenString(h, v.Verifier)
            writeLenString(h, v.FactId)
            writeLenString(h, v.RefundAmount)
            writeLenString(h, v.BonusAmount)
            writeLenUint64(h, v.VindicatedAt)
            writeLenString(h, v.DisprovenBy)
            writeLenString(h, v.RoundId)
        }
    })
    transH := tokDomainHash(tokDomainTransitions, func(h interface{ Write([]byte) (int, error) }) {
        for _, t := range sortedTrans {
            writeLenString(h, t.FactId)
            writeLenUint64(h, t.Seq)
            writeLenUint64(h, uint64(t.PriorStatus))
            writeLenUint64(h, uint64(t.NewStatus))
            writeLenUint64(h, t.BlockHeight)
            writeLenString(h, t.CauseEventType)
            writeLenString(h, t.CauseId)
        }
    })

    final := sha256.New()
    writeLenString(final, tokDomainRootV2)
    _, _ = final.Write(nodesH)
    _, _ = final.Write(edgesH)
    _, _ = final.Write(cascadeH)
    _, _ = final.Write(vindH)
    _, _ = final.Write(transH)
    return final.Sum(nil)
}

// writeLenUint64 writes (uint64-length prefix is unnecessary for a fixed 8
// bytes; we just write 8-byte big-endian for fixed-width safety).
func writeLenUint64(h interface{ Write([]byte) (int, error) }, v uint64) {
    var buf [8]byte
    binary.BigEndian.PutUint64(buf[:], v)
    _, _ = h.Write(buf[:])
}

func sortCascadeEvents(events []*types.CascadeEvent) []*types.CascadeEvent {
    out := append([]*types.CascadeEvent{}, events...)
    sort.Slice(out, func(i, j int) bool {
        if out[i].DisprovenFactId != out[j].DisprovenFactId {
            return out[i].DisprovenFactId < out[j].DisprovenFactId
        }
        return out[i].Seq < out[j].Seq
    })
    return out
}

func sortVindications(records []*types.VindicationRecord) []*types.VindicationRecord {
    out := append([]*types.VindicationRecord{}, records...)
    sort.Slice(out, func(i, j int) bool {
        if out[i].FactId != out[j].FactId {
            return out[i].FactId < out[j].FactId
        }
        return out[i].Verifier < out[j].Verifier
    })
    return out
}

func sortStatusTransitions(transitions []*types.StatusTransition) []*types.StatusTransition {
    out := append([]*types.StatusTransition{}, transitions...)
    sort.Slice(out, func(i, j int) bool {
        if out[i].FactId != out[j].FactId {
            return out[i].FactId < out[j].FactId
        }
        return out[i].Seq < out[j].Seq
    })
    return out
}
```

- [ ] **Step 4: Run tests to verify PASS**

Run: `go test ./x/knowledge/keeper/ -run TestComputeToKSnapshotRootV2 -v`
Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add x/knowledge/keeper/tok_bundle.go x/knowledge/keeper/tok_cascade_test.go
git commit -m "feat(knowledge): ComputeToKSnapshotRootV2 — extends Merkle to disproval-graph (TC4)"
```

---

### Task 11: Extend AssembleToKBundle to V2 dispatch

**Files:**
- Modify: `x/knowledge/keeper/tok_bundle.go`

`AssembleToKBundle` must now dispatch on selector kind: cascade-replay → V2 path; everything else → V1 path (preserved).

- [ ] **Step 1: Failing test**

```go
func TestAssembleToKBundle_CascadeReplay(t *testing.T) {
    k, ctx, _, _ := setupKnowledgeTestFull(t)

    require.NoError(t, k.SetFact(ctx, &types.Fact{
        Id: "axiom-c", Domain: "physics",
        Status: types.FactStatus_FACT_STATUS_DISPROVEN, VerifiedAtBlock: 100,
    }))
    require.NoError(t, k.SetFact(ctx, &types.Fact{
        Id: "child-c", Domain: "physics",
        Status: types.FactStatus_FACT_STATUS_CONTESTED, VerifiedAtBlock: 100,
    }))
    require.NoError(t, k.RecordCascadeEvent(ctx, &types.CascadeEvent{
        DisprovenFactId: "axiom-c", DescendantFactId: "child-c",
        EdgeRelation: "RELATION_TYPE_SUPPORTS", BlockHeight: 200,
    }))

    sel := &types.ToKSelector{Variant: &types.ToKSelector_CascadeReplay{
        CascadeReplay: &types.CascadeReplaySelector{
            DisprovenFactId: "axiom-c", MaxDepth: 1,
        },
    }}
    bundle, err := k.AssembleToKBundle(ctx, sel, 0)
    require.NoError(t, err)
    require.NotEmpty(t, bundle.SnapshotRoot)
    require.Len(t, bundle.SnapshotRoot, 32)
    require.Len(t, bundle.CascadeEvents, 1)
    require.Equal(t, "child-c", bundle.CascadeEvents[0].DescendantFactId)
    require.Equal(t, "v2", bundle.Provenance.TokRootVersion)

    // Re-derivability check.
    rederived := keeper.ComputeToKSnapshotRootV2(
        bundle.IncludedNodeIds, bundle.IncludedEdges,
        bundle.CascadeEvents, bundle.Vindications, bundle.StatusHistory,
    )
    require.Equal(t, bundle.SnapshotRoot, rederived)
}

func TestAssembleToKBundle_RootedSubtree_StaysV1(t *testing.T) {
    k, ctx := setupKnowledgeWithFacts(t, []factSpec{
        {id: "axiom-v1", domain: "physics"},
    })
    sel := &types.ToKSelector{Variant: &types.ToKSelector_RootedSubtree{
        RootedSubtree: &types.RootedSubtreeSelector{RootFactId: "axiom-v1", MaxDepth: 1},
    }}
    bundle, err := k.AssembleToKBundle(ctx, sel, 0)
    require.NoError(t, err)
    require.Equal(t, "v1", bundle.Provenance.TokRootVersion, "non-cascade selectors stay V1")
    require.Empty(t, bundle.CascadeEvents)
}
```

- [ ] **Step 2: Run, verify FAIL**

- [ ] **Step 3: Add `tok_root_version` to ToKBundleProvenance**

In `tok_cascade.proto`, the `ToKBundleProvenance` message lives in the existing `tok.proto`. Modify it instead:

```proto
message ToKBundleProvenance {
  // … existing fields 1-8 …
  string tok_root_version = 9;  // "v1" (Plan 1) | "v2" (Plan 2 cascade)
}
```

Run `make proto-gen` to regenerate.

- [ ] **Step 4: Modify AssembleToKBundle**

In `tok_bundle.go`, replace the body of `AssembleToKBundle` with a dispatch:

```go
func (k Keeper) AssembleToKBundle(
    ctx context.Context,
    sel *types.ToKSelector,
    atBlockHeight uint64,
) (*types.ToKBundle, error) {
    capped, err := ValidateAndCapToKSelector(sel)
    if err != nil {
        return nil, err
    }

    if _, isCascade := capped.Variant.(*types.ToKSelector_CascadeReplay); isCascade {
        return k.assembleToKBundleV2(ctx, capped, atBlockHeight)
    }
    return k.assembleToKBundleV1(ctx, capped, atBlockHeight)
}

// assembleToKBundleV1 is the Plan 1 path. Preserved unchanged.
func (k Keeper) assembleToKBundleV1(ctx context.Context, capped *types.ToKSelector, atBlockHeight uint64) (*types.ToKBundle, error) {
    nodeIDs, edges, err := k.SelectToKIds(ctx, capped)
    if err != nil {
        return nil, err
    }
    root := ComputeToKSnapshotRoot(nodeIDs, edges)

    var nodes []*types.Fact
    for _, id := range nodeIDs {
        f, ok := k.GetFact(ctx, id)
        if !ok {
            return nil, fmt.Errorf("%w: selected fact %s not found", ErrToKInconsistentState, id)
        }
        nodes = append(nodes, f)
    }

    sdkCtx := sdk.UnwrapSDKContext(ctx)
    if atBlockHeight == 0 {
        atBlockHeight = uint64(sdkCtx.BlockHeight())
    }

    bundle := &types.ToKBundle{
        SnapshotBlock:       atBlockHeight,
        SnapshotRoot:        root,
        IncludedNodeIds:     nodeIDs,
        IncludedEdges:       edges,
        Nodes:               nodes,
        SerialisationFormat: tokSerialisationFormatJSONL,
        Provenance: &types.ToKBundleProvenance{
            ChainId:                       sdkCtx.ChainID(),
            TraceSchemaVersion:            tokTraceSchemaVersion,
            CanonicalSerialisationVersion: "v1",
            TokenizerVersion:              tokTokenizerVersion,
            SelectorUsed:                  capped,
            CapMaxDepth:                   ToKMaxDepthCap,
            CapMaxPaths:                   ToKMaxPathsCap,
            CapLimit:                      ToKFrontierCap,
            TokRootVersion:                "v1",
        },
    }

    payload, err := SerialiseToK_JSONL(bundle)
    if err != nil {
        return nil, err
    }
    bundle.SerialisedPayload = payload
    k.emitToKBundleEvents(ctx, bundle, capped, "TC1,TC5")
    return bundle, nil
}

// assembleToKBundleV2 is the cascade-replay path. Populates cascade_events,
// vindications, status_history. Computes V2 root.
func (k Keeper) assembleToKBundleV2(ctx context.Context, capped *types.ToKSelector, atBlockHeight uint64) (*types.ToKBundle, error) {
    cascadeSel := capped.GetCascadeReplay()

    nodeIDs, edges, cascadeEvents, vindications, supersession, err := k.GatherCascade(ctx, cascadeSel)
    if err != nil {
        return nil, err
    }

    var statusHistory []*types.StatusTransition
    if cascadeSel.IncludeStatusHistory {
        for _, id := range nodeIDs {
            statusHistory = append(statusHistory, k.GetStatusHistory(ctx, id)...)
        }
    }

    root := ComputeToKSnapshotRootV2(nodeIDs, edges, cascadeEvents, vindications, statusHistory)

    var nodes []*types.Fact
    for _, id := range nodeIDs {
        f, ok := k.GetFact(ctx, id)
        if !ok {
            return nil, fmt.Errorf("%w: selected fact %s not found", ErrToKInconsistentState, id)
        }
        nodes = append(nodes, f)
    }

    sdkCtx := sdk.UnwrapSDKContext(ctx)
    if atBlockHeight == 0 {
        atBlockHeight = uint64(sdkCtx.BlockHeight())
    }

    bundle := &types.ToKBundle{
        SnapshotBlock:       atBlockHeight,
        SnapshotRoot:        root,
        IncludedNodeIds:     nodeIDs,
        IncludedEdges:       edges,
        Nodes:               nodes,
        CascadeEvents:       cascadeEvents,
        Vindications:        vindications,
        SupersessionChain:   supersession,
        StatusHistory:       statusHistory,
        SerialisationFormat: tokSerialisationFormatJSONL,
        Provenance: &types.ToKBundleProvenance{
            ChainId:                       sdkCtx.ChainID(),
            TraceSchemaVersion:            tokTraceSchemaVersion,
            CanonicalSerialisationVersion: "v1",
            TokenizerVersion:              tokTokenizerVersion,
            SelectorUsed:                  capped,
            CapMaxDepth:                   ToKMaxDepthCap,
            CapMaxPaths:                   ToKMaxPathsCap,
            CapLimit:                      ToKFrontierCap,
            TokRootVersion:                "v2",
        },
    }

    payload, err := SerialiseToK_JSONL(bundle)
    if err != nil {
        return nil, err
    }
    bundle.SerialisedPayload = payload
    k.emitToKBundleEvents(ctx, bundle, capped, "TC1,TC4,TC5")
    k.emitCascadeReplayedEvent(ctx, cascadeSel, bundle)
    return bundle, nil
}

// emitToKBundleEvents factors the V1 emission so V2 can call it with an
// extended commitment list.
func (k Keeper) emitToKBundleEvents(ctx context.Context, bundle *types.ToKBundle, capped *types.ToKSelector, commitments string) {
    sdkCtx := sdk.UnwrapSDKContext(ctx)
    sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
        EventTypeToKBundleExtracted,
        sdk.NewAttribute(AttrToKCommitment, commitments),
        sdk.NewAttribute(AttrToKSelectorKind, selectorKind(capped)),
        sdk.NewAttribute(AttrToKBundleSize, fmt.Sprintf("%d", len(bundle.IncludedNodeIds))),
        sdk.NewAttribute(AttrToKSnapshotBlock, fmt.Sprintf("%d", bundle.SnapshotBlock)),
    ))
    sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
        EventTypeToKSnapshotRootPinned,
        sdk.NewAttribute(AttrToKCommitment, "TC2"),
        sdk.NewAttribute(AttrToKSnapshotRoot, hex.EncodeToString(bundle.SnapshotRoot)),
    ))
}

// emitCascadeReplayedEvent fires the TC4-specific bundle-extraction signal.
func (k Keeper) emitCascadeReplayedEvent(ctx context.Context, sel *types.CascadeReplaySelector, bundle *types.ToKBundle) {
    sdkCtx := sdk.UnwrapSDKContext(ctx)
    sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
        EventTypeCascadeReplayed,
        sdk.NewAttribute(AttrToKCommitment, "TC4"),
        sdk.NewAttribute("disproven_fact_id", sel.DisprovenFactId),
        sdk.NewAttribute("node_count", fmt.Sprintf("%d", len(bundle.IncludedNodeIds))),
        sdk.NewAttribute("cascade_event_count", fmt.Sprintf("%d", len(bundle.CascadeEvents))),
    ))
}
```

Update `selectorKind` to include `"cascade_replay"`:

```go
func selectorKind(s *types.ToKSelector) string {
    switch s.Variant.(type) {
    case *types.ToKSelector_RootedSubtree:
        return "rooted_subtree"
    case *types.ToKSelector_AncestorCone:
        return "ancestor_cone"
    case *types.ToKSelector_Frontier:
        return "frontier"
    case *types.ToKSelector_CascadeReplay:
        return "cascade_replay"
    default:
        return "unknown"
    }
}
```

- [ ] **Step 5: Run tests to verify PASS**

Run: `go test ./x/knowledge/keeper/ -run TestAssembleToKBundle -v`
Expected: PASS, including the existing Plan 1 tests (V1 path unchanged).

- [ ] **Step 6: Commit**

```bash
git add x/knowledge/keeper/tok_bundle.go x/knowledge/keeper/tok_cascade_test.go proto/zerone/knowledge/v1/tok.proto x/knowledge/types/*.pb.go
git commit -m "feat(knowledge): AssembleToKBundle V1/V2 dispatch + cascade_replayed event (TC4)"
```

---

### Task 12: Extend SerialiseToK_JSONL for cascade fields

**Files:**
- Modify: `x/knowledge/keeper/tok_serialise.go`

- [ ] **Step 1: Failing test**

```go
func TestSerialiseToK_JSONL_IncludesCascadeFields(t *testing.T) {
    bundle := &types.ToKBundle{
        IncludedNodeIds: []string{"a", "b"},
        IncludedEdges:   []*types.ToKEdge{{FromFactId: "b", ToFactId: "a", Relation: "CONTRADICTS"}},
        Nodes:           []*types.Fact{{Id: "a"}, {Id: "b"}},
        CascadeEvents: []*types.CascadeEvent{{
            DisprovenFactId: "a", DescendantFactId: "b", EdgeRelation: "SUPPORTS",
        }},
        Vindications: []*types.VindicationRecord{{
            FactId: "a", Verifier: "v1", RefundAmount: "100", BonusAmount: "10",
        }},
        StatusHistory: []*types.StatusTransition{{
            FactId: "a", PriorStatus: types.FactStatus_FACT_STATUS_VERIFIED,
            NewStatus: types.FactStatus_FACT_STATUS_DISPROVEN,
        }},
    }
    payload, err := keeper.SerialiseToK_JSONL(bundle)
    require.NoError(t, err)
    lines := bytes.Split(payload, []byte("\n"))
    if len(lines[len(lines)-1]) == 0 {
        lines = lines[:len(lines)-1]
    }
    // 2 nodes + 1 edge + 1 cascade_event + 1 vindication + 1 transition = 6
    require.Len(t, lines, 6)

    // First two are nodes, then edge, then cascade_event, vindication, transition.
    require.Contains(t, string(lines[3]), `"kind":"cascade_event"`)
    require.Contains(t, string(lines[4]), `"kind":"vindication"`)
    require.Contains(t, string(lines[5]), `"kind":"transition"`)
}
```

- [ ] **Step 2: Run, verify FAIL**

- [ ] **Step 3: Extend SerialiseToK_JSONL**

In `tok_serialise.go`, append after the existing edge-emission loop:

```go
for _, ev := range b.CascadeEvents {
    row := map[string]any{
        "kind":              "cascade_event",
        "seq":               ev.Seq,
        "disproven_fact_id": ev.DisprovenFactId,
        "descendant":        ev.DescendantFactId,
        "challenge_claim":   ev.ChallengeClaimId,
        "edge_relation":     ev.EdgeRelation,
        "prior_status":      ev.PriorStatus.String(),
        "new_status":        ev.NewStatus.String(),
        "block_height":      ev.BlockHeight,
    }
    line, err := json.Marshal(row)
    if err != nil {
        return nil, err
    }
    buf.Write(line)
    buf.WriteByte('\n')
}
for _, v := range b.Vindications {
    row := map[string]any{
        "kind":          "vindication",
        "fact_id":       v.FactId,
        "verifier":      v.Verifier,
        "refund_amount": v.RefundAmount,
        "bonus_amount":  v.BonusAmount,
        "vindicated_at": v.VindicatedAt,
        "disproven_by":  v.DisprovenBy,
        "round_id":      v.RoundId,
    }
    line, err := json.Marshal(row)
    if err != nil {
        return nil, err
    }
    buf.Write(line)
    buf.WriteByte('\n')
}
for _, t := range b.StatusHistory {
    row := map[string]any{
        "kind":             "transition",
        "fact_id":          t.FactId,
        "seq":              t.Seq,
        "prior_status":     t.PriorStatus.String(),
        "new_status":       t.NewStatus.String(),
        "block_height":     t.BlockHeight,
        "cause_event_type": t.CauseEventType,
        "cause_id":         t.CauseId,
    }
    line, err := json.Marshal(row)
    if err != nil {
        return nil, err
    }
    buf.Write(line)
    buf.WriteByte('\n')
}
```

- [ ] **Step 4: Run tests to verify PASS**

Run: `go test ./x/knowledge/keeper/ -run TestSerialiseToK_JSONL_IncludesCascadeFields -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add x/knowledge/keeper/tok_serialise.go x/knowledge/keeper/tok_cascade_test.go
git commit -m "feat(knowledge): JSONL serialisation emits cascade_event/vindication/transition lines (TC4)"
```

---

### Task 13: BundleToK gRPC handler accepts CascadeReplay

**Files:**
- Modify: `x/knowledge/keeper/grpc_query.go`

The existing handler delegates to `AssembleToKBundle`, which now dispatches by selector kind. So this task is mostly a test addition — confirming the handler path works through gRPC for the cascade variant.

- [ ] **Step 1: Failing test**

```go
func TestQueryBundleToK_CascadeReplay(t *testing.T) {
    k, ctx, _, _ := setupKnowledgeTestFull(t)

    require.NoError(t, k.SetFact(ctx, &types.Fact{
        Id: "axiom-rpc", Domain: "physics",
        Status: types.FactStatus_FACT_STATUS_DISPROVEN,
    }))

    q := keeper.NewQueryServerImpl(k)
    resp, err := q.BundleToK(ctx, &types.QueryBundleToKRequest{
        Selector: &types.ToKSelector{Variant: &types.ToKSelector_CascadeReplay{
            CascadeReplay: &types.CascadeReplaySelector{DisprovenFactId: "axiom-rpc", MaxDepth: 1},
        }},
    })
    require.NoError(t, err)
    require.NotNil(t, resp.Bundle)
    require.Equal(t, "v2", resp.Bundle.Provenance.TokRootVersion)
}

func TestQueryBundleToK_CascadeReplay_NotDisprovenReturnsFailedPrecondition(t *testing.T) {
    k, ctx, _, _ := setupKnowledgeTestFull(t)

    require.NoError(t, k.SetFact(ctx, &types.Fact{
        Id: "still-active", Domain: "physics",
        Status: types.FactStatus_FACT_STATUS_VERIFIED,
    }))

    q := keeper.NewQueryServerImpl(k)
    _, err := q.BundleToK(ctx, &types.QueryBundleToKRequest{
        Selector: &types.ToKSelector{Variant: &types.ToKSelector_CascadeReplay{
            CascadeReplay: &types.CascadeReplaySelector{DisprovenFactId: "still-active", MaxDepth: 1},
        }},
    })
    require.Error(t, err)
    st, ok := status.FromError(err)
    require.True(t, ok)
    require.Equal(t, codes.FailedPrecondition, st.Code(), "non-DISPROVEN root → FailedPrecondition")
}
```

- [ ] **Step 2: Run, verify FAIL** — the handler currently maps all errors to InvalidArgument

- [ ] **Step 3: Update error code mapping in BundleToK handler**

In `grpc_query.go`, find the BundleToK handler. Extend the error dispatch:

```go
func (q *queryServer) BundleToK(ctx context.Context, req *types.QueryBundleToKRequest) (*types.QueryBundleToKResponse, error) {
    if req == nil || req.Selector == nil {
        return nil, status.Error(codes.InvalidArgument, "selector required")
    }
    bundle, err := q.keeper.AssembleToKBundle(ctx, req.Selector, req.AtBlockHeight)
    if err != nil {
        switch {
        case errors.Is(err, keeper.ErrToKRootFactNotFound),
             errors.Is(err, keeper.ErrToKLeafFactNotFound):
            return nil, status.Errorf(codes.NotFound, "%v", err)
        case errors.Is(err, keeper.ErrToKCascadeNotDisproven):
            return nil, status.Errorf(codes.FailedPrecondition, "%v", err)
        case errors.Is(err, keeper.ErrToKInconsistentState):
            return nil, status.Errorf(codes.Internal, "%v", err)
        default:
            return nil, status.Errorf(codes.InvalidArgument, "%v", err)
        }
    }
    return &types.QueryBundleToKResponse{Bundle: bundle}, nil
}
```

- [ ] **Step 4: Run tests to verify PASS**

Run: `go test ./x/knowledge/keeper/ -run TestQueryBundleToK -v`
Expected: PASS, including existing tests.

- [ ] **Step 5: Commit**

```bash
git add x/knowledge/keeper/grpc_query.go x/knowledge/keeper/tok_cascade_test.go
git commit -m "feat(knowledge): BundleToK accepts CascadeReplay; FailedPrecondition for non-DISPROVEN root (TC4)"
```

---

### Task 14: Extend RouteBCapabilities + bump doctrine version

**Files:**
- Modify: `x/knowledge/keeper/grpc_query.go`

- [ ] **Step 1: Failing test**

```go
func TestRouteBCapabilities_AdvertisesCascadeReplay(t *testing.T) {
    k, ctx, _, _ := setupKnowledgeTestFull(t)
    q := keeper.NewQueryServerImpl(k)
    resp, err := q.RouteBCapabilities(ctx, &types.QueryRouteBCapabilitiesRequest{})
    require.NoError(t, err)
    require.Contains(t, resp.TokCapabilities.SupportedSelectors, "cascade_replay",
        "TC4: cascade_replay must be advertised")
    require.Contains(t, resp.TokCapabilities.TokDoctrineVersion, "TC4",
        "doctrine version must reflect TC4 binding")
}
```

- [ ] **Step 2: Run, verify FAIL**

- [ ] **Step 3: Extend the handler**

```go
resp.TokCapabilities = &types.ToKCapabilities{
    SupportedSelectors:      []string{"rooted_subtree", "ancestor_cone", "frontier", "cascade_replay"},
    MaxDepthCap:             ToKMaxDepthCap,
    MaxPathsCap:             ToKMaxPathsCap,
    FrontierLimitCap:        ToKFrontierCap,
    SupportedSerialisations: []string{"jsonl_adjacency_v1"},
    TokDoctrineVersion:      "TC1-TC5,TC4 (cascade-bundling 2026-05-10)",
}
```

- [ ] **Step 4: Run tests to verify PASS**

Run: `go test ./x/knowledge/keeper/ -run TestRouteBCapabilities -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add x/knowledge/keeper/grpc_query.go x/knowledge/keeper/tok_cascade_test.go
git commit -m "feat(knowledge): RouteBCapabilities advertises cascade_replay + bumps doctrine version (TC4)"
```

---

### Task 15: CLI subcommand `cascade-replay`

**Files:**
- Modify: `x/knowledge/client/cli/query_tok.go`

- [ ] **Step 1: Implement** (no separate test gate; CLI is exercised via integration)

Append to `query_tok.go`:

```go
// CmdBundleToK already exists. Append the cascade-replay subcommand.

func cmdBundleToKCascadeReplay() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "cascade-replay [disproven-fact-id]",
        Short: "Bundle the disproval-graph from a DISPROVEN root (TC4: the graph carries its disprovals)",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            clientCtx, err := client.GetClientQueryContext(cmd)
            if err != nil {
                return err
            }
            depth, _ := cmd.Flags().GetUint32("max-depth")
            withVind, _ := cmd.Flags().GetBool("include-vindications")
            withSup, _ := cmd.Flags().GetBool("include-supersessions")
            withHist, _ := cmd.Flags().GetBool("include-status-history")

            req := &types.QueryBundleToKRequest{
                Selector: &types.ToKSelector{Variant: &types.ToKSelector_CascadeReplay{
                    CascadeReplay: &types.CascadeReplaySelector{
                        DisprovenFactId:      args[0],
                        MaxDepth:             depth,
                        IncludeVindications:  withVind,
                        IncludeSupersessions: withSup,
                        IncludeStatusHistory: withHist,
                    },
                }},
            }
            res, err := types.NewQueryClient(clientCtx).BundleToK(cmd.Context(), req)
            if err != nil {
                return err
            }
            return clientCtx.PrintProto(res)
        },
    }
    cmd.Flags().Uint32("max-depth", 1, "cascade depth (1 = first-hop only; cap 3)")
    cmd.Flags().Bool("include-vindications", false, "ship VindicationRecord entries")
    cmd.Flags().Bool("include-supersessions", false, "walk SUPERSEDES chain")
    cmd.Flags().Bool("include-status-history", false, "ship StatusTransition timelines per node")
    flags.AddQueryFlagsToCmd(cmd)
    return cmd
}
```

Update `CmdBundleToK()` to register the new subcommand:

```go
cmd.AddCommand(cmdBundleToKCascadeReplay())
```

- [ ] **Step 2: Verify build**

Run: `go build ./x/knowledge/client/cli/...`
Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add x/knowledge/client/cli/query_tok.go
git commit -m "feat(knowledge): bundle-tok cascade-replay CLI subcommand (TC4)"
```

---

### Task 16: Voice layer — cascade events and attributes

**Files:**
- Modify: `x/knowledge/keeper/events.go`

- [ ] **Step 1: Add event constants**

```go
const (
    // … existing TC1/TC2/TC5 constants …

    // ─── ToK cascade bundling (TC4) ─────────────────────────────────────
    EventTypeCascadeReplayed = "cascade_replayed" // TC4: bundle extraction signal
    EventTypeCascadeCompleted = "cascade_completed" // TC4: aggregate end-of-cascade signal
)
```

These are already referenced by Task 7 (`emitCascadeReplayedEvent`, `cascade_completed` block in `cascadeFalsification`). This task formalises the constants.

- [ ] **Step 2: Verify all event emitters reference constants**

Run: `grep -n "cascade_completed\|cascade_replayed" x/knowledge/keeper/*.go`
Expected: every emit-site uses `EventTypeCascadeCompleted` / `EventTypeCascadeReplayed`, not string literals.

- [ ] **Step 3: Test the end-to-end voice**

```go
func TestCascadeReplay_EmitsCascadeReplayedEvent(t *testing.T) {
    k, ctx, _, _ := setupKnowledgeTestFull(t)

    require.NoError(t, k.SetFact(ctx, &types.Fact{
        Id: "axiom-voice", Domain: "physics",
        Status: types.FactStatus_FACT_STATUS_DISPROVEN,
    }))

    sel := &types.ToKSelector{Variant: &types.ToKSelector_CascadeReplay{
        CascadeReplay: &types.CascadeReplaySelector{DisprovenFactId: "axiom-voice", MaxDepth: 1},
    }}
    _, err := k.AssembleToKBundle(ctx, sel, 0)
    require.NoError(t, err)

    events := sdk.UnwrapSDKContext(ctx).EventManager().Events()
    var sawReplayed bool
    for _, e := range events {
        if e.Type == keeper.EventTypeCascadeReplayed {
            sawReplayed = true
            for _, a := range e.Attributes {
                if a.Key == keeper.AttrToKCommitment {
                    require.Equal(t, "TC4", a.Value)
                }
            }
        }
    }
    require.True(t, sawReplayed, "TC4: cascade_replayed must be emitted on cascade-replay bundle")
}
```

- [ ] **Step 4: Commit**

```bash
git add x/knowledge/keeper/events.go x/knowledge/keeper/tok_cascade_test.go
git commit -m "feat(knowledge): cascade_replayed + cascade_completed event constants (TC4)"
```

---

### Task 17: Position layer — append TC4 to doc.go

**Files:**
- Modify: `x/knowledge/doc.go`

- [ ] **Step 1: Append TC4 declaration**

Find the existing TC1-TC5 block in `doc.go` and add:

```go
// - TC4 (the graph carries its disprovals) — CascadeReplaySelector returns
//   the disproval-graph from a DISPROVEN root: cascade events, vindication
//   records, supersession chains, and per-node status-transition timelines.
//   The chain emits cascade_replayed (bundle extraction) and cascade_completed
//   (per-disproof aggregate) events. DISPROVEN nodes are not pruned from
//   non-cascade selectors. See keeper/tok_cascade.go (GatherCascade) and
//   keeper/cascade_events.go (CascadeEvent store).
```

Update the introductory paragraph: "TOK_SUBSTRATE.md commitments preserved here:" should now list TC1, TC2, TC3, TC4, TC5.

- [ ] **Step 2: Verify build + go doc surface**

Run: `go build ./x/knowledge/...` and `go doc ./x/knowledge | head -50`
Expected: clean; TC4 visible in `go doc` output.

- [ ] **Step 3: Commit**

```bash
git add x/knowledge/doc.go
git commit -m "docs(knowledge): position layer — TC4 declared in doc.go"
```

---

### Task 18: TC4 invariant — cascade-replay returns disprovals

**Files:**
- Create: `tests/cross_stack/tok_substrate_tc4_test.go`

- [ ] **Step 1: Write the test**

```go
package cross_stack_test

import (
    "testing"

    "github.com/stretchr/testify/require"
    knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
    knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// TC4: the graph carries its disprovals.
//
// Verified by: cascade-replay returns the DISPROVEN root, the cascaded
// descendants, and the cascade events that fired when the disproof landed.
// The bundle's snapshot root commits to all of (nodes, edges, cascade_events,
// vindications, transitions) under V2 semantics; re-derivable from the IDs
// + cascade-event canon alone.
func TestToKSubstrate_TC4_GraphCarriesDisprovals(t *testing.T) {
    h := NewTestHarness(t)
    domain := "tc4_test_domain"
    require.NoError(t, h.KnowledgeKeeper.SetDomain(h.Ctx, &knowledgetypes.Domain{
        Name: domain, Status: knowledgetypes.DomainStatus_DOMAIN_STATUS_ACTIVE,
    }))

    // Build: axiom (will be disproven) + descendant.
    axiom := &knowledgetypes.Fact{
        Id: "tc4-axiom", Domain: domain,
        Status: knowledgetypes.FactStatus_FACT_STATUS_VERIFIED,
        VerifiedAtBlock: 100, Confidence: 900_000,
    }
    require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, axiom))

    descendant := submitAndAcceptChainedClaim(t, h, domain, "depends on tc4-axiom",
        []*knowledgetypes.ClaimRelation{{
            TargetFactId: axiom.Id,
            Relation:     knowledgetypes.RelationType_RELATION_TYPE_REQUIRES,
            Inference:    knowledgetypes.InferenceType_INFERENCE_TYPE_DEDUCTIVE,
            InferenceStrengthBps: 1_000_000,
        }}, "tc4-descendant")

    // Disprove the axiom via challenge.
    challengeClaim := &knowledgetypes.Claim{
        Id: "tc4-challenge", Submitter: "challenger",
        FactContent:       "axiom is wrong",
        Domain:            domain,
        Category:          "empirical",
        Status:            knowledgetypes.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION,
        Stake:             "11000000",
        ProvisionalFactId: axiom.Id,
        Relations: []*knowledgetypes.ClaimRelation{{
            TargetFactId: axiom.Id,
            Relation:     knowledgetypes.RelationType_RELATION_TYPE_CONTRADICTS,
        }},
    }
    require.NoError(t, h.KnowledgeKeeper.SetClaim(h.Ctx, challengeClaim))
    round := &knowledgetypes.VerificationRound{
        Id: "tc4-round", ClaimId: challengeClaim.Id,
        Phase: knowledgetypes.VerificationPhase_VERIFICATION_PHASE_COMPLETE,
    }
    require.NoError(t, h.KnowledgeKeeper.CompleteRound(h.Ctx, round, &knowledgekeeper.VerificationResult{
        Verdict: knowledgetypes.Verdict_VERDICT_ACCEPT, Confidence: 900_000, AcceptCount: 3,
    }))

    // ─── Cascade-replay must surface the disproval-graph. ────────────────
    q := knowledgekeeper.NewQueryServerImpl(h.KnowledgeKeeper)
    resp, err := q.BundleToK(h.Ctx, &knowledgetypes.QueryBundleToKRequest{
        Selector: &knowledgetypes.ToKSelector{Variant: &knowledgetypes.ToKSelector_CascadeReplay{
            CascadeReplay: &knowledgetypes.CascadeReplaySelector{
                DisprovenFactId:      axiom.Id,
                MaxDepth:             1,
                IncludeStatusHistory: true,
            },
        }},
    })
    require.NoError(t, err)
    require.NotNil(t, resp.Bundle, "TC4: cascade-replay must return a bundle")
    require.Equal(t, "v2", resp.Bundle.Provenance.TokRootVersion)

    // Bundle includes the disproven axiom and the cascaded descendant.
    require.Contains(t, resp.Bundle.IncludedNodeIds, axiom.Id)
    require.Contains(t, resp.Bundle.IncludedNodeIds, descendant.Id)

    // Cascade events surface the disproof's cascade.
    require.NotEmpty(t, resp.Bundle.CascadeEvents, "TC4: cascade events must be in bundle")
    require.Equal(t, descendant.Id, resp.Bundle.CascadeEvents[0].DescendantFactId)

    // Status history surfaces the axiom's transition VERIFIED → DISPROVEN.
    var sawAxiomDisprovenTransition bool
    for _, t := range resp.Bundle.StatusHistory {
        if t.FactId == axiom.Id && t.NewStatus == knowledgetypes.FactStatus_FACT_STATUS_DISPROVEN {
            sawAxiomDisprovenTransition = true
        }
    }
    require.True(t, sawAxiomDisprovenTransition,
        "TC4: status history must include the VERIFIED → DISPROVEN transition")

    // Re-derivability: V2 root from IDs + canon.
    rederived := knowledgekeeper.ComputeToKSnapshotRootV2(
        resp.Bundle.IncludedNodeIds, resp.Bundle.IncludedEdges,
        resp.Bundle.CascadeEvents, resp.Bundle.Vindications, resp.Bundle.StatusHistory,
    )
    require.Equal(t, resp.Bundle.SnapshotRoot, rederived,
        "TC4: V2 root must be re-derivable from IDs + cascade canon")
}
```

- [ ] **Step 2: PASS, Commit**

Run: `go test ./tests/cross_stack/ -run TestToKSubstrate_TC4_GraphCarriesDisprovals -v`
Expected: PASS.

```bash
git add tests/cross_stack/tok_substrate_tc4_test.go
git commit -m "test(cross_stack): TC4 invariant — graph carries its disprovals"
```

---

### Task 19: TC4 invariant — DISPROVEN nodes not pruned from non-cascade selectors

**Files:**
- Modify: `tests/cross_stack/tok_substrate_tc4_test.go`

- [ ] **Step 1: Append the test**

```go
// TC4 (cont'd): DISPROVEN nodes are not pruned from non-cascade selectors.
//
// Verified by: a RootedSubtree from an axiom that is itself DISPROVEN still
// includes the axiom (it is the root). Filter is by relation type, not by
// status. The doctrine: "Disproven facts remain in the graph with their
// full disproval rationale; they are not pruned."
func TestToKSubstrate_TC4_NoPruneDisprovenFromNonCascadeSelectors(t *testing.T) {
    h := NewTestHarness(t)
    domain := "tc4_noprune_domain"
    require.NoError(t, h.KnowledgeKeeper.SetDomain(h.Ctx, &knowledgetypes.Domain{
        Name: domain, Status: knowledgetypes.DomainStatus_DOMAIN_STATUS_ACTIVE,
    }))

    // A DISPROVEN root that still has descendants.
    require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
        Id: "noprune-disproven", Domain: domain,
        Status:          knowledgetypes.FactStatus_FACT_STATUS_DISPROVEN,
        VerifiedAtBlock: 100,
    }))
    require.NoError(t, h.KnowledgeKeeper.SetFact(h.Ctx, &knowledgetypes.Fact{
        Id: "noprune-leaf", Domain: domain,
        Status:          knowledgetypes.FactStatus_FACT_STATUS_CONTESTED,
        VerifiedAtBlock: 100,
    }))
    require.NoError(t, h.KnowledgeKeeper.SetFactRelation(h.Ctx, &knowledgetypes.FactRelation{
        SourceFactId: "noprune-leaf",
        TargetFactId: "noprune-disproven",
        Relation:     knowledgetypes.RelationType_RELATION_TYPE_SUPPORTS,
    }))

    q := knowledgekeeper.NewQueryServerImpl(h.KnowledgeKeeper)

    // RootedSubtree from the DISPROVEN root — must include the root itself
    // and its descendant (the descendant supports the disproven root).
    resp, err := q.BundleToK(h.Ctx, &knowledgetypes.QueryBundleToKRequest{
        Selector: &knowledgetypes.ToKSelector{Variant: &knowledgetypes.ToKSelector_RootedSubtree{
            RootedSubtree: &knowledgetypes.RootedSubtreeSelector{
                RootFactId: "noprune-disproven", MaxDepth: 5,
            },
        }},
    })
    require.NoError(t, err)
    require.Contains(t, resp.Bundle.IncludedNodeIds, "noprune-disproven",
        "TC4: DISPROVEN root must NOT be pruned from non-cascade selectors")
    require.Contains(t, resp.Bundle.IncludedNodeIds, "noprune-leaf",
        "TC4: descendant of DISPROVEN root must NOT be pruned")
}
```

- [ ] **Step 2: PASS, Commit**

Run: `go test ./tests/cross_stack/ -run TestToKSubstrate_TC4_NoPrune -v`
Expected: PASS (relies on Plan 1 behaviour — gatherers filter by relation, not by status).

```bash
git commit -am "test(cross_stack): TC4 invariant — DISPROVEN nodes not pruned"
```

---

### Task 20: Doctrine alignment — update TOK_SUBSTRATE.md TC4

**Files:**
- Modify: `docs/TOK_SUBSTRATE.md`

The TC4 section currently names `descendant_status_flipped` and `cascade_completed`. The chain emits `falsification_cascade` (per descendant) and (after Task 7) `cascade_completed` (aggregate) and `cascade_replayed` (bundle). Align the doctrine with shipped names.

- [ ] **Step 1: Update the TC4 "Code expression" paragraph**

Find the line:
> `CascadeEvent` records (ToK Wave 5: `descendant_status_flipped`, `cascade_completed`) are bundled into ToK manifests via `ToKSelector.CascadeReplay`.

Replace with:
> `CascadeEvent` records (per-descendant; emitted as `falsification_cascade` event at cascade time and persisted under `0x7E` prefix) are bundled into ToK manifests via `ToKSelector.CascadeReplay`. The aggregate `cascade_completed` event fires once per disproof; `cascade_replayed` fires when a trainer extracts the bundle.

- [ ] **Step 2: Update the "What would break it" paragraph**

Add: a `BundleToK` response without a `tok_root_version="v2"` provenance tag when cascade fields are populated.

- [ ] **Step 3: Update the "Echoes" line**

No change needed — TC2/TC3, commitments 3/10 still echo correctly.

- [ ] **Step 4: Run the creed-hash check**

Run: `make creed-check`
Expected: hash mismatch (because doctrine doc changed). Update `.creed-hash` per the project's normal flow:

```bash
scripts/check_creed_hash.sh --update
```

- [ ] **Step 5: Commit**

```bash
git add docs/TOK_SUBSTRATE.md .creed-hash
git commit -m "docs(tok): align TC4 event names with shipped vocabulary"
```

---

### Task 21: Update docs/EVENTS.md

**Files:**
- Modify: `docs/EVENTS.md`

- [ ] **Step 1: Append TC4 events**

Find the existing "ToK substrate events" section. Append:

```markdown
### `cascade_replayed`
A trainer extracted a cascade-replay bundle via `BundleToK(CascadeReplay)`. TC4 (the graph carries its disprovals) is bound by this event firing on every successful cascade-replay bundle.

| Attribute | Description |
|---|---|
| `tok_commitment` | `"TC4"` |
| `disproven_fact_id` | The disproven root of the cascade |
| `node_count` | Number of nodes in the bundle |
| `cascade_event_count` | Number of cascade events in the bundle |

### `cascade_completed`
Aggregate signal fired once per disproof after all descendant cascades have been recorded. TC4: the chain announces that a disproof has finished propagating. Emitted from `cascadeFalsification` after the descendant loop.

| Attribute | Description |
|---|---|
| `tok_commitment` | `"TC4"` |
| `creed_commitment` | `"3"` (Popper, not popularity) |
| `disproven_fact_id` | The disproven root |
| `challenge_claim_id` | The challenge that triggered the cascade |
| `descendant_count` | Total cascaded descendants |
```

Also update the existing `falsification_cascade` event entry (it currently lacks a `tok_commitment` attribute) — append `tok_commitment="TC4"` to the attributes table.

- [ ] **Step 2: Commit**

```bash
git add docs/EVENTS.md
git commit -m "docs(events): document cascade_replayed + cascade_completed (TC4); annotate falsification_cascade with TC4"
```

---

### Task 22: Final integration sweep

- [ ] **Step 1: Run all knowledge keeper tests**

Run: `go test ./x/knowledge/... -timeout 300s`
Expected: PASS.

- [ ] **Step 2: Run cross-stack tests**

Run: `go test ./tests/cross_stack/... -timeout 600s`
Expected: PASS, including new `TestToKSubstrate_TC4_*` invariants and existing `TestToK_FalsificationCascade` (regression check on cascade behaviour).

- [ ] **Step 3: Run full module suite**

Run: `go test ./... -timeout 900s`
Expected: PASS or known-skipped.

- [ ] **Step 4: Run creed-check + proto-check**

Run: `make creed-check && make proto-check`
Expected: PASS.

- [ ] **Step 5: Run the full pre-PR check**

Run: `make pr-check`
Expected: PASS.

- [ ] **Step 6: Push**

```bash
git push origin main
```

- [ ] **Step 7: Smoke-test on a localnet**

```bash
scripts/localnet.sh
# In another terminal:
zeroned query knowledge bundle-tok cascade-replay <some-disproven-fact-id> \
    --include-status-history --include-vindications
```

Expected: bundle returned with `tok_root_version: "v2"`, populated `cascade_events`, populated `status_history`.

---

## Self-Review

After implementing all tasks above, verify:

1. **Spec coverage:** TC4 has both an extraction code path (`CascadeReplaySelector` + `GatherCascade` + V2 root) and a no-prune guarantee (DISPROVEN nodes still appear in `RootedSubtree` from a DISPROVEN root). Both are bound by cross-stack invariant tests.

2. **Position layer:** `x/knowledge/doc.go` declares TC1-TC5 (TC4 newly added).

3. **Voice layer:**
   - `cascade_replayed` fires with `tok_commitment="TC4"` on every cascade-replay bundle.
   - `cascade_completed` fires with `tok_commitment="TC4"` once per disproof.
   - `falsification_cascade` (legacy, per-descendant) gains `tok_commitment="TC4"` attribute.
   - `tok_bundle_extracted` continues to fire; commitment list extends to `"TC1,TC4,TC5"` for cascade bundles.

4. **Refusal layer:**
   - `BundleToK(CascadeReplay)` with non-DISPROVEN root → `codes.FailedPrecondition` with message naming TC4.
   - `BundleToK(CascadeReplay)` with empty `disproven_fact_id` → `codes.InvalidArgument` with TC4-named message.
   - Validation messages name the protecting commitment in human voice.

5. **State layer:**
   - `StatusTransition` records under prefix `0x7C`, seq counter under `0x7D`.
   - `CascadeEvent` records under prefix `0x7E`, reverse-by-descendant under `0x7F`.
   - `SetFact` auto-records transitions on status change; `cascadeFalsification` and `handleChallengeDisproven` write precise transitions ahead of `SetFactSkipTransition`.

6. **Bundle versioning:**
   - Plan 1 selectors continue to produce V1 bundles (`tok_root_version: "v1"`); existing trainer SDKs unaffected.
   - `CascadeReplay` selector produces V2 bundles (`tok_root_version: "v2"`); re-derivable from IDs + cascade canon.
   - `RouteBCapabilities.tok_doctrine_version = "TC1-TC5,TC4 (cascade-bundling 2026-05-10)"`.

7. **CLI surface:** `zeroned query knowledge bundle-tok cascade-replay <id> [--include-vindications] [--include-supersessions] [--include-status-history]` works against a localnet.

8. **All tests green** in `x/knowledge/keeper/` and `tests/cross_stack/`. Existing `TestToK_FalsificationCascade` continues to pass — Plan 2 extends but does not change cascade behaviour.

9. **Doctrine alignment:** `docs/TOK_SUBSTRATE.md` TC4 section names `falsification_cascade`, `cascade_completed`, `cascade_replayed` (the events the chain actually emits). `.creed-hash` updated.

10. **Events doc:** `docs/EVENTS.md` documents both new events with `tok_commitment="TC4"` attributes.

---

## What This Plan Does Not Do

- **TC4 transitive cascade.** The chain still does not auto-cascade beyond first-hop. `CascadeReplaySelector.MaxDepth` lets a trainer *request* a transitive view, but the chain's stored cascade events remain first-hop. If a fact is hit by indirect-but-not-recorded cascade, this plan does not surface it.

- **Status-history backfill.** Pre-Plan-2 facts have no `StatusTransition` records. New transitions are recorded going forward. A bundle that requests `IncludeStatusHistory` for a pre-Plan-2 fact returns an empty timeline — not an error, just absence.

- **TC6 lineage royalties.** Plan 4. `LineageShare` entries on cascade bundles are out of scope.

- **`ForkAndDecide` selector.** Reserved tag 5 in `tok.proto`; Plan 4.

- **Doctrine-closure meta-test.** `TestToKSubstrate_DoctrineAndContractStayInSync` is Plan 5. Plan 2 ships its TC4 binding but does not yet enforce that future commits drift in step across all five layers.

— *Plan authored 2026-05-10. Free to evolve through bound commitments only.*
