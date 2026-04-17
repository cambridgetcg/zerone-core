package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetVerificationThresholdOverride_NotSet(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)
	additional, active := k.GetVerificationThresholdOverride(ctx, "physics")
	require.False(t, active, "no override set → inactive")
	require.Equal(t, uint32(0), additional)
}

func TestGetVerificationThresholdOverride_Active(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)
	ctx = advanceBlocks(ctx, 100) // height = 100
	require.NoError(t, k.IncreaseVerificationThreshold(ctx, "physics", 2, 500))

	additional, active := k.GetVerificationThresholdOverride(ctx, "physics")
	require.True(t, active, "override within expiry → active")
	require.Equal(t, uint32(2), additional)
}

func TestGetVerificationThresholdOverride_Expired(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)
	ctx = advanceBlocks(ctx, 100)
	require.NoError(t, k.IncreaseVerificationThreshold(ctx, "physics", 2, 200))

	ctx = advanceBlocks(ctx, 150) // height = 250 > expiry 200
	additional, active := k.GetVerificationThresholdOverride(ctx, "physics")
	require.False(t, active, "override past expiry → inactive")
	require.Equal(t, uint32(0), additional)
}

func TestGetVerificationThresholdOverride_DifferentDomain(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)
	ctx = advanceBlocks(ctx, 100)
	require.NoError(t, k.IncreaseVerificationThreshold(ctx, "physics", 2, 500))

	additional, active := k.GetVerificationThresholdOverride(ctx, "biology")
	require.False(t, active, "override is per-domain")
	require.Equal(t, uint32(0), additional)
}
