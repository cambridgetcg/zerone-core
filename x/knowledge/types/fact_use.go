package types

import (
	"fmt"
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Version-1 ceilings are consensus backstops, not unbounded governance knobs.
const (
	FactUseReceiptVersion      uint32 = 1
	MaxFactUseConsumers               = 32
	MaxFactUsePerConsumerEpoch uint64 = 100
	MaxFactUsePerEpoch         uint64 = 1_000
	FactUseRetentionEpochs     uint64 = 2
	FactUsePruneBatchSize             = 100   // examined records, not just deletions
	MaxFactUseRetainedReceipts        = 2_000 // reject new reports when pruning lags
	MaxFactUseConsumerBytes           = 128
	MaxFactUseFactIDBytes             = 128
	MaxFactUseReceiptKeyBytes         = 1 + 8 + MaxFactUseConsumerBytes + 1 + MaxFactUseFactIDBytes
)

// CanonicalFactUseConsumer rejects aliases rather than silently changing the
// signed message. Admission, quota keys, receipt keys and rating must all use
// this same canonical SDK account spelling. No hard-coded bech32 HRP: use the
// chain's configured SDK account prefix and address verifier.
func CanonicalFactUseConsumer(consumer string) (string, error) {
	if len(consumer) == 0 || len(consumer) > MaxFactUseConsumerBytes {
		return "", fmt.Errorf("consumer must contain 1..%d bytes", MaxFactUseConsumerBytes)
	}
	addr, err := sdk.AccAddressFromBech32(consumer)
	if err != nil {
		return "", fmt.Errorf("invalid consumer address: %w", err)
	}
	canonical := addr.String()
	if consumer != canonical {
		return "", fmt.Errorf("consumer must be a canonical SDK account address")
	}
	return canonical, nil
}

// ValidateFactUseFactID bounds both work and composite keys. IDs are case
// sensitive; never normalize them into another fact. This accepts the existing
// hex IDs and portable named IDs, but no separators, controls or whitespace.
func ValidateFactUseFactID(factID string) error {
	if len(factID) == 0 || len(factID) > MaxFactUseFactIDBytes {
		return fmt.Errorf("fact_id must contain 1..%d bytes", MaxFactUseFactIDBytes)
	}
	for _, c := range []byte(factID) {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' {
			continue
		}
		return fmt.Errorf("fact_id must contain only ASCII letters, digits, '.', '_' or '-'")
	}
	return nil
}

// ValidateFactUsePolicy is also called by Params.Validate. Old params with
// absent limits need explicit migration defaults, not a zero-as-unlimited rule.
func (p *Params) ValidateFactUsePolicy() error {
	if p == nil {
		return fmt.Errorf("params must not be nil")
	}
	if len(p.FactUseConsumers) > MaxFactUseConsumers {
		return fmt.Errorf("fact_use_consumers exceeds %d accounts", MaxFactUseConsumers)
	}
	seen := make(map[string]bool, len(p.FactUseConsumers))
	for _, consumer := range p.FactUseConsumers {
		canonical, err := CanonicalFactUseConsumer(consumer)
		if err != nil {
			return fmt.Errorf("fact_use_consumers: %w", err)
		}
		if seen[canonical] {
			return fmt.Errorf("fact_use_consumers contains a duplicate account")
		}
		seen[canonical] = true
	}
	if p.FactUseMaxPerConsumerEpoch == 0 || p.FactUseMaxPerConsumerEpoch > MaxFactUsePerConsumerEpoch {
		return fmt.Errorf("fact_use_max_per_consumer_epoch must be 1..%d", MaxFactUsePerConsumerEpoch)
	}
	if p.FactUseMaxPerEpoch == 0 || p.FactUseMaxPerEpoch > MaxFactUsePerEpoch {
		return fmt.Errorf("fact_use_max_per_epoch must be 1..%d", MaxFactUsePerEpoch)
	}
	if p.FactUseMaxPerConsumerEpoch > p.FactUseMaxPerEpoch {
		return fmt.Errorf("fact_use_max_per_consumer_epoch must not exceed fact_use_max_per_epoch")
	}
	if p.FitnessEpochBlocks == 0 || p.FitnessEpochBlocks > math.MaxInt64/FactUseRetentionEpochs {
		return fmt.Errorf("fitness_epoch_blocks must permit a two-epoch receipt expiry within int64 heights")
	}
	// Params alone cannot observe historical self-reports. Stateful callers
	// must also use ValidateFactUseNonEconomicState with the persisted latch.
	return ValidateFactUseNonEconomicState(p, nil, false)
}

// ValidateFactUseNonEconomicState rejects economic eligibility for self-report
// counters while enabled or after any accepted report, even after disabling and
// pruning every receipt. A successful check is not economic authorization or
// proof of readership. Nil pruning is compatible with absent legacy state only
// when no receipts are retained. This validator never initializes or clears it.
// The keeper must set EverReported atomically on the first accepted report and
// never clear it in pruning or parameter updates; reversal requires a future
// explicit named upgrade. Epoch freezing is separate from this permanent latch.
func ValidateFactUseNonEconomicState(params *Params, pruning *FactUsePruningState, hasRetainedReceipts bool) error {
	if params == nil {
		return fmt.Errorf("params must not be nil")
	}
	if hasRetainedReceipts && !pruning.GetEverReported() {
		return fmt.Errorf("retained fact-use receipts require ever_reported to be true")
	}
	if (params.FactUseEnabled || pruning.GetEverReported()) && (params.FitnessWeightQueryBps != 0 || params.FitnessWeightSatisfactionBps != 0 || params.MetabolismEnergyPerQuery != 0) {
		return fmt.Errorf("fact use enabled or ever_reported requires fitness_weight_query_bps, fitness_weight_satisfaction_bps and metabolism_energy_per_query to be zero")
	}
	return nil
}

// ValidateFactUseParamChange is for the consensus UpdateParams path. The keeper
// must supply error-checked stored pruning state and a retained-state observation,
// never substitute nil/false on read failure. Include expired-but-unpruned
// markers: disabling the beta must not reinterpret their epoch keys or expiry.
// EverReported permanently guards economics but does not itself freeze epochs.
func ValidateFactUseParamChange(current, proposed *Params, pruning *FactUsePruningState, hasRetainedReceipts bool) error {
	if current == nil || proposed == nil {
		return fmt.Errorf("current and proposed params must not be nil")
	}
	if err := proposed.ValidateFactUsePolicy(); err != nil {
		return err
	}
	if err := ValidateFactUseNonEconomicState(proposed, pruning, hasRetainedReceipts); err != nil {
		return err
	}
	if (current.FactUseEnabled || proposed.FactUseEnabled || hasRetainedReceipts) && current.FitnessEpochBlocks != proposed.FitnessEpochBlocks {
		return fmt.Errorf("fitness_epoch_blocks is frozen while fact use is enabled or receipts are retained")
	}
	return nil
}

// FactUseEpoch shares the existing floor(height / FitnessEpochBlocks)
// convention. Epoch zero is a real epoch, never a current-epoch sentinel.
func FactUseEpoch(height int64, epochBlocks uint64) (uint64, error) {
	if height < 0 || epochBlocks == 0 || epochBlocks > math.MaxInt64/FactUseRetentionEpochs {
		return 0, fmt.Errorf("invalid fact-use height or epoch length")
	}
	return uint64(height) / epochBlocks, nil
}

// FactUseExpiryHeight is the exclusive marker-retention deadline, NOT the
// rating deadline and NOT evidence of the time of an off-chain use.
func FactUseExpiryHeight(epoch, epochBlocks uint64) (uint64, error) {
	if epochBlocks == 0 || epochBlocks > math.MaxInt64/FactUseRetentionEpochs || epoch > math.MaxUint64-FactUseRetentionEpochs {
		return 0, fmt.Errorf("invalid fact-use epoch or epoch length")
	}
	endEpoch := epoch + FactUseRetentionEpochs
	if endEpoch > math.MaxInt64/epochBlocks {
		return 0, fmt.Errorf("fact-use expiry exceeds supported block height")
	}
	return endEpoch * epochBlocks, nil
}

// Validate checks a receipt's self-contained version-1 shape against its frozen
// epoch convention. It does not prove a committed tx, fact existence, cohort
// admission at use time, current validity, or actual off-chain consumption.
func (r *FactUseReceipt) Validate(epochBlocks uint64) error {
	if r == nil || r.Version != FactUseReceiptVersion {
		return fmt.Errorf("unsupported or missing fact-use receipt version")
	}
	if _, err := CanonicalFactUseConsumer(r.Consumer); err != nil {
		return err
	}
	if err := ValidateFactUseFactID(r.FactId); err != nil {
		return err
	}
	if r.UseHeight == 0 || r.UseHeight > math.MaxInt64 {
		return fmt.Errorf("receipt use_height must be a positive int64 height")
	}
	epoch, err := FactUseEpoch(int64(r.UseHeight), epochBlocks)
	if err != nil {
		return err
	}
	if r.Epoch != epoch {
		return fmt.Errorf("receipt epoch does not match use_height")
	}
	expiry, err := FactUseExpiryHeight(epoch, epochBlocks)
	if err != nil {
		return err
	}
	if r.ExpiryHeight != expiry {
		return fmt.Errorf("receipt expiry_height does not match retention convention")
	}
	switch r.Rating {
	case FactUseRating_FACT_USE_RATING_UNRATED:
		if r.RatingHeight != 0 {
			return fmt.Errorf("unrated receipt must have zero rating_height")
		}
	case FactUseRating_FACT_USE_RATING_USEFUL, FactUseRating_FACT_USE_RATING_NOT_USEFUL:
		if r.RatingHeight < r.UseHeight || r.RatingHeight > math.MaxInt64 || r.RatingHeight/epochBlocks != epoch {
			return fmt.Errorf("rating_height must be at or after use_height in the same epoch")
		}
	default:
		return fmt.Errorf("invalid fact-use rating state")
	}
	return nil
}
