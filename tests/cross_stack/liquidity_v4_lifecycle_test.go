package cross_stack_test

import (
	"bytes"
	"testing"

	"cosmossdk.io/core/appmodule"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

	liquiditypoolkeeper "github.com/zerone-chain/zerone/x/liquiditypool/keeper"
	liquiditypooltypes "github.com/zerone-chain/zerone/x/liquiditypool/types"
)

func TestLiquidityPoolV4RealAppLifecycle(t *testing.T) {
	h := NewTestHarness(t)
	// This legacy harness commits InitChain without FinalizeBlock, so genesis
	// cache writes are not flushed (see the proper restart harness). Restore the
	// native-lineage fact explicitly before exercising production liquidity.
	require.NoError(t, h.KnowledgeKeeper.WriteMigrationMarker(
		h.Ctx,
		"chain_lineage_native_consolidation-safety-v1",
		"genesis",
	))
	keeper := h.App.LiquidityPoolKeeper
	keeper.SetParams(h.Ctx, keeper.GetParams(h.Ctx))
	msgServer := liquiditypoolkeeper.NewMsgServerImpl(keeper)

	creator := sdk.AccAddress(bytes.Repeat([]byte{0x31}, 20))
	trader := sdk.AccAddress(bytes.Repeat([]byte{0x32}, 20))
	const counterDenom = "uatom"

	// This test deliberately calls the production message server at the state
	// transition boundary. The app's ante handler/signature verification is
	// covered elsewhere; all liquidity state, bank custody, module-account
	// mint/burn permissions, indexes, and lifecycle hooks below are real.
	bankMsgServer := bankkeeper.NewMsgServerImpl(h.BankKeeper)
	_, err := bankMsgServer.SetSendEnabled(h.Ctx, banktypes.NewMsgSetSendEnabled(
		h.BankKeeper.GetAuthority(),
		[]*banktypes.SendEnabled{
			{Denom: liquiditypooltypes.ZRNDenom, Enabled: true},
			{Denom: counterDenom, Enabled: true},
		},
		nil,
	))
	require.NoError(t, err)

	params := keeper.GetParams(h.Ctx)
	params.AllowedPoolDenoms = []string{counterDenom}
	params.PoolCreators = []string{creator.String()}
	_, err = msgServer.UpdateParams(h.Ctx, &liquiditypooltypes.MsgUpdateParams{
		Authority: keeper.GetAuthority(),
		Params:    params,
	})
	require.NoError(t, err)

	require.NoError(t, h.FundAccount(creator, sdk.NewCoins(
		sdk.NewCoin(liquiditypooltypes.ZRNDenom, sdkmath.NewInt(50_000_000_000)),
		sdk.NewCoin(counterDenom, sdkmath.NewInt(50_000_000_000)),
	)))
	require.NoError(t, h.FundAccount(trader, sdk.NewCoins(
		sdk.NewCoin(liquiditypooltypes.ZRNDenom, sdkmath.NewInt(2_000_000)),
	)))

	first, err := msgServer.CreatePool(h.Ctx, &liquiditypooltypes.MsgCreatePool{
		Creator:    creator.String(),
		DenomA:     liquiditypooltypes.ZRNDenom,
		DenomB:     counterDenom,
		AmountA:    "10000000000",
		AmountB:    "20000000000",
		SwapFeeBps: 0,
	})
	require.NoError(t, err)
	require.Equal(t, "pool-1", first.PoolId)

	firstPool, found := keeper.GetPool(h.Ctx, first.PoolId)
	require.True(t, found)
	require.Equal(t, liquiditypooltypes.PoolStatus_POOL_STATUS_ACTIVE, firstPool.Status)
	require.Equal(t, liquiditypooltypes.LPDenom(first.PoolId), firstPool.LpDenom)
	require.Equal(t, firstPool.LpTokenSupply, h.BankKeeper.GetSupply(h.Ctx, firstPool.LpDenom).Amount.String())
	require.Equal(t, first.PoolId, keeper.GetPoolByDenoms(
		h.Ctx, liquiditypooltypes.ZRNDenom, counterDenom,
	).PoolId)
	require.NotContains(t, keeper.GetParams(h.Ctx).AllowedPoolDenoms, counterDenom)

	// Advance the full app, then invoke the registered liquidity module's
	// production lifecycle hook explicitly. Some in-memory harness setups stop
	// their aggregate BeginBlock pass before later custom modules.
	h.AdvanceBlocks(1)
	liquidityLifecycle, ok := h.App.ModuleManager.Modules[liquiditypooltypes.ModuleName].(appmodule.HasBeginBlocker)
	require.True(t, ok)
	require.NoError(t, liquidityLifecycle.BeginBlock(h.Ctx))
	accumulator, found := keeper.GetTWAPAccumulator(h.Ctx, first.PoolId)
	require.True(t, found)
	require.Equal(t, uint64(h.Ctx.BlockHeight()), accumulator.LastBlock)

	added, err := msgServer.AddLiquidity(h.Ctx, &liquiditypooltypes.MsgAddLiquidity{
		Sender:      creator.String(),
		PoolId:      first.PoolId,
		AmountA:     "1000000000",
		AmountB:     "2000000000",
		MinLpTokens: "1",
	})
	require.NoError(t, err)
	require.NotEqual(t, "0", added.LpTokensMinted)

	swapped, err := msgServer.Swap(h.Ctx, &liquiditypooltypes.MsgSwap{
		Sender:        trader.String(),
		PoolId:        first.PoolId,
		TokenInDenom:  liquiditypooltypes.ZRNDenom,
		TokenInAmount: "1000000",
		MinTokenOut:   "1",
	})
	require.NoError(t, err)
	require.NotEqual(t, "0", swapped.TokenOutAmount)

	firstPool, found = keeper.GetPool(h.Ctx, first.PoolId)
	require.True(t, found)
	firstLPDenom := firstPool.LpDenom
	firstLPSupply := firstPool.LpTokenSupply
	require.Equal(t, firstLPSupply, h.BankKeeper.GetSupply(h.Ctx, firstLPDenom).Amount.String())

	_, err = msgServer.RemoveLiquidity(h.Ctx, &liquiditypooltypes.MsgRemoveLiquidity{
		Sender:     creator.String(),
		PoolId:     first.PoolId,
		LpTokens:   firstLPSupply,
		MinAmountA: "1",
		MinAmountB: "1",
	})
	require.NoError(t, err)

	tombstone, found := keeper.GetPool(h.Ctx, first.PoolId)
	require.True(t, found)
	require.Equal(t, liquiditypooltypes.PoolStatus_POOL_STATUS_CLOSED, tombstone.Status)
	require.Equal(t, "0", tombstone.ReserveA)
	require.Equal(t, "0", tombstone.ReserveB)
	require.Equal(t, "0", tombstone.LpTokenSupply)
	require.Equal(t, uint64(h.Ctx.BlockHeight()), tombstone.ClosedAtBlock)
	require.Nil(t, keeper.GetPoolByDenoms(h.Ctx, liquiditypooltypes.ZRNDenom, counterDenom))
	require.Equal(t, uint64(0), keeper.CountOpenPools(h.Ctx))
	_, found = keeper.GetTWAPAccumulator(h.Ctx, first.PoolId)
	require.False(t, found)
	require.True(t, keeper.IsTWAPHistoryDeletionScheduled(h.Ctx, first.PoolId))
	keeper.ProcessTWAPGarbageCollection(h.Ctx)
	observationCount := 0
	keeper.IterateTWAPObservations(h.Ctx, first.PoolId, func(*liquiditypooltypes.TWAPObservation) bool {
		observationCount++
		return false
	})
	require.Zero(t, observationCount)
	require.True(t, h.BankKeeper.GetSupply(h.Ctx, firstLPDenom).Amount.IsZero())

	params = keeper.GetParams(h.Ctx)
	params.AllowedPoolDenoms = []string{counterDenom}
	_, err = msgServer.UpdateParams(h.Ctx, &liquiditypooltypes.MsgUpdateParams{
		Authority: keeper.GetAuthority(),
		Params:    params,
	})
	require.NoError(t, err)

	second, err := msgServer.CreatePool(h.Ctx, &liquiditypooltypes.MsgCreatePool{
		Creator:    creator.String(),
		DenomA:     liquiditypooltypes.ZRNDenom,
		DenomB:     counterDenom,
		AmountA:    "10000000000",
		AmountB:    "20000000000",
		SwapFeeBps: 0,
	})
	require.NoError(t, err)
	require.Equal(t, "pool-2", second.PoolId)

	replacement, found := keeper.GetPool(h.Ctx, second.PoolId)
	require.True(t, found)
	require.NotEqual(t, first.PoolId, replacement.PoolId)
	require.NotEqual(t, firstLPDenom, replacement.LpDenom)
	require.Equal(t, second.PoolId, keeper.GetPoolByDenoms(
		h.Ctx, liquiditypooltypes.ZRNDenom, counterDenom,
	).PoolId)
	unchangedTombstone, found := keeper.GetPool(h.Ctx, first.PoolId)
	require.True(t, found)
	require.Equal(t, liquiditypooltypes.PoolStatus_POOL_STATUS_CLOSED, unchangedTombstone.Status)

	exported := keeper.ExportGenesis(h.Ctx)
	require.NoError(t, exported.Validate())
	require.Equal(t, uint64(3), exported.NextPoolId)
	require.Len(t, exported.Pools, 2)

	invariantReport, broken := liquiditypoolkeeper.StateConsistencyInvariant(keeper)(h.Ctx)
	require.False(t, broken, invariantReport)
}
