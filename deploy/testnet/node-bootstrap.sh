#!/usr/bin/env bash
# Retired 2026-07-29: the live networks predate the consolidated source head.
# A current-main binary must not be installed onto either network without its
# release-bound upgrade packet.

set -euo pipefail

cat >&2 <<'EOF'
FAIL: automatic live-network bootstrap is paused.

zerone-testnet-1 and zerone-1 are already-running networks. The consolidated
main branch contains consensus-sensitive changes that require a scheduled
upgrade; building latest main and installing it directly is unsafe.

Read:
  deploy/testnet/JOIN.md
  networks/zerone-testnet-1/README.md
  docs/VALIDATOR-GUIDE.md

Use only a future join packet that pins the release commit, binary digest,
genesis representation, peer identities, and approved upgrade height.
EOF

exit 1
