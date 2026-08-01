# Founder Renunciation v1

> **Status: SOURCE CANDIDATE / RELEASE NO-GO.** No upgrade height, release
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

Immediately after `LoadLatestVersion`, before returning an ABCI app, the
binary performs an uncached, read-only composite H1→H2 proof. It accepts only:

1. an empty height-zero bootstrap with no lineage, done, plan, H2 plan-identity
   digest, Params, or module-account evidence;
2. exact pending H2: the full K6/P2/L5/V1 map, H1 migrated marker and positive
   completed height, no native markers, H2 marker and plan-identity digest truly
   absent/done zero,
   strict persisted V1 Params that can migrate to valid V2, an absent
   vesting module account or its exact H1 `Minter`/`Burner` permissions, and
   byte-identical canonical committed/local H2 plans at `latest+1`;
3. exact completed H2: the full K6/P2/L5/V2 map, both migrated markers, a
   well-formed consensus-committed SHA-256 of the complete canonical H2 plan
   identity, `0 < H1 done < H2 done <= latest`, valid zeroed V2 Params, and a
   vesting module account that is either absent or, if present, has no
   permissions; or
4. direct H2 genesis: the full target map, both native markers exactly
   `genesis`, both migration markers and the H2 plan-identity digest absent,
   both done heights zero, strict persisted V2 Params, and an absent or
   permissionless vesting module account.

Every VersionMap comparison is full-map equality. Missing, unknown, partial,
intermediate, stale, conflicting, skipped, malformed, unreadable, plan-less,
or noncanonical evidence refuses process startup. An old H1 prestate cannot be
run by H2; H1 must first complete under its independently accepted binary.

At H, the handler re-reads and re-proves the live H1/H2 markers and done
heights, full V1 VersionMap, exact passed/committed/local plan identity,
canonical `Plan.Info`, strict migratable Params, and absence of unsafe skips
before any state mutation. It then runs only vesting_rewards V1→V2, asserts the
full target VersionMap and zero/valid Params, reconciles only the existing
vesting module account to an empty permission set (without lazy creation or
unrelated account rewrites), re-proves the H1 marker plus H2 completion/digest
absence, writes the append-only SHA-256 plan-identity marker, and writes the
generic H2 migrated marker last as the handler completion seal. The SDK subsequently persists
the VersionMap, clears the plan, and writes the H2 done height in the same
cached PreBlock transaction; any error rolls the whole block back.

For pending and historical H2 packets, `Plan.Info` must be a non-empty public
JSON object no larger than 4,096 bytes, compactly encoded with sorted keys,
canonical integers, no duplicate keys, and no trailing value. At execution the
handler hashes deterministic fixed-field JSON containing the exact plan name,
height, and canonical `Info`; a retained historical disk packet must recompute
to that consensus digest. The digest does not by itself bind a source commit or
executable digest.

## Height-bound executable semantics

Cosmos SDK runs x/upgrade in PreBlock. Therefore fees already collected in
block H−1 are routed by vesting_rewards V2 BeginBlock at H, after the H2
migration has cleared the founder tap. The complete research slice reaches
`research_fund`; the founder receives zero; no transaction-presence block
reward is minted. Persisted restart tests prove exact fee conservation and
unchanged total supply across this boundary.

There is deliberately no phase facade around non-consensus service surfaces.
Once an operator starts the H2 process at the halted H−1 state, queries and
CheckTx use H2 code even though committed module metadata and Params remain V1
until PreBlock at H. For example, the legacy founder-status query reports
`active=false` while still exposing the stored legacy fields, coupling audit
reports disabled/zero effective coupling, and a parameter update cannot clear
or restore legacy retired fields outside the named migration. These
query/CheckTx differences do not mutate consensus state and are not evidence
that H2 completed; only the composite completion marker/plan digest/done/
VersionMap/Params/permission poststate is.

## Source provenance

This replacement line descends from the independently accepted H1 commit
`65c19cd8b00bdfff9b80705b776fd0d49719398a`. The former H2 commit
`4bffb6d218819bed1c29c7a0be7779ad31c64a97` and the superseded candidate
`c0943ea91a4cc86e6b232b7675c7991795fd5d30` are explicitly rejected
provenance. The first exposed V2 execution without complete H1/H2 activation
evidence; the second did not strictly prove native V2 genesis before lineage
markers and did not retain completed `Plan.Info` identity. Neither may be
tagged, released, deployed, scheduled, or treated as accepted H2 source. This
additive descendant preserves `c0943ea` unchanged as its parent for audit
provenance. This document still authorizes no upgrade height or activation; the
new candidate commit and reproducible binary digests require a separate
independent audit and release ceremony.

## Historical genesis compatibility

The hash-bound zerone-1 and public-testnet genesis artifacts retain their
historical v1 founder fields and must not be rewritten. The H2 application's V2
`GenesisState.Validate` rejects those values, and native H2 `InitChain` strictly
re-proves the stored zeroed Params and account permissions before writing
lineage markers. Existing `genesis-check` profiles retain their separately
named historical/release behavior and are not evidence that a legacy artifact
is valid as native H2 genesis. Historical validation, replay, or state
reconstruction must use the exact v1 release binary appropriate to the
artifact; an existing network crosses the boundary only through the named
state migration above.

Publishing, merging, or deploying the static dashboard does not satisfy these
requirements and must not be described as a chain activation.

## Exact claim boundary

After a verified activation, Zerone may say that v2 has no founder-specific
revenue recipient and ordinary parameter governance cannot restore one.

It may not say that founders can never benefit under general permissionless
rules, that a future coordinated software upgrade is metaphysically
impossible, or that validator/operator/governance control is decentralised.
This upgrade does not retire ordinary validator/delegator economics or
concentrated validator and upgrade control, the bootstrap registrar's bounded
admission discretion, other default-disabled governance issuance paths, or the
identity-bound research-voter and disbursement surface. The current trust
disclosure remains authoritative for operational control; each excluded
surface requires its own evidence and, if retired, its own named boundary.
