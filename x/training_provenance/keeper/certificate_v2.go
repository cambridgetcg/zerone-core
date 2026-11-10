package keeper

import (
	"context"
	"fmt"

	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	"github.com/zerone-chain/zerone/x/training_provenance/types"
)

// TrainingProvenancePredicateTypeV2 names the spec-only commit that holds the
// predicate v2 specification text. Like the v1 URI, it must never follow a
// mutable branch.
const TrainingProvenancePredicateTypeV2 = "https://github.com/cambridgetcg/zerone-core/blob/43f3a04178acab749758e383e064154854ad8b96/docs/specs/attestations/training-provenance-v2.md"

// Review states of predicate v2. They describe review coverage only and are
// never combined with the warning grade.
const (
	ReviewStateReviewed       = "reviewed"
	ReviewStatePartlyReviewed = "partly_reviewed"
	ReviewStateUnreviewed     = "unreviewed"
	ReviewStateEmpty          = "empty"
)

// reviewCoverage holds the predicate v2 review counts. reviewed+unreviewed is
// the number of included fact ID entries; doctrine and missing are disjoint
// subsets of unreviewed.
type reviewCoverage struct {
	reviewed   uint64
	unreviewed uint64
	doctrine   uint64
	missing    uint64
}

// BuildCertificateV2 synthesizes the predicate v2 certificate. It builds the
// unchanged v1 certificate first, so identity, coverage, audit counts and
// error classes are the v1 ones, and the warning reading is the v1 grade and
// explanation under a name that does not suggest review. It then adds review
// coverage. Like v1, it is read-only and computed live.
func (k Keeper) BuildCertificateV2(ctx context.Context, manifestID string) (*types.ProvenanceCertificateV2, error) {
	certificate, err := k.BuildCertificate(ctx, manifestID)
	if err != nil {
		return nil, err
	}
	return k.certificateV2FromV1(ctx, manifestID, certificate, errCertificateDataLoss)
}

// BuildInTotoStatementV2 projects the v2 certificate into an unsigned in-toto
// Statement v1 with the predicate v2 type. It builds the v1 statement first,
// so eligibility, subject and chain fields, and error classes are exactly
// those of predicate v1.
func (k Keeper) BuildInTotoStatementV2(ctx context.Context, manifestID string) (*types.InTotoStatementV1PredicateV2, error) {
	statement, err := k.BuildInTotoStatement(ctx, manifestID)
	if err != nil {
		return nil, err
	}
	if statement.Predicate == nil || statement.Predicate.Certificate == nil {
		return nil, fmt.Errorf("%w: v1 statement has no certificate", errInTotoDataLoss)
	}
	certificate, err := k.certificateV2FromV1(ctx, manifestID, statement.Predicate.Certificate, errInTotoDataLoss)
	if err != nil {
		return nil, err
	}
	return &types.InTotoStatementV1PredicateV2{
		StatementType: statement.StatementType,
		Subject:       statement.Subject,
		PredicateType: TrainingProvenancePredicateTypeV2,
		Predicate: &types.TrainingProvenancePredicateV2{
			SourceChainId:     statement.Predicate.SourceChainId,
			ObservedOnChainId: statement.Predicate.ObservedOnChainId,
			Certificate:       certificate,
		},
	}, nil
}

// certificateV2FromV1 copies a v1 certificate into the v2 shape and adds the
// review reading. The manifest is read again in the same query context, so it
// is the record the v1 certificate described.
func (k Keeper) certificateV2FromV1(
	ctx context.Context,
	manifestID string,
	v1 *types.ProvenanceCertificate,
	dataLoss error,
) (*types.ProvenanceCertificateV2, error) {
	manifest, ok := k.knowledgeKeeper.GetTrainingManifest(ctx, manifestID)
	if !ok || manifest == nil || manifest.ManifestId != v1.ManifestId {
		return nil, fmt.Errorf("%w: manifest %s changed while building its v2 certificate", dataLoss, manifestID)
	}
	coverage := k.countReviewCoverage(ctx, manifest.IncludedFactIds)
	factCount := uint64(len(manifest.IncludedFactIds))
	if factCount != v1.FactCount {
		return nil, fmt.Errorf("%w: manifest %s fact count changed while building its v2 certificate", dataLoss, manifestID)
	}
	state, explanation := computeReviewState(coverage)

	return &types.ProvenanceCertificateV2{
		ManifestId:            v1.ManifestId,
		PipelineId:            v1.PipelineId,
		MerkleRoot:            v1.MerkleRoot,
		FactCount:             v1.FactCount,
		FinalizedAtBlock:      v1.FinalizedAtBlock,
		Status:                v1.Status,
		Domains:               v1.Domains,
		PrivilegedActionCount: v1.PrivilegedActionCount,
		IncidentCount:         v1.IncidentCount,
		CartelResolutionCount: v1.CartelResolutionCount,
		WarningGrade:          v1.TrustGrade,
		WarningExplanation:    v1.TrustExplanation,
		ComputedAtBlock:       v1.ComputedAtBlock,
		SourceChainId:         v1.SourceChainId,
		ReviewedFactCount:     coverage.reviewed,
		UnreviewedFactCount:   coverage.unreviewed,
		DoctrineFactCount:     coverage.doctrine,
		MissingFactCount:      coverage.missing,
		ReviewState:           state,
		ReviewExplanation:     explanation,
	}, nil
}

// countReviewCoverage classifies each included fact ID entry once, by the
// first rule that applies (training-provenance-v2.md, "Review reading"):
//
//  1. no fact under the ID: unreviewed and missing;
//  2. empty claim_id: unreviewed, and doctrine when the category or method
//     is doctrine;
//  3. no round returned through the claim-to-round index: unreviewed;
//  4. the round names a different claim: unreviewed;
//  5. the round records no reveal: unreviewed;
//  6. otherwise: reviewed.
//
// An unknown is never counted as reviewed.
func (k Keeper) countReviewCoverage(ctx context.Context, factIDs []string) reviewCoverage {
	var coverage reviewCoverage
	for _, factID := range factIDs {
		fact, ok := k.knowledgeKeeper.GetFact(ctx, factID)
		switch {
		case !ok || fact == nil:
			coverage.unreviewed++
			coverage.missing++
		case fact.ClaimId == "":
			coverage.unreviewed++
			if isDoctrineFact(fact) {
				coverage.doctrine++
			}
		case k.claimHasRevealedReview(ctx, fact.ClaimId):
			coverage.reviewed++
		default:
			coverage.unreviewed++
		}
	}
	return coverage
}

func (k Keeper) claimHasRevealedReview(ctx context.Context, claimID string) bool {
	round, ok := k.knowledgeKeeper.GetRoundByClaimID(ctx, claimID)
	if !ok || round == nil || round.ClaimId != claimID {
		return false
	}
	for _, reveal := range round.Reveals {
		if reveal != nil {
			return true
		}
	}
	return false
}

func isDoctrineFact(fact *knowledgetypes.Fact) bool {
	return fact.Category == knowledgetypes.DoctrineCategory ||
		fact.MethodId == knowledgetypes.DoctrineMethodId
}

// computeReviewState derives the review state and its one-sentence
// explanation from the review counts. The wording is fixed by predicate v2;
// changing it requires a new predicate version.
func computeReviewState(c reviewCoverage) (string, string) {
	total := c.reviewed + c.unreviewed
	if total == 0 {
		return ReviewStateEmpty, "fact_count=0: no fact IDs are included directly in this manifest, so there is no review to count"
	}
	explanation := fmt.Sprintf(
		"reviewed_fact_count=%d of fact_count=%d have a retained verification round with at least one revealed vote; unreviewed_fact_count=%d, including doctrine_fact_count=%d and missing_fact_count=%d",
		c.reviewed, total, c.unreviewed, c.doctrine, c.missing,
	)
	switch {
	case c.unreviewed == 0:
		return ReviewStateReviewed, explanation
	case c.reviewed == 0:
		return ReviewStateUnreviewed, explanation
	default:
		return ReviewStatePartlyReviewed, explanation
	}
}
