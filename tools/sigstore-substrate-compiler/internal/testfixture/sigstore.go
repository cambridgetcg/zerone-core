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

	"github.com/cyberphone/json-canonicalization/go/src/webpki.org/jsoncanonicalizer"
	"github.com/digitorus/timestamp"
	openapiruntime "github.com/go-openapi/runtime"
	ct "github.com/google/certificate-transparency-go"
	"github.com/google/certificate-transparency-go/tls"
	ctx509 "github.com/google/certificate-transparency-go/x509"
	ctx509util "github.com/google/certificate-transparency-go/x509util"
	ssldsse "github.com/secure-systems-lab/go-securesystemslib/dsse"
	protobundle "github.com/sigstore/protobuf-specs/gen/pb-go/bundle/v1"
	protocommon "github.com/sigstore/protobuf-specs/gen/pb-go/common/v1"
	protorekor "github.com/sigstore/protobuf-specs/gen/pb-go/rekor/v1"
	rekortilespb "github.com/sigstore/rekor-tiles/v2/pkg/generated/protobuf"
	rekornote "github.com/sigstore/rekor-tiles/v2/pkg/note"
	rekorv2hashedrekord "github.com/sigstore/rekor-tiles/v2/pkg/types/hashedrekord"
	"github.com/sigstore/rekor/pkg/generated/models"
	"github.com/sigstore/rekor/pkg/pki"
	rekortypes "github.com/sigstore/rekor/pkg/types"
	"github.com/sigstore/rekor/pkg/types/hashedrekord"
	"github.com/sigstore/sigstore-go/pkg/bundle"
	fulciocertificate "github.com/sigstore/sigstore-go/pkg/fulcio/certificate"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/sign"
	"github.com/sigstore/sigstore-go/pkg/testing/ca"
	"github.com/sigstore/sigstore-go/pkg/tlog"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	sigstoresignature "github.com/sigstore/sigstore/pkg/signature"
	f_log "github.com/transparency-dev/formats/log"
	"github.com/transparency-dev/merkle/rfc6962"
	"golang.org/x/mod/sumdb/note"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	CertificateIssuer      = "https://issuer.fixture.invalid"
	CertificateSAN         = "fixture-signer@example.invalid"
	SourceRepositoryDigest = "1111111111111111111111111111111111111111"
	PredicateType          = "https://slsa.dev/provenance/v1"
	StatementType          = "https://in-toto.io/Statement/v1"
	TimestampAuthorityURI  = "https://virtual.tsa.sigstore.dev"
	RekorV2URI             = "https://rekor-v2.fixture.invalid"
)

var oidcIssuerOID = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 57264, 1, 1}

func derStringExtension(oid asn1.ObjectIdentifier, value string) pkix.Extension {
	encoded, err := asn1.Marshal(value)
	if err != nil {
		// Encoding a Go string as a DER UTF8String cannot fail. Keep this helper
		// local to the hermetic test fixture so production code never panics on
		// certificate input.
		panic(err)
	}
	return pkix.Extension{Id: oid, Value: encoded}
}

// Material contains a mutually pinned bundle, trusted root, and exact policy.
type Material struct {
	BundleJSON      []byte
	TrustedRootJSON []byte
	Payload         []byte
	ArtifactDigest  string
	IntegratedTime  time.Time
	ObserverTime    time.Time
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
	return generate(true, false)
}

// GenerateWithoutSCT returns an otherwise valid fixture whose leaf
// certificate has no SCT. It exists solely to prove the verifier's SCT gate.
func GenerateWithoutSCT() (*Material, error) {
	return generate(false, false)
}

// GenerateMessageSignature returns a valid v0.3 message-signature bundle for
// a plain artifact. This is the shape used by the release-component verifier.
func GenerateMessageSignature() (*Material, error) {
	return generate(true, true)
}

// GenerateMessageSignatureWithoutSCT returns an otherwise valid plain
// artifact fixture without a certificate-transparency SCT.
func GenerateMessageSignatureWithoutSCT() (*Material, error) {
	return generate(false, true)
}

// GenerateMessageSignatureWithRFC3161RekorV2 returns the production-shaped
// component fixture: a message signature, a Rekor v2 inclusion proof with no
// integrated time or SET, and an RFC 3161 countersignature over the exact
// component signature.
func GenerateMessageSignatureWithRFC3161RekorV2() (*Material, error) {
	return generateWithOptions(true, true, true, true)
}

func generate(includeSCT, messageSignature bool) (*Material, error) {
	return generateWithOptions(includeSCT, messageSignature, false, false)
}

func generateWithOptions(includeSCT, messageSignature, includeRFC3161, rekorV2 bool) (*Material, error) {
	integratedTime := time.Now().UTC().Truncate(time.Second)

	virtualSigstore, err := ca.NewVirtualSigstore()
	if err != nil {
		return nil, fmt.Errorf("create virtual Sigstore services: %w", err)
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

	artifact := []byte("zerone hermetic Sigstore fixture artifact")
	artifactHash := sha256.Sum256(artifact)
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
	var (
		transparency sign.Transparency
		rekorV2Log   *localRekorV2
	)
	if rekorV2 {
		rekorV2Log, err = newLocalRekorV2()
		if err != nil {
			return nil, err
		}
		transparency = sign.NewRekor(&sign.RekorOptions{
			BaseURL:  RekorV2URI,
			ClientV2: rekorV2Log,
			Version:  2,
		})
	} else {
		transparency = &localTransparencyLog{
			virtualSigstore: virtualSigstore,
			integratedTime:  integratedTime,
			artifact:        artifact,
		}
	}
	keypair, err := sign.NewEphemeralKeypair(nil)
	if err != nil {
		return nil, fmt.Errorf("create signing key: %w", err)
	}
	var content sign.Content = &sign.DSSEData{Data: payload, PayloadType: bundle.IntotoMediaType}
	materialPayload := payload
	if messageSignature {
		content = &sign.PlainData{Data: artifact}
		materialPayload = artifact
	}
	protobufBundle, err := sign.Bundle(
		content,
		keypair,
		sign.BundleOptions{
			CertificateProvider: certificateProvider,
			TransparencyLogs:    []sign.Transparency{transparency},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("sign bundle: %w", err)
	}
	var (
		observerTime         time.Time
		timestampAuthorities []root.TimestampingAuthority
	)
	if includeRFC3161 {
		message := protobufBundle.GetMessageSignature()
		if message == nil || len(message.GetSignature()) == 0 {
			return nil, fmt.Errorf("RFC 3161 fixture requires a message signature")
		}
		timestampResponse, err := virtualSigstore.TimestampResponse(message.GetSignature())
		if err != nil {
			return nil, fmt.Errorf("create fixture RFC 3161 timestamp: %w", err)
		}
		parsedTimestamp, err := timestamp.ParseResponse(timestampResponse)
		if err != nil {
			return nil, fmt.Errorf("parse fixture RFC 3161 timestamp: %w", err)
		}
		observerTime = parsedTimestamp.Time.UTC()
		protobufBundle.VerificationMaterial.TimestampVerificationData = &protobundle.TimestampVerificationData{
			Rfc3161Timestamps: []*protocommon.RFC3161SignedTimestamp{{
				SignedTimestamp: timestampResponse,
			}},
		}
		timestampAuthorities = virtualSigstore.TimestampingAuthorities()
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
	var rekorLogs map[string]*root.TransparencyLog
	if rekorV2Log != nil {
		rekorLogs, err = rekorV2Log.trustedLogs(integratedTime)
	} else {
		rekorLogs, err = serializableRekorLogs(virtualSigstore.RekorLogs(), integratedTime)
		if err != nil {
			return nil, err
		}
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
		timestampAuthorities,
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
		Payload:         materialPayload,
		ArtifactDigest:  "sha256:" + artifactHex,
		IntegratedTime:  integratedTime,
		ObserverTime:    observerTime,
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
		ExtraExtensions: []pkix.Extension{
			{Id: oidcIssuerOID, Value: []byte(CertificateIssuer)},
			derStringExtension(fulciocertificate.OIDSourceRepositoryDigest, SourceRepositoryDigest),
		},
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

// localRekorV2 is the server side of sign.Rekor's injected v2 client. It
// validates the request exactly as Rekor v2 does, persists the canonical entry
// bytes in the returned bundle entry, and signs a one-leaf checkpoint. It has
// no clock field because Rekor v2 does not authenticate integrated time.
type localRekorV2 struct {
	key *ecdsa.PrivateKey
}

func newLocalRekorV2() (*localRekorV2, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("create fixture Rekor v2 key: %w", err)
	}
	return &localRekorV2{key: key}, nil
}

func (l *localRekorV2) Add(ctx context.Context, request any) (*protorekor.TransparencyLogEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	hashedRekordRequest, ok := request.(*rekortilespb.HashedRekordRequestV002)
	if !ok {
		return nil, fmt.Errorf("fixture Rekor v2 requires a hashedrekord v0.0.2 request, got %T", request)
	}
	if hashedRekordRequest.GetSignature() == nil || hashedRekordRequest.GetSignature().GetVerifier() == nil {
		return nil, fmt.Errorf("fixture Rekor v2 request has no signature verifier")
	}
	algorithm := hashedRekordRequest.GetSignature().GetVerifier().GetKeyDetails()
	algorithmRegistry, err := sigstoresignature.NewAlgorithmRegistryConfig(
		[]protocommon.PublicKeyDetails{algorithm},
	)
	if err != nil {
		return nil, fmt.Errorf("configure fixture Rekor v2 algorithms: %w", err)
	}
	entry, err := rekorv2hashedrekord.ToLogEntry(hashedRekordRequest, algorithmRegistry)
	if err != nil {
		return nil, fmt.Errorf("validate fixture Rekor v2 entry: %w", err)
	}
	serializedEntry, err := protojson.Marshal(entry)
	if err != nil {
		return nil, fmt.Errorf("marshal fixture Rekor v2 entry: %w", err)
	}
	canonicalBody, err := jsoncanonicalizer.Transform(serializedEntry)
	if err != nil {
		return nil, fmt.Errorf("canonicalize fixture Rekor v2 entry: %w", err)
	}
	leafHash, err := rekorv2hashedrekord.ToEntryHash(
		hashedRekordRequest.GetDigest(),
		hashedRekordRequest.GetSignature(),
	)
	if err != nil {
		return nil, fmt.Errorf("hash fixture Rekor v2 entry: %w", err)
	}
	if canonicalHash := rfc6962.DefaultHasher.HashLeaf(canonicalBody); !bytes.Equal(canonicalHash, leafHash) {
		return nil, fmt.Errorf("fixture Rekor v2 canonical body and reconstructed hash differ")
	}

	signerVerifier, err := sigstoresignature.LoadECDSASignerVerifier(l.key, crypto.SHA256)
	if err != nil {
		return nil, fmt.Errorf("load fixture Rekor v2 signer: %w", err)
	}
	noteSigner, err := rekornote.NewNoteSigner(ctx, "rekor-v2.fixture.invalid", signerVerifier)
	if err != nil {
		return nil, fmt.Errorf("create fixture Rekor v2 note signer: %w", err)
	}
	checkpoint := f_log.Checkpoint{
		Origin: "rekor-v2.fixture.invalid",
		Size:   1,
		Hash:   leafHash,
	}.Marshal()
	signedCheckpoint, err := note.Sign(&note.Note{Text: string(checkpoint)}, noteSigner)
	if err != nil {
		return nil, fmt.Errorf("sign fixture Rekor v2 checkpoint: %w", err)
	}

	rekorLogID, err := logID(l.key.Public())
	if err != nil {
		return nil, fmt.Errorf("calculate fixture Rekor v2 log ID: %w", err)
	}
	return &protorekor.TransparencyLogEntry{
		LogIndex: 0,
		LogId:    &protocommon.LogId{KeyId: append([]byte(nil), rekorLogID...)},
		KindVersion: &protorekor.KindVersion{
			Kind:    "hashedrekord",
			Version: "0.0.2",
		},
		// Rekor v2 intentionally has neither IntegratedTime nor an
		// InclusionPromise. The RFC 3161 timestamp is the sole observer time.
		InclusionProof: &protorekor.InclusionProof{
			LogIndex: 0,
			TreeSize: 1,
			RootHash: append([]byte(nil), leafHash...),
			Hashes:   [][]byte{},
			Checkpoint: &protorekor.Checkpoint{
				Envelope: string(signedCheckpoint),
			},
		},
		CanonicalizedBody: canonicalBody,
	}, nil
}

func (l *localRekorV2) trustedLogs(at time.Time) (map[string]*root.TransparencyLog, error) {
	rekorLogID, err := logID(l.key.Public())
	if err != nil {
		return nil, fmt.Errorf("calculate fixture Rekor v2 log ID: %w", err)
	}
	return map[string]*root.TransparencyLog{
		hex.EncodeToString(rekorLogID): {
			BaseURL:             RekorV2URI,
			ID:                  append([]byte(nil), rekorLogID...),
			ValidityPeriodStart: at.Add(-time.Hour),
			ValidityPeriodEnd:   at.Add(time.Hour),
			HashFunc:            crypto.SHA256,
			PublicKey:           l.key.Public(),
			SignatureHashFunc:   crypto.SHA256,
		},
	}, nil
}

type localTransparencyLog struct {
	virtualSigstore *ca.VirtualSigstore
	integratedTime  time.Time
	artifact        []byte
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
	messageSignature := protobufBundle.GetMessageSignature()
	if envelope == nil && messageSignature == nil {
		return fmt.Errorf("fixture transparency log requires signed bundle content")
	}
	if envelope != nil && messageSignature != nil {
		return fmt.Errorf("fixture transparency log received ambiguous signed content")
	}
	if messageSignature != nil {
		return l.addMessageSignatureEntry(certificate, protobufBundle, messageSignature.GetSignature())
	}
	if len(envelope.GetSignatures()) != 1 {
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

func (l *localTransparencyLog) addMessageSignatureEntry(
	certificate *x509.Certificate,
	protobufBundle *protobundle.Bundle,
	signature []byte,
) error {
	certificatePEM, err := cryptoutils.MarshalCertificateToPEM(certificate)
	if err != nil {
		return fmt.Errorf("marshal fixture transparency certificate: %w", err)
	}
	artifactHash := sha256.Sum256(l.artifact)
	properties := rekortypes.ArtifactProperties{
		ArtifactHash:   hex.EncodeToString(artifactHash[:]),
		SignatureBytes: signature,
		PublicKeyBytes: [][]byte{certificatePEM},
		PKIFormat:      string(pki.X509),
	}
	proposed, err := rekortypes.NewProposedEntry(
		context.Background(),
		hashedrekord.KIND,
		hashedrekord.New().DefaultVersion(),
		properties,
	)
	if err != nil {
		return fmt.Errorf("create fixture hashedrekord proposal: %w", err)
	}
	entryImpl, err := rekortypes.CreateVersionedEntry(proposed)
	if err != nil {
		return fmt.Errorf("create fixture hashedrekord entry: %w", err)
	}
	canonicalProposal, err := rekortypes.CanonicalizeEntry(context.Background(), entryImpl)
	if err != nil {
		return fmt.Errorf("canonicalize fixture hashedrekord proposal: %w", err)
	}
	decodedProposal, err := models.UnmarshalProposedEntry(
		bytes.NewReader(canonicalProposal),
		openapiruntime.JSONConsumer(),
	)
	if err != nil {
		return fmt.Errorf("decode fixture hashedrekord proposal: %w", err)
	}
	entryImpl, err = rekortypes.UnmarshalEntry(decodedProposal)
	if err != nil {
		return fmt.Errorf("decode fixture hashedrekord entry: %w", err)
	}
	canonicalBody, err := entryImpl.Canonicalize(context.Background())
	if err != nil {
		return fmt.Errorf("canonicalize fixture hashedrekord entry: %w", err)
	}

	rekorLogID, err := l.virtualSigstore.RekorLogID()
	if err != nil {
		return fmt.Errorf("calculate fixture Rekor log ID: %w", err)
	}
	rawLogID, err := hex.DecodeString(rekorLogID)
	if err != nil {
		return fmt.Errorf("decode fixture Rekor log ID: %w", err)
	}
	// The virtual log returns a single-leaf inclusion proof, so the only valid
	// index is zero. The integrated time—not the synthetic index—is the fixture
	// value consumed by production policy.
	const logIndex int64 = 0
	payload := tlog.RekorPayload{
		Body:           base64.StdEncoding.EncodeToString(canonicalBody),
		IntegratedTime: l.integratedTime.Unix(),
		LogIndex:       logIndex,
		LogID:          rekorLogID,
	}
	signedEntryTimestamp, err := l.virtualSigstore.RekorSignPayload(payload)
	if err != nil {
		return fmt.Errorf("sign fixture Rekor promise: %w", err)
	}
	inclusionProof, err := l.virtualSigstore.GetInclusionProof(canonicalBody)
	if err != nil {
		return fmt.Errorf("create fixture Rekor inclusion proof: %w", err)
	}
	entry, err := tlog.NewEntry(
		canonicalBody,
		l.integratedTime.Unix(),
		logIndex,
		rawLogID,
		signedEntryTimestamp,
		inclusionProof,
	)
	if err != nil {
		return fmt.Errorf("assemble fixture Rekor entry: %w", err)
	}
	transparencyEntry := entry.TransparencyLogEntry()
	transparencyEntry.KindVersion = &protorekor.KindVersion{
		Kind:    hashedrekord.KIND,
		Version: hashedrekord.New().DefaultVersion(),
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
