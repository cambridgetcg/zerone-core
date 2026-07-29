#!/usr/bin/env bash
# Fail-closed mirror of deploy/testnet/node-bootstrap.sh.

set -euo pipefail

cat >&2 <<'EOF'
FAIL: Zerone live-node bootstrap is paused.

The live networks predate the consolidated source head. Do not build and
install current main outside a signed, governance-scheduled upgrade packet.
Read this skill's SKILL.md and references/operator-guide.md.
EOF

exit 1
