#!/usr/bin/env bash
# Fail-closed tombstone for the legacy Fly deployment path.

set -euo pipefail

cat >&2 <<'EOF'
FAIL: the zerone-testnet-1 deployment image is retired.

The running legacy network predates this consolidated source. Do not seed,
reset, resume, or replace its validator from current main. A future deployment
must use a signed release packet and the governance-scheduled
consolidation-safety-v1 upgrade.
EOF

exit 1
