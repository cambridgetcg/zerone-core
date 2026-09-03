package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func TestAgentEconomyStatusQueryReportsExactLineage(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)
	query := keeper.NewQueryServerImpl(k)

	response, err := query.AgentEconomyStatus(
		ctx,
		&types.QueryAgentEconomyStatusRequest{},
	)
	require.NoError(t, err)
	require.False(t, response.Activated)
	require.Equal(t, types.AgentEconomyLineageNone, response.Lineage)
	require.Empty(t, response.Marker)
	require.Empty(t, response.MarkerValue)
	require.Equal(t, uint64(ctx.BlockHeight()), response.SnapshotBlockHeight)

	require.NoError(t, k.WriteMigrationMarker(
		ctx,
		types.AgentEconomyNativeMarker,
		types.AgentEconomyActivationValue,
	))
	response, err = query.AgentEconomyStatus(
		ctx,
		&types.QueryAgentEconomyStatusRequest{},
	)
	require.NoError(t, err)
	require.True(t, response.Activated)
	require.Equal(t, types.AgentEconomyLineageNative, response.Lineage)
	require.Equal(t, types.AgentEconomyNativeMarker, response.Marker)
	require.Equal(t, types.AgentEconomyActivationValue, response.MarkerValue)
}

func TestAgentEconomyStatusQueryFailsOnAmbiguousLineage(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)
	for _, marker := range []string{
		types.AgentEconomyUpgradeMarker,
		types.AgentEconomyNativeMarker,
	} {
		require.NoError(t, k.WriteMigrationMarker(
			ctx,
			marker,
			types.AgentEconomyActivationValue,
		))
	}

	response, err := keeper.NewQueryServerImpl(k).AgentEconomyStatus(
		ctx,
		&types.QueryAgentEconomyStatusRequest{},
	)
	require.Nil(t, response)
	require.Equal(t, codes.Internal, status.Code(err))
	require.ErrorContains(t, err, "conflicting agent-economy lineages")
}
