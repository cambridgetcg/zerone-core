package keeper_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"testing"

	sdkquery "github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func TestReadCanonicalProjectionWire(t *testing.T) {
	k, ctx := setupKnowledgeWithFacts(t, []factSpec{{id: "fact-a", domain: "general"}, {id: "fact-b", domain: "general"}})
	ctx = ctx.WithBlockHeight(1000).WithChainID("zerone-1")
	for _, id := range []string{"fact-a", "fact-b"} {
		f, _ := k.GetFact(ctx, id)
		f.Content = "Canonical knowledge " + id
		f.Confidence = 880000
		f.VerifiedAtBlock = 800
		f.LastVerifiedBlock = 800
		f.ClaimType = types.ClaimType_CLAIM_TYPE_ASSERTION
		require.NoError(t, k.SetFact(ctx, f))
	}
	rel := &types.FactRelation{SourceFactId: "fact-b", TargetFactId: "fact-a", Relation: types.RelationType_RELATION_TYPE_SUPPORTS, Inference: types.InferenceType_INFERENCE_TYPE_EMPIRICAL, InferenceStrengthBps: 750000, CreatedAtBlock: 900, MethodId: "M-EMPIRICAL"}
	require.NoError(t, k.SetFactRelation(ctx, rel))
	before, _ := k.GetFact(ctx, "fact-a")
	q := keeper.NewQueryServerImpl(*k)
	point, err := q.Fact(ctx, &types.QueryFactRequest{Id: "fact-a", TrackQuery: true, Querier: "legacy-querier"})
	require.NoError(t, err)
	require.False(t, k.HasQueryReceipt(ctx, "legacy-querier", "fact-a"))
	require.True(t, proto.Equal(rel, point.Fact.IncomingRelations[0]))
	after, _ := k.GetFact(ctx, "fact-a")
	require.True(t, proto.Equal(before, after), "query must not change counters or persisted relation arrays")
	response, err := q.Facts(ctx, &types.QueryFactsRequest{Pagination: &sdkquery.PageRequest{Limit: 100}})
	require.NoError(t, err)
	// The normal test genesis also contains doctrine; do not erase it to make
	// the query look like a two-record hand fixture.
	projected := map[string]*types.Fact{}
	for _, fact := range response.Facts {
		projected[fact.Id] = fact
	}
	require.True(t, proto.Equal(rel, projected["fact-a"].IncomingRelations[0]))
	require.True(t, proto.Equal(rel, projected["fact-b"].OutgoingRelations[0]))
	wire, err := (protojson.MarshalOptions{EmitUnpopulated: true}).Marshal(response)
	require.NoError(t, err)
	// Consumed by the dashboard integration test, not a hand-authored fixture.
	if os.Getenv("TOK_READ_WIRE") == "1" {
		fmt.Println("TOK_READ_WIRE=" + base64.StdEncoding.EncodeToString(wire))
		var ids []string
		var edges []*types.ToKEdge
		seen := map[string]bool{}
		for _, fact := range response.Facts {
			ids = append(ids, fact.Id)
			for _, relation := range append(fact.OutgoingRelations, fact.IncomingRelations...) {
				key := relation.SourceFactId + "/" + relation.TargetFactId
				if seen[key] {
					continue
				}
				seen[key] = true
				edges = append(edges, &types.ToKEdge{FromFactId: relation.SourceFactId, ToFactId: relation.TargetFactId, Relation: relation.Relation.String(), Inference: relation.Inference.String()})
			}
		}
		fmt.Printf("TOK_PROJECTION_ROOT=%x\n", keeper.ComputeToKSnapshotRoot(ids, edges))
	}
}

func TestReadBudgets(t *testing.T) {
	t.Run("nodes", func(t *testing.T) {
		specs := []factSpec{{id: "root"}}
		for i := 0; i < 128; i++ {
			specs = append(specs, factSpec{id: fmt.Sprintf("child-%03d", i), supports: []string{"root"}})
		}
		k, ctx := setupKnowledgeWithFacts(t, specs)
		q := keeper.NewQueryServerImpl(*k)
		r, err := q.BundleToK(ctx, &types.QueryBundleToKRequest{Selector: &types.ToKSelector{Variant: &types.ToKSelector_RootedSubtree{RootedSubtree: &types.RootedSubtreeSelector{RootFactId: "root"}}}})
		require.Nil(t, r)
		require.Equal(t, codes.ResourceExhausted, status.Code(err))
	})
	t.Run("edges", func(t *testing.T) {
		var specs []factSpec
		for i := 0; i < 24; i++ {
			specs = append(specs, factSpec{id: fmt.Sprintf("n-%02d", i), domain: "dense"})
		}
		k, ctx := setupKnowledgeWithFacts(t, specs)
		for _, a := range specs {
			for _, b := range specs {
				if a.id != b.id {
					require.NoError(t, k.SetFactRelation(ctx, &types.FactRelation{SourceFactId: a.id, TargetFactId: b.id, Relation: types.RelationType_RELATION_TYPE_SUPPORTS}))
				}
			}
		}
		_, _, err := k.GatherRootedSubtree(ctx, &types.RootedSubtreeSelector{RootFactId: "n-00", MaxDepth: 32})
		require.Equal(t, codes.ResourceExhausted, status.Code(err))
	})
	t.Run("filtered ghost edges count before following", func(t *testing.T) {
		k, ctx := setupKnowledgeWithFacts(t, []factSpec{{id: "root"}})
		for i := 0; i < 1025; i++ {
			require.NoError(t, k.SetFactRelation(ctx, &types.FactRelation{SourceFactId: fmt.Sprintf("ghost-%04d", i), TargetFactId: "root", Relation: types.RelationType_RELATION_TYPE_CONTRADICTS}))
		}
		_, _, err := k.GatherRootedSubtree(ctx, &types.RootedSubtreeSelector{RootFactId: "root", MaxDepth: 1})
		require.Equal(t, codes.ResourceExhausted, status.Code(err))
	})
	t.Run("sparse frontier", func(t *testing.T) {
		var specs []factSpec
		for i := 0; i < 1030; i++ {
			specs = append(specs, factSpec{id: fmt.Sprintf("f-%04d", i), domain: "sparse"})
		}
		k, ctx := setupKnowledgeWithFacts(t, specs)
		_, _, err := k.GatherFrontier(ctx, &types.FrontierSelector{Domain: "sparse", Limit: 1})
		require.Equal(t, codes.ResourceExhausted, status.Code(err))
		r, err := keeper.NewQueryServerImpl(*k).Facts(ctx, &types.QueryFactsRequest{Status: "FACT_STATUS_DISPROVEN"})
		require.Nil(t, r)
		require.Equal(t, codes.ResourceExhausted, status.Code(err))
	})
	t.Run("output and cancellation and synthetic height", func(t *testing.T) {
		k, ctx := setupKnowledgeWithFacts(t, []factSpec{{id: "root"}})
		ctx = ctx.WithBlockHeight(777)
		q := keeper.NewQueryServerImpl(*k)
		req := &types.QueryBundleToKRequest{Selector: &types.ToKSelector{Variant: &types.ToKSelector_RootedSubtree{RootedSubtree: &types.RootedSubtreeSelector{RootFactId: "root"}}}}
		req.AtBlockHeight = 777
		_, err := q.BundleToK(ctx, req)
		require.Equal(t, codes.InvalidArgument, status.Code(err))
		req.AtBlockHeight = 0
		result, err := q.BundleToK(ctx, req)
		require.NoError(t, err)
		require.Equal(t, uint64(777), result.Bundle.SnapshotBlock)
		canceled, cancel := context.WithCancel(ctx)
		cancel()
		_, err = q.BundleToK(canceled, req)
		require.Equal(t, codes.Canceled, status.Code(err))
		_, err = q.Fact(canceled, &types.QueryFactRequest{Id: "root"})
		require.Equal(t, codes.Canceled, status.Code(err))
		f, _ := k.GetFact(ctx, "root")
		f.Content = strings.Repeat("x", keeper.ToKReadMaxBytes)
		require.NoError(t, k.SetFact(ctx, f))
		_, err = q.Fact(ctx, &types.QueryFactRequest{Id: "root"})
		require.Equal(t, codes.ResourceExhausted, status.Code(err))
	})
}

func TestReadCascadeHistoryIsExactAndBounded(t *testing.T) {
	k, ctx := setupKnowledgeWithFacts(t, nil)
	ctx = ctx.WithBlockHeight(2000)
	fact := &types.Fact{Id: "root", Status: types.FactStatus_FACT_STATUS_DISPROVEN}
	require.NoError(t, k.SetFactSkipTransition(ctx, fact))
	selector := &types.ToKSelector{Variant: &types.ToKSelector_CascadeReplay{CascadeReplay: &types.CascadeReplaySelector{DisprovenFactId: "root", IncludeStatusHistory: true}}}
	bundle, err := k.AssembleToKBundle(ctx, selector, 0)
	require.NoError(t, err)
	require.Empty(t, bundle.StatusHistory, "missing history must not be synthesized")
	for i := 0; i < 3; i++ {
		require.NoError(t, k.RecordStatusTransition(ctx, &types.StatusTransition{FactId: "root", PriorStatus: types.FactStatus_FACT_STATUS_VERIFIED, NewStatus: types.FactStatus_FACT_STATUS_DISPROVEN, BlockHeight: uint64(17 + i*400), CauseEventType: "reviewed-change", CauseId: fmt.Sprintf("claim-%d", i)}))
	}
	bundle, err = k.AssembleToKBundle(ctx, selector, 0)
	require.NoError(t, err)
	actual := k.GetStatusHistory(ctx, "root")
	require.Len(t, bundle.StatusHistory, len(actual))
	for i := range actual {
		require.True(t, proto.Equal(actual[i], bundle.StatusHistory[i]))
	}
	require.Equal(t, keeper.ComputeToKSnapshotRootV2(bundle.IncludedNodeIds, bundle.IncludedEdges, bundle.CascadeEvents, bundle.Vindications, actual), bundle.SnapshotRoot)
	for i := 3; i < 1030; i++ {
		require.NoError(t, k.RecordStatusTransition(ctx, &types.StatusTransition{FactId: "root", PriorStatus: types.FactStatus_FACT_STATUS_VERIFIED, NewStatus: types.FactStatus_FACT_STATUS_DISPROVEN, BlockHeight: 1500, CauseId: "history-cap"}))
	}
	response, err := keeper.NewQueryServerImpl(*k).BundleToK(ctx, &types.QueryBundleToKRequest{Selector: selector})
	require.Nil(t, response)
	require.Equal(t, codes.ResourceExhausted, status.Code(err))
}

func TestReadIteratorCloseFailureIsNotSuccess(t *testing.T) {
	k, ctx, store := setupFeedbackStore(t)
	feedbackFact(t, k, ctx, "root")
	store.closeError = true
	q := keeper.NewQueryServerImpl(k)
	point, err := q.Fact(ctx, &types.QueryFactRequest{Id: "root"})
	require.Nil(t, point)
	require.Equal(t, codes.Internal, status.Code(err))
	require.Contains(t, err.Error(), "injected feedback store failure")
	facts, err := q.Facts(ctx, &types.QueryFactsRequest{Pagination: &sdkquery.PageRequest{Limit: 1}})
	require.Nil(t, facts)
	require.Equal(t, codes.Internal, status.Code(err))
	require.Zero(t, store.open)
}

func TestReadZeroConfidenceMinimum(t *testing.T) {
	k, ctx := setupKnowledgeWithFacts(t, []factSpec{{id: "zero"}, {id: "root", supports: []string{"zero"}}})
	f, _ := k.GetFact(ctx, "root")
	f.Confidence = 880000
	f.AxiomDistance = 1
	require.NoError(t, k.SetFact(ctx, f))
	q := keeper.NewQueryServerImpl(*k)
	proof, err := q.ProofTree(ctx, &types.QueryProofTreeRequest{FactId: "root", IncludeAxioms: true})
	require.NoError(t, err)
	require.Zero(t, proof.MinimumConfidenceInTree)
	trust, err := q.TrustProfile(ctx, &types.QueryTrustProfileRequest{FactId: "root"})
	require.NoError(t, err)
	require.Zero(t, trust.MinimumConfidenceInAncestry)
}
