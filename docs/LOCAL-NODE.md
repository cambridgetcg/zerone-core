# A persistent local Zerone node

The public [node guide](https://zerone.ai/nodes/) and
[machine-readable recipe](https://zerone.ai/nodes/guide.json) publish the same
source commit, helper checksum and commands. Check out that exact revision,
build with Go 1.25.14 and run the helper using Python 3.11+ on Linux or macOS.
The build requires Git, make, the platform C build tools and network access
for dependencies. No live account or wallet is needed.

```sh
GOTOOLCHAIN=go1.25.14 make build
./build/zeroned version --long
python3 scripts/local-node.py init --home "$HOME/zerone-local" --binary "$PWD/build/zeroned"
python3 scripts/local-node.py start --home "$HOME/zerone-local"
```

The start command remains in the foreground. In a second terminal, from the
same source checkout:

```sh
python3 scripts/local-node.py status --home "$HOME/zerone-local"
curl --fail --max-time 10 http://127.0.0.1:47657/status
```

Successful status checks identify `zerone-local-1`, match the local node
identity and observe advancing block heights. The curl response exposes
`result.node_info.network` and `result.sync_info.latest_block_height` for
an agent's own read-only integration. The local RPC accepts the running
source's CometBFT methods. REST, gRPC and gRPC-Web are disabled; the
[source API reference](API.md) also describes interfaces that this helper
does not expose.

## Lifecycle and files

Initialization requires a new absolute home under an existing directory.
It creates one local validator and a funded test account using the test
keyring, never imported keys. Test keys and balances have no live-network
value. The helper retains the binary under `bin/zeroned`; rebuilding the
checkout later does not silently replace a running sandbox's executable.

Press Ctrl-C in the start terminal to stop its child process. Data and keys
remain in the chosen home. Run the same `start` command to resume. There is
no reset command. For another experiment, choose a different new home and
free ports at initialization:

```sh
python3 scripts/local-node.py init --home "$HOME/zerone-local-two" \
  --binary "$PWD/build/zeroned" --chain-id zerone-local-2 \
  --rpc-port 48657 --p2p-port 48656
python3 scripts/local-node.py start --home "$HOME/zerone-local-two"
```

RPC and P2P bind to `127.0.0.1`; peer discovery, state sync and external
peers are disabled. The helper checks retained binary and configuration
integrity on restart. A conflicting port or changed configuration produces
an error; it does not terminate another process or repair a different home.
Keep the terminal's error output when diagnosing a failed start, but never
publish the keyring, signing keys or a complete node home.

## Existing public network

This creates a separate local chain. It does not replicate `zerone-1`, add
a public validator, admit a live account or establish reward entitlement.
The current source is newer than the installed legacy runtime, and a
compatible public replica package has not been published. Public records
remain available through the website's selected
[RPC status](https://zerone.ai/api/rpc/status) and dashboard reads.

The [current joining status](../deploy/mainnet/JOIN.md),
[trust model](../deploy/mainnet/TRUST.md), [ledger census](reports/authenticated-ledger-census-2026-09-09.md)
and [settlement history](reports/knowledge-settlement-history-2026-09-09.md)
describe the live release and evidence limits. Local development and source
contributions can proceed while the public replica and admission release
requirements remain unresolved.
