package keeper

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/big"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/schedule/types"
)

type Keeper struct {
	storeService store.KVStoreService
	cdc          codec.BinaryCodec
	bankKeeper   types.BankKeeper
	emergency    types.EmergencyKeeper
	authority    string
}

func NewKeeper(
	storeService store.KVStoreService,
	cdc codec.BinaryCodec,
	bankKeeper types.BankKeeper,
	emergency types.EmergencyKeeper,
	authority string,
) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid schedule authority: %v", err))
	}
	if emergency == nil {
		panic("schedule emergency keeper is required")
	}
	return Keeper{
		storeService: storeService,
		cdc:          cdc,
		bankKeeper:   bankKeeper,
		emergency:    emergency,
		authority:    authority,
	}
}

func (k Keeper) Authority() string { return k.authority }

func (k Keeper) Logger(ctx context.Context) log.Logger {
	return sdk.UnwrapSDKContext(ctx).Logger().With("module", "x/"+types.ModuleName)
}

func (k Keeper) SetParams(ctx context.Context, params *types.Params) error {
	if err := params.Validate(); err != nil {
		return err
	}
	k.mustSet(ctx, types.ParamsKey, k.mustMarshal(params))
	return nil
}

func (k Keeper) GetParams(ctx context.Context) *types.Params {
	bz := k.mustGet(ctx, types.ParamsKey)
	if bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	k.mustUnmarshal(bz, &params)
	return &params
}

func (k Keeper) AllocateScheduleID(ctx context.Context) (string, error) {
	next := k.PeekNextScheduleID(ctx)
	if next == ^uint64(0) {
		return "", types.ErrScheduleLimit.Wrap("schedule id space exhausted")
	}
	k.SetNextScheduleID(ctx, next+1)
	return types.FormatScheduleID(next), nil
}

func (k Keeper) PeekNextScheduleID(ctx context.Context) uint64 {
	bz := k.mustGet(ctx, types.ScheduleCounterKey)
	if bz == nil {
		return 1
	}
	if len(bz) != 8 {
		panic("corrupt schedule counter")
	}
	next := binary.BigEndian.Uint64(bz)
	if next == 0 {
		panic("corrupt zero schedule counter")
	}
	return next
}

func (k Keeper) SetNextScheduleID(ctx context.Context, next uint64) {
	if next == 0 {
		panic("schedule counter must be positive")
	}
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, next)
	k.mustSet(ctx, types.ScheduleCounterKey, bz)
}

func (k Keeper) SetSchedule(ctx context.Context, schedule *types.Schedule) {
	k.mustSet(ctx, types.ScheduleKey(schedule.Id), k.mustMarshal(schedule))
}

func (k Keeper) GetSchedule(ctx context.Context, id string) (*types.Schedule, bool) {
	bz := k.mustGet(ctx, types.ScheduleKey(id))
	if bz == nil {
		return nil, false
	}
	var schedule types.Schedule
	k.mustUnmarshal(bz, &schedule)
	return &schedule, true
}

func (k Keeper) IterateSchedules(ctx context.Context, cb func(*types.Schedule) bool) {
	k.iterate(ctx, types.ScheduleKeyPrefix, types.PrefixEndBytes(types.ScheduleKeyPrefix), func(_ []byte, value []byte) bool {
		var schedule types.Schedule
		k.mustUnmarshal(value, &schedule)
		return cb(&schedule)
	})
}

func (k Keeper) GetAllSchedules(ctx context.Context) []*types.Schedule {
	var schedules []*types.Schedule
	k.IterateSchedules(ctx, func(schedule *types.Schedule) bool {
		schedules = append(schedules, schedule)
		return false
	})
	return schedules
}

func (k Keeper) AddScheduleIndexes(ctx context.Context, schedule *types.Schedule) {
	creator := mustAddress(schedule.Creator)
	k.mustSet(ctx, types.CreatorKey(creator, schedule.Id), []byte{1})
	if schedule.Status == types.ScheduleStatus_SCHEDULE_STATUS_ACTIVE {
		k.mustSet(ctx, types.ActiveCreatorKey(creator, schedule.Id), []byte{1})
		k.mustSet(ctx, types.DueKey(schedule.NextExecutionHeight, schedule.Id), []byte{1})
	}
}

func (k Keeper) RemoveActiveIndexes(ctx context.Context, schedule *types.Schedule) {
	creator := mustAddress(schedule.Creator)
	k.mustDelete(ctx, types.ActiveCreatorKey(creator, schedule.Id))
	if schedule.NextExecutionHeight != 0 {
		k.mustDelete(ctx, types.DueKey(schedule.NextExecutionHeight, schedule.Id))
	}
}

func (k Keeper) CountActiveByCreator(ctx context.Context, creator sdk.AccAddress, stopAfter uint32) uint32 {
	var count uint32
	prefix := types.ActiveCreatorAddressPrefix(creator)
	k.iterate(ctx, prefix, types.PrefixEndBytes(prefix), func(_, _ []byte) bool {
		count++
		return stopAfter > 0 && count >= stopAfter
	})
	return count
}

func (k Keeper) SetReceipt(ctx context.Context, receipt *types.ExecutionReceipt) {
	key := types.ReceiptKey(receipt.ScheduleId, receipt.Sequence)
	if k.mustGet(ctx, key) != nil || k.mustGet(ctx, types.OccurrenceKey(receipt.OccurrenceId)) != nil {
		panic(fmt.Sprintf("duplicate schedule occurrence %s", receipt.OccurrenceId))
	}
	k.mustSet(ctx, key, k.mustMarshal(receipt))
	k.mustSet(ctx, types.OccurrenceKey(receipt.OccurrenceId), key)
}

func (k Keeper) GetReceipt(ctx context.Context, scheduleID string, sequence uint32) (*types.ExecutionReceipt, bool) {
	return k.getReceiptAtKey(ctx, types.ReceiptKey(scheduleID, sequence))
}

func (k Keeper) GetReceiptByOccurrence(ctx context.Context, occurrenceID string) (*types.ExecutionReceipt, bool) {
	key := k.mustGet(ctx, types.OccurrenceKey(occurrenceID))
	if key == nil {
		return nil, false
	}
	return k.getReceiptAtKey(ctx, key)
}

func (k Keeper) getReceiptAtKey(ctx context.Context, key []byte) (*types.ExecutionReceipt, bool) {
	bz := k.mustGet(ctx, key)
	if bz == nil {
		return nil, false
	}
	var receipt types.ExecutionReceipt
	k.mustUnmarshal(bz, &receipt)
	return &receipt, true
}

func (k Keeper) GetAllReceipts(ctx context.Context) []*types.ExecutionReceipt {
	var receipts []*types.ExecutionReceipt
	k.iterate(ctx, types.ReceiptKeyPrefix, types.PrefixEndBytes(types.ReceiptKeyPrefix), func(_ []byte, value []byte) bool {
		var receipt types.ExecutionReceipt
		k.mustUnmarshal(value, &receipt)
		receipts = append(receipts, &receipt)
		return false
	})
	return receipts
}

func (k Keeper) TotalEscrow(ctx context.Context) *big.Int {
	bz := k.mustGet(ctx, types.TotalEscrowKey)
	if bz == nil {
		return new(big.Int)
	}
	value, err := types.ParseNonNegativeAmount(string(bz))
	if err != nil {
		panic(fmt.Sprintf("corrupt total schedule escrow: %v", err))
	}
	return value
}

func (k Keeper) SetTotalEscrow(ctx context.Context, amount *big.Int) {
	if amount.Sign() < 0 {
		panic("negative total schedule escrow")
	}
	k.mustSet(ctx, types.TotalEscrowKey, []byte(amount.String()))
}

func (k Keeper) AddTotalEscrow(ctx context.Context, delta *big.Int) error {
	next, err := k.nextTotalEscrow(ctx, delta)
	if err != nil {
		return err
	}
	k.SetTotalEscrow(ctx, next)
	return nil
}

// nextTotalEscrow validates a liability change without writing it. Message
// handlers call this before asking x/bank to add coins to the module account;
// otherwise the bank balance itself could overflow the SDK integer range
// before the tracked-liability update gets a chance to reject the change.
func (k Keeper) nextTotalEscrow(ctx context.Context, delta *big.Int) (*big.Int, error) {
	next := new(big.Int).Add(k.TotalEscrow(ctx), delta)
	if next.Sign() < 0 {
		return nil, types.ErrEscrowInvariant.Wrap("total liability would become negative")
	}
	if next.BitLen() > 256 {
		return nil, types.ErrEscrowInvariant.Wrap("total liability exceeds the SDK 256-bit coin range")
	}
	return next, nil
}

func (k Keeper) AssertEscrowInvariant(ctx context.Context) error {
	moduleAddr := authtypes.NewModuleAddress(types.ModuleName)
	liability := k.TotalEscrow(ctx)
	expected := sdk.NewCoins()
	if liability.Sign() > 0 {
		expected = sdk.NewCoins(sdk.NewCoin(types.Denom, sdkmath.NewIntFromBigInt(liability)))
	}
	balances := k.bankKeeper.GetAllBalances(ctx, moduleAddr)
	if !balances.Equal(expected) {
		return types.ErrEscrowInvariant.Wrapf(
			"module balances %s != exact tracked liability %s",
			balances,
			expected,
		)
	}
	return nil
}

func (k Keeper) InitGenesis(ctx context.Context, genesis *types.GenesisState) {
	chainID := sdk.UnwrapSDKContext(ctx).ChainID()
	if err := genesis.ValidateForChainID(chainID); err != nil {
		panic(fmt.Sprintf("invalid schedule genesis: %v", err))
	}
	for _, schedule := range genesis.Schedules {
		if schedule.Status != types.ScheduleStatus_SCHEDULE_STATUS_ACTIVE {
			continue
		}
		creator := mustAddress(schedule.Creator)
		recipient := mustAddress(schedule.Recipient)
		if k.bankKeeper.BlockedAddr(creator) {
			panic(fmt.Sprintf("invalid schedule genesis: active creator %s cannot receive refunds", schedule.Creator))
		}
		if k.bankKeeper.BlockedAddr(recipient) {
			panic(fmt.Sprintf("invalid schedule genesis: active recipient %s is blocked", schedule.Recipient))
		}
	}
	if err := k.SetParams(ctx, genesis.Params); err != nil {
		panic(err)
	}
	k.SetNextScheduleID(ctx, genesis.NextScheduleId)
	total, _ := types.ParseNonNegativeAmount(genesis.TotalEscrowUzrn)
	k.SetTotalEscrow(ctx, total)
	for _, schedule := range genesis.Schedules {
		k.SetSchedule(ctx, schedule)
		k.AddScheduleIndexes(ctx, schedule)
	}
	for _, receipt := range genesis.Receipts {
		k.SetReceipt(ctx, receipt)
	}
	if err := k.AssertEscrowInvariant(ctx); err != nil {
		panic(err)
	}
}

func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	genesis := &types.GenesisState{
		Params:          k.GetParams(ctx),
		Schedules:       k.GetAllSchedules(ctx),
		Receipts:        k.GetAllReceipts(ctx),
		NextScheduleId:  k.PeekNextScheduleID(ctx),
		TotalEscrowUzrn: k.TotalEscrow(ctx).String(),
	}
	chainID := sdk.UnwrapSDKContext(ctx).ChainID()
	if err := genesis.ValidateForChainID(chainID); err != nil {
		panic(fmt.Sprintf("refuse invalid schedule genesis export: %v", err))
	}
	if err := k.AssertEscrowInvariant(ctx); err != nil {
		panic(fmt.Sprintf("refuse schedule genesis export with broken escrow invariant: %v", err))
	}
	return genesis
}

func (k Keeper) mustMarshal(message proto.Message) []byte {
	bz, err := proto.Marshal(message)
	if err != nil {
		panic(fmt.Sprintf("marshal schedule state: %v", err))
	}
	return bz
}

func (k Keeper) mustUnmarshal(bz []byte, message proto.Message) {
	if err := proto.Unmarshal(bz, message); err != nil {
		panic(fmt.Sprintf("unmarshal schedule state: %v", err))
	}
}

func (k Keeper) mustGet(ctx context.Context, key []byte) []byte {
	bz, err := k.storeService.OpenKVStore(ctx).Get(key)
	if err != nil {
		panic(fmt.Sprintf("read schedule state: %v", err))
	}
	return bz
}

func (k Keeper) mustSet(ctx context.Context, key, value []byte) {
	if err := k.storeService.OpenKVStore(ctx).Set(key, value); err != nil {
		panic(fmt.Sprintf("write schedule state: %v", err))
	}
}

func (k Keeper) mustDelete(ctx context.Context, key []byte) {
	if err := k.storeService.OpenKVStore(ctx).Delete(key); err != nil {
		panic(fmt.Sprintf("delete schedule state: %v", err))
	}
}

func (k Keeper) iterate(ctx context.Context, start, end []byte, cb func([]byte, []byte) bool) {
	iterator, err := k.storeService.OpenKVStore(ctx).Iterator(start, end)
	if err != nil {
		panic(fmt.Sprintf("iterate schedule state: %v", err))
	}
	defer func() {
		if err := iterator.Close(); err != nil {
			panic(fmt.Sprintf("close schedule state iterator: %v", err))
		}
	}()
	for ; iterator.Valid(); iterator.Next() {
		key := append([]byte(nil), iterator.Key()...)
		value := append([]byte(nil), iterator.Value()...)
		if cb(key, value) {
			return
		}
	}
	if err := scheduleIteratorTerminalError(iterator); err != nil {
		panic(fmt.Sprintf("iterate schedule state: %v", err))
	}
}

// scheduleIteratorTerminalError follows the database iterator contract while
// tolerating only the Cosmos SDK cache iterators' exact normal-exhaustion
// artifacts. All other visible traversal failures must stop the local node;
// treating a truncated active or due index as complete could diverge state.
func scheduleIteratorTerminalError(iterator interface {
	Valid() bool
	Error() error
}) error {
	err := iterator.Error()
	if err == nil {
		return nil
	}
	if !iterator.Valid() {
		switch err.Error() {
		case "invalid cacheMergeIterator", "invalid memIterator":
			return nil
		}
	}
	return err
}

func mustAddress(address string) sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		panic(fmt.Sprintf("corrupt schedule address %q: %v", address, err))
	}
	return addr
}
