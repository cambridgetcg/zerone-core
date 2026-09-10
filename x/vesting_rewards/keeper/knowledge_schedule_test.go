package keeper

import (
	"bytes"
	"context"
	"errors"
	"testing"

	corestore "cosmossdk.io/core/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

func setupKnowledgeScheduleKeeper(t *testing.T) (Keeper, sdk.Context) {
	t.Helper()
	k, ctx := setupStrictMigrationKeeper(t)
	k.InitGenesis(ctx, types.DefaultGenesis())
	return k, ctx
}

func scheduleRecipient() string { return sdk.AccAddress(bytes.Repeat([]byte{0x35}, 20)).String() }

func seedKnowledgeSchedule(t *testing.T, k Keeper, ctx sdk.Context) *types.VestingSchedule {
	t.Helper()
	require.NoError(t, k.CreateVestingScheduleFromKnowledge(ctx, "claim", "fact", scheduleRecipient(), "200000", "empirical"))
	schedule, found := k.GetVestingByClaimId(ctx, "claim")
	require.True(t, found)
	return schedule
}

func vestingSnapshot(t *testing.T, k Keeper, ctx sdk.Context) map[string][]byte {
	t.Helper()
	iter, err := k.storeService.OpenKVStore(ctx).Iterator(nil, nil)
	require.NoError(t, err)
	defer iter.Close()
	result := make(map[string][]byte)
	for ; iter.Valid(); iter.Next() {
		result[string(iter.Key())] = bytes.Clone(iter.Value())
	}
	require.NoError(t, iter.Error())
	return result
}

func TestKnowledgeScheduleRetryPreservesEveryStoredField(t *testing.T) {
	for _, status := range []types.VestingStatus{types.VestingStatusActive, types.VestingStatusPaused, types.VestingStatusCompleted, types.VestingStatusFalsified, types.VestingStatusAbandoned} {
		t.Run(string(status), func(t *testing.T) {
			k, ctx := setupKnowledgeScheduleKeeper(t)
			schedule := seedKnowledgeSchedule(t, k, ctx)
			schedule.Status = string(status)
			schedule.ReleasedAmount, schedule.ClaimableAmount = "75000", "1000"
			schedule.TotalPausedBlocks, schedule.PausedAtBlock = 34, 1050
			schedule.LastClaimBlock, schedule.UpdatedAt = 1080, 1090
			schedule.DefenseCount, schedule.ReplicationCount = 3, 4
			schedule.CorroborationCount, schedule.CitationCount = 5, 6
			k.SetVestingSchedule(ctx, schedule)
			// A retry must not consume today's changed settings or recreate
			// timestamps and progress using the later invocation height.
			k.SetCategoryConfig(ctx, &types.CategoryConfig{Category: string(types.CategoryPeerReviewed), MaxRelease: 500000, CliffBlocks: 888})
			before := vestingSnapshot(t, k, ctx)
			eventCount := len(ctx.EventManager().Events())
			require.NoError(t, k.CreateVestingScheduleFromKnowledge(ctx.WithBlockHeight(2000), "claim", "fact", scheduleRecipient(), "200000", "empirical"))
			require.Equal(t, before, vestingSnapshot(t, k, ctx))
			require.Len(t, ctx.EventManager().Events(), eventCount)
			require.NoError(t, k.ValidateKnowledgeVestingState(ctx))
		})
	}
}

func TestKnowledgeScheduleRejectsConflictingRequest(t *testing.T) {
	for _, field := range []string{"fact", "recipient", "amount", "category", "source", "malformed-progress"} {
		t.Run(field, func(t *testing.T) {
			k, ctx := setupKnowledgeScheduleKeeper(t)
			schedule := seedKnowledgeSchedule(t, k, ctx)
			fact, recipient, amount, category := "fact", scheduleRecipient(), "200000", "empirical"
			switch field {
			case "fact":
				fact = "other-fact"
			case "recipient":
				recipient = "other-recipient"
			case "amount":
				amount = "200001"
			case "category":
				category = "formal"
			case "source":
				schedule.Source = string(types.SourceFalsification)
				k.SetVestingSchedule(ctx, schedule)
			case "malformed-progress":
				schedule.ReleasedAmount = "unknown"
				k.SetVestingSchedule(ctx, schedule)
			}
			before := vestingSnapshot(t, k, ctx)
			eventCount := len(ctx.EventManager().Events())
			require.Error(t, k.CreateVestingScheduleFromKnowledge(ctx.WithBlockHeight(2000), "claim", fact, recipient, amount, category))
			require.Equal(t, before, vestingSnapshot(t, k, ctx))
			require.Len(t, ctx.EventManager().Events(), eventCount)
		})
	}
}

type scheduleFault struct {
	operation  string
	prefix     []byte
	afterWrite bool
}
type scheduleFaultService struct {
	delegate corestore.KVStoreService
	fault    *scheduleFault
}

func (s scheduleFaultService) OpenKVStore(ctx context.Context) corestore.KVStore {
	return scheduleFaultStore{s.delegate.OpenKVStore(ctx), s.fault}
}

type scheduleFaultStore struct {
	corestore.KVStore
	fault *scheduleFault
}

func (s scheduleFaultStore) Iterator(start, end []byte) (corestore.Iterator, error) {
	iter, err := s.KVStore.Iterator(start, end)
	if err != nil || (s.fault.operation != "iterator-error" && s.fault.operation != "iterator-close") {
		return iter, err
	}
	return scheduleFaultIterator{iter, s.fault.operation}, nil
}

type scheduleFaultIterator struct {
	corestore.Iterator
	operation string
}

func (i scheduleFaultIterator) Error() error {
	if i.operation == "iterator-error" {
		return errors.New("injected schedule iterator error")
	}
	return i.Iterator.Error()
}

func (i scheduleFaultIterator) Close() error {
	err := i.Iterator.Close()
	if i.operation == "iterator-close" {
		return errors.Join(err, errors.New("injected schedule iterator close error"))
	}
	return err
}

func (s scheduleFaultStore) Get(key []byte) ([]byte, error) {
	if s.fault.operation == "get" && bytes.HasPrefix(key, s.fault.prefix) {
		return nil, errors.New("injected schedule read failure")
	}
	return s.KVStore.Get(key)
}
func (s scheduleFaultStore) Set(key, value []byte) error {
	if s.fault.operation == "set" && bytes.HasPrefix(key, s.fault.prefix) {
		if s.fault.afterWrite {
			if err := s.KVStore.Set(key, value); err != nil {
				return err
			}
		}
		return errors.New("injected schedule write failure")
	}
	return s.KVStore.Set(key, value)
}

func TestKnowledgeScheduleCreationIsAtomicAtEveryWrite(t *testing.T) {
	for _, prefix := range [][]byte{types.VestingScheduleKeyPrefix, types.ClaimRecordKeyPrefix, types.VestingByRecipientPrefix, types.ActiveVestingPrefix} {
		for _, afterWrite := range []bool{false, true} {
			t.Run(string([]byte{'0' + prefix[0]})+map[bool]string{true: "-after", false: "-before"}[afterWrite], func(t *testing.T) {
				k, ctx := setupKnowledgeScheduleKeeper(t)
				before := vestingSnapshot(t, k, ctx)
				fault := &scheduleFault{operation: "set", prefix: prefix, afterWrite: afterWrite}
				k.storeService = scheduleFaultService{k.storeService, fault}
				eventCount := len(ctx.EventManager().Events())
				require.ErrorContains(t, k.CreateVestingScheduleFromKnowledge(ctx, "claim", "fact", scheduleRecipient(), "200000", "empirical"), "injected schedule write failure")
				require.Equal(t, before, vestingSnapshot(t, k, ctx))
				require.Len(t, ctx.EventManager().Events(), eventCount)
				fault.operation = ""
				seedKnowledgeSchedule(t, k, ctx.WithBlockHeight(1001))
				require.Len(t, k.GetAllVestingSchedules(ctx), 1)
				require.NoError(t, k.ValidateKnowledgeVestingState(ctx))
			})
		}
	}
}

func TestKnowledgeScheduleReadFailuresNeverBecomeAbsence(t *testing.T) {
	for _, prefix := range [][]byte{types.ClaimRecordKeyPrefix, types.VestingScheduleKeyPrefix, types.VestingByRecipientPrefix, types.ActiveVestingPrefix, types.CategoryConfigKeyPrefix} {
		t.Run(string([]byte{'0' + prefix[0]}), func(t *testing.T) {
			k, ctx := setupKnowledgeScheduleKeeper(t)
			if prefix[0] != types.CategoryConfigKeyPrefix[0] {
				seedKnowledgeSchedule(t, k, ctx)
			}
			before := vestingSnapshot(t, k, ctx)
			k.storeService = scheduleFaultService{k.storeService, &scheduleFault{operation: "get", prefix: prefix}}
			require.ErrorContains(t, k.CreateVestingScheduleFromKnowledge(ctx.WithBlockHeight(2000), "claim", "fact", scheduleRecipient(), "200000", "empirical"), "injected schedule read failure")
			require.Equal(t, before, vestingSnapshot(t, k, ctx))
		})
	}
}

func TestKnowledgeScheduleRejectsBrokenIndexesWithoutRepair(t *testing.T) {
	for _, damage := range []string{"dangling-claim", "malformed-primary", "wrong-primary-id", "missing-recipient", "missing-active", "wrong-active-value", "candidate-collision"} {
		t.Run(damage, func(t *testing.T) {
			k, ctx := setupKnowledgeScheduleKeeper(t)
			schedule := seedKnowledgeSchedule(t, k, ctx)
			store := k.storeService.OpenKVStore(ctx)
			switch damage {
			case "dangling-claim":
				require.NoError(t, store.Set(vestingStateKey(types.ClaimRecordKeyPrefix, "claim"), []byte("missing")))
			case "malformed-primary":
				require.NoError(t, store.Set(vestingStateKey(types.VestingScheduleKeyPrefix, schedule.Id), []byte{0xff}))
			case "wrong-primary-id":
				id := schedule.Id
				schedule.Id = "wrong"
				bz, err := proto.Marshal(schedule)
				require.NoError(t, err)
				require.NoError(t, store.Set(vestingStateKey(types.VestingScheduleKeyPrefix, id), bz))
			case "missing-recipient":
				require.NoError(t, store.Delete(vestingStateKey(types.VestingByRecipientPrefix, schedule.Recipient+"/"+schedule.Id)))
			case "missing-active":
				require.NoError(t, store.Delete(vestingStateKey(types.ActiveVestingPrefix, schedule.Id)))
			case "wrong-active-value":
				require.NoError(t, store.Set(vestingStateKey(types.ActiveVestingPrefix, schedule.Id), []byte{2}))
			case "candidate-collision":
				require.NoError(t, store.Delete(vestingStateKey(types.ClaimRecordKeyPrefix, "claim")))
			}
			before := vestingSnapshot(t, k, ctx)
			require.Error(t, k.CreateVestingScheduleFromKnowledge(ctx, "claim", "fact", scheduleRecipient(), "200000", "empirical"))
			require.Equal(t, before, vestingSnapshot(t, k, ctx))
			require.Error(t, k.ValidateKnowledgeVestingState(ctx))
		})
	}
}

func TestKnowledgeScheduleCategoryAbsenceAndCorruption(t *testing.T) {
	for _, damage := range []string{"absent", "malformed", "wrong-category", "invalid-rate", "zero-half-life", "cliff-overflow"} {
		t.Run(damage, func(t *testing.T) {
			k, ctx := setupKnowledgeScheduleKeeper(t)
			key := vestingStateKey(types.CategoryConfigKeyPrefix, string(types.CategoryPeerReviewed))
			store := k.storeService.OpenKVStore(ctx)
			if damage == "absent" {
				require.NoError(t, store.Delete(key))
			} else if damage == "malformed" {
				require.NoError(t, store.Set(key, []byte{0xff}))
			} else {
				cfg, _ := k.GetCategoryConfig(ctx, types.CategoryPeerReviewed)
				switch damage {
				case "wrong-category":
					cfg.Category = "wrong"
				case "invalid-rate":
					cfg.MaxRelease = 1_000_001
				case "zero-half-life":
					cfg.HalfLifeBlocks = 0
				case "cliff-overflow":
					cfg.CliffBlocks = ^uint64(0)
				}
				bz, err := proto.Marshal(cfg)
				require.NoError(t, err)
				require.NoError(t, store.Set(key, bz))
			}
			before := vestingSnapshot(t, k, ctx)
			err := k.CreateVestingScheduleFromKnowledge(ctx, "claim", "fact", scheduleRecipient(), "200000", "empirical")
			if damage == "absent" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Equal(t, before, vestingSnapshot(t, k, ctx))
			}
		})
	}
}

func TestKnowledgeVestingUpgradePreservesDuplicateLegacyClaimsAndExport(t *testing.T) {
	k, ctx := setupKnowledgeScheduleKeeper(t)
	first := seedKnowledgeSchedule(t, k, ctx)
	second := proto.Clone(first).(*types.VestingSchedule)
	second.Id = "second-legacy-schedule"
	second.Status = string(types.VestingStatusCompleted)
	second.ReleasedAmount = "170000"
	k.SetVestingSchedule(ctx, second)
	k.SetClaimScheduleIndex(ctx, "claim", first.Id)
	before := vestingSnapshot(t, k, ctx)
	require.NoError(t, k.ValidateKnowledgeVestingState(ctx))
	require.Equal(t, before, vestingSnapshot(t, k, ctx))
	gs := k.ExportGenesis(ctx)
	k2, ctx2 := setupStrictMigrationKeeper(t)
	k2.InitGenesis(ctx2, gs)
	require.NoError(t, k2.ValidateKnowledgeVestingState(ctx2))
	require.NoError(t, k2.CreateVestingScheduleFromKnowledge(ctx2.WithBlockHeight(3000), "claim", "fact", scheduleRecipient(), "200000", "empirical"))
	require.Len(t, k2.GetAllVestingSchedules(ctx2), 2)
	selected, found := k2.GetVestingByClaimId(ctx2, "claim")
	require.True(t, found)
	require.Equal(t, first.Id, selected.Id)
	terminal, found := k2.GetVestingSchedule(ctx2, second.Id)
	require.True(t, found)
	require.True(t, proto.Equal(second, terminal))
}

func TestKnowledgeVestingUpgradeRefusesOrphanIndexesAndScanBounds(t *testing.T) {
	for _, prefix := range [][]byte{types.ClaimRecordKeyPrefix, types.VestingByRecipientPrefix, types.ActiveVestingPrefix} {
		k, ctx := setupKnowledgeScheduleKeeper(t)
		seedKnowledgeSchedule(t, k, ctx)
		require.NoError(t, k.storeService.OpenKVStore(ctx).Set(vestingStateKey(prefix, "orphan"), []byte{1}))
		require.Error(t, k.ValidateKnowledgeVestingState(ctx))
	}
	k, ctx := setupKnowledgeScheduleKeeper(t)
	seedKnowledgeSchedule(t, k, ctx)
	before := vestingSnapshot(t, k, ctx)
	require.ErrorContains(t, k.validateKnowledgeVestingState(ctx, 1, 64<<20), "scan bounds")
	require.ErrorContains(t, k.validateKnowledgeVestingState(ctx, 100_000, 1), "scan bounds")
	require.Equal(t, before, vestingSnapshot(t, k, ctx))
}

func TestKnowledgeVestingUpgradeNestedCachesAndIteratorErrors(t *testing.T) {
	k, ctx := setupKnowledgeScheduleKeeper(t)
	cache, _ := ctx.CacheContext()
	nested, _ := cache.CacheContext()
	seedKnowledgeSchedule(t, k, nested)
	require.NoError(t, k.ValidateKnowledgeVestingState(nested))
	for _, operation := range []string{"iterator-error", "iterator-close"} {
		t.Run(operation, func(t *testing.T) {
			broken := k
			broken.storeService = scheduleFaultService{k.storeService, &scheduleFault{operation: operation}}
			require.ErrorContains(t, broken.ValidateKnowledgeVestingState(nested), "injected schedule iterator")
		})
	}
}

// Every bank method fails the test if construction or an idempotent retry
// starts reading/minting/transferring money. Actual handoff bank-state checks
// belong to the cross-module integration fixture as well.
type scheduleBankGuard struct{ types.BankKeeper }

func (scheduleBankGuard) MintCoins(context.Context, string, sdk.Coins) error {
	panic("schedule construction called bank mint")
}
func (scheduleBankGuard) SendCoinsFromModuleToAccount(context.Context, string, sdk.AccAddress, sdk.Coins) error {
	panic("schedule construction called bank transfer")
}
func (scheduleBankGuard) SendCoinsFromModuleToModule(context.Context, string, string, sdk.Coins) error {
	panic("schedule construction called bank transfer")
}
func (scheduleBankGuard) GetAllBalances(context.Context, sdk.AccAddress) sdk.Coins {
	panic("schedule construction called bank balance")
}
func (scheduleBankGuard) GetSupply(context.Context, string) sdk.Coin {
	panic("schedule construction called bank supply")
}

func TestKnowledgeScheduleConstructionHasNoBankEffect(t *testing.T) {
	k, ctx := setupKnowledgeScheduleKeeper(t)
	k.bankKeeper = scheduleBankGuard{}
	seedKnowledgeSchedule(t, k, ctx)
	require.NoError(t, k.CreateVestingScheduleFromKnowledge(ctx.WithBlockHeight(2000), "claim", "fact", scheduleRecipient(), "200000", "empirical"))
}
