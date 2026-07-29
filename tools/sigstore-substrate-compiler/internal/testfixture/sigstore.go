// Package testfixture builds a hermetic Sigstore environment for integration
// tests. It uses the public sigstore-go signing and test-CA APIs and never
// contacts Fulcio, Rekor, TUF, or a timestamp service.
package testfixture

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"

	ct "github.com/google/certificate-transparency-go"
	"github.com/google/certificate-transparency-go/tls"
	ctx509 "github.com/google/certificate-transparency-go/x509"
	ctx509util "github.com/google/certificate-transparency-go/x509util"
	ssldsse "github.com/secure-systems-lab/go-securesystemslib/dsse"
	protobundle "github.com/sigstore/protobuf-specs/gen/pb-go/bundle/v1"
	protorekor "github.com/sigstore/protobuf-specs/gen/pb-go/rekor/v1"
	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/sign"
	"github.com/sigstore/sigstore-go/pkg/testing/ca"
	"github.com/sigstore/sigstore-go/pkg/tlog"
)

const (
	CertificateIssuer = "https://issuer.fixture.invalid"
	CertificateSAN    = "fixture-signer@example.invalid"
	PredicateType     = "https://slsa.dev/provenance/v1"
	StatementType     = "https://in-toto.io/Statement/v1"
)

var oidcIssuerOID = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 57264, 1, 1}

// Material contains a mutually pinned bundle, trusted root, and exact policy.
type Material struct {
	BundleJSON      []byte
	TrustedRootJSON []byte
	Payload         []byte
	ArtifactDigest  string
}

// Write stores the fixture as bounded regular files, matching production use.
func (m *Material) Write(dir string) (bundlePath, trustedRootPath string, err error) {
	bundlePath = filepath.Join(dir, "fixture.sigstore.json")
	trustedRootPath = filepath.Join(dir, "trusted-root.json")
	if err := os.WriteFile(bundlePath, m.BundleJSON, 0o600); err != nil {
		return "", "", fmt.Errorf("write bundle: %w", err)
	}
	if err := os.WriteFile(trustedRootPath, m.TrustedRootJSON, 0o600); err != nil {
		return "", "", fmt.Errorf("write trusted root: %w", err)
	}
	return bundlePath, trustedRootPath, nil
}

// Generate returns a valid v0.3 DSSE bundle with a Fulcio certificate, an
// embedded SCT, and a Rekor v1 inclusion proof.
func Generate() (*Material, error) {
	return generate(true)
}

// GenerateWithoutSCT returns an otherwise valid fixture whose leaf
// certificate has no SCT. It exists solely to prove the verifier's SCT gate.
func GenerateWithoutSCT() (*Material, error) {
	return generate(false)
}

func generate(includeSCT bool) (*Material, error) {
	integratedTime := time.Now().UTC().Truncate(time.Second)

	virtualSigstore, err := ca.NewVirtualSigstore()
	if err != nil {
		return nil, fmt.Errorf("create virtual Rekor: %w", err)
	}
	fulcioRoot, fulcioRootKey, err := ca.GenerateRootCa()
	if err != nil {
		return nil, fmt.Errorf("create Fulcio root: %w", err)
	}
	fulcioIntermediate, fulcioIntermediateKey, err := ca.GenerateFulcioIntermediate(fulcioRoot, fulcioRootKey)
	if err != nil {
		return nil, fmt.Errorf("create Fulcio intermediate: %w", err)
	}
	ctLogKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("create CT log key: %w", err)
	}

	artifactHash := sha256.Sum256([]byte("zerone hermetic Sigstore fixture artifact"))
	artifactHex := hex.EncodeToString(artifactHash[:])
	payload, err := json.Marshal(struct {
		Type    string `json:"_type"`
		Subject []struct {
			Name   string            `json:"name"`
			Digest map[string]string `json:"digest"`
		} `json:"subject"`
		PredicateType string         `json:"predicateType"`
		Predicate     map[string]any `json:"predicate"`
	}{
		Type: StatementType,
		Subject: []struct {
			Name   string            `json:"name"`
			Digest map[string]string `json:"digest"`
		}{
			{
				Name:   "zerone-fixture-artifact",
				Digest: map[string]string{"sha256": artifactHex},
			},
		},
		PredicateType: PredicateType,
		Predicate:     map[string]any{"fixture": true},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal statement: %w", err)
	}

	certificateProvider := &localCertificateProvider{
		intermediate:    fulcioIntermediate,
		intermediateKey: fulcioIntermediateKey,
		ctLogKey:        ctLogKey,
		integratedTime:  integratedTime,
		includeSCT:      includeSCT,
	}
	transparency := &localTransparencyLog{
		virtualSigstore: virtualSigstore,
		integratedTime:  integratedTime,
	}
	keypair, err := sign.NewEphemeralKeypair(nil)
	if err != nil {
		return nil, fmt.Errorf("create signing key: %w", err)
	}
	protobufBundle, err := sign.Bundle(
		&sign.DSSEData{Data: payload, PayloadType: bundle.IntotoMediaType},
		keypair,
		sign.BundleOptions{
			CertificateProvider: certificateProvider,
			TransparencyLogs:    []sign.Transparency{transparency},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("sign bundle: %w", err)
	}
	loadedBundle, err := bundle.NewBundle(protobufBundle)
	if err != nil {
		return nil, fmt.Errorf("validate generated bundle: %w", err)
	}
	bundleJSON, err := loadedBundle.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshal generated bundle: %w", err)
	}

	ctLogID, err := logID(ctLogKey.Public())
	if err != nil {
		return nil, fmt.Errorf("calculate CT log ID: %w", err)
	}
	ctLogs := map[string]*root.TransparencyLog{
		hex.EncodeToString(ctLogID): {
			BaseURL:             "https://ctlog.fixture.invalid",
			ID:                  append([]byte(nil), ctLogID...),
			ValidityPeriodStart: integratedTime.Add(-time.Hour),
			ValidityPeriodEnd:   integratedTime.Add(time.Hour),
			HashFunc:            crypto.SHA256,
			PublicKey:           ctLogKey.Public(),
			SignatureHashFunc:   crypto.SHA256,
		},
	}
	rekorLogs, err := serializableRekorLogs(virtualSigstore.RekorLogs(), integratedTime)
	if err != nil {
		return nil, err
	}
	trustedRoot, err := root.NewTrustedRoot(
		root.TrustedRootMediaType01,
		[]root.CertificateAuthority{&root.FulcioCertificateAuthority{
			Root:                fulcioRoot,
			Intermediates:       []*x509.Certificate{fulcioIntermediate},
			ValidityPeriodStart: integratedTime.Add(-time.Hour),
			ValidityPeriodEnd:   integratedTime.Add(time.Hour),
			URI:                 "https://fulcio.fixture.invalid",
		}},
		ctLogs,
		nil,
		rekorLogs,
	)
	if err != nil {
		return nil, fmt.Errorf("create trusted root: %w", err)
	}
	trustedRootJSON, err := trustedRoot.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshal trusted root: %w", err)
	}

	return &Material{
		BundleJSON:      bundleJSON,
		TrustedRootJSON: trustedRootJSON,
		Payload:         payload,
		ArtifactDigest:  "sha256:" + artifactHex,
	}, nil
}

type localCertificateProvider struct {
	intermediate    *x509.Certificate
	intermediateKey *ecdsa.PrivateKey
	ctLogKey        *ecdsa.PrivateKey
	integratedTime  time.Time
	includeSCT      bool
}

func (p *localCertificateProvider) GetCertificate(_ context.Context, keypair sign.Keypair, _ *sign.CertificateProviderOptions) ([]byte, error) {
	template := &x509.Certificate{
		SerialNumber:   big.NewInt(42),
		EmailAddresses: []string{CertificateSAN},
		NotBefore:      p.integratedTime.Add(-time.Minute),
		NotAfter:       p.integratedTime.Add(10 * time.Minute),
		KeyUsage:       x509.KeyUsageDigitalSignature,
		ExtKeyUsage:    []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
		ExtraExtensions: []pkix.Extension{{
			Id:    oidcIssuerOID,
			Value: []byte(CertificateIssuer),
		}},
	}
	precertificateDER, err := x509.CreateCertificate(
		rand.Reader,
		template,
		p.intermediate,
		keypair.GetPublicKey(),
		p.intermediateKey,
	)
	if err != nil {
		return nil, fmt.Errorf("create leaf precertificate: %w", err)
	}
	if !p.includeSCT {
		return precertificateDER, nil
	}
	precertificate, err := x509.ParseCertificate(precertificateDER)
	if err != nil {
		return nil, fmt.Errorf("parse leaf precertificate: %w", err)
	}
	sctExtension, err := makeSCTExtension(precertificate, p.intermediate, p.ctLogKey, p.integratedTime)
	if err != nil {
		return nil, err
	}

	finalTemplate := *template
	finalTemplate.ExtraExtensions = append(
		append([]pkix.Extension(nil), template.ExtraExtensions...),
		sctExtension,
	)
	certificateDER, err := x509.CreateCertificate(
		rand.Reader,
		&finalTemplate,
		p.intermediate,
		keypair.GetPublicKey(),
		p.intermediateKey,
	)
	if err != nil {
		return nil, fmt.Errorf("create SCT-bearing leaf certificate: %w", err)
	}
	certificate, err := x509.ParseCertificate(certificateDER)
	if err != nil {
		return nil, fmt.Errorf("parse SCT-bearing leaf certificate: %w", err)
	}
	defangedTBS, err := ctx509.RemoveSCTList(certificate.RawTBSCertificate)
	if err != nil {
		return nil, fmt.Errorf("remove generated SCT list: %w", err)
	}
	if !bytes.Equal(defangedTBS, precertificate.RawTBSCertificate) {
		return nil, fmt.Errorf("generated SCT does not cover the final certificate")
	}
	return certificateDER, nil
}

func makeSCTExtension(precertificate, issuer *x509.Certificate, ctLogKey *ecdsa.PrivateKey, timestamp time.Time) (pkix.Extension, error) {
	id, err := logID(ctLogKey.Public())
	if err != nil {
		return pkix.Extension{}, fmt.Errorf("calculate SCT log ID: %w", err)
	}
	var keyID [sha256.Size]byte
	copy(keyID[:], id)
	sct := ct.SignedCertificateTimestamp{
		SCTVersion: ct.V1,
		LogID:      ct.LogID{KeyID: keyID},
		Timestamp:  uint64(timestamp.UnixMilli()),
	}
	entry := ct.LogEntry{
		Leaf: ct.MerkleTreeLeaf{
			Version:  ct.V1,
			LeafType: ct.TimestampedEntryLeafType,
			TimestampedEntry: &ct.TimestampedEntry{
				Timestamp: sct.Timestamp,
				EntryType: ct.PrecertLogEntryType,
				PrecertEntry: &ct.PreCert{
					IssuerKeyHash:  sha256.Sum256(issuer.RawSubjectPublicKeyInfo),
					TBSCertificate: precertificate.RawTBSCertificate,
				},
			},
		},
	}
	signatureInput, err := ct.SerializeSCTSignatureInput(sct, entry)
	if err != nil {
		return pkix.Extension{}, fmt.Errorf("serialize SCT signature input: %w", err)
	}
	digest := sha256.Sum256(signatureInput)
	signature, err := ecdsa.SignASN1(rand.Reader, ctLogKey, digest[:])
	if err != nil {
		return pkix.Extension{}, fmt.Errorf("sign SCT: %w", err)
	}
	sct.Signature = ct.DigitallySigned{
		Algorithm: tls.SignatureAndHashAlgorithm{
			Hash:      tls.SHA256,
			Signature: tls.ECDSA,
		},
		Signature: signature,
	}
	sctList, err := ctx509util.MarshalSCTsIntoSCTList([]*ct.SignedCertificateTimestamp{&sct})
	if err != nil {
		return pkix.Extension{}, fmt.Errorf("marshal SCT list: %w", err)
	}
	sctBytes, err := tls.Marshal(*sctList)
	if err != nil {
		return pkix.Extension{}, fmt.Errorf("TLS-marshal SCT list: %w", err)
	}
	extensionValue, err := asn1.Marshal(sctBytes)
	if err != nil {
		return pkix.Extension{}, fmt.Errorf("ASN.1-marshal SCT list: %w", err)
	}
	return pkix.Extension{
		Id:    asn1.ObjectIdentifier(ctx509.OIDExtensionCTSCT),
		Value: extensionValue,
	}, nil
}

type localTransparencyLog struct {
	virtualSigstore *ca.VirtualSigstore
	integratedTime  time.Time
}

func (l *localTransparencyLog) GetTransparencyLogEntry(_ context.Context, certificatePEM []byte, protobufBundle *protobundle.Bundle) error {
	block, _ := pem.Decode(certificatePEM)
	if block == nil || block.Type != "CERTIFICATE" {
		return fmt.Errorf("fixture transparency log received invalid certificate PEM")
	}
	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("parse fixture transparency certificate: %w", err)
	}
	envelope := protobufBundle.GetDsseEnvelope()
	if envelope == nil || len(envelope.GetSignatures()) != 1 {
		return fmt.Errorf("fixture transparency log requires one DSSE signature")
	}
	signature := envelope.GetSignatures()[0]
	rekorEnvelope := &ssldsse.Envelope{
		PayloadType: envelope.GetPayloadType(),
		Payload:     base64.StdEncoding.EncodeToString(envelope.GetPayload()),
		Signatures: []ssldsse.Signature{{
			KeyID: signature.GetKeyid(),
			Sig:   base64.StdEncoding.EncodeToString(signature.GetSig()),
		}},
	}
	entry, err := l.virtualSigstore.GenerateTlogEntry(
		certificate,
		rekorEnvelope,
		signature.GetSig(),
		l.integratedTime.Unix(),
		true,
	)
	if err != nil {
		return fmt.Errorf("create fixture Rekor entry: %w", err)
	}
	transparencyEntry := entry.TransparencyLogEntry()
	rekorLogID, err := l.virtualSigstore.RekorLogID()
	if err != nil {
		return fmt.Errorf("calculate fixture Rekor log ID: %w", err)
	}
	signedEntryTimestamp, err := l.virtualSigstore.RekorSignPayload(tlog.RekorPayload{
		Body:           base64.StdEncoding.EncodeToString(transparencyEntry.GetCanonicalizedBody()),
		IntegratedTime: transparencyEntry.GetIntegratedTime(),
		LogIndex:       transparencyEntry.GetLogIndex(),
		LogID:          rekorLogID,
	})
	if err != nil {
		return fmt.Errorf("sign fixture Rekor promise: %w", err)
	}
	transparencyEntry.KindVersion = &protorekor.KindVersion{
		Kind:    "dsse",
		Version: "0.0.1",
	}
	transparencyEntry.InclusionPromise = &protorekor.InclusionPromise{
		SignedEntryTimestamp: signedEntryTimestamp,
	}
	protobufBundle.VerificationMaterial.TlogEntries = append(
		protobufBundle.VerificationMaterial.TlogEntries,
		transparencyEntry,
	)
	return nil
}

func serializableRekorLogs(logs map[string]*root.TransparencyLog, integratedTime time.Time) (map[string]*root.TransparencyLog, error) {
	for encodedID, log := range logs {
		rawID, err := hex.DecodeString(encodedID)
		if err != nil {
			return nil, fmt.Errorf("decode fixture Rekor log ID: %w", err)
		}
		log.ID = rawID
		log.ValidityPeriodStart = integratedTime.Add(-time.Hour)
		log.ValidityPeriodEnd = integratedTime.Add(time.Hour)
	}
	return logs, nil
}

func logID(publicKey crypto.PublicKey) ([]byte, error) {
	publicKeyDER, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(publicKeyDER)
	return digest[:], nil
}
