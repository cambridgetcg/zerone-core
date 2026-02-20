# R5-6 — Toolbox Tests: Comprehensive Test Suite

## Goal

Write comprehensive tests for the entire toolbox module, covering all keeper
functionality, trust engine, dynamic pricing, free tier, composability,
and adversarial scenarios.

## Working Directory

`/Users/yuai/Desktop/Zerone`

## Reference

- `/Users/yuai/Desktop/legible_money/x/toolbox/keeper/*_test.go` — 10,255 LOC of tests
- All R5 implementation files in `x/toolbox/keeper/`

**Depends on R5-1 through R5-5** (full module must exist).

## Test Categories

### 1. Tool Registration & Lifecycle
- Register tool happy path (all tool types)
- Register with invalid category → error
- Register with invalid license → error
- Register exceeding max dependencies → error
- Upgrade tool → new ID, linked to previous
- Deprecate → calls still work during grace period
- Retire → calls fail immediately
- Status transitions: draft → testing → active → deprecated → retired

### 2. Contributor Management
- Add contributor → pending until accepted
- Accept contributorship → active
- Share reallocations sum correctly (≤ 1M)
- Lock shares → changes require governance
- Max contributors enforcement
- Contributor revenue tracking (total_earned updates)

### 3. Revenue Distribution
- Basic call: 55/22/13/10 split
- Contributor pro-rata distribution
- Protocol sub-split: 50/30/20
- Revenue with governance-changed splits
- Zero-price tool → no distribution
- Free tier call → no distribution but call recorded

### 4. Trust Engine
- 5-component calculation with known inputs
- Usage component: unique callers, recency decay
- Reliability component: success rate with min-call interpolation
- Peer component: multi-hop dampening, same-author penalty
- Contributor component: tier-based scoring
- BeginBlocker trust update cycle
- Verified status grant and demotion (grace period)
- Trust tier boundaries (all 5 tiers)

### 5. Dependency DAG
- Simple dependency chain: A → B → C
- Cycle detection: A → B → C → A (rejected)
- Self-dependency rejected
- Depth limit enforcement
- Dependency count limit
- Trust tier eligibility: tier 0 tools can't be depended on
- Dependency tree query returns correct structure
- UpdateDependency swaps correctly

### 6. Composite Tool Execution & Revenue Cascade
- Composite tool calls all dependencies
- Revenue cascades: each dep gets full distribution
- Own revenue = total - dependency costs
- Nested composites (composite depends on composite)
- Depth limit on composite execution

### 7. Dynamic Pricing
- **Demand tracking:** ring buffer fills and wraps correctly
- **Surge pricing:**
  - Essential tier: no surge regardless of utilization
  - Standard tier: linear surge, caps at 2×
  - Heavy tier: exponential surge, caps at 10×
  - Below threshold: no surge
  - Above critical: capped at max
- **USD-stable pricing:** conversion with oracle price, clamped to min/max
- **USD-stable + surge:** both applied correctly

### 8. Free Tier
- Eligible call (essential category, old enough home) → free
- Non-essential category → not free
- No home → not free
- Home too young → not free
- Allowance exhausted (51st call) → charged
- Epoch reset → fresh allowance
- Free tier disabled via params → charged

### 9. Adversarial Scenarios
- **Sybil free tier abuse:** Create many homes to get more free calls → blocked by min_home_age
- **Circular dependency exploit:** Try to create revenue loops → blocked by cycle detection
- **Trust manipulation:** Self-calling to inflate usage → deduplicated within 10 blocks
- **Surge gaming:** Massive call spike to grief other users → capped by max multiplier
- **Revenue drain:** Tool with 100% contributor share → impossible, splits enforced
- **Share inflation:** Add contributors until shares > 1M → validation prevents
- **Retired dependency:** Depend on retired tool → rejected at registration
- **Ghost contributor:** Contributor who never accepts → revenue goes to accepted only

### 10. Edge Cases
- Tool with 0 contributors → all revenue to protocol/research/burn
- Tool with price "0" → free, but not free-tier (always free)
- Concurrent calls in same block → demand window handles correctly
- Very large numbers (max uint64 amounts) → no overflow

## Test Utilities

Create a test helper `keeper/testutil_test.go`:
- `setupKeeper()` — mock bank, knowledge, billing, home, BVM, staking keepers
- `registerTestTool(...)` — convenience helper
- `callTestTool(...)` — convenience helper
- Mock keepers that track calls for assertion

Use the same testing patterns as `x/channels/keeper/keeper_test.go` in this repo.

## Conventions
- Token: uzrn. Module path: github.com/zerone-chain/zerone
- BPS: 1,000,000 scale
- Run `go build ./...` and `go test ./x/toolbox/...` before finishing
- Target: all tests passing, zero failures
