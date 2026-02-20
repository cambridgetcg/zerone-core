# R4 — Agent Infrastructure

**Goal:** Agents can register, build homes, use tools, form partnerships, open payment channels.

## Sessions

| # | File | Scope |
|---|------|-------|
| R4-1 | R4-1-home.md | Home proto + module: agent homes, guardians, keys, sessions, dead-man switch |
| R4-2 | R4-2-partnerships.md | Partnerships proto + module: formation, tiers, consensus ops, safety freezes |
| R4-3 | R4-3-bvm.md | BVM proto + module: contract lifecycle, opcodes, gas bridge, scheduled execution |
| R4-4 | R4-4-channels.md | Channels proto + module: payment channels, state updates, disputes |
| R4-5 | R4-5-infra.md | Schedule + compute_pool + discovery protos + modules (batched) |

**Exit criteria:** Agent can register home, deploy BVM contract, open payment channel.

## Dependencies (from R1–R3)
- `x/auth` — account management
- `x/staking` — validator set
- `x/billing` — pricing integration
- `x/knowledge` — fact queries (BVM bridge)
- `x/gov` — parameter updates
- `x/common` — BasisPoints, RevenueSplit
- `x/tokens` — token operations
