#!/usr/bin/env bash
set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
bundle_directory="${repository_root}/integrations/chain-registry/zerone"
registry_commit="ecf22848fa4cdd2efefac6cda0e1552c61e2702b"

if (( $# > 1 )); then
  echo "usage: $0 [chain-registry-checkout]" >&2
  exit 2
fi

if (( $# == 1 )); then
  registry_directory="$(cd "$1" && pwd)"
else
  validation_directory="$(mktemp -d /tmp/zerone-chain-registry-validate.XXXXXX)"
  registry_directory="${validation_directory}/chain-registry"
  git init --quiet "$registry_directory"
  git -C "$registry_directory" remote add origin https://github.com/cosmos/chain-registry.git
  git -C "$registry_directory" fetch --quiet --depth 1 origin "$registry_commit"
  git -C "$registry_directory" checkout --quiet --detach FETCH_HEAD
fi

if [[ -e "${registry_directory}/zerone" ]]; then
  echo "refusing to overwrite existing ${registry_directory}/zerone" >&2
  exit 1
fi

cp -R "$bundle_directory" "${registry_directory}/zerone"
(
  cd "$registry_directory"
  npx --yes @chain-registry/cli@1.47.0 validate --registryDir . --logLevel error
)
