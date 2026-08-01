# Liquiditypool v5 — atomic H1 readiness and operator runbook

> **Status: source target, not network-activated.** No upgrade height, pool,
> oracle quote, validator rollout, or external mainnet liquidity is authorised
> by this document. The filename is retained so existing links do not break;
> there is no longer a `liquiditypool-safety-v2` software-upgrade handler.

## One activation boundary

Existing networks activate this source line through one governance-scheduled
upgrade: `consolidation-safety-v1` at H1. Its exact release binary calls
`ModuleManager.RunMigrations`, reconciles stored module-account permissions,
and writes `upgrade_marker_consolidation-safety-v1=migrated`.

The post-H1 module-version map must include:

| Module | Pre-H1 | Post-H1 | Boundary |
|---|---:|---:|---|
| `liquiditypool` | 3 | 5 | v4 lifecycle/admission safety, then v5 zero protocol skim |
| `vesting_rewards` | 1 | 2 | zero founder tap and zero transaction-presence mint |

The H1 binary must not advertise or register a future H2 handler. Native pool
creation and oracle use are separate governance actions after H1; they are not
a second software-upgrade height.

## Economic-neutrality boundary

H1 changes prospective runtime economics without rewriting history:

- `protocol_fee_bps` becomes a retired compatibility field fixed at zero;
- a swap transfers no pool value to `fee_collector`; the complete input is
  recorded in reserves after output leaves, so the configured swap fee accrues
  pro rata to bearer LP shares;
- `founder_share_bps` and `founder_address` become retired zero/empty fields;
  every research deposit goes in full to `research_fund`;
- `block_reward`, `floor_reward`, and `empty_block_reward_rate` become retired
  zero fields; raw transaction presence no longer causes proposer issuance;
  and
- actual transaction fees still route through `fee_collector`. Retirement of
  automatic minting does not make real execution free.

This is the operative distinction: LP income compensates contributed capital
and pool risk; transaction-fee income compensates consensus/execution; neither
governance identity nor a named founder receives an automatic percentage. See
[Economic neutrality](tokenomics/ECONOMIC-NEUTRALITY.md) for the complete
rule.

The checked-in mainnet genesis remains an immutable historical artifact. It
truthfully records `protocol_fee_bps = 450000`,
`founder_share_bps = 70000` with no founder address, and the former block
reward settings at launch. H1 migrates live module state. Do not edit or
rehash historical genesis to make it resemble post-H1 state.

## Mandatory bound snapshot

Before governance schedules H1, take one same-height snapshot and bind it into
the signed release packet. Record at least:

1. chain ID, height, block ID, app hash, and applied/pending upgrade plans;
2. complete on-chain module-version map;
3. liquiditypool Params, every pool/TWAP record, every LP denom bank supply,
   module-account balances, and module-account permissions;
4. vesting_rewards Params, founder status, total bank supply, module balances,
   research/development balances, and the last stored reward-routing record;
5. `x/distribution` community tax and `fee_collector` balance;
6. bank SendEnabled state for `uzrn` and any proposed counter-denom; and
7. exact release commit, reproducible binary/image digest, SBOM/provenance,
   operator signatures, and recovery contacts.

The reviewed H1 assumption is zero native pools and an empty
`billing_quote_denoms` allowlist. If either differs, H1 is **NO-GO** until a
separate state-specific review accounts for every reserve, LP holder, oracle
consumer, and balance delta.

The v3→v4 migration remains fail-closed:

- a structurally valid positive-reserve/positive-LP pool becomes `EXIT_ONLY`;
- an exact zero-reserve/zero-LP pool becomes an immutable `CLOSED` tombstone;
  and
- malformed, partial-zero, locked, duplicate-pair, mismatched-denom,
  mismatched-ID/LP-denom, or otherwise inconsistent state aborts migration.

The v4→v5 migration changes the retired protocol-fee parameter only. It must
not change a pool, reserve, LP supply, bank balance, TWAP, index, or custody
liability. The vesting_rewards v1→v2 migration clears the retired founder and
automatic-reward fields without moving balances or erasing historical events.

## Release and recovery gates

H1 is ready to schedule only when all of these are green on the exact release
artifact:

- targeted keeper, app, integration, cross-stack, race, migration-failure,
  invalid-transaction, and export/import tests pass;
- a full halt → binary swap → migration → restart rehearsal produces matching
  app hashes on every validator;
- the rehearsal starts from the bound live snapshot, not a convenient default
  genesis;
- liquidity custody is solvent, LP bank supply equals recorded LP supply, and
  the post-migration delta is explained field by field;
- the H1 binary reports liquiditypool v5 and vesting_rewards v2, recognises H1,
  and does not recognise the removed H2 name;
- `allowed_pool_denoms`, `pool_creators`, and `billing_quote_denoms` remain
  empty after migration;
- the governance proposal's `plan.name`, height, `info` checksums, and signed
  packet all identify the same release; and
- rollback authority, validator fencing, backups, and incident communication
  have been rehearsed.

Before H1, governance may cancel the plan. At a failed halt, do not let one
operator improvise a divergent state. Restore from the bound pre-H1 snapshot
or restart the old binary with `--unsafe-skip-upgrades <H1>` only under an
explicit network-wide recovery decision. Never run old and new binaries at
the same height, and never restore a consensus signer into two live nodes.

## Post-H1 acceptance

Before accepting ordinary traffic, every validator must agree on height and
app hash, and operators must verify:

- `zeroned query upgrade applied consolidation-safety-v1` reports H1;
- `upgrade_marker_consolidation-safety-v1` reads `migrated`;
- module versions are liquiditypool 5 and vesting_rewards 2;
- liquidity `protocol_fee_bps` is zero;
- founder share/address and block/floor/empty-block reward fields are
  zero/empty;
- no unexpected bank, supply, reserve, research, development, or fee-collector
  delta occurred during migration;
- automatic transaction-presence issuance remains zero even when a block
  contains a stateless-valid transaction that fails execution; and
- pool, creator, and oracle admission remain empty.

Failure of any check is an incident, not permission to patch state manually.

## Pool property and lifecycle

Liquiditypool v5 retains the v4 lifecycle state machine:

| Status | Swap | Add liquidity | Remove liquidity | Meaning |
|---|---:|---:|---:|---|
| `ACTIVE` | yes | yes | yes | Operating pool; oracle-eligible only after separate quote admission |
| `SWAPS_PAUSED` | no | yes | yes | Trading paused; maintenance and exits remain |
| `EXIT_ONLY` | no | no | yes | Wind-down; no new capital or trading |
| `CLOSED` | no | no | no | Immutable tombstone after final LP exit |
| `UNSPECIFIED` | no | no | no | Invalid and fail-closed |

The creator funds both initial reserves and receives the initial LP supply as
a receipt for that capital. LP tokens are ordinary transferable bank denoms.
Any holder may redeem pro rata; the creator has no special withdrawal or
status authority. The final withdrawal returns the remaining reserves, burns
the final shares, records zero reserves/supply, closes the pool, and retires
its live pair/TWAP indexes.

Governance may pause swaps or move a pool to `EXIT_ONLY`, but it cannot disable
an open pool's LP exit. There is no governance or founder reserve-withdrawal
message.

## Finite bounds and fee units

Percentage-like liquidity fields use an integer denominator of 1,000,000 even
where the legacy protobuf name ends in `_bps`:

| Field/value | Meaning after H1 |
|---|---|
| `default_swap_fee_bps = 3000` | New-pool default: 0.3% |
| maximum per-pool fee `100000` | 10% safety ceiling |
| `protocol_fee_bps = 0` | Retired; nonzero Params are invalid |

The pool fee is part of constant-product output calculation and remains in
reserves for LPs. It is not a protocol tax. Events retain a zero
`protocol_fee` compatibility attribute.

Open pools are bounded to 1–64 by Params, with a reviewed default of 16. The
monotonic pool record namespace has a 10,000-record lifetime cap; IDs and
closed tombstones are not reused. TWAP retention remains bounded, and an
ordinary Params update cannot synchronously shrink its retained window.

## Separate governance admission after H1

H1 deliberately leaves pool creation frozen. A later standard Cosmos
governance proposal may submit a complete liquiditypool `MsgUpdateParams` to
admit one counter-denom grant and one or more initial capital providers. This
is a full Params replacement: proposal construction must preserve every other
reviewed value, especially zero protocol fee and an empty billing quote list.

Before admitting a pair, the governance record and signed capital packet must
bind:

- counter-denom identity, origin, base/display exponents, and bank SendEnabled
  state for both assets;
- capital owner, exact or bounded initial amounts, resulting price, LP
  ownership, fee, slippage assumptions, and exit rights;
- independent balance proof and a minimum economically meaningful depth;
- monitoring, pause, wind-down, and recovery contacts; and
- the fact that the admitted creator chooses the initial ratio and receives
  all initial LP shares for the reserves it actually contributes.

The counter-denom grant is consumed on successful creation. Removing creator
admission without replacing it with a pair-scoped launch mechanism would
create a first-transaction race and is not authorised by H1.

Oracle admission is another later governance decision. A full TWAP window
proves elapsed observation time, not external value or sufficient depth.
Before adding a `billing_quote_denom`, bind and enforce its exponent,
base-unit conversion, independent price reference or reviewed price band, and
decimal-normalised per-denom oracle TVL floor. The global one-base-unit
`min_reserve` is only an execution floor.

## External Osmosis testnet liquidity is separate

The `osmo-test-5` ZRN/OSMO proof-of-concept pool is external testnet state. It
does not activate this module, inherit native status/exit controls, satisfy H1
invariants, feed billing, or authorise Osmosis mainnet liquidity. Its price,
depth, and operator-owned LP position are evidence of plumbing only.

See [ZRN external liquidity](tokenomics/LIQUIDITY-TRANSPARENCY.md),
[Economic neutrality](tokenomics/ECONOMIC-NEUTRALITY.md), and the generic
[release-packet procedure](UPGRADES.md).
