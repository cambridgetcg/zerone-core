package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/zerone-chain/zerone/tools/sigstore-substrate-compiler/internal/testfixture"
)

func TestRunRejectsPositionalArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"bundle.json"}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "positional arguments") {
		t.Fatalf("expected positional-argument error, got %v", err)
	}
}

func TestRunFailsClosedWhenRequiredPolicyIsMissing(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run(nil, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "bundle path must be non-empty") {
		t.Fatalf("expected missing-policy error, got %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("failed verification wrote %d bytes to stdout", stdout.Len())
	}
}

func TestRunHermeticEndToEndSuccess(t *testing.T) {
	material, err := testfixture.Generate()
	if err != nil {
		t.Fatalf("generate Sigstore fixture: %v", err)
	}
	bundlePath, trustedRootPath, err := material.Write(t.TempDir())
	if err != nil {
		t.Fatalf("write Sigstore fixture: %v", err)
	}

	const (
		sourceURL      = "https://attestations.fixture.invalid/build-42.sigstore.json"
		fetchedAtBlock = uint64(42)
	)
	args := []string{
		"--bundle", bundlePath,
		"--trusted-root", trustedRootPath,
		"--certificate-issuer", testfixture.CertificateIssuer,
		"--certificate-san", testfixture.CertificateSAN,
		"--artifact-digest", material.ArtifactDigest,
		"--predicate-type", testfixture.PredicateType,
		"--source-url", sourceURL,
		"--fetched-at-block", "42",
	}
	var stdout, stderr bytes.Buffer
	if err := run(args, &stdout, &stderr); err != nil {
		t.Fatalf("run compiler: %v (stderr: %s)", err, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("successful run wrote stderr: %s", stderr.String())
	}

	var output struct {
		AdapterID string `json:"adapter_id"`
		Source    struct {
			AdapterID      string `json:"adapter_id"`
			SourceID       string `json:"source_id"`
			SourceURL      string `json:"source_url"`
			ContentHash    []byte `json:"content_hash"`
			FetchedAtBlock uint64 `json:"fetched_at_block"`
		} `json:"source"`
		LinkHash []byte `json:"link_hash"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode compiler output: %v\n%s", err, stdout.String())
	}
	if output.AdapterID != "sigstore-in-toto-v1" || output.Source.AdapterID != output.AdapterID {
		t.Fatalf("unexpected adapter IDs: link=%q source=%q", output.AdapterID, output.Source.AdapterID)
	}
	if output.Source.SourceURL != sourceURL || output.Source.FetchedAtBlock != fetchedAtBlock {
		t.Fatalf("unexpected source metadata: %#v", output.Source)
	}
	payloadHash := sha256.Sum256(material.Payload)
	wantSourceID := "sha256:" + hex.EncodeToString(payloadHash[:])
	if output.Source.SourceID != wantSourceID {
		t.Fatalf("source ID: want %q, got %q", wantSourceID, output.Source.SourceID)
	}
	bundleHash := sha256.Sum256(material.BundleJSON)
	if !bytes.Equal(output.Source.ContentHash, bundleHash[:]) {
		t.Fatal("content hash does not commit to the exact verified bundle")
	}
	if len(output.LinkHash) != sha256.Size {
		t.Fatalf("link hash length: want %d, got %d", sha256.Size, len(output.LinkHash))
	}
}
