package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
)

func TestDomainStats_SetGet(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)
	stats := keeper.DomainStats{Domain: "physics", ActiveCount: 5, AtRiskCount: 2, TotalEnergy: 500000, LastUpdated: 100}
	k.SetDomainStats(ctx, &stats)
	got, found := k.GetDomainStats(ctx, "physics")
	require.True(t, found)
	require.Equal(t, uint64(5), got.ActiveCount)
	require.Equal(t, uint64(2), got.AtRiskCount)
	require.Equal(t, uint64(500000), got.TotalEnergy)
	require.Equal(t, uint64(100), got.LastUpdated)
}

func TestDomainStats_GetNotFound(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)
	got, found := k.GetDomainStats(ctx, "nonexistent")
	require.False(t, found)
	require.Equal(t, "nonexistent", got.Domain)
	require.Equal(t, uint64(0), got.ActiveCount)
}

func TestDomainStats_IncrementDecrement(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)
	k.IncrementDomainFactCount(ctx, "physics", true, 500000)  // active
	k.IncrementDomainFactCount(ctx, "physics", true, 300000)  // active
	k.IncrementDomainFactCount(ctx, "physics", false, 100000) // at-risk
	got, found := k.GetDomainStats(ctx, "physics")
	require.True(t, found)
	require.Equal(t, uint64(2), got.ActiveCount)
	require.Equal(t, uint64(1), got.AtRiskCount)
	require.Equal(t, uint64(900000), got.TotalEnergy)

	k.DecrementDomainFactCount(ctx, "physics", true, 500000)
	got, _ = k.GetDomainStats(ctx, "physics")
	require.Equal(t, uint64(1), got.ActiveCount)
	require.Equal(t, uint64(400000), got.TotalEnergy)
}

func TestDomainStats_DecrementUnderflow(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)
	// Decrement when already at zero — should not underflow
	k.DecrementDomainFactCount(ctx, "physics", true, 100)
	got, _ := k.GetDomainStats(ctx, "physics")
	require.Equal(t, uint64(0), got.ActiveCount)
	require.Equal(t, uint64(0), got.TotalEnergy)
}

func TestDomainStats_Transition(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)
	k.IncrementDomainFactCount(ctx, "physics", true, 500000)
	k.IncrementDomainFactCount(ctx, "physics", true, 300000)

	// Transition one from active → at-risk
	k.TransitionDomainFactStatus(ctx, "physics", false)
	got, _ := k.GetDomainStats(ctx, "physics")
	require.Equal(t, uint64(1), got.ActiveCount)
	require.Equal(t, uint64(1), got.AtRiskCount)

	// Transition it back: at-risk → active
	k.TransitionDomainFactStatus(ctx, "physics", true)
	got, _ = k.GetDomainStats(ctx, "physics")
	require.Equal(t, uint64(2), got.ActiveCount)
	require.Equal(t, uint64(0), got.AtRiskCount)
}
