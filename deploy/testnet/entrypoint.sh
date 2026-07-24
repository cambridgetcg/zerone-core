#!/usr/bin/env bash
# First-boot seeder for the zerone-testnet-1 fly node.
# On an empty volume: lay down the baked genesis + node key, take the
# validator signing key from a fly secret (falls back to the baked key for
# local docker runs), and tune config for a public single validator.
set -euo pipefail

HOME_DIR="${ZERONE_HOME:-/data/.zeroned}"
SEED="/testnet-seed"

if [ ! -f "${HOME_DIR}/config/genesis.json" ]; then
  echo "[entrypoint] fresh volume — seeding ${HOME_DIR}"
  zeroned init "${MONIKER:-zerone-testnet-fly}" --chain-id zerone-testnet-1 \
    --default-denom uzrn --home "${HOME_DIR}" >/dev/null 2>&1

  cp "${SEED}/genesis.json"  "${HOME_DIR}/config/genesis.json"
  cp "${SEED}/node_key.json" "${HOME_DIR}/config/node_key.json"

  # Validator signing key: prefer the fly secret, else the baked play key.
  if [ -n "${PRIV_VALIDATOR_KEY_B64:-}" ]; then
    echo "${PRIV_VALIDATOR_KEY_B64}" | base64 -d > "${HOME_DIR}/config/priv_validator_key.json"
    echo "[entrypoint] validator key from secret"
  else
    cp "${SEED}/priv_validator_key.json" "${HOME_DIR}/config/priv_validator_key.json"
    echo "[entrypoint] validator key from baked seed (play testnet)"
  fi

  CFG="${HOME_DIR}/config/config.toml"
  APP="${HOME_DIR}/config/app.toml"

  # RPC + P2P bind on all interfaces.
  sed -i 's|^laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://0.0.0.0:26657"|' "${CFG}"
  # A public single-validator node serves clients, not a strict peer mesh.
  sed -i 's|^addr_book_strict = true|addr_book_strict = false|' "${CFG}"
  sed -i 's|^allow_duplicate_ip = false|allow_duplicate_ip = true|' "${CFG}"
  sed -i 's|^cors_allowed_origins = \[\]|cors_allowed_origins = ["*"]|' "${CFG}"

  # REST already enabled by default; just make it public + CORS, and gRPC public.
  sed -i 's|^address = "tcp://localhost:1317"|address = "tcp://0.0.0.0:1317"|' "${APP}"
  sed -i 's|^enabled-unsafe-cors = false|enabled-unsafe-cors = true|' "${APP}"
  sed -i 's|^address = "localhost:9090"|address = "0.0.0.0:9090"|' "${APP}"
else
  echo "[entrypoint] existing volume — resuming"
fi

# Advertise the public P2P address on EVERY boot (the dedicated IP is known
# only after allocation, and peers must dial a routable address to sync).
if [ -n "${EXTERNAL_ADDRESS:-}" ]; then
  sed -i "s|^external_address = .*|external_address = \"${EXTERNAL_ADDRESS}\"|" "${HOME_DIR}/config/config.toml"
  echo "[entrypoint] external_address = ${EXTERNAL_ADDRESS}"
fi

# The node validates app.toml's minimum-gas-prices BEFORE the --minimum-gas-prices
# flag override, so it must be non-empty in app.toml. Set it on EVERY boot
# (handles a resumed volume too); replace an existing line or append if missing.
APP="${HOME_DIR}/config/app.toml"
if grep -q '^minimum-gas-prices' "${APP}"; then
  sed -i 's|^minimum-gas-prices = .*|minimum-gas-prices = "0.025uzrn"|' "${APP}"
else
  printf '\nminimum-gas-prices = "0.025uzrn"\n' >> "${APP}"
fi
echo "[entrypoint] $(grep '^minimum-gas-prices' "${APP}")"

# ── cosmovisor supervision ───────────────────────────────────────────────
# Without this, an upgrade height halts the chain and it stays halted until a
# human notices and redeploys — that is how a two-minute swap became a 28-hour
# outage once already. cosmovisor watches for the halt and restarts the node on
# the new binary by itself.
#
# The image binary stays the source of truth: it is re-installed on every boot,
# so "deploy an image" still means "that is the binary that runs", and rolling
# back by redeploying a previous image still works. cosmovisor only decides
# WHEN to swap, never what the deployed binary is.
CV="${HOME_DIR}/cosmovisor"
mkdir -p "${CV}/genesis/bin"
cp /usr/local/bin/zeroned "${CV}/genesis/bin/zeroned"

# x/upgrade writes data/upgrade-info.json when a plan executes. cosmovisor reads
# it on every start and expects a binary staged at upgrades/<name>/bin. If none
# is there it tries to download one from the plan's `info` field — and our plans
# carry an empty info, which makes cosmovisor exit with "plan info must not be
# blank" on a loop.
#
# Downloading is the wrong answer anyway: the deployed image already contains
# the binary we intend to run. Stage it under the upgrade name so cosmovisor is
# satisfied without reaching out to the network, and so a node that has already
# passed an upgrade can still boot.
UINFO="${HOME_DIR}/data/upgrade-info.json"
if [ -f "${UINFO}" ]; then
  UNAME="$(jq -r '.name // empty' "${UINFO}" 2>/dev/null || true)"
  if [ -n "${UNAME}" ]; then
    mkdir -p "${CV}/upgrades/${UNAME}/bin"
    cp /usr/local/bin/zeroned "${CV}/upgrades/${UNAME}/bin/zeroned"
    echo "[entrypoint] staged image binary for upgrade '${UNAME}'"
  fi
fi

# If an upgrade already ran, cosmovisor's `current` points into upgrades/<name>.
# Re-assert the image binary there too, otherwise a redeploy would silently
# leave the old downloaded binary running and the image would stop being truth.
if [ -L "${CV}/current" ]; then
  CUR_TARGET="$(readlink -f "${CV}/current" || true)"
  if [ -n "${CUR_TARGET}" ] && [ -d "${CUR_TARGET}/bin" ] && [ "${CUR_TARGET}" != "${CV}/genesis" ]; then
    cp /usr/local/bin/zeroned "${CUR_TARGET}/bin/zeroned"
    echo "[entrypoint] refreshed upgrade binary at ${CUR_TARGET}/bin"
  fi
fi

export DAEMON_NAME=zeroned
export DAEMON_HOME="${HOME_DIR}"
export DAEMON_RESTART_AFTER_UPGRADE=true
# Off by design: binaries come from the deployed image, which is reviewed and
# reproducible, not from a URL fetched by a validator at halt time. Staging
# above is what makes this safe to leave disabled.
export DAEMON_ALLOW_DOWNLOAD_BINARIES="${DAEMON_ALLOW_DOWNLOAD_BINARIES:-false}"
export UNSAFE_SKIP_BACKUP="${UNSAFE_SKIP_BACKUP:-true}"
echo "[entrypoint] cosmovisor: download=${DAEMON_ALLOW_DOWNLOAD_BINARIES} restart_after_upgrade=true"

exec cosmovisor run start --home "${HOME_DIR}" --minimum-gas-prices 0.025uzrn ${EXTRA_START_FLAGS:-}
