package keeper

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/billing/types"
)

// ---------- Provider Operations ----------

// SetProvider stores a provider in the KV store.
func (k Keeper) SetProvider(ctx context.Context, provider *types.Provider) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := providerKey(provider.Address)
	bz, err := proto.Marshal(provider)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal provider: %v", err))
	}
	_ = kvStore.Set(key, bz)
}

// GetProvider retrieves a provider by address.
func (k Keeper) GetProvider(ctx context.Context, address string) (*types.Provider, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := providerKey(address)
	bz, err := kvStore.Get(key)
	if err != nil || bz == nil {
		return nil, false
	}
	var provider types.Provider
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

// GetAllProviders returns all providers.
func (k Keeper) GetAllProviders(ctx context.Context) []*types.Provider {
	var providers []*types.Provider
	k.IterateProviders(ctx, func(p *types.Provider) bool {
		providers = append(providers, p)
		return false
	})
	return providers
}

// IterateProviders iterates over all providers. Return true from cb to stop.
func (k Keeper) IterateProviders(ctx context.Context, cb func(*types.Provider) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.ProviderKeyPrefix, prefixEndBytes(types.ProviderKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var provider types.Provider
		if err := proto.Unmarshal(iter.Value(), &provider); err != nil {
			continue
		}
		if cb(&provider) {
			break
		}
	}
}

// ---------- Domain Index Operations ----------

// SetDomainIndex adds a provider to the domain index.
func (k Keeper) SetDomainIndex(ctx context.Context, domain string, address string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := domainIndexKey(domain, address)
	_ = kvStore.Set(key, []byte(address))
}

// DeleteDomainIndex removes a provider from the domain index.
func (k Keeper) DeleteDomainIndex(ctx context.Context, domain string, address string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(domainIndexKey(domain, address))
}

// GetProvidersByDomain returns provider addresses for a given domain.
func (k Keeper) GetProvidersByDomain(ctx context.Context, domain string) []string {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := domainPrefix(domain)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var addresses []string
	for ; iter.Valid(); iter.Next() {
		addresses = append(addresses, string(iter.Value()))
	}
	return addresses
}

// ---------- Key Construction Helpers ----------

func providerKey(address string) []byte {
	return append(types.ProviderKeyPrefix, []byte(address)...)
}

func domainPrefix(domain string) []byte {
	return append(types.DomainIndexPrefix, []byte(domain+"/")...)
}

func domainIndexKey(domain string, address string) []byte {
	return append(types.DomainIndexPrefix, []byte(domain+"/"+address)...)
}
