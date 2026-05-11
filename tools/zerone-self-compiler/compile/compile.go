// Package compile turns ZERONE git-commit metadata into a deterministic
// SubstrateLink for the zerone-self-v1 adapter. See
// docs/specs/adapters/zerone-self-v1.md for the spec this implements.
//
// The compiler is split into a pure-function library (this file) and a
// thin CLI wrapper (../main.go). Cross-stack tests exercise the library
// with synthetic CommitMeta values; the CLI binary uses git to populate
// CommitMeta from real history. Determinism is the library's job —
// identical CommitMeta in, identical SubstrateLink (and link_hash) out.
package compile

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"time"

	substratebridgekeeper "github.com/zerone-chain/zerone/x/substrate_bridge/keeper"
	substratebridgetypes "github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

const (
	// AdapterID is the canonical adapter identifier — must match the
	// AdapterRegistration registered on chain via gov LIP.
	AdapterID = "zerone-self-v1"

	// SourceURLBase is the canonical URL prefix for ZERONE's git history.
	// Per-fork forks register their own adapter against their own URL.
	SourceURLBase = "https://codeberg.org/zerone-dev/zerone/commit/"

	// SelfDomain is the knowledge domain dedicated to facts about ZERONE
	// itself. Must exist as a registered Domain in x/knowledge for
	// pending_claims to land.
	SelfDomain = "zerone_self"

	// MethodologyID names the verification methodology applied to claims
	// produced by this adapter. Verification panels apply this methodology
	// (commit-content vs. claim-content cross-check) rather than
	// re-running code review.
	MethodologyID = "git-commit-attestation-v1"

	// ShortHashLen is the prefix length used in the canonical claim text.
	ShortHashLen = 12
)

// CommitMeta is the deterministic input to the compiler. The CLI populates
// this from `git show --no-patch --format=...`; tests construct it directly.
type CommitMeta struct {
	Hash         string    // full 40-char SHA, lowercase
	Author       string    // canonical "Name <email>" form from git
	Date         time.Time // committer date, UTC
	Subject      string    // first line of commit message
	TouchedFiles []string  // paths relative to repo root; order-independent
}

// Validate enforces preconditions the compiler needs to produce a
// well-formed link.
func (m *CommitMeta) Validate() error {
	if len(m.Hash) != 40 {
		return fmt.Errorf("commit hash must be 40 chars, got %d (%q)", len(m.Hash), m.Hash)
	}
	if m.Hash != strings.ToLower(m.Hash) {
		return fmt.Errorf("commit hash must be lowercase: %q", m.Hash)
	}
	if m.Author == "" {
		return fmt.Errorf("author must be non-empty")
	}
	if m.Date.IsZero() {
		return fmt.Errorf("date must be set")
	}
	if m.Subject == "" {
		return fmt.Errorf("subject must be non-empty")
	}
	return nil
}

// CanonicalClaimContent builds the deterministic one-line claim text per §4
// of the adapter spec.
func CanonicalClaimContent(m *CommitMeta) string {
	return fmt.Sprintf("Commit %s by %s at %s: %s",
		m.Hash[:ShortHashLen],
		m.Author,
		m.Date.UTC().Format(time.RFC3339),
		m.Subject,
	)
}

// canonicalCommitContentHash is the per-source content-hash recorded on the
// ExternalSource. It's a sha256 over the deterministic CommitMeta projection
// used by the claim and by the axis heuristics.
func canonicalCommitContentHash(m *CommitMeta) []byte {
	files := append([]string(nil), m.TouchedFiles...)
	sort.Strings(files)
	pieces := []string{
		m.Hash,
		m.Author,
		m.Date.UTC().Format(time.RFC3339),
		m.Subject,
		strings.Join(files, "\n"),
	}
	h := sha256.Sum256([]byte(strings.Join(pieces, "\x00")))
	return h[:]
}

// ComputeAxisProjection applies the heuristics from §5 of the adapter spec.
// Heuristics are deterministic functions of TouchedFiles only; same files
// in, same projection out.
func ComputeAxisProjection(m *CommitMeta) *substratebridgetypes.AxisProjection {
	touches := func(prefixes ...string) bool {
		for _, f := range m.TouchedFiles {
			for _, p := range prefixes {
				if strings.HasPrefix(f, p) {
					return true
				}
			}
		}
		return false
	}
	touchesAny := func(predicates ...func(string) bool) bool {
		for _, f := range m.TouchedFiles {
			for _, p := range predicates {
				if p(f) {
					return true
				}
			}
		}
		return false
	}
	isTestFile := func(f string) bool {
		return strings.HasPrefix(f, "tests/") || strings.HasSuffix(f, "_test.go")
	}

	proj := &substratebridgetypes.AxisProjection{
		AxisAttribution: 500_000, // baseline: every commit is attribution
	}
	if touches("proto/") {
		proj.AxisSubstrate = 50_000
	} else {
		proj.AxisSubstrate = 10_000
	}
	if touchesAny(isTestFile) {
		proj.AxisVerification = 100_000
	} else {
		proj.AxisVerification = 30_000
	}
	if touches("docs/superpowers/specs/", "docs/USEFUL_WORK.md") {
		proj.AxisClassification = 50_000
	} else {
		proj.AxisClassification = 10_000
	}
	if touches("tools/", "scripts/") {
		proj.AxisTooling = 200_000
	} else {
		proj.AxisTooling = 100_000
	}
	if touchesContainsCLI(m.TouchedFiles) {
		proj.AxisInterface = 100_000
	} else {
		proj.AxisInterface = 30_000
	}
	return proj
}

func touchesContainsCLI(files []string) bool {
	for _, f := range files {
		// Match x/<module>/client/cli/ regardless of module name.
		if strings.HasPrefix(f, "x/") && strings.Contains(f, "/client/cli/") {
			return true
		}
	}
	return false
}

// Compile turns CommitMeta into a SubstrateLink with the link_hash filled
// in. The FetchedAtBlock field is set by the CALLER (CLI passes 0 by
// default for offline derivation; on-chain submission re-stamps it). The
// returned link's LinkHash is the canonical hash computed by
// substrate_bridge's keeper helper — the same function the chain itself
// will use to verify the submitted link.
//
// Determinism property:
//
//	compile(m) == compile(m')  iff  CommitMeta(m) == CommitMeta(m')
//	(equal hashes, equal LinkHash bytes)
func Compile(m CommitMeta, fetchedAtBlock uint64) (*substratebridgetypes.SubstrateLink, error) {
	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("invalid commit meta: %w", err)
	}

	link := &substratebridgetypes.SubstrateLink{
		AdapterId:  AdapterID,
		CitedFacts: nil, // commits don't cite facts directly; lineage handles parent commits
		PendingClaims: []*substratebridgetypes.PendingClaim{
			{
				ClaimContent:  CanonicalClaimContent(&m),
				Domain:        SelfDomain,
				MethodologyId: MethodologyID,
			},
		},
		RecursionWeight: ComputeAxisProjection(&m),
		Source: &substratebridgetypes.ExternalSource{
			AdapterId:      AdapterID,
			SourceId:       m.Hash,
			SourceUrl:      SourceURLBase + m.Hash,
			ContentHash:    canonicalCommitContentHash(&m),
			FetchedAtBlock: fetchedAtBlock,
		},
	}
	link.LinkHash = substratebridgekeeper.ComputeLinkHash(link)
	return link, nil
}
