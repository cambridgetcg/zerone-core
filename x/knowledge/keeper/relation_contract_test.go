package keeper_test

import (
	"bytes"
	"fmt"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Characterization originating at 7a68ca51, NOT a repair or approval of
// ambiguous admissions. Exercise both absent and enabled knowledge-v12 marker
// 0x87 explicitly; DefaultGenesis now selects v12. Retained augmentations are
// admitted before review neutrality, then finalized under each selection.
// These are in-memory handler/phase/query tests with actual SDK stores. The
// tracking bank records calls, not actual transfers or conservation of custody.
func relationContractFixture(t *testing.T, genesis *types.GenesisState) (keeper.Keeper, sdk.Context, *trackingBankKeeper, *storetypes.KVStoreKey) {
	t.Helper()
	db := dbm.NewMemDB()
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	ms.SetIAVLSyncPruning(true)
	key := storetypes.NewKVStoreKey(types.StoreKey)
	ms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())
	ctx := sdk.NewContext(ms, cmtproto.Header{ChainID: "relation-contract-test", Height: 100}, false, log.NewNopLogger())
	bank := newTrackingBankKeeper()
	k := keeper.NewKeeper(runtime.NewKVStoreService(key), codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), "authority", bank, newTrackingStakingKeeper())
	require.NoError(t, k.InitGenesis(ctx, genesis))
	require.NoError(t, k.SetDomain(ctx, &types.Domain{Name: "physics", Status: types.DomainStatus_DOMAIN_STATUS_ACTIVE}))
	for _, id := range []string{"target-a", "target-b"} {
		require.NoError(t, k.SetFact(ctx, &types.Fact{Id: id, Content: "Existing scoped target observation.", Domain: "physics", Category: "empirical", Status: types.FactStatus_FACT_STATUS_VERIFIED}))
	}
	return k, ctx.WithEventManager(sdk.NewEventManager()), bank, key
}

func relationContractSnapshot(t *testing.T, ctx sdk.Context, key *storetypes.KVStoreKey) map[string][]byte {
	t.Helper()
	rows := make(map[string][]byte)
	it := ctx.KVStore(key).Iterator(nil, nil)
	defer func() { require.NoError(t, it.Close()) }()
	for ; it.Valid(); it.Next() {
		rows[string(it.Key())] = bytes.Clone(it.Value())
	}
	return rows
}

// Admit the real input, commit/reveal scheme-2 ballots and advance real phases.
// No direct SetClaim, SetRound, SetFactRelation or supplied verdict fabricates
// the relationship under test. Seed Facts above are explicit fixture prestate.
func completeRelationClaim(t *testing.T, k keeper.Keeper, ctx sdk.Context, bank *trackingBankKeeper, relations []*types.ClaimRelation) (sdk.Context, string) {
	t.Helper()
	server := keeper.NewMsgServerImpl(k)
	response, err := server.SubmitClaim(ctx, &types.MsgSubmitClaim{
		Submitter:   makeValidBech32Addr("relation-contract-author"),
		FactContent: "A bounded observation with explicitly selected target relations.",
		Domain:      "physics", Category: "empirical", Stake: "1000000", Relations: relations,
	})
	require.NoError(t, err)
	claim, found := k.GetClaim(ctx, response.ClaimId)
	require.True(t, found)
	round, found := k.GetVerificationRound(ctx, claim.VerificationRoundId)
	require.True(t, found)
	require.EqualValues(t, types.CommitmentSchemeReviewV2, round.CommitmentScheme)
	require.EqualValues(t, types.ReviewPolicyNeutral, round.ReviewPolicyVersion)
	attestation := &types.ReviewAttestation{Reason: "Synthetic fixture check of the stated observation only."}
	for i := 0; i < 3; i++ {
		verifier := makeValidBech32Addr(fmt.Sprintf("relation-reviewer-%d", i))
		bank.balances[verifier] = sdk.NewCoins(sdk.NewInt64Coin("uzrn", 100_000_000))
		salt := []byte(fmt.Sprintf("relation-salt-sixteen-bytes-%d", i))
		hash, err := types.ComputeReviewCommitmentV2(round.CommitmentChainId, round.Id, verifier, "accept", 800_000, salt, attestation)
		require.NoError(t, err)
		_, err = server.SubmitCommitment(ctx, &types.MsgSubmitCommitment{Verifier: verifier, RoundId: round.Id, CommitHash: hash})
		require.NoError(t, err)
	}
	ctx = ctx.WithBlockHeight(int64(round.CommitDeadline + 1))
	require.NoError(t, k.AdvanceRoundPhases(ctx))
	for i := 0; i < 3; i++ {
		_, err = server.SubmitReveal(ctx, &types.MsgSubmitReveal{
			Verifier: makeValidBech32Addr(fmt.Sprintf("relation-reviewer-%d", i)), RoundId: round.Id,
			Vote: "accept", Confidence: 800_000, Salt: []byte(fmt.Sprintf("relation-salt-sixteen-bytes-%d", i)), Attestation: attestation,
		})
		require.NoError(t, err)
	}
	ctx = ctx.WithBlockHeight(int64(round.RevealDeadline + 1))
	require.NoError(t, k.AdvanceRoundPhases(ctx))
	finished, found := k.GetVerificationRound(ctx, round.Id)
	require.True(t, found)
	require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_COMPLETE, finished.Phase)
	require.Equal(t, types.Verdict_VERDICT_ACCEPT, finished.Verdict)
	return ctx, claim.Id
}

func TestRelationContractRetainedAugmentationEndpointMismatch(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("marker-0x87-%t", enabled), func(t *testing.T) {
			relationContractRetainedAugmentationEndpointMismatch(t, enabled)
		})
	}
}

func requireRelationContractMarker(t *testing.T, k keeper.Keeper, ctx sdk.Context, key *storetypes.KVStoreKey, enabled bool) {
	t.Helper()
	require.Equal(t, []byte{0x87}, []byte(keeper.ClaimIntegrityEnabledStoreKey))
	selected, err := k.ClaimIntegrityEnabled(ctx)
	require.NoError(t, err)
	require.Equal(t, enabled, selected)
	if enabled {
		require.Equal(t, []byte{1}, ctx.KVStore(key).Get([]byte{0x87}))
	} else {
		require.Nil(t, ctx.KVStore(key).Get([]byte{0x87}))
	}
}

func relationContractRetainedAugmentationEndpointMismatch(t *testing.T, enabled bool) {
	for _, tc := range []struct {
		name              string
		verdict           types.AugmentationVerdict
		funded, collision bool
		payout, remaining string
	}{
		{"volunteer-equivalent", types.AugmentationVerdict_AUGMENTATION_VERDICT_EQUIVALENT, false, false, "", ""},
		{"volunteer-superior", types.AugmentationVerdict_AUGMENTATION_VERDICT_SUPERIOR, false, false, "", ""},
		{"bounty-equivalent", types.AugmentationVerdict_AUGMENTATION_VERDICT_EQUIVALENT, true, false, "1000", "500"},
		{"bounty-superior", types.AugmentationVerdict_AUGMENTATION_VERDICT_SUPERIOR, true, false, "1500", "0"},
		{"existing-fact-id-collision", types.AugmentationVerdict_AUGMENTATION_VERDICT_EQUIVALENT, false, true, "", ""},
		{"drift-negative-control", types.AugmentationVerdict_AUGMENTATION_VERDICT_DRIFT, false, false, "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			genesis := types.DefaultGenesis()
			genesis.RecordIntegrityEnabled = false
			genesis.ReviewNeutralityEnabled = false
			genesis.ClaimRecordsEnabled = false
			genesis.FundSettlementEnabled = false
			genesis.ClaimIntegrityEnabled = false // Predecessor claim semantics.
			k, ctx, bank, key := relationContractFixture(t, genesis)
			qualification := newMockDomainQualificationKeeper()
			voters := []string{makeValidBech32Addr("aug-voter-1"), makeValidBech32Addr("aug-voter-2"), makeValidBech32Addr("aug-voter-3")}
			for _, voter := range voters {
				qualification.qualify(voter, "physics")
			}
			k.SetDomainQualificationKeeper(qualification)
			server := keeper.NewMsgServerImpl(k)
			submitter := makeValidBech32Addr("retained-augmentation-author")
			augID, bountyID := "pending-variant", ""
			if tc.collision {
				augID = "target-b"
				// Explicit genuine Fact-to-Fact fixture prestate, not an augmentation edge.
				require.NoError(t, k.SetFactRelation(ctx, &types.FactRelation{SourceFactId: augID, TargetFactId: "target-a", Relation: types.RelationType_RELATION_TYPE_SUPPORTS}))
			}
			ownEdge := bytes.Clone(ctx.KVStore(key).Get(types.FactRelationKey(augID, "target-a")))
			if tc.funded {
				bountyID = "retained-bounty"
				_, err := server.CreateAugmentationBounty(ctx, &types.MsgCreateAugmentationBounty{
					Id: bountyID, Sponsor: makeValidBech32Addr("retained-bounty-sponsor"), TargetFactId: "target-a",
					RewardPerVariant: 1000, MaxVariants: 1, ExpiresAtBlock: 200,
				})
				require.NoError(t, err)
				// This tracking mock records account-to-module sends without
				// applying them. Supply the declared backing explicitly; FX7
				// must not allow the later export to certify an unbacked ledger.
				bank.moduleBalances[types.TrainingFundModuleName] = sdk.NewCoins(
					sdk.NewCoin("uzrn", k.GetEscrowLocked(ctx, bountyID)),
				)
			}
			_, err := server.SubmitAugmentation(ctx, &types.MsgSubmitAugmentation{
				Id: augID, BountyId: bountyID, OriginalFactId: "target-a", VariantContent: "A retained reformulation fixture.", Submitter: submitter,
			})
			require.NoError(t, err)
			_, err = server.VoteOnAugmentation(ctx, &types.MsgVoteOnAugmentation{AugmentationId: augID, Verifier: voters[0], Vote: tc.verdict})
			require.NoError(t, err)
			before, found := k.GetAugmentation(ctx, augID)
			require.True(t, found)
			require.Zero(t, before.VerdictBlock)
			require.NoError(t, k.EnableRecordIntegrity(ctx))
			require.NoError(t, k.EnableReviewNeutrality(ctx)) // In-memory retirement selection, not an upgrade rehearsal.
			require.NoError(t, k.EnableClaimRecords(ctx))
			require.NoError(t, k.EnableFundSettlement(ctx))
			if enabled {
				require.NoError(t, k.EnableClaimIntegrity(ctx))
			}
			requireRelationContractMarker(t, k, ctx, key, enabled)
			require.NotPanics(t, func() { k.ExportGenesis(ctx) }, "the pending retained record alone does not spoil export")
			callsBefore := len(bank.sendCalls)
			for _, voter := range voters[1:] {
				_, err = server.VoteOnAugmentation(ctx, &types.MsgVoteOnAugmentation{AugmentationId: augID, Verifier: voter, Vote: tc.verdict})
				require.NoError(t, err)
			}
			after, found := k.GetAugmentation(ctx, augID)
			require.True(t, found)
			require.Equal(t, tc.verdict, after.Verdict)
			require.Equal(t, uint64(ctx.BlockHeight()), after.VerdictBlock)
			require.Equal(t, before.VerdictVoters, after.VerdictVoters[:1])
			require.Equal(t, before.VerdictVoteStakes, after.VerdictVoteStakes[:1])
			require.Equal(t, before.VerdictVoteCalibrationBps, after.VerdictVoteCalibrationBps[:1])
			require.Equal(t, tc.payout, after.PayoutAmount)
			passing := tc.verdict != types.AugmentationVerdict_AUGMENTATION_VERDICT_DRIFT
			require.Equal(t, passing, after.Accepted)
			require.Empty(t, qualification.outcomes, "retained finalization does not reopen retired score feedback")
			if tc.funded {
				require.Equal(t, tc.remaining, k.GetEscrowLocked(ctx, bountyID).String())
				require.Len(t, bank.sendCalls, callsBefore+1)
				require.Equal(t, submitter, bank.sendCalls[callsBefore].to)
				require.Equal(t, tc.payout+"uzrn", bank.sendCalls[callsBefore].coins.String())
			} else {
				require.Len(t, bank.sendCalls, callsBefore)
			}
			stored := relationContractSnapshot(t, ctx, key)
			projection, err := keeper.NewQueryServerImpl(k).AugmentationsByFact(ctx, &types.QueryAugmentationsByFactRequest{FactId: "target-a"})
			require.NoError(t, err)
			require.Len(t, projection.Augmentations, 1)
			require.True(t, proto.Equal(after, projection.Augmentations[0]), "augmentation history retains its own non-Fact owner")
			rels, err := k.GetFactRelations(ctx, augID)
			require.NoError(t, err)
			if passing {
				require.Len(t, rels, 1)
				wantRelation := types.RelationType_RELATION_TYPE_REFORMULATES
				if enabled && tc.collision {
					wantRelation = types.RelationType_RELATION_TYPE_SUPPORTS
				}
				require.Equal(t, wantRelation, rels[0].Relation)
				require.Equal(t, "target-a", rels[0].TargetFactId)
				require.Equal(t, augID, rels[0].SourceFactId)
				forward := ctx.KVStore(key).Get(types.FactRelationKey(augID, "target-a"))
				require.NotEmpty(t, forward)
				require.Equal(t, forward, ctx.KVStore(key).Get(types.FactRelationReverseKey("target-a", augID)))
				if enabled && tc.collision {
					require.Equal(t, ownEdge, forward, "v12 withholds the colliding augmentation edge and preserves the genuine Fact edge byte-for-byte")
				}
			} else {
				require.Empty(t, rels)
			}
			withheld := v11Event(ctx, "zerone.knowledge.augmentation_edge_withheld")
			if enabled && tc.collision {
				require.Equal(t, []map[string]string{{"augmentation_id": augID, "original_fact_id": "target-a", "reason": "augmentation id is the id of a fact"}}, withheld)
			} else {
				require.Empty(t, withheld)
			}
			_, factExists := k.GetFact(ctx, augID)
			require.Equal(t, tc.collision, factExists)
			exported, err := k.ExportFactRelationGenesis(ctx)
			if passing && !tc.collision {
				require.ErrorContains(t, err, "fact relation references an absent Fact")
				require.Nil(t, exported)
				require.PanicsWithError(t, "export fact relations: fact relation references an absent Fact", func() { k.ExportGenesis(ctx) })
				t.Logf("marker=%t: retained augmentation %s wrote a non-Fact edge; full genesis export panics: %v", enabled, augID, err)
			} else {
				require.NoError(t, err)
				require.NotPanics(t, func() { k.ExportGenesis(ctx) })
			}
			require.Equal(t, stored, relationContractSnapshot(t, ctx, key), "export refusal must not silently filter or repair stored history")
		})
	}
}

func TestRelationContractPredecessorCountBound(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("marker-0x87-%t", enabled), func(t *testing.T) {
			relationContractCountBound(t, enabled)
		})
	}
}

func relationContractCountBound(t *testing.T, enabled bool) {
	for _, count := range []int{16, 17} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			genesis := types.DefaultGenesis()
			genesis.ClaimIntegrityEnabled = enabled
			k, ctx, bank, key := relationContractFixture(t, genesis)
			requireRelationContractMarker(t, k, ctx, key, enabled)
			relations := make([]*types.ClaimRelation, count)
			for i := range relations {
				relations[i] = &types.ClaimRelation{TargetFactId: "target-a", Relation: types.RelationType_RELATION_TYPE_CITES}
			}
			before := relationContractSnapshot(t, ctx, key)
			calls := len(bank.sendCalls)
			response, err := keeper.NewMsgServerImpl(k).SubmitClaim(ctx, &types.MsgSubmitClaim{
				Submitter: makeValidBech32Addr("bounded-relations-author"), FactContent: "A bounded relation admission fixture.",
				Domain: "physics", Category: "empirical", Stake: "1000000", Relations: relations,
			})
			if count == 16 {
				require.NoError(t, err, "both selections cap count, not target uniqueness")
				claim, found := k.GetClaim(ctx, response.ClaimId)
				require.True(t, found)
				require.Len(t, claim.Relations, count)
			} else {
				require.ErrorIs(t, err, types.ErrInvalidClaim)
				require.ErrorContains(t, err, "claim carries 17 relations, max 16")
				require.Nil(t, response)
				require.Equal(t, before, relationContractSnapshot(t, ctx, key))
				require.Len(t, bank.sendCalls, calls)
				require.Empty(t, ctx.EventManager().Events())
			}
		})
	}
}

func TestRelationContractPredecessorLastPairWriteSurvives(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("marker-0x87-%t", enabled), func(t *testing.T) {
			relationContractLastPairWriteSurvives(t, enabled)
		})
	}
}

func relationContractLastPairWriteSurvives(t *testing.T, enabled bool) {
	for _, tc := range []struct {
		name         string
		first, last  types.RelationType
		secondTarget string
		wantEdges    int
	}{
		{"distinct-target-control", types.RelationType_RELATION_TYPE_SUPPORTS, types.RelationType_RELATION_TYPE_CITES, "target-b", 2},
		{"supports-then-cites", types.RelationType_RELATION_TYPE_SUPPORTS, types.RelationType_RELATION_TYPE_CITES, "target-a", 1},
		{"cites-then-supports", types.RelationType_RELATION_TYPE_CITES, types.RelationType_RELATION_TYPE_SUPPORTS, "target-a", 1},
		{"same-type-last-metadata", types.RelationType_RELATION_TYPE_SUPPORTS, types.RelationType_RELATION_TYPE_SUPPORTS, "target-a", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			genesis := types.DefaultGenesis()
			genesis.ClaimIntegrityEnabled = enabled
			k, ctx, bank, key := relationContractFixture(t, genesis)
			requireRelationContractMarker(t, k, ctx, key, enabled)
			inputs := []*types.ClaimRelation{
				{TargetFactId: "target-a", Relation: tc.first, InferenceStrengthBps: 100_000},
				{TargetFactId: tc.secondTarget, Relation: tc.last, InferenceStrengthBps: 200_000},
			}
			ctx, claimID := completeRelationClaim(t, k, ctx, bank, inputs)
			before := relationContractSnapshot(t, ctx, key)
			history, err := keeper.NewQueryServerImpl(k).ClaimHistory(ctx, &types.QueryClaimHistoryRequest{Id: claimID})
			require.NoError(t, err)
			require.Equal(t, types.ClaimStatus_CLAIM_STATUS_ACCEPTED, history.Record.Claim.Status)
			require.Len(t, history.Record.Claim.Relations, len(inputs))
			for i, input := range inputs {
				require.True(t, proto.Equal(input, history.Record.Claim.Relations[i]), "input history remains ordered and complete")
			}
			require.Len(t, history.Record.Facts, 1)
			derived := history.Record.Facts[0]
			require.Len(t, derived.OutgoingRelations, tc.wantEdges)
			expected := map[string]*types.ClaimRelation{}
			for _, input := range inputs {
				expected[input.TargetFactId] = input
			}
			for _, rel := range derived.OutgoingRelations {
				want, ok := expected[rel.TargetFactId]
				require.True(t, ok, "only explicitly proposed targets survive")
				require.Equal(t, derived.Fact.Id, rel.SourceFactId)
				require.Equal(t, want.Relation, rel.Relation)
				require.Equal(t, want.InferenceStrengthBps, rel.InferenceStrengthBps)
				forward := ctx.KVStore(key).Get(types.FactRelationKey(rel.SourceFactId, rel.TargetFactId))
				reverse := ctx.KVStore(key).Get(types.FactRelationReverseKey(rel.TargetFactId, rel.SourceFactId))
				require.NotEmpty(t, forward)
				require.Equal(t, forward, reverse)
			}
			bundle, err := k.AssembleToKBundle(ctx, &types.ToKSelector{Variant: &types.ToKSelector_AncestorCone{
				AncestorCone: &types.AncestorConeSelector{LeafFactId: derived.Fact.Id},
			}}, uint64(ctx.BlockHeight()))
			require.NoError(t, err)
			require.Len(t, bundle.IncludedEdges, tc.wantEdges, "ToK reads surviving canonical pairs, not Claim inputs")
			wantNodes := []string{derived.Fact.Id}
			for target := range expected {
				wantNodes = append(wantNodes, target)
			}
			require.ElementsMatch(t, wantNodes, bundle.IncludedNodeIds)
			for _, edge := range bundle.IncludedEdges {
				want, ok := expected[edge.ToFactId]
				require.True(t, ok)
				require.Equal(t, derived.Fact.Id, edge.FromFactId)
				require.Equal(t, want.Relation.String(), edge.Relation)
			}
			exported, err := k.ExportFactRelationGenesis(ctx)
			require.NoError(t, err, "valid endpoints do not reveal that a proposed edge was overwritten")
			selected := make(map[string]*types.FactRelation)
			for _, rel := range exported.Relations {
				if rel.SourceFactId == derived.Fact.Id {
					require.NotContains(t, selected, rel.TargetFactId)
					selected[rel.TargetFactId] = rel
				}
			}
			require.Len(t, selected, tc.wantEdges)
			for _, rel := range derived.OutgoingRelations {
				require.True(t, proto.Equal(rel, selected[rel.TargetFactId]), "export must retain the same surviving endpoints, type and metadata as history")
			}
			require.Equal(t, before, relationContractSnapshot(t, ctx, key), "read projections must not reconstruct missing edges")
		})
	}
}
