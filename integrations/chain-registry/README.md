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
  `0.025uzrn`; and
- the complete public CometBFT RPC endpoint.

The metadata deliberately omits a recommended application version, binary
URLs, REST/gRPC endpoints, snapshots, and IBC channels. The running binary
reports `version: dev` and `commit: unknown`; the only complete public RPC is
currently HTTP; the HTTPS dashboard edge is intentionally restricted and is
not advertised as a general RPC/REST endpoint; there is no published snapshot
service; and no external IBC channel is recommended. This is an operational
metadata posture only: the published genesis enables ICS-20 send/receive and
the ICA host with a wildcard message allow-list.
Peer metadata is intentionally omitted while node joining is paused; the sole
validator is not a public seed service.

SLIP-0044 value 118 is the established Cosmos wallet derivation path used by
Zerone and many Cosmos SDK chains; it is not presented as a Zerone-specific
coin-type registration. The `zrn` Bech32 HRP needs its own SLIP-0173 registry
entry before an upstream submission should be represented as complete. That
registration is proposed in
[satoshilabs/slips#2039](https://github.com/satoshilabs/slips/pull/2039).
The external proposal
[cosmos/chain-registry#7859](https://github.com/cosmos/chain-registry/pull/7859)
predates this consolidation and is stale. It must be refreshed from this local
bundle before merge; this repository does not treat the open PR as current
evidence.

Validate local facts and then run the pinned upstream validator:

```bash
node scripts/check-chain-registry.mjs
scripts/validate-chain-registry.sh
```

The validator copies the bundle into a temporary checkout of upstream commit
`ecf22848fa4cdd2efefac6cda0e1552c61e2702b` and runs
`@chain-registry/cli@1.47.0`. It never overwrites an existing upstream
`zerone/` directory.
