# R3-1 — Billing Module: Query Pricing

## Goal

Port the billing module — dynamic pricing for knowledge queries based on
fact confidence, novelty, and freshness. Also handles provider registration
and the 3-tier oracle (manual → TWAP → fallback) for ZRN/USD conversion.

## Working Directory

`/Users/yuai/Desktop/Zerone`

## Reference

- `/Users/yuai/Desktop/legible_money/x/billing/` — full module
- `/Users/yuai/Desktop/legible_money/x/billing/keeper/pricing.go` — pricing curves
- `/Users/yuai/Desktop/legible_money/x/billing/keeper/dynamic_pricing.go` — USD oracle
- `/Users/yuai/Desktop/legible_money/x/billing/keeper/keeper_test.go` — 25 tests
- `/Users/yuai/Desktop/legible_money/docs/PARAMETERS.md` — billing params

## Proto Files

### `proto/zerone/billing/v1/types.proto`
```protobuf
message Provider {
  string address = 1;
  string name = 2;
  string status = 3;              // "active", "suspended"
  uint64 registered_at_block = 4;
  string total_revenue = 5;       // uzrn lifetime
  uint64 total_queries = 6;
}

message QueryQuote {
  string fact_id = 1;
  string base_price = 2;          // uzrn
  uint64 confidence_multiplier_bps = 3;
  uint64 novelty_multiplier_bps = 4;
  uint64 freshness_multiplier_bps = 5;
  string effective_price = 6;     // uzrn after all multipliers
}

message DynamicPricingConfig {
  string manual_zrn_price_usd = 1;  // micro-USD (6 decimals), manual override
  uint64 twap_window_blocks = 2;    // blocks for TWAP calculation
  uint64 staleness_blocks = 3;      // max age before oracle is "stale"
}
```

### `proto/zerone/billing/v1/genesis.proto`
```protobuf
message Params {
  // Base pricing
  string base_query_price = 1;              // default: "1000000" (1 ZRN)
  uint64 confidence_weight_bps = 2;          // how much confidence affects price
  uint64 novelty_weight_bps = 3;
  uint64 freshness_weight_bps = 4;

  // Revenue split (uses shared RevenueSplit from common)
  zerone.common.v1.RevenueSplit revenue_split = 5;

  // Dynamic pricing
  DynamicPricingConfig dynamic_pricing = 6;

  // Provider requirements
  string min_provider_stake = 7;             // uzrn
}
```

### `proto/zerone/billing/v1/tx.proto`
- MsgRegisterProvider
- MsgDeregisterProvider
- MsgQueryFact (pays for a knowledge query)
- MsgBatchQueryFacts
- MsgUpdateParams

### `proto/zerone/billing/v1/query.proto`
- QueryProvider, QueryProviders
- QueryQuote (get price without paying)
- QueryBatchQuote
- QueryParams
- QueryZRNPriceUSD (current oracle price)

## Key Implementation

### Pricing curves

```go
// CalculateQueryPrice returns the price for querying a specific fact.
// Price = BaseFee × (1 + confidenceBoost + noveltyBoost + freshnessBoost)
func (k Keeper) CalculateQueryPrice(ctx sdk.Context, fact *knowledgetypes.Fact) (*types.QueryQuote, error) {
    params := k.GetParams(ctx)
    base := parseUint(params.BaseQueryPrice)

    // Higher confidence = higher price (verified knowledge is worth more)
    confBoost := fact.Confidence * params.ConfidenceWeightBps / 1_000_000

    // Higher novelty = higher price (new facts are more valuable)
    novelty := k.calculateNovelty(ctx, fact) // based on recency + uniqueness
    noveltyBoost := novelty * params.NoveltyWeightBps / 1_000_000

    // Fresher facts cost more
    freshness := k.calculateFreshness(ctx, fact) // decays with age
    freshnessBoost := freshness * params.FreshnessWeightBps / 1_000_000

    effective := base * (1_000_000 + confBoost + noveltyBoost + freshnessBoost) / 1_000_000
    // ...
}
```

### 3-tier oracle

```go
// GetZRNPriceUSD returns ZRN price in micro-USD using 3-tier fallback:
// 1. Manual override (if set and non-zero)
// 2. TWAP from liquidity pool (if available and not stale)
// 3. Fallback to 0 (indicates unavailable)
func (k Keeper) GetZRNPriceUSD(ctx sdk.Context) uint64 {
    config := k.GetParams(ctx).DynamicPricing

    // Tier 1: Manual
    if manual := parseUint(config.ManualZrnPriceUsd); manual > 0 {
        return manual
    }

    // Tier 2: TWAP (from liquiditypool keeper, if wired)
    if k.liquidityPoolKeeper != nil {
        twap := k.liquidityPoolKeeper.GetTWAP(ctx, "uzrn", config.TwapWindowBlocks)
        if twap > 0 {
            lastUpdate := k.liquidityPoolKeeper.GetLastPriceUpdateHeight(ctx)
            if uint64(ctx.BlockHeight())-lastUpdate <= config.StalenessBlocks {
                return twap
            }
        }
    }

    // Tier 3: Unavailable
    return 0
}
```

### Cross-module keeper interface

```go
// LiquidityPoolKeeper — set post-init (pool may not exist yet)
type LiquidityPoolKeeper interface {
    GetTWAP(ctx sdk.Context, denom string, windowBlocks uint64) uint64
    GetLastPriceUpdateHeight(ctx sdk.Context) uint64
}
```

## Tests

Port 25 tests from draft. Add:

| Test | Validates |
|------|-----------|
| `TestPricing_ConfidenceCurve` | Higher confidence → higher price |
| `TestPricing_NoveltyCurve` | Newer facts → higher price |
| `TestPricing_FreshnessCurve` | Fresher facts → higher price |
| `TestOracle_ManualOverride` | Manual price takes priority |
| `TestOracle_TWAPFallback` | TWAP used when manual is 0 |
| `TestOracle_StalenessCutoff` | Stale TWAP → returns 0 |
| `TestOracle_Unavailable` | No oracle → returns 0 |
| `TestRevenueSplit_FromParams` | Split is governance-adjustable |
| `TestBatchQuery_PricingCorrect` | Batch pricing = sum of individual |

## Verification

```bash
make proto-gen
go build ./...
go test ./x/billing/... -count=1 -v
```

## Commit

```
feat(billing): query pricing with confidence/novelty/freshness curves, 3-tier oracle
```
