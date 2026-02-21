# R8 — App Wiring + Ante Handlers + Genesis

**Goal:** Full chain boots, produces blocks, and passes smoke tests. All 32 modules fully
wired in app.go. Custom ante decorators ported. Genesis with axioms. CLI complete. Cosmovisor ready.

## Sessions

| # | File | Scope |
|---|------|-------|
| R8-1 | R8-1-app-wiring.md | Wire remaining 3 unwired keepers (home, partnerships, toolbox) + verify all 32 modules boot |
| R8-2 | R8-2-ante-handlers.md | Port 5 custom ante decorators + gas cost table from draft |
| R8-3 | R8-3-genesis.md | Full testnet genesis: 777 axioms, all module defaults, research fund, founder/AI accounts |
| R8-4 | R8-4-cli.md | CLI: all tx + query commands, quick-register, dashboard, explore helpers |
| R8-5 | R8-5-upgrade.md | x/upgrade: cosmovisor integration, migration stubs for every module |

**Exit criteria:** `zeroned start` boots a single-validator chain. Full ante handler chain active. Smoke test passes.

## Dependencies (from R1–R7)
- All 32 modules implemented and individually tested
- app.go has module manager, BeginBlocker/EndBlocker ordering
- Basic ante handler chain (SDK standard decorators only)

## Parallelism
- **Wave 1** (parallel): R8-1, R8-2, R8-3 (independent — app wiring, ante, genesis)
- **Wave 2** (parallel): R8-4, R8-5 (need R8-1 complete for full module list)
