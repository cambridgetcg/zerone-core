#!/usr/bin/env bash
set -euo pipefail
umask 077

if [[ "${ZERONE_OPERATION_CONTEXT:-genesis}" == "recovery" ]]; then
  echo "mainnet-ceremony.sh is genesis-only and must not be used for validator recovery" >&2
  exit 1
fi

cat >&2 <<'EOF'
REFUSED: scripts/mainnet-ceremony.sh is a retired zerone-1 launch artifact.

zerone-1 is already live. This historical builder is not authority to
re-genesis, reset, or replace its keys. A new network requires its own signed,
release-bound ceremony and explicit GO decision.
EOF
exit 1
