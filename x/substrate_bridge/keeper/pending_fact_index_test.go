package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPendingFactIndex_LinkLookup(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeper(t)
	require.NoError(t, k.LinkPendingClaim(ctx, "claim-1", "att-A"))
	require.NoError(t, k.LinkPendingClaim(ctx, "claim-2", "att-A"))
	require.NoError(t, k.LinkPendingClaim(ctx, "claim-3", "att-B"))

	got, found := k.GetAttestationForPendingClaim(ctx, "claim-1")
	require.True(t, found)
	require.Equal(t, "att-A", got)
	got, found = k.GetAttestationForPendingClaim(ctx, "claim-3")
	require.True(t, found)
	require.Equal(t, "att-B", got)
}

func TestPendingFactIndex_AttestationForwardWalk(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeper(t)
	require.NoError(t, k.LinkPendingClaim(ctx, "claim-1", "att-A"))
	require.NoError(t, k.LinkPendingClaim(ctx, "claim-2", "att-A"))
	require.NoError(t, k.LinkPendingClaim(ctx, "claim-3", "att-A"))

	claims := k.PendingClaimsFor(ctx, "att-A")
	require.Len(t, claims, 3)
}

func TestPendingFactIndex_UnlinkBoth(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeper(t)
	require.NoError(t, k.LinkPendingClaim(ctx, "claim-1", "att-A"))
	require.NoError(t, k.UnlinkPendingClaim(ctx, "claim-1", "att-A"))
	_, found := k.GetAttestationForPendingClaim(ctx, "claim-1")
	require.False(t, found)
	require.Empty(t, k.PendingClaimsFor(ctx, "att-A"))
}
