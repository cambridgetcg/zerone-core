#!/usr/bin/env bash
# Retired 2026-07-29. Keep the path fail-closed so cached operator commands
# cannot broadcast onboarding transactions from an unreviewed moving branch.

set -euo pipefail

cat >&2 <<'EOF'
FAIL: automated zerone-1 onboarding is paused in this source release.

This script will not create or inspect keys, admit an address, grant fees,
claim tokens, transfer funds, or broadcast a transaction. Onboarding requires
a separately reviewed operator packet plus explicit authority for each live
action.

Read:
  skills/zerone-onboarding/SKILL.md
  deploy/mainnet/TRUST.md
  docs/VALIDATOR-GUIDE.md
EOF

exit 1
