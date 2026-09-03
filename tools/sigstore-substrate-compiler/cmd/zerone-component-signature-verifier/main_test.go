package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/zerone-chain/zerone/tools/sigstore-substrate-compiler/internal/testfixture"
	"github.com/zerone-chain/zerone/tools/sigstore-substrate-compiler/verification"
)

func TestRunEmitsStrictVerificationResult(t *testing.T) {
	material, err := testfixture.GenerateMessageSignature()
	if err != nil {
		t.Fatalf("generate component fixture: %v", err)
	}
	bundlePath, trustedRootPath, err := material.Write(t.TempDir())
	if err != nil {
		t.Fatalf("write component fixture: %v", err)
	}

	var stdout, stderr bytes.Buffer
	err = run([]string{
		"--bundle", bundlePath,
		"--trusted-root", trustedRootPath,
		"--certificate-issuer", testfixture.CertificateIssuer,
		"--certificate-san", testfixture.CertificateSAN,
		"--source-repository-digest", testfixture.SourceRepositoryDigest,
		"--artifact-digest", material.ArtifactDigest,
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run verifier: %v (stderr: %s)", err, stderr.String())
	}
	var result verification.ComponentVerification
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode verifier output: %v", err)
	}
	if result.Schema != verification.ComponentVerificationSchema || result.Result != "verified" {
		t.Fatalf("unexpected verifier result: %#v", result)
	}
	if len(result.VerifiedTimestamps) == 0 {
		t.Fatal("verifier omitted authenticated transparency-log time")
	}
}

func TestRunRejectsPositionalArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"bundle.json"}, &stdout, &stderr); err == nil {
		t.Fatal("expected positional argument rejection")
	}
}
