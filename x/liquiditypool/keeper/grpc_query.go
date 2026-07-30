package keeper

import (
	"bytes"
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkquery "github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/protobuf/proto"

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
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	pool, found := q.Keeper.GetPool(ctx, req.PoolId)
	if !found {
		return nil, types.ErrPoolNotFound
	}
	return &types.QueryPoolResponse{Pool: pool}, nil
}

func (q queryServer) Pools(goCtx context.Context, req *types.QueryPoolsRequest) (*types.QueryPoolsResponse, error) {
	if req == nil {
		req = &types.QueryPoolsRequest{}
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	pools, page, err := q.paginatePools(ctx, req.Pagination)
	if err != nil {
		return nil, err
	}
	return &types.QueryPoolsResponse{Pools: pools, Pagination: page}, nil
}

func (q queryServer) paginatePools(
	ctx sdk.Context,
	request *sdkquery.PageRequest,
) ([]*types.Pool, *sdkquery.PageResponse, error) {
	page := &sdkquery.PageRequest{}
	if request != nil {
		*page = *request
	}
	if page.Offset > 0 && len(page.Key) > 0 {
		return nil, nil, fmt.Errorf("pagination offset and key cannot both be set")
	}
	if page.Limit == 0 {
		page.Limit = sdkquery.DefaultLimit
	}
	if page.Limit > sdkquery.DefaultLimit {
		page.Limit = sdkquery.DefaultLimit
	}
	if len(page.Key) > 0 && !bytes.HasPrefix(page.Key, types.PoolKeyPrefix) {
		return nil, nil, fmt.Errorf("invalid pool pagination key")
	}

	store := q.storeService.OpenKVStore(ctx)
	start, end := types.PoolKeyPrefix, prefixEndBytes(types.PoolKeyPrefix)
	var (
		iter interface {
			Valid() bool
			Next()
			Key() []byte
			Value() []byte
			Close() error
		}
		err error
	)
	if page.Reverse {
		reverseEnd := end
		if len(page.Key) > 0 {
			reverseEnd = append(append([]byte(nil), page.Key...), 0)
		}
		iter, err = store.ReverseIterator(start, reverseEnd)
	} else {
		if len(page.Key) > 0 {
			start = page.Key
		}
		iter, err = store.Iterator(start, end)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to iterate pools: %w", err)
	}
	defer iter.Close()

	var (
		pools   []*types.Pool
		seen    uint64
		total   uint64
		nextKey []byte
	)
	for ; iter.Valid(); iter.Next() {
		if len(page.Key) == 0 && seen < page.Offset {
			seen++
			total++
			continue
		}
		if uint64(len(pools)) < page.Limit {
			var pool types.Pool
			if err := proto.Unmarshal(iter.Value(), &pool); err != nil {
				return nil, nil, fmt.Errorf("failed to decode pool: %w", err)
			}
			pools = append(pools, &pool)
			seen++
			total++
			continue
		}
		if nextKey == nil {
			nextKey = append([]byte(nil), iter.Key()...)
		}
		total++
		if !page.CountTotal {
			break
		}
	}
	if page.CountTotal && len(page.Key) == 0 {
		// total already includes skipped offset entries and every remaining
		// iterator entry.
	} else {
		total = 0
	}
	return pools, &sdkquery.PageResponse{NextKey: nextKey, Total: total}, nil
}

func (q queryServer) TWAP(goCtx context.Context, req *types.QueryTWAPRequest) (*types.QueryTWAPResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	twap, windowUsed, err := q.Keeper.GetTWAP(ctx, req.PoolId, req.BaseDenom, req.Window)
	if err != nil {
		return nil, err
	}
	return &types.QueryTWAPResponse{Twap: twap.String(), WindowUsed: windowUsed}, nil
}

func (q queryServer) SimulateSwap(
	goCtx context.Context,
	req *types.QuerySimulateSwapRequest,
) (*types.QuerySimulateSwapResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	result, err := q.Keeper.CheckedSwapQuote(
		ctx, req.PoolId, req.TokenInDenom, req.TokenInAmount,
	)
	if err != nil {
		return nil, err
	}
	return &types.QuerySimulateSwapResponse{Result: result}, nil
}

func (q queryServer) Params(goCtx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	return &types.QueryParamsResponse{Params: q.Keeper.GetParams(ctx)}, nil
}
