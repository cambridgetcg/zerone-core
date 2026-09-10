package keeper

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"math/big"

	corestore "cosmossdk.io/core/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

func vestingStateKey(prefix []byte, suffix string) []byte {
	key := make([]byte, len(prefix)+len(suffix))
	copy(key, prefix)
	copy(key[len(prefix):], suffix)
	return key
}

// createKnowledgeVestingSchedule uses the existing claim index as its retry
// identity. It never reconstructs an already-created schedule from today's
// height or category settings, nor changes its release progress or status.
func (k Keeper) createKnowledgeVestingSchedule(ctx sdk.Context, claimID, factID, recipient, amount, epistemicCategory string) error {
	if claimID == "" || factID == "" || recipient == "" {
		return fmt.Errorf("knowledge vesting requires claim, fact and recipient")
	}
	total, ok := new(big.Int).SetString(amount, 10)
	if !ok || total.Sign() <= 0 {
		return types.ErrInvalidRewardAmount
	}
	category := mapEpistemicToVestingCategory(epistemicCategory)
	store := k.storeService.OpenKVStore(ctx)
	id, err := store.Get(vestingStateKey(types.ClaimRecordKeyPrefix, claimID))
	if err != nil {
		return fmt.Errorf("read knowledge claim index: %w", err)
	}
	if id != nil {
		if len(id) == 0 {
			return fmt.Errorf("empty knowledge claim index")
		}
		bz, err := store.Get(vestingStateKey(types.VestingScheduleKeyPrefix, string(id)))
		if err != nil {
			return fmt.Errorf("read indexed knowledge schedule: %w", err)
		}
		schedule, err := decodeIndexedVestingSchedule(bz, string(id))
		if err != nil {
			return err
		}
		storedTotal, ok := new(big.Int).SetString(schedule.TotalAmount, 10)
		if !ok || storedTotal.Cmp(total) != 0 || schedule.ClaimId != claimID ||
			schedule.FactId != factID || schedule.Recipient != recipient ||
			schedule.Category != string(category) || schedule.Source != string(types.SourceVerification) {
			return fmt.Errorf("knowledge claim conflicts with existing vesting schedule")
		}
		// Validate only basic stored arithmetic, never recompute historical
		// reserve or release settings from a potentially changed category.
		progress := make([]*big.Int, 0, 3)
		for _, amount := range []string{schedule.ReleasedAmount, schedule.ClaimableAmount, schedule.ReserveAmount} {
			value, ok := new(big.Int).SetString(amount, 10)
			if !ok || value.Sign() < 0 || value.Cmp(total) > 0 {
				return fmt.Errorf("malformed knowledge vesting progress")
			}
			progress = append(progress, value)
		}
		if new(big.Int).Add(progress[0], progress[1]).Cmp(total) > 0 {
			return fmt.Errorf("knowledge vesting release exceeds total")
		}
		return checkVestingScheduleIndexes(store, schedule)
	}

	cfg, err := k.knowledgeCategoryConfig(ctx, category)
	if err != nil {
		return err
	}
	_, err = k.createVestingScheduleWithConfig(ctx, claimID, factID, recipient, amount, category, types.SourceVerification, cfg)
	return err
}

// True absence retains the legacy category default. Unreadable or corrupt
// stored configuration cannot silently select different economic settings.
func (k Keeper) knowledgeCategoryConfig(ctx sdk.Context, category types.VestingCategoryStr) (*types.CategoryConfig, error) {
	bz, err := k.storeService.OpenKVStore(ctx).Get(vestingStateKey(types.CategoryConfigKeyPrefix, string(category)))
	if err != nil {
		return nil, fmt.Errorf("read knowledge vesting category: %w", err)
	}
	var cfg *types.CategoryConfig
	if bz == nil {
		for _, candidate := range types.DefaultCategoryConfigs() {
			if candidate.Category == string(category) {
				cfg = candidate
				break
			}
		}
	} else {
		cfg = new(types.CategoryConfig)
		if err := proto.Unmarshal(bz, cfg); err != nil {
			return nil, fmt.Errorf("decode knowledge vesting category: %w", err)
		}
	}
	if cfg == nil || cfg.Category != string(category) || cfg.MaxRelease > 1_000_000 || cfg.HalfLifeBlocks == 0 ||
		ctx.BlockHeight() < 0 || cfg.CliffBlocks > math.MaxUint64-uint64(ctx.BlockHeight()) {
		return nil, types.ErrInvalidCategory
	}
	return cfg, nil
}

func decodeIndexedVestingSchedule(bz []byte, id string) (*types.VestingSchedule, error) {
	if len(bz) == 0 {
		return nil, fmt.Errorf("vesting index has no primary schedule")
	}
	schedule := new(types.VestingSchedule)
	if err := proto.Unmarshal(bz, schedule); err != nil {
		return nil, fmt.Errorf("decode indexed vesting schedule: %w", err)
	}
	if schedule.Id != id || id == "" || schedule.Recipient == "" {
		return nil, fmt.Errorf("vesting primary key/identity mismatch")
	}
	if _, err := vestingScheduleActive(schedule.Status); err != nil {
		return nil, err
	}
	return schedule, nil
}

func vestingScheduleActive(status string) (bool, error) {
	switch status {
	case string(types.VestingStatusActive), string(types.VestingStatusPaused):
		return true, nil
	case string(types.VestingStatusCompleted), string(types.VestingStatusFalsified), string(types.VestingStatusAbandoned):
		return false, nil
	default:
		return false, fmt.Errorf("unknown vesting schedule status")
	}
}

func checkVestingScheduleIndexes(store corestore.KVStore, schedule *types.VestingSchedule) error {
	recipient, err := store.Get(vestingStateKey(types.VestingByRecipientPrefix, schedule.Recipient+"/"+schedule.Id))
	if err != nil {
		return fmt.Errorf("read vesting recipient index: %w", err)
	}
	if !bytes.Equal(recipient, []byte{1}) {
		return fmt.Errorf("missing or malformed vesting recipient index")
	}
	active, err := store.Get(vestingStateKey(types.ActiveVestingPrefix, schedule.Id))
	if err != nil {
		return fmt.Errorf("read vesting active index: %w", err)
	}
	wantActive, err := vestingScheduleActive(schedule.Status)
	if err != nil {
		return err
	}
	if (wantActive && !bytes.Equal(active, []byte{1})) || (!wantActive && active != nil) {
		return fmt.Errorf("vesting status/active index mismatch")
	}
	return nil
}

// ValidateKnowledgeVestingState checks the primary/index assumptions needed by
// constant-time knowledge handoffs. It is for a one-time upgrade check, never
// a per-reward scan. Multiple legacy schedules for one claim are preserved:
// their explicit claim index must select one existing schedule for that claim.
// The scan admits at most 100,000 selected keys and 64 MiB of key/value bytes.
// Exceeding either refuses this upgrade; these are not ordinary admission caps.
func (k Keeper) ValidateKnowledgeVestingState(ctx sdk.Context) error {
	return k.validateKnowledgeVestingState(ctx, 100_000, 64<<20)
}

// The SDK cache iterators report these exact errors at normal exhaustion.
// Every other traversal error, and every Close error, remains fatal.
func terminalVestingIteratorError(iter corestore.Iterator) error {
	err := iter.Error()
	if err != nil && !iter.Valid() && (err.Error() == "invalid cacheMergeIterator" || err.Error() == "invalid memIterator") {
		return nil
	}
	return err
}

func (k Keeper) validateKnowledgeVestingState(ctx sdk.Context, maxEntries, maxBytes int) error {
	store := k.storeService.OpenKVStore(ctx)
	schedules := make(map[string]*types.VestingSchedule)
	claims := make(map[string]string)
	recipients := make(map[string]bool)
	active := make(map[string]bool)
	entries, size := 0, 0
	for _, prefix := range [][]byte{types.VestingScheduleKeyPrefix, types.ClaimRecordKeyPrefix, types.VestingByRecipientPrefix, types.ActiveVestingPrefix} {
		iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
		if err != nil {
			return fmt.Errorf("scan vesting indexes: %w", err)
		}
		err = func() (err error) {
			defer func() { err = errors.Join(err, terminalVestingIteratorError(iter), iter.Close()) }()
			for ; iter.Valid(); iter.Next() {
				key, value := iter.Key(), iter.Value()
				entries++
				size += len(key) + len(value)
				if entries > maxEntries || size > maxBytes {
					return fmt.Errorf("vesting index validation exceeds upgrade scan bounds")
				}
				if !bytes.HasPrefix(key, prefix) || len(key) <= len(prefix) {
					return fmt.Errorf("invalid vesting index key")
				}
				suffix := string(key[len(prefix):])
				switch prefix[0] {
				case types.VestingScheduleKeyPrefix[0]:
					schedule, err := decodeIndexedVestingSchedule(value, suffix)
					if err != nil {
						return err
					}
					schedules[suffix] = schedule
				case types.ClaimRecordKeyPrefix[0]:
					if len(value) == 0 {
						return fmt.Errorf("empty vesting claim index")
					}
					claims[suffix] = string(value)
				case types.VestingByRecipientPrefix[0], types.ActiveVestingPrefix[0]:
					if !bytes.Equal(value, []byte{1}) {
						return fmt.Errorf("malformed vesting membership index")
					}
					if prefix[0] == types.VestingByRecipientPrefix[0] {
						recipients[suffix] = true
					} else {
						active[suffix] = true
					}
				}
			}
			return nil
		}()
		if err != nil {
			return err
		}
	}
	for _, schedule := range schedules {
		if schedule.ClaimId != "" {
			selected := schedules[claims[schedule.ClaimId]]
			if selected == nil || selected.ClaimId != schedule.ClaimId {
				return fmt.Errorf("vesting primary has missing or conflicting claim index")
			}
		}
		key := schedule.Recipient + "/" + schedule.Id
		if !recipients[key] {
			return fmt.Errorf("vesting primary has no recipient index")
		}
		delete(recipients, key)
		wantActive, _ := vestingScheduleActive(schedule.Status)
		if active[schedule.Id] != wantActive {
			return fmt.Errorf("vesting primary has inconsistent active index")
		}
		delete(active, schedule.Id)
	}
	for claim, id := range claims {
		if schedule := schedules[id]; schedule == nil || schedule.ClaimId != claim {
			return fmt.Errorf("orphan or conflicting vesting claim index")
		}
	}
	if len(recipients) != 0 || len(active) != 0 {
		return fmt.Errorf("orphan vesting membership index")
	}
	return nil
}
