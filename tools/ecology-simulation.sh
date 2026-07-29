#!/usr/bin/env bash
set -euo pipefail

cat >&2 <<'EOF'
REFUSED: tools/ecology-simulation.sh is a retired interactive chain demo.

It previously inherited implicit CLI and REST targets and could mutate a live
configured network. Use the deterministic cross-stack ecology tests or an
explicit disposable localnet rehearsal instead.
EOF
exit 1
