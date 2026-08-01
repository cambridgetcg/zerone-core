# Founder Renunciation v1

> **Status: SOURCE-COMPLETE / RELEASE NO-GO.** No upgrade height, release
> digest, validator quorum, rollback rehearsal, or live-state evidence is
> selected by this document. Do not run this source against an existing
> network except through the named `founder-renunciation-v1` plan after its
> release packet passes.

## Decision

Zerone retires status-derived founder revenue. Vesting-rewards consensus v2:

- fixes `founder_share_bps` at `0` and `founder_address` at empty;
- fixes the already-retired `block_reward`, `floor_reward`, and
  `empty_block_reward_rate` compatibility fields at zero;
- rejects restoration through parameter validation and the storage boundary;
- removes founder calculation and account-send branches from revenue routing;
- sends the complete research allocation to `research_fund`;
- keeps legacy protobuf field numbers, zero-valued outputs, query shape, event
  attribute, error codes, and historical reward records; and
- clears those legacy founder and automatic-reward Params during v1→v2
  migration, regardless of their prior values, while preserving every
  unrelated parameter.

The migration does not erase or claw back past transfers. V1 transferred a
configured share synchronously; it did not accrue a separate founder balance.
Claims about whether any historical transfer occurred require release-bound
chain evidence and are outside this source decision.

## Activation contract

The only intended network boundary is the exact upgrade name
`founder-renunciation-v1`. A release packet must bind at least:

1. the source commit and reproducible binary digests for every platform;
2. pre-upgrade module version `vesting_rewards=1` and live Params/history
   observations from independent nodes, with every other module already at
   the exact version targeted by the release binary;
3. the governance-selected height and matching upgrade-info payload;
4. migration, export, restart, rollback, and halt recovery rehearsals; and
5. post-upgrade queries proving module version `2`, share `0`, address empty,
   full research routing, and rejection of a restoration attempt.

The combined source binary refuses every other upgrade plan while the chain's
version map still reports `vesting_rewards=1`. This prevents the migration
from silently riding `consolidation-safety-v1` or a historical testnet name.
The earlier plan must use its exact reviewed binary targeting
vesting_rewards v1. The removed `liquiditypool-safety-v2` name has no handler.

The dedicated handler also refuses to run unless its prestate is exactly
`vesting_rewards=1`, its binary targets exactly v2, and every other module is
already current. It asserts the v2 poststate before writing its handler marker.
It therefore cannot mark an already-migrated chain or carry an unrelated
module migration under the founder-renunciation name.

## Historical genesis compatibility

The hash-bound zerone-1 and public-testnet genesis artifacts retain their
historical v1 founder fields and must not be rewritten. V2 validation and the
current genesis-check profiles intentionally reject those legacy values for a
new v2 genesis. Historical validation, replay, or state reconstruction must use
the exact v1 release binary appropriate to the artifact; an existing network
crosses the boundary only through the named state migration above.

Publishing, merging, or deploying the static dashboard does not satisfy these
requirements and must not be described as a chain activation.

## Exact claim boundary

After a verified activation, Zerone may say that v2 has no founder-specific
revenue recipient and ordinary parameter governance cannot restore one.

It may not say that founders can never benefit under general permissionless
rules, that a future coordinated software upgrade is metaphysically
impossible, or that validator/operator/governance control is decentralised.
The current trust disclosure remains authoritative for operational control.
