# R5-4 — Dynamic Pricing: Demand, Surge, USD-Stable, Free Tier

## Goal

Implement demand tracking, surge pricing (3-tier), USD-stable pricing,
and the free tier for the toolbox module.

## Working Directory

`/Users/yuai/Desktop/Zerone`

## Reference

- `/Users/yuai/Desktop/legible_money/x/toolbox/keeper/demand.go` — demand window tracking
- `/Users/yuai/Desktop/legible_money/x/toolbox/keeper/surge.go` — surge multiplier calculation
- `/Users/yuai/Desktop/legible_money/x/toolbox/keeper/usd_pricing.go` — USD-stable pricing
- `/Users/yuai/Desktop/legible_money/x/toolbox/keeper/free_tier.go` — free tier logic

**Depends on R5-1 and R5-2** (types and keeper must exist).

## Demand Tracking (`keeper/demand.go`)

### DemandWindow Ring Buffer
Each tool has a DemandWindow — a ring buffer of (block_height, call_count) entries.

- On each tool call: increment call count for current block
- Window size = params.demand_window_size (default 1000 blocks ≈ 42 min)
- Also track a global demand window (tool_id = "__global__")

### Utilization Ratio
```
per_tool_utilization = sum(window_calls) / (window_size * target_calls_per_block_per_tool)
global_utilization = sum(global_calls) / (window_size * target_global_calls_per_block)
```
Both capped at 1,000,000 BPS (100%).

## Surge Pricing (`keeper/surge.go`)

### Three Pricing Tiers

| Tier | Categories | Surge Behavior | Max Multiplier |
|------|-----------|----------------|----------------|
| Essential (0) | data_retrieval, utility, formatting | **No surge** | 1.0× |
| Standard (1) | data_analysis, verification, communication, monitoring | Linear surge | 2.0× |
| Heavy (2) | computation, composite, integration | Exponential surge | 10.0× |

### Surge Calculation

```
if utilization < surge_threshold (50%):
  multiplier = 1.0×

if utilization >= surge_threshold and < surge_critical (80%):
  Standard: linear interpolation 1.0× → tier_max
  Heavy: quadratic interpolation 1.0× → tier_max

if utilization >= surge_critical (80%):
  Standard: tier_max (capped)
  Heavy: exponential up to tier_max (capped at params.max_surge_multiplier_bps)
```

Essential tier ALWAYS returns 1.0× (no surge).

### Output
`GetSurgeMultiplier(ctx, toolID, category) uint64` — returns multiplier in BPS (1,000,000 = 1.0×).

## USD-Stable Pricing (`keeper/usd_pricing.go`)

For tools with `target_price_usd` set (non-zero):

1. Get ZRN/USD price from x/billing keeper's oracle (3-tier: manual → TWAP → fallback)
2. Convert: `uzrn_price = target_price_usd * 1,000,000 / zrn_price_usd`
3. Clamp to [min_price_per_call, max_price_per_call] if set
4. Apply surge multiplier on top

For tools WITHOUT target_price_usd: use fixed price_per_call, then apply surge.

### Effective Price Calculation
```go
func (k Keeper) GetEffectivePrice(ctx, tool) uint64 {
    // 1. Check free tier first
    if k.TryFreeTierCall(ctx, caller, tool) {
        return 0
    }
    
    // 2. Base price
    var basePrice uint64
    if tool.TargetPriceUSD != "" && tool.TargetPriceUSD != "0" {
        basePrice = k.CalculateUSDStablePrice(ctx, tool)
    } else {
        basePrice = parseUint(tool.PricePerCall)
    }
    
    // 3. Apply surge
    surge := k.GetSurgeMultiplier(ctx, tool.ID, tool.Category)
    effectivePrice = basePrice * surge / 1,000,000
    
    return effectivePrice
}
```

## Free Tier (`keeper/free_tier.go`)

### Rules
- 50 free essential-category calls per epoch per agent (params.free_calls_per_epoch)
- Gated by x/home registration age (params.min_home_age_blocks ≈ 7 hours) — anti-sybil
- Only applies to Essential tier categories (data_retrieval, utility, formatting)
- Free calls don't generate revenue (no distribution)

### Implementation
```go
func (k Keeper) TryFreeTierCall(ctx, caller, tool) bool {
    if !params.FreeCallsEnabled { return false }
    if !IsEssentialCategory(tool.Category) { return false }
    
    // Check home age (anti-sybil)
    home := k.homeKeeper.GetHomeByOwner(ctx, caller)
    if home == nil { return false }
    age := currentBlock - home.CreatedAtBlock
    if age < params.MinHomeAgeBlocks { return false }
    
    // Check/update allowance
    allowance := k.GetFreeAllowance(ctx, caller)
    currentEpoch := currentBlock / params.DemandWindowSize
    if allowance.Epoch != currentEpoch {
        allowance = FreeAllowance{Caller: caller, Epoch: currentEpoch, UsedCalls: 0}
    }
    if allowance.UsedCalls >= params.FreeCallsPerEpoch { return false }
    
    allowance.UsedCalls++
    k.SetFreeAllowance(ctx, allowance)
    return true
}
```

## Integration with CallTool

The CallTool message handler (from R5-2) should call `GetEffectivePrice` to determine
the actual cost, then charge accordingly. If price is 0 (free tier), skip payment
and revenue distribution but still record the call and update demand/trust.

## Conventions
- Token: uzrn. Module path: github.com/zerone-chain/zerone
- BPS: 1,000,000 scale. Surge multipliers also in BPS (1,000,000 = 1.0×, 2,000,000 = 2.0×)
- Use `math/big` for overflow-safe math
- Run `go build ./...` before finishing
