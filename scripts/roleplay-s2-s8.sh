#!/usr/bin/env bash
# Scenarios 2-8 — continuation of roleplay test
# Less strict error handling

B=./build/zeroned
H0="$HOME/.zeroned/localnet/val0"
H1="$HOME/.zeroned/localnet/val1"
H2="$HOME/.zeroned/localnet/val2"
H3="$HOME/.zeroned/localnet/val3"
NODE="tcp://localhost:26601"
CHAIN="zerone-localnet"

tx0() { $B "$@" --keyring-backend=test --home="$H0" --chain-id="$CHAIN" --node="$NODE" --gas=200000 --fees=200000uzrn -y 2>&1; }
tx1() { $B "$@" --keyring-backend=test --home="$H1" --chain-id="$CHAIN" --node="$NODE" --gas=200000 --fees=200000uzrn -y 2>&1; }
tx2() { $B "$@" --keyring-backend=test --home="$H2" --chain-id="$CHAIN" --node="$NODE" --gas=200000 --fees=200000uzrn -y 2>&1; }
tx3() { $B "$@" --keyring-backend=test --home="$H3" --chain-id="$CHAIN" --node="$NODE" --gas=200000 --fees=200000uzrn -y 2>&1; }
q() { $B "$@" --node="$NODE" 2>&1; }

wait_b() { echo "  ... waiting ${1}s"; sleep "$1"; }

get_code() { echo "$1" | grep "^code:" | head -1 | awk '{print $2}'; }
get_hash() { echo "$1" | grep "txhash:" | awk '{print $2}'; }
get_raw() { echo "$1" | grep "raw_log:" | head -1; }

extract_attr() {
    local txhash="$1" etype="$2" akey="$3"
    $B query tx "$txhash" --node="$NODE" --output=json 2>/dev/null | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    for evt in data.get('events', []):
        if '$etype' in evt.get('type', ''):
            for attr in evt.get('attributes', []):
                if attr.get('key') == '$akey':
                    print(attr['value'])
                    sys.exit(0)
except: pass
print('')
" 2>/dev/null || echo ""
}

ALICE=zrn1yulq2lnk5ymytum50pk7n2ypxz7557vr0hj3vs
BOB=zrn12kf7t89r200unrc9cwm9kl9f20wah5y7sztv6d
ROGUE=zrn12yhvlme06302rmj3njahm766hwpsvhvvq9pces
VAL0=zrn19x242r6eujyr3p4rjcgclve8lmnjxvmg6v4cpl
VAL1=zrn16edknx7gwp8mtl7nsm9jxe2h2gnxwkwht38cc5
VAL2=zrn1tnhw6eghqzwqyzmlka3mgk0lv7k4j6g0yym6uf
VAL3=zrn1tyxaf5jntaxfw5njfhg5yryvpdjnj4vts24jv4
S1_FACT_ID=1170d19975345ee009ec656e3bd57e92

echo "═══ SCENARIO 2: Challenge Flow (Adversarial) ═══"

SALT_R0=$(openssl rand -hex 16)
SALT_R1=$(openssl rand -hex 16)
COMMIT_R0=$( (printf "accept"; printf '%s' "$SALT_R0" | xxd -r -p) | shasum -a 256 | awk '{print $1}')
COMMIT_R1=$( (printf "accept"; printf '%s' "$SALT_R1" | xxd -r -p) | shasum -a 256 | awk '{print $1}')

echo "[S2.1] Rogue submits bogus claim..."
S2_TX=$(tx0 tx knowledge submit-claim \
    "Speed of light varies with observer mood" \
    general empirical 1000000 --from=rogue)
echo "  code=$(get_code "$S2_TX")"
S2_HASH=$(get_hash "$S2_TX")
echo "  hash=$S2_HASH"

wait_b 5
S2_CLAIM=$(extract_attr "$S2_HASH" "submit_claim" "claim_id")
S2_ROUND=$(extract_attr "$S2_HASH" "verification_round" "round_id")
echo "  claim=$S2_CLAIM round=$S2_ROUND"

echo "[S2.2] Verifying bogus claim (val0+val1)..."
tx0 tx knowledge submit-commitment "$S2_ROUND" "$COMMIT_R0" --from=val0 | head -1
wait_b 2
tx1 tx knowledge submit-commitment "$S2_ROUND" "$COMMIT_R1" --from=val1 | head -1
wait_b 30
tx0 tx knowledge submit-reveal "$S2_ROUND" accept "$SALT_R0" --from=val0 | head -1
wait_b 2
tx1 tx knowledge submit-reveal "$S2_ROUND" accept "$SALT_R1" --from=val1 | head -1
wait_b 18

echo "[S2.3] Checking round result..."
S2_RR=$(q query knowledge verification-round "$S2_ROUND")
echo "$S2_RR" | grep -E "verdict:|phase:"

# Get fact
S2_FACT=""
S2_ROGUE_FACTS=$(q query knowledge facts-by-submitter "$ROGUE")
if echo "$S2_ROGUE_FACTS" | grep -q "  id:"; then
    S2_FACT=$(echo "$S2_ROGUE_FACTS" | grep "  id:" | head -1 | awk '{print $2}')
fi
echo "  rogue_fact=$S2_FACT"

if [ -n "$S2_FACT" ]; then
    echo "[S2.4] Alice challenges bogus fact..."
    S2_CHAL=$(tx0 tx knowledge challenge-fact "$S2_FACT" 11000000 \
        "No empirical basis - contradicts special relativity" --from=alice)
    echo "  challenge code=$(get_code "$S2_CHAL")"
    echo "  $(get_raw "$S2_CHAL")"

    wait_b 3
    echo "[S2.5] Checking fact status after challenge..."
    S2_FACT_STATUS=$(q query knowledge fact "$S2_FACT")
    echo "$S2_FACT_STATUS" | grep -E "status:|confidence:"

    echo "[S2.6] Initiating dispute..."
    S2_DISP=$(tx0 tx disputes initiate-dispute "$S2_FACT" 1000000 \
        "Claim contradicts well-established physics" --from=alice)
    S2_DISP_CODE=$(get_code "$S2_DISP")
    S2_DISP_HASH=$(get_hash "$S2_DISP")
    echo "  dispute code=$S2_DISP_CODE"
    echo "  $(get_raw "$S2_DISP")"

    if [ "$S2_DISP_CODE" = "0" ]; then
        wait_b 5
        S2_DID=$(extract_attr "$S2_DISP_HASH" "dispute" "dispute_id")
        echo "  dispute_id=$S2_DID"

        if [ -n "$S2_DID" ]; then
            echo "[S2.7] Arbiter (val2) votes..."
            # Need to wait for arbitration phase
            S2_VOTE=$(tx2 tx disputes arbiter-vote "$S2_DID" "challenger" \
                "Claim is pseudoscience" --from=val2)
            echo "  vote code=$(get_code "$S2_VOTE")"
            echo "  $(get_raw "$S2_VOTE")"
        fi
    fi
else
    echo "  SKIP: Bogus claim didn't become fact"
fi

# SCENARIO 3 (partnership collaboration) retired with x/partnerships
# (2026-07 slim cut): collaboration lives on the agenttool layer.

echo ""
echo "═══ SCENARIO 4: Domain Expansion ═══"

echo "[S4.1] Bob proposes 'bioethics' domain..."
S4_PROP=$(tx0 tx knowledge propose-domain "bioethics" \
    "Bioethics and medical ethics frameworks" 4 --from=bob)
echo "  code=$(get_code "$S4_PROP")"
echo "  $(get_raw "$S4_PROP")"

wait_b 3
echo "[S4.2] Sage-1 endorses..."
echo "  code=$(get_code "$(tx0 tx knowledge endorse-domain-proposal "bioethics" --from=val0)")"
wait_b 3
echo "[S4.3] Sage-2 endorses..."
echo "  code=$(get_code "$(tx1 tx knowledge endorse-domain-proposal "bioethics" --from=val1)")"
wait_b 3
echo "[S4.4] Arbiter endorses..."
echo "  code=$(get_code "$(tx2 tx knowledge endorse-domain-proposal "bioethics" --from=val2)")"

wait_b 3
echo "[S4.5] Domain status:"
q query knowledge domain "bioethics"

echo "[S4.6] Bob submits first claim in bioethics..."
S4_CLAIM=$(tx0 tx knowledge submit-claim \
    "Informed consent requires understanding risks benefits and alternatives" \
    bioethics analytic 2000000 --from=bob)
S4_CLAIM_CODE=$(get_code "$S4_CLAIM")
S4_CLAIM_HASH=$(get_hash "$S4_CLAIM")
echo "  code=$S4_CLAIM_CODE"

echo ""
echo "═══ SCENARIO 5: Qualification Gate Test ═══"

echo "[S5.1] Sage-2 qualifies in general domain..."
S5_QUAL=$(tx1 tx qualification qualify-by-stake "general" 100000000 --from=val1)
echo "  code=$(get_code "$S5_QUAL")"
echo "  $(get_raw "$S5_QUAL")"

wait_b 5
# Get bioethics round from S4
S5_ROUND=""
if [ -n "$S4_CLAIM_HASH" ]; then
    S5_ROUND=$(extract_attr "$S4_CLAIM_HASH" "verification_round" "round_id")
fi
echo "  bioethics_round=$S5_ROUND"

if [ -n "$S5_ROUND" ]; then
    echo "[S5.2] Sage-2 tries to verify in bioethics (NOT qualified)..."
    SALT_GATE=$(openssl rand -hex 16)
    COMMIT_GATE=$( (printf "accept"; printf '%s' "$SALT_GATE" | xxd -r -p) | shasum -a 256 | awk '{print $1}')
    S5_GATE=$(tx1 tx knowledge submit-commitment "$S5_ROUND" "$COMMIT_GATE" --from=val1)
    S5_GATE_CODE=$(get_code "$S5_GATE")
    echo "  unqualified_commit code=$S5_GATE_CODE"
    if [ "$S5_GATE_CODE" = "0" ]; then
        echo "  *** QUALIFICATION GATE NOT ENFORCED ***"
    else
        echo "  ✓ Qualification gate enforced"
    fi
fi

# SCENARIO 6 (research bounty) retired with x/research (2026-07 slim cut):
# funded work listings live on the agenttool layer; x/sponsorship covers
# on-chain fact bounties.

echo ""
echo "═══ SCENARIO 7: Capture Defense ═══"

echo "[S7.1] Rogue floods general domain..."
for i in 1 2 3 4 5; do
    S7_F=$(tx0 tx knowledge submit-claim \
        "Dubious flooding claim $i in general" \
        general empirical 1000000 --from=rogue)
    echo "  claim $i code=$(get_code "$S7_F")"
    sleep 3
done

echo "[S7.2] Analyze domain..."
S7_ANA=$(tx0 tx capture-defense analyze-domain "general" --from=val0)
echo "  code=$(get_code "$S7_ANA")"

wait_b 3
echo "[S7.3] Alice submits capture challenge..."
S7_CC=$(tx0 tx capture-challenge submit-challenge "general" \
    "Account is flooding domain with low-quality claims" \
    10000000 --from=alice)
echo "  code=$(get_code "$S7_CC")"
echo "  $(get_raw "$S7_CC")"

# SCENARIO 8 (coercion signal) retired with x/partnerships (2026-07 slim cut).

echo ""
echo "═══ FINAL STATE ═══"

echo "=== Balances ==="
for name_addr in "Alice:$ALICE" "Bob:$BOB" "Rogue:$ROGUE" "Sage1:$VAL0" "Sage2:$VAL1"; do
    name="${name_addr%%:*}"
    addr="${name_addr#*:}"
    bal=$(q query bank balances "$addr" | grep "amount:" | head -1 | awk '{print $2}' | tr -d '"')
    echo "  $name: ${bal:-0} uzrn"
done

echo "=== Facts ==="
q query knowledge facts-by-domain general | grep -c "  id:" || echo "0"
echo " facts in general domain"

echo ""
echo "DONE — $(date)"
