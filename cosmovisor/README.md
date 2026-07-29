# Cosmovisor — rehearsal reference only

This directory documents the shape of a Cosmos SDK binary-swap rehearsal. It
is not a live-network installation guide. Current `main` is moving source, no
Cosmovisor version is approved here, and no binary in this tree is authorized
for `zerone-1` or the legacy `zerone-testnet-1`.

A network-specific release packet must pin:

- the exact Cosmovisor version and checksum;
- the exact Zerone source commit and binary/image digests;
- the upgrade name and height;
- the network chain ID, genesis representation, and node home;
- pre-halt, post-migration, and rollback checks; and
- operator authorization.

It must keep `DAEMON_ALLOW_DOWNLOAD_BINARIES=false`. Never install
`cosmovisor@latest`, build a moving branch into a live node home, or infer
deployment authority from a passed governance proposal alone.

For an isolated local rehearsal, use a disposable home and chain ID such as
`zerone-rehearsal-1`. The conceptual layout is:

```text
<rehearsal-home>/cosmovisor/
├── genesis/bin/zeroned
└── upgrades/<approved-upgrade-name>/bin/zeroned
```

Populate both binaries from exact artifacts supplied by the rehearsal packet,
verify their digests before use, disable downloads, and destroy the disposable
home after the drill. See [UPGRADES.md](../docs/UPGRADES.md) for the protocol
model.
