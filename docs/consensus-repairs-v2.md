# consensus-repairs-v2

Status: source implementation and local synthetic tests, not deployment or scheduling authority.
This successor is built on the reviewed v2 stack at `4571be0f` and follows the
`consensus-repairs-v1` carrier. Both the carrier/code and its separate authority
geometry refresh must travel together. No height is selected by this document.

## Activation contract

`consensus-repairs-v2` is a height-only plan with empty Info and no deprecated
fields. Its exact source is v1's complete target map. Only alignment 2→3 and
knowledge 13→14 change. Alignment owns Q1's stored category, observation cadence,
threshold and event changes. Knowledge owns Q2 round accounting, Q5 calibration
and snapshots, and PR37 confidence floors/growth. FX7 and FX8 only serialize
retained genesis state; governance stays version 4 and neither receives a marker.
Downstream governance consumes alignment's stored category, not a new gov code path.

The handler accepts `zerone-dev-1` or an isolated `zerone-rehearsal-` lineage. It
requires exact supplied/stored/compiled maps, completed H3/accounting/claim/v1
lineage, absent v2 receipt/done height and all four v2 keys absent. Present-empty,
malformed and already-enabled keys refuse. It requires the exact scheduled plan,
no unsafe skip, and committed H−1 plus SDK/header H. Startup admits the source
only at its matching local/on-chain H−1 boundary. Target restart validates all
four marker readbacks and each receipt identity, digest and bounded summary.
Native/imported selection has no invented upgrade receipt or done height.

V1's handler, source and target maps remain frozen. This binary cannot execute
v1: the v1 binary must first complete its own rung, and this binary starts at the
separately scheduled v2 boundary. Handler registration is not replay capability.

The two migrations are guarded no-ops with a v2-specific context token. V1's token
cannot authorize them. The handler first computes its expected write bytes from
the read-only census, then uses one SDK cache for migrations, Q5 score refresh,
all four `ConsensusRepairsV2Markers()` enables/readbacks and the receipt. It
compares a framed, ordered SHA-256 stream of every store against the source
merged with exactly those expected writes, and separately compares knowledge
0x87. Only then are state and events committed. SDK upgrade bookkeeping follows
handler success. No module-account permission reconciliation runs.

Allowed carrier writes:

- alignment 0x09; knowledge 0x91, 0x92, 0x93, each `{1}`;
- changed knowledge 0x5a calibration scores, derived by
  `ComputeSmoothedAgentCalibrationScore`; no counts, timestamps or per-method
  data change, and the legacy permissive recompute helper is not used;
- knowledge 0x7f01 receipt `upgrade_receipt_consensus-repairs-v2`.

The calibration refresh and 0x92 are in the SAME cache. Existing fact snapshots,
confidence, provenance, round histories, balances and all v1/claim receipts are
unchanged. The permanent receipt holds six fixed-shape summaries (counts,
digests and held-finding counts), never per-record inventories.

## Censuses: held history is not a payment claim

Each inventory has a 100,000 selected-row / 64 MiB selected-byte ceiling. Exceeding
it refuses without truncation. Raw bytes are charged before decoding. Only rows
in the declared read manifest are retained; unrelated history is streamed.
There is deliberately no global state-size ceiling. The handler performs three
whole-state streaming passes (census, expected image, actual image); startup adds
another census. These scans bound retained selected inputs, not total I/O time.

| Census | Declared inputs | Meaning |
| --- | --- | --- |
| Q1 health | alignment params/state; custom staking registry 0x01 | Occupied registry keys are observations, even at zero active stake. Malformed keys block, never masquerade as empty. No old category is rewritten. |
| Q2 rounds | knowledge claims 0x02, rounds 0x03, params 0x06, active index 0x21 | Policy, phase, verdict and unmatched commits are held findings; no old slash/reputation replay or inferred debt. Missing historical parents are reported, not reconstructed. |
| Q5 calibration | canonical knowledge 0x5a and facts 0x01 | Show stored/proposed score and genuine-zero status. Refresh only changed score bytes; preserve historical fact snapshots. |
| PR37 support | knowledge facts 0x01 and claim citations 0x02 | Separate resolved zero contribution, missing support and contradiction edges. An old stored floor of zero alone does not identify which case occurred. No historical rewrite. |
| FX7 accounting | knowledge bounties 0x62, escrow 0x6b, counters 0x35/0x36; exact training-fund bank balance | Strict raw inventory catches malformed/noncanonical/orphan escrow and underbacking before repaired export. Retain inactive/zero escrow and exact counter keys. |
| FX8 retained | gov LIPs 0x01, proposals 0x06, next ID 0x08, first voters 0x0e, community votes 0x0f, transitions 0x14 | Strict raw inventory and retained-record validation; zero/MaxUint64 counters refuse, absence stays absent. No replay of current eligibility, ballots or phase policy. |

FX7's repaired ExportGenesis can panic on underbacked escrow, orphan/noncanonical
bounties and counters that are not exactly eight bytes. Non-escrow training-fund
outflows make underbacking plausible; nothing here asserts a real balance.
FX8 export likewise fails closed on corrupt retained records or missing LIP
metadata. Run the raw census on the stopped copy BEFORE depending on a repaired
export. A successful focused census is not a certificate for unrelated exporters.
Empty inventories do not prove that a prior export preserved all history.

## Repair effects and explicit limits

Q1 persists `not_observed` at the next scheduled observation when the registered
validator keyspace is empty. Enabling the marker does not recompute past health.
Knowledge claim pacing and capture-defense analysis read the stored category
without reading this marker. `not_observed` maps to neutral 1x pacing; governance
knowledge-param voting periods are not halved for this category. The corrected
40–70% hysteresis band can also avoid a previously halved voting period. A first
degraded/critical observation now enables accelerated observation cadence.

`GlobalPacing` and `GetHealthCategory` expose the raw fourth string
`not_observed` to clients. The generated TypeScript health category is `string`,
not a three-value enum. This is a source compatibility observation, not a browser
or deployed-client test; external clients must not assume exactly three values.

Q1 registry iterator/Close failures now have the explicit halt class
`alignment/types.ErrHealthObservationFault`, preserving the underlying cause.
It is NOT a malformed marker: it is a fail-closed sensor-input I/O fault on a
scheduled observation, before effects. Marker errors continue to wrap
`activation.ErrMarkerFault` and propagate from entry points before any writes.
Neither error is logged-and-skipped. Marker-off legacy paths remain unchanged.

Q2 missed reveals on legacy-policy INCONCLUSIVE/EXPIRED rounds are prospective.
Policy-1 neutral review creates no monetary debt. The existing asymmetry where
decisive rounds skip missed-reveal slashes at nonpositive VerificationReward is
not changed. Retained all-false INCONCLUSIVE verification-history entries still
feed capture-defense correlation; that separate review issue is not repaired here.

Q5's genuine-zero snapshot requires `TotalSubmissions > 0`; challenger-only
records retain unknown/neutral fallback. Private legacy comparison adapters now
compile only in `_test.go`. PR37 creation clamps on resolved support even when
its contribution is zero and prevents zero-confidence verified/active facts
from gaining confidence merely with time. PROVISIONAL and the deferred PR37
cooling repair are unchanged. Existing zero floors are not retroactively decoded.

## Genesis and offline tooling

Native/imported genesis requires all ten v1 and four v2 selections explicitly
true, with exact snake_case spelling. Case-fold (including Unicode-fold) and
camelCase aliases refuse even beside canonical true. Static v2 selections are
pinned to registry order and store locations. `initializeAccountingAuthority`
reads every marker back after InitGenesis and before sealing lineage.

`genesis-check` exercises that shared selector and validates FX7 conservation /
imported bank backing and FX8 retained fields with their actual decoding rules.
FX7/FX8 stay presence-selected, with no markers. The key-only fork-genesis tool
preserves false or omitted selections rather than enabling them; its output then
fails current app selection until a separately authorized valid source is used.

Build and run offline (place binaries outside the checkout):

    GOFLAGS=-mod=readonly GOPROXY=off go build -p 4 -buildvcs=false -trimpath -o <output>/consensus-repairs-census ./tools/consensus-repairs-census
    <output>/consensus-repairs-census --repairs v2 --copied-db --home <absolute-stopped-copy> --chain-id zerone-dev-1 --expected-height <committed-height> --expected-app-hash <lowercase-64-hex>

The independent tuple is committed height and `last_block_app_hash`, NOT that
block header's pre-state hash. LastCommitID and WorkingHash bind loaded per-store
roots; this is not a full Merkle re-verification of each read leaf. The command
uses read-only GoLevelDB, requires an existing regular LOCK and refuses symlink
components/protected homes. Chain ID is an asserted label. Exit 0 means no census
blockers, 2 includes blockers, 1 is an argument/open/identity/output error. It
never imports, schedules, connects, writes source files or repairs state.

Dev-1 runtime/gateway tooling recognizes 13→14 and the changed sibling versions;
that code has only been tested offline. No deployment packet or binary digest is
published by this carrier document.

## What the real dev-1 copy must measure

No real chain copy or node was accessed for this source work. Before scheduling:

1. Use independently acquired stopped copies without live keys. First prove v1
   completion (including its own census); never weaken a v1 blocker to get v2 in.
2. Run the raw v2 census at a preactivation height and again at H−1. Record each
   selected prefix's rows/bytes, malformed records, held findings, FX7 backing,
   FX8 counter headroom, score changes and historical export-loss limits.
3. Measure startup and handler wall time, peak memory and full-store I/O on that
   actual size. The synthetic large-state probe is not a real sizing estimate.
4. Rehearse old binary stopping before H, v2 starting only at H−1, exact H diff,
   done/version/marker/receipt readback and H+1 restart. Compare nodes including
   amd64 through scheduled health observations, inconclusive/expired rounds and
   confidence-growth epochs. Record AppHash agreement and transaction gas.
5. Repaired export/import must preserve FX7/FX8 inventories and all selections,
   without invented receipts. On disposable copies inject malformed markers,
   underbacking and a late score/marker failure; prove refusal and atomicity.
6. Have independent code and geometry reviewers inspect the exact final head.
   Draft CI is skipped, not passed. Required completed CI remains an integration
   gate, not permission to mark this draft ready or merge it.

Yu decides scheduling after that evidence, including any held-history or
underbacking remediation. This work selects no height, signs nothing, touches
no existing keys/homes, and grants no H4/H5 authority or new coin-route permission.
