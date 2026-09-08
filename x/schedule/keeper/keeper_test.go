package keeper_test

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"

	corestore "cosmossdk.io/core/store"
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
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/schedule/keeper"
	"github.com/zerone-chain/zerone/x/schedule/types"
)

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("zrn", "zrnpub")
	config.SetBech32PrefixForValidator("zrnvaloper", "zrnvaloperpub")
	config.SetBech32PrefixForConsensusNode("zrnvalcons", "zrnvalconspub")
}

type mockBankKeeper struct {
	key                 *storetypes.KVStoreKey
	blocked             map[string]bool
	extraModuleBalances sdk.Coins
	failModuleToAccount bool
	failRefundTo        map[string]bool
	failModuleToModule  bool
}

type mockEmergencyKeeper struct {
	halted       bool
	releaseBlock uint64
}

type faultingKVStoreService struct {
	corestore.KVStoreService
	terminalErr error
	closeErr    error
}

func (s faultingKVStoreService) OpenKVStore(ctx context.Context) corestore.KVStore {
	return faultingKVStore{
		KVStore:     s.KVStoreService.OpenKVStore(ctx),
		terminalErr: s.terminalErr,
		closeErr:    s.closeErr,
	}
}

type faultingKVStore struct {
	corestore.KVStore
	terminalErr error
	closeErr    error
}

func (s faultingKVStore) Iterator(start, end []byte) (corestore.Iterator, error) {
	iterator, err := s.KVStore.Iterator(start, end)
	if err != nil {
		return nil, err
	}
	return &faultingIterator{
		Iterator:    iterator,
		terminalErr: s.terminalErr,
		closeErr:    s.closeErr,
	}, nil
}

type faultingIterator struct {
	corestore.Iterator
	terminalErr error
	closeErr    error
}

func (i *faultingIterator) Error() error {
	if i.terminalErr != nil && !i.Valid() {
		return i.terminalErr
	}
	return i.Iterator.Error()
}

func (i *faultingIterator) Close() error {
	return errors.Join(i.Iterator.Close(), i.closeErr)
}

func (m *mockEmergencyKeeper) IsHalted(context.Context) bool { return m.halted }

func (m *mockEmergencyKeeper) GetQuarantineReleaseBlock(context.Context) uint64 {
	return m.releaseBlock
}

func (m *mockBankKeeper) accountKey(address sdk.AccAddress) []byte {
	return []byte("account/" + address.String())
}

func (m *mockBankKeeper) moduleKey(module string) []byte {
	return []byte("module/" + module)
}

func (m *mockBankKeeper) amount(ctx context.Context, key []byte) *big.Int {
	bz := sdk.UnwrapSDKContext(ctx).KVStore(m.key).Get(key)
	if bz == nil {
		return new(big.Int)
	}
	value, ok := new(big.Int).SetString(string(bz), 10)
	if !ok {
		panic("corrupt mock bank balance")
	}
	return value
}

func (m *mockBankKeeper) setAmount(ctx context.Context, key []byte, amount *big.Int) {
	if amount.Sign() < 0 {
		panic("negative mock bank balance")
	}
	sdk.UnwrapSDKContext(ctx).KVStore(m.key).Set(key, []byte(amount.String()))
}

func (m *mockBankKeeper) fund(ctx context.Context, address sdk.AccAddress, amount int64) {
	m.setAmount(ctx, m.accountKey(address), big.NewInt(amount))
}

func (m *mockBankKeeper) accountBalance(ctx context.Context, address sdk.AccAddress) *big.Int {
	return m.amount(ctx, m.accountKey(address))
}

func (m *mockBankKeeper) moduleBalance(ctx context.Context, module string) *big.Int {
	return m.amount(ctx, m.moduleKey(module))
}

func (m *mockBankKeeper) SpendableCoins(ctx context.Context, address sdk.AccAddress) sdk.Coins {
	amount := m.accountBalance(ctx, address)
	if amount.Sign() == 0 {
		return sdk.Coins{}
	}
	return sdk.NewCoins(sdk.NewCoin(types.Denom, sdkmath.NewIntFromBigInt(amount)))
}

func (m *mockBankKeeper) SendCoinsFromAccountToModule(ctx context.Context, sender sdk.AccAddress, recipientModule string, coins sdk.Coins) error {
	amount := coins.AmountOf(types.Denom).BigInt()
	balance := m.accountBalance(ctx, sender)
	if balance.Cmp(amount) < 0 {
		return errors.New("insufficient account balance")
	}
	m.setAmount(ctx, m.accountKey(sender), new(big.Int).Sub(balance, amount))
	m.setAmount(ctx, m.moduleKey(recipientModule), new(big.Int).Add(m.moduleBalance(ctx, recipientModule), amount))
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipient sdk.AccAddress, coins sdk.Coins) error {
	if m.failModuleToAccount || m.failRefundTo[recipient.String()] || m.BlockedAddr(recipient) {
		return errors.New("module-to-account transfer rejected")
	}
	amount := coins.AmountOf(types.Denom).BigInt()
	balance := m.moduleBalance(ctx, senderModule)
	if balance.Cmp(amount) < 0 {
		return errors.New("insufficient module balance")
	}
	m.setAmount(ctx, m.moduleKey(senderModule), new(big.Int).Sub(balance, amount))
	m.setAmount(ctx, m.accountKey(recipient), new(big.Int).Add(m.accountBalance(ctx, recipient), amount))
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, coins sdk.Coins) error {
	if m.failModuleToModule {
		return errors.New("module-to-module transfer rejected")
	}
	amount := coins.AmountOf(types.Denom).BigInt()
	balance := m.moduleBalance(ctx, senderModule)
	if balance.Cmp(amount) < 0 {
		return errors.New("insufficient module balance")
	}
	m.setAmount(ctx, m.moduleKey(senderModule), new(big.Int).Sub(balance, amount))
	m.setAmount(ctx, m.moduleKey(recipientModule), new(big.Int).Add(m.moduleBalance(ctx, recipientModule), amount))
	return nil
}

func (m *mockBankKeeper) GetBalance(ctx context.Context, address sdk.AccAddress, denom string) sdk.Coin {
	if denom != types.Denom {
		return sdk.NewCoin(denom, sdkmath.ZeroInt())
	}
	var amount *big.Int
	switch address.String() {
	case authtypes.NewModuleAddress(types.ModuleName).String():
		amount = m.moduleBalance(ctx, types.ModuleName)
	case authtypes.NewModuleAddress(authtypes.FeeCollectorName).String():
		amount = m.moduleBalance(ctx, authtypes.FeeCollectorName)
	default:
		amount = m.accountBalance(ctx, address)
	}
	return sdk.NewCoin(denom, sdkmath.NewIntFromBigInt(amount))
}

func (m *mockBankKeeper) GetAllBalances(ctx context.Context, address sdk.AccAddress) sdk.Coins {
	balance := m.GetBalance(ctx, address, types.Denom)
	balances := sdk.NewCoins(m.extraModuleBalances...)
	if balance.IsZero() {
		return balances
	}
	return balances.Add(balance)
}

func (m *mockBankKeeper) BlockedAddr(address sdk.AccAddress) bool {
	return m.blocked[address.String()]
}

type harness struct {
	keeper    keeper.Keeper
	server    types.MsgServer
	ctx       sdk.Context
	bank      *mockBankKeeper
	emergency *mockEmergencyKeeper
	store     corestore.KVStoreService
}

func setup(t *testing.T) harness {
	t.Helper()
	h := setupWithGenesisAndBlocked(t, types.DefaultGenesis(), nil)
	params := types.DefaultParams()
	params.AcceptNewSchedules = true
	require.NoError(t, h.keeper.SetParams(h.ctx, params))
	return h
}

func setupWithGenesis(t *testing.T, genesis *types.GenesisState) harness {
	t.Helper()
	return setupWithGenesisAndBlocked(t, genesis, nil)
}

func setupWithGenesisAndBlocked(
	t *testing.T,
	genesis *types.GenesisState,
	blocked []sdk.AccAddress,
) harness {
	t.Helper()
	scheduleKey := storetypes.NewKVStoreKey(types.StoreKey)
	bankKey := storetypes.NewKVStoreKey("schedule_test_bank")
	database := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(database, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(scheduleKey, storetypes.StoreTypeIAVL, database)
	stateStore.MountStoreWithDB(bankKey, storetypes.StoreTypeIAVL, database)
	require.NoError(t, stateStore.LoadLatestVersion())
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	bank := &mockBankKeeper{key: bankKey, blocked: map[string]bool{}}
	for _, address := range blocked {
		bank.blocked[address.String()] = true
	}
	emergency := &mockEmergencyKeeper{}
	authority := testAddress("authority").String()
	scheduleStore := runtime.NewKVStoreService(scheduleKey)
	k := keeper.NewKeeper(scheduleStore, cdc, bank, emergency, authority)
	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 100, ChainID: "schedule-test-1"}, false, log.NewNopLogger())
	total, err := types.ParseNonNegativeAmount(genesis.TotalEscrowUzrn)
	require.NoError(t, err)
	bank.setAmount(ctx, bank.moduleKey(types.ModuleName), total)
	k.InitGenesis(ctx, genesis)
	return harness{
		keeper: k, server: keeper.NewMsgServerImpl(k), ctx: ctx, bank: bank,
		emergency: emergency, store: scheduleStore,
	}
}

func testAddress(seed string) sdk.AccAddress {
	bz := make([]byte, 20)
	copy(bz, []byte(seed))
	return sdk.AccAddress(bz)
}

func createSchedule(t *testing.T, h harness, creator, recipient sdk.AccAddress, first uint64, interval uint64, count uint32, amount string) *types.Schedule {
	t.Helper()
	h.bank.fund(h.ctx, creator, 10_000_000)
	response, err := h.server.CreateSchedule(h.ctx, &types.MsgCreateSchedule{
		Creator: creator.String(), Recipient: recipient.String(), AmountPerExecutionUzrn: amount,
		FirstExecutionHeight: first, IntervalBlocks: interval, ExecutionCount: count,
	})
	require.NoError(t, err)
	schedule, found := h.keeper.GetSchedule(h.ctx, response.ScheduleId)
	require.True(t, found)
	return schedule
}

func TestDefaultGenesisClosesAdmission(t *testing.T) {
	h := setup(t)
	params := types.DefaultParams()
	require.False(t, params.AcceptNewSchedules)
	require.NoError(t, h.keeper.SetParams(h.ctx, params))
	creator := testAddress("closed-creator")
	recipient := testAddress("closed-recipient")
	h.bank.fund(h.ctx, creator, 1_000_000)
	_, err := h.server.CreateSchedule(h.ctx, &types.MsgCreateSchedule{
		Creator: creator.String(), Recipient: recipient.String(), AmountPerExecutionUzrn: "1",
		FirstExecutionHeight: 102, ExecutionCount: 1,
	})
	require.ErrorIs(t, err, types.ErrAdmissionClosed)
	require.Zero(t, h.keeper.TotalEscrow(h.ctx).Sign())
}

func TestEscrowInvariantRejectsUntrackedDenomination(t *testing.T) {
	h := setup(t)
	h.bank.extraModuleBalances = sdk.NewCoins(sdk.NewInt64Coin("uatom", 1))
	require.ErrorContains(t, h.keeper.AssertEscrowInvariant(h.ctx), "exact tracked liability")
}

func TestExportGenesisFailsClosedOnChainDomainAndEscrowDrift(t *testing.T) {
	t.Run("receipt belongs to another chain domain", func(t *testing.T) {
		h := setup(t)
		created := createSchedule(
			t,
			h,
			testAddress("export-domain-creator"),
			testAddress("export-domain-recipient"),
			102,
			0,
			1,
			"9",
		)
		processed, err := h.keeper.ProcessDueSchedules(h.ctx.WithBlockHeight(102))
		require.NoError(t, err)
		require.Equal(t, uint32(1), processed)
		_, found := h.keeper.GetReceipt(h.ctx, created.Id, 1)
		require.True(t, found)

		requirePanicContains(t, "chain-bound occurrence", func() {
			h.keeper.ExportGenesis(
				h.ctx.WithBlockHeight(102).WithChainID("different-chain-2"),
			)
		})
		_, found = h.keeper.GetReceipt(h.ctx, created.Id, 1)
		require.True(t, found, "failed export must not mutate receipt state")
	})

	t.Run("untracked module denomination", func(t *testing.T) {
		h := setup(t)
		h.bank.extraModuleBalances = sdk.NewCoins(sdk.NewInt64Coin("uatom", 1))
		requirePanicContains(t, "broken escrow invariant", func() {
			h.keeper.ExportGenesis(h.ctx)
		})
		require.Equal(
			t,
			sdk.NewCoins(sdk.NewInt64Coin("uatom", 1)),
			h.bank.extraModuleBalances,
			"failed export must not mutate bank state",
		)
	})
}

func requirePanicContains(t *testing.T, expected string, call func()) {
	t.Helper()
	defer func() {
		value := recover()
		require.NotNil(t, value, "expected panic containing %q", expected)
		require.Contains(t, fmt.Sprint(value), expected)
	}()
	call()
}

func TestScheduleIterationFailsClosedOnVisibleIteratorErrors(t *testing.T) {
	h := setup(t)
	newFaultingKeeper := func(terminalErr, closeErr error) keeper.Keeper {
		registry := codectypes.NewInterfaceRegistry()
		return keeper.NewKeeper(
			faultingKVStoreService{
				KVStoreService: h.store,
				terminalErr:    terminalErr,
				closeErr:       closeErr,
			},
			codec.NewProtoCodec(registry),
			h.bank,
			h.emergency,
			testAddress("authority").String(),
		)
	}

	require.PanicsWithValue(t, "iterate schedule state: injected terminal iterator failure", func() {
		newFaultingKeeper(errors.New("injected terminal iterator failure"), nil).GetAllSchedules(h.ctx)
	})
	require.PanicsWithValue(t, "close schedule state iterator: injected close iterator failure", func() {
		newFaultingKeeper(nil, errors.New("injected close iterator failure")).GetAllSchedules(h.ctx)
	})
	require.PanicsWithValue(t, "iterate schedule state: invalid cacheMergeIterator: injected", func() {
		newFaultingKeeper(errors.New("invalid cacheMergeIterator: injected"), nil).GetAllSchedules(h.ctx)
	})
	for _, tolerated := range []string{"invalid cacheMergeIterator", "invalid memIterator"} {
		require.NotPanics(t, func() {
			newFaultingKeeper(errors.New(tolerated), nil).GetAllSchedules(h.ctx)
		})
	}
}

func TestTotalEscrowOverflowIsRejectedBeforeStateMutation(t *testing.T) {
	h := setup(t)
	maximum := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	h.keeper.SetTotalEscrow(h.ctx, maximum)

	err := h.keeper.AddTotalEscrow(h.ctx, big.NewInt(1))
	require.ErrorIs(t, err, types.ErrEscrowInvariant)
	require.ErrorContains(t, err, "256-bit")
	require.Equal(t, maximum.String(), h.keeper.TotalEscrow(h.ctx).String())
}

func TestCreateRejectsAggregateEscrowOverflowBeforeBankMutation(t *testing.T) {
	h := setup(t)
	maximum := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	h.keeper.SetTotalEscrow(h.ctx, maximum)
	h.bank.setAmount(h.ctx, h.bank.moduleKey(types.ModuleName), maximum)
	creator := testAddress("aggregate-overflow")
	recipient := testAddress("overflow-recipient")
	h.bank.fund(h.ctx, creator, 1_000_000)
	creatorBefore := new(big.Int).Set(h.bank.accountBalance(h.ctx, creator))

	_, err := h.server.CreateSchedule(h.ctx, &types.MsgCreateSchedule{
		Creator:                creator.String(),
		Recipient:              recipient.String(),
		AmountPerExecutionUzrn: "1",
		FirstExecutionHeight:   102,
		IntervalBlocks:         0,
		ExecutionCount:         1,
	})
	require.ErrorIs(t, err, types.ErrEscrowInvariant)
	require.ErrorContains(t, err, "256-bit")
	require.Equal(t, creatorBefore.String(), h.bank.accountBalance(h.ctx, creator).String())
	require.Equal(t, maximum.String(), h.bank.moduleBalance(h.ctx, types.ModuleName).String())
	require.Equal(t, maximum.String(), h.keeper.TotalEscrow(h.ctx).String())
	require.Equal(t, uint64(1), h.keeper.PeekNextScheduleID(h.ctx))
	require.Empty(t, h.keeper.GetAllSchedules(h.ctx))
}

func TestExhaustedScheduleIDSentinelExportsImportsAndRejectsWithoutMutation(t *testing.T) {
	h := setup(t)
	creator := testAddress("last-id-creator")
	recipient := testAddress("last-id-recipient")
	h.bank.fund(h.ctx, creator, 1_000_000)
	h.keeper.SetNextScheduleID(h.ctx, ^uint64(0)-1)

	created, err := h.server.CreateSchedule(h.ctx, &types.MsgCreateSchedule{
		Creator:                creator.String(),
		Recipient:              recipient.String(),
		AmountPerExecutionUzrn: "1",
		FirstExecutionHeight:   102,
		ExecutionCount:         1,
	})
	require.NoError(t, err)
	require.Equal(t, types.FormatScheduleID(^uint64(0)-1), created.ScheduleId)
	require.Equal(t, ^uint64(0), h.keeper.PeekNextScheduleID(h.ctx))

	exported := h.keeper.ExportGenesis(h.ctx)
	require.Equal(t, ^uint64(0), exported.NextScheduleId)
	require.NoError(t, exported.ValidateForChainID(h.ctx.ChainID()))

	imported := setupWithGenesis(t, proto.Clone(exported).(*types.GenesisState))
	_, found := imported.keeper.GetSchedule(imported.ctx, created.ScheduleId)
	require.True(t, found)
	require.Equal(t, ^uint64(0), imported.keeper.PeekNextScheduleID(imported.ctx))

	nextCreator := testAddress("exhausted-creator")
	nextRecipient := testAddress("exhausted-recipient")
	imported.bank.fund(imported.ctx, nextCreator, 1_000_000)
	creatorBefore := new(big.Int).Set(imported.bank.accountBalance(imported.ctx, nextCreator))
	moduleBefore := new(big.Int).Set(imported.bank.moduleBalance(imported.ctx, types.ModuleName))
	liabilityBefore := new(big.Int).Set(imported.keeper.TotalEscrow(imported.ctx))
	schedulesBefore := len(imported.keeper.GetAllSchedules(imported.ctx))

	_, err = imported.server.CreateSchedule(imported.ctx, &types.MsgCreateSchedule{
		Creator:                nextCreator.String(),
		Recipient:              nextRecipient.String(),
		AmountPerExecutionUzrn: "1",
		FirstExecutionHeight:   102,
		ExecutionCount:         1,
	})
	require.ErrorIs(t, err, types.ErrScheduleLimit)
	require.Equal(t, creatorBefore.String(), imported.bank.accountBalance(imported.ctx, nextCreator).String())
	require.Equal(t, moduleBefore.String(), imported.bank.moduleBalance(imported.ctx, types.ModuleName).String())
	require.Equal(t, liabilityBefore.String(), imported.keeper.TotalEscrow(imported.ctx).String())
	require.Equal(t, schedulesBefore, len(imported.keeper.GetAllSchedules(imported.ctx)))
	require.Equal(t, ^uint64(0), imported.keeper.PeekNextScheduleID(imported.ctx))
}

func TestCreateFullyPrefundsPrincipalAndFees(t *testing.T) {
	h := setup(t)
	creator := testAddress("create-creator")
	recipient := testAddress("create-recipient")
	schedule := createSchedule(t, h, creator, recipient, 102, 10, 3, "1000")
	require.Equal(t, "3000", schedule.PrincipalRemainingUzrn)
	require.Equal(t, "300000", schedule.FeeRemainingUzrn)
	require.Equal(t, "303000", h.keeper.TotalEscrow(h.ctx).String())
	require.Equal(t, "303000", h.bank.moduleBalance(h.ctx, types.ModuleName).String())
	require.NoError(t, h.keeper.AssertEscrowInvariant(h.ctx))
	require.Equal(t, uint32(1), h.keeper.CountActiveByCreator(h.ctx, creator, 2))
}

func TestOneShotExecutesExactlyOnceAndRoutesFee(t *testing.T) {
	h := setup(t)
	creator := testAddress("once-creator")
	recipient := testAddress("once-recipient")
	schedule := createSchedule(t, h, creator, recipient, 102, 0, 1, "5000")
	dueCtx := h.ctx.WithBlockHeight(102)
	processed, err := h.keeper.ProcessDueSchedules(dueCtx)
	require.NoError(t, err)
	require.Equal(t, uint32(1), processed)
	require.Equal(t, "5000", h.bank.accountBalance(dueCtx, recipient).String())
	require.Equal(t, types.DefaultExecutionFeeUzrn, h.bank.moduleBalance(dueCtx, authtypes.FeeCollectorName).String())
	require.Zero(t, h.bank.moduleBalance(dueCtx, types.ModuleName).Sign())
	require.Zero(t, h.keeper.TotalEscrow(dueCtx).Sign())
	stored, found := h.keeper.GetSchedule(dueCtx, schedule.Id)
	require.True(t, found)
	require.Equal(t, types.ScheduleStatus_SCHEDULE_STATUS_COMPLETED, stored.Status)
	require.Equal(t, uint32(1), stored.ExecutionCount)
	receipt, found := h.keeper.GetReceipt(dueCtx, schedule.Id, 1)
	require.True(t, found)
	require.Equal(t, types.ExecutionOutcome_EXECUTION_OUTCOME_SUCCEEDED, receipt.Outcome)
	require.Equal(t, types.OccurrenceID(dueCtx.ChainID(), schedule.Id, 1, 1, 102), receipt.OccurrenceId)
	processed, err = h.keeper.ProcessDueSchedules(dueCtx.WithBlockHeight(103))
	require.NoError(t, err)
	require.Zero(t, processed)
	require.Equal(t, "5000", h.bank.accountBalance(dueCtx, recipient).String())
}

func TestRecurringUsesFixedDelayAndCASSeesExecution(t *testing.T) {
	h := setup(t)
	creator := testAddress("repeat-creator")
	recipient := testAddress("repeat-recipient")
	schedule := createSchedule(t, h, creator, recipient, 102, 10, 2, "7")
	lateCtx := h.ctx.WithBlockHeight(105)
	processed, err := h.keeper.ProcessDueSchedules(lateCtx)
	require.NoError(t, err)
	require.Equal(t, uint32(1), processed)
	stored, _ := h.keeper.GetSchedule(lateCtx, schedule.Id)
	require.Equal(t, uint64(115), stored.NextExecutionHeight)
	require.Equal(t, uint32(1), stored.ExecutionCount)
	_, err = h.server.CancelSchedule(lateCtx, &types.MsgCancelSchedule{
		Creator: creator.String(), ScheduleId: schedule.Id, ExpectedRevision: 1, ExpectedExecutionCount: 0,
	})
	require.ErrorIs(t, err, types.ErrExecutionConflict)
}

func TestDueProcessingIsBoundedAndLexicographic(t *testing.T) {
	h := setup(t)
	params := h.keeper.GetParams(h.ctx)
	params.MaxDueRecordsPerBlock = 2
	require.NoError(t, h.keeper.SetParams(h.ctx, params))
	recipient := testAddress("bounded-recipient")
	var schedules []*types.Schedule
	for i := 0; i < 3; i++ {
		schedules = append(schedules, createSchedule(t, h, testAddress(string(rune('a'+i))+"-creator"), recipient, 102, 0, 1, "1"))
	}
	dueCtx := h.ctx.WithBlockHeight(102)
	processed, err := h.keeper.ProcessDueSchedules(dueCtx)
	require.NoError(t, err)
	require.Equal(t, uint32(2), processed)
	first, _ := h.keeper.GetSchedule(dueCtx, schedules[0].Id)
	second, _ := h.keeper.GetSchedule(dueCtx, schedules[1].Id)
	third, _ := h.keeper.GetSchedule(dueCtx, schedules[2].Id)
	require.Equal(t, types.ScheduleStatus_SCHEDULE_STATUS_COMPLETED, first.Status)
	require.Equal(t, types.ScheduleStatus_SCHEDULE_STATUS_COMPLETED, second.Status)
	require.Equal(t, types.ScheduleStatus_SCHEDULE_STATUS_ACTIVE, third.Status)
	processed, err = h.keeper.ProcessDueSchedules(dueCtx.WithBlockHeight(103))
	require.NoError(t, err)
	require.Equal(t, uint32(1), processed)
}

func TestFeeTransferFailureRollsBackActionAndRefundsEverything(t *testing.T) {
	h := setup(t)
	creator := testAddress("failure-creator")
	recipient := testAddress("failure-recipient")
	h.bank.fund(h.ctx, creator, 1_000_000)
	initial := new(big.Int).Set(h.bank.accountBalance(h.ctx, creator))
	response, err := h.server.CreateSchedule(h.ctx, &types.MsgCreateSchedule{
		Creator: creator.String(), Recipient: recipient.String(), AmountPerExecutionUzrn: "5000",
		FirstExecutionHeight: 102, ExecutionCount: 1,
	})
	require.NoError(t, err)
	h.bank.failModuleToModule = true
	dueCtx := h.ctx.WithBlockHeight(102)
	processed, err := h.keeper.ProcessDueSchedules(dueCtx)
	require.NoError(t, err)
	require.Equal(t, uint32(1), processed)
	require.Zero(t, h.bank.accountBalance(dueCtx, recipient).Sign(), "principal send in discarded cache must not leak")
	require.Equal(t, initial.String(), h.bank.accountBalance(dueCtx, creator).String())
	require.Zero(t, h.bank.moduleBalance(dueCtx, types.ModuleName).Sign())
	stored, _ := h.keeper.GetSchedule(dueCtx, response.ScheduleId)
	require.Equal(t, types.ScheduleStatus_SCHEDULE_STATUS_FAILED, stored.Status)
	require.Equal(t, uint32(1), stored.ExecutionCount)
	receipt, found := h.keeper.GetReceipt(dueCtx, response.ScheduleId, 1)
	require.True(t, found)
	require.Equal(t, types.ExecutionOutcome_EXECUTION_OUTCOME_FAILED_AND_REFUNDED, receipt.Outcome)
	require.Equal(t, types.FailureCodeBankTransfer, receipt.FailureCode)
}

func TestRefundFailureFailStopsWithoutPartialStateAndRetryIsExact(t *testing.T) {
	h := setup(t)
	creator := testAddress("fail-stop-creator")
	recipient := testAddress("fail-stop-recipient")
	h.bank.fund(h.ctx, creator, 1_000_000)
	created, err := h.server.CreateSchedule(h.ctx, &types.MsgCreateSchedule{
		Creator: creator.String(), Recipient: recipient.String(), AmountPerExecutionUzrn: "5000",
		FirstExecutionHeight: 102, ExecutionCount: 1,
	})
	require.NoError(t, err)
	creatorAfterEscrow := new(big.Int).Set(h.bank.accountBalance(h.ctx, creator))
	escrow := new(big.Int).Set(h.keeper.TotalEscrow(h.ctx))

	// The principal succeeds in the action cache, the fee transfer fails, and
	// the compensating refund then fails in its own cache. Neither cache may
	// leak a partial transfer or lifecycle transition into committed state.
	h.bank.failModuleToModule = true
	h.bank.failRefundTo = map[string]bool{creator.String(): true}
	failingCtx := h.ctx.WithBlockHeight(102)
	processed, err := h.keeper.ProcessDueSchedules(failingCtx)
	require.ErrorIs(t, err, types.ErrEscrowInvariant)
	require.ErrorContains(t, err, "refund failed")
	require.Zero(t, processed)
	require.Equal(t, creatorAfterEscrow.String(), h.bank.accountBalance(failingCtx, creator).String())
	require.Zero(t, h.bank.accountBalance(failingCtx, recipient).Sign())
	require.Zero(t, h.bank.moduleBalance(failingCtx, authtypes.FeeCollectorName).Sign())
	require.Equal(t, escrow.String(), h.bank.moduleBalance(failingCtx, types.ModuleName).String())
	require.Equal(t, escrow.String(), h.keeper.TotalEscrow(failingCtx).String())
	require.NoError(t, h.keeper.AssertEscrowInvariant(failingCtx))
	stored, found := h.keeper.GetSchedule(failingCtx, created.ScheduleId)
	require.True(t, found)
	require.Equal(t, types.ScheduleStatus_SCHEDULE_STATUS_ACTIVE, stored.Status)
	require.Zero(t, stored.ExecutionCount)
	require.Equal(t, uint64(102), stored.NextExecutionHeight)
	require.Equal(t, uint32(1), h.keeper.CountActiveByCreator(failingCtx, creator, 1))
	_, found = h.keeper.GetReceipt(failingCtx, created.ScheduleId, 1)
	require.False(t, found)

	// Once the deterministic bank failure is removed, the same due record is
	// processed once using its original due height.
	h.bank.failModuleToModule = false
	h.bank.failRefundTo = nil
	retryCtx := h.ctx.WithBlockHeight(103)
	processed, err = h.keeper.ProcessDueSchedules(retryCtx)
	require.NoError(t, err)
	require.Equal(t, uint32(1), processed)
	require.Equal(t, creatorAfterEscrow.String(), h.bank.accountBalance(retryCtx, creator).String())
	require.Equal(t, "5000", h.bank.accountBalance(retryCtx, recipient).String())
	require.Equal(t, types.DefaultExecutionFeeUzrn, h.bank.moduleBalance(retryCtx, authtypes.FeeCollectorName).String())
	require.Zero(t, h.bank.moduleBalance(retryCtx, types.ModuleName).Sign())
	receipt, found := h.keeper.GetReceipt(retryCtx, created.ScheduleId, 1)
	require.True(t, found)
	require.Equal(t, uint64(102), receipt.DueHeight)
	require.Equal(t, uint64(103), receipt.ExecutedHeight)
}

func TestCommittedFeeAndTransferTermsSurviveProspectiveParamChanges(t *testing.T) {
	h := setup(t)
	oldCreator := testAddress("old-terms-creator")
	oldRecipient := testAddress("old-terms-recipient")
	oldSchedule := createSchedule(t, h, oldCreator, oldRecipient, 102, 10, 2, "101")
	require.Equal(t, types.DefaultExecutionFeeUzrn, oldSchedule.ExecutionFeeUzrn)

	params := h.keeper.GetParams(h.ctx)
	params.ExecutionFeeUzrn = "7"
	params.MaxTransferPerExecutionUzrn = "1"
	params.MaxExecutionsPerSchedule = 1
	params.MinIntervalBlocks = 20
	require.NoError(t, h.keeper.SetParams(h.ctx, params))

	newCreator := testAddress("new-terms-creator")
	newRecipient := testAddress("new-terms-recipient")
	newSchedule := createSchedule(t, h, newCreator, newRecipient, 103, 0, 1, "1")
	require.Equal(t, "7", newSchedule.ExecutionFeeUzrn)

	for _, height := range []int64{102, 103, 112} {
		_, err := h.keeper.ProcessDueSchedules(h.ctx.WithBlockHeight(height))
		require.NoError(t, err)
	}

	oldFirst, found := h.keeper.GetReceipt(h.ctx, oldSchedule.Id, 1)
	require.True(t, found)
	oldSecond, found := h.keeper.GetReceipt(h.ctx, oldSchedule.Id, 2)
	require.True(t, found)
	newReceipt, found := h.keeper.GetReceipt(h.ctx, newSchedule.Id, 1)
	require.True(t, found)
	require.Equal(t, types.DefaultExecutionFeeUzrn, oldFirst.FeeUzrn)
	require.Equal(t, types.DefaultExecutionFeeUzrn, oldSecond.FeeUzrn)
	require.Equal(t, "101", oldFirst.AmountUzrn)
	require.Equal(t, "101", oldSecond.AmountUzrn)
	require.Equal(t, "7", newReceipt.FeeUzrn)
	require.Equal(t, "200007", h.bank.moduleBalance(h.ctx, authtypes.FeeCollectorName).String())
	require.Equal(t, "202", h.bank.accountBalance(h.ctx, oldRecipient).String())
	require.Equal(t, "1", h.bank.accountBalance(h.ctx, newRecipient).String())
	require.Zero(t, h.keeper.TotalEscrow(h.ctx).Sign())
}

func TestClosedAdmissionAllowsOnlyNonIncreasingAmendment(t *testing.T) {
	h := setup(t)
	creator := testAddress("amend-creator")
	recipient := testAddress("amend-recipient")
	schedule := createSchedule(t, h, creator, recipient, 110, 10, 2, "100")
	params := h.keeper.GetParams(h.ctx)
	params.AcceptNewSchedules = false
	require.NoError(t, h.keeper.SetParams(h.ctx, params))
	before := new(big.Int).Set(h.bank.accountBalance(h.ctx, creator))
	response, err := h.server.UpdateSchedule(h.ctx, &types.MsgUpdateSchedule{
		Creator: creator.String(), ScheduleId: schedule.Id, ExpectedRevision: 1, ExpectedExecutionCount: 0,
		Recipient: recipient.String(), AmountPerExecutionUzrn: "50", NextExecutionHeight: 111,
		IntervalBlocks: 0, RemainingExecutions: 1,
	})
	require.NoError(t, err)
	require.True(t, response.Refunded)
	require.True(t, h.bank.accountBalance(h.ctx, creator).Cmp(before) > 0)
	_, err = h.server.UpdateSchedule(h.ctx, &types.MsgUpdateSchedule{
		Creator: creator.String(), ScheduleId: schedule.Id, ExpectedRevision: 2, ExpectedExecutionCount: 0,
		Recipient: recipient.String(), AmountPerExecutionUzrn: "500", NextExecutionHeight: 112,
		IntervalBlocks: 10, RemainingExecutions: 2,
	})
	require.ErrorIs(t, err, types.ErrAdmissionClosed)
}

func TestCancelRefundsExactRemainingLiability(t *testing.T) {
	h := setup(t)
	creator := testAddress("cancel-creator")
	recipient := testAddress("cancel-recipient")
	schedule := createSchedule(t, h, creator, recipient, 110, 10, 2, "100")
	before := new(big.Int).Set(h.bank.accountBalance(h.ctx, creator))
	response, err := h.server.CancelSchedule(h.ctx, &types.MsgCancelSchedule{
		Creator: creator.String(), ScheduleId: schedule.Id, ExpectedRevision: 1, ExpectedExecutionCount: 0,
	})
	require.NoError(t, err)
	require.Equal(t, "200200", response.RefundedUzrn)
	require.Equal(t, "200200", new(big.Int).Sub(h.bank.accountBalance(h.ctx, creator), before).String())
	require.Zero(t, h.keeper.TotalEscrow(h.ctx).Sign())
	stored, _ := h.keeper.GetSchedule(h.ctx, schedule.Id)
	require.Equal(t, types.ScheduleStatus_SCHEDULE_STATUS_CANCELLED, stored.Status)
	require.Equal(t, uint32(0), h.keeper.CountActiveByCreator(h.ctx, creator, 1))
}

func TestBlockedRecipientRejectedBeforeEscrow(t *testing.T) {
	h := setup(t)
	creator := testAddress("blocked-creator")
	recipient := testAddress("blocked-recipient")
	h.bank.blocked[recipient.String()] = true
	h.bank.fund(h.ctx, creator, 1_000_000)
	_, err := h.server.CreateSchedule(h.ctx, &types.MsgCreateSchedule{
		Creator: creator.String(), Recipient: recipient.String(), AmountPerExecutionUzrn: "1",
		FirstExecutionHeight: 102, ExecutionCount: 1,
	})
	require.ErrorIs(t, err, types.ErrBlockedRecipient)
	require.Equal(t, "1000000", h.bank.accountBalance(h.ctx, creator).String())
	require.Zero(t, h.keeper.TotalEscrow(h.ctx).Sign())
}

func TestRecurringScheduleCompletesEveryOccurrence(t *testing.T) {
	h := setup(t)
	creator := testAddress("three-run-creator")
	recipient := testAddress("three-run-recipient")
	schedule := createSchedule(t, h, creator, recipient, 102, 10, 3, "7")

	for sequence, height := range []int64{102, 112, 122} {
		ctx := h.ctx.WithBlockHeight(height)
		processed, err := h.keeper.ProcessDueSchedules(ctx)
		require.NoError(t, err)
		require.Equal(t, uint32(1), processed)
		receipt, found := h.keeper.GetReceipt(ctx, schedule.Id, uint32(sequence+1))
		require.True(t, found)
		require.Equal(t, uint64(height), receipt.DueHeight)
		require.Equal(t, uint64(height), receipt.ExecutedHeight)
	}

	stored, found := h.keeper.GetSchedule(h.ctx.WithBlockHeight(122), schedule.Id)
	require.True(t, found)
	require.Equal(t, types.ScheduleStatus_SCHEDULE_STATUS_COMPLETED, stored.Status)
	require.Equal(t, uint32(3), stored.ExecutionCount)
	require.Zero(t, stored.RemainingExecutions)
	require.Equal(t, "21", h.bank.accountBalance(h.ctx, recipient).String())
	require.Zero(t, h.keeper.TotalEscrow(h.ctx).Sign())
	require.NoError(t, h.keeper.AssertEscrowInvariant(h.ctx))
}

func TestEmergencyPauseGraceCancellationAndBoundedRelease(t *testing.T) {
	h := setup(t)
	recipient := testAddress("grace-recipient")
	backlog := createSchedule(t, h, testAddress("backlog-creator"), recipient, 102, 0, 1, "11")
	cancellableCreator := testAddress("grace-cancel-creator")
	cancellable := createSchedule(t, h, cancellableCreator, recipient, 102, 0, 1, "13")

	h.emergency.halted = true
	for _, height := range []int64{102, 103} {
		processed, err := h.keeper.ProcessDueSchedules(h.ctx.WithBlockHeight(height))
		require.NoError(t, err)
		require.Zero(t, processed)
	}
	_, found := h.keeper.GetReceipt(h.ctx, backlog.Id, 1)
	require.False(t, found)

	// Admission reopens after the resume block, but due execution remains paused
	// for ten complete blocks so users can cancel or reduce committed work.
	h.emergency.halted = false
	h.emergency.releaseBlock = 103
	for height := int64(104); height <= 113; height++ {
		processed, err := h.keeper.ProcessDueSchedules(h.ctx.WithBlockHeight(height))
		require.NoError(t, err)
		require.Zero(t, processed)
	}
	cancelCtx := h.ctx.WithBlockHeight(108)
	_, err := h.server.CancelSchedule(cancelCtx, &types.MsgCancelSchedule{
		Creator:                cancellableCreator.String(),
		ScheduleId:             cancellable.Id,
		ExpectedRevision:       1,
		ExpectedExecutionCount: 0,
	})
	require.NoError(t, err)

	releaseCtx := h.ctx.WithBlockHeight(114)
	processed, err := h.keeper.ProcessDueSchedules(releaseCtx)
	require.NoError(t, err)
	require.Equal(t, uint32(1), processed)
	storedBacklog, _ := h.keeper.GetSchedule(releaseCtx, backlog.Id)
	storedCancelled, _ := h.keeper.GetSchedule(releaseCtx, cancellable.Id)
	require.Equal(t, types.ScheduleStatus_SCHEDULE_STATUS_COMPLETED, storedBacklog.Status)
	require.Equal(t, types.ScheduleStatus_SCHEDULE_STATUS_CANCELLED, storedCancelled.Status)
	require.Equal(t, "11", h.bank.accountBalance(releaseCtx, recipient).String())
	require.Zero(t, h.keeper.TotalEscrow(releaseCtx).Sign())
}

func TestDelayedRecurringExecutionOverflowFailsAndRefunds(t *testing.T) {
	h := setup(t)
	creator := testAddress("overflow-creator")
	recipient := testAddress("overflow-recipient")
	schedule := createSchedule(t, h, creator, recipient, 102, 10, 2, "17")
	creatorAfterEscrow := new(big.Int).Set(h.bank.accountBalance(h.ctx, creator))

	delayedCtx := h.ctx.WithBlockHeight(int64(types.MaxSDKBlockHeight - 5))
	processed, err := h.keeper.ProcessDueSchedules(delayedCtx)
	require.NoError(t, err)
	require.Equal(t, uint32(1), processed)
	stored, _ := h.keeper.GetSchedule(delayedCtx, schedule.Id)
	require.Equal(t, types.ScheduleStatus_SCHEDULE_STATUS_FAILED, stored.Status)
	require.Equal(t, types.FailureCodeNextHeight, stored.TerminalReason)
	require.Equal(t, uint32(1), stored.ExecutionCount)
	require.Equal(t, "0", h.bank.accountBalance(delayedCtx, recipient).String())
	require.Equal(
		t,
		"200034",
		new(big.Int).Sub(h.bank.accountBalance(delayedCtx, creator), creatorAfterEscrow).String(),
	)
	receipt, found := h.keeper.GetReceipt(delayedCtx, schedule.Id, 1)
	require.True(t, found)
	require.Equal(t, types.ExecutionOutcome_EXECUTION_OUTCOME_FAILED_AND_REFUNDED, receipt.Outcome)
	require.Equal(t, types.FailureCodeNextHeight, receipt.FailureCode)
	require.Zero(t, h.keeper.TotalEscrow(delayedCtx).Sign())
}

func TestAddressesAreCanonicalizedAcrossMessagesAndQueries(t *testing.T) {
	h := setup(t)
	creator := testAddress("canonical-creator")
	recipient := testAddress("canonical-recipient")
	h.bank.fund(h.ctx, creator, 1_000_000)
	created, err := h.server.CreateSchedule(h.ctx, &types.MsgCreateSchedule{
		Creator:                strings.ToUpper(creator.String()),
		Recipient:              strings.ToUpper(recipient.String()),
		AmountPerExecutionUzrn: "5",
		FirstExecutionHeight:   110,
		IntervalBlocks:         0,
		ExecutionCount:         1,
	})
	require.NoError(t, err)
	stored, _ := h.keeper.GetSchedule(h.ctx, created.ScheduleId)
	require.Equal(t, creator.String(), stored.Creator)
	require.Equal(t, recipient.String(), stored.Recipient)

	query := keeper.NewQueryServerImpl(h.keeper)
	listed, err := query.SchedulesByCreator(h.ctx, &types.QuerySchedulesByCreatorRequest{
		Creator: strings.ToUpper(creator.String()),
		Limit:   1,
	})
	require.NoError(t, err)
	require.Len(t, listed.Schedules, 1)

	updated, err := h.server.UpdateSchedule(h.ctx, &types.MsgUpdateSchedule{
		Creator:                strings.ToUpper(creator.String()),
		ScheduleId:             created.ScheduleId,
		ExpectedRevision:       1,
		ExpectedExecutionCount: 0,
		Recipient:              strings.ToUpper(recipient.String()),
		AmountPerExecutionUzrn: "4",
		NextExecutionHeight:    111,
		IntervalBlocks:         0,
		RemainingExecutions:    1,
	})
	require.NoError(t, err)
	require.Equal(t, uint64(2), updated.Revision)
	_, err = h.server.CancelSchedule(h.ctx, &types.MsgCancelSchedule{
		Creator:                strings.ToUpper(creator.String()),
		ScheduleId:             created.ScheduleId,
		ExpectedRevision:       2,
		ExpectedExecutionCount: 0,
	})
	require.NoError(t, err)
}

func TestScheduleAndReceiptCursorPaginationHasNoGapsOrDuplicates(t *testing.T) {
	h := setup(t)
	creator := testAddress("page-creator")
	recipient := testAddress("page-recipient")
	var schedules []*types.Schedule
	for i := 0; i < 3; i++ {
		schedules = append(schedules, createSchedule(t, h, creator, recipient, 130+uint64(i), 0, 1, "1"))
	}
	query := keeper.NewQueryServerImpl(h.keeper)

	firstPage, err := query.SchedulesByCreator(h.ctx, &types.QuerySchedulesByCreatorRequest{
		Creator: creator.String(),
		Limit:   2,
	})
	require.NoError(t, err)
	require.Equal(t, []string{schedules[0].Id, schedules[1].Id}, []string{
		firstPage.Schedules[0].Id,
		firstPage.Schedules[1].Id,
	})
	require.Equal(t, schedules[1].Id, firstPage.NextKey)

	secondPage, err := query.SchedulesByCreator(h.ctx, &types.QuerySchedulesByCreatorRequest{
		Creator:    creator.String(),
		StartAfter: firstPage.NextKey,
		Limit:      2,
	})
	require.NoError(t, err)
	require.Len(t, secondPage.Schedules, 1)
	require.Equal(t, schedules[2].Id, secondPage.Schedules[0].Id)
	require.Empty(t, secondPage.NextKey)

	receiptHarness := setup(t)
	receiptSchedule := createSchedule(
		t,
		receiptHarness,
		testAddress("receipt-page-creator"),
		testAddress("receipt-page-recipient"),
		102,
		10,
		3,
		"1",
	)
	for _, height := range []int64{102, 112, 122} {
		processed, err := receiptHarness.keeper.ProcessDueSchedules(receiptHarness.ctx.WithBlockHeight(height))
		require.NoError(t, err)
		require.Equal(t, uint32(1), processed)
	}
	receiptQuery := keeper.NewQueryServerImpl(receiptHarness.keeper)
	firstReceipts, err := receiptQuery.ReceiptsBySchedule(receiptHarness.ctx, &types.QueryReceiptsByScheduleRequest{
		ScheduleId: receiptSchedule.Id,
		Limit:      2,
	})
	require.NoError(t, err)
	require.Len(t, firstReceipts.Receipts, 2)
	require.Equal(t, uint32(1), firstReceipts.Receipts[0].Sequence)
	require.Equal(t, uint32(2), firstReceipts.Receipts[1].Sequence)
	require.Equal(t, uint32(3), firstReceipts.NextSequence)

	secondReceipts, err := receiptQuery.ReceiptsBySchedule(receiptHarness.ctx, &types.QueryReceiptsByScheduleRequest{
		ScheduleId:    receiptSchedule.Id,
		StartSequence: firstReceipts.NextSequence,
		Limit:         2,
	})
	require.NoError(t, err)
	require.Len(t, secondReceipts.Receipts, 1)
	require.Equal(t, uint32(3), secondReceipts.Receipts[0].Sequence)
	require.Zero(t, secondReceipts.NextSequence)

	_, err = receiptQuery.ReceiptsBySchedule(receiptHarness.ctx, &types.QueryReceiptsByScheduleRequest{
		ScheduleId: receiptSchedule.Id,
		Limit:      receiptHarness.keeper.GetParams(receiptHarness.ctx).MaxQueryLimit + 1,
	})
	require.ErrorIs(t, err, types.ErrInvalidSchedule)
}

func TestActiveGenesisRebuildsIndexesAndExecutesRemainingOccurrence(t *testing.T) {
	h := setup(t)
	creator := testAddress("active-import-creator")
	recipient := testAddress("active-import-recipient")
	schedule := createSchedule(t, h, creator, recipient, 102, 10, 2, "19")
	processed, err := h.keeper.ProcessDueSchedules(h.ctx.WithBlockHeight(102))
	require.NoError(t, err)
	require.Equal(t, uint32(1), processed)
	exported := h.keeper.ExportGenesis(h.ctx.WithBlockHeight(102))
	require.NoError(t, exported.ValidateForChainID(h.ctx.ChainID()))
	require.Len(t, exported.Receipts, 1)

	tamperedCurrentTerms := proto.Clone(exported).(*types.GenesisState)
	tamperedCurrentTerms.Schedules[0].Recipient = testAddress("tampered-current").String()
	require.ErrorContains(t, tamperedCurrentTerms.Validate(), "at current revision does not match stored action terms")

	legitimateNewerTerms := proto.Clone(exported).(*types.GenesisState)
	legitimateNewerTerms.Schedules[0].Revision++
	legitimateNewerTerms.Schedules[0].Recipient = testAddress("amended-recipient").String()
	require.NoError(t, legitimateNewerTerms.ValidateForChainID(h.ctx.ChainID()))

	imported := setupWithGenesis(t, proto.Clone(exported).(*types.GenesisState))
	require.Equal(t, uint32(1), imported.keeper.CountActiveByCreator(imported.ctx, creator, 2))
	query := keeper.NewQueryServerImpl(imported.keeper)
	listed, err := query.SchedulesByCreator(
		imported.ctx,
		&types.QuerySchedulesByCreatorRequest{Creator: creator.String(), Limit: 2},
	)
	require.NoError(t, err)
	require.Len(t, listed.Schedules, 1)
	require.Equal(t, schedule.Id, listed.Schedules[0].Id)
	require.Equal(t, uint64(112), listed.Schedules[0].NextExecutionHeight)
	indexedReceipt, err := query.Receipt(imported.ctx, &types.QueryReceiptRequest{
		OccurrenceId: exported.Receipts[0].OccurrenceId,
	})
	require.NoError(t, err)
	require.Equal(t, uint32(1), indexedReceipt.Receipt.Sequence)

	dueCtx := imported.ctx.WithBlockHeight(112)
	processed, err = imported.keeper.ProcessDueSchedules(dueCtx)
	require.NoError(t, err)
	require.Equal(t, uint32(1), processed)
	stored, found := imported.keeper.GetSchedule(dueCtx, schedule.Id)
	require.True(t, found)
	require.Equal(t, types.ScheduleStatus_SCHEDULE_STATUS_COMPLETED, stored.Status)
	require.Equal(t, uint32(2), stored.ExecutionCount)
	require.Equal(t, "19", imported.bank.accountBalance(dueCtx, recipient).String())
	require.Zero(t, imported.keeper.TotalEscrow(dueCtx).Sign())
	require.Zero(t, imported.keeper.CountActiveByCreator(dueCtx, creator, 1))
}

func TestNonEmptyGenesisRoundTripAndReceiptTampering(t *testing.T) {
	h := setup(t)
	schedule := createSchedule(
		t,
		h,
		testAddress("export-creator"),
		testAddress("export-recipient"),
		102,
		10,
		2,
		"19",
	)
	for _, height := range []int64{102, 112} {
		_, err := h.keeper.ProcessDueSchedules(h.ctx.WithBlockHeight(height))
		require.NoError(t, err)
	}
	exported := h.keeper.ExportGenesis(h.ctx.WithBlockHeight(112))
	require.NoError(t, exported.ValidateForChainID(h.ctx.ChainID()))
	require.Len(t, exported.Schedules, 1)
	require.Len(t, exported.Receipts, 2)

	imported := setupWithGenesis(t, proto.Clone(exported).(*types.GenesisState))
	stored, found := imported.keeper.GetSchedule(imported.ctx, schedule.Id)
	require.True(t, found)
	require.Equal(t, types.ScheduleStatus_SCHEDULE_STATUS_COMPLETED, stored.Status)
	for sequence := uint32(1); sequence <= 2; sequence++ {
		_, found := imported.keeper.GetReceipt(imported.ctx, schedule.Id, sequence)
		require.True(t, found)
	}

	tamperedAction := proto.Clone(exported).(*types.GenesisState)
	tamperedAction.Receipts[0].ActionSha256 = strings.Repeat("0", 64)
	require.ErrorContains(t, tamperedAction.Validate(), "action_sha256 does not match")

	tamperedOccurrence := proto.Clone(exported).(*types.GenesisState)
	tamperedOccurrence.Receipts[0].OccurrenceId = strings.Repeat("0", 64)
	require.NoError(t, tamperedOccurrence.Validate())
	require.ErrorContains(
		t,
		tamperedOccurrence.ValidateForChainID(h.ctx.ChainID()),
		"chain-bound occurrence",
	)

	missingReceipt := proto.Clone(exported).(*types.GenesisState)
	missingReceipt.Receipts = missingReceipt.Receipts[:1]
	require.ErrorContains(t, missingReceipt.Validate(), "does not equal receipt count")

	tamperedTerminalRevision := proto.Clone(exported).(*types.GenesisState)
	tamperedTerminalRevision.Schedules[0].Revision++
	require.ErrorContains(t, tamperedTerminalRevision.Validate(), "terminal receipt revision")

	tamperedTerminalTerms := proto.Clone(exported).(*types.GenesisState)
	tamperedTerminalTerms.Schedules[0].Recipient = testAddress("tampered-terminal").String()
	require.ErrorContains(t, tamperedTerminalTerms.Validate(), "at current revision does not match stored action terms")

	reservedFailureReceipt := proto.Clone(exported.Receipts[len(exported.Receipts)-1]).(*types.ExecutionReceipt)
	reservedFailureReceipt.Outcome = types.ExecutionOutcome_EXECUTION_OUTCOME_FAILED_AND_REFUNDED
	reservedFailureReceipt.FailureCode = types.FailureCodeEscrowInvariant
	require.False(t, types.IsKnownFailureCode(types.FailureCodeEscrowInvariant))
	require.ErrorContains(t, types.ValidateReceipt(reservedFailureReceipt), "known failure_code")
}

func TestActiveGenesisRejectsBlockedParticipant(t *testing.T) {
	h := setup(t)
	recipient := testAddress("blocked-import")
	createSchedule(t, h, testAddress("import-creator"), recipient, 110, 0, 1, "23")
	exported := h.keeper.ExportGenesis(h.ctx)

	require.PanicsWithValue(
		t,
		"invalid schedule genesis: active recipient "+recipient.String()+" is blocked",
		func() {
			setupWithGenesisAndBlocked(
				t,
				proto.Clone(exported).(*types.GenesisState),
				[]sdk.AccAddress{recipient},
			)
		},
	)
}
