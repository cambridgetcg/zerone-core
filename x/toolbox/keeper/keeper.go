package keeper

import (
	"context"
	"encoding/json"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/toolbox/types"
)

// Keeper manages the toolbox module's state.
type Keeper struct {
	storeService store.KVStoreService
	cdc          codec.BinaryCodec
	authority    string

	// Required keepers (set in constructor).
	bankKeeper   types.BankKeeper
	researchFund types.ResearchFundDepositor

	// Optional keepers (set post-init via setters).
	discoveryKeeper types.DiscoveryKeeper
	bvmKeeper       types.BvmKeeper
	knowledgeKeeper types.KnowledgeKeeper
	stakingKeeper   types.StakingKeeper
	billingKeeper   types.BillingKeeper
	homeKeeper      types.HomeKeeper
}

// NewKeeper creates a new toolbox module Keeper.
func NewKeeper(
	storeService store.KVStoreService,
	cdc codec.BinaryCodec,
	authority string,
	bk types.BankKeeper,
	rfd types.ResearchFundDepositor,
) Keeper {
	if rfd == nil {
		panic("toolbox: ResearchFundDepositor is required")
	}
	return Keeper{
		storeService: storeService,
		cdc:          cdc,
		authority:    authority,
		bankKeeper:   bk,
		researchFund: rfd,
	}
}

// Logger returns a module-scoped logger.
func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", "x/"+types.ModuleName)
}

// GetAuthority returns the module authority address.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// ---------- Optional keeper setters ----------

func (k *Keeper) SetDiscoveryKeeper(dk types.DiscoveryKeeper) { k.discoveryKeeper = dk }
func (k *Keeper) SetBvmKeeper(bk types.BvmKeeper)            { k.bvmKeeper = bk }
func (k *Keeper) SetKnowledgeKeeper(kk types.KnowledgeKeeper) { k.knowledgeKeeper = kk }
func (k *Keeper) SetStakingKeeper(sk types.StakingKeeper)     { k.stakingKeeper = sk }
func (k *Keeper) SetBillingKeeper(bk types.BillingKeeper)     { k.billingKeeper = bk }
func (k *Keeper) SetHomeKeeper(hk types.HomeKeeper)           { k.homeKeeper = hk }

// ---------- Genesis ----------

// InitGenesis initializes the module's state from genesis.
func (k Keeper) InitGenesis(ctx context.Context, genState *types.GenesisState) {
	if genState.Params != nil {
		k.SetParams(ctx, genState.Params)
	}
	for _, tool := range genState.Tools {
		k.SetTool(ctx, tool)
	}
}

// ExportGenesis exports the module's state.
func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	params := k.GetParams(ctx)
	tools := k.GetAllTools(ctx)
	return &types.GenesisState{
		Params: params,
		Tools:  tools,
	}
}

// ExportGenesisJSON exports the module's state as JSON.
func (k Keeper) ExportGenesisJSON(ctx context.Context) json.RawMessage {
	genState := k.ExportGenesis(ctx)
	bz, err := json.Marshal(genState)
	if err != nil {
		panic("failed to marshal toolbox genesis: " + err.Error())
	}
	return bz
}
