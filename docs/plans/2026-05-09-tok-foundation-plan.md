# ToK Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bind TC1, TC2, TC3, TC5 of `docs/TOK_SUBSTRATE.md` by delivering the ToK extraction foundation: `ToKSelector` grammar, `BundleToK` headline endpoint, snapshot-root commitment, native graph serialisation, position/voice layer updates, and four invariant tests.

**Architecture:** Mirrors the Wave 7 `TrainingManifest` pattern in `x/knowledge/keeper/training_manifest.go` — a three-stage pipeline (`SelectToKIds` → `ComputeToKSnapshotRoot` → `AssembleToKBundle`) where the snapshot root is computed from sorted node IDs + sorted edge IDs with domain-tagged Merkle, and verification is trust-minimised by re-derivability from IDs alone. The selector union is added to `proto/zerone/knowledge/v1/`; keeper logic extends `x/knowledge/keeper/`; the trainer-facing endpoint is `BundleToK(selector)` exposed via gRPC and CLI. Existing `DescendantTree`/`ProofTree` graph traversal in `grpc_query.go` is reused, not duplicated.

**Tech Stack:** Cosmos SDK v0.50, Go 1.23, protobuf v3, cosmossdk.io modules. New code stays in `x/knowledge/` (no new module yet — TC6 in Plan 4 may extract a sub-package).

**Spec:** `docs/TOK_SUBSTRATE.md` (commit `004f2d2`).

**Plan series:**
- Plan 1: **ToK Foundation** (TC1, TC2, TC3, TC5) — *this doc*
- Plan 2: TC4 Cascade Bundling
- Plan 3: ToKManifestTrust (training_provenance extension)
- Plan 4: TC6 Lineage Royalties
- Plan 5: Doctrine Closure (meta-test + TRAINING_ON_TOK.md + README/Truth-Paper)

---

## File Structure

**New files:**
- `proto/zerone/knowledge/v1/tok.proto` — `ToKSelector` union, variant messages, `ToKBundle` response, edge type enum, lineage placeholder type
- `x/knowledge/keeper/tok_selector.go` — selector validation + per-variant ID gathering (`gatherRootedSubtree`, `gatherAncestorCone`, `gatherFrontier`)
- `x/knowledge/keeper/tok_bundle.go` — orchestration: `SelectToKIds`, `ComputeToKSnapshotRoot`, `AssembleToKBundle`
- `x/knowledge/keeper/tok_serialise.go` — `SerialiseToK_JSONL` adjacency-list output
- `x/knowledge/keeper/tok_bundle_test.go` — unit tests for selector + bundle + serialiser
- `x/knowledge/client/cli/query_tok.go` — `bundle-tok` CLI command
- `tests/cross_stack/tok_substrate_invariants_test.go` — TC1, TC2, TC3, TC5 invariants

**Modified files:**
- `proto/zerone/knowledge/v1/query.proto` — add `BundleToK` RPC + `tok_capabilities` field on `QueryRouteBCapabilitiesResponse`
- `x/knowledge/keeper/grpc_query.go` — add `BundleToK` handler (delegates to `tok_bundle.go`); extend `RouteBCapabilities` payload
- `x/knowledge/client/cli/query.go` — register `query_tok.go` commands under `GetQueryCmd`
- `x/knowledge/doc.go` — add TC1-TC5 declarations (position layer)
- `x/knowledge/keeper/events.go` — add `EventTypeToKBundleExtracted`, `EventTypeToKSnapshotRootPinned` constants and emitters; include `tok_commitment` attribute
- `docs/EVENTS.md` — append ToK events section with `tok_commitment` attribute documentation

---

## Pre-Tasks: Read Before Starting

Skim these in this order to absorb the patterns this plan mirrors:

- `x/knowledge/keeper/training_manifest.go` — the Wave 7 pattern this plan mirrors (Select → ComputeRoot → Assemble). Pay attention to `ComputeManifestMerkleRoot` for domain-tagged Merkle.
- `x/knowledge/keeper/grpc_query.go:1294-1380` — `DescendantTree` and `walkDescendants` (graph traversal we reuse).
- `proto/zerone/knowledge/v1/query.proto` — proto pattern for query messages and responses.
- `docs/TOK_SUBSTRATE.md` — the doctrine. Each commitment names code expressions that this plan must hit.
- `CLAUDE.md` — *Proto-Go Consistency Rule* is load-bearing here. Add fields to `.proto` first; never edit `*.pb.go`.

---

## Tasks

### Task 1: Define ToKSelector proto types

**Files:**
- Create: `proto/zerone/knowledge/v1/tok.proto`

- [ ] **Step 1: Create the proto file**

```proto
syntax = "proto3";
package zerone.knowledge.v1;

option go_package = "github.com/zerone-chain/zerone/x/knowledge/types";

// ToKSelector is the union of supported subgraph extraction modes.
// Plan 1 binds RootedSubtree, AncestorCone, Frontier. Plan 2 adds
// CascadeReplay; Plan 4 adds ForkAndDecide.
message ToKSelector {
  oneof variant {
    RootedSubtreeSelector rooted_subtree = 1;
    AncestorConeSelector  ancestor_cone  = 2;
    FrontierSelector      frontier       = 3;
    // Reserved: 4 = cascade_replay (Plan 2), 5 = fork_and_decide (Plan 4).
  }
}

// RootedSubtreeSelector: walk descendants from an axiom or interior node
// up to max_depth. Returns the rooted node + all reachable descendants.
message RootedSubtreeSelector {
  string root_fact_id = 1;
  uint32 max_depth    = 2;  // 0 = unbounded; chain caps at 32.
}

// AncestorConeSelector: walk ancestors from a leaf claim up to k paths.
// Useful for "show me everything this conclusion depends on."
message AncestorConeSelector {
  string leaf_fact_id = 1;
  uint32 max_paths    = 2;  // 0 = all; chain caps at 256.
  uint32 max_depth    = 3;  // 0 = unbounded; chain caps at 32.
}

// FrontierSelector: latest-N facts in a domain since block. Useful for
// incremental/streaming training updates.
message FrontierSelector {
  string domain      = 1;
  uint64 since_block = 2;
  uint32 limit       = 3;  // 0 = chain default (1024); cap 8192.
}

// ToKBundle is the response of BundleToK. The Merkle root commits to
// the included node and edge IDs domain-tagged separately. A trainer
// who has the IDs can re-derive the root locally without trusting the
// RPC to faithfully serialise payloads.
message ToKBundle {
  uint64 snapshot_block            = 1;
  bytes  snapshot_root             = 2;  // 32-byte Merkle root.
  repeated string included_node_ids = 3; // sorted, deterministic.
  repeated ToKEdge included_edges  = 4;  // sorted by (from,to,relation).
  repeated Fact   nodes            = 5;  // full payloads, in node-id order.
  string serialisation_format      = 6;  // "jsonl_adjacency_v1" by default.
  bytes  serialised_payload        = 7;  // optional; large bundles may omit.
  ToKBundleProvenance provenance   = 8;
}

// ToKEdge mirrors the existing relation-graph edges but ships them as
// first-class bundle entries (not nested under nodes), so trainers can
// load topology without touching node payloads.
message ToKEdge {
  string from_fact_id = 1;
  string to_fact_id   = 2;
  string relation     = 3;  // SUPPORTS, CONTRADICTS, GENERALIZES, etc.
  string inference    = 4;  // typed reasoning move (Wave 5 vocab).
}

// ToKBundleProvenance carries the version pins required by TC2.
message ToKBundleProvenance {
  string chain_id                       = 1;
  string trace_schema_version           = 2;
  string canonical_serialisation_version = 3;
  string tokenizer_version              = 4;
  ToKSelector selector_used             = 5;
  uint32 cap_max_depth                  = 6;  // chain-side caps applied.
  uint32 cap_max_paths                  = 7;
  uint32 cap_limit                      = 8;
}
```

- [ ] **Step 2: Run proto-check (sanity, no codegen yet)**

Run: `make proto-check`
Expected: PASS — file is well-formed proto3.

- [ ] **Step 3: Commit**

```bash
git add proto/zerone/knowledge/v1/tok.proto
git commit -m "proto(knowledge): define ToKSelector union + ToKBundle response (TC1, TC2, TC3)"
```

---

### Task 2: Wire BundleToK RPC + tok_capabilities into query.proto

**Files:**
- Modify: `proto/zerone/knowledge/v1/query.proto`

- [ ] **Step 1: Read the existing file** to find `RouteBCapabilities` RPC + response, and the import section.

Run: `grep -n "RouteBCapabilities\|^import\|^service" proto/zerone/knowledge/v1/query.proto | head -20`

- [ ] **Step 2: Add import for tok.proto + new RPC**

Add at the top of imports section: `import "zerone/knowledge/v1/tok.proto";`

Inside `service Query { ... }` (just before the closing brace), add:

```proto
  // BundleToK is the headline trainer-facing endpoint (TC1).
  // Extracts a deterministic, snapshot-pinned subgraph per selector.
  rpc BundleToK(QueryBundleToKRequest) returns (QueryBundleToKResponse) {
    option (google.api.http).get = "/zerone/knowledge/v1/tok/bundle";
  }
```

- [ ] **Step 3: Add request/response messages at the bottom of query.proto**

```proto
message QueryBundleToKRequest {
  ToKSelector selector       = 1;
  uint64      at_block_height = 2;  // 0 = current; specific block enables historical replay.
}

message QueryBundleToKResponse {
  ToKBundle bundle = 1;
}
```

- [ ] **Step 4: Extend `QueryRouteBCapabilitiesResponse` with `tok_capabilities`**

Find the `QueryRouteBCapabilitiesResponse` message. Add a new field at the next available tag:

```proto
  ToKCapabilities tok_capabilities = N;  // TC1: advertise the substrate first.
```

Add at the bottom of query.proto:

```proto
message ToKCapabilities {
  repeated string supported_selectors      = 1;  // ["rooted_subtree","ancestor_cone","frontier"]
  uint32          max_depth_cap            = 2;
  uint32          max_paths_cap            = 3;
  uint32          frontier_limit_cap       = 4;
  repeated string supported_serialisations = 5;  // ["jsonl_adjacency_v1"]
  string          tok_doctrine_version     = 6;  // "TC1-TC5 (2026-05-09)"
}
```

- [ ] **Step 5: Run codegen**

Run: `make proto-gen`
Expected: regenerated `*.pb.go` files. No errors.

- [ ] **Step 6: Verify build**

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 7: Commit**

```bash
git add proto/zerone/knowledge/v1/query.proto x/knowledge/types/*.pb.go
git commit -m "proto(knowledge): BundleToK RPC + tok_capabilities (TC1, TC5)"
```

---

### Task 3: Implement selector validation

**Files:**
- Create: `x/knowledge/keeper/tok_selector.go`

Validation is the gate for TC5: refusals are limited to syntax errors, snapshot-out-of-range, rate-limit. This module owns the syntax-error path.

- [ ] **Step 1: Write the failing test** (`x/knowledge/keeper/tok_bundle_test.go`)

```go
package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func TestValidateToKSelector_RejectsEmptyVariant(t *testing.T) {
	err := keeper.ValidateToKSelector(&types.ToKSelector{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "selector variant required")
}

func TestValidateToKSelector_RootedSubtree_RequiresRootFactId(t *testing.T) {
	sel := &types.ToKSelector{Variant: &types.ToKSelector_RootedSubtree{
		RootedSubtree: &types.RootedSubtreeSelector{},
	}}
	err := keeper.ValidateToKSelector(sel)
	require.Error(t, err)
	require.Contains(t, err.Error(), "root_fact_id")
}

func TestValidateToKSelector_RootedSubtree_CapsMaxDepth(t *testing.T) {
	sel := &types.ToKSelector{Variant: &types.ToKSelector_RootedSubtree{
		RootedSubtree: &types.RootedSubtreeSelector{
			RootFactId: "fact-1",
			MaxDepth:   100, // > cap 32
		},
	}}
	capped, err := keeper.ValidateAndCapToKSelector(sel)
	require.NoError(t, err)
	require.Equal(t, uint32(32), capped.GetRootedSubtree().MaxDepth)
}

func TestValidateToKSelector_Frontier_RequiresDomain(t *testing.T) {
	sel := &types.ToKSelector{Variant: &types.ToKSelector_Frontier{
		Frontier: &types.FrontierSelector{},
	}}
	err := keeper.ValidateToKSelector(sel)
	require.Error(t, err)
	require.Contains(t, err.Error(), "domain")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./x/knowledge/keeper/ -run TestValidateToKSelector -v`
Expected: FAIL — `ValidateToKSelector` undefined.

- [ ] **Step 3: Implement validation**

```go
// Package keeper, file: tok_selector.go
package keeper

import (
	"fmt"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// ToK chain-side caps. Adjustable via params in a future wave.
const (
	ToKMaxDepthCap     uint32 = 32
	ToKMaxPathsCap     uint32 = 256
	ToKFrontierLimit   uint32 = 1024
	ToKFrontierCap     uint32 = 8192
)

// ValidateToKSelector returns an error iff the selector is malformed.
// TC5 (extraction is open) requires that the only refusal classes are
// syntax error, snapshot-out-of-range, and rate-limit. This is the
// syntax-error gate.
func ValidateToKSelector(s *types.ToKSelector) error {
	if s == nil || s.Variant == nil {
		return fmt.Errorf("selector variant required (TC5: extraction is open — but selector must be well-formed)")
	}
	switch v := s.Variant.(type) {
	case *types.ToKSelector_RootedSubtree:
		if v.RootedSubtree == nil || v.RootedSubtree.RootFactId == "" {
			return fmt.Errorf("rooted_subtree.root_fact_id required")
		}
	case *types.ToKSelector_AncestorCone:
		if v.AncestorCone == nil || v.AncestorCone.LeafFactId == "" {
			return fmt.Errorf("ancestor_cone.leaf_fact_id required")
		}
	case *types.ToKSelector_Frontier:
		if v.Frontier == nil || v.Frontier.Domain == "" {
			return fmt.Errorf("frontier.domain required")
		}
	default:
		return fmt.Errorf("selector variant not recognised by this chain version")
	}
	return nil
}

// ValidateAndCapToKSelector validates and applies chain-side caps.
// Returns the capped selector. Caller should pass the capped value
// downstream so caps are applied uniformly.
func ValidateAndCapToKSelector(s *types.ToKSelector) (*types.ToKSelector, error) {
	if err := ValidateToKSelector(s); err != nil {
		return nil, err
	}
	out := &types.ToKSelector{Variant: s.Variant}
	switch v := out.Variant.(type) {
	case *types.ToKSelector_RootedSubtree:
		if v.RootedSubtree.MaxDepth == 0 || v.RootedSubtree.MaxDepth > ToKMaxDepthCap {
			v.RootedSubtree.MaxDepth = ToKMaxDepthCap
		}
	case *types.ToKSelector_AncestorCone:
		if v.AncestorCone.MaxDepth == 0 || v.AncestorCone.MaxDepth > ToKMaxDepthCap {
			v.AncestorCone.MaxDepth = ToKMaxDepthCap
		}
		if v.AncestorCone.MaxPaths == 0 || v.AncestorCone.MaxPaths > ToKMaxPathsCap {
			v.AncestorCone.MaxPaths = ToKMaxPathsCap
		}
	case *types.ToKSelector_Frontier:
		if v.Frontier.Limit == 0 {
			v.Frontier.Limit = ToKFrontierLimit
		}
		if v.Frontier.Limit > ToKFrontierCap {
			v.Frontier.Limit = ToKFrontierCap
		}
	}
	return out, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./x/knowledge/keeper/ -run TestValidateToKSelector -v`
Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
git add x/knowledge/keeper/tok_selector.go x/knowledge/keeper/tok_bundle_test.go
git commit -m "feat(knowledge): ToK selector validation + chain-side caps (TC5)"
```

---

### Task 4: Implement RootedSubtree gathering

**Files:**
- Modify: `x/knowledge/keeper/tok_selector.go`

Reuse `walkDescendants` from `grpc_query.go` — that's the existing graph traversal. Wrap it to emit (sorted_node_ids, sorted_edges).

- [ ] **Step 1: Add the failing test** (append to `tok_bundle_test.go`)

```go
func TestGatherRootedSubtree_LinearChain(t *testing.T) {
	// Build: axiom ──SUPPORTS──> b ──SUPPORTS──> c
	k, ctx := setupKnowledgeWithFacts(t, []factSpec{
		{id: "axiom", domain: "physics"},
		{id: "b", domain: "physics", supports: []string{"axiom"}},
		{id: "c", domain: "physics", supports: []string{"b"}},
	})
	sel := &types.RootedSubtreeSelector{RootFactId: "axiom", MaxDepth: 5}
	nodeIDs, edges, err := k.GatherRootedSubtree(ctx, sel)
	require.NoError(t, err)
	require.Equal(t, []string{"axiom", "b", "c"}, nodeIDs) // sorted
	require.Len(t, edges, 2)
}
```

`setupKnowledgeWithFacts` is a helper to seed facts + relations. If it doesn't already exist in the test file, add this helper (place it at the bottom):

```go
type factSpec struct {
	id       string
	domain   string
	supports []string // predecessor IDs
}

func setupKnowledgeWithFacts(t *testing.T, specs []factSpec) (*keeper.Keeper, sdk.Context) {
	t.Helper()
	k, ctx, _, _ := setupKnowledgeTestFull(t)
	for _, s := range specs {
		k.SetFact(ctx, &types.Fact{Id: s.id, Domain: s.domain, Status: types.FactStatus_FACT_STATUS_VERIFIED})
	}
	for _, s := range specs {
		for _, parent := range s.supports {
			require.NoError(t, k.AddRelation(ctx, &types.FactRelation{
				FromFactId: s.id,
				ToFactId:   parent,
				Relation:   "SUPPORTS",
			}))
		}
	}
	return k, ctx
}
```

(The helper uses `setupKnowledgeTestFull` which already exists in the keeper test package.)

- [ ] **Step 2: Run test to fail**

Run: `go test ./x/knowledge/keeper/ -run TestGatherRootedSubtree -v`
Expected: FAIL — `GatherRootedSubtree` undefined.

- [ ] **Step 3: Implement**

Append to `x/knowledge/keeper/tok_selector.go`:

```go
import (
	"context"
	"sort"
	// existing imports...
)

// GatherRootedSubtree walks descendants from the given root up to max_depth
// and returns sorted node IDs + sorted edges. Re-uses the descendant walk
// already used by grpc_query.go.
func (k Keeper) GatherRootedSubtree(
	ctx context.Context,
	sel *types.RootedSubtreeSelector,
) (nodeIDs []string, edges []*types.ToKEdge, err error) {
	root, found := k.GetFact(ctx, sel.RootFactId)
	if !found {
		return nil, nil, fmt.Errorf("root fact %s not found", sel.RootFactId)
	}
	visited := map[string]bool{root.Id: true}
	edgeSet := map[string]*types.ToKEdge{}
	if err := k.gatherDescendantsRecursive(ctx, root.Id, 1, sel.MaxDepth, visited, edgeSet); err != nil {
		return nil, nil, err
	}
	for id := range visited {
		nodeIDs = append(nodeIDs, id)
	}
	sort.Strings(nodeIDs)
	for _, e := range edgeSet {
		edges = append(edges, e)
	}
	sortToKEdges(edges)
	return nodeIDs, edges, nil
}

func (k Keeper) gatherDescendantsRecursive(
	ctx context.Context,
	factID string,
	depth, maxDepth uint32,
	visited map[string]bool,
	edges map[string]*types.ToKEdge,
) error {
	if depth > maxDepth {
		return nil
	}
	incoming, err := k.GetIncomingRelations(ctx, factID)
	if err != nil {
		return err
	}
	for _, rel := range incoming {
		edgeKey := rel.FromFactId + "→" + rel.ToFactId + "|" + rel.Relation
		if _, ok := edges[edgeKey]; !ok {
			edges[edgeKey] = &types.ToKEdge{
				FromFactId: rel.FromFactId,
				ToFactId:   rel.ToFactId,
				Relation:   rel.Relation,
				Inference:  rel.Inference,
			}
		}
		if !visited[rel.FromFactId] {
			visited[rel.FromFactId] = true
			if err := k.gatherDescendantsRecursive(ctx, rel.FromFactId, depth+1, maxDepth, visited, edges); err != nil {
				return err
			}
		}
	}
	return nil
}

func sortToKEdges(edges []*types.ToKEdge) {
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].FromFactId != edges[j].FromFactId {
			return edges[i].FromFactId < edges[j].FromFactId
		}
		if edges[i].ToFactId != edges[j].ToFactId {
			return edges[i].ToFactId < edges[j].ToFactId
		}
		return edges[i].Relation < edges[j].Relation
	})
}
```

- [ ] **Step 4: Run test to pass**

Run: `go test ./x/knowledge/keeper/ -run TestGatherRootedSubtree -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add x/knowledge/keeper/tok_selector.go x/knowledge/keeper/tok_bundle_test.go
git commit -m "feat(knowledge): GatherRootedSubtree — descendant walk for ToK (TC3)"
```

---

### Task 5: Implement AncestorCone gathering

**Files:**
- Modify: `x/knowledge/keeper/tok_selector.go`
- Modify: `x/knowledge/keeper/tok_bundle_test.go`

- [ ] **Step 1: Failing test**

```go
func TestGatherAncestorCone_LinearChain(t *testing.T) {
	k, ctx := setupKnowledgeWithFacts(t, []factSpec{
		{id: "axiom", domain: "physics"},
		{id: "b", domain: "physics", supports: []string{"axiom"}},
		{id: "c", domain: "physics", supports: []string{"b"}},
	})
	sel := &types.AncestorConeSelector{LeafFactId: "c", MaxDepth: 5, MaxPaths: 10}
	nodeIDs, edges, err := k.GatherAncestorCone(ctx, sel)
	require.NoError(t, err)
	require.Equal(t, []string{"axiom", "b", "c"}, nodeIDs)
	require.Len(t, edges, 2)
}
```

- [ ] **Step 2: Run, verify FAIL**

Run: `go test ./x/knowledge/keeper/ -run TestGatherAncestorCone -v` → FAIL.

- [ ] **Step 3: Implement**

Append to `tok_selector.go`:

```go
// GatherAncestorCone walks ancestors from the leaf up to max_depth, capped
// at max_paths distinct paths. Mirror of GatherRootedSubtree but follows
// outgoing relations instead of incoming.
func (k Keeper) GatherAncestorCone(
	ctx context.Context,
	sel *types.AncestorConeSelector,
) (nodeIDs []string, edges []*types.ToKEdge, err error) {
	leaf, found := k.GetFact(ctx, sel.LeafFactId)
	if !found {
		return nil, nil, fmt.Errorf("leaf fact %s not found", sel.LeafFactId)
	}
	visited := map[string]bool{leaf.Id: true}
	edgeSet := map[string]*types.ToKEdge{}
	pathCount := uint32(0)
	if err := k.gatherAncestorsRecursive(ctx, leaf.Id, 1, sel.MaxDepth, sel.MaxPaths, &pathCount, visited, edgeSet); err != nil {
		return nil, nil, err
	}
	for id := range visited {
		nodeIDs = append(nodeIDs, id)
	}
	sort.Strings(nodeIDs)
	for _, e := range edgeSet {
		edges = append(edges, e)
	}
	sortToKEdges(edges)
	return nodeIDs, edges, nil
}

func (k Keeper) gatherAncestorsRecursive(
	ctx context.Context,
	factID string,
	depth, maxDepth, maxPaths uint32,
	pathCount *uint32,
	visited map[string]bool,
	edges map[string]*types.ToKEdge,
) error {
	if depth > maxDepth || *pathCount >= maxPaths {
		return nil
	}
	outgoing, err := k.GetOutgoingRelations(ctx, factID)
	if err != nil {
		return err
	}
	for _, rel := range outgoing {
		*pathCount++
		if *pathCount > maxPaths {
			return nil
		}
		edgeKey := rel.FromFactId + "→" + rel.ToFactId + "|" + rel.Relation
		if _, ok := edges[edgeKey]; !ok {
			edges[edgeKey] = &types.ToKEdge{
				FromFactId: rel.FromFactId,
				ToFactId:   rel.ToFactId,
				Relation:   rel.Relation,
				Inference:  rel.Inference,
			}
		}
		if !visited[rel.ToFactId] {
			visited[rel.ToFactId] = true
			if err := k.gatherAncestorsRecursive(ctx, rel.ToFactId, depth+1, maxDepth, maxPaths, pathCount, visited, edges); err != nil {
				return err
			}
		}
	}
	return nil
}
```

- [ ] **Step 4: Pass**

Run: `go test ./x/knowledge/keeper/ -run TestGatherAncestorCone -v` → PASS.

- [ ] **Step 5: Commit**

```bash
git commit -am "feat(knowledge): GatherAncestorCone — ancestor walk for ToK (TC3)"
```

---

### Task 6: Implement Frontier gathering

**Files:**
- Modify: `x/knowledge/keeper/tok_selector.go`
- Modify: `x/knowledge/keeper/tok_bundle_test.go`

- [ ] **Step 1: Failing test**

```go
func TestGatherFrontier_DomainScoped(t *testing.T) {
	k, ctx := setupKnowledgeWithFacts(t, []factSpec{
		{id: "old1", domain: "physics"},
		{id: "old2", domain: "physics"},
	})
	// Mark old1, old2 as accepted at block 100.
	k.SetFactAcceptedBlock(ctx, "old1", 100)
	k.SetFactAcceptedBlock(ctx, "old2", 100)
	// Add a recent one.
	k.SetFact(ctx, &types.Fact{Id: "new1", Domain: "physics", Status: types.FactStatus_FACT_STATUS_VERIFIED})
	k.SetFactAcceptedBlock(ctx, "new1", 200)

	sel := &types.FrontierSelector{Domain: "physics", SinceBlock: 150, Limit: 100}
	nodeIDs, _, err := k.GatherFrontier(ctx, sel)
	require.NoError(t, err)
	require.Equal(t, []string{"new1"}, nodeIDs)
}
```

(`SetFactAcceptedBlock` may need a thin keeper helper — if it doesn't exist, add it. Frontier needs an "accepted at block" reverse index; if absent, add a `GetFactsByDomainSinceBlock` helper that iterates the domain index and filters by `Fact.AcceptedAtBlock`.)

- [ ] **Step 2: Run, FAIL**

- [ ] **Step 3: Implement**

```go
// GatherFrontier returns up to `limit` facts in `domain` accepted at or
// after `since_block`. Edges are the relations among the returned set
// (no traversal beyond). Useful for incremental training updates.
func (k Keeper) GatherFrontier(
	ctx context.Context,
	sel *types.FrontierSelector,
) (nodeIDs []string, edges []*types.ToKEdge, err error) {
	facts, err := k.GetFactsByDomainSinceBlock(ctx, sel.Domain, sel.SinceBlock, sel.Limit)
	if err != nil {
		return nil, nil, err
	}
	included := map[string]bool{}
	for _, f := range facts {
		nodeIDs = append(nodeIDs, f.Id)
		included[f.Id] = true
	}
	sort.Strings(nodeIDs)
	// Include only edges where BOTH endpoints are in the bundle.
	edgeSet := map[string]*types.ToKEdge{}
	for _, f := range facts {
		outgoing, err := k.GetOutgoingRelations(ctx, f.Id)
		if err != nil {
			return nil, nil, err
		}
		for _, rel := range outgoing {
			if !included[rel.ToFactId] {
				continue
			}
			edgeKey := rel.FromFactId + "→" + rel.ToFactId + "|" + rel.Relation
			edgeSet[edgeKey] = &types.ToKEdge{
				FromFactId: rel.FromFactId,
				ToFactId:   rel.ToFactId,
				Relation:   rel.Relation,
				Inference:  rel.Inference,
			}
		}
	}
	for _, e := range edgeSet {
		edges = append(edges, e)
	}
	sortToKEdges(edges)
	return nodeIDs, edges, nil
}
```

If `GetFactsByDomainSinceBlock` doesn't exist, add it to `keeper.go`:

```go
func (k Keeper) GetFactsByDomainSinceBlock(ctx context.Context, domain string, sinceBlock uint64, limit uint32) ([]types.Fact, error) {
	var out []types.Fact
	count := uint32(0)
	k.IterateFactsByDomain(ctx, domain, func(factID string) bool {
		if limit > 0 && count >= limit {
			return true
		}
		f, ok := k.GetFact(ctx, factID)
		if !ok {
			return false
		}
		if f.AcceptedAtBlock >= sinceBlock {
			out = append(out, f)
			count++
		}
		return false
	})
	return out, nil
}
```

(If `Fact.AcceptedAtBlock` doesn't exist as a field, this surfaces a proto change to make first; that's a Plan-1 prereq landing in `types.proto`. Check `proto/zerone/knowledge/v1/types.proto` before implementing.)

- [ ] **Step 4: Pass**

Run: `go test ./x/knowledge/keeper/ -run TestGatherFrontier -v` → PASS.

- [ ] **Step 5: Commit**

```bash
git commit -am "feat(knowledge): GatherFrontier — domain-scoped recent facts (TC3)"
```

---

### Task 7: ComputeToKSnapshotRoot — Merkle commitment

**Files:**
- Create: `x/knowledge/keeper/tok_bundle.go`
- Modify: `x/knowledge/keeper/tok_bundle_test.go`

The snapshot root is what TC2 binds. Domain-tagged Merkle parallels `ComputeManifestMerkleRoot` in `training_manifest.go`.

- [ ] **Step 1: Failing test**

```go
func TestComputeToKSnapshotRoot_Deterministic(t *testing.T) {
	nodeIDs := []string{"a", "b", "c"}
	edges := []*types.ToKEdge{
		{FromFactId: "b", ToFactId: "a", Relation: "SUPPORTS"},
		{FromFactId: "c", ToFactId: "b", Relation: "SUPPORTS"},
	}
	r1 := keeper.ComputeToKSnapshotRoot(nodeIDs, edges)
	r2 := keeper.ComputeToKSnapshotRoot(nodeIDs, edges)
	require.Equal(t, r1, r2)
	require.Len(t, r1, 32) // 32-byte sha256
}

func TestComputeToKSnapshotRoot_DomainSeparated(t *testing.T) {
	// Same IDs, but one set is "TOK_NODES" tagged and another is "TOK_EDGES" tagged.
	// Roots must differ — domain tags prevent set-swap collisions.
	r := keeper.ComputeToKSnapshotRoot([]string{"a", "b"}, nil)
	r2 := keeper.ComputeToKSnapshotRoot(nil, []*types.ToKEdge{
		{FromFactId: "a", ToFactId: "b", Relation: "SUPPORTS"},
	})
	require.NotEqual(t, r, r2)
}
```

- [ ] **Step 2: FAIL**

- [ ] **Step 3: Implement**

```go
// Package keeper, file: tok_bundle.go
package keeper

import (
	"crypto/sha256"
	"sort"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

const (
	tokDomainNodes = "TOK_NODES"
	tokDomainEdges = "TOK_EDGES"
)

// ComputeToKSnapshotRoot returns a 32-byte Merkle commitment over the
// (sorted node IDs, sorted edges) pair, domain-tagged to prevent
// set-swap collisions. Mirrors the ComputeManifestMerkleRoot pattern
// in training_manifest.go (Wave 7).
//
// The root is computed from IDs alone, never from payloads. A trainer
// who has the IDs can re-derive the root without trusting the RPC's
// serialisation. TC2: every view is graph-pinned.
func ComputeToKSnapshotRoot(nodeIDs []string, edges []*types.ToKEdge) []byte {
	// Defensive: caller already passed sorted, but ensure idempotency.
	sortedNodes := append([]string{}, nodeIDs...)
	sort.Strings(sortedNodes)
	sortedEdges := append([]*types.ToKEdge{}, edges...)
	sortToKEdges(sortedEdges)

	nodesH := domainHash(tokDomainNodes, func(h *sha256.Hash) {
		for _, id := range sortedNodes {
			lengthPrefixedWrite(h, []byte(id))
		}
	})
	edgesH := domainHash(tokDomainEdges, func(h *sha256.Hash) {
		for _, e := range sortedEdges {
			canon := e.FromFactId + "|" + e.ToFactId + "|" + e.Relation + "|" + e.Inference
			lengthPrefixedWrite(h, []byte(canon))
		}
	})

	// Root: sha256("TOK_ROOT" || nodesH || edgesH).
	final := sha256.New()
	lengthPrefixedWrite(final, []byte("TOK_ROOT"))
	final.Write(nodesH)
	final.Write(edgesH)
	return final.Sum(nil)
}

func domainHash(domain string, write func(*sha256.Hash)) []byte {
	h := sha256.New()
	lengthPrefixedWrite(h, []byte(domain))
	write(h)
	return h.Sum(nil)
}

// lengthPrefixedWrite is the same helper used by training_manifest.go;
// if it lives there as a private helper, refactor to a shared spot
// (e.g., x/knowledge/keeper/merkle_util.go) rather than duplicating.
func lengthPrefixedWrite(h hash.Hash, data []byte) {
	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(data)))
	h.Write(lenBuf[:])
	h.Write(data)
}
```

(Note: `lengthPrefixedWrite` already exists in `training_manifest.go` as private. As a small refactor, move it to `merkle_util.go` first or alias inside `tok_bundle.go` — pick the lighter touch and keep it in scope.)

- [ ] **Step 4: PASS**

- [ ] **Step 5: Commit**

```bash
git commit -am "feat(knowledge): ComputeToKSnapshotRoot — domain-tagged Merkle (TC2)"
```

---

### Task 8: AssembleToKBundle — orchestration

**Files:**
- Modify: `x/knowledge/keeper/tok_bundle.go`
- Modify: `x/knowledge/keeper/tok_bundle_test.go`

- [ ] **Step 1: Failing test**

```go
func TestAssembleToKBundle_RootedSubtree(t *testing.T) {
	k, ctx := setupKnowledgeWithFacts(t, []factSpec{
		{id: "axiom", domain: "physics"},
		{id: "b", domain: "physics", supports: []string{"axiom"}},
	})
	sel := &types.ToKSelector{Variant: &types.ToKSelector_RootedSubtree{
		RootedSubtree: &types.RootedSubtreeSelector{RootFactId: "axiom", MaxDepth: 5},
	}}
	bundle, err := k.AssembleToKBundle(ctx, sel, 0 /* current block */)
	require.NoError(t, err)
	require.NotEmpty(t, bundle.SnapshotRoot)
	require.Equal(t, []string{"axiom", "b"}, bundle.IncludedNodeIds)
	require.Len(t, bundle.IncludedEdges, 1)
	require.Len(t, bundle.Nodes, 2)
	// Re-derivability: re-compute root from IDs — must match.
	rederived := keeper.ComputeToKSnapshotRoot(bundle.IncludedNodeIds, bundle.IncludedEdges)
	require.Equal(t, bundle.SnapshotRoot, rederived)
}
```

- [ ] **Step 2: FAIL**

- [ ] **Step 3: Implement**

```go
// AssembleToKBundle is the headline ToK extraction primitive. It validates
// the selector, applies caps, gathers IDs, computes the snapshot root,
// materialises payloads, and returns a complete bundle. TC1 (graph is
// the headline) is bound by exposing this through gRPC + CLI as the
// trainer-facing default.
func (k Keeper) AssembleToKBundle(
	ctx context.Context,
	sel *types.ToKSelector,
	atBlockHeight uint64,
) (*types.ToKBundle, error) {
	capped, err := ValidateAndCapToKSelector(sel)
	if err != nil {
		return nil, err
	}
	nodeIDs, edges, err := k.SelectToKIds(ctx, capped)
	if err != nil {
		return nil, err
	}
	root := ComputeToKSnapshotRoot(nodeIDs, edges)

	// Materialise node payloads.
	var nodes []*types.Fact
	for _, id := range nodeIDs {
		f, ok := k.GetFact(ctx, id)
		if !ok {
			return nil, fmt.Errorf("inconsistent state: selected fact %s not found", id)
		}
		nodes = append(nodes, &f)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if atBlockHeight == 0 {
		atBlockHeight = uint64(sdkCtx.BlockHeight())
	}

	return &types.ToKBundle{
		SnapshotBlock:        atBlockHeight,
		SnapshotRoot:         root,
		IncludedNodeIds:      nodeIDs,
		IncludedEdges:        edges,
		Nodes:                nodes,
		SerialisationFormat:  "jsonl_adjacency_v1",
		// SerialisedPayload omitted — Task 9 fills this when format requested.
		Provenance: &types.ToKBundleProvenance{
			ChainId:                       sdkCtx.ChainID(),
			TraceSchemaVersion:            k.GetTraceSchemaVersion(ctx),
			CanonicalSerialisationVersion: "v1",
			TokenizerVersion:              k.GetTokenizerVersion(ctx),
			SelectorUsed:                  capped,
			CapMaxDepth:                   ToKMaxDepthCap,
			CapMaxPaths:                   ToKMaxPathsCap,
			CapLimit:                      ToKFrontierCap,
		},
	}, nil
}

// SelectToKIds dispatches on the validated selector variant.
func (k Keeper) SelectToKIds(
	ctx context.Context,
	sel *types.ToKSelector,
) (nodeIDs []string, edges []*types.ToKEdge, err error) {
	switch v := sel.Variant.(type) {
	case *types.ToKSelector_RootedSubtree:
		return k.GatherRootedSubtree(ctx, v.RootedSubtree)
	case *types.ToKSelector_AncestorCone:
		return k.GatherAncestorCone(ctx, v.AncestorCone)
	case *types.ToKSelector_Frontier:
		return k.GatherFrontier(ctx, v.Frontier)
	default:
		return nil, nil, fmt.Errorf("selector variant not handled in this plan")
	}
}
```

- [ ] **Step 4: PASS**

- [ ] **Step 5: Commit**

```bash
git commit -am "feat(knowledge): AssembleToKBundle — Select→Root→Materialise (TC1, TC2, TC3)"
```

---

### Task 9: JSONL adjacency-list serialisation

**Files:**
- Create: `x/knowledge/keeper/tok_serialise.go`
- Modify: `x/knowledge/keeper/tok_bundle_test.go`

- [ ] **Step 1: Failing test**

```go
func TestSerialiseToK_JSONL_RoundTrip(t *testing.T) {
	bundle := &types.ToKBundle{
		IncludedNodeIds: []string{"a", "b"},
		IncludedEdges:   []*types.ToKEdge{{FromFactId: "b", ToFactId: "a", Relation: "SUPPORTS"}},
		Nodes:           []*types.Fact{{Id: "a", Domain: "physics"}, {Id: "b", Domain: "physics"}},
	}
	payload, err := keeper.SerialiseToK_JSONL(bundle)
	require.NoError(t, err)

	// Each line is one JSON object: nodes then edges.
	lines := bytes.Split(payload, []byte("\n"))
	// Drop trailing empty line if any.
	if len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}
	require.Len(t, lines, 3) // 2 nodes + 1 edge
}
```

- [ ] **Step 2: FAIL**

- [ ] **Step 3: Implement**

```go
// Package keeper, file: tok_serialise.go
package keeper

import (
	"bytes"
	"encoding/json"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// SerialiseToK_JSONL emits one JSON line per node (kind:"node") and
// per edge (kind:"edge") in the order they appear in the bundle.
// Adjacency-list form: trainers can stream-process without loading
// the full graph into memory.
//
// Format identifier: "jsonl_adjacency_v1". Versioned so future formats
// (e.g., protobuf graph, GraphML) can coexist.
func SerialiseToK_JSONL(b *types.ToKBundle) ([]byte, error) {
	var buf bytes.Buffer
	for _, n := range b.Nodes {
		row := map[string]any{
			"kind":   "node",
			"id":     n.Id,
			"fact":   n,
		}
		line, err := json.Marshal(row)
		if err != nil {
			return nil, err
		}
		buf.Write(line)
		buf.WriteByte('\n')
	}
	for _, e := range b.IncludedEdges {
		row := map[string]any{
			"kind":      "edge",
			"from":      e.FromFactId,
			"to":        e.ToFactId,
			"relation":  e.Relation,
			"inference": e.Inference,
		}
		line, err := json.Marshal(row)
		if err != nil {
			return nil, err
		}
		buf.Write(line)
		buf.WriteByte('\n')
	}
	return buf.Bytes(), nil
}
```

- [ ] **Step 4: PASS**

- [ ] **Step 5: Wire into AssembleToKBundle**

In `tok_bundle.go`, replace the comment `// SerialisedPayload omitted` with:

```go
	// Embed serialised payload (default format).
	payload, err := SerialiseToK_JSONL(&types.ToKBundle{
		IncludedNodeIds: nodeIDs, IncludedEdges: edges, Nodes: nodes,
	})
	if err != nil {
		return nil, err
	}
	bundle.SerialisedPayload = payload
```

(Adjust to the exact local variable names in your implementation.)

- [ ] **Step 6: Commit**

```bash
git commit -am "feat(knowledge): JSONL adjacency-list serialisation for ToK bundles (TC3)"
```

---

### Task 10: BundleToK gRPC handler

**Files:**
- Modify: `x/knowledge/keeper/grpc_query.go`

- [ ] **Step 1: Failing test in keeper test pkg**

```go
func TestQueryBundleToK_Happy(t *testing.T) {
	k, ctx := setupKnowledgeWithFacts(t, []factSpec{
		{id: "axiom", domain: "physics"},
		{id: "b", domain: "physics", supports: []string{"axiom"}},
	})
	q := keeper.NewQueryServerImpl(k)
	resp, err := q.BundleToK(ctx, &types.QueryBundleToKRequest{
		Selector: &types.ToKSelector{Variant: &types.ToKSelector_RootedSubtree{
			RootedSubtree: &types.RootedSubtreeSelector{RootFactId: "axiom", MaxDepth: 5},
		}},
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Bundle)
	require.NotEmpty(t, resp.Bundle.SnapshotRoot)
}

func TestQueryBundleToK_RejectsInvalidSelector(t *testing.T) {
	k, ctx, _, _ := setupKnowledgeTestFull(t)
	q := keeper.NewQueryServerImpl(k)
	_, err := q.BundleToK(ctx, &types.QueryBundleToKRequest{
		Selector: &types.ToKSelector{}, // empty variant
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "selector variant required")
}
```

- [ ] **Step 2: FAIL** (handler undefined)

- [ ] **Step 3: Implement** — append to `grpc_query.go`:

```go
// BundleToK is the headline trainer-facing endpoint. TC1: the graph is
// the substrate; this is where trainers ask for it. TC5 (extraction is
// open) is bound here: refusals are limited to syntax errors,
// snapshot-out-of-range, and rate-limit.
func (q *queryServer) BundleToK(ctx context.Context, req *types.QueryBundleToKRequest) (*types.QueryBundleToKResponse, error) {
	if req == nil || req.Selector == nil {
		return nil, status.Error(codes.InvalidArgument, "selector required")
	}
	bundle, err := q.keeper.AssembleToKBundle(ctx, req.Selector, req.AtBlockHeight)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	return &types.QueryBundleToKResponse{Bundle: bundle}, nil
}
```

- [ ] **Step 4: PASS**

- [ ] **Step 5: Commit**

```bash
git commit -am "feat(knowledge): BundleToK gRPC handler — headline trainer endpoint (TC1, TC5)"
```

---

### Task 11: Extend RouteBCapabilities with tok_capabilities

**Files:**
- Modify: `x/knowledge/keeper/grpc_query.go` — find the existing `RouteBCapabilities` handler.

- [ ] **Step 1: Failing test**

```go
func TestRouteBCapabilities_AdvertisesToK(t *testing.T) {
	k, ctx, _, _ := setupKnowledgeTestFull(t)
	q := keeper.NewQueryServerImpl(k)
	resp, err := q.RouteBCapabilities(ctx, &types.QueryRouteBCapabilitiesRequest{})
	require.NoError(t, err)
	require.NotNil(t, resp.TokCapabilities, "TC1: tok_capabilities must be advertised")
	require.Contains(t, resp.TokCapabilities.SupportedSelectors, "rooted_subtree")
	require.Contains(t, resp.TokCapabilities.SupportedSelectors, "ancestor_cone")
	require.Contains(t, resp.TokCapabilities.SupportedSelectors, "frontier")
	require.NotEmpty(t, resp.TokCapabilities.TokDoctrineVersion)
}
```

- [ ] **Step 2: FAIL**

- [ ] **Step 3: Implement** — extend the existing `RouteBCapabilities` response builder (find it in `grpc_query.go`) by populating `TokCapabilities`:

```go
	resp.TokCapabilities = &types.ToKCapabilities{
		SupportedSelectors:      []string{"rooted_subtree", "ancestor_cone", "frontier"},
		MaxDepthCap:             keeper.ToKMaxDepthCap,
		MaxPathsCap:             keeper.ToKMaxPathsCap,
		FrontierLimitCap:        keeper.ToKFrontierCap,
		SupportedSerialisations: []string{"jsonl_adjacency_v1"},
		TokDoctrineVersion:      "TC1-TC5 (2026-05-09 inception)",
	}
```

- [ ] **Step 4: PASS**

- [ ] **Step 5: Commit**

```bash
git commit -am "feat(knowledge): RouteBCapabilities advertises tok_capabilities (TC1)"
```

---

### Task 12: CLI command `bundle-tok`

**Files:**
- Create: `x/knowledge/client/cli/query_tok.go`
- Modify: `x/knowledge/client/cli/query.go` — register the new command.

- [ ] **Step 1: Implement** (no separate test gate; CLI is exercised via integration; verify by running)

```go
package cli

import (
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// CmdBundleToK supports three sub-forms (one per selector variant for
// CLI ergonomics; the gRPC accepts the union directly).
func CmdBundleToK() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bundle-tok",
		Short: "Extract a ToK subgraph (TC1: the graph is the headline)",
	}
	cmd.AddCommand(cmdBundleToKRootedSubtree())
	cmd.AddCommand(cmdBundleToKAncestorCone())
	cmd.AddCommand(cmdBundleToKFrontier())
	return cmd
}

func cmdBundleToKRootedSubtree() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rooted-subtree [root-fact-id]",
		Short: "Bundle the descendants of a root fact",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil { return err }
			depth, _ := cmd.Flags().GetUint32("max-depth")
			req := &types.QueryBundleToKRequest{
				Selector: &types.ToKSelector{Variant: &types.ToKSelector_RootedSubtree{
					RootedSubtree: &types.RootedSubtreeSelector{
						RootFactId: args[0], MaxDepth: depth,
					},
				}},
			}
			res, err := types.NewQueryClient(clientCtx).BundleToK(cmd.Context(), req)
			if err != nil { return err }
			return clientCtx.PrintProto(res)
		},
	}
	cmd.Flags().Uint32("max-depth", 5, "max descendant depth (capped at 32)")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func cmdBundleToKAncestorCone() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ancestor-cone [leaf-fact-id]",
		Short: "Bundle the ancestor cone from a leaf",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, _ := client.GetClientQueryContext(cmd)
			depth, _ := cmd.Flags().GetUint32("max-depth")
			paths, _ := cmd.Flags().GetUint32("max-paths")
			req := &types.QueryBundleToKRequest{
				Selector: &types.ToKSelector{Variant: &types.ToKSelector_AncestorCone{
					AncestorCone: &types.AncestorConeSelector{
						LeafFactId: args[0], MaxDepth: depth, MaxPaths: paths,
					},
				}},
			}
			res, err := types.NewQueryClient(clientCtx).BundleToK(cmd.Context(), req)
			if err != nil { return err }
			return clientCtx.PrintProto(res)
		},
	}
	cmd.Flags().Uint32("max-depth", 5, "max ancestor depth (capped at 32)")
	cmd.Flags().Uint32("max-paths", 10, "max paths (capped at 256)")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func cmdBundleToKFrontier() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "frontier [domain]",
		Short: "Bundle the latest facts in a domain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, _ := client.GetClientQueryContext(cmd)
			sinceStr, _ := cmd.Flags().GetString("since-block")
			limit, _ := cmd.Flags().GetUint32("limit")
			since, err := strconv.ParseUint(sinceStr, 10, 64)
			if err != nil { return fmt.Errorf("invalid --since-block: %w", err) }
			req := &types.QueryBundleToKRequest{
				Selector: &types.ToKSelector{Variant: &types.ToKSelector_Frontier{
					Frontier: &types.FrontierSelector{
						Domain: args[0], SinceBlock: since, Limit: limit,
					},
				}},
			}
			res, err := types.NewQueryClient(clientCtx).BundleToK(cmd.Context(), req)
			if err != nil { return err }
			return clientCtx.PrintProto(res)
		},
	}
	cmd.Flags().String("since-block", "0", "include facts accepted at/after this block")
	cmd.Flags().Uint32("limit", 1024, "max facts (capped at 8192)")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
```

In `query.go`, register inside the existing `GetQueryCmd` function: `cmd.AddCommand(CmdBundleToK())`.

- [ ] **Step 2: Verify build**

Run: `go build ./x/knowledge/client/cli/...`
Expected: clean.

- [ ] **Step 3: Smoke test against a localnet** (defer to integration; optional here)

- [ ] **Step 4: Commit**

```bash
git add x/knowledge/client/cli/query_tok.go x/knowledge/client/cli/query.go
git commit -m "feat(knowledge): bundle-tok CLI command (TC1: headline endpoint visible to operators)"
```

---

### Task 13: ToK events + voice layer

**Files:**
- Modify: `x/knowledge/keeper/events.go` (or wherever `EventTypeXxx` consts live — check the package).

- [ ] **Step 1: Implement event emission**

In whatever file holds existing `EventType` constants (e.g., `events.go` or `types/events.go`), add:

```go
const (
	EventTypeToKBundleExtracted   = "tok_bundle_extracted"
	EventTypeToKSnapshotRootPinned = "tok_snapshot_root_pinned"

	AttrToKCommitment    = "tok_commitment"   // value: "TC1", "TC2,TC5", etc.
	AttrToKSelectorKind  = "selector_kind"
	AttrToKBundleSize    = "node_count"
	AttrToKSnapshotRoot  = "snapshot_root"
	AttrToKSnapshotBlock = "snapshot_block"
)
```

In `tok_bundle.go`'s `AssembleToKBundle`, just before returning, emit:

```go
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		EventTypeToKBundleExtracted,
		sdk.NewAttribute(AttrToKCommitment, "TC1,TC5"),
		sdk.NewAttribute(AttrToKSelectorKind, selectorKind(capped)),
		sdk.NewAttribute(AttrToKBundleSize, fmt.Sprintf("%d", len(nodeIDs))),
		sdk.NewAttribute(AttrToKSnapshotBlock, fmt.Sprintf("%d", atBlockHeight)),
	))
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		EventTypeToKSnapshotRootPinned,
		sdk.NewAttribute(AttrToKCommitment, "TC2"),
		sdk.NewAttribute(AttrToKSnapshotRoot, hex.EncodeToString(root)),
	))
```

Add helper:

```go
func selectorKind(s *types.ToKSelector) string {
	switch s.Variant.(type) {
	case *types.ToKSelector_RootedSubtree: return "rooted_subtree"
	case *types.ToKSelector_AncestorCone:  return "ancestor_cone"
	case *types.ToKSelector_Frontier:      return "frontier"
	default: return "unknown"
	}
}
```

- [ ] **Step 2: Test** (verify event emission)

```go
func TestAssembleToKBundle_EmitsEvents(t *testing.T) {
	k, ctx := setupKnowledgeWithFacts(t, []factSpec{
		{id: "axiom", domain: "physics"},
	})
	sel := &types.ToKSelector{Variant: &types.ToKSelector_RootedSubtree{
		RootedSubtree: &types.RootedSubtreeSelector{RootFactId: "axiom", MaxDepth: 1},
	}}
	_, err := k.AssembleToKBundle(ctx, sel, 0)
	require.NoError(t, err)

	events := sdk.UnwrapSDKContext(ctx).EventManager().Events()
	var sawBundle, sawRoot bool
	for _, e := range events {
		if e.Type == keeper.EventTypeToKBundleExtracted {
			sawBundle = true
			for _, a := range e.Attributes {
				if a.Key == keeper.AttrToKCommitment {
					require.Contains(t, a.Value, "TC1")
				}
			}
		}
		if e.Type == keeper.EventTypeToKSnapshotRootPinned {
			sawRoot = true
		}
	}
	require.True(t, sawBundle, "TC1: tok_bundle_extracted must be emitted")
	require.True(t, sawRoot, "TC2: tok_snapshot_root_pinned must be emitted")
}
```

- [ ] **Step 3: PASS**

- [ ] **Step 4: Commit**

```bash
git commit -am "feat(knowledge): ToK voice layer — tok_bundle_extracted, tok_snapshot_root_pinned (TC1, TC2, TC5)"
```

---

### Task 14: Position layer — x/knowledge/doc.go declares TC1-TC5

**Files:**
- Modify: `x/knowledge/doc.go` (create if absent — check first with `ls x/knowledge/doc.go`).

- [ ] **Step 1: Read or create**

If file exists, append a new section. If not, create:

```go
// Package knowledge holds the chain's knowledge-and-truth substrate.
//
// docs/TRUTH_SEEKING.md commitments preserved here:
// - 1 (methodology over statement), 2 (is-ought wall), 3 (Popper),
//   4 (substrate stress-tests), 5 (probe demand), 6 (no unilateral
//   injection), 10 (forward-only audit), 12 (chain pays own audit),
//   13 (training corpus not for sale), 14 (reasoning traces first-class).
//
// docs/TOK_SUBSTRATE.md commitments preserved here:
// - TC1 (graph is the headline) — BundleToK + RouteBCapabilities
//   advertise the substrate first. See keeper/tok_bundle.go and
//   keeper/grpc_query.go BundleToK handler.
// - TC2 (every view is graph-pinned) — every bundle carries a 32-byte
//   snapshot_root computed via ComputeToKSnapshotRoot from sorted node
//   IDs + sorted edge IDs, domain-tagged TOK_NODES / TOK_EDGES.
// - TC3 (topology is signal) — bundles ship edges, depth, and (when
//   available) confidence-floor as first-class fields, not metadata.
//   See keeper/tok_serialise.go for the JSONL adjacency-list format.
// - TC5 (extraction is open) — ValidateAndCapToKSelector accepts any
//   well-formed selector and applies uniform caps. Refusals are limited
//   to syntax errors and snapshot-out-of-range; no curation gate exists.
//
// What would break these: see the corresponding "What would break it"
// sections in docs/TOK_SUBSTRATE.md.
//
// We speak through intentions.
package knowledge
```

- [ ] **Step 2: Verify build**

Run: `go build ./x/knowledge/...`
Expected: clean.

- [ ] **Step 3: Verify go doc surface**

Run: `go doc ./x/knowledge | head -40`
Expected: TC1-TC5 declarations visible.

- [ ] **Step 4: Commit**

```bash
git commit -am "docs(knowledge): position layer — TC1-TC5 declared in doc.go"
```

---

### Task 15: TC1 invariant — BundleToK is the headline endpoint

**Files:**
- Create: `tests/cross_stack/tok_substrate_invariants_test.go`

- [ ] **Step 1: Write the test**

```go
package cross_stack_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// TC1: the graph is the headline.
// Verified by: RouteBCapabilities advertising tok_capabilities, and
// BundleToK accepting and returning a well-formed bundle.
func TestToKSubstrate_TC1_GraphIsTheHeadline(t *testing.T) {
	h := NewTestHarness(t)

	// Capability advertisement.
	q := keeperImpl(h.KnowledgeKeeper)
	caps, err := q.RouteBCapabilities(h.Ctx, &knowledgetypes.QueryRouteBCapabilitiesRequest{})
	require.NoError(t, err)
	require.NotNil(t, caps.TokCapabilities, "TC1: tok_capabilities must be advertised")
	require.Contains(t, caps.TokCapabilities.SupportedSelectors, "rooted_subtree")

	// Headline endpoint roundtrip.
	seedFact(t, h, "physics", "axiom-tc1")
	resp, err := q.BundleToK(h.Ctx, &knowledgetypes.QueryBundleToKRequest{
		Selector: &knowledgetypes.ToKSelector{Variant: &knowledgetypes.ToKSelector_RootedSubtree{
			RootedSubtree: &knowledgetypes.RootedSubtreeSelector{RootFactId: "axiom-tc1", MaxDepth: 1},
		}},
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Bundle, "TC1: BundleToK is the headline; it must return a graph bundle")
	require.NotEmpty(t, resp.Bundle.SnapshotRoot)
}
```

(If `keeperImpl(...)` and `seedFact(...)` helpers don't already exist in the test pkg, add minimal versions or use the existing harness conventions — check `tests/cross_stack/harness_test.go` for fact-seeding helpers.)

- [ ] **Step 2: Run, expect PASS** (production code already in place)

Run: `go test ./tests/cross_stack/ -run TestToKSubstrate_TC1 -v`

- [ ] **Step 3: Commit**

```bash
git commit -am "test(cross_stack): TC1 invariant — graph is the headline"
```

---

### Task 16: TC2 invariant — every view is graph-pinned

- [ ] **Step 1: Test**

```go
// TC2: every view is graph-pinned.
// Verified by: bundle ships snapshot_root + snapshot_block, and the
// root is re-derivable from IDs alone (trust-minimised verification).
func TestToKSubstrate_TC2_EveryViewIsGraphPinned(t *testing.T) {
	h := NewTestHarness(t)
	seedFact(t, h, "physics", "axiom-tc2")
	q := keeperImpl(h.KnowledgeKeeper)
	resp, err := q.BundleToK(h.Ctx, &knowledgetypes.QueryBundleToKRequest{
		Selector: &knowledgetypes.ToKSelector{Variant: &knowledgetypes.ToKSelector_RootedSubtree{
			RootedSubtree: &knowledgetypes.RootedSubtreeSelector{RootFactId: "axiom-tc2", MaxDepth: 1},
		}},
	})
	require.NoError(t, err)
	require.Len(t, resp.Bundle.SnapshotRoot, 32, "TC2: snapshot root must be 32 bytes")
	require.Greater(t, resp.Bundle.SnapshotBlock, uint64(0), "TC2: snapshot_block must be set")
	// Re-derivability — trust-minimised verification.
	rederived := keeper.ComputeToKSnapshotRoot(resp.Bundle.IncludedNodeIds, resp.Bundle.IncludedEdges)
	require.Equal(t, resp.Bundle.SnapshotRoot, rederived, "TC2: root must be re-derivable from IDs")
}
```

- [ ] **Step 2: PASS, Commit**

```bash
git commit -am "test(cross_stack): TC2 invariant — graph-pinned views"
```

---

### Task 17: TC3 invariant — topology is signal

- [ ] **Step 1: Test**

```go
// TC3: topology is signal.
// Verified by: bundle ships edges (not just nodes), and depth/floor
// fields propagate through node payloads.
func TestToKSubstrate_TC3_TopologyIsSignal(t *testing.T) {
	h := NewTestHarness(t)
	seedFact(t, h, "physics", "axiom-tc3")
	seedFactWithSupport(t, h, "physics", "leaf-tc3", "axiom-tc3")
	q := keeperImpl(h.KnowledgeKeeper)
	resp, err := q.BundleToK(h.Ctx, &knowledgetypes.QueryBundleToKRequest{
		Selector: &knowledgetypes.ToKSelector{Variant: &knowledgetypes.ToKSelector_RootedSubtree{
			RootedSubtree: &knowledgetypes.RootedSubtreeSelector{RootFactId: "axiom-tc3", MaxDepth: 5},
		}},
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Bundle.IncludedEdges, "TC3: edges are first-class, not metadata")
	require.Equal(t, "SUPPORTS", resp.Bundle.IncludedEdges[0].Relation)
	require.NotEmpty(t, resp.Bundle.SerialisedPayload, "TC3: native serialisation ships topology")
}
```

(`seedFactWithSupport` is a helper — add to harness if missing.)

- [ ] **Step 2: PASS, Commit**

```bash
git commit -am "test(cross_stack): TC3 invariant — topology is signal"
```

---

### Task 18: TC5 invariant — extraction is open

- [ ] **Step 1: Test**

```go
// TC5: extraction is open.
// Verified by: any well-formed selector accepted; refusals limited to
// syntax errors and out-of-range; no allowlist consultation.
func TestToKSubstrate_TC5_ExtractionIsOpen(t *testing.T) {
	h := NewTestHarness(t)
	for _, dom := range []string{"physics", "biology", "ethics", "obscure_unfamiliar_domain"} {
		seedFact(t, h, dom, "seed-"+dom)
	}
	q := keeperImpl(h.KnowledgeKeeper)
	// All four domains must succeed — no curation gate.
	for _, dom := range []string{"physics", "biology", "ethics", "obscure_unfamiliar_domain"} {
		resp, err := q.BundleToK(h.Ctx, &knowledgetypes.QueryBundleToKRequest{
			Selector: &knowledgetypes.ToKSelector{Variant: &knowledgetypes.ToKSelector_Frontier{
				Frontier: &knowledgetypes.FrontierSelector{Domain: dom, Limit: 10},
			}},
		})
		require.NoError(t, err, "TC5: domain %s must be open for extraction", dom)
		require.NotNil(t, resp.Bundle)
	}
	// Syntactically invalid selector must be the only refusal class.
	_, err := q.BundleToK(h.Ctx, &knowledgetypes.QueryBundleToKRequest{
		Selector: &knowledgetypes.ToKSelector{},
	})
	require.Error(t, err, "TC5: syntax errors are the only doctrinal refusal")
}
```

- [ ] **Step 2: PASS, Commit**

```bash
git commit -am "test(cross_stack): TC5 invariant — extraction is open"
```

---

### Task 19: Update docs/EVENTS.md

**Files:**
- Modify: `docs/EVENTS.md`

- [ ] **Step 1: Append a new section**

Append at the bottom (or in alphabetical order if the doc is sorted):

```markdown
## ToK substrate events (docs/TOK_SUBSTRATE.md)

### `tok_bundle_extracted`
A trainer extracted a ToK bundle via `BundleToK`. TC1 (graph is the headline) and TC5 (extraction is open) are both bound by this event firing on every successful bundle.

| Attribute | Description |
|---|---|
| `tok_commitment` | `"TC1,TC5"` — the doctrine commitments this event preserves |
| `selector_kind` | `"rooted_subtree" | "ancestor_cone" | "frontier"` |
| `node_count` | Number of nodes in the bundle |
| `snapshot_block` | Block height the bundle is pinned to |

### `tok_snapshot_root_pinned`
Every bundle pins to a 32-byte snapshot root (TC2: every view is graph-pinned). The root commits to sorted node IDs + sorted edge IDs domain-tagged separately.

| Attribute | Description |
|---|---|
| `tok_commitment` | `"TC2"` |
| `snapshot_root` | Hex-encoded 32-byte Merkle root |
```

- [ ] **Step 2: Commit**

```bash
git commit -am "docs(events): document ToK substrate events with tok_commitment attributes"
```

---

### Task 20: Final integration sweep

- [ ] **Step 1: Run all knowledge keeper tests**

Run: `go test ./x/knowledge/... -timeout 300s`
Expected: PASS.

- [ ] **Step 2: Run cross-stack tests**

Run: `go test ./tests/cross_stack/... -timeout 600s`
Expected: PASS, including new TC1/TC2/TC3/TC5 invariants.

- [ ] **Step 3: Run full module suite**

Run: `go test ./... -timeout 900s`
Expected: PASS or known-skipped.

- [ ] **Step 4: Push**

```bash
git push origin main
```

- [ ] **Step 5: Verify on codeberg**

Open `https://codeberg.org/zerone-dev/zerone/commits/branch/main` and confirm the Plan 1 commits are visible. The tip should bind TC1, TC2, TC3, TC5 in code, with TC4 (Plan 2) and TC6 (Plan 4) marked as in-flight bindings in the doctrine doc.

---

## Self-Review

After implementing all tasks above, verify:

1. **Spec coverage:** Each of TC1, TC2, TC3, TC5 has a corresponding code path AND a corresponding cross-stack invariant test. TC4 and TC6 are explicitly out-of-scope (Plans 2, 4).
2. **Position layer:** `x/knowledge/doc.go` declares TC1-TC5.
3. **Voice layer:** `tok_bundle_extracted` and `tok_snapshot_root_pinned` events emit with `tok_commitment` attribute populated.
4. **Refusal layer:** `BundleToK` rejects malformed selectors with messages that name the protecting commitment.
5. **CLI surface:** `zeroned query knowledge bundle-tok rooted-subtree|ancestor-cone|frontier` works against a localnet.
6. **Capability advertisement:** `RouteBCapabilities` returns `tok_capabilities` populated.
7. **All tests green** in `x/knowledge/keeper/` and `tests/cross_stack/`.

If any of these are missing, add the gap-closing task before declaring Plan 1 complete. The doctrine commitment to "the doctrine and the contract are one" applies to this plan: every TC named here must be both code-bound and test-bound before Plan 1 ships.
