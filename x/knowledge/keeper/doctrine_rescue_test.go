package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// TestResurrectDoctrineFacts proves the doctrine-metabolism-exempt-v1 migration:
// starving doctrine facts (EXPIRED, energy 0, at-risk) are restored to VERIFIED
// at full energy with the at-risk clock cleared — the exact rescue the upgrade
// handler runs on the live chain before block 260,000.
func TestResurrectDoctrineFacts(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)

	// Reproduce the pre-fix starving state: drive every genesis doctrine fact to
	// EXPIRED / energy 0 / at-risk, exactly where the 47 sit on mainnet today.
	var ids []string
	k.IterateFacts(ctx, func(f *types.Fact) bool {
		if f.Category == types.DoctrineCategory || f.MethodId == types.DoctrineMethodId {
			f.Status = types.FactStatus_FACT_STATUS_EXPIRED
			f.Energy = 0
			f.AtRiskSinceEpoch = 1
			require.NoError(t, k.SetFact(ctx, f))
			ids = append(ids, f.Id)
		}
		return false
	})
	require.NotEmpty(t, ids, "genesis should seed doctrine facts")

	n, err := k.ResurrectDoctrineFacts(ctx)
	require.NoError(t, err)
	require.Equal(t, len(ids), n)

	for _, id := range ids {
		got, found := k.GetFact(ctx, id)
		require.True(t, found)
		require.Equal(t, types.FactStatus_FACT_STATUS_VERIFIED, got.Status, "doctrine fact %s must be VERIFIED", id)
		require.Positive(t, got.Energy, "doctrine fact %s must have energy", id)
		require.Equal(t, uint64(0), got.AtRiskSinceEpoch, "doctrine fact %s at-risk clock must be cleared", id)
	}
}

// TestResurrectDoctrineFacts_Idempotent proves re-running the migration is a
// no-op on already-alive doctrine — safe if the handler ever runs twice.
func TestResurrectDoctrineFacts_Idempotent(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)

	n1, err := k.ResurrectDoctrineFacts(ctx)
	require.NoError(t, err)
	n2, err := k.ResurrectDoctrineFacts(ctx)
	require.NoError(t, err)
	require.Equal(t, n1, n2, "count is stable across runs")

	k.IterateFacts(ctx, func(f *types.Fact) bool {
		if f.Category == types.DoctrineCategory || f.MethodId == types.DoctrineMethodId {
			require.Equal(t, types.FactStatus_FACT_STATUS_VERIFIED, f.Status)
			require.Positive(t, f.Energy)
		}
		return false
	})
}
