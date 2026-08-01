# Upgrading a Live Zerone — the One-Page Runbook

> **Not live deployment authority.** Use this checklist only inside a
> network-specific, independently signed release packet. Never build moving
> `main` and install it into a default node home.

This is the concise planned-upgrade checklist. The canonical
[Upgrade and Incident Operations
runbook](UPGRADE_AND_INCIDENT_OPERATIONS.md) defines release authorization,
the four state axes, hostile-event recovery, and the meanings of *halt*,
*quarantine*, and *resume*. For the code-side migration recipe see
[UPGRADE_PROTOCOL.md](UPGRADE_PROTOCOL.md).

The old/new binary handoff was exercised on localnet on 2026-07-06. That
exercise is evidence for its recorded binaries and state only; every release
still requires its own artifact attestation, H−1 rehearsal, and operator
readiness.

## The whole model

A named upgrade handler in `app/upgrades.go` defines the deterministic state
transition. Governance schedules its name at height H. An old binary that
does not contain the handler reaches H, writes `data/upgrade-info.json`, and
stops its ABCI transition **before H commits**, so H−1 remains the last
committed state. The staged new binary starts from H−1, runs the handler at H,
and may commit H. If the running binary already contains the handler, it may
execute at H without a visible process stop.

This scheduled `x/upgrade` boundary is not an `x/emergency` halt ceremony.
`x/emergency` is an application transaction quarantine; it does not stop
block production. Neither mechanism authorizes automatic recovery after a
failure.

Standard SDK governance is the sole executable software-upgrade authority.
The custom Zerone `x/gov` upgrade category is retired: new submissions,
staking, stage advancement, and plan attachments fail closed; imported
non-terminal records are marked failed without calling `ScheduleUpgrade`.
Historical custom plan records remain queryable and require manual legacy
stake reconciliation because no per-staker escrow ledger exists.

## Carried-forward safety checkpoints

Existing networks have one pending consolidation boundary:
`consolidation-safety-v1` at H1. The exact H1 binary calls module migrations,
reconciles stored module-account permissions, and records its handler marker.
The resulting module-version map must include `knowledge=6`, `claiming_pot=2`,
`liquiditypool=5`, and `vesting_rewards=2`. There is no
`liquiditypool-safety-v2` H2 handler: advertising that redundant future name
early is unsafe under the SDK upgrade guard.

Before scheduling H1, bind a same-height live snapshot proving there are zero
native pools and no billing quote-denom allowlist. Valid legacy pools migrate
`EXIT_ONLY`, so a network with any existing pool still needs a separately
reviewed transition. After H1, verify `allowed_pool_denoms`, `pool_creators`,
and `billing_quote_denoms` are empty before accepting any Params update. The
complete no-go gates, economic-neutrality transition, status lifecycle,
governance path, and Osmosis-testnet separation are in
[LIQUIDITYPOOL-SAFETY-V2.md](LIQUIDITYPOOL-SAFETY-V2.md).

For a legacy SDK v0.50 / IBC-Go v8 network, `sdk-0.53-ibc-10` is a later,
guarded boundary. It refuses to run until the committed source map already
contains the four prerequisite versions above and the
`upgrade_marker_consolidation-safety-v1=migrated` marker. Its broad
`RunMigrations` call is therefore not an alternate path for economic or
consolidation activation. Do not improvise a combined or reordered sequence.
These are two frozen release binaries, not merely two handler names: build and
rehearse H1 from a reviewed pre-SDK-0.53 source lineage whose unrelated module
targets match the live v0.50/v8 state. The post-SDK integrated tip is not an H1
artifact for that legacy state; use it only for the later SDK/IBC boundary after
H1 is committed and independently verified.

## Operator steps

1. **Freeze and attest the release.** Add the new named handler (copy the
   `v1.0.3-testnet` block in
   `app/upgrades.go`: `RunMigrations` + `ReconcileModuleAccountPerms` + a
   marker), a lineage entry in `app/upgrade_registry.go`, a
   `RegisterStoreUpgrades` case if store keys change. Freeze the source
   commit, build reproducibly, record checksums and provenance, and rehearse
   an old-binary-to-new-binary handoff from a trusted database copy. A
   same-binary handler test is necessary but is not proof of that handoff.

2. **Schedule via standard SDK governance only after attestation.** Submit one
   canonical plan with enough lead time for voting, independent review,
   staging, H−1 backup, IBC preparation, and cancellation:

   ```json
   {"messages": [{"@type": "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade",
     "authority": "<gov module address>",
     "plan": {"name": "<handler name>", "height": "<H>", "info": "<handler-specific payload>"}}],
    "deposit": "10000000uzrn", "title": "...", "summary": "..."}
   ```

   `zeroned tx gov submit-proposal plan.json --from <key>` → vote → passes.
   Verify the exact name, H, and `info` with `zeroned query upgrade plan` and
   bind the query evidence to the signed release manifest.

   The `sdk-0.53-ibc-10` handler is stricter: its `info` is not free-form
   checksum text. It must be the canonical legacy-IBC keyset commitment
   documented in [OPEN_CRYPTO_SDK.md](standards/OPEN_CRYPTO_SDK.md), produced
   by the read-only
   [`ibc-v10-keyset-manifest`](../tools/ibc-v10-keyset-manifest/README.md)
   tool from a trusted frozen-state raw validator database snapshot used for
   the migration rehearsal. Preserve its height and app-hash evidence and keep
   IBC state frozen through activation. A normal genesis export cannot see the
   committed keys.

   This same named handler activates the incident-operations and signer-policy
   hardening. There is no separately executable
   `upgrade-incident-operations-v1` handler; scheduling that retired draft name
   would stop at H without an implementation. Verify the source module-version
   response still contains `capability=1`, `feeibc=2`, `ibc=6`,
   `transfer=5`, `interchainaccounts=3`, `emergency=1`, `gov=5`, and
   `zerone_gov=2` before approving this plan.

   The new binary also inspects the dynamically mounted `feeibc` store's
   canonical immutable H-1 tree after staging its deletion. Any presence of the
   IBC-Go v8 `locked` key, regardless of value, refuses startup before a commit.
   This check works with IAVL fast nodes enabled or disabled and is independent
   of both the exported-state census and the IBC-only keyset manifest. If it
   fires, preserve the database, restart the old binary, and investigate and
   remediate the severe fee-accounting condition; do not simply delete the
   flag.

3. **Stage the exact binary before H.** Validators use the tested Cosmovisor
   `v1.7.1`, not `@latest`. Put the independently verified artifact at:

   ```text
   $DAEMON_HOME/cosmovisor/upgrades/<name>/bin/zeroned
   ```

   Keep this validator policy:

   ```shell
   export DAEMON_ALLOW_DOWNLOAD_BINARIES=false
   export DAEMON_DOWNLOAD_MUST_HAVE_CHECKSUM=true
   export DAEMON_RESTART_AFTER_UPGRADE=true
   export UNSAFE_SKIP_BACKUP=false
   ```

   Downloads remain off even when `info` contains a URL. The checksum setting
   is defense in depth, not verification of a manually staged binary. Verify
   the staged SHA-256 against an independently authenticated release manifest
   and ensure the backup filesystem has sufficient free space.

4. **Declare readiness before activation.** Operators representing more than
   two-thirds voting power must report the plan name and H, old and staged
   binary digests, Cosmovisor version/configuration, current app hash, signer
   state, and H−1 snapshot/restore readiness. Record the current 1/1 custodial
   exception honestly where it applies; it provides no independent Byzantine
   fault tolerance.

5. **Observe the H−1/H boundary.**
   - H−1 is the last committed state when an old binary without the handler
     stops at H.
   - Preserve the H−1 header, commit, app hash, snapshot digest, logs,
     `upgrade-info.json`, and signer state.
   - Do not repeatedly restart a failing binary or delete validator signing
     state.
   - Manual operation installs the exact attested binary; Cosmovisor selects
     the already-staged path. The new handler runs at H and H may then commit.

6. **Verify before reopening.** Confirm that
   `zeroned query upgrade applied <name>` returns H; all reporting nodes agree
   on the H block hash and app hash; the expected marker and module versions
   exist; supply and module accounts reconcile; IBC clients, channels,
   commitments, escrow, and voucher supply reconcile; restart and functional
   probes pass. Public transaction APIs and IBC reopen through explicit
   operations-manifest decisions, never automatically.

## Failure and recovery boundary

- **Before H:** standard governance may submit `MsgCancelUpgrade` while the
  chain can still process it.
- **At H before H commits:** H−1 is canonical. The preferred recovery is a
  corrected, attested binary for the same agreed plan. A coordinated
  `--unsafe-skip-upgrades H` is an exceptional chain-wide decision made while
  nodes remain on H−1. It skips the migration; it does not roll back state and
  must never be unilateral.
- **After H commits:** the migration is canonical history. Never run the old
  binary or use `--unsafe-skip-upgrades H` to pretend H did not occur. Repair
  forward with a new named upgrade. If committed state cannot be repaired
  deterministically, follow the explicit fork/re-genesis process in the
  canonical runbook.

Zerone has no generic arbitrary-height finalized-state rollback. A local
one-height database repair is not a network recovery protocol.

## Rules that keep upgrades boring

- **Handlers must be deterministic.** No map iteration without sorting, no
  lazy account creation (`GetModuleAccount` CREATES missing accounts and
  consumes account numbers in iteration order — the localnet drill caught a
  three-way AppHash divergence from exactly this). Touch only state that
  exists; log what you change.
- **Every handler keeps the `ReconcileModuleAccountPerms` call** — stored
  module-account permissions drift from code otherwise, and bank checks the
  stored ones.
- **Never edit an old handler after it ran anywhere.** New change = new name.
- **Never fetch a validator binary at H.** Pin Cosmovisor, keep downloads off,
  verify the staged digest, and retain the H−1 backup.
- **Test the H−1/H boundary, not just the handler.**
  `RunUpgradeHandlerForTests`
  exercises migrations only; rehearse schedule → old binary stops before H
  commits → staged binary starts from H−1 → H commits → postconditions.
