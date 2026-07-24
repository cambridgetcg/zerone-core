package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/substrate_bridge/keeper"
	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

// The Adapters query is the surface an agent actually reaches for when it asks
// "which knowledge adapters can I use?". It has to answer honestly in both
// modes: unfiltered listing, and a status filter served off the 0x89 reverse
// index. These pin the query path itself, not just the keeper beneath it —
// the filter previously reached the registry through a full scan, so an index
// that returned nothing would never have been noticed here.
func TestQueryAdapters(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeper(t)
	q := keeper.NewQueryServerImpl(k)

	for _, a := range []struct {
		id     string
		status types.AdapterStatus
	}{
		{"agenttool-invocation-v1", types.AdapterStatus_ADAPTER_STATUS_ACTIVE},
		{"wikipedia-en-v1", types.AdapterStatus_ADAPTER_STATUS_ACTIVE},
		{"retired-v0", types.AdapterStatus_ADAPTER_STATUS_SUSPENDED},
	} {
		require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{AdapterId: a.id, Status: a.status}))
	}

	ids := func(res *types.QueryAdaptersResponse) []string {
		out := make([]string, 0, len(res.Adapters))
		for _, a := range res.Adapters {
			out = append(out, a.AdapterId)
		}
		return out
	}

	t.Run("unfiltered lists every adapter", func(t *testing.T) {
		res, err := q.Adapters(ctx, &types.QueryAdaptersRequest{})
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"agenttool-invocation-v1", "wikipedia-en-v1", "retired-v0"}, ids(res))
	})

	t.Run("active filter returns only active", func(t *testing.T) {
		res, err := q.Adapters(ctx, &types.QueryAdaptersRequest{
			StatusFilter: types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
		})
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"agenttool-invocation-v1", "wikipedia-en-v1"}, ids(res))
	})

	t.Run("suspended filter returns only suspended", func(t *testing.T) {
		res, err := q.Adapters(ctx, &types.QueryAdaptersRequest{
			StatusFilter: types.AdapterStatus_ADAPTER_STATUS_SUSPENDED,
		})
		require.NoError(t, err)
		require.Equal(t, []string{"retired-v0"}, ids(res))
	})

	// A status nothing holds must come back empty rather than falling open to
	// the full list — an over-broad answer to a narrow question is a lie.
	t.Run("unmatched status returns empty, not everything", func(t *testing.T) {
		res, err := q.Adapters(ctx, &types.QueryAdaptersRequest{
			StatusFilter: types.AdapterStatus_ADAPTER_STATUS_TOMBSTONED,
		})
		require.NoError(t, err)
		require.Empty(t, res.Adapters)
	})

	t.Run("filter follows a status transition", func(t *testing.T) {
		require.NoError(t, k.SuspendAdapter(ctx, "wikipedia-en-v1", "incident"))

		active, err := q.Adapters(ctx, &types.QueryAdaptersRequest{
			StatusFilter: types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
		})
		require.NoError(t, err)
		require.Equal(t, []string{"agenttool-invocation-v1"}, ids(active))

		suspended, err := q.Adapters(ctx, &types.QueryAdaptersRequest{
			StatusFilter: types.AdapterStatus_ADAPTER_STATUS_SUSPENDED,
		})
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"retired-v0", "wikipedia-en-v1"}, ids(suspended))
	})
}
