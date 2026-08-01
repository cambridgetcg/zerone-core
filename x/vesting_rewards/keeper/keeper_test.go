package keeper_test

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"google.golang.org/protobuf/proto"

	commontypes "github.com/zerone-chain/zerone/x/common/types"
	vestingrewards "github.com/zerone-chain/zerone/x/vesting_rewards"
	"github.com/zerone-chain/zerone/x/vesting_rewards/keeper"
	"github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("zrn", "zrnpub")
	config.Seal()
}

// ---------- Test Setup ----------

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		t.Fatalf("failed to load latest version: %v", err)
	}

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	k := keeper.NewKeeper(cdc, runtime.NewKVStoreService(storeKey), nil, nil, "authority")
	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 1000}, false, log.NewNopLogger())

	gs := types.DefaultGenesis()
	k.InitGenesis(ctx, gs)

	return k, ctx
}

func setupMsgServer(t *testing.T) (types.MsgServer, keeper.Keeper, sdk.Context) {
	t.Helper()
	k, ctx := setupKeeper(t)
	return keeper.NewMsgServerImpl(k), k, ctx
}

// ---------- CreateVesting Tests ----------

func TestCreateVesting_Success(t *testing.T) {
	ms, k, ctx := setupMsgServer(t)

	recipient := sdk.AccAddress("recipient1__________").String()

	resp, err := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  recipient,
		Amount:       "1000000000000000000",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-1",
	})
	if err != nil {
		t.Fatalf("create vesting failed: %v", err)
	}
	if resp.VestingId == "" {
		t.Fatal("expected non-empty vesting ID")
	}

	schedule, found := k.GetVestingSchedule(ctx, resp.VestingId)
	if !found {
		t.Fatal("schedule not found")
	}
	if schedule.Recipient != recipient {
		t.Errorf("expected recipient %s, got %s", recipient, schedule.Recipient)
	}
	if schedule.Status != string(types.VestingStatusActive) {
		t.Errorf("expected active status, got %s", schedule.Status)
	}
	if schedule.TotalAmount != "1000000000000000000" {
		t.Errorf("expected total amount 1000000000000000000, got %s", schedule.TotalAmount)
	}
}

func TestCreateVesting_Unauthorized(t *testing.T) {
	ms, _, ctx := setupMsgServer(t)

	_, err := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "not-authority",
		Beneficiary:  sdk.AccAddress("recipient1__________").String(),
		Amount:       "1000000000000000000",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-1",
	})
	if err == nil {
		t.Fatal("expected unauthorized error")
	}
}

func TestCreateVesting_InvalidCategory(t *testing.T) {
	ms, _, ctx := setupMsgServer(t)

	resp, err := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  sdk.AccAddress("recipient1__________").String(),
		Amount:       "1000000000000000000",
		Category:     types.VestingCategory_VESTING_CATEGORY_UNSPECIFIED,
		LinkedFactId: "fact-1",
	})
	if err != nil {
		t.Fatalf("expected unspecified category to fall back to default, got error: %v", err)
	}
	if resp.VestingId == "" {
		t.Fatal("expected non-empty vesting ID")
	}
}

// ---------- PauseVesting Tests ----------

func TestPauseVesting_Success(t *testing.T) {
	ms, k, ctx := setupMsgServer(t)

	resp, _ := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  sdk.AccAddress("recipient1__________").String(),
		Amount:       "1000000000000000000",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-1",
	})

	_, err := ms.PauseVesting(ctx, &types.MsgPauseVesting{
		Authority: "authority",
		VestingId: resp.VestingId,
		Reason:    "active challenge",
	})
	if err != nil {
		t.Fatalf("pause vesting failed: %v", err)
	}

	schedule, _ := k.GetVestingSchedule(ctx, resp.VestingId)
	if schedule.Status != string(types.VestingStatusPaused) {
		t.Errorf("expected paused status, got %s", schedule.Status)
	}
	if schedule.PausedAtBlock != 1000 {
		t.Errorf("expected paused at block 1000, got %d", schedule.PausedAtBlock)
	}
}

func TestPauseVesting_Unauthorized(t *testing.T) {
	ms, _, ctx := setupMsgServer(t)

	resp, _ := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  sdk.AccAddress("recipient1__________").String(),
		Amount:       "1000000000000000000",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-1",
	})

	_, err := ms.PauseVesting(ctx, &types.MsgPauseVesting{
		Authority: "not-authority",
		VestingId: resp.VestingId,
	})
	if err == nil {
		t.Fatal("expected unauthorized error")
	}
}

// ---------- ResumeVesting Tests ----------

func TestResumeVesting_Success(t *testing.T) {
	ms, k, ctx := setupMsgServer(t)

	resp, _ := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  sdk.AccAddress("recipient1__________").String(),
		Amount:       "1000000000000000000",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-1",
	})

	ms.PauseVesting(ctx, &types.MsgPauseVesting{
		Authority: "authority", VestingId: resp.VestingId, Reason: "challenge",
	})

	ctx = ctx.WithBlockHeight(1100)
	_, err := ms.ResumeVesting(ctx, &types.MsgResumeVesting{
		Authority: "authority", VestingId: resp.VestingId,
	})
	if err != nil {
		t.Fatalf("resume vesting failed: %v", err)
	}

	schedule, _ := k.GetVestingSchedule(ctx, resp.VestingId)
	if schedule.Status != string(types.VestingStatusActive) {
		t.Errorf("expected active status, got %s", schedule.Status)
	}
	if schedule.TotalPausedBlocks != 100 {
		t.Errorf("expected 100 paused blocks, got %d", schedule.TotalPausedBlocks)
	}
	if schedule.PausedAtBlock != 0 {
		t.Errorf("expected paused_at_block reset to 0, got %d", schedule.PausedAtBlock)
	}
}

func TestResumeVesting_NotPaused(t *testing.T) {
	ms, k, ctx := setupMsgServer(t)

	resp, _ := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  sdk.AccAddress("recipient1__________").String(),
		Amount:       "1000000000000000000",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-1",
	})

	_, err := ms.ResumeVesting(ctx, &types.MsgResumeVesting{
		Authority: "authority", VestingId: resp.VestingId,
	})
	if err != nil {
		t.Fatalf("resume vesting should not error for non-paused: %v", err)
	}

	schedule, _ := k.GetVestingSchedule(ctx, resp.VestingId)
	if schedule.Status != string(types.VestingStatusActive) {
		t.Errorf("expected active status unchanged, got %s", schedule.Status)
	}
}

// ---------- AccelerateVesting Tests ----------

func TestAccelerateVesting_Defense(t *testing.T) {
	ms, k, ctx := setupMsgServer(t)

	resp, _ := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  sdk.AccAddress("recipient1__________").String(),
		Amount:       "1000000000000000000",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-1",
	})

	_, err := ms.AccelerateVesting(ctx, &types.MsgAccelerateVesting{
		Authority:          "authority",
		VestingId:          resp.VestingId,
		AccelerationFactor: 1000000,
	})
	if err != nil {
		t.Fatalf("accelerate vesting failed: %v", err)
	}

	schedule, _ := k.GetVestingSchedule(ctx, resp.VestingId)
	if schedule.DefenseCount != 1 {
		t.Errorf("expected defense count 1, got %d", schedule.DefenseCount)
	}
}

func TestAccelerateVesting_Replication(t *testing.T) {
	ms, k, ctx := setupMsgServer(t)

	resp, _ := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  sdk.AccAddress("recipient1__________").String(),
		Amount:       "1000000000000000000",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-1",
	})

	_, err := ms.AccelerateVesting(ctx, &types.MsgAccelerateVesting{
		Authority:          "authority",
		VestingId:          resp.VestingId,
		AccelerationFactor: 600000,
	})
	if err != nil {
		t.Fatalf("accelerate vesting failed: %v", err)
	}

	schedule, _ := k.GetVestingSchedule(ctx, resp.VestingId)
	if schedule.ReplicationCount != 1 {
		t.Errorf("expected replication count 1, got %d", schedule.ReplicationCount)
	}
}

func TestAccelerateVesting_Unauthorized(t *testing.T) {
	ms, _, ctx := setupMsgServer(t)

	resp, _ := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  sdk.AccAddress("recipient1__________").String(),
		Amount:       "1000000000000000000",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-1",
	})

	_, err := ms.AccelerateVesting(ctx, &types.MsgAccelerateVesting{
		Authority:          "not-authority",
		VestingId:          resp.VestingId,
		AccelerationFactor: 1000000,
	})
	if err == nil {
		t.Fatal("expected unauthorized error")
	}
}

// ---------- CompleteVesting Tests ----------

func TestCompleteVesting_ByRecipient(t *testing.T) {
	ms, k, ctx := setupMsgServer(t)

	recipient := sdk.AccAddress("recipient1__________").String()
	resp, _ := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  recipient,
		Amount:       "1000000000000000000",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-1",
	})

	_, err := ms.CompleteVesting(ctx, &types.MsgCompleteVesting{
		Authority: recipient, VestingId: resp.VestingId,
	})
	if err != nil {
		t.Fatalf("complete vesting failed: %v", err)
	}

	schedule, _ := k.GetVestingSchedule(ctx, resp.VestingId)
	if schedule.Status != string(types.VestingStatusCompleted) {
		t.Errorf("expected completed status, got %s", schedule.Status)
	}
}

func TestCompleteVesting_ByAuthority(t *testing.T) {
	ms, k, ctx := setupMsgServer(t)

	resp, _ := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  sdk.AccAddress("recipient1__________").String(),
		Amount:       "1000000000000000000",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-1",
	})

	_, err := ms.CompleteVesting(ctx, &types.MsgCompleteVesting{
		Authority: "authority", VestingId: resp.VestingId,
	})
	if err != nil {
		t.Fatalf("complete vesting by authority failed: %v", err)
	}

	schedule, _ := k.GetVestingSchedule(ctx, resp.VestingId)
	if schedule.Status != string(types.VestingStatusCompleted) {
		t.Errorf("expected completed status, got %s", schedule.Status)
	}
}

func TestCompleteVesting_Unauthorized(t *testing.T) {
	ms, _, ctx := setupMsgServer(t)

	resp, _ := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  sdk.AccAddress("recipient1__________").String(),
		Amount:       "1000000000000000000",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-1",
	})

	_, err := ms.CompleteVesting(ctx, &types.MsgCompleteVesting{
		Authority: sdk.AccAddress("random______________").String(),
		VestingId: resp.VestingId,
	})
	if err == nil {
		t.Fatal("expected unauthorized error")
	}
}

func TestCompleteVesting_AlreadyCompleted(t *testing.T) {
	ms, _, ctx := setupMsgServer(t)

	recipient := sdk.AccAddress("recipient1__________").String()
	resp, _ := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  recipient,
		Amount:       "1000000000000000000",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-1",
	})

	ms.CompleteVesting(ctx, &types.MsgCompleteVesting{
		Authority: recipient, VestingId: resp.VestingId,
	})

	_, err := ms.CompleteVesting(ctx, &types.MsgCompleteVesting{
		Authority: recipient, VestingId: resp.VestingId,
	})
	if err == nil {
		t.Fatal("expected error for already completed")
	}
}

func TestCompleteVesting_FalsifiedSchedule(t *testing.T) {
	ms, k, ctx := setupMsgServer(t)

	recipient := sdk.AccAddress("recipient1__________").String()
	resp, _ := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  recipient,
		Amount:       "1000000000000000000",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-1",
	})

	schedule, _ := k.GetVestingSchedule(ctx, resp.VestingId)
	schedule.Status = string(types.VestingStatusFalsified)
	k.SetVestingSchedule(ctx, schedule)

	_, err := ms.CompleteVesting(ctx, &types.MsgCompleteVesting{
		Authority: recipient, VestingId: resp.VestingId,
	})
	if err == nil {
		t.Fatal("expected error for falsified schedule")
	}
}

// ---------- ValidateBasic Tests ----------

func TestMsgCreateVesting_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     *types.MsgCreateVesting
		wantErr bool
	}{
		{"valid", &types.MsgCreateVesting{
			Authority:   "authority",
			Beneficiary: sdk.AccAddress("recipient1__________").String(),
			Amount:      "1000",
		}, false},
		{"missing authority", &types.MsgCreateVesting{
			Beneficiary: sdk.AccAddress("recipient1__________").String(),
			Amount:      "1000",
		}, true},
		{"missing beneficiary", &types.MsgCreateVesting{
			Authority: "authority",
			Amount:    "1000",
		}, true},
		{"zero amount", &types.MsgCreateVesting{
			Authority:   "authority",
			Beneficiary: sdk.AccAddress("recipient1__________").String(),
			Amount:      "0",
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMsgAccelerateVesting_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     *types.MsgAccelerateVesting
		wantErr bool
	}{
		{"valid", &types.MsgAccelerateVesting{
			Authority: "authority", VestingId: "v1", AccelerationFactor: 1000000,
		}, false},
		{"missing vesting id", &types.MsgAccelerateVesting{
			Authority: "authority", AccelerationFactor: 1000000,
		}, true},
		{"missing authority", &types.MsgAccelerateVesting{
			VestingId: "v1", AccelerationFactor: 1000000,
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// ---------- Mock Bank/Staking Keepers ----------

type mockBankKeeper struct {
	mintedCoins   sdk.Coins
	sentToAccount map[string]sdk.Coins
	sentToModule  map[string]sdk.Coins
	balances      map[string]sdk.Coins
	supply        map[string]sdkmath.Int
	mintErr       error
	sendAccErr    error
	sendModErr    error
}

func newMockBankKeeper() *mockBankKeeper {
	return &mockBankKeeper{
		sentToAccount: make(map[string]sdk.Coins),
		sentToModule:  make(map[string]sdk.Coins),
		supply:        make(map[string]sdkmath.Int),
	}
}

func (m *mockBankKeeper) MintCoins(_ context.Context, _ string, amounts sdk.Coins) error {
	if m.mintErr != nil {
		return m.mintErr
	}
	m.mintedCoins = m.mintedCoins.Add(amounts...)
	for _, coin := range amounts {
		cur, ok := m.supply[coin.Denom]
		if !ok {
			cur = sdkmath.ZeroInt()
		}
		m.supply[coin.Denom] = cur.Add(coin.Amount)
	}
	return nil
}

func (m *mockBankKeeper) GetSupply(_ context.Context, denom string) sdk.Coin {
	if amt, ok := m.supply[denom]; ok {
		return sdk.NewCoin(denom, amt)
	}
	return sdk.NewCoin(denom, sdkmath.ZeroInt())
}

func (m *mockBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, _ string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	if m.sendAccErr != nil {
		return m.sendAccErr
	}
	key := recipientAddr.String()
	m.sentToAccount[key] = m.sentToAccount[key].Add(amt...)
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToModule(_ context.Context, _ string, recipientModule string, amt sdk.Coins) error {
	if m.sendModErr != nil {
		return m.sendModErr
	}
	m.sentToModule[recipientModule] = m.sentToModule[recipientModule].Add(amt...)
	return nil
}

func (m *mockBankKeeper) GetAllBalances(_ context.Context, addr sdk.AccAddress) sdk.Coins {
	if m.balances != nil {
		if coins, ok := m.balances[addr.String()]; ok {
			return coins
		}
	}
	return sdk.Coins{}
}

type mockStakingKeeper struct {
	activeCount uint32
	// validators maps consensus address (bech32) → validator record.
	validators map[string]stakingtypes.Validator
}

func (m *mockStakingKeeper) GetActiveValidatorCount(_ context.Context) uint32 {
	return m.activeCount
}

func (m *mockStakingKeeper) GetValidatorByConsAddr(_ context.Context, consAddr sdk.ConsAddress) (stakingtypes.Validator, error) {
	if v, ok := m.validators[consAddr.String()]; ok {
		return v, nil
	}
	return stakingtypes.Validator{}, stakingtypes.ErrNoValidatorFound
}

// mockDistrKeeper mimics x/distribution's withdraw-address mapping:
// returns the mapped address when set, else the delegator itself.
type mockDistrKeeper struct {
	withdrawAddrs map[string]sdk.AccAddress
}

func (m *mockDistrKeeper) GetDelegatorWithdrawAddr(_ context.Context, delAddr sdk.AccAddress) (sdk.AccAddress, error) {
	if w, ok := m.withdrawAddrs[delAddr.String()]; ok {
		return w, nil
	}
	return delAddr, nil
}

func setupKeeperWithBank(t *testing.T, bk *mockBankKeeper, sk *mockStakingKeeper) (keeper.Keeper, sdk.Context) {
	t.Helper()
	// Preserve the retired reward calculator as an isolated unit-test fixture.
	// AppModule.BeginBlock never calls it in consensus v2 and valid Params keep
	// both values at zero.
	gs := types.DefaultGenesis()
	gs.Params.BlockReward = "10000000"
	gs.Params.FloorReward = "100000"
	return setupKeeperWithBankAndGenesis(t, bk, sk, gs)
}

func setupKeeperWithBankAndGenesis(t *testing.T, bk *mockBankKeeper, sk *mockStakingKeeper, gs *types.GenesisState) (keeper.Keeper, sdk.Context) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		t.Fatalf("failed to load latest version: %v", err)
	}

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	k := keeper.NewKeeper(cdc, runtime.NewKVStoreService(storeKey), bk, sk, "authority")
	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 1000}, false, log.NewNopLogger())

	k.InitGenesis(ctx, gs)

	return k, ctx
}

// ---------- Block Reward Distribution Tests ----------

func TestDistributeBlockReward_InjectedLegacyParamsRemainInert(t *testing.T) {
	bk := newMockBankKeeper()
	sk := &mockStakingKeeper{activeCount: 22}
	k, ctx := setupKeeperWithBank(t, bk, sk)

	mintedBefore := k.GetTotalMinted(ctx)
	if mintedBefore.Sign() != 0 {
		t.Fatalf("expected 0 total minted at genesis, got %s", mintedBefore.String())
	}

	producer := sdk.AccAddress("producer____________").String()

	dist, err := k.DistributeBlockReward(ctx, producer, 22, true)
	if err != nil {
		t.Fatalf("distribute block reward failed: %v", err)
	}

	if dist.TotalMinted != "0" || dist.ProducerReward != "0" || dist.ResearchShare != "0" ||
		dist.DevelopmentAmount != "0" || dist.ProtocolShare != "0" || dist.FounderShare != "0" {
		t.Fatalf("retired method returned a non-zero distribution: %+v", dist)
	}

	mintedAfter := k.GetTotalMinted(ctx)
	if mintedAfter.Cmp(mintedBefore) != 0 {
		t.Fatal("retired method changed total minted")
	}
	if !bk.mintedCoins.IsZero() {
		t.Fatalf("retired method minted %s", bk.mintedCoins)
	}
	if len(bk.sentToAccount) != 0 || len(bk.sentToModule) != 0 {
		t.Fatalf("retired method transferred value: accounts=%v modules=%v", bk.sentToAccount, bk.sentToModule)
	}
	if _, found := k.GetBlockRewardDistribution(ctx, uint64(ctx.BlockHeight())); found {
		t.Fatal("retired method wrote a new block reward record")
	}
}

func TestDistributeBlockReward_RecordsValidatorCountButNeverScalesMint(t *testing.T) {
	bk := newMockBankKeeper()
	sk := &mockStakingKeeper{activeCount: 3}
	k, ctx := setupKeeperWithBank(t, bk, sk)

	producer := sdk.AccAddress("producer____________").String()

	dist, err := k.DistributeBlockReward(ctx, producer, 3, true)
	if err != nil {
		t.Fatalf("distribute block reward failed: %v", err)
	}

	if dist.ValidatorCount != 3 {
		t.Errorf("expected validator count 3, got %d", dist.ValidatorCount)
	}
	if dist.TotalMinted != "0" {
		t.Fatalf("validator count reactivated retired issuance: %+v", dist)
	}
}

func TestDistributeBlockReward_EmptyBlock(t *testing.T) {
	bk := newMockBankKeeper()
	sk := &mockStakingKeeper{activeCount: 22}
	k, ctx := setupKeeperWithBank(t, bk, sk)

	mintedBefore := k.GetTotalMinted(ctx)
	producer := sdk.AccAddress("producer____________").String()

	dist, err := k.DistributeBlockReward(ctx, producer, 22, false)
	if err != nil {
		t.Fatalf("distribute block reward failed: %v", err)
	}

	if dist.TotalMinted != "0" {
		t.Errorf("expected 0 minted for empty block, got %s", dist.TotalMinted)
	}
	mintedAfter := k.GetTotalMinted(ctx)
	if mintedAfter.Cmp(mintedBefore) != 0 {
		t.Errorf("expected total minted unchanged for empty block")
	}
}

func TestDistributeBlockReward_NilBankKeeper(t *testing.T) {
	k, ctx := setupKeeper(t)

	producer := sdk.AccAddress("producer____________").String()

	dist, err := k.DistributeBlockReward(ctx, producer, 22, true)
	if err != nil {
		t.Fatalf("distribute block reward failed: %v", err)
	}

	if dist.TotalMinted != "0" {
		t.Fatalf("valid v2 params must not mint, got distribution %+v", dist)
	}
	total := k.GetTotalMinted(ctx)
	if total.Sign() != 0 {
		t.Fatalf("valid v2 params changed total minted to %s", total)
	}
}

// ---------- Claim Rewards Tests ----------

func TestClaimRewards_SendsCoins(t *testing.T) {
	bk := newMockBankKeeper()
	sk := &mockStakingKeeper{activeCount: 22}
	k, ctx := setupKeeperWithBank(t, bk, sk)

	recipient := sdk.AccAddress("recipient1__________").String()

	schedule, err := k.CreateVestingSchedule(ctx, "claim-1", "fact-1", recipient,
		"1000000000000000000", types.CategoryFormalProof, types.SourceVerification)
	if err != nil {
		t.Fatalf("create vesting failed: %v", err)
	}

	ctx = ctx.WithBlockHeight(20000)

	claimed, err := k.ClaimRewards(ctx, recipient, schedule.Id)
	if err != nil {
		t.Fatalf("claim rewards failed: %v", err)
	}

	if claimed == "0" {
		t.Fatal("expected non-zero claimed amount after cliff")
	}

	if _, ok := bk.sentToAccount[recipient]; !ok {
		t.Fatal("expected SendCoinsFromModuleToAccount for recipient")
	}
}

func TestClaimRewards_NilBankKeeper(t *testing.T) {
	k, ctx := setupKeeper(t)

	recipient := sdk.AccAddress("recipient1__________").String()

	schedule, err := k.CreateVestingSchedule(ctx, "claim-1", "fact-1", recipient,
		"1000000000000000000", types.CategoryFormalProof, types.SourceVerification)
	if err != nil {
		t.Fatalf("create vesting failed: %v", err)
	}

	ctx = ctx.WithBlockHeight(20000)

	claimed, err := k.ClaimRewards(ctx, recipient, schedule.Id)
	if err != nil {
		t.Fatalf("claim rewards with nil bank failed: %v", err)
	}
	if claimed == "0" {
		t.Fatal("expected non-zero claimed amount")
	}
}

// ---------- 4-Way Revenue Split Tests ----------

func TestDistributeRevenue_4WaySplit(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Default split: contributor 55%, protocol 22%, research 3.33%, development 19.67%
	routing, err := k.DistributeRevenue(ctx, types.SourceBlockProduction, "10000",
		sdk.AccAddress("recipient___________").String(), "")
	if err != nil {
		t.Fatalf("distribute revenue failed: %v", err)
	}

	// 10000 * 550000 / 1000000 = 5500
	if routing.ContributorShare != "5500" {
		t.Errorf("expected contributor share 5500, got %s", routing.ContributorShare)
	}
	// 10000 * 220000 / 1000000 = 2200
	if routing.ProtocolShare != "2200" {
		t.Errorf("expected protocol share 2200, got %s", routing.ProtocolShare)
	}
	// 10000 * 33300 / 1000000 = 333
	if routing.ResearchShare != "333" {
		t.Errorf("expected research share 333, got %s", routing.ResearchShare)
	}
	// development = 10000 - 5500 - 2200 - 333 = 1967
	if routing.DevelopmentAmount != "1967" {
		t.Errorf("expected development amount 1967, got %s", routing.DevelopmentAmount)
	}
}

func TestDistributeRevenue_ProtocolSubSplit(t *testing.T) {
	k, ctx := setupKeeper(t)

	routing, err := k.DistributeRevenue(ctx, types.SourceBlockProduction, "10000000",
		sdk.AccAddress("recipient___________").String(), "")
	if err != nil {
		t.Fatalf("distribute revenue failed: %v", err)
	}

	// Protocol share: 10000000 * 220000 / 1000000 = 2200000
	protocolBig := new(big.Int)
	protocolBig.SetString(routing.ProtocolShare, 10)

	// Citation (50%): 2200000 * 500000 / 1000000 = 1100000
	if routing.CitationPool != "1100000" {
		t.Errorf("expected citation pool 1100000, got %s", routing.CitationPool)
	}
	// Verification (30%): 2200000 * 300000 / 1000000 = 660000
	if routing.VerificationPool != "660000" {
		t.Errorf("expected verification pool 660000, got %s", routing.VerificationPool)
	}
	// Treasury = remainder: 2200000 - 1100000 - 660000 = 440000
	if routing.TreasuryShare != "440000" {
		t.Errorf("expected treasury share 440000, got %s", routing.TreasuryShare)
	}
}

func TestDistributeRevenue_InvalidAmount(t *testing.T) {
	k, ctx := setupKeeper(t)

	_, err := k.DistributeRevenue(ctx, types.SourceBlockProduction, "0", "addr", "")
	if err == nil {
		t.Fatal("expected error for zero amount")
	}

	_, err = k.DistributeRevenue(ctx, types.SourceBlockProduction, "notanumber", "addr", "")
	if err == nil {
		t.Fatal("expected error for non-numeric amount")
	}
}

func TestDistributeRevenue_SplitSumsToTotal(t *testing.T) {
	k, ctx := setupKeeper(t)

	routing, err := k.DistributeRevenue(ctx, types.SourceBlockProduction, "999999",
		sdk.AccAddress("recipient___________").String(), "")
	if err != nil {
		t.Fatalf("distribute revenue failed: %v", err)
	}

	contributor := new(big.Int)
	contributor.SetString(routing.ContributorShare, 10)
	protocol := new(big.Int)
	protocol.SetString(routing.ProtocolShare, 10)
	research := new(big.Int)
	research.SetString(routing.ResearchShare, 10)
	development := new(big.Int)
	development.SetString(routing.DevelopmentAmount, 10)

	total := new(big.Int).Add(contributor, protocol)
	total.Add(total, research)
	total.Add(total, development)

	if total.Int64() != 999999 {
		t.Errorf("expected split to sum to 999999, got %s", total.String())
	}
}

// ---------- Development Fund Deposit Tests ----------

func TestDistributeBlockReward_DoesNotDepositToDevelopmentFund(t *testing.T) {
	bk := newMockBankKeeper()
	sk := &mockStakingKeeper{activeCount: 22}
	k, ctx := setupKeeperWithBank(t, bk, sk)

	producer := sdk.AccAddress("producer____________").String()

	dist, err := k.DistributeBlockReward(ctx, producer, 22, true)
	if err != nil {
		t.Fatalf("distribute block reward failed: %v", err)
	}

	if dist.DevelopmentAmount != "0" {
		t.Fatalf("retired method returned a development amount: %+v", dist)
	}

	devCoins := bk.sentToModule["development_fund"]
	if !devCoins.AmountOf("uzrn").IsZero() {
		t.Errorf("retired method sent %s uzrn to development_fund", devCoins.AmountOf("uzrn"))
	}

}

// ---------- Falsify Claim Tests ----------

func TestFalsifyClaim_ClawbackCalculation(t *testing.T) {
	bk := newMockBankKeeper()
	sk := &mockStakingKeeper{activeCount: 22}
	k, ctx := setupKeeperWithBank(t, bk, sk)
	// The clawback is only reachable once the PoT layer has adjudicated the
	// linked fact false; this test is about the arithmetic, so grant that.
	k.SetKnowledgeKeeper(&stubKnowledgeKeeper{disproven: map[string]bool{"fact-1": true}})

	recipient := sdk.AccAddress("recipient1__________").String()

	schedule, err := k.CreateVestingSchedule(ctx, "claim-falsify", "fact-1", recipient,
		"1000000000000000000", types.CategoryFormalProof, types.SourceVerification)
	if err != nil {
		t.Fatalf("create vesting failed: %v", err)
	}

	ctx = ctx.WithBlockHeight(20000)
	k.ClaimRewards(ctx, recipient, schedule.Id)

	challenger := sdk.AccAddress("challenger__________").String()
	record, err := k.FalsifyClaim(ctx, "claim-falsify", challenger)
	if err != nil {
		t.Fatalf("falsify claim failed: %v", err)
	}

	if record.VestingId != schedule.Id {
		t.Errorf("expected vesting ID %s, got %s", schedule.Id, record.VestingId)
	}

	updated, _ := k.GetVestingSchedule(ctx, schedule.Id)
	if updated.Status != string(types.VestingStatusFalsified) {
		t.Errorf("expected falsified status, got %s", updated.Status)
	}
}

func TestFalsifyClaim_AlreadyFalsified(t *testing.T) {
	k, ctx := setupKeeper(t)
	k.SetKnowledgeKeeper(&stubKnowledgeKeeper{disproven: map[string]bool{"fact-1": true}})

	recipient := sdk.AccAddress("recipient1__________").String()

	k.CreateVestingSchedule(ctx, "claim-dup", "fact-1", recipient,
		"1000000000000000000", types.CategoryFormalProof, types.SourceVerification)

	_, err := k.FalsifyClaim(ctx, "claim-dup", "challenger")
	if err != nil {
		t.Fatalf("first falsify failed: %v", err)
	}

	_, err = k.FalsifyClaim(ctx, "claim-dup", "challenger")
	if err == nil {
		t.Fatal("expected error for already falsified claim")
	}
}

// ---------- Full Lifecycle Test ----------

func TestVestingFullLifecycle(t *testing.T) {
	ms, k, ctx := setupMsgServer(t)

	recipient := sdk.AccAddress("recipient1__________").String()

	createResp, err := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  recipient,
		Amount:       "10000000000000000000",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-lifecycle",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	ms.AccelerateVesting(ctx, &types.MsgAccelerateVesting{
		Authority: "authority", VestingId: createResp.VestingId,
		AccelerationFactor: 1000000,
	})

	ctx = ctx.WithBlockHeight(2000)
	ms.PauseVesting(ctx, &types.MsgPauseVesting{
		Authority: "authority", VestingId: createResp.VestingId, Reason: "dispute",
	})

	schedule, _ := k.GetVestingSchedule(ctx, createResp.VestingId)
	if schedule.Status != string(types.VestingStatusPaused) {
		t.Fatalf("expected paused, got %s", schedule.Status)
	}

	ctx = ctx.WithBlockHeight(3000)
	ms.ResumeVesting(ctx, &types.MsgResumeVesting{
		Authority: "authority", VestingId: createResp.VestingId,
	})

	schedule, _ = k.GetVestingSchedule(ctx, createResp.VestingId)
	if schedule.Status != string(types.VestingStatusActive) {
		t.Fatalf("expected active after resume, got %s", schedule.Status)
	}
	if schedule.TotalPausedBlocks != 1000 {
		t.Errorf("expected 1000 paused blocks, got %d", schedule.TotalPausedBlocks)
	}

	ms.AccelerateVesting(ctx, &types.MsgAccelerateVesting{
		Authority: "authority", VestingId: createResp.VestingId,
		AccelerationFactor: 600000,
	})

	schedule, _ = k.GetVestingSchedule(ctx, createResp.VestingId)
	if schedule.DefenseCount != 1 || schedule.ReplicationCount != 1 {
		t.Errorf("expected 1/1 defense/replication, got %d/%d", schedule.DefenseCount, schedule.ReplicationCount)
	}

	_, err = ms.CompleteVesting(ctx, &types.MsgCompleteVesting{
		Authority: recipient, VestingId: createResp.VestingId,
	})
	if err != nil {
		t.Fatalf("complete failed: %v", err)
	}

	schedule, _ = k.GetVestingSchedule(ctx, createResp.VestingId)
	if schedule.Status != string(types.VestingStatusCompleted) {
		t.Errorf("expected completed, got %s", schedule.Status)
	}
}

// ==================== Retired Automatic-Issuance Tests ====================

func setupMintKeeper(t *testing.T, bk *mockBankKeeper, totalMinted string, blockHeight int64) (keeper.Keeper, sdk.Context) {
	t.Helper()
	gs := types.DefaultGenesis()
	// Inject invalid pre-v2 values directly into keeper state. The public genesis
	// path rejects them; this fixture proves the retired Go method is still inert
	// if validation is bypassed by an internal caller.
	gs.Params.BlockReward = "10000000"
	gs.Params.FloorReward = "100000"
	gs.Params.InitialFundBalance = totalMinted

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		t.Fatalf("failed to load latest version: %v", err)
	}

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	sk := &mockStakingKeeper{activeCount: 22}

	k := keeper.NewKeeper(cdc, runtime.NewKVStoreService(storeKey), bk, sk, "authority")
	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: blockHeight}, false, log.NewNopLogger())

	k.InitGenesis(ctx, gs)

	if totalMinted != "" && totalMinted != "0" {
		amt, ok := new(big.Int).SetString(totalMinted, 10)
		if ok && amt.Sign() > 0 {
			bk.supply["uzrn"] = sdkmath.NewIntFromBigInt(amt)
		}
	}

	return k, ctx
}

func TestRetiredBlockRewardIgnoresLegacyDecayAtEveryEpoch(t *testing.T) {
	for _, height := range []int64{0, 100000, 200000, 500000, 1000000, 1000 * 100000} {
		t.Run(fmt.Sprintf("height_%d", height), func(t *testing.T) {
			bk := newMockBankKeeper()
			k, ctx := setupMintKeeper(t, bk, "0", height)
			dist, err := k.DistributeBlockReward(
				ctx,
				sdk.AccAddress("producer____________").String(),
				22,
				true,
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if dist.TotalMinted != "0" || !bk.mintedCoins.IsZero() {
				t.Fatalf("legacy decay fields reactivated issuance at height %d: dist=%+v minted=%s", height, dist, bk.mintedCoins)
			}
		})
	}
}

func TestMintWithCap_SupplyExhausted(t *testing.T) {
	bk := newMockBankKeeper()
	maxSupply := "222222222000000"
	// Set supply to exactly maxSupply so remaining = 0 from the start.
	k, ctx := setupMintKeeper(t, bk, maxSupply, 0)

	actual, err := k.MintWithCap(ctx, types.ModuleName, big.NewInt(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if actual.Sign() != 0 {
		t.Errorf("expected 0 mint when supply exhausted, got %s", actual)
	}
}

func TestMintWithCap_EnforcesSupplyLimit(t *testing.T) {
	bk := newMockBankKeeper()
	k, ctx := setupMintKeeper(t, bk, "0", 0)

	maxSupply := new(big.Int)
	maxSupply.SetString(types.MaxSupplyUzrn, 10)
	overMax := new(big.Int).Add(maxSupply, big.NewInt(1))

	actual, err := k.MintWithCap(ctx, types.ModuleName, overMax)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if actual.Cmp(maxSupply) != 0 {
		t.Errorf("expected mint capped to %s, got %s", maxSupply.String(), actual.String())
	}

	totalMinted := k.GetTotalMinted(ctx)
	if totalMinted.Cmp(maxSupply) != 0 {
		t.Errorf("expected total minted %s, got %s", maxSupply.String(), totalMinted.String())
	}

	actual2, err := k.MintWithCap(ctx, types.ModuleName, big.NewInt(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if actual2.Sign() != 0 {
		t.Errorf("expected 0 mint when supply exhausted, got %s", actual2.String())
	}
}

func TestMintWithCap_SupplyMonotonic(t *testing.T) {
	// No burn recycling in the new model: supply monotonically increases to cap.
	// Once the cap is reached, no further minting is possible.
	bk := newMockBankKeeper()
	nearCap := "222222221999500"
	k, ctx := setupMintKeeper(t, bk, nearCap, 0)

	// Mint exactly the remaining 500
	actual, err := k.MintWithCap(ctx, types.ModuleName, big.NewInt(500))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if actual.Int64() != 500 {
		t.Errorf("expected 500 minted, got %s", actual.String())
	}

	// Supply exhausted — no more minting possible
	actual2, err := k.MintWithCap(ctx, types.ModuleName, big.NewInt(1000))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if actual2.Sign() != 0 {
		t.Errorf("expected 0 mint when supply exhausted, got %s", actual2.String())
	}

	// Repeated attempts also yield zero (no burn recycling to free headroom)
	actual3, err := k.MintWithCap(ctx, types.ModuleName, big.NewInt(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if actual3.Sign() != 0 {
		t.Errorf("expected 0 after cap permanently reached, got %s", actual3.String())
	}
}

func TestTotalMintedGetSet(t *testing.T) {
	k, ctx := setupKeeper(t)

	k.SetTotalMinted(ctx, big.NewInt(999999))
	total := k.GetTotalMinted(ctx)
	if total.Int64() != 999999 {
		t.Errorf("expected 999999, got %s", total.String())
	}

	k.SetTotalMinted(ctx, big.NewInt(1999999))
	total = k.GetTotalMinted(ctx)
	if total.Int64() != 1999999 {
		t.Errorf("expected 1999999, got %s", total.String())
	}
}

func TestExportGenesis_RetiredRewardDoesNotChangeTotalMinted(t *testing.T) {
	bk := newMockBankKeeper()
	k, ctx := setupMintKeeper(t, bk, "0", 0)

	producer := sdk.AccAddress("producer____________").String()
	k.DistributeBlockReward(ctx, producer, 22, true)

	exported := k.ExportGenesis(ctx)

	if exported.Params.InitialFundBalance != "0" {
		t.Errorf("retired reward changed exported mint ledger to %s", exported.Params.InitialFundBalance)
	}
}

func TestExportGenesis_PreservesTerminalSchedulesAndHistory(t *testing.T) {
	k, ctx := setupKeeper(t)
	recipient := sdk.AccAddress("history_recipient___").String()
	schedules := []*types.VestingSchedule{
		{Id: "vesting-active", ClaimId: "claim-active", Recipient: recipient, Status: string(types.VestingStatusActive)},
		{Id: "vesting-completed", ClaimId: "claim-completed", Recipient: recipient, Status: string(types.VestingStatusCompleted)},
		{Id: "vesting-falsified", ClaimId: "claim-falsified", Recipient: recipient, Status: string(types.VestingStatusFalsified)},
	}
	for _, schedule := range schedules {
		k.SetVestingSchedule(ctx, schedule)
	}
	k.SetClawbackRecord(ctx, &types.ClawbackRecord{
		Id: "clawback-1", VestingId: "vesting-falsified", UnvestedForfeited: "7",
	})
	k.SetBlockRewardDistribution(ctx, &types.BlockRewardDistribution{
		BlockHeight: 9, TotalMinted: "11", FundBalanceAfter: "22",
	})

	exported := k.ExportGenesis(ctx)
	if len(exported.VestingSchedules) != 3 {
		t.Fatalf("exported %d schedules, want 3 including terminal history", len(exported.VestingSchedules))
	}
	if len(exported.ClawbackRecords) != 1 || len(exported.BlockRewardDistributions) != 1 {
		t.Fatalf(
			"export lost history: clawbacks=%d block_rewards=%d",
			len(exported.ClawbackRecords),
			len(exported.BlockRewardDistributions),
		)
	}

	bz, err := proto.Marshal(exported)
	if err != nil {
		t.Fatalf("marshal exported genesis: %v", err)
	}
	var restored types.GenesisState
	if err := proto.Unmarshal(bz, &restored); err != nil {
		t.Fatalf("unmarshal exported genesis: %v", err)
	}
	k2, ctx2 := setupKeeper(t)
	k2.InitGenesis(ctx2, &restored)

	for _, schedule := range schedules {
		got, found := k2.GetVestingSchedule(ctx2, schedule.Id)
		if !found || got.Status != schedule.Status {
			t.Fatalf("schedule %s not restored with status %s", schedule.Id, schedule.Status)
		}
	}
	if _, found := k2.GetVestingByClaimId(ctx2, "claim-falsified"); !found {
		t.Fatal("terminal claim index not restored")
	}
	if got := len(k2.GetVestingSchedulesByRecipient(ctx2, recipient)); got != 3 {
		t.Fatalf("recipient index restored %d schedules, want 3", got)
	}
	if got := len(k2.GetAllActiveVestingSchedules(ctx2)); got != 1 {
		t.Fatalf("active index restored %d schedules, want 1", got)
	}
	if _, found := k2.GetClawbackRecord(ctx2, "clawback-1"); !found {
		t.Fatal("clawback record not restored")
	}
	if got := k2.GetAllBlockRewardDistributions(ctx2); len(got) != 1 {
		t.Fatalf("block-reward history restored %d records, want 1", len(got))
	}
}

func TestExportGenesis_PreservesExplicitClaimIndexWithDuplicateLegacyClaims(t *testing.T) {
	k, ctx := setupKeeper(t)
	recipient := sdk.AccAddress("duplicate_claim_idx_").String()

	// Primary-key iteration restores "vesting-z" after "vesting-a", which
	// would silently change the selected schedule without an explicit index.
	k.SetVestingSchedule(ctx, &types.VestingSchedule{
		Id: "vesting-z", ClaimId: "legacy-duplicate-claim", Recipient: recipient,
		Status: string(types.VestingStatusActive),
	})
	k.SetVestingSchedule(ctx, &types.VestingSchedule{
		Id: "vesting-a", ClaimId: "legacy-duplicate-claim", Recipient: recipient,
		Status: string(types.VestingStatusPaused),
	})
	before, found := k.GetVestingByClaimId(ctx, "legacy-duplicate-claim")
	require.True(t, found)
	require.Equal(t, "vesting-a", before.Id)

	exported := k.ExportGenesis(ctx)
	require.NoError(t, exported.Validate())
	require.Len(t, exported.ClaimScheduleIndexes, 1)
	require.Equal(t, "vesting-a", exported.ClaimScheduleIndexes[0].VestingId)

	bz, err := proto.Marshal(exported)
	require.NoError(t, err)
	var restored types.GenesisState
	require.NoError(t, proto.Unmarshal(bz, &restored))

	k2, ctx2 := setupKeeper(t)
	k2.InitGenesis(ctx2, &restored)
	after, found := k2.GetVestingByClaimId(ctx2, "legacy-duplicate-claim")
	require.True(t, found)
	require.Equal(t, before.Id, after.Id,
		"relaunch must preserve the live claim index, not primary-key replay order")
}

// ---------- Retired Block Reward Accounting ----------

func TestDistributeBlockReward_HasNoOutflowsOrRetainedSkim(t *testing.T) {
	bk := newMockBankKeeper()
	sk := &mockStakingKeeper{activeCount: 22}
	k, ctx := setupKeeperWithBank(t, bk, sk)

	producer := sdk.AccAddress("producer____________").String()
	dist, err := k.DistributeBlockReward(ctx, producer, 22, true)
	if err != nil {
		t.Fatalf("distribute block reward failed: %v", err)
	}

	if dist.TotalMinted != "0" || dist.ProducerReward != "0" || dist.ResearchShare != "0" ||
		dist.DevelopmentAmount != "0" || dist.ProtocolShare != "0" || dist.FounderShare != "0" {
		t.Fatalf("retired reward returned non-zero accounting: %+v", dist)
	}
	if !bk.mintedCoins.IsZero() || len(bk.sentToAccount) != 0 || len(bk.sentToModule) != 0 {
		t.Fatalf("retired reward moved value: minted=%s accounts=%v modules=%v", bk.mintedCoins, bk.sentToAccount, bk.sentToModule)
	}
}

// ---------- Fee Routing Tests ----------

func TestRouteFees_SweepsFeeCollector(t *testing.T) {
	bk := newMockBankKeeper()
	feeCollectorModuleAddr := authtypes.NewModuleAddress(authtypes.FeeCollectorName)
	bk.balances = map[string]sdk.Coins{
		feeCollectorModuleAddr.String(): sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewInt(100000))),
	}

	sk := &mockStakingKeeper{activeCount: 22}
	k, ctx := setupKeeperWithBank(t, bk, sk)

	err := k.RouteFees(ctx)
	if err != nil {
		t.Fatalf("RouteFees failed: %v", err)
	}

	// Research: 3.33% of 100000 = 3330
	researchCoins := bk.sentToModule["research_fund"]
	if researchCoins.AmountOf("uzrn").Int64() != 3330 {
		t.Errorf("expected 3330 uzrn to research_fund, got %d", researchCoins.AmountOf("uzrn").Int64())
	}

	// Development: 19.67% of 100000 = 19670
	devCoins := bk.sentToModule["development_fund"]
	if devCoins.AmountOf("uzrn").Int64() != 19670 {
		t.Errorf("expected 19670 uzrn to development_fund, got %d", devCoins.AmountOf("uzrn").Int64())
	}

}

func TestRouteFees_EmptyFeeCollector(t *testing.T) {
	bk := newMockBankKeeper()
	bk.balances = map[string]sdk.Coins{}

	sk := &mockStakingKeeper{activeCount: 22}
	k, ctx := setupKeeperWithBank(t, bk, sk)

	err := k.RouteFees(ctx)
	if err != nil {
		t.Fatalf("RouteFees failed: %v", err)
	}

	if len(bk.sentToModule) > 0 {
		t.Errorf("expected no module sends, got %v", bk.sentToModule)
	}
}

func TestRouteFees_OnlyProcessesUzrn(t *testing.T) {
	feeCollectorModuleAddr := authtypes.NewModuleAddress(authtypes.FeeCollectorName)
	bk := newMockBankKeeper()
	bk.balances = map[string]sdk.Coins{
		feeCollectorModuleAddr.String(): sdk.NewCoins(
			sdk.NewCoin("uzrn", sdkmath.NewInt(100000)),
			sdk.NewCoin("atom", sdkmath.NewInt(50000)),
		),
	}

	sk := &mockStakingKeeper{activeCount: 22}
	k, ctx := setupKeeperWithBank(t, bk, sk)

	err := k.RouteFees(ctx)
	if err != nil {
		t.Fatalf("RouteFees failed: %v", err)
	}

	researchCoins := bk.sentToModule["research_fund"]
	if researchCoins.AmountOf("uzrn").Int64() != 3330 {
		t.Errorf("expected 3330 uzrn to research_fund, got %d", researchCoins.AmountOf("uzrn").Int64())
	}
	if researchCoins.AmountOf("atom").Int64() != 0 {
		t.Errorf("expected 0 atom to research_fund, got %d", researchCoins.AmountOf("atom").Int64())
	}
}

// ==================== Retired Founder Auto-Split Tests ====================

func setupFounderKeeper(t *testing.T, bk *mockBankKeeper, founderAddr string, govHeight uint64) (keeper.Keeper, sdk.Context) {
	t.Helper()
	gs := types.DefaultGenesis()
	gs.Params.FounderShareBps = 70000
	gs.Params.FounderAddress = founderAddr
	gs.Params.GovernanceActivationHeight = govHeight
	gs.Params.BlockReward = "10000000"
	gs.Params.FloorReward = "100000"
	return setupKeeperWithBankAndGenesis(t, bk, &mockStakingKeeper{activeCount: 22}, gs)
}

func TestFounderAutoSplitRetiredForLegacyState(t *testing.T) {
	// Even a directly injected legacy state cannot activate the retired tap.
	// The entire legacy automatic-reward method is inert, including founder,
	// proposer, public-good, and protocol outputs.
	bk := newMockBankKeeper()
	founderAddr := sdk.AccAddress("founder_____________").String()
	k, ctx := setupFounderKeeper(t, bk, founderAddr, 0)

	producer := sdk.AccAddress("producer____________").String()
	dist, err := k.DistributeBlockReward(ctx, producer, 22, true)
	if err != nil {
		t.Fatalf("distribute block reward failed: %v", err)
	}

	if dist.TotalMinted != "0" || dist.FounderShare != "0" || dist.ResearchShare != "0" || dist.ProducerReward != "0" {
		t.Fatalf("legacy founder/reward fields reactivated value: %+v", dist)
	}

	founderCoins := bk.sentToAccount[founderAddr]
	if founderCoins.AmountOf("uzrn").Int64() != 0 {
		t.Errorf("expected 0 uzrn to founder, got %d", founderCoins.AmountOf("uzrn").Int64())
	}
	if len(bk.sentToModule) != 0 {
		t.Errorf("retired automatic reward sent module funds: %v", bk.sentToModule)
	}
}

func TestFounderSplitDisabled(t *testing.T) {
	bk := newMockBankKeeper()
	k, ctx := setupFounderKeeper(t, bk, "", 0)

	producer := sdk.AccAddress("producer____________").String()
	dist, err := k.DistributeBlockReward(ctx, producer, 22, true)
	if err != nil {
		t.Fatalf("distribute block reward failed: %v", err)
	}

	if dist.ResearchShare != "0" {
		t.Errorf("retired automatic reward returned research share %s", dist.ResearchShare)
	}
	if dist.FounderShare != "0" {
		t.Errorf("expected founder share 0 when disabled, got %s", dist.FounderShare)
	}
}

func TestFounderSplitRetiredRegardlessOfDeprecatedHeight(t *testing.T) {
	bk := newMockBankKeeper()
	founderAddr := sdk.AccAddress("founder_____________").String()
	k, ctx := setupFounderKeeper(t, bk, founderAddr, 500)

	producer := sdk.AccAddress("producer____________").String()
	dist, err := k.DistributeBlockReward(ctx, producer, 22, true)
	if err != nil {
		t.Fatalf("distribute block reward failed: %v", err)
	}

	if dist.FounderShare != "0" {
		t.Errorf("expected retired founder share 0, got %s", dist.FounderShare)
	}
	if dist.ResearchShare != "0" {
		t.Errorf("retired automatic reward returned research share %s", dist.ResearchShare)
	}
}

// ==================== DepositToResearchFund Tests ====================

func TestDepositToResearchFund_RetiredFounderGetsNothing(t *testing.T) {
	bk := newMockBankKeeper()
	founderAddr := sdk.AccAddress("founder_____________").String()
	k, ctx := setupFounderKeeper(t, bk, founderAddr, 0)

	depositCoins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewInt(100000)))
	err := k.DepositToResearchFund(ctx, "knowledge", depositCoins)
	if err != nil {
		t.Fatalf("DepositToResearchFund failed: %v", err)
	}

	researchCoins := bk.sentToModule["research_fund"]
	if researchCoins.AmountOf("uzrn").Int64() != 100000 {
		t.Errorf("expected 100000 to research_fund, got %d", researchCoins.AmountOf("uzrn").Int64())
	}

	founderCoins := bk.sentToAccount[founderAddr]
	if founderCoins.AmountOf("uzrn").Int64() != 0 {
		t.Errorf("expected 0 to founder, got %d", founderCoins.AmountOf("uzrn").Int64())
	}
}

func TestDepositToResearchFund_NoFounderAddress(t *testing.T) {
	bk := newMockBankKeeper()
	k, ctx := setupFounderKeeper(t, bk, "", 0)

	depositCoins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewInt(100000)))
	err := k.DepositToResearchFund(ctx, "knowledge", depositCoins)
	if err != nil {
		t.Fatalf("DepositToResearchFund failed: %v", err)
	}

	researchCoins := bk.sentToModule["research_fund"]
	if researchCoins.AmountOf("uzrn").Int64() != 100000 {
		t.Errorf("expected 100000 to research_fund (no founder), got %d", researchCoins.AmountOf("uzrn").Int64())
	}

	if len(bk.sentToAccount) > 0 {
		t.Errorf("expected no account sends with empty founder, got %v", bk.sentToAccount)
	}
}

func TestDepositToResearchFund_RetiredRegardlessOfDeprecatedHeight(t *testing.T) {
	bk := newMockBankKeeper()
	founderAddr := sdk.AccAddress("founder_____________").String()
	k, ctx := setupFounderKeeper(t, bk, founderAddr, 500)

	depositCoins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewInt(100000)))
	err := k.DepositToResearchFund(ctx, "knowledge", depositCoins)
	if err != nil {
		t.Fatalf("DepositToResearchFund failed: %v", err)
	}

	researchCoins := bk.sentToModule["research_fund"]
	if researchCoins.AmountOf("uzrn").Int64() != 100000 {
		t.Errorf("expected 100000 to research_fund, got %d", researchCoins.AmountOf("uzrn").Int64())
	}

	founderCoins := bk.sentToAccount[founderAddr]
	if founderCoins.AmountOf("uzrn").Int64() != 0 {
		t.Errorf("expected 0 to configured founder, got %d", founderCoins.AmountOf("uzrn").Int64())
	}
}

func TestDepositToResearchFund_ZeroAmount(t *testing.T) {
	bk := newMockBankKeeper()
	founderAddr := sdk.AccAddress("founder_____________").String()
	k, ctx := setupFounderKeeper(t, bk, founderAddr, 0)

	err := k.DepositToResearchFund(ctx, "knowledge", sdk.Coins{})
	if err != nil {
		t.Fatalf("DepositToResearchFund with zero amount failed: %v", err)
	}

	if len(bk.sentToModule) > 0 {
		t.Errorf("expected no module sends for zero amount, got %v", bk.sentToModule)
	}
}

func TestDepositToResearchFund_EmitsEvent(t *testing.T) {
	bk := newMockBankKeeper()
	founderAddr := sdk.AccAddress("founder_____________").String()
	k, ctx := setupFounderKeeper(t, bk, founderAddr, 0)

	depositCoins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewInt(100000)))
	err := k.DepositToResearchFund(ctx, "billing", depositCoins)
	if err != nil {
		t.Fatalf("DepositToResearchFund failed: %v", err)
	}

	events := ctx.EventManager().Events()
	found := false
	for _, event := range events {
		if event.Type == "zerone.vesting_rewards.research_fund_deposit" {
			found = true
			attrs := make(map[string]string)
			for _, attr := range event.Attributes {
				attrs[attr.Key] = attr.Value
			}
			if attrs["source_module"] != "billing" {
				t.Errorf("expected source_module=billing, got %s", attrs["source_module"])
			}
			if attrs["total"] != "100000" {
				t.Errorf("expected total=100000, got %s", attrs["total"])
			}
			if attrs["research"] != "100000" {
				t.Errorf("expected research=100000, got %s", attrs["research"])
			}
			if attrs["founder"] != "0" {
				t.Errorf("expected founder=0, got %s", attrs["founder"])
			}
		}
	}
	if !found {
		t.Error("expected research_fund_deposit event to be emitted")
	}
}

// ==================== Retired Founder Share Governance Tests ====================

func TestUpdateParamsRejectsUnsafeRewardConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*types.Params)
	}{
		{
			name: "zero reward epoch",
			mutate: func(p *types.Params) {
				p.BlocksPerRewardEpoch = 0
			},
		},
		{
			name: "non-numeric block reward",
			mutate: func(p *types.Params) {
				p.BlockReward = "not-an-integer"
			},
		},
		{
			name: "transaction-presence block reward reactivated",
			mutate: func(p *types.Params) {
				p.BlockReward = "1"
			},
		},
		{
			name: "floor reward reactivated",
			mutate: func(p *types.Params) {
				p.FloorReward = "1"
			},
		},
		{
			name: "empty block reward reactivated",
			mutate: func(p *types.Params) {
				p.EmptyBlockRewardRate = 1
			},
		},
		{
			name: "empty reward above 100 percent",
			mutate: func(p *types.Params) {
				p.EmptyBlockRewardRate = 10_001
			},
		},
		{
			name: "clawback above 100 percent",
			mutate: func(p *types.Params) {
				p.ReleasedClawbackRate = 10_001
			},
		},
		{
			name: "knowledge floor above 100 percent",
			mutate: func(p *types.Params) {
				p.KnowledgeCouplingFloorBps = 1_000_001
			},
		},
		{
			name: "overflow-shaped revenue component",
			mutate: func(p *types.Params) {
				p.RevenueSplit.ContributorBps = ^uint64(0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms, k, ctx := setupMsgServer(t)
			before := k.GetParams(ctx)
			proposed := types.DefaultParams()
			tt.mutate(proposed)

			if _, err := ms.UpdateParams(ctx, &types.MsgUpdateParams{
				Authority: "authority",
				Params:    proposed,
			}); err == nil {
				t.Fatal("expected invalid params to be rejected")
			}

			after := k.GetParams(ctx)
			if after.BlocksPerRewardEpoch != before.BlocksPerRewardEpoch ||
				after.BlockReward != before.BlockReward ||
				after.EmptyBlockRewardRate != before.EmptyBlockRewardRate ||
				after.ReleasedClawbackRate != before.ReleasedClawbackRate ||
				after.KnowledgeCouplingFloorBps != before.KnowledgeCouplingFloorBps ||
				after.RevenueSplit.ContributorBps != before.RevenueSplit.ContributorBps {
				t.Fatal("rejected parameter update mutated keeper state")
			}
		})
	}
}

func TestUpdateParamsRejectsNilParams(t *testing.T) {
	ms, _, ctx := setupMsgServer(t)
	if _, err := ms.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: "authority",
		Params:    nil,
	}); err == nil {
		t.Fatal("expected nil params to be rejected")
	}
}

func TestFounderShareGovernance(t *testing.T) {
	founderAddr := sdk.AccAddress("founder_____________").String()
	for _, mutate := range []func(*types.Params){
		func(params *types.Params) { params.FounderShareBps = 1 },
		func(params *types.Params) { params.FounderAddress = founderAddr },
		func(params *types.Params) {
			params.FounderShareBps = 70_000
			params.FounderAddress = founderAddr
		},
	} {
		ms, k, ctx := setupMsgServer(t)
		proposed := types.DefaultParams()
		mutate(proposed)
		if _, err := ms.UpdateParams(ctx, &types.MsgUpdateParams{
			Authority: "authority",
			Params:    proposed,
		}); err == nil {
			t.Fatal("governance reactivated a retired founder field")
		}
		stored := k.GetParams(ctx)
		if stored.FounderShareBps != 0 || stored.FounderAddress != "" {
			t.Fatalf("rejected update mutated retired fields: %+v", stored)
		}
	}

	ms, k, ctx := setupMsgServer(t)
	if _, err := ms.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: "authority",
		Params:    types.DefaultParams(),
	}); err != nil {
		t.Fatalf("zero/empty compatibility fields should remain updateable: %v", err)
	}
	if params := k.GetParams(ctx); params.FounderShareBps != 0 || params.FounderAddress != "" {
		t.Fatalf("retired fields changed: %+v", params)
	}
}

func TestGovernanceCannotAdvertiseRetiredRewardScheduleChanges(t *testing.T) {
	for _, mutate := range []func(*types.Params){
		func(params *types.Params) { params.RewardDecayBps-- },
		func(params *types.Params) { params.BlocksPerRewardEpoch++ },
		func(params *types.Params) { params.MinValidatorsForFullReward++ },
		func(params *types.Params) { params.KnowledgeCouplingTargetBps-- },
		func(params *types.Params) { params.KnowledgeCouplingFloorBps-- },
	} {
		ms, k, ctx := setupMsgServer(t)
		before := k.GetParams(ctx)
		proposed := proto.Clone(before).(*types.Params)
		mutate(proposed)
		if _, err := ms.UpdateParams(ctx, &types.MsgUpdateParams{
			Authority: "authority",
			Params:    proposed,
		}); err == nil {
			t.Fatal("governance changed an inert retired reward-schedule field")
		}
		require.True(t, proto.Equal(before, k.GetParams(ctx)),
			"rejected inert-field update mutated state")
	}
}

// ==================== Founder Withdraw-Address Routing Tests ====================

func TestDepositToResearchFund_NoFounderWithdrawAddressRouting(t *testing.T) {
	bk := newMockBankKeeper()
	founderAddr := sdk.AccAddress("founder_____________")
	withdrawAddr := sdk.AccAddress("founder_withdraw____")
	k, ctx := setupFounderKeeper(t, bk, founderAddr.String(), 0)
	k.SetDistributionKeeper(&mockDistrKeeper{withdrawAddrs: map[string]sdk.AccAddress{
		founderAddr.String(): withdrawAddr,
	}})

	depositCoins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewInt(100000)))
	if err := k.DepositToResearchFund(ctx, "billing", depositCoins); err != nil {
		t.Fatalf("DepositToResearchFund failed: %v", err)
	}

	if got := bk.sentToAccount[withdrawAddr.String()].AmountOf("uzrn").Int64(); got != 0 {
		t.Errorf("expected 0 uzrn at withdraw address, got %d", got)
	}
	if _, ok := bk.sentToAccount[founderAddr.String()]; ok {
		t.Errorf("founder share paid to FounderAddress directly despite withdraw mapping: %v", bk.sentToAccount[founderAddr.String()])
	}
}

func TestDepositToResearchFund_NoFounderDefaultRouting(t *testing.T) {
	bk := newMockBankKeeper()
	founderAddr := sdk.AccAddress("founder_____________")
	k, ctx := setupFounderKeeper(t, bk, founderAddr.String(), 0)
	k.SetDistributionKeeper(&mockDistrKeeper{})

	depositCoins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewInt(100000)))
	if err := k.DepositToResearchFund(ctx, "billing", depositCoins); err != nil {
		t.Fatalf("DepositToResearchFund failed: %v", err)
	}

	if got := bk.sentToAccount[founderAddr.String()].AmountOf("uzrn").Int64(); got != 0 {
		t.Errorf("expected 0 uzrn at founder address, got %d", got)
	}
}

// ==================== Proposer Reward Resolution Tests ====================

func TestResolveProposerRewardAddress_OperatorNotConsensus(t *testing.T) {
	consAddr1 := sdk.ConsAddress("consaddr1___________")
	consAddr2 := sdk.ConsAddress("consaddr2___________")
	op1 := sdk.ValAddress("operator1___________")
	op2 := sdk.ValAddress("operator2___________")

	sk := &mockStakingKeeper{
		activeCount: 22,
		validators: map[string]stakingtypes.Validator{
			consAddr1.String(): {OperatorAddress: op1.String()},
			consAddr2.String(): {OperatorAddress: op2.String()},
		},
	}
	k, ctx := setupKeeperWithBank(t, newMockBankKeeper(), sk)

	tests := []struct {
		name   string
		cons   sdk.ConsAddress
		wantOp sdk.ValAddress
	}{
		{"validator 1", consAddr1, op1},
		{"validator 2", consAddr2, op2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := k.ResolveProposerRewardAddress(ctx, tt.cons)
			if err != nil {
				t.Fatalf("ResolveProposerRewardAddress failed: %v", err)
			}
			want := sdk.AccAddress(tt.wantOp)
			if !got.Equals(want) {
				t.Errorf("expected operator account %s, got %s", want, got)
			}
			if got.Equals(sdk.AccAddress(tt.cons)) {
				t.Error("reward resolved to the unspendable consensus address")
			}
		})
	}
}

func TestResolveProposerRewardAddress_WithdrawAddressHonored(t *testing.T) {
	consAddr := sdk.ConsAddress("consaddr1___________")
	op := sdk.ValAddress("operator1___________")
	withdrawAddr := sdk.AccAddress("operator_withdraw___")

	sk := &mockStakingKeeper{
		activeCount: 22,
		validators: map[string]stakingtypes.Validator{
			consAddr.String(): {OperatorAddress: op.String()},
		},
	}
	k, ctx := setupKeeperWithBank(t, newMockBankKeeper(), sk)
	k.SetDistributionKeeper(&mockDistrKeeper{withdrawAddrs: map[string]sdk.AccAddress{
		sdk.AccAddress(op).String(): withdrawAddr,
	}})

	got, err := k.ResolveProposerRewardAddress(ctx, consAddr)
	if err != nil {
		t.Fatalf("ResolveProposerRewardAddress failed: %v", err)
	}
	if !got.Equals(withdrawAddr) {
		t.Errorf("expected withdraw address %s, got %s", withdrawAddr, got)
	}
}

func TestResolveProposerRewardAddress_UnknownProposer(t *testing.T) {
	sk := &mockStakingKeeper{activeCount: 22}
	k, ctx := setupKeeperWithBank(t, newMockBankKeeper(), sk)

	if _, err := k.ResolveProposerRewardAddress(ctx, sdk.ConsAddress("unknown_cons________")); err == nil {
		t.Fatal("expected error for unknown consensus address")
	}
}

func TestResolveProposerRewardAddress_NoStakingKeeper(t *testing.T) {
	k, ctx := setupKeeper(t) // nil staking keeper

	if _, err := k.ResolveProposerRewardAddress(ctx, sdk.ConsAddress("consaddr1___________")); err == nil {
		t.Fatal("expected error when staking keeper is not wired")
	}
}

// ==================== BeginBlock Proposer Remap Tests ====================

func TestBeginBlock_DoesNotMintForTransactionPresence(t *testing.T) {
	consAddr1 := sdk.ConsAddress("consaddr1___________")
	consAddr2 := sdk.ConsAddress("consaddr2___________")
	op1 := sdk.ValAddress("operator1___________")
	op2 := sdk.ValAddress("operator2___________")

	bk := newMockBankKeeper()
	sk := &mockStakingKeeper{
		activeCount: 22,
		validators: map[string]stakingtypes.Validator{
			consAddr1.String(): {OperatorAddress: op1.String()},
			consAddr2.String(): {OperatorAddress: op2.String()},
		},
	}
	k, ctx := setupKeeperWithBank(t, bk, sk)

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	am := vestingrewards.NewAppModule(cdc, k)

	blocks := []sdk.ConsAddress{
		consAddr1,
		consAddr2,
	}
	for _, cons := range blocks {
		blockCtx := ctx.WithBlockHeader(cmtproto.Header{Height: 1000, ProposerAddress: cons})
		if err := am.BeginBlock(blockCtx); err != nil {
			t.Fatalf("BeginBlock failed: %v", err)
		}
	}

	if !bk.mintedCoins.IsZero() {
		t.Fatalf("transaction presence must not mint coins, minted %v", bk.mintedCoins)
	}
	if len(bk.sentToAccount) != 0 {
		t.Fatalf("transaction presence must not pay proposer or operator accounts, got %v", bk.sentToAccount)
	}
}

func TestBeginBlock_UnresolvableProposerSkipsReward(t *testing.T) {
	// Better to skip a block's emission than to mint coins nobody can spend.
	bk := newMockBankKeeper()
	sk := &mockStakingKeeper{activeCount: 22} // no validators registered
	k, ctx := setupKeeperWithBank(t, bk, sk)

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	am := vestingrewards.NewAppModule(cdc, k)
	blockCtx := ctx.WithBlockHeader(cmtproto.Header{Height: 1000, ProposerAddress: sdk.ConsAddress("unknown_cons________")})
	if err := am.BeginBlock(blockCtx); err != nil {
		t.Fatalf("BeginBlock should skip, not fail: %v", err)
	}

	if !bk.mintedCoins.IsZero() {
		t.Errorf("expected no minting for unresolvable proposer, minted %v", bk.mintedCoins)
	}
	if len(bk.sentToAccount) != 0 {
		t.Errorf("expected no account payouts, got %v", bk.sentToAccount)
	}
}

func TestFounderShareRetiredAtEveryHeight(t *testing.T) {
	// Directly injected legacy fields remain inert at every height. Consensus
	// v2 additionally clears them during migration and refuses reactivation.
	bk := newMockBankKeeper()
	founderAddr := sdk.AccAddress("founder_____________").String()

	// Set GovernanceActivationHeight to 500 — should be ignored.
	k, ctx := setupFounderKeeper(t, bk, founderAddr, 500)

	// Test at block 1 (before activation height)
	ctx1 := ctx.WithBlockHeight(1)
	producer := sdk.AccAddress("producer____________").String()
	dist1, err := k.DistributeBlockReward(ctx1, producer, 22, true)
	if err != nil {
		t.Fatalf("block 1 reward failed: %v", err)
	}
	if dist1.FounderShare != "0" {
		t.Errorf("expected founder share retired at block 1, got %s", dist1.FounderShare)
	}

	// Test at block 10000 (well after activation height)
	bk2 := newMockBankKeeper()
	k2, ctx2 := setupFounderKeeper(t, bk2, founderAddr, 500)
	ctx2 = ctx2.WithBlockHeight(10000)
	dist2, err := k2.DistributeBlockReward(ctx2, producer, 22, true)
	if err != nil {
		t.Fatalf("block 10000 reward failed: %v", err)
	}
	if dist2.FounderShare != "0" {
		t.Errorf("expected founder share retired at block 10000, got %s", dist2.FounderShare)
	}

	// Both should yield the same founder share amount
	if dist1.FounderShare != dist2.FounderShare {
		t.Errorf("founder share should be consistent regardless of block height: block1=%s, block10000=%s",
			dist1.FounderShare, dist2.FounderShare)
	}
}

// ==================== New Query Tests ====================

func TestQueryResearchFundBalance(t *testing.T) {
	bk := newMockBankKeeper()
	sk := &mockStakingKeeper{activeCount: 22}
	k, ctx := setupKeeperWithBank(t, bk, sk)

	qs := keeper.NewQueryServerImpl(k)

	resp, err := qs.ResearchFundBalance(ctx, &types.QueryResearchFundBalanceRequest{})
	if err != nil {
		t.Fatalf("ResearchFundBalance query failed: %v", err)
	}
	if resp.Balance != "0" {
		t.Errorf("expected balance 0, got %s", resp.Balance)
	}
	if resp.Denom != "uzrn" {
		t.Errorf("expected denom uzrn, got %s", resp.Denom)
	}
}

func TestQueryFounderShareStatus_LegacyFieldsRemainInactive(t *testing.T) {
	bk := newMockBankKeeper()
	founderAddr := sdk.AccAddress("founder_____________").String()
	k, ctx := setupFounderKeeper(t, bk, founderAddr, 0)

	qs := keeper.NewQueryServerImpl(k)

	resp, err := qs.FounderShareStatus(ctx, &types.QueryFounderShareStatusRequest{})
	if err != nil {
		t.Fatalf("FounderShareStatus query failed: %v", err)
	}
	if resp.Active {
		t.Error("retired founder share must remain inactive")
	}
	if resp.FounderShareBps != 70000 {
		t.Errorf("expected 70000 bps, got %d", resp.FounderShareBps)
	}
	if resp.FounderAddress != founderAddr {
		t.Errorf("expected founder address %s, got %s", founderAddr, resp.FounderAddress)
	}
}

func TestQueryFounderShareStatus_Inactive(t *testing.T) {
	k, ctx := setupKeeper(t)

	qs := keeper.NewQueryServerImpl(k)

	resp, err := qs.FounderShareStatus(ctx, &types.QueryFounderShareStatusRequest{})
	if err != nil {
		t.Fatalf("FounderShareStatus query failed: %v", err)
	}
	if resp.Active {
		t.Error("expected founder share to be inactive")
	}
	if resp.FounderShareBps != 0 || resp.FounderAddress != "" {
		t.Fatalf("retired compatibility fields are not zero/empty: %+v", resp)
	}
}

// ─── SupplyCouplingAudit (L0 thesis metric) ──────────────────────────────

// stubKnowledgeKeeper exposes independently configurable legacy acceptance and
// survived-challenge rates plus disproven fact IDs.
type stubKnowledgeKeeper struct {
	rate         uint64
	survivedRate uint64
	disproven    map[string]bool
}

func (s *stubKnowledgeKeeper) GetVerificationRate(_ context.Context) uint64 { return s.rate }

func (s *stubKnowledgeKeeper) GetSurvivedChallengeRate(_ context.Context) uint64 {
	return s.survivedRate
}

func (s *stubKnowledgeKeeper) IsFactDisproven(_ context.Context, factID string) bool {
	return s.disproven[factID]
}

func TestQuerySupplyCouplingAudit_NilKnowledgeKeeper(t *testing.T) {
	bk := newMockBankKeeper()
	sk := &mockStakingKeeper{activeCount: 22}
	k, ctx := setupKeeperWithBank(t, bk, sk)

	qs := keeper.NewQueryServerImpl(k)

	resp, err := qs.SupplyCouplingAudit(ctx, &types.QuerySupplyCouplingAuditRequest{})
	if err != nil {
		t.Fatalf("SupplyCouplingAudit failed: %v", err)
	}
	if resp.CouplingEnabled {
		t.Error("coupling must be disabled when knowledge keeper is nil")
	}
	if resp.EffectiveCouplingMultiplierBps != 0 {
		t.Errorf("retired coupling multiplier must be zero, got %d", resp.EffectiveCouplingMultiplierBps)
	}
	if resp.MaxSupply != types.MaxSupplyUzrn {
		t.Errorf("expected max supply %s, got %s", types.MaxSupplyUzrn, resp.MaxSupply)
	}
}

func TestQuerySupplyCouplingAudit_RateAboveTarget(t *testing.T) {
	bk := newMockBankKeeper()
	sk := &mockStakingKeeper{activeCount: 22}
	k, ctx := setupKeeperWithBank(t, bk, sk)

	k.SetKnowledgeKeeper(&stubKnowledgeKeeper{rate: 850_000, survivedRate: 850_000}) // above 70% target
	qs := keeper.NewQueryServerImpl(k)

	resp, err := qs.SupplyCouplingAudit(ctx, &types.QuerySupplyCouplingAuditRequest{})
	if err != nil {
		t.Fatalf("SupplyCouplingAudit failed: %v", err)
	}
	if resp.CouplingEnabled {
		t.Error("retired coupling must remain disabled when telemetry is wired")
	}
	if resp.EffectiveCouplingMultiplierBps != 0 {
		t.Errorf("retired coupling multiplier must be zero, got %d", resp.EffectiveCouplingMultiplierBps)
	}
	if resp.VerificationRateBps != 850_000 {
		t.Errorf("expected verification rate 850000, got %d", resp.VerificationRateBps)
	}
}

func TestQuerySupplyCouplingAudit_RateBelowTarget(t *testing.T) {
	bk := newMockBankKeeper()
	sk := &mockStakingKeeper{activeCount: 22}
	k, ctx := setupKeeperWithBank(t, bk, sk)

	// Legacy rate telemetry remains visible but no multiplier is applied.
	k.SetKnowledgeKeeper(&stubKnowledgeKeeper{rate: 350_000, survivedRate: 350_000})
	qs := keeper.NewQueryServerImpl(k)

	resp, err := qs.SupplyCouplingAudit(ctx, &types.QuerySupplyCouplingAuditRequest{})
	if err != nil {
		t.Fatalf("SupplyCouplingAudit failed: %v", err)
	}
	if resp.CouplingEnabled || resp.EffectiveCouplingMultiplierBps != 0 {
		t.Errorf("legacy telemetry reactivated coupling: %+v", resp)
	}
}

func TestQuerySupplyCouplingAudit_UsesSurvivalRateForAppliedMultiplier(t *testing.T) {
	bk := newMockBankKeeper()
	k, ctx := setupKeeperWithBank(t, bk, &mockStakingKeeper{activeCount: 22})
	k.SetKnowledgeKeeper(&stubKnowledgeKeeper{
		rate:         850_000,
		survivedRate: 350_000,
	})

	resp, err := keeper.NewQueryServerImpl(k).SupplyCouplingAudit(
		ctx,
		&types.QuerySupplyCouplingAuditRequest{},
	)
	if err != nil {
		t.Fatalf("SupplyCouplingAudit failed: %v", err)
	}
	if resp.VerificationRateBps != 850_000 {
		t.Fatalf("legacy verification rate = %d, want 850000", resp.VerificationRateBps)
	}
	if resp.SurvivedChallengeRateBps != 350_000 {
		t.Fatalf("survival rate = %d, want 350000", resp.SurvivedChallengeRateBps)
	}
	if resp.CouplingEnabled || resp.EffectiveCouplingMultiplierBps != 0 {
		t.Fatalf("observed survival rate reactivated retired coupling: %+v", resp)
	}
}

// ==================== Custom RevenueSplit Params Test ====================

func TestDistributeRevenue_CustomSplit(t *testing.T) {
	// Test with a custom governance-adjusted split: 40/30/20/10
	gs := types.DefaultGenesis()
	gs.Params.RevenueSplit = &commontypes.RevenueSplit{
		ContributorBps: 400000,
		ProtocolBps:    300000,
		ResearchBps:    200000,
		DevelopmentBps: 100000,
	}
	gs.Params.ProtocolSubSplit = &commontypes.ProtocolSubSplit{
		CitationBps:     600000,
		VerificationBps: 300000,
		TreasuryBps:     100000,
	}

	bk := newMockBankKeeper()
	k, ctx := setupKeeperWithBankAndGenesis(t, bk, &mockStakingKeeper{activeCount: 22}, gs)

	routing, err := k.DistributeRevenue(ctx, types.SourceBlockProduction, "1000000",
		sdk.AccAddress("recipient___________").String(), "")
	if err != nil {
		t.Fatalf("distribute revenue failed: %v", err)
	}

	if routing.ContributorShare != "400000" {
		t.Errorf("expected contributor 400000, got %s", routing.ContributorShare)
	}
	if routing.ProtocolShare != "300000" {
		t.Errorf("expected protocol 300000, got %s", routing.ProtocolShare)
	}
	if routing.ResearchShare != "200000" {
		t.Errorf("expected research 200000, got %s", routing.ResearchShare)
	}
	if routing.DevelopmentAmount != "100000" {
		t.Errorf("expected development 100000, got %s", routing.DevelopmentAmount)
	}
	if routing.CitationPool != "180000" {
		t.Errorf("expected citation pool 180000, got %s", routing.CitationPool)
	}
	if routing.VerificationPool != "90000" {
		t.Errorf("expected verification pool 90000, got %s", routing.VerificationPool)
	}
	if routing.TreasuryShare != "30000" {
		t.Errorf("expected treasury 30000, got %s", routing.TreasuryShare)
	}
}
