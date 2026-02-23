package keeper

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/partnerships/types"
)

// GetLockMultiplier returns the reward multiplier for a given lock tier.
func GetLockMultiplier(lockTier uint32) *big.Int {
	if lockTier > 5 {
		lockTier = 5
	}
	return new(big.Int).SetUint64(types.LockTiers[lockTier].MultiplierBps)
}

// DistributeReward distributes a gross reward through the partnership.
func (k Keeper) DistributeReward(
	ctx sdk.Context,
	partnership *types.Partnership,
	grossAmount string,
) (humanShare string, agentShare string, commonPotAdd string, err error) {
	gross := new(big.Int)
	if _, ok := gross.SetString(grossAmount, 10); !ok || gross.Sign() <= 0 {
		return "", "", "", fmt.Errorf("invalid gross amount: %s", grossAmount)
	}

	params := k.GetParams(ctx)

	// Step 1: Apply lock multiplier
	multiplier := GetLockMultiplier(partnership.LockTier)
	adjusted := new(big.Int).Mul(gross, multiplier)
	adjusted.Div(adjusted, bpsDenom)

	// Step 2: Calculate common pot share
	commonPotShareBps := new(big.Int).SetUint64(params.CommonPotShareBps)
	potShare := new(big.Int).Mul(adjusted, commonPotShareBps)
	potShare.Div(potShare, bpsDenom)

	// Step 3: Remaining goes to individual splits
	remaining := new(big.Int).Sub(adjusted, potShare)

	humanBps := new(big.Int).SetUint64(partnership.SplitHumanBps)
	humanAmt := new(big.Int).Mul(remaining, humanBps)
	humanAmt.Div(humanAmt, bpsDenom)

	agentAmt := new(big.Int).Sub(remaining, humanAmt)

	// Step 4: Update partnership metrics
	currentPot := new(big.Int)
	if partnership.CommonPotBalance != "" {
		currentPot.SetString(partnership.CommonPotBalance, 10)
	}
	currentPot.Add(currentPot, potShare)
	partnership.CommonPotBalance = currentPot.String()

	currentEarned := new(big.Int)
	if partnership.TotalEarned != "" {
		currentEarned.SetString(partnership.TotalEarned, 10)
	}
	currentEarned.Add(currentEarned, adjusted)
	partnership.TotalEarned = currentEarned.String()

	k.SetPartnership(ctx, partnership)

	return humanAmt.String(), agentAmt.String(), potShare.String(), nil
}
