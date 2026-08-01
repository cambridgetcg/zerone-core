package keeper

import (
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/liquiditypool/types"
)

type checkedSwapQuote struct {
	pool       *types.Pool
	reserveIn  *big.Int
	reserveOut *big.Int
	tokenIn    *big.Int
	tokenOut   *big.Int
	feeAmount  *big.Int
	denomOut   string
}

// requireLiquidityV5Activated makes the retired protocol-fee field an
// activation sentinel shared by executable swaps and their public quote path.
// A v5 binary started against pre-H1 state must neither execute nor advertise
// the post-H1 economics before the named migration writes zero.
func (k Keeper) requireLiquidityV5Activated(ctx sdk.Context) error {
	if protocolFeeBps := k.GetParams(ctx).ProtocolFeeBps; protocolFeeBps != 0 {
		return fmt.Errorf(
			"liquiditypool consensus v5 is not activated: protocol_fee_bps=%d; require migrated zero",
			protocolFeeBps,
		)
	}
	return nil
}

// CheckedSwapQuote is the public quote surface shared by gRPC simulation and
// execution. A successful result has passed the same pool-state and bank-denom
// gates against the same unchanged state: lifecycle, lock, integer bounds,
// zero output, minimum reserve, and send-enabled checks are all evaluated
// here. Final delivery still checks the sender's balance and can race later
// state changes.
func (k Keeper) CheckedSwapQuote(
	ctx sdk.Context,
	poolID, tokenInDenom, tokenInAmount string,
) (*types.SwapResult, error) {
	if err := k.requireLiquidityV5Activated(ctx); err != nil {
		return nil, err
	}
	quote, err := k.checkedSwapQuote(ctx, poolID, tokenInDenom, tokenInAmount)
	if err != nil {
		return nil, err
	}
	return &types.SwapResult{
		TokenOutDenom:  quote.denomOut,
		TokenOutAmount: quote.tokenOut.String(),
		FeeAmount:      quote.feeAmount.String(),
		PriceImpactBps: CalculatePriceImpactBpsWithFee(
			quote.reserveIn,
			quote.reserveOut,
			quote.tokenIn,
			quote.tokenOut,
			quote.pool.SwapFeeBps,
		),
	}, nil
}

func (k Keeper) checkedSwapQuote(
	ctx sdk.Context,
	poolID, tokenInDenom, tokenInAmount string,
) (*checkedSwapQuote, error) {
	pool, found := k.GetPool(ctx, poolID)
	if !found {
		return nil, types.ErrPoolNotFound
	}
	if !types.CanSwap(pool.Status) {
		return nil, types.ErrPoolNotActive
	}
	if pool.Locked {
		return nil, types.ErrPoolLocked
	}
	if err := types.ValidatePoolDenom(tokenInDenom); err != nil {
		return nil, err
	}
	tokenIn, err := types.ParsePositiveAmount(tokenInAmount)
	if err != nil {
		return nil, err
	}
	reserveA, err := types.ParsePositiveAmount(pool.ReserveA)
	if err != nil {
		return nil, types.ErrInvalidPoolState.Wrapf("reserve_a: %v", err)
	}
	reserveB, err := types.ParsePositiveAmount(pool.ReserveB)
	if err != nil {
		return nil, types.ErrInvalidPoolState.Wrapf("reserve_b: %v", err)
	}

	var reserveIn, reserveOut *big.Int
	var denomOut string
	switch tokenInDenom {
	case pool.DenomA:
		reserveIn, reserveOut, denomOut = reserveA, reserveB, pool.DenomB
	case pool.DenomB:
		reserveIn, reserveOut, denomOut = reserveB, reserveA, pool.DenomA
	default:
		return nil, types.ErrDenomNotInPool
	}

	tokenOut, feeAmount := CalculateSwapOutput(reserveIn, reserveOut, tokenIn, pool.SwapFeeBps)
	if tokenOut.Sign() <= 0 {
		return nil, types.ErrZeroSwapOutput
	}
	minReserve, err := types.ParseNonNegativeAmount(k.GetParams(ctx).MinReserve)
	if err != nil {
		return nil, types.ErrInvalidPoolState.Wrapf("min_reserve: %v", err)
	}
	if new(big.Int).Sub(reserveOut, tokenOut).Cmp(minReserve) < 0 {
		return nil, types.ErrReserveBelowMinimum
	}
	if k.bankKeeper != nil {
		sendCoins := []sdk.Coin{
			sdk.NewCoin(tokenInDenom, sdkmath.NewIntFromBigInt(tokenIn)),
			sdk.NewCoin(denomOut, sdkmath.NewIntFromBigInt(tokenOut)),
		}
		if err := k.bankKeeper.IsSendEnabledCoins(ctx, sendCoins...); err != nil {
			return nil, fmt.Errorf("swap contains a send-disabled denom: %w", err)
		}
	}
	return &checkedSwapQuote{
		pool:       pool,
		reserveIn:  reserveIn,
		reserveOut: reserveOut,
		tokenIn:    tokenIn,
		tokenOut:   tokenOut,
		feeAmount:  feeAmount,
		denomOut:   denomOut,
	}, nil
}
