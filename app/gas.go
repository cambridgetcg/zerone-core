// Package app provides gas cost definitions for Zerone transactions.
//
// Translated from core/billing/gas.ts. All gas costs are denominated in gas units.
// Fee = gasUsed * gasPrice, where gasPrice is in uzrn per gas unit.
package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Gas limits.
const (
	BlockGasLimit       uint64 = 33_333_333
	TxGasLimit          uint64 = 11_111_111
	SelfInvocationLimit uint64 = 5_555_555
	MinGasLimit         uint64 = 22_222
)

// Gas price constants (in uzrn per gas unit).
const (
	MinGasPrice      uint64 = 1
	BaseGasPrice     uint64 = 1_111_111
	PriorityMultiple uint64 = 2
)

// TransactionGasCosts maps transaction types to their base gas cost.
// Translated from TRANSACTION_GAS_COSTS in core/billing/gas.ts.
var TransactionGasCosts = map[string]uint64{
	// Core transactions
	"transfer": 21_000,
	"stake":    50_000,
	"unstake":  50_000,

	// Knowledge / consensus
	"claim_submit":        100_000,
	"verification_commit": 30_000,
	"verification_reveal": 40_000,
	"challenge_fact":      80_000,
	"add_fact":            120_000,

	// Staking / delegation
	"delegate":           60_000,
	"undelegate":         60_000,
	"register_validator": 100_000,
	"redelegate":         80_000,

	// Staking (extended — hand-written Zerone staking types)
	"update_validator_stake": 60_000,
	"zerone_delegate":        60_000,
	"zerone_undelegate":      60_000,
	"zerone_redelegate":      80_000,

	// System transactions (typically fee-exempt when issued by consensus)
	"verification_reward": 50_000,
	"slash_validator":     80_000,
	"epoch_transition":    0,
	"emergency_halt":      0,
	"emergency_revert":    0,
	"emergency_resume":    0,
	"block_reward":        0,

	// Agent identity
	"register_account":      50_000,
	"rotate_key":            30_000,
	"create_session":        40_000,
	"revoke_session":        25_000,
	"recover_account":       80_000,
	"freeze_account":        30_000,
	"unfreeze_account":      30_000,
	"set_recovery_config":   60_000,
	"initiate_recovery":     80_000,
	"submit_recovery_shard": 40_000,
	"challenge_recovery":    50_000,
	"execute_recovery":      80_000,

	// Knowledge pruning
	"patronize_fact": 50_000,
	"rate_fact":      20_000,

	// Knowledge (extended — hand-written types)
	"submit_contradiction":       80_000,
	"challenge_provisional_fact": 80_000,
	"update_extended_params":     40_000,

	// Ontology
	"propose_domain":   80_000,
	"endorse_domain":   30_000,
	"challenge_domain": 50_000,
	"register_stratum": 60_000,

	// Governance
	"governance_propose":   100_000,
	"governance_vote":      30_000,
	"governance_stake_lip": 50_000,
	"submit_lip":           100_000,
	"cast_vote":            30_000,
	"lock_vote":            40_000,
	"unlock_vote":          30_000,
	"switch_vote":          35_000,
	"commit_review":        40_000,
	"reveal_review":        50_000,
	"finalize_review":      60_000,
	"submit_disbursement":  80_000,
	"execute_disbursement": 100_000,

	// Vesting / rewards
	"create_vesting":     80_000,
	"claim_vesting":      40_000,
	"pause_vesting":      25_000,
	"resume_vesting":     25_000,
	"accelerate_vesting": 50_000,
	"falsify_vesting":    60_000,
	"complete_vesting":   40_000,

	// Emergency
	"propose_halt":   50_000,
	"vote_halt":      25_000,
	"propose_revert": 50_000,
	"vote_revert":    25_000,
	"propose_resume": 50_000,
	"vote_resume":    25_000,

	// Claiming pots
	"create_pot":     80_000,
	"fund_pot":       40_000,
	"claim_from_pot": 30_000,
	"close_pot":      40_000,

	// Agent Home
	"create_home":        150_000,
	"update_home":        40_000,
	"update_memory_cid":  30_000,
	"home_start_session": 30_000,
	"home_end_session":   20_000,
	"home_register_key":  50_000,
	"home_revoke_key":    25_000,
	"configure_guardian": 50_000,
	"acknowledge_alert":  20_000,
	"set_spending_limit": 40_000,

	// Domain qualification
	"request_qualification":       80_000,
	"endorse_validator":           50_000,
	"renew_qualification":         30_000,
	"withdraw_qualification":      50_000,
	"update_qualification_params": 40_000,

	// Capture defense
	"request_capture_qualification":  80_000,
	"endorse_capture_qualification":  40_000,
	"update_capture_defense_params":  40_000,

	// Capture challenge
	"submit_capture_challenge":        100_000,
	"add_challenge_evidence":          50_000,
	"resolve_capture_challenge":       80_000,
	"update_capture_challenge_params": 40_000,

	// Ontology (extended)
	"vote_domain_proposal":       30_000,
	"update_domain":              40_000,
	"register_logic_zone":        60_000,
	"acknowledge_incompleteness": 30_000,
	"update_ontology_params":     40_000,

	// Governance (extended)
	"attach_upgrade_plan":   60_000,
	"amend_lip":             60_000,
	"advance_lip_stage":     40_000,
	"withdraw_lip":          30_000,
	"stake_disbursement":    50_000,
	"advance_disbursement":  40_000,
	"cancel_disbursement":   30_000,
	"submit_research_spend": 80_000,
	"vote_research_spend":   30_000,
	"set_research_voters":   50_000,
	"register_operator":     80_000,
	"add_agent":             30_000,
	"remove_agent":          30_000,
	"slash_operator":        80_000,
	"create_deployment":     80_000,
	"advance_deployment":    40_000,
	"approve_deployment":    40_000,
	"rollback_deployment":   60_000,

	// Alignment
	"activate_alignment": 40_000,

	// Liquidity pool
	"create_pool":     80_000,
	"lp_swap":         40_000,
	"lp_add_liquidity":    60_000,
	"lp_remove_liquidity": 60_000,

	// IBC rate limiting
	"add_rate_limit":    40_000,
	"remove_rate_limit": 30_000,
}

// Fee routing constants (basis points, 1000000 = 100%).
const (
	ResearchContributionBPS uint64 = 70000  // 7% to research fund
	ValidatorFeeBPS         uint64 = 930000 // 93% to block producer / validators
	BurnRateBPS             uint64 = 50000  // 5% burn rate (applied at penalty sites)
)

// Bootstrap gas-free period: first 480,000 blocks (~14 days at 2521ms).
// During bootstrap, essential PoT transactions are gas-free to allow the
// network to begin verifying truth before any ZRN is minted.
const BootstrapEndBlock int64 = 480_000

// BootstrapGasFreeTypes lists message type URLs exempt from gas during bootstrap.
var BootstrapGasFreeTypes = map[string]bool{
	"/zerone.knowledge.v1.MsgSubmitClaim":      true,
	"/zerone.knowledge.v1.MsgSubmitCommitment": true,
	"/zerone.knowledge.v1.MsgSubmitReveal":     true,
	"/zerone.staking.v1.MsgRegisterValidator":  true,
	"/zerone.auth.v1.MsgRegisterAccount":       true,
}

// Module account names for fee routing.
const (
	ResearchFundName = "research_fund"
	BurnModuleName   = "burn"
)

// Protocol treasury module account name. NOTE: the chain also carries a
// "protocol_treasury" account (see app.go maccPerms) — a naming split-brain
// documented in docs/plans/2026-07-06-slim-cut-migration-map.md; do not
// "fix" one side without a coordinated state migration.
const (
	TreasuryProtocolName = "treasury_protocol"
)

// SystemTxTypes are transaction types that bypass gas fees (issued by consensus).
var SystemTxTypes = map[string]bool{
	"verification_reward": true,
	"slash_validator":     true,
	"epoch_transition":    true,
	"emergency_halt":      true,
	"emergency_revert":    true,
	"emergency_resume":    true,
	"block_reward":        true,
}

// EstimateTransactionGas returns the base gas cost for a transaction type.
// Returns MinGasLimit if the type is unknown.
func EstimateTransactionGas(txType string) uint64 {
	if cost, ok := TransactionGasCosts[txType]; ok {
		return cost
	}
	return MinGasLimit
}

// CalculateGasCost computes fee = gasUsed * gasPrice.
func CalculateGasCost(gasUsed uint64, gasPrice sdk.DecCoin) sdk.Coin {
	fee := gasPrice.Amount.MulInt64(int64(gasUsed)).TruncateInt()
	return sdk.NewCoin(gasPrice.Denom, fee)
}

// IsSystemTransaction returns true if the tx type bypasses gas fees.
func IsSystemTransaction(txType string) bool {
	return SystemTxTypes[txType]
}
