package keeper

import (
	"context"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zerone-chain/zerone/x/sponsorship/types"
)

type queryServer struct {
	types.UnimplementedQueryServer
	Keeper
}

func NewQueryServerImpl(k Keeper) types.QueryServer { return &queryServer{Keeper: k} }

func (q queryServer) Params(ctx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	params, err := q.getParamsChecked(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryParamsResponse{Params: params}, nil
}

func (q queryServer) BountyOrder(ctx context.Context, req *types.QueryBountyOrderRequest) (*types.QueryBountyOrderResponse, error) {
	if req == nil || req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "bounty id is required")
	}
	o, ok, err := q.getBountyOrderChecked(ctx, req.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !ok {
		return nil, status.Error(codes.NotFound, types.ErrBountyNotFound.Error())
	}
	return &types.QueryBountyOrderResponse{Order: o}, nil
}

func (q queryServer) BountyOrders(ctx context.Context, _ *types.QueryBountyOrdersRequest) (*types.QueryBountyOrdersResponse, error) {
	orders, err := q.getAllBountyOrdersChecked(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryBountyOrdersResponse{Orders: orders}, nil
}

func (q queryServer) Fulfillment(
	ctx context.Context,
	req *types.QueryFulfillmentRequest,
) (*types.QueryFulfillmentResponse, error) {
	if req == nil || req.BountyId == "" || req.FactId == "" {
		return nil, status.Error(codes.InvalidArgument, "bounty id and fact id are required")
	}
	fulfillment, found, err := q.getFulfillmentChecked(ctx, req.BountyId, req.FactId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !found {
		return nil, status.Error(codes.NotFound, "fulfillment not found")
	}
	factOwner, _, err := q.consumptionOwnerChecked(
		ctx,
		types.FactConsumptionKey(fulfillment.FactId),
		"fact",
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	receiptOwner := ""
	if fulfillment.WorkReceiptHash != "" {
		receiptOwner, _, err = q.consumptionOwnerChecked(
			ctx,
			types.ReceiptConsumptionKey(fulfillment.WorkReceiptHash),
			"receipt",
		)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}
	nullifierOwner := ""
	if fulfillment.SettlementNullifier != "" {
		nullifierOwner, _, err = q.consumptionOwnerChecked(
			ctx,
			types.SettlementNullifierKey(fulfillment.SettlementNullifier),
			"settlement nullifier",
		)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}
	height, err := nonNegativeQueryHeight(ctx)
	if err != nil {
		return nil, err
	}
	return &types.QueryFulfillmentResponse{
		Fulfillment:                           fulfillment,
		FactConsumedByBountyId:                factOwner,
		ReceiptConsumedByBountyId:             receiptOwner,
		SettlementNullifierConsumedByBountyId: nullifierOwner,
		SnapshotBlockHeight:                   height,
	}, nil
}

func (q queryServer) EscrowAccounting(
	ctx context.Context,
	_ *types.QueryEscrowAccountingRequest,
) (*types.QueryEscrowAccountingResponse, error) {
	liability, err := q.TotalEscrowLiability(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	moduleAddr := sdk.AccAddress(authtypes.NewModuleAddress(types.ModuleName))
	balance := q.bankKeeper.GetBalance(ctx, moduleAddr, "uzrn").Amount.BigInt()
	surplus := new(big.Int).Sub(new(big.Int).Set(balance), liability)
	height, err := nonNegativeQueryHeight(ctx)
	if err != nil {
		return nil, err
	}
	return &types.QueryEscrowAccountingResponse{
		Denom:               "uzrn",
		PersistedLiability:  liability.String(),
		ModuleBalance:       balance.String(),
		Surplus:             surplus.String(),
		Solvent:             surplus.Sign() >= 0,
		SnapshotBlockHeight: height,
	}, nil
}

func nonNegativeQueryHeight(ctx context.Context) (uint64, error) {
	height := sdk.UnwrapSDKContext(ctx).BlockHeight()
	if height < 0 {
		return 0, status.Error(codes.Internal, "negative consensus block height")
	}
	return uint64(height), nil
}
