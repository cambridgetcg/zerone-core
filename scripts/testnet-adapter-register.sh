#!/usr/bin/env bash
# Retired 2026-07-29: this path formerly submitted and voted a live proposal.

set -euo pipefail

cat >&2 <<'EOF'
FAIL: legacy-testnet adapter registration is paused.

This script will not read operator keys, submit a proposal, cast a vote, or
broadcast any transaction. Adapter activation requires a release-compatible,
explicitly authorised governance packet.

Read:
  networks/zerone-testnet-1/README.md
  docs/VALIDATOR-GUIDE.md
EOF

exit 1
