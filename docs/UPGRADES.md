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

## Pending atomic release

Existing networks have one pending software-upgrade boundary:
`consolidation-safety-v1` at H1. The exact H1 binary calls module migrations,
reconciles stored module-account permissions, and records one handler marker.
Among its other safety changes, the resulting module-version map must report:

- `liquiditypool = 5`; and
- `vesting_rewards = 2`.

There is no `liquiditypool-safety-v2` H2 handler in the H1 binary. Advertising
a future handler early is unsafe under the SDK upgrade guard, and a redundant
second height would not add a technical activation boundary. Native pool
creation and oracle use remain disabled after H1 until their separate
governance, capital, bank-send, price/depth, and recovery gates pass.

Before scheduling H1, bind a same-height live snapshot proving there are zero
native pools and no billing quote-denom allowlist. Valid legacy pools migrate
`EXIT_ONLY`, so a network with any existing pool still needs a separately
reviewed transition. After H1, verify `allowed_pool_denoms`, `pool_creators`,
and `billing_quote_denoms` are empty before accepting any Params update. The
complete no-go gates, economic-neutrality transition, status lifecycle,
governance path, and Osmosis-testnet separation are in
[LIQUIDITYPOOL-SAFETY-V2.md](LIQUIDITYPOOL-SAFETY-V2.md).

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

H1 verification does not authorize a pool by itself. Invariants and
application lifecycle tests must also pass on the exact release, and native
pool creation plus oracle quote allowlisting remain separate governance
actions. H1 must additionally prove `protocol_fee_bps = 0`, founder fields are
zero/empty, transaction-presence rewards are zero, and real transaction-fee
routing still operates.

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
