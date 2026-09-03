# `zerone-testnet-1`

`zerone-testnet-1` is a live legacy playground. Its public RPC was observed
serving the chain at `http://37.16.28.121:26657` on 2026-07-29. Testnet state
has play value only and may be reset or withdrawn without notice.

## Consolidation safety status

| Field | Value |
|---|---|
| Chain ID | `zerone-testnet-1` |
| Current posture | live legacy network; read-only observation supported |
| Base denomination | `uzrn` |
| Public RPC | `http://37.16.28.121:26657` (unsigned legacy endpoint) |
| Consolidated source | not yet activated on the network |
| Validator/full-node joining | paused |

The running network predates this consolidated source release and has not
completed the ordered H1 `consolidation-safety-v1`, H2
`founder-renunciation-v1`, and H3 `sdk-0.53-ibc-10` boundaries. H1 preserves
vesting_rewards V1 and H2 alone advances it to V2. Building the current `main`
branch and connecting it as a validator or replaying it from genesis is not an
authorised or compatibility-tested path.

Safe observation:

```bash
curl -fsS http://37.16.28.121:26657/status |
  jq '{chain_id: .result.node_info.network,
       height: .result.sync_info.latest_block_height,
       catching_up: .result.sync_info.catching_up}'
```

Treat the IP endpoint as operational telemetry, not as release provenance.
The repository does not currently publish a signed, release-bound join packet
for this legacy deployment.

## What is paused

Until a specific release commit, binary digest, live genesis representation,
peer identity, upgrade height, and post-upgrade verification are published:

- do not run `deploy/testnet/node-bootstrap.sh`;
- do not use current source to start a full node or validator;
- do not submit validator-admission transactions based on an old guide; and
- do not infer current parameters or economics from historical reports.

The retired join guides now point here. Future participation instructions must
be bound to a signed release and the network's actual upgrade state.

The separate `zerone-2` relaunch remains **NO-GO** until its signed ceremony and
authority gates are complete; see
[`deploy/networks/zerone-2/README.md`](../../deploy/networks/zerone-2/README.md).
