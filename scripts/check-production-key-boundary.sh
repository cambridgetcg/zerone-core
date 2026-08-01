#!/usr/bin/env bash
# Static repository boundary for production validator key material.
set -euo pipefail

repository_root="$(git rev-parse --show-toplevel)"
cd "${repository_root}"

failed=0
while IFS= read -r -d '' path; do
  case "${path##*/}" in
    priv_validator_key.json | priv_validator_state.json | node_key.json | *.mnemonic | *.seed)
      printf 'tracked secret-shaped validator artifact: %s\n' "${path}" >&2
      failed=1
      ;;
  esac
  case "/${path}/" in
    */keyring*/*)
      printf 'tracked validator keyring artifact: %s\n' "${path}" >&2
      failed=1
      ;;
  esac
done < <(git ls-files -z)

required_docker_denies=(
  '**/priv_validator_key.json'
  '**/priv_validator_state.json'
  '**/node_key.json'
  '**/*.mnemonic'
  '**/*.seed'
  '**/keyring*/'
)
for pattern in "${required_docker_denies[@]}"; do
  if ! grep -Fxq "${pattern}" .dockerignore; then
    printf 'missing .dockerignore validator-key denial: %s\n' "${pattern}" >&2
    failed=1
  fi
done

genesis_only_scripts=(
  deploy/mainnet/make-genesis.sh
  deploy/testnet/make-genesis.sh
  deploy/testnet/regen-genesis.sh
  scripts/genesis-ceremony.sh
  scripts/mainnet-ceremony.sh
)
for path in "${genesis_only_scripts[@]}"; do
  if ! grep -Fxq 'umask 077' "${path}"; then
    printf 'genesis key-handling script lacks umask 077: %s\n' "${path}" >&2
    failed=1
  fi
  if ! grep -Fq 'ZERONE_OPERATION_CONTEXT' "${path}" ||
    ! grep -Fq 'must not be used for validator recovery' "${path}"; then
    printf 'genesis key-handling script lacks recovery refusal: %s\n' "${path}" >&2
    failed=1
  fi
done

if ((failed != 0)); then
  exit 1
fi

printf 'production validator-key boundary passed\n'
