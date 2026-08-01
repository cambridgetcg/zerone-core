#!/usr/bin/env bash
# Retired 2026-07-29: never reset a running chain from a moving source branch.

set -euo pipefail
umask 077

if [[ "${ZERONE_OPERATION_CONTEXT:-genesis}" == "recovery" ]]; then
  echo "regen-genesis.sh is genesis-only and must not be used for validator recovery" >&2
  exit 1
fi

cat >&2 <<'EOF'
FAIL: zerone-testnet-1 genesis regeneration is paused.

This script will not read identity material, rewrite artifacts, or prepare a
volume reset. Any reset requires a separately authorised, signed ceremony and
public transition notice.
EOF

exit 1
