package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/liquiditypool/types"
)

const (
	h1UpgradeName        = "consolidation-safety-v1"
	h1MigrationMarker    = "upgrade_marker_consolidation-safety-v1"
	nativeLineageMarker  = "chain_lineage_native_consolidation-safety-v1"
	nativeLineageGenesis = "genesis"
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

// requireLiquidityV5Activated proves exactly one accepted chain lineage: the
// global named H1 upgrade completed at or before this height, or this chain was
// born natively from the v5 source. It then checks the retired-fee sentinel as
// defense in depth. The local zero is never sole activation evidence: zero was
// legal before H1 and historical MsgUpdateParams could write it directly.
func (k Keeper) requireLiquidityV5Activated(ctx sdk.Context) error {
	h1Marker, h1Found, nativeMarker, nativeFound, doneHeight, err := k.readActivationEvidence(ctx)
	if err != nil {
		return fmt.Errorf("liquiditypool consensus v5 activation evidence unreadable: %w", err)
	}
	currentHeight := ctx.BlockHeight()
	migratedLineage := h1Found && h1Marker == "migrated" && !nativeFound &&
		doneHeight > 0 && currentHeight > 0 && doneHeight <= currentHeight
	nativeLineage := !h1Found && nativeFound && nativeMarker == nativeLineageGenesis &&
		doneHeight == 0 && currentHeight > 0
	if !migratedLineage && !nativeLineage {
		return fmt.Errorf(
			"liquiditypool consensus v5 is not activated: require exactly one valid migrated or native lineage; H1=%q (present=%t), native=%q (present=%t), done=%d, current=%d",
			h1Marker,
			h1Found,
			nativeMarker,
			nativeFound,
			doneHeight,
			currentHeight,
		)
	}
	params, err := k.getStoredParamsChecked(ctx)
	if err != nil {
		return fmt.Errorf(
			"liquiditypool consensus v5 is not activated: params proof invalid: %w",
			err,
		)
	}
	if protocolFeeBps := params.ProtocolFeeBps; protocolFeeBps != 0 {
		return fmt.Errorf(
			"liquiditypool consensus v5 is not activated: protocol_fee_bps=%d; require zero after accepted lineage proof",
			protocolFeeBps,
		)
	}
	return nil
}

// readActivationEvidence turns a missing dependency and even a malformed
// cross-module store panic into an ordinary fail-closed error. It performs no
// writes and deliberately reads both facts from the same SDK context.
func (k Keeper) readActivationEvidence(ctx context.Context) (
	h1Marker string,
	h1Found bool,
	nativeMarker string,
	nativeFound bool,
	doneHeight int64,
	err error,
) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("panic while reading H1 evidence: %v", recovered)
		}
	}()
	if k.activationEvidence == nil {
		return "", false, "", false, 0, fmt.Errorf("activation evidence reader is not configured")
	}
	h1Marker, h1Found, err = k.activationEvidence.ReadMigrationMarkerPresenceChecked(
		ctx,
		h1MigrationMarker,
	)
	if err != nil {
		return "", false, "", false, 0, fmt.Errorf("read H1 migration marker: %w", err)
	}
	nativeMarker, nativeFound, err = k.activationEvidence.ReadMigrationMarkerPresenceChecked(
		ctx,
		nativeLineageMarker,
	)
	if err != nil {
		return "", false, "", false, 0, fmt.Errorf("read native lineage marker: %w", err)
	}
	doneHeight, err = k.activationEvidence.GetDoneHeight(ctx, h1UpgradeName)
	if err != nil {
		return "", false, "", false, 0, fmt.Errorf("read H1 done height: %w", err)
	}
	return h1Marker, h1Found, nativeMarker, nativeFound, doneHeight, nil
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
	// Keep the proof inside the private primitive so future same-package
	// callers cannot accidentally bypass the activation wall.
	if err := k.requireLiquidityV5Activated(ctx); err != nil {
		return nil, err
	}
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
