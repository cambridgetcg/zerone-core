package types

import (
	"fmt"
	"math/big"
	"reflect"

	commontypes "github.com/zerone-chain/zerone/x/common/types"
)

// DefaultRevenueSplit returns the default 4-way revenue split.
// contributor 55%, protocol 22%, research 3.33%, development 19.67%.
func DefaultRevenueSplit() *commontypes.RevenueSplit {
	return &commontypes.RevenueSplit{
		ContributorBps: 550000, // 55%
		ProtocolBps:    220000, // 22%
		ResearchBps:    33300,  // 3.33%
		DevelopmentBps: 196700, // 19.67%
	}
}

// DefaultProtocolSubSplit returns the default protocol sub-split.
// citation 50%, verification 30%, treasury 20%.
func DefaultProtocolSubSplit() *commontypes.ProtocolSubSplit {
	return &commontypes.ProtocolSubSplit{
		CitationBps:     500000,
		VerificationBps: 300000,
		TreasuryBps:     200000,
	}
}

// DefaultParams returns default module parameters.
func DefaultParams() *Params {
	return &Params{
		BlockReward:                "0",    // arbitrary-transaction block minting retired
		RewardDecayBps:             994478, // ~1-year half-life (0.994478x per 100K-block epoch)
		BlocksPerRewardEpoch:       100000, // ~2.9 days at 2521ms
		RevenueSplit:               DefaultRevenueSplit(),
		ProtocolSubSplit:           DefaultProtocolSubSplit(),
		FounderShareBps:            0,  // retired wire field
		FounderAddress:             "", // retired wire field
		GovernanceActivationHeight: 0,  // no sunset
		CategoryRewardConfigs:      DefaultCategoryRewardConfigs(),
		ResearchFundModuleAccount:  ResearchFundModuleName,
		VestingEnabled:             true,
		ReleasedClawbackRate:       3300, // 33% of released clawed back
		MinValidatorsForFullReward: 22,
		EmptyBlockRewardRate:       0,   // 0% for empty blocks (PoT)
		FloorReward:                "0", // arbitrary-transaction block minting retired
		InitialFundBalance:         "0", // pure PoT

		// Retired reward-coupling wire fields, pinned for deterministic legacy
		// queries. Consensus v2 never uses them to drive automatic issuance.
		KnowledgeCouplingTargetBps: 700_000,
		KnowledgeCouplingFloorBps:  500_000,
	}
}

// DefaultCategoryRewardConfigs returns retired per-category compatibility data.
func DefaultCategoryRewardConfigs() []*CategoryRewardConfig {
	return []*CategoryRewardConfig{
		{Category: string(CategoryAxiomatic), MultiplierBps: 1200000},     // 1.2x
		{Category: string(CategoryFormalProof), MultiplierBps: 1100000},   // 1.1x
		{Category: string(CategoryOnChain), MultiplierBps: 1000000},       // 1.0x
		{Category: string(CategoryCryptographic), MultiplierBps: 1050000}, // 1.05x
		{Category: string(CategoryComputational), MultiplierBps: 1000000}, // 1.0x
		{Category: string(CategoryPeerReviewed), MultiplierBps: 900000},   // 0.9x
		{Category: string(CategoryReplicated), MultiplierBps: 950000},     // 0.95x
		{Category: string(CategoryOracleFeed), MultiplierBps: 800000},     // 0.8x
		{Category: string(CategoryAttestation), MultiplierBps: 850000},    // 0.85x
		{Category: string(CategoryContested), MultiplierBps: 600000},      // 0.6x
	}
}

// DefaultCategoryConfigs returns release curve configs based on scientometric research.
func DefaultCategoryConfigs() []*CategoryConfig {
	return []*CategoryConfig{
		{Category: string(CategoryAxiomatic), HalfLifeBlocks: 1_111_111, CliffBlocks: 11111, MaxRelease: 950000},
		{Category: string(CategoryFormalProof), HalfLifeBlocks: 555_555, CliffBlocks: 5555, MaxRelease: 920000},
		{Category: string(CategoryOnChain), HalfLifeBlocks: 222_222, CliffBlocks: 1111, MaxRelease: 900000},
		{Category: string(CategoryCryptographic), HalfLifeBlocks: 222_222, CliffBlocks: 3333, MaxRelease: 900000},
		{Category: string(CategoryComputational), HalfLifeBlocks: 333_333, CliffBlocks: 2222, MaxRelease: 880000},
		{Category: string(CategoryPeerReviewed), HalfLifeBlocks: 111_111, CliffBlocks: 5555, MaxRelease: 850000},
		{Category: string(CategoryReplicated), HalfLifeBlocks: 111_111, CliffBlocks: 3333, MaxRelease: 880000},
		{Category: string(CategoryOracleFeed), HalfLifeBlocks: 55_555, CliffBlocks: 555, MaxRelease: 800000},
		{Category: string(CategoryAttestation), HalfLifeBlocks: 77_777, CliffBlocks: 2222, MaxRelease: 800000},
		{Category: string(CategoryContested), HalfLifeBlocks: 22_222, CliffBlocks: 1111, MaxRelease: 600000},
	}
}

// DefaultGenesis returns the default genesis state.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:           DefaultParams(),
		CategoryConfigs:  DefaultCategoryConfigs(),
		VestingSchedules: []*VestingSchedule{},
	}
}

// Validate performs basic genesis state validation.
func (gs *GenesisState) Validate() error {
	if gs.Params != nil {
		if err := ValidateParams(gs.Params); err != nil {
			return err
		}
	}
	if len(gs.CategoryConfigs) == 0 {
		return fmt.Errorf("at least one category config required")
	}
	for _, cfg := range gs.CategoryConfigs {
		if cfg == nil {
			return fmt.Errorf("category config must not be nil")
		}
		if cfg.MaxRelease > 1000000 {
			return fmt.Errorf("max release for %s cannot exceed 100%% (1000000 bps)", cfg.Category)
		}
		if cfg.HalfLifeBlocks == 0 {
			return fmt.Errorf("half-life for %s must be positive", cfg.Category)
		}
	}

	schedules := make(map[string]*VestingSchedule, len(gs.VestingSchedules))
	claimIDs := make(map[string]struct{})
	for i, schedule := range gs.VestingSchedules {
		if schedule == nil {
			return fmt.Errorf("vesting schedule at index %d must not be nil", i)
		}
		if schedule.Id == "" {
			return fmt.Errorf("vesting schedule at index %d has empty id", i)
		}
		if _, duplicate := schedules[schedule.Id]; duplicate {
			return fmt.Errorf("duplicate vesting schedule id %q", schedule.Id)
		}
		schedules[schedule.Id] = schedule
		if schedule.ClaimId != "" {
			claimIDs[schedule.ClaimId] = struct{}{}
		}
	}

	// Empty indexes are accepted for backward-compatible genesis and are
	// derived by schedule replay. Once indexes are present, require a complete,
	// internally consistent snapshot of the live claim lookup.
	if len(gs.ClaimScheduleIndexes) > 0 {
		seenClaims := make(map[string]struct{}, len(gs.ClaimScheduleIndexes))
		for i, index := range gs.ClaimScheduleIndexes {
			if index == nil || index.ClaimId == "" || index.VestingId == "" {
				return fmt.Errorf("claim schedule index at position %d must name claim_id and vesting_id", i)
			}
			if _, duplicate := seenClaims[index.ClaimId]; duplicate {
				return fmt.Errorf("duplicate claim schedule index for claim %q", index.ClaimId)
			}
			seenClaims[index.ClaimId] = struct{}{}
			schedule, found := schedules[index.VestingId]
			if !found {
				return fmt.Errorf("claim %q indexes missing vesting schedule %q", index.ClaimId, index.VestingId)
			}
			if schedule.ClaimId != index.ClaimId {
				return fmt.Errorf(
					"claim %q index targets schedule %q with claim_id %q",
					index.ClaimId,
					index.VestingId,
					schedule.ClaimId,
				)
			}
		}
		for claimID := range claimIDs {
			if _, found := seenClaims[claimID]; !found {
				return fmt.Errorf("claim %q has schedules but no claim schedule index", claimID)
			}
		}
	}
	return nil
}

// ValidateParams validates vesting_rewards module parameters.
func ValidateParams(p *Params) error {
	if p == nil {
		return fmt.Errorf("params must not be nil")
	}
	if err := validateRevenueSplit(p.RevenueSplit); err != nil {
		return err
	}
	if err := validateProtocolSubSplit(p.ProtocolSubSplit); err != nil {
		return err
	}
	if p.BlocksPerRewardEpoch == 0 {
		return fmt.Errorf("blocks_per_reward_epoch must be positive")
	}
	if p.RewardDecayBps == 0 {
		return fmt.Errorf("reward_decay_bps must be positive")
	}
	if p.RewardDecayBps > 1000000 {
		return fmt.Errorf("reward_decay_bps cannot exceed 1000000 (1.0)")
	}
	if err := validateNonNegativeInteger("block_reward", p.BlockReward); err != nil {
		return err
	}
	if err := validateNonNegativeInteger("floor_reward", p.FloorReward); err != nil {
		return err
	}
	if p.BlockReward != "0" || p.FloorReward != "0" || p.EmptyBlockRewardRate != 0 {
		return ErrAutomaticRewardRetired
	}
	if err := validateNonNegativeInteger("initial_fund_balance", p.InitialFundBalance); err != nil {
		return err
	}
	if p.FounderShareBps != 0 || p.FounderAddress != "" {
		return ErrFounderShareRetired
	}
	if p.ReleasedClawbackRate > 10_000 {
		return fmt.Errorf("released_clawback_rate cannot exceed 10000 (100%%)")
	}
	if p.EmptyBlockRewardRate > 10_000 {
		return fmt.Errorf("empty_block_reward_rate cannot exceed 10000 (100%%)")
	}
	if p.KnowledgeCouplingTargetBps > 1_000_000 {
		return fmt.Errorf("knowledge_coupling_target_bps cannot exceed 1000000 (100%%)")
	}
	if p.KnowledgeCouplingFloorBps > 1_000_000 {
		return fmt.Errorf("knowledge_coupling_floor_bps cannot exceed 1000000 (100%%)")
	}
	return nil
}

// FounderShareCapBps is retained for Go API compatibility. Version 2 retires
// the identity-based founder tap, so its only valid value is zero.
const FounderShareCapBps = 0

// ValidateFounderShareChange keeps the retired compatibility fields fixed at
// zero/empty. No governance vote can recreate an identity-based revenue tap.
func ValidateFounderShareChange(current *Params, proposed *Params) error {
	if proposed == nil {
		return nil
	}
	if proposed.FounderShareBps != 0 || proposed.FounderAddress != "" {
		return ErrFounderShareRetired
	}
	return nil
}

// ValidateRuntimeParamChange rejects updates to compatibility fields that
// current production logic does not consume. Letting queries advertise changed
// values while execution ignores them would create a false control surface.
func ValidateRuntimeParamChange(current, proposed *Params) error {
	if current == nil || proposed == nil {
		return fmt.Errorf("current and proposed params must not be nil")
	}
	if proposed.GovernanceActivationHeight != current.GovernanceActivationHeight {
		return fmt.Errorf("governance_activation_height is deprecated and runtime-immutable")
	}
	if !reflect.DeepEqual(proposed.CategoryRewardConfigs, current.CategoryRewardConfigs) {
		return fmt.Errorf("category_reward_configs are non-operative and runtime-immutable")
	}
	if proposed.ResearchFundModuleAccount != current.ResearchFundModuleAccount {
		return fmt.Errorf("research_fund_module_account is compatibility-only and runtime-immutable")
	}
	if proposed.VestingEnabled != current.VestingEnabled {
		return fmt.Errorf("vesting_enabled is compatibility-only and runtime-immutable")
	}
	if proposed.InitialFundBalance != current.InitialFundBalance {
		return fmt.Errorf("initial_fund_balance is genesis/export bookkeeping and runtime-immutable")
	}
	if proposed.RewardDecayBps != current.RewardDecayBps ||
		proposed.BlocksPerRewardEpoch != current.BlocksPerRewardEpoch ||
		proposed.MinValidatorsForFullReward != current.MinValidatorsForFullReward ||
		proposed.KnowledgeCouplingTargetBps != current.KnowledgeCouplingTargetBps ||
		proposed.KnowledgeCouplingFloorBps != current.KnowledgeCouplingFloorBps {
		return fmt.Errorf("retired block-reward schedule fields are runtime-immutable")
	}
	return nil
}

// validateRevenueSplit checks that the revenue split sums to 1,000,000.
func validateRevenueSplit(split *commontypes.RevenueSplit) error {
	if split == nil {
		return nil // defaults used
	}
	components := []struct {
		name  string
		value uint64
	}{
		{"contributor_bps", split.ContributorBps},
		{"protocol_bps", split.ProtocolBps},
		{"research_bps", split.ResearchBps},
		{"development_bps", split.DevelopmentBps},
	}
	for _, component := range components {
		if component.value > 1_000_000 {
			return fmt.Errorf("revenue split %s cannot exceed 1000000", component.name)
		}
	}
	total := split.ContributorBps + split.ProtocolBps + split.ResearchBps + split.DevelopmentBps
	if total != 1000000 {
		return fmt.Errorf("revenue split must sum to 1000000, got %d", total)
	}
	return nil
}

// validateProtocolSubSplit checks that the protocol sub-split sums to 1,000,000.
func validateProtocolSubSplit(split *commontypes.ProtocolSubSplit) error {
	if split == nil {
		return nil // defaults used
	}
	components := []struct {
		name  string
		value uint64
	}{
		{"citation_bps", split.CitationBps},
		{"verification_bps", split.VerificationBps},
		{"treasury_bps", split.TreasuryBps},
	}
	for _, component := range components {
		if component.value > 1_000_000 {
			return fmt.Errorf("protocol sub-split %s cannot exceed 1000000", component.name)
		}
	}
	total := split.CitationBps + split.VerificationBps + split.TreasuryBps
	if total != 1000000 {
		return fmt.Errorf("protocol sub-split must sum to 1000000, got %d", total)
	}
	return nil
}

func validatePositiveInteger(name, value string) error {
	n, ok := new(big.Int).SetString(value, 10)
	if !ok || n.Sign() <= 0 || n.String() != value {
		return fmt.Errorf("%s must be a canonical positive base-10 integer", name)
	}
	return nil
}

func validateNonNegativeInteger(name, value string) error {
	n, ok := new(big.Int).SetString(value, 10)
	if !ok || n.Sign() < 0 || n.String() != value {
		return fmt.Errorf("%s must be a canonical non-negative base-10 integer", name)
	}
	return nil
}
