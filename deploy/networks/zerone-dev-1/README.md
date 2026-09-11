# zerone-dev-1 runtime

This directory runs a **fresh, valueless development chain**. It does not import
zerone-1, activate zerone-2, or announce that deployment has happened. Consult the
dated publication at [zerone.ai/development](https://zerone.ai/development/) for
the actual source, binary, genesis and network-descriptor pins.

The initial chain has **one operator-controlled validator seat**
(`staking.max_validators = 1`). Genesis allocates 2,000,000 development ZRN to the
operator, of which 1,000,000 is bonded, and 1,000,000 development ZRN to the faucet.
The gateway can distribute at most 100,000 ZRN in 100 grants of 1,000 ZRN, once per
address, with five grants per IP/hour and ten globally/hour. The bonded operator
stake exceeds that entire public grant budget. These are development allocations,
not money, production allocations, or credentials. Consensus membership would
require a separate explicit plan. Distinct reviewer accounts do not establish
independent controllers.

All current native accounting and knowledge flags are enabled. Default source
genesis/initialization supplies the standard methodologies. Commit and reveal
windows each last 3,600 blocks; aggregation lasts five. The configured one-second
commit timeout is a pacing target, not a wall-clock promise. Default application
pruning retains recent versions; transaction indexing and block history are kept.
Snapshots are produced every 1,000 blocks, keeping two. This is not an archival
retention promise: measure volume growth and publish any future retention change.

## Build a clean image

Use a clean checkout of the exact intended commit. The exporter reads Git blobs,
refuses dirty/untracked inputs, and creates a new context containing only Go
production application code, the three Swagger embed inputs, go.mod/go.sum,
runtime.py, gateway.py and Dockerfile. It excludes test fixtures, deployment
archives, node homes, keyrings, Git metadata and arbitrary repository files.
Do not upload the repository root to a remote image builder.

```sh
SOURCE_COMMIT=$(git rev-parse HEAD)
CONTEXT="$HOME/zerone-dev-build-context"
python3 -I -B deploy/networks/zerone-dev-1/build-context.py \
  --repo "$PWD" --commit "$SOURCE_COMMIT" --output "$CONTEXT"
docker build --platform linux/amd64 --tag zerone-dev-1:local "$CONTEXT"
```

The Dockerfile pins official Go 1.25.14 and Debian image digests plus the Debian
APT snapshot. It uses CGO_ENABLED=0, -mod=readonly, -trimpath and -buildvcs=false;
the embedded source/version come from the exported Git receipt. The runtime
image retains `/opt/zerone-dev/{build.json,source-context.json}`. `build.json`
binds the binary SHA-256 to its embedded commit. An image digest and build receipt
are provenance, not a runtime security audit or an ongoing maintenance promise.

## Persistent operator runtime

The image entrypoint runs:

```sh
python3 -I -B /opt/zerone-dev/runtime.py run \
  --home /data/.zeroned --binary /usr/local/bin/zeroned
```

First boot requires that home to be absent. It generates new P2P and consensus
identities and SDK `validator`/`faucet` accounts on the private volume, validates
native genesis, constructs the validator transaction, configures listeners, and
writes the durable faucet journal before the final runtime manifest. It accepts
no key import, legacy state, reset, or parameter override for an existing home.
An interrupted initialization remains on disk and is refused; it is not silently
discarded. The SDK test keyring is unencrypted and is appropriate only for this
explicit development custody model. Never distribute the private home.

Subsequent starts require the same binary, genesis, configuration and static
identity bytes, a present signing-state file, and the existing private faucet
journal. Node and gateway stop together when either exits. SIGINT/SIGTERM stop
both cleanly; a forced stop is reported as a failure. Shared runtime logs use
stdout/stderr for Fly collection rather than growing files on the state volume.
The local test profile retains private diagnostic logs.

The lock excludes a second cooperating runtime using the same home. It does not
fence a copied signer on another host, or someone directly invoking zeroned. A
restore or machine replacement still requires the operator to stop/fence the old
instance and move the complete coherent stopped home, including signing high-water
state. Do not start two machines from copies of this validator volume.

`fly.toml` describes one always-running machine in London, two shared CPUs, 4 GiB
RAM, a private `zerone_dev_1_data` volume mounted at `/data`, HTTPS gateway port8080,
and public TCP P2P26656. Start with a 20 GiB volume as operational headroom for
application state, block history and snapshots; this is not a measured minimum or
unlimited retention. Use one machine and the immediate deployment strategy.
Public raw TCP26656 requires a dedicated IPv4 for IPv4 peers (IPv6 is separate).
RPC26657 stays on loopback. REST, gRPC, pprof and unsafe RPC are disabled. The
gateway permits bounded public query and signed sync-broadcast requests, not
arbitrary admin calls or server-side participant signing.

The gateway writes immutable public network.json only after the node is ready,
including the exact file-genesis hash and separately canonicalized RPC-genesis
hash. The operator must publish and independently check those new-network pins
before inviting participation. Gateway readiness is checked at `/healthz`;
runtime stdout first reports advancing node readiness, then supervises gateway
liveness. The public health route checks the node, not scientific claims.

## Run a fresh full node

Use Python 3.11+ and a binary built from the **runtime source commit** in the
verified development publication. A native build need not share the Linux image
binary hash. Build at the exact runtime commit, then keep its bytes unchanged.
Verify the public descriptor and copy its `source_commit`, `genesis_sha256`,
`peer` and `rpc_url` from the pinned publication. Do not derive trust solely from
an unverified endpoint.

```sh
RUNTIME_COMMIT='REPLACE_WITH_VERIFIED_40_HEX_RUNTIME_COMMIT'
GENESIS_SHA256='REPLACE_WITH_VERIFIED_64_HEX_GENESIS_HASH'
PEER='REPLACE_WITH_VERIFIED_NODE_ID@zerone-dev-1.fly.dev:26656'
FULL_HOME="$HOME/zerone-dev-full-node"
curl --fail --max-time 30 --max-filesize 16777216 \
  https://zerone-dev-1.fly.dev/genesis.json -o zerone-dev-genesis.json
python3 -I -B deploy/networks/zerone-dev-1/runtime.py join \
  --home "$FULL_HOME" --binary "$PWD/build/zeroned" \
  --source-commit "$RUNTIME_COMMIT" \
  --genesis zerone-dev-genesis.json --genesis-sha256 "$GENESIS_SHA256" \
  --peer "$PEER" --reference-rpc https://zerone-dev-1.fly.dev
python3 -I -B deploy/networks/zerone-dev-1/runtime.py run \
  --home "$FULL_HOME" --binary "$PWD/build/zeroned"
```

In another terminal:

```sh
python3 -I -B deploy/networks/zerone-dev-1/runtime.py status \
  --home "$FULL_HOME" --binary "$PWD/build/zeroned"
```

`join` creates fresh zero-power identities, validates the supplied genesis and
its pin, checks the reference chain/peer identity, and configures genesis replay
through the persistent P2P peer. No faucet/account key is created. Local RPC is
127.0.0.1:26657; P2P listens on26656. The helper refuses a nonzero reported local
voting power. `status` compares a common committed block ID and header AppHash
with the pinned reference node; these are cross-checked node observations, not
an independent signature-verification receipt or independent consensus. It never
compares a same-height post-state application root against that header's prior
state root. Restart with the same `run` command; Ctrl-C preserves the home.

## Local tests

Only the explicit `--local-test` profile permits a fresh validator outside
`/data/.zeroned`, alternate loopback ports and accelerated windows:

```sh
python3 -I -B deploy/networks/zerone-dev-1/runtime.py init \
  --home "$HOME/zerone-dev-runtime-fixture" --binary "$PWD/build/zeroned" \
  --source-commit "$(git rev-parse HEAD)" --local-test --review-window-blocks 60 \
  --rpc-port 49657 --p2p-port 49656 --gateway-port 49880
python3 -I -B deploy/networks/zerone-dev-1/runtime.py run \
  --home "$HOME/zerone-dev-runtime-fixture" --binary "$PWD/build/zeroned"
python3 -I -B -m unittest discover -s deploy/networks/zerone-dev-1 -p 'test_*.py' -v
ZERONE_SHARED_TEST_BINARY="$PWD/build/zeroned" python3 -I -B -m unittest discover \
  -s deploy/networks/zerone-dev-1 -p test_runtime.py -v
```

The optional native test creates two new private homes, uses actual CLI genesis
and daemon processes, checks zero-power full-node replay and common-block
agreement, stops/restarts the validator, verifies signing-state progress, and
tests same-home and missing-state refusal. It uses test identities and no external
node network. It does not establish cross-host fencing or production recovery.
