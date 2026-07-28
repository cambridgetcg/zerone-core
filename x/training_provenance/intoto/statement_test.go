package intoto

import (
	"encoding/json"
	"math"
	"testing"

	attestationv1 "github.com/in-toto/attestation/go/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	provenancetypes "github.com/zerone-chain/zerone/x/training_provenance/types"
)

const testMerkleRoot = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestBuildStatement(t *testing.T) {
	t.Parallel()

	statement, err := BuildStatement("zerone-2", testCertificate())
	require.NoError(t, err)
	require.NoError(t, statement.Validate())
	require.Equal(t, attestationv1.StatementTypeUri, statement.Type)
	require.Equal(t, PredicateTypeURI, statement.PredicateType)
	require.Len(t, statement.Subject, 1)
	require.Equal(t, "training-manifest/manifest-7", statement.Subject[0].Name)
	require.Equal(t, testMerkleRoot, statement.Subject[0].Digest["sha256"])
	require.Equal(t, "cosmos:zerone-2", statement.Subject[0].Annotations.Fields["sourceChain"].GetStringValue())

	source := statement.Predicate.Fields["source"].GetStructValue()
	require.Equal(t, "cosmos:zerone-2", source.Fields["chain"].GetStringValue())
	require.Equal(t, "18446744073709551615", source.Fields["computedAtBlock"].GetStringValue())
	require.Equal(t, provenancetypes.ModuleName, source.Fields["module"].GetStringValue())

	manifest := statement.Predicate.Fields["manifest"].GetStructValue()
	require.Equal(t, "manifest-7", manifest.Fields["id"].GetStringValue())
	require.Equal(t, "9007199254740993", manifest.Fields["factCount"].GetStringValue())
	require.Equal(t, "finalized", manifest.Fields["status"].GetStringValue())

	coverage := statement.Predicate.Fields["domainCoverage"].GetListValue().Values
	require.Len(t, coverage, 2)
	require.Equal(t, "biology", coverage[0].GetStructValue().Fields["domain"].GetStringValue())
	require.Equal(t, "sciences", coverage[1].GetStructValue().Fields["domain"].GetStringValue())

	trust := statement.Predicate.Fields["trust"].GetStructValue()
	require.Equal(t, "C", trust.Fields["grade"].GetStringValue())
	require.Equal(t, "2", trust.Fields["signals"].GetStructValue().Fields["includedFactPrivilegedActionCount"].GetStringValue())
}

func TestBuildStatementIsStableAcrossDomainOrder(t *testing.T) {
	t.Parallel()

	first := testCertificate()
	second := testCertificate()
	second.Domains[0], second.Domains[1] = second.Domains[1], second.Domains[0]

	firstStatement, err := BuildStatement("zerone-2", first)
	require.NoError(t, err)
	secondStatement, err := BuildStatement("zerone-2", second)
	require.NoError(t, err)
	require.True(t, proto.Equal(firstStatement, secondStatement))

	firstJSON, err := MarshalStatementJSON(firstStatement)
	require.NoError(t, err)
	secondJSON, err := MarshalStatementJSON(secondStatement)
	require.NoError(t, err)
	require.Equal(t, firstJSON, secondJSON)

	var document map[string]any
	require.NoError(t, json.Unmarshal(firstJSON, &document))
	require.Equal(t, attestationv1.StatementTypeUri, document["_type"])
	require.Equal(t, PredicateTypeURI, document["predicateType"])
}

func TestMarshalStatementJSONRejectsLegacyStatementType(t *testing.T) {
	t.Parallel()

	statement, err := BuildStatement("zerone-2", testCertificate())
	require.NoError(t, err)
	statement.Type = "https://in-toto.io/Statement/v0.1"
	require.NoError(t, statement.Validate(), "upstream validator intentionally accepts the legacy type")
	_, err = MarshalStatementJSON(statement)
	require.Error(t, err)
}

func TestBuildStatementRejectsInvalidInputs(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		chain string
		cert  *provenancetypes.ProvenanceCertificate
	}{
		"nil certificate": {chain: "zerone-2"},
		"empty manifest": {
			chain: "zerone-2",
			cert:  &provenancetypes.ProvenanceCertificate{MerkleRoot: testMerkleRoot},
		},
		"empty chain": {
			chain: "",
			cert:  testCertificate(),
		},
		"invalid digest": {
			chain: "zerone-2",
			cert: func() *provenancetypes.ProvenanceCertificate {
				cert := testCertificate()
				cert.MerkleRoot = "not-a-sha256-digest"
				return cert
			}(),
		},
		"nil domain": {
			chain: "zerone-2",
			cert: func() *provenancetypes.ProvenanceCertificate {
				cert := testCertificate()
				cert.Domains = append(cert.Domains, nil)
				return cert
			}(),
		},
		"draft status": {
			chain: "zerone-2",
			cert: func() *provenancetypes.ProvenanceCertificate {
				cert := testCertificate()
				cert.Status = "MANIFEST_STATUS_DRAFT"
				return cert
			}(),
		},
		"unsupported trust grade": {
			chain: "zerone-2",
			cert: func() *provenancetypes.ProvenanceCertificate {
				cert := testCertificate()
				cert.TrustGrade = "D"
				return cert
			}(),
		},
		"uppercase digest": {
			chain: "zerone-2",
			cert: func() *provenancetypes.ProvenanceCertificate {
				cert := testCertificate()
				cert.MerkleRoot = "ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789"
				return cert
			}(),
		},
		"duplicate domain": {
			chain: "zerone-2",
			cert: func() *provenancetypes.ProvenanceCertificate {
				cert := testCertificate()
				cert.Domains = append(cert.Domains, &provenancetypes.DomainCoverage{Domain: "biology"})
				return cert
			}(),
		},
		"empty domain": {
			chain: "zerone-2",
			cert: func() *provenancetypes.ProvenanceCertificate {
				cert := testCertificate()
				cert.Domains[0].Domain = ""
				return cert
			}(),
		},
		"empty pipeline": {
			chain: "zerone-2",
			cert: func() *provenancetypes.ProvenanceCertificate {
				cert := testCertificate()
				cert.PipelineId = ""
				return cert
			}(),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := BuildStatement(tc.chain, tc.cert)
			require.Error(t, err)
		})
	}
}

func testCertificate() *provenancetypes.ProvenanceCertificate {
	return &provenancetypes.ProvenanceCertificate{
		ManifestId:            "manifest-7",
		PipelineId:            "pipeline-3",
		MerkleRoot:            testMerkleRoot,
		FactCount:             1<<53 + 1,
		FinalizedAtBlock:      41,
		Status:                "MANIFEST_STATUS_FINALIZED",
		PrivilegedActionCount: 2,
		IncidentCount:         1,
		CartelResolutionCount: 0,
		TrustGrade:            "C",
		TrustExplanation:      "yellow flags accumulating",
		ComputedAtBlock:       math.MaxUint64,
		Domains: []*provenancetypes.DomainCoverage{
			{Domain: "sciences", FactCount: 9, AvgQualifiedWeight: 800_000, ActiveVoterCount: 3},
			{Domain: "biology", FactCount: 2, AvgQualifiedWeight: 700_000, ActiveVoterCount: 2},
		},
	}
}
