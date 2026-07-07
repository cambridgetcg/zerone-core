#!/usr/bin/env bash
# Register the agenttool-invocation-v1 adapter on the single-validator public
# testnet via a real gov proposal (faucet deposits, validator votes yes).
# Testnet uses a CHEAP 1 ZRN attestation bond so agents can witness freely;
# the 22.2 ZRN floor is a mainnet-genesis value (scripts/agenttool-adapter-register.sh).
set -euo pipefail

BINARY="${BINARY:-$HOME/Desktop/zerone/build/zeroned}"
NODE="${NODE:-http://37.16.28.121:26657}"
CHAIN_ID="${CHAIN_ID:-zerone-testnet-1}"
KR="${KR:-$HOME/.zeroned/testnet-ops}"
GOV_AUTHORITY="zrn10d07y265gmmuvt4z0w9aw880jnsr700j47tt89"
ADAPTER_ID="agenttool-invocation-v1"
WITNESS_REWARD="222000"
MIN_BOND="1000000" # 1 ZRN — testnet-cheap
TX=(--keyring-backend test --home "${KR}" --chain-id "${CHAIN_ID}" --node "${NODE}"
    --broadcast-mode sync --yes -o json --gas 400000 --gas-prices 1uzrn)

info() { printf '\033[1;34m  ->\033[0m %s\n' "$*"; }
die()  { printf '\033[1;31mFAIL\033[0m %s\n' "$*" >&2; exit 1; }
bcast() { local o c h; o=$(cat); c=$(jq -r .code <<<"$o"); h=$(jq -r .txhash <<<"$o"); [ "$c" = "0" ] || die "code $c: $(jq -r .raw_log <<<"$o" | head -c 200)"; echo "$h"; }
wait_tx() { for i in $(seq 1 20); do if o=$("${BINARY}" query tx "$1" --node "${NODE}" -o json 2>/dev/null); then [ "$(jq -r .code <<<"$o")" = "0" ] || die "$2 failed"; return 0; fi; sleep 2; done; die "$2 tx not found"; }

PROP=$(mktemp); trap 'rm -f "$PROP"' EXIT
cat > "$PROP" <<JSON
{
  "messages": [{
    "@type": "/zerone.substrate_bridge.v1.MsgRegisterAdapter",
    "authority": "${GOV_AUTHORITY}",
    "adapter": {
      "adapter_id": "${ADAPTER_ID}", "source_type": "agenttool", "version": "1.1.0",
      "min_attestation_bond_uzrn": "${MIN_BOND}", "min_per_claim_bond_uzrn": "100000",
      "allowed_class_ids": [], "status": 1, "witness_reward_uzrn": "${WITNESS_REWARD}"
    }
  }],
  "metadata": "agenttool bridge: register ${ADAPTER_ID}",
  "deposit": "10000000uzrn",
  "title": "Register adapter ${ADAPTER_ID}",
  "summary": "Gov-register the agenttool marketplace invocation adapter on the public testnet. Witness-only attestations escrow ${WITNESS_REWARD} uzrn under the challenge window and mint via MintWithCap only on survival."
}
JSON

info "submit proposal (faucet deposits 10 ZRN)"
H=$("${BINARY}" tx gov submit-proposal "$PROP" --from faucet "${TX[@]}" | bcast); wait_tx "$H" "submit"
PID=$("${BINARY}" query gov proposals --node "${NODE}" -o json | jq -r '.proposals[-1].id')
info "proposal ${PID}: vote yes (validator)"
H=$("${BINARY}" tx gov vote "${PID}" yes --from validator "${TX[@]}" | bcast); wait_tx "$H" "vote"

info "waiting for voting period (~130s)..."
for i in $(seq 1 30); do
  ST=$("${BINARY}" query gov proposal "${PID}" --node "${NODE}" -o json 2>/dev/null | jq -r '.proposal.status')
  [ "$ST" = "PROPOSAL_STATUS_PASSED" ] && { info "PASSED"; break; }
  [ "$ST" = "PROPOSAL_STATUS_REJECTED" ] && die "REJECTED"
  sleep 6
done
"${BINARY}" query substrate_bridge adapters --node "${NODE}" -o json | jq -r '.adapters[] | "adapter: \(.adapter_id) status=\(.status) min_bond=\(.min_attestation_bond_uzrn) reward=\(.witness_reward_uzrn)"'
