package keeper_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"testing"

	corestore "cosmossdk.io/core/store"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/emergency/keeper"
	"github.com/zerone-chain/zerone/x/emergency/types"
)

func TestHaltReasonFilterActivation(t *testing.T) {
	k, _, ctx, key := setupHaltReasonKeepers(t)
	storeHaltReasonCeremony(t, k, ctx, "legacy", "research_fund", "finalized", 100)
	before := haltReasonStoreBytes(t, ctx, key)
	enabled, err := k.HaltReasonFilterEnabled(ctx)
	require.NoError(t, err)
	require.False(t, enabled)
	cache, write := ctx.CacheContext()
	require.NoError(t, k.EnableHaltReasonFilter(cache))
	enabled, err = k.HaltReasonFilterEnabled(cache)
	require.NoError(t, err)
	require.True(t, enabled)
	enabled, err = k.HaltReasonFilterEnabled(ctx)
	require.NoError(t, err)
	require.False(t, enabled, "uncommitted activation must remain private to its cache")
	write()
	require.Equal(t, []byte{1}, ctx.KVStore(key).Get(types.HaltReasonFilterEnabledKey))
	after := haltReasonStoreBytes(t, ctx, key)
	require.Equal(t, append(before, types.HaltReasonFilterEnabledKey, []byte{1}), after, "activation changes only its new singleton")
	require.NoError(t, k.EnableHaltReasonFilter(ctx))
	require.Equal(t, after, haltReasonStoreBytes(t, ctx, key), "idempotent enable must not rewrite history")
	freshKeeper := keeper.NewKeeper(runtime.NewKVStoreService(key), nil, "authority", nil)
	enabled, err = freshKeeper.HaltReasonFilterEnabled(ctx)
	require.NoError(t, err)
	require.True(t, enabled, "selection belongs to the store, not the keeper instance")
}

func TestHaltReasonFilterMalformedMarker(t *testing.T) {
	for _, value := range [][]byte{{0}, {2}, {1, 0}} {
		t.Run(hex.EncodeToString(value), func(t *testing.T) {
			k, _, ctx, key := setupHaltReasonKeepers(t)
			ctx.KVStore(key).Set(types.HaltReasonFilterEnabledKey, value)
			enabled, err := k.HaltReasonFilterEnabled(ctx)
			require.ErrorContains(t, err, "malformed halt reason filter marker")
			require.False(t, enabled)
			require.Error(t, k.EnableHaltReasonFilter(ctx), "enable must not repair a corrupt marker")
			require.Equal(t, value, ctx.KVStore(key).Get(types.HaltReasonFilterEnabledKey))
			require.Panics(t, func() { keeper.NewGovEmergencyAdapter(k).CountHaltsForReason(ctx, "research_fund", 0) })
		})
	}
}

type haltReasonFaultService struct {
	corestore.KVStoreService
	getErr, setErr error
}

func (s haltReasonFaultService) OpenKVStore(ctx context.Context) corestore.KVStore {
	return haltReasonFaultStore{s.KVStoreService.OpenKVStore(ctx), s.getErr, s.setErr}
}

type haltReasonFaultStore struct {
	corestore.KVStore
	getErr, setErr error
}

func (s haltReasonFaultStore) Get(key []byte) ([]byte, error) {
	if bytes.Equal(key, types.HaltReasonFilterEnabledKey) && s.getErr != nil {
		return nil, s.getErr
	}
	return s.KVStore.Get(key)
}

func (s haltReasonFaultStore) Set(key, value []byte) error {
	if bytes.Equal(key, types.HaltReasonFilterEnabledKey) && s.setErr != nil {
		return s.setErr
	}
	return s.KVStore.Set(key, value)
}

func TestHaltReasonFilterStoreFailures(t *testing.T) {
	_, _, ctx, key := setupHaltReasonKeepers(t)
	failure := errors.New("injected marker failure")
	for _, tc := range []struct {
		name           string
		getErr, setErr error
	}{
		{"read", failure, nil}, {"write", nil, failure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			k := keeper.NewKeeper(haltReasonFaultService{runtime.NewKVStoreService(key), tc.getErr, tc.setErr}, nil, "authority", nil)
			require.ErrorIs(t, k.EnableHaltReasonFilter(ctx), failure)
			require.Nil(t, ctx.KVStore(key).Get(types.HaltReasonFilterEnabledKey))
			if tc.getErr != nil {
				enabled, err := k.HaltReasonFilterEnabled(ctx)
				require.ErrorIs(t, err, failure)
				require.False(t, enabled)
				require.Panics(t, func() { keeper.NewGovEmergencyAdapter(k).CountHaltsForReason(ctx, "research_fund", 0) })
			}
		})
	}
}

func TestGovEmergencyAdapterMalformedPayloadFailsClosed(t *testing.T) {
	wrongID, err := proto.Marshal(&types.EmergencyHaltProposal{Id: "different", Reason: "research_fund"})
	require.NoError(t, err)
	for _, tc := range []struct {
		name    string
		payload []byte
	}{
		{"invalid_wire", []byte{0xff}}, {"absent_payload", nil}, {"wrong_id", wrongID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			k, g, ctx, _ := setupHaltReasonKeepers(t)
			require.NoError(t, k.SetCeremony(ctx, &types.EmergencyCeremony{
				Id: "halt", Type: "halt", Phase: "finalized", StartBlock: 45_000_000, ProposalData: tc.payload,
			}))
			a := keeper.NewGovEmergencyAdapter(k)
			require.Equal(t, uint64(1), a.CountHaltsForReason(ctx, "research_fund", 40_000_000), "legacy never decoded payloads")
			require.NoError(t, k.EnableHaltReasonFilter(ctx))
			require.Panics(t, func() { a.CountHaltsForReason(ctx, "research_fund", 40_000_000) })
			require.Panics(t, func() { g.CheckPhaseExitConditions(ctx) }, "unknown evidence must not authorize forward progress")
			require.Panics(t, func() { g.ValidatePhaseRollback(ctx) }, "unknown evidence must not authorize rollback")
			require.Zero(t, a.CountHaltsForReason(ctx, "research_fund", 46_000_000), "out-of-window payloads are irrelevant")
		})
	}
}
