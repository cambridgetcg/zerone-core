package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// ResurrectDoctrineFacts selects doctrine as metabolism does. With the marker
// off the label decides, so claim-derived facts labelled doctrine (by category
// or by method) are promoted to VERIFIED at full energy with the chain's own
// doctrine. With it on only claimless doctrine is rescued, and the
// claim-derived facts are left exactly as they were.
func TestDoctrineIdentity_RescueFollowsProvenance(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(markerName(enabled), func(t *testing.T) {
			k, ctx, _ := setupDoctrineIdentityTest(t, doctrineIdentityGenesis(enabled))
			addedDoctrineID := seedConfidenceFacts(t, k, ctx)
			byMethod := makeEnergyFact("fact-labelled-method", "Claim-derived doctrine method", "physics", 0, types.FactStatus_FACT_STATUS_EXPIRED)
			byMethod.ClaimId = "claim-labelled-method"
			byMethod.Category = "formal"
			byMethod.MethodId = types.DoctrineMethodId
			require.NoError(t, k.SetFact(ctx, byMethod))
			conjecture := makeEnergyFact("fact-labelled-conjecture", "Claim-derived doctrine conjecture", "physics", 100, types.FactStatus_FACT_STATUS_PROVISIONAL)
			conjecture.ClaimId = "claim-labelled-conjecture"
			conjecture.Category = types.DoctrineCategory
			conjecture.ClaimType = types.ClaimType_CLAIM_TYPE_CONJECTURE
			require.NoError(t, k.SetFact(ctx, conjecture))

			// Starve every fact that carries the label, as the 47 were before
			// doctrine-metabolism-exempt-v1.
			k.IterateFacts(ctx, func(f *types.Fact) bool {
				if f.ClaimId == "" && (f.Category == types.DoctrineCategory || f.MethodId == types.DoctrineMethodId) {
					f.Status = types.FactStatus_FACT_STATUS_EXPIRED
					f.Energy = 0
					f.AtRiskSinceEpoch = 1
					require.NoError(t, k.SetFact(ctx, f))
				}
				return false
			})
			claimDerived := []string{labelledID, byMethod.Id, conjecture.Id}
			before := map[string]*types.Fact{}
			for _, id := range claimDerived {
				before[id] = mustGetFact(t, k, ctx, id)
			}

			n, err := k.ResurrectDoctrineFacts(ctx)
			require.NoError(t, err)
			chain := 47 + 1 // genesis doctrine and the authority's fact
			if !enabled {
				require.Equal(t, chain+len(claimDerived), n, "marker off: the label decides")
				for _, id := range claimDerived {
					got := mustGetFact(t, k, ctx, id)
					require.Equal(t, types.FactStatus_FACT_STATUS_VERIFIED, got.Status, id)
				}
				return
			}
			require.Equal(t, chain, n, "marker on: only claimless doctrine is rescued")
			for _, id := range claimDerived {
				require.Equal(t, before[id], mustGetFact(t, k, ctx, id), "marker on: %s is left as it was", id)
			}
			for _, id := range []string{genesisDoctrineID, addedDoctrineID} {
				got := mustGetFact(t, k, ctx, id)
				require.Equal(t, types.FactStatus_FACT_STATUS_VERIFIED, got.Status, id)
				require.Positive(t, got.Energy, id)
				require.Zero(t, got.AtRiskSinceEpoch, id)
			}
		})
	}
}
