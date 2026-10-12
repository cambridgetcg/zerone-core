package cross_stack_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

	lpkeeper "github.com/zerone-chain/zerone/x/liquiditypool/keeper"
	lptypes "github.com/zerone-chain/zerone/x/liquiditypool/types"
	tokenstypes "github.com/zerone-chain/zerone/x/tokens/types"
)

const (
	lpCacheCounterDenom = "uatom"
	lpCachePoolID       = "pool-1"
	lpCacheInitial      = int64(10_000_000_000)
)

type lpCacheFixture struct {
	h          *TestHarness
	actor      sdk.AccAddress
	module     sdk.AccAddress
	server     lptypes.MsgServer
	exitShares string
}

// Snapshot complete raw LP, SDK bank and SDK auth stores, not just exported
// genesis: this includes secondary indexes, zero-key deletion and TWAP GC keys.
// Events are copied as bytes so later append operations cannot alias a snapshot.
type lpCacheState struct {
	Stores                map[string]map[string][]byte
	Events                []byte
	Actor, Module, Supply sdk.Coins
}

func lpCacheSnapshot(t *testing.T, f *lpCacheFixture, ctx sdk.Context) lpCacheState {
	t.Helper()
	s := lpCacheState{Stores: make(map[string]map[string][]byte)}
	for _, name := range []string{lptypes.StoreKey, banktypes.StoreKey, authtypes.StoreKey} {
		entries := make(map[string][]byte)
		iter := ctx.KVStore(f.h.App.GetStoreKeyForTests(name)).Iterator(nil, nil)
		for ; iter.Valid(); iter.Next() {
			entries[string(iter.Key())] = bytes.Clone(iter.Value())
		}
		require.NoError(t, iter.Close())
		s.Stores[name] = entries
	}
	var err error
	s.Events, err = json.Marshal(ctx.EventManager().Events())
	require.NoError(t, err)
	s.Actor = f.h.BankKeeper.GetAllBalances(ctx, f.actor)
	s.Module = f.h.BankKeeper.GetAllBalances(ctx, f.module)
	f.h.BankKeeper.IterateTotalSupply(ctx, func(coin sdk.Coin) bool {
		s.Supply = append(s.Supply, coin)
		return false
	})
	return s
}

func lpCacheNewFixture(t *testing.T) *lpCacheFixture {
	t.Helper()
	h := NewTestHarness(t)
	t.Cleanup(func() { require.NoError(t, h.App.Close()) })
	f := &lpCacheFixture{
		h:          h,
		actor:      sdk.AccAddress(bytes.Repeat([]byte{0x71}, 20)),
		module:     authtypes.NewModuleAddress(lptypes.ModuleName),
		server:     lpkeeper.NewMsgServerImpl(h.App.LiquidityPoolKeeper),
		exitShares: "1000000000",
	}
	_, err := bankkeeper.NewMsgServerImpl(h.BankKeeper).SetSendEnabled(h.Ctx, banktypes.NewMsgSetSendEnabled(
		h.BankKeeper.GetAuthority(), []*banktypes.SendEnabled{
			{Denom: lptypes.ZRNDenom, Enabled: true},
			{Denom: lpCacheCounterDenom, Enabled: true},
		}, nil,
	))
	require.NoError(t, err)
	params := h.App.LiquidityPoolKeeper.GetParams(h.Ctx)
	params.AllowedPoolDenoms = []string{lpCacheCounterDenom}
	params.PoolCreators = []string{f.actor.String(), authtypes.NewModuleAddress(tokenstypes.ModuleName).String()}
	_, err = f.server.UpdateParams(h.Ctx, &lptypes.MsgUpdateParams{
		Authority: h.App.LiquidityPoolKeeper.GetAuthority(), Params: params,
	})
	require.NoError(t, err)
	require.NoError(t, h.FundAccount(f.actor, sdk.NewCoins(
		sdk.NewInt64Coin(lptypes.ZRNDenom, 50_000_000_000),
		sdk.NewInt64Coin(lpCacheCounterDenom, 50_000_000_000),
	)))
	account := h.AccountKeeper.GetModuleAccount(h.Ctx, lptypes.ModuleName)
	require.True(t, account.HasPermission(authtypes.Minter))
	require.True(t, account.HasPermission(authtypes.Burner))
	return f
}

func lpCacheDropPermission(t *testing.T, f *lpCacheFixture, permission string) {
	t.Helper()
	account, ok := f.h.AccountKeeper.GetModuleAccount(f.h.Ctx, lptypes.ModuleName).(*authtypes.ModuleAccount)
	require.True(t, ok)
	require.True(t, account.HasPermission(permission))
	kept := make([]string, 0, len(account.Permissions))
	for _, p := range account.Permissions {
		if p != permission {
			kept = append(kept, p)
		}
	}
	account.Permissions = kept
	f.h.AccountKeeper.SetModuleAccount(f.h.Ctx, account)
	require.False(t, f.h.AccountKeeper.GetModuleAccount(f.h.Ctx, lptypes.ModuleName).HasPermission(permission))
}

func (f *lpCacheFixture) create(ctx sdk.Context) (any, error) {
	return f.server.CreatePool(ctx, &lptypes.MsgCreatePool{
		Creator: f.actor.String(), DenomA: lptypes.ZRNDenom, DenomB: lpCacheCounterDenom,
		AmountA: "10000000000", AmountB: "10000000000", SwapFeeBps: 0,
	})
}

// Permission failures in the real SDK panic rather than returning an error.
// Recover only in the test so their exact class and prior effects are asserted;
// this helper supplies no transaction cache, undo, replacement bank, or wrapper.
func lpCacheInvoke(call func() (any, error)) (response any, err error, panicked bool) {
	defer func() {
		if value := recover(); value != nil {
			panicked = true
			var ok bool
			err, ok = value.(error)
			if !ok {
				err = fmt.Errorf("unexpected non-error panic: %v", value)
			}
		}
	}()
	response, err = call()
	return
}

func lpCacheDelta(t *testing.T, before, after sdk.Coins, denom string, delta int64) {
	t.Helper()
	require.Equal(t, before.AmountOf(denom).Add(sdkmath.NewInt(delta)).String(), after.AmountOf(denom).String(), denom)
}

func (f *lpCacheFixture) call(ctx sdk.Context, operation string) (any, error) {
	switch operation {
	case "create":
		return f.create(ctx)
	case "add":
		return f.server.AddLiquidity(ctx, &lptypes.MsgAddLiquidity{
			Sender: f.actor.String(), PoolId: lpCachePoolID,
			AmountA: "1000000000", AmountB: "1000000000", MinLpTokens: "1",
		})
	case "swap":
		return f.server.Swap(ctx, &lptypes.MsgSwap{
			Sender: f.actor.String(), PoolId: lpCachePoolID,
			TokenInDenom: lptypes.ZRNDenom, TokenInAmount: "1000000", MinTokenOut: "1",
		})
	case "remove":
		return f.server.RemoveLiquidity(ctx, &lptypes.MsgRemoveLiquidity{
			Sender: f.actor.String(), PoolId: lpCachePoolID, LpTokens: f.exitShares,
			MinAmountA: "1", MinAmountB: "1",
		})
	default:
		panic("unknown test operation: " + operation)
	}
}

func lpCacheNextContext(f *lpCacheFixture) {
	// A later context height exercises real handler TWAP writes. This is not
	// a block replay: no AdvanceBlocks, manager, or lifecycle hook is invoked.
	f.h.Ctx = f.h.Ctx.WithBlockHeight(f.h.Ctx.BlockHeight() + 7).WithEventManager(sdk.NewEventManager())
	f.h.Ctx.EventManager().EmitEvent(sdk.NewEvent("lp_cache.base_sentinel"))
}

func lpCachePrepareFault(t *testing.T, operation, fault string) *lpCacheFixture {
	t.Helper()
	f := lpCacheNewFixture(t)
	if operation != "create" {
		_, err := f.create(f.h.Ctx)
		require.NoError(t, err)
	}
	switch fault {
	case "missing_minter":
		lpCacheDropPermission(t, f, authtypes.Minter)
	case "missing_burner":
		lpCacheDropPermission(t, f, authtypes.Burner)
	case "blocked_payout":
		// Use a genuinely blocked module address without changing the app's
		// blocklist or replacing its restricted bank keeper. This synthetic
		// direct caller is NOT a claim that a module can sign an external tx.
		blocked := authtypes.NewModuleAddress(tokenstypes.ModuleName)
		require.True(t, f.h.BankKeeper.BlockedAddr(blocked))
		coins := sdk.NewCoins(
			sdk.NewInt64Coin(lptypes.ZRNDenom, 20_000_000_000),
			sdk.NewInt64Coin(lpCacheCounterDenom, 20_000_000_000),
		)
		if operation == "remove" {
			coins = sdk.NewCoins(sdk.NewInt64Coin(lptypes.LPDenom(lpCachePoolID), 1_000_000_000))
		}
		require.NoError(t, f.h.BankKeeper.SendCoinsFromAccountToModule(f.h.Ctx, f.actor, tokenstypes.ModuleName, coins))
		f.actor = blocked
	case "unbacked_output":
		// Deliberately corrupt backing through a real bank transfer, leaving
		// the pool record unchanged. The rollback baseline is AFTER this
		// setup; discarding a failed call does not repair pre-existing deficits.
		denom := lpCacheCounterDenom
		if operation == "remove" {
			// Sorted SDK withdrawal coins debit uatom before failing on uzrn.
			denom = lptypes.ZRNDenom
		}
		sink := sdk.AccAddress(bytes.Repeat([]byte{0x72}, 20))
		backing := f.h.BankKeeper.GetBalance(f.h.Ctx, f.module, denom)
		require.True(t, backing.Amount.IsPositive())
		require.NoError(t, f.h.BankKeeper.SendCoinsFromModuleToAccount(f.h.Ctx, lptypes.ModuleName, sink, sdk.NewCoins(backing)))
		require.True(t, f.h.BankKeeper.GetBalance(f.h.Ctx, f.module, denom).IsZero())
	default:
		t.Fatalf("unknown test fault %q", fault)
	}
	lpCacheNextContext(f)
	return f
}

func lpCacheEventCount(ctx sdk.Context, kind string) int {
	count := 0
	for _, event := range ctx.EventManager().Events() {
		if event.Type == kind {
			count++
		}
	}
	return count
}

func lpCacheSuccessEvent(operation string) string {
	switch operation {
	case "create":
		return "zerone.liquiditypool.pool_created"
	case "add":
		return "zerone.liquiditypool.liquidity_added"
	case "swap":
		return "zerone.liquiditypool.swap"
	case "remove":
		return "zerone.liquiditypool.liquidity_removed"
	default:
		panic("unknown test operation: " + operation)
	}
}

func lpCacheAssertLateEffects(t *testing.T, f *lpCacheFixture, ctx sdk.Context, operation, fault string, before, partial lpCacheState) {
	t.Helper()
	k := f.h.App.LiquidityPoolKeeper
	lpDenom := lptypes.LPDenom(lpCachePoolID)
	priorLP, dirtyLP := before.Stores[lptypes.StoreKey], partial.Stores[lptypes.StoreKey]
	require.NotEqual(t, before.Stores[banktypes.StoreKey], partial.Stores[banktypes.StoreKey], "a real bank effect must precede the fault")
	require.Equal(t, before.Stores[authtypes.StoreKey], partial.Stores[authtypes.StoreKey])
	require.Equal(t, priorLP[string(lptypes.ParamsKey)], dirtyLP[string(lptypes.ParamsKey)], "late failure must not consume the create admission")
	for _, denom := range []string{lptypes.ZRNDenom, lpCacheCounterDenom} {
		lpCacheDelta(t, before.Supply, partial.Supply, denom, 0)
	}
	require.Equal(t, 1, lpCacheEventCount(ctx, banktypes.EventTypeTransfer), "real deposit or LP collection event")
	require.Zero(t, lpCacheEventCount(ctx, lpCacheSuccessEvent(operation)))
	pool, found := k.GetPool(ctx, lpCachePoolID)
	require.True(t, found)
	require.Equal(t, lpDenom, pool.LpDenom)
	require.Equal(t, lptypes.PoolStatus_POOL_STATUS_ACTIVE, pool.Status)
	require.Equal(t, uint64(2), k.GetNextPoolId(ctx))
	require.Equal(t, uint64(1), k.CountOpenPools(ctx))
	require.Equal(t, lpCachePoolID, k.GetPoolByDenoms(ctx, lptypes.ZRNDenom, lpCacheCounterDenom).PoolId)
	if operation == "create" {
		require.NotEqual(t, priorLP[string(lptypes.PoolCounterKey)], dirtyLP[string(lptypes.PoolCounterKey)])
		_, hasTWAP := k.GetTWAPAccumulator(ctx, lpCachePoolID)
		require.False(t, hasTWAP, "create fails before TWAP initialization")
		require.Nil(t, priorLP[string(lptypes.PoolKey(lpCachePoolID))])
	} else {
		require.Equal(t, priorLP[string(lptypes.PoolCounterKey)], dirtyLP[string(lptypes.PoolCounterKey)])
		acc, hasTWAP := k.GetTWAPAccumulator(ctx, lpCachePoolID)
		require.True(t, hasTWAP)
		require.Equal(t, uint64(ctx.BlockHeight()), acc.LastBlock)
		require.NotEqual(t, priorLP[string(lptypes.TWAPKey(lpCachePoolID))], dirtyLP[string(lptypes.TWAPKey(lpCachePoolID))])
		observationKey := string(lptypes.TWAPObservationKey(lpCachePoolID, uint64(ctx.BlockHeight())))
		require.Nil(t, priorLP[observationKey])
		require.NotEmpty(t, dirtyLP[observationKey])
	}
	minted, burned := int64(0), int64(0)
	switch operation {
	case "create", "add":
		deposit, reserves := lpCacheInitial, lpCacheInitial
		if operation == "add" {
			deposit, reserves = 1_000_000_000, 11_000_000_000
		}
		for _, denom := range []string{lptypes.ZRNDenom, lpCacheCounterDenom} {
			lpCacheDelta(t, before.Actor, partial.Actor, denom, -deposit)
			lpCacheDelta(t, before.Module, partial.Module, denom, deposit)
		}
		require.Equal(t, sdkmath.NewInt(reserves).String(), pool.ReserveA)
		require.Equal(t, sdkmath.NewInt(reserves).String(), pool.ReserveB)
		require.Equal(t, sdkmath.NewInt(reserves).String(), pool.LpTokenSupply, "SetPool precedes MintCoins")
		require.False(t, pool.Locked)
		if fault == "blocked_payout" {
			minted = deposit
		}
		lpCacheDelta(t, before.Module, partial.Module, lpDenom, minted)
		lpCacheDelta(t, before.Actor, partial.Actor, lpDenom, 0)
	case "swap":
		lpCacheDelta(t, before.Actor, partial.Actor, lptypes.ZRNDenom, -1_000_000)
		lpCacheDelta(t, before.Module, partial.Module, lptypes.ZRNDenom, 1_000_000)
		lpCacheDelta(t, before.Actor, partial.Actor, lpCacheCounterDenom, 0)
		lpCacheDelta(t, before.Module, partial.Module, lpCacheCounterDenom, 0)
		require.Equal(t, priorLP[string(lptypes.PoolKey(lpCachePoolID))], dirtyLP[string(lptypes.PoolKey(lpCachePoolID))], "error unlock restores old reserves, not the deposit or TWAP")
	case "remove":
		lpCacheDelta(t, before.Actor, partial.Actor, lpDenom, -1_000_000_000)
		if fault != "missing_burner" {
			burned = 1_000_000_000
		}
		lpCacheDelta(t, before.Module, partial.Module, lpDenom, 1_000_000_000-burned)
		for _, denom := range []string{lptypes.ZRNDenom, lpCacheCounterDenom} {
			lpCacheDelta(t, before.Actor, partial.Actor, denom, 0)
			debit := int64(0)
			if fault == "unbacked_output" && denom == lpCacheCounterDenom {
				debit = -1_000_000_000
			}
			lpCacheDelta(t, before.Module, partial.Module, denom, debit)
		}
		require.Equal(t, "10000000000", pool.ReserveA)
		require.Equal(t, "10000000000", pool.ReserveB)
		require.Equal(t, "10000000000", pool.LpTokenSupply, "record supply changes only after payout")
		require.Equal(t, fault == "missing_burner", pool.Locked, "SDK panic bypasses the handler's error-only unlock")
		require.False(t, k.IsTWAPHistoryDeletionScheduled(ctx, lpCachePoolID))
	}
	lpCacheDelta(t, before.Supply, partial.Supply, lpDenom, minted-burned)
	wantMint, wantBurn := 0, 0
	if minted > 0 {
		wantMint = 1
	}
	if burned > 0 {
		wantBurn = 1
	}
	require.Equal(t, wantMint, lpCacheEventCount(ctx, banktypes.EventTypeCoinMint))
	require.Equal(t, wantBurn, lpCacheEventCount(ctx, banktypes.EventTypeCoinBurn))
}

func lpCacheAmount(t *testing.T, value string) int64 {
	t.Helper()
	amount, ok := sdkmath.NewIntFromString(value)
	require.True(t, ok)
	require.True(t, amount.IsPositive())
	require.True(t, amount.IsInt64())
	return amount.Int64()
}

// A success is invisible to the parent until writeCache, which must publish
// both the exact child store bytes and the child events after existing events.
func TestLiquidityCacheWriteSuccess(t *testing.T) {
	for _, operation := range []string{"create", "add", "swap", "remove"} {
		t.Run(operation, func(t *testing.T) {
			f := lpCacheNewFixture(t)
			k := f.h.App.LiquidityPoolKeeper
			if operation != "create" {
				_, err := f.create(f.h.Ctx)
				require.NoError(t, err)
			}
			lpDenom := lptypes.LPDenom(lpCachePoolID)
			if operation == "remove" {
				f.exitShares = f.h.BankKeeper.GetBalance(f.h.Ctx, f.actor, lpDenom).Amount.String()
			}
			lpCacheNextContext(f)
			before := lpCacheSnapshot(t, f, f.h.Ctx)
			var quote *lptypes.SwapResult
			if operation == "swap" {
				var err error
				quote, err = k.CheckedSwapQuote(f.h.Ctx, lpCachePoolID, lptypes.ZRNDenom, "1000000")
				require.NoError(t, err)
			}
			child, writeCache := f.h.Ctx.CacheContext()
			response, err, panicked := lpCacheInvoke(func() (any, error) { return f.call(child, operation) })
			require.False(t, panicked)
			require.NoError(t, err)
			require.NotNil(t, response)
			pending := lpCacheSnapshot(t, f, child)
			require.NotEqual(t, before.Stores[banktypes.StoreKey], pending.Stores[banktypes.StoreKey])
			require.Equal(t, before, lpCacheSnapshot(t, f, f.h.Ctx), "successful child is not automatically written")
			require.Equal(t, 1, lpCacheEventCount(child, lpCacheSuccessEvent(operation)))
			require.Equal(t, 2, lpCacheEventCount(child, banktypes.EventTypeTransfer))
			var parentEvents sdk.Events
			require.NoError(t, json.Unmarshal(before.Events, &parentEvents))
			expectedEvents, err := json.Marshal(append(parentEvents, child.EventManager().Events()...))
			require.NoError(t, err)
			writeCache()
			committed := lpCacheSnapshot(t, f, f.h.Ctx)
			require.Equal(t, pending.Stores, committed.Stores, "successful write must publish exact child stores")
			require.Equal(t, expectedEvents, committed.Events, "write must append exact child events once")
			require.Equal(t, pending.Actor, committed.Actor)
			require.Equal(t, pending.Module, committed.Module)
			require.Equal(t, pending.Supply, committed.Supply)
			pool, found := k.GetPool(f.h.Ctx, lpCachePoolID)
			require.True(t, found)
			require.False(t, pool.Locked)
			require.Equal(t, lpDenom, pool.LpDenom)
			require.Equal(t, pool.LpTokenSupply, committed.Supply.AmountOf(lpDenom).String())
			require.Equal(t, uint64(2), k.GetNextPoolId(f.h.Ctx))
			for _, denom := range []string{lptypes.ZRNDenom, lpCacheCounterDenom} {
				lpCacheDelta(t, before.Supply, committed.Supply, denom, 0)
			}
			if operation != "create" {
				for _, key := range [][]byte{lptypes.ParamsKey, lptypes.PoolCounterKey} {
					require.Equal(t, before.Stores[lptypes.StoreKey][string(key)], committed.Stores[lptypes.StoreKey][string(key)])
				}
			} else {
				require.Equal(t, lpCachePoolID, response.(*lptypes.MsgCreatePoolResponse).PoolId)
				require.NotEqual(t, before.Stores[lptypes.StoreKey][string(lptypes.ParamsKey)], committed.Stores[lptypes.StoreKey][string(lptypes.ParamsKey)])
				require.NotEqual(t, before.Stores[lptypes.StoreKey][string(lptypes.PoolCounterKey)], committed.Stores[lptypes.StoreKey][string(lptypes.PoolCounterKey)])
				require.NotContains(t, k.GetParams(f.h.Ctx).AllowedPoolDenoms, lpCacheCounterDenom)
			}
			switch operation {
			case "create", "add":
				deposit, reserves := lpCacheInitial, lpCacheInitial
				if operation == "add" {
					deposit, reserves = 1_000_000_000, 11_000_000_000
					added := response.(*lptypes.MsgAddLiquidityResponse)
					require.Equal(t, "1000000000", added.ActualA)
					require.Equal(t, "1000000000", added.ActualB)
					require.Equal(t, "1000000000", added.LpTokensMinted)
				}
				for _, denom := range []string{lptypes.ZRNDenom, lpCacheCounterDenom} {
					lpCacheDelta(t, before.Actor, committed.Actor, denom, -deposit)
					lpCacheDelta(t, before.Module, committed.Module, denom, deposit)
				}
				lpCacheDelta(t, before.Actor, committed.Actor, lpDenom, deposit)
				lpCacheDelta(t, before.Module, committed.Module, lpDenom, 0)
				lpCacheDelta(t, before.Supply, committed.Supply, lpDenom, deposit)
				require.Equal(t, sdkmath.NewInt(reserves).String(), pool.ReserveA)
				require.Equal(t, sdkmath.NewInt(reserves).String(), pool.ReserveB)
				require.Equal(t, 1, lpCacheEventCount(child, banktypes.EventTypeCoinMint))
			case "swap":
				swapped := response.(*lptypes.MsgSwapResponse)
				require.Equal(t, quote.TokenOutAmount, swapped.TokenOutAmount)
				require.Equal(t, quote.FeeAmount, swapped.FeeAmount)
				out := lpCacheAmount(t, swapped.TokenOutAmount)
				lpCacheDelta(t, before.Actor, committed.Actor, lptypes.ZRNDenom, -1_000_000)
				lpCacheDelta(t, before.Module, committed.Module, lptypes.ZRNDenom, 1_000_000)
				lpCacheDelta(t, before.Actor, committed.Actor, lpCacheCounterDenom, out)
				lpCacheDelta(t, before.Module, committed.Module, lpCacheCounterDenom, -out)
				require.Equal(t, committed.Module.AmountOf(lptypes.ZRNDenom).String(), pool.ReserveA)
				require.Equal(t, committed.Module.AmountOf(lpCacheCounterDenom).String(), pool.ReserveB)
				require.Equal(t, before.Supply, committed.Supply)
			case "remove":
				removed := response.(*lptypes.MsgRemoveLiquidityResponse)
				require.Equal(t, "10000000000", removed.AmountA)
				require.Equal(t, "10000000000", removed.AmountB)
				for _, denom := range []string{lptypes.ZRNDenom, lpCacheCounterDenom} {
					lpCacheDelta(t, before.Actor, committed.Actor, denom, lpCacheInitial)
					require.True(t, committed.Module.AmountOf(denom).IsZero())
				}
				require.True(t, committed.Actor.AmountOf(lpDenom).IsZero())
				require.True(t, committed.Module.AmountOf(lpDenom).IsZero())
				require.True(t, committed.Supply.AmountOf(lpDenom).IsZero())
				require.Equal(t, "0", pool.ReserveA)
				require.Equal(t, "0", pool.ReserveB)
				require.Equal(t, lptypes.PoolStatus_POOL_STATUS_CLOSED, pool.Status)
				require.Equal(t, uint64(f.h.Ctx.BlockHeight()), pool.ClosedAtBlock)
				require.Nil(t, k.GetPoolByDenoms(f.h.Ctx, lptypes.ZRNDenom, lpCacheCounterDenom))
				require.Zero(t, k.CountOpenPools(f.h.Ctx))
				_, found = k.GetTWAPAccumulator(f.h.Ctx, lpCachePoolID)
				require.False(t, found)
				require.True(t, k.IsTWAPHistoryDeletionScheduled(f.h.Ctx, lpCachePoolID))
				require.NotEmpty(t, committed.Stores[lptypes.StoreKey][string(lptypes.TWAPObservationKey(lpCachePoolID, uint64(f.h.Ctx.BlockHeight())))], "close queues observations rather than eagerly deleting them")
				require.Equal(t, 1, lpCacheEventCount(child, banktypes.EventTypeCoinBurn))
			}
			if operation != "remove" {
				require.Equal(t, lptypes.PoolStatus_POOL_STATUS_ACTIVE, pool.Status)
				require.Equal(t, uint64(1), k.CountOpenPools(f.h.Ctx))
				require.Equal(t, lpCachePoolID, k.GetPoolByDenoms(f.h.Ctx, lptypes.ZRNDenom, lpCacheCounterDenom).PoolId)
				acc, found := k.GetTWAPAccumulator(f.h.Ctx, lpCachePoolID)
				require.True(t, found)
				require.Equal(t, uint64(f.h.Ctx.BlockHeight()), acc.LastBlock)
				require.False(t, k.IsTWAPHistoryDeletionScheduled(f.h.Ctx, lpCachePoolID))
			}
			t.Logf("operation=%s: successful child invisible before write; exact bank/auth/LP stores and ordered events published once", operation)
		})
	}
}

// TestLiquidityCacheLateFailures characterizes caller-owned SDK CacheContext
// atomicity using the installed app keeper, including its canonical-LP mint
// restriction. No effectful bank wrapper or production source mutation is used.
// Missing permissions and insufficient backing are deliberately damaged fixture
// prestates, not alleged normal admission paths or observations of live funds.
func TestLiquidityCacheLateFailures(t *testing.T) {
	cases := []struct{ operation, fault string }{
		{"create", "missing_minter"}, {"create", "blocked_payout"},
		{"add", "missing_minter"}, {"add", "blocked_payout"},
		{"swap", "blocked_payout"}, {"swap", "unbacked_output"},
		{"remove", "missing_burner"}, {"remove", "blocked_payout"}, {"remove", "unbacked_output"},
	}
	for _, tc := range cases {
		t.Run(tc.operation+"_"+tc.fault, func(t *testing.T) {
			for _, cached := range []bool{true, false} {
				name := "direct_uncached"
				if cached {
					name = "discard_cache"
				}
				t.Run(name, func(t *testing.T) {
					f := lpCachePrepareFault(t, tc.operation, tc.fault)
					before := lpCacheSnapshot(t, f, f.h.Ctx)
					callCtx := f.h.Ctx
					if cached {
						callCtx, _ = f.h.Ctx.CacheContext()
					}
					response, err, panicked := lpCacheInvoke(func() (any, error) { return f.call(callCtx, tc.operation) })
					require.Nil(t, response)
					require.Error(t, err)
					switch tc.fault {
					case "missing_minter", "missing_burner":
						require.True(t, panicked)
						require.ErrorIs(t, err, sdkerrors.ErrUnauthorized)
						action := "mint"
						if tc.fault == "missing_burner" {
							action = "burn"
						}
						require.ErrorContains(t, err, "does not have permissions to "+action+" tokens")
					default:
						require.False(t, panicked)
						if tc.fault == "blocked_payout" {
							require.ErrorIs(t, err, sdkerrors.ErrUnauthorized)
							require.ErrorContains(t, err, "is not allowed to receive funds")
						} else {
							require.ErrorIs(t, err, sdkerrors.ErrInsufficientFunds)
							require.ErrorContains(t, err, "spendable balance")
						}
						prefix := "failed to send LP tokens"
						if tc.operation == "swap" {
							prefix = "output transfer failed"
						}
						if tc.operation == "remove" {
							prefix = "failed to return underlying assets"
						}
						require.ErrorContains(t, err, prefix)
					}
					partial := lpCacheSnapshot(t, f, callCtx)
					lpCacheAssertLateEffects(t, f, callCtx, tc.operation, tc.fault, before, partial)
					if cached {
						// Discard: deliberately never invoke this branch's write function.
						require.Equal(t, before, lpCacheSnapshot(t, f, f.h.Ctx), "discarded cache must preserve complete base stores and events")
					} else {
						// No per-message cache. The harness base itself is an app test
						// context, not a claim of a committed disk/ABCI transaction.
						require.NotEqual(t, before.Actor.String(), partial.Actor.String(), "raw handler is not self-atomic")
						require.NotEqual(t, before.Events, partial.Events, "raw handler also leaves bank events")
						require.Equal(t, partial, lpCacheSnapshot(t, f, f.h.Ctx))
					}
					t.Logf("operation=%s fault=%s cached=%t panic=%t: real prior bank/state effects verified; error=%v", tc.operation, tc.fault, cached, panicked, err)
				})
			}
		})
	}
}
