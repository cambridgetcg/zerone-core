# R3-3 — Dynamic Pricing: USD-Stable + Oracle Wiring

## Goal

Wire the billing module's 3-tier oracle to the liquidity pool's TWAP.
When ZRN price changes, query prices auto-adjust to maintain USD stability.
Also port the USD-stable tool pricing infrastructure that will be used by
the toolbox module in R5.

## Dependencies

- R3-1 (billing) and R3-2 (liquiditypool) must be complete

## Working Directory

`/Users/yuai/Desktop/Zerone`

## Reference

- `/Users/yuai/Desktop/legible_money/x/billing/keeper/dynamic_pricing.go`
- `/Users/yuai/Desktop/legible_money/reports/batch-20/B20-3-usd-stable-pricing.md`
- `/Users/yuai/Desktop/legible_money/reports/batch-17/B17-3-dynamic-pricing.md`

## Implementation

### 1. Wire LiquidityPoolKeeper into Billing

In `app/app.go`:
```go
app.BillingKeeper.SetLiquidityPoolKeeper(app.LiquidityPoolKeeper)
```

In `x/billing/types/expected_keepers.go` — already has the interface from R3-1.

### 2. USD-stable query pricing

Enhance `CalculateQueryPrice` to adjust base price for ZRN/USD:

```go
func (k Keeper) CalculateQueryPrice(ctx sdk.Context, fact *knowledgetypes.Fact) (*types.QueryQuote, error) {
    params := k.GetParams(ctx)

    // Get ZRN price in USD
    zrnPriceUSD := k.GetZRNPriceUSD(ctx) // from 3-tier oracle

    var basePriceUZRN uint64
    if zrnPriceUSD > 0 {
        // USD-stable: target $X per query, convert to uzrn
        targetUSD := parseUint(params.BaseQueryPriceUSD) // micro-USD
        basePriceUZRN = targetUSD * 1_000_000 / zrnPriceUSD
        // Clamp to [min, max]
        basePriceUZRN = clamp(basePriceUZRN, params.MinQueryPrice, params.MaxQueryPrice)
    } else {
        // Oracle unavailable — fall back to fixed ZRN price
        basePriceUZRN = parseUint(params.BaseQueryPrice)
    }

    // Apply confidence/novelty/freshness multipliers (existing from R3-1)
    // ...
}
```

### 3. New billing params for USD pricing

Add to Params:
```protobuf
string base_query_price_usd = 10;   // micro-USD, e.g. "10000" = $0.01
string min_query_price = 11;         // uzrn floor
string max_query_price = 12;         // uzrn ceiling
```

### 4. Shared USD pricing utility

Create `x/common/pricing/usd_stable.go`:
```go
// CalculateUSDStablePrice converts a USD target price to uzrn using the oracle.
// Reusable by billing AND toolbox modules.
func CalculateUSDStablePrice(targetUSD, zrnPriceUSD, minUZRN, maxUZRN uint64) uint64 {
    if zrnPriceUSD == 0 {
        return 0 // oracle unavailable
    }
    price := targetUSD * 1_000_000 / zrnPriceUSD
    return clamp(price, minUZRN, maxUZRN)
}
```

### 5. Price feed events

Emit events when oracle price changes significantly (>5% from last emitted):
```go
ctx.EventManager().EmitEvent(sdk.NewEvent(
    "oracle_price_update",
    sdk.NewAttribute("denom", "uzrn"),
    sdk.NewAttribute("price_usd", strconv.FormatUint(price, 10)),
    sdk.NewAttribute("source", source), // "manual", "twap", "fallback"
))
```

## Tests

| Test | Validates |
|------|-----------|
| `TestUSDStable_OracleAvailable` | Price adjusts with ZRN/USD rate |
| `TestUSDStable_ZRNAppreciates` | ZRN 10x → query costs 10x fewer uzrn |
| `TestUSDStable_ZRNDepreciates` | ZRN 0.1x → query costs 10x more uzrn |
| `TestUSDStable_FloorClamp` | Price never below min_query_price |
| `TestUSDStable_CeilingClamp` | Price never above max_query_price |
| `TestUSDStable_OracleUnavailable` | Falls back to fixed ZRN price |
| `TestUSDStable_OracleStale` | Stale TWAP → fallback |
| `TestWiring_BillingUsesPool` | Full integration: pool → TWAP → billing price |
| `TestSharedUtility_Reusable` | Common pricing utility works standalone |

## Verification

```bash
go build ./...
go test ./x/billing/... ./x/liquiditypool/... ./x/common/... -count=1 -v
```

## Commit

```
feat(billing): USD-stable pricing via TWAP oracle, shared pricing utility
```

## Do NOT

- Build a second oracle (reuse billing's 3-tier)
- Use floating point for price calculations
- Skip the floor/ceiling clamps (ZRN at $10,000 shouldn't make queries cost dust)
- Make USD pricing mandatory (it's opt-in via params, fixed ZRN is the fallback)
