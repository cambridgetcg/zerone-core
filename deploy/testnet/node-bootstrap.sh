#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════════════════
# zerone-testnet-1 node bootstrap — fresh Ubuntu/Debian VM → synced node
# ═══════════════════════════════════════════════════════════════════════════
# Installs Go + build tools, builds zeroned, pulls the live genesis, wires the
# seed + gas floor, installs a systemd unit, and starts syncing. Idempotent-ish:
# re-running rebuilds and re-seeds config but keeps your data/keys.
#
# READ THIS BEFORE RUNNING. It uses sudo (packages, /usr/local, systemd).
# Full walkthrough: deploy/testnet/RUN-A-NODE.md
#
# Env overrides:
#   MONIKER      node name (default: hostname)
#   RPC          a network RPC to pull genesis + seed from (default the public node)
#   GO_VERSION   Go toolchain (default 1.24.0)
# ═══════════════════════════════════════════════════════════════════════════
set -euo pipefail

MONIKER="${MONIKER:-$(hostname)}"
RPC="${RPC:-http://37.16.28.121:26657}"
GO_VERSION="${GO_VERSION:-1.24.0}"
CHAIN_ID="zerone-testnet-1"
SEED="9a9c6b9d36c55d21c32b1ee8749adf8dd7c6b0d4@37.16.28.121:26656"
REPO="https://github.com/cambridgetcg/zerone-core"
HOME_DIR="${HOME}/.zeroned"

say() { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
die() { printf '\033[1;31mFAIL\033[0m %s\n' "$*" >&2; exit 1; }

command -v sudo >/dev/null || die "sudo required"
case "$(uname -m)" in
  x86_64|amd64) GOARCH=amd64 ;;
  aarch64|arm64) GOARCH=arm64 ;;
  *) die "unsupported arch $(uname -m)" ;;
esac

say "installing build deps"
sudo apt-get update -y
sudo apt-get install -y git build-essential jq curl

if ! command -v go >/dev/null || ! go version 2>/dev/null | grep -q "go${GO_VERSION%.*}"; then
  say "installing Go ${GO_VERSION} (${GOARCH})"
  curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${GOARCH}.tar.gz" | sudo tar -C /usr/local -xz
fi
export PATH="$PATH:/usr/local/go/bin"

say "building zeroned (this takes a few minutes)"
SRC="${HOME}/zerone-core"
if [ -d "${SRC}/.git" ]; then git -C "${SRC}" pull --ff-only; else git clone "${REPO}" "${SRC}"; fi
( cd "${SRC}" && make build )
sudo install "${SRC}/build/zeroned" /usr/local/bin/zeroned
say "zeroned $(zeroned version 2>/dev/null || echo built)"

if [ ! -f "${HOME_DIR}/config/genesis.json" ]; then
  say "initialising node home"
  zeroned init "${MONIKER}" --chain-id "${CHAIN_ID}" --default-denom uzrn >/dev/null 2>&1
fi

say "fetching live genesis from ${RPC}"
curl -fsSL "${RPC}/genesis" | jq .result.genesis > "${HOME_DIR}/config/genesis.json"
GENHASH=$(sha256sum "${HOME_DIR}/config/genesis.json" | awk '{print $1}')
say "genesis sha256: ${GENHASH}  (compare against deploy/testnet/JOIN.md)"

say "wiring seed + gas floor"
CFG="${HOME_DIR}/config/config.toml"; APP="${HOME_DIR}/config/app.toml"
sed -i "s|^seeds = .*|seeds = \"${SEED}\"|" "${CFG}"
if grep -q '^minimum-gas-prices' "${APP}"; then
  sed -i 's|^minimum-gas-prices = .*|minimum-gas-prices = "0.025uzrn"|' "${APP}"
else
  printf '\nminimum-gas-prices = "0.025uzrn"\n' >> "${APP}"
fi

say "installing systemd unit"
sudo tee /etc/systemd/system/zeroned.service >/dev/null <<UNIT
[Unit]
Description=zerone node
After=network-online.target
[Service]
User=${USER}
ExecStart=/usr/local/bin/zeroned start --minimum-gas-prices 0.025uzrn
Restart=on-failure
RestartSec=3
LimitNOFILE=65535
[Install]
WantedBy=multi-user.target
UNIT
sudo systemctl daemon-reload
sudo systemctl enable --now zeroned

say "node started. watch it sync:"
echo "    journalctl -u zeroned -f"
echo "    zeroned status | jq '{height: .sync_info.latest_block_height, catching_up: .sync_info.catching_up}'"
echo
say "when catching_up=false you're at the tip and verifying every block yourself."
echo "Become a validator or snapshot to free storage: deploy/testnet/RUN-A-NODE.md"
