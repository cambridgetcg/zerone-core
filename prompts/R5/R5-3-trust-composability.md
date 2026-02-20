# R5-3 — Trust Engine + Composability

## Goal

Implement the 5-component trust scoring engine, dependency DAG validation
with cycle detection, and revenue cascade through dependency chains.

## Working Directory

`/Users/yuai/Desktop/Zerone`

## Reference

- `/Users/yuai/Desktop/legible_money/x/toolbox/keeper/trust.go` — trust engine (~400 LOC)
- `/Users/yuai/Desktop/legible_money/x/toolbox/keeper/dependencies.go` — DAG validation (~200 LOC)
- `/Users/yuai/Desktop/legible_money/x/toolbox/keeper/composite_executor.go` — composite tool execution
- `/Users/yuai/Desktop/legible_money/x/toolbox/keeper/bvm_executor.go` — BVM tool execution

**Depends on R5-1 and R5-2** (types and keeper must exist).

## Trust Engine (`keeper/trust.go`)

### 5-Component Weighted Scoring

Each component scores 0-1,000,000. Final score = weighted sum:

| Component | Weight | Measures |
|-----------|--------|----------|
| Usage | 30% (300,000) | Unique callers + total calls with recency decay |
| Verification | 25% (250,000) | Linked knowledge facts' verification status |
| Reliability | 20% (200,000) | Success rate over min 100 calls |
| Peer | 15% (150,000) | Trust of dependent tools (dampened by 50% per hop, max 3 hops) |
| Contributor | 10% (100,000) | Staking tier of contributors |

### Per-Component Details

**Usage Component:**
- Score based on unique callers in window (min 10 for full score)
- Apply recency half-life decay (240,000 blocks ≈ 1 week)
- Deduplicate same caller within 10 blocks

**Verification Component:**
- Query linked knowledge facts via KnowledgeKeeper
- Score = average confidence of linked facts
- 0 if no linked facts

**Reliability Component:**
- success_rate_bps = successes * 1,000,000 / total_calls
- Requires min 100 calls for meaningful score
- Below min: interpolate between 500,000 (neutral) and actual rate

**Peer Component:**
- Traverse dependency tree (max 3 hops)
- Each hop dampened by 50% (PeerDampeningBps = 500,000)
- Same-author dependencies penalized 50% (SameAuthorPenaltyBps = 500,000)
- Score = average of dampened dependency trust scores

**Contributor Component:**
- Map each contributor's staking tier to score:
  - Apprentice: 100,000
  - Verified: 400,000
  - Bonded: 700,000
  - Guardian: 1,000,000
  - Unknown/neutral: 500,000
- Score = weighted average by share_bps

### BeginBlocker Trust Update
Every `blocks_per_trust_update` blocks:
- Iterate all active tools
- Compute full 5-component score
- Store TrustSnapshot
- Update tool.TrustScore
- Check Verified status: if score drops below 700,000 (VerifiedMinRetentionScore), start grace period. If grace period expires and still below, remove is_verified

### Trust Tier Boundaries
```
Tier 0 (Unverified):  0 – 100,000
Tier 1 (Emerging):    100,001 – 300,000
Tier 2 (Established): 300,001 – 600,000
Tier 3 (Trusted):     600,001 – 800,000
Tier 4 (Verified):    800,001 – 1,000,000
```

## Dependency DAG (`keeper/dependencies.go`)

### Cycle Detection
- `CheckDependencyCycles(toolID, depIDs)` — iterative DFS bounded by max_dependency_depth
- Returns ErrDependencyCycle if adding depIDs would create a cycle back to toolID

### Dependency Validation
- `ValidateDependencies(depIDs, params)` — check all deps exist, not retired, trust tier ≥ 1 (Emerging), count ≤ max_dependencies, no duplicates

### Dependency Tree Query
- `GetDependencyTree(toolID, maxDepth)` — recursive tree construction for the query endpoint

## Composite Tool Execution (`keeper/composite_executor.go`)

When a composite tool is called:
1. Load tool and its dependency_ids
2. For each dependency, recursively call (with depth limit)
3. Sum dependency costs
4. ownRevenue = payment - dependencyCosts
5. Distribute ownRevenue to this tool's contributors
6. Each dependency already distributed its own revenue during its call

This creates a **revenue cascade** — payment flows through the entire DAG.

## BVM Tool Execution (`keeper/bvm_executor.go`)

For bvm_contract tools:
1. Load the BVM contract address
2. Call via BVMKeeper.CallContract with tool_gas_limit
3. Return success/failure + gas used

## Conventions
- Token: uzrn. Module path: github.com/zerone-chain/zerone
- BPS: 1,000,000 scale
- Use `math/big` for overflow-safe multiplication: `safeMulDiv(a, b, c)`
- Run `go build ./...` before finishing
