package cross_stack_test

import (
	"bytes"
	"encoding/json"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

	zeroneapp "github.com/zerone-chain/zerone/app"
	zeroneauthtypes "github.com/zerone-chain/zerone/x/auth/types"
	emergencytypes "github.com/zerone-chain/zerone/x/emergency/types"
	liquiditypoolkeeper "github.com/zerone-chain/zerone/x/liquiditypool/keeper"
	liquiditypooltypes "github.com/zerone-chain/zerone/x/liquiditypool/types"
)

// These tests characterize the existing message/bank contracts with synthetic
// funds. Direct message servers do not authenticate signers or run app ante.
// No block-advance helper, production policy relaxation, or mock bank is used.
type lpBearerFixture struct {
	h                  *TestHarness
	lp                 liquiditypooltypes.MsgServer
	bank               banktypes.MsgServer
	creator, holder    sdk.AccAddress
	poolID             string
	funding            sdk.Coins
	supplyAfterFunding sdk.Coins
}

func lpBearerNew(t *testing.T, counterAmount string, reverse bool) *lpBearerFixture {
	t.Helper()
	h := NewTestHarness(t)
	t.Cleanup(func() { require.NoError(t, h.App.Close()) })
	f := &lpBearerFixture{
		h: h, lp: liquiditypoolkeeper.NewMsgServerImpl(h.App.LiquidityPoolKeeper),
		bank:    bankkeeper.NewMsgServerImpl(h.BankKeeper),
		creator: sdk.AccAddress(bytes.Repeat([]byte{0x71}, 20)),
		holder:  sdk.AccAddress(bytes.Repeat([]byte{0x72}, 20)),
	}
	f.lpBearerSendEnabled(t, true, liquiditypooltypes.ZRNDenom, "uatom")
	params := h.App.LiquidityPoolKeeper.GetParams(h.Ctx)
	require.Equal(t, liquiditypooltypes.DefaultParams().MinInitialLiquidity, params.MinInitialLiquidity)
	require.Equal(t, liquiditypooltypes.DefaultParams().MinReserve, params.MinReserve)
	params.AllowedPoolDenoms = []string{"uatom"}
	params.PoolCreators = []string{f.creator.String()}
	_, err := f.lp.UpdateParams(h.Ctx, &liquiditypooltypes.MsgUpdateParams{
		Authority: h.App.LiquidityPoolKeeper.GetAuthority(), Params: params,
	})
	require.NoError(t, err)
	f.funding = sdk.NewCoins(
		sdk.NewCoin(liquiditypooltypes.ZRNDenom, lpBearerInt(t, params.MinInitialLiquidity)),
		sdk.NewCoin("uatom", lpBearerInt(t, counterAmount)),
	)
	require.NoError(t, h.FundAccount(f.creator, f.funding))
	// Default genesis already contains native supply for validator custody.
	for _, coin := range f.funding {
		f.supplyAfterFunding = f.supplyAfterFunding.Add(h.BankKeeper.GetSupply(h.Ctx, coin.Denom))
	}
	msg := &liquiditypooltypes.MsgCreatePool{
		Creator: f.creator.String(), DenomA: liquiditypooltypes.ZRNDenom, DenomB: "uatom",
		AmountA: params.MinInitialLiquidity, AmountB: counterAmount,
	}
	if reverse {
		msg.DenomA, msg.DenomB = msg.DenomB, msg.DenomA
		msg.AmountA, msg.AmountB = msg.AmountB, msg.AmountA
	}
	created, err := f.lp.CreatePool(h.Ctx, msg)
	require.NoError(t, err)
	f.poolID = created.PoolId
	require.NotEqual(t, f.creator, f.holder)
	require.NotContains(t, h.App.LiquidityPoolKeeper.GetParams(h.Ctx).PoolCreators, f.holder.String())
	f.lpBearerConserved(t)
	return f
}

func lpBearerInt(t *testing.T, value string) sdkmath.Int {
	t.Helper()
	n, ok := sdkmath.NewIntFromString(value)
	require.True(t, ok, value)
	return n
}

func (f *lpBearerFixture) lpBearerPool(t *testing.T) *liquiditypooltypes.Pool {
	t.Helper()
	pool, found := f.h.App.LiquidityPoolKeeper.GetPool(f.h.Ctx, f.poolID)
	require.True(t, found)
	return pool
}

func (f *lpBearerFixture) lpBearerSendEnabled(t *testing.T, enabled bool, denoms ...string) {
	t.Helper()
	entries := make([]*banktypes.SendEnabled, 0, len(denoms))
	for _, denom := range denoms {
		entries = append(entries, &banktypes.SendEnabled{Denom: denom, Enabled: enabled})
	}
	_, err := f.bank.SetSendEnabled(f.h.Ctx, banktypes.NewMsgSetSendEnabled(
		f.h.BankKeeper.GetAuthority(), entries, nil,
	))
	require.NoError(t, err)
	for _, denom := range denoms {
		require.Equal(t, enabled, f.h.BankKeeper.IsSendEnabledCoin(f.h.Ctx, sdk.NewInt64Coin(denom, 1)))
	}
}

func (f *lpBearerFixture) lpBearerTransfer(t *testing.T, amount string) {
	t.Helper()
	pool := f.lpBearerPool(t)
	f.lpBearerSendEnabled(t, true, pool.LpDenom)
	coins := sdk.NewCoins(sdk.NewCoin(pool.LpDenom, lpBearerInt(t, amount)))
	creatorBefore := f.h.BankKeeper.GetBalance(f.h.Ctx, f.creator, pool.LpDenom).Amount
	holderBefore := f.h.BankKeeper.GetBalance(f.h.Ctx, f.holder, pool.LpDenom).Amount
	_, err := f.bank.Send(f.h.Ctx, &banktypes.MsgSend{
		FromAddress: f.creator.String(), ToAddress: f.holder.String(), Amount: coins,
	})
	require.NoError(t, err)
	require.Equal(t, creatorBefore.Sub(coins[0].Amount).String(), f.h.GetBalance(f.creator, pool.LpDenom).Amount.String())
	require.Equal(t, holderBefore.Add(coins[0].Amount).String(), f.h.GetBalance(f.holder, pool.LpDenom).Amount.String())
	f.lpBearerConserved(t)
}

func (f *lpBearerFixture) lpBearerStatus(t *testing.T, status liquiditypooltypes.PoolStatus) {
	t.Helper()
	_, err := f.lp.SetPoolStatus(f.h.Ctx, &liquiditypooltypes.MsgSetPoolStatus{
		Authority: f.h.App.LiquidityPoolKeeper.GetAuthority(), PoolId: f.poolID, Status: status,
	})
	require.NoError(t, err)
	require.Equal(t, status, f.lpBearerPool(t).Status)
}

func (f *lpBearerFixture) lpBearerConserved(t *testing.T) {
	t.Helper()
	pool := f.lpBearerPool(t)
	module := authtypes.NewModuleAddress(liquiditypooltypes.ModuleName)
	for _, coin := range f.funding {
		total := sdkmath.ZeroInt()
		for _, addr := range []sdk.AccAddress{f.creator, f.holder, module} {
			total = total.Add(f.h.GetBalance(addr, coin.Denom).Amount)
		}
		require.Equal(t, coin.Amount.String(), total.String(), "underlying custody: %s", coin.Denom)
		require.Equal(t, f.supplyAfterFunding.AmountOf(coin.Denom).String(), f.h.BankKeeper.GetSupply(f.h.Ctx, coin.Denom).Amount.String())
	}
	require.Equal(t, pool.ReserveA, f.h.GetBalance(module, pool.DenomA).Amount.String())
	require.Equal(t, pool.ReserveB, f.h.GetBalance(module, pool.DenomB).Amount.String())
	require.Equal(t, pool.LpTokenSupply, f.h.BankKeeper.GetSupply(f.h.Ctx, pool.LpDenom).Amount.String())
	require.Equal(t, pool.LpTokenSupply, f.h.GetBalance(f.creator, pool.LpDenom).Amount.Add(
		f.h.GetBalance(f.holder, pool.LpDenom).Amount,
	).String())
	require.True(t, f.h.GetBalance(module, pool.LpDenom).IsZero())
	report, broken := liquiditypoolkeeper.StateConsistencyInvariant(f.h.App.LiquidityPoolKeeper)(f.h.Ctx)
	require.False(t, broken, report)
}

// Read the entire bank and liquidity stores, including indexes/TWAP, plus
// emitted events. Equality proves early refusals have no effects in this scope;
// it is not a full-app AppHash or gas/fee/sequence equivalence assertion.
func (f *lpBearerFixture) lpBearerSnapshot(t *testing.T, ctx sdk.Context, extraKeys ...string) map[string]string {
	t.Helper()
	state := make(map[string]string)
	keys := append([]string{banktypes.StoreKey, liquiditypooltypes.StoreKey}, extraKeys...)
	for _, key := range keys {
		iter := ctx.KVStore(f.h.App.GetStoreKeyForTests(key)).Iterator(nil, nil)
		for ; iter.Valid(); iter.Next() {
			state[key+":"+string(iter.Key())] = string(iter.Value())
		}
		require.NoError(t, iter.Close())
	}
	events, err := json.Marshal(ctx.EventManager().Events())
	require.NoError(t, err)
	state["events"] = string(events)
	return state
}

func (f *lpBearerFixture) lpBearerRedeem(t *testing.T, sender sdk.AccAddress, shares string) {
	t.Helper()
	before := f.lpBearerPool(t)
	amount := lpBearerInt(t, shares)
	supply := lpBearerInt(t, before.LpTokenSupply)
	// Independent integer floor expectations; do not reuse the production
	// CalculateWithdrawalAmounts implementation as the test oracle.
	wantA := lpBearerInt(t, before.ReserveA).Mul(amount).Quo(supply)
	wantB := lpBearerInt(t, before.ReserveB).Mul(amount).Quo(supply)
	balanceA := f.h.GetBalance(sender, before.DenomA).Amount
	balanceB := f.h.GetBalance(sender, before.DenomB).Amount
	lpBalance := f.h.GetBalance(sender, before.LpDenom).Amount
	response, err := f.lp.RemoveLiquidity(f.h.Ctx, &liquiditypooltypes.MsgRemoveLiquidity{
		Sender: sender.String(), PoolId: f.poolID, LpTokens: shares, MinAmountA: "1", MinAmountB: "1",
	})
	require.NoError(t, err)
	require.Equal(t, wantA.String(), response.AmountA)
	require.Equal(t, wantB.String(), response.AmountB)
	after := f.lpBearerPool(t)
	require.Equal(t, lpBearerInt(t, before.ReserveA).Sub(wantA).String(), after.ReserveA)
	require.Equal(t, lpBearerInt(t, before.ReserveB).Sub(wantB).String(), after.ReserveB)
	require.Equal(t, supply.Sub(amount).String(), after.LpTokenSupply)
	require.Equal(t, balanceA.Add(wantA).String(), f.h.GetBalance(sender, before.DenomA).Amount.String())
	require.Equal(t, balanceB.Add(wantB).String(), f.h.GetBalance(sender, before.DenomB).Amount.String())
	require.Equal(t, lpBalance.Sub(amount).String(), f.h.GetBalance(sender, before.LpDenom).Amount.String())
	f.lpBearerConserved(t)
}

func TestLiquidityBearerTransferredRedemption(t *testing.T) {
	for _, status := range []liquiditypooltypes.PoolStatus{
		liquiditypooltypes.PoolStatus_POOL_STATUS_ACTIVE,
		liquiditypooltypes.PoolStatus_POOL_STATUS_SWAPS_PAUSED,
		liquiditypooltypes.PoolStatus_POOL_STATUS_EXIT_ONLY,
	} {
		t.Run(status.String(), func(t *testing.T) {
			f := lpBearerNew(t, "20000000000", false)
			pool := f.lpBearerPool(t)
			f.lpBearerTransfer(t, pool.LpTokenSupply)
			f.lpBearerStatus(t, status)

			// Creator metadata gives no right to redeem shares no longer held.
			// Failed direct consumers discard caches; late-error atomicity is
			// a separate test group, not assumed from the handler return value.
			before := f.lpBearerSnapshot(t, f.h.Ctx)
			cache, _ := f.h.Ctx.CacheContext()
			response, err := f.lp.RemoveLiquidity(cache, &liquiditypooltypes.MsgRemoveLiquidity{
				Sender: f.creator.String(), PoolId: f.poolID, LpTokens: pool.LpTokenSupply,
			})
			require.ErrorContains(t, err, "failed to collect LP tokens")
			require.Nil(t, response)
			require.Equal(t, before, f.lpBearerSnapshot(t, f.h.Ctx))

			part := lpBearerInt(t, pool.LpTokenSupply).QuoRaw(3)
			f.lpBearerRedeem(t, f.holder, part.String())
			require.Equal(t, f.creator.String(), f.lpBearerPool(t).Creator)
			require.Equal(t, status, f.lpBearerPool(t).Status)
			f.lpBearerRedeem(t, f.holder, f.lpBearerPool(t).LpTokenSupply)
			closed := f.lpBearerPool(t)
			require.Equal(t, liquiditypooltypes.PoolStatus_POOL_STATUS_CLOSED, closed.Status)
			require.Equal(t, uint64(f.h.Ctx.BlockHeight()), closed.ClosedAtBlock)
			require.Nil(t, f.h.App.LiquidityPoolKeeper.GetPoolByDenoms(f.h.Ctx, pool.DenomA, pool.DenomB))
			require.Zero(t, f.h.App.LiquidityPoolKeeper.CountOpenPools(f.h.Ctx))
			require.True(t, f.h.App.LiquidityPoolKeeper.IsTWAPHistoryDeletionScheduled(f.h.Ctx, f.poolID))
			_, found := f.h.App.LiquidityPoolKeeper.GetTWAPAccumulator(f.h.Ctx, f.poolID)
			require.False(t, found)

			before = f.lpBearerSnapshot(t, f.h.Ctx)
			_, err = f.lp.RemoveLiquidity(f.h.Ctx, &liquiditypooltypes.MsgRemoveLiquidity{
				Sender: f.holder.String(), PoolId: f.poolID, LpTokens: "1",
			})
			require.ErrorIs(t, err, liquiditypooltypes.ErrPoolExitDisabled)
			_, err = f.lp.SetPoolStatus(f.h.Ctx, &liquiditypooltypes.MsgSetPoolStatus{
				Authority: f.h.App.LiquidityPoolKeeper.GetAuthority(), PoolId: f.poolID,
				Status: liquiditypooltypes.PoolStatus_POOL_STATUS_ACTIVE,
			})
			require.ErrorIs(t, err, liquiditypooltypes.ErrClosedPoolImmutable)
			require.Equal(t, before, f.lpBearerSnapshot(t, f.h.Ctx))
		})
	}
}

func TestLiquidityBearerZeroRoundedOutput(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		name := "zero_b"
		if reverse {
			name = "zero_a"
		}
		t.Run(name, func(t *testing.T) {
			// Default minimum native capital remains intact. Valid asymmetry,
			// not a lowered guard, makes one LP worth 100uzrn but <1uatom.
			f := lpBearerNew(t, "1000000", reverse)
			pool := f.lpBearerPool(t)
			require.Equal(t, "100000000", pool.LpTokenSupply)
			f.lpBearerTransfer(t, "1")
			f.lpBearerStatus(t, liquiditypooltypes.PoolStatus_POOL_STATUS_EXIT_ONLY)
			for _, sender := range []sdk.AccAddress{f.holder, f.creator} {
				before := f.lpBearerSnapshot(t, f.h.Ctx)
				response, err := f.lp.RemoveLiquidity(f.h.Ctx, &liquiditypooltypes.MsgRemoveLiquidity{
					Sender: sender.String(), PoolId: f.poolID, LpTokens: "1",
					// Empty minima are intentional: this is the zero-output
					// guard, not a user slippage limit or ownership failure.
				})
				require.ErrorIs(t, err, liquiditypooltypes.ErrZeroAmount)
				require.ErrorContains(t, err, "withdrawal rounds one asset to zero")
				require.Nil(t, response)
				require.Equal(t, before, f.lpBearerSnapshot(t, f.h.Ctx))
				f.lpBearerConserved(t)
			}

			// Floor remainders stay in custody for the last bearer. This
			// particular residual becomes redeemable after the majority exit;
			// EXIT_ONLY alone did not guarantee the earlier small exit.
			f.lpBearerRedeem(t, f.creator, lpBearerInt(t, pool.LpTokenSupply).SubRaw(1).String())
			residual := f.lpBearerPool(t)
			require.Equal(t, "1", residual.LpTokenSupply)
			if reverse {
				require.Equal(t, "1", residual.ReserveA)
				require.Equal(t, "100", residual.ReserveB)
			} else {
				require.Equal(t, "100", residual.ReserveA)
				require.Equal(t, "1", residual.ReserveB)
			}
			f.lpBearerRedeem(t, f.holder, "1")
			require.Equal(t, liquiditypooltypes.PoolStatus_POOL_STATUS_CLOSED, f.lpBearerPool(t).Status)
		})
	}
}

func TestLiquidityBearerSendDisabledRedemption(t *testing.T) {
	for _, tc := range []struct {
		name   string
		lp     bool
		assets []string
	}{
		{name: "lp", lp: true},
		{name: "native", assets: []string{liquiditypooltypes.ZRNDenom}},
		{name: "counter", assets: []string{"uatom"}},
		{name: "lp_and_both_assets", lp: true, assets: []string{liquiditypooltypes.ZRNDenom, "uatom"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := lpBearerNew(t, "20000000000", false)
			pool := f.lpBearerPool(t)
			f.lpBearerTransfer(t, pool.LpTokenSupply)
			f.lpBearerStatus(t, liquiditypooltypes.PoolStatus_POOL_STATUS_EXIT_ONLY)
			disabled := append([]string{}, tc.assets...)
			if tc.lp {
				disabled = append(disabled, pool.LpDenom)
			}
			f.lpBearerSendEnabled(t, false, disabled...)

			// Module collection/burn/payout deliberately differs from public
			// SDK MsgSend. The bank flag does not trap already-held LPs.
			part := lpBearerInt(t, pool.LpTokenSupply).QuoRaw(2)
			f.lpBearerRedeem(t, f.holder, part.String())
			for _, denom := range disabled {
				require.True(t, f.h.GetBalance(f.holder, denom).Amount.IsPositive())
				before := f.lpBearerSnapshot(t, f.h.Ctx)
				response, err := f.bank.Send(f.h.Ctx, &banktypes.MsgSend{
					FromAddress: f.holder.String(), ToAddress: f.creator.String(),
					Amount: sdk.NewCoins(sdk.NewInt64Coin(denom, 1)),
				})
				require.ErrorIs(t, err, banktypes.ErrSendDisabled)
				require.Nil(t, response)
				require.Equal(t, before, f.lpBearerSnapshot(t, f.h.Ctx))
			}
			f.lpBearerRedeem(t, f.holder, f.lpBearerPool(t).LpTokenSupply)
			require.Equal(t, liquiditypooltypes.PoolStatus_POOL_STATUS_CLOSED, f.lpBearerPool(t).Status)
			for _, denom := range disabled {
				require.False(t, f.h.BankKeeper.IsSendEnabledCoin(f.h.Ctx, sdk.NewInt64Coin(denom, 1)))
			}
		})
	}
}

func TestLiquidityBearerGovernedRecreation(t *testing.T) {
	f := lpBearerNew(t, "20000000000", false)
	first := f.lpBearerPool(t)
	f.lpBearerTransfer(t, first.LpTokenSupply)
	f.lpBearerStatus(t, liquiditypooltypes.PoolStatus_POOL_STATUS_EXIT_ONLY)

	before := f.lpBearerSnapshot(t, f.h.Ctx)
	for _, sender := range []sdk.AccAddress{f.creator, f.holder} {
		_, err := f.lp.SetPoolStatus(f.h.Ctx, &liquiditypooltypes.MsgSetPoolStatus{
			Authority: sender.String(), PoolId: f.poolID, Status: liquiditypooltypes.PoolStatus_POOL_STATUS_ACTIVE,
		})
		require.ErrorIs(t, err, liquiditypooltypes.ErrUnauthorized)
	}
	require.Equal(t, before, f.lpBearerSnapshot(t, f.h.Ctx))
	// EXIT_ONLY is governance-reversible, not an irreversible retirement.
	f.lpBearerStatus(t, liquiditypooltypes.PoolStatus_POOL_STATUS_ACTIVE)
	f.lpBearerStatus(t, liquiditypooltypes.PoolStatus_POOL_STATUS_EXIT_ONLY)
	f.lpBearerRedeem(t, f.holder, first.LpTokenSupply)
	closedBytes, err := json.Marshal(f.lpBearerPool(t))
	require.NoError(t, err)

	create := &liquiditypooltypes.MsgCreatePool{
		Creator: f.creator.String(), DenomA: first.DenomA, DenomB: first.DenomB,
		AmountA: first.ReserveA, AmountB: first.ReserveB,
	}
	before = f.lpBearerSnapshot(t, f.h.Ctx)
	response, err := f.lp.CreatePool(f.h.Ctx, create)
	require.ErrorIs(t, err, liquiditypooltypes.ErrPoolDenomNotAllowed)
	require.Nil(t, response)
	require.Equal(t, before, f.lpBearerSnapshot(t, f.h.Ctx))

	// Refunds are real bearer balances, not creator-owned bookkeeping.
	// Return them by public SDK MsgSend before the admitted creator recreates.
	_, err = f.bank.Send(f.h.Ctx, &banktypes.MsgSend{
		FromAddress: f.holder.String(), ToAddress: f.creator.String(), Amount: f.funding,
	})
	require.NoError(t, err)
	f.lpBearerConserved(t)
	params := f.h.App.LiquidityPoolKeeper.GetParams(f.h.Ctx)
	params.AllowedPoolDenoms = []string{"uatom"}
	_, err = f.lp.UpdateParams(f.h.Ctx, &liquiditypooltypes.MsgUpdateParams{
		Authority: f.h.App.LiquidityPoolKeeper.GetAuthority(), Params: params,
	})
	require.NoError(t, err)
	before = f.lpBearerSnapshot(t, f.h.Ctx)
	create.Creator = f.holder.String()
	response, err = f.lp.CreatePool(f.h.Ctx, create)
	require.ErrorIs(t, err, liquiditypooltypes.ErrPoolCreatorNotAllowed)
	require.Nil(t, response)
	require.Equal(t, before, f.lpBearerSnapshot(t, f.h.Ctx))
	create.Creator = f.creator.String()
	response, err = f.lp.CreatePool(f.h.Ctx, create)
	require.NoError(t, err)
	f.poolID = response.PoolId
	second := f.lpBearerPool(t)
	require.Equal(t, "pool-1", first.PoolId)
	require.Equal(t, "pool-2", second.PoolId)
	require.NotEqual(t, first.LpDenom, second.LpDenom)
	require.Equal(t, second.PoolId, f.h.App.LiquidityPoolKeeper.GetPoolByDenoms(f.h.Ctx, first.DenomA, first.DenomB).PoolId)
	require.True(t, f.h.BankKeeper.GetSupply(f.h.Ctx, first.LpDenom).IsZero())
	require.Empty(t, f.h.App.LiquidityPoolKeeper.GetParams(f.h.Ctx).AllowedPoolDenoms)
	f.lpBearerConserved(t)
	f.lpBearerTransfer(t, second.LpTokenSupply)
	f.lpBearerRedeem(t, f.holder, second.LpTokenSupply)
	unchanged, found := f.h.App.LiquidityPoolKeeper.GetPool(f.h.Ctx, first.PoolId)
	require.True(t, found)
	unchangedBytes, err := json.Marshal(unchanged)
	require.NoError(t, err)
	require.Equal(t, closedBytes, unchangedBytes)
	exported := f.h.App.LiquidityPoolKeeper.ExportGenesis(f.h.Ctx)
	require.NoError(t, exported.Validate())
	require.Len(t, exported.Pools, 2)
	require.Equal(t, uint64(3), exported.NextPoolId)
}

func TestLiquidityBearerGlobalAnteOwners(t *testing.T) {
	// These are the production public ante decorators with real app keepers
	// and an unsigned SDK TxConfig transaction carrying the real LP message.
	// They characterize policy AFTER the signature precondition, not signature
	// authentication, complete app ante, fee charging, BaseApp or transport.
	// Emergency/freeze state is fixture-seeded at its owner; no governance or
	// emergency ceremony authorization is claimed by these setter calls.
	for _, tc := range []struct {
		name          string
		emergency     emergencytypes.EmergencyStatus
		frozenHolder  bool
		frozenCreator bool
		releaseLatch  bool
		nextHeight    bool
		wantErr       error
	}{
		{name: "normal", emergency: emergencytypes.StatusNormal},
		{name: "halt_vote_not_quarantine", emergency: emergencytypes.StatusHaltVoting},
		{name: "halted", emergency: emergencytypes.StatusHalted, wantErr: emergencytypes.ErrChainHalted},
		{name: "revert_voting", emergency: emergencytypes.StatusRevertVoting, wantErr: emergencytypes.ErrChainHalted},
		{name: "reverting", emergency: emergencytypes.StatusReverting, wantErr: emergencytypes.ErrChainHalted},
		{name: "resume_voting", emergency: emergencytypes.StatusResumeVoting, wantErr: emergencytypes.ErrChainHalted},
		{name: "normal_same_block_release_latch", emergency: emergencytypes.StatusNormal, releaseLatch: true, wantErr: emergencytypes.ErrChainHalted},
		{name: "normal_next_block_release", emergency: emergencytypes.StatusNormal, releaseLatch: true, nextHeight: true},
		{name: "frozen_holder", emergency: emergencytypes.StatusNormal, frozenHolder: true, wantErr: zeroneauthtypes.ErrAccountFrozen},
		{name: "frozen_creator_not_holder", emergency: emergencytypes.StatusNormal, frozenCreator: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := lpBearerNew(t, "20000000000", false)
			pool := f.lpBearerPool(t)
			f.lpBearerTransfer(t, pool.LpTokenSupply)
			f.lpBearerStatus(t, liquiditypooltypes.PoolStatus_POOL_STATUS_EXIT_ONLY)
			// The quarantine latch has separate CheckTx/Finalize semantics.
			// Exercise explicit execution height, not a fake block replay.
			f.h.Ctx = f.h.Ctx.WithIsCheckTx(false)
			f.h.EmergencyKeeper.SetEmergencyStatus(f.h.Ctx, tc.emergency)
			if tc.releaseLatch {
				f.h.EmergencyKeeper.SetQuarantineReleaseBlock(f.h.Ctx, uint64(f.h.Ctx.BlockHeight()))
			}
			if tc.nextHeight {
				f.h.Ctx = f.h.Ctx.WithBlockHeight(f.h.Ctx.BlockHeight() + 1)
			}
			for _, account := range []struct {
				addr   sdk.AccAddress
				frozen bool
			}{{f.holder, tc.frozenHolder}, {f.creator, tc.frozenCreator}} {
				if account.frozen {
					f.h.AuthKeeper.SetAccount(f.h.Ctx, &zeroneauthtypes.Account{
						Address: account.addr.String(), AccountType: "agent",
						Flags:           &zeroneauthtypes.AccountFlags{Frozen: true},
						LastActiveBlock: uint64(f.h.Ctx.BlockHeight()),
					})
				}
			}
			msg := &liquiditypooltypes.MsgRemoveLiquidity{
				Sender: f.holder.String(), PoolId: f.poolID, LpTokens: pool.LpTokenSupply,
				MinAmountA: "1", MinAmountB: "1",
			}
			builder := f.h.App.TxConfig().NewTxBuilder()
			require.NoError(t, builder.SetMsgs(msg))
			tx := builder.GetTx()
			signers, err := tx.GetSigners()
			require.NoError(t, err)
			require.Equal(t, [][]byte{f.holder.Bytes()}, signers)

			extraStores := []string{authtypes.StoreKey, zeroneauthtypes.StoreKey, emergencytypes.StoreKey}
			before := f.lpBearerSnapshot(t, f.h.Ctx, extraStores...)
			entered := false
			var response *liquiditypooltypes.MsgRemoveLiquidityResponse
			_, err = zeroneapp.NewEmergencyHaltDecorator(f.h.EmergencyKeeper).AnteHandle(
				f.h.Ctx, tx, false,
				func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
					return zeroneapp.NewZeroneAccountDecorator(f.h.AuthKeeper).AnteHandle(
						ctx, tx, simulate,
						func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
							entered = true
							var exitErr error
							response, exitErr = f.lp.RemoveLiquidity(ctx, msg)
							return ctx, exitErr
						},
					)
				},
			)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				require.False(t, entered, "ante owner must refuse before the LP handler")
				require.Nil(t, response)
				require.Equal(t, before, f.lpBearerSnapshot(t, f.h.Ctx, extraStores...))

				// Control: the identical raw handler succeeds inside an
				// uncommitted cache. Calling it alone does NOT exercise ante.
				cache, _ := f.h.Ctx.CacheContext()
				direct, directErr := f.lp.RemoveLiquidity(cache, msg)
				require.NoError(t, directErr)
				require.Equal(t, pool.ReserveA, direct.AmountA)
				require.Equal(t, pool.ReserveB, direct.AmountB)
				require.True(t, f.h.BankKeeper.GetSupply(cache, pool.LpDenom).IsZero())
				require.Equal(t, pool.ReserveA, f.h.BankKeeper.GetBalance(cache, f.holder, pool.DenomA).Amount.String())
				require.Equal(t, pool.ReserveB, f.h.BankKeeper.GetBalance(cache, f.holder, pool.DenomB).Amount.String())
				report, broken := liquiditypoolkeeper.StateConsistencyInvariant(f.h.App.LiquidityPoolKeeper)(cache)
				require.False(t, broken, report)
				require.Equal(t, before, f.lpBearerSnapshot(t, f.h.Ctx, extraStores...))
			} else {
				require.NoError(t, err)
				require.True(t, entered)
				require.NotNil(t, response)
				require.Equal(t, pool.ReserveA, response.AmountA)
				require.Equal(t, pool.ReserveB, response.AmountB)
				require.Equal(t, liquiditypooltypes.PoolStatus_POOL_STATUS_CLOSED, f.lpBearerPool(t).Status)
			}
			f.lpBearerConserved(t)
		})
	}
}
