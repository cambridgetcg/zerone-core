# Batch R1 — Scaffold + Proto Foundation + Auth + Staking

## Goal

From zero to a buildable chain binary that handles account registration
and validator staking. CI runs on every push. Proto tooling generates
all types. x/upgrade is wired from the start.

## Context

This is a clean rewrite of the Legible Money draft (277k lines, 22 batches,
33 modules, all tests passing). The draft is the spec — we're porting
proven designs into a proto-first, production-quality codebase.

**Key references in the draft repo (`/Users/yuai/Desktop/legible_money/`):**
- `x/auth/` — account registration, sessions, recovery, freeze
- `x/staking/` — 4-tier validator system, delegation, reputation
- `proto/legible/auth/` — existing proto definitions
- `proto/legible/staking/` — existing proto definitions
- `reports/PROGRESS-REPORT.md` — full module census
- `docs/PARAMETERS.md` — all parameter defaults
- `reports/audits/` — security findings to bake in

**Naming changes:**
- Module path: `github.com/zerone-chain/zerone`
- Binary: `zeroned`
- Token: ZRN / uzrn (6 decimals)
- Chain ID: `zerone-testnet-1`
- All proto packages: `zerone.<module>.v1`

## Sessions (5)

| ID | Focus | Dependencies |
|----|-------|-------------|
| R1-1 | Repo scaffold: go.mod, Makefile, CI, buf, proto-gen, directory structure | None |
| R1-2 | Core proto: shared types, module options, x/upgrade integration | R1-1 |
| R1-3 | Auth proto + full module port | R1-2 |
| R1-4 | Staking proto + full module port | R1-2 |
| R1-5 | Genesis framework + integration test: InitGenesis/ExportGenesis, `zeroned init` works | R1-3, R1-4 |

## Run Order

- **Wave 1:** R1-1
- **Wave 2:** R1-2
- **Wave 3 (parallel):** R1-3, R1-4
- **Wave 4:** R1-5
