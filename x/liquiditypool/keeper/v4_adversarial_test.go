package keeper_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"reflect"
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
	sdkquery "github.com/cosmos/cosmos-sdk/types/query"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/liquiditypool/keeper"
	"github.com/zerone-chain/zerone/x/liquiditypool/types"
)

const v4TestBlockHeight int64 = 777

type v4Harness struct {
	msgServer   types.MsgServer
	queryServer types.QueryServer
	keeper      keeper.Keeper
	ctx         sdk.Context
	bank        *mockBankKeeper
	store       corestore.KVStoreService
	authority   string
}

type v4BankSnapshot struct {
	balances       map[string]map[string]int64
	moduleBalances map[string]map[string]int64
}

type v4InvariantBankKeeper struct {
	*mockBankKeeper
}

// These methods keep the legacy package-wide bank mock aligned with the
// production BankKeeper boundary added by v4. Defining them here avoids
// changing the pre-existing test harness while giving supply-integrity checks
// a real accounting view.
func (m *mockBankKeeper) GetSupply(_ context.Context, denom string) sdk.Coin {
	total := new(big.Int)
	for _, coins := range m.balances {
		total.Add(total, big.NewInt(coins[denom]))
	}
	for _, coins := range m.moduleBalances {
		total.Add(total, big.NewInt(coins[denom]))
	}
	return sdk.NewCoin(denom, sdkmath.NewIntFromBigInt(total))
}

func (m *mockBankKeeper) IsSendEnabledCoins(_ context.Context, coins ...sdk.Coin) error {
	for _, coin := range coins {
		if m.sendDisabled[coin.Denom] {
			return fmt.Errorf("%s is send-disabled", coin.Denom)
		}
	}
	return nil
}

func (m *v4InvariantBankKeeper) GetBalance(
	ctx context.Context,
	address sdk.AccAddress,
	denom string,
) sdk.Coin {
	if address.String() == authtypes.NewModuleAddress(types.ModuleName).String() {
		amount := int64(0)
		if balances := m.moduleBalances[types.ModuleName]; balances != nil {
			amount = balances[denom]
		}
		return sdk.NewCoin(denom, sdkmath.NewInt(amount))
	}
	return m.mockBankKeeper.GetBalance(ctx, address, denom)
}

func newV4Harness(t *testing.T) v4Harness {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		t.Fatalf("load test store: %v", err)
	}

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	bank := newMockBankKeeper()
	authority := sdk.AccAddress(bytes.Repeat([]byte{0x41}, 20)).String()
	k := keeper.NewKeeper(cdc, runtime.NewKVStoreService(storeKey), bank, authority)
	ctx := sdk.NewContext(
		stateStore,
		cmtproto.Header{Height: v4TestBlockHeight, ChainID: testChainID},
		false,
		log.NewNopLogger(),
	)
	params := types.DefaultParams()
	params.AllowedPoolDenoms = []string{
		"uatom",
		"uosmo",
		"ujuno",
		"ucreatea",
		"ucreateb",
	}
	params.PoolCreators = []string{authority}
	k.SetParams(ctx, params)

	return v4Harness{
		msgServer:   keeper.NewMsgServerImpl(k),
		queryServer: keeper.NewQueryServerImpl(k),
		keeper:      k,
		ctx:         ctx,
		bank:        bank,
		store:       runtime.NewKVStoreService(storeKey),
		authority:   authority,
	}
}

func newV4InvariantHarness(t *testing.T) v4Harness {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		t.Fatalf("load invariant test store: %v", err)
	}

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	bank := newMockBankKeeper()
	invariantBank := &v4InvariantBankKeeper{mockBankKeeper: bank}
	authority := sdk.AccAddress(bytes.Repeat([]byte{0x41}, 20)).String()
	storeService := runtime.NewKVStoreService(storeKey)
	k := keeper.NewKeeper(cdc, storeService, invariantBank, authority)
	ctx := sdk.NewContext(
		stateStore,
		cmtproto.Header{Height: v4TestBlockHeight, ChainID: testChainID},
		false,
		log.NewNopLogger(),
	)
	params := types.DefaultParams()
	params.AllowedPoolDenoms = []string{"uatom"}
	params.PoolCreators = []string{authority}
	k.SetParams(ctx, params)

	return v4Harness{
		msgServer:   keeper.NewMsgServerImpl(k),
		queryServer: keeper.NewQueryServerImpl(k),
		keeper:      k,
		ctx:         ctx,
		bank:        bank,
		store:       storeService,
		authority:   authority,
	}
}

func v4Address(seed byte) string {
	return sdk.AccAddress(bytes.Repeat([]byte{seed}, 20)).String()
}

func v4Fund(bank *mockBankKeeper, address string, denoms ...string) {
	for _, denom := range denoms {
		bank.setBalance(address, denom, 1_000_000_000_000_000)
	}
}

func v4CreatePool(
	t *testing.T,
	h v4Harness,
	denomB string,
	amountA string,
	amountB string,
) *types.Pool {
	t.Helper()

	v4Fund(h.bank, h.authority, types.ZRNDenom, denomB)
	response, err := h.msgServer.CreatePool(h.ctx, &types.MsgCreatePool{
		Creator: h.authority,
		DenomA:  types.ZRNDenom,
		DenomB:  denomB,
		AmountA: amountA,
		AmountB: amountB,
	})
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	pool, found := h.keeper.GetPool(h.ctx, response.PoolId)
	if !found {
		t.Fatalf("created pool %q missing", response.PoolId)
	}
	return pool
}

func v4ReadmitPoolDenom(h v4Harness, denom string) {
	params := h.keeper.GetParams(h.ctx)
	for _, admitted := range params.AllowedPoolDenoms {
		if admitted == denom {
			return
		}
	}
	params.AllowedPoolDenoms = append(params.AllowedPoolDenoms, denom)
	h.keeper.SetParams(h.ctx, params)
}

func v4CopyBalances(source map[string]map[string]int64) map[string]map[string]int64 {
	result := make(map[string]map[string]int64, len(source))
	for owner, coins := range source {
		result[owner] = make(map[string]int64, len(coins))
		for denom, amount := range coins {
			result[owner][denom] = amount
		}
	}
	return result
}

func v4SnapshotBank(bank *mockBankKeeper) v4BankSnapshot {
	return v4BankSnapshot{
		balances:       v4CopyBalances(bank.balances),
		moduleBalances: v4CopyBalances(bank.moduleBalances),
	}
}

func v4AssertBankUnchanged(t *testing.T, bank *mockBankKeeper, before v4BankSnapshot) {
	t.Helper()
	after := v4SnapshotBank(bank)
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("bank state mutated on rejected operation:\nbefore=%#v\nafter=%#v", before, after)
	}
}

func v4AssertPoolUnchanged(t *testing.T, h v4Harness, before *types.Pool) {
	t.Helper()
	after, found := h.keeper.GetPool(h.ctx, before.PoolId)
	if !found {
		t.Fatalf("pool %q disappeared on rejected operation", before.PoolId)
	}
	if !proto.Equal(after, before) {
		t.Fatalf("pool mutated on rejected operation:\nbefore=%s\nafter=%s", before, after)
	}
}

func v4CountTWAPObservations(h v4Harness, poolID string) int {
	count := 0
	h.keeper.IterateTWAPObservations(h.ctx, poolID, func(_ *types.TWAPObservation) bool {
		count++
		return false
	})
	return count
}

func v4AssertErrorWithoutPanic(t *testing.T, operation func() error) {
	t.Helper()
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("operation panicked instead of returning an error: %v", recovered)
		}
	}()
	if err := operation(); err == nil {
		t.Fatal("expected operation to fail")
	}
}

func TestV4SwapRejectsZeroOutputWithoutMutation(t *testing.T) {
	h := newV4Harness(t)
	pool := v4CreatePool(t, h, "uatom", "10000000000", "1")
	sender := v4Address(0x42)
	v4Fund(h.bank, sender, types.ZRNDenom)

	poolBefore := proto.Clone(pool).(*types.Pool)
	bankBefore := v4SnapshotBank(h.bank)

	if _, err := h.queryServer.SimulateSwap(h.ctx, &types.QuerySimulateSwapRequest{
		PoolId:        pool.PoolId,
		TokenInDenom:  types.ZRNDenom,
		TokenInAmount: "1",
	}); err == nil {
		t.Fatal("zero-output quote succeeded")
	}
	v4AssertPoolUnchanged(t, h, poolBefore)
	v4AssertBankUnchanged(t, h.bank, bankBefore)

	if _, err := h.msgServer.Swap(h.ctx, &types.MsgSwap{
		Sender:        sender,
		PoolId:        pool.PoolId,
		TokenInDenom:  types.ZRNDenom,
		TokenInAmount: "1",
	}); err == nil {
		t.Fatal("zero-output swap succeeded")
	}
	v4AssertPoolUnchanged(t, h, poolBefore)
	v4AssertBankUnchanged(t, h.bank, bankBefore)
}

func TestV4AddLiquidityRejectsZeroMintDonationWithoutMutation(t *testing.T) {
	h := newV4Harness(t)
	pool := v4CreatePool(t, h, "uatom", "10000000000", "1")
	sender := v4Address(0x43)
	v4Fund(h.bank, sender, pool.DenomA, pool.DenomB)

	poolBefore := proto.Clone(pool).(*types.Pool)
	bankBefore := v4SnapshotBank(h.bank)
	if _, err := h.msgServer.AddLiquidity(h.ctx, &types.MsgAddLiquidity{
		Sender:  sender,
		PoolId:  pool.PoolId,
		AmountA: "1",
		AmountB: "1",
	}); err == nil {
		t.Fatal("deposit that mints zero LP tokens succeeded")
	}
	v4AssertPoolUnchanged(t, h, poolBefore)
	v4AssertBankUnchanged(t, h.bank, bankBefore)
}

func TestV4FinalLPExitTombstonesPoolAndCannotBeReseeded(t *testing.T) {
	h := newV4Harness(t)
	pool := v4CreatePool(t, h, "uatom", "10000000000", "4000000000")
	oldPoolID := pool.PoolId
	oldLPDenom := pool.LpDenom
	oldSupply := pool.LpTokenSupply
	if observations := v4CountTWAPObservations(h, oldPoolID); observations == 0 {
		t.Fatal("new pool has no TWAP checkpoint to retire")
	}
	for _, admitted := range h.keeper.GetParams(h.ctx).AllowedPoolDenoms {
		if admitted == pool.DenomB {
			t.Fatalf("successful creation did not consume one-shot denom grant %s", admitted)
		}
	}

	response, err := h.msgServer.RemoveLiquidity(h.ctx, &types.MsgRemoveLiquidity{
		Sender:     h.authority,
		PoolId:     oldPoolID,
		LpTokens:   oldSupply,
		MinAmountA: "1",
		MinAmountB: "1",
	})
	if err != nil {
		t.Fatalf("full LP exit: %v", err)
	}
	if response.AmountA != pool.ReserveA || response.AmountB != pool.ReserveB {
		t.Fatalf(
			"full exit returned %s/%s, want %s/%s",
			response.AmountA,
			response.AmountB,
			pool.ReserveA,
			pool.ReserveB,
		)
	}

	tombstone, found := h.keeper.GetPool(h.ctx, oldPoolID)
	if !found {
		t.Fatal("final exit deleted the pool instead of retaining a tombstone")
	}
	if tombstone.Status != types.PoolStatus_POOL_STATUS_CLOSED {
		t.Fatalf("final pool status = %s, want CLOSED", tombstone.Status)
	}
	if tombstone.ReserveA != "0" || tombstone.ReserveB != "0" || tombstone.LpTokenSupply != "0" {
		t.Fatalf(
			"closed tombstone retained economic state: reserves=%s/%s supply=%s",
			tombstone.ReserveA,
			tombstone.ReserveB,
			tombstone.LpTokenSupply,
		)
	}
	if tombstone.ClosedAtBlock != uint64(v4TestBlockHeight) {
		t.Fatalf("closed_at_block = %d, want %d", tombstone.ClosedAtBlock, v4TestBlockHeight)
	}
	if tombstone.Locked {
		t.Fatal("closed tombstone retained its transaction lock")
	}
	if indexed := h.keeper.GetPoolByDenoms(h.ctx, pool.DenomA, pool.DenomB); indexed != nil {
		t.Fatalf("closed pool remains in active pair index: %s", indexed.PoolId)
	}
	if _, found := h.keeper.GetTWAPAccumulator(h.ctx, oldPoolID); found {
		t.Fatal("closed pool retained an active TWAP accumulator")
	}
	if !h.keeper.IsTWAPHistoryDeletionScheduled(h.ctx, oldPoolID) {
		t.Fatal("closed pool did not schedule bounded TWAP history cleanup")
	}
	if observations := v4CountTWAPObservations(h, oldPoolID); observations == 0 {
		t.Fatal("final exit synchronously deleted history instead of using bounded cleanup")
	}
	if err := h.keeper.ExportGenesis(h.ctx).Validate(); err != nil {
		t.Fatalf("queued closed-pool history leaked into exported genesis: %v", err)
	}
	h.keeper.ProcessTWAPGarbageCollection(h.ctx)
	if observations := v4CountTWAPObservations(h, oldPoolID); observations != 0 {
		t.Fatalf("bounded cleanup left %d observations from one-checkpoint pool", observations)
	}
	if h.keeper.IsTWAPHistoryDeletionScheduled(h.ctx, oldPoolID) {
		t.Fatal("completed TWAP history cleanup retained its queue marker")
	}

	sender := v4Address(0x44)
	v4Fund(h.bank, sender, pool.DenomA, pool.DenomB)
	bankBefore := v4SnapshotBank(h.bank)
	if _, err := h.msgServer.AddLiquidity(h.ctx, &types.MsgAddLiquidity{
		Sender:  sender,
		PoolId:  oldPoolID,
		AmountA: "10000000000",
		AmountB: "4000000000",
	}); err == nil {
		t.Fatal("permissionless liquidity reseeded a closed tombstone")
	}
	if got, _ := h.keeper.GetPool(h.ctx, oldPoolID); !proto.Equal(got, tombstone) {
		t.Fatalf("rejected reseed mutated tombstone:\nbefore=%s\nafter=%s", tombstone, got)
	}
	if _, err := h.msgServer.Swap(h.ctx, &types.MsgSwap{
		Sender:        sender,
		PoolId:        oldPoolID,
		TokenInDenom:  pool.DenomA,
		TokenInAmount: "1",
	}); err == nil {
		t.Fatal("swap succeeded against a closed tombstone")
	}
	if _, err := h.msgServer.RemoveLiquidity(h.ctx, &types.MsgRemoveLiquidity{
		Sender:   sender,
		PoolId:   oldPoolID,
		LpTokens: "1",
	}); err == nil {
		t.Fatal("LP exit succeeded against a closed tombstone")
	}
	if _, err := h.msgServer.SetPoolStatus(h.ctx, &types.MsgSetPoolStatus{
		Authority: h.authority,
		PoolId:    oldPoolID,
		Status:    types.PoolStatus_POOL_STATUS_ACTIVE,
	}); err == nil {
		t.Fatal("governance reopened a closed tombstone")
	}
	if got, _ := h.keeper.GetPool(h.ctx, oldPoolID); !proto.Equal(got, tombstone) {
		t.Fatalf("closed-pool operation mutated tombstone:\nbefore=%s\nafter=%s", tombstone, got)
	}
	v4AssertBankUnchanged(t, h.bank, bankBefore)

	v4ReadmitPoolDenom(h, pool.DenomB)
	replacement := v4CreatePool(t, h, pool.DenomB, "10000000000", "4000000000")
	if replacement.PoolId == oldPoolID {
		t.Fatalf("governed replacement reused closed pool ID %q", oldPoolID)
	}
	if replacement.LpDenom == oldLPDenom {
		t.Fatalf("governed replacement reused LP denom %q", oldLPDenom)
	}
	if replacement.Status != types.PoolStatus_POOL_STATUS_ACTIVE {
		t.Fatalf("replacement status = %s, want ACTIVE", replacement.Status)
	}
	if indexed := h.keeper.GetPoolByDenoms(h.ctx, pool.DenomA, pool.DenomB); indexed == nil || indexed.PoolId != replacement.PoolId {
		t.Fatalf("active pair index does not select replacement pool %q", replacement.PoolId)
	}
}

func TestV4MaxRetentionFinalExitUsesBoundedTWAPGarbageCollection(t *testing.T) {
	h := newV4Harness(t)
	h.ctx = h.ctx.WithBlockHeight(20_000)
	params := h.keeper.GetParams(h.ctx)
	params.TwapWindowBlocks = types.MaxTWAPWindowBlocks
	h.keeper.SetParams(h.ctx, params)
	pool := v4CreatePool(t, h, "uatom", "10000000000", "4000000000")

	acc, found := h.keeper.GetTWAPAccumulator(h.ctx, pool.PoolId)
	if !found {
		t.Fatal("created pool missing accumulator")
	}
	acc.StartBlock = 10_000
	acc.LastBlock = 20_000
	h.keeper.SetTWAPAccumulator(h.ctx, acc)
	for height := uint64(10_000); height <= 20_000; height++ {
		h.keeper.SetTWAPObservation(h.ctx, &types.TWAPObservation{
			PoolId:       pool.PoolId,
			BlockHeight:  height,
			CumPriceAToB: "0",
			CumPriceBToA: "0",
		})
	}
	if got := v4CountTWAPObservations(h, pool.PoolId); got != 10_001 {
		t.Fatalf("test fixture retained %d observations, want 10001", got)
	}

	const txGasLimit = uint64(11_111_111)
	gasMeter := storetypes.NewGasMeter(txGasLimit)
	exitCtx := h.ctx.WithGasMeter(gasMeter)
	if _, err := h.msgServer.RemoveLiquidity(exitCtx, &types.MsgRemoveLiquidity{
		Sender:     h.authority,
		PoolId:     pool.PoolId,
		LpTokens:   pool.LpTokenSupply,
		MinAmountA: "1",
		MinAmountB: "1",
	}); err != nil {
		t.Fatalf("max-retention final exit failed: %v", err)
	}
	if consumed := gasMeter.GasConsumed(); consumed >= txGasLimit {
		t.Fatalf("final exit consumed %d gas against %d limit", consumed, txGasLimit)
	}
	if got := v4CountTWAPObservations(h, pool.PoolId); got != 10_001 {
		t.Fatalf("final exit synchronously touched checkpoint set: got %d", got)
	}

	h.keeper.ProcessTWAPGarbageCollection(h.ctx)
	afterOneBlock := v4CountTWAPObservations(h, pool.PoolId)
	if afterOneBlock <= 0 || afterOneBlock >= 10_001 {
		t.Fatalf("first bounded cleanup left %d observations", afterOneBlock)
	}
	for block := 0; block < 200 && h.keeper.IsTWAPHistoryDeletionScheduled(h.ctx, pool.PoolId); block++ {
		h.keeper.ProcessTWAPGarbageCollection(h.ctx)
	}
	if got := v4CountTWAPObservations(h, pool.PoolId); got != 0 {
		t.Fatalf("bounded cleanup did not finish within 200 blocks: %d observations", got)
	}
	if h.keeper.IsTWAPHistoryDeletionScheduled(h.ctx, pool.PoolId) {
		t.Fatal("bounded cleanup retained marker after deleting every checkpoint")
	}
}

func TestV4GovernedPoolStatusEnforcesLeastDangerousExitRights(t *testing.T) {
	t.Run("active permits swap add and remove", func(t *testing.T) {
		h := newV4Harness(t)
		pool := v4CreatePool(t, h, "uatom", "10000000000", "5000000000")
		provider := v4Address(0x45)
		v4Fund(h.bank, provider, pool.DenomA, pool.DenomB)
		if _, err := h.msgServer.Swap(h.ctx, &types.MsgSwap{
			Sender:        provider,
			PoolId:        pool.PoolId,
			TokenInDenom:  pool.DenomB,
			TokenInAmount: "1000000",
		}); err != nil {
			t.Fatalf("swap in active mode: %v", err)
		}
		if _, err := h.msgServer.AddLiquidity(h.ctx, &types.MsgAddLiquidity{
			Sender:  provider,
			PoolId:  pool.PoolId,
			AmountA: "1000000000",
			AmountB: "500000000",
		}); err != nil {
			t.Fatalf("add liquidity in active mode: %v", err)
		}
		supply, _ := new(big.Int).SetString(pool.LpTokenSupply, 10)
		exitShares := supply.Quo(supply, big.NewInt(10)).String()
		if _, err := h.msgServer.RemoveLiquidity(h.ctx, &types.MsgRemoveLiquidity{
			Sender:   h.authority,
			PoolId:   pool.PoolId,
			LpTokens: exitShares,
		}); err != nil {
			t.Fatalf("LP exit in active mode: %v", err)
		}
	})

	t.Run("swaps paused permits liquidity maintenance and exit", func(t *testing.T) {
		h := newV4Harness(t)
		pool := v4CreatePool(t, h, "uatom", "10000000000", "5000000000")
		if _, err := h.msgServer.SetPoolStatus(h.ctx, &types.MsgSetPoolStatus{
			Authority: h.authority,
			PoolId:    pool.PoolId,
			Status:    types.PoolStatus_POOL_STATUS_SWAPS_PAUSED,
		}); err != nil {
			t.Fatalf("pause swaps: %v", err)
		}
		if _, err := h.queryServer.SimulateSwap(h.ctx, &types.QuerySimulateSwapRequest{
			PoolId:        pool.PoolId,
			TokenInDenom:  pool.DenomA,
			TokenInAmount: "1000000",
		}); err == nil {
			t.Fatal("swap quote succeeded while swaps were paused")
		}
		if _, err := h.msgServer.Swap(h.ctx, &types.MsgSwap{
			Sender:        v4Address(0x46),
			PoolId:        pool.PoolId,
			TokenInDenom:  pool.DenomA,
			TokenInAmount: "1000000",
		}); err == nil {
			t.Fatal("swap execution succeeded while swaps were paused")
		}

		provider := v4Address(0x46)
		v4Fund(h.bank, provider, pool.DenomA, pool.DenomB)
		if _, err := h.msgServer.AddLiquidity(h.ctx, &types.MsgAddLiquidity{
			Sender:  provider,
			PoolId:  pool.PoolId,
			AmountA: "1000000000",
			AmountB: "500000000",
		}); err != nil {
			t.Fatalf("liquidity maintenance while swaps paused: %v", err)
		}

		originalSupply, _ := new(big.Int).SetString(pool.LpTokenSupply, 10)
		exitShares := originalSupply.Quo(originalSupply, big.NewInt(10)).String()
		if _, err := h.msgServer.RemoveLiquidity(h.ctx, &types.MsgRemoveLiquidity{
			Sender:   h.authority,
			PoolId:   pool.PoolId,
			LpTokens: exitShares,
		}); err != nil {
			t.Fatalf("LP exit while swaps paused: %v", err)
		}
	})

	t.Run("exit only rejects new risk but permits withdrawal", func(t *testing.T) {
		h := newV4Harness(t)
		pool := v4CreatePool(t, h, "uatom", "10000000000", "5000000000")
		if _, err := h.msgServer.SetPoolStatus(h.ctx, &types.MsgSetPoolStatus{
			Authority: h.authority,
			PoolId:    pool.PoolId,
			Status:    types.PoolStatus_POOL_STATUS_EXIT_ONLY,
		}); err != nil {
			t.Fatalf("enter exit-only mode: %v", err)
		}

		provider := v4Address(0x47)
		v4Fund(h.bank, provider, pool.DenomA, pool.DenomB)
		if _, err := h.msgServer.AddLiquidity(h.ctx, &types.MsgAddLiquidity{
			Sender:  provider,
			PoolId:  pool.PoolId,
			AmountA: "1000000000",
			AmountB: "500000000",
		}); err == nil {
			t.Fatal("liquidity addition succeeded in exit-only mode")
		}
		if _, err := h.queryServer.SimulateSwap(h.ctx, &types.QuerySimulateSwapRequest{
			PoolId:        pool.PoolId,
			TokenInDenom:  pool.DenomA,
			TokenInAmount: "1000000",
		}); err == nil {
			t.Fatal("swap quote succeeded in exit-only mode")
		}
		if _, err := h.msgServer.Swap(h.ctx, &types.MsgSwap{
			Sender:        provider,
			PoolId:        pool.PoolId,
			TokenInDenom:  pool.DenomA,
			TokenInAmount: "1000000",
		}); err == nil {
			t.Fatal("swap execution succeeded in exit-only mode")
		}

		supply, _ := new(big.Int).SetString(pool.LpTokenSupply, 10)
		exitShares := supply.Quo(supply, big.NewInt(10)).String()
		if _, err := h.msgServer.RemoveLiquidity(h.ctx, &types.MsgRemoveLiquidity{
			Sender:   h.authority,
			PoolId:   pool.PoolId,
			LpTokens: exitShares,
		}); err != nil {
			t.Fatalf("LP exit in exit-only mode: %v", err)
		}
	})

	t.Run("unauthorized and direct-close transitions are rejected", func(t *testing.T) {
		h := newV4Harness(t)
		pool := v4CreatePool(t, h, "uatom", "10000000000", "5000000000")
		before := proto.Clone(pool).(*types.Pool)
		for name, message := range map[string]*types.MsgSetPoolStatus{
			"unauthorized": {
				Authority: v4Address(0x48),
				PoolId:    pool.PoolId,
				Status:    types.PoolStatus_POOL_STATUS_SWAPS_PAUSED,
			},
			"direct close": {
				Authority: h.authority,
				PoolId:    pool.PoolId,
				Status:    types.PoolStatus_POOL_STATUS_CLOSED,
			},
		} {
			message := message
			t.Run(name, func(t *testing.T) {
				if _, err := h.msgServer.SetPoolStatus(h.ctx, message); err == nil {
					t.Fatal("forbidden status transition succeeded")
				}
				v4AssertPoolUnchanged(t, h, before)
			})
		}
	})
}

func TestV4SwapQuoteAndExecutionHonorBothDenomSendControls(t *testing.T) {
	for _, disabledDenom := range []string{types.ZRNDenom, "uatom"} {
		disabledDenom := disabledDenom
		t.Run(disabledDenom, func(t *testing.T) {
			h := newV4Harness(t)
			pool := v4CreatePool(t, h, "uatom", "10000000000", "5000000000")
			trader := v4Address(0x49)
			v4Fund(h.bank, trader, pool.DenomA, pool.DenomB)
			h.bank.sendDisabled[disabledDenom] = true

			poolBefore := proto.Clone(pool).(*types.Pool)
			bankBefore := v4SnapshotBank(h.bank)
			if _, err := h.queryServer.SimulateSwap(h.ctx, &types.QuerySimulateSwapRequest{
				PoolId:        pool.PoolId,
				TokenInDenom:  pool.DenomA,
				TokenInAmount: "1000000",
			}); err == nil {
				t.Fatalf("quote ignored send-disabled denom %s", disabledDenom)
			}
			if _, err := h.msgServer.Swap(h.ctx, &types.MsgSwap{
				Sender:        trader,
				PoolId:        pool.PoolId,
				TokenInDenom:  pool.DenomA,
				TokenInAmount: "1000000",
			}); err == nil {
				t.Fatalf("execution ignored send-disabled denom %s", disabledDenom)
			}
			v4AssertPoolUnchanged(t, h, poolBefore)
			v4AssertBankUnchanged(t, h.bank, bankBefore)

			// Incident send controls must stop risk-increasing swaps without
			// trapping the LP's proportional withdrawal.
			supply, _ := new(big.Int).SetString(pool.LpTokenSupply, 10)
			if _, err := h.msgServer.RemoveLiquidity(h.ctx, &types.MsgRemoveLiquidity{
				Sender:   h.authority,
				PoolId:   pool.PoolId,
				LpTokens: new(big.Int).Quo(supply, big.NewInt(10)).String(),
			}); err != nil {
				t.Fatalf("LP exit was trapped by send-disabled denom %s: %v", disabledDenom, err)
			}
		})
	}
}

func TestV4TWAPUsesRetainedTrailingWindowAndReportsYoungSpan(t *testing.T) {
	h := newV4Harness(t)
	params := h.keeper.GetParams(h.ctx)
	params.TwapWindowBlocks = 10
	h.keeper.SetParams(h.ctx, params)
	pool := v4CreatePool(t, h, "uatom", "10000000000", "5000000000")

	height782 := h.ctx.WithBlockHeight(782)
	h.keeper.UpdateTWAPAccumulator(height782, pool)
	pool.ReserveB = "10000000000"
	h.keeper.SetPool(height782, pool)
	height787 := h.ctx.WithBlockHeight(787)
	h.keeper.UpdateTWAPAccumulator(height787, pool)

	trailing, windowUsed, err := h.keeper.GetTWAP(height787, pool.PoolId, pool.DenomA, 5)
	if err != nil {
		t.Fatalf("five-block trailing TWAP: %v", err)
	}
	if trailing.String() != "1000000" || windowUsed != 5 {
		t.Fatalf("five-block trailing TWAP = %s over %d blocks, want 1000000 over 5", trailing, windowUsed)
	}
	sinceCreation, windowUsed, err := h.keeper.GetTWAP(height787, pool.PoolId, pool.DenomA, 10)
	if err != nil {
		t.Fatalf("ten-block TWAP: %v", err)
	}
	if sinceCreation.String() != "750000" || windowUsed != 10 {
		t.Fatalf("ten-block TWAP = %s over %d blocks, want 750000 over 10", sinceCreation, windowUsed)
	}
	if trailing.Cmp(sinceCreation) == 0 {
		t.Fatal("trailing TWAP silently returned the since-creation average")
	}

	if _, _, err := h.keeper.GetTWAP(height787, pool.PoolId, pool.DenomA, 11); err == nil {
		t.Fatal("TWAP request beyond retained window succeeded")
	}
	if _, _, err := h.keeper.GetTWAP(height787, pool.PoolId, "uosmo", 5); err == nil {
		t.Fatal("TWAP request for denom outside the pool succeeded")
	}

	young := newV4Harness(t)
	youngParams := young.keeper.GetParams(young.ctx)
	youngParams.TwapWindowBlocks = 10
	young.keeper.SetParams(young.ctx, youngParams)
	youngPool := v4CreatePool(t, young, "uatom", "10000000000", "5000000000")
	height780 := young.ctx.WithBlockHeight(780)
	young.keeper.UpdateTWAPAccumulator(height780, youngPool)
	price, youngWindow, err := young.keeper.GetTWAP(
		height780,
		youngPool.PoolId,
		youngPool.DenomA,
		10,
	)
	if err != nil {
		t.Fatalf("young-pool TWAP: %v", err)
	}
	if price.String() != "500000" || youngWindow != 3 {
		t.Fatalf("young-pool TWAP = %s over %d blocks, want 500000 over 3", price, youngWindow)
	}
}

func TestV4TWAPPruningRetainsAtMostWindowPlusOneCheckpoints(t *testing.T) {
	h := newV4Harness(t)
	params := h.keeper.GetParams(h.ctx)
	params.TwapWindowBlocks = 3
	h.keeper.SetParams(h.ctx, params)
	pool := v4CreatePool(t, h, "uatom", "10000000000", "5000000000")

	for height := int64(v4TestBlockHeight + 1); height <= v4TestBlockHeight+13; height++ {
		h.keeper.UpdateTWAPAccumulator(h.ctx.WithBlockHeight(height), pool)
	}

	var heights []uint64
	h.keeper.IterateTWAPObservations(h.ctx, pool.PoolId, func(observation *types.TWAPObservation) bool {
		heights = append(heights, observation.BlockHeight)
		return false
	})
	if len(heights) > int(params.TwapWindowBlocks)+1 {
		t.Fatalf(
			"retained %d TWAP checkpoints for window %d: %v",
			len(heights),
			params.TwapWindowBlocks,
			heights,
		)
	}
	wantEarliest := uint64(v4TestBlockHeight + 13 - int64(params.TwapWindowBlocks))
	if len(heights) == 0 || heights[0] < wantEarliest {
		t.Fatalf("stale TWAP checkpoint survived pruning: heights=%v earliest_allowed=%d", heights, wantEarliest)
	}
}

func TestV4PoolsPaginationUsesOpaqueContinuationKeyAndLimit(t *testing.T) {
	h := newV4Harness(t)
	for _, denom := range []string{"uatom", "uosmo", "ujuno", "ucreatea", "ucreateb"} {
		v4CreatePool(t, h, denom, "10000000000", "5000000000")
	}

	defaultPage, err := h.queryServer.Pools(h.ctx, &types.QueryPoolsRequest{})
	if err != nil {
		t.Fatalf("default pool page: %v", err)
	}
	if len(defaultPage.Pools) != 5 || defaultPage.Pagination == nil ||
		defaultPage.Pagination.Total != 0 {
		t.Fatalf("default page must stay bounded without forcing a total scan: %v", defaultPage)
	}

	first, err := h.queryServer.Pools(h.ctx, &types.QueryPoolsRequest{
		Pagination: &sdkquery.PageRequest{Limit: 2, CountTotal: true},
	})
	if err != nil {
		t.Fatalf("first pool page: %v", err)
	}
	if len(first.Pools) != 2 || first.Pools[0].PoolId != "pool-1" || first.Pools[1].PoolId != "pool-2" {
		t.Fatalf("unexpected first page: %v", first.Pools)
	}
	if first.Pagination == nil || len(first.Pagination.NextKey) == 0 || first.Pagination.Total != 5 {
		t.Fatalf("invalid first-page metadata: %v", first.Pagination)
	}

	second, err := h.queryServer.Pools(h.ctx, &types.QueryPoolsRequest{
		Pagination: &sdkquery.PageRequest{Key: first.Pagination.NextKey, Limit: 2},
	})
	if err != nil {
		t.Fatalf("second pool page: %v", err)
	}
	if len(second.Pools) != 2 || second.Pools[0].PoolId != "pool-3" || second.Pools[1].PoolId != "pool-4" {
		t.Fatalf("unexpected second page: %v", second.Pools)
	}
	if second.Pagination == nil || len(second.Pagination.NextKey) == 0 {
		t.Fatalf("second page omitted continuation metadata: %v", second.Pagination)
	}

	last, err := h.queryServer.Pools(h.ctx, &types.QueryPoolsRequest{
		Pagination: &sdkquery.PageRequest{Key: second.Pagination.NextKey, Limit: 2},
	})
	if err != nil {
		t.Fatalf("last pool page: %v", err)
	}
	if len(last.Pools) != 1 || last.Pools[0].PoolId != "pool-5" {
		t.Fatalf("unexpected final page: %v", last.Pools)
	}
	if last.Pagination == nil || len(last.Pagination.NextKey) != 0 {
		t.Fatalf("final page has spurious continuation key: %v", last.Pagination)
	}

	if _, err := h.queryServer.Pools(h.ctx, &types.QueryPoolsRequest{
		Pagination: &sdkquery.PageRequest{Key: []byte{0xff}, Limit: 1},
	}); err == nil {
		t.Fatal("invalid pool page key was accepted")
	}
	if _, err := h.queryServer.Pools(h.ctx, &types.QueryPoolsRequest{
		Pagination: &sdkquery.PageRequest{
			Key:    first.Pagination.NextKey,
			Offset: 1,
			Limit:  1,
		},
	}); err == nil {
		t.Fatal("pagination request combining key and offset was accepted")
	}
}

func TestV4GenesisRoundTripPreservesAndDerivesNextPoolID(t *testing.T) {
	source := newV4Harness(t)
	first := v4CreatePool(t, source, "uatom", "10000000000", "5000000000")
	second := v4CreatePool(t, source, "uosmo", "12000000000", "6000000000")

	exported := source.keeper.ExportGenesis(source.ctx)
	if exported.NextPoolId != 3 {
		t.Fatalf("exported next_pool_id = %d, want 3", exported.NextPoolId)
	}

	tests := []struct {
		name         string
		nextPoolID   uint64
		wantNextPool uint64
	}{
		{name: "explicit counter", nextPoolID: exported.NextPoolId, wantNextPool: 3},
		{name: "legacy zero sentinel derives max plus one", nextPoolID: 0, wantNextPool: 3},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			imported := proto.Clone(exported).(*types.GenesisState)
			imported.NextPoolId = tt.nextPoolID

			target := newV4Harness(t)
			target.keeper.InitGenesis(target.ctx, imported)
			if got := target.keeper.GetNextPoolId(target.ctx); got != tt.wantNextPool {
				t.Fatalf("next pool ID after import = %d, want %d", got, tt.wantNextPool)
			}

			third := v4CreatePool(t, target, "ujuno", "14000000000", "7000000000")
			if third.PoolId != "pool-3" {
				t.Fatalf("first post-import pool ID = %q, want pool-3", third.PoolId)
			}
			if third.LpDenom == first.LpDenom || third.LpDenom == second.LpDenom {
				t.Fatalf("post-import pool reused historical LP denom %q", third.LpDenom)
			}
			for _, historical := range []*types.Pool{first, second} {
				got, found := target.keeper.GetPool(target.ctx, historical.PoolId)
				if !found || got.LpDenom != historical.LpDenom {
					t.Fatalf("historical pool %q overwritten after import", historical.PoolId)
				}
			}
		})
	}
}

func v4LegacyPool(
	poolID,
	counterDenom,
	reserveA,
	reserveB,
	lpSupply,
	creator string,
) *types.Pool {
	return &types.Pool{
		PoolId:         poolID,
		DenomA:         types.ZRNDenom,
		DenomB:         counterDenom,
		ReserveA:       reserveA,
		ReserveB:       reserveB,
		SwapFeeBps:     3_000,
		LpTokenSupply:  lpSupply,
		LpDenom:        types.LPDenom(poolID),
		Creator:        creator,
		CreatedAtBlock: 100,
	}
}

func v4SetLegacyParams(h v4Harness) {
	params := types.DefaultParams()
	params.ProtocolFeeBps = 450_000
	params.MaxPools = 0
	params.AllowedPoolDenoms = nil
	params.PoolCreators = nil
	params.BillingQuoteDenoms = nil
	h.keeper.SetParams(h.ctx, params)
}

func TestV5MigrationOnlyRetiresProtocolSkim(t *testing.T) {
	h := newV4InvariantHarness(t)
	params := types.DefaultParams()
	params.ProtocolFeeBps = 450_000
	params.AllowedPoolDenoms = []string{"uatom"}
	params.PoolCreators = []string{h.authority}
	h.keeper.SetParams(h.ctx, params)

	pool := v4LegacyPool(
		"pool-1", "uatom", "10000000000", "5000000000", "7071067811", h.authority,
	)
	pool.Status = types.PoolStatus_POOL_STATUS_EXIT_ONLY
	h.keeper.SetPool(h.ctx, pool)
	acc := &types.TWAPAccumulator{
		PoolId: pool.PoolId, LastBlock: 700, StartBlock: 600,
		CumPriceAToB: "123", CumPriceBToA: "456",
	}
	h.keeper.SetTWAPAccumulator(h.ctx, acc)
	h.keeper.SetNextPoolId(h.ctx, 2)
	h.bank.moduleBalances[types.ModuleName] = map[string]int64{
		pool.DenomA: 10_000_000_000,
		pool.DenomB: 5_000_000_000,
	}
	h.bank.setBalance(h.authority, pool.LpDenom, 7_071_067_811)

	poolBefore := proto.Clone(pool).(*types.Pool)
	accBefore := proto.Clone(acc).(*types.TWAPAccumulator)
	bankBefore := v4SnapshotBank(h.bank)
	nextBefore := h.keeper.GetNextPoolId(h.ctx)

	if err := keeper.NewMigrator(h.keeper).Migrate4to5(h.ctx); err != nil {
		t.Fatalf("migrate v4 to v5: %v", err)
	}
	migrated := h.keeper.GetParams(h.ctx)
	if migrated.ProtocolFeeBps != 0 {
		t.Fatalf("protocol fee = %d, want retired zero", migrated.ProtocolFeeBps)
	}
	params.ProtocolFeeBps = 0
	if !proto.Equal(params, migrated) {
		t.Fatalf("migration changed unrelated params:\nwant=%s\ngot=%s", params, migrated)
	}
	v4AssertPoolUnchanged(t, h, poolBefore)
	gotAcc, found := h.keeper.GetTWAPAccumulator(h.ctx, pool.PoolId)
	if !found || !proto.Equal(accBefore, gotAcc) {
		t.Fatalf("migration changed TWAP state: found=%t got=%v", found, gotAcc)
	}
	if indexed := h.keeper.GetPoolByDenoms(h.ctx, pool.DenomA, pool.DenomB); indexed == nil || indexed.PoolId != pool.PoolId {
		t.Fatalf("migration changed pair index: %v", indexed)
	}
	if got := h.keeper.GetNextPoolId(h.ctx); got != nextBefore {
		t.Fatalf("migration changed next pool ID: got %d want %d", got, nextBefore)
	}
	v4AssertBankUnchanged(t, h.bank, bankBefore)
}

func TestV4MigrationClassifiesLegacyPoolsAndRebuildsIdentityState(t *testing.T) {
	h := newV4InvariantHarness(t)
	v4SetLegacyParams(h)

	positive := v4LegacyPool(
		"pool-5",
		"uatom",
		"10000000000",
		"5000000000",
		"7071067811",
		h.authority,
	)
	exhausted := v4LegacyPool("pool-3", "uosmo", "0", "0", "0", h.authority)
	h.keeper.SetPool(h.ctx, positive)
	h.keeper.SetPool(h.ctx, exhausted)
	h.bank.moduleBalances[types.ModuleName] = map[string]int64{
		positive.DenomA: 10_000_000_000,
		positive.DenomB: 5_000_000_000,
	}
	h.bank.setBalance(h.authority, positive.LpDenom, 7_071_067_811)
	for _, pool := range []*types.Pool{positive, exhausted} {
		h.keeper.SetTWAPAccumulator(h.ctx, &types.TWAPAccumulator{
			PoolId:       pool.PoolId,
			LastBlock:    500,
			StartBlock:   100,
			CumPriceAToB: "123",
			CumPriceBToA: "456",
		})
	}

	if err := keeper.NewMigrator(h.keeper).Migrate3to4(h.ctx); err != nil {
		t.Fatalf("migrate v3 to v4: %v", err)
	}
	gotExitOnly, found := h.keeper.GetPool(h.ctx, positive.PoolId)
	if !found || gotExitOnly.Status != types.PoolStatus_POOL_STATUS_EXIT_ONLY {
		t.Fatalf("positive legacy pool was not quarantined EXIT_ONLY: found=%t pool=%v", found, gotExitOnly)
	}
	gotClosed, found := h.keeper.GetPool(h.ctx, exhausted.PoolId)
	if !found || gotClosed.Status != types.PoolStatus_POOL_STATUS_CLOSED {
		t.Fatalf("exhausted legacy pool was not tombstoned: found=%t pool=%v", found, gotClosed)
	}
	if gotClosed.ClosedAtBlock != uint64(v4TestBlockHeight) {
		t.Fatalf("migrated tombstone close height = %d, want %d", gotClosed.ClosedAtBlock, v4TestBlockHeight)
	}
	if indexed := h.keeper.GetPoolByDenoms(h.ctx, exhausted.DenomA, exhausted.DenomB); indexed != nil {
		t.Fatalf("migrated tombstone remains indexed: %s", indexed.PoolId)
	}
	if indexed := h.keeper.GetPoolByDenoms(h.ctx, positive.DenomA, positive.DenomB); indexed == nil || indexed.PoolId != positive.PoolId {
		t.Fatalf("migrated open-pair index does not round-trip pool %s", positive.PoolId)
	}
	if got := h.keeper.CountOpenPools(h.ctx); got != 1 {
		t.Fatalf("open pool count = %d, want 1", got)
	}
	if _, found := h.keeper.GetTWAPAccumulator(h.ctx, exhausted.PoolId); found {
		t.Fatal("migrated tombstone retained legacy TWAP state")
	}
	exitOnlyAccumulator, found := h.keeper.GetTWAPAccumulator(h.ctx, positive.PoolId)
	if !found {
		t.Fatal("migrated EXIT_ONLY pool has no fresh TWAP state")
	}
	if exitOnlyAccumulator.StartBlock != uint64(v4TestBlockHeight) ||
		exitOnlyAccumulator.LastBlock != uint64(v4TestBlockHeight) ||
		exitOnlyAccumulator.CumPriceAToB != "0" ||
		exitOnlyAccumulator.CumPriceBToA != "0" {
		t.Fatalf("legacy TWAP was not reset truthfully: %v", exitOnlyAccumulator)
	}
	var observations int
	h.keeper.IterateTWAPObservations(h.ctx, positive.PoolId, func(observation *types.TWAPObservation) bool {
		observations++
		if observation.BlockHeight != uint64(v4TestBlockHeight) {
			t.Errorf("fresh observation height = %d, want %d", observation.BlockHeight, v4TestBlockHeight)
		}
		return false
	})
	if observations != 1 {
		t.Fatalf("fresh EXIT_ONLY TWAP observation count = %d, want 1", observations)
	}
	if got := h.keeper.GetNextPoolId(h.ctx); got != 6 {
		t.Fatalf("derived next pool ID = %d, want 6", got)
	}
	params := h.keeper.GetParams(h.ctx)
	if params.MaxPools != types.DefaultParams().MaxPools {
		t.Fatalf("migrated max_pools = %d, want default %d", params.MaxPools, types.DefaultParams().MaxPools)
	}
	if params.ProtocolFeeBps != 0 {
		t.Fatalf("migrated protocol_fee_bps = %d, want retired zero", params.ProtocolFeeBps)
	}
	if err := h.keeper.ExportGenesis(h.ctx).Validate(); err != nil {
		t.Fatalf("post-migration export is invalid: %v", err)
	}
}

func TestV4MigrationRejectsUnbackedLegacyPool(t *testing.T) {
	h := newV4InvariantHarness(t)
	v4SetLegacyParams(h)
	h.keeper.SetPool(h.ctx, v4LegacyPool(
		"pool-1",
		"uatom",
		"10000000000",
		"5000000000",
		"7071067811",
		h.authority,
	))

	if err := keeper.NewMigrator(h.keeper).Migrate3to4(h.ctx); err == nil {
		t.Fatal("migration activated a legacy pool without bank custody or LP supply")
	}
}

func TestV4MigrationClampsLegacyMaxPoolsAboveHardCap(t *testing.T) {
	h := newV4InvariantHarness(t)
	params := types.DefaultParams()
	params.MaxPools = types.MaxPoolsCap + 1
	params.AllowedPoolDenoms = nil
	params.PoolCreators = nil
	params.BillingQuoteDenoms = nil
	h.keeper.SetParams(h.ctx, params)

	if err := keeper.NewMigrator(h.keeper).Migrate3to4(h.ctx); err != nil {
		t.Fatalf("migrate v3 max_pools above v4 hard cap: %v", err)
	}
	if got := h.keeper.GetParams(h.ctx).MaxPools; got != types.MaxPoolsCap {
		t.Fatalf("migrated max_pools = %d, want hard cap %d", got, types.MaxPoolsCap)
	}
}

func TestV4MigrationRejectsAmbiguousOrMalformedLegacyPoolsWithoutPanic(t *testing.T) {
	tests := []struct {
		name  string
		pools func(string) []*types.Pool
	}{
		{
			name: "partial-zero reserve and supply state",
			pools: func(creator string) []*types.Pool {
				return []*types.Pool{
					v4LegacyPool("pool-1", "uatom", "0", "1", "1", creator),
				}
			},
		},
		{
			name: "noncanonical reserve",
			pools: func(creator string) []*types.Pool {
				return []*types.Pool{
					v4LegacyPool("pool-1", "uatom", "010000000000", "1", "1", creator),
				}
			},
		},
		{
			name: "invalid counter denom",
			pools: func(creator string) []*types.Pool {
				return []*types.Pool{
					v4LegacyPool("pool-1", "bad denom", "10000000000", "1", "100000", creator),
				}
			},
		},
		{
			name: "LP denom and ID mismatch",
			pools: func(creator string) []*types.Pool {
				pool := v4LegacyPool("pool-1", "uatom", "10000000000", "1", "100000", creator)
				pool.LpDenom = types.LPDenom("pool-2")
				return []*types.Pool{pool}
			},
		},
		{
			name: "persistent transaction lock",
			pools: func(creator string) []*types.Pool {
				pool := v4LegacyPool("pool-1", "uatom", "10000000000", "1", "100000", creator)
				pool.Locked = true
				return []*types.Pool{pool}
			},
		},
		{
			name: "duplicate open pair",
			pools: func(creator string) []*types.Pool {
				return []*types.Pool{
					v4LegacyPool("pool-1", "uatom", "10000000000", "1", "100000", creator),
					v4LegacyPool("pool-2", "uatom", "20000000000", "2", "200000", creator),
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			h := newV4Harness(t)
			v4SetLegacyParams(h)
			for _, pool := range tt.pools(h.authority) {
				h.keeper.SetPool(h.ctx, pool)
			}
			v4AssertErrorWithoutPanic(t, func() error {
				return keeper.NewMigrator(h.keeper).Migrate3to4(h.ctx)
			})
		})
	}
}

func TestV4CreatePoolRejectsPreexistingLPDenomSupply(t *testing.T) {
	h := newV4Harness(t)
	v4Fund(h.bank, h.authority, types.ZRNDenom, "uatom")
	if h.bank.moduleBalances["orphan"] == nil {
		h.bank.moduleBalances["orphan"] = make(map[string]int64)
	}
	h.bank.moduleBalances["orphan"][types.LPDenom("pool-1")] = 1

	bankBefore := v4SnapshotBank(h.bank)
	v4AssertErrorWithoutPanic(t, func() error {
		_, err := h.msgServer.CreatePool(h.ctx, &types.MsgCreatePool{
			Creator: h.authority,
			DenomA:  types.ZRNDenom,
			DenomB:  "uatom",
			AmountA: "10000000000",
			AmountB: "5000000000",
		})
		return err
	})
	if _, found := h.keeper.GetPool(h.ctx, "pool-1"); found {
		t.Fatal("pool was created on top of a pre-existing LP denom supply")
	}
	if got := h.keeper.GetNextPoolId(h.ctx); got != 1 {
		t.Fatalf("rejected creation advanced next pool ID to %d", got)
	}
	v4AssertBankUnchanged(t, h.bank, bankBefore)
}

func TestV4CreatePoolRejectsLifetimeRecordCapWithoutMutation(t *testing.T) {
	h := newV4Harness(t)
	v4Fund(h.bank, h.authority, types.ZRNDenom, "uatom")
	h.keeper.SetNextPoolId(h.ctx, types.MaxPoolRecordsCap+1)
	bankBefore := v4SnapshotBank(h.bank)

	_, err := h.msgServer.CreatePool(h.ctx, &types.MsgCreatePool{
		Creator: h.authority,
		DenomA:  types.ZRNDenom,
		DenomB:  "uatom",
		AmountA: "10000000000",
		AmountB: "5000000000",
	})
	if !errors.Is(err, types.ErrPoolRecordCapReached) {
		t.Fatalf("record-cap create error = %v", err)
	}
	if got := h.keeper.CountPools(h.ctx); got != 0 {
		t.Fatalf("record-cap rejection wrote %d pools", got)
	}
	v4AssertBankUnchanged(t, h.bank, bankBefore)
}

func TestV4StateConsistencyInvariantDetectsCustodyAndLPSupplyCorruption(t *testing.T) {
	t.Run("healthy state", func(t *testing.T) {
		h := newV4InvariantHarness(t)
		v4CreatePool(t, h, "uatom", "10000000000", "5000000000")
		if message, broken := keeper.StateConsistencyInvariant(h.keeper)(h.ctx); broken {
			t.Fatalf("healthy state broke invariant: %s", message)
		}
	})

	t.Run("reserve liability exceeds custody", func(t *testing.T) {
		h := newV4InvariantHarness(t)
		pool := v4CreatePool(t, h, "uatom", "10000000000", "5000000000")
		reserve, _ := new(big.Int).SetString(pool.ReserveA, 10)
		pool.ReserveA = reserve.Add(reserve, big.NewInt(1)).String()
		h.keeper.SetPool(h.ctx, pool)
		if _, broken := keeper.StateConsistencyInvariant(h.keeper)(h.ctx); !broken {
			t.Fatal("reserve liability exceeding custody did not break invariant")
		}
	})

	t.Run("recorded LP supply diverges from bank supply", func(t *testing.T) {
		h := newV4InvariantHarness(t)
		pool := v4CreatePool(t, h, "uatom", "10000000000", "5000000000")
		supply, _ := new(big.Int).SetString(pool.LpTokenSupply, 10)
		pool.LpTokenSupply = supply.Add(supply, big.NewInt(1)).String()
		h.keeper.SetPool(h.ctx, pool)
		if _, broken := keeper.StateConsistencyInvariant(h.keeper)(h.ctx); !broken {
			t.Fatal("LP bank/recorded supply divergence did not break invariant")
		}
	})

	t.Run("open-pool index does not round-trip", func(t *testing.T) {
		h := newV4InvariantHarness(t)
		pool := v4CreatePool(t, h, "uatom", "10000000000", "5000000000")
		store := h.store.OpenKVStore(h.ctx)
		if err := store.Set(types.OpenPoolIndexKey(pool.PoolId), []byte("pool-999")); err != nil {
			t.Fatalf("corrupt open-pool index: %v", err)
		}
		if _, broken := keeper.StateConsistencyInvariant(h.keeper)(h.ctx); !broken {
			t.Fatal("open-pool index corruption did not break invariant")
		}
	})
}

func TestV4CheckedQuoteMatchesExecutionAndConservesBalances(t *testing.T) {
	h := newV4Harness(t)
	pool := v4CreatePool(t, h, "uatom", "10000000000", "5000000000")
	sender := v4Address(0x45)
	const amountIn int64 = 123_456_789
	h.bank.setBalance(sender, pool.DenomB, amountIn)
	h.bank.setBalance(sender, pool.DenomA, 0)

	quote, err := h.queryServer.SimulateSwap(h.ctx, &types.QuerySimulateSwapRequest{
		PoolId:        pool.PoolId,
		TokenInDenom:  pool.DenomB,
		TokenInAmount: new(big.Int).SetInt64(amountIn).String(),
	})
	if err != nil {
		t.Fatalf("checked quote: %v", err)
	}
	if quote.Result == nil {
		t.Fatal("checked quote returned a nil result")
	}
	if quote.Result.TokenOutAmount == "0" {
		t.Fatal("checked quote returned zero output")
	}

	moduleBefore := v4SnapshotBank(h.bank)
	executed, err := h.msgServer.Swap(h.ctx, &types.MsgSwap{
		Sender:        sender,
		PoolId:        pool.PoolId,
		TokenInDenom:  pool.DenomB,
		TokenInAmount: new(big.Int).SetInt64(amountIn).String(),
		MinTokenOut:   quote.Result.TokenOutAmount,
	})
	if err != nil {
		t.Fatalf("execute checked quote: %v", err)
	}
	if executed.TokenOutAmount != quote.Result.TokenOutAmount {
		t.Fatalf("execution output = %s, quote = %s", executed.TokenOutAmount, quote.Result.TokenOutAmount)
	}
	if executed.FeeAmount != quote.Result.FeeAmount {
		t.Fatalf("execution fee = %s, quote = %s", executed.FeeAmount, quote.Result.FeeAmount)
	}

	tokenOut, ok := new(big.Int).SetString(executed.TokenOutAmount, 10)
	if !ok {
		t.Fatalf("execution returned malformed output %q", executed.TokenOutAmount)
	}
	after, found := h.keeper.GetPool(h.ctx, pool.PoolId)
	if !found {
		t.Fatal("pool missing after swap")
	}
	wantReserveA := new(big.Int)
	wantReserveA.SetString(pool.ReserveA, 10)
	wantReserveA.Sub(wantReserveA, tokenOut)
	wantReserveB := new(big.Int)
	wantReserveB.SetString(pool.ReserveB, 10)
	wantReserveB.Add(wantReserveB, big.NewInt(amountIn))
	if after.ReserveA != wantReserveA.String() || after.ReserveB != wantReserveB.String() {
		t.Fatalf(
			"reserve conservation failed: got %s/%s, want %s/%s",
			after.ReserveA,
			after.ReserveB,
			wantReserveA,
			wantReserveB,
		)
	}

	if got := h.bank.balances[sender][pool.DenomB]; got != 0 {
		t.Fatalf("sender input balance = %d, want 0", got)
	}
	if got := h.bank.balances[sender][pool.DenomA]; got != tokenOut.Int64() {
		t.Fatalf("sender output balance = %d, want %s", got, tokenOut)
	}
	if got := h.bank.moduleBalances[types.ModuleName][pool.DenomB] - moduleBefore.moduleBalances[types.ModuleName][pool.DenomB]; got != amountIn {
		t.Fatalf("module input delta = %d, want %d", got, amountIn)
	}
	if got := moduleBefore.moduleBalances[types.ModuleName][pool.DenomA] - h.bank.moduleBalances[types.ModuleName][pool.DenomA]; got != tokenOut.Int64() {
		t.Fatalf("module output delta = %d, want %s", got, tokenOut)
	}
}

func TestV4MsgServerRejectsHostileWireAmountsWithoutPanic(t *testing.T) {
	over256 := new(big.Int).Lsh(big.NewInt(1), 256).String()
	invalid := []string{"0", "-1", "+1", "01", "1e3", " 1", "1 ", over256}

	operations := []struct {
		name      string
		needsPool bool
		operation func(v4Harness, *types.Pool, string) error
	}{
		{
			name: "create amount_a",
			operation: func(h v4Harness, _ *types.Pool, value string) error {
				_, err := h.msgServer.CreatePool(h.ctx, &types.MsgCreatePool{
					Creator: h.authority, DenomA: types.ZRNDenom, DenomB: "ucreatea",
					AmountA: value, AmountB: "1000000", SwapFeeBps: 3_000,
				})
				return err
			},
		},
		{
			name: "create amount_b",
			operation: func(h v4Harness, _ *types.Pool, value string) error {
				_, err := h.msgServer.CreatePool(h.ctx, &types.MsgCreatePool{
					Creator: h.authority, DenomA: types.ZRNDenom, DenomB: "ucreateb",
					AmountA: "10000000000", AmountB: value, SwapFeeBps: 3_000,
				})
				return err
			},
		},
		{
			name:      "swap token_in_amount",
			needsPool: true,
			operation: func(h v4Harness, pool *types.Pool, value string) error {
				_, err := h.msgServer.Swap(h.ctx, &types.MsgSwap{
					Sender: v4Address(0x50), PoolId: pool.PoolId,
					TokenInDenom: pool.DenomA, TokenInAmount: value,
				})
				return err
			},
		},
		{
			name:      "swap min_token_out",
			needsPool: true,
			operation: func(h v4Harness, pool *types.Pool, value string) error {
				_, err := h.msgServer.Swap(h.ctx, &types.MsgSwap{
					Sender: v4Address(0x51), PoolId: pool.PoolId,
					TokenInDenom: pool.DenomA, TokenInAmount: "1000000", MinTokenOut: value,
				})
				return err
			},
		},
		{
			name:      "add amount_a",
			needsPool: true,
			operation: func(h v4Harness, pool *types.Pool, value string) error {
				_, err := h.msgServer.AddLiquidity(h.ctx, &types.MsgAddLiquidity{
					Sender: v4Address(0x52), PoolId: pool.PoolId,
					AmountA: value, AmountB: "1000000",
				})
				return err
			},
		},
		{
			name:      "add amount_b",
			needsPool: true,
			operation: func(h v4Harness, pool *types.Pool, value string) error {
				_, err := h.msgServer.AddLiquidity(h.ctx, &types.MsgAddLiquidity{
					Sender: v4Address(0x53), PoolId: pool.PoolId,
					AmountA: "1000000", AmountB: value,
				})
				return err
			},
		},
		{
			name:      "add min_lp_tokens",
			needsPool: true,
			operation: func(h v4Harness, pool *types.Pool, value string) error {
				_, err := h.msgServer.AddLiquidity(h.ctx, &types.MsgAddLiquidity{
					Sender: v4Address(0x54), PoolId: pool.PoolId,
					AmountA: "1000000", AmountB: "1000000", MinLpTokens: value,
				})
				return err
			},
		},
		{
			name:      "remove lp_tokens",
			needsPool: true,
			operation: func(h v4Harness, pool *types.Pool, value string) error {
				_, err := h.msgServer.RemoveLiquidity(h.ctx, &types.MsgRemoveLiquidity{
					Sender: h.authority, PoolId: pool.PoolId, LpTokens: value,
				})
				return err
			},
		},
		{
			name:      "remove min_amount_a",
			needsPool: true,
			operation: func(h v4Harness, pool *types.Pool, value string) error {
				_, err := h.msgServer.RemoveLiquidity(h.ctx, &types.MsgRemoveLiquidity{
					Sender: h.authority, PoolId: pool.PoolId,
					LpTokens: "1", MinAmountA: value,
				})
				return err
			},
		},
		{
			name:      "remove min_amount_b",
			needsPool: true,
			operation: func(h v4Harness, pool *types.Pool, value string) error {
				_, err := h.msgServer.RemoveLiquidity(h.ctx, &types.MsgRemoveLiquidity{
					Sender: h.authority, PoolId: pool.PoolId,
					LpTokens: "1", MinAmountB: value,
				})
				return err
			},
		},
	}

	for _, operation := range operations {
		operation := operation
		t.Run(operation.name, func(t *testing.T) {
			for _, value := range invalid {
				value := value
				t.Run(value, func(t *testing.T) {
					h := newV4Harness(t)
					v4Fund(h.bank, h.authority, types.ZRNDenom, "ucreatea", "ucreateb")
					var pool *types.Pool
					if operation.needsPool {
						pool = v4CreatePool(t, h, "uatom", "10000000000", "5000000000")
						for seed := byte(0x50); seed <= 0x54; seed++ {
							v4Fund(h.bank, v4Address(seed), pool.DenomA, pool.DenomB)
						}
					}
					bankBefore := v4SnapshotBank(h.bank)
					var poolBefore *types.Pool
					if pool != nil {
						poolBefore = proto.Clone(pool).(*types.Pool)
					}

					v4AssertErrorWithoutPanic(t, func() error {
						return operation.operation(h, pool, value)
					})
					v4AssertBankUnchanged(t, h.bank, bankBefore)
					if poolBefore != nil {
						v4AssertPoolUnchanged(t, h, poolBefore)
					}
				})
			}
		})
	}
}

func TestV4MsgServerRejectsInvalidDenomsWithoutPanic(t *testing.T) {
	tests := []struct {
		name      string
		operation func(v4Harness) error
	}{
		{
			name: "create malformed counter denom",
			operation: func(h v4Harness) error {
				_, err := h.msgServer.CreatePool(h.ctx, &types.MsgCreatePool{
					Creator: h.authority, DenomA: types.ZRNDenom, DenomB: "bad denom",
					AmountA: "10000000000", AmountB: "1000000", SwapFeeBps: 3_000,
				})
				return err
			},
		},
		{
			name: "swap malformed input denom",
			operation: func(h v4Harness) error {
				pool := v4CreatePool(t, h, "uatom", "10000000000", "5000000000")
				_, err := h.msgServer.Swap(h.ctx, &types.MsgSwap{
					Sender: v4Address(0x60), PoolId: pool.PoolId,
					TokenInDenom: "bad denom", TokenInAmount: "1",
				})
				return err
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			h := newV4Harness(t)
			v4AssertErrorWithoutPanic(t, func() error {
				return tt.operation(h)
			})
		})
	}
}

func TestV4CheckedQuoteRejectsHostileWireInputWithoutPanic(t *testing.T) {
	h := newV4Harness(t)
	pool := v4CreatePool(t, h, "uatom", "10000000000", "5000000000")
	poolBefore := proto.Clone(pool).(*types.Pool)
	bankBefore := v4SnapshotBank(h.bank)
	over256 := new(big.Int).Lsh(big.NewInt(1), 256).String()

	for _, value := range []string{"", "0", "-1", "+1", "01", "1e3", " 1", "1 ", over256} {
		value := value
		t.Run(value, func(t *testing.T) {
			v4AssertErrorWithoutPanic(t, func() error {
				_, err := h.queryServer.SimulateSwap(h.ctx, &types.QuerySimulateSwapRequest{
					PoolId:        pool.PoolId,
					TokenInDenom:  pool.DenomA,
					TokenInAmount: value,
				})
				return err
			})
			v4AssertPoolUnchanged(t, h, poolBefore)
			v4AssertBankUnchanged(t, h.bank, bankBefore)
		})
	}

	v4AssertErrorWithoutPanic(t, func() error {
		_, err := h.queryServer.SimulateSwap(h.ctx, &types.QuerySimulateSwapRequest{
			PoolId:        pool.PoolId,
			TokenInDenom:  "bad denom",
			TokenInAmount: "1",
		})
		return err
	})
	v4AssertPoolUnchanged(t, h, poolBefore)
	v4AssertBankUnchanged(t, h.bank, bankBefore)
}
