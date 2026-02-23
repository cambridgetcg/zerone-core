package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/compute_pool/types"
)

// Keeper manages the compute_pool module's state.
type Keeper struct {
	storeService store.KVStoreService
	cdc          codec.BinaryCodec
	authority    string

	bankKeeper types.BankKeeper
}

// NewKeeper creates a new compute_pool module Keeper.
func NewKeeper(
	storeService store.KVStoreService,
	cdc codec.BinaryCodec,
	authority string,
	bk types.BankKeeper,
) Keeper {
	return Keeper{
		storeService: storeService,
		cdc:          cdc,
		authority:    authority,
		bankKeeper:   bk,
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

// prefixEndBytes returns the end key for prefix iteration (exclusive).
func prefixEndBytes(prefix []byte) []byte {
	if len(prefix) == 0 {
		return nil
	}
	end := make([]byte, len(prefix))
	copy(end, prefix)
	for i := len(end) - 1; i >= 0; i-- {
		end[i]++
		if end[i] != 0 {
			return end
		}
	}
	return nil
}

// ---------- Params ----------

// SetParams sets module parameters.
func (k Keeper) SetParams(ctx context.Context, params *types.Params) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(params)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal params: %v", err))
	}
	_ = kvStore.Set(types.ParamsKey, bz)
}

// GetParams returns module parameters.
func (k Keeper) GetParams(ctx context.Context) *types.Params {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.ParamsKey)
	if err != nil || bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := proto.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
	return &params
}

// ---------- Provider Operations ----------

// providerKey builds the KV key for a provider by address.
func providerKey(address string) []byte {
	return append(types.ProviderKeyPrefix, []byte(address)...)
}

// SetProvider stores a compute provider in the KV store.
func (k Keeper) SetProvider(ctx context.Context, provider *types.ComputeProvider) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := providerKey(provider.Address)
	bz, err := proto.Marshal(provider)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal provider: %v", err))
	}
	_ = kvStore.Set(key, bz)
}

// GetProvider retrieves a compute provider by address.
func (k Keeper) GetProvider(ctx context.Context, address string) (*types.ComputeProvider, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := providerKey(address)
	bz, err := kvStore.Get(key)
	if err != nil || bz == nil {
		return nil, false
	}
	var provider types.ComputeProvider
	if err := proto.Unmarshal(bz, &provider); err != nil {
		return nil, false
	}
	return &provider, true
}

// DeleteProvider removes a provider from the store.
func (k Keeper) DeleteProvider(ctx context.Context, address string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(providerKey(address))
}

// GetAllProviders returns all compute providers.
func (k Keeper) GetAllProviders(ctx context.Context) []*types.ComputeProvider {
	var providers []*types.ComputeProvider
	k.IterateProviders(ctx, func(p *types.ComputeProvider) bool {
		providers = append(providers, p)
		return false
	})
	return providers
}

// IterateProviders iterates over all providers. Return true from cb to stop.
func (k Keeper) IterateProviders(ctx context.Context, cb func(*types.ComputeProvider) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.ProviderKeyPrefix, prefixEndBytes(types.ProviderKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var provider types.ComputeProvider
		if err := proto.Unmarshal(iter.Value(), &provider); err != nil {
			continue
		}
		if cb(&provider) {
			break
		}
	}
}

// ---------- Credit Operations ----------

// creditKey builds the KV key for a credit by validator address.
func creditKey(validatorAddr string) []byte {
	return append(types.CreditKeyPrefix, []byte(validatorAddr)...)
}

// SetCredit stores a compute credit in the KV store.
func (k Keeper) SetCredit(ctx context.Context, credit *types.ComputeCredit) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := creditKey(credit.ValidatorAddr)
	bz, err := proto.Marshal(credit)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal credit: %v", err))
	}
	_ = kvStore.Set(key, bz)
}

// GetCredit retrieves a compute credit by validator address.
func (k Keeper) GetCredit(ctx context.Context, validatorAddr string) (*types.ComputeCredit, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := creditKey(validatorAddr)
	bz, err := kvStore.Get(key)
	if err != nil || bz == nil {
		return nil, false
	}
	var credit types.ComputeCredit
	if err := proto.Unmarshal(bz, &credit); err != nil {
		return nil, false
	}
	return &credit, true
}

// GetAllCredits returns all compute credits.
func (k Keeper) GetAllCredits(ctx context.Context) []*types.ComputeCredit {
	var credits []*types.ComputeCredit
	k.IterateCredits(ctx, func(c *types.ComputeCredit) bool {
		credits = append(credits, c)
		return false
	})
	return credits
}

// IterateCredits iterates over all credits. Return true from cb to stop.
func (k Keeper) IterateCredits(ctx context.Context, cb func(*types.ComputeCredit) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.CreditKeyPrefix, prefixEndBytes(types.CreditKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var credit types.ComputeCredit
		if err := proto.Unmarshal(iter.Value(), &credit); err != nil {
			continue
		}
		if cb(&credit) {
			break
		}
	}
}

// ---------- Genesis ----------

// InitGenesis initializes the module's state from genesis.
func (k Keeper) InitGenesis(ctx context.Context, genState *types.GenesisState) {
	if genState.Params != nil {
		k.SetParams(ctx, genState.Params)
	}
	for _, provider := range genState.Providers {
		k.SetProvider(ctx, provider)
	}
	for _, credit := range genState.Credits {
		k.SetCredit(ctx, credit)
	}
}

// ExportGenesis exports the module's state.
func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	params := k.GetParams(ctx)
	providers := k.GetAllProviders(ctx)
	credits := k.GetAllCredits(ctx)
	return &types.GenesisState{
		Params:    params,
		Providers: providers,
		Credits:   credits,
	}
}
