package keeper_test

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/substrate_bridge/keeper"
	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

func TestAdapterRegistry_WriteAndGet(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeper(t)
	adapter := &types.AdapterRegistration{
		AdapterId:              "wikipedia-en-v1",
		SourceType:             "wikipedia",
		Version:                "1.0.0",
		CompilerBinaryHash:     []byte{0xde, 0xad, 0xbe, 0xef},
		MinAttestationBondUzrn: "222000",
		MinPerClaimBondUzrn:    "222",
		Status:                 types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
		RegisteredViaLipId:     "LIP-0001",
		RegisteredAtBlock:      100,
	}
	require.NoError(t, k.WriteAdapter(ctx, adapter))
	got, found := k.GetAdapter(ctx, "wikipedia-en-v1")
	require.True(t, found)
	require.Equal(t, adapter.AdapterId, got.AdapterId)
	require.Equal(t, adapter.Status, got.Status)
}

func TestAdapterRegistry_GetMissing(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeper(t)
	_, found := k.GetAdapter(ctx, "missing")
	require.False(t, found)
}

func TestAdapterRegistry_SuspendChangesStatus(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeper(t)
	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId: "test-adapter",
		Status:    types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
	}))
	require.NoError(t, k.SuspendAdapter(ctx, "test-adapter", "incident"))
	got, _ := k.GetAdapter(ctx, "test-adapter")
	require.Equal(t, types.AdapterStatus_ADAPTER_STATUS_SUSPENDED, got.Status)
}

func TestAdapterRegistry_TombstoneIsForwardOnly(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeper(t)
	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId: "doomed-adapter",
		Status:    types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
	}))
	require.NoError(t, k.TombstoneAdapter(ctx, "doomed-adapter"))
	got, _ := k.GetAdapter(ctx, "doomed-adapter")
	require.Equal(t, types.AdapterStatus_ADAPTER_STATUS_TOMBSTONED, got.Status)
	require.Greater(t, got.TombstonedAtBlock, uint64(0))

	err := k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId: "doomed-adapter",
		Status:    types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
	})
	require.ErrorIs(t, err, types.ErrAdapterTombstoned)
}

// collectByStatus drains IterateAdaptersByStatus into a sorted ID slice.
func collectByStatus(k keeper.Keeper, ctx sdk.Context, status types.AdapterStatus) []string {
	var ids []string
	k.IterateAdaptersByStatus(ctx, status, func(a *types.AdapterRegistration) bool {
		ids = append(ids, a.AdapterId)
		return false
	})
	sort.Strings(ids)
	return ids
}

// An agent asking "which adapters are ACTIVE?" must be able to tell an empty
// answer from a broken one. The reverse index reader used to strip only the
// status byte and not the 0x89 prefix, so every lookup missed and the callback
// never fired — the registry answered "none" no matter what was registered.
// Adapter IDs here differ in length on purpose: a prefix-arithmetic slip that
// happens to work for one width should still fail the others.
func TestAdapterRegistry_IterateByStatus_YieldsMatchingOnly(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeper(t)
	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId: "wikipedia-en-v1",
		Status:    types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
	}))
	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId: "a",
		Status:    types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
	}))
	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId: "quiet-one",
		Status:    types.AdapterStatus_ADAPTER_STATUS_SUSPENDED,
	}))

	require.Equal(t, []string{"a", "wikipedia-en-v1"},
		collectByStatus(k, ctx, types.AdapterStatus_ADAPTER_STATUS_ACTIVE))
	require.Equal(t, []string{"quiet-one"},
		collectByStatus(k, ctx, types.AdapterStatus_ADAPTER_STATUS_SUSPENDED))
}

func TestAdapterRegistry_IterateByStatus_FollowsTransition(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeper(t)
	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId: "shifting-adapter",
		Status:    types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
	}))
	require.NoError(t, k.SuspendAdapter(ctx, "shifting-adapter", "incident"))

	require.Empty(t, collectByStatus(k, ctx, types.AdapterStatus_ADAPTER_STATUS_ACTIVE))
	require.Equal(t, []string{"shifting-adapter"},
		collectByStatus(k, ctx, types.AdapterStatus_ADAPTER_STATUS_SUSPENDED))
}

func TestAdapterRegistry_IterateByStatus_EarlyStop(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeper(t)
	for _, id := range []string{"adapter-one", "adapter-two", "adapter-three"} {
		require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
			AdapterId: id,
			Status:    types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
		}))
	}
	seen := 0
	k.IterateAdaptersByStatus(ctx, types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
		func(*types.AdapterRegistration) bool {
			seen++
			return true
		})
	require.Equal(t, 1, seen)
}

// The agenttool-seam-v1 migration. A nil AxisBounds meant "unbounded" — the most
// permissive setting on chain, reachable only by omission. This replaces it with
// an explicit empty ceiling so that state stops existing, without disturbing an
// adapter that already declared one.
func TestDeclareMissingAxisBounds(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeper(t)

	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId: "agenttool-invocation-v1", Status: types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
	}))
	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId: "suspended-no-bounds", Status: types.AdapterStatus_ADAPTER_STATUS_SUSPENDED,
	}))
	// Already declares a real ceiling — must survive untouched.
	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId: "bounded-v1", Status: types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
		AxisBounds: &types.AxisBounds{AxisSubstrateMax: 4242},
	}))

	declared, err := k.DeclareMissingAxisBounds(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, declared)

	for _, id := range []string{"agenttool-invocation-v1", "suspended-no-bounds"} {
		got, found := k.GetAdapter(ctx, id)
		require.True(t, found)
		require.NotNil(t, got.AxisBounds, "%s must declare a ceiling after migration", id)
		require.Zero(t, got.AxisBounds.AxisSubstrateMax)
		require.Zero(t, got.AxisBounds.AxisInterfaceMax)
	}

	untouched, _ := k.GetAdapter(ctx, "bounded-v1")
	require.Equal(t, uint64(4242), untouched.AxisBounds.AxisSubstrateMax)

	// Idempotent: a second pass has nothing left to declare.
	again, err := k.DeclareMissingAxisBounds(ctx)
	require.NoError(t, err)
	require.Zero(t, again)
}

// Tombstoned adapters are terminal and WriteAdapter refuses to rewrite them, so
// the migration must skip rather than fail the whole upgrade on one dead row.
func TestDeclareMissingAxisBounds_SkipsTombstoned(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeper(t)
	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId: "dead-v1", Status: types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
	}))
	require.NoError(t, k.TombstoneAdapter(ctx, "dead-v1"))

	declared, err := k.DeclareMissingAxisBounds(ctx)
	require.NoError(t, err)
	require.Zero(t, declared)

	got, _ := k.GetAdapter(ctx, "dead-v1")
	require.Nil(t, got.AxisBounds)
}
