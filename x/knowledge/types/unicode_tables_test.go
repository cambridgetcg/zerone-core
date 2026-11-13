package types

// Regression tests for DET-2 (consensus review 2026-11-05): canonical-form
// hashes and index keys follow the build toolchain's Unicode tables. The
// witnesses below are code points assigned after Unicode 15.0.0, taken from
// the review's reproduction. Under Go 1.25.14 and 1.26.10 (Unicode 15.0.0)
// they normalise as asserted here; under Go 1.27.3 (Unicode 17.0.0) every one
// of them changes, and so does each hash.

import (
	"strconv"
	"strings"
	"testing"

	"golang.org/x/text/unicode/norm"
)

func TestValidateUnicodeTables_BuildToolchain(t *testing.T) {
	if err := ValidateUnicodeTables(); err != nil {
		t.Fatalf("this toolchain's Unicode tables are not the chain's pin: %v", err)
	}
}

func TestValidateUnicodeTables_RefusesOtherTables(t *testing.T) {
	if err := validateUnicodeTables(ConsensusUnicodeVersion, ConsensusUnicodeVersion, "go1.25.14"); err != nil {
		t.Fatalf("pinned tables refused: %v", err)
	}
	for _, tc := range []struct{ stdlib, nfc string }{
		{"17.0.0", "17.0.0"}, // Go 1.27
		{ConsensusUnicodeVersion, "17.0.0"},
		{"17.0.0", ConsensusUnicodeVersion},
		{"", ""},
	} {
		err := validateUnicodeTables(tc.stdlib, tc.nfc, "go1.27.3")
		if err == nil {
			t.Fatalf("stdlib %q, norm %q: accepted, want refusal", tc.stdlib, tc.nfc)
		}
		for _, want := range []string{ConsensusUnicodeVersion, "go1.27.3", "would fork"} {
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("refusal %q does not mention %q", err, want)
			}
		}
	}
}

// The pin is a promise about behaviour, not only a version string: these
// witnesses must normalise exactly as Unicode 15.0.0 tables do. If this fails
// while ValidateUnicodeTables passes, the pin itself was changed and the chain
// would fork from every binary built before the change.
func TestConsensusNormalisationMatchesPinnedTables(t *testing.T) {
	// U+A7CB LATIN CAPITAL LETTER RAMS HORN (Unicode 16): no lowercase mapping
	// and not printable under 15.0.0; Unicode 16 and later lower it to U+0264.
	if got := strings.ToLower("\uA7CB"); got != "\uA7CB" {
		t.Errorf("ToLower(U+A7CB) = %+q, want unchanged", got)
	}
	if strconv.IsPrint(0xA7CB) {
		t.Errorf("IsPrint(U+A7CB) = true, want false")
	}
	// U+10D50 GARAY CAPITAL LETTER A (Unicode 16): no lowercase under 15.0.0.
	if got := strings.ToLower("\U00010D50"); got != "\U00010D50" {
		t.Errorf("ToLower(U+10D50) = %+q, want unchanged", got)
	}
	// U+105D2 U+0307 (Todhri, Unicode 16): composes to U+105C9 only with
	// newer NFC tables.
	if got := norm.NFC.String("\U000105D2\u0307"); got != "\U000105D2\u0307" {
		t.Errorf("NFC(U+105D2 U+0307) = %+q, want unchanged", got)
	}

	for _, tc := range []struct {
		name string
		form string
		hash string
	}{
		{
			name: "v2 form from structure",
			form: BuildCanonicalFormV2(ClaimType_CLAIM_TYPE_ASSERTION, &ClaimStructure{Subject: "\U000105D2\u0307", Predicate: "p"}, "d"),
			hash: "dcde89e2cebe5485ca5ee944534998e367b21c7e47b7b215653b9b8b73e26a3a",
		},
		{
			name: "v1 form supplied by the caller",
			form: NormalizeCanonicalForm("assert(\"d\", \"\uA7CB\", \"p\")"),
			hash: "6c93cdd843c304642461ec6af59b2481ca30e883d060b888d9e11239730942c2",
		},
		{
			name: "v1 form from structure",
			form: BuildCanonicalForm(ClaimType_CLAIM_TYPE_ASSERTION, &ClaimStructure{Subject: "\uA7CB", Predicate: "p"}, "d"),
			hash: "485edc9347d0b9353592344fe0a3b7547084167385c1ac33fb6ec1cb9cf1b13b",
		},
	} {
		if got := HashCanonicalForm(tc.form); got != tc.hash {
			t.Errorf("%s: hash(%+q) = %s, want %s", tc.name, tc.form, got, tc.hash)
		}
	}
}
