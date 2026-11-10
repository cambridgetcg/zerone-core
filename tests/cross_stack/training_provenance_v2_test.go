package cross_stack_test

import (
	"bytes"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	kt "github.com/zerone-chain/zerone/x/knowledge/types"
	ontologytypes "github.com/zerone-chain/zerone/x/ontology/types"
	provkeeper "github.com/zerone-chain/zerone/x/training_provenance/keeper"
	provtypes "github.com/zerone-chain/zerone/x/training_provenance/types"
)

// TestTrainingProvenanceV2_ReviewCoverage reads predicate v2 against the
// production keepers. Doctrine comes from the harness genesis. The reviewed
// fact comes from a claim that went through the real admission, commitment,
// reveal and phase handlers (completeCorrectionReview). Three funded test
// accounts are not three independent controllers, and nothing here is a live
// network or proof that a fact is true.
//
// It shows the gap v2 closes: a doctrine-only manifest that v1 grades A reads
// "unreviewed" in v2, with the same A as its warning grade.
func TestTrainingProvenanceV2_ReviewCoverage(t *testing.T) {
	h := NewTestHarness(t)
	t.Cleanup(func() { require.NoError(t, h.App.Close()) })
	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	qs := provkeeper.NewQueryServerImpl(h.TrainingProvenanceKeeper)

	// Select the current retained, neutral, funded review policy explicitly,
	// as the correction-admission fixture does.
	require.NoError(t, h.KnowledgeKeeper.EnableRecordIntegrity(h.Ctx))
	require.NoError(t, h.KnowledgeKeeper.EnableReviewNeutrality(h.Ctx))
	require.NoError(t, h.KnowledgeKeeper.EnableFundSettlement(h.Ctx))
	params, err := h.KnowledgeKeeper.GetParams(h.Ctx)
	require.NoError(t, err)
	// Shorten deadlines only; keep the normal quorum and funding rules.
	params.CommitPhaseBlocks, params.RevealPhaseBlocks, params.AggregationPhaseBlocks = 2, 2, 1
	require.NoError(t, h.KnowledgeKeeper.SetParams(h.Ctx, params))

	const domain = "provenance_review_coverage"
	require.NoError(t, h.KnowledgeKeeper.SetDomain(h.Ctx, &kt.Domain{Name: domain, Status: kt.DomainStatus_DOMAIN_STATUS_ACTIVE}))
	h.App.ZeroneOntologyKeeper.SetDomain(h.Ctx, &ontologytypes.Domain{Name: domain, Status: "active", Stratum: uint32(ontologytypes.StratumEmpirical), Depth: 1})

	author := sdk.AccAddress(bytes.Repeat([]byte{91}, 20))
	operator := sdk.AccAddress(bytes.Repeat([]byte{92}, 20)).String()
	var reviewers []string
	for _, seed := range []byte{91, 93, 94, 95} {
		addr := sdk.AccAddress(bytes.Repeat([]byte{seed}, 20))
		require.NoError(t, h.FundAccount(addr, sdk.NewCoins(sdk.NewInt64Coin("uzrn", 500_000_000))))
		if seed != 91 {
			reviewers = append(reviewers, addr.String())
		}
	}

	// A fact made through a real claim round.
	submitted, err := ms.SubmitClaim(h.Ctx, &kt.MsgSubmitClaim{
		Submitter: author.String(), FactContent: "The provenance fixture observation holds under condition B.",
		Domain: domain, Category: "empirical", Stake: params.MinReviewFee,
	})
	require.NoError(t, err)
	claim, found := h.KnowledgeKeeper.GetClaim(h.Ctx, submitted.ClaimId)
	require.True(t, found)
	round := completeCorrectionReview(t, h, claim.VerificationRoundId, reviewers, "accept")
	require.NotEmpty(t, round.Reveals)
	reviewedFactID := knowledgekeeper.GenerateFactID(claim.Id, uint64(h.Ctx.BlockHeight()))
	reviewedFact, found := h.KnowledgeKeeper.GetFact(h.Ctx, reviewedFactID)
	require.True(t, found)
	require.Equal(t, claim.Id, reviewedFact.ClaimId)

	// Doctrine loaded at genesis has no claim.
	doctrine, found := h.KnowledgeKeeper.GetFact(h.Ctx, "commitment-1")
	require.True(t, found, "harness genesis must load doctrine")
	require.Empty(t, doctrine.ClaimId)
	require.Equal(t, kt.DoctrineCategory, doctrine.Category)

	require.NoError(t, h.KnowledgeKeeper.SetTrainingPipeline(h.Ctx, &kt.TrainingPipeline{
		Id: "pipe-prov-v2", OperatorAddress: operator, TokenizerVersion: 1,
		MethodologySetVersion: 1, Status: "declared",
	}))
	createFinalized := func(id string, selector *kt.CorpusSelector) {
		t.Helper()
		_, err := ms.CreateTrainingManifest(h.Ctx, &kt.MsgCreateTrainingManifest{
			Creator: operator, Id: id, PipelineId: "pipe-prov-v2", CorpusSelector: selector,
		})
		require.NoError(t, err)
		_, err = ms.FinalizeTrainingManifest(h.Ctx, &kt.MsgFinalizeTrainingManifest{Creator: operator, ManifestId: id})
		require.NoError(t, err)
	}
	createFinalized("manifest-v2-doctrine", &kt.CorpusSelector{MethodId: kt.DoctrineMethodId})
	createFinalized("manifest-v2-reviewed", &kt.CorpusSelector{DomainWhitelist: []string{domain}})
	createFinalized("manifest-v2-mixed", &kt.CorpusSelector{DomainWhitelist: []string{domain, kt.DoctrineDomainToK}})

	// The selector never includes a missing fact, so this record is written
	// directly with a valid root: a stored manifest whose second fact ID has
	// no fact in the store.
	for id, factIDs := range map[string][]string{
		"manifest-v2-with-missing": {reviewedFactID, "fact-never-stored"},
		"manifest-v2-only-missing": {"fact-never-stored"},
	} {
		ids := knowledgekeeper.SelectedManifestIDs{FactIDs: factIDs}
		require.NoError(t, h.KnowledgeKeeper.SetTrainingManifest(h.Ctx, &kt.TrainingManifest{
			ManifestId: id, PipelineId: "pipe-prov-v2", Creator: operator,
			IncludedFactIds: factIDs, MerkleRoot: knowledgekeeper.ComputeManifestMerkleRoot(ids),
			ChainId: h.Ctx.ChainID(), FinalizedAtBlock: uint64(h.Ctx.BlockHeight()),
			Status: kt.ManifestStatus_MANIFEST_STATUS_FINALIZED,
		}))
	}

	read := func(id string) (*provtypes.ProvenanceCertificate, *provtypes.ProvenanceCertificateV2) {
		t.Helper()
		v1, err := qs.ProvenanceCertificate(h.Ctx, &provtypes.QueryProvenanceCertificateRequest{ManifestId: id})
		require.NoError(t, err)
		v2, err := qs.ProvenanceCertificateV2(h.Ctx, &provtypes.QueryProvenanceCertificateV2Request{ManifestId: id})
		require.NoError(t, err)
		cert := v2.Certificate
		require.Equal(t, v1.Certificate.TrustGrade, cert.WarningGrade)
		require.Equal(t, v1.Certificate.TrustExplanation, cert.WarningExplanation)
		require.Equal(t, v1.Certificate.FactCount, cert.FactCount)
		require.Equal(t, cert.FactCount, cert.ReviewedFactCount+cert.UnreviewedFactCount)

		statement, err := qs.InTotoStatementV2(h.Ctx, &provtypes.QueryInTotoStatementV2Request{ManifestId: id})
		require.NoError(t, err)
		require.Equal(t, provkeeper.TrainingProvenancePredicateTypeV2, statement.PredicateType)
		require.Equal(t, h.Ctx.ChainID(), statement.Predicate.SourceChainId)
		require.True(t, proto.Equal(cert, statement.Predicate.Certificate))
		return v1.Certificate, cert
	}

	t.Run("doctrine only reads unreviewed with warning grade A", func(t *testing.T) {
		v1, cert := read("manifest-v2-doctrine")
		require.Equal(t, "A", v1.TrustGrade, "v1 grades a doctrine-only manifest A")
		require.Equal(t, "A", cert.WarningGrade)
		require.Equal(t, provkeeper.ReviewStateUnreviewed, cert.ReviewState)
		require.Greater(t, cert.FactCount, uint64(0))
		require.Zero(t, cert.ReviewedFactCount)
		require.Equal(t, cert.FactCount, cert.UnreviewedFactCount)
		require.Equal(t, cert.FactCount, cert.DoctrineFactCount)
		require.Zero(t, cert.MissingFactCount)
	})

	t.Run("a fact made through a real claim round reads reviewed", func(t *testing.T) {
		_, cert := read("manifest-v2-reviewed")
		require.Equal(t, uint64(1), cert.FactCount)
		require.Equal(t, uint64(1), cert.ReviewedFactCount)
		require.Zero(t, cert.UnreviewedFactCount)
		require.Equal(t, provkeeper.ReviewStateReviewed, cert.ReviewState)
	})

	t.Run("reviewed and doctrine facts together read partly reviewed", func(t *testing.T) {
		_, cert := read("manifest-v2-mixed")
		require.Greater(t, cert.FactCount, uint64(1))
		require.Equal(t, uint64(1), cert.ReviewedFactCount)
		require.Equal(t, cert.FactCount-1, cert.UnreviewedFactCount)
		require.Equal(t, cert.UnreviewedFactCount, cert.DoctrineFactCount)
		require.Equal(t, provkeeper.ReviewStatePartlyReviewed, cert.ReviewState)
	})

	t.Run("a missing fact ID counts as unreviewed", func(t *testing.T) {
		_, cert := read("manifest-v2-with-missing")
		require.Equal(t, uint64(2), cert.FactCount)
		require.Equal(t, uint64(1), cert.ReviewedFactCount)
		require.Equal(t, uint64(1), cert.UnreviewedFactCount)
		require.Equal(t, uint64(1), cert.MissingFactCount)
		require.Zero(t, cert.DoctrineFactCount)
		require.Equal(t, provkeeper.ReviewStatePartlyReviewed, cert.ReviewState)

		_, only := read("manifest-v2-only-missing")
		require.Equal(t, uint64(1), only.MissingFactCount)
		require.Equal(t, uint64(1), only.UnreviewedFactCount)
		require.Equal(t, provkeeper.ReviewStateUnreviewed, only.ReviewState)
		_, found := h.KnowledgeKeeper.GetFact(h.Ctx, "fact-never-stored")
		require.False(t, found, "a query must not create the missing fact it counts")
	})
}
