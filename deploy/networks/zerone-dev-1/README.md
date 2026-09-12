# zerone-dev-1 runtime

This directory runs the **zerone-dev-1 valueless development chain**, separate
from zerone-1 and the unactivated zerone-2. The dated publication at
[zerone.ai/development](https://zerone.ai/development/) identifies the exact
source, binary, genesis and network descriptor.

On 2026-09-12, the operator deployed source
`ac5684e33b6946a31f6ea260ebf280eace55b033` and checked these public inputs through
both HTTPS and the authenticated provider connection. [release.json](release.json)
binds the immutable runtime image and compatible Linux/macOS packages;
[network.json](network.json) and [genesis.json](genesis.json) retain the exact
public bytes. These files contain no private keys.

A fresh operator-test participant received funds, registered as an agent, and
submitted claim `8db0bccfb17b9a5ad1cf2389b79abd38` at height 432 with execution
code zero. A separate zero-power full node replayed public P2P blocks and matched
the claim history at height 510, then preserved it through restart. One normal
hosted-validator restart preserved the entire height-831 block and claim history,
static identities, genesis, descriptor and faucet journal; blocks advanced to 926.
The packaged macOS client also read the public claim successfully. These are
dated operator checks, not independent controller, security-audit or archive
completeness claims. The public example has no staged reviews or challenges.

The initial chain has **one operator-controlled validator seat**
(`staking.max_validators = 1`). Genesis allocates 2,000,000 development ZRN to the
operator, of which 1,000,000 is bonded, and 1,000,000 development ZRN to the faucet.
The gateway can distribute at most 100,000 ZRN in 100 grants of 1,000 ZRN, once per
address, with five grants per IP/hour and ten globally/hour. The bonded operator
stake exceeds that entire public grant budget. These are development allocations,
not money, production allocations, or credentials. Consensus membership would
require a separate explicit plan. Distinct reviewer accounts do not establish
independent controllers.

The original published version-10 genesis enables its native accounting and knowledge flags. Current version-11 source genesis additionally enables `fund_settlement_enabled`; this does not relabel the original genesis or announce an applied hosted upgrade. Default source
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

For the original version-10 genesis, use Python 3.11+ and the **verified predecessor package** (runtime helper and binary from `ac5684e33b6946a31f6ea260ebf280eace55b033`). After publication of the version-11 upgrade packet, run `join` with that predecessor, then use the staged replay below. Do not run the version-11 native initializer against the original version-10 genesis. For a newly created version-11 test genesis, use its matching current source helper and binary. A native build need not share the Linux image
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

## Stage a knowledge 10-to-11 upgrade

This path preserves the original chain ID, genesis, runtime manifest, identities, signing state, application history and faucet journal. It does not initialize or reset a home, replace a participant client, schedule governance, or activate zerone-1. The operator must publish an externally verified upgrade packet and both source-built target binaries after the application rehearsal and release checks; no height is implied by this source guide.

The packet is a small JSON object with exactly these fields (placeholders are not executable pins):

```json
{
  "schema": "zerone-development-upgrade/v1",
  "chain_id": "zerone-dev-1",
  "genesis_sha256": "ORIGINAL_FILE_GENESIS_SHA256",
  "predecessor_descriptor_sha256": "ORIGINAL_DESCRIPTOR_SHA256",
  "plan": {"name": "knowledge-fund-settlement-v1", "height": "ACTUAL_ACTIVATION_HEIGHT", "info": ""},
  "predecessor": {
    "knowledge_version": 10,
    "source_commit": "EXACT_PREDECESSOR_COMMIT",
    "binaries": {"linux-amd64": "PREDECESSOR_LINUX_SHA256", "darwin-arm64": "PREDECESSOR_DARWIN_SHA256"}
  },
  "target": {
    "knowledge_version": 11,
    "source_commit": "EXACT_TARGET_COMMIT",
    "binaries": {"linux-amd64": "TARGET_LINUX_SHA256", "darwin-arm64": "TARGET_DARWIN_SHA256"}
  }
}
```

Take the packet's own SHA-256 from the reviewed source publication, outside the downloaded packet. Keep the already verified predecessor package; the successor release need not redistribute its immutable archive. Extract the successor into a different new directory. Stop the existing cooperating runtime cleanly before staging. An operator must also fence any other copy of the validator signer; the local lock is not cross-host fencing.

For a new full node, first use the predecessor helper's `join` command with the original genesis. It creates a zero-power home without starting replay. Then use the successor helper for all staged operations:

```sh
UPGRADE_HELPER='/ABSOLUTE/PATH/TO/SUCCESSOR/runtime/runtime.py'
PREDECESSOR_BINARY='/ABSOLUTE/PATH/TO/VERIFIED/PREDECESSOR/bin/zeroned'
TARGET_BINARY='/ABSOLUTE/PATH/TO/VERIFIED/SUCCESSOR/bin/zeroned'
UPGRADE_PACKET='/ABSOLUTE/PATH/TO/VERIFIED/upgrade.json'
UPGRADE_SHA256='REPLACE_WITH_EXTERNAL_PACKET_SHA256'
NODE_HOME='/ABSOLUTE/PATH/TO/EXISTING/PRIVATE/NODE/HOME'
python3 -I -B "$UPGRADE_HELPER" stage-upgrade \
  --home "$NODE_HOME" --predecessor-binary "$PREDECESSOR_BINARY" \
  --binary "$TARGET_BINARY" --upgrade-packet "$UPGRADE_PACKET" \
  --upgrade-packet-sha256 "$UPGRADE_SHA256"
python3 -I -B "$UPGRADE_HELPER" run-upgrade \
  --home "$NODE_HOME" --upgrade-packet-sha256 "$UPGRADE_SHA256"
```

Staging copies the two verified executables and packet into a new `fund-settlement-upgrade` directory and binds the untouched original manifest. Existing or incomplete staging is preserved and refused, not overwritten. Before the halt record exists, `run-upgrade` starts only the predecessor. At its actual halt it checks the application's ABCI height **H−1** and matching on-chain plan when the old process is still available; Comet's stored block height can already be H. It stops the old process before starting the target. The target application's independent startup guard requires the exact committed H−1 state, matching local/on-chain plan, prior version map and migration lineage. A local plan file alone never establishes a successful upgrade.

After the target advances, the helper checks knowledge version 11 and the exact applied height before starting the gateway. It preserves a first applied observation and publishes a separate `public/network-knowledge-11.json`; original `public/network.json` stays untouched. The selected gateway `/network.json` then serves the new descriptor. Original `/genesis.json` and faucet grant history stay unchanged. A restart with the same `run-upgrade` command chooses the predecessor before a halt record and the target afterward; application startup remains the authority on whether either is valid.

In another terminal after the target is running:

```sh
python3 -I -B "$UPGRADE_HELPER" upgrade-status \
  --home "$NODE_HOME" --upgrade-packet-sha256 "$UPGRADE_SHA256"
```

A full node additionally reports zero power and a matching common block against its pinned reference endpoint. These are local application and endpoint observations, not independent consensus signatures. Preserve the entire home on stop. The same staging applies to the operated validator, but its gateway needs the successor `gateway.py` beside the successor runtime helper; participant/full-node packages do not need that gateway.

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

The separate staged native test uses real signed SDK governance, fresh participant homes and the unmodified predecessor client. It exercises both restart choices, the automatic binary transition and a pre-upgrade commitment revealed after activation:

```sh
ZERONE_UPGRADE_PREDECESSOR_ROOT='/ABSOLUTE/PATH/TO/VERIFIED/PREDECESSOR/CHECKOUT' \
ZERONE_UPGRADE_TARGET_BINARY="$PWD/build/zeroned" \
python3 -I -B deploy/networks/zerone-dev-1/test_upgrade_native.py
```

Its accelerated fresh genesis, test keys and unused alternate-platform test pin are not a public release packet or independent participant demonstration.
