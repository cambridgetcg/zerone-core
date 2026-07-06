# Upgrading a Live Zerone — the One-Page Runbook

*Proven end-to-end on the localnet 2026-07-06: gov-scheduled halt at the
exact height, binary swap, migrations + module-account reconcile, all four
validators resumed in lockstep. For the code-side migration recipe see
[UPGRADE_PROTOCOL.md](UPGRADE_PROTOCOL.md).*

## The whole model in four sentences

A named upgrade handler in `app/upgrades.go` says what the new binary does
at the switch. Governance schedules the name at a height. Every validator's
old binary halts at that height (`UPGRADE <name> NEEDED`) and writes
`data/upgrade-info.json`. The new binary — swapped in by hand or by
cosmovisor — runs the handler and the chain resumes.

## Operator steps

1. **Ship the code.** New named handler (copy the `v1.0.3-testnet` block in
   `app/upgrades.go`: `RunMigrations` + `ReconcileModuleAccountPerms` + a
   marker), a lineage entry in `app/upgrade_registry.go`, a
   `RegisterStoreUpgrades` case if store keys change. Build the new binary.

2. **Schedule via governance.** One proposal:

   ```json
   {"messages": [{"@type": "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade",
     "authority": "<gov module address>",
     "plan": {"name": "<handler name>", "height": "<H>", "info": "<binary checksums>"}}],
    "deposit": "10000000uzrn", "title": "...", "summary": "..."}
   ```

   `zeroned tx gov submit-proposal plan.json --from <key>` → vote → passes.
   Verify: `zeroned query upgrade plan`.

3. **Stage the new binary before H.** Manual: have it ready. Cosmovisor:
   place it at `cosmovisor/upgrades/<name>/bin/zeroned` under `DAEMON_HOME`
   (`make cosmovisor-init` scaffolds this; keep
   `DAEMON_ALLOW_DOWNLOAD_BINARIES=false` on real networks).

4. **At H the chain halts by itself.** Every node panics
   `UPGRADE <name> NEEDED` and writes `data/upgrade-info.json`. This is the
   mechanism working, not an outage.

5. **Swap and restart.** Manual: replace the binary, start the node.
   Cosmovisor: it does both on its own. The handler runs once at H
   (migrations, permissions reconcile, marker) and blocks continue.

6. **Verify.** `zeroned query upgrade applied <name>` returns the height;
   the knowledge marker `upgrade_marker_<version>` reads `migrated`; all
   validators report the same height and keep producing.

## Rollback

Before H: gov `MsgCancelUpgrade`. After a bad halt: restart old binaries
with `--unsafe-skip-upgrades <H>` — by social consensus only, never
unilaterally.

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
