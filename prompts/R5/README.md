# R5 — Toolbox Platform

**Goal:** The tool ecosystem works end-to-end. Tools register, earn revenue, build trust, handle surge pricing, offer free tiers, and compose via dependency DAGs.

## Sessions

| # | File | Scope |
|---|------|-------|
| R5-1 | R5-1-toolbox-proto.md | Toolbox proto + types: tools, contributors, trust, calls, demand, free tier |
| R5-2 | R5-2-toolbox-keeper.md | Toolbox keeper: registration, revenue distribution, contributor shares, state |
| R5-3 | R5-3-trust-composability.md | Trust engine + composability: 5-component scoring, DAG validation, revenue cascade |
| R5-4 | R5-4-dynamic-pricing.md | Dynamic pricing: demand tracking, surge pricing, USD-stable, free tier |
| R5-5 | R5-5-purpose-prompter.md | Purpose Prompter: knowledge scout, purpose analyzer, path formatter, composite tool |
| R5-6 | R5-6-toolbox-tests.md | Toolbox tests: comprehensive test suite + adversarial simulations |

**Exit criteria:** Purpose Prompter callable. Revenue cascades through dependency chain. Free tier works.

## Dependencies (from R1–R4)
- `x/billing` — pricing oracle, dynamic pricing config
- `x/knowledge` — fact queries (knowledge scout)
- `x/bvm` — contract execution (BVM-backed tools)
- `x/home` — agent homes (free tier age gating)
- `x/common` — BasisPoints, RevenueSplit
- `x/gov` — parameter updates
