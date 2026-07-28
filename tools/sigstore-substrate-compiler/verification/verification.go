// Package verification applies Zerone's fail-closed Sigstore policy to a
// local bundle and returns the exact accepted bundle and verified DSSE payload.
package verification

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	intoto "github.com/in-toto/attestation/go/v1"
	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/verify"
	"golang.org/x/sys/unix"
)

const (
	MinBundleVersion = "0.3"
	StatementTypeV1  = "https://in-toto.io/Statement/v1"

	maxBundleBytes      = 16 << 20
	maxTrustedRootBytes = 4 << 20
	maxPayloadBytes     = 8 << 20
)

// Policy contains every caller-controlled verification decision. Issuer and
// SAN are exact values; regex identity matching is intentionally unavailable.
type Policy struct {
	BundlePath        string
	TrustedRootPath   string
	CertificateIssuer string
	CertificateSAN    string
	ArtifactDigest    string
	PredicateType     string
}

// VerifiedAttestation is the narrow boundary between crypto verification and
// deterministic substrate-link compilation. Its proof and payload bytes are
// intentionally private so callers cannot pair an unrelated bundle and
// payload and present that pair to the public compiler API.
type VerifiedAttestation struct {
	bundleJSON  []byte
	dssePayload []byte
}

// CompilationMaterial returns defensive copies of the exact proof bundle and
// signed payload accepted by VerifyBundle. A VerifiedAttestation cannot be
// populated outside this package.
func (v *VerifiedAttestation) CompilationMaterial() (bundleJSON, dssePayload []byte, err error) {
	if v == nil || len(v.bundleJSON) == 0 || len(v.dssePayload) == 0 {
		return nil, nil, errors.New("verified attestation is empty")
	}
	return append([]byte(nil), v.bundleJSON...), append([]byte(nil), v.dssePayload...), nil
}

// Validate checks the complete fail-closed policy and returns its decoded
// sha256 artifact digest.
func (p Policy) Validate() ([]byte, error) {
	required := []struct {
		name  string
		value string
	}{
		{"bundle path", p.BundlePath},
		{"trusted-root path", p.TrustedRootPath},
		{"certificate issuer", p.CertificateIssuer},
		{"certificate SAN", p.CertificateSAN},
		{"artifact digest", p.ArtifactDigest},
		{"predicate type", p.PredicateType},
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
		return nil, fmt.Errorf("artifact digest must use sha256:<64 lowercase hex>")
	}
	hexDigest := strings.TrimPrefix(p.ArtifactDigest, prefix)
	if len(hexDigest) != 64 || hexDigest != strings.ToLower(hexDigest) {
		return nil, fmt.Errorf("artifact digest must use sha256:<64 lowercase hex>")
	}
	digest, err := hex.DecodeString(hexDigest)
	if err != nil {
		return nil, fmt.Errorf("artifact digest must use sha256:<64 lowercase hex>: %w", err)
	}

	predicateURI, err := url.ParseRequestURI(p.PredicateType)
	if err != nil || predicateURI.Scheme == "" {
		return nil, fmt.Errorf("predicate type must be an absolute URI")
	}
	if predicateURI.User != nil {
		return nil, fmt.Errorf("predicate type must not contain credentials")
	}

	return digest, nil
}

// VerifyBundle verifies a local Sigstore bundle without TUF, network lookups,
// or current-time fallbacks. The local trusted-root bytes, exact certificate
// identity, artifact digest, and predicate type are all mandatory.
func VerifyBundle(policy Policy) (*VerifiedAttestation, error) {
	artifactDigest, err := policy.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid verification policy: %w", err)
	}

	bundleJSON, err := readRegularFile(policy.BundlePath, maxBundleBytes)
	if err != nil {
		return nil, fmt.Errorf("read bundle: %w", err)
	}
	loadedBundle := &bundle.Bundle{}
	if err := loadedBundle.UnmarshalJSON(bundleJSON); err != nil {
		return nil, fmt.Errorf("parse bundle: %w", err)
	}
	if !loadedBundle.MinVersion(MinBundleVersion) {
		version, versionErr := loadedBundle.Version()
		if versionErr != nil {
			return nil, fmt.Errorf("read bundle version: %w", versionErr)
		}
		return nil, fmt.Errorf("bundle version %s is below required v%s", version, MinBundleVersion)
	}

	envelope, err := loadedBundle.Envelope()
	if err != nil {
		return nil, fmt.Errorf("bundle must contain a DSSE envelope: %w", err)
	}
	if envelope.PayloadType != bundle.IntotoMediaType {
		return nil, fmt.Errorf("DSSE payload type must be %q, got %q", bundle.IntotoMediaType, envelope.PayloadType)
	}
	payload, err := envelope.DecodeB64Payload()
	if err != nil {
		return nil, fmt.Errorf("decode DSSE payload: %w", err)
	}
	if len(payload) == 0 {
		return nil, errors.New("decoded DSSE payload is empty")
	}
	if len(payload) > maxPayloadBytes {
		return nil, fmt.Errorf("decoded DSSE payload exceeds %d-byte limit", maxPayloadBytes)
	}

	trustedRootJSON, err := readRegularFile(policy.TrustedRootPath, maxTrustedRootBytes)
	if err != nil {
		return nil, fmt.Errorf("read trusted root: %w", err)
	}
	trustedRoot, err := root.NewTrustedRootFromJSON(trustedRootJSON)
	if err != nil {
		return nil, fmt.Errorf("parse trusted root: %w", err)
	}

	certificateIdentity, err := verify.NewShortCertificateIdentity(
		policy.CertificateIssuer,
		"",
		policy.CertificateSAN,
		"",
	)
	if err != nil {
		return nil, fmt.Errorf("configure exact certificate identity: %w", err)
	}
	verifier, err := verify.NewVerifier(
		trustedRoot,
		verify.WithSignedCertificateTimestamps(1),
		verify.WithTransparencyLog(1),
		verify.WithObserverTimestamps(1),
	)
	if err != nil {
		return nil, fmt.Errorf("configure Sigstore verifier: %w", err)
	}

	result, err := verifier.Verify(
		loadedBundle,
		verify.NewPolicy(
			verify.WithArtifactDigest("sha256", artifactDigest),
			verify.WithCertificateIdentity(certificateIdentity),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("verify Sigstore bundle: %w", err)
	}
	if result == nil || result.Statement == nil {
		return nil, errors.New("verified bundle did not contain an in-toto statement")
	}
	if err := validateStatement(result.Statement, policy.PredicateType); err != nil {
		return nil, err
	}
	if result.VerifiedIdentity == nil {
		return nil, errors.New("Sigstore verifier returned no verified certificate identity")
	}

	return &VerifiedAttestation{
		bundleJSON:  append([]byte(nil), bundleJSON...),
		dssePayload: append([]byte(nil), payload...),
	}, nil
}

func validateStatement(statement *intoto.Statement, expectedPredicateType string) error {
	if statement == nil {
		return errors.New("verified bundle did not contain an in-toto statement")
	}
	if statement.Type != StatementTypeV1 {
		return fmt.Errorf("in-toto statement type must be exactly %q, got %q", StatementTypeV1, statement.Type)
	}
	if statement.PredicateType != expectedPredicateType {
		return fmt.Errorf("predicate type must be exactly %q, got %q", expectedPredicateType, statement.PredicateType)
	}
	if len(statement.Subject) == 0 {
		return errors.New("in-toto statement must contain at least one subject")
	}
	for index, subject := range statement.Subject {
		if subject == nil || len(subject.Digest) == 0 {
			return fmt.Errorf("in-toto subject %d must contain a digest", index)
		}
		for algorithm, digest := range subject.Digest {
			if strings.TrimSpace(algorithm) == "" || strings.TrimSpace(digest) == "" {
				return fmt.Errorf("in-toto subject %d contains an empty digest algorithm or value", index)
			}
		}
	}
	return nil
}

func readRegularFile(path string, limit int64) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%q is not a regular file", path)
	}

	// O_NONBLOCK prevents a path swapped to a FIFO between Stat and Open from
	// hanging the verifier indefinitely. Re-check the opened descriptor to
	// close that race for all non-regular file types.
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("open %q: invalid file descriptor", path)
	}
	defer file.Close()
	openedInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !openedInfo.Mode().IsRegular() {
		return nil, fmt.Errorf("%q is not a regular file", path)
	}

	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%q exceeds %d-byte limit", path, limit)
	}
	return data, nil
}
