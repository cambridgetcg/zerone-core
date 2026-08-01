# Consolidation Safety H1 — SDK 0.50 source boundary

Status: **source only; activation NO-GO**

The earlier source snapshot
`1c0cee9cca80233bf72acc78ee310795db721317` was rejected by independent
review and is retained only as rejected provenance. It is not an acceptable
H1 source or binary identity. The additive replacement described here must
receive a new commit identity and pass a fresh independent audit.

This branch constructs the historical H1 binary boundary on Cosmos SDK
`v0.50.15` and IBC-Go `v8.8.0`. It is based on source commit
`315dafae4ec76952b3994d6740a1c2f18d3ec231`, the last reviewed point with
module targets `knowledge=6`, `claiming_pot=2`, `liquiditypool=4`, and
`vesting_rewards=1` before economic changes were combined in one later
commit.

## Exact state transition

Only the named `consolidation-safety-v1` plan may cross this boundary:

- `knowledge`: `5 → 6`
- `claiming_pot`: `1 → 2`
- `liquiditypool`: `3 → 5`
- `vesting_rewards`: `1 → 1` (explicitly unchanged)

Every other module-version entry must be present and already equal to this
binary's target. Unknown entries, missing entries, intermediate versions,
already-migrated bundle members, and unrelated catch-up work are refused.
Other named handlers cannot carry any H1 transition.

“Exact” above describes the module `VersionMap`; it does not claim that H1
writes only three modules' KV stores. After the migrations, H1 retains the
pre-existing permanent `ReconcileModuleAccountPerms` safety step used by every
named upgrade in this source line. It deterministically repairs an existing
x/auth module-account record only when its stored permissions differ from this
binary's declared `maccPerms`. That can change an unrelated module-account
record, but it neither changes module versions nor edits vesting parameters or
moves balances. Removing the step would reopen previously fixed permission
drift and make H1 weaker than its reviewed base.

Liquidity v5 retires `protocol_fee_bps` at zero. The direct `3 → 5` path runs
the lifecycle/index migration before the economic-neutrality migration, while
normalizing the retired field before v4 whole-state validation. Swap execution
keeps the complete fee in pool reserves for LP holders and cannot route a
protocol skim.

The zero field is defense in depth, not activation evidence: zero was legal in
legacy state and an old `MsgUpdateParams` could write it. Every liquidity
message (`CreatePool`, `Swap`, `AddLiquidity`, `RemoveLiquidity`,
`UpdateParams`, and `SetPoolStatus`) and both authoritative quote surfaces
(`CheckedSwapQuote` and gRPC `SimulateSwap`) require all of the following
before reading mutation-dependent pool state:

- an actual, decodable, consensus-valid Params record with
  `protocol_fee_bps=0`; and
- exactly one valid lineage:
  - migrated: H1 marker `migrated`, native marker absent, and
    `0 < H1 done height <= current height`; or
  - native: H1 marker truly absent, native marker
    `chain_lineage_native_consolidation-safety-v1=genesis`, H1 done height
    zero, and a positive current height.

Missing readers, missing Params, read failures, present-empty or forged
markers, both markers together, zero/future migrated done heights, nonzero
fees, and a direct `4 → 5` migration without global lineage evidence all fail
without pool, Params, TWAP, bank, lock, counter, or event mutation.

## Startup lineage wall

The v5 app-wide behavior is protected before ABCI service, not only at the
liquidity call sites. Immediately after `LoadLatestVersion`, and before
`NewZeroneApp` returns, an uncached read-only committed-state check accepts
only these deliberately separate states:

1. **Uninitialized bootstrap:** committed height zero, empty VersionMap, no
   lineage marker, no H1 done height, and no on-chain or disk plan. This is
   only the pre-`InitChain` construction state. `InitChainer` writes the
   separate native marker; an empty VersionMap is never treated as native or
   migrated lineage.
2. **Pending historical H1:** the exact complete pre-VersionMap, both lineage
   markers truly absent, H1 done height zero, no unsafe skip at H1, and an
   exact committed H1 plan at `latest+1`. Local `upgrade-info.json` must exist
   and match the committed name, height, and Info byte for byte.
3. **Completed historical H1:** the exact complete post-VersionMap, only the
   H1 marker with value `migrated`, and a positive done height no greater than
   latest. A retained disk halt packet is allowed only when it is canonical H1
   metadata for that exact completed height; conflicting or stale evidence is
   refused.
4. **Native v5 genesis:** the exact complete target VersionMap, only the native
   marker with value `genesis`, H1 done height zero, and no local upgrade-info
   packet. A native chain does not claim that H1 ran.

Every VersionMap comparison is full-map equality. Missing or unknown keys,
unrelated catch-up, intermediate versions, and partial pre/post mixtures are
refused. A stale legacy database with no exact halted H1 plan therefore cannot
start this binary at all; K6/P2/L5 behavior is unavailable because no app is
returned to serve ABCI.

For pending H1 only, `Plan.Info` is a public, non-secret JSON object of at most
4,096 bytes. It must contain at least one key and use compact, sorted-key
canonical encoding; duplicate keys, whitespace variants, trailing values,
and non-integer or non-canonical numbers are refused. This removes local versus
on-chain parsing ambiguity. It defines no release fields and does **not** bind
a source commit, executable, checksum, signer, or release packet; those remain
separate release-ceremony requirements.

H1 does not contain or execute the founder-renunciation migration. The former
H2 candidate `4bffb6d218819bed1c29c7a0be7779ad31c64a97` is rejected activation
NO-GO provenance: it could expose v2 behavior on a plan-less restart and is
neither frozen nor accepted release source. A replacement H2 descended from
the final audited H1 source remains pending and unbound; H2 alone may change
`vesting_rewards` from `1 → 2`. The later SDK 0.53 / IBC-Go 10 H3 binary must
consume independent evidence of both boundaries.

## Evidence and limitations

Successful historical H1 execution produces the conjunction of:

1. the exact poststate module map (`K6/P2/L5/V1` plus every other exact target),
2. the append-only knowledge-store marker
   `upgrade_marker_consolidation-safety-v1=migrated`, and
3. x/upgrade's positive completed height for `consolidation-safety-v1`.

The handler requires the marker key to be absent before any migration. A
forged value, an exact preseeded value, a historical present-but-empty value,
or an unreadable store all fail closed. Marker reads/writes propagate storage
errors and conflicting writes cannot overwrite first evidence.

This tuple is consensus state evidence, not a cryptographic attestation of a
source commit or executable. A later malicious binary could imitate names or
state writes. Before any activation decision, operators still need a
height-bound trusted state proof, reviewed reproducible build provenance that
binds the exact source commit and executable digest, signed ceremony artifacts,
isolated restore/migrate/restart rehearsal, rollback and fencing, custody
remediation, and a separately authorized governance height.

No tag, release, binary publication, upgrade schedule, validator deployment,
network halt, or chain activation is authorized or performed by this source
branch.
