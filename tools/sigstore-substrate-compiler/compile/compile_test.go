package compile

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	substratebridgekeeper "github.com/zerone-chain/zerone/x/substrate_bridge/keeper"
	substratebridgetypes "github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

type fixtureMaterial struct {
	bundleJSON     []byte
	dssePayload    []byte
	sourceURL      string
	fetchedAtBlock uint64
}

func fixture() fixtureMaterial {
	return fixtureMaterial{
		bundleJSON:     []byte(`{"mediaType":"application/vnd.dev.sigstore.bundle.v0.3+json","dsseEnvelope":{"payload":"fixture"}}`),
		dssePayload:    []byte(`{"_type":"https://in-toto.io/Statement/v1","subject":[{"name":"artifact","digest":{"sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}],"predicateType":"https://slsa.dev/provenance/v1","predicate":{}}`),
		sourceURL:      "https://example.invalid/attestations/build-42.sigstore.json",
		fetchedAtBlock: 42,
	}
}

func compileFixture(input fixtureMaterial) (*substratebridgetypes.SubstrateLink, error) {
	return compileMaterial(input.bundleJSON, input.dssePayload, input.sourceURL, input.fetchedAtBlock)
}

func TestCompileWitnessOnlyDeterministic(t *testing.T) {
	input := fixture()
	first, err := compileFixture(input)
	if err != nil {
		t.Fatalf("compile first: %v", err)
	}
	second, err := compileFixture(input)
	if err != nil {
		t.Fatalf("compile second: %v", err)
	}

	if first.AdapterId != AdapterID {
		t.Fatalf("adapter_id: want %q, got %q", AdapterID, first.AdapterId)
	}
	if first.Source == nil {
		t.Fatal("source must be present")
	}
	if first.Source.AdapterId != AdapterID {
		t.Fatalf("source.adapter_id: want %q, got %q", AdapterID, first.Source.AdapterId)
	}
	if len(first.CitedFacts) != 0 {
		t.Fatalf("witness-only link must have zero citations, got %d", len(first.CitedFacts))
	}
	if len(first.PendingClaims) != 0 {
		t.Fatalf("witness-only link must have zero pending claims, got %d", len(first.PendingClaims))
	}
	if first.RecursionWeight != nil {
		t.Fatal("witness-only link must have nil recursion weight")
	}

	wantContentHash := sha256.Sum256(input.bundleJSON)
	if !bytes.Equal(first.Source.ContentHash, wantContentHash[:]) {
		t.Fatalf("content_hash is not sha256 of exact bundle JSON")
	}
	wantPayloadHash := sha256.Sum256(input.dssePayload)
	wantSourceID := "sha256:" + hex.EncodeToString(wantPayloadHash[:])
	if first.Source.SourceId != wantSourceID {
		t.Fatalf("source_id: want %q, got %q", wantSourceID, first.Source.SourceId)
	}
	if first.Source.FetchedAtBlock != input.fetchedAtBlock {
		t.Fatalf("fetched_at_block: want %d, got %d", input.fetchedAtBlock, first.Source.FetchedAtBlock)
	}
	if !bytes.Equal(first.LinkHash, second.LinkHash) {
		t.Fatal("identical inputs produced different link hashes")
	}
	if !bytes.Equal(first.LinkHash, substratebridgekeeper.ComputeLinkHash(first)) {
		t.Fatal("link_hash does not match substrate_bridge canonical hash")
	}
}

func TestCompilePayloadAndBundleMutationChangesAllHashes(t *testing.T) {
	original := fixture()
	mutated := fixture()
	mutated.dssePayload = append(append([]byte(nil), mutated.dssePayload...), '\n')
	mutated.bundleJSON = append(append([]byte(nil), mutated.bundleJSON...), '\n')

	first, err := compileFixture(original)
	if err != nil {
		t.Fatal(err)
	}
	second, err := compileFixture(mutated)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(first.Source.ContentHash, second.Source.ContentHash) {
		t.Fatal("bundle mutation did not change content_hash")
	}
	if first.Source.SourceId == second.Source.SourceId {
		t.Fatal("payload mutation did not change source_id")
	}
	if bytes.Equal(first.LinkHash, second.LinkHash) {
		t.Fatal("bundle mutation did not change link_hash")
	}
}

func TestBundleProofMutationChangesContentAndLinkHashesOnly(t *testing.T) {
	original := fixture()
	mutated := fixture()
	mutated.bundleJSON = append(append([]byte(nil), mutated.bundleJSON...), '\n')

	first, err := compileFixture(original)
	if err != nil {
		t.Fatal(err)
	}
	second, err := compileFixture(mutated)
	if err != nil {
		t.Fatal(err)
	}

	if first.Source.SourceId != second.Source.SourceId {
		t.Fatal("same payload bytes produced different source_id")
	}
	if bytes.Equal(first.Source.ContentHash, second.Source.ContentHash) {
		t.Fatal("bundle proof mutation did not change content_hash")
	}
	if bytes.Equal(first.LinkHash, second.LinkHash) {
		t.Fatal("bundle proof mutation did not change link_hash")
	}
}

func TestCompileRejectsUnsafeInput(t *testing.T) {
	tests := map[string]func(*fixtureMaterial){
		"empty bundle":       func(i *fixtureMaterial) { i.bundleJSON = nil },
		"empty payload":      func(i *fixtureMaterial) { i.dssePayload = nil },
		"empty source URL":   func(i *fixtureMaterial) { i.sourceURL = "" },
		"non HTTPS URL":      func(i *fixtureMaterial) { i.sourceURL = "http://example.invalid/a" },
		"relative URL":       func(i *fixtureMaterial) { i.sourceURL = "/attestations/a" },
		"URL credentials":    func(i *fixtureMaterial) { i.sourceURL = "https://user@example.invalid/a" },
		"URL query":          func(i *fixtureMaterial) { i.sourceURL = "https://example.invalid/a?token=secret" },
		"empty URL query":    func(i *fixtureMaterial) { i.sourceURL = "https://example.invalid/a?" },
		"URL fragment":       func(i *fixtureMaterial) { i.sourceURL = "https://example.invalid/a#fragment" },
		"surrounding spaces": func(i *fixtureMaterial) { i.sourceURL = " https://example.invalid/a" },
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			input := fixture()
			mutate(&input)
			if _, err := compileFixture(input); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestCompileRejectsMissingVerifiedAttestation(t *testing.T) {
	_, err := Compile(Input{SourceURL: fixture().sourceURL})
	if err == nil {
		t.Fatal("expected opaque verification boundary to reject nil attestation")
	}
}

func TestSourceURLIsAuditOnlyInCanonicalChainHash(t *testing.T) {
	firstInput := fixture()
	secondInput := fixture()
	secondInput.sourceURL = "https://mirror.example.invalid/attestations/build-42.sigstore.json"

	first, err := compileFixture(firstInput)
	if err != nil {
		t.Fatal(err)
	}
	second, err := compileFixture(secondInput)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(first.LinkHash, second.LinkHash) {
		t.Fatal("substrate_bridge canonical hash unexpectedly includes source_url")
	}
}
