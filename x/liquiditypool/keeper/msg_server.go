package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/liquiditypool/types"
)

type msgServer struct {
	types.UnimplementedMsgServer
	Keeper
}

func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// CreatePool creates a governed constant-product AMM pool.
func (m msgServer) CreatePool(goCtx context.Context, msg *types.MsgCreatePool) (*types.MsgCreatePoolResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	params := m.Keeper.GetParams(ctx)
	if msg.SwapFeeBps != 0 {
		return nil, types.ErrInvalidSwapFee.Wrap("custom create-time fees are disabled; use the governed default")
	}
	creatorAllowed := false
	for _, creator := range params.PoolCreators {
		if creator == msg.Creator {
			creatorAllowed = true
			break
		}
	}
	if !creatorAllowed {
		return nil, types.ErrPoolCreatorNotAllowed
	}
	counterDenom := msg.DenomA
	if counterDenom == types.ZRNDenom {
		counterDenom = msg.DenomB
	}
	denomAdmissionIndex := -1
	for index, denom := range params.AllowedPoolDenoms {
		if denom == counterDenom {
			denomAdmissionIndex = index
			break
		}
	}
	if denomAdmissionIndex < 0 {
		return nil, types.ErrPoolDenomNotAllowed
	}
	if m.Keeper.CountOpenPools(ctx) >= params.MaxPools {
		return nil, types.ErrMaxPoolsReached
	}
	if existing := m.Keeper.GetPoolByDenoms(ctx, msg.DenomA, msg.DenomB); existing != nil {
		return nil, types.ErrPoolAlreadyExists
	}

	amountA, _ := types.ParsePositiveAmount(msg.AmountA)
	amountB, _ := types.ParsePositiveAmount(msg.AmountB)
	var zrnAmount *big.Int
	if msg.DenomA == types.ZRNDenom {
		zrnAmount = amountA
	} else {
		zrnAmount = amountB
	}
	minLiquidity, err := types.ParsePositiveAmount(params.MinInitialLiquidity)
	if err != nil {
		return nil, types.ErrInvalidPoolState.Wrapf("min_initial_liquidity: %v", err)
	}
	if zrnAmount.Cmp(minLiquidity) < 0 {
		return nil, types.ErrInsufficientLiquidity
	}

	feeBps := params.DefaultSwapFeeBps
	if feeBps > types.MaxSwapFeeBps {
		return nil, types.ErrInvalidSwapFee
	}

	counter := m.Keeper.GetNextPoolId(ctx)
	if counter > types.MaxPoolRecordsCap {
		return nil, types.ErrPoolRecordCapReached
	}
	poolID := fmt.Sprintf("pool-%d", counter)
	if _, exists := m.Keeper.GetPool(ctx, poolID); exists {
		return nil, types.ErrInvalidPoolState.Wrapf("next pool ID %s already exists", poolID)
	}
	if m.Keeper.bankKeeper != nil &&
		!m.Keeper.bankKeeper.GetSupply(ctx, types.LPDenom(poolID)).Amount.IsZero() {
		return nil, types.ErrInvalidPoolState.Wrapf(
			"LP denom %s already has bank supply", types.LPDenom(poolID),
		)
	}
	lpTokens := CalculateLPTokensForDeposit(amountA, amountB, amountA, amountB, new(big.Int))
	if lpTokens.Sign() <= 0 {
		return nil, types.ErrZeroLPTokens
	}

	creatorAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, fmt.Errorf("invalid creator address: %w", err)
	}
	deposit := sdk.NewCoins(
		sdk.NewCoin(msg.DenomA, sdkmath.NewIntFromBigInt(amountA)),
		sdk.NewCoin(msg.DenomB, sdkmath.NewIntFromBigInt(amountB)),
	)
	if m.Keeper.bankKeeper != nil {
		if err := m.Keeper.bankKeeper.IsSendEnabledCoins(ctx, deposit...); err != nil {
			return nil, fmt.Errorf("initial liquidity contains a send-disabled denom: %w", err)
		}
		if err := m.Keeper.bankKeeper.SendCoinsFromAccountToModule(
			ctx, creatorAddr, types.ModuleName, deposit,
		); err != nil {
			return nil, fmt.Errorf("initial liquidity transfer failed: %w", err)
		}
	}

	m.Keeper.IncrementPoolCounter(ctx)
	if ctx.BlockHeight() < 0 {
		return nil, types.ErrInvalidPoolState.Wrap("negative creation height")
	}
	pool := &types.Pool{
		PoolId:         poolID,
		DenomA:         msg.DenomA,
		DenomB:         msg.DenomB,
		ReserveA:       amountA.String(),
		ReserveB:       amountB.String(),
		SwapFeeBps:     feeBps,
		LpTokenSupply:  lpTokens.String(),
		LpDenom:        types.LPDenom(poolID),
		Creator:        msg.Creator,
		CreatedAtBlock: uint64(ctx.BlockHeight()),
		Status:         types.PoolStatus_POOL_STATUS_ACTIVE,
	}
	m.Keeper.SetPool(ctx, pool)

	if m.Keeper.bankKeeper != nil {
		lpCoins := sdk.NewCoins(sdk.NewCoin(pool.LpDenom, sdkmath.NewIntFromBigInt(lpTokens)))
		if err := m.Keeper.bankKeeper.MintCoins(ctx, types.ModuleName, lpCoins); err != nil {
			return nil, fmt.Errorf("failed to mint LP tokens: %w", err)
		}
		if err := m.Keeper.bankKeeper.SendCoinsFromModuleToAccount(
			ctx, types.ModuleName, creatorAddr, lpCoins,
		); err != nil {
			return nil, fmt.Errorf("failed to send LP tokens: %w", err)
		}
	}
	// Counter-denom admission is a one-shot governance grant. Consuming it on
	// successful creation prevents an allowlisted creator from repeatedly
	// closing/recreating a pair until the permanent pool-ID namespace fills.
	params.AllowedPoolDenoms = append(
		params.AllowedPoolDenoms[:denomAdmissionIndex:denomAdmissionIndex],
		params.AllowedPoolDenoms[denomAdmissionIndex+1:]...,
	)
	m.Keeper.SetParams(ctx, params)
	m.Keeper.UpdateTWAPAccumulator(ctx, pool)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.liquiditypool.pool_created",
		sdk.NewAttribute("pool_id", poolID),
		sdk.NewAttribute("denom_a", msg.DenomA),
		sdk.NewAttribute("denom_b", msg.DenomB),
		sdk.NewAttribute("reserve_a", amountA.String()),
		sdk.NewAttribute("reserve_b", amountB.String()),
		sdk.NewAttribute("lp_tokens", lpTokens.String()),
		sdk.NewAttribute("status", pool.Status.String()),
		sdk.NewAttribute("counter_denom_admission_consumed", counterDenom),
	))
	return &types.MsgCreatePoolResponse{PoolId: poolID}, nil
}

func (m msgServer) Swap(goCtx context.Context, msg *types.MsgSwap) (*types.MsgSwapResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	// A plan-less restart of the v5 source against legacy v3/v4 state must not
	// silently change fee routing before the named H1 migration. The legacy
	// nonzero field is therefore an activation sentinel: fail closed until the
	// 3→5 or 4→5 migration retires it at zero.
	if err := m.Keeper.requireLiquidityV5Activated(ctx); err != nil {
		return nil, err
	}
	quote, err := m.Keeper.checkedSwapQuote(ctx, msg.PoolId, msg.TokenInDenom, msg.TokenInAmount)
	if err != nil {
		return nil, err
	}
	minOut, err := types.ParseOptionalPositiveAmount(msg.MinTokenOut)
	if err != nil {
		return nil, fmt.Errorf("invalid min_token_out: %w", err)
	}
	if minOut != nil && quote.tokenOut.Cmp(minOut) < 0 {
		return nil, types.ErrSlippageExceeded
	}
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, fmt.Errorf("invalid sender address: %w", err)
	}

	inCoins := sdk.NewCoins(sdk.NewCoin(msg.TokenInDenom, sdkmath.NewIntFromBigInt(quote.tokenIn)))
	if m.Keeper.bankKeeper != nil {
		if err := m.Keeper.bankKeeper.IsSendEnabledCoins(ctx, inCoins...); err != nil {
			return nil, fmt.Errorf("swap input denom is send-disabled: %w", err)
		}
	}
	m.Keeper.UpdateTWAPAccumulator(ctx, quote.pool)
	quote.pool.Locked = true
	m.Keeper.SetPool(ctx, quote.pool)
	unlock := func() {
		quote.pool.Locked = false
		m.Keeper.SetPool(ctx, quote.pool)
	}

	// Consensus v5 has no protocol skim. The complete input amount remains in
	// module custody, while the constant-product spread increases reserves for
	// LP holders pro rata. Keep the zero-valued event field for wire/indexer
	// compatibility with the pre-v5 implementation.
	protocolFee := new(big.Int)
	if m.Keeper.bankKeeper != nil {
		if err := m.Keeper.bankKeeper.SendCoinsFromAccountToModule(
			ctx, senderAddr, types.ModuleName, inCoins,
		); err != nil {
			unlock()
			return nil, fmt.Errorf("input transfer failed: %w", err)
		}
		outCoins := sdk.NewCoins(sdk.NewCoin(quote.denomOut, sdkmath.NewIntFromBigInt(quote.tokenOut)))
		if err := m.Keeper.bankKeeper.SendCoinsFromModuleToAccount(
			ctx, types.ModuleName, senderAddr, outCoins,
		); err != nil {
			unlock()
			return nil, fmt.Errorf("output transfer failed: %w", err)
		}
	}

	newReserveIn := new(big.Int).Add(quote.reserveIn, quote.tokenIn)
	newReserveIn.Sub(newReserveIn, protocolFee)
	newReserveOut := new(big.Int).Sub(quote.reserveOut, quote.tokenOut)
	if msg.TokenInDenom == quote.pool.DenomA {
		quote.pool.ReserveA = newReserveIn.String()
		quote.pool.ReserveB = newReserveOut.String()
	} else {
		quote.pool.ReserveB = newReserveIn.String()
		quote.pool.ReserveA = newReserveOut.String()
	}
	quote.pool.Locked = false
	m.Keeper.SetPool(ctx, quote.pool)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.liquiditypool.swap",
		sdk.NewAttribute("pool_id", msg.PoolId),
		sdk.NewAttribute("sender", msg.Sender),
		sdk.NewAttribute("token_in", msg.TokenInDenom),
		sdk.NewAttribute("amount_in", quote.tokenIn.String()),
		sdk.NewAttribute("token_out", quote.denomOut),
		sdk.NewAttribute("amount_out", quote.tokenOut.String()),
		sdk.NewAttribute("fee", quote.feeAmount.String()),
		sdk.NewAttribute("protocol_fee", protocolFee.String()),
	))
	return &types.MsgSwapResponse{
		TokenOutAmount: quote.tokenOut.String(),
		FeeAmount:      quote.feeAmount.String(),
	}, nil
}

func (m msgServer) AddLiquidity(goCtx context.Context, msg *types.MsgAddLiquidity) (*types.MsgAddLiquidityResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	pool, found := m.Keeper.GetPool(ctx, msg.PoolId)
	if !found {
		return nil, types.ErrPoolNotFound
	}
	if !types.CanAddLiquidity(pool.Status) {
		return nil, types.ErrPoolAddDisabled
	}
	if pool.Locked {
		return nil, types.ErrPoolLocked
	}
	desiredA, err := types.ParsePositiveAmount(msg.AmountA)
	if err != nil {
		return nil, err
	}
	desiredB, err := types.ParsePositiveAmount(msg.AmountB)
	if err != nil {
		return nil, err
	}
	minLP, err := types.ParseOptionalPositiveAmount(msg.MinLpTokens)
	if err != nil {
		return nil, fmt.Errorf("invalid min_lp_tokens: %w", err)
	}
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, fmt.Errorf("invalid sender address: %w", err)
	}
	reserveA, err := types.ParsePositiveAmount(pool.ReserveA)
	if err != nil {
		return nil, types.ErrInvalidPoolState.Wrapf("reserve_a: %v", err)
	}
	reserveB, err := types.ParsePositiveAmount(pool.ReserveB)
	if err != nil {
		return nil, types.ErrInvalidPoolState.Wrapf("reserve_b: %v", err)
	}
	totalSupply, err := types.ParsePositiveAmount(pool.LpTokenSupply)
	if err != nil {
		return nil, types.ErrInvalidPoolState.Wrapf("lp_token_supply: %v", err)
	}
	actualA, actualB, lpTokens := CalculateDepositForShares(
		reserveA, reserveB, desiredA, desiredB, totalSupply,
	)
	if lpTokens.Sign() <= 0 || actualA.Sign() <= 0 || actualB.Sign() <= 0 {
		return nil, types.ErrZeroLPTokens
	}
	if minLP != nil && lpTokens.Cmp(minLP) < 0 {
		return nil, types.ErrSlippageExceeded
	}
	deposit := sdk.NewCoins(
		sdk.NewCoin(pool.DenomA, sdkmath.NewIntFromBigInt(actualA)),
		sdk.NewCoin(pool.DenomB, sdkmath.NewIntFromBigInt(actualB)),
	)
	if m.Keeper.bankKeeper != nil {
		if err := m.Keeper.bankKeeper.IsSendEnabledCoins(ctx, deposit...); err != nil {
			return nil, fmt.Errorf("liquidity deposit contains a send-disabled denom: %w", err)
		}
	}

	m.Keeper.UpdateTWAPAccumulator(ctx, pool)
	pool.Locked = true
	m.Keeper.SetPool(ctx, pool)
	unlock := func() {
		pool.Locked = false
		m.Keeper.SetPool(ctx, pool)
	}
	if m.Keeper.bankKeeper != nil {
		if err := m.Keeper.bankKeeper.SendCoinsFromAccountToModule(
			ctx, senderAddr, types.ModuleName, deposit,
		); err != nil {
			unlock()
			return nil, fmt.Errorf("liquidity transfer failed: %w", err)
		}
	}

	pool.ReserveA = new(big.Int).Add(reserveA, actualA).String()
	pool.ReserveB = new(big.Int).Add(reserveB, actualB).String()
	pool.LpTokenSupply = new(big.Int).Add(totalSupply, lpTokens).String()
	pool.Locked = false
	m.Keeper.SetPool(ctx, pool)

	if m.Keeper.bankKeeper != nil {
		lpCoins := sdk.NewCoins(sdk.NewCoin(pool.LpDenom, sdkmath.NewIntFromBigInt(lpTokens)))
		if err := m.Keeper.bankKeeper.MintCoins(ctx, types.ModuleName, lpCoins); err != nil {
			return nil, fmt.Errorf("failed to mint LP tokens: %w", err)
		}
		if err := m.Keeper.bankKeeper.SendCoinsFromModuleToAccount(
			ctx, types.ModuleName, senderAddr, lpCoins,
		); err != nil {
			return nil, fmt.Errorf("failed to send LP tokens: %w", err)
		}
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.liquiditypool.liquidity_added",
		sdk.NewAttribute("pool_id", msg.PoolId),
		sdk.NewAttribute("sender", msg.Sender),
		sdk.NewAttribute("amount_a", actualA.String()),
		sdk.NewAttribute("amount_b", actualB.String()),
		sdk.NewAttribute("lp_tokens", lpTokens.String()),
	))
	return &types.MsgAddLiquidityResponse{
		LpTokensMinted: lpTokens.String(),
		ActualA:        actualA.String(),
		ActualB:        actualB.String(),
	}, nil
}

func (m msgServer) RemoveLiquidity(goCtx context.Context, msg *types.MsgRemoveLiquidity) (*types.MsgRemoveLiquidityResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	pool, found := m.Keeper.GetPool(ctx, msg.PoolId)
	if !found {
		return nil, types.ErrPoolNotFound
	}
	if !types.CanRemoveLiquidity(pool.Status) {
		return nil, types.ErrPoolExitDisabled
	}
	if pool.Locked {
		return nil, types.ErrPoolLocked
	}
	lpTokens, err := types.ParsePositiveAmount(msg.LpTokens)
	if err != nil {
		return nil, err
	}
	minA, err := types.ParseOptionalPositiveAmount(msg.MinAmountA)
	if err != nil {
		return nil, fmt.Errorf("invalid min_amount_a: %w", err)
	}
	minB, err := types.ParseOptionalPositiveAmount(msg.MinAmountB)
	if err != nil {
		return nil, fmt.Errorf("invalid min_amount_b: %w", err)
	}
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, fmt.Errorf("invalid sender address: %w", err)
	}
	totalSupply, err := types.ParsePositiveAmount(pool.LpTokenSupply)
	if err != nil {
		return nil, types.ErrInvalidPoolState.Wrapf("lp_token_supply: %v", err)
	}
	if lpTokens.Cmp(totalSupply) > 0 {
		return nil, types.ErrInsufficientLP
	}
	reserveA, err := types.ParsePositiveAmount(pool.ReserveA)
	if err != nil {
		return nil, types.ErrInvalidPoolState.Wrapf("reserve_a: %v", err)
	}
	reserveB, err := types.ParsePositiveAmount(pool.ReserveB)
	if err != nil {
		return nil, types.ErrInvalidPoolState.Wrapf("reserve_b: %v", err)
	}
	amountA, amountB := CalculateWithdrawalAmounts(reserveA, reserveB, lpTokens, totalSupply)
	if amountA.Sign() <= 0 || amountB.Sign() <= 0 {
		return nil, types.ErrZeroAmount.Wrap("withdrawal rounds one asset to zero")
	}
	if (minA != nil && amountA.Cmp(minA) < 0) || (minB != nil && amountB.Cmp(minB) < 0) {
		return nil, types.ErrSlippageExceeded
	}

	m.Keeper.UpdateTWAPAccumulator(ctx, pool)
	pool.Locked = true
	m.Keeper.SetPool(ctx, pool)
	unlock := func() {
		pool.Locked = false
		m.Keeper.SetPool(ctx, pool)
	}
	if m.Keeper.bankKeeper != nil {
		lpCoins := sdk.NewCoins(sdk.NewCoin(pool.LpDenom, sdkmath.NewIntFromBigInt(lpTokens)))
		if err := m.Keeper.bankKeeper.SendCoinsFromAccountToModule(
			ctx, senderAddr, types.ModuleName, lpCoins,
		); err != nil {
			unlock()
			return nil, fmt.Errorf("failed to collect LP tokens: %w", err)
		}
		if err := m.Keeper.bankKeeper.BurnCoins(ctx, types.ModuleName, lpCoins); err != nil {
			unlock()
			return nil, fmt.Errorf("failed to burn LP tokens: %w", err)
		}
		withdrawal := sdk.NewCoins(
			sdk.NewCoin(pool.DenomA, sdkmath.NewIntFromBigInt(amountA)),
			sdk.NewCoin(pool.DenomB, sdkmath.NewIntFromBigInt(amountB)),
		)
		// No SendEnabled gate here: a disabled asset must never trap LPs.
		if err := m.Keeper.bankKeeper.SendCoinsFromModuleToAccount(
			ctx, types.ModuleName, senderAddr, withdrawal,
		); err != nil {
			unlock()
			return nil, fmt.Errorf("failed to return underlying assets: %w", err)
		}
	}

	newReserveA := new(big.Int).Sub(reserveA, amountA)
	newReserveB := new(big.Int).Sub(reserveB, amountB)
	newSupply := new(big.Int).Sub(totalSupply, lpTokens)
	pool.ReserveA = newReserveA.String()
	pool.ReserveB = newReserveB.String()
	pool.LpTokenSupply = newSupply.String()
	pool.Locked = false
	closed := newSupply.Sign() == 0
	if closed {
		if newReserveA.Sign() != 0 || newReserveB.Sign() != 0 {
			return nil, types.ErrInvalidPoolState.Wrap("final exit left non-zero reserves")
		}
		if ctx.BlockHeight() <= 0 {
			return nil, types.ErrInvalidPoolState.Wrap("close height must be positive")
		}
		pool.Status = types.PoolStatus_POOL_STATUS_CLOSED
		pool.ClosedAtBlock = uint64(ctx.BlockHeight())
	}
	m.Keeper.SetPool(ctx, pool)
	if closed {
		m.Keeper.ScheduleTWAPHistoryDeletion(ctx, pool.PoolId)
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.liquiditypool.liquidity_removed",
		sdk.NewAttribute("pool_id", msg.PoolId),
		sdk.NewAttribute("sender", msg.Sender),
		sdk.NewAttribute("lp_tokens_burned", lpTokens.String()),
		sdk.NewAttribute("amount_a", amountA.String()),
		sdk.NewAttribute("amount_b", amountB.String()),
		sdk.NewAttribute("pool_closed", fmt.Sprintf("%t", closed)),
	))
	return &types.MsgRemoveLiquidityResponse{
		AmountA: amountA.String(),
		AmountB: amountB.String(),
	}, nil
}

func (m msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if m.Keeper.GetAuthority() != msg.Authority {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", m.Keeper.GetAuthority(), msg.Authority)
	}
	if msg.Params == nil {
		return nil, fmt.Errorf("params cannot be nil")
	}
	if err := msg.Params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	if m.Keeper.CountOpenPools(ctx) > msg.Params.MaxPools {
		return nil, types.ErrMaxPoolsReached.Wrap("max_pools cannot be lower than current open pool count")
	}
	currentParams := m.Keeper.GetParams(ctx)
	if msg.Params.TwapWindowBlocks < currentParams.TwapWindowBlocks {
		return nil, types.ErrTWAPWindowUnavailable.Wrap(
			"twap_window_blocks cannot decrease in v4; retained keys require a bounded shrink migration",
		)
	}
	m.Keeper.SetParams(ctx, msg.Params)
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.liquiditypool.update_params",
		sdk.NewAttribute("authority", msg.Authority),
	))
	return &types.MsgUpdateParamsResponse{}, nil
}

func (m msgServer) SetPoolStatus(goCtx context.Context, msg *types.MsgSetPoolStatus) (*types.MsgSetPoolStatusResponse, error) {
	if m.Keeper.GetAuthority() != msg.Authority {
		return nil, types.ErrUnauthorized
	}
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	pool, found := m.Keeper.GetPool(ctx, msg.PoolId)
	if !found {
		return nil, types.ErrPoolNotFound
	}
	if pool.Status == types.PoolStatus_POOL_STATUS_CLOSED {
		return nil, types.ErrClosedPoolImmutable
	}
	if pool.Locked {
		return nil, types.ErrPoolLocked
	}
	if _, err := types.ParsePositiveAmount(pool.ReserveA); err != nil {
		return nil, types.ErrInvalidPoolState.Wrapf("reserve_a: %v", err)
	}
	if _, err := types.ParsePositiveAmount(pool.ReserveB); err != nil {
		return nil, types.ErrInvalidPoolState.Wrapf("reserve_b: %v", err)
	}
	if _, err := types.ParsePositiveAmount(pool.LpTokenSupply); err != nil {
		return nil, types.ErrInvalidPoolState.Wrapf("lp_token_supply: %v", err)
	}
	previous := pool.Status
	m.Keeper.UpdateTWAPAccumulator(ctx, pool)
	pool.Status = msg.Status
	m.Keeper.SetPool(ctx, pool)
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.liquiditypool.pool_status_changed",
		sdk.NewAttribute("pool_id", pool.PoolId),
		sdk.NewAttribute("previous_status", previous.String()),
		sdk.NewAttribute("status", pool.Status.String()),
		sdk.NewAttribute("authority", msg.Authority),
	))
	return &types.MsgSetPoolStatusResponse{}, nil
}
