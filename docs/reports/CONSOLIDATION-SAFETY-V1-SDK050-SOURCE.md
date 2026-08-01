# Consolidation Safety H1 — SDK 0.50 source boundary

Status: **source only; activation NO-GO**

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
protocol skim. A plan-less restart against legacy state cannot silently gain
that behavior: the nonzero legacy field acts as an activation sentinel and the
swap and public simulation/quote paths fail closed until H1 has migrated it to
zero.

H1 does not contain or execute the founder-renunciation migration. The exact
H2 source is separately frozen at
`4bffb6d218819bed1c29c7a0be7779ad31c64a97`; H2 alone changes
`vesting_rewards` from `1 → 2`. The later SDK 0.53 / IBC-Go 10 H3 binary must
consume independent evidence of both boundaries.

## Evidence and limitations

Successful H1 execution produces the conjunction of:

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
