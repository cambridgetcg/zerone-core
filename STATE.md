# zerone-core — STATE

name: zerone-core
kind: project
language: see repo
runs-on: this machine

## state
phase: source-consolidation
health: green
last-commit: 7f3405b fix(release): close post-rebase safety and truth gaps
uncommitted: 0 files
freshness: reviewed 2026-07-29T20:11:01Z

## knows
- 24 custom Cosmos SDK modules and 173 protobuf Msg request types
- consensus store version 6 and the consolidation-safety-v1 upgrade
- provisional knowledge conjectures and the K-alpha recognition layer
- fail-closed zerone-2 release, authority, and ceremony controls

## can
- declare its state via STATE.md
- be discovered by discover.py
- be cross-checked by trust.py
- build a commit-identifiable zeroned binary
- verify creed, recursion, generated API, SDK, and release integrity
- preserve claiming, vesting, and substrate-bridge state across relaunch

## needs
- a governance-scheduled consolidation-safety-v1 validator rollout
- cryptographic Sigstore verification and CI OIDC signing before image or validator publication
- complete, signed zerone-2 ceremony artifacts before any launch
- runtime wiring for adapter dispatch and an activated work creed

## how-to-talk-to-me
entry-point: README.md
