# SDK 0.53 / IBC-Go 10 H3 source integration

Status: **H3 source-integration candidate at report freeze; activation NO-GO**

This report records the three separately named and reviewed source boundaries
required for the Zerone SDK 0.50 / IBC-Go 8 to SDK 0.53 / IBC-Go 10
transition. H1 and H2 are accepted source-only pins; H3 remains a local,
source-integrated candidate whose fresh final gates passed on 2026-08-01. The
exact H3 commit and tree are necessarily recorded in the final
handoff rather than self-pinned by this report. This is not a release manifest and authorizes no tag,
binary, schedule, deployment, halt, or activation.

## Source pins

| Boundary | Plan | Exact source commit | Transition | Audit status |
|---|---|---|---|---|
| H1 | `consolidation-safety-v1` | `65c19cd8b00bdfff9b80705b776fd0d49719398a` | `knowledge 5→6`, `claiming_pot 1→2`, `liquiditypool 3→5`; `vesting_rewards` remains `1` | **ACCEPTED for source integration; source only** |
| H2 | `founder-renunciation-v1` | `36728afbf71905a077a0863b41536fa9279109dd` (tree `dfeff2c71ca9c36896a3a76608600cd870d21a1f`) | `vesting_rewards 1→2` only | **ACCEPTED for source integration; source-only branch `release/founder-renunciation-v1-sdk050-replacement-source`** |
| H3 | `sdk-0.53-ibc-10` | **Containing local integration commit; exact commit/tree reported externally** | SDK/IBC migration after H1 and H2 | **Source gates passed; activation NO-GO** |

The H3 worktree was initially constructed from
`df4c65eb6fffbcf11d871a1f19c21dc2073d49f1`. That is historical working-base
provenance, not a release pin. Two complete, conflict-free checkpoint
transplants were performed on 2026-08-01:

1. `github/main` was freshly fetched at
   `02999d82aef34dc816308cd39ae86e7106f8c237`; the branch was fast-forwarded
   from its strict ancestor `32f4bb69ab7147cd5169fc313208aa850a8cbab2`, and
   the checkpoint was reapplied without conflicts.
2. After `github/main` advanced again, it was freshly fetched at
   `4f2a33effdd09b6e0e4baf98244f72457b505926`; the intermediate base
   `02999d82aef34dc816308cd39ae86e7106f8c237` was proved a strict ancestor,
   the branch was fast-forwarded, and the next complete checkpoint was
   reapplied without conflicts. Those five upstream commits touched only the
   dashboard, frontier-commons documentation/fixtures/tests, and repository
   hash manifests—not H3 integration paths.

At each transplant, the tracked binary diff and both untracked files retained
their recorded pre-transplant SHA-256 identities. Subsequent local test
hardening was applied only after the final transplant.

The rejected H1 snapshot
`1c0cee9cca80233bf72acc78ee310795db721317` and rejected H2 candidate
`4bffb6d218819bed1c29c7a0be7779ad31c64a97`, plus superseded H2 candidate
`c0943ea91a4cc86e6b232b7675c7991795fd5d30`, are retained only as explicit
negative provenance. None is an accepted source or binary identity.

## H3 state contract

H3 currently pins the complete 40-entry pre-H3 module VersionMap. Its compact
canonical JSON SHA-256 is:

```text
de3f0e0d9769adf2a7375f921d78f25365bc2f9a8b42d8c80de5982affa20127
```

Accepted H2 independently freezes the same 40-entry target and digest.
Every expected entry is negatively tested for absence and wrong version; H3
also refuses every additional entry, including target-only
`06-solomachine` and `07-tendermint`. A structural-parity audit against the
accepted H2 commit confirms its exported strict persisted-Params reader, the
five retired values (`founder_share_bps=0`, empty `founder_address`,
`block_reward="0"`, `floor_reward="0"`, and
`empty_block_reward_rate=0`), and absent-or-exact-empty
`vesting_rewards` module-account permissions.

Before any H3 handler mutation, the current integration requires all of the
following:

1. the exact complete VersionMap and pinned digest above;
2. `upgrade_marker_consolidation-safety-v1=migrated` and a positive immutable
   x/upgrade H1 done height, with the H1 native-genesis marker truly absent;
3. `upgrade_marker_founder-renunciation-v1=migrated` and a positive immutable
   x/upgrade H2 done height, with the H2 native-genesis marker truly absent;
4. a present `upgrade_plan_identity_founder-renunciation-v1` value containing
   exactly 64 lowercase hexadecimal characters;
5. strict activation ordering `H1 < H2 < H3`;
6. an exact persisted (never default-fallback) valid v2 `vesting_rewards`
   Params record with founder share/address and every automatic block-reward
   field zero/empty, plus an existing `vesting_rewards` module account named
   correctly with exactly empty permissions (account absence is accepted); and
7. both the generic migrated and native-genesis H3 markers truly absent, with
   the H3 x/upgrade done height exactly zero.

All absence decisions use presence-aware reads. A persisted marker key whose
value is empty is present evidence and is refused rather than being treated as
absence. Done heights that decode as negative, including uint64 overflow into
the signed height domain, are likewise refused.

The SDK 0.53 binary neither registers nor advertises H1 or H2 and installs no
store loader for either plan. Its older/current handlers refuse a VersionMap
that could still carry any H1 or H2 module transition. H3 retains its separate
legacy-root startup and destructive-loader gates.

The H2 handler writes the activation-specific plan-identity digest first and
the generic migrated marker as its final handler write; x/upgrade records the
done height only after successful handler completion. H3 consumes that digest
as observed state evidence. H3 source deliberately does not compile an expected
value because the exact H2 plan name/height/canonical `Plan.Info` tuple is fixed
only by a future activation packet.

Accepted H3 states remain disjoint:

- the pre-H3 handler/startup path accepts only the migrated H1→H2 lineage above;
  it accepts no H1/H2 native-genesis marker and requires all H3 completion and
  native-genesis evidence to be absent;
- post-H3 startup accepts only an exact generic H3 migrated marker, a truly
  absent H3 native-genesis marker, and a positive H3 done height no later than
  the latest committed height. It re-proves the retained exact H1/H2 migrated
  markers, truly absent H1/H2 native-genesis markers, lowercase H2 plan digest,
  strict `0 < H1 < H2 < H3 done <= latest` ordering, and the strict
  founder-renunciation zero poststate; or
- a chain born natively from H3 accepts only the separate
  `chain_lineage_native_sdk-0.53-ibc-10=genesis` state, with the generic H3
  marker truly absent and the H3 done height exactly zero. Every H1/H2 migrated,
  native-genesis, and plan-digest marker must be truly absent; both H1/H2 done
  heights must be exactly zero; and the strict founder-renunciation zero
  poststate must still hold. That native state has no H2 plan and therefore is
  not permitted to invent an H2 plan digest.

Native InitChain decodes and validates the authored `vesting_rewards` genesis
before module initialization, then re-runs the full native H3 partition proof
against stored state after all module initialization. The native H3 marker is
written only after that proof accepts exact H1/H2/H3 marker and done-height
absence plus the strict founder/account zero poststate; failed InitChain state
cannot seal the marker or the later per-store sentinels.

The activation-preflight report contract is `zerone.activation-preflight/v5`.
It emits the full source-map digest and explicit exact-map and ordered-lineage
checks, the observed H2 plan-identity digest, plus a checked persisted
founder-renunciation zero-poststate proof.
The exact-height handoff consumer pins the same digest and refuses a report
with a missing, empty, uppercase, or malformed H2 plan-identity digest before
any Fly invocation. Read-only Fly config validation/show and machine inventory
may precede the remaining v5 schema, source-map, and proof-check gates; every
proof gate must pass before the first Fly mutation.

## Evidence is not executable attestation

The VersionMap, migration markers, and x/upgrade done heights are consensus
state facts. Together they establish the named, ordered state boundaries that
H3 is willing to consume. They do **not** cryptographically attest which
source commit or executable produced those facts: another executable could
imitate the same writes.

Final release review must separately bind each accepted source commit to its
reproducibly built executable and image digests, signatures, build recipe,
height-specific plan, trusted H−1 state, isolated rehearsal, rollback/fencing
evidence, and independent operator authorization. This draft supplies none of
those release authorities.

## Fresh source-gate evidence

After the final transplant, `git ls-remote` independently reconfirmed
`github/main` at `4f2a33effdd09b6e0e4baf98244f72457b505926`. The following gates then
passed from the final H3 worktree:

- `go test -p 2 ./... -count=1`;
- fresh full `./app`, `./tests/cross_stack`, `./deploy`, and
  `./tools/operations-rehearsal` package suites;
- `go vet ./...`, `go mod verify`, and `go mod tidy` with no `go.mod` or
  `go.sum` drift;
- `gofmt`, `git diff --check`, `bash -n` on the exact-height Fly handoff, and
  the complete `make creed-check` hash suite; and
- independent tests from the clean accepted H2 worktree at
  `36728afbf71905a077a0863b41536fa9279109dd` for `./app`,
  `./x/vesting_rewards/keeper`, and `./tests/cross_stack`.

The H2 tests and direct source audit separately prove the exact 40-entry map
digest, strict exported persisted-Params reader, all five retired values,
absent-or-empty completed permissions, ordered migrated marker/done-height
facts, plan-identity digest ordering, and rollback of both digest and migration
state if the final H2 completion seal fails.

## Remaining NO-GO gates

- reproducibly build and bind each accepted source to executable and image
  digests, signatures, an SBOM, and a reviewed build recipe;
- review a concrete activation packet containing the exact plan, height, and
  canonical `Plan.Info`, then bind it to trusted H−1 app hash and snapshot
  evidence;
- complete an isolated activation/rollback/fencing rehearsal; and
- obtain independent operator authorization under reviewed key-custody and
  incident/halt procedures.

At report freeze, no tag, release, push, deployment, scheduling, or activation
action had occurred. A later source-only branch, pull request, or merge would
not authorize any tag, binary, deployment, schedule, halt, or activation.
Until every remaining release item is satisfied, activation remains **NO-GO**.
