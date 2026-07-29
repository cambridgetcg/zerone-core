#!/usr/bin/env bash
# Retired 2026-07-29. Historical genesis material remains in git history.

set -euo pipefail

cat >&2 <<'EOF'
FAIL: generating a replacement zerone-testnet-1 genesis is paused.

The chain ID is already live as a legacy network. Reusing it with a new
genesis requires an explicit reset decision and a signed ceremony packet; this
source publication grants neither.
EOF

exit 1
