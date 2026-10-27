# Cosmos Chain Registry submission bundle

`zerone/` is laid out to copy directly into the root of the upstream
[Cosmos Chain Registry](https://github.com/cosmos/chain-registry). The JSON
uses upstream-relative `$schema` paths intentionally.

The bundle describes the live `zerone-1` custodial mainnet. Its facts are
limited to values confirmed from the committed genesis and deployment
configuration or from the running node:

- `zerone-1`, `uzrn`, the ZRN denomination units, and the 1,814,400-second
  unbonding period;
- the pinned Cosmos SDK, CometBFT, and Go source versions;
- the consensus-enforced wallet fee floor of `1uzrn` per gas, while separately
  retaining checks that the running node's local mempool threshold is
  `0.025uzrn`;
- the complete public CometBFT RPC endpoint; and
- the live CometBFT `/genesis` of `zerone-1` behind the read-only TLS edge
  ([RPC_EDGE.md](../../deploy/RPC_EDGE.md)) as the public genesis URL. Its
  `jq .result.genesis` SHA-256 is the canonical RPC representation pinned in
  [JOIN.md](../../deploy/mainnet/JOIN.md). The committed `genesis.json`
  artifact hashes differently because it is the SDK genesis encoding; its
  `chain_id`, `genesis_time`, `app_state` and consensus params are the same.

The metadata deliberately omits a recommended application version, binary
URLs, REST/gRPC endpoints, snapshots, IBC channels, and links into this
repository, which is private and returns 404 to outside readers. The running
binary reports `version: dev` and `commit: unknown`; the only complete public
RPC is currently HTTP; the read-only TLS edge accepts GET/HEAD only and
refuses JSON-RPC POST and broadcast, so it serves the genesis URL but is not
listed as `apis.rpc`; the HTTPS dashboard edge is intentionally restricted and
is not advertised as a general RPC/REST endpoint; there is no published
snapshot service; and no external IBC channel is recommended. REST is not
advertised because the running v0.50.15 lineage does not register the
CometBFT, node and tx gRPC-gateway routes, so the upstream REST test path
`/cosmos/base/tendermint/v1beta1/syncing` returns 501; current source
registers them, so REST can be listed after an upgrade that carries them.
This is an operational metadata posture only: the published genesis enables
ICS-20 send/receive and the ICA host with a wildcard message allow-list.
Peer metadata is intentionally omitted: the listed node is the sole validator,
not a seed service, and the project does not advertise public peers for
`zerone-1` while validator admission is paused.

SLIP-0044 value 118 is the established Cosmos wallet derivation path used by
Zerone and many Cosmos SDK chains; it is not presented as a Zerone-specific
coin-type registration. The `zrn` Bech32 HRP is registered in SLIP-0173
([satoshilabs/slips#2039](https://github.com/satoshilabs/slips/pull/2039),
merged 2026-07-31).
[cosmos/chain-registry#7859](https://github.com/cosmos/chain-registry/pull/7859)
was merged on 2026-07-31 from an earlier snapshot, with 0.025 gas prices, a
seed entry and links into this private repository. This bundle is the
corrected source: submit it upstream whenever the two differ, and re-check
both after each `zerone-1` upgrade.

Validate local facts and then run the pinned upstream validator:

```bash
node scripts/check-chain-registry.mjs
scripts/validate-chain-registry.sh
```

The validator copies the bundle into a temporary checkout of upstream commit
`ecf22848fa4cdd2efefac6cda0e1552c61e2702b` and runs
`@chain-registry/cli@1.47.0`. It never overwrites an existing upstream
`zerone/` directory.
