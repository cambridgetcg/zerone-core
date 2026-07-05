package main

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

func strPtr(s string) *string { return &s }

// settledInvocation returns a fully-settled invocation fixture.
func settledInvocation() *Invocation {
	return &Invocation{
		ID:            "529ff750-fec4-4bfc-863e-8b101afbe1d8",
		ListingID:     "57809f83-c588-4174-ad7e-6ed60c134499",
		BuyerDID:      "did:at:09c5e59e-0374-4d80-a2c1-d8f1acbdfe9a",
		Amount:        53,
		Currency:      "GBP",
		Status:        "settled",
		CompletionSig: strPtr("c2lnbmF0dXJl"),
		CreatedAt:     "2026-07-05T10:09:51.018Z",
		CompletedAt:   strPtr("2026-07-05T11:00:00.000Z"),
		SettledAt:     strPtr("2026-07-05T11:00:00.100Z"),
	}
}

// ---------------------------------------------------------------------------
// TestCheckAttestable — table-driven refusal rules
// ---------------------------------------------------------------------------

func TestCheckAttestable(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*Invocation)
		wantErr bool
	}{
		{"settled with sig is attestable", func(i *Invocation) {}, false},
		{"escrowed refused", func(i *Invocation) { i.Status = "escrowed" }, true},
		{"refunded refused", func(i *Invocation) { i.Status = "refunded" }, true},
		{"declined refused", func(i *Invocation) { i.Status = "declined" }, true},
		{"completed but not settled refused", func(i *Invocation) { i.Status = "completed" }, true},
		{"settled without sig refused", func(i *Invocation) { i.CompletionSig = nil }, true},
		{"settled with empty sig refused", func(i *Invocation) { i.CompletionSig = strPtr("") }, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inv := settledInvocation()
			tc.mutate(inv)
			err := checkAttestable(inv)
			if (err != nil) != tc.wantErr {
				t.Fatalf("checkAttestable() error = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestContentHash — determinism and sensitivity
// ---------------------------------------------------------------------------

func TestContentHashDeterministic(t *testing.T) {
	h1, canon1, err := contentHash(settledInvocation())
	if err != nil {
		t.Fatal(err)
	}
	h2, canon2, err := contentHash(settledInvocation())
	if err != nil {
		t.Fatal(err)
	}
	if string(h1) != string(h2) {
		t.Fatal("same invocation produced different hashes")
	}
	if string(canon1) != string(canon2) {
		t.Fatal("same invocation produced different canonical bytes")
	}
	if len(h1) != 32 {
		t.Fatalf("expected sha256 (32 bytes), got %d", len(h1))
	}
}

func TestContentHashSensitive(t *testing.T) {
	base, _, _ := contentHash(settledInvocation())
	mutations := map[string]func(*Invocation){
		"amount":         func(i *Invocation) { i.Amount = 54 },
		"buyer":          func(i *Invocation) { i.BuyerDID = "did:at:other" },
		"completion_sig": func(i *Invocation) { i.CompletionSig = strPtr("b3RoZXI=") },
		"status":         func(i *Invocation) { i.Status = "refunded" },
	}
	for name, mutate := range mutations {
		inv := settledInvocation()
		mutate(inv)
		h, _, _ := contentHash(inv)
		if string(h) == string(base) {
			t.Errorf("mutating %s did not change the content hash", name)
		}
	}
}

// ---------------------------------------------------------------------------
// TestBuildLinkRoundTrip — the load-bearing contract with the CLI
// ---------------------------------------------------------------------------

// The submit-attestation command reads the link file with encoding/json into
// types.SubstrateLink (client/cli/tx.go readJSONFile). This pins that the
// relay's output survives that exact decode with every field intact.
func TestBuildLinkRoundTrip(t *testing.T) {
	cfg := loadConfig()
	inv := settledInvocation()
	linkBytes, err := buildLink(cfg, inv, 2222)
	if err != nil {
		t.Fatal(err)
	}

	var link types.SubstrateLink
	if err := json.Unmarshal(linkBytes, &link); err != nil {
		t.Fatalf("link JSON does not decode into types.SubstrateLink: %v", err)
	}
	if link.Source == nil {
		t.Fatal("decoded link has nil source")
	}
	if link.Source.SourceId != inv.ID {
		t.Errorf("source_id = %q, want %q", link.Source.SourceId, inv.ID)
	}
	if link.Source.FetchedAtBlock != 2222 {
		t.Errorf("fetched_at_block = %d, want 2222", link.Source.FetchedAtBlock)
	}
	wantHash, _, _ := contentHash(inv)
	if base64.StdEncoding.EncodeToString(wantHash) != base64.StdEncoding.EncodeToString(link.Source.ContentHash) {
		t.Error("decoded content_hash does not match recomputed canonical hash")
	}
	if len(link.CitedFacts) != 0 || len(link.PendingClaims) != 0 {
		t.Error("witness-only link must carry no cited facts or pending claims")
	}
}
