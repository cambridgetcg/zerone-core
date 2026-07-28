// Package intoto projects Zerone training provenance into ecosystem-standard
// in-toto Statement v1 objects. It is query-side only and owns no state.
package intoto

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"

	attestationv1 "github.com/in-toto/attestation/go/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/zerone-chain/zerone/x/common/caip"
	provenancetypes "github.com/zerone-chain/zerone/x/training_provenance/types"
)

const (
	// PredicateTypeURI identifies Zerone's versioned training-provenance
	// predicate. Its schema is documented in
	// docs/specs/in-toto-training-provenance-v1.md.
	PredicateTypeURI = "https://zerone.money/attestations/training-provenance/v1"

	subjectNamePrefix = "training-manifest/"
)

var sha256DigestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// BuildStatement turns a live ProvenanceCertificate into an unsigned in-toto
// Statement v1. The chain ID is represented using the CAIP-2 Cosmos profile so
// relying parties can apply an explicit source-network policy. This field is
// metadata; it is not a cryptographic chain-origin proof.
//
// The result is deliberately not a DSSE envelope. A caller that needs an
// authenticated attestation should use the serialized JSON as the DSSE payload
// and sign DSSE's pre-authentication encoding of payload type and payload.
func BuildStatement(chainID string, cert *provenancetypes.ProvenanceCertificate) (*attestationv1.Statement, error) {
	if cert == nil {
		return nil, fmt.Errorf("provenance certificate is required")
	}
	if cert.ManifestId == "" {
		return nil, fmt.Errorf("manifest_id is required")
	}
	if cert.PipelineId == "" {
		return nil, fmt.Errorf("pipeline_id is required")
	}
	if !sha256DigestPattern.MatchString(cert.MerkleRoot) {
		return nil, fmt.Errorf("merkle_root must be 64 lowercase hexadecimal characters")
	}
	status, err := predicateManifestStatus(cert.Status)
	if err != nil {
		return nil, err
	}
	if !validTrustGrade(cert.TrustGrade) {
		return nil, fmt.Errorf("unsupported trust grade %q", cert.TrustGrade)
	}

	sourceChain, err := caip.CosmosChainID(chainID)
	if err != nil {
		return nil, err
	}

	domains := append([]*provenancetypes.DomainCoverage(nil), cert.Domains...)
	seenDomains := make(map[string]struct{}, len(domains))
	for i, domain := range domains {
		if domain == nil {
			return nil, fmt.Errorf("domain coverage at index %d is nil", i)
		}
		if domain.Domain == "" {
			return nil, fmt.Errorf("domain coverage at index %d has an empty domain", i)
		}
		if _, exists := seenDomains[domain.Domain]; exists {
			return nil, fmt.Errorf("duplicate domain coverage %q", domain.Domain)
		}
		seenDomains[domain.Domain] = struct{}{}
	}
	sort.SliceStable(domains, func(i, j int) bool {
		if domains[i].Domain != domains[j].Domain {
			return domains[i].Domain < domains[j].Domain
		}
		if domains[i].FactCount != domains[j].FactCount {
			return domains[i].FactCount < domains[j].FactCount
		}
		if domains[i].AvgQualifiedWeight != domains[j].AvgQualifiedWeight {
			return domains[i].AvgQualifiedWeight < domains[j].AvgQualifiedWeight
		}
		return domains[i].ActiveVoterCount < domains[j].ActiveVoterCount
	})

	domainCoverage := make([]any, 0, len(domains))
	for _, domain := range domains {
		domainCoverage = append(domainCoverage, map[string]any{
			"activeVoterCount":   decimal(uint64(domain.ActiveVoterCount)),
			"avgQualifiedWeight": decimal(domain.AvgQualifiedWeight),
			"domain":             domain.Domain,
			"factCount":          decimal(domain.FactCount),
		})
	}

	predicate, err := structpb.NewStruct(map[string]any{
		"source": map[string]any{
			"chain":           sourceChain,
			"computedAtBlock": decimal(cert.ComputedAtBlock),
			"module":          provenancetypes.ModuleName,
		},
		"manifest": map[string]any{
			"factCount":        decimal(cert.FactCount),
			"finalizedAtBlock": decimal(cert.FinalizedAtBlock),
			"id":               cert.ManifestId,
			"pipelineId":       cert.PipelineId,
			"status":           status,
		},
		"domainCoverage": domainCoverage,
		"trust": map[string]any{
			"explanation": cert.TrustExplanation,
			"grade":       cert.TrustGrade,
			"signals": map[string]any{
				"coveredDomainCartelResolutionCount": decimal(uint64(cert.CartelResolutionCount)),
				"includedFactPrivilegedActionCount":  decimal(uint64(cert.PrivilegedActionCount)),
				"knowledgeModuleIncidentCount":       decimal(uint64(cert.IncidentCount)),
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("build in-toto predicate: %w", err)
	}

	annotations, err := structpb.NewStruct(map[string]any{
		"sourceChain": sourceChain,
	})
	if err != nil {
		return nil, fmt.Errorf("build in-toto subject annotations: %w", err)
	}

	statement := &attestationv1.Statement{
		Type: attestationv1.StatementTypeUri,
		Subject: []*attestationv1.ResourceDescriptor{{
			Name:        subjectNamePrefix + cert.ManifestId,
			Digest:      map[string]string{attestationv1.AlgorithmSHA256.String(): cert.MerkleRoot},
			Annotations: annotations,
		}},
		PredicateType: PredicateTypeURI,
		Predicate:     predicate,
	}
	if err := statement.Validate(); err != nil {
		return nil, fmt.Errorf("invalid in-toto statement: %w", err)
	}
	return statement, nil
}

// MarshalStatementJSON validates and renders a Statement with the field names
// required by in-toto, including "_type" and "predicateType".
func MarshalStatementJSON(statement *attestationv1.Statement) ([]byte, error) {
	if statement == nil {
		return nil, fmt.Errorf("in-toto statement is required")
	}
	if statement.Type != attestationv1.StatementTypeUri {
		return nil, fmt.Errorf("expected in-toto Statement v1 type, got %q", statement.Type)
	}
	if err := statement.Validate(); err != nil {
		return nil, fmt.Errorf("invalid in-toto statement: %w", err)
	}
	bz, err := (protojson.MarshalOptions{Indent: "  "}).Marshal(statement)
	if err != nil {
		return nil, fmt.Errorf("marshal in-toto statement: %w", err)
	}
	return bz, nil
}

func predicateManifestStatus(status string) (string, error) {
	switch status {
	case "MANIFEST_STATUS_FINALIZED":
		return "finalized", nil
	case "MANIFEST_STATUS_ATTESTED":
		return "attested", nil
	case "MANIFEST_STATUS_SUPERSEDED":
		return "superseded", nil
	default:
		return "", fmt.Errorf("manifest status %q is not exportable", status)
	}
}

func validTrustGrade(grade string) bool {
	switch grade {
	case "A", "B", "C", "F":
		return true
	default:
		return false
	}
}

// structpb represents JSON numbers as float64. Decimal strings keep every
// uint64 exact across languages and beyond JavaScript's 53-bit integer range.
func decimal(value uint64) string {
	return strconv.FormatUint(value, 10)
}
