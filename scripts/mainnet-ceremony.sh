#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════════════════
# Zerone mainnet genesis ceremony — deterministic pipeline
# ═══════════════════════════════════════════════════════════════════════════
#
# Design of record: docs/plans/2026-07-07-mainnet-genesis-design.md
# (§2 parameter list, §3 blockers 5-6, §8/§8a/§8b/§9/§10 confirmations).
#
# Pipeline (fixed order; every step is idempotent from a clean OUT_DIR):
#   1. zeroned init
#   2. add-genesis-account: 5 × 11,333 ZRN validator stake accounts
#      (11,111 ZRN locked+delegated stake + 222 ZRN spendable gas seed),
#      5 × 111 ZRN operator floats, 1 × 2,222 ZRN onboarding multisig
#      (total bank supply exactly 59,442 ZRN = 59,442,000,000 uzrn)
#   3. jq-patch the 5 stake accounts to
#      /cosmos.vesting.v1beta1.PermanentLockedAccount with
#      original_vesting = 11,111 ZRN ONLY (§13 gas seed: the extra 222 ZRN
#      of account balance stays SPENDABLE so validators can pay their own
#      vote gas at block 0) — the add-genesis-account --vesting flags are
#      dead code (cmd/zeroned/cmd/genesis.go:87-90 discards them)
#   4. jq knowledge.bootstrap_fund_allocation = "0" (suppresses the default
#      22,222 ZRN InitGenesis mint; day-0 supply stays exactly 58,332)
#   5. SDK-gov params to uzrn (default 'stake' denom would permanently kill
#      the authority-message surface)
#   6. substrate_bridge hardened economics + agenttool-invocation-v1
#      adapter ACTIVE at genesis (via tools/ceremony-inject)
#   7. IBC dark: transfer send/receive disabled, ICA host disabled with
#      empty allow_messages, uzrn rate limits pre-provisioned channel-0..7
#   8. emergency council / knowledge guardians / liquiditypool / staking /
#      slashing / distribution / zerone_gov / claiming_pot params per §2
#   9. founder economics per §12: DORMANT by default (FounderAddress stays
#      ""); set FOUNDER_ADDRESS in ceremony.env only if governance-preceded
#      activation is intended. PQ recovery commitments (§9) recorded in
#      GENESIS-MANIFEST.md.
#  10. creed genesis pin + 8 work_creed inception pins (tools/ceremony-inject)
#  11. claiming_pot bootstrap whitelist injection (tools/bootstrap-loader)
#      when a snapshot file is provided
#  12. gentx (5 × fully-self-bonded 11,111 ZRN) → collect-gentxs → validate
#  13. emit OUT_DIR/genesis.json + genesis.sha256 + GENESIS-MANIFEST.md
#
# Modes:
#   real  — ceremony.env provided (arg 1 or $CEREMONY_ENV). All addresses,
#           PQ commitment hashes and pre-signed gentx files come from the
#           ceremony inputs. The script never generates a real key.
#   drill — no ceremony.env: everything (operator keys, consensus keys,
#           whitelist, PQ commitments) is generated deterministically from
#           a fixed, PUBLIC mnemonic so two drill runs produce
#           byte-identical genesis files (TEST ceremony-repro, design §4).
#           A drill artifact is NEVER money.
#
# ceremony.env (real mode) must define:
#   CHAIN_ID GENESIS_TIME
#   VAL1_NAME VAL1_STAKE_ADDR VAL1_FLOAT_ADDR PQ_COMMITMENT_1   (…through 5)
#   ONBOARDING_ADDR BOOTSTRAP_REGISTRAR AGENTTOOL_OPS_ADDR
#   GENTX_DIR                     directory of 5 pre-signed gentx *.json
# and may define:
#   WHITELIST_SNAPSHOT            day-0 agenttool DID snapshot (one bech32
#                                 address per line) for bootstrap pots
#   FOUNDER_ADDRESS + PQ_COMMITMENT_FOUNDER   (activates the dormant share
#                                 — see design §12 before doing this)
#
# Env overrides: BINARY OUT_DIR CHAIN_ID GENESIS_TIME

set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BINARY="${BINARY:-${PROJECT_ROOT}/build/zeroned}"
OUT_DIR="${OUT_DIR:-${PROJECT_ROOT}/ceremony-out}"
HOME_DIR="${OUT_DIR}/home"
GENESIS="${HOME_DIR}/config/genesis.json"
KR=(--keyring-backend test)

# ── §2 canonical amounts (uzrn) ─────────────────────────────────────────────
STAKE_UZRN=11111000000          # 11,111 ZRN per validator, locked + self-delegated (original_vesting)
GAS_SEED_UZRN=222000000         # 222 ZRN per validator, SPENDABLE gas seed (§13, ~1,100 votes)
STAKE_BALANCE_UZRN=11333000000  # 11,111 locked stake + 222 gas seed = per-validator account balance
FLOAT_UZRN=111000000            # 111 ZRN operator float
ONBOARD_UZRN=2222000000         # 2,222 ZRN onboarding 3-of-5 multisig
TOTAL_SUPPLY_UZRN=59442000000   # 5×(stake+gas) + 5×float + onboarding = 59,442 ZRN, exactly
N_VALIDATORS=5                  # Alpha, Beta, Gamma, Yu, Ai (design §10a)

# ── §2 canonical parameters ─────────────────────────────────────────────────
GOV_MIN_DEPOSIT_UZRN=100000000            # 100 ZRN
GOV_EXPEDITED_DEPOSIT_UZRN=300000000      # 300 ZRN
GOV_VOTING_PERIOD="259200s"               # 3d
GOV_EXPEDITED_VOTING_PERIOD="86400s"      # 1d
GOV_QUORUM="0.334000000000000000"
GOV_THRESHOLD="0.500000000000000000"
GOV_VETO="0.334000000000000000"
GOV_MIN_INITIAL_DEPOSIT_RATIO="0.250000000000000000"
ZERONE_GOV_MIN_VOTE_STAKE="1000000"       # 1 ZRN — bonus wallets are governance-inert
BRIDGE_WITNESS_WINDOW="274176"            # ~8d > gov end-to-end
BRIDGE_PER_CLAIM_BOND="22200"
BRIDGE_ATTESTATION_MIN_BOND="22200000"    # 22.2 ZRN
BRIDGE_MAX_CLAIMS_PER_ATTESTATION=1000
BRIDGE_MIN_VERIFIED_BPS=6667              # 2/3, 10k-BPS scale in this module
BRIDGE_REJECTION_BPS=1000
RATE_LIMIT_UZRN="2222000000"              # 2,222 ZRN per 24h per channel
RATE_LIMIT_WINDOW_BLOCKS=34272            # ~24h
RATE_LIMIT_CHANNELS=8                     # channel-0 .. channel-7
COUNCIL_EXPIRY_BLOCK=6168960              # ~180d
MAX_REVERT_DEPTH=1111                     # ~47min
MIN_GUARDIAN_STAKE="11111000000"          # 11,111 ZRN
MIN_DISTINCT_VOTERS=4
KNOWLEDGE_VETO_WINDOW="34272"             # ~1d
PROBE_BOUNTY_PER_BLOCK="100000"           # 0.1 ZRN
PROBE_BOUNTY_POOL_CAP="22222000000"       # 22,222 ZRN
LP_MAX_POOLS=3
LP_PROTOCOL_FEE_BPS=0                     # dead-code honesty: 100% of fees to LPs
STAKING_MAX_VALIDATORS=33
STAKING_UNBONDING="1814400s"              # 21d
STAKING_MIN_COMMISSION="0.050000000000000000"
SLASHING_WINDOW="34272"
SLASHING_MIN_SIGNED="0.050000000000000000"
SLASHING_JAIL="3600s"
SLASHING_DOWNTIME="0.000100000000000000"
COMMUNITY_TAX="0.100000000000000000"      # 'foundation = network fee' made mechanical
CPOT_EMISSION_CAP="222222000000"          # 222,222 ZRN = 0.1% of max supply
CPOT_DAILY_ADMISSION_CAP=5000

# ── drill fixtures (PUBLIC — a drill artifact is never money) ───────────────
DRILL_MNEMONIC="now aware tomorrow wire robust regular unveil swallow trigger about immune wool humor allow inch runway sock acoustic scare weather outdoor shield attract direct"
DRILL_GENESIS_TIME="2026-01-01T00:00:00Z"
DRILL_CHAIN_ID="zerone-drill-1"

info() { printf '\033[1;34m  ->\033[0m %s\n' "$*"; }
ok()   { printf '\033[1;32m  OK\033[0m %s\n' "$*"; }
die()  { printf '\033[1;31mFAIL\033[0m %s\n' "$*" >&2; exit 1; }

[ -x "${BINARY}" ] || die "zeroned binary not found at ${BINARY} (run 'make build')"
command -v jq >/dev/null || die "jq is required"

# jq in-place patch of the working genesis.
patch() { # patch '<jq filter>' [--arg k v ...]
  local filter="$1"; shift
  local tmp
  tmp=$(mktemp)
  jq "$@" "${filter}" "${GENESIS}" > "${tmp}" || die "jq patch failed: ${filter}"
  mv "${tmp}" "${GENESIS}"
}

# ── mode resolution ─────────────────────────────────────────────────────────
CEREMONY_ENV="${1:-${CEREMONY_ENV:-}}"
if [ -n "${CEREMONY_ENV}" ]; then
  [ -f "${CEREMONY_ENV}" ] || die "ceremony env file not found: ${CEREMONY_ENV}"
  MODE=real
  # shellcheck disable=SC1090
  source "${CEREMONY_ENV}"
  for v in CHAIN_ID GENESIS_TIME ONBOARDING_ADDR BOOTSTRAP_REGISTRAR AGENTTOOL_OPS_ADDR GENTX_DIR; do
    [ -n "${!v:-}" ] || die "ceremony.env missing ${v}"
  done
  for i in $(seq 1 ${N_VALIDATORS}); do
    for v in VAL${i}_NAME VAL${i}_STAKE_ADDR VAL${i}_FLOAT_ADDR PQ_COMMITMENT_${i}; do
      [ -n "${!v:-}" ] || die "ceremony.env missing ${v}"
    done
  done
  if [ -n "${FOUNDER_ADDRESS:-}" ]; then
    [ -n "${PQ_COMMITMENT_FOUNDER:-}" ] || die "FOUNDER_ADDRESS set but PQ_COMMITMENT_FOUNDER missing (design §9)"
  fi
else
  MODE=drill
  CHAIN_ID="${CHAIN_ID:-${DRILL_CHAIN_ID}}"
  GENESIS_TIME="${GENESIS_TIME:-${DRILL_GENESIS_TIME}}"
fi
info "mode=${MODE} chain_id=${CHAIN_ID} out=${OUT_DIR}"

# ── clean slate + tool builds ───────────────────────────────────────────────
rm -rf "${OUT_DIR}"
mkdir -p "${OUT_DIR}/bin" "${OUT_DIR}/gentxs"
info "building ceremony tools"
(cd "${PROJECT_ROOT}" && go build -o "${OUT_DIR}/bin/ceremony-inject" ./tools/ceremony-inject)
(cd "${PROJECT_ROOT}" && go build -o "${OUT_DIR}/bin/bootstrap-loader" ./tools/bootstrap-loader)
INJECT="${OUT_DIR}/bin/ceremony-inject"
LOADER="${OUT_DIR}/bin/bootstrap-loader"

# ── 1. init ─────────────────────────────────────────────────────────────────
info "zeroned init"
"${BINARY}" init zerone-ceremony --chain-id "${CHAIN_ID}" --home "${HOME_DIR}" >/dev/null 2>&1
patch '.genesis_time = $t' --arg t "${GENESIS_TIME}"

# ── drill key material ──────────────────────────────────────────────────────
drill_key() { # drill_key <name> <hd-account-index> → address on stdout
  local name="$1" acct="$2"
  echo "${DRILL_MNEMONIC}" \
    | "${BINARY}" keys add "${name}" --recover --account "${acct}" \
        "${KR[@]}" --home "${HOME_DIR}" >/dev/null 2>&1
  "${BINARY}" keys show "${name}" -a "${KR[@]}" --home "${HOME_DIR}"
}

if [ "${MODE}" = drill ]; then
  info "drill: deriving deterministic keys (PUBLIC mnemonic — never money)"
  for i in $(seq 1 ${N_VALIDATORS}); do
    declare "VAL${i}_NAME=drill-val${i}"
    declare "VAL${i}_STAKE_ADDR=$(drill_key "val${i}" "${i}")"
    declare "VAL${i}_FLOAT_ADDR=$(drill_key "float${i}" "1${i}")"
    # §9 PQ recovery commitment: sha256 of a (drill-only) recovery secret.
    declare "PQ_COMMITMENT_${i}=$(printf 'drill-recovery-secret-%d' "${i}" | shasum -a 256 | cut -d' ' -f1)"
  done
  ONBOARDING_ADDR="$(drill_key onboarding 21)"
  AGENTTOOL_OPS_ADDR="$(drill_key agenttool-ops 31)"
  BOOTSTRAP_REGISTRAR="$(drill_key registrar 32)"
  WHITELIST_SNAPSHOT="${OUT_DIR}/drill-whitelist.txt"
  {
    echo "# drill day-0 cohort (deterministic, PUBLIC keys)"
    drill_key wl1 41
    drill_key wl2 42
    drill_key wl3 43
  } > "${WHITELIST_SNAPSHOT}"
fi

STAKE_ADDRS=()
FLOAT_ADDRS=()
for i in $(seq 1 ${N_VALIDATORS}); do
  sa="VAL${i}_STAKE_ADDR"; fa="VAL${i}_FLOAT_ADDR"
  STAKE_ADDRS+=("${!sa}"); FLOAT_ADDRS+=("${!fa}")
done

# ── 2. genesis accounts (the ONLY nine balances) ────────────────────────────
info "adding the 11 genesis balances (5 stake+gas + 5 float + onboarding)"
for a in "${STAKE_ADDRS[@]}"; do
  "${BINARY}" add-genesis-account "${a}" "${STAKE_BALANCE_UZRN}uzrn" --home "${HOME_DIR}" >/dev/null
done
for a in "${FLOAT_ADDRS[@]}"; do
  "${BINARY}" add-genesis-account "${a}" "${FLOAT_UZRN}uzrn" --home "${HOME_DIR}" >/dev/null
done
"${BINARY}" add-genesis-account "${ONBOARDING_ADDR}" "${ONBOARD_UZRN}uzrn" --home "${HOME_DIR}" >/dev/null

SUPPLY=$(jq -r '.app_state.bank.supply[] | select(.denom=="uzrn") | .amount' "${GENESIS}")
[ "${SUPPLY}" = "${TOTAL_SUPPLY_UZRN}" ] || die "bank supply ${SUPPLY} != ${TOTAL_SUPPLY_UZRN}"
ok "bank supply exactly ${TOTAL_SUPPLY_UZRN} uzrn (59,442 ZRN, 0.0267% of cap)"

# ── 3. PermanentLockedAccount patch ─────────────────────────────────────────
# add-genesis-account's --vesting flags are dead code (verified:
# cmd/zeroned/cmd/genesis.go:87-90 discards vestingStart/vestingEnd), so the
# lock is applied by the ceremony itself: genesis stake is slashable
# consensus collateral that can NEVER be transferred or sold (§1, §8b).
# §13 gas seed: original_vesting = STAKE_UZRN ONLY (not the full account
# balance) — the extra 222 ZRN gas seed stays SPENDABLE. Vesting accounts may
# delegate locked coins, so the gentx still self-delegates the full 11,111 ZRN.
info "converting the ${N_VALIDATORS} stake accounts to PermanentLockedAccount (lock 11,111, leave 222 spendable)"
for a in "${STAKE_ADDRS[@]}"; do
  patch '
    .app_state.auth.accounts = [ .app_state.auth.accounts[] |
      if (.["@type"] == "/cosmos.auth.v1beta1.BaseAccount" and .address == $addr) then
        { "@type": "/cosmos.vesting.v1beta1.PermanentLockedAccount",
          "base_vesting_account": {
            "base_account": { "address": $addr, "pub_key": null,
                              "account_number": .account_number, "sequence": .sequence },
            "original_vesting": [ { "denom": "uzrn", "amount": $amt } ],
            "delegated_free": [],
            "delegated_vesting": [],
            "end_time": "0" } }
      else . end ]' \
    --arg addr "${a}" --arg amt "${STAKE_UZRN}"
done
LOCKED=$(jq '[.app_state.auth.accounts[] | select(.["@type"]=="/cosmos.vesting.v1beta1.PermanentLockedAccount")] | length' "${GENESIS}")
[ "${LOCKED}" = "${N_VALIDATORS}" ] || die "expected ${N_VALIDATORS} PermanentLockedAccounts, got ${LOCKED}"
ok "${LOCKED} PermanentLockedAccounts (55,555 ZRN never transferable; 1,110 ZRN spendable gas seed)"

# ── 4. knowledge fund zero + guardian params ────────────────────────────────
# "0" verified to skip the 22,222 ZRN InitGenesis mint (x/knowledge/keeper/genesis.go:93).
info "knowledge: bootstrap fund 0, guardians = council, bounty discipline"
COUNCIL_JSON=$(jq -nc --arg ops "${AGENTTOOL_OPS_ADDR}" \
  --arg a "${STAKE_ADDRS[0]}" --arg b "${STAKE_ADDRS[1]}" \
  --arg c "${STAKE_ADDRS[2]}" --arg d "${STAKE_ADDRS[3]}" \
  '[$a,$b,$c,$d,$ops]')
patch '
  .app_state.knowledge.bootstrap_fund_allocation = "0"
  | .app_state.knowledge.params.guardian_addresses = $council
  | .app_state.knowledge.params.add_fact_veto_window_blocks = $veto
  | .app_state.knowledge.params.probe_bounty_mint_per_block = $bounty
  | .app_state.knowledge.params.probe_bounty_max_pool_size = $pool' \
  --argjson council "${COUNCIL_JSON}" --arg veto "${KNOWLEDGE_VETO_WINDOW}" \
  --arg bounty "${PROBE_BOUNTY_PER_BLOCK}" --arg pool "${PROBE_BOUNTY_POOL_CAP}"

# ── 5. SDK x/gov to uzrn ────────────────────────────────────────────────────
info "SDK gov: deposits in uzrn, 3d voting, quorum 0.334"
patch '
  .app_state.gov.params.min_deposit            = [ { "denom": "uzrn", "amount": $dep } ]
  | .app_state.gov.params.expedited_min_deposit = [ { "denom": "uzrn", "amount": $edep } ]
  | .app_state.gov.params.voting_period            = $vp
  | .app_state.gov.params.expedited_voting_period  = $evp
  | .app_state.gov.params.quorum                   = $q
  | .app_state.gov.params.threshold                = $th
  | .app_state.gov.params.veto_threshold           = $veto
  | .app_state.gov.params.min_initial_deposit_ratio = $midr' \
  --arg dep "${GOV_MIN_DEPOSIT_UZRN}" --arg edep "${GOV_EXPEDITED_DEPOSIT_UZRN}" \
  --arg vp "${GOV_VOTING_PERIOD}" --arg evp "${GOV_EXPEDITED_VOTING_PERIOD}" \
  --arg q "${GOV_QUORUM}" --arg th "${GOV_THRESHOLD}" --arg veto "${GOV_VETO}" \
  --arg midr "${GOV_MIN_INITIAL_DEPOSIT_RATIO}"

# zerone x/gov: bonus wallets (0.222 ZRN) stay below the LIP vote floor.
patch '.app_state.zerone_gov.params.min_vote_stake = $s' --arg s "${ZERONE_GOV_MIN_VOTE_STAKE}"

# ── 6. substrate_bridge hardened economics + adapter ────────────────────────
info "substrate_bridge: 8d witness window, 22.2 ZRN attestation bond, 2/3 verified"
patch '
  .app_state.substrate_bridge.params.witness_reward_challenge_window_blocks = $win
  | .app_state.substrate_bridge.params.per_pending_claim_bond_uzrn          = $pcb
  | .app_state.substrate_bridge.params.attestation_min_bond_uzrn            = $amb
  | .app_state.substrate_bridge.params.max_pending_claims_per_attestation   = $maxc
  | .app_state.substrate_bridge.params.min_verified_ratio_for_settle_bps    = $minv
  | .app_state.substrate_bridge.params.pending_claim_rejection_threshold_bps = $rej' \
  --arg win "${BRIDGE_WITNESS_WINDOW}" --arg pcb "${BRIDGE_PER_CLAIM_BOND}" \
  --arg amb "${BRIDGE_ATTESTATION_MIN_BOND}" \
  --argjson maxc "${BRIDGE_MAX_CLAIMS_PER_ATTESTATION}" \
  --argjson minv "${BRIDGE_MIN_VERIFIED_BPS}" --argjson rej "${BRIDGE_REJECTION_BPS}"
"${INJECT}" adapter "${GENESIS}"

# ── 7. IBC dark at genesis ──────────────────────────────────────────────────
info "IBC dark: transfer disabled, ICA host off, rate limits pre-provisioned"
RATE_LIMITS=$(jq -nc --arg amt "${RATE_LIMIT_UZRN}" --argjson win "${RATE_LIMIT_WINDOW_BLOCKS}" --argjson n "${RATE_LIMIT_CHANNELS}" '
  [ range(0; $n) | { channel_id: "channel-\(.)", denom: "uzrn",
      max_send: $amt, max_recv: $amt, window_blocks: $win,
      current_send: "0", current_recv: "0", window_start: 0 } ]')
patch '
  .app_state.transfer.params.send_enabled    = false
  | .app_state.transfer.params.receive_enabled = false
  | .app_state.interchainaccounts.host_genesis_state.params.host_enabled   = false
  | .app_state.interchainaccounts.host_genesis_state.params.allow_messages = []
  | .app_state.ibcratelimit.rate_limits = $limits' \
  --argjson limits "${RATE_LIMITS}"

# ── 8. council / staking / slashing / distribution / LP / claiming_pot ──────
info "emergency council (4 operators + agenttool ops), 180d expiry, 1,111-block revert cap"
patch '
  .app_state.emergency.params.genesis_council     = $council
  | .app_state.emergency.params.council_expiry_block = $expiry
  | .app_state.emergency.params.max_revert_depth     = $depth
  | .app_state.emergency.params.min_guardian_stake   = $stake
  | .app_state.emergency.params.min_distinct_voters  = $voters' \
  --argjson council "${COUNCIL_JSON}" --argjson expiry "${COUNCIL_EXPIRY_BLOCK}" \
  --argjson depth "${MAX_REVERT_DEPTH}" --arg stake "${MIN_GUARDIAN_STAKE}" \
  --argjson voters "${MIN_DISTINCT_VOTERS}"

info "staking/slashing/distribution/liquiditypool/claiming_pot per §2"
patch '
  .app_state.staking.params.max_validators      = $maxv
  | .app_state.staking.params.unbonding_time      = $unbond
  | .app_state.staking.params.min_commission_rate = $minc
  | .app_state.slashing.params.signed_blocks_window       = $win
  | .app_state.slashing.params.min_signed_per_window      = $minsig
  | .app_state.slashing.params.downtime_jail_duration     = $jail
  | .app_state.slashing.params.slash_fraction_downtime    = $down
  | .app_state.distribution.params.community_tax = $tax
  | .app_state.liquiditypool.params.max_pools        = $pools
  | .app_state.liquiditypool.params.protocol_fee_bps = $pfee
  | .app_state.claiming_pot.params.bootstrap_registrar          = $registrar
  | .app_state.claiming_pot.params.bootstrap_emission_cap_uzrn  = $cap
  | .app_state.claiming_pot.params.bootstrap_daily_admission_cap = $daily' \
  --argjson maxv "${STAKING_MAX_VALIDATORS}" --arg unbond "${STAKING_UNBONDING}" \
  --arg minc "${STAKING_MIN_COMMISSION}" \
  --arg win "${SLASHING_WINDOW}" --arg minsig "${SLASHING_MIN_SIGNED}" \
  --arg jail "${SLASHING_JAIL}" --arg down "${SLASHING_DOWNTIME}" \
  --arg tax "${COMMUNITY_TAX}" \
  --argjson pools "${LP_MAX_POOLS}" --argjson pfee "${LP_PROTOCOL_FEE_BPS}" \
  --arg registrar "${BOOTSTRAP_REGISTRAR}" --arg cap "${CPOT_EMISSION_CAP}" \
  --argjson daily "${CPOT_DAILY_ADMISSION_CAP}"

# ── 9. founder economics (§12: dormant by default) ──────────────────────────
if [ -n "${FOUNDER_ADDRESS:-}" ]; then
  info "founder share ACTIVATED at ceremony (deviates from §12 default — must be a recorded decision)"
  patch '.app_state.vesting_rewards.params.founder_address = $f' --arg f "${FOUNDER_ADDRESS}"
else
  info "founder share dormant (§12: founder takes nothing by protocol)"
fi

# ── 10. creed genesis pin + work_creed inception pins ───────────────────────
info "pinning Genesis Creed + 8 work_creed inception sub-creeds at block 0"
"${INJECT}" creed "${GENESIS}" "${PROJECT_ROOT}/.creed-hash" "${PROJECT_ROOT}/.sub-creed-hashes"

# ── 11. day-0 bootstrap whitelist ───────────────────────────────────────────
if [ -n "${WHITELIST_SNAPSHOT:-}" ]; then
  [ -f "${WHITELIST_SNAPSHOT}" ] || die "whitelist snapshot not found: ${WHITELIST_SNAPSHOT}"
  info "injecting bootstrap pots from ${WHITELIST_SNAPSHOT}"
  "${LOADER}" validate "${WHITELIST_SNAPSHOT}"
  "${LOADER}" inject "${WHITELIST_SNAPSHOT}" "${GENESIS}"
else
  info "no whitelist snapshot — zero genesis bootstrap pots (registrar admits post-launch)"
fi

# ── 12. gentx / collect / validate ──────────────────────────────────────────
if [ "${MODE}" = drill ]; then
  info "drill: generating ${N_VALIDATORS} self-bonded gentxs (11,111 ZRN each, locked stake)"
  for i in $(seq 1 ${N_VALIDATORS}); do
    VHOME="${OUT_DIR}/gentx-homes/val${i}"
    "${BINARY}" init "drill-val${i}" --chain-id "${CHAIN_ID}" --home "${VHOME}" >/dev/null 2>&1
    # Deterministic drill-only consensus key so TEST ceremony-repro holds.
    "${INJECT}" drill-consensus-key "zerone-drill-val${i}" \
      "${VHOME}/config/priv_validator_key.json" 2>/dev/null
    echo "${DRILL_MNEMONIC}" \
      | "${BINARY}" keys add "val${i}" --recover --account "${i}" \
          "${KR[@]}" --home "${VHOME}" >/dev/null 2>&1
    cp "${GENESIS}" "${VHOME}/config/genesis.json"
    NODE_ID=$(printf 'zerone-drill-val%d' "${i}" | shasum -a 1 | cut -c1-40)
    "${BINARY}" genesis gentx "val${i}" "${STAKE_UZRN}uzrn" \
      --chain-id "${CHAIN_ID}" --home "${VHOME}" "${KR[@]}" \
      --moniker "drill-val${i}" --ip 127.0.0.1 --node-id "${NODE_ID}" \
      --commission-rate 0.05 --commission-max-rate 0.20 \
      --commission-max-change-rate 0.01 --min-self-delegation 1 \
      --output-document "${OUT_DIR}/gentxs/gentx-val${i}.json" >/dev/null 2>&1
  done
else
  info "real: collecting pre-signed gentxs from ${GENTX_DIR}"
  n=$(ls "${GENTX_DIR}"/*.json 2>/dev/null | wc -l | tr -d ' ')
  [ "${n}" = "${N_VALIDATORS}" ] || die "expected ${N_VALIDATORS} gentx files in ${GENTX_DIR}, found ${n}"
  cp "${GENTX_DIR}"/*.json "${OUT_DIR}/gentxs/"
fi

info "collect-gentxs + validate"
"${BINARY}" genesis collect-gentxs --gentx-dir "${OUT_DIR}/gentxs" --home "${HOME_DIR}" >/dev/null 2>&1
"${BINARY}" genesis validate "${GENESIS}" >/dev/null || die "genesis validation failed"
GENTX_COUNT=$(jq '.app_state.genutil.gen_txs | length' "${GENESIS}")
[ "${GENTX_COUNT}" = "${N_VALIDATORS}" ] || die "expected ${N_VALIDATORS} gen_txs, got ${GENTX_COUNT}"
ok "genesis validates; ${GENTX_COUNT} gentxs collected (55,555 ZRN fully bonded at block 0)"

# ── 13. artifacts: genesis.json + sha256 + GENESIS-MANIFEST.md ──────────────
cp "${GENESIS}" "${OUT_DIR}/genesis.json"
GENESIS_SHA=$(shasum -a 256 "${OUT_DIR}/genesis.json" | cut -d' ' -f1)
echo "${GENESIS_SHA}  genesis.json" > "${OUT_DIR}/genesis.sha256"
CREED_HASH=$(tr -d '[:space:]' < "${PROJECT_ROOT}/.creed-hash")
SUBCREED_SHA=$(shasum -a 256 "${PROJECT_ROOT}/.sub-creed-hashes" | cut -d' ' -f1)
BINARY_VERSION=$("${BINARY}" version 2>&1)

if [ -n "${WHITELIST_SNAPSHOT:-}" ]; then
  WL_COUNT=$(grep -cv -e '^#' -e '^[[:space:]]*$' "${WHITELIST_SNAPSHOT}" || true)
  WL_SHA=$(shasum -a 256 "${WHITELIST_SNAPSHOT}" | cut -d' ' -f1)
  WL_LINE="${WL_COUNT} addresses (snapshot sha256 \`${WL_SHA}\`) — one 0.222 ZRN bootstrap pot each"
else
  WL_LINE="none — zero genesis bootstrap pots; day-0 cohort enters via the registrar"
fi
if [ -n "${FOUNDER_ADDRESS:-}" ]; then
  FOUNDER_LINE="| founder (ACTIVATED) | \`${FOUNDER_ADDRESS}\` | protocol share 70000 bps of research slice | PQ commitment \`${PQ_COMMITMENT_FOUNDER}\` |"
else
  FOUNDER_LINE="| founder | — | **founder takes nothing by protocol** (design §12: FounderAddress dormant; patronage + governance grants only) | — |"
fi

MANIFEST="${OUT_DIR}/GENESIS-MANIFEST.md"
{
  cat <<EOF
# Zerone Genesis Manifest — ${CHAIN_ID}

- Mode: **${MODE}**$( [ "${MODE}" = drill ] && printf ' — deterministic fixtures from a PUBLIC mnemonic; this artifact is NEVER money' )
- Genesis time: \`${GENESIS_TIME}\`
- Binary: \`${BINARY_VERSION}\`
- **genesis.json sha256: \`${GENESIS_SHA}\`**
- Creed canonical hash (block-0 pin): \`${CREED_HASH}\`
- .sub-creed-hashes sha256 (8 inception pins): \`${SUBCREED_SHA}\`

## Supply invariant (design §4 / §10 zero-ALLOCATION clause)

Total bank supply **exactly ${TOTAL_SUPPLY_UZRN} uzrn (59,442 ZRN = 0.0267% of the 222,222,222 cap)**:
55,555 ZRN permanently-locked bonded validator collateral + 1,110 ZRN spendable
validator gas seed (§13: 222 ZRN each, so validators can pay their own vote gas
at block 0) + 2,777 ZRN enumerated operational float. No team / foundation /
investor / research / faucet balance. Everything else mints on participation
under MintWithCap.

## Accounts (the only eleven balances)

| role | address | uzrn | account type | PQ recovery commitment (§9) |
|------|---------|------|--------------|------------------------------|
EOF
  for i in $(seq 1 ${N_VALIDATORS}); do
    nm="VAL${i}_NAME"; sa="VAL${i}_STAKE_ADDR"; pq="PQ_COMMITMENT_${i}"
    echo "| validator ${i} stake+gas (${!nm}) | \`${!sa}\` | ${STAKE_BALANCE_UZRN} | PermanentLockedAccount (original_vesting ${STAKE_UZRN} fully self-bonded; 222 ZRN spendable gas seed, §13) | \`${!pq}\` |"
  done
  for i in $(seq 1 ${N_VALIDATORS}); do
    nm="VAL${i}_NAME"; fa="VAL${i}_FLOAT_ADDR"
    echo "| validator ${i} float (${!nm}) | \`${!fa}\` | ${FLOAT_UZRN} | BaseAccount (gas/deposits/votes only — never person-transfers/pools/markets, §8b) | — |"
  done
  cat <<EOF
| onboarding 3-of-5 multisig | \`${ONBOARDING_ADDR}\` | ${ONBOARD_UZRN} | BaseAccount (feegrant engine for the claim subsidy, §8b) | — |
${FOUNDER_LINE}

## Keys and roles

| role | address |
|------|---------|
| emergency council | $(echo "${COUNCIL_JSON}" | jq -r 'map("`"+.+"`") | join(", ")') |
| agenttool ops (5th council key) | \`${AGENTTOOL_OPS_ADDR}\` |
| bootstrap registrar (claiming_pot) | \`${BOOTSTRAP_REGISTRAR}\` |

Council expiry: block ${COUNCIL_EXPIRY_BLOCK} (~180d). Max revert depth: ${MAX_REVERT_DEPTH} blocks (~47 min).

## Parameters of record (§2)

- SDK gov: min_deposit ${GOV_MIN_DEPOSIT_UZRN}uzrn, expedited ${GOV_EXPEDITED_DEPOSIT_UZRN}uzrn, voting ${GOV_VOTING_PERIOD}, expedited ${GOV_EXPEDITED_VOTING_PERIOD}, quorum ${GOV_QUORUM}
- zerone_gov min_vote_stake: ${ZERONE_GOV_MIN_VOTE_STAKE} uzrn (bonus wallets are governance-inert)
- knowledge.bootstrap_fund_allocation: "0" (no 22,222 ZRN InitGenesis mint)
- substrate_bridge: witness window ${BRIDGE_WITNESS_WINDOW} blocks (~8d), attestation min bond ${BRIDGE_ATTESTATION_MIN_BOND} uzrn, per-claim bond ${BRIDGE_PER_CLAIM_BOND} uzrn, ≤${BRIDGE_MAX_CLAIMS_PER_ATTESTATION} claims, min verified ${BRIDGE_MIN_VERIFIED_BPS}/10000, rejection ${BRIDGE_REJECTION_BPS}/10000
- adapter: agenttool-invocation-v1 ACTIVE at genesis, witness reward 222000 uzrn
- IBC: transfer send/receive disabled; ICA host disabled, allow_messages []; uzrn rate limit ${RATE_LIMIT_UZRN} uzrn / ${RATE_LIMIT_WINDOW_BLOCKS} blocks on channel-0..channel-$((RATE_LIMIT_CHANNELS-1))
- claiming_pot: registrar above, lifetime emission cap ${CPOT_EMISSION_CAP} uzrn (0.1% of max supply), daily admission cap ${CPOT_DAILY_ADMISSION_CAP}
- staking: max_validators ${STAKING_MAX_VALIDATORS}, unbonding ${STAKING_UNBONDING}, min commission ${STAKING_MIN_COMMISSION}
- slashing: window ${SLASHING_WINDOW}, min signed ${SLASHING_MIN_SIGNED}, jail ${SLASHING_JAIL}, downtime slash ${SLASHING_DOWNTIME}
- distribution community_tax: ${COMMUNITY_TAX} ('foundation = network fee' realized as community-pool income only)
- liquiditypool: max_pools ${LP_MAX_POOLS}, protocol_fee_bps ${LP_PROTOCOL_FEE_BPS}, zero genesis pools (creation frozen until the LP hardening trio lands)
- launch runbook (node config, not genesis): all validators run minimum-gas-prices=0.025uzrn from block 0; NO zero-fee bootstrap epoch

## Day-0 bootstrap whitelist

${WL_LINE}

## Gentxs

EOF
  for f in "${OUT_DIR}/gentxs/"*.json; do
    echo "- \`$(basename "${f}")\` sha256 \`$(shasum -a 256 "${f}" | cut -d' ' -f1)\`"
  done
} > "${MANIFEST}"

ok "artifacts written to ${OUT_DIR}"
ok "genesis.json sha256: ${GENESIS_SHA}"
ok "manifest: ${MANIFEST}"
echo
echo "Audit: ZERONE_GENESIS_ARTIFACT=${OUT_DIR}/genesis.json go test ./tests/cross_stack/ -run TestGenesisArtifact -v"
