# Zerone upgrade protocol — release-packet template

> **Not live deployment authority.** The commands and sequence below describe
> a local rehearsal and the fields a network-specific, signed release packet
> must bind. Do not build moving `main`, install into a default node home, or
> act on this page alone.

*Proven end-to-end on a localnet 2026-07-06: gov-scheduled halt at the
exact height, binary swap, migrations + module-account reconcile, all four
validators resumed in lockstep. For the code-side migration recipe see
[UPGRADE_PROTOCOL.md](UPGRADE_PROTOCOL.md).*

## The whole model in four sentences

A named upgrade handler in `app/upgrades.go` says what the new binary does
at the switch. Governance schedules the name at a height. Every validator's
old binary halts at that height (`UPGRADE <name> NEEDED`) and writes
`data/upgrade-info.json`. The new binary — swapped in by hand or by
cosmovisor — runs the handler and the chain resumes.

## Pending release order

Existing networks have two exact, ordered SDK 0.50 source boundaries:

1. `consolidation-safety-v1` (H1), using independently accepted source commit
   `65c19cd8b00bdfff9b80705b776fd0d49719398a`; then
2. `founder-renunciation-v1` (H2), using a separately audited descendant.

H1 produces the exact K6/P2/L5/V1 VersionMap, migrated marker, and positive
done height. Only that completed state may halt for H2. The H2 binary cannot
start on an H1 prestate, a plan-less V1 state, or a different plan; it advances
only vesting_rewards V1→V2 and reconciles its existing module account to empty
permissions.

Release digests are boundary-specific. Never use H2 to execute H1 and never
build moving `main` for either pending plan. The rejected former H2 commit
`4bffb6d218819bed1c29c7a0be7779ad31c64a97` is not release provenance.
See [FOUNDER-RENUNCIATION-V1.md](FOUNDER-RENUNCIATION-V1.md) for the composite
startup and handler evidence wall.

## Operator steps

1. **Prepare and review the code.** New named handler (copy the
   `v1.0.3-testnet` block in
   `app/upgrades.go`: `RunMigrations` + `ReconcileModuleAccountPerms` + a
   marker), a lineage entry in `app/upgrade_registry.go`, a
   `RegisterStoreUpgrades` case if store keys change. A release process must
   produce exact, signed binary/image digests; a local `make build` is not a
   production artifact.

2. **Schedule via governance.** One proposal:

   ```json
   {"messages": [{"@type": "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade",
     "authority": "<gov module address>",
     "plan": {"name": "<handler name>", "height": "<H>", "info": "<binary checksums>"}}],
    "deposit": "10000000uzrn", "title": "...", "summary": "..."}
   ```

   `zeroned tx gov submit-proposal plan.json --from <key>` → vote → passes.
   Verify: `zeroned query upgrade plan`.

3. **Stage the approved binary before H.** Verify it against the release
   packet. If the packet authorizes Cosmovisor, place the exact artifact at
   `cosmovisor/upgrades/<name>/bin/zeroned` under the explicitly named
   `DAEMON_HOME`; keep `DAEMON_ALLOW_DOWNLOAD_BINARIES=false`.

4. **At H the chain halts by itself.** Every node panics
   `UPGRADE <name> NEEDED` and writes `data/upgrade-info.json`. This is the
   mechanism working, not an outage.

5. **Swap and restart.** Manual: replace the binary, start the node.
   Cosmovisor: it does both on its own. The handler runs once at H
   (migrations, permissions reconcile, marker) and blocks continue.

6. **Verify.** `zeroned query upgrade applied <name>` returns the height;
   the knowledge marker `upgrade_marker_<version>` reads `migrated`; all
   validators report the same height and keep producing.

For `liquiditypool-safety-v2`, verification does not authorize a pool by
itself. Invariants and application lifecycle tests must also pass on the exact
release, and native pool creation plus oracle quote allowlisting remain
separate governance actions.

For `founder-renunciation-v1`, verify `vesting_rewards=2`, both ordered
H1/H2 markers and done heights, zero/empty founder and automatic-reward
compatibility fields, empty stored vesting module permissions, unchanged
supply, complete research routing, and a failed ordinary-governance
restoration attempt. See
[FOUNDER-RENUNCIATION-V1.md](FOUNDER-RENUNCIATION-V1.md).

## Rollback

Before H: governance may submit `MsgCancelUpgrade`. H2 deliberately refuses
`--unsafe-skip-upgrades` at either the inherited H1 completion height or the
H2 height; skipping would destroy the proof this binary requires. Recovery
after a bad halt must use the signed ceremony runbook and exact prior binary,
with validators agreeing on state and next action—never a unilateral skip.

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
- **Test the halt, not just the handler.** `RunUpgradeHandlerForTests`
  exercises migrations only; the localnet drill (schedule → halt → swap →
  resume) is the real rehearsal and takes ~10 minutes.
