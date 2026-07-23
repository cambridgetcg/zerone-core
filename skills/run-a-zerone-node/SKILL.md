---
name: run-a-zerone-node
description: >-
  Stand up a zerone full node or validator from a fresh Ubuntu/Debian VM, on
  the testnet (zerone-testnet-1, default) or the mainnet (zerone-1, via
  NETWORK=mainnet). Use when an agent or operator wants to sync and
  independently verify the chain, check the pulled genesis sha256 against the
  documented hash, run 24/7 under systemd, become a validator, or snapshot to
  free storage. Bundles the repo's node-bootstrap.sh under scripts/. Not for
  lightweight queries or joining as a citizen — zerone-onboarding covers the
  no-node lanes.
---

# Run a zerone node

Running your own node is how you stop being a guest and become an
**operator**: your node independently verifies every block against the rules —
it doesn't trust the validators, it *checks* them. Full walkthrough:
`deploy/testnet/RUN-A-NODE.md` in the zerone-core repo (covers both networks).

## One-shot bootstrap (fresh Ubuntu 22.04+)

The repo's bootstrap script is bundled at `scripts/node-bootstrap.sh`
(byte-identical copy of `deploy/testnet/node-bootstrap.sh`). **Read it before
running** — it uses sudo for packages, /usr/local, and systemd:

```
less scripts/node-bootstrap.sh             # read it first
bash scripts/node-bootstrap.sh             # testnet (default)
NETWORK=mainnet bash scripts/node-bootstrap.sh   # zerone-1 mainnet
```

It installs Go + build tools, builds `zeroned`, initialises your home, pulls
the live genesis, wires the seed and the gas floor, installs a systemd unit,
and starts syncing. Env overrides: `NETWORK`, `MONIKER`, `RPC`, `GO_VERSION`.

## Network parameters

| | Mainnet | Testnet |
|---|---|---|
| Chain ID | `zerone-1` | `zerone-testnet-1` |
| RPC | `http://169.155.55.44:26657` | `http://37.16.28.121:26657` |
| P2P seed | `ed8c8d49dc23f3478b2f3eddb49b8f8087828b6e@169.155.55.44:26656` | `9a9c6b9d36c55d21c32b1ee8749adf8dd7c6b0d4@37.16.28.121:26656` |

## Verify the genesis before trusting it

Pull the genesis from the network RPC and hash it:

```
curl -s http://169.155.55.44:26657/genesis | jq .result.genesis > ~/.zeroned/config/genesis.json
sha256sum ~/.zeroned/config/genesis.json
```

Compare against the documented hash:

- **Mainnet `zerone-1`** (`deploy/mainnet/JOIN.md`):
  `16ac346f329d2a931ad9a7d51dbe9e35605482b006ef39b3ac7804376e9bcb66`
  (of `curl RPC/genesis | jq .result.genesis`)
- **Testnet `zerone-testnet-1`** (`deploy/testnet/JOIN.md`):
  `a2a5499fcd43668f328b0ab504ad9f7c3aadd65f7abd8a4f3991b927872a6a2a`
  — re-published on each testnet reset; always verify against the live
  `RPC/genesis`.

A mismatch means you are NOT on the documented network — stop and ask.
The bootstrap script prints the same sha256 for you to compare.

## Confirm you're synced

```
zeroned status | jq '{height: .sync_info.latest_block_height, catching_up: .sync_info.catching_up}'
```

`catching_up: false` means you are at the tip, verifying every block
yourself. Query anything locally: `zeroned query bank total`.

## Go further

- **Become a validator** — funded key + synced node, then
  `zeroned tx staking create-validator ...`; on the mainnet this is the
  earned, self-bonded Phase-3 path and moves the published decentralization
  metric. Recipe: `references/operator-guide.md`.
- **Snapshots on free storage** — tar the data dir, publish height + sha256 so
  others can verify: `references/operator-guide.md`.
- **Free compute honestly compared** (Oracle/GCP/AWS/fly.io, with caveats):
  `references/operator-guide.md`.

Don't run every node on the same provider/account — that recreates the
centralization you're trying to avoid. Every independent node moves the dial
from *custodial* toward *decentralized* — on the mainnet that movement is the
whole game (`deploy/mainnet/JOIN.md`).
