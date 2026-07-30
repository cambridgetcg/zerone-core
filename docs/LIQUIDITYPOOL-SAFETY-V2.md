# Liquiditypool safety v2 — readiness and operator runbook

> **Status: source-approved, not network-activated.** No upgrade height is
> selected by this document. Publishing a binary that knows this upgrade name
> does not authorize native pool creation, native oracle use, a validator
> rollout, or external mainnet liquidity.

`liquiditypool-safety-v2` is the named post-consolidation readiness checkpoint
for `x/liquiditypool` consensus version 4. It is not the technical switch for
v4 semantics: the first scheduled handler whose binary reports consensus v4
and calls `RunMigrations` activates them (normally the already-pending
`consolidation-safety-v1` at H1). Both names must still be scheduled through
Cosmos governance at distinct heights, in that order.

## Why the sequence has two names

Every Zerone upgrade handler calls `ModuleManager.RunMigrations`. Therefore the
earlier `consolidation-safety-v1` handler can advance liquiditypool from v3 to
v4 when validators install a binary containing both releases. The later
`liquiditypool-safety-v2` handler is deliberately safe in that already-active
state:
`RunMigrations` skips a module already at v4, module-account permissions are
reconciled again, and the dedicated
`upgrade_marker_liquiditypool-safety-v2=migrated` boundary is recorded.

H1 therefore activates the v4 code and state transition. It does **not**
authorize opening a native market. The later readiness checkpoint, release
gates below, and subsequent governance messages remain required.

### Mandatory live-state preflight

The v3→v4 migration handles legacy pools as follows:

- a structurally valid pool with positive reserves and positive LP supply
  becomes `EXIT_ONLY`, preserving withdrawals without silently enabling
  trading or deposits;
- an exact zero-reserve, zero-LP empty pool becomes an immutable `CLOSED`
  tombstone at the upgrade height, with its pair index and TWAP retired; and
- a partial-zero, malformed, locked, duplicate-pair, mismatched-denom,
  mismatched-ID/LP-denom, or otherwise inconsistent pool aborts the migration.

The migration is fail-closed even if state changes after a census: a positive
legacy pool cannot trade until governance explicitly activates it. Operators
must nevertheless prove at a bound pre-upgrade height that the existing
network has:

1. zero native liquidity pools; and
2. an empty billing quote-denom allowlist.

The two creation allowlists do not exist in v3 state. Immediately after the
first v4 migration pass, verify that `allowed_pool_denoms` and `pool_creators`
were initialized empty before accepting any transaction that could change
liquiditypool params.

Record the chain ID, height, app hash, query responses, release commit, and
binary digest in the signed release packet. If any pool exists, or any quote
denom is allowlisted, the combined two-upgrade sequence is **NO-GO** until a
separate reviewed transition accounts for it. A valid legacy pool migrates
`EXIT_ONLY`; that quarantine is a safety net, not permission to skip the
census or bank/state audit.

## Activation gates

Native pool creation and oracle allowlisting remain forbidden until every gate
is green:

- `consolidation-safety-v1` is applied and its post-upgrade state is verified;
- `liquiditypool-safety-v2` is later applied at a distinct height;
- the on-chain module version is liquiditypool v4 and the named handler marker
  is present;
- the deterministic migration bank/state audit passes, and the same
  state-consistency function passes against the rehearsed migrated state
  (the current app does not ship x/crisis);
- liquiditypool keeper and application lifecycle tests pass on the exact
  release commit, including migration failure fixtures;
- an export/import and cross-height restart rehearsal passes with matching app
  hashes;
- `max_pools` is a nonzero finite value; the reviewed v4 default is 16 and the
  consensus validation ceiling is 64. Migration maps legacy zero/unlimited to
  16 and clamps a larger legacy policy value to 64 after separately rejecting
  more than 64 actual open pools;
- `allowed_pool_denoms` and `pool_creators` remain empty until governance
  deliberately admits the first pair and its initial-liquidity funder;
- bank send controls are queried for both `uzrn` and the proposed
  counter-denom; creation remains a NO-GO while either denom is disabled;
- fee inputs and outputs are checked as strict integer parts-per-million;
- the first-pool release packet identifies its capital owner, assets, initial price,
  fee, slippage assumptions, and exit rights; and
- the oracle remains disabled until a separate governance decision allowlists
  a quote denom backed by an `ACTIVE`, adequately funded pool. Before that
  decision, the release packet must bind the quote denom's base-unit exponent,
  the consumer conversion rule, and a decimal-normalized per-denom oracle TVL
  floor. The global `min_reserve` is only a raw base-unit execution floor and
  is not an oracle-liquidity policy.

A minimum focused source gate is:

```bash
go test ./x/liquiditypool/... ./tests/cross_stack/... ./app/...
go test -race ./x/liquiditypool/keeper ./tests/cross_stack
```

The signed release packet must identify the exact commands and results. A green
developer checkout is evidence, not deployment authority.

## Pool status and lifecycle

Version 4 has an explicit governance-controlled state machine:

| Status | Swap | Add liquidity | Remove liquidity | Meaning |
|---|---:|---:|---:|---|
| `ACTIVE` | yes | yes | yes | Fully operating pool; eligible for an allowlisted oracle quote. |
| `SWAPS_PAUSED` | no | yes | yes | Trading stopped while LP maintenance and exits remain possible. |
| `EXIT_ONLY` | no | no | yes | Wind-down state; no new capital or trading. |
| `CLOSED` | no | no | no | Final immutable tombstone produced by the last LP exit. |
| `UNSPECIFIED` | no | no | no | Invalid/fail-closed value, never an activation shortcut. |

The final withdrawal of the complete LP supply:

1. returns the remaining proportional reserves;
2. leaves zero recorded reserves and zero LP supply;
3. sets `CLOSED` and records `closed_at_block`;
4. removes the active pair index and live TWAP accumulator immediately, then
   drains retained checkpoint keys through bounded BeginBlock garbage
   collection; and
5. permanently refuses swap, add, remove, or status-based reopening.

Closing preserves historical pool identity without leaving a re-seedable empty
pool. A later governance-approved pool for the same pair must receive a fresh
pool ID and fresh LP denom.

`MsgSetPoolStatus` is governance-authorized. Operational pauses should preserve
the least-dangerous available exit:

- use `SWAPS_PAUSED` when trading must stop but liquidity maintenance is safe;
- use `EXIT_ONLY` for an orderly wind-down; and
- never treat `CLOSED` as a reversible pause.

## Finite pool growth and strict PPM math

Version 4 removes the old “zero means unlimited” pool-cap posture. The reviewed
default is 16 open pools and consensus validation rejects values outside
1–64. Closed tombstones do not become live pools again. Raising the setting
above the release default requires a deliberate governance decision; no
parameter update can restore an unbounded active set.

The monotonic pool-ID namespace and immutable tombstone registry have a
separate hard lifetime cap of 10,000 records in v4. Gaps count as consumed and
IDs are never reused. This bounds export, audit, query, and cleanup work; it is
not a substitute for admission discipline or a promise of indefinite
recreation. A future version may compact old tombstones into a cumulative
commitment before deliberately extending that boundary.

Liquiditypool percentage values use an integer denominator of `1,000,000`:

| PPM | Percentage |
|---:|---:|
| `3,000` | 0.3% |
| `100,000` | 10% |
| `450,000` | 45% |
| `1,000,000` | 100% |

Legacy protobuf fields containing `_bps` retain their wire identity for
compatibility, but their unit is PPM. Operators, SDKs, proposals, events, and
reviews must not divide them by 10,000 or describe them as conventional basis
points. Values above the applicable PPM ceiling must be rejected; no floating
point arithmetic belongs in proposal construction or consensus checks.

## Governance path

### 1. Apply the pending consolidation boundary

Submit and pass `/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade` with:

```json
{
  "@type": "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade",
  "authority": "<governance-module-address>",
  "plan": {
    "name": "consolidation-safety-v1",
    "height": "<H1>",
    "info": "<signed-release-packet reference and checksums>"
  }
}
```

All validators stage the exact approved binary, halt at `H1`, migrate, restart,
and verify the applied plan. Do not create a pool or allowlist an oracle quote
denom after this step.

### 2. Apply the liquidity readiness checkpoint

After the H1 post-upgrade checks and a later governance vote, schedule:

```json
{
  "@type": "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade",
  "authority": "<governance-module-address>",
  "plan": {
    "name": "liquiditypool-safety-v2",
    "height": "<H2 greater than H1>",
    "info": "<signed-release-packet reference and checksums>"
  }
}
```

At H2, verify:

- `zeroned query upgrade applied liquiditypool-safety-v2` reports H2;
- validators agree on app hash and continue producing blocks;
- liquiditypool remains at consensus v4, whether H1 or H2 first advanced it;
- the dedicated handler marker reads `migrated`; and
- module-account permissions are healthy and the deterministic liquiditypool
  bank/state audit passes on the migrated rehearsal/live snapshot.

### 3. Governance admits a pair and its funder

The v4 migration leaves `allowed_pool_denoms` and `pool_creators` empty, so
pool creation remains fail-closed even if a binary is installed early.
After every activation gate passes, governance may use
`/zerone.liquiditypool.v1.MsgUpdateParams` to admit:

- one creation grant for the proposed counter-denom in
  `allowed_pool_denoms`; and
- one or more designated initial-liquidity accounts in `pool_creators`.

`MsgUpdateParams` replaces the complete params object. Proposal construction
must preserve every unrelated reviewed value, including the finite pool cap,
PPM fees, reserve floors, TWAP bound, and still-empty billing quote allowlist.
Asset admission does not itself turn on the oracle.
The counter-denom grant is consumed on successful creation. Replacing a later
closed pair therefore requires a fresh governance re-admission; an admitted
creator cannot churn the permanent pool-ID namespace unilaterally.

Liquiditypool admission also does not override `x/bank` send controls.
Governance must query both denoms and, where necessary, separately execute
`/cosmos.bank.v1beta1.MsgSetSendEnabled` for `uzrn` and the admitted
counter-denom before pool creation. Treat that bank change as an explicit
activation action in the release packet; do not infer sendability from a
successful mint, IBC receipt, balance query, or liquiditypool params update.

Governance approval and capital ownership are deliberately separate. The
governance module account is not the pool funder and must not be debited for
initial liquidity.

### 4. The admitted creator funds and creates the pool

An admitted account then signs and submits
`/zerone.liquiditypool.v1.MsgCreatePool` as `creator`, funding both amounts
from its own bank balance. The legacy `swap_fee_bps` request field must be
zero: v4 always uses the governance-controlled default, so a creator cannot
select a private fee at creation.

A new pool starts `ACTIVE`; the create transaction is the activation act, not
a harmless setup step. The preceding governance decision and public release
packet must therefore identify the capital owner, admitted assets, initial
price, governed fee, slippage assumptions, and exit rights.

Subsequent operational state changes use a governance proposal containing
`/zerone.liquiditypool.v1.MsgSetPoolStatus`, for example:

```json
{
  "@type": "/zerone.liquiditypool.v1.MsgSetPoolStatus",
  "authority": "<governance-module-address>",
  "pool_id": "pool-<n>",
  "status": "POOL_STATUS_EXIT_ONLY"
}
```

Parameter changes use the governance-authorized liquiditypool
`MsgUpdateParams`. Quote-denom allowlisting is a separate oracle activation
decision and should follow, not accompany, the first pool creation unless the
release packet explicitly reviews both actions together. `GetZRNPrice` reports
a raw base-unit reserve ratio scaled by 1,000,000; it does not infer display
decimals. Do not allowlist a quote denom until its exponent, conversion into
the billing unit, and decimal-normalized minimum oracle TVL are specified and
enforced for that denom.

## External Osmosis testnet liquidity is separate

The existing `osmo-test-5` ZRN/OSMO proof-of-concept pool is external testnet
state. It:

- does not activate `x/liquiditypool`;
- is not controlled by native `PoolStatus`;
- does not satisfy the native v4 upgrade or invariant gates;
- does not feed Zerone's native billing oracle; and
- does not authorize Osmosis mainnet liquidity.

IBC testnet transfers and the external pool may continue as bounded plumbing
tests under their own rate limits and disclosures. Never use their presence,
price, depth, or successful swaps as evidence that the native pool is safe to
open.

See [ZRN external liquidity](tokenomics/LIQUIDITY-TRANSPARENCY.md) for the
separate testnet position and [UPGRADES.md](UPGRADES.md) for the generic
release-packet procedure.
