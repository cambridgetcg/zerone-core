package verification

import (
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/fulcio/certificate"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/verify"
)

const (
	ComponentVerificationSchema = "zerone.component-signature-verification/v1"
	ComponentBundleMediaType    = "application/vnd.dev.sigstore.bundle.v0.3+json"
	componentBundleVersion      = "v0.3"
)

// ComponentPolicy is the complete offline verification policy for a release
// component's Sigstore message-signature bundle. The trusted root and identity
// are ceremony inputs; no network or ambient TUF state is consulted.
type ComponentPolicy struct {
	BundlePath        string
	TrustedRootPath   string
	CertificateIssuer string
	CertificateSAN    string
	// SourceRepositoryDigest is the exact Git commit from Fulcio's
	// sourceRepositoryDigest certificate extension.
	SourceRepositoryDigest string
	ArtifactDigest         string
}

// ComponentTimestamp is an authenticated observer timestamp returned by
// sigstore-go: either Rekor v1 SET time or an RFC 3161 TSA countersignature.
type ComponentTimestamp struct {
	Type      string `json:"type"`
	URI       string `json:"uri"`
	Timestamp string `json:"timestamp"`
}

// ComponentVerification is deliberately narrow and stable so the Python
// authority-chain verifier can consume it as a fail-closed subprocess result.
type ComponentVerification struct {
	Schema                 string               `json:"schema"`
	Result                 string               `json:"result"`
	ArtifactDigest         string               `json:"artifact_digest"`
	BundleMediaType        string               `json:"bundle_media_type"`
	CertificateIssuer      string               `json:"certificate_issuer"`
	CertificateSAN         string               `json:"certificate_san"`
	SourceRepositoryDigest string               `json:"source_repository_digest"`
	VerifiedTimestamps     []ComponentTimestamp `json:"verified_timestamps"`
}

// Validate checks all caller-controlled component verification inputs and
// returns the decoded SHA-256 digest.
func (p ComponentPolicy) Validate() ([]byte, error) {
	required := []struct {
		name  string
		value string
	}{
		{"bundle path", p.BundlePath},
		{"trusted-root path", p.TrustedRootPath},
		{"certificate issuer", p.CertificateIssuer},
		{"certificate SAN", p.CertificateSAN},
		{"source repository digest", p.SourceRepositoryDigest},
		{"artifact digest", p.ArtifactDigest},
	}
	if len(p.SourceRepositoryDigest) != 40 || p.SourceRepositoryDigest != strings.ToLower(p.SourceRepositoryDigest) {
		return nil, errors.New("source repository digest must be exactly 40 lowercase hex characters")
	}
	if _, err := hex.DecodeString(p.SourceRepositoryDigest); err != nil {
		return nil, errors.New("source repository digest must be exactly 40 lowercase hex characters")
	}
	for _, field := range required {
		if field.value == "" {
			return nil, fmt.Errorf("%s must be non-empty", field.name)
		}
		if field.value != strings.TrimSpace(field.value) {
			return nil, fmt.Errorf("%s must not have surrounding whitespace", field.name)
		}
	}

	const prefix = "sha256:"
	if !strings.HasPrefix(p.ArtifactDigest, prefix) {
		return nil, errors.New("artifact digest must use sha256:<64 lowercase hex>")
	}
	hexDigest := strings.TrimPrefix(p.ArtifactDigest, prefix)
	if len(hexDigest) != 64 || hexDigest != strings.ToLower(hexDigest) {
		return nil, errors.New("artifact digest must use sha256:<64 lowercase hex>")
	}
	digest, err := hex.DecodeString(hexDigest)
	if err != nil {
		return nil, fmt.Errorf("artifact digest must use sha256:<64 lowercase hex>: %w", err)
	}
	return digest, nil
}

// VerifyComponent verifies a v0.3 Sigstore message-signature bundle entirely
// offline. It requires a certificate SCT, a Rekor inclusion proof, and a
// cryptographically authenticated observer timestamp. It never falls back to
// the verifier host's current time.
func VerifyComponent(policy ComponentPolicy) (*ComponentVerification, error) {
	artifactDigest, err := policy.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid component verification policy: %w", err)
	}

	bundleJSON, err := readRegularFile(policy.BundlePath, maxBundleBytes)
	if err != nil {
		return nil, fmt.Errorf("read component bundle: %w", err)
	}
	loadedBundle := &bundle.Bundle{}
	if err := loadedBundle.UnmarshalJSON(bundleJSON); err != nil {
		return nil, fmt.Errorf("parse component bundle: %w", err)
	}
	if loadedBundle.GetMessageSignature() == nil || loadedBundle.GetDsseEnvelope() != nil {
		return nil, errors.New("component bundle must contain a messageSignature, not a DSSE envelope")
	}
	material := loadedBundle.GetVerificationMaterial()
	if material == nil || len(material.GetTlogEntries()) == 0 {
		return nil, errors.New("component bundle must contain a Rekor inclusion proof")
	}
	// Require the proof on every supplied entry. Requiring merely one proof
	// permits an untrusted-log entry to carry the syntactic proof while a
	// trusted SET-only entry satisfies the verifier's log/time thresholds.
	for index, entry := range material.GetTlogEntries() {
		if entry.GetInclusionProof() == nil {
			return nil, fmt.Errorf("component bundle Rekor entry %d has no inclusion proof", index)
		}
	}
	version, err := loadedBundle.Version()
	if err != nil {
		return nil, fmt.Errorf("read component bundle version: %w", err)
	}
	mediaType, err := bundle.MediaTypeString(version)
	if err != nil {
		return nil, fmt.Errorf("canonicalize component bundle media type: %w", err)
	}
	if version != componentBundleVersion || loadedBundle.GetMediaType() != ComponentBundleMediaType || mediaType != ComponentBundleMediaType {
		return nil, fmt.Errorf("component bundle media type must be exactly %q", ComponentBundleMediaType)
	}

	trustedRootJSON, err := readRegularFile(policy.TrustedRootPath, maxTrustedRootBytes)
	if err != nil {
		return nil, fmt.Errorf("read component trusted root: %w", err)
	}
	trustedRoot, err := root.NewTrustedRootFromJSON(trustedRootJSON)
	if err != nil {
		return nil, fmt.Errorf("parse component trusted root: %w", err)
	}

	sanMatcher, err := verify.NewSANMatcher(policy.CertificateSAN, "")
	if err != nil {
		return nil, fmt.Errorf("configure exact component certificate SAN: %w", err)
	}
	issuerMatcher, err := verify.NewIssuerMatcher(policy.CertificateIssuer, "")
	if err != nil {
		return nil, fmt.Errorf("configure exact component certificate issuer: %w", err)
	}
	certificateIdentity, err := verify.NewCertificateIdentity(
		sanMatcher,
		issuerMatcher,
		certificate.Extensions{SourceRepositoryDigest: policy.SourceRepositoryDigest},
	)
	if err != nil {
		return nil, fmt.Errorf("configure exact component certificate identity: %w", err)
	}
	verifier, err := verify.NewVerifier(
		trustedRoot,
		verify.WithSignedCertificateTimestamps(1),
		verify.WithTransparencyLog(1),
		verify.WithObserverTimestamps(1),
	)
	if err != nil {
		return nil, fmt.Errorf("configure component Sigstore verifier: %w", err)
	}

	result, err := verifier.Verify(
		loadedBundle,
		verify.NewPolicy(
			verify.WithArtifactDigest("sha256", artifactDigest),
			verify.WithCertificateIdentity(certificateIdentity),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("verify component Sigstore bundle: %w", err)
	}
	if result == nil || result.Signature == nil || result.Signature.Certificate == nil {
		return nil, errors.New("component verifier returned no verified signing certificate")
	}
	if result.VerifiedIdentity == nil {
		return nil, errors.New("component verifier returned no verified certificate identity")
	}

	verifiedTimestamps := make([]ComponentTimestamp, 0, len(result.VerifiedTimestamps))
	for _, timestamp := range result.VerifiedTimestamps {
		if timestamp.Type != "Tlog" && timestamp.Type != "TimestampAuthority" {
			continue
		}
		if timestamp.Timestamp.IsZero() || timestamp.URI == "" {
			return nil, errors.New("component verifier returned an incomplete observer timestamp")
		}
		verifiedTimestamps = append(verifiedTimestamps, ComponentTimestamp{
			Type:      timestamp.Type,
			URI:       timestamp.URI,
			Timestamp: timestamp.Timestamp.UTC().Format(time.RFC3339Nano),
		})
	}
	if len(verifiedTimestamps) == 0 {
		return nil, errors.New("component verifier returned no authenticated observer timestamp")
	}
	sort.Slice(verifiedTimestamps, func(i, j int) bool {
		if verifiedTimestamps[i].Timestamp == verifiedTimestamps[j].Timestamp {
			if verifiedTimestamps[i].Type == verifiedTimestamps[j].Type {
				return verifiedTimestamps[i].URI < verifiedTimestamps[j].URI
			}
			return verifiedTimestamps[i].Type < verifiedTimestamps[j].Type
		}
		return verifiedTimestamps[i].Timestamp < verifiedTimestamps[j].Timestamp
	})

	return &ComponentVerification{
		Schema:                 ComponentVerificationSchema,
		Result:                 "verified",
		ArtifactDigest:         policy.ArtifactDigest,
		BundleMediaType:        mediaType,
		CertificateIssuer:      policy.CertificateIssuer,
		CertificateSAN:         policy.CertificateSAN,
		SourceRepositoryDigest: policy.SourceRepositoryDigest,
		VerifiedTimestamps:     verifiedTimestamps,
	}, nil
}
