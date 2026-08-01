#!/usr/bin/env bash
set -euo pipefail
umask 077

if [[ "${ZERONE_OPERATION_CONTEXT:-genesis}" == "recovery" ]]; then
  echo "make-genesis.sh is genesis-only and must not be used for validator recovery" >&2
  exit 1
fi

cat >&2 <<'EOF'
REFUSED: deploy/mainnet/make-genesis.sh is retired.

zerone-1 is already live. Rebuilding its genesis, mnemonics, node key, or
consensus key from moving source is not a release operation and is not
authorized by this repository.

Use the published genesis and manifest for read-only audit. Any future
ceremony must use a separately reviewed, release-bound, signed ceremony packet
with an explicit network decision; this source consolidation does not create
one.
EOF
exit 1
