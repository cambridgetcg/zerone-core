package keeper

import (
	"context"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/liquiditypool/types"
)

type queryServer struct {
	types.UnimplementedQueryServer
	Keeper
}

func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

var _ types.QueryServer = queryServer{}

func (q queryServer) Pool(goCtx context.Context, req *types.QueryPoolRequest) (*types.QueryPoolResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	pool, found := q.Keeper.GetPool(ctx, req.PoolId)
	if !found {
		return nil, types.ErrPoolNotFound
	}
	return &types.QueryPoolResponse{Pool: pool}, nil
}

func (q queryServer) Pools(goCtx context.Context, _ *types.QueryPoolsRequest) (*types.QueryPoolsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	var pools []*types.Pool
	q.Keeper.IteratePools(ctx, func(p *types.Pool) bool {
		pools = append(pools, p)
		return false
	})
	return &types.QueryPoolsResponse{Pools: pools}, nil
}

func (q queryServer) TWAP(goCtx context.Context, req *types.QueryTWAPRequest) (*types.QueryTWAPResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	twap, windowUsed, err := q.Keeper.GetTWAP(ctx, req.PoolId, req.BaseDenom, req.Window)
	if err != nil {
		return nil, err
	}
	return &types.QueryTWAPResponse{
		Twap:       twap.String(),
		WindowUsed: windowUsed,
	}, nil
}

func (q queryServer) SimulateSwap(goCtx context.Context, req *types.QuerySimulateSwapRequest) (*types.QuerySimulateSwapResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	pool, found := q.Keeper.GetPool(ctx, req.PoolId)
	if !found {
		return nil, types.ErrPoolNotFound
	}

	tokenIn := new(big.Int)
	if _, ok := tokenIn.SetString(req.TokenInAmount, 10); !ok || tokenIn.Sign() <= 0 {
		return nil, types.ErrZeroAmount
	}

	reserveA := new(big.Int)
	reserveA.SetString(pool.ReserveA, 10)
	reserveB := new(big.Int)
	reserveB.SetString(pool.ReserveB, 10)

	var reserveIn, reserveOut *big.Int
	var denomOut string

	switch req.TokenInDenom {
	case pool.DenomA:
		reserveIn = reserveA
		reserveOut = reserveB
		denomOut = pool.DenomB
	case pool.DenomB:
		reserveIn = reserveB
		reserveOut = reserveA
		denomOut = pool.DenomA
	default:
		return nil, types.ErrDenomNotInPool
	}

	tokenOut, feeAmount := CalculateSwapOutput(reserveIn, reserveOut, tokenIn, pool.SwapFeeBps)
	priceImpact := CalculatePriceImpactBps(reserveIn, reserveOut, tokenIn, tokenOut)

	return &types.QuerySimulateSwapResponse{
		Result: &types.SwapResult{
			TokenOutDenom:  denomOut,
			TokenOutAmount: tokenOut.String(),
			FeeAmount:      feeAmount.String(),
			PriceImpactBps: priceImpact,
		},
	}, nil
}

func (q queryServer) Params(goCtx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := q.Keeper.GetParams(ctx)
	return &types.QueryParamsResponse{Params: params}, nil
}
