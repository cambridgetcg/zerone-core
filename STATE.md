# zerone-core — STATE

name: zerone-core
kind: project
language: see repo
runs-on: this machine

## state
phase: source-consolidation
health: green
last-commit: e0a354b build: bind binary provenance and refresh integrity pins
uncommitted: 0 files
freshness: reviewed 2026-07-29T16:31:34Z

## knows
- 23 custom Cosmos SDK modules and 166 protobuf Msg request types
- consensus store version 6 and the consolidation-safety-v1 upgrade
- provisional knowledge conjectures and the K-alpha recognition layer
- fail-closed zerone-2 release, authority, and ceremony controls

## can
- declare its state via STATE.md
- be discovered by discover.py
- be cross-checked by trust.py
- build a commit-identifiable zeroned binary
- verify creed, recursion, generated API, SDK, and release integrity

## needs
- a governance-scheduled consolidation-safety-v1 validator rollout
- complete, signed zerone-2 ceremony artifacts before any launch
- bounded designs for capture recovery and training-fund replay protection

## how-to-talk-to-me
entry-point: README.md
