package verification

import (
	"bytes"
	"strings"
	"testing"
	"time"

	protobundle "github.com/sigstore/protobuf-specs/gen/pb-go/bundle/v1"
	protorekor "github.com/sigstore/protobuf-specs/gen/pb-go/rekor/v1"
	"github.com/zerone-chain/zerone/tools/sigstore-substrate-compiler/internal/testfixture"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func validComponentPolicy() ComponentPolicy {
	return ComponentPolicy{
		BundlePath:             "component.sigstore.json",
		TrustedRootPath:        "trusted-root.json",
		CertificateIssuer:      "https://token.actions.githubusercontent.com",
		CertificateSAN:         "https://github.com/example/project/.github/workflows/ci.yml@refs/heads/main",
		SourceRepositoryDigest: testfixture.SourceRepositoryDigest,
		ArtifactDigest:         "sha256:" + strings.Repeat("a", 64),
	}
}

func TestComponentPolicyValidate(t *testing.T) {
	policy := validComponentPolicy()
	digest, err := policy.Validate()
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if len(digest) != 32 {
		t.Fatalf("decoded digest length: want 32, got %d", len(digest))
	}
}

func TestComponentPolicyRejectsIncompleteOrAmbiguousInputs(t *testing.T) {
	tests := map[string]func(*ComponentPolicy){
		"missing bundle":        func(p *ComponentPolicy) { p.BundlePath = "" },
		"missing trusted root":  func(p *ComponentPolicy) { p.TrustedRootPath = "" },
		"missing issuer":        func(p *ComponentPolicy) { p.CertificateIssuer = "" },
		"missing SAN":           func(p *ComponentPolicy) { p.CertificateSAN = "" },
		"missing source digest": func(p *ComponentPolicy) { p.SourceRepositoryDigest = "" },
		"bad source digest":     func(p *ComponentPolicy) { p.SourceRepositoryDigest = strings.Repeat("g", 40) },
		"missing digest":        func(p *ComponentPolicy) { p.ArtifactDigest = "" },
		"wrong algorithm":       func(p *ComponentPolicy) { p.ArtifactDigest = "sha512:" + strings.Repeat("a", 64) },
		"short digest":          func(p *ComponentPolicy) { p.ArtifactDigest = "sha256:" + strings.Repeat("a", 62) },
		"uppercase digest":      func(p *ComponentPolicy) { p.ArtifactDigest = "sha256:" + strings.Repeat("A", 64) },
		"non-hex digest":        func(p *ComponentPolicy) { p.ArtifactDigest = "sha256:" + strings.Repeat("g", 64) },
		"issuer whitespace":     func(p *ComponentPolicy) { p.CertificateIssuer = " issuer" },
		"SAN whitespace":        func(p *ComponentPolicy) { p.CertificateSAN += " " },
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			policy := validComponentPolicy()
			mutate(&policy)
			if _, err := policy.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestVerifyComponentHermeticEndToEnd(t *testing.T) {
	material := generateComponentMaterial(t, false)
	policy := writeComponentPolicy(t, material)

	result, err := VerifyComponent(policy)
	if err != nil {
		t.Fatalf("verify hermetic component bundle: %v", err)
	}
	if result.Schema != ComponentVerificationSchema || result.Result != "verified" {
		t.Fatalf("unexpected result envelope: %#v", result)
	}
	if result.ArtifactDigest != material.ArtifactDigest {
		t.Fatalf("artifact digest: want %q, got %q", material.ArtifactDigest, result.ArtifactDigest)
	}
	if result.BundleMediaType != "application/vnd.dev.sigstore.bundle.v0.3+json" {
		t.Fatalf("bundle media type: %q", result.BundleMediaType)
	}
	if result.CertificateIssuer != testfixture.CertificateIssuer || result.CertificateSAN != testfixture.CertificateSAN {
		t.Fatalf("unexpected verified identity: %#v", result)
	}
	if result.SourceRepositoryDigest != testfixture.SourceRepositoryDigest {
		t.Fatalf("source repository digest: %q", result.SourceRepositoryDigest)
	}
	if len(result.VerifiedTimestamps) != 1 {
		t.Fatalf("verified timestamps: want 1, got %d", len(result.VerifiedTimestamps))
	}
	if got, want := result.VerifiedTimestamps[0].Timestamp, material.IntegratedTime.Format("2006-01-02T15:04:05Z07:00"); got != want {
		t.Fatalf("verified timestamp: want %q, got %q", want, got)
	}
	if result.VerifiedTimestamps[0].Type != "Tlog" {
		t.Fatalf("verified timestamp type: %q", result.VerifiedTimestamps[0].Type)
	}
}

func TestVerifyComponentHermeticRFC3161RekorV2EndToEnd(t *testing.T) {
	material, err := testfixture.GenerateMessageSignatureWithRFC3161RekorV2()
	if err != nil {
		t.Fatalf("generate RFC 3161/Rekor v2 fixture: %v", err)
	}
	if material.ObserverTime.IsZero() {
		t.Fatal("fixture did not record its RFC 3161 observer time")
	}

	protobufBundle := &protobundle.Bundle{}
	if err := protojson.Unmarshal(material.BundleJSON, protobufBundle); err != nil {
		t.Fatalf("parse RFC 3161/Rekor v2 fixture: %v", err)
	}
	verificationMaterial := protobufBundle.GetVerificationMaterial()
	if verificationMaterial == nil || len(verificationMaterial.GetTlogEntries()) != 1 {
		t.Fatalf("expected exactly one Rekor v2 entry, got %#v", verificationMaterial)
	}
	entry := verificationMaterial.GetTlogEntries()[0]
	if entry.GetKindVersion().GetKind() != "hashedrekord" || entry.GetKindVersion().GetVersion() != "0.0.2" {
		t.Fatalf("expected Rekor v2 hashedrekord entry, got %#v", entry.GetKindVersion())
	}
	if entry.GetInclusionProof() == nil {
		t.Fatal("Rekor v2 fixture has no inclusion proof")
	}
	if entry.GetIntegratedTime() != 0 || entry.GetInclusionPromise() != nil {
		t.Fatalf("Rekor v2 fixture unexpectedly carries v1 observer fields: %#v", entry)
	}
	timestamps := verificationMaterial.GetTimestampVerificationData().GetRfc3161Timestamps()
	if len(timestamps) != 1 || len(timestamps[0].GetSignedTimestamp()) == 0 {
		t.Fatalf("expected exactly one RFC 3161 timestamp, got %#v", timestamps)
	}

	policy := writeComponentPolicy(t, material)
	result, err := VerifyComponent(policy)
	if err != nil {
		t.Fatalf("verify RFC 3161/Rekor v2 component bundle: %v", err)
	}
	if len(result.VerifiedTimestamps) != 1 {
		t.Fatalf("verified timestamps: want 1, got %d", len(result.VerifiedTimestamps))
	}
	verifiedTimestamp := result.VerifiedTimestamps[0]
	if verifiedTimestamp.Type != "TimestampAuthority" {
		t.Fatalf("verified timestamp type: want TimestampAuthority, got %q", verifiedTimestamp.Type)
	}
	if verifiedTimestamp.URI != testfixture.TimestampAuthorityURI {
		t.Fatalf("verified timestamp URI: want %q, got %q", testfixture.TimestampAuthorityURI, verifiedTimestamp.URI)
	}
	if got, want := verifiedTimestamp.Timestamp, material.ObserverTime.UTC().Format(time.RFC3339Nano); got != want {
		t.Fatalf("verified timestamp: want %q, got %q", want, got)
	}

	t.Run("requires RFC 3161 observer", func(t *testing.T) {
		missingTimestampPolicy := policy
		missingTimestampPolicy.BundlePath = writeMutatedBundle(t, material.BundleJSON, func(bundle *protobundle.Bundle) {
			bundle.VerificationMaterial.TimestampVerificationData = nil
		})
		if _, err := VerifyComponent(missingTimestampPolicy); err == nil || !strings.Contains(err.Error(), "threshold not met for verified signed & log entry integrated timestamps") {
			t.Fatalf("expected missing RFC 3161 observer to fail, got %v", err)
		}
	})

	t.Run("authenticates exact component signature", func(t *testing.T) {
		tamperedTimestampPolicy := policy
		tamperedTimestampPolicy.BundlePath = writeMutatedBundle(t, material.BundleJSON, func(bundle *protobundle.Bundle) {
			timestamp := bundle.VerificationMaterial.TimestampVerificationData.Rfc3161Timestamps[0]
			timestamp.SignedTimestamp[len(timestamp.SignedTimestamp)-1] ^= 0xff
		})
		if _, err := VerifyComponent(tamperedTimestampPolicy); err == nil || !strings.Contains(err.Error(), "threshold not met for verified signed & log entry integrated timestamps") {
			t.Fatalf("expected signature-mismatched RFC 3161 timestamp to fail, got %v", err)
		}
	})

	t.Run("authenticates Rekor v2 inclusion proof", func(t *testing.T) {
		tamperedProofPolicy := policy
		tamperedProofPolicy.BundlePath = writeMutatedBundle(t, material.BundleJSON, func(bundle *protobundle.Bundle) {
			checkpoint := []byte(bundle.VerificationMaterial.TlogEntries[0].InclusionProof.Checkpoint.Envelope)
			checkpoint[len(checkpoint)/2] ^= 0x01
			bundle.VerificationMaterial.TlogEntries[0].InclusionProof.Checkpoint.Envelope = string(checkpoint)
		})
		if _, err := VerifyComponent(tamperedProofPolicy); err == nil || !strings.Contains(err.Error(), "verifying log entry") {
			t.Fatalf("expected tampered Rekor v2 inclusion proof to fail, got %v", err)
		}
	})
}

func TestVerifyComponentRequiresExactPolicyMatches(t *testing.T) {
	material := generateComponentMaterial(t, false)
	basePolicy := writeComponentPolicy(t, material)

	tests := map[string]func(*ComponentPolicy){
		"certificate SAN": func(policy *ComponentPolicy) {
			policy.CertificateSAN = "other-signer@example.invalid"
		},
		"certificate issuer": func(policy *ComponentPolicy) {
			policy.CertificateIssuer = "https://other-issuer.fixture.invalid"
		},
		"source repository digest": func(policy *ComponentPolicy) {
			policy.SourceRepositoryDigest = strings.Repeat("2", 40)
		},
		"artifact digest": func(policy *ComponentPolicy) {
			policy.ArtifactDigest = "sha256:" + strings.Repeat("0", 64)
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			policy := basePolicy
			mutate(&policy)
			if _, err := VerifyComponent(policy); err == nil {
				t.Fatalf("expected exact %s mismatch to fail", name)
			}
		})
	}
}

func TestVerifyComponentRejectsDSSEBundle(t *testing.T) {
	material, err := testfixture.Generate()
	if err != nil {
		t.Fatalf("generate DSSE fixture: %v", err)
	}
	policy := writeComponentPolicy(t, material)
	_, err = VerifyComponent(policy)
	if err == nil || !strings.Contains(err.Error(), "messageSignature") {
		t.Fatalf("expected DSSE shape rejection, got %v", err)
	}
}

func TestVerifyComponentRequiresExactBundleMediaType(t *testing.T) {
	material := generateComponentMaterial(t, false)
	policy := writeComponentPolicy(t, material)
	policy.BundlePath = writeMutatedBundle(t, material.BundleJSON, func(bundle *protobundle.Bundle) {
		bundle.MediaType = "application/vnd.dev.sigstore.bundle.v0.3.1+json"
	})

	_, err := VerifyComponent(policy)
	if err == nil || !strings.Contains(err.Error(), ComponentBundleMediaType) {
		t.Fatalf("expected exact media-type rejection, got %v", err)
	}
}

func TestVerifyComponentRequiresSigstoreEvidence(t *testing.T) {
	material := generateComponentMaterial(t, false)
	basePolicy := writeComponentPolicy(t, material)

	t.Run("signed certificate timestamp", func(t *testing.T) {
		withoutSCT := generateComponentMaterial(t, true)
		policy := writeComponentPolicy(t, withoutSCT)
		_, err := VerifyComponent(policy)
		if err == nil || !strings.Contains(err.Error(), "SCT entries") {
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
			wantErr: "must contain a Rekor inclusion proof",
		},
		"inclusion proof": {
			mutate: func(bundle *protobundle.Bundle) {
				bundle.VerificationMaterial.TlogEntries[0].InclusionProof = nil
			},
			wantErr: "inclusion proof missing",
		},
		"observer timestamp": {
			mutate: func(bundle *protobundle.Bundle) {
				bundle.VerificationMaterial.TlogEntries[0].InclusionPromise = nil
			},
			wantErr: "threshold not met for verified signed & log entry integrated timestamps",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			policy := basePolicy
			policy.BundlePath = writeMutatedBundle(t, material.BundleJSON, test.mutate)
			_, err := VerifyComponent(policy)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("expected %s failure containing %q, got %v", name, test.wantErr, err)
			}
		})
	}
}

func TestVerifyComponentRejectsSplitProofAcrossTrustedAndUnknownLogs(t *testing.T) {
	material := generateComponentMaterial(t, false)
	policy := writeComponentPolicy(t, material)
	policy.BundlePath = writeMutatedBundle(t, material.BundleJSON, func(bundle *protobundle.Bundle) {
		trusted := bundle.VerificationMaterial.TlogEntries[0]
		unknown := proto.Clone(trusted).(*protorekor.TransparencyLogEntry)
		unknown.LogId.KeyId = bytes.Repeat([]byte{0x42}, 32)
		trusted.InclusionProof = nil
		bundle.VerificationMaterial.TlogEntries = append(
			bundle.VerificationMaterial.TlogEntries,
			unknown,
		)
	})

	_, err := VerifyComponent(policy)
	if err == nil || !strings.Contains(err.Error(), "entry 0 has no inclusion proof") {
		t.Fatalf("expected per-entry inclusion-proof rejection, got %v", err)
	}
}

func generateComponentMaterial(t *testing.T, withoutSCT bool) *testfixture.Material {
	t.Helper()
	var (
		material *testfixture.Material
		err      error
	)
	if withoutSCT {
		material, err = testfixture.GenerateMessageSignatureWithoutSCT()
	} else {
		material, err = testfixture.GenerateMessageSignature()
	}
	if err != nil {
		t.Fatalf("generate component Sigstore fixture: %v", err)
	}
	return material
}

func writeComponentPolicy(t *testing.T, material *testfixture.Material) ComponentPolicy {
	t.Helper()
	bundlePath, trustedRootPath, err := material.Write(t.TempDir())
	if err != nil {
		t.Fatalf("write component fixture: %v", err)
	}
	return ComponentPolicy{
		BundlePath:             bundlePath,
		TrustedRootPath:        trustedRootPath,
		CertificateIssuer:      testfixture.CertificateIssuer,
		CertificateSAN:         testfixture.CertificateSAN,
		SourceRepositoryDigest: testfixture.SourceRepositoryDigest,
		ArtifactDigest:         material.ArtifactDigest,
	}
}
