package keeper

import (
	"context"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	"github.com/zerone-chain/zerone/x/training_provenance/types"
)

// reviewKnowledgeStub serves manifests, facts, claim-indexed rounds and
// privileged actions from maps. It reads only; nothing here adjudicates.
type reviewKnowledgeStub struct {
	manifests  map[string]*knowledgetypes.TrainingManifest
	facts      map[string]*knowledgetypes.Fact
	rounds     map[string]*knowledgetypes.VerificationRound
	privileged []*knowledgetypes.PrivilegedAction
}

func (s *reviewKnowledgeStub) GetTrainingManifest(_ context.Context, id string) (*knowledgetypes.TrainingManifest, bool) {
	manifest, ok := s.manifests[id]
	return manifest, ok
}

func (s *reviewKnowledgeStub) GetFact(_ context.Context, id string) (*knowledgetypes.Fact, bool) {
	fact, ok := s.facts[id]
	return fact, ok
}

func (s *reviewKnowledgeStub) IteratePrivilegedActions(_ context.Context, cb func(*knowledgetypes.PrivilegedAction) bool) {
	for _, action := range s.privileged {
		if cb(action) {
			return
		}
	}
}

func (*reviewKnowledgeStub) IterateIncidents(context.Context, func(*knowledgetypes.IncidentRecord) bool) {
}

func (s *reviewKnowledgeStub) GetRoundByClaimID(_ context.Context, claimID string) (*knowledgetypes.VerificationRound, bool) {
	round, ok := s.rounds[claimID]
	return round, ok
}

func revealed(claimID string) *knowledgetypes.VerificationRound {
	return &knowledgetypes.VerificationRound{
		Id:      "round-" + claimID,
		ClaimId: claimID,
		Phase:   knowledgetypes.VerificationPhase_VERIFICATION_PHASE_COMPLETE,
		Reveals: []*knowledgetypes.RevealEntry{{Verifier: "reviewer-1", Vote: "accept"}},
	}
}

// reviewFixture holds one fact for every branch of the counting rule.
func reviewFixture() *reviewKnowledgeStub {
	return &reviewKnowledgeStub{
		manifests: map[string]*knowledgetypes.TrainingManifest{},
		facts: map[string]*knowledgetypes.Fact{
			"reviewed":   {Id: "reviewed", Domain: "physics", ClaimId: "claim-reviewed"},
			"reviewed-2": {Id: "reviewed-2", Domain: "physics", ClaimId: "claim-reviewed-2"},
			"doctrine": knowledgetypes.BuildDoctrineFact(
				"doctrine", knowledgetypes.DoctrineDomainTruthSeeking, "a commitment"),
			"doctrine-category": {Id: "doctrine-category", Domain: "doctrine_tok", Category: knowledgetypes.DoctrineCategory},
			"doctrine-method":   {Id: "doctrine-method", Domain: "doctrine_tok", MethodId: knowledgetypes.DoctrineMethodId},
			"genesis-plain":     {Id: "genesis-plain", Domain: "biology", Category: "empirical"},
			"no-round":          {Id: "no-round", Domain: "biology", ClaimId: "claim-no-round"},
			"wrong-claim-round": {Id: "wrong-claim-round", Domain: "biology", ClaimId: "claim-wrong"},
			"no-reveal":         {Id: "no-reveal", Domain: "biology", ClaimId: "claim-no-reveal"},
			"nil-reveal":        {Id: "nil-reveal", Domain: "biology", ClaimId: "claim-nil-reveal"},
			"nil-round":         {Id: "nil-round", Domain: "biology", ClaimId: "claim-nil-round"},
			// A claim can carry the doctrine category label. It is never counted
			// as doctrine; its round decides.
			"claim-labelled-doctrine": {
				Id: "claim-labelled-doctrine", Domain: "doctrine_tok",
				Category: knowledgetypes.DoctrineCategory, ClaimId: "claim-labelled",
			},
			"claim-labelled-doctrine-unrevealed": {
				Id: "claim-labelled-doctrine-unrevealed", Domain: "doctrine_tok",
				Category: knowledgetypes.DoctrineCategory, ClaimId: "claim-labelled-unrevealed",
			},
			"nil-fact": nil,
		},
		rounds: map[string]*knowledgetypes.VerificationRound{
			"claim-reviewed":   revealed("claim-reviewed"),
			"claim-reviewed-2": revealed("claim-reviewed-2"),
			"claim-wrong":      revealed("claim-someone-else"),
			"claim-no-reveal": {
				Id: "round-no-reveal", ClaimId: "claim-no-reveal",
				Phase:   knowledgetypes.VerificationPhase_VERIFICATION_PHASE_EXPIRED,
				Commits: []*knowledgetypes.CommitEntry{{Verifier: "reviewer-1"}},
			},
			"claim-nil-reveal": {
				Id: "round-nil-reveal", ClaimId: "claim-nil-reveal",
				Reveals: []*knowledgetypes.RevealEntry{nil},
			},
			"claim-nil-round": nil,
			"claim-labelled":  revealed("claim-labelled"),
			"claim-labelled-unrevealed": {
				Id: "round-labelled-unrevealed", ClaimId: "claim-labelled-unrevealed",
			},
		},
	}
}

func addFlatManifest(stub *reviewKnowledgeStub, id string, status knowledgetypes.ManifestStatus, factIDs ...string) {
	stub.manifests[id] = &knowledgetypes.TrainingManifest{
		ManifestId:      id,
		PipelineId:      "pipeline-" + id,
		ChainId:         "zerone-origin-1",
		IncludedFactIds: factIDs,
		MerkleRoot: knowledgekeeper.ComputeManifestMerkleRoot(
			knowledgekeeper.SelectedManifestIDs{FactIDs: factIDs}),
		FinalizedAtBlock: 12,
		Status:           status,
	}
}

func newReviewQueryServer(stub *reviewKnowledgeStub) (types.QueryServer, context.Context) {
	k := NewKeeper(nil)
	k.SetKnowledgeKeeper(stub)
	ctx := sdk.WrapSDKContext(
		sdk.Context{}.
			WithChainID("zerone-observer-1").
			WithBlockHeight(77),
	)
	return NewQueryServerImpl(k), ctx
}

func TestReviewCoverageCountingRule(t *testing.T) {
	finalized := knowledgetypes.ManifestStatus_MANIFEST_STATUS_FINALIZED
	for _, tc := range []struct {
		name     string
		factIDs  []string
		coverage reviewCoverage
	}{
		{"retained round with a reveal", []string{"reviewed"}, reviewCoverage{reviewed: 1}},
		{"doctrine built at genesis", []string{"doctrine"}, reviewCoverage{unreviewed: 1, doctrine: 1}},
		{"doctrine category without claim", []string{"doctrine-category"}, reviewCoverage{unreviewed: 1, doctrine: 1}},
		{"doctrine method without claim", []string{"doctrine-method"}, reviewCoverage{unreviewed: 1, doctrine: 1}},
		{"genesis or authority fact without claim", []string{"genesis-plain"}, reviewCoverage{unreviewed: 1}},
		{"claim without a retained round", []string{"no-round"}, reviewCoverage{unreviewed: 1}},
		{"nil round", []string{"nil-round"}, reviewCoverage{unreviewed: 1}},
		{"round names another claim", []string{"wrong-claim-round"}, reviewCoverage{unreviewed: 1}},
		{"round with commits but no reveal", []string{"no-reveal"}, reviewCoverage{unreviewed: 1}},
		{"round with only a nil reveal", []string{"nil-reveal"}, reviewCoverage{unreviewed: 1}},
		{"doctrine label on a reviewed claim", []string{"claim-labelled-doctrine"}, reviewCoverage{reviewed: 1}},
		{"doctrine label on an unrevealed claim", []string{"claim-labelled-doctrine-unrevealed"}, reviewCoverage{unreviewed: 1}},
		{"missing fact ID", []string{"absent"}, reviewCoverage{unreviewed: 1, missing: 1}},
		{"nil fact record", []string{"nil-fact"}, reviewCoverage{unreviewed: 1, missing: 1}},
		{"each entry counts", []string{"reviewed", "reviewed"}, reviewCoverage{reviewed: 2}},
		{"no facts", nil, reviewCoverage{}},
		{
			"every branch together",
			[]string{
				"absent", "claim-labelled-doctrine", "claim-labelled-doctrine-unrevealed",
				"doctrine", "doctrine-category", "doctrine-method", "genesis-plain",
				"nil-fact", "nil-reveal", "nil-round", "no-reveal", "no-round",
				"reviewed", "reviewed-2", "wrong-claim-round",
			},
			reviewCoverage{reviewed: 3, unreviewed: 12, doctrine: 3, missing: 2},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stub := reviewFixture()
			addFlatManifest(stub, "manifest", finalized, tc.factIDs...)
			query, ctx := newReviewQueryServer(stub)

			response, err := query.ProvenanceCertificateV2(ctx,
				&types.QueryProvenanceCertificateV2Request{ManifestId: "manifest"})
			require.NoError(t, err)
			cert := response.Certificate
			require.Equal(t, tc.coverage.reviewed, cert.ReviewedFactCount)
			require.Equal(t, tc.coverage.unreviewed, cert.UnreviewedFactCount)
			require.Equal(t, tc.coverage.doctrine, cert.DoctrineFactCount)
			require.Equal(t, tc.coverage.missing, cert.MissingFactCount)
			require.Equal(t, cert.FactCount, cert.ReviewedFactCount+cert.UnreviewedFactCount)
			state, explanation := computeReviewState(tc.coverage)
			require.Equal(t, state, cert.ReviewState)
			require.Equal(t, explanation, cert.ReviewExplanation)
			// No privileged actions, incidents or cartel resolutions are
			// counted here, so the warning grade is A whatever the review
			// reading says. A is not review.
			require.Equal(t, "A", cert.WarningGrade)
		})
	}
}

func TestCertificateV2KeepsV1ReadingAndAddsReview(t *testing.T) {
	finalized := knowledgetypes.ManifestStatus_MANIFEST_STATUS_FINALIZED
	stub := reviewFixture()
	addFlatManifest(stub, "doctrine-only", finalized, "doctrine", "doctrine-category")
	addFlatManifest(stub, "reviewed-only", finalized, "reviewed")
	addFlatManifest(stub, "mixed", finalized, "reviewed", "doctrine", "absent")
	addFlatManifest(stub, "empty", finalized)
	addFlatManifest(stub, "intervened-reviewed", finalized, "reviewed", "reviewed-2")
	stub.privileged = []*knowledgetypes.PrivilegedAction{nil, {Target: "reviewed-2"}}
	query, ctx := newReviewQueryServer(stub)

	// The two readings move independently: an unreviewed manifest can carry
	// warning grade A, and a reviewed one can carry B.
	for manifestID, want := range map[string]struct {
		warning string
		state   string
	}{
		"doctrine-only":       {"A", ReviewStateUnreviewed},
		"reviewed-only":       {"A", ReviewStateReviewed},
		"mixed":               {"A", ReviewStatePartlyReviewed},
		"empty":               {"A", ReviewStateEmpty},
		"intervened-reviewed": {"B", ReviewStateReviewed},
	} {
		t.Run(manifestID, func(t *testing.T) {
			v1, err := query.ProvenanceCertificate(ctx,
				&types.QueryProvenanceCertificateRequest{ManifestId: manifestID})
			require.NoError(t, err)
			v2, err := query.ProvenanceCertificateV2(ctx,
				&types.QueryProvenanceCertificateV2Request{ManifestId: manifestID})
			require.NoError(t, err)
			a, b := v1.Certificate, v2.Certificate

			// The warning reading is the v1 grade and wording, renamed.
			require.Equal(t, a.TrustGrade, b.WarningGrade)
			require.Equal(t, a.TrustExplanation, b.WarningExplanation)
			require.Equal(t, want.warning, b.WarningGrade)
			require.Equal(t, want.state, b.ReviewState)

			// Every shared field is the v1 value.
			require.Equal(t, a.ManifestId, b.ManifestId)
			require.Equal(t, a.PipelineId, b.PipelineId)
			require.Equal(t, a.MerkleRoot, b.MerkleRoot)
			require.Equal(t, a.FactCount, b.FactCount)
			require.Equal(t, a.FinalizedAtBlock, b.FinalizedAtBlock)
			require.Equal(t, a.Status, b.Status)
			require.Len(t, b.Domains, len(a.Domains))
			for i := range a.Domains {
				require.True(t, proto.Equal(a.Domains[i], b.Domains[i]))
			}
			require.Equal(t, a.PrivilegedActionCount, b.PrivilegedActionCount)
			require.Equal(t, a.IncidentCount, b.IncidentCount)
			require.Equal(t, a.CartelResolutionCount, b.CartelResolutionCount)
			require.Equal(t, a.ComputedAtBlock, b.ComputedAtBlock)
			require.Equal(t, a.SourceChainId, b.SourceChainId)

			// Domain coverage never counts a missing fact.
			var covered uint64
			for _, domain := range b.Domains {
				covered += domain.FactCount
			}
			require.LessOrEqual(t, covered, b.FactCount-b.MissingFactCount)

			statement, err := query.InTotoStatementV2(ctx,
				&types.QueryInTotoStatementV2Request{ManifestId: manifestID})
			require.NoError(t, err)
			require.True(t, proto.Equal(b, statement.Predicate.Certificate),
				"the certificate and the statement must carry the same v2 reading")
		})
	}
}

func TestInTotoStatementV2Projection(t *testing.T) {
	stub := reviewFixture()
	addFlatManifest(stub, "manifest/a", knowledgetypes.ManifestStatus_MANIFEST_STATUS_ATTESTED,
		"reviewed", "doctrine", "absent")
	query, ctx := newReviewQueryServer(stub)

	v1, err := query.InTotoStatement(ctx, &types.QueryInTotoStatementRequest{ManifestId: "manifest/a"})
	require.NoError(t, err)
	statement, err := query.InTotoStatementV2(ctx, &types.QueryInTotoStatementV2Request{ManifestId: "manifest/a"})
	require.NoError(t, err)

	require.Equal(t, InTotoStatementType, statement.StatementType)
	require.Equal(t, TrainingProvenancePredicateTypeV2, statement.PredicateType)
	require.NotEqual(t, TrainingProvenancePredicateType, statement.PredicateType)
	require.NotContains(t, statement.PredicateType, "/blob/main/")
	require.Len(t, statement.Subject, 1)
	require.True(t, proto.Equal(v1.Subject[0], statement.Subject[0]))
	require.Equal(t, "zerone://zerone-origin-1/training-corpus/manifest%2Fa", statement.Subject[0].Name)
	require.Equal(t, v1.Predicate.SourceChainId, statement.Predicate.SourceChainId)
	require.Equal(t, "zerone-observer-1", statement.Predicate.ObservedOnChainId)

	wire, err := protojson.Marshal(statement)
	require.NoError(t, err)
	root := stub.manifests["manifest/a"].MerkleRoot
	require.JSONEq(t, `{
		"_type": "https://in-toto.io/Statement/v1",
		"subject": [{
			"name": "zerone://zerone-origin-1/training-corpus/manifest%2Fa",
			"digest": {"sha256": "`+root+`"}
		}],
		"predicateType": "https://github.com/cambridgetcg/zerone-core/blob/fbfd5ef046c4b0c5eb263ff9a8d14d62796c2219/docs/specs/attestations/training-provenance-v2.md",
		"predicate": {
			"sourceChainId": "zerone-origin-1",
			"observedOnChainId": "zerone-observer-1",
			"certificate": {
				"manifestId": "manifest/a",
				"pipelineId": "pipeline-manifest/a",
				"merkleRoot": "`+root+`",
				"factCount": "3",
				"finalizedAtBlock": "12",
				"status": "MANIFEST_STATUS_ATTESTED",
				"domains": [
					{"domain": "doctrine_truth_seeking", "factCount": "1"},
					{"domain": "physics", "factCount": "1"}
				],
				"warningGrade": "A",
				"warningExplanation": "no privileged actions touched the manifest's facts; no incidents touched the knowledge module; no cartel resolutions in covered domains",
				"computedAtBlock": "77",
				"sourceChainId": "zerone-origin-1",
				"reviewedFactCount": "1",
				"unreviewedFactCount": "2",
				"doctrineFactCount": "1",
				"missingFactCount": "1",
				"reviewState": "partly_reviewed",
				"reviewExplanation": "reviewed_fact_count=1 of fact_count=3 have a retained verification round with at least one revealed vote; unreviewed_fact_count=2, including doctrine_fact_count=1 and missing_fact_count=1"
			}
		}
	}`, string(wire))
}

func TestPredicateTypeV2PinsTheSpecificationCommit(t *testing.T) {
	require.Equal(t,
		"https://github.com/cambridgetcg/zerone-core/blob/fbfd5ef046c4b0c5eb263ff9a8d14d62796c2219/docs/specs/attestations/training-provenance-v2.md",
		TrainingProvenancePredicateTypeV2)
	// v1 keeps its own pin.
	require.Equal(t,
		"https://github.com/cambridgetcg/zerone-core/blob/394bbef01df1b131223b1e874d554932d8dcd87c/docs/specs/attestations/training-provenance-v1.md",
		TrainingProvenancePredicateType)
}

func TestQueryV2StatusCodesMatchV1(t *testing.T) {
	validRoot := emptyFlatManifestRoot()
	stub := reviewFixture()
	stub.manifests = map[string]*knowledgetypes.TrainingManifest{
		"draft": {
			ManifestId: "draft", ChainId: "zerone-origin-1", MerkleRoot: validRoot,
			Status: knowledgetypes.ManifestStatus_MANIFEST_STATUS_DRAFT,
		},
		"composed": {
			ManifestId: "composed", ChainId: "zerone-origin-1", MerkleRoot: validRoot,
			ParentManifestId: "parent", ParentMerkleRoot: validRoot, CompositionDepth: 1,
			Status: knowledgetypes.ManifestStatus_MANIFEST_STATUS_FINALIZED,
		},
		"wrong-root": {
			ManifestId: "wrong-root", ChainId: "zerone-origin-1",
			MerkleRoot: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Status:     knowledgetypes.ManifestStatus_MANIFEST_STATUS_FINALIZED,
		},
		"mismatched-id": {
			ManifestId: "different-id", ChainId: "zerone-origin-1", MerkleRoot: validRoot,
			Status: knowledgetypes.ManifestStatus_MANIFEST_STATUS_FINALIZED,
		},
		"legacy-empty-source": {
			ManifestId: "legacy-empty-source", MerkleRoot: validRoot,
			Status: knowledgetypes.ManifestStatus_MANIFEST_STATUS_FINALIZED,
		},
	}
	query, ctx := newReviewQueryServer(stub)

	for name, req := range map[string]*types.QueryProvenanceCertificateV2Request{
		"nil request": nil,
		"empty id":    {},
	} {
		t.Run("certificate "+name, func(t *testing.T) {
			_, err := query.ProvenanceCertificateV2(ctx, req)
			require.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}
	for name, req := range map[string]*types.QueryInTotoStatementV2Request{
		"nil request": nil,
		"empty id":    {},
	} {
		t.Run("statement "+name, func(t *testing.T) {
			_, err := query.InTotoStatementV2(ctx, req)
			require.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}

	for _, manifestID := range []string{"missing", "draft", "composed", "wrong-root", "mismatched-id", "legacy-empty-source"} {
		t.Run(manifestID, func(t *testing.T) {
			_, v1Err := query.ProvenanceCertificate(ctx, &types.QueryProvenanceCertificateRequest{ManifestId: manifestID})
			_, v2Err := query.ProvenanceCertificateV2(ctx, &types.QueryProvenanceCertificateV2Request{ManifestId: manifestID})
			require.Equal(t, status.Code(v1Err), status.Code(v2Err))

			_, v1Err = query.InTotoStatement(ctx, &types.QueryInTotoStatementRequest{ManifestId: manifestID})
			_, v2Err = query.InTotoStatementV2(ctx, &types.QueryInTotoStatementV2Request{ManifestId: manifestID})
			require.Equal(t, status.Code(v1Err), status.Code(v2Err))
		})
	}

	_, err := query.ProvenanceCertificateV2(ctx, &types.QueryProvenanceCertificateV2Request{ManifestId: "missing"})
	require.Equal(t, codes.NotFound, status.Code(err))
	_, err = query.InTotoStatementV2(ctx, &types.QueryInTotoStatementV2Request{ManifestId: "draft"})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
	_, err = query.InTotoStatementV2(ctx, &types.QueryInTotoStatementV2Request{ManifestId: "composed"})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
	_, err = query.ProvenanceCertificateV2(ctx, &types.QueryProvenanceCertificateV2Request{ManifestId: "wrong-root"})
	require.Equal(t, codes.DataLoss, status.Code(err))

	// The certificate query, like v1's, describes a draft or a composed
	// child's delta; only the in-toto statement refuses them.
	draft, err := query.ProvenanceCertificateV2(ctx, &types.QueryProvenanceCertificateV2Request{ManifestId: "draft"})
	require.NoError(t, err)
	require.Equal(t, ReviewStateEmpty, draft.Certificate.ReviewState)

	unwired := NewQueryServerImpl(NewKeeper(nil))
	_, err = unwired.ProvenanceCertificateV2(context.Background(),
		&types.QueryProvenanceCertificateV2Request{ManifestId: "manifest"})
	require.Equal(t, codes.Internal, status.Code(err))
	_, err = unwired.InTotoStatementV2(context.Background(),
		&types.QueryInTotoStatementV2Request{ManifestId: "manifest"})
	require.Equal(t, codes.Internal, status.Code(err))
}
