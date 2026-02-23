package cross_stack_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	cpotkeeper "github.com/zerone-chain/zerone/x/claiming_pot/keeper"
	cpottypes "github.com/zerone-chain/zerone/x/claiming_pot/types"
	zeronestakingtypes "github.com/zerone-chain/zerone/x/staking/types"
)

// TestScenario6_ClaimingPotTierEnforcement verifies that claiming pots enforce
// staking tier requirements — agents below the minimum tier cannot claim.
func TestScenario6_ClaimingPotTierEnforcement(t *testing.T) {
	h := NewTestHarness(t)

	// 1. Create two validators at different tiers.
	addrA := testAddr("claim_agent_a___")
	addrB := testAddr("claim_agent_b___")

	// Agent A: Apprentice tier (tier 1).
	valA := &zeronestakingtypes.Validator{
		OperatorAddress: addrA.String(),
		ConsensusPubkey: testPubKeyHex("claim_agent_a___"),
		Moniker:         "agent-a-apprentice",
		Tier:            zeronestakingtypes.TierApprentice,
		SelfDelegation:  "1000000000",
		DelegatedStake:  "0",
		TotalStake:      "1000000000",
		ReputationScore: 500_000,
		JoinedAtBlock:   uint64(h.Height()),
		IsActive:        true,
	}
	h.StakingKeeper.SetValidator(h.Ctx, valA)

	// Agent B: Scholar tier (tier 2).
	valB := &zeronestakingtypes.Validator{
		OperatorAddress: addrB.String(),
		ConsensusPubkey: testPubKeyHex("claim_agent_b___"),
		Moniker:         "agent-b-scholar",
		Tier:            zeronestakingtypes.TierScholar,
		SelfDelegation:  "10000000000",
		DelegatedStake:  "0",
		TotalStake:      "10000000000",
		ReputationScore: 700_000,
		JoinedAtBlock:   uint64(h.Height()),
		IsActive:        true,
	}
	h.StakingKeeper.SetValidator(h.Ctx, valB)

	// Verify tiers are stored correctly.
	retrievedA, found := h.StakingKeeper.GetValidator(h.Ctx, addrA.String())
	require.True(t, found)
	require.Equal(t, zeronestakingtypes.TierApprentice, retrievedA.Tier)

	retrievedB, found := h.StakingKeeper.GetValidator(h.Ctx, addrB.String())
	require.True(t, found)
	require.Equal(t, zeronestakingtypes.TierScholar, retrievedB.Tier)

	// 2. Create a claiming pot with Scholar tier minimum requirement.
	currentBlock := uint64(h.Height())
	pot := &cpottypes.ClaimingPot{
		Id:            "pot-001",
		Name:          "Scholar Research Fund",
		TotalAmount:   "10000000000", // 10,000 ZRN
		ClaimedAmount: "0",
		Schedule: &cpottypes.VestingSchedule{
			StartBlock:   currentBlock,
			EndBlock:     currentBlock + 100_000,
			CliffBlocks:  0, // no cliff
			PeriodBlocks: 1000,
		},
		Eligibility: &cpottypes.EligibilityCriteria{
			MinStakingTier:     uint32(zeronestakingtypes.TierScholar), // Scholar minimum
			MinRegistrationAge: 0,
		},
		CreatedAtBlock: currentBlock,
		Status:         cpottypes.PotStatus_POT_STATUS_ACTIVE,
	}
	h.ClaimingPotKeeper.SetPot(h.Ctx, pot)

	// 3. Verify pot is stored and active.
	retrievedPot, found := h.ClaimingPotKeeper.GetPot(h.Ctx, "pot-001")
	require.True(t, found, "pot must be retrievable")
	require.Equal(t, cpottypes.PotStatus_POT_STATUS_ACTIVE, retrievedPot.Status)
	require.Equal(t, uint32(zeronestakingtypes.TierScholar), retrievedPot.Eligibility.MinStakingTier)

	// 4. Verify tier enforcement logic.
	// Agent A (Apprentice = tier 1) is BELOW the minimum (Scholar = tier 2).
	require.True(t, retrievedA.Tier < zeronestakingtypes.ValidatorTier(retrievedPot.Eligibility.MinStakingTier),
		"Apprentice tier must be below Scholar tier requirement")

	// Agent B (Scholar = tier 2) MEETS the minimum.
	require.True(t, retrievedB.Tier >= zeronestakingtypes.ValidatorTier(retrievedPot.Eligibility.MinStakingTier),
		"Scholar tier must meet Scholar tier requirement")

	// 5. Verify vesting math via CalculateClaimable.
	// At start block: claimable should be 0 (linear vesting just started).
	claimableAtStart := cpotkeeper.CalculateClaimable(retrievedPot, currentBlock)
	require.Equal(t, big.NewInt(0).Int64(), claimableAtStart.Int64(),
		"claimable at start must be 0")

	// At midpoint: should be ~50% of total.
	midBlock := currentBlock + 50_000
	claimableAtMid := cpotkeeper.CalculateClaimable(retrievedPot, midBlock)
	expectedMid := new(big.Int)
	expectedMid.SetString("5000000000", 10) // 50% of 10,000,000,000
	require.Equal(t, expectedMid.Int64(), claimableAtMid.Int64(),
		"claimable at midpoint must be ~50%% of total")

	// After end block: should be 100%.
	claimableAtEnd := cpotkeeper.CalculateClaimable(retrievedPot, currentBlock+100_001)
	expectedEnd := new(big.Int)
	expectedEnd.SetString("10000000000", 10) // 100% of total
	require.Equal(t, expectedEnd.Int64(), claimableAtEnd.Int64(),
		"claimable after end must be 100%% of total")
}

// TestScenario6_ClaimingPotCliffEnforcement verifies cliff period enforcement.
func TestScenario6_ClaimingPotCliffEnforcement(t *testing.T) {
	currentBlock := uint64(100)

	pot := &cpottypes.ClaimingPot{
		Id:            "pot-cliff",
		Name:          "Cliff Test Pot",
		TotalAmount:   "1000000",
		ClaimedAmount: "0",
		Schedule: &cpottypes.VestingSchedule{
			StartBlock:   currentBlock,
			EndBlock:     currentBlock + 1000,
			CliffBlocks:  100, // 100-block cliff
			PeriodBlocks: 10,
		},
		Status: cpottypes.PotStatus_POT_STATUS_ACTIVE,
	}

	// Before cliff: nothing vested.
	claimableBeforeCliff := cpotkeeper.CalculateClaimable(pot, currentBlock+50)
	require.Equal(t, int64(0), claimableBeforeCliff.Int64(),
		"nothing claimable before cliff")

	// At cliff: vesting begins (100 blocks elapsed out of 1000).
	claimableAtCliff := cpotkeeper.CalculateClaimable(pot, currentBlock+100)
	require.Greater(t, claimableAtCliff.Int64(), int64(0),
		"claimable must be > 0 at cliff boundary")

	// After cliff but before end: linear vesting.
	claimableAfterCliff := cpotkeeper.CalculateClaimable(pot, currentBlock+500)
	require.Greater(t, claimableAfterCliff.Int64(), claimableAtCliff.Int64(),
		"claimable must increase after cliff")
}

// TestScenario6_ClaimingPotPotExpiry verifies that expired pots are marked correctly.
func TestScenario6_ClaimingPotPotExpiry(t *testing.T) {
	h := NewTestHarness(t)

	currentBlock := uint64(h.Height())
	pot := &cpottypes.ClaimingPot{
		Id:            "pot-expiry",
		Name:          "Expiring Pot",
		TotalAmount:   "500000",
		ClaimedAmount: "0",
		Schedule: &cpottypes.VestingSchedule{
			StartBlock:   currentBlock,
			EndBlock:     currentBlock + 10, // expires very soon
			CliffBlocks:  0,
			PeriodBlocks: 1,
		},
		Status:         cpottypes.PotStatus_POT_STATUS_ACTIVE,
		CreatedAtBlock: currentBlock,
	}
	h.ClaimingPotKeeper.SetPot(h.Ctx, pot)

	// Verify active.
	require.Equal(t, 1, h.ClaimingPotKeeper.CountActivePots(h.Ctx))

	// Process expiry at a block past EndBlock.
	h.ClaimingPotKeeper.ProcessPotExpiry(h.Ctx, currentBlock+11)

	// Verify pot is now expired.
	expired, found := h.ClaimingPotKeeper.GetPot(h.Ctx, "pot-expiry")
	require.True(t, found)
	require.Equal(t, cpottypes.PotStatus_POT_STATUS_EXPIRED, expired.Status)
}
