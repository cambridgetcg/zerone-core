#!/usr/bin/env bash
# Retired 2026-07-29: cached callers must fail before touching keys or chain.

set -euo pipefail

cat >&2 <<'EOF'
FAIL: zerone-testnet-1 onboarding is paused for this source head.

This script will not inspect keys, fund an account, create a home, register a
key, or broadcast a transaction. The live legacy testnet predates
consolidation-safety-v1 and is observe-only until a current onboarding packet
is published.

Read:
  networks/zerone-testnet-1/README.md
  skills/zerone-onboarding/SKILL.md
EOF

exit 1
