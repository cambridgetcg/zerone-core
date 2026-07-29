#!/usr/bin/env bash
set -euo pipefail

cat >&2 <<'EOF'
REFUSED: tools/ecology-test.sh is a retired interactive chain demo.

The former script broadcast transactions with implicit CLI node, chain, home,
and keys. That can target whichever network an operator last configured.
Rebuild the scenario as an isolated Go/localnet test with explicit disposable
state before running it again.
EOF
exit 1
