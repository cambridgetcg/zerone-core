package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"cosmossdk.io/log"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	provenancekeeper "github.com/zerone-chain/zerone/x/training_provenance/keeper"
	provenancetypes "github.com/zerone-chain/zerone/x/training_provenance/types"
)

// TestTrainingProvenanceV2ReviewCoverageComposition follows stored records
// through the application's real knowledge keeper and the v2 query methods.
// The facts and rounds are synthetic records written directly: they test the
// read contract, not claim adjudication, independent reviewers, model
// training or a deployed network.
func TestTrainingProvenanceV2ReviewCoverageComposition(t *testing.T) {
	db := dbm.NewMemDB()
	application := NewZeroneApp(log.NewNopLogger(), db, nil, true,
		simtestutil.NewAppOptionsWithFlagHome(t.TempDir()),
		baseapp.SetChainID("provenance-observer"), baseapp.SetIAVLSyncPruning(true))
	t.Cleanup(func() { require.NoError(t, application.Close()) })
	ctx := application.NewUncachedContext(false, cmtproto.Header{ChainID: "provenance-observer", Height: 77})

	rounds := []*knowledgetypes.VerificationRound{
		{
			Id: "round-reviewed", ClaimId: "claim-reviewed",
			Phase:   knowledgetypes.VerificationPhase_VERIFICATION_PHASE_COMPLETE,
			Verdict: knowledgetypes.Verdict_VERDICT_ACCEPT,
			Reveals: []*knowledgetypes.RevealEntry{{Verifier: "synthetic-reviewer", Vote: "accept"}},
		},
		{
			Id: "round-silent", ClaimId: "claim-silent",
			Phase: knowledgetypes.VerificationPhase_VERIFICATION_PHASE_EXPIRED,
		},
	}
	for _, round := range rounds {
		require.NoError(t, application.KnowledgeKeeper.SetVerificationRound(ctx, round))
	}
	facts := []*knowledgetypes.Fact{
		knowledgetypes.BuildDoctrineFact("fact-doctrine", knowledgetypes.DoctrineDomainTruthSeeking, "synthetic doctrine"),
		{Id: "fact-genesis", Domain: "mathematics"},
		{Id: "fact-no-round", Domain: "mathematics", ClaimId: "claim-without-round"},
		{Id: "fact-reviewed", Domain: "mathematics", ClaimId: "claim-reviewed"},
		{Id: "fact-silent", Domain: "mathematics", ClaimId: "claim-silent"},
	}
	for _, fact := range facts {
		require.NoError(t, application.KnowledgeKeeper.SetFact(ctx, fact))
	}
	// fact-missing is included but never stored.
	ids := knowledgekeeper.SelectedManifestIDs{FactIDs: []string{
		"fact-doctrine", "fact-genesis", "fact-missing", "fact-no-round", "fact-reviewed", "fact-silent",
	}}
	manifest := &knowledgetypes.TrainingManifest{
		ManifestId: "review-coverage", PipelineId: "synthetic-pipeline",
		IncludedFactIds: ids.FactIDs, MerkleRoot: knowledgekeeper.ComputeManifestMerkleRoot(ids),
		ChainId: "provenance-origin", FinalizedAtBlock: 12,
		Status: knowledgetypes.ManifestStatus_MANIFEST_STATUS_FINALIZED,
	}
	require.NoError(t, application.KnowledgeKeeper.SetTrainingManifest(ctx, manifest))
	query := provenancekeeper.NewQueryServerImpl(application.TrainingProvenanceKeeper)

	v1, err := query.ProvenanceCertificate(ctx,
		&provenancetypes.QueryProvenanceCertificateRequest{ManifestId: manifest.ManifestId})
	require.NoError(t, err)
	response, err := query.ProvenanceCertificateV2(ctx,
		&provenancetypes.QueryProvenanceCertificateV2Request{ManifestId: manifest.ManifestId})
	require.NoError(t, err)
	cert := response.Certificate

	// v1 grades this manifest A. v2 keeps that as the warning grade and shows
	// that only one of six included fact IDs has a revealed review.
	require.Equal(t, "A", v1.Certificate.TrustGrade)
	require.Equal(t, v1.Certificate.TrustGrade, cert.WarningGrade)
	require.Equal(t, v1.Certificate.TrustExplanation, cert.WarningExplanation)
	require.Equal(t, uint64(6), cert.FactCount)
	require.Equal(t, uint64(1), cert.ReviewedFactCount)
	require.Equal(t, uint64(5), cert.UnreviewedFactCount)
	require.Equal(t, uint64(1), cert.DoctrineFactCount)
	require.Equal(t, uint64(1), cert.MissingFactCount)
	require.Equal(t, provenancekeeper.ReviewStatePartlyReviewed, cert.ReviewState)

	statement, err := query.InTotoStatementV2(ctx,
		&provenancetypes.QueryInTotoStatementV2Request{ManifestId: manifest.ManifestId})
	require.NoError(t, err)
	require.True(t, proto.Equal(cert, statement.Predicate.Certificate), "both v2 readers must expose the same counts and wording")
	require.Equal(t, provenancekeeper.TrainingProvenancePredicateTypeV2, statement.PredicateType)
	require.Equal(t, "provenance-origin", statement.Predicate.SourceChainId)
	require.Equal(t, "provenance-observer", statement.Predicate.ObservedOnChainId)
	require.Equal(t, manifest.MerkleRoot, statement.Subject[0].Digest["sha256"])

	// The SDK consumes this same synthetic wire fixture. Keep its JSON shape,
	// counts and wording bound to the real query producer.
	wire, err := protojson.Marshal(statement)
	require.NoError(t, err)
	fixture, err := os.ReadFile("../sdk/typescript/tests/fixtures/provenance-v2-review-coverage.json")
	require.NoError(t, err)
	require.JSONEq(t, string(fixture), string(wire))

	// Queries do not rewrite the producer records they describe.
	gotManifest, ok := application.KnowledgeKeeper.GetTrainingManifest(ctx, manifest.ManifestId)
	require.True(t, ok)
	require.True(t, proto.Equal(manifest, gotManifest))
	for _, want := range facts {
		got, ok := application.KnowledgeKeeper.GetFact(ctx, want.Id)
		require.True(t, ok)
		require.True(t, proto.Equal(want, got))
	}
	for _, want := range rounds {
		got, ok := application.KnowledgeKeeper.GetVerificationRound(ctx, want.Id)
		require.True(t, ok)
		require.True(t, proto.Equal(want, got))
	}
	_, ok = application.KnowledgeKeeper.GetFact(ctx, "fact-missing")
	require.False(t, ok, "a query must not create the missing fact it counts")
}

// The v2 query methods are reachable through the application's REST gateway
// beside the v1 ones.
func TestRegisterAPIRoutesIncludesTrainingProvenanceV2(t *testing.T) {
	db := dbm.NewMemDB()
	application := NewZeroneApp(log.NewNopLogger(), db, nil, true,
		simtestutil.NewAppOptionsWithFlagHome(t.TempDir()))
	t.Cleanup(func() { require.NoError(t, application.Close()) })
	clientCtx := client.Context{
		Codec:             application.AppCodec(),
		InterfaceRegistry: application.InterfaceRegistry(),
		TxConfig:          application.TxConfig(),
	}
	apiServer := api.New(clientCtx, log.NewNopLogger(), grpc.NewServer())
	application.RegisterAPIRoutes(apiServer, config.APIConfig{})

	for _, route := range []string{
		"/zerone/training_provenance/v1/certificate-v2/missing",
		"/zerone/training_provenance/v1/in-toto-v2/missing",
	} {
		t.Run(route, func(t *testing.T) {
			response := httptest.NewRecorder()
			apiServer.Router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, route, nil))
			require.NotEqual(t, http.StatusNotFound, response.Code)
		})
	}
	unknown := httptest.NewRecorder()
	apiServer.Router.ServeHTTP(unknown,
		httptest.NewRequest(http.MethodGet, "/zerone/training_provenance/v1/in-toto-v3/missing", nil))
	require.Equal(t, http.StatusNotFound, unknown.Code)
}
