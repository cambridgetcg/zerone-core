# consensus-repairs-v1 — draft carrier, not deployment authority

This successor starts from the reviewed stack at `8accab3c84866e0ca4116ecd1b9a22c7cc3d2f15`.
It does not schedule an upgrade, activate a network, move legacy funds, or change
`app/app.go`, the frozen H2 map, or an authority-geometry pin. Independent review
and a dev-1-copy rehearsal remain necessary.

## Conditional Q1 blocker, not a claimed fix

The requested Q1 “not observed means neutral” behavior is **not in this stack**.
The input-selection repair changes the supply denominator/domain count but does
not add a registry-observation signal or a third health state. See
`x/staking/keeper/alignment_adapters.go`, `x/alignment/keeper/sensors.go:185-191`
and `x/alignment/module.go`.

The census blocks the Q1 discontinuity only when active custom stake is zero,
the legacy mint ledger (`vesting_rewards/0x0b`) is zero, and bank `uzrn` supply
is positive: legacy staking ratio is neutral and the fixed reading is zero.
An empty registry and an all-inactive registry are equivalent here. If both
denominators are positive, both zero-stake readings are zero; if both are zero,
both are neutral. Every other well-formed input is held and
surfaced, including both ratio readings. A conditional refusal is **not an
implementation of neutral pacing**. Real copy values decide whether a separate
Q1 fix is necessary; seeded fixtures do not establish dev-1 readiness.
Fresh-genesis marker validation does not implement Q1. The #124 endorsement
gate is not included: there are exactly ten markers, not an eleventh.

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
   and free of faults; no census blocker. The public census runs inside a
   throwaway CacheContext, never released even if a future helper writes.
3. Run all eleven guarded no-op migrations in one cache. Require the returned
   map to equal the target. Do not call `ReconcileModuleAccountPerms`.
4. Enable every entry by looping over `ConsensusRepairsV1Markers()`. Read back
   each entry immediately and require true. There is no partial-success mode.
5. Write `upgrade_receipt_consensus-repairs-v1` through the knowledge migration
   marker API: height, chain ID, SHA-256 of the canonical JSON plan, all ten
   marker readbacks and a compact census receipt. Every named census has one
   fixed-shape summary containing record/byte counts, its digest, held-finding
   count, zero blockers and pending-rollback count. No per-record finding,
   rollback identity or arbitrary status string is permanently receipted.
6. Compare the streamed SHA-256 of every mounted persistent store with the
   streamed source merged with exactly ten marker keys and that receipt. The
   expected digest is computed **before** migrations/marker writers. A separate
   knowledge `0x87` prefix digest must also remain equal. No complete image is
   materialised or cloned. Any unexpected insert, deletion, replacement or later
   marker fault discards the entire cache. Emit one digest event per census,
   then release the cache. SDK bookkeeping is outside this handler's write set.
7. The SDK records the target version map, protocol version, done height and
   clears the plan. The app test checks that complete outer store diff too.

At startup the new binary accepts target state only with all markers true and
coherent receipt/done history, or native/imported genesis with neither invented
receipt nor done height. Source state is admitted only at the exact local and
on-chain H-1 plan, and the census runs again there. Earlier heights are refused.
The handler repeats source validation at H because a preflight is not a lock.
At H, the full state is read three times (census, expected merged image and
post-image); startup at H-1 adds a fourth pass. Include these reads in the
stopped-copy wall-time and memory measurements.

## Census contract and outcomes

`CensusConsensusRepairsV1` reads raw records rather than legacy iterator helpers
that skip decode failures. The shared implementation is used by both the
handler and the offline command. It writes no state and emits no events.

Each of the nine censuses has its own 100,000-record / 64 MiB key-plus-value
budget, charged only for its declared reads below. The union of those selected
rows is retained once; the full state is streamed, not retained. Unrelated
alignment observations, SDK staking history and bank accounts do not consume a
census budget. A selected-prefix overflow refuses rather than truncates. There
is no whole-image or marker/receipt-headroom ceiling. Full-image hashing is
still linear in the entire state: peak memory and wall time at H must be
measured on an authorized stopped copy, not inferred from a synthetic pass.

| Census | Store / read prefix (hex; singleton means exact key) |
|---|---|
| seat_runoffs | zerone_gov: `10`, singleton `12` |
| reformulations | knowledge: singleton `06`, `62`, `63`, `65` |
| verification_histories | capture_defense: `05` |
| qualification_escrow | qualification: singleton `00`, `01`; bank: exact qualification module `uzrn` balance |
| challenge_backing | capture_challenge: `01`, `02`; bank: exact capture_challenge module `uzrn` balance |
| alignment_inputs | vesting_rewards: singleton `0b`; zerone_staking: `01`; zerone_ontology: `02`; bank: exact `uzrn` supply |
| halt_reasons | emergency: `02`; zerone_gov: `01`, `06`, singleton `0d`, `14` |
| adapter_minima | substrate_bridge: `84` |
| ibc_refunds | zerone_ibcrl: `01`, `02`, `03` |

Bank keys use the SDK collection codecs, not manually reconstructed addresses.
Marker presence and official readers are checked separately for all ten rows.
The report gives full-image totals, per-store and first-byte-prefix record/byte
counts, and per-census selected sizes/digests. `census_reads` breaks each census
down by its exact selected store/prefix or singleton, including zero counts.
These counts include unselected
alignment and staking history, without imposing a ceiling on those histories.

Each finding is `held` or `blocker`. `held` means retained without redistribution;
the reason distinguishes a real execution hold from retained history or a
prospective input discontinuity. It never establishes an escrow entitlement.
Decode failures in inspected records, bad key/identity joins, malformed amounts, iterator errors and
resource exhaustion refuse activation. Present-but-empty markers are faults.

The offline JSON retains per-record findings and the complete status histogram.
Each finding names its census (marker faults are outside the nine censuses).
The permanent receipt contains only global height/digest/size and nine compact
`censuses` summaries. `held_findings` counts findings, including aggregate
findings, not distinct held records; `pending_rollbacks` is a separate count
under `halt_reasons`. The offline rollback list appears once, not again in
findings. This receipt format supersedes the unactivated draft's detailed
receipt; it is a consensus-changing carrier revision, not a live-state migration.

| Census | Held / retained (details offline, counts/digests receipted) | Blockers |
|---|---|---|
| #117 seat runoffs | Original IDs and history remain unchanged | Bad key/ID, duplicate/missing/wrong-seat parents, incomplete child pairs, orphan parents, counter at/below a stored ID or exhausted allocation headroom |
| #118 reformulations | Pending invalid ballots/top ties; NUL bounty/augmentation IDs and ambiguous indexes held under zc-reformulation-id-delimiter, with counts; pooled escrow unchanged | Decode/identity faults, missing/dangling/foreign or malformed index values; indexes are matched to primary identities, not split on ambiguous NULs |
| #119 histories | Count of histories with at least two submission blocks plus census digest; future analysis uses integer variance | Malformed history or identity |
| #120 qualification | Each EXPIRED record with a **positive** retained amount, its amount, module balance and ambiguous payment history; zero/empty amounts are not per-record holds | Invalid amount/key, StakeLockPeriod addition overflow; no historical refund is inferred |
| #121 challenge backing | Backing total and balance; all records untouched | Any RESOLVED+REJECTED record, inconsistent status/resolution, or balance below pools plus OPEN/EVIDENCE/UNDER_REVIEW refundable stakes |
| #122/#124 alignment inputs | Bank supply, legacy mint ledger, registry/active stake/count, both staking ratios, all/active domains and full status histogram; every status other than exact `active` is held, including `archived` and arbitrary strings | Malformed inputs; zero active stake AND zero mint ledger AND positive bank supply |
| #125 halt reasons | Legacy/scoped counts and unknown evidence; each pending rollback's LIP ID, activation block, due-now flag and execute/cancel prediction on current state; out-of-window finalized halt payloads retained without decoding | Malformed envelopes/metadata or identity/LIP mismatch; malformed or mismatched halt payloads only inside the inclusive phase-start/current-height window, not the mere existence of a pending rollback |
| #126 adapter minima | Unsupported required-domain minima retained; new work refused by the selected semantics | Malformed registration/key; fresh-genesis unsupported minima separately fail `tools/genesis-check` |
| #127 IBC refunds | Count plus digest of legacy flows retained, no quota debit on refund; capacity retained until window reset | Occupied `0x03` generation prefix, malformed generation/flow/rule, pre-carrier charge identity, invalid amounts or overflow-prone window |

Pending rollback forecasts use the activation predicate, not the admission
check (which would reject the pending proposal itself). Valid scoped halts or
gridlock justify a valid target despite unknown evidence; otherwise the repair
cancels at activation. A zero activation block means unscheduled, not malformed.
The forecast is per record against current state, **not** future-state simulation
or reconstructed admission evidence; preceding BeginBlock work and later state
can change the outcome. Tests compare forecasts with the real keeper. No census
rewrites history, redistributes funds or supplies scheduling authority.

The global digest hashes sorted, length-framed `(store, key, value)` bytes.
Each named census hashes only the selected rows above in the same deterministic
store/key order and framing. Unrelated rows change the global digest but not
that census's digest. These are
state-integrity commitments, not scientific truth or proof of payment.

## Genesis and faults

`validateClaimIntegrityGenesisSelection` chains the new selection check before
InitGenesis writes. Every registry field must be explicitly true; absence,
false, null, case-variant names and camelCase aliases refuse, even alongside a
canonical true field. A static selection list and marker-location test pin the
contract to the actual registry. After InitGenesis, initializeAccountingAuthority
reads every marker back before sealing lineage and fails InitChain if any is
off or malformed. Pre-carrier exports may omit false fields (`omitempty`);
absent or false is rejected. Export/import carries the selection without
manufacturing applied history. The genesis checker uses the same ten-marker
validator and refuses required-domain adapters with DISTINGUISHED or unknown
minima under either snake_case or camelCase spelling. Both spellings together
are rejected. UNSPECIFIED, PROBATIONARY and ACTIVE are the supported floors;
an open adapter has no qualification floor.

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

    <output>/consensus-repairs-census --copied-db --home <absolute-copy-home> --chain-id zerone-dev-1 --expected-height <committed-height> --expected-app-hash <lowercase-64-hex>

Only copied `data/application.db` is opened, with backend-enforced read-only
GoLevelDB options and fast-node rebuilding disabled. No key, configuration or
genesis file is read. Symlink path components and protected default/private home
names are refused. GoLevelDB only is supported in this draft; not Pebble or a
JSON genesis export. The copy must already include a regular, non-symlink
`data/application.db/LOCK`: GoLevelDB otherwise creates it even with ReadOnly.
The tool refuses a missing LOCK before opening, rather than creating one.
Do not point it at a running or only database.

Exit 0: completed census with no blocker. Exit 2: JSON report with blockers.
Exit 1: invalid arguments, identity mismatch, open/load/close/output fault.
The chain ID is explicitly an assertion supplied by the operator, not something
an application DB proves. The expected AppHash is the one committed at the
supplied height (`last_block_app_hash`), not the pre-state hash in that height's
block header. Both LastCommitID and WorkingHash must match. WorkingHash binds
the loaded per-store roots to the AppHash; it is not a full Merkle
re-verification of the stored tree or each leaf value read by the census.
A copy for activation is at most H-1; at/after H its markers block.
The offline census uses committed height, while the receipt uses H. Do not
compare their JSON byte-for-byte: height differs, and StartBlock,
PhaseStartedAtBlock or rollback activation at a boundary can change findings.
The report includes the stored module version map. It can run before
claim-integrity, but a successful census alone does not satisfy the handler's
exact predecessor/plan checks. Tests compare every copied source file before
and after both passing and failing scans.

## Rehearsal still required — separate lane

For **each** census collect selected record/byte counts per store and prefix,
plus whole-store/first-byte counts (especially alignment `03`–`05` and SDK
staking historical info). Record handler peak memory and wall time at H.
Supply these real values to Yu, not extrapolations from the synthetic tests:

- #117: seat/runoff/counter inventory and any failed relationships.
- #118: NUL bounty and augmentation/reference counts, index collisions/faults,
  invalid-vote/top-tie holds.
- #119: history counts and digest, including multi-sample count.
- #120: positive-amount EXPIRED rows and total against qualification escrow.
- #121: module bank balance against pools plus refundable active stakes.
- #122: registry, active count/stake, mint ledger, bank supply and both staking
  ratios; also latest health category and composite score under each selection.
- #124: the complete status histogram, with only exact `active` counted active.
- #125: each pending LIP, activation block, due-now state and predicted outcome.
- #126: unsupported required-domain minima and malformed registrations.
- #127: flow/rule/generation counts and digest, retained capacity, and measured
  transaction gas in the separate rehearsal.

Anything the copy finds goes to Yu with actual numbers. This source lane has
not acquired a copy or supplied real dev-1 timings, balances or distributions.

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

- Obtain independent review; resolve Q1 in its owner if the measured condition blocks.
- Rehearse the dev-1 copy; re-run the census at H−k and cancel before H if it blocks.
- Ask Yu for both claim-integrity and carrier heights together only after that.
- If #120 finds positive ambiguous EXPIRED amounts, attach the census and obtain
  Yu's word on the hold. Empty findings require no hold decision. Any #121
  coverage or #127 capacity-retention question is conditional on real findings.
- Nothing here authorizes zerone-1 evolution/relaunch, scheduling, signing,
  broadcasting, deployment, publishing a ready PR or merging.
