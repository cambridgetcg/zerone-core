#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════════════════
# Recursion Check — verify cited recursion source bindings
# ═══════════════════════════════════════════════════════════════════════════
#
# Runs the cited tests for each numbered section in RECURSIVE_ZERONE.md.
# Passing proves those source capabilities, not live-network activation.
#
# Use this before merging changes that touch any module participating in
# the recursion catalog (sponsorship, substrate_bridge, claiming_pot,
# vesting_rewards, knowledge, creed, work_creed).
#
# Usage:
#   scripts/recursion-check.sh         # run all
#   scripts/recursion-check.sh quick   # one-test-per-recursion sample
#
# ═══════════════════════════════════════════════════════════════════════════

set -uo pipefail

MODE="${1:-full}"

RED='\033[1;31m'
GREEN='\033[1;32m'
BLUE='\033[1;34m'
YELLOW='\033[1;33m'
RESET='\033[0m'

PASS=0
FAIL=0
TOTAL=0
FAILED_RECURSIONS=()

run_recursion() {
  local n="$1"
  local title="$2"
  local test_pattern="$3"

  TOTAL=$((TOTAL + 1))
  printf "  %b[%2d]%b %-58s " "${BLUE}" "${n}" "${RESET}" "${title}"

  # -count=1 disables Go's test cache so we get a real run.
  if go test -count=1 -run "${test_pattern}" ./tests/cross_stack/ >/tmp/recursion-check-${n}.log 2>&1; then
    printf "%bPASS%b\n" "${GREEN}" "${RESET}"
    PASS=$((PASS + 1))
  else
    printf "%bFAIL%b\n" "${RED}" "${RESET}"
    FAIL=$((FAIL + 1))
    FAILED_RECURSIONS+=("${n}: ${title}")
    if [ "${MODE}" = "verbose" ]; then
      sed -n '/^---/,/^FAIL/p' /tmp/recursion-check-${n}.log | head -20
    fi
  fi
}

echo
echo "═══════════════════════════════════════════════════════════════════"
echo "  Recursion Source-Binding Check — docs/RECURSIVE_ZERONE.md"
echo "═══════════════════════════════════════════════════════════════════"
echo

# Recursion 1: chain attests to its own becoming
run_recursion 1 "self-adapter compiler and dormant bridge are test-bound" \
  "TestZeroneSelfAdapter"

# Recursion 2: sponsorship primitives can fund self-documentation
run_recursion 2 "self-domain sponsorship primitives compose in tests" \
  "TestZeroneSelf_ScaffoldedEconomicLoopRequiresManualBridgeState|TestZeroneSelf_MultipleFulfillmentsCompoundEarnings"

# Recursion 3: chain pays its builders twice for the same verified work
run_recursion 3 "double-payment primitives compose in tests" \
  "TestRecursiveDoublePayment_ManuallyStagedStateExercisesTwoPayouts"

# Recursion 4: chain's lineage graph includes its own commits
run_recursion 4 "self-lineage primitives compose in tests" \
  "TestRecursiveLineage_AccountingAttributesDownstreamRewardUpstream|TestRecursiveLineage_MultipleCitationsCompoundAccounting"

# Recursion 5: creed cannot move faster than governance
run_recursion 5 "creed source hash is repository-bound" \
  "TestTruthSeeking_CreedHashIsPinned"

# Recursion 6: useful-work sub-creeds are source-hash bound
run_recursion 6 "useful-work sub-creeds are source-hash bound" \
  "TestSubCreed_(Alignment|Augmentation|Curation|Evaluation|Foundation|Substrate|Tools|Training)_StaysInSync"

# Recursion 7: participation grows through participation
run_recursion 7 "participation grows through participation" \
  "TestLateBootstrap|TestScenario13e_BootstrapPotsDoNotExpire"

# Recursion 8: economy is hard-capped and self-circulating
run_recursion 8 "economy is hard-capped and self-circulating" \
  "TestEmissionCap_BootstrapClaimMintsOnDemand|TestScenario13_ProtocolDefaultGenesisHasNoBalances|TestScenario13c_ClaimingPotMinterPermission|TestSubstrateBridge_HappyPathSettlement|TestSponsorship_NoMintingHappens"

# Recursion 9: autonomous audit budget capability
run_recursion 9 "autonomous audit budget capability is test-bound" \
  "TestTruthSeeking_AuditBudgetIsAutonomous|TestTruthSeeking_ChainPaysForOwnAudit|TestMoat_ProbeBountyPoolAccumulatesAndFundsBonuses|TestMoat_ProbeBountyPoolRespectsCap"

# Recursion 10: recursion catalog and voice layer audit their own bindings
run_recursion 10 "recursion catalog and voice audit their own bindings" \
  "TestRecursiveZerone_TestNamesCitedInDoctrineExist|TestRecursiveVoiceAudit_StagedRecursionEventsCarryDoctrineAttributes"

echo
echo "═══════════════════════════════════════════════════════════════════"
if [ "${FAIL}" -eq 0 ]; then
  printf "  %bAll %d cited source bindings pass.%b\n" "${GREEN}" "${TOTAL}" "${RESET}"
  echo "  This result does not assert live deployment or public-path reachability."
  echo "═══════════════════════════════════════════════════════════════════"
  exit 0
else
  printf "  %b%d/%d recursions failed.%b\n" "${RED}" "${FAIL}" "${TOTAL}" "${RESET}"
  for r in "${FAILED_RECURSIONS[@]}"; do
    printf "    %b%s%b\n" "${YELLOW}" "${r}" "${RESET}"
  done
  echo "  Logs in /tmp/recursion-check-*.log"
  echo "═══════════════════════════════════════════════════════════════════"
  exit 1
fi
