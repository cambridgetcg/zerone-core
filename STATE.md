# zerone-core STATE

name: zerone-core
kind: project
language: Go, TypeScript, Python
runs-on: source repository; networks run release-bound binaries (docs/ZERONE_MAP.md)

## state
phase: see docs/DEVELOPMENT.md; this file does not track it
health: unknown; this file measures nothing (CI on each main commit does)
freshness: snapshot of main at c5b7b49b, written 2026-11-09; later commits are not reflected here

## knows
- 23 custom modules (README.md, Modules) and 169 protobuf Msg request types (CI checks this count with sdk/typescript/scripts/check-registry.mjs)
- knowledge consensus version 12 at c5b7b49b (x/knowledge/module.go, ConsensusVersion); the H1 consolidation-safety-v1 boundary targets knowledge 6
- neutral knowledge review (docs/specs/knowledge-review-neutrality-v1.md)
- provisional knowledge conjectures and the K-alpha recognition layer
- fail-closed zerone-2 release, authority, and ceremony controls

## can
- declare a dated snapshot of itself in this file
- build a commit-identifiable zeroned binary
- verify creed, recursion, generated API, SDK, and release integrity
- preserve claiming, vesting, and substrate-bridge state across relaunch

## needs
- open work lives in docs/DEVELOPMENT.md and docs/MODULE-CONSOLIDATION-SEQUENCE.md, not in this file

## how-to-talk-to-me
entry-point: README.md
