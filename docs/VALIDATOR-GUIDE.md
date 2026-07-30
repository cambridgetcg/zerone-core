# Zerone Validator Guide

This guide covers everything you need to join **zerone-testnet-1** as a validator.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Node Setup](#node-setup)
- [Becoming a Validator](#becoming-a-validator)
- [Validator Tiers](#validator-tiers)
- [Proof of Truth Participation](#proof-of-truth-participation)
- [Cosmovisor Setup](#cosmovisor-setup)
- [Monitoring](#monitoring)
- [Troubleshooting](#troubleshooting)

---

## Prerequisites

### Hardware Requirements

| Component | Minimum | Recommended |
|-----------|---------|-------------|
| CPU       | 4 cores | 8+ cores    |
| RAM       | 16 GB   | 32 GB       |
| Disk      | 500 GB SSD | 1 TB NVMe |
| Network   | 100 Mbps | 1 Gbps     |

### Software Requirements

- **Go** 1.25.12 exactly ([install guide](https://go.dev/doc/install))
- **jq** 1.6+ (`brew install jq` on macOS, `apt install jq` on Ubuntu)
- **make**
- **git**
- **OS**: Ubuntu 22.04+ or macOS 13+

---

## Installation

### Build from source

```bash
git clone https://github.com/zerone-chain/zerone.git
cd zerone
make install
```

This installs the `zeroned` binary to `$(go env GOPATH)/bin/zeroned`.

Verify:

```bash
zeroned version
```

### Docker (easiest)

Build and run with Docker — no Go toolchain required:

```bash
# Build the image
docker build -t zerone:latest .

# Verify
docker run --rm zerone:latest version

# Initialize and run a node
docker run -v ~/.zeroned:/root/.zeroned zerone:latest init my-node --chain-id zerone-testnet-1
docker compose up -d
```

For validators using the pinned Cosmovisor supervisor:

```bash
export ZERONE_VERSION='vX.Y.Z' # exact governed semantic version
export ZERONE_COMMIT='<40-lowercase-reviewed-source-commit>'
export ZERONE_SOURCE_DATE_EPOCH='<positive-reviewed-unix-epoch>'

docker build \
  --build-arg VERSION="${ZERONE_VERSION}" \
  --build-arg COMMIT="${ZERONE_COMMIT}" \
  --build-arg SOURCE_DATE_EPOCH="${ZERONE_SOURCE_DATE_EPOCH}" \
  -f Dockerfile.validator \
  -t "zerone-validator:${ZERONE_VERSION}" .
```

### Pre-built binary

Pin an exact release tag and platform artifact. Never install from a mutable
`latest` URL on a validator:

```bash
export ZERONE_VERSION='vX.Y.Z' # replace with the governed, exact release tag
export ZERONE_ARTIFACT='zeroned-linux-amd64' # or the exact arm64/darwin artifact
export ZERONE_RELEASE_BASE="https://github.com/zerone-chain/zerone/releases/download/${ZERONE_VERSION}"

curl --fail --location --proto '=https' --tlsv1.2 \
  "${ZERONE_RELEASE_BASE}/${ZERONE_ARTIFACT}" \
  --output "${ZERONE_ARTIFACT}"

# Obtain this value from the separately authenticated, signed release
# manifest/provenance named by the upgrade operations manifest.
export ZERONE_BINARY_SHA256='<64-lowercase-hex-digest>'
if command -v sha256sum >/dev/null 2>&1; then
  ZERONE_ACTUAL_SHA256="$(sha256sum "${ZERONE_ARTIFACT}" | awk '{print $1}')"
else
  ZERONE_ACTUAL_SHA256="$(shasum -a 256 "${ZERONE_ARTIFACT}" | awk '{print $1}')"
fi
test "${ZERONE_ACTUAL_SHA256}" = "${ZERONE_BINARY_SHA256}"

chmod 0755 "${ZERONE_ARTIFACT}"
sudo install -m 0755 "${ZERONE_ARTIFACT}" /usr/local/bin/zeroned
zeroned version
```

Do not trust a checksum merely because it is downloaded beside the binary.
Verify the signed release manifest/provenance, source commit, upgrade name,
and target height through an independent channel as described in the
[canonical operations runbook](UPGRADE_AND_INCIDENT_OPERATIONS.md).

---

## Node Setup

### Option A: Automated (recommended)

Use the join-testnet script:

```bash
scripts/join-testnet.sh --moniker "my-validator"
```

With Cosmovisor and systemd:

```bash
scripts/join-testnet.sh \
  --moniker "my-validator" \
  --genesis /path/to/genesis.json \
  --cosmovisor \
  --systemd
```

See `scripts/join-testnet.sh --help` for all options.

### Option B: Manual

#### 1. Initialize the node

```bash
zeroned init "my-validator" --chain-id zerone-testnet-1
```

This creates the default config at `$HOME/.zeroned/`.

#### 2. Install the genesis file

Copy the official genesis file to your config directory:

```bash
cp genesis.json $HOME/.zeroned/config/genesis.json
zeroned genesis validate
```

#### 3. Configure seed nodes

Add seed nodes from [`seeds.txt`](../seeds.txt) to your `config.toml`:

```toml
# $HOME/.zeroned/config/config.toml
[p2p]
seeds = "node-id1@ip1:26656,node-id2@ip2:26656"
```

#### 4. Set minimum gas price

```toml
# $HOME/.zeroned/config/app.toml
minimum-gas-prices = "0.025uzrn"
```

See [`config/config.toml.template`](../config/config.toml.template) and
[`config/app.toml.template`](../config/app.toml.template) for full
recommended configurations.

#### 5. Start the node

```bash
zeroned start --minimum-gas-prices 0.025uzrn
```

Wait for your node to fully sync before registering as a validator:

```bash
zeroned status | jq '.sync_info.catching_up'
# Should return: false
```

---

## Becoming a Validator

Zerone currently has two independent staking systems. Cosmos SDK
`x/staking` controls the CometBFT block-producing validator set.
`x/zerone_staking` controls Proof-of-Truth tiers and Guardian eligibility.
An external validator that needs both roles must register in both; neither
record creates the other.

### Step 1: Create a key

```bash
zeroned keys add my-validator
```

Save the mnemonic securely. Your address will have the `zrn1...` prefix.

### Step 2: Fund your account

Obtain testnet ZRN from the faucet or another validator. You need enough to
cover your self-delegation plus transaction fees.

### Step 3: Register your account

The one-shot onboarding command generates a separate Ed25519 identity key,
derives its `did:zrn` identifier, saves the private identity locally, and
registers the public identity on-chain:

```bash
zeroned tx zerone_auth onboard human \
  --identity-out "$HOME/.zeroned/identities/my-validator.ed25519.json" \
  --from my-validator \
  --chain-id zerone-testnet-1 \
  --fees 5000uzrn
```

Keep that identity file separate from the transaction key and consensus key.
The valid account types are `agent`, `human`, `contract`, and `system`; there
is no `validator` account type.

### Step 4: Create the consensus validator

Cosmos SDK v0.53 takes a reviewed JSON file. The `pubkey` is the exact JSON
object emitted by the new node's Comet consensus key:

```bash
zeroned comet show-validator > validator-pubkey.json

jq -n \
  --slurpfile pubkey validator-pubkey.json \
  '{
    pubkey: $pubkey[0],
    amount: "1000000uzrn",
    moniker: "My Validator",
    identity: "",
    website: "https://example.com",
    security: "",
    details: "A Zerone validator",
    "commission-rate": "0.05",
    "commission-max-rate": "0.20",
    "commission-max-change-rate": "0.01",
    "min-self-delegation": "1"
  }' > validator.json

zeroned tx staking create-validator validator.json \
  --from my-validator \
  --chain-id zerone-testnet-1 \
  --gas auto --gas-adjustment 1.4 \
  --fees 5000uzrn
```

Hash and review both JSON files before broadcast. Then query the resulting
`zrnvaloper1...` record and the Comet validator set at fixed heights; a
successful transaction alone does not prove that the intended public key and
power are active.

### Step 5: Register for Proof of Truth

The custom CLI takes a lowercase hex public key and a raw integer number of
`uzrn`, not a coin string. This is a second, separate self-delegation:

```bash
CONSENSUS_PUBKEY_HEX="$(
  zeroned comet show-validator |
    jq -r .key |
    openssl base64 -d -A |
    xxd -p -c 256
)"

zeroned tx zerone_staking register-validator \
  "${CONSENSUS_PUBKEY_HEX}" \
  111000 \
  --commission 500 \
  --moniker "My Validator" \
  --identity "did:zrn:<identity-public-key-hex>" \
  --website "https://example.com" \
  --details "A Zerone validator" \
  --from my-validator \
  --chain-id zerone-testnet-1 \
  --fees 5000uzrn
```

Arguments:
- `pubkey-hex` - raw 32-byte CometBFT consensus public key as 64 lowercase
  hex characters
- `self-delegation` - raw integer amount in `uzrn` (for example, `111000`
  for the Apprentice minimum)

Flags:
- `--commission` - Commission rate in basis points (BPS). Max 10,000 (= 100%). Example: `500` = 5%
- `--moniker` - Human-readable validator name (max 70 characters)
- `--identity` - DID for validator identity (max 128 characters)
- `--website` - Website URL (max 140 characters)
- `--details` - Description (max 2,000 characters)

The custom record does not make the node a Comet block producer, and the SDK
record does not make it a Proof-of-Truth validator. The custom commission is
basis points, while the SDK commission fields are decimal fractions. The
custom module currently checks only that `consensus_pubkey` is non-empty; it
does not decode the hex, prove key possession, or bind the field to the SDK
validator. Treat it as locally verified metadata, not consensus identity
proof. See the [key-replacement
runbook](UPGRADE_AND_INCIDENT_OPERATIONS.md#1141-one-validator-key-replacement)
before changing either identity on a live validator.

### Fees start at block 1

The former 480,000-block gas-free bootstrap window is retired:
`app.BootstrapEndBlock` is `0`. Validators must configure and publish a
non-zero `minimum-gas-prices` policy from launch. Any onboarding subsidy must
use an explicit, auditable mechanism such as fee grants; do not assume the
listed Proof-of-Truth or registration messages are generally fee-free.

---

## Validator Tiers

<!-- Source: x/staking/types/types.go:66-132 -->

Zerone validators are organized into four tiers with increasing requirements
and rewards:

| Tier | Min Stake | Min Reputation | Min Verifications | Reward Multiplier | Selection Weight |
|------|-----------|----------------|-------------------|-------------------|-----------------|
| **Apprentice** | 111,000 uzrn (0.111 ZRN) | -- | -- | 0.1x | 0.1x |
| **Verified** | 1,110,000 uzrn (1.11 ZRN) | 77% | 22 | 0.5x | 0.5x |
| **Scholar** | 1,111,000,000 uzrn (1,111 ZRN) | 50% | 11 | 1.0x | 1.0x |
| **Guardian** | 11,111,000,000 uzrn (11,111 ZRN) | 77% | 333 | 2.0x | 1.5x |

### Tier details

**Apprentice** — Entry tier. Can verify `protocol`, `computational`, and
`formal` claims. Higher slash multiplier (1.5x) to discourage Sybil
behavior. Maximum 111 Apprentice validators (Sybil cap).

**Verified** — Proven verifiers with 77% accuracy and 22+ verifications.
Gains access to `empirical` claims. Slash multiplier 1.2x.

**Scholar** — Block-producing tier. Requires substantial stake (1,111 ZRN).
Full category access including `oracle` and `attestation`. Standard slash
multiplier (1.0x). Subject to `MaxValidators` cap (default: 100).

**Guardian** — Highest tier. Requires 11,111 ZRN stake, 77% accuracy, and
333 verifications. Zero tolerance for slashing. Gains access to `predictive`,
`social`, and `contested` categories. 2.0x reward multiplier with 3x
contested verification multiplier.

### Tier progression

Your tier is computed automatically based on your current stake, reputation
score, verification count, and accuracy. Increase your self-delegation with:

```bash
zeroned tx zerone_staking update-stake 1000000000 \
  --increase \
  --from my-validator \
  --chain-id zerone-testnet-1
```

The custom amount is again a raw integer in `uzrn`. This transaction does not
change Cosmos SDK consensus power.

---

## Proof of Truth Participation

Zerone validators participate in **Proof of Truth (PoT)** consensus — a
three-phase knowledge verification process.

### Verification phases

<!-- Source: x/knowledge/types/genesis.go:7-16 -->

1. **Commit phase** (10 blocks) — Validators submit a hash commitment of
   their verification judgment.
2. **Reveal phase** (10 blocks) — Validators reveal their actual judgment.
   Missing a reveal is slashed at 10%.
3. **Aggregation phase** (5 blocks) — The network aggregates all revealed
   judgments to determine the claim's truth status.

### Claim lifecycle

A claim is submitted with a minimum stake of 1 ZRN. Between 3 and 22
verifiers are selected based on tier and reputation. If the aggregated
confidence reaches 77%, the claim is accepted as a fact. Below 30%, it is
rejected. Between 30% and 50%, it enters a provisional state that can be
challenged.

### Slashing

<!-- Source: x/knowledge/types/genesis.go:23-27 -->

| Offense | Slash Rate |
|---------|-----------|
| Wrong verification | 5% of stake |
| Missed reveal | 10% of stake |
| Equivocation (conflicting votes) | 20% of stake |
| Invalid claim submission | 22% of stake |

### Rewards

Correct verifications earn 3 ZRN per round (decaying by 0.1% per epoch).
Citation economics distribute 15% of rewards to cited fact authors, with a
20% bonus for cross-domain citations.

---

## Cosmovisor Setup

Cosmovisor switches to a previously staged binary when an on-chain
`x/upgrade` plan reaches H. An old binary without the handler stops before H
commits, leaving H−1 as the last committed state; the staged binary starts
from H−1 and may commit H. Once H commits, recover forward with a new named
upgrade or an explicitly authorized fork—never by restarting the old binary
as if H did not happen.

Use this section with the canonical [Upgrade and Incident Operations
runbook](UPGRADE_AND_INCIDENT_OPERATIONS.md) and the full
[`cosmovisor/README.md`](../cosmovisor/README.md). An `x/emergency`
transaction quarantine is separate from this planned upgrade boundary and
does not stop block production.

Quick setup:

```bash
# Install the version exercised by this Zerone release; never use @latest on
# a validator.
go install cosmossdk.io/tools/cosmovisor/cmd/cosmovisor@v1.7.1
go version -m "$(command -v cosmovisor)" |
  awk '$1 == "mod" && $2 == "cosmossdk.io/tools/cosmovisor" { print $2, $3 }'

export DAEMON_NAME=zeroned
export DAEMON_HOME=$HOME/.zeroned
export DAEMON_ALLOW_DOWNLOAD_BINARIES=false
export DAEMON_DOWNLOAD_MUST_HAVE_CHECKSUM=true
export DAEMON_RESTART_AFTER_UPGRADE=true
export DAEMON_LOG_BUFFER_SIZE=512
export UNSAFE_SKIP_BACKUP=false

# Initialize the configured daemon home with the exact installed genesis
# binary. The repository Make target writes a repository-local test tree and
# is not a substitute for validator initialization.
cosmovisor init "$(command -v zeroned)"

# Start via cosmovisor
cosmovisor run start
```

The version check must report
`cosmossdk.io/tools/cosmovisor v1.7.1`. Keep binary downloads disabled and
stage the exact independently verified release at
`$DAEMON_HOME/cosmovisor/upgrades/<upgrade-name>/bin/zeroned` before H.
Compare its SHA-256 to the authenticated release manifest on every validator.
`DAEMON_DOWNLOAD_MUST_HAVE_CHECKSUM=true` is defense in depth if download
policy is ever changed; it does not verify a manually staged binary.
`UNSAFE_SKIP_BACKUP=false` retains Cosmovisor's pre-upgrade data backup, so
verify the backup location has sufficient free space during rehearsal.

For `Dockerfile.validator`, bind-mount a persistent `DAEMON_HOME` whose
`config/genesis.json`, peers, key material, and node configuration were
prepared and independently verified before startup. The image entrypoint
copies its pinned `/usr/local/bin/zeroned` into
`cosmovisor/genesis/bin/zeroned` only when that path is absent; it refuses to
overwrite an existing non-executable path. It does not create or trust a
genesis file for you.

Or use the join script with `--cosmovisor` flag:

```bash
scripts/join-testnet.sh --moniker "my-validator" --cosmovisor
```

---

## Monitoring

### Prometheus metrics

Both CometBFT and the application expose Prometheus metrics when enabled in
the configuration templates:

- **CometBFT metrics**: `http://localhost:26660/metrics`
- **App telemetry**: enabled via `app.toml` telemetry section

### Key metrics to watch

| Metric | Description |
|--------|-------------|
| `cometbft_consensus_height` | Current block height |
| `cometbft_consensus_validators` | Number of active validators |
| `cometbft_consensus_missing_validators` | Validators missing from last block |
| `cometbft_consensus_rounds` | Number of rounds in current height |
| `cometbft_p2p_peers` | Number of connected peers |
| `cometbft_mempool_size` | Transactions in mempool |

### Status checks

```bash
# Node sync status
zeroned status | jq '.sync_info'

# Validator info
zeroned query staking validators --output json | jq

# Your validator's signing info
zeroned query slashing signing-info "$(zeroned comet show-validator)"
```

---

## Troubleshooting

### Node won't start

**"Genesis file not found"**
```bash
# Verify genesis file exists
ls -la $HOME/.zeroned/config/genesis.json

# Re-validate
zeroned genesis validate
```

**"Address already in use"**

Another process is using port 26656/26657. Check and kill it:
```bash
lsof -i :26656
lsof -i :26657
```

### Node is not syncing

**Check peers:**
```bash
zeroned status | jq '.node_info.network'
curl -s localhost:26657/net_info | jq '.result.n_peers'
```

If zero peers, verify your `seeds` and `persistent_peers` in `config.toml`
and ensure port 26656 is open in your firewall.

**State sync (fast catch-up):**

If the chain has been running for a while, enable state sync in
`config.toml` to bootstrap quickly. You need a trusted block height and hash
from an RPC node:

```bash
# Get a recent block
LATEST=$(curl -s https://rpc.zerone.network/block | jq -r '.result.block.header.height')
TRUST_HEIGHT=$((LATEST - 2000))
TRUST_HASH=$(curl -s "https://rpc.zerone.network/block?height=${TRUST_HEIGHT}" | jq -r '.result.block_id.hash')
```

Then set in `config.toml`:
```toml
[statesync]
enable = true
rpc_servers = "https://rpc.zerone.network:443,https://rpc2.zerone.network:443"
trust_height = <TRUST_HEIGHT>
trust_hash = "<TRUST_HASH>"
```

### Validator is jailed

If your validator misses too many blocks, it may be jailed. To unjail:

```bash
zeroned tx slashing unjail \
  --from my-validator \
  --chain-id zerone-testnet-1 \
  --fees 5000uzrn
```

**"Transactions are not being processed"**

SDK v0.50 defaults `max-txs = -1` in `app.toml`, which activates the
NoOpMempool (silently dropping all transactions). Fix:

```toml
# $HOME/.zeroned/config/app.toml
[mempool]
max-txs = 5000
```

Or run `configure-node.sh` which sets this automatically.

### Common errors

| Error | Cause | Fix |
|-------|-------|-----|
| `validator already registered` | Duplicate registration | Your validator is already active |
| `insufficient funds` | Not enough ZRN | Fund your account before registering |
| `commission exceeds 100%` | `--commission` > 10,000 | Use BPS: 500 = 5%, 1000 = 10% |
| `moniker too long` | > 70 characters | Shorten your moniker |
| `min self delegation not met` | Stake below 111,000 uzrn | Increase your self-delegation |

### Restarting safely

```bash
# Graceful stop
kill -TERM $(pgrep zeroned)

# Or if using systemd
sudo systemctl stop zeroned

# Start again
zeroned start --minimum-gas-prices 0.025uzrn
```

---

## Further Reading

- [Parameters Reference](PARAMETERS.md) — All governance-adjustable parameters
- [FAQ](FAQ.md) — Frequently asked questions
- [Cosmovisor Setup](../cosmovisor/README.md) — Pinned upgrade supervision
