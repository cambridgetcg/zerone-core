#!/usr/bin/env bash
set -euo pipefail

export ZERONE_CHAIN_ID="zerone-testnet-1"
export ZERONE_SEED="/testnet-seed"
export ZERONE_DEFAULT_MONIKER="zerone-testnet-fly"
export ZERONE_GENESIS_SHA256="a2a5499fcd43668f328b0ab504ad9f7c3aadd65f7abd8a4f3991b927872a6a2a"

common_entrypoint="/usr/local/libexec/zerone-fly-validator-entrypoint"
if [[ ! -x "${common_entrypoint}" ]]; then
  script_directory="${BASH_SOURCE[0]%/*}"
  if [[ "${script_directory}" == "${BASH_SOURCE[0]}" ]]; then
    script_directory="."
  fi
  export ZERONE_SEED="${script_directory}/artifacts"
  common_entrypoint="${script_directory}/../fly-validator-entrypoint-common.sh"
fi
exec "${common_entrypoint}"
