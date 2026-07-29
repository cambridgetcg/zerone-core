package verification

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	intoto "github.com/in-toto/attestation/go/v1"
	protobundle "github.com/sigstore/protobuf-specs/gen/pb-go/bundle/v1"
	"github.com/zerone-chain/zerone/tools/sigstore-substrate-compiler/internal/testfixture"
	"golang.org/x/sys/unix"
	"google.golang.org/protobuf/encoding/protojson"
)

func validPolicy() Policy {
	return Policy{
		BundlePath:        "provenance.sigstore.json",
		TrustedRootPath:   "trusted-root.json",
		CertificateIssuer: "https://token.actions.githubusercontent.com",
		CertificateSAN:    "https://github.com/example/project/.github/workflows/release.yml@refs/heads/main",
		ArtifactDigest:    "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		PredicateType:     "https://slsa.dev/provenance/v1",
	}
}

func TestPolicyValidate(t *testing.T) {
	policy := validPolicy()
	digest, err := policy.Validate()
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if len(digest) != 32 {
		t.Fatalf("decoded digest length: want 32, got %d", len(digest))
	}
}

func TestPolicyValidateRejectsIncompleteOrAmbiguousInputs(t *testing.T) {
	tests := map[string]func(*Policy){
		"missing bundle":       func(p *Policy) { p.BundlePath = "" },
		"missing trusted root": func(p *Policy) { p.TrustedRootPath = "" },
		"missing issuer":       func(p *Policy) { p.CertificateIssuer = "" },
		"missing SAN":          func(p *Policy) { p.CertificateSAN = "" },
		"missing digest":       func(p *Policy) { p.ArtifactDigest = "" },
		"wrong algorithm":      func(p *Policy) { p.ArtifactDigest = "sha512:" + strings.Repeat("a", 64) },
		"short digest":         func(p *Policy) { p.ArtifactDigest = "sha256:" + strings.Repeat("a", 62) },
		"uppercase digest":     func(p *Policy) { p.ArtifactDigest = "sha256:" + strings.Repeat("A", 64) },
		"non-hex digest":       func(p *Policy) { p.ArtifactDigest = "sha256:" + strings.Repeat("g", 64) },
		"relative predicate":   func(p *Policy) { p.PredicateType = "slsa/provenance/v1" },
		"predicate credentials": func(p *Policy) {
			p.PredicateType = "https://user@example.invalid/predicate/v1"
		},
		"issuer whitespace": func(p *Policy) { p.CertificateIssuer = " issuer" },
		"SAN whitespace":    func(p *Policy) { p.CertificateSAN += " " },
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			policy := validPolicy()
			mutate(&policy)
			if _, err := policy.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestValidateStatementRequiresExactV1AndPredicate(t *testing.T) {
	const predicate = "https://slsa.dev/provenance/v1"
	valid := &intoto.Statement{
		Type:          StatementTypeV1,
		PredicateType: predicate,
		Subject: []*intoto.ResourceDescriptor{
			{Digest: map[string]string{"sha256": strings.Repeat("a", 64)}},
		},
	}
	if err := validateStatement(valid, predicate); err != nil {
		t.Fatalf("valid statement: %v", err)
	}
	oldVersion := &intoto.Statement{
		Type:          "https://in-toto.io/Statement/v0.1",
		PredicateType: valid.PredicateType,
		Subject:       valid.Subject,
	}
	if err := validateStatement(oldVersion, predicate); err == nil {
		t.Fatal("expected old statement version rejection")
	}
	wrongPredicate := &intoto.Statement{
		Type:          valid.Type,
		PredicateType: "https://slsa.dev/provenance/v0.2",
		Subject:       valid.Subject,
	}
	if err := validateStatement(wrongPredicate, predicate); err == nil {
		t.Fatal("expected predicate mismatch rejection")
	}
}

func TestValidateStatementRequiresDigestOnEverySubject(t *testing.T) {
	const predicate = "https://slsa.dev/provenance/v1"
	tests := map[string][]*intoto.ResourceDescriptor{
		"no subjects":        nil,
		"nil subject":        {nil},
		"empty digest":       {{Digest: map[string]string{}}},
		"empty algorithm":    {{Digest: map[string]string{"": strings.Repeat("a", 64)}}},
		"empty digest value": {{Digest: map[string]string{"sha256": ""}}},
		"one invalid of two": {
			{Digest: map[string]string{"sha256": strings.Repeat("a", 64)}},
			{},
		},
	}
	for name, subjects := range tests {
		t.Run(name, func(t *testing.T) {
			statement := &intoto.Statement{
				Type:          StatementTypeV1,
				PredicateType: predicate,
				Subject:       subjects,
			}
			if err := validateStatement(statement, predicate); err == nil {
				t.Fatal("expected subject-shape validation error")
			}
		})
	}
}

func TestVerifiedAttestationCompilationMaterialIsOpaqueAndCopied(t *testing.T) {
	verified := &VerifiedAttestation{
		bundleJSON:  []byte("bundle"),
		dssePayload: []byte("payload"),
	}
	bundleJSON, dssePayload, err := verified.CompilationMaterial()
	if err != nil {
		t.Fatalf("compilation material: %v", err)
	}
	bundleJSON[0] = 'B'
	dssePayload[0] = 'P'

	secondBundle, secondPayload, err := verified.CompilationMaterial()
	if err != nil {
		t.Fatalf("second compilation material: %v", err)
	}
	if string(secondBundle) != "bundle" || string(secondPayload) != "payload" {
		t.Fatal("caller mutated the verified attestation through returned bytes")
	}
}

func TestVerifiedAttestationCompilationMaterialRejectsEmpty(t *testing.T) {
	var nilAttestation *VerifiedAttestation
	if _, _, err := nilAttestation.CompilationMaterial(); err == nil {
		t.Fatal("expected nil attestation rejection")
	}
	if _, _, err := (&VerifiedAttestation{}).CompilationMaterial(); err == nil {
		t.Fatal("expected empty attestation rejection")
	}
}

func TestReadRegularFileRejectsFIFONonBlocking(t *testing.T) {
	fifoPath := filepath.Join(t.TempDir(), "bundle.fifo")
	if err := unix.Mkfifo(fifoPath, 0o600); err != nil {
		t.Fatalf("create FIFO: %v", err)
	}

	result := make(chan error, 1)
	go func() {
		_, err := readRegularFile(fifoPath, 1024)
		result <- err
	}()

	select {
	case err := <-result:
		if err == nil {
			t.Fatal("expected FIFO rejection")
		}
		if !strings.Contains(err.Error(), "not a regular file") {
			t.Fatalf("expected regular-file error, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("readRegularFile blocked while opening a FIFO")
	}
}

func TestVerifyBundleHermeticEndToEnd(t *testing.T) {
	material := generateMaterial(t, false)
	policy := writeMaterialPolicy(t, material)

	verified, err := VerifyBundle(policy)
	if err != nil {
		t.Fatalf("verify hermetic bundle: %v", err)
	}
	bundleJSON, payload, err := verified.CompilationMaterial()
	if err != nil {
		t.Fatalf("read verified material: %v", err)
	}
	if !bytes.Equal(bundleJSON, material.BundleJSON) {
		t.Fatal("verified proof bytes differ from the local bundle")
	}
	if !bytes.Equal(payload, material.Payload) {
		t.Fatal("verified payload bytes differ from the signed statement")
	}
}

func TestVerifyBundleRequiresExactPolicyMatches(t *testing.T) {
	material := generateMaterial(t, false)
	basePolicy := writeMaterialPolicy(t, material)

	tests := map[string]func(*Policy){
		"certificate SAN": func(policy *Policy) {
			policy.CertificateSAN = "other-signer@example.invalid"
		},
		"certificate issuer": func(policy *Policy) {
			policy.CertificateIssuer = "https://other-issuer.fixture.invalid"
		},
		"artifact digest": func(policy *Policy) {
			policy.ArtifactDigest = "sha256:" + strings.Repeat("0", 64)
		},
		"predicate type": func(policy *Policy) {
			policy.PredicateType = "https://predicate.fixture.invalid/other"
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			policy := basePolicy
			mutate(&policy)
			if _, err := VerifyBundle(policy); err == nil {
				t.Fatalf("expected exact %s mismatch to fail", name)
			}
		})
	}
}

func TestVerifyBundleRequiresSigstoreEvidence(t *testing.T) {
	material := generateMaterial(t, false)
	basePolicy := writeMaterialPolicy(t, material)

	t.Run("signed certificate timestamp", func(t *testing.T) {
		withoutSCT := generateMaterial(t, true)
		policy := writeMaterialPolicy(t, withoutSCT)
		_, err := VerifyBundle(policy)
		if err == nil {
			t.Fatal("expected missing SCT evidence to fail")
		}
		if !strings.Contains(err.Error(), "SCT entries") {
			t.Fatalf("expected SCT threshold failure, got %v", err)
		}
	})

	tests := map[string]struct {
		mutate  func(*protobundle.Bundle)
		wantErr string
	}{
		"transparency log": {
			mutate: func(bundle *protobundle.Bundle) {
				bundle.VerificationMaterial.TlogEntries = nil
			},
			wantErr: "not enough verified log entries",
		},
		"observer timestamp": {
			mutate: func(bundle *protobundle.Bundle) {
				// Keep the independently verifiable inclusion proof, but remove
				// the signed Rekor promise that authenticates integratedTime.
				bundle.VerificationMaterial.TlogEntries[0].InclusionPromise = nil
			},
			wantErr: "threshold not met for verified signed & log entry integrated timestamps",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			policy := basePolicy
			policy.BundlePath = writeMutatedBundle(t, material.BundleJSON, test.mutate)
			_, err := VerifyBundle(policy)
			if err == nil {
				t.Fatalf("expected missing %s evidence to fail", name)
			}
			if !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("expected %s failure, got %v", name, err)
			}
		})
	}
}

func generateMaterial(t *testing.T, withoutSCT bool) *testfixture.Material {
	t.Helper()
	var (
		material *testfixture.Material
		err      error
	)
	if withoutSCT {
		material, err = testfixture.GenerateWithoutSCT()
	} else {
		material, err = testfixture.Generate()
	}
	if err != nil {
		t.Fatalf("generate Sigstore fixture: %v", err)
	}
	return material
}

func writeMaterialPolicy(t *testing.T, material *testfixture.Material) Policy {
	t.Helper()
	bundlePath, trustedRootPath, err := material.Write(t.TempDir())
	if err != nil {
		t.Fatalf("write Sigstore fixture: %v", err)
	}
	return Policy{
		BundlePath:        bundlePath,
		TrustedRootPath:   trustedRootPath,
		CertificateIssuer: testfixture.CertificateIssuer,
		CertificateSAN:    testfixture.CertificateSAN,
		ArtifactDigest:    material.ArtifactDigest,
		PredicateType:     testfixture.PredicateType,
	}
}

func writeMutatedBundle(t *testing.T, bundleJSON []byte, mutate func(*protobundle.Bundle)) string {
	t.Helper()
	protobufBundle := &protobundle.Bundle{}
	if err := protojson.Unmarshal(bundleJSON, protobufBundle); err != nil {
		t.Fatalf("parse generated bundle: %v", err)
	}
	mutate(protobufBundle)
	mutatedJSON, err := protojson.Marshal(protobufBundle)
	if err != nil {
		t.Fatalf("marshal mutated bundle: %v", err)
	}
	path := filepath.Join(t.TempDir(), "mutated.sigstore.json")
	if err := os.WriteFile(path, mutatedJSON, 0o600); err != nil {
		t.Fatalf("write mutated bundle: %v", err)
	}
	return path
}
