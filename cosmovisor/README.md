# Cosmovisor Setup for Zerone

[Cosmovisor](https://docs.cosmos.network/v0.50/build/tooling/cosmovisor) is a process manager that watches for governance-approved upgrade proposals and swaps the `zeroned` binary automatically.

## Directory Structure

```
cosmovisor/
├── genesis/
│   └── bin/
│       └── zeroned          ← initial binary (copy or symlink)
└── upgrades/
    └── v1.0.0-testnet/
        └── bin/
            └── zeroned      ← upgraded binary (placed before upgrade height)
```

## Quick Start

### 1. Install Cosmovisor

```bash
go install cosmossdk.io/tools/cosmovisor/cmd/cosmovisor@latest
```

### 2. Initialize

```bash
make cosmovisor-init
```

Or manually:

```bash
cp $(go env GOPATH)/bin/zeroned cosmovisor/genesis/bin/zeroned
```

### 3. Set Environment Variables

```bash
export DAEMON_NAME=zeroned
export DAEMON_HOME=$HOME/.zeroned
export DAEMON_ALLOW_DOWNLOAD_BINARIES=false
export DAEMON_RESTART_AFTER_UPGRADE=true
export DAEMON_LOG_BUFFER_SIZE=512
```

### 4. Run

```bash
cosmovisor run start
```

## Preparing an Upgrade

1. Build the new binary: `make build`
2. Copy to the upgrade directory:
   ```bash
   cp build/zeroned cosmovisor/upgrades/v1.0.0-testnet/bin/zeroned
   ```
3. Submit a governance proposal with the upgrade plan name matching `v1.0.0-testnet`
4. Cosmovisor will swap binaries at the specified block height
