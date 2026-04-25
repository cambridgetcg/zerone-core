package types

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultParams returns the default capture-challenge parameters.
//
// These values are this module's expression of commitment 9 from
// docs/TRUTH_SEEKING.md (cartel detection has consequence). See doc.go
// for the contract; the values below are how that contract is priced.
func DefaultParams() *Params {
	return &Params{
		// MinChallengeStake is the cost of accusing a validator of
		// cartel behavior. Set too low, the system drowns in
		// nuisance challenges and real ones are diluted. Set too
		// high, only well-funded actors can challenge — and the
		// chain wants the asymmetry to be the OTHER way: easy enough
		// to challenge a real cartel, costly enough to discourage
		// nuisance. 10 ZRN is meaningful but not gatekeeping.
		MinChallengeStake:         "10000000",  // 10 ZRN — accusation costs

		// EvidencePeriod + ReviewPeriod are the twin time windows
		// that operationalise commitment 9: detection only matters
		// if the chain actually adjudicates. Evidence period gives
		// the challenger time to produce the case; review period
		// gives the panel time to weigh it. Both bounded so an
		// allegation cannot hang forever.
		EvidencePeriodBlocks:      5000,        // ~3.5 hours — challenger's burden of proof window
		ReviewPeriodBlocks:        20000,       // ~14 hours — panel's adjudication window

		// DomainPauseBlocks: when an UPHELD challenge raises the
		// verification threshold for the affected domain, the pause
		// is the cooling-off interval before normal verification
		// resumes. Short enough not to halt the domain; long enough
		// for honest validators to notice and re-evaluate their
		// recent votes.
		DomainPauseBlocks:         1000,        // ~42 minutes

		// RewardRateBps + SlashRateBps are the asymmetric payoffs
		// of commitment 9. Successful challengers receive 10% of
		// the bounty pool; convicted validators lose 5% of stake.
		// The reward to challengers is paid from the per-fact
		// bounty contribution above, so the chain pays for its own
		// cartel detection (commitment 12) without taxing victims.
		RewardRateBps:             100000,      // 10% — challengers paid from bounty pool
		SlashRateBps:              50000,       // 5%  — convicted validators lose 5% of stake
		BountyContributionPerFact: "1000",      // 0.001 ZRN — every fact funds future challenges
		RiskAnalysisInterval:      1000,        // ~42 minutes — risk model recomputes regularly
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
