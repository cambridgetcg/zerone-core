package keeper_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	corestore "cosmossdk.io/core/store"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	home "github.com/zerone-chain/zerone/x/home"
	"github.com/zerone-chain/zerone/x/home/types"
)

// Exercise the real JSON wrappers and message server with separate IAVL stores.
// The bank is the existing synthetic fixture, not a native payment or node.
func TestHomeGenesisJSONRoundTripPreservesAllocator(t *testing.T) {
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	sourceMsgs, source, sourceCtx, sourceBank := setupMsgServer(t)
	sourceModule := home.NewAppModule(cdc, source)
	sourceModule.InitGenesis(sourceCtx, cdc, sourceModule.DefaultGenesis(cdc))
	alice, bob := testAddr("genesis-alice"), testAddr("genesis-bob")
	sourceBank.setBalance(alice, "uzrn", 100_000_000)

	for i := 1; i <= 2; i++ {
		resp, err := sourceMsgs.CreateHome(sourceCtx, &types.MsgCreateHome{
			Owner: alice, Name: fmt.Sprintf("Alice home %d", i),
		})
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("home-%d", i), resp.HomeId)
	}
	_, err := sourceMsgs.RegisterKey(sourceCtx, &types.MsgRegisterKey{
		Owner: alice, HomeId: "home-1", KeyHash: "synthetic-alice-key",
		KeyType: "cosmos", Role: "admin", Permissions: []string{"read", "write"},
	})
	require.NoError(t, err)

	originals := source.GetAllHomes(sourceCtx)
	require.Len(t, originals, 2)
	originalKey, found := source.GetKeyRegistration(sourceCtx, "home-1", "synthetic-alice-key")
	require.True(t, found)
	exported := sourceModule.ExportGenesis(sourceCtx, cdc)
	require.NoError(t, sourceModule.ValidateGenesis(cdc, nil, exported))
	// Existing exports have no allocator field; no schema extension is needed.
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(exported, &fields))
	require.Len(t, fields, 3)
	require.Contains(t, fields, "params")
	require.Contains(t, fields, "homes")
	require.Contains(t, fields, "key_sets")
	require.Equal(t, "home-3", source.GetNextHomeID(sourceCtx))

	targetMsgs, target, targetCtx, targetBank := setupMsgServer(t)
	targetModule := home.NewAppModule(cdc, target)
	targetModule.InitGenesis(targetCtx, cdc, exported)
	require.Equal(t, exported, targetModule.ExportGenesis(targetCtx, cdc), "import must preserve the existing JSON projection")
	targetBank.setBalance(bob, "uzrn", 100_000_000)
	for i := 3; i <= 4; i++ {
		resp, err := targetMsgs.CreateHome(targetCtx, &types.MsgCreateHome{
			Owner: bob, Name: fmt.Sprintf("Bob home %d", i),
		})
		require.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("home-%d", i), resp.HomeId, "relaunch must not reuse Alice's IDs")
	}
	assert.Len(t, target.GetAllHomes(targetCtx), 4)
	assert.Equal(t, []string{"home-1", "home-2"}, target.GetHomesByOwner(targetCtx, alice))
	assert.Equal(t, []string{"home-3", "home-4"}, target.GetHomesByOwner(targetCtx, bob))
	for _, original := range originals {
		got, found := target.GetHome(targetCtx, original.HomeId)
		require.True(t, found)
		assert.True(t, proto.Equal(original, got), "existing home %s changed: %v", original.HomeId, got)
	}
	gotKey, found := target.GetKeyRegistration(targetCtx, "home-1", "synthetic-alice-key")
	require.True(t, found)
	assert.True(t, proto.Equal(originalKey, gotKey), "Alice's key must stay on Alice's unchanged home")

	// A second export/import must retain the new high-water mark too.
	relaunchMsgs, relaunch, relaunchCtx, relaunchBank := setupMsgServer(t)
	relaunchModule := home.NewAppModule(cdc, relaunch)
	relaunchModule.InitGenesis(relaunchCtx, cdc, targetModule.ExportGenesis(targetCtx, cdc))
	relaunchBank.setBalance(bob, "uzrn", 100_000_000)
	resp, err := relaunchMsgs.CreateHome(relaunchCtx, &types.MsgCreateHome{Owner: bob, Name: "After second relaunch"})
	require.NoError(t, err)
	assert.Equal(t, "home-5", resp.HomeId)

	_, err = targetMsgs.UpdateHome(targetCtx, &types.MsgUpdateHome{Owner: bob, HomeId: "home-1", Name: "Not Bob's home"})
	assert.ErrorIs(t, err, types.ErrUnauthorized)
	_, err = targetMsgs.UpdateHome(targetCtx, &types.MsgUpdateHome{Owner: alice, HomeId: "home-1", Name: "Still Alice's home"})
	assert.NoError(t, err, "original owner must not be locked out")
}

func TestHomeInitGenesisAllocatorBounds(t *testing.T) {
	tests := []struct {
		name string
		ids  []string
		next string
	}{
		{name: "empty", next: "home-1"},
		{name: "sparse unordered numeric maximum", ids: []string{"home-100", "home-2", "home-9"}, next: "home-101"},
		{name: "zero", ids: []string{"home-0"}, next: "home-1"},
		{name: "largest allocatable successor", ids: []string{"home-18446744073709551614"}, next: "home-18446744073709551615"},
		// Legacy genesis validation accepts arbitrary IDs. Preserve them literally;
		// they cannot collide with IDs emitted by the decimal uint64 allocator.
		{name: "legacy noncanonical identifiers", ids: []string{
			"gen-home-1", "custom", "home-", "home--1", "home-+99", "home-099",
			"home-2suffix", "home- 99", "home-18446744073709551616",
			"home-018446744073709551615", "home-2",
		}, next: "home-3"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msgs, k, ctx, bank := setupMsgServer(t)
			cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
			module := home.NewAppModule(cdc, k)
			genesis := types.DefaultGenesis()
			owner, creator := testAddr("import-owner"), testAddr("new-owner")
			for _, id := range tc.ids {
				genesis.Homes = append(genesis.Homes, &types.AgentHome{
					HomeId: id, OwnerAddress: owner, Name: id, Status: "archived",
				})
			}
			data, err := json.Marshal(genesis)
			require.NoError(t, err)
			require.NoError(t, module.ValidateGenesis(cdc, nil, data))
			module.InitGenesis(ctx, cdc, data)
			bank.setBalance(creator, "uzrn", 100_000_000)
			resp, err := msgs.CreateHome(ctx, &types.MsgCreateHome{Owner: creator, Name: "Fresh"})
			require.NoError(t, err)
			require.Equal(t, tc.next, resp.HomeId)
			for _, original := range genesis.Homes {
				got, found := k.GetHome(ctx, original.HomeId)
				require.True(t, found)
				require.True(t, proto.Equal(original, got), "imported ID %q must be untouched", original.HomeId)
			}
		})
	}
}

func TestHomeInitGenesisRejectsExhaustedAllocatorBeforeWrites(t *testing.T) {
	var observed *genesisWriteService
	k, ctx, _ := setupKeeperWithStoreWrapper(t, func(s corestore.KVStoreService) corestore.KVStoreService {
		observed = &genesisWriteService{KVStoreService: s}
		return observed
	})
	observed.writes = nil // Exclude fixture parameter initialization.
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	module := home.NewAppModule(cdc, k)
	before := module.ExportGenesis(ctx, cdc)
	genesis := types.DefaultGenesis()
	genesis.Params.HomeCreationFee = "123"
	genesis.Homes = []*types.AgentHome{
		{HomeId: "home-1", OwnerAddress: testAddr("overflow-owner")},
		{HomeId: "home-18446744073709551615", OwnerAddress: testAddr("overflow-owner")},
	}
	data, err := json.Marshal(genesis)
	require.NoError(t, err)
	require.PanicsWithValue(t, "home ID space exhausted", func() { module.InitGenesis(ctx, cdc, data) })
	require.Empty(t, observed.writes, "exhaustion must be detected before any store write")
	require.Equal(t, before, module.ExportGenesis(ctx, cdc), "refuse before importing any state")
	require.Equal(t, "home-1", k.GetNextHomeID(ctx))
}

func TestHomeInitGenesisCounterWriteFailure(t *testing.T) {
	var observed *genesisWriteService
	k, ctx, _ := setupKeeperWithStoreWrapper(t, func(s corestore.KVStoreService) corestore.KVStoreService {
		observed = &genesisWriteService{KVStoreService: s}
		return observed
	})
	observed.writes = nil
	observed.failCounter = true
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	module := home.NewAppModule(cdc, k)
	before := module.ExportGenesis(ctx, cdc)
	owner := testAddr("counter-write-owner")
	genesis := types.DefaultGenesis()
	genesis.Params.HomeCreationFee = "123"
	genesis.Homes = []*types.AgentHome{{HomeId: "home-2", OwnerAddress: owner}}
	data, err := json.Marshal(genesis)
	require.NoError(t, err)
	require.PanicsWithValue(t, "failed to restore home ID counter: synthetic counter write failure", func() {
		module.InitGenesis(ctx, cdc, data)
	})
	require.Equal(t, [][]byte{types.HomeCounterKey}, observed.writes, "only the failed counter write may be attempted")
	require.Equal(t, before, module.ExportGenesis(ctx, cdc))
	require.Empty(t, k.GetHomesByOwner(ctx, owner))
	counter, err := observed.OpenKVStore(ctx).Get(types.HomeCounterKey)
	require.NoError(t, err)
	require.Nil(t, counter)
}

type genesisWriteService struct {
	corestore.KVStoreService
	writes      [][]byte
	failCounter bool
}

func (s *genesisWriteService) OpenKVStore(ctx context.Context) corestore.KVStore {
	return genesisWriteStore{KVStore: s.KVStoreService.OpenKVStore(ctx), owner: s}
}

type genesisWriteStore struct {
	corestore.KVStore
	owner *genesisWriteService
}

func (s genesisWriteStore) Set(key, value []byte) error {
	s.owner.writes = append(s.owner.writes, bytes.Clone(key))
	if s.owner.failCounter && bytes.Equal(key, types.HomeCounterKey) {
		return errors.New("synthetic counter write failure")
	}
	return s.KVStore.Set(key, value)
}
