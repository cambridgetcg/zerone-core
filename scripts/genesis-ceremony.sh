#!/usr/bin/env bash
set -euo pipefail

cat >&2 <<'EOF'
REFUSED: scripts/genesis-ceremony.sh is a retired pre-launch ceremony.

It must not compete with the immutable genesis of an already-running network.
Use a network-specific, release-bound ceremony packet for a genuinely new
chain. The 2026-07 source consolidation authorizes no genesis or reset.
EOF
exit 1
