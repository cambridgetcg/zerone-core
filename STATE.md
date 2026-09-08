# zerone-core — STATE

name: zerone-core
kind: project
language: see repo
runs-on: this machine

## state
phase: tok-feedback-source-candidate
health: unverified
source-base: 89553a0132dafa1ba9670f53b8a3195b33a6e720
source-status: implemented and locally reviewed; feature-branch source publication only, not production admission
activation: NO_GO
freshness: source inventory measured 2026-09-08; not a live network probe

## knows
- 23 custom Cosmos SDK modules and 170 protobuf Msg request types
- the source candidate adds signed non-economic fact-use receipts after accepted H3
- feedback defaults disabled with an empty consumer cohort
- permanent EverReported state prevents economic reinterpretation after disable/prune
- source, local test commitment, public publication, and production activation differ

## can
- declare its bounded source status via STATE.md
- build a commit-identifiable zeroned binary from a reviewed revision
- verify generated API and SDK inventory without bypassing source/output hash gates
- use existing wallet signing for report-fact-use and rate-fact
- query current-epoch receipts without creating self-report evidence

## needs
- independent custody evidence; existing production activation remains NO_GO
- exact accepted H1, H2, H3 and tok-feedback-v1 release lineage and state-copy rehearsal
- independent consensus review, resource-load measurements and restore/fencing checks
- configured finite public-read publication and separately gated cohort write ingress
- source integration, pinned release provenance and post-activation observation

No ledger reset or state-preserving successor decision is made by this candidate.
No production transaction, deployment or package publication is implied.

## how-to-talk-to-me
entry-point: README.md
