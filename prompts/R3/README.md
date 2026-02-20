# Batch R3 — Economic Layer

## Goal

Revenue flows, billing, dynamic pricing, governance, and token management.
After this batch, knowledge queries have a price, revenue splits work,
governance can change parameters, and the 2-of-2 research fund spending
mechanism is live.

## Context

R2 gave us the knowledge layer (PoT consensus) and vesting rewards (block
rewards + revenue routing). R3 builds the economic infrastructure on top:
how agents pay for queries, how prices adjust to demand, and how governance
controls all parameters.

**Draft reference modules:**
- `/Users/yuai/Desktop/legible_money/x/billing/` — query pricing
- `/Users/yuai/Desktop/legible_money/x/liquiditypool/` — AMM + TWAP oracle
- `/Users/yuai/Desktop/legible_money/x/gov/` — LIP governance
- `/Users/yuai/Desktop/legible_money/x/tokens/` — mint, delegate, wrap

## Sessions (6)

| ID | Focus | Dependencies |
|----|-------|-------------|
| R3-1 | Billing proto + module: query pricing, confidence/novelty/freshness curves | None |
| R3-2 | Liquidity pool proto + module: AMM, TWAP oracle for ZRN price | None |
| R3-3 | Dynamic pricing: USD-stable base, oracle wiring (billing ↔ liquiditypool) | R3-1, R3-2 |
| R3-4 | Gov proto + module: LIP lifecycle, stake-weighted voting, quorum | None |
| R3-5 | Research fund governance: 2-of-2 designated voter proposals | R3-4 |
| R3-6 | Tokens proto + module: mint, delegate power, wrap/unwrap | None |

## Run Order

- **Wave 1 (parallel):** R3-1, R3-2, R3-4, R3-6 (all independent)
- **Wave 2:** R3-3 (depends on R3-1, R3-2)
- **Wave 3:** R3-5 (depends on R3-4)

## Exit Criteria

- Knowledge query returns a price based on confidence/novelty/freshness
- ZRN price available via TWAP oracle
- USD-stable pricing adjusts base with oracle
- LIP governance proposal can change module params
- 2-of-2 research fund proposal works (submit → vote → execute/reject)
