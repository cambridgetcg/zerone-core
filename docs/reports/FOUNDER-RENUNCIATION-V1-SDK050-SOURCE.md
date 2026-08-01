# Founder Renunciation v1 — SDK 0.50 replacement source report

**Date:** 2026-08-01
**Status:** SOURCE CANDIDATE / ACTIVATION NO-GO
**Named boundary:** `founder-renunciation-v1` (H2)
**Verification toolchain:** Go 1.25.7 on darwin/arm64; module directive Go 1.24

## Provenance

This replacement is a direct, single-parent descendant of the independently
accepted H1 source commit:

`65c19cd8b00bdfff9b80705b776fd0d49719398a`

The former H2 commit below is explicitly rejected provenance:

`4bffb6d218819bed1c29c7a0be7779ad31c64a97`

That rejected source did not require the complete H1→H2 activation proof at
startup. The later candidate `c0943ea91a4cc86e6b232b7675c7991795fd5d30`
is also explicitly rejected: native genesis could write lineage markers
without strict stored V2 Params/account proof, and completed state did not
retain exact canonical `Plan.Info` identity. Neither rejected commit may be
tagged, released, deployed, scheduled, or treated as accepted H2 source. This
corrective source is an additive descendant of immutable `c0943ea`; the final
candidate commit is identified externally by the independent auditor after all
source is committed, because a commit cannot truthfully embed its own hash.

No source commit, report, binary, or dashboard publication selects an upgrade
height. This report authorizes no push, pull request, tag, release, deployment,
governance proposal, schedule, validator instruction, or activation.

`go mod tidy` under Go 1.25.7 made a classification-only manifest correction:
the already imported `cosmossdk.io/client/v2`, `github.com/cosmos/go-bip39`,
and `golang.org/x/text` modules are direct, while `cosmossdk.io/depinject` is
transitive. No module version or `go.sum` entry changed. This is mechanical
toolchain hygiene, not an H2 dependency or consensus change.

## Exact executable surface

H2 pins the complete module-manager target, not only the module that changes.
The canonical JSON has 40 entries and SHA-256:

`de3f0e0d9769adf2a7375f921d78f25365bc2f9a8b42d8c80de5982affa20127`

```json
{"alignment":1,"auth":5,"bank":4,"capability":1,"capture_challenge":1,"capture_defense":1,"claiming_pot":2,"consensus":1,"counterexamples":1,"creed":1,"distribution":3,"emergency":1,"evidence":1,"feegrant":2,"feeibc":2,"genutil":1,"gov":5,"home":1,"ibc":6,"ibcratelimit":1,"interchainaccounts":3,"knowledge":6,"liquiditypool":5,"qualification":1,"slashing":4,"sponsorship":1,"staking":5,"substrate_bridge":1,"tokens":1,"training_provenance":1,"transfer":5,"trust_score":1,"upgrade":2,"vesting":1,"vesting_rewards":2,"work_creed":1,"zerone_auth":1,"zerone_gov":2,"zerone_ontology":1,"zerone_staking":1}
```

The only H2 module migration is `vesting_rewards` v1→v2. Its exact pre-map is
the JSON above with `vesting_rewards=1`; every other entry must already equal
the target. The handler refuses missing, unknown, stale, intermediate, or
unrelated module migration entries. The H2 binary also refuses to execute H1:
H1 must finish under its separately accepted exact source.

## Composite startup proof

After `LoadLatestVersion`, startup performs uncached, read-only evidence reads
and accepts exactly four states:

1. **Bootstrap:** height zero; empty VersionMap; no lineage marker, done height,
   H2 plan-identity digest, plan, Params, or vesting module-account evidence.
2. **H2 pending after H1:** exact full H2 pre-map; exact H1 `migrated` marker;
   `0 < H1 done <= latest`; both native markers absent; H2 marker and
   plan-identity digest truly absent; H2 done zero; strict persisted V1 Params
   that become valid V2 Params by
   clearing only the retired fields; a never-created vesting account or exact
   H1 `Minter`/`Burner` permissions; and identical committed/local canonical H2
   plans at `latest+1`, with H1 completed earlier and neither height skipped.
3. **H2 completed after H1:** exact full target; both exact `migrated` markers;
   a well-formed consensus-committed SHA-256 of deterministic fixed-field JSON
   containing the H2 plan name, height, and canonical `Info`; no native markers;
   `0 < H1 done < H2 done <= latest`; valid V2 Params with founder and
   automatic-reward fields pinned to zero/empty; and an absent or permissionless
   vesting module account.
4. **Native H2 genesis:** exact full target; exact inherited H1 and H2 native
   `genesis` markers; no migration marker or H2 plan-identity digest; both done
   heights zero; valid strictly persisted V2 Params; an absent or permissionless
   vesting account; and no local historical upgrade packet.

Reads propagate store and disk errors and convert evidence-read panics into
startup refusal. A present empty marker is not absence. Pending H2 requires the
on-chain and local plan identities to match in name, height, and canonical
`Info`. Completed H2 may retain only a canonical historical local packet whose
complete identity recomputes to the consensus digest; the digest remains the
identity commitment if the disk packet is absent. Any live on-chain H1/H2 plan,
stale plan, conflicting packet, or unsafe skip refuses startup.

At native genesis the app validates non-nil V2 Params before module init, then
strictly reads the stored Params and proves an absent or exactly permissionless
`vesting_rewards` module account immediately after module init and before either
native lineage marker is written. Missing, corrupt, legacy, Minter/Burner, base
account, or wrong-name evidence returns an InitChain error with both markers
absent.

## Handler ordering and state scope

At height H the handler re-reads and re-proves the exact full pre-map, H1/H2
markers and done ordering, strict Params, existing-account permissions,
passed/committed/local plan identity, canonical `Info`, and skip configuration
before mutation. It then:

1. runs the sole vesting-rewards v1→v2 migration;
2. proves the exact full post-map and valid zeroed Params;
3. removes permissions from only an existing `vesting_rewards` module account,
   preserving address, account number, and sequence, without lazy creation or
   unrelated auth reconciliation;
4. re-reads the H1 marker and proves both the H2 plan-identity marker and generic
   migration marker are still absent;
5. writes the append-only lowercase SHA-256 plan-identity marker; and
6. writes the generic H2 migration marker last as the handler completion seal.

Cosmos SDK x/upgrade then persists the VersionMap, deletes the plan, and writes
the done height inside the same cached PreBlock transaction. Any handler error
rolls back migration, Params, permission, balance, plan, digest marker,
completion marker, event, and done-height mutations together. Error and panic
injection on the final completion-marker write prove the preceding digest write
and all earlier mutations disappear from the fresh committed restart root.

## Vesting-rewards v2 semantics

Consensus v2 permanently pins `founder_share_bps=0`,
`founder_address=""`, `block_reward="0"`, `floor_reward="0"`, and
`empty_block_reward_rate=0`. Validation and the ordinary storage boundary
reject restoration. The automatic block-reward path is structurally inert: it
does not mint, transfer, emit, or create new reward-history records. Historical
wire fields, query shapes, error-code identities, zero-valued event attributes,
and pre-v2 records remain compatible and truthful.

This boundary does not retire claiming-pot, substrate-bridge, configurable
knowledge, configurable token-emission, fee-routing, vesting, or general
permissionless earning surfaces. Each has its own authority and evidence
boundary. It also does not erase past transfers or change disclosed validator,
operator, or governance control.

## Height semantics and non-consensus surfaces

At H, x/upgrade runs in PreBlock before module BeginBlock. Fees accumulated at
H−1 therefore route under V2 at H: 3.33% reaches `research_fund`, 19.67% reaches
`development_fund`, and 77% remains for normal Cosmos distribution; founder
and automatic block reward are zero. The persisted restart test proves exact
conservation for 1,000,000 uzrn and unchanged supply.

There is intentionally no phase facade for query or CheckTx behavior. Once an
operator starts the H2 binary at halted H−1, those non-consensus services use
H2 code while committed metadata and Params remain V1 until PreBlock at H.
Only the composite VersionMap/marker/done/Params/permission poststate proves
activation.

## Source test evidence

The source suite includes:

- exhaustive pure-state matrices for all four startup states and every class
  of partial VersionMap, marker, done-order, Params, permission, plan, disk,
  canonical-Info, and unsafe-skip ambiguity;
- exact 40-entry target JSON and digest pin tests;
- real SDK `FinalizeBlock`/x/upgrade PreBlock execution at H followed by a
  fresh persisted restart;
- exact H−1 fee routing, balance conservation, unchanged supply, no founder
  payment, no automatic reward, and no vesting-account balance movement;
- proof that H2 removes only vesting permissions while deliberately injected
  unrelated module-account permission drift remains unchanged;
- failure injection after migration and permission reconciliation, plus error
  and panic injection exactly on the final H2 marker after the digest write,
  proving byte-identical cached rollback and fresh restart state;
- native real-InitChain refusal for nil/corrupt/legacy Params, strict post-init
  missing/corrupt Params, Minter/Burner, base-account, and wrong-name evidence,
  with valid absent/permissionless-account cases;
- absent, empty, malformed, forged, or packet-mismatched plan digests; canonical
  but different historical `Info`; corrupt/missing Params; present-empty marker;
  malformed done bytes; marker read errors; corrupt disk packet; and the valid
  completed state with an absent retained disk packet;
- vesting migration/immutability/unit tests, integration economics,
  simulation invariants, historical-handler boundary tests, protobuf/Swagger
  regeneration, TypeScript wire generation and consumer build, creed hashes,
  and recursive source-binding checks.

## Release decision

Source completion is not release readiness. Activation remains **NO-GO** until
an independent audit accepts the final local replacement commit and a separate
release packet binds reproducible binary digests, governance-selected height,
canonical plan `Info`, validator quorum, live H1 prestate observations,
backup/export evidence, halt/rollback rehearsal, and post-H2 queries. The
packet must continue to reject the old H2 provenance named above.
