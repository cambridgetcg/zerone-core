package keeper

import (
	"bytes"
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
	cdc          codec.Codec
	storeService store.KVStoreService
	authority    string
}

// NewKeeper creates a new Keeper instance.
func NewKeeper(
	cdc codec.Codec,
	storeService store.KVStoreService,
	authority string,
) Keeper {
	return Keeper{
		cdc:          cdc,
		storeService: storeService,
		authority:    authority,
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

// terminalAuthIteratorError tolerates the Cosmos SDK cache iterator's normal
// EOF defect: cacheMerge and mem iterators report their exhausted state as an
// "invalid ...Iterator" error. Every other traversal error is still surfaced,
// and callers check Close independently.
func terminalAuthIteratorError(iter interface {
	Valid() bool
	Error() error
}) error {
	err := iter.Error()
	if err == nil {
		return nil
	}
	if !iter.Valid() {
		switch err.Error() {
		case "invalid cacheMergeIterator", "invalid memIterator":
			return nil
		}
	}
	return err
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
	if err != nil {
		panic(fmt.Sprintf("failed to read auth account %q: %v", address, err))
	}
	if bz == nil {
		return nil, false
	}
	var account types.Account
	if err := proto.Unmarshal(bz, &account); err != nil {
		panic(fmt.Sprintf("failed to decode auth account %q: %v", address, err))
	}
	if account.Address != address {
		panic(fmt.Sprintf("auth account store key %q does not match encoded address %q", address, account.Address))
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
	if err != nil {
		panic(fmt.Sprintf("failed to read auth DID mapping %q: %v", did, err))
	}
	if bz == nil {
		return nil, false
	}
	var mapping types.DIDMapping
	if err := proto.Unmarshal(bz, &mapping); err != nil {
		panic(fmt.Sprintf("failed to decode auth DID mapping %q: %v", did, err))
	}
	if mapping.Did != did {
		panic(fmt.Sprintf("auth DID mapping store key %q does not match encoded DID %q", did, mapping.Did))
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
	if err != nil {
		panic(fmt.Sprintf("failed to read auth params: %v", err))
	}
	if bz == nil {
		panic("auth params are missing from initialized state")
	}
	var params types.Params
	if err := proto.Unmarshal(bz, &params); err != nil {
		panic(fmt.Sprintf("failed to decode auth params: %v", err))
	}
	if err := params.Validate(); err != nil {
		panic(fmt.Sprintf("invalid auth params in state: %v", err))
	}
	return &params
}

// SetLastRotation stores the block height of last key rotation.
func (k Keeper) SetLastRotation(ctx sdk.Context, address string, height uint64) {
	kvStore := k.storeService.OpenKVStore(ctx)
	if err := kvStore.Set(types.LastRotationKey(address), types.Uint64ToBytes(height)); err != nil {
		panic(fmt.Sprintf("failed to store last key rotation: %v", err))
	}
}

// GetLastRotation retrieves the block height of last key rotation.
func (k Keeper) GetLastRotation(ctx sdk.Context, address string) uint64 {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.LastRotationKey(address))
	if err != nil {
		panic(fmt.Sprintf("failed to read last key rotation: %v", err))
	}
	if bz == nil {
		return 0
	}
	if len(bz) != 8 {
		panic(fmt.Sprintf("invalid last key rotation encoding for %s", address))
	}
	return types.BytesToUint64(bz)
}

// GetAuthority returns the module authority address.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// IterateAccounts iterates over all accounts.
func (k Keeper) IterateAccounts(ctx sdk.Context, cb func(*types.Account) bool) {
	if err := k.iterateAccountsChecked(ctx, cb); err != nil {
		panic(fmt.Sprintf("failed to iterate auth accounts: %v", err))
	}
}

// iterateAccountsChecked scans accounts without concealing storage or decode
// failures. Consensus code must never treat corrupt state as an empty result.
func (k Keeper) iterateAccountsChecked(ctx sdk.Context, cb func(*types.Account) bool) error {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.AccountKeyPrefix, prefixEndBytes(types.AccountKeyPrefix))
	if err != nil {
		return err
	}

	for ; iter.Valid(); iter.Next() {
		var account types.Account
		if err := proto.Unmarshal(iter.Value(), &account); err != nil {
			_ = iter.Close()
			return fmt.Errorf("decode account at key %x: %w", iter.Key(), err)
		}
		if !bytes.Equal(iter.Key(), types.AccountKey(account.Address)) {
			_ = iter.Close()
			return fmt.Errorf("account store key does not match encoded address %q", account.Address)
		}
		if cb(&account) {
			break
		}
	}
	if err := terminalAuthIteratorError(iter); err != nil {
		_ = iter.Close()
		return err
	}
	return iter.Close()
}

// InitGenesis initializes the module state from genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, data *types.GenesisState) error {
	if err := data.Validate(); err != nil {
		return fmt.Errorf("invalid auth genesis: %w", err)
	}
	if err := k.SetParams(ctx, data.Params); err != nil {
		return fmt.Errorf("failed to set params: %w", err)
	}

	for _, account := range data.Accounts {
		k.SetAccount(ctx, account)
	}

	for _, mapping := range data.DidMappings {
		k.SetDIDMapping(ctx, mapping)
	}

	for _, rotation := range data.LastKeyRotations {
		k.SetLastRotation(ctx, rotation.Address, rotation.Height)
	}

	return nil
}

// ExportGenesis exports the module state for genesis.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	params := k.GetParams(ctx)

	var accounts []*types.Account
	if err := k.iterateAccountsChecked(ctx, func(account *types.Account) bool {
		accounts = append(accounts, account)
		return false
	}); err != nil {
		panic(fmt.Sprintf("failed to export auth accounts: %v", err))
	}

	var mappings []*types.DIDMapping
	kvStore := k.storeService.OpenKVStore(ctx)
	didIter, err := kvStore.Iterator(types.DIDMappingPrefix, prefixEndBytes(types.DIDMappingPrefix))
	if err != nil {
		panic(fmt.Sprintf("failed to iterate auth DID mappings: %v", err))
	}
	for ; didIter.Valid(); didIter.Next() {
		mapping := new(types.DIDMapping)
		if err := proto.Unmarshal(didIter.Value(), mapping); err != nil {
			_ = didIter.Close()
			panic(fmt.Sprintf("failed to decode auth DID mapping at key %x: %v", didIter.Key(), err))
		}
		if !bytes.Equal(didIter.Key(), types.DIDMappingKey(mapping.Did)) {
			_ = didIter.Close()
			panic(fmt.Sprintf("auth DID mapping store key does not match encoded DID %q", mapping.Did))
		}
		mappings = append(mappings, mapping)
	}
	if err := terminalAuthIteratorError(didIter); err != nil {
		_ = didIter.Close()
		panic(fmt.Sprintf("failed while iterating auth DID mappings: %v", err))
	}
	if err := didIter.Close(); err != nil {
		panic(fmt.Sprintf("failed to close auth DID mapping iterator: %v", err))
	}

	var rotations []*types.KeyRotationRecord
	rotationIter, err := kvStore.Iterator(types.LastRotationPrefix, prefixEndBytes(types.LastRotationPrefix))
	if err != nil {
		panic(fmt.Sprintf("failed to iterate auth key rotations: %v", err))
	}
	for ; rotationIter.Valid(); rotationIter.Next() {
		key := rotationIter.Key()
		if len(key) <= len(types.LastRotationPrefix) || !bytes.Equal(key[:len(types.LastRotationPrefix)], types.LastRotationPrefix) {
			_ = rotationIter.Close()
			panic(fmt.Sprintf("invalid auth key rotation store key %x", key))
		}
		value := rotationIter.Value()
		if len(value) != 8 {
			_ = rotationIter.Close()
			panic(fmt.Sprintf("invalid auth key rotation encoding at key %x", key))
		}
		address := string(key[len(types.LastRotationPrefix):])
		if !bytes.Equal(key, types.LastRotationKey(address)) {
			_ = rotationIter.Close()
			panic(fmt.Sprintf("auth key rotation store key does not match encoded address %q", address))
		}
		rotations = append(rotations, &types.KeyRotationRecord{
			Address: address,
			Height:  types.BytesToUint64(value),
		})
	}
	if err := terminalAuthIteratorError(rotationIter); err != nil {
		_ = rotationIter.Close()
		panic(fmt.Sprintf("failed while iterating auth key rotations: %v", err))
	}
	if err := rotationIter.Close(); err != nil {
		panic(fmt.Sprintf("failed to close auth key rotation iterator: %v", err))
	}

	genesis := &types.GenesisState{
		Params:           params,
		Accounts:         accounts,
		DidMappings:      mappings,
		LastKeyRotations: rotations,
	}
	// Export validation is chain-aware because the historical zerone-1
	// registration path accepted short/case-insensitive DID suffixes and an
	// empty operational-key hash on unrotated accounts. The validator preserves
	// only those exact committed encodings on zerone-1; it never normalizes a
	// DID or backfills a hash, proof of possession, or rotation history.
	// InitGenesis and every other chain remain on GenesisState.Validate's strict
	// path.
	if err := genesis.ValidateForExport(ctx.ChainID()); err != nil {
		panic(fmt.Sprintf("refusing to export invalid auth genesis: %v", err))
	}
	return genesis
}
