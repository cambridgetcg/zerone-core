#!/usr/bin/env bash
set -euo pipefail

export ZERONE_CHAIN_ID="zerone-1"
export ZERONE_SEED="/mainnet-seed"
export ZERONE_DEFAULT_MONIKER="zerone-1-fly"
export ZERONE_GENESIS_SHA256="c30a523b9764fb76c84a53d99fcdabb966d16e7a4d3f15426ab7af5e8576170e"

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
