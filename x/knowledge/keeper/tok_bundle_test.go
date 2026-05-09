package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func TestValidateToKSelector_RejectsEmptyVariant(t *testing.T) {
	err := keeper.ValidateToKSelector(&types.ToKSelector{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "selector variant required")
}

func TestValidateToKSelector_RootedSubtree_RequiresRootFactId(t *testing.T) {
	sel := &types.ToKSelector{Variant: &types.ToKSelector_RootedSubtree{
		RootedSubtree: &types.RootedSubtreeSelector{},
	}}
	err := keeper.ValidateToKSelector(sel)
	require.Error(t, err)
	require.Contains(t, err.Error(), "root_fact_id")
}

func TestValidateToKSelector_RootedSubtree_CapsMaxDepth(t *testing.T) {
	sel := &types.ToKSelector{Variant: &types.ToKSelector_RootedSubtree{
		RootedSubtree: &types.RootedSubtreeSelector{
			RootFactId: "fact-1",
			MaxDepth:   100, // > cap 32
		},
	}}
	capped, err := keeper.ValidateAndCapToKSelector(sel)
	require.NoError(t, err)
	require.Equal(t, uint32(32), capped.GetRootedSubtree().MaxDepth)
}

func TestValidateToKSelector_Frontier_RequiresDomain(t *testing.T) {
	sel := &types.ToKSelector{Variant: &types.ToKSelector_Frontier{
		Frontier: &types.FrontierSelector{},
	}}
	err := keeper.ValidateToKSelector(sel)
	require.Error(t, err)
	require.Contains(t, err.Error(), "domain")
}
