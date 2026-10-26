package keeper

import (
	"bytes"
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

const appIAVLInitSentinelKey = "_iavl_init"

func (k Keeper) InitGenesis(ctx context.Context, gs *types.GenesisState) error {
	if err := gs.Validate(); err != nil {
		return err
	}
	if gs.Params != nil {
		if err := k.SetParams(ctx, gs.Params); err != nil {
			return err
		}
	}
	for _, a := range gs.Adapters {
		if err := k.WriteAdapter(ctx, a); err != nil {
			return err
		}
	}

	store := sdk.UnwrapSDKContext(ctx).KVStore(k.storeKey)
	for _, entry := range gs.StateEntries {
		// GenesisState.Validate restricts these entries to replay/economic
		// keyspaces and excludes Params/Adapter records, so raw restoration
		// cannot bypass their typed validation paths.
		store.Set(bytes.Clone(entry.Key), bytes.Clone(entry.Value))
	}
	// Only an explicit true writes the marker; absence keeps legacy semantics.
	if gs.QualificationSemanticsEnabled {
		if err := k.EnableQualificationSemantics(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	params := k.GetParams(ctx)
	qualificationSemantics, err := k.QualificationSemanticsEnabled(ctx)
	if err != nil {
		panic(err)
	}
	gs := &types.GenesisState{Params: params, QualificationSemanticsEnabled: qualificationSemantics}
	k.IterateAdapters(ctx, func(a *types.AdapterRegistration) bool {
		gs.Adapters = append(gs.Adapters, a)
		return false
	})

	store := sdk.UnwrapSDKContext(ctx).KVStore(k.storeKey)
	iter := store.Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		// InitChainer writes this exact infrastructure sentinel into every
		// module store. It is app bootstrap state, not substrate-bridge state,
		// so it must not enter the portable module genesis.
		if bytes.Equal(iter.Key(), []byte(appIAVLInitSentinelKey)) {
			continue
		}
		if types.IsTypedGenesisStateKey(iter.Key()) {
			continue
		}
		if !types.IsAllowedGenesisStateKey(iter.Key()) {
			panic(fmt.Sprintf("substrate_bridge export has unhandled state key %x", iter.Key()))
		}
		gs.StateEntries = append(gs.StateEntries, &types.GenesisStateEntry{
			Key:   bytes.Clone(iter.Key()),
			Value: bytes.Clone(iter.Value()),
		})
	}
	return gs
}
