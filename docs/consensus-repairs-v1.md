# consensus-repairs-v1 — draft carrier, not deployment authority

This successor starts from the reviewed stack at `8accab3c84866e0ca4116ecd1b9a22c7cc3d2f15`.
It does not schedule an upgrade, activate a network, move legacy funds, or change
`app/app.go`, the frozen H2 map, or an authority-geometry pin. Independent review
and a dev-1-copy rehearsal remain necessary.

## Blocking prerequisite, not a claimed fix

The requested Q1 “not observed means neutral” behavior is **not in this stack**.
The input-selection repair changes the supply denominator/domain count but does
not add a registry-observation signal or a third health state. See
`x/staking/keeper/alignment_adapters.go`, `x/alignment/keeper/sensors.go:185-191`
and `x/alignment/module.go`.

This draft's census refuses an empty custom-staking registry, before the height.
That is conservative refusal, **not an implementation of neutral pacing**. Do
not treat a passing synthetic fixture with a seeded registry as evidence that
dev-1 is ready. The Q1 prerequisite must be resolved in its owning, separately
reviewed fix before treating this carrier as a release candidate. Fresh-genesis
marker validation also does not implement Q1. The #124 endorsement gate is not
included: there are exactly ten markers, not an eleventh.

## Version boundary

The exact source is `claimIntegrityTargetVersionMap()`; the target differs only
in these eleven entries:

| Module | Source | Target |
|---|---:|---:|
| zerone_gov | 3 | 4 |
| knowledge | 12 | 13 |
| capture_defense | 1 | 2 |
| qualification | 1 | 2 |
| capture_challenge | 1 | 2 |
| vesting_rewards | 3 | 4 |
| alignment | 1 | 2 |
| zerone_ontology | 1 | 2 |
| emergency | 2 | 3 |
| substrate_bridge | 1 | 2 |
| ibcratelimit | 1 | 2 |

`home` stays at 1. Each module registers the shared
`consensusrepairsmigration.Require` as a guarded no-op migration. Only the named
application handler supplies that execution context. Earlier release fixtures
pin their original compiled targets; the historical target maps are not edited.
The five knowledge guards, survival handoff and accounting guards delegate the
new boundary to the exact full-map owner check.

## Handler, in order

1. The existing `app/upgrades.go` registration hook installs the handler. The
   existing module PreBlock pass runs it before BeginBlock at H. Require exact
   committed H-1, execution/header H, the named positive-height plan, empty Info,
   no deprecated fields, and the exact on-chain plan. The source chain must be
   `zerone-dev-1` or an explicitly isolated `zerone-rehearsal-` chain.
2. Require exact supplied/stored source and compiled target maps; completed or
   native H3/accounting lineage; completed claim-integrity and its predecessors;
   no carrier receipt, done height or unsafe-skip height; every marker absent/off
   and free of faults; no census blocker.
3. Run all eleven guarded no-op migrations in one cache. Require the returned
   map to equal the target. Do not call `ReconcileModuleAccountPerms`.
4. Enable every entry by looping over `ConsensusRepairsV1Markers()`. Read back
   each entry immediately and require true. There is no partial-success mode.
5. Write `upgrade_receipt_consensus-repairs-v1` through the knowledge migration
   marker API: height, chain ID, SHA-256 of the canonical JSON plan, all ten
   marker readbacks, full census findings, counts and digests.
6. Compare every mounted persistent store byte-for-byte with the preimage plus
   exactly ten marker keys and that receipt. This includes keeping knowledge
   `0x87` unchanged. Any unexpected write or later marker fault discards the
   entire cache. Emit one digest event per census, then release the cache.
7. The SDK records the target version map, protocol version, done height and
   clears the plan. The app test checks that complete outer store diff too.

At startup the new binary accepts target state only with all markers true and
coherent receipt/done history, or native/imported genesis with neither invented
receipt nor done height. Source state is admitted only at the exact local and
on-chain H-1 plan, and the census runs again there. Earlier heights are refused.
The handler repeats source validation at H because a preflight is not a lock.

## Census contract and outcomes

`CensusConsensusRepairsV1` reads raw records rather than legacy iterator helpers
that skip decode failures. The shared implementation is used by both the
handler and the offline command. It writes no state and emits no events.

This draft imposes one **global** ceiling over the complete mounted-store image:
100,000 records or 64 MiB of key-plus-value bytes. It also reserves ten marker
rows, one receipt row and a conservative receipt envelope before activation.
This is stricter than separate per-prefix limits: a larger real chain is a
blocker, not a partial scan. Real-copy timing and size have not been measured.

Each finding is `held` or `blocker`. `held` means retained without redistribution;
the reason distinguishes a real execution hold from retained history or a
prospective input discontinuity. It never establishes an escrow entitlement.
Decode failures, bad key/identity joins, malformed amounts, iterator errors and
resource exhaustion refuse activation. Present-but-empty markers are faults.

| Census | Held / retained and receipted | Blockers |
|---|---|---|
| #117 seat runoffs | Original IDs and history remain unchanged | Bad key/ID, duplicate/missing/wrong-seat parents, incomplete child pairs, orphan parents, counter at/below a stored ID or exhausted allocation headroom |
| #118 reformulations | Pending invalid ballots/top ties; pooled escrow remains governed by the repair's hold rule | NUL bounty/augmentation IDs (also inactive/finalized), malformed records or missing/dangling/foreign bounty indexes |
| #119 histories | Histories with at least two submission blocks; future analysis uses integer variance | Malformed history or identity |
| #120 qualification | Every EXPIRED record with a retained nonempty amount, its amount, module balance and ambiguous payment history | Invalid amount/key, StakeLockPeriod addition overflow; no historical refund is inferred |
| #121 challenge backing | Backing total and balance; all records untouched | Any RESOLVED+REJECTED record, inconsistent status/resolution, or balance below pools plus OPEN/EVIDENCE/UNDER_REVIEW refundable stakes |
| #122/#124 alignment inputs | Bank supply, legacy mint ledger, complete custom registry and active stake/count, all/active domain counts | Malformed inputs; empty registry while Q1 is unimplemented |
| #125 halt reasons | Legacy/scoped counts and unknown evidence; scoped unknown contributes one conservative forward-gate count, not rollback justification | Malformed envelopes or proposal/key IDs; every pending legacy rollback conservatively blocks because historical admission evidence was not recorded |
| #126 adapter minima | Unsupported required-domain minima retained; new work refused by the selected semantics | Malformed registration/key; fresh-genesis unsupported minima separately fail `tools/genesis-check` |
| #127 IBC refunds | Legacy flows retained, no quota debit on refund; capacity retained until window reset | Occupied `0x03` generation prefix, malformed generation/flow/rule, pre-carrier charge identity, invalid amounts or overflow-prone window |

The pending-rollback and complete-image bounds are intentionally conservative
restrictions. Do not “resolve” them by deleting history, redistributing money or
raising a limit at the activation height. Reconciliation requires separate
review and authority. A clean census is not a decision to schedule.

The global digest hashes sorted, length-framed `(store, key, value)` bytes.
Each named census also commits the complete stores it reads (including bank
where relevant), using fixed store order and the raw-store digests. These are
state-integrity commitments, not scientific truth or proof of payment.

## Genesis and faults

`validateClaimIntegrityGenesisSelection` chains the new selection check before
InitGenesis writes. Every registry field must be explicitly true; absence,
false and null refuse. Export/import carries the selection without manufacturing
applied history. The genesis checker refuses required-domain adapters with
DISTINGUISHED or unknown minima; UNSPECIFIED, PROBATIONARY and ACTIVE are the
only supported floors. An open adapter has no qualification floor.

A malformed marker in a block hook halts every node deterministically; in a
message it fails the transaction. The #127 receive panic and the pinned gov
ABCI panic are accepted no-error-path sites. Only errors wrapping
`activation.ErrMarkerFault` acquire the new downstream halt behavior; ordinary
errors keep legacy handling. The composed app test uses the real
capture-defense → capture-challenge adapter and verifies this propagation.
`Enable` never repairs a bad byte. Repair forward through a separately reviewed
state repair, never log-and-skip or silently downgrade.

## Offline re-check

Build without network access using the existing dependency cache:

    GOFLAGS='-mod=readonly -p=4' GOPROXY=off go build -buildvcs=false -trimpath -o <output>/consensus-repairs-census ./tools/consensus-repairs-census

After separately authorized acquisition of a stopped-node copy, with independent
height/AppHash evidence:

    <output>/consensus-repairs-census --copied-db --home <absolute-copy-home> --chain-id zerone-dev-1 --expected-height <H> --expected-app-hash <lowercase-64-hex>

Only copied `data/application.db` is opened, with backend-enforced read-only
GoLevelDB options and fast-node rebuilding disabled. No key, configuration or
genesis file is read. Symlink path components and protected default/private home
names are refused. GoLevelDB only is supported in this draft; not Pebble or a
JSON genesis export. Do not point it at a running or only database.

Exit 0: completed census with no blocker. Exit 2: JSON report with blockers.
Exit 1: invalid arguments, identity mismatch, open/load/close/output fault.
The chain ID is explicitly an assertion supplied by the operator, not something
an application DB proves. Committed height and AppHash are read and compared.
The report includes the stored module version map. It can run before
claim-integrity, but a successful census alone does not satisfy the handler's
exact predecessor/plan checks. Tests compare every copied source file before
and after both passing and failing scans.

## Rehearsal still required — separate lane

No dev-1 node or state copy was accessed for this carrier's source work. The
supplied host reading reports height 3,224,032 and knowledge 11, but explicitly
omits the complete version map, qualification escrow and NUL-ID inventory.
Acquire those under a separate authorization; do not treat the partial reading
as a complete census.

The built `zeroned --help` has no `in-place-testnet` command. SDK v0.53.8 supplies
`server.InPlaceTestnetCreator` through `AddTestnetCreatorCommand`, but Zerone
currently calls only `server.AddCommands`. The rehearsal lane must address this
before producing blocks on a copy. No swap utility was built or invoked here,
and no original validator key may be copied. Prefer the SDK path if exposed by
the rehearsal candidate; otherwise build/review the isolated swap tool.

The rehearsal must still demonstrate A→claim-integrity→carrier, old-binary halt,
exact H-1/H startup, real-data census, exact diff/receipt, restart and advancing
blocks across alignment, capture analysis, gov tally and Route B. Include an
amd64 node and AppHash agreement, architecture-sensitive #119 tests, malformed
marker halt consistency, export/import, wall time and whole-transaction #127
gas. The plan's #123 HomeCounter restoration is also absent from this stack's
`x/home/keeper/keeper.go:465-478`; its nonempty-home export/import check remains
unproved and is not fixed by keeping the module version at 1.

The dev-1 runtime recognizes the exact 10→11, 11→12 and 12→13 named packet pairs;
applied-carrier checks require all eleven target versions plus unchanged
home/staking. Gateway descriptors accept knowledge 12/13 and retain versioned
publications. These are source-level contracts, not a tested multi-rung staging
workflow: the existing immutable single preparation directory and original
manifest are deliberately not automatically replaced or promoted. The rehearsal
lane must account for sequential deployment custody rather than overwrite them.

## Decisions and delivery boundary

- Resolve Q1 and obtain independent review of the exact carrier candidate.
- Rehearse the dev-1 copy; re-run the census at H−k and cancel before H if it blocks.
- Ask Yu for both claim-integrity and carrier heights together only after that.
- If #120 finds positive ambiguous EXPIRED amounts, attach the census and obtain
  Yu's word on the hold. Empty findings require no hold decision. Any #121
  coverage or #127 capacity-retention question is conditional on real findings.
- Nothing here authorizes zerone-1 evolution/relaunch, scheduling, signing,
  broadcasting, deployment, publishing a ready PR or merging.
