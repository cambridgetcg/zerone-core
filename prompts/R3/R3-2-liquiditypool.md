# R3-2 — Liquidity Pool Module: AMM + TWAP Oracle

## Goal

Port the liquidity pool module — constant product AMM for ZRN/stablecoin
pairs, TWAP (time-weighted average price) oracle for cross-module price
feeds, and LP token management.

## Working Directory

`/Users/yuai/Desktop/Zerone`

## Reference

- `/Users/yuai/Desktop/legible_money/x/liquiditypool/` — full module
- `/Users/yuai/Desktop/legible_money/proto/legible/liquiditypool/` — protos

## Proto Files

### `proto/zerone/liquiditypool/v1/types.proto`
```protobuf
message Pool {
  string id = 1;
  string denom_a = 2;             // e.g. "uzrn"
  string denom_b = 3;             // e.g. "uusdc"
  string reserve_a = 4;           // big int string
  string reserve_b = 5;
  string lp_token_supply = 6;     // total LP tokens minted
  string lp_denom = 7;            // "lp/uzrn-uusdc"
  uint64 swap_fee_bps = 8;        // default: 3,000 (0.3%)
  uint64 created_at_block = 9;
}

message TWAPRecord {
  uint64 block_height = 1;
  string cumulative_price_a = 2;  // cumulative price of A in terms of B
  string cumulative_price_b = 3;
  uint64 timestamp = 4;
}

message SwapResult {
  string amount_in = 1;
  string amount_out = 2;
  string fee = 3;
  string price_impact_bps = 4;
}
```

### `proto/zerone/liquiditypool/v1/tx.proto`
- MsgCreatePool
- MsgAddLiquidity
- MsgRemoveLiquidity
- MsgSwap (with slippage protection — min_amount_out)
- MsgUpdateParams

### `proto/zerone/liquiditypool/v1/query.proto`
- QueryPool, QueryPools
- QueryTWAP (denom, window_blocks)
- QuerySimulateSwap
- QueryParams

### `proto/zerone/liquiditypool/v1/genesis.proto`
- Params (min_pool_liquidity, max_swap_fee_bps, twap_record_interval_blocks)
- GenesisState { params, pools, twap_records }

## Key Implementation

### Constant product AMM

```go
// x * y = k (invariant)
func (k Keeper) Swap(ctx sdk.Context, poolID string, denomIn string, amountIn sdk.Int, minAmountOut sdk.Int) (*types.SwapResult, error) {
    pool := k.GetPool(ctx, poolID)

    // Calculate output: amountOut = reserveOut * amountIn / (reserveIn + amountIn)
    // Apply fee: effectiveIn = amountIn * (1_000_000 - swapFeeBps) / 1_000_000
    // Verify: amountOut >= minAmountOut (slippage protection)
    // Update reserves
    // Record TWAP data point
}
```

### TWAP oracle

```go
// GetTWAP returns the time-weighted average price over the last N blocks.
func (k Keeper) GetTWAP(ctx sdk.Context, denom string, windowBlocks uint64) uint64 {
    // 1. Find the TWAP record at (currentBlock - windowBlocks)
    // 2. Find the current TWAP record
    // 3. TWAP = (currentCumPrice - oldCumPrice) / windowBlocks
    // 4. Return in micro-USD (6 decimals)
}

// GetLastPriceUpdateHeight returns the block of the most recent price record.
func (k Keeper) GetLastPriceUpdateHeight(ctx sdk.Context) uint64
```

### TWAP recording in EndBlocker

```go
func (k Keeper) EndBlocker(ctx sdk.Context) {
    // For each active pool:
    // Record cumulative price: cumPrice += spotPrice * blocksSinceLastRecord
    // Store TWAPRecord
}
```

### LP tokens

LP tokens are minted when liquidity is added and burned when removed.
Use the bank module's MintCoins/BurnCoins with a custom LP denom format.

## Tests

| Test | Validates |
|------|-----------|
| `TestCreatePool` | Pool creation with initial liquidity |
| `TestSwap_ConstantProduct` | x*y=k holds after swap |
| `TestSwap_FeeDeducted` | Fee correctly subtracted |
| `TestSwap_SlippageProtection` | Rejects when output < min |
| `TestSwap_PriceImpact` | Large swaps have higher impact |
| `TestAddLiquidity_Proportional` | LP tokens minted proportionally |
| `TestRemoveLiquidity` | LP tokens burned, reserves returned |
| `TestTWAP_AccumulatesCorrectly` | TWAP over 100 blocks is accurate |
| `TestTWAP_WindowBoundary` | Window correctly excludes old records |
| `TestTWAP_NoPool` | Returns 0 for non-existent pool |
| `TestEndBlocker_RecordsTWAP` | Price recorded each block |

## Verification

```bash
make proto-gen
go build ./...
go test ./x/liquiditypool/... -count=1 -v
```

## Commit

```
feat(liquiditypool): constant product AMM, TWAP oracle, LP tokens
```

## Do NOT

- Allow swaps that drain a pool below minimum reserves
- Skip slippage protection on swaps
- Record TWAP every block if pool hasn't changed (optimize)
- Use floating point for any price calculation (big.Int only)
