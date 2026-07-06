package keeper

import (
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/auth/types"
)

// Keeper manages Zerone account state with 4-layer key architecture.
type Keeper struct {
	cdc           codec.Codec
	storeService  store.KVStoreService
	accountKeeper types.CosmosAccountKeeper
	authority     string
}

// NewKeeper creates a new Keeper instance.
func NewKeeper(
	cdc codec.Codec,
	storeService store.KVStoreService,
	accountKeeper types.CosmosAccountKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:           cdc,
		storeService:  storeService,
		accountKeeper: accountKeeper,
		authority:     authority,
	}
}

// prefixEndBytes returns the end key for a prefix scan (exclusive upper bound).
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

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}

// SetAccount stores a Zerone account.
func (k Keeper) SetAccount(ctx sdk.Context, account *types.Account) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(account)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal account: %v", err))
	}
	if err := kvStore.Set(types.AccountKey(account.Address), bz); err != nil {
		panic(fmt.Sprintf("failed to store account: %v", err))
	}
}

// GetAccount retrieves a Zerone account by bech32 address.
func (k Keeper) GetAccount(ctx sdk.Context, address string) (*types.Account, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.AccountKey(address))
	if err != nil || bz == nil {
		return nil, false
	}
	var account types.Account
	if err := proto.Unmarshal(bz, &account); err != nil {
		return nil, false
	}
	return &account, true
}

// GetAccountByDID retrieves a Zerone account by DID.
func (k Keeper) GetAccountByDID(ctx sdk.Context, did string) (*types.Account, bool) {
	address, found := k.GetAddressForDID(ctx, did)
	if !found {
		return nil, false
	}
	return k.GetAccount(ctx, address)
}

// SetDIDMapping stores a DID -> bech32 mapping.
func (k Keeper) SetDIDMapping(ctx sdk.Context, mapping *types.DIDMapping) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(mapping)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal DID mapping: %v", err))
	}
	if err := kvStore.Set(types.DIDMappingKey(mapping.Did), bz); err != nil {
		panic(fmt.Sprintf("failed to store DID mapping: %v", err))
	}
}

// GetDIDMapping retrieves a DID mapping.
func (k Keeper) GetDIDMapping(ctx sdk.Context, did string) (*types.DIDMapping, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.DIDMappingKey(did))
	if err != nil || bz == nil {
		return nil, false
	}
	var mapping types.DIDMapping
	if err := proto.Unmarshal(bz, &mapping); err != nil {
		return nil, false
	}
	return &mapping, true
}

// GetAddressForDID returns the bech32 address for a DID.
func (k Keeper) GetAddressForDID(ctx sdk.Context, did string) (string, bool) {
	mapping, found := k.GetDIDMapping(ctx, did)
	if !found {
		return "", false
	}
	return mapping.Bech32, true
}

// SetParams sets module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params *types.Params) error {
	if err := params.Validate(); err != nil {
		return err
	}
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal params: %w", err)
	}
	if err := kvStore.Set(types.ParamsKey, bz); err != nil {
		return fmt.Errorf("failed to store params: %w", err)
	}
	return nil
}

// GetParams retrieves module parameters.
func (k Keeper) GetParams(ctx sdk.Context) *types.Params {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.ParamsKey)
	if err != nil || bz == nil {
		p := types.DefaultParams()
		return &p
	}
	var params types.Params
	if err := proto.Unmarshal(bz, &params); err != nil {
		p := types.DefaultParams()
		return &p
	}
	return &params
}

// SetLastRotation stores the block height of last key rotation.
func (k Keeper) SetLastRotation(ctx sdk.Context, address string, height uint64) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Set(types.LastRotationKey(address), types.Uint64ToBytes(height))
}

// GetLastRotation retrieves the block height of last key rotation.
func (k Keeper) GetLastRotation(ctx sdk.Context, address string) uint64 {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.LastRotationKey(address))
	if err != nil || bz == nil {
		return 0
	}
	return types.BytesToUint64(bz)
}

// GetAuthority returns the module authority address.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// IterateAccounts iterates over all accounts.
func (k Keeper) IterateAccounts(ctx sdk.Context, cb func(*types.Account) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.AccountKeyPrefix, prefixEndBytes(types.AccountKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var account types.Account
		if err := proto.Unmarshal(iter.Value(), &account); err != nil {
			continue
		}
		if cb(&account) {
			break
		}
	}
}

// InitGenesis initializes the module state from genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, data *types.GenesisState) error {
	if data.Params != nil {
		if err := k.SetParams(ctx, data.Params); err != nil {
			return fmt.Errorf("failed to set params: %w", err)
		}
	} else {
		p := types.DefaultParams()
		if err := k.SetParams(ctx, &p); err != nil {
			return fmt.Errorf("failed to set default params: %w", err)
		}
	}

	for _, account := range data.Accounts {
		if account != nil {
			k.SetAccount(ctx, account)
		}
	}

	for _, mapping := range data.DidMappings {
		if mapping != nil {
			k.SetDIDMapping(ctx, mapping)
		}
	}

	return nil
}

// ExportGenesis exports the module state for genesis.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	params := k.GetParams(ctx)

	var accounts []*types.Account
	k.IterateAccounts(ctx, func(account *types.Account) bool {
		accounts = append(accounts, account)
		return false
	})

	var mappings []*types.DIDMapping
	kvStore := k.storeService.OpenKVStore(ctx)
	didIter, err := kvStore.Iterator(types.DIDMappingPrefix, prefixEndBytes(types.DIDMappingPrefix))
	if err == nil {
		defer didIter.Close()
		for ; didIter.Valid(); didIter.Next() {
			mapping := new(types.DIDMapping)
			if err := proto.Unmarshal(didIter.Value(), mapping); err != nil {
				continue
			}
			mappings = append(mappings, mapping)
		}
	}

	return &types.GenesisState{
		Params:      params,
		Accounts:    accounts,
		DidMappings: mappings,
	}
}

