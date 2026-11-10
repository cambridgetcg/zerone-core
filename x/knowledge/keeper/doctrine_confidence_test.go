package keeper_test

import (
	"testing"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Doctrine carries no growing confidence under the doctrine-identity marker:
// the confidence growth epoch (AdvanceConfidence) and the fitness epoch
// (UpdateAllFitnessScores) neither grow nor clamp the chain's own doctrine.
// No stored value is reset. With the marker off both steps grow and clamp
// doctrine as before.

const (
	genesisDoctrineID = "commitment-1"
	labelledID        = "fact-labelled"
	ordinaryID        = "fact-ordinary"
	ordinaryHighID    = "fact-ordinary-high"
)

// seedConfidenceFacts adds, beside the 47 genesis doctrine facts, a doctrine
// fact from MsgAddFact (no claim), a claim-derived fact labelled doctrine, an
// ordinary fact and an ordinary fact one growth step below the cap. It returns
// the added doctrine fact's ID.
func seedConfidenceFacts(t *testing.T, k keeper.Keeper, ctx sdk.Context) string {
	t.Helper()
	resp, err := keeper.NewMsgServerImpl(k).AddFact(ctx, &types.MsgAddFact{
		Authority:  k.GetAuthority(),
		Content:    "A doctrine fact added by the chain's authority",
		Domain:     "physics",
		Category:   types.DoctrineCategory,
		Confidence: 800_000,
	})
	require.NoError(t, err)
	require.Empty(t, mustGetFact(t, k, ctx, resp.FactId).ClaimId)

	labelled := makeEnergyFact(labelledID, "Claim-derived doctrine label", "physics", 500_000, types.FactStatus_FACT_STATUS_VERIFIED)
	labelled.ClaimId = "claim-labelled"
	labelled.Category = types.DoctrineCategory
	labelled.Confidence = 500_000
	require.NoError(t, k.SetFact(ctx, labelled))

	ordinary := makeEnergyFact(ordinaryID, "An ordinary verified fact", "physics", 500_000, types.FactStatus_FACT_STATUS_VERIFIED)
	ordinary.ClaimId = "claim-ordinary"
	ordinary.Category = "empirical"
	ordinary.Confidence = 500_000
	require.NoError(t, k.SetFact(ctx, ordinary))

	high := makeEnergyFact(ordinaryHighID, "An ordinary verified fact near the cap", "physics", 500_000, types.FactStatus_FACT_STATUS_VERIFIED)
	high.ClaimId = "claim-ordinary-high"
	high.Category = "empirical"
	high.Confidence = 879_000
	require.NoError(t, k.SetFact(ctx, high))
	return resp.FactId
}

func confidences(k keeper.Keeper, ctx sdk.Context) map[string]uint64 {
	out := map[string]uint64{}
	k.IterateFacts(ctx, func(f *types.Fact) bool {
		out[f.Id] = f.Confidence
		return false
	})
	return out
}

// isChainDoctrine mirrors the keeper's provenance predicate for the facts
// these tests seed: no claim, and the doctrine category or method.
func isChainDoctrine(f *types.Fact) bool {
	return f.ClaimId == "" && (f.Category == types.DoctrineCategory || f.MethodId == types.DoctrineMethodId)
}

// The same facts run through one epoch step in both marker states. With the
// marker off doctrine grows and is clamped (genesis doctrine from 1,000,000 to
// the 880,000 cap). With it on, every claimless doctrine fact keeps the value it
// had, while every other fact, the claim-derived one labelled doctrine
// included, ends exactly where it ends with the marker off: ordinary facts
// still grow, and one near the cap is still clamped.
func TestDoctrineIdentity_EpochStepsLeaveDoctrineConfidenceAlone(t *testing.T) {
	for _, step := range []string{"growth", "fitness"} {
		t.Run(step, func(t *testing.T) {
			run := func(enabled bool) (keeper.Keeper, sdk.Context, string, map[string]uint64, map[string]uint64) {
				k, ctx, _ := setupDoctrineIdentityTest(t, doctrineIdentityGenesis(enabled))
				added := seedConfidenceFacts(t, k, ctx)
				before := confidences(k, ctx)
				if step == "growth" {
					require.NoError(t, k.AdvanceConfidence(ctx))
				} else {
					require.NoError(t, k.UpdateAllFitnessScores(ctx))
				}
				return k, ctx, added, before, confidences(k, ctx)
			}
			kOff, ctxOff, addedDoctrineID, before, off := run(false)
			kOn, ctxOn, addedOn, beforeOn, on := run(true)
			require.Equal(t, addedDoctrineID, addedOn)
			require.Equal(t, before, beforeOn, "both runs start from the same facts")

			require.EqualValues(t, types.DoctrineConfidence, before[genesisDoctrineID])
			require.EqualValues(t, 880_000, off[genesisDoctrineID], "marker off: genesis doctrine is grown and clamped to the cap")
			require.EqualValues(t, types.DoctrineConfidence, on[genesisDoctrineID], "marker on: genesis doctrine keeps the value it was born with")
			require.EqualValues(t, 808_800, off[addedDoctrineID], "marker off: authority doctrine grows")
			require.EqualValues(t, 800_000, on[addedDoctrineID], "marker on: authority doctrine keeps its value")
			require.EqualValues(t, 505_500, on[ordinaryID], "marker on: an ordinary fact still grows")
			require.EqualValues(t, 880_000, on[ordinaryHighID], "marker on: an ordinary fact is still clamped")
			require.EqualValues(t, 505_500, on[labelledID], "marker on: a claim-derived fact labelled doctrine grows like any other")

			doctrine := 0
			kOn.IterateFacts(ctxOn, func(f *types.Fact) bool {
				if isChainDoctrine(f) {
					doctrine++
					require.Equal(t, before[f.Id], on[f.Id], "marker on: doctrine %s is neither grown nor clamped", f.Id)
					return false
				}
				require.Equal(t, off[f.Id], on[f.Id], "marker on: %s ends where it ends with the marker off", f.Id)
				return false
			})
			require.Equal(t, 47+1, doctrine)
			kOff.IterateFacts(ctxOff, func(f *types.Fact) bool {
				if isChainDoctrine(f) && before[f.Id] > 0 {
					require.Greater(t, off[f.Id], uint64(0))
					require.NotEqual(t, before[f.Id], off[f.Id], "marker off: doctrine %s moves", f.Id)
				}
				return false
			})

			if step == "fitness" {
				// The fitness step still scores doctrine; only its confidence
				// is left alone.
				fact := mustGetFact(t, kOn, ctxOn, genesisDoctrineID)
				require.Equal(t, uint64(ctxOn.BlockHeight()), fact.FitnessUpdatedBlock)
			}
		})
	}
}

// A chain that has already clamped its doctrine and turns the marker on later
// keeps the clamped value: nothing resets it to 1,000,000, and it no longer
// moves.
func TestDoctrineIdentity_LaterActivationResetsNothing(t *testing.T) {
	k, ctx, _ := setupDoctrineIdentityTest(t, doctrineIdentityGenesis(false))
	addedDoctrineID := seedConfidenceFacts(t, k, ctx)
	require.NoError(t, k.AdvanceConfidence(ctx))
	require.EqualValues(t, 880_000, mustGetFact(t, k, ctx, genesisDoctrineID).Confidence)

	require.NoError(t, k.EnableDoctrineIdentity(ctx))
	ordinary := mustGetFact(t, k, ctx, ordinaryID).Confidence
	require.NoError(t, k.AdvanceConfidence(ctx))
	require.NoError(t, k.UpdateAllFitnessScores(ctx))
	require.EqualValues(t, 880_000, mustGetFact(t, k, ctx, genesisDoctrineID).Confidence)
	require.EqualValues(t, 808_800, mustGetFact(t, k, ctx, addedDoctrineID).Confidence, "the value it held at activation")
	require.Greater(t, mustGetFact(t, k, ctx, ordinaryID).Confidence, ordinary, "ordinary facts keep growing")
}

// With the doctrine-identity marker on and the zero-confidence floor off, the
// growth step keeps the predecessor's rule for ordinary facts: a fact at zero
// still gains the minimum unit. With the floor on it stays at zero.
func TestDoctrineIdentity_GrowthKeepsTheZeroFloorSelection(t *testing.T) {
	for _, floor := range []bool{false, true} {
		t.Run(map[bool]string{false: "floor off", true: "floor on"}[floor], func(t *testing.T) {
			gs := doctrineIdentityGenesis(true)
			gs.ZeroConfidenceFloorEnabled = floor
			k, ctx, _ := setupDoctrineIdentityTest(t, gs)
			require.NoError(t, k.SetFact(ctx, &types.Fact{Id: "zero", ClaimId: "claim-zero", Domain: "physics", Status: types.FactStatus_FACT_STATUS_VERIFIED}))
			require.NoError(t, k.AdvanceConfidence(ctx))
			want := uint64(1)
			if floor {
				want = 0
			}
			require.Equal(t, want, mustGetFact(t, k, ctx, "zero").Confidence)
			require.EqualValues(t, types.DoctrineConfidence, mustGetFact(t, k, ctx, genesisDoctrineID).Confidence)
		})
	}
}

// BeginBlocker selects the marker at entry, and both of its bodies (Q2 on and
// the frozen pre-Q2 body) carry that selection to the confidence growth epoch
// and to the fitness epoch.
func TestDoctrineIdentity_BeginBlockerCarriesSelectionToConfidenceSteps(t *testing.T) {
	for _, step := range []string{"growth", "fitness"} {
		for _, q2 := range []bool{false, true} {
			for _, enabled := range []bool{false, true} {
				t.Run(step+"/"+map[bool]string{false: "pre-q2 body", true: "q2 body"}[q2]+"/"+markerName(enabled), func(t *testing.T) {
					gs := doctrineIdentityGenesis(enabled)
					gs.InconclusiveRoundAccountingEnabled = q2
					k, ctx, _ := setupDoctrineIdentityTest(t, gs)
					seedConfidenceFacts(t, k, ctx)

					params, err := k.GetParams(ctx)
					require.NoError(t, err)
					height := params.ConfidenceGrowthEpoch
					if step == "fitness" {
						height = params.FitnessEpochBlocks
					}
					require.NotZero(t, height%map[string]uint64{"growth": params.FitnessEpochBlocks, "fitness": params.ConfidenceGrowthEpoch}[step], "only one of the two steps runs at this height")
					ctx = ctx.WithBlockHeader(cmtproto.Header{Height: int64(height)})
					require.NoError(t, k.BeginBlocker(ctx))

					want := uint64(880_000)
					if enabled {
						want = types.DoctrineConfidence
					}
					require.Equal(t, want, mustGetFact(t, k, ctx, genesisDoctrineID).Confidence)
					require.EqualValues(t, 505_500, mustGetFact(t, k, ctx, ordinaryID).Confidence, "the step ran")
				})
			}
		}
	}
}
