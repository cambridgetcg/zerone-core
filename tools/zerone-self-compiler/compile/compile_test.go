package compile_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/zerone-chain/zerone/tools/zerone-self-compiler/compile"
)

// fixture is a synthetic, deterministic commit used across tests so output
// is byte-stable regardless of git state.
func fixture() compile.CommitMeta {
	t, _ := time.Parse(time.RFC3339, "2026-05-11T17:52:35Z")
	return compile.CommitMeta{
		Hash:    "80cf9c0400327e016e41cc9df441371056c958ef",
		Author:  "YOU <alpha@ai-love.cc>",
		Date:    t,
		Subject: "spec(external-surface): nested design",
		TouchedFiles: []string{
			"docs/superpowers/specs/external-surface.md",
			"x/sponsorship/client/cli/tx.go",
			"proto/zerone/sponsorship/v1/tx.proto",
			"tests/cross_stack/sponsorship_test.go",
			"tools/sponsorship-demo.sh",
		},
	}
}

func TestCompile_HappyPath(t *testing.T) {
	link, err := compile.Compile(fixture(), 42)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if link.AdapterId != compile.AdapterID {
		t.Errorf("adapter_id: want %q, got %q", compile.AdapterID, link.AdapterId)
	}
	if len(link.PendingClaims) != 1 {
		t.Fatalf("pending_claims: want 1, got %d", len(link.PendingClaims))
	}
	want := "Commit 80cf9c040032 by YOU <alpha@ai-love.cc> at 2026-05-11T17:52:35Z: spec(external-surface): nested design"
	if link.PendingClaims[0].ClaimContent != want {
		t.Errorf("claim_content:\n  want: %s\n  got:  %s", want, link.PendingClaims[0].ClaimContent)
	}
	if link.PendingClaims[0].Domain != compile.SelfDomain {
		t.Errorf("domain: want %q, got %q", compile.SelfDomain, link.PendingClaims[0].Domain)
	}
	if link.Source.SourceId != "80cf9c0400327e016e41cc9df441371056c958ef" {
		t.Errorf("source_id: %q", link.Source.SourceId)
	}
	if link.Source.FetchedAtBlock != 42 {
		t.Errorf("fetched_at_block: want 42, got %d", link.Source.FetchedAtBlock)
	}
	if len(link.LinkHash) != 32 {
		t.Errorf("link_hash must be 32 bytes (sha256), got %d", len(link.LinkHash))
	}
}

func TestCompile_AxisHeuristics(t *testing.T) {
	cases := []struct {
		name             string
		files            []string
		wantSubstrate    uint64
		wantVerification uint64
		wantClassification uint64
		wantTooling      uint64
		wantInterface    uint64
	}{
		{
			name:             "proto_change",
			files:            []string{"proto/foo/v1/bar.proto"},
			wantSubstrate:    50_000,
			wantVerification: 30_000,
			wantClassification: 10_000,
			wantTooling:      100_000,
			wantInterface:    30_000,
		},
		{
			name:             "test_change",
			files:            []string{"tests/cross_stack/foo_test.go"},
			wantSubstrate:    10_000,
			wantVerification: 100_000,
			wantClassification: 10_000,
			wantTooling:      100_000,
			wantInterface:    30_000,
		},
		{
			name:             "spec_change",
			files:            []string{"docs/superpowers/specs/foo.md"},
			wantSubstrate:    10_000,
			wantVerification: 30_000,
			wantClassification: 50_000,
			wantTooling:      100_000,
			wantInterface:    30_000,
		},
		{
			name:             "tooling_change",
			files:            []string{"tools/foo/main.go", "scripts/bar.sh"},
			wantSubstrate:    10_000,
			wantVerification: 30_000,
			wantClassification: 10_000,
			wantTooling:      200_000,
			wantInterface:    30_000,
		},
		{
			name:             "cli_change",
			files:            []string{"x/sponsorship/client/cli/tx.go"},
			wantSubstrate:    10_000,
			wantVerification: 30_000,
			wantClassification: 10_000,
			wantTooling:      100_000,
			wantInterface:    100_000,
		},
	}
	t0, _ := time.Parse(time.RFC3339, "2026-05-11T17:52:35Z")
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			meta := compile.CommitMeta{
				Hash:         "80cf9c0400327e016e41cc9df441371056c958ef",
				Author:       "YOU <x@x>",
				Date:         t0,
				Subject:      "test",
				TouchedFiles: c.files,
			}
			link, err := compile.Compile(meta, 0)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			rw := link.RecursionWeight
			if rw.AxisSubstrate != c.wantSubstrate {
				t.Errorf("axis_substrate: want %d, got %d", c.wantSubstrate, rw.AxisSubstrate)
			}
			if rw.AxisVerification != c.wantVerification {
				t.Errorf("axis_verification: want %d, got %d", c.wantVerification, rw.AxisVerification)
			}
			if rw.AxisClassification != c.wantClassification {
				t.Errorf("axis_classification: want %d, got %d", c.wantClassification, rw.AxisClassification)
			}
			if rw.AxisTooling != c.wantTooling {
				t.Errorf("axis_tooling: want %d, got %d", c.wantTooling, rw.AxisTooling)
			}
			if rw.AxisInterface != c.wantInterface {
				t.Errorf("axis_interface: want %d, got %d", c.wantInterface, rw.AxisInterface)
			}
			if rw.AxisAttribution != 500_000 {
				t.Errorf("axis_attribution baseline: want 500000, got %d", rw.AxisAttribution)
			}
		})
	}
}

func TestCompile_Deterministic(t *testing.T) {
	// Two compiles of the same CommitMeta must produce identical LinkHash
	// AND identical Source.ContentHash. This is the property that lets
	// validators re-derive a submitter's link to check for compiler drift.
	a, err := compile.Compile(fixture(), 1)
	if err != nil {
		t.Fatal(err)
	}
	b, err := compile.Compile(fixture(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.LinkHash, b.LinkHash) {
		t.Errorf("link_hash differs across compiles of identical input")
	}
	if !bytes.Equal(a.Source.ContentHash, b.Source.ContentHash) {
		t.Errorf("source.content_hash differs across compiles of identical input")
	}
}

func TestCompile_TouchedFilesOrderInvariant(t *testing.T) {
	// Re-arranging TouchedFiles must not change the link's content_hash
	// (the compiler sorts before hashing). This is what keeps validators
	// honest — they re-derive from git's natural order, but the hash is
	// stable regardless.
	m1 := fixture()
	m2 := fixture()
	m2.TouchedFiles = []string{
		m1.TouchedFiles[3], m1.TouchedFiles[0], m1.TouchedFiles[4],
		m1.TouchedFiles[1], m1.TouchedFiles[2],
	}
	a, _ := compile.Compile(m1, 0)
	b, _ := compile.Compile(m2, 0)
	if !bytes.Equal(a.Source.ContentHash, b.Source.ContentHash) {
		t.Errorf("content_hash must be invariant to TouchedFiles order")
	}
}

func TestCompile_ValidationRefusesBadInput(t *testing.T) {
	bad := func(mutate func(*compile.CommitMeta)) compile.CommitMeta {
		m := fixture()
		mutate(&m)
		return m
	}
	cases := map[string]func(*compile.CommitMeta){
		"short_hash":     func(m *compile.CommitMeta) { m.Hash = "abcdef" },
		"uppercase_hash": func(m *compile.CommitMeta) { m.Hash = "ABC123" + m.Hash[6:] },
		"empty_author":   func(m *compile.CommitMeta) { m.Author = "" },
		"zero_date":      func(m *compile.CommitMeta) { m.Date = time.Time{} },
		"empty_subject":  func(m *compile.CommitMeta) { m.Subject = "" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := compile.Compile(bad(mutate), 0)
			if err == nil {
				t.Errorf("expected error for %s", name)
			}
		})
	}
}
