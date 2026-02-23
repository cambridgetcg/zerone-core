package keeper_test

import (
	"context"
	"fmt"
	"testing"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/schedule/keeper"
	"github.com/zerone-chain/zerone/x/schedule/types"
)

// -----------------------------------------------------------------------
// Mock BankKeeper
// -----------------------------------------------------------------------

type mockBankKeeper struct {
	balances map[string]sdkmath.Int
}

func newMockBankKeeper() *mockBankKeeper {
	return &mockBankKeeper{
		balances: make(map[string]sdkmath.Int),
	}
}

func (m *mockBankKeeper) setBalance(addr sdk.AccAddress, denom string, amount sdkmath.Int) {
	m.balances[addr.String()+"/"+denom] = amount
}

func (m *mockBankKeeper) SendCoins(_ context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error {
	for _, coin := range amt {
		fromKey := fromAddr.String() + "/" + coin.Denom
		toKey := toAddr.String() + "/" + coin.Denom

		fromBal, ok := m.balances[fromKey]
		if !ok {
			fromBal = sdkmath.ZeroInt()
		}
		if fromBal.LT(coin.Amount) {
			return fmt.Errorf("insufficient balance")
		}
		m.balances[fromKey] = fromBal.Sub(coin.Amount)

		toBal, ok := m.balances[toKey]
		if !ok {
			toBal = sdkmath.ZeroInt()
		}
		m.balances[toKey] = toBal.Add(coin.Amount)
	}
	return nil
}

func (m *mockBankKeeper) SendCoinsFromAccountToModule(_ context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	for _, coin := range amt {
		key := senderAddr.String() + "/" + coin.Denom
		bal, ok := m.balances[key]
		if !ok {
			bal = sdkmath.ZeroInt()
		}
		if bal.LT(coin.Amount) {
			return fmt.Errorf("insufficient balance for send to module")
		}
		m.balances[key] = bal.Sub(coin.Amount)
		modKey := recipientModule + "/" + coin.Denom
		modBal, ok := m.balances[modKey]
		if !ok {
			modBal = sdkmath.ZeroInt()
		}
		m.balances[modKey] = modBal.Add(coin.Amount)
	}
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	for _, coin := range amt {
		modKey := senderModule + "/" + coin.Denom
		modBal, ok := m.balances[modKey]
		if !ok {
			modBal = sdkmath.ZeroInt()
		}
		if modBal.LT(coin.Amount) {
			return fmt.Errorf("insufficient module balance")
		}
		m.balances[modKey] = modBal.Sub(coin.Amount)

		key := recipientAddr.String() + "/" + coin.Denom
		bal, ok := m.balances[key]
		if !ok {
			bal = sdkmath.ZeroInt()
		}
		m.balances[key] = bal.Add(coin.Amount)
	}
	return nil
}

// -----------------------------------------------------------------------
// Test Setup
// -----------------------------------------------------------------------

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("zrn", "zrnpub")
	config.SetBech32PrefixForValidator("zrnvaloper", "zrnvaloperpub")
	config.SetBech32PrefixForConsensusNode("zrnvalcons", "zrnvalconspub")
}

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context, *mockBankKeeper) {
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

	bk := newMockBankKeeper()

	authority := sdk.AccAddress([]byte("authority-addr------")).String()
	k := keeper.NewKeeper(runtime.NewKVStoreService(storeKey), cdc, authority, bk)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 100}, false, log.NewNopLogger())

	// Set default params
	k.SetParams(ctx, types.DefaultParams())

	return k, ctx, bk
}

func testAddr(i int) string {
	addr := sdk.AccAddress([]byte(fmt.Sprintf("test-addr-%010d", i)))
	return addr.String()
}

// -----------------------------------------------------------------------
// Tests: Params
// -----------------------------------------------------------------------

func TestSetGetParams(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Verify defaults
	params := k.GetParams(ctx)
	if params.MaxActivePerAccount != 20 {
		t.Errorf("expected MaxActivePerAccount 20, got %d", params.MaxActivePerAccount)
	}
	if params.MaxGasPerBlock != 50000000 {
		t.Errorf("expected MaxGasPerBlock 50000000, got %d", params.MaxGasPerBlock)
	}
	if params.MinIntervalBlocks != 10 {
		t.Errorf("expected MinIntervalBlocks 10, got %d", params.MinIntervalBlocks)
	}
	if params.MinFeePerExecution != "10000" {
		t.Errorf("expected MinFeePerExecution 10000, got %s", params.MinFeePerExecution)
	}
	if params.MaxCompoundDepth != 3 {
		t.Errorf("expected MaxCompoundDepth 3, got %d", params.MaxCompoundDepth)
	}

	// Set custom params and verify persistence
	custom := &types.Params{
		MaxActivePerAccount: 50,
		MaxGasPerBlock:      100000000,
		MinIntervalBlocks:   20,
		MinFeePerExecution:  "50000",
		MaxCompoundDepth:    5,
	}
	k.SetParams(ctx, custom)

	got := k.GetParams(ctx)
	if got.MaxActivePerAccount != 50 {
		t.Errorf("expected MaxActivePerAccount 50, got %d", got.MaxActivePerAccount)
	}
	if got.MaxGasPerBlock != 100000000 {
		t.Errorf("expected MaxGasPerBlock 100000000, got %d", got.MaxGasPerBlock)
	}
	if got.MinIntervalBlocks != 20 {
		t.Errorf("expected MinIntervalBlocks 20, got %d", got.MinIntervalBlocks)
	}
	if got.MinFeePerExecution != "50000" {
		t.Errorf("expected MinFeePerExecution 50000, got %s", got.MinFeePerExecution)
	}
	if got.MaxCompoundDepth != 5 {
		t.Errorf("expected MaxCompoundDepth 5, got %d", got.MaxCompoundDepth)
	}
}

// -----------------------------------------------------------------------
// Tests: Sequence
// -----------------------------------------------------------------------

func TestSequence(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Initial sequence should be 0
	seq := k.GetSequence(ctx)
	if seq != 0 {
		t.Errorf("expected initial sequence 0, got %d", seq)
	}

	// NextSequence should increment
	next1 := k.NextSequence(ctx)
	if next1 != 1 {
		t.Errorf("expected first NextSequence 1, got %d", next1)
	}

	next2 := k.NextSequence(ctx)
	if next2 != 2 {
		t.Errorf("expected second NextSequence 2, got %d", next2)
	}

	// GetSequence should reflect the last value
	seq = k.GetSequence(ctx)
	if seq != 2 {
		t.Errorf("expected sequence 2 after two increments, got %d", seq)
	}

	// SetSequence should overwrite
	k.SetSequence(ctx, 100)
	seq = k.GetSequence(ctx)
	if seq != 100 {
		t.Errorf("expected sequence 100 after set, got %d", seq)
	}

	next3 := k.NextSequence(ctx)
	if next3 != 101 {
		t.Errorf("expected NextSequence 101 after set to 100, got %d", next3)
	}
}

// -----------------------------------------------------------------------
// Tests: Process CRUD
// -----------------------------------------------------------------------

func TestSetGetProcess(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Process should not exist initially
	_, found := k.GetProcess(ctx, "sched-1")
	if found {
		t.Error("expected process not found")
	}

	// Create and store a process
	process := &types.ScheduleProcess{
		Id:              "sched-1",
		Creator:         testAddr(1),
		Status:          "active",
		ExecutionCount:  0,
		MaxExecutions:   10,
		RemainingFee:    "1000000",
		FeePerExecution: "100000",
		NextExecuteAt:   200,
		CreatedAtBlock:  100,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           "every_n_blocks",
				IntervalBlocks: 50,
			},
		},
	}
	k.SetProcess(ctx, process)

	// Retrieve and verify
	got, found := k.GetProcess(ctx, "sched-1")
	if !found {
		t.Fatal("expected process to be found")
	}
	if got.Id != "sched-1" {
		t.Errorf("expected id sched-1, got %s", got.Id)
	}
	if got.Creator != testAddr(1) {
		t.Errorf("expected creator %s, got %s", testAddr(1), got.Creator)
	}
	if got.Status != "active" {
		t.Errorf("expected status active, got %s", got.Status)
	}
	if got.RemainingFee != "1000000" {
		t.Errorf("expected remaining_fee 1000000, got %s", got.RemainingFee)
	}
	if got.FeePerExecution != "100000" {
		t.Errorf("expected fee_per_execution 100000, got %s", got.FeePerExecution)
	}
	if got.NextExecuteAt != 200 {
		t.Errorf("expected next_execute_at 200, got %d", got.NextExecuteAt)
	}
	if got.MaxExecutions != 10 {
		t.Errorf("expected max_executions 10, got %d", got.MaxExecutions)
	}

	// Update process
	got.Status = "paused"
	got.NextExecuteAt = 0
	k.SetProcess(ctx, got)

	updated, found := k.GetProcess(ctx, "sched-1")
	if !found {
		t.Fatal("expected updated process to be found")
	}
	if updated.Status != "paused" {
		t.Errorf("expected status paused, got %s", updated.Status)
	}
	if updated.NextExecuteAt != 0 {
		t.Errorf("expected next_execute_at 0, got %d", updated.NextExecuteAt)
	}

	// Delete process
	k.DeleteProcess(ctx, updated)
	_, found = k.GetProcess(ctx, "sched-1")
	if found {
		t.Error("expected process to be deleted")
	}
}

// -----------------------------------------------------------------------
// Tests: GetProcessesByCreator
// -----------------------------------------------------------------------

func TestGetProcessesByCreator(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	creator1 := testAddr(1)
	creator2 := testAddr(2)

	// Store processes for two different creators
	for i := 0; i < 3; i++ {
		k.SetProcess(ctx, &types.ScheduleProcess{
			Id:              fmt.Sprintf("sched-c1-%d", i),
			Creator:         creator1,
			Status:          "active",
			RemainingFee:    "1000000",
			FeePerExecution: "100000",
			NextExecuteAt:   uint64(200 + i*50),
			Condition: &types.ScheduleCondition{
				TimeCondition: &types.TimeCondition{
					Type:           "every_n_blocks",
					IntervalBlocks: 50,
				},
			},
		})
	}

	k.SetProcess(ctx, &types.ScheduleProcess{
		Id:              "sched-c2-0",
		Creator:         creator2,
		Status:          "active",
		RemainingFee:    "500000",
		FeePerExecution: "50000",
		NextExecuteAt:   300,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           "every_n_blocks",
				IntervalBlocks: 100,
			},
		},
	})

	// Query by creator1
	c1Processes := k.GetProcessesByCreator(ctx, creator1)
	if len(c1Processes) != 3 {
		t.Errorf("expected 3 processes for creator1, got %d", len(c1Processes))
	}

	// Query by creator2
	c2Processes := k.GetProcessesByCreator(ctx, creator2)
	if len(c2Processes) != 1 {
		t.Errorf("expected 1 process for creator2, got %d", len(c2Processes))
	}

	// Query by nonexistent creator
	c3Processes := k.GetProcessesByCreator(ctx, testAddr(99))
	if len(c3Processes) != 0 {
		t.Errorf("expected 0 processes for unknown creator, got %d", len(c3Processes))
	}

	// Verify CountActiveByCreator
	activeCount := k.CountActiveByCreator(ctx, creator1)
	if activeCount != 3 {
		t.Errorf("expected 3 active for creator1, got %d", activeCount)
	}
}

// -----------------------------------------------------------------------
// Tests: CreateScheduleHandler
// -----------------------------------------------------------------------

func TestCreateScheduleHandler(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	creator := testAddr(1)
	creatorAddr, _ := sdk.AccAddressFromBech32(creator)
	bk.setBalance(creatorAddr, "uzrn", sdkmath.NewInt(10000000))

	srv := keeper.NewMsgServerImpl(k)
	resp, err := srv.CreateSchedule(ctx, &types.MsgCreateSchedule{
		Creator: creator,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           "every_n_blocks",
				IntervalBlocks: 50,
			},
		},
		MaxExecutions:   5,
		FeePerExecution: "100000",
		PrepaidFee:      "500000",
		TargetAddress:   testAddr(2),
		CallData:        "invoke:something",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.ProcessId == "" {
		t.Error("expected non-empty process ID")
	}

	// Verify the process was stored
	process, found := k.GetProcess(ctx, resp.ProcessId)
	if !found {
		t.Fatal("expected process to be found after creation")
	}
	if process.Status != "active" {
		t.Errorf("expected status active, got %s", process.Status)
	}
	if process.Creator != creator {
		t.Errorf("expected creator %s, got %s", creator, process.Creator)
	}
	if process.FeePerExecution != "100000" {
		t.Errorf("expected fee_per_execution 100000, got %s", process.FeePerExecution)
	}
	if process.RemainingFee != "500000" {
		t.Errorf("expected remaining_fee 500000, got %s", process.RemainingFee)
	}
	if process.MaxExecutions != 5 {
		t.Errorf("expected max_executions 5, got %d", process.MaxExecutions)
	}
	if process.TargetAddress != testAddr(2) {
		t.Errorf("expected target_address %s, got %s", testAddr(2), process.TargetAddress)
	}
	if process.CallData != "invoke:something" {
		t.Errorf("expected call_data invoke:something, got %s", process.CallData)
	}
	// NextExecuteAt should be current height (100) + interval (50) = 150
	if process.NextExecuteAt != 150 {
		t.Errorf("expected next_execute_at 150, got %d", process.NextExecuteAt)
	}
	if process.CreatedAtBlock != 100 {
		t.Errorf("expected created_at_block 100, got %d", process.CreatedAtBlock)
	}

	// Verify fee was deducted from creator
	creatorBal := bk.balances[creatorAddr.String()+"/uzrn"]
	expectedBal := sdkmath.NewInt(9500000) // 10000000 - 500000
	if !creatorBal.Equal(expectedBal) {
		t.Errorf("expected creator balance %s, got %s", expectedBal, creatorBal)
	}

	// Verify fee was deposited into module account
	modBal := bk.balances[types.ModuleName+"/uzrn"]
	expectedModBal := sdkmath.NewInt(500000)
	if !modBal.Equal(expectedModBal) {
		t.Errorf("expected module balance %s, got %s", expectedModBal, modBal)
	}
}

func TestCreateScheduleHandler_AtBlock(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	creator := testAddr(1)
	creatorAddr, _ := sdk.AccAddressFromBech32(creator)
	bk.setBalance(creatorAddr, "uzrn", sdkmath.NewInt(10000000))

	srv := keeper.NewMsgServerImpl(k)
	resp, err := srv.CreateSchedule(ctx, &types.MsgCreateSchedule{
		Creator: creator,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           "at_block",
				ExecuteAtBlock: 200,
			},
		},
		FeePerExecution: "100000",
		PrepaidFee:      "100000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	process, found := k.GetProcess(ctx, resp.ProcessId)
	if !found {
		t.Fatal("expected process to be found")
	}
	// For at_block, NextExecuteAt should be the target block
	if process.NextExecuteAt != 200 {
		t.Errorf("expected next_execute_at 200, got %d", process.NextExecuteAt)
	}
}

func TestCreateScheduleHandler_InsufficientBalance(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	creator := testAddr(1)
	creatorAddr, _ := sdk.AccAddressFromBech32(creator)
	bk.setBalance(creatorAddr, "uzrn", sdkmath.NewInt(100)) // very low balance

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.CreateSchedule(ctx, &types.MsgCreateSchedule{
		Creator: creator,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           "every_n_blocks",
				IntervalBlocks: 50,
			},
		},
		FeePerExecution: "100000",
		PrepaidFee:      "500000",
	})
	if err == nil {
		t.Error("expected insufficient balance error")
	}
}

func TestCreateScheduleHandler_FeeBelowMinimum(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	creator := testAddr(1)
	creatorAddr, _ := sdk.AccAddressFromBech32(creator)
	bk.setBalance(creatorAddr, "uzrn", sdkmath.NewInt(10000000))

	srv := keeper.NewMsgServerImpl(k)
	// MinFeePerExecution in default params is 10000; submit 100 which is below minimum
	_, err := srv.CreateSchedule(ctx, &types.MsgCreateSchedule{
		Creator: creator,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           "every_n_blocks",
				IntervalBlocks: 50,
			},
		},
		FeePerExecution: "100",
		PrepaidFee:      "500000",
	})
	if err == nil {
		t.Error("expected insufficient fee error for fee below minimum")
	}
}

func TestCreateScheduleHandler_NilCondition(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	creator := testAddr(1)
	creatorAddr, _ := sdk.AccAddressFromBech32(creator)
	bk.setBalance(creatorAddr, "uzrn", sdkmath.NewInt(10000000))

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.CreateSchedule(ctx, &types.MsgCreateSchedule{
		Creator:         creator,
		Condition:       nil,
		FeePerExecution: "100000",
		PrepaidFee:      "500000",
	})
	if err == nil {
		t.Error("expected error for nil condition")
	}
}

func TestCreateScheduleHandler_IntervalBelowMinimum(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	creator := testAddr(1)
	creatorAddr, _ := sdk.AccAddressFromBech32(creator)
	bk.setBalance(creatorAddr, "uzrn", sdkmath.NewInt(10000000))

	srv := keeper.NewMsgServerImpl(k)
	// MinIntervalBlocks is 10; submit 5 which is below minimum
	_, err := srv.CreateSchedule(ctx, &types.MsgCreateSchedule{
		Creator: creator,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           "every_n_blocks",
				IntervalBlocks: 5,
			},
		},
		FeePerExecution: "100000",
		PrepaidFee:      "500000",
	})
	if err == nil {
		t.Error("expected error for interval below minimum")
	}
}

// -----------------------------------------------------------------------
// Tests: PauseResumeSchedule
// -----------------------------------------------------------------------

func TestPauseResumeSchedule(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	creator := testAddr(1)
	creatorAddr, _ := sdk.AccAddressFromBech32(creator)
	bk.setBalance(creatorAddr, "uzrn", sdkmath.NewInt(10000000))

	srv := keeper.NewMsgServerImpl(k)

	// Create a schedule
	createResp, err := srv.CreateSchedule(ctx, &types.MsgCreateSchedule{
		Creator: creator,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           "every_n_blocks",
				IntervalBlocks: 50,
			},
		},
		FeePerExecution: "100000",
		PrepaidFee:      "500000",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	processId := createResp.ProcessId

	// Verify it is active
	process, _ := k.GetProcess(ctx, processId)
	if process.Status != "active" {
		t.Errorf("expected status active, got %s", process.Status)
	}
	if process.NextExecuteAt == 0 {
		t.Error("expected non-zero next_execute_at for active schedule")
	}

	// Pause
	_, err = srv.PauseSchedule(ctx, &types.MsgPauseSchedule{
		Creator:   creator,
		ProcessId: processId,
	})
	if err != nil {
		t.Fatalf("pause failed: %v", err)
	}

	process, _ = k.GetProcess(ctx, processId)
	if process.Status != "paused" {
		t.Errorf("expected status paused, got %s", process.Status)
	}
	if process.NextExecuteAt != 0 {
		t.Errorf("expected next_execute_at 0 for paused schedule, got %d", process.NextExecuteAt)
	}

	// Pausing an already paused schedule should fail
	_, err = srv.PauseSchedule(ctx, &types.MsgPauseSchedule{
		Creator:   creator,
		ProcessId: processId,
	})
	if err == nil {
		t.Error("expected error when pausing already-paused schedule")
	}

	// Resume
	_, err = srv.ResumeSchedule(ctx, &types.MsgResumeSchedule{
		Creator:   creator,
		ProcessId: processId,
	})
	if err != nil {
		t.Fatalf("resume failed: %v", err)
	}

	process, _ = k.GetProcess(ctx, processId)
	if process.Status != "active" {
		t.Errorf("expected status active after resume, got %s", process.Status)
	}
	if process.NextExecuteAt == 0 {
		t.Error("expected non-zero next_execute_at after resume")
	}

	// Resuming an active schedule should fail
	_, err = srv.ResumeSchedule(ctx, &types.MsgResumeSchedule{
		Creator:   creator,
		ProcessId: processId,
	})
	if err == nil {
		t.Error("expected error when resuming active schedule")
	}
}

// -----------------------------------------------------------------------
// Tests: CancelSchedule
// -----------------------------------------------------------------------

func TestCancelSchedule(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	creator := testAddr(1)
	creatorAddr, _ := sdk.AccAddressFromBech32(creator)
	bk.setBalance(creatorAddr, "uzrn", sdkmath.NewInt(10000000))

	srv := keeper.NewMsgServerImpl(k)

	// Create a schedule
	createResp, err := srv.CreateSchedule(ctx, &types.MsgCreateSchedule{
		Creator: creator,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           "every_n_blocks",
				IntervalBlocks: 50,
			},
		},
		FeePerExecution: "100000",
		PrepaidFee:      "500000",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	processId := createResp.ProcessId

	// Record balance after creation (prepaid fee deducted)
	balAfterCreate := bk.balances[creatorAddr.String()+"/uzrn"]

	// Cancel
	cancelResp, err := srv.CancelSchedule(ctx, &types.MsgCancelSchedule{
		Creator:   creator,
		ProcessId: processId,
	})
	if err != nil {
		t.Fatalf("cancel failed: %v", err)
	}

	// Verify refund amount
	if cancelResp.RefundedAmount != "500000" {
		t.Errorf("expected refunded_amount 500000, got %s", cancelResp.RefundedAmount)
	}

	// Verify process is cancelled
	process, found := k.GetProcess(ctx, processId)
	if !found {
		t.Fatal("expected process to still exist after cancel")
	}
	if process.Status != "cancelled" {
		t.Errorf("expected status cancelled, got %s", process.Status)
	}
	if process.RemainingFee != "0" {
		t.Errorf("expected remaining_fee 0, got %s", process.RemainingFee)
	}
	if process.NextExecuteAt != 0 {
		t.Errorf("expected next_execute_at 0, got %d", process.NextExecuteAt)
	}

	// Verify funds returned to creator
	balAfterCancel := bk.balances[creatorAddr.String()+"/uzrn"]
	expectedBal := balAfterCreate.Add(sdkmath.NewInt(500000))
	if !balAfterCancel.Equal(expectedBal) {
		t.Errorf("expected creator balance %s after refund, got %s", expectedBal, balAfterCancel)
	}

	// Cancelling an already cancelled schedule should fail
	_, err = srv.CancelSchedule(ctx, &types.MsgCancelSchedule{
		Creator:   creator,
		ProcessId: processId,
	})
	if err == nil {
		t.Error("expected error when cancelling already-cancelled schedule")
	}
}

// -----------------------------------------------------------------------
// Tests: FundSchedule
// -----------------------------------------------------------------------

func TestFundSchedule(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	creator := testAddr(1)
	creatorAddr, _ := sdk.AccAddressFromBech32(creator)
	bk.setBalance(creatorAddr, "uzrn", sdkmath.NewInt(10000000))

	srv := keeper.NewMsgServerImpl(k)

	// Create a schedule with limited funds
	createResp, err := srv.CreateSchedule(ctx, &types.MsgCreateSchedule{
		Creator: creator,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           "every_n_blocks",
				IntervalBlocks: 50,
			},
		},
		FeePerExecution: "100000",
		PrepaidFee:      "200000",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	processId := createResp.ProcessId

	// Verify initial remaining fee
	process, _ := k.GetProcess(ctx, processId)
	if process.RemainingFee != "200000" {
		t.Errorf("expected initial remaining_fee 200000, got %s", process.RemainingFee)
	}

	// Fund the schedule with additional funds
	_, err = srv.FundSchedule(ctx, &types.MsgFundSchedule{
		Creator:   creator,
		ProcessId: processId,
		Amount:    "300000",
	})
	if err != nil {
		t.Fatalf("fund failed: %v", err)
	}

	// Verify increased remaining fee
	process, _ = k.GetProcess(ctx, processId)
	if process.RemainingFee != "500000" {
		t.Errorf("expected remaining_fee 500000 after funding, got %s", process.RemainingFee)
	}

	// Verify creator balance decreased
	creatorBal := bk.balances[creatorAddr.String()+"/uzrn"]
	// Started with 10M, paid 200K prepaid + 300K funding = 500K total deducted
	expectedBal := sdkmath.NewInt(9500000)
	if !creatorBal.Equal(expectedBal) {
		t.Errorf("expected creator balance %s, got %s", expectedBal, creatorBal)
	}
}

func TestFundSchedule_ReactivateExhausted(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	creator := testAddr(1)
	creatorAddr, _ := sdk.AccAddressFromBech32(creator)
	bk.setBalance(creatorAddr, "uzrn", sdkmath.NewInt(10000000))

	// Manually create an exhausted process
	process := &types.ScheduleProcess{
		Id:              "sched-exhausted",
		Creator:         creator,
		Status:          "exhausted",
		RemainingFee:    "0",
		FeePerExecution: "100000",
		NextExecuteAt:   0,
		CreatedAtBlock:  50,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           "every_n_blocks",
				IntervalBlocks: 50,
			},
		},
	}
	k.SetProcess(ctx, process)

	// Also seed the module account so funding transfer works
	// (funding goes from creator to module)
	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.FundSchedule(ctx, &types.MsgFundSchedule{
		Creator:   creator,
		ProcessId: "sched-exhausted",
		Amount:    "500000",
	})
	if err != nil {
		t.Fatalf("fund failed: %v", err)
	}

	// Verify process is reactivated
	updated, found := k.GetProcess(ctx, "sched-exhausted")
	if !found {
		t.Fatal("expected process to be found")
	}
	if updated.Status != "active" {
		t.Errorf("expected status active after funding exhausted, got %s", updated.Status)
	}
	if updated.RemainingFee != "500000" {
		t.Errorf("expected remaining_fee 500000, got %s", updated.RemainingFee)
	}
	if updated.NextExecuteAt == 0 {
		t.Error("expected non-zero next_execute_at after reactivation")
	}
}

func TestFundSchedule_CompletedFails(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	creator := testAddr(1)
	creatorAddr, _ := sdk.AccAddressFromBech32(creator)
	bk.setBalance(creatorAddr, "uzrn", sdkmath.NewInt(10000000))

	// Manually create a completed process
	k.SetProcess(ctx, &types.ScheduleProcess{
		Id:              "sched-done",
		Creator:         creator,
		Status:          "completed",
		RemainingFee:    "0",
		FeePerExecution: "100000",
	})

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.FundSchedule(ctx, &types.MsgFundSchedule{
		Creator:   creator,
		ProcessId: "sched-done",
		Amount:    "100000",
	})
	if err == nil {
		t.Error("expected error when funding a completed schedule")
	}
}

// -----------------------------------------------------------------------
// Tests: ProcessDueSchedules
// -----------------------------------------------------------------------

func TestProcessDueSchedules(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	creator := testAddr(1)
	creatorAddr, _ := sdk.AccAddressFromBech32(creator)
	bk.setBalance(creatorAddr, "uzrn", sdkmath.NewInt(50000000))

	srv := keeper.NewMsgServerImpl(k)

	// Create a periodic schedule: every 50 blocks from block 100
	createResp, err := srv.CreateSchedule(ctx, &types.MsgCreateSchedule{
		Creator: creator,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           "every_n_blocks",
				IntervalBlocks: 50,
			},
		},
		FeePerExecution: "100000",
		PrepaidFee:      "500000",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	processId := createResp.ProcessId

	// Verify next execution is at block 150
	process, _ := k.GetProcess(ctx, processId)
	if process.NextExecuteAt != 150 {
		t.Errorf("expected next_execute_at 150, got %d", process.NextExecuteAt)
	}

	// Process at block 120 -- nothing due
	ctx120 := ctx.WithBlockHeight(120)
	count := k.ProcessDueSchedules(ctx120)
	if count != 0 {
		t.Errorf("expected 0 executions at block 120, got %d", count)
	}

	// Process at block 150 -- schedule is due
	ctx150 := ctx.WithBlockHeight(150)
	count = k.ProcessDueSchedules(ctx150)
	if count != 1 {
		t.Errorf("expected 1 execution at block 150, got %d", count)
	}

	// Verify execution count incremented and fee deducted
	process, _ = k.GetProcess(ctx150, processId)
	if process.ExecutionCount != 1 {
		t.Errorf("expected execution_count 1, got %d", process.ExecutionCount)
	}
	if process.RemainingFee != "400000" {
		t.Errorf("expected remaining_fee 400000, got %s", process.RemainingFee)
	}
	if process.Status != "active" {
		t.Errorf("expected status active, got %s", process.Status)
	}
	// Next execution should be at 150 + 50 = 200
	if process.NextExecuteAt != 200 {
		t.Errorf("expected next_execute_at 200, got %d", process.NextExecuteAt)
	}
}

func TestProcessDueSchedules_MaxExecutionsReached(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	creator := testAddr(1)
	creatorAddr, _ := sdk.AccAddressFromBech32(creator)
	bk.setBalance(creatorAddr, "uzrn", sdkmath.NewInt(50000000))

	srv := keeper.NewMsgServerImpl(k)

	// Create a schedule with max_executions=1
	createResp, err := srv.CreateSchedule(ctx, &types.MsgCreateSchedule{
		Creator: creator,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           "every_n_blocks",
				IntervalBlocks: 50,
			},
		},
		MaxExecutions:   1,
		FeePerExecution: "100000",
		PrepaidFee:      "500000",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	processId := createResp.ProcessId

	// Process at block 150
	ctx150 := ctx.WithBlockHeight(150)
	count := k.ProcessDueSchedules(ctx150)
	if count != 1 {
		t.Errorf("expected 1 execution, got %d", count)
	}

	// Should be completed (max executions reached)
	process, _ := k.GetProcess(ctx150, processId)
	if process.Status != "completed" {
		t.Errorf("expected status completed, got %s", process.Status)
	}
	if process.ExecutionCount != 1 {
		t.Errorf("expected execution_count 1, got %d", process.ExecutionCount)
	}
}

func TestProcessDueSchedules_Exhausted(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	creator := testAddr(1)

	// Create process with fee less than fee_per_execution
	process := &types.ScheduleProcess{
		Id:              "sched-low",
		Creator:         creator,
		Status:          "active",
		RemainingFee:    "50000",  // less than fee_per_execution
		FeePerExecution: "100000", // need 100000 per exec
		NextExecuteAt:   100,
		CreatedAtBlock:  50,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           "every_n_blocks",
				IntervalBlocks: 50,
			},
		},
	}
	k.SetProcess(ctx, process)

	// Process at block 100 -- fee insufficient
	count := k.ProcessDueSchedules(ctx)
	if count != 0 {
		t.Errorf("expected 0 executions (insufficient fee), got %d", count)
	}

	// Verify process is marked as exhausted
	updated, found := k.GetProcess(ctx, "sched-low")
	if !found {
		t.Fatal("expected process to be found")
	}
	if updated.Status != "exhausted" {
		t.Errorf("expected status exhausted, got %s", updated.Status)
	}
}

func TestProcessDueSchedules_Expired(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	creator := testAddr(1)

	// Create process that expires at block 90 (before current block 100)
	process := &types.ScheduleProcess{
		Id:              "sched-expired",
		Creator:         creator,
		Status:          "active",
		RemainingFee:    "1000000",
		FeePerExecution: "100000",
		NextExecuteAt:   100,
		ExpiresAtBlock:  90, // expired before current block
		CreatedAtBlock:  50,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           "every_n_blocks",
				IntervalBlocks: 50,
			},
		},
	}
	k.SetProcess(ctx, process)

	count := k.ProcessDueSchedules(ctx)
	if count != 0 {
		t.Errorf("expected 0 executions (expired), got %d", count)
	}

	updated, found := k.GetProcess(ctx, "sched-expired")
	if !found {
		t.Fatal("expected process to be found")
	}
	if updated.Status != "completed" {
		t.Errorf("expected status completed for expired process, got %s", updated.Status)
	}
}

func TestProcessDueSchedules_OneTimeAtBlock(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	creator := testAddr(1)
	creatorAddr, _ := sdk.AccAddressFromBech32(creator)
	bk.setBalance(creatorAddr, "uzrn", sdkmath.NewInt(50000000))

	srv := keeper.NewMsgServerImpl(k)

	// Create a one-time "at_block" schedule
	createResp, err := srv.CreateSchedule(ctx, &types.MsgCreateSchedule{
		Creator: creator,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           "at_block",
				ExecuteAtBlock: 150,
			},
		},
		FeePerExecution: "100000",
		PrepaidFee:      "100000",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	processId := createResp.ProcessId

	// Process at target block
	ctx150 := ctx.WithBlockHeight(150)
	count := k.ProcessDueSchedules(ctx150)
	if count != 1 {
		t.Errorf("expected 1 execution for at_block schedule, got %d", count)
	}

	// One-time schedule should be completed after execution
	process, _ := k.GetProcess(ctx150, processId)
	if process.Status != "completed" {
		t.Errorf("expected status completed for one-time at_block schedule, got %s", process.Status)
	}
}

func TestProcessDueSchedules_FeeDeduction(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	creator := testAddr(1)
	creatorAddr, _ := sdk.AccAddressFromBech32(creator)
	bk.setBalance(creatorAddr, "uzrn", sdkmath.NewInt(50000000))

	srv := keeper.NewMsgServerImpl(k)

	// Create with exactly 3x the fee_per_execution
	createResp, err := srv.CreateSchedule(ctx, &types.MsgCreateSchedule{
		Creator: creator,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           "every_n_blocks",
				IntervalBlocks: 10,
			},
		},
		FeePerExecution: "100000",
		PrepaidFee:      "300000",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	processId := createResp.ProcessId

	// Execute 3 times: blocks 110, 120, 130
	for _, height := range []int64{110, 120, 130} {
		ctxH := ctx.WithBlockHeight(height)
		k.ProcessDueSchedules(ctxH)
	}

	process, _ := k.GetProcess(ctx, processId)
	if process.ExecutionCount != 3 {
		t.Errorf("expected 3 executions, got %d", process.ExecutionCount)
	}
	if process.RemainingFee != "0" {
		t.Errorf("expected remaining_fee 0 after 3 executions, got %s", process.RemainingFee)
	}

	// The 4th execution attempt at block 140 should mark as exhausted
	ctx140 := ctx.WithBlockHeight(140)
	count := k.ProcessDueSchedules(ctx140)
	if count != 0 {
		t.Errorf("expected 0 executions (exhausted), got %d", count)
	}
	process, _ = k.GetProcess(ctx140, processId)
	if process.Status != "exhausted" {
		t.Errorf("expected status exhausted, got %s", process.Status)
	}
}

// -----------------------------------------------------------------------
// Tests: CompoundConditions
// -----------------------------------------------------------------------

func TestCompoundConditions_AND(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	creator := testAddr(1)
	creatorAddr, _ := sdk.AccAddressFromBech32(creator)
	bk.setBalance(creatorAddr, "uzrn", sdkmath.NewInt(50000000))

	srv := keeper.NewMsgServerImpl(k)

	// Create a compound AND condition: time + time
	createResp, err := srv.CreateSchedule(ctx, &types.MsgCreateSchedule{
		Creator: creator,
		Condition: &types.ScheduleCondition{
			CompoundCondition: &types.CompoundCondition{
				Operator: "and",
				Conditions: []*types.ScheduleCondition{
					{
						TimeCondition: &types.TimeCondition{
							Type:           "every_n_blocks",
							IntervalBlocks: 50,
						},
					},
					{
						TimeCondition: &types.TimeCondition{
							Type:           "every_n_blocks",
							IntervalBlocks: 100,
						},
					},
				},
			},
		},
		FeePerExecution: "100000",
		PrepaidFee:      "500000",
	})
	if err != nil {
		t.Fatalf("create with AND compound failed: %v", err)
	}

	// Verify process was stored
	process, found := k.GetProcess(ctx, createResp.ProcessId)
	if !found {
		t.Fatal("expected compound AND process to be found")
	}
	if process.Status != "active" {
		t.Errorf("expected status active, got %s", process.Status)
	}
	// For compound, NextExecuteAt should be the earliest sub-condition
	// Both: 100+50=150 and 100+100=200, earliest is 150
	if process.NextExecuteAt != 150 {
		t.Errorf("expected next_execute_at 150 (earliest), got %d", process.NextExecuteAt)
	}
}

func TestCompoundConditions_OR(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	creator := testAddr(1)
	creatorAddr, _ := sdk.AccAddressFromBech32(creator)
	bk.setBalance(creatorAddr, "uzrn", sdkmath.NewInt(50000000))

	srv := keeper.NewMsgServerImpl(k)

	// Create a compound OR condition
	createResp, err := srv.CreateSchedule(ctx, &types.MsgCreateSchedule{
		Creator: creator,
		Condition: &types.ScheduleCondition{
			CompoundCondition: &types.CompoundCondition{
				Operator: "or",
				Conditions: []*types.ScheduleCondition{
					{
						TimeCondition: &types.TimeCondition{
							Type:           "every_n_blocks",
							IntervalBlocks: 50,
						},
					},
					{
						TimeCondition: &types.TimeCondition{
							Type:           "every_n_blocks",
							IntervalBlocks: 200,
						},
					},
				},
			},
		},
		FeePerExecution: "100000",
		PrepaidFee:      "500000",
	})
	if err != nil {
		t.Fatalf("create with OR compound failed: %v", err)
	}

	process, found := k.GetProcess(ctx, createResp.ProcessId)
	if !found {
		t.Fatal("expected compound OR process to be found")
	}
	// Earliest: min(100+50, 100+200) = 150
	if process.NextExecuteAt != 150 {
		t.Errorf("expected next_execute_at 150 (earliest), got %d", process.NextExecuteAt)
	}
}

func TestCompoundConditions_InvalidOperator(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	creator := testAddr(1)
	creatorAddr, _ := sdk.AccAddressFromBech32(creator)
	bk.setBalance(creatorAddr, "uzrn", sdkmath.NewInt(50000000))

	srv := keeper.NewMsgServerImpl(k)

	_, err := srv.CreateSchedule(ctx, &types.MsgCreateSchedule{
		Creator: creator,
		Condition: &types.ScheduleCondition{
			CompoundCondition: &types.CompoundCondition{
				Operator: "xor", // invalid
				Conditions: []*types.ScheduleCondition{
					{TimeCondition: &types.TimeCondition{Type: "every_n_blocks", IntervalBlocks: 50}},
					{TimeCondition: &types.TimeCondition{Type: "every_n_blocks", IntervalBlocks: 100}},
				},
			},
		},
		FeePerExecution: "100000",
		PrepaidFee:      "500000",
	})
	if err == nil {
		t.Error("expected error for invalid compound operator")
	}
}

func TestCompoundConditions_TooFewSubConditions(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	creator := testAddr(1)
	creatorAddr, _ := sdk.AccAddressFromBech32(creator)
	bk.setBalance(creatorAddr, "uzrn", sdkmath.NewInt(50000000))

	srv := keeper.NewMsgServerImpl(k)

	_, err := srv.CreateSchedule(ctx, &types.MsgCreateSchedule{
		Creator: creator,
		Condition: &types.ScheduleCondition{
			CompoundCondition: &types.CompoundCondition{
				Operator: "and",
				Conditions: []*types.ScheduleCondition{
					{TimeCondition: &types.TimeCondition{Type: "every_n_blocks", IntervalBlocks: 50}},
					// Only 1 sub-condition, need at least 2
				},
			},
		},
		FeePerExecution: "100000",
		PrepaidFee:      "500000",
	})
	if err == nil {
		t.Error("expected error for compound condition with fewer than 2 sub-conditions")
	}
}

func TestCompoundConditions_DepthExceeded(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	creator := testAddr(1)
	creatorAddr, _ := sdk.AccAddressFromBech32(creator)
	bk.setBalance(creatorAddr, "uzrn", sdkmath.NewInt(50000000))

	srv := keeper.NewMsgServerImpl(k)

	// MaxCompoundDepth is 3 by default. Build depth 4 nesting.
	leaf := &types.ScheduleCondition{
		TimeCondition: &types.TimeCondition{Type: "every_n_blocks", IntervalBlocks: 50},
	}
	// depth 1
	depth1 := &types.ScheduleCondition{
		CompoundCondition: &types.CompoundCondition{
			Operator:   "and",
			Conditions: []*types.ScheduleCondition{leaf, leaf},
		},
	}
	// depth 2
	depth2 := &types.ScheduleCondition{
		CompoundCondition: &types.CompoundCondition{
			Operator:   "and",
			Conditions: []*types.ScheduleCondition{depth1, leaf},
		},
	}
	// depth 3
	depth3 := &types.ScheduleCondition{
		CompoundCondition: &types.CompoundCondition{
			Operator:   "and",
			Conditions: []*types.ScheduleCondition{depth2, leaf},
		},
	}
	// depth 4 (exceeds max of 3)
	depth4 := &types.ScheduleCondition{
		CompoundCondition: &types.CompoundCondition{
			Operator:   "and",
			Conditions: []*types.ScheduleCondition{depth3, leaf},
		},
	}

	_, err := srv.CreateSchedule(ctx, &types.MsgCreateSchedule{
		Creator:         creator,
		Condition:       depth4,
		FeePerExecution: "100000",
		PrepaidFee:      "500000",
	})
	if err == nil {
		t.Error("expected error for compound depth exceeding max")
	}
}

// -----------------------------------------------------------------------
// Tests: MaxSchedulesExceeded
// -----------------------------------------------------------------------

func TestMaxSchedulesExceeded(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	creator := testAddr(1)
	creatorAddr, _ := sdk.AccAddressFromBech32(creator)
	bk.setBalance(creatorAddr, "uzrn", sdkmath.NewInt(100000000))

	// Set a low limit for testing
	params := types.DefaultParams()
	params.MaxActivePerAccount = 3
	k.SetParams(ctx, params)

	srv := keeper.NewMsgServerImpl(k)

	// Create up to the limit
	for i := 0; i < 3; i++ {
		_, err := srv.CreateSchedule(ctx, &types.MsgCreateSchedule{
			Creator: creator,
			Condition: &types.ScheduleCondition{
				TimeCondition: &types.TimeCondition{
					Type:           "every_n_blocks",
					IntervalBlocks: 50,
				},
			},
			FeePerExecution: "10000",
			PrepaidFee:      "100000",
		})
		if err != nil {
			t.Fatalf("create schedule %d failed: %v", i, err)
		}
	}

	// Verify we have 3 active
	activeCount := k.CountActiveByCreator(ctx, creator)
	if activeCount != 3 {
		t.Errorf("expected 3 active schedules, got %d", activeCount)
	}

	// The 4th should fail
	_, err := srv.CreateSchedule(ctx, &types.MsgCreateSchedule{
		Creator: creator,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           "every_n_blocks",
				IntervalBlocks: 50,
			},
		},
		FeePerExecution: "10000",
		PrepaidFee:      "100000",
	})
	if err == nil {
		t.Error("expected max schedules exceeded error")
	}

	// A different creator should still be able to create
	creator2 := testAddr(2)
	creator2Addr, _ := sdk.AccAddressFromBech32(creator2)
	bk.setBalance(creator2Addr, "uzrn", sdkmath.NewInt(100000000))

	_, err = srv.CreateSchedule(ctx, &types.MsgCreateSchedule{
		Creator: creator2,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           "every_n_blocks",
				IntervalBlocks: 50,
			},
		},
		FeePerExecution: "10000",
		PrepaidFee:      "100000",
	})
	if err != nil {
		t.Errorf("expected creator2 to succeed, got error: %v", err)
	}
}

// -----------------------------------------------------------------------
// Tests: Unauthorized
// -----------------------------------------------------------------------

func TestUnauthorized_Pause(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	creator := testAddr(1)
	creatorAddr, _ := sdk.AccAddressFromBech32(creator)
	bk.setBalance(creatorAddr, "uzrn", sdkmath.NewInt(10000000))

	srv := keeper.NewMsgServerImpl(k)

	createResp, err := srv.CreateSchedule(ctx, &types.MsgCreateSchedule{
		Creator: creator,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           "every_n_blocks",
				IntervalBlocks: 50,
			},
		},
		FeePerExecution: "100000",
		PrepaidFee:      "500000",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Another user tries to pause
	attacker := testAddr(99)
	_, err = srv.PauseSchedule(ctx, &types.MsgPauseSchedule{
		Creator:   attacker,
		ProcessId: createResp.ProcessId,
	})
	if err == nil {
		t.Error("expected unauthorized error for non-creator pause")
	}
}

func TestUnauthorized_Resume(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	creator := testAddr(1)
	creatorAddr, _ := sdk.AccAddressFromBech32(creator)
	bk.setBalance(creatorAddr, "uzrn", sdkmath.NewInt(10000000))

	srv := keeper.NewMsgServerImpl(k)

	createResp, err := srv.CreateSchedule(ctx, &types.MsgCreateSchedule{
		Creator: creator,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           "every_n_blocks",
				IntervalBlocks: 50,
			},
		},
		FeePerExecution: "100000",
		PrepaidFee:      "500000",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Pause first (by the creator)
	_, err = srv.PauseSchedule(ctx, &types.MsgPauseSchedule{
		Creator:   creator,
		ProcessId: createResp.ProcessId,
	})
	if err != nil {
		t.Fatalf("pause failed: %v", err)
	}

	// Another user tries to resume
	attacker := testAddr(99)
	_, err = srv.ResumeSchedule(ctx, &types.MsgResumeSchedule{
		Creator:   attacker,
		ProcessId: createResp.ProcessId,
	})
	if err == nil {
		t.Error("expected unauthorized error for non-creator resume")
	}
}

func TestUnauthorized_Cancel(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	creator := testAddr(1)
	creatorAddr, _ := sdk.AccAddressFromBech32(creator)
	bk.setBalance(creatorAddr, "uzrn", sdkmath.NewInt(10000000))

	srv := keeper.NewMsgServerImpl(k)

	createResp, err := srv.CreateSchedule(ctx, &types.MsgCreateSchedule{
		Creator: creator,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           "every_n_blocks",
				IntervalBlocks: 50,
			},
		},
		FeePerExecution: "100000",
		PrepaidFee:      "500000",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Another user tries to cancel
	attacker := testAddr(99)
	_, err = srv.CancelSchedule(ctx, &types.MsgCancelSchedule{
		Creator:   attacker,
		ProcessId: createResp.ProcessId,
	})
	if err == nil {
		t.Error("expected unauthorized error for non-creator cancel")
	}
}

func TestUnauthorized_Fund(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	creator := testAddr(1)
	creatorAddr, _ := sdk.AccAddressFromBech32(creator)
	bk.setBalance(creatorAddr, "uzrn", sdkmath.NewInt(10000000))

	srv := keeper.NewMsgServerImpl(k)

	createResp, err := srv.CreateSchedule(ctx, &types.MsgCreateSchedule{
		Creator: creator,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           "every_n_blocks",
				IntervalBlocks: 50,
			},
		},
		FeePerExecution: "100000",
		PrepaidFee:      "500000",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Another user tries to fund
	attacker := testAddr(99)
	_, err = srv.FundSchedule(ctx, &types.MsgFundSchedule{
		Creator:   attacker,
		ProcessId: createResp.ProcessId,
		Amount:    "100000",
	})
	if err == nil {
		t.Error("expected unauthorized error for non-creator fund")
	}
}

func TestUnauthorized_UpdateParams(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	// Non-authority tries to update params
	_, err := srv.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: testAddr(99),
		Params:    types.DefaultParams(),
	})
	if err == nil {
		t.Error("expected unauthorized error for non-authority update params")
	}

	// Authority should succeed
	authority := k.GetAuthority()
	_, err = srv.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: authority,
		Params:    types.DefaultParams(),
	})
	if err != nil {
		t.Errorf("expected authority update to succeed, got: %v", err)
	}
}

// -----------------------------------------------------------------------
// Tests: Genesis
// -----------------------------------------------------------------------

func TestGenesis(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	creator := testAddr(1)

	// Store a few processes
	k.SetProcess(ctx, &types.ScheduleProcess{
		Id:              "sched-1",
		Creator:         creator,
		Status:          "active",
		RemainingFee:    "1000000",
		FeePerExecution: "100000",
		NextExecuteAt:   200,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{Type: "every_n_blocks", IntervalBlocks: 50},
		},
	})
	k.SetProcess(ctx, &types.ScheduleProcess{
		Id:              "sched-2",
		Creator:         testAddr(2),
		Status:          "paused",
		RemainingFee:    "500000",
		FeePerExecution: "50000",
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{Type: "at_block", ExecuteAtBlock: 300},
		},
	})
	k.SetSequence(ctx, 10)

	// Export genesis
	genState := k.ExportGenesis(ctx)
	if genState.Params == nil {
		t.Fatal("expected non-nil params in genesis")
	}
	if len(genState.Processes) != 2 {
		t.Errorf("expected 2 processes in genesis, got %d", len(genState.Processes))
	}
	if genState.Sequence != 10 {
		t.Errorf("expected sequence 10, got %d", genState.Sequence)
	}

	// Import into a new keeper
	k2, ctx2, _ := setupKeeper(t)
	k2.InitGenesis(ctx2, genState)

	// Verify imported state
	got := k2.ExportGenesis(ctx2)
	if len(got.Processes) != 2 {
		t.Errorf("expected 2 processes after import, got %d", len(got.Processes))
	}
	if got.Sequence != 10 {
		t.Errorf("expected sequence 10 after import, got %d", got.Sequence)
	}

	// Verify individual process retrieval
	p1, found := k2.GetProcess(ctx2, "sched-1")
	if !found {
		t.Fatal("expected sched-1 to be found after genesis import")
	}
	if p1.Status != "active" {
		t.Errorf("expected status active, got %s", p1.Status)
	}
	if p1.RemainingFee != "1000000" {
		t.Errorf("expected remaining_fee 1000000, got %s", p1.RemainingFee)
	}
}

func TestGenesisValidation(t *testing.T) {
	// Valid genesis
	valid := types.DefaultGenesis()
	if err := valid.Validate(); err != nil {
		t.Errorf("unexpected error for valid genesis: %v", err)
	}

	// Invalid: nil params
	nilParams := &types.GenesisState{Params: nil}
	if err := nilParams.Validate(); err == nil {
		t.Error("expected error for nil params in genesis")
	}

	// Invalid: zero MaxActivePerAccount
	badParams := &types.GenesisState{
		Params: &types.Params{
			MaxActivePerAccount: 0,
			MinIntervalBlocks:   10,
			MaxCompoundDepth:    3,
		},
	}
	if err := badParams.Validate(); err == nil {
		t.Error("expected error for zero MaxActivePerAccount")
	}

	// Invalid: zero MinIntervalBlocks
	badInterval := &types.GenesisState{
		Params: &types.Params{
			MaxActivePerAccount: 20,
			MinIntervalBlocks:   0,
			MaxCompoundDepth:    3,
		},
	}
	if err := badInterval.Validate(); err == nil {
		t.Error("expected error for zero MinIntervalBlocks")
	}

	// Invalid: zero MaxCompoundDepth
	badDepth := &types.GenesisState{
		Params: &types.Params{
			MaxActivePerAccount: 20,
			MinIntervalBlocks:   10,
			MaxCompoundDepth:    0,
		},
	}
	if err := badDepth.Validate(); err == nil {
		t.Error("expected error for zero MaxCompoundDepth")
	}
}

// -----------------------------------------------------------------------
// Tests: IterateProcesses
// -----------------------------------------------------------------------

func TestIterateProcesses(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	for i := 0; i < 5; i++ {
		k.SetProcess(ctx, &types.ScheduleProcess{
			Id:              fmt.Sprintf("sched-%d", i),
			Creator:         testAddr(i),
			Status:          "active",
			RemainingFee:    "1000000",
			FeePerExecution: "100000",
			NextExecuteAt:   uint64(200 + i*50),
			Condition: &types.ScheduleCondition{
				TimeCondition: &types.TimeCondition{Type: "every_n_blocks", IntervalBlocks: 50},
			},
		})
	}

	var collected []*types.ScheduleProcess
	k.IterateProcesses(ctx, func(p *types.ScheduleProcess) bool {
		collected = append(collected, p)
		return false
	})
	if len(collected) != 5 {
		t.Errorf("expected 5 processes, got %d", len(collected))
	}

	// Test early termination
	var partial []*types.ScheduleProcess
	k.IterateProcesses(ctx, func(p *types.ScheduleProcess) bool {
		partial = append(partial, p)
		return len(partial) >= 2 // stop after 2
	})
	if len(partial) != 2 {
		t.Errorf("expected 2 processes with early termination, got %d", len(partial))
	}
}

// -----------------------------------------------------------------------
// Tests: GetDueProcesses
// -----------------------------------------------------------------------

func TestGetDueProcesses(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Store processes with different NextExecuteAt values
	k.SetProcess(ctx, &types.ScheduleProcess{
		Id: "early", Creator: testAddr(1), Status: "active",
		RemainingFee: "1000000", FeePerExecution: "100000",
		NextExecuteAt: 50,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{Type: "every_n_blocks", IntervalBlocks: 50},
		},
	})
	k.SetProcess(ctx, &types.ScheduleProcess{
		Id: "on-time", Creator: testAddr(2), Status: "active",
		RemainingFee: "1000000", FeePerExecution: "100000",
		NextExecuteAt: 100,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{Type: "every_n_blocks", IntervalBlocks: 50},
		},
	})
	k.SetProcess(ctx, &types.ScheduleProcess{
		Id: "future", Creator: testAddr(3), Status: "active",
		RemainingFee: "1000000", FeePerExecution: "100000",
		NextExecuteAt: 200,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{Type: "every_n_blocks", IntervalBlocks: 50},
		},
	})

	// At block 100, "early" (50) and "on-time" (100) should be due
	dueIds := k.GetDueProcesses(ctx, 100)
	if len(dueIds) != 2 {
		t.Errorf("expected 2 due processes at block 100, got %d", len(dueIds))
	}

	// At block 50, only "early" should be due
	dueIds = k.GetDueProcesses(ctx, 50)
	if len(dueIds) != 1 {
		t.Errorf("expected 1 due process at block 50, got %d", len(dueIds))
	}

	// At block 200, all should be due
	dueIds = k.GetDueProcesses(ctx, 200)
	if len(dueIds) != 3 {
		t.Errorf("expected 3 due processes at block 200, got %d", len(dueIds))
	}

	// At block 10, none should be due
	dueIds = k.GetDueProcesses(ctx, 10)
	if len(dueIds) != 0 {
		t.Errorf("expected 0 due processes at block 10, got %d", len(dueIds))
	}
}

// -----------------------------------------------------------------------
// Tests: Events
// -----------------------------------------------------------------------

func TestCreateSchedule_EmitsEvent(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	creator := testAddr(1)
	creatorAddr, _ := sdk.AccAddressFromBech32(creator)
	bk.setBalance(creatorAddr, "uzrn", sdkmath.NewInt(10000000))

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.CreateSchedule(ctx, &types.MsgCreateSchedule{
		Creator: creator,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           "every_n_blocks",
				IntervalBlocks: 50,
			},
		},
		FeePerExecution: "100000",
		PrepaidFee:      "500000",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	found := false
	for _, ev := range ctx.EventManager().Events() {
		if ev.Type == "zerone.schedule.schedule_created" {
			found = true
			for _, attr := range ev.Attributes {
				if attr.Key == "creator" && attr.Value != creator {
					t.Errorf("expected creator %s, got %s", creator, attr.Value)
				}
				if attr.Key == "prepaid_fee" && attr.Value != "500000" {
					t.Errorf("expected prepaid_fee 500000, got %s", attr.Value)
				}
			}
		}
	}
	if !found {
		t.Error("expected schedule_created event")
	}
}
