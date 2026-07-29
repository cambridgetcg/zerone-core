package cli

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

func TestBindLinkAdapterPopulatesRelayOmissions(t *testing.T) {
	link := &types.SubstrateLink{
		Source: &types.ExternalSource{SourceId: "invocation-1"},
	}

	require.NoError(t, bindLinkAdapter(link, "agenttool-invocation-v1"))
	require.Equal(t, "agenttool-invocation-v1", link.AdapterId)
	require.Equal(t, "agenttool-invocation-v1", link.Source.AdapterId)
}

func TestBindLinkAdapterRejectsExplicitSourceConflict(t *testing.T) {
	link := &types.SubstrateLink{
		Source: &types.ExternalSource{
			AdapterId: "different-adapter-v1",
			SourceId:  "invocation-1",
		},
	}

	err := bindLinkAdapter(link, "agenttool-invocation-v1")
	require.ErrorContains(t, err, "conflicts")
	require.Empty(t, link.AdapterId)
	require.Equal(t, "different-adapter-v1", link.Source.AdapterId)
}
