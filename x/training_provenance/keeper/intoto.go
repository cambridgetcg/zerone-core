package keeper

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	"github.com/zerone-chain/zerone/x/training_provenance/types"
)

const (
	// InTotoStatementType is the current stable in-toto Statement format.
	InTotoStatementType = "https://in-toto.io/Statement/v1"

	// TrainingProvenancePredicateType versions Zerone's predicate semantics at
	// an immutable source revision. Predicate TypeURIs must not drift when a
	// repository branch advances.
	TrainingProvenancePredicateType = "https://github.com/cambridgetcg/zerone-core/blob/03033b0d1cc3665b08335bfe095abe9feb27ba89/docs/specs/attestations/training-provenance-v1.md"
)

var (
	errInTotoManifestNotFound      = errors.New("training manifest not found")
	errInTotoProjectionUnavailable = errors.New("in-toto projection unavailable")
	errInTotoDataLoss              = errors.New("in-toto projection data loss")
)

// BuildInTotoStatement projects a live ProvenanceCertificate into an unsigned
// in-toto Statement v1. Predicate v1 deliberately accepts only sealed,
// root (non-composed) manifests: the existing certificate synthesizer reports
// a composed child's delta rather than recursively materializing its parent.
// It owns no state and changes no consensus behavior.
func (k Keeper) BuildInTotoStatement(ctx context.Context, manifestID string) (*types.InTotoStatementV1, error) {
	if k.knowledgeKeeper == nil {
		return nil, errors.New("knowledge keeper not wired")
	}
	manifest, ok := k.knowledgeKeeper.GetTrainingManifest(ctx, manifestID)
	if !ok || manifest == nil {
		return nil, fmt.Errorf("%w: %s", errInTotoManifestNotFound, manifestID)
	}
	if manifest.ManifestId != manifestID {
		return nil, fmt.Errorf(
			"%w: manifest key %q contains ID %q",
			errInTotoDataLoss,
			manifestID,
			manifest.ManifestId,
		)
	}
	hasParentID := manifest.ParentManifestId != ""
	hasParentRoot := manifest.ParentMerkleRoot != ""
	hasCompositionDepth := manifest.CompositionDepth != 0
	if hasParentID != hasParentRoot || hasParentID != hasCompositionDepth {
		return nil, fmt.Errorf(
			"%w: manifest %s has inconsistent composition metadata",
			errInTotoDataLoss,
			manifestID,
		)
	}
	if hasParentID {
		return nil, fmt.Errorf(
			"%w: composed manifest %s requires recursive parent coverage",
			errInTotoProjectionUnavailable,
			manifestID,
		)
	}
	switch manifest.Status {
	case knowledgetypes.ManifestStatus_MANIFEST_STATUS_FINALIZED,
		knowledgetypes.ManifestStatus_MANIFEST_STATUS_ATTESTED,
		knowledgetypes.ManifestStatus_MANIFEST_STATUS_SUPERSEDED:
	default:
		return nil, fmt.Errorf(
			"%w: manifest %s is not eligible (status %s)",
			errInTotoProjectionUnavailable,
			manifestID,
			manifest.Status,
		)
	}
	if err := validateManifestMerkleRoot(manifestID, manifest); err != nil {
		return nil, fmt.Errorf("%w: %v", errInTotoDataLoss, err)
	}

	certificate, err := k.BuildCertificate(ctx, manifestID)
	if err != nil {
		return nil, fmt.Errorf("build certificate: %w", err)
	}
	if certificate.ManifestId != manifestID {
		return nil, fmt.Errorf(
			"%w: requested manifest %q produced certificate for %q",
			errInTotoDataLoss,
			manifestID,
			certificate.ManifestId,
		)
	}
	chainID := sdk.UnwrapSDKContext(ctx).ChainID()
	statement, err := certificateToInTotoStatement(certificate, chainID)
	if err != nil {
		return nil, err
	}
	return statement, nil
}

func certificateToInTotoStatement(certificate *types.ProvenanceCertificate, observedOnChainID string) (*types.InTotoStatementV1, error) {
	if certificate == nil {
		return nil, fmt.Errorf("certificate is required")
	}
	if observedOnChainID == "" {
		return nil, fmt.Errorf("observed_on_chain_id is required")
	}
	if certificate.SourceChainId == "" {
		return nil, fmt.Errorf("%w: certificate source_chain_id is required", errInTotoProjectionUnavailable)
	}
	if certificate.ManifestId == "" {
		return nil, fmt.Errorf("%w: certificate manifest_id is required", errInTotoDataLoss)
	}

	root, err := hex.DecodeString(certificate.MerkleRoot)
	if err != nil || len(root) != 32 {
		return nil, fmt.Errorf("%w: manifest merkle_root must be a 32-byte SHA-256 hex digest", errInTotoDataLoss)
	}
	canonicalRoot := strings.ToLower(certificate.MerkleRoot)

	return &types.InTotoStatementV1{
		StatementType: InTotoStatementType,
		Subject: []*types.InTotoSubject{
			{
				Name: fmt.Sprintf(
					"zerone://%s/training-corpus/%s",
					url.PathEscape(certificate.SourceChainId),
					url.PathEscape(certificate.ManifestId),
				),
				Digest: map[string]string{"sha256": canonicalRoot},
			},
		},
		PredicateType: TrainingProvenancePredicateType,
		Predicate: &types.TrainingProvenancePredicate{
			SourceChainId:     certificate.SourceChainId,
			ObservedOnChainId: observedOnChainID,
			Certificate:       certificate,
		},
	}, nil
}
