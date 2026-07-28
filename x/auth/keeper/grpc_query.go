package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zerone-chain/zerone/x/auth/types"
)

var _ types.QueryServer = queryServer{}

type queryServer struct {
	Keeper
	types.UnimplementedQueryServer
}

// NewQueryServerImpl returns an implementation of the QueryServer interface.
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

// Account returns a Zerone account by bech32 address.
func (qs queryServer) Account(goCtx context.Context, req *types.QueryAccountRequest) (*types.QueryAccountResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	account, found := qs.GetAccount(ctx, req.Address)
	if !found {
		return nil, types.ErrAccountNotFound
	}

	return &types.QueryAccountResponse{Account: account}, nil
}

// AccountByDID returns a Zerone account by DID.
func (qs queryServer) AccountByDID(goCtx context.Context, req *types.QueryAccountByDIDRequest) (*types.QueryAccountByDIDResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	account, found := qs.GetAccountByDID(ctx, req.Did)
	if !found {
		return nil, types.ErrAccountNotFound
	}

	return &types.QueryAccountByDIDResponse{Account: account}, nil
}

// Params returns the module parameters.
func (qs queryServer) Params(goCtx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	params := qs.GetParams(ctx)

	return &types.QueryParamsResponse{Params: params}, nil
}

// FrozenAccounts returns all frozen accounts.
func (qs queryServer) FrozenAccounts(goCtx context.Context, _ *types.QueryFrozenAccountsRequest) (*types.QueryFrozenAccountsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	var frozen []*types.Account
	qs.IterateAccounts(ctx, func(account *types.Account) bool {
		if account.Flags != nil && account.Flags.Frozen {
			frozen = append(frozen, account)
		}
		return false
	})

	if frozen == nil {
		frozen = []*types.Account{}
	}

	return &types.QueryFrozenAccountsResponse{Accounts: frozen}, nil
}

// AccountIdentifier projects a registered Zerone account into CAIP-2 and
// CAIP-10 identifiers. It performs no writes and emits no consensus events.
func (qs queryServer) AccountIdentifier(goCtx context.Context, req *types.QueryAccountIdentifierRequest) (*types.QueryAccountIdentifierResponse, error) {
	if req == nil || req.Address == "" {
		return nil, status.Error(codes.InvalidArgument, "account address is required")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	rawChainID := ctx.ChainID()
	reference, err := types.CosmosChainReference(rawChainID)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "cannot derive CAIP-2 chain reference: %v", err)
	}

	account, found := qs.GetAccount(ctx, req.Address)
	if !found {
		// Preserve a useful client-input distinction for absent keys: malformed
		// or non-canonical addresses are InvalidArgument, while a canonical
		// address with no record is NotFound.
		if _, err := types.CAIP10AccountID(rawChainID, req.Address); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid account identifier: %v", err)
		}
		return nil, status.Error(codes.NotFound, types.ErrAccountNotFound.Error())
	}
	if account.Address != req.Address {
		return nil, status.Error(codes.DataLoss, "stored account address does not match its lookup key")
	}

	mapping, found := qs.GetDIDMapping(ctx, account.Did)
	if !found ||
		mapping.Did != account.Did ||
		mapping.Bech32 != account.Address ||
		mapping.PubKey != account.PublicKey {
		return nil, status.Error(codes.DataLoss, "stored account and DID mapping are inconsistent")
	}

	accountID, err := types.CAIP10AccountID(rawChainID, account.Address)
	if err != nil {
		// Registration and historical genesis validation admit a wider address
		// set than the draft Cosmos CAIP-10 profile. A stored-but-unprojectable
		// record is server state, not malformed client input.
		return nil, status.Errorf(codes.FailedPrecondition, "registered account address is not CAIP-projectable: %v", err)
	}

	frozen := account.Flags != nil && account.Flags.Frozen
	return &types.QueryAccountIdentifierResponse{
		Identifier: &types.ChainAccountIdentifier{
			Namespace:      types.CosmosCAIPNamespace,
			Reference:      reference,
			RawChainId:     rawChainID,
			AccountId:      accountID,
			Address:        account.Address,
			Did:            account.Did,
			AccountType:    account.AccountType,
			Frozen:         frozen,
			CreatedAtBlock: account.CreatedAtBlock,
		},
	}, nil
}
