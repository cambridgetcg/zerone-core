package keeper

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"

	corestore "cosmossdk.io/core/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/zerone-chain/zerone/x/knowledge/types"
	"google.golang.org/protobuf/proto"
)

// walkFactRelationGenesis visits both canonical index namespaces, with one
// shared budget. Neither a failed iterator nor an over-budget inventory may
// produce a usable partial export or replacement.
func walkFactRelationGenesis(store corestore.KVStore, visit func(key, value []byte) error) error {
	entries, size := 0, 0
	for _, prefix := range [][]byte{types.FactRelationPrefix, types.FactRelationReversePrefix} {
		err := func() (err error) {
			it, err := store.Iterator(prefix, prefixEndBytes(prefix))
			if err != nil {
				return err
			}
			defer func() { err = errors.Join(err, it.Close()) }()
			for ; it.Valid(); it.Next() {
				key, value := it.Key(), it.Value()
				entries++
				size += len(key) + len(value)
				if entries > types.MaxFactRelationGenesisEntries || size > types.MaxFactRelationGenesisBytes {
					return fmt.Errorf("fact relation genesis exceeds inventory bound")
				}
				if err := visit(key, value); err != nil {
					return err
				}
			}
			return historyIteratorError(it)
		}()
		if err != nil {
			return err
		}
	}
	return nil
}

// ExportFactRelationGenesis reads the entire canonical graph, rather than
// reconstructing it from embedded Fact arrays or Claim proposals. Every pair
// must have equal forward/reverse bytes and existing, readable Fact endpoints.
// Canonical wire equality also refuses duplicate or overflowing protobuf fields
// that decoding and a later JSON export would otherwise silently normalize.
func (k Keeper) ExportFactRelationGenesis(ctx context.Context) (*types.FactRelationGenesis, error) {
	store := k.storeService.OpenKVStore(ctx)
	forward := make(map[string][]byte)
	reverse := make(map[string][]byte)
	relations := make(map[string]*types.FactRelation)
	err := walkFactRelationGenesis(store, func(key, value []byte) error {
		rel := new(types.FactRelation)
		if err := proto.Unmarshal(value, rel); err != nil {
			return fmt.Errorf("decode fact relation: %w", err)
		}
		if err := types.ValidateFactRelationRecord(rel); err != nil {
			return err
		}
		canonical, err := marshalOpts.Marshal(rel)
		if err != nil {
			return err
		}
		if !bytes.Equal(value, canonical) {
			return fmt.Errorf("noncanonical fact relation protobuf payload")
		}
		forwardKey := types.FactRelationKey(rel.SourceFactId, rel.TargetFactId)
		expectedKey := forwardKey
		isForward := bytes.HasPrefix(key, types.FactRelationPrefix)
		if !isForward {
			expectedKey = types.FactRelationReverseKey(rel.TargetFactId, rel.SourceFactId)
		}
		if !bytes.Equal(key, expectedKey) {
			return fmt.Errorf("fact relation key/payload mismatch")
		}
		if isForward {
			forward[string(forwardKey)] = bytes.Clone(value)
			relations[string(forwardKey)] = rel
		} else {
			reverse[string(forwardKey)] = bytes.Clone(value)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(forward) != len(reverse) {
		return nil, fmt.Errorf("fact relation forward/reverse inventory mismatch")
	}
	keys := make([]string, 0, len(forward))
	for key, value := range forward {
		other, found := reverse[key]
		if !found || !bytes.Equal(value, other) {
			return nil, fmt.Errorf("fact relation missing or unequal reverse payload")
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	state := &types.FactRelationGenesis{Relations: make([]*types.FactRelation, 0, len(keys))}
	checkedFacts := make(map[string]bool)
	for _, key := range keys {
		rel := relations[key]
		for _, id := range []string{rel.SourceFactId, rel.TargetFactId} {
			if checkedFacts[id] {
				continue
			}
			_, found, err := k.getFactChecked(ctx, id)
			if err != nil {
				return nil, fmt.Errorf("read fact relation endpoint: %w", err)
			}
			if !found {
				return nil, fmt.Errorf("fact relation references an absent Fact")
			}
			checkedFacts[id] = true
		}
		state.Relations = append(state.Relations, rel)
	}
	return state, nil
}

// ImportFactRelationGenesis applies an explicit inventory after doctrine
// seeding. Nil preserves the historical seed behavior; present empty removes all
// canonical edges. Only the two relation namespaces are replaced, atomically.
// The full genesis validates endpoints without inferring edges from Fact data.
func (k Keeper) ImportFactRelationGenesis(ctx context.Context, gs *types.GenesisState) error {
	if err := types.ValidateFactRelationGenesis(gs); err != nil {
		return err
	}
	if gs.FactRelationState == nil {
		return nil
	}
	cache, write := sdk.UnwrapSDKContext(ctx).CacheContext()
	store := k.storeService.OpenKVStore(cache)
	var keys [][]byte
	if err := walkFactRelationGenesis(store, func(key, _ []byte) error {
		keys = append(keys, bytes.Clone(key))
		return nil
	}); err != nil {
		return err
	}
	for _, key := range keys {
		if err := store.Delete(key); err != nil {
			return err
		}
	}
	for _, rel := range gs.FactRelationState.Relations {
		if err := k.SetFactRelation(cache, rel); err != nil {
			return err
		}
	}
	write()
	return nil
}
