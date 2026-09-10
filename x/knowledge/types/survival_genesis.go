package types

import (
	"fmt"
	"math/big"
)

// ValidateSurvivalPendingReward checks the existing handoff representation.
// It adds no category, address-format, deadline or reward-eligibility rule.
func ValidateSurvivalPendingReward(reward *SurvivalPendingReward) error {
	if reward == nil {
		return fmt.Errorf("survival pending reward must not be nil")
	}
	if reward.ClaimId == "" || reward.FactId == "" || reward.Recipient == "" {
		return fmt.Errorf("survival pending reward requires claim_id, fact_id and recipient")
	}
	// Claim submission uses this same base-10 parser and retains the original
	// string. In particular, a leading '+' or zeroes must not strand a reward
	// that the existing submission path already accepted.
	amount, ok := new(big.Int).SetString(reward.Amount, 10)
	if !ok || amount.Sign() <= 0 {
		return fmt.Errorf("survival pending reward amount must be a positive base-10 integer")
	}
	return nil
}

// ValidateSurvivalPendingRewards refuses ambiguous duplicate primary records.
func ValidateSurvivalPendingRewards(rewards []*SurvivalPendingReward) error {
	seen := make(map[string]bool, len(rewards))
	for index, reward := range rewards {
		if err := ValidateSurvivalPendingReward(reward); err != nil {
			return fmt.Errorf("survival pending reward %d: %w", index, err)
		}
		if seen[reward.FactId] {
			return fmt.Errorf("duplicate survival pending fact_id: %s", reward.FactId)
		}
		seen[reward.FactId] = true
	}
	return nil
}
