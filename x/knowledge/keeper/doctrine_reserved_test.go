package keeper_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// The doctrine label, under the doctrine-identity marker (0x94). Doctrine is
// what the chain's own process wrote, so with the marker on: admission refuses
// the category "doctrine", a challenge of doctrine does not inherit it, and
// the two epoch steps that grow and clamp confidence leave doctrine alone.
// With the marker off every path behaves as before.

func markerName(enabled bool) string {
	return map[bool]string{false: "marker off", true: "marker on"}[enabled]
}

// withRecordMarkers turns on the record markers up to depth, each with the
// predecessors its genesis validation requires: 1 record integrity, 2 review
// neutrality, 3 claim records, 4 fund settlement.
func withRecordMarkers(gs *types.GenesisState, depth int) *types.GenesisState {
	gs.RecordIntegrityEnabled = depth >= 1
	gs.ReviewNeutralityEnabled = depth >= 2
	gs.ClaimRecordsEnabled = depth >= 3
	gs.FundSettlementEnabled = depth >= 4
	return gs
}

// setupDoctrineReservedTest is setupDoctrineIdentityTest with a chain ID,
// which record integrity's commitment scheme requires.
func setupDoctrineReservedTest(t *testing.T, gs *types.GenesisState) (keeper.Keeper, sdk.Context, *trackingBankKeeper) {
	t.Helper()
	k, ctx, bk := setupDoctrineIdentityTest(t, gs)
	return k, ctx.WithChainID("doctrine-reserved-test"), bk
}

func countClaims(k keeper.Keeper, ctx sdk.Context) int {
	n := 0
	k.IterateClaims(ctx, func(*types.Claim) bool { n++; return false })
	return n
}

// admitWithCategory submits one admission of the given kind carrying category
// and returns the new claim's ID.
func admitWithCategory(t *testing.T, k keeper.Keeper, ctx sdk.Context, bk *trackingBankKeeper, kind, category string) (string, error) {
	t.Helper()
	ms := keeper.NewMsgServerImpl(k)
	author := makeValidBech32Addr("doctrine-reserved-author")
	bk.balances[author] = sdk.NewCoins(sdk.NewInt64Coin("uzrn", 100_000_000))
	switch kind {
	case "conjecture", "conjecture/funded":
		resp, err := ms.PostConjecture(ctx, &types.MsgPostConjecture{
			Proposer:               author,
			Statement:              "A conjecture whose category is under test: " + category,
			FalsificationPredicate: "A reproducible counterexample under the stated condition.",
			Domain:                 "physics",
			Category:               category,
			Stake:                  "11000000",
		})
		if err != nil {
			return "", err
		}
		return resp.ClaimId, nil
	case "contradiction", "contradiction/records":
		target := makeTestFact(t, k, ctx, "fact-contradicted", "An established assertion", "physics", "empirical", "zrn1author", 800_000)
		resp, err := ms.SubmitContradiction(ctx, &types.MsgSubmitContradiction{
			Submitter:    author,
			FactId:       target.Id,
			CounterClaim: "A counter-claim whose category is under test: " + category,
			Category:     category,
			Stake:        "11000000",
		})
		if err != nil {
			return "", err
		}
		return resp.CounterFactId, nil
	case "claim/legacy":
		resp, err := ms.SubmitClaim(ctx, &types.MsgSubmitClaim{
			Submitter:   author,
			FactContent: "A reproducible observation whose category is under test: " + category,
			Domain:      "physics",
			Category:    category,
			Stake:       "11000000",
		})
		if err != nil {
			return "", err
		}
		return resp.ClaimId, nil
	}
	t.Fatalf("unknown admission %q", kind)
	return "", nil
}

// Each admission path that copies a submitter's category refuses exactly
// "doctrine" with the marker on, on every branch of its entry point. An empty
// category, an ordinary one and near misses of the label ("Doctrine",
// "doctrine ") are admitted as before and keep their value, and with the
// marker off "doctrine" is admitted too.
func TestDoctrineReserved_AdmissionRefusesOnlyTheDoctrineCategory(t *testing.T) {
	for _, kind := range []string{"conjecture", "conjecture/funded", "contradiction", "contradiction/records", "claim/legacy"} {
		for _, enabled := range []bool{false, true} {
			for _, category := range []string{types.DoctrineCategory, "", "empirical", "Doctrine", "doctrine "} {
				t.Run(kind+"/"+markerName(enabled)+"/category="+category, func(t *testing.T) {
					depth := map[string]int{"conjecture/funded": 4, "contradiction/records": 3}[kind]
					k, ctx, bk := setupDoctrineReservedTest(t, withRecordMarkers(doctrineIdentityGenesis(enabled), depth))
					if kind == "claim/legacy" {
						recordIntegrity, err := k.RecordIntegrityEnabled(ctx)
						require.NoError(t, err)
						require.False(t, recordIntegrity, "the legacy claim path is the one under test")
					}

					claims, sends, events := countClaims(k, ctx), len(bk.sendCalls), len(ctx.EventManager().Events())
					id, err := admitWithCategory(t, k, ctx, bk, kind, category)
					if enabled && category == types.DoctrineCategory {
						require.ErrorIs(t, err, keeper.ErrDoctrineCategoryReserved)
						require.EqualError(t, err, `category "doctrine" is reserved for doctrine written by the chain's authority`)
						require.Equal(t, claims, countClaims(k, ctx), "no claim is written")
						require.Equal(t, sends, len(bk.sendCalls), "no coin moves")
						require.Len(t, ctx.EventManager().Events(), events, "no event")
						return
					}
					require.NoError(t, err)
					claim, found := k.GetClaim(ctx, id)
					require.True(t, found)
					require.Equal(t, category, claim.Category)
				})
			}
		}
	}
}

// With record integrity on, MsgSubmitClaim already refuses "doctrine": it is
// not one of the five epistemic categories. That path does not read the
// doctrine-identity marker, and its answer is the same in both marker states.
func TestDoctrineReserved_RecordIntegritySubmitClaimUnchanged(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(markerName(enabled), func(t *testing.T) {
			k, ctx, bk := setupDoctrineReservedTest(t, withRecordMarkers(doctrineIdentityGenesis(enabled), 1))
			ms := keeper.NewMsgServerImpl(k)
			author := makeValidBech32Addr("doctrine-reserved-record")
			bk.balances[author] = sdk.NewCoins(sdk.NewInt64Coin("uzrn", 100_000_000))
			msg := &types.MsgSubmitClaim{
				Submitter:   author,
				FactContent: "A reproducible observation with a retained test procedure.",
				Domain:      "physics",
				Category:    types.DoctrineCategory,
				Stake:       "11000000",
			}
			_, err := ms.SubmitClaim(ctx, msg)
			require.EqualError(t, err, `unknown epistemic category "doctrine"`)
			require.NotErrorIs(t, err, keeper.ErrDoctrineCategoryReserved)

			msg.Category = "empirical"
			resp, err := ms.SubmitClaim(ctx, msg)
			require.NoError(t, err)
			claim, found := k.GetClaim(ctx, resp.ClaimId)
			require.True(t, found)
			require.Equal(t, "empirical", claim.Category)
		})
	}
}

// openChallenge challenges target and returns the challenge claim.
func openChallenge(t *testing.T, k keeper.Keeper, ctx sdk.Context, bk *trackingBankKeeper, kind string, target *types.Fact) *types.Claim {
	t.Helper()
	ms := keeper.NewMsgServerImpl(k)
	challenger := makeValidBech32Addr("doctrine-reserved-challenger")
	bk.balances[challenger] = sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewInt(1_000_000_000)))
	var roundID string
	switch kind {
	case "challenge", "challenge/records":
		params, err := k.GetParams(ctx)
		require.NoError(t, err)
		resp, err := ms.ChallengeFact(ctx, &types.MsgChallengeFact{
			Challenger: challenger,
			FactId:     target.Id,
			Reason:     "The repeated measurement differs under the stated condition.",
			Stake:      keeper.EffectiveMinChallengeStake(params, target.Confidence).String(),
		})
		require.NoError(t, err)
		roundID = resp.RoundId
	case "provisional", "provisional/records":
		resp, err := ms.ChallengeProvisionalFact(ctx, &types.MsgChallengeProvisionalFact{
			Challenger:   challenger,
			FactId:       target.Id,
			Reason:       "The proposed falsification condition holds in this example.",
			CounterClaim: "The recorded counterexample contradicts the conjecture.",
			Stake:        "11000000",
		})
		require.NoError(t, err)
		roundID = resp.ChallengeId
	default:
		t.Fatalf("unknown challenge %q", kind)
	}
	round, found := k.GetVerificationRound(ctx, roundID)
	require.True(t, found)
	claim, found := k.GetClaim(ctx, round.ClaimId)
	require.True(t, found)
	return claim
}

// A challenge copies its target's category, except that with the marker on a
// target labelled doctrine gives its challenge an empty category. Challenging
// doctrine stays possible, on both record-integrity branches of both challenge
// messages, and an accepted challenge makes a fact that is not labelled
// doctrine. A target with an ordinary category passes it on in both states.
func TestDoctrineReserved_ChallengeDoesNotInheritTheLabel(t *testing.T) {
	for _, kind := range []string{"challenge", "challenge/records", "provisional", "provisional/records"} {
		for _, enabled := range []bool{false, true} {
			for _, category := range []string{types.DoctrineCategory, "empirical"} {
				t.Run(kind+"/"+markerName(enabled)+"/target="+category, func(t *testing.T) {
					depth := map[string]int{"challenge/records": 1, "provisional/records": 1}[kind]
					k, ctx, bk := setupDoctrineReservedTest(t, withRecordMarkers(doctrineIdentityGenesis(enabled), depth))

					var target *types.Fact
					if kind == "challenge" || kind == "challenge/records" {
						if category == types.DoctrineCategory {
							target = mustGetFact(t, k, ctx, "commitment-1") // genesis doctrine
						} else {
							target = makeTestFact(t, k, ctx, "fact-ordinary", "An ordinary verified fact", "physics", category, "zrn1author", 800_000)
						}
					} else {
						// A conjecture's fact, labelled by its proposer.
						target = &types.Fact{
							Id: "fact-conjecture", ClaimId: "claim-conjecture", Content: "An open conjecture",
							Domain: "physics", Category: category, Submitter: "zrn1author",
							Status: types.FactStatus_FACT_STATUS_PROVISIONAL, ClaimType: types.ClaimType_CLAIM_TYPE_CONJECTURE,
						}
						require.NoError(t, k.SetFact(ctx, target))
					}

					claim := openChallenge(t, k, ctx, bk, kind, target)
					want := category
					if enabled && category == types.DoctrineCategory {
						want = ""
					}
					require.Equal(t, want, claim.Category)
					require.Equal(t, target.Id, claim.ProvisionalFactId)

					round, found := k.GetVerificationRound(ctx, claim.VerificationRoundId)
					require.True(t, found)
					require.NoError(t, k.CompleteRound(ctx, round, &keeper.VerificationResult{
						Verdict:    types.Verdict_VERDICT_ACCEPT,
						Confidence: 900_000,
					}))
					var made *types.Fact
					k.IterateFacts(ctx, func(f *types.Fact) bool {
						if f.ClaimId == claim.Id {
							made = f
							return true
						}
						return false
					})
					require.NotNil(t, made, "the accepted challenge made a fact")
					require.Equal(t, want, made.Category)
					if enabled {
						require.NotEqual(t, types.DoctrineCategory, made.Category, "an accepted challenge never makes a fact labelled doctrine")
					}
				})
			}
		}
	}
}
