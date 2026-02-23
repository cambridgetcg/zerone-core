package types

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func DefaultParams() *Params {
	return &Params{
		MinChallengeStake:         "10000000",  // 10 ZRN
		EvidencePeriodBlocks:      5000,        // ~3.5 hours
		ReviewPeriodBlocks:        20000,       // ~14 hours
		DomainPauseBlocks:         1000,        // ~42 minutes
		RewardRateBps:             100000,      // 10% of bounty pool
		SlashRateBps:              50000,       // 5% of validator stake
		BountyContributionPerFact: "1000",      // 0.001 ZRN per fact
		RiskAnalysisInterval:      1000,        // every ~42 minutes
	}
}

func (p *Params) Validate() error {
	stake := new(big.Int)
	if _, ok := stake.SetString(p.MinChallengeStake, 10); !ok || stake.Sign() <= 0 {
		return fmt.Errorf("min_challenge_stake must be a positive integer string")
	}
	if p.EvidencePeriodBlocks == 0 {
		return fmt.Errorf("evidence_period_blocks must be > 0")
	}
	if p.ReviewPeriodBlocks == 0 {
		return fmt.Errorf("review_period_blocks must be > 0")
	}
	if p.RewardRateBps > 1_000_000 {
		return fmt.Errorf("reward_rate_bps must be <= 1_000_000")
	}
	if p.SlashRateBps > 1_000_000 {
		return fmt.Errorf("slash_rate_bps must be <= 1_000_000")
	}
	return nil
}

// ---------- Msg Validation ----------

func (msg *MsgSubmitChallenge) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Challenger); err != nil {
		return fmt.Errorf("invalid challenger address: %w", err)
	}
	if msg.Domain == "" {
		return fmt.Errorf("domain is required")
	}
	if len(msg.AccusedValidators) == 0 {
		return fmt.Errorf("at least one accused validator is required")
	}
	stake := new(big.Int)
	if _, ok := stake.SetString(msg.Stake, 10); !ok || stake.Sign() <= 0 {
		return fmt.Errorf("stake must be a positive integer string")
	}
	return nil
}

func (msg *MsgSubmitChallenge) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Challenger)
	return []sdk.AccAddress{addr}
}

func (msg *MsgAddEvidence) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Challenger); err != nil {
		return fmt.Errorf("invalid challenger address: %w", err)
	}
	if msg.ChallengeId == "" {
		return fmt.Errorf("challenge_id is required")
	}
	if msg.Description == "" {
		return fmt.Errorf("description is required")
	}
	return nil
}

func (msg *MsgAddEvidence) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Challenger)
	return []sdk.AccAddress{addr}
}

func (msg *MsgResolveChallenge) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if msg.ChallengeId == "" {
		return fmt.Errorf("challenge_id is required")
	}
	if msg.Outcome == ChallengeOutcome_CHALLENGE_OUTCOME_UNSPECIFIED {
		return fmt.Errorf("outcome must be specified")
	}
	return nil
}

func (msg *MsgResolveChallenge) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

func (msg *MsgFundBountyPool) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if msg.Domain == "" {
		return fmt.Errorf("domain is required")
	}
	amount := new(big.Int)
	if _, ok := amount.SetString(msg.Amount, 10); !ok || amount.Sign() <= 0 {
		return fmt.Errorf("amount must be a positive integer string")
	}
	return nil
}

func (msg *MsgFundBountyPool) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}
