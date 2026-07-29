#!/usr/bin/env bash
# Retired 2026-07-29: never reset a running chain from a moving source branch.

set -euo pipefail

cat >&2 <<'EOF'
FAIL: zerone-testnet-1 genesis regeneration is paused.

This script will not read identity material, rewrite artifacts, or prepare a
volume reset. Any reset requires a separately authorised, signed ceremony and
public transition notice.
EOF

exit 1
