#!/usr/bin/env bash
# Repository-owned H1 ritual. Defaults to an OFFLINE plan; see TESTNET.md.
set -euo pipefail
exec python3 "$(dirname "${BASH_SOURCE[0]}")/ship-consolidation-safety.py" "$@"
