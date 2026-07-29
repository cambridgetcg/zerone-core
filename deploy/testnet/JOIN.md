# `zerone-testnet-1` — Legacy Network Notice

`zerone-testnet-1` is live as a resettable, play-value legacy network. Its
public RPC was observed serving blocks at `http://37.16.28.121:26657` on
2026-07-29.

This is not a validator or full-node join guide.

The network predates the consolidated source on `main` and has not activated
the `consolidation-safety-v1` upgrade. Building the current branch and replaying
or validating this chain is therefore paused until a release-bound upgrade
packet is published.

Read-only observation:

```bash
curl -fsS http://37.16.28.121:26657/status |
  jq '{chain_id: .result.node_info.network,
       height: .result.sync_info.latest_block_height,
       catching_up: .result.sync_info.catching_up}'

curl -fsS \
  'http://37.16.28.121:1317/cosmos/bank/v1beta1/supply/by_denom?denom=uzrn'
```

The IP endpoints are unsigned operational surfaces and may change or
disappear. They are not proof of the binary, commit, genesis bytes, parameters,
or authority chain currently running.

Do not:

- run `node-bootstrap.sh`;
- start a validator or replay node from current `main`;
- rely on the historical genesis hash as a release signature;
- buy or request credentials based on an old copy of this guide; or
- treat testnet ZRN or state as persistent or valuable.

Joining can reopen only when the repository publishes the exact release commit,
binary digest, canonical live genesis representation, peer identities, upgrade
height, and post-upgrade verification.

See the canonical
[network status](../../networks/zerone-testnet-1/README.md) and the
[validator safety guide](../../docs/VALIDATOR-GUIDE.md).
