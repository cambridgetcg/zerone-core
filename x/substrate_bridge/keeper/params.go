package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

func (k Keeper) GetParams(ctx context.Context) *types.Params {
	store := sdk.UnwrapSDKContext(ctx).KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		p := types.DefaultParams()
		return &p
	}
	var p types.Params
	k.cdc.MustUnmarshal(bz, &p)
	return &p
}

func (k Keeper) SetParams(ctx context.Context, p *types.Params) error {
	if p == nil {
		return fmt.Errorf("nil params")
	}
	if err := p.Validate(); err != nil {
		return err
	}
	sdk.UnwrapSDKContext(ctx).KVStore(k.storeKey).Set(types.ParamsKey, k.cdc.MustMarshal(p))
	return nil
}
