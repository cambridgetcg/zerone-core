package keeper

import (
	"bytes"
	"context"
	"errors"
	"testing"

	corestore "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/types"
	vestingkeeper "github.com/zerone-chain/zerone/x/vesting_rewards/keeper"
	vestingtypes "github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

type handoffFault struct {
	op     string
	prefix []byte
	after  bool
}

type handoffFaultService struct {
	delegate corestore.KVStoreService
	fault    *handoffFault
}

func (s handoffFaultService) OpenKVStore(ctx context.Context) corestore.KVStore {
	return handoffFaultStore{s.delegate.OpenKVStore(ctx), s.fault}
}

type handoffFaultStore struct {
	corestore.KVStore
	fault *handoffFault
}

func (s handoffFaultStore) fails(op string, key []byte) bool {
	return s.fault.op == op && bytes.HasPrefix(key, s.fault.prefix)
}

func (s handoffFaultStore) Get(key []byte) ([]byte, error) {
	if s.fails("get", key) {
		return nil, errors.New("injected read failure")
	}
	return s.KVStore.Get(key)
}

func (s handoffFaultStore) Set(key, value []byte) error {
	if s.fails("set", key) {
		if s.fault.after {
			if err := s.KVStore.Set(key, value); err != nil {
				return err
			}
		}
		return errors.New("injected set failure")
	}
	return s.KVStore.Set(key, value)
}

func (s handoffFaultStore) Delete(key []byte) error {
	if s.fails("delete", key) {
		if s.fault.after {
			if err := s.KVStore.Delete(key); err != nil {
				return err
			}
		}
		return errors.New("injected delete failure")
	}
	return s.KVStore.Delete(key)
}

type handoffFixture struct {
	k       Keeper
	v       vestingkeeper.Keeper
	ctx     sdk.Context
	keys    []*storetypes.KVStoreKey
	kFault  *handoffFault
	vFault  *handoffFault
	pending SurvivalPendingReward
}

func setupHandoff(t *testing.T) handoffFixture {
	t.Helper()
	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	keys := []*storetypes.KVStoreKey{storetypes.NewKVStoreKey(types.StoreKey), storetypes.NewKVStoreKey(vestingtypes.StoreKey)}
	for _, key := range keys {
		ms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	}
	require.NoError(t, ms.LoadLatestVersion())
	ctx := sdk.NewContext(ms, cmtproto.Header{Height: 100}, false, log.NewNopLogger())
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	kFault, vFault := new(handoffFault), new(handoffFault)
	// Real module stores and adapter, with no bank implementation: a bank call
	// in a schedule-only handoff would panic and fail these tests.
	k := NewKeeper(handoffFaultService{runtime.NewKVStoreService(keys[0]), kFault}, cdc, "authority", nil, nil)
	v := vestingkeeper.NewKeeper(cdc, handoffFaultService{runtime.NewKVStoreService(keys[1]), vFault}, nil, nil, "authority")
	k.SetVestingRewardsKeeper(vestingkeeper.NewVestingRewardsKeeperAdapter(v))
	pr := SurvivalPendingReward{ClaimId: "claim", FactId: "fact", Recipient: "recipient", Amount: "200000", Category: "empirical", Deadline: 100}
	require.NoError(t, k.SetFact(ctx, &types.Fact{Id: pr.FactId, ClaimId: pr.ClaimId, Submitter: pr.Recipient, Status: types.FactStatus_FACT_STATUS_VERIFIED}))
	require.NoError(t, k.SetSurvivalPendingReward(ctx, pr))
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	return handoffFixture{k, v, ctx, keys, kFault, vFault, pr}
}

func (f handoffFixture) snapshot(t *testing.T) map[string][]byte {
	t.Helper()
	result := make(map[string][]byte)
	for _, key := range f.keys {
		iter := f.ctx.KVStore(key).Iterator(nil, nil)
		for ; iter.Valid(); iter.Next() {
			result[key.Name()+":"+string(iter.Key())] = bytes.Clone(iter.Value())
		}
		require.NoError(t, closeSurvivalIterator(iter))
	}
	return result
}

func TestSurvivalHandoffFaultsPreserveBothModulesAndRetry(t *testing.T) {
	cases := []struct {
		name, module, op string
		prefix           []byte
	}{
		{"pending-read", "knowledge", "get", types.SurvivalPendingRewardPrefix},
		{"fact-read", "knowledge", "get", types.FactKeyPrefix},
		{"claim-index-read", "vesting", "get", vestingtypes.ClaimRecordKeyPrefix},
		{"schedule-write", "vesting", "set", vestingtypes.VestingScheduleKeyPrefix},
		{"claim-index-write", "vesting", "set", vestingtypes.ClaimRecordKeyPrefix},
		{"recipient-index-write", "vesting", "set", vestingtypes.VestingByRecipientPrefix},
		{"active-index-write", "vesting", "set", vestingtypes.ActiveVestingPrefix},
		{"pending-delete", "knowledge", "delete", types.SurvivalPendingRewardPrefix},
		{"deadline-delete", "knowledge", "delete", types.SurvivalDeadlineIndexPrefix},
	}
	for _, tc := range cases {
		for _, after := range []bool{false, true} {
			if tc.op == "get" && after {
				continue
			}
			name := tc.name + "/before"
			if after {
				name = tc.name + "/after-staging"
			}
			t.Run(name, func(t *testing.T) {
				f := setupHandoff(t)
				before := f.snapshot(t)
				fault := f.kFault
				if tc.module == "vesting" {
					fault = f.vFault
				}
				*fault = handoffFault{tc.op, tc.prefix, after}
				f.k.SweepSurvivedRewards(f.ctx)
				require.Equal(t, before, f.snapshot(t))
				require.Empty(t, f.ctx.EventManager().Events())
				*fault = handoffFault{}
				f.k.SweepSurvivedRewards(f.ctx.WithBlockHeight(101))
				_, pending := f.k.GetSurvivalPendingReward(f.ctx, "fact")
				require.False(t, pending)
				schedule, found := f.v.GetVestingByClaimId(f.ctx, "claim")
				require.True(t, found)
				require.Equal(t, "200000", schedule.TotalAmount)
				require.Equal(t, "0", schedule.ReleasedAmount)
				require.Len(t, f.ctx.EventManager().Events(), 2)
				afterSuccess := f.snapshot(t)
				f.k.SweepSurvivedRewards(f.ctx.WithBlockHeight(102))
				require.Equal(t, afterSuccess, f.snapshot(t))
				require.Len(t, f.ctx.EventManager().Events(), 2)
			})
		}
	}
}

func TestSurvivalHandoffMissingDependencyAndConflictRetainPending(t *testing.T) {
	f := setupHandoff(t)
	before := f.snapshot(t)
	f.k.SetVestingRewardsKeeper(nil)
	require.Error(t, f.k.releaseSurvivalReward(f.ctx, "fact"))
	require.Equal(t, before, f.snapshot(t))
	require.Empty(t, f.ctx.EventManager().Events())
	f.k.SetVestingRewardsKeeper(vestingkeeper.NewVestingRewardsKeeperAdapter(f.v))
	require.NoError(t, f.v.CreateVestingScheduleFromKnowledge(f.ctx, "claim", "different-fact", "recipient", "200000", "empirical"))
	f.ctx = f.ctx.WithEventManager(sdk.NewEventManager())
	before = f.snapshot(t)
	require.Error(t, f.k.releaseSurvivalReward(f.ctx, "fact"))
	require.Equal(t, before, f.snapshot(t))
	require.Empty(t, f.ctx.EventManager().Events())
}

func TestSurvivalHandoffExistingScheduleProgressSurvivesDeleteFailure(t *testing.T) {
	f := setupHandoff(t)
	require.NoError(t, f.v.CreateVestingScheduleFromKnowledge(f.ctx, "claim", "fact", "recipient", "200000", "empirical"))
	schedule, found := f.v.GetVestingByClaimId(f.ctx, "claim")
	require.True(t, found)
	schedule.ReleasedAmount, schedule.ClaimableAmount = "50000", "1000"
	schedule.LastClaimBlock = 101
	f.v.SetVestingSchedule(f.ctx, schedule)
	scheduleKey := append(bytes.Clone(vestingtypes.VestingScheduleKeyPrefix), []byte(schedule.Id)...)
	storedSchedule := bytes.Clone(f.ctx.KVStore(f.keys[1]).Get(scheduleKey))
	f.ctx = f.ctx.WithEventManager(sdk.NewEventManager())
	before := f.snapshot(t)
	*f.kFault = handoffFault{"delete", types.SurvivalDeadlineIndexPrefix, true}
	require.Error(t, f.k.releaseSurvivalReward(f.ctx.WithBlockHeight(200), "fact"))
	require.Equal(t, before, f.snapshot(t))
	require.Empty(t, f.ctx.EventManager().Events())
	*f.kFault = handoffFault{}
	require.NoError(t, f.k.releaseSurvivalReward(f.ctx.WithBlockHeight(201), "fact"))
	after, found := f.v.GetVestingByClaimId(f.ctx, "claim")
	require.True(t, found)
	require.True(t, proto.Equal(schedule, after))
	require.Equal(t, storedSchedule, f.ctx.KVStore(f.keys[1]).Get(scheduleKey))
	require.Len(t, f.ctx.EventManager().Events(), 1) // handoff only, no new schedule
}

func TestSurvivalChallengeKeepsReachableRetry(t *testing.T) {
	f := setupHandoff(t)
	fact, _ := f.k.GetFact(f.ctx, "fact")
	fact.Status = types.FactStatus_FACT_STATUS_CHALLENGED
	require.NoError(t, f.k.SetFact(f.ctx, fact))
	f.ctx = f.ctx.WithEventManager(sdk.NewEventManager())
	before := f.snapshot(t)
	f.k.SweepSurvivedRewards(f.ctx)
	require.Equal(t, before, f.snapshot(t))
	require.Empty(t, f.ctx.EventManager().Events())
	*f.vFault = handoffFault{"set", vestingtypes.VestingScheduleKeyPrefix, false}
	f.k.handleChallengeSurvival(f.ctx, &types.Claim{Id: "challenge", ProvisionalFactId: "fact", Submitter: "challenger"})
	fact, _ = f.k.GetFact(f.ctx, "fact")
	require.Equal(t, types.FactStatus_FACT_STATUS_ACTIVE, fact.Status)
	_, pending := f.k.GetSurvivalPendingReward(f.ctx, "fact")
	require.True(t, pending)
	*f.vFault = handoffFault{}
	f.ctx = f.ctx.WithEventManager(sdk.NewEventManager())
	f.k.SweepSurvivedRewards(f.ctx.WithBlockHeight(101))
	_, pending = f.k.GetSurvivalPendingReward(f.ctx, "fact")
	require.False(t, pending)
	require.Len(t, f.ctx.EventManager().Events(), 2)
}

func TestSurvivalIndexRepairAndDeadlineAreConservative(t *testing.T) {
	f := setupHandoff(t)
	store := f.ctx.KVStore(f.keys[0])
	pendingBytes := bytes.Clone(store.Get(survivalPendingKey("fact")))
	store.Delete(survivalDeadlineKey(100, "fact"))
	store.Set(survivalDeadlineKey(1, "orphan"), []byte{1})
	before := f.snapshot(t)
	*f.kFault = handoffFault{"set", types.SurvivalDeadlineIndexPrefix, true}
	require.Error(t, f.k.RebuildSurvivalDeadlineIndex(f.ctx))
	require.Equal(t, before, f.snapshot(t))
	*f.kFault = handoffFault{}
	require.NoError(t, f.k.RebuildSurvivalDeadlineIndex(f.ctx))
	require.Equal(t, pendingBytes, store.Get(survivalPendingKey("fact")))
	require.Equal(t, []byte{1}, store.Get(survivalDeadlineKey(100, "fact")))
	require.Nil(t, store.Get(survivalDeadlineKey(1, "orphan")))
	// A stale earlier index must never pay a future primary obligation early.
	store.Set(survivalDeadlineKey(1, "fact"), []byte{1})
	before = f.snapshot(t)
	f.k.SweepSurvivedRewards(f.ctx.WithBlockHeight(99))
	require.Equal(t, before, f.snapshot(t))
	conflict := f.pending
	conflict.Deadline = 101
	require.Error(t, f.k.SetSurvivalPendingReward(f.ctx, conflict))
	require.Equal(t, before, f.snapshot(t))
}

func TestSurvivalCorruptFactIsNotCancellation(t *testing.T) {
	for _, status := range []int32{-1, 0, 999} {
		t.Run(types.FactStatus(status).String(), func(t *testing.T) {
			f := setupHandoff(t)
			bz := []byte("not protobuf")
			if status >= 0 {
				var err error
				bz, err = proto.Marshal(&types.Fact{Id: "fact", Status: types.FactStatus(status)})
				require.NoError(t, err)
			}
			f.ctx.KVStore(f.keys[0]).Set(types.FactKey("fact"), bz)
			before := f.snapshot(t)
			f.k.SweepSurvivedRewards(f.ctx)
			require.Equal(t, before, f.snapshot(t))
			require.Empty(t, f.ctx.EventManager().Events())
		})
	}
}

func TestSurvivalAmbiguousPendingBytesArePreservedByRefusal(t *testing.T) {
	for _, suffix := range []string{
		`,"amount":"300000"}`, `,"Amount":"300000"}`, `,"unknown":1}`,
		`,"category":null}`, `,"category":123}`, ",\"category\":\"\xff\"}",
		`,"category":"\ud800"}`, `,"category":"\udfff"}`,
	} {
		t.Run(suffix, func(t *testing.T) {
			f := setupHandoff(t)
			bz := []byte(`{"claim_id":"claim","fact_id":"fact","recipient":"recipient","amount":"200000","deadline":100` + suffix)
			f.ctx.KVStore(f.keys[0]).Set(survivalPendingKey("fact"), bz)
			before := f.snapshot(t)
			require.Error(t, f.k.releaseSurvivalReward(f.ctx, "fact"))
			require.Error(t, f.k.RebuildSurvivalDeadlineIndex(f.ctx))
			_, err := f.k.GetAllSurvivalPendingRewards(f.ctx)
			require.Error(t, err)
			require.Equal(t, before, f.snapshot(t))
			require.Empty(t, f.ctx.EventManager().Events())
		})
	}
	valid := []byte(`{"claim_id":"claim","fact_id":"fact","recipient":"recipient","amount":"+200000","deadline":0,"category":"\ud83d\ude00"}`)
	pr, err := decodeSurvivalPendingReward(valid, "fact")
	require.NoError(t, err)
	require.Equal(t, "😀", pr.Category)
	require.Equal(t, "+200000", pr.Amount)
	_, err = decodeSurvivalPendingReward(bytes.ReplaceAll(valid, []byte(`,"deadline":0`), nil), "fact")
	require.Error(t, err)
}

func TestSurvivalPendingCreationAndCancellationAreAtomic(t *testing.T) {
	for _, prefix := range [][]byte{types.SurvivalPendingRewardPrefix, types.SurvivalDeadlineIndexPrefix} {
		f := setupHandoff(t)
		before := f.snapshot(t)
		*f.kFault = handoffFault{"set", prefix, true}
		newPending := f.pending
		newPending.FactId = "another-fact"
		require.Error(t, f.k.SetSurvivalPendingReward(f.ctx, newPending))
		require.Equal(t, before, f.snapshot(t))
		*f.kFault = handoffFault{"delete", prefix, true}
		f.k.cancelSurvivalReward(f.ctx, "fact")
		require.Equal(t, before, f.snapshot(t))
		require.Empty(t, f.ctx.EventManager().Events())
	}
}
