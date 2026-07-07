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

exec zeroned start --home "${HOME_DIR}" --minimum-gas-prices 0.025uzrn
