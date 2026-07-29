---
name: run-a-zerone-node
description: >-
  Explain Zerone node-release requirements and the current pause on joining
  zerone-1 or zerone-testnet-1. Use when an agent or operator asks to run a
  node, validate, bootstrap, restore a snapshot, or connect current source to
  a live Zerone network. This skill is fail-closed until a signed,
  release-bound join packet exists; it may perform read-only status checks but
  must not install, start, reset, or reconfigure a node.
---

# Zerone node participation — paused

Do not bootstrap a live Zerone node from the current `main` branch.

Both `zerone-1` and `zerone-testnet-1` predate this consolidated source. The
branch contains consensus-sensitive behavior that requires the named
`consolidation-safety-v1` upgrade. Replaying a live chain or replacing a
validator binary with this source outside that upgrade path is unsafe.

The bundled `scripts/node-bootstrap.sh` intentionally exits with failure.

## Allowed read-only checks

You may report public endpoint status without changing local or remote state:

```bash
curl -fsS http://169.155.55.44:26657/status |
  jq '{chain_id: .result.node_info.network,
       height: .result.sync_info.latest_block_height,
       catching_up: .result.sync_info.catching_up}'

curl -fsS http://37.16.28.121:26657/status |
  jq '{chain_id: .result.node_info.network,
       height: .result.sync_info.latest_block_height,
       catching_up: .result.sync_info.catching_up}'
```

Treat those unsigned IP endpoints as observations, not as provenance.

## Required before this skill can run a node

Require a network-specific packet containing:

1. the exact source commit and reproducible binary or image digest;
2. signatures and the accepted provenance policy;
3. canonical genesis representation and hash;
4. seed and persistent-peer identities;
5. governance-approved upgrade name and height;
6. snapshot/state-sync trust material, if used; and
7. explicit authorization for the requested network phase.

Do not infer any of these from a moving branch, historical guide, public RPC,
or running node. See `references/operator-guide.md` and
`docs/VALIDATOR-GUIDE.md`.
