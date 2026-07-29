#!/usr/bin/env bash
# Retired 2026-07-29. Keep this path as a fail-closed stub so cached commands
# cannot silently configure current main against the legacy testnet.

set -euo pipefail

cat >&2 <<'EOF'
FAIL: joining zerone-testnet-1 from current main is paused.

The live legacy testnet predates this consolidated source and has not activated
consolidation-safety-v1. This script will not initialize, reset, download,
configure, or start a node.

Read:
  networks/zerone-testnet-1/README.md
  deploy/testnet/JOIN.md
  docs/VALIDATOR-GUIDE.md
EOF

exit 1
