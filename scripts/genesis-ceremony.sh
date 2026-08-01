#!/usr/bin/env bash
set -euo pipefail
umask 077

if [[ "${ZERONE_OPERATION_CONTEXT:-genesis}" == "recovery" ]]; then
  echo "REFUSED: this retired genesis ceremony must not be used for validator recovery." >&2
  exit 1
fi

cat >&2 <<'EOF'
REFUSED: scripts/genesis-ceremony.sh is a retired pre-launch ceremony.

It must not compete with the immutable genesis of an already-running network.
Use a network-specific, release-bound ceremony packet for a genuinely new
chain. The 2026-07 source consolidation authorizes no genesis or reset.
EOF
exit 1
