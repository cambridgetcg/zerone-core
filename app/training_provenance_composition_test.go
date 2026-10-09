package app

import (
	"fmt"
	"testing"

	"cosmossdk.io/log"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	challengetypes "github.com/zerone-chain/zerone/x/capture_challenge/types"
	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	provenancekeeper "github.com/zerone-chain/zerone/x/training_provenance/keeper"
	provenancetypes "github.com/zerone-chain/zerone/x/training_provenance/types"
)

// TestTrainingProvenanceDomainWarningComposition follows actual stored records
// through the application's installed challenge adapter and both query methods.
// Synthetic challenge records test the read contract, not challenge adjudication,
// independent controllers, actual model training or a deployed network.
func TestTrainingProvenanceDomainWarningComposition(t *testing.T) {
	db := dbm.NewMemDB()
	application := NewZeroneApp(log.NewNopLogger(), db, nil, true,
		simtestutil.NewAppOptionsWithFlagHome(t.TempDir()),
		baseapp.SetChainID("provenance-observer"), baseapp.SetIAVLSyncPruning(true))
	t.Cleanup(func() { require.NoError(t, application.Close()) })
	ctx := application.NewUncachedContext(false, cmtproto.Header{ChainID: "provenance-observer", Height: 77})

	fact := &knowledgetypes.Fact{Id: "included-fact", Domain: "mathematics"}
	require.NoError(t, application.KnowledgeKeeper.SetFact(ctx, fact))
	ids := knowledgekeeper.SelectedManifestIDs{FactIDs: []string{fact.Id}}
	manifest := &knowledgetypes.TrainingManifest{
		ManifestId: "domain-warning", PipelineId: "synthetic-pipeline",
		IncludedFactIds: ids.FactIDs, MerkleRoot: knowledgekeeper.ComputeManifestMerkleRoot(ids),
		ChainId: "provenance-origin", FinalizedAtBlock: 12,
		Status: knowledgetypes.ManifestStatus_MANIFEST_STATUS_FINALIZED,
	}
	require.NoError(t, application.KnowledgeKeeper.SetTrainingManifest(ctx, manifest))
	query := provenancekeeper.NewQueryServerImpl(application.TrainingProvenanceKeeper)

	resolved := challengetypes.ChallengeStatus_CHALLENGE_STATUS_RESOLVED
	upheld := challengetypes.ChallengeOutcome_CHALLENGE_OUTCOME_UPHELD
	cases := []struct {
		name    string
		domain  string
		status  challengetypes.ChallengeStatus
		outcome challengetypes.ChallengeOutcome
		count   uint32
	}{
		{"other-domain", "biology", resolved, upheld, 0},
		{"unresolved", "mathematics", challengetypes.ChallengeStatus_CHALLENGE_STATUS_UNDER_REVIEW, upheld, 0},
		{"rejected", "mathematics", resolved, challengetypes.ChallengeOutcome_CHALLENGE_OUTCOME_REJECTED, 0},
		{"partial", "mathematics", resolved, challengetypes.ChallengeOutcome_CHALLENGE_OUTCOME_PARTIAL, 0},
		{"no-resolution", "mathematics", resolved, challengetypes.ChallengeOutcome_CHALLENGE_OUTCOME_UNSPECIFIED, 0},
		{"covered-domain-unlinked-panel", "mathematics", resolved, upheld, 1},
		{"second-covered-resolution", "mathematics", resolved, upheld, 2},
	}
	var stored []*challengetypes.CaptureChallenge
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			challenge := &challengetypes.CaptureChallenge{
				Id: tc.name, Domain: tc.domain, Status: tc.status,
				// No panel or fact relationship is asserted by this fixture.
				AccusedValidators: []string{"unrelated-validator"},
			}
			if tc.outcome != challengetypes.ChallengeOutcome_CHALLENGE_OUTCOME_UNSPECIFIED {
				challenge.Resolution = &challengetypes.CaptureResolution{Outcome: tc.outcome}
			}
			application.CaptureChallengeKeeper.SetChallenge(ctx, challenge)
			stored = append(stored, challenge)

			response, err := query.ProvenanceCertificate(ctx,
				&provenancetypes.QueryProvenanceCertificateRequest{ManifestId: manifest.ManifestId})
			require.NoError(t, err)
			cert := response.Certificate
			require.Equal(t, tc.count, cert.CartelResolutionCount)
			require.Equal(t, uint32(0), cert.PrivilegedActionCount)
			require.Equal(t, uint32(0), cert.IncidentCount)
			require.Equal(t, uint64(1), cert.FactCount)
			require.Len(t, cert.Domains, 1)
			require.Equal(t, fact.Domain, cert.Domains[0].Domain)
			require.Equal(t, uint64(77), cert.ComputedAtBlock)
			if tc.count == 0 {
				require.Equal(t, "A", cert.TrustGrade)
			} else {
				require.Equal(t, "F", cert.TrustGrade)
				require.Equal(t, fmt.Sprintf("cartel_resolutions=%d in covered domains; domain-level warning only, not evidence about this manifest's specific adjudication panel", tc.count), cert.TrustExplanation)
			}

			statement, err := query.InTotoStatement(ctx,
				&provenancetypes.QueryInTotoStatementRequest{ManifestId: manifest.ManifestId})
			require.NoError(t, err)
			require.True(t, proto.Equal(cert, statement.Predicate.Certificate), "both readers must expose the same explanation and counts")
			require.Equal(t, provenancekeeper.TrainingProvenancePredicateType, statement.PredicateType)
			require.Equal(t, "provenance-origin", statement.Predicate.SourceChainId)
			require.Equal(t, "provenance-observer", statement.Predicate.ObservedOnChainId)
			require.Equal(t, manifest.MerkleRoot, statement.Subject[0].Digest["sha256"])

			// Queries do not rewrite the producer records they describe.
			gotManifest, ok := application.KnowledgeKeeper.GetTrainingManifest(ctx, manifest.ManifestId)
			require.True(t, ok)
			require.True(t, proto.Equal(manifest, gotManifest))
			gotFact, ok := application.KnowledgeKeeper.GetFact(ctx, fact.Id)
			require.True(t, ok)
			require.True(t, proto.Equal(fact, gotFact))
			for _, want := range stored {
				got, ok := application.CaptureChallengeKeeper.GetChallenge(ctx, want.Id)
				require.True(t, ok)
				require.True(t, proto.Equal(want, got))
			}
		})
	}
}
