package keeper

import (
	"context"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	"github.com/zerone-chain/zerone/x/training_provenance/types"
)

func TestCertificateToInTotoStatement(t *testing.T) {
	root := "7e4a9b03c4d6f8e1023456789abcdef07e4a9b03c4d6f8e1023456789abcdef0"
	certificate := &types.ProvenanceCertificate{
		ManifestId:    "manifest/a",
		MerkleRoot:    root,
		TrustGrade:    "A",
		SourceChainId: "zerone-origin-1",
	}

	statement, err := certificateToInTotoStatement(certificate, "zerone-observer-2")
	require.NoError(t, err)
	require.Equal(t, InTotoStatementType, statement.StatementType)
	require.Equal(t, TrainingProvenancePredicateType, statement.PredicateType)
	require.NotContains(t, statement.PredicateType, "/blob/main/")
	require.Len(t, statement.Subject, 1)
	require.Equal(t, "zerone://zerone-origin-1/training-corpus/manifest%2Fa", statement.Subject[0].Name)
	require.Equal(t, root, statement.Subject[0].Digest["sha256"])
	require.Equal(t, "zerone-origin-1", statement.Predicate.SourceChainId)
	require.Equal(t, "zerone-observer-2", statement.Predicate.ObservedOnChainId)
	require.Same(t, certificate, statement.Predicate.Certificate)

	jsonStatement, err := protojson.Marshal(statement)
	require.NoError(t, err)
	require.JSONEq(t, `{
		"_type": "https://in-toto.io/Statement/v1",
		"subject": [{
			"name": "zerone://zerone-origin-1/training-corpus/manifest%2Fa",
			"digest": {"sha256": "7e4a9b03c4d6f8e1023456789abcdef07e4a9b03c4d6f8e1023456789abcdef0"}
		}],
		"predicateType": "https://github.com/cambridgetcg/zerone-core/blob/394bbef01df1b131223b1e874d554932d8dcd87c/docs/specs/attestations/training-provenance-v1.md",
		"predicate": {
			"sourceChainId": "zerone-origin-1",
			"observedOnChainId": "zerone-observer-2",
			"certificate": {
				"manifestId": "manifest/a",
				"merkleRoot": "7e4a9b03c4d6f8e1023456789abcdef07e4a9b03c4d6f8e1023456789abcdef0",
				"trustGrade": "A",
				"sourceChainId": "zerone-origin-1"
			}
		}
	}`, string(jsonStatement))
}

func TestCertificateToInTotoStatementRejectsInvalidDigest(t *testing.T) {
	_, err := certificateToInTotoStatement(&types.ProvenanceCertificate{
		ManifestId:    "manifest",
		MerkleRoot:    "not-a-sha256",
		SourceChainId: "zerone-1",
	}, "zerone-1")
	require.ErrorContains(t, err, "32-byte SHA-256")
}

type inTotoKnowledgeStub struct {
	manifests map[string]*knowledgetypes.TrainingManifest
}

func (s *inTotoKnowledgeStub) GetTrainingManifest(_ context.Context, id string) (*knowledgetypes.TrainingManifest, bool) {
	manifest, ok := s.manifests[id]
	return manifest, ok
}

func (*inTotoKnowledgeStub) GetFact(context.Context, string) (*knowledgetypes.Fact, bool) {
	return nil, false
}

func (*inTotoKnowledgeStub) IteratePrivilegedActions(context.Context, func(*knowledgetypes.PrivilegedAction) bool) {
}

func (*inTotoKnowledgeStub) IterateIncidents(context.Context, func(*knowledgetypes.IncidentRecord) bool) {
}

func newInTotoQueryTestServer(manifests map[string]*knowledgetypes.TrainingManifest) (types.QueryServer, context.Context) {
	k := NewKeeper(nil)
	k.SetKnowledgeKeeper(&inTotoKnowledgeStub{manifests: manifests})
	ctx := sdk.WrapSDKContext(
		sdk.Context{}.
			WithChainID("zerone-observer-1").
			WithBlockHeight(77),
	)
	return NewQueryServerImpl(k), ctx
}

func emptyFlatManifestRoot() string {
	return knowledgekeeper.ComputeManifestMerkleRoot(knowledgekeeper.SelectedManifestIDs{})
}

func TestQueryInTotoStatementStatusCodes(t *testing.T) {
	validRoot := emptyFlatManifestRoot()
	query, ctx := newInTotoQueryTestServer(map[string]*knowledgetypes.TrainingManifest{
		"draft": {
			ManifestId: "draft",
			ChainId:    "zerone-origin-1",
			Status:     knowledgetypes.ManifestStatus_MANIFEST_STATUS_DRAFT,
		},
		"composed": {
			ManifestId:       "composed",
			ChainId:          "zerone-origin-1",
			MerkleRoot:       validRoot,
			ParentManifestId: "parent",
			ParentMerkleRoot: validRoot,
			CompositionDepth: 1,
			Status:           knowledgetypes.ManifestStatus_MANIFEST_STATUS_FINALIZED,
		},
		"corrupt-root": {
			ManifestId: "corrupt-root",
			ChainId:    "zerone-origin-1",
			MerkleRoot: "not-a-sha256",
			Status:     knowledgetypes.ManifestStatus_MANIFEST_STATUS_FINALIZED,
		},
		"wrong-root": {
			ManifestId: "wrong-root",
			ChainId:    "zerone-origin-1",
			MerkleRoot: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Status:     knowledgetypes.ManifestStatus_MANIFEST_STATUS_FINALIZED,
		},
		"legacy-empty-source": {
			ManifestId: "legacy-empty-source",
			MerkleRoot: validRoot,
			Status:     knowledgetypes.ManifestStatus_MANIFEST_STATUS_FINALIZED,
		},
		"mismatched-id": {
			ManifestId: "different-id",
			ChainId:    "zerone-origin-1",
			MerkleRoot: validRoot,
			Status:     knowledgetypes.ManifestStatus_MANIFEST_STATUS_FINALIZED,
		},
		"orphan-parent-root": {
			ManifestId:       "orphan-parent-root",
			ChainId:          "zerone-origin-1",
			MerkleRoot:       validRoot,
			ParentMerkleRoot: validRoot,
			Status:           knowledgetypes.ManifestStatus_MANIFEST_STATUS_FINALIZED,
		},
		"orphan-depth": {
			ManifestId:       "orphan-depth",
			ChainId:          "zerone-origin-1",
			MerkleRoot:       validRoot,
			CompositionDepth: 1,
			Status:           knowledgetypes.ManifestStatus_MANIFEST_STATUS_FINALIZED,
		},
	})

	for name, req := range map[string]*types.QueryInTotoStatementRequest{
		"nil request": nil,
		"empty id":    {},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := query.InTotoStatement(ctx, req)
			require.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}

	_, err := query.InTotoStatement(ctx, &types.QueryInTotoStatementRequest{ManifestId: "missing"})
	require.Equal(t, codes.NotFound, status.Code(err))

	for _, manifestID := range []string{"draft", "composed", "legacy-empty-source"} {
		t.Run(manifestID, func(t *testing.T) {
			_, err := query.InTotoStatement(ctx, &types.QueryInTotoStatementRequest{ManifestId: manifestID})
			require.Equal(t, codes.FailedPrecondition, status.Code(err))
		})
	}

	for _, manifestID := range []string{"corrupt-root", "wrong-root"} {
		t.Run(manifestID, func(t *testing.T) {
			_, err := query.InTotoStatement(ctx, &types.QueryInTotoStatementRequest{ManifestId: manifestID})
			require.Equal(t, codes.DataLoss, status.Code(err))
			require.ErrorContains(t, err, "data loss")
			require.ErrorContains(t, err, "included ID sets")
		})
	}

	_, err = query.InTotoStatement(ctx, &types.QueryInTotoStatementRequest{ManifestId: "mismatched-id"})
	require.Equal(t, codes.DataLoss, status.Code(err))
	require.ErrorContains(t, err, "data loss")

	for _, manifestID := range []string{"orphan-parent-root", "orphan-depth"} {
		t.Run(manifestID, func(t *testing.T) {
			_, err := query.InTotoStatement(ctx, &types.QueryInTotoStatementRequest{ManifestId: manifestID})
			require.Equal(t, codes.DataLoss, status.Code(err))
			require.ErrorContains(t, err, "inconsistent composition metadata")
		})
	}
}

func TestQueryInTotoStatementAcceptsFlatSealedStatuses(t *testing.T) {
	validRoot := emptyFlatManifestRoot()
	accepted := []knowledgetypes.ManifestStatus{
		knowledgetypes.ManifestStatus_MANIFEST_STATUS_FINALIZED,
		knowledgetypes.ManifestStatus_MANIFEST_STATUS_ATTESTED,
		knowledgetypes.ManifestStatus_MANIFEST_STATUS_SUPERSEDED,
	}

	for _, manifestStatus := range accepted {
		t.Run(manifestStatus.String(), func(t *testing.T) {
			query, ctx := newInTotoQueryTestServer(map[string]*knowledgetypes.TrainingManifest{
				"sealed": {
					ManifestId: "sealed",
					ChainId:    "zerone-origin-1",
					MerkleRoot: validRoot,
					Status:     manifestStatus,
				},
			})

			statement, err := query.InTotoStatement(ctx, &types.QueryInTotoStatementRequest{ManifestId: "sealed"})
			require.NoError(t, err)
			require.Equal(t, validRoot, statement.Subject[0].Digest["sha256"])
			require.Equal(t, "zerone-origin-1", statement.Predicate.SourceChainId)
			require.Equal(t, "zerone-observer-1", statement.Predicate.ObservedOnChainId)
		})
	}
}

func TestProvenanceQueriesValidateAllIncludedIDClasses(t *testing.T) {
	includedIDs := knowledgekeeper.SelectedManifestIDs{
		FactIDs:                []string{"fact-1"},
		TraceIDs:               []string{"trace-1"},
		PairIDs:                []string{"pair-1"},
		DriftAugmentationIDs:   []string{"drift-1"},
		NormativeCommitmentIDs: []string{"normative-1"},
	}
	validRoot := knowledgekeeper.ComputeManifestMerkleRoot(includedIDs)
	query, ctx := newInTotoQueryTestServer(map[string]*knowledgetypes.TrainingManifest{
		"all-ids": {
			ManifestId:                     "all-ids",
			ChainId:                        "zerone-origin-1",
			MerkleRoot:                     validRoot,
			IncludedFactIds:                includedIDs.FactIDs,
			IncludedTraceIds:               includedIDs.TraceIDs,
			IncludedPairIds:                includedIDs.PairIDs,
			IncludedDriftAugmentationIds:   includedIDs.DriftAugmentationIDs,
			IncludedNormativeCommitmentIds: includedIDs.NormativeCommitmentIDs,
			Status:                         knowledgetypes.ManifestStatus_MANIFEST_STATUS_FINALIZED,
		},
		"mutated": {
			ManifestId:                     "mutated",
			ChainId:                        "zerone-origin-1",
			MerkleRoot:                     validRoot,
			IncludedFactIds:                includedIDs.FactIDs,
			IncludedTraceIds:               includedIDs.TraceIDs,
			IncludedPairIds:                includedIDs.PairIDs,
			IncludedDriftAugmentationIds:   includedIDs.DriftAugmentationIDs,
			IncludedNormativeCommitmentIds: []string{"normative-1", "normative-2"},
			Status:                         knowledgetypes.ManifestStatus_MANIFEST_STATUS_FINALIZED,
		},
	})

	statement, err := query.InTotoStatement(
		ctx,
		&types.QueryInTotoStatementRequest{ManifestId: "all-ids"},
	)
	require.NoError(t, err)
	require.Equal(t, validRoot, statement.Subject[0].Digest["sha256"])

	certificate, err := query.ProvenanceCertificate(
		ctx,
		&types.QueryProvenanceCertificateRequest{ManifestId: "all-ids"},
	)
	require.NoError(t, err)
	require.Equal(t, validRoot, certificate.Certificate.MerkleRoot)

	_, err = query.InTotoStatement(
		ctx,
		&types.QueryInTotoStatementRequest{ManifestId: "mutated"},
	)
	require.Equal(t, codes.DataLoss, status.Code(err))

	_, err = query.ProvenanceCertificate(
		ctx,
		&types.QueryProvenanceCertificateRequest{ManifestId: "mutated"},
	)
	require.Equal(t, codes.DataLoss, status.Code(err))
}

func TestQueryInTotoStatementUnwiredKeeperIsInternal(t *testing.T) {
	query := NewQueryServerImpl(NewKeeper(nil))
	_, err := query.InTotoStatement(
		context.Background(),
		&types.QueryInTotoStatementRequest{ManifestId: "manifest"},
	)
	require.Equal(t, codes.Internal, status.Code(err))
	require.ErrorContains(t, err, "knowledge keeper not wired")
}

func TestQueryInTotoStatementEmptyObservedChainIsInternal(t *testing.T) {
	validRoot := emptyFlatManifestRoot()
	k := NewKeeper(nil)
	k.SetKnowledgeKeeper(&inTotoKnowledgeStub{manifests: map[string]*knowledgetypes.TrainingManifest{
		"manifest": {
			ManifestId: "manifest",
			ChainId:    "zerone-origin-1",
			MerkleRoot: validRoot,
			Status:     knowledgetypes.ManifestStatus_MANIFEST_STATUS_FINALIZED,
		},
	}})
	ctx := sdk.WrapSDKContext(sdk.Context{}.WithBlockHeight(77))

	_, err := NewQueryServerImpl(k).InTotoStatement(
		ctx,
		&types.QueryInTotoStatementRequest{ManifestId: "manifest"},
	)
	require.Equal(t, codes.Internal, status.Code(err))
	require.ErrorContains(t, err, "observed_on_chain_id")
}

func TestQueryProvenanceCertificateStatusCodes(t *testing.T) {
	query, ctx := newInTotoQueryTestServer(map[string]*knowledgetypes.TrainingManifest{
		"valid": {
			ManifestId: "valid",
			ChainId:    "zerone-origin-1",
			MerkleRoot: emptyFlatManifestRoot(),
			Status:     knowledgetypes.ManifestStatus_MANIFEST_STATUS_FINALIZED,
		},
		"mismatched-id": {
			ManifestId: "different-id",
			ChainId:    "zerone-origin-1",
			MerkleRoot: emptyFlatManifestRoot(),
			Status:     knowledgetypes.ManifestStatus_MANIFEST_STATUS_FINALIZED,
		},
		"wrong-root": {
			ManifestId: "wrong-root",
			ChainId:    "zerone-origin-1",
			MerkleRoot: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Status:     knowledgetypes.ManifestStatus_MANIFEST_STATUS_FINALIZED,
		},
	})

	for name, req := range map[string]*types.QueryProvenanceCertificateRequest{
		"nil request": nil,
		"empty id":    {},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := query.ProvenanceCertificate(ctx, req)
			require.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}

	_, err := query.ProvenanceCertificate(ctx, &types.QueryProvenanceCertificateRequest{ManifestId: "missing"})
	require.Equal(t, codes.NotFound, status.Code(err))

	_, err = query.ProvenanceCertificate(ctx, &types.QueryProvenanceCertificateRequest{ManifestId: "mismatched-id"})
	require.Equal(t, codes.DataLoss, status.Code(err))
	require.ErrorContains(t, err, `manifest key "mismatched-id" contains ID "different-id"`)

	_, err = query.ProvenanceCertificate(ctx, &types.QueryProvenanceCertificateRequest{ManifestId: "wrong-root"})
	require.Equal(t, codes.DataLoss, status.Code(err))
	require.ErrorContains(t, err, "included ID sets")

	response, err := query.ProvenanceCertificate(ctx, &types.QueryProvenanceCertificateRequest{ManifestId: "valid"})
	require.NoError(t, err)
	require.Equal(t, "valid", response.Certificate.ManifestId)

	unwired := NewQueryServerImpl(NewKeeper(nil))
	_, err = unwired.ProvenanceCertificate(
		context.Background(),
		&types.QueryProvenanceCertificateRequest{ManifestId: "valid"},
	)
	require.Equal(t, codes.Internal, status.Code(err))
	require.ErrorContains(t, err, "knowledge keeper not wired")
}
