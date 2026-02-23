package keeper

import (
	"context"
	"encoding/json"
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/icaauth/types"
)

type Keeper struct {
	cdc             codec.Codec
	storeService    store.KVStoreService
	icaController   types.ICAControllerKeeper
	authority       string
}

func NewKeeper(
	cdc codec.Codec,
	storeService store.KVStoreService,
	icaController types.ICAControllerKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:           cdc,
		storeService:  storeService,
		icaController: icaController,
		authority:     authority,
	}
}

func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", "x/"+types.ModuleName)
}

func (k Keeper) GetAuthority() string {
	return k.authority
}

// ---- Params ----

func (k Keeper) SetParams(ctx context.Context, params *types.Params) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(params)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal params: %v", err))
	}
	_ = kvStore.Set(types.ParamsKey, bz)
}

func (k Keeper) GetParams(ctx context.Context) *types.Params {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.ParamsKey)
	if err != nil || bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := json.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
	return &params
}

// ---- Record CRUD ----

func recordKey(owner string) []byte {
	return append(types.RecordKeyPrefix, []byte(owner)...)
}

func (k Keeper) GetRecord(ctx context.Context, owner string) (*types.InterchainAccountRecord, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(recordKey(owner))
	if err != nil || bz == nil {
		return nil, false
	}
	var rec types.InterchainAccountRecord
	if err := json.Unmarshal(bz, &rec); err != nil {
		return nil, false
	}
	return &rec, true
}

func (k Keeper) SetRecord(ctx context.Context, rec *types.InterchainAccountRecord) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(rec)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal record: %v", err))
	}
	_ = kvStore.Set(recordKey(rec.Owner), bz)
}

func (k Keeper) GetAllRecords(ctx context.Context) []*types.InterchainAccountRecord {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.RecordKeyPrefix, prefixEndBytes(types.RecordKeyPrefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var records []*types.InterchainAccountRecord
	for ; iter.Valid(); iter.Next() {
		var rec types.InterchainAccountRecord
		if err := json.Unmarshal(iter.Value(), &rec); err != nil {
			continue
		}
		records = append(records, &rec)
	}
	return records
}

// AddRemoteAccount adds a new remote account to an owner's record.
func (k Keeper) AddRemoteAccount(ctx context.Context, owner string, account *types.RemoteAccount) {
	rec, found := k.GetRecord(ctx, owner)
	if !found {
		rec = &types.InterchainAccountRecord{
			Owner:    owner,
			Accounts: nil,
		}
	}
	rec.Accounts = append(rec.Accounts, account)
	k.SetRecord(ctx, rec)
}

// GetRemoteAccounts returns all remote accounts for an owner.
func (k Keeper) GetRemoteAccounts(ctx context.Context, owner string) []*types.RemoteAccount {
	rec, found := k.GetRecord(ctx, owner)
	if !found {
		return nil
	}
	return rec.Accounts
}

// GetRemoteAccountByConnection finds a remote account by owner and connection ID.
func (k Keeper) GetRemoteAccountByConnection(ctx context.Context, owner, connectionID string) (*types.RemoteAccount, bool) {
	rec, found := k.GetRecord(ctx, owner)
	if !found {
		return nil, false
	}
	for _, acct := range rec.Accounts {
		if acct.ConnectionId == connectionID {
			return acct, true
		}
	}
	return nil, false
}

// UpdateRemoteAccountAddress updates a remote account's address and sets it active.
func (k Keeper) UpdateRemoteAccountAddress(ctx context.Context, owner, connectionID, remoteAddress string) {
	rec, found := k.GetRecord(ctx, owner)
	if !found {
		return
	}
	for _, acct := range rec.Accounts {
		if acct.ConnectionId == connectionID {
			acct.RemoteAddress = remoteAddress
			acct.Active = true
			break
		}
	}
	k.SetRecord(ctx, rec)
}

// ---- Genesis ----

func (k Keeper) InitGenesis(ctx context.Context, genState *types.GenesisState) {
	if genState.Params != nil {
		k.SetParams(ctx, genState.Params)
	} else {
		k.SetParams(ctx, types.DefaultParams())
	}
	for _, rec := range genState.Records {
		k.SetRecord(ctx, rec)
	}
}

func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	return &types.GenesisState{
		Params:  k.GetParams(ctx),
		Records: k.GetAllRecords(ctx),
	}
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
