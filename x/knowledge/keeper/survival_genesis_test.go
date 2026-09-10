package keeper_test

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"testing"

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
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
	vestingkeeper "github.com/zerone-chain/zerone/x/vesting_rewards/keeper"
	vestingtypes "github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

// Both real keepers share one cacheable multistore. The bank spy is deliberately
// external to those stores: these tests require no banking calls at all.
type survivalGenesisBank struct{ *trackingBankKeeper }

func (b survivalGenesisBank) GetAllBalances(_ context.Context, addr sdk.AccAddress) sdk.Coins {
	return b.balances[addr.String()]
}

func (b survivalGenesisBank) GetSupply(_ context.Context, denom string) sdk.Coin {
	return sdk.NewInt64Coin(denom, 12345)
}

type survivalGenesisFixture struct {
	k    keeper.Keeper
	v    vestingkeeper.Keeper
	ctx  sdk.Context
	key  *storetypes.KVStoreKey
	vkey *storetypes.KVStoreKey
	bank survivalGenesisBank
}

func newSurvivalGenesisFixture(t *testing.T) survivalGenesisFixture {
	t.Helper()
	db := dbm.NewMemDB()
	cms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	key := storetypes.NewKVStoreKey(types.StoreKey)
	vkey := storetypes.NewKVStoreKey(vestingtypes.StoreKey)
	cms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	cms.MountStoreWithDB(vkey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, cms.LoadLatestVersion())
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	bank := survivalGenesisBank{newTrackingBankKeeper()}
	bank.moduleBalances["test_existing_fund"] = sdk.NewCoins(sdk.NewInt64Coin("uzrn", 12345))
	k := keeper.NewKeeper(runtime.NewKVStoreService(key), cdc, "authority", bank, newTrackingStakingKeeper())
	v := vestingkeeper.NewKeeper(cdc, runtime.NewKVStoreService(vkey), bank, nil, "authority")
	k.SetVestingRewardsKeeper(vestingkeeper.NewVestingRewardsKeeperAdapter(v))
	return survivalGenesisFixture{k, v, sdk.NewContext(cms, cmtproto.Header{Height: 100}, false, log.NewNopLogger()), key, vkey, bank}
}

func pendingGenesisExample() *types.SurvivalPendingReward {
	return &types.SurvivalPendingReward{
		ClaimId: "pending-claim", FactId: "pending-fact", Recipient: "opaque-historical-recipient",
		Amount: "200000", Category: "empirical", PartnershipId: "historical-partnership", Deadline: 150,
	}
}

func pendingKeeperValue(reward *types.SurvivalPendingReward) keeper.SurvivalPendingReward {
	return keeper.SurvivalPendingReward{
		ClaimId: reward.ClaimId, FactId: reward.FactId, Recipient: reward.Recipient,
		Amount: reward.Amount, Category: reward.Category, PartnershipId: reward.PartnershipId, Deadline: reward.Deadline,
	}
}

func pendingDeadlineKey(reward *types.SurvivalPendingReward) []byte {
	key := append([]byte{}, types.SurvivalDeadlineIndexPrefix...)
	key = binary.BigEndian.AppendUint64(key, reward.Deadline)
	return append(key, reward.FactId...)
}

func requireSurvivalBankUnchanged(t *testing.T, fixture survivalGenesisFixture) {
	t.Helper()
	require.Empty(t, fixture.bank.minted)
	require.Empty(t, fixture.bank.sendCalls)
	require.Empty(t, fixture.bank.balances)
	require.Equal(t, map[string]sdk.Coins{"test_existing_fund": sdk.NewCoins(sdk.NewInt64Coin("uzrn", 12345))}, fixture.bank.moduleBalances)
}

func TestSurvivalGenesisRoundTripPreservesPendingAndScheduleProgress(t *testing.T) {
	for _, status := range []string{string(vestingtypes.VestingStatusPaused), string(vestingtypes.VestingStatusCompleted)} {
		t.Run(status, func(t *testing.T) {
			source := newSurvivalGenesisFixture(t)
			require.NoError(t, source.k.InitGenesis(source.ctx, types.DefaultGenesis()))
			source.v.InitGenesis(source.ctx, vestingtypes.DefaultGenesis())
			reward := pendingGenesisExample()
			require.NoError(t, source.k.SetSurvivalPendingReward(source.ctx, pendingKeeperValue(reward)))
			require.NoError(t, source.k.SetFact(source.ctx, &types.Fact{Id: reward.FactId, Status: types.FactStatus_FACT_STATUS_ACTIVE}))
			schedule, err := source.v.CreateVestingSchedule(source.ctx, reward.ClaimId, reward.FactId, reward.Recipient, reward.Amount, vestingtypes.CategoryPeerReviewed, vestingtypes.SourceVerification)
			require.NoError(t, err)
			schedule.ReleasedAmount, schedule.ClaimableAmount = "1000", "2000"
			schedule.Status, schedule.LastClaimBlock = status, 112
			schedule.TotalPausedBlocks, schedule.PausedAtBlock = 9, 130
			schedule.DefenseCount, schedule.ReplicationCount, schedule.UpdatedAt = 2, 3, 140
			source.v.SetVestingSchedule(source.ctx, schedule)
			wantSchedule, err := proto.Marshal(schedule)
			require.NoError(t, err)

			// Exercise the actual generated JSON and binary genesis codecs. This
			// is the pending/schedule slice, not a full knowledge-state export claim.
			knowledgeJSON, err := protojson.Marshal(source.k.ExportGenesis(source.ctx))
			require.NoError(t, err)
			knowledgeGenesis := new(types.GenesisState)
			require.NoError(t, protojson.Unmarshal(knowledgeJSON, knowledgeGenesis))
			vestingBytes, err := proto.Marshal(source.v.ExportGenesis(source.ctx))
			require.NoError(t, err)
			vestingGenesis := new(vestingtypes.GenesisState)
			require.NoError(t, proto.Unmarshal(vestingBytes, vestingGenesis))
			target := newSurvivalGenesisFixture(t)
			require.NoError(t, target.k.InitGenesis(target.ctx, knowledgeGenesis))
			target.v.InitGenesis(target.ctx, vestingGenesis)
			pending, found := target.k.GetSurvivalPendingReward(target.ctx, reward.FactId)
			require.True(t, found)
			require.Equal(t, pendingKeeperValue(reward), pending)
			require.Equal(t, []byte{1}, target.ctx.KVStore(target.key).Get(pendingDeadlineKey(reward)))
			require.NoError(t, target.v.ValidateKnowledgeVestingState(target.ctx))
			primaryKey := append(append([]byte{}, vestingtypes.VestingScheduleKeyPrefix...), schedule.Id...)
			require.Equal(t, wantSchedule, target.ctx.KVStore(target.vkey).Get(primaryKey))

			// An imported pending record can acknowledge a matching schedule at
			// a later height without resetting progress or creating another one.
			target.ctx = target.ctx.WithBlockHeight(200).WithEventManager(sdk.NewEventManager())
			target.k.SweepSurvivedRewards(target.ctx)
			_, found = target.k.GetSurvivalPendingReward(target.ctx, reward.FactId)
			require.False(t, found)
			require.Nil(t, target.ctx.KVStore(target.key).Get(pendingDeadlineKey(reward)))
			require.Len(t, target.v.GetAllVestingSchedules(target.ctx), 1)
			require.Equal(t, wantSchedule, target.ctx.KVStore(target.vkey).Get(primaryKey))
			for _, event := range target.ctx.EventManager().Events() {
				require.NotEqual(t, "zerone.vesting_rewards.schedule_created", event.Type)
			}
			target.k.SweepSurvivedRewards(target.ctx.WithBlockHeight(201))
			require.Len(t, target.v.GetAllVestingSchedules(target.ctx), 1)
			requireSurvivalBankUnchanged(t, source)
			requireSurvivalBankUnchanged(t, target)
		})
	}
}

func TestSurvivalGenesisDeadlineZeroAndLegacyCategoryRemainRetryable(t *testing.T) {
	f := newSurvivalGenesisFixture(t)
	gs := types.DefaultGenesis()
	reward := pendingGenesisExample()
	reward.Deadline, reward.Category = 0, "legacy-unknown-category"
	gs.SurvivalPendingRewards = []*types.SurvivalPendingReward{reward}
	gs.Facts = []*types.Fact{{Id: reward.FactId, Status: types.FactStatus_FACT_STATUS_VERIFIED}}
	require.NoError(t, gs.Validate())
	require.NoError(t, f.k.InitGenesis(f.ctx, gs))
	f.v.InitGenesis(f.ctx, vestingtypes.DefaultGenesis())
	require.Equal(t, []byte{1}, f.ctx.KVStore(f.key).Get(pendingDeadlineKey(reward)))
	f.k.SweepSurvivedRewards(f.ctx)
	schedule, found := f.v.GetVestingByClaimId(f.ctx, reward.ClaimId)
	require.True(t, found)
	require.Equal(t, string(vestingtypes.CategoryPeerReviewed), schedule.Category)
	require.Equal(t, reward.Amount, schedule.TotalAmount)
	require.Equal(t, "0", schedule.ReleasedAmount)
	f.k.SweepSurvivedRewards(f.ctx.WithBlockHeight(101))
	require.Len(t, f.v.GetAllVestingSchedules(f.ctx), 1)
	requireSurvivalBankUnchanged(t, f)
}

func TestSurvivalGenesisRejectsAmbiguousPendingBeforeWrites(t *testing.T) {
	cases := map[string]func(*types.SurvivalPendingReward) []*types.SurvivalPendingReward{
		"nil": func(r *types.SurvivalPendingReward) []*types.SurvivalPendingReward {
			return []*types.SurvivalPendingReward{r, nil}
		},
		"duplicate": func(r *types.SurvivalPendingReward) []*types.SurvivalPendingReward {
			return []*types.SurvivalPendingReward{r, proto.Clone(r).(*types.SurvivalPendingReward)}
		},
	}
	for _, field := range []string{"claim", "fact", "recipient", "amount"} {
		cases["empty-"+field] = func(r *types.SurvivalPendingReward) []*types.SurvivalPendingReward {
			switch field {
			case "claim":
				r.ClaimId = ""
			case "fact":
				r.FactId = ""
			case "recipient":
				r.Recipient = ""
			case "amount":
				r.Amount = ""
			}
			return []*types.SurvivalPendingReward{r}
		}
	}
	for _, amount := range []string{"0", "000", "-1", "1.1", " 1", "1e6"} {
		cases["amount-"+amount] = func(r *types.SurvivalPendingReward) []*types.SurvivalPendingReward {
			r.Amount = amount
			return []*types.SurvivalPendingReward{r}
		}
	}
	for name, values := range cases {
		t.Run(name, func(t *testing.T) {
			f := newSurvivalGenesisFixture(t)
			gs := types.DefaultGenesis()
			gs.SurvivalPendingRewards = values(pendingGenesisExample())
			require.Error(t, gs.Validate())
			require.Error(t, f.k.InitGenesis(f.ctx, gs))
			iter := f.ctx.KVStore(f.key).Iterator(nil, nil)
			defer iter.Close()
			require.False(t, iter.Valid(), "invalid pending list must be rejected before any module writes")
			require.Empty(t, f.ctx.EventManager().Events())
			requireSurvivalBankUnchanged(t, f)
		})
	}
}

func TestSurvivalGenesisExportRefusesUnreadablePending(t *testing.T) {
	valid, err := json.Marshal(pendingKeeperValue(pendingGenesisExample()))
	require.NoError(t, err)
	for name, value := range map[string][]byte{
		"malformed":     []byte("{"),
		"key-mismatch":  valid,
		"unknown-field": append(append([]byte{}, valid[:len(valid)-1]...), []byte(",\"unsupported_obligation\":true}")...),
	} {
		t.Run(name, func(t *testing.T) {
			f := newSurvivalGenesisFixture(t)
			require.NoError(t, f.k.InitGenesis(f.ctx, types.DefaultGenesis()))
			factID := "pending-fact"
			if name == "key-mismatch" {
				factID = "wrong-fact"
			}
			key := append(append([]byte{}, types.SurvivalPendingRewardPrefix...), factID...)
			f.ctx.KVStore(f.key).Set(key, value)
			require.Panics(t, func() { f.k.ExportGenesis(f.ctx) })
			require.Equal(t, value, f.ctx.KVStore(f.key).Get(key))
			requireSurvivalBankUnchanged(t, f)
		})
	}
}

func TestSurvivalGenesisAbsentHistoricalFieldDoesNotInventPending(t *testing.T) {
	f := newSurvivalGenesisFixture(t)
	gs := types.DefaultGenesis()
	gs.Facts = []*types.Fact{{Id: "historical-accepted", Status: types.FactStatus_FACT_STATUS_VERIFIED}}
	require.NoError(t, f.k.InitGenesis(f.ctx, gs))
	require.Empty(t, f.k.ExportGenesis(f.ctx).SurvivalPendingRewards)
	requireSurvivalBankUnchanged(t, f)
}

func TestSurvivalGenesisPreservesAcceptedAmountSpellings(t *testing.T) {
	for _, amount := range []string{"+200000", "000200000"} {
		t.Run(amount, func(t *testing.T) {
			source := newSurvivalGenesisFixture(t)
			require.NoError(t, source.k.InitGenesis(source.ctx, types.DefaultGenesis()))
			source.v.InitGenesis(source.ctx, vestingtypes.DefaultGenesis())
			fact := &types.Fact{Id: "amount-fact", Status: types.FactStatus_FACT_STATUS_VERIFIED}
			claim := &types.Claim{Id: "amount-claim", Submitter: "historical-recipient", Stake: amount, Category: "empirical"}
			source.k.EscrowSubmitterReward(source.ctx, fact, claim)
			pending, found := source.k.GetSurvivalPendingReward(source.ctx, fact.Id)
			require.True(t, found, "accepted claim amount must create its pending reward")
			require.Equal(t, amount, pending.Amount)
			encoded, err := proto.Marshal(source.k.ExportGenesis(source.ctx))
			require.NoError(t, err)
			genesis := new(types.GenesisState)
			require.NoError(t, proto.Unmarshal(encoded, genesis))
			require.NoError(t, genesis.Validate())
			target := newSurvivalGenesisFixture(t)
			require.NoError(t, target.k.InitGenesis(target.ctx, genesis))
			target.v.InitGenesis(target.ctx, vestingtypes.DefaultGenesis())
			after, found := target.k.GetSurvivalPendingReward(target.ctx, fact.Id)
			require.True(t, found)
			require.Equal(t, pending, after, "import must preserve the original amount bytes")
			target.k.SweepSurvivedRewards(target.ctx.WithBlockHeight(int64(pending.Deadline)))
			schedule, found := target.v.GetVestingByClaimId(target.ctx, claim.Id)
			require.True(t, found)
			require.Equal(t, amount, schedule.TotalAmount)
			requireSurvivalBankUnchanged(t, source)
			requireSurvivalBankUnchanged(t, target)
		})
	}
}
