package keeper

import (
	"context"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/dialectic/types"
)

// Keeper is a pure read-only synthesizer. State is just Params.
type Keeper struct {
	cdc          codec.BinaryCodec
	storeService store.KVStoreService
	authority    string

	knowledge types.KnowledgeKeeper
}

func NewKeeper(storeService store.KVStoreService, cdc codec.BinaryCodec, authority string) Keeper {
	return Keeper{
		cdc:          cdc,
		storeService: storeService,
		authority:    authority,
	}
}

func (k *Keeper) SetKnowledgeKeeper(kk types.KnowledgeKeeper) { k.knowledge = kk }

func (k Keeper) Logger(ctx context.Context) log.Logger {
	return sdk.UnwrapSDKContext(ctx).Logger().With("module", "x/"+types.ModuleName)
}

func (k Keeper) Authority() string { return k.authority }

// ─── Params ──────────────────────────────────────────────────────────

func (k Keeper) GetParams(ctx context.Context) types.Params {
	bz, err := k.storeService.OpenKVStore(ctx).Get(types.ParamsKey)
	if err != nil || bz == nil {
		return *types.DefaultParams()
	}
	var p types.Params
	if err := k.cdc.Unmarshal(bz, &p); err != nil {
		return *types.DefaultParams()
	}
	return p
}

func (k Keeper) SetParams(ctx context.Context, p types.Params) error {
	bz, err := k.cdc.Marshal(&p)
	if err != nil {
		return err
	}
	return k.storeService.OpenKVStore(ctx).Set(types.ParamsKey, bz)
}

func (k Keeper) InitGenesis(ctx context.Context, gs *types.GenesisState) {
	if gs == nil || gs.Params == nil {
		return
	}
	_ = k.SetParams(ctx, *gs.Params)
}

func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	p := k.GetParams(ctx)
	return &types.GenesisState{Params: &p}
}
