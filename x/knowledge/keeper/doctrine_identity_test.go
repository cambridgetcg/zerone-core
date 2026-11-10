package keeper_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// The doctrine-identity repair (marker 0x94): with the marker on, the
// metabolism exemption follows a fact's provenance, not its free-text label.
// A fact made from a claim is metabolised whatever its category says; the
// chain's own doctrine (genesis, MsgAddFact, a matured guardian-veto injection)
// stays exempt. With the marker off, the label alone decides, as before.

// doctrineIdentityGenesis is setupKnowledgeTestWithBank's predecessor genesis
// with the doctrine-identity marker set on or off. Only the marker varies.
func doctrineIdentityGenesis(enabled bool) *types.GenesisState {
	gs := types.DefaultGenesis()
	gs.ReviewNeutralityEnabled = false
	gs.ClaimRecordsEnabled = false
	gs.FundSettlementEnabled = false
	gs.ClaimIntegrityEnabled = false
	gs.RecordIntegrityEnabled = false
	gs.ReformulationVindicationOrderingEnabled = false
	gs.BootstrapFundAllocation = "22222000000"
	gs.DoctrineIdentityEnabled = enabled
	return gs
}

func setupDoctrineIdentityTest(t *testing.T, gs *types.GenesisState) (keeper.Keeper, sdk.Context, *trackingBankKeeper) {
	t.Helper()
	k, ctx, bk := setupKnowledgeTestWithBankGenesis(t, gs)
	enabled, err := k.DoctrineIdentityEnabled(ctx)
	require.NoError(t, err)
	require.Equal(t, gs.DoctrineIdentityEnabled, enabled)
	return k, ctx, bk
}

func mustGetFact(t *testing.T, k keeper.Keeper, ctx sdk.Context, id string) *types.Fact {
	t.Helper()
	fact, found := k.GetFact(ctx, id)
	require.True(t, found, id)
	return fact
}

// A fact made from a claim, labelled doctrine by category or by method, is
// exempt while the marker is off and metabolised once a carrier turns it on.
func TestDoctrineIdentity_ClaimDerivedDoctrineLabelIsMetabolisedOnlyWithMarker(t *testing.T) {
	for _, tc := range []struct {
		name             string
		category, method string
	}{
		{"category", types.DoctrineCategory, "M-LEGACY"},
		{"method", "formal", types.DoctrineMethodId},
	} {
		t.Run(tc.name, func(t *testing.T) {
			k, ctx, _ := setupDoctrineIdentityTest(t, doctrineIdentityGenesis(false))
			setDomainNormalPressure(k, ctx, "physics")

			fact := makeEnergyFact("fact-labelled", "Claim-derived doctrine label", "physics", 500_000, types.FactStatus_FACT_STATUS_VERIFIED)
			fact.ClaimId = "claim-labelled"
			fact.Category = tc.category
			fact.MethodId = tc.method
			require.NoError(t, k.SetFact(ctx, fact))
			before := mustGetFact(t, k, ctx, fact.Id)

			require.NoError(t, k.ProcessMetabolism(ctx, 1))
			require.Equal(t, before, mustGetFact(t, k, ctx, fact.Id), "marker off: the label alone exempts the fact")

			require.NoError(t, k.EnableDoctrineIdentity(ctx))
			require.NoError(t, k.ProcessMetabolism(ctx, 2))
			after := mustGetFact(t, k, ctx, fact.Id)
			// The same base drain as TestMetabolism_BaseDrain: 10,000.
			require.Equal(t, uint64(490_000), after.Energy, "marker on: a fact with a claim is metabolised")
			require.Equal(t, uint64(ctx.BlockHeight()), after.EnergyLastUpdated)
		})
	}
}

// In both marker states the chain's own doctrine stays exempt: the 47 genesis
// doctrine facts, a doctrine fact the authority adds with MsgAddFact, and a
// claimless fact that carries only the doctrine_authorship method (as an
// imported genesis fact could). An ordinary fact in the same run is
// metabolised, so the run did happen and the exemption is not a blanket one.
func TestDoctrineIdentity_ChainDoctrineStaysExemptInBothMarkerStates(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T) (keeper.Keeper, sdk.Context)
	}{
		{"marker off", func(t *testing.T) (keeper.Keeper, sdk.Context) {
			k, ctx, _ := setupDoctrineIdentityTest(t, doctrineIdentityGenesis(false))
			return k, ctx
		}},
		{"native genesis", func(t *testing.T) (keeper.Keeper, sdk.Context) {
			k, ctx, _ := setupDoctrineIdentityTest(t, doctrineIdentityGenesis(true))
			return k, ctx
		}},
		{"enabled by a carrier", func(t *testing.T) (keeper.Keeper, sdk.Context) {
			k, ctx, _ := setupDoctrineIdentityTest(t, doctrineIdentityGenesis(false))
			require.NoError(t, k.EnableDoctrineIdentity(ctx))
			return k, ctx
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			k, ctx := tc.setup(t)
			setDomainNormalPressure(k, ctx, "physics")

			resp, err := keeper.NewMsgServerImpl(k).AddFact(ctx, &types.MsgAddFact{
				Authority:  k.GetAuthority(),
				Content:    "A doctrine fact added by the chain's authority",
				Domain:     "physics",
				Category:   types.DoctrineCategory,
				Confidence: 800_000,
			})
			require.NoError(t, err)
			added := mustGetFact(t, k, ctx, resp.FactId)
			require.Empty(t, added.ClaimId)
			require.Equal(t, types.DoctrineCategory, added.Category)

			methodOnly := makeEnergyFact("fact-method-only", "A claimless fact with only the doctrine method", "physics", 500_000, types.FactStatus_FACT_STATUS_VERIFIED)
			methodOnly.Category = "formal"
			methodOnly.MethodId = types.DoctrineMethodId
			require.NoError(t, k.SetFact(ctx, methodOnly))

			ordinary := makeEnergyFact("fact-ordinary", "An ordinary verified fact", "physics", 500_000, types.FactStatus_FACT_STATUS_VERIFIED)
			require.NoError(t, k.SetFact(ctx, ordinary))

			doctrine := map[string]*types.Fact{}
			k.IterateFacts(ctx, func(f *types.Fact) bool {
				if f.Category == types.DoctrineCategory || f.MethodId == types.DoctrineMethodId {
					doctrine[f.Id] = f
				}
				return false
			})
			require.Len(t, doctrine, 47+2, "47 genesis doctrine facts, the added one and the method-only one")
			require.Contains(t, doctrine, methodOnly.Id)

			require.NoError(t, k.ProcessMetabolism(ctx, 1))

			for id, before := range doctrine {
				require.Equal(t, before, mustGetFact(t, k, ctx, id), "doctrine without a claim stays exempt: %s", id)
			}
			require.Less(t, mustGetFact(t, k, ctx, ordinary.Id).Energy, ordinary.Energy, "the ordinary fact was metabolised")
		})
	}
}

// End to end: with the marker off, a conjecture labelled doctrine is admitted
// (no admission rule reads the category), accepted, and becomes a PROVISIONAL
// fact that keeps the label and gains a claim. ProcessMetabolism exempts it
// while the marker stays off and metabolises it once a carrier turns the
// marker on. With the marker on, admission refuses the label
// (doctrine_reserved_test.go), so such a fact can only predate the marker.
func TestDoctrineIdentity_DoctrineLabelledConjectureEndToEnd(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		name := "marker off"
		if enabled {
			name = "enabled by a carrier"
		}
		t.Run(name, func(t *testing.T) {
			k, ctx, bk := setupDoctrineIdentityTest(t, doctrineIdentityGenesis(false))
			ms := keeper.NewMsgServerImpl(k)

			proposer := makeValidBech32Addr("doctrine-label-conjecture")
			bk.balances[proposer] = sdk.NewCoins(sdk.NewInt64Coin("uzrn", 10_000_000))
			resp, err := ms.PostConjecture(ctx, &types.MsgPostConjecture{
				Proposer:               proposer,
				Statement:              "Every sufficiently large even number is the sum of two primes",
				FalsificationPredicate: "Exhibit an even number above the bound that is not a sum of two primes",
				Category:               types.DoctrineCategory,
				Stake:                  "1000000",
			})
			require.NoError(t, err, "marker off: the doctrine label is not refused at admission")

			round, found := k.GetVerificationRound(ctx, resp.RoundId)
			require.True(t, found)
			require.NoError(t, k.CompleteRound(ctx, round, &keeper.VerificationResult{
				Verdict:    types.Verdict_VERDICT_ACCEPT,
				Confidence: 900_000,
			}))

			var fact *types.Fact
			k.IterateFacts(ctx, func(f *types.Fact) bool {
				if f.ClaimId == resp.ClaimId {
					fact = f
					return true
				}
				return false
			})
			require.NotNil(t, fact, "the accepted conjecture made a fact")
			require.Equal(t, types.DoctrineCategory, fact.Category, "the label rides through to the fact")
			require.Equal(t, types.FactStatus_FACT_STATUS_PROVISIONAL, fact.Status)
			require.Zero(t, fact.Confidence)
			require.NotZero(t, fact.Energy)

			if enabled {
				require.NoError(t, k.EnableDoctrineIdentity(ctx))
			}
			require.NoError(t, k.ProcessMetabolism(ctx, 1))
			after := mustGetFact(t, k, ctx, fact.Id)
			if !enabled {
				require.Equal(t, fact, after, "marker off: the label exempts a claim-derived fact")
				return
			}
			require.Less(t, after.Energy, fact.Energy, "marker on: the claim-derived fact is metabolised")
		})
	}
}

// End to end: an accepted challenge of a genesis doctrine fact makes a new fact
// from the challenge claim. The challenger does not choose its category. With
// the marker off the claim inherits "doctrine" from its target, and the fact
// it makes is exempt. With the marker on the claim's category is empty
// (challengeClaimCategory), so the fact is not labelled doctrine, and it is
// metabolised.
func TestDoctrineIdentity_AcceptedChallengeOfDoctrineEndToEnd(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "marker off", true: "marker on"}[enabled], func(t *testing.T) {
			k, ctx, bk := setupDoctrineIdentityTest(t, doctrineIdentityGenesis(enabled))
			target := mustGetFact(t, k, ctx, "commitment-1")
			require.Empty(t, target.ClaimId)
			require.Equal(t, types.DoctrineCategory, target.Category)

			params, err := k.GetParams(ctx)
			require.NoError(t, err)
			stake := keeper.EffectiveMinChallengeStake(params, target.Confidence)
			challenger := makeValidBech32Addr("doctrine-challenger")
			bk.balances[challenger] = sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(stake)))

			resp, err := keeper.NewMsgServerImpl(k).ChallengeFact(ctx, &types.MsgChallengeFact{
				Challenger: challenger,
				FactId:     target.Id,
				Reason:     "A challenge of genesis doctrine",
				Stake:      stake.String(),
			})
			require.NoError(t, err)
			round, found := k.GetVerificationRound(ctx, resp.RoundId)
			require.True(t, found)
			claim, found := k.GetClaim(ctx, round.ClaimId)
			require.True(t, found)
			wantCategory := types.DoctrineCategory
			if enabled {
				wantCategory = ""
			}
			require.Equal(t, wantCategory, claim.Category, "the challenge claim inherits the label only with the marker off")

			require.NoError(t, k.CompleteRound(ctx, round, &keeper.VerificationResult{
				Verdict:    types.Verdict_VERDICT_ACCEPT,
				Confidence: 900_000,
			}))

			var fact *types.Fact
			k.IterateFacts(ctx, func(f *types.Fact) bool {
				if f.ClaimId == claim.Id {
					fact = f
					return true
				}
				return false
			})
			require.NotNil(t, fact, "the accepted challenge made a fact")
			require.Equal(t, wantCategory, fact.Category, "the claim's category rides through to the fact")
			require.NotZero(t, fact.Energy)

			require.NoError(t, k.ProcessMetabolism(ctx, 1))
			after := mustGetFact(t, k, ctx, fact.Id)
			if !enabled {
				require.Equal(t, fact, after, "marker off: the inherited label exempts a claim-derived fact")
				return
			}
			require.Less(t, after.Energy, fact.Energy, "marker on: the claim-derived fact is metabolised")
		})
	}
}

// A doctrine fact the authority adds while a guardian-veto window is set is
// queued, and materialised at the end of the window without a claim. It stays
// exempt in both marker states, while an ordinary fact in the same run is
// metabolised.
func TestDoctrineIdentity_VetoWindowDoctrineStaysExempt(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "marker off", true: "marker on"}[enabled], func(t *testing.T) {
			k, ctx, _ := setupDoctrineIdentityTest(t, doctrineIdentityGenesis(enabled))
			params, err := k.GetParams(ctx)
			require.NoError(t, err)
			params.AddFactVetoWindowBlocks = 10
			params.GuardianAddresses = []string{makeValidBech32Addr("doctrine-guardian")}
			require.NoError(t, k.SetParams(ctx, params))
			require.True(t, k.AddFactVetoEnabled(ctx))

			resp, err := keeper.NewMsgServerImpl(k).AddFact(ctx, &types.MsgAddFact{
				Authority:  k.GetAuthority(),
				Content:    "A doctrine fact added through the guardian-veto window",
				Domain:     "physics",
				Category:   types.DoctrineCategory,
				Confidence: 800_000,
			})
			require.NoError(t, err)
			_, found := k.GetFact(ctx, resp.FactId)
			require.False(t, found, "queued, not yet a fact")

			ctx = ctx.WithBlockHeight(ctx.BlockHeight() + int64(params.AddFactVetoWindowBlocks))
			k.MaterializeMaturedFactInjections(ctx, uint64(ctx.BlockHeight()))
			added := mustGetFact(t, k, ctx, resp.FactId)
			require.Empty(t, added.ClaimId)
			require.Equal(t, types.DoctrineCategory, added.Category)

			setDomainNormalPressure(k, ctx, "physics")
			ordinary := makeEnergyFact("fact-ordinary", "An ordinary verified fact", "physics", 500_000, types.FactStatus_FACT_STATUS_VERIFIED)
			require.NoError(t, k.SetFact(ctx, ordinary))

			require.NoError(t, k.ProcessMetabolism(ctx, 1))
			require.Equal(t, added, mustGetFact(t, k, ctx, added.Id), "doctrine without a claim stays exempt")
			require.Less(t, mustGetFact(t, k, ctx, ordinary.Id).Energy, ordinary.Energy, "the ordinary fact was metabolised")
		})
	}
}

// BeginBlocker selects the marker at entry and both of its bodies (Q2 on and
// the frozen pre-Q2 body) carry that selection to metabolism at a fitness
// epoch.
func TestDoctrineIdentity_BeginBlockerCarriesEntrySelection(t *testing.T) {
	for _, q2 := range []bool{false, true} {
		for _, enabled := range []bool{false, true} {
			t.Run(map[bool]string{false: "pre-q2 body", true: "q2 body"}[q2]+"/"+map[bool]string{false: "marker off", true: "marker on"}[enabled], func(t *testing.T) {
				gs := doctrineIdentityGenesis(enabled)
				gs.InconclusiveRoundAccountingEnabled = q2
				k, ctx, _ := setupDoctrineIdentityTest(t, gs)
				setDomainNormalPressure(k, ctx, "physics")

				fact := makeEnergyFact("fact-labelled", "Claim-derived doctrine label", "physics", 500_000, types.FactStatus_FACT_STATUS_VERIFIED)
				fact.ClaimId = "claim-labelled"
				fact.Category = types.DoctrineCategory
				require.NoError(t, k.SetFact(ctx, fact))

				params, err := k.GetParams(ctx)
				require.NoError(t, err)
				ctx = ctx.WithBlockHeader(cmtproto.Header{Height: int64(params.FitnessEpochBlocks)})
				require.NoError(t, k.BeginBlocker(ctx))

				after := mustGetFact(t, k, ctx, fact.Id)
				if !enabled {
					require.Equal(t, fact.Energy, after.Energy, "marker off: exempt")
					require.Zero(t, after.EnergyLastUpdated)
					return
				}
				require.Less(t, after.Energy, fact.Energy, "marker on: metabolised")
				require.Equal(t, params.FitnessEpochBlocks, after.EnergyLastUpdated)
			})
		}
	}
}
