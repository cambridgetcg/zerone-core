package types

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/text/unicode/norm"
)

// Canonical form v2 (knowledge v12, plan Task 9; P2). Each structure field is
// normalised (NFC, trimmed, whitespace runs collapsed to one space) and quoted
// on its own, so object, scope, temporal_scope, case and claim type all take
// part in the hash, and no field value can imitate a delimiter.

func boilingPoint() *ClaimStructure {
	return &ClaimStructure{Subject: "boiling point of compound Z", Predicate: "equals degrees C at 1 atm", Object: "78.4", Scope: "pure sample"}
}

// The synthetic example of plan §4.3. The hash was computed independently
// with `printf '%s' <form> | shasum -a 256`.
func TestCanonicalFormV2_PlanExample(t *testing.T) {
	const want = `zrn.cf.v2/CLAIM_TYPE_ASSERTION(domain="chemistry",subject="boiling point of compound Z",predicate="equals degrees C at 1 atm",object="78.4",scope="pure sample",temporal_scope="")`
	form := BuildCanonicalFormV2(ClaimType_CLAIM_TYPE_ASSERTION, boilingPoint(), "chemistry")
	require.Equal(t, want, form)
	require.True(t, strings.HasPrefix(form, CanonicalFormV2Prefix))
	require.Equal(t, "7c754ede41adc2cd62468985e42079ad673e090a77be7fedd4373cbac26cc790", HashCanonicalForm(form))
}

func TestCanonicalFormV2_EmptyWithoutStructureOrSubject(t *testing.T) {
	require.Empty(t, BuildCanonicalFormV2(ClaimType_CLAIM_TYPE_ASSERTION, nil, "chemistry"))
	for _, subject := range []string{"", " ", "\t\n", "\u00a0\u3000"} {
		s := boilingPoint()
		s.Subject = subject
		require.Empty(t, BuildCanonicalFormV2(ClaimType_CLAIM_TYPE_ASSERTION, s, "chemistry"), "subject %q", subject)
	}
}

func allClaimTypes() []ClaimType {
	var out []ClaimType
	for ct := ClaimType_CLAIM_TYPE_UNSPECIFIED; ct <= ClaimType_CLAIM_TYPE_CONJECTURE; ct++ {
		out = append(out, ct)
	}
	return out
}

// Every field takes part in the hash for every claim type; v1 dropped object
// and temporal_scope from most types and scope from OBSERVATION.
func TestCanonicalFormV2_EveryFieldChangesTheHashForEveryType(t *testing.T) {
	edits := map[string]func(*ClaimStructure, *string){
		"domain":         func(_ *ClaimStructure, d *string) { *d = "physics" },
		"subject":        func(s *ClaimStructure, _ *string) { s.Subject = "boiling point of compound Y" },
		"predicate":      func(s *ClaimStructure, _ *string) { s.Predicate = "equals degrees F at 1 atm" },
		"object":         func(s *ClaimStructure, _ *string) { s.Object = "81.0" },
		"scope":          func(s *ClaimStructure, _ *string) { s.Scope = "impure sample" },
		"temporal_scope": func(s *ClaimStructure, _ *string) { s.TemporalScope = "2021-01-01" },
	}
	for _, ct := range allClaimTypes() {
		base := HashCanonicalForm(BuildCanonicalFormV2(ct, boilingPoint(), "chemistry"))
		for name, edit := range edits {
			s, domain := boilingPoint(), "chemistry"
			edit(s, &domain)
			got := HashCanonicalForm(BuildCanonicalFormV2(ct, s, domain))
			require.NotEqual(t, base, got, "%s: %s does not change the hash", ct, name)
		}
	}
}

// v2 never lowercases: 5 mW and 5 MW are different findings.
func TestCanonicalFormV2_KeepsCase(t *testing.T) {
	milli, mega := boilingPoint(), boilingPoint()
	milli.Scope, mega.Scope = "output 5 mW", "output 5 MW"
	require.NotEqual(t,
		BuildCanonicalFormV2(ClaimType_CLAIM_TYPE_ASSERTION, milli, "physics"),
		BuildCanonicalFormV2(ClaimType_CLAIM_TYPE_ASSERTION, mega, "physics"))
	milli, mega = boilingPoint(), boilingPoint()
	milli.Object, mega.Object = "5 mW", "5 MW"
	require.NotEqual(t,
		BuildCanonicalFormV2(ClaimType_CLAIM_TYPE_ASSERTION, milli, "physics"),
		BuildCanonicalFormV2(ClaimType_CLAIM_TYPE_ASSERTION, mega, "physics"))
}

// Fields are named, so a value cannot move between optional slots.
func TestCanonicalFormV2_ObjectOnlyAndScopeOnlyDiffer(t *testing.T) {
	for _, ct := range allClaimTypes() {
		objectOnly := &ClaimStructure{Subject: "compound Z", Predicate: "is measured", Object: "pure sample"}
		scopeOnly := &ClaimStructure{Subject: "compound Z", Predicate: "is measured", Scope: "pure sample"}
		temporalOnly := &ClaimStructure{Subject: "compound Z", Predicate: "is measured", TemporalScope: "pure sample"}
		forms := map[string]bool{}
		for _, s := range []*ClaimStructure{objectOnly, scopeOnly, temporalOnly} {
			forms[BuildCanonicalFormV2(ct, s, "chemistry")] = true
		}
		require.Len(t, forms, 3, "%s", ct)
	}
}

// The claim type is spelled out; only UNSPECIFIED is defaulted to ASSERTION.
func TestCanonicalFormV2_ClaimTypeTakesPart(t *testing.T) {
	s := boilingPoint()
	assertion := BuildCanonicalFormV2(ClaimType_CLAIM_TYPE_ASSERTION, s, "chemistry")
	require.Equal(t, assertion, BuildCanonicalFormV2(ClaimType_CLAIM_TYPE_UNSPECIFIED, s, "chemistry"))
	require.NotEqual(t, assertion, BuildCanonicalFormV2(ClaimType_CLAIM_TYPE_COMPUTATIONAL, s, "chemistry"))
	forms := map[string]ClaimType{}
	for _, ct := range allClaimTypes() {
		if ct == ClaimType_CLAIM_TYPE_UNSPECIFIED {
			continue
		}
		form := BuildCanonicalFormV2(ct, s, "chemistry")
		require.True(t, strings.HasPrefix(form, CanonicalFormV2Prefix+ct.String()+"("), form)
		if prior, seen := forms[form]; seen {
			t.Fatalf("%s and %s share form %s", prior, ct, form)
		}
		forms[form] = ct
	}
}

// Quotes, commas, parentheses, equals signs, backslashes, escapes and
// control characters inside a value never imitate a field boundary.
func TestCanonicalFormV2_DelimitersNeverCollide(t *testing.T) {
	values := []string{
		"a", "b", "a,b", `a"`, `"a`, `a\`, `\`, `\"`, `a",predicate="b`, `",object="`, `a"),scope=("`,
		"a)", "(a", "a=b", "a\x00", "a\x01", "a\x1b", "a\x7f", `a\x00`, `a\u0000`, "a\u200b", "\u200b",
		"a\u00e9", `a\u00e9`, "a\U0001f600", `a\U0001f600`, "a\ue00f", "a\u0378",
	}
	seen := map[string]string{}
	record := func(label string, s *ClaimStructure) {
		form := BuildCanonicalFormV2(ClaimType_CLAIM_TYPE_ASSERTION, s, "chemistry")
		require.NotEmpty(t, form)
		hash := HashCanonicalForm(form)
		if prior, dup := seen[hash]; dup {
			t.Fatalf("%s and %s collide: %s", prior, label, form)
		}
		seen[hash] = label
	}
	for _, x := range values {
		for _, y := range values {
			for _, z := range values {
				record(fmt.Sprintf("spo(%q,%q,%q)", x, y, z), &ClaimStructure{Subject: x, Predicate: y, Object: z})
				record(fmt.Sprintf("ost(%q,%q,%q)", x, y, z), &ClaimStructure{Subject: "s", Predicate: "p", Object: x, Scope: y, TemporalScope: z})
			}
		}
	}
}

// Equivalent spellings (NFC versus NFD, whitespace runs, padding) give one
// form; normalisation is idempotent at the field and at the form level; and
// the lookup normaliser leaves a v2 form exactly as built.
func TestCanonicalFormV2_NormalizationIsIdempotent(t *testing.T) {
	composed := &ClaimStructure{Subject: "caf\u00e9 au lait", Predicate: "is served at 70 \u00b0C", Object: "70", Scope: "in Z\u00fcrich"}
	decomposed := &ClaimStructure{Subject: "  cafe\u0301\t au\u00a0lait\n", Predicate: "is served at 70\u3000\u00b0C", Object: " 70 ", Scope: "in Zu\u0308rich\u2028"}
	require.NotEqual(t, composed.Subject, strings.TrimSpace(decomposed.Subject), "the inputs differ before normalisation")
	require.Equal(t,
		BuildCanonicalFormV2(ClaimType_CLAIM_TYPE_ASSERTION, composed, "chemistry"),
		BuildCanonicalFormV2(ClaimType_CLAIM_TYPE_ASSERTION, decomposed, "chemistry"))

	tricky := []string{
		"plain", "  padded  ", "a\t\n b", "cafe\u0301", "\u0301leading mark", "x\u200b\u0307", "x\ue00f\u0307",
		"x\u00ad\u0301", "\u2028line\u2029", "a\u0085b", "Zu\u0308rich", "Z\u00fcrich", "\U0001f600", "a\x00b", `a"b\c`, "a\u0378\u0301",
	}
	for _, v := range tricky {
		once := normalizeCanonicalFieldV2(v)
		require.Equal(t, once, normalizeCanonicalFieldV2(once), "field %q", v)

		s := &ClaimStructure{Subject: "s " + v, Predicate: v + " p", Object: v, Scope: v, TemporalScope: v}
		form := BuildCanonicalFormV2(ClaimType_CLAIM_TYPE_OBSERVATION, s, "chemistry")
		require.Equal(t, form, NormalizeCanonicalFormForLookup(form), "a built form is its own lookup form: %q", v)
		require.Equal(t, form, NormalizeCanonicalFormForLookup(" \n"+form+"\t "), "padding is trimmed: %q", v)
		require.Equal(t, form, norm.NFC.String(form), "a built form is NFC: %q", v)
		for i := 0; i < len(form); i++ {
			require.Less(t, form[i], byte(0x80), "a built form is ASCII: %q", v)
		}

		renormalized := &ClaimStructure{
			Subject: normalizeCanonicalFieldV2(s.Subject), Predicate: normalizeCanonicalFieldV2(s.Predicate),
			Object: normalizeCanonicalFieldV2(s.Object), Scope: normalizeCanonicalFieldV2(s.Scope),
			TemporalScope: normalizeCanonicalFieldV2(s.TemporalScope),
		}
		require.Equal(t, form, BuildCanonicalFormV2(ClaimType_CLAIM_TYPE_OBSERVATION, renormalized, "chemistry"), "%q", v)
	}

	for _, input := range []string{
		`zrn.cf.v2/CLAIM_TYPE_ASSERTION(domain="chemistry",subject="a",predicate="b",object="",scope="",temporal_scope="")`,
		"  zrn.cf.v2/CLAIM_TYPE_ASSERTION(domain=\"x\",subject=\"cafe\u0301\")\n",
		`ASSERT("Physics",  "Water")`, "  assert(\"physics\")  ", "", "   ",
	} {
		once := NormalizeCanonicalFormForLookup(input)
		require.Equal(t, once, NormalizeCanonicalFormForLookup(once), "%q", input)
	}
}

// A v2 form is hashed verbatim after NFC and trim (never lowercased); any
// other form keeps the v1 lookup normalisation.
func TestNormalizeCanonicalFormForLookup(t *testing.T) {
	v2 := BuildCanonicalFormV2(ClaimType_CLAIM_TYPE_ASSERTION, boilingPoint(), "chemistry")
	require.Equal(t, v2, NormalizeCanonicalFormForLookup("\t"+v2+"  "))
	require.NotEqual(t, strings.ToLower(v2), NormalizeCanonicalFormForLookup(v2))
	for _, v1 := range []string{
		`ASSERT("Physics",  "Water")`,
		"  assert(\"physics\", \"water\")  ",
		`relate("astronomy", "earth", "orbits", "sun")`,
		"ZRN.CF.V2/CLAIM_TYPE_ASSERTION(domain=\"x\")",
	} {
		require.Equal(t, NormalizeCanonicalForm(v1), NormalizeCanonicalFormForLookup(v1), "%q", v1)
	}
}

// BuildCanonicalForm (v1) is unchanged; pre-v12 records and the marker-off
// path keep hashing exactly as on knowledge 11. Values recorded on a4272bfa.
func TestCanonicalFormV1_GoldenFormsUnchanged(t *testing.T) {
	s := &ClaimStructure{Subject: " Boiling point of Compound Z ", Predicate: "Equals  degrees C", Object: "78.4", Scope: "Pure sample", TemporalScope: "2019-03-01"}
	const assertForm = `assert("chemistry", "boiling point of compound z", "equals  degrees c", "pure sample")`
	const assertHash = "653382030ca315c084a0e4914c73cf5a17d95d10163509c6c59a28befae10e16"
	for _, tc := range []struct {
		ct         ClaimType
		form, hash string
	}{
		{ClaimType_CLAIM_TYPE_UNSPECIFIED, assertForm, assertHash},
		{ClaimType_CLAIM_TYPE_ASSERTION, assertForm, assertHash},
		{ClaimType_CLAIM_TYPE_RELATION, `relate("chemistry", "boiling point of compound z", "equals  degrees c", "78.4", "pure sample")`, "ba3421dbe8ca10ae4ebfbfd72f1d8d47b1b8a0cf6f68e70e1ba382bcc28fa332"},
		{ClaimType_CLAIM_TYPE_DEFINITION, `define("chemistry", "boiling point of compound z", "equals  degrees c")`, "94ae45f760fac88805c9624046973dd8fd0d041314bcdf4e56dc33b0559e7237"},
		{ClaimType_CLAIM_TYPE_CONSTRAINT, `constrain("chemistry", "boiling point of compound z", "equals  degrees c", "pure sample")`, "32748a8cb9d35eae4cfc2207e62f927a3692dd0ffc3316608fae975dfa7846b5"},
		{ClaimType_CLAIM_TYPE_NEGATION, `negate("chemistry", "boiling point of compound z", "equals  degrees c", "pure sample")`, "94d844f6d9bd27d6f08f0f258640fdd61d4bab027b0250e661de870676da0316"},
		{ClaimType_CLAIM_TYPE_OBSERVATION, `observe("chemistry", "boiling point of compound z", "equals  degrees c", "2019-03-01")`, "844d9fe9d7ef272710da75e4c3cbc97680e1366e6e0d705f0177f6e0c2254112"},
		{ClaimType_CLAIM_TYPE_COMPUTATIONAL, assertForm, assertHash},
		{ClaimType_CLAIM_TYPE_CONJECTURE, assertForm, assertHash},
	} {
		form := BuildCanonicalForm(tc.ct, s, "Chemistry")
		require.Equal(t, tc.form, form, "%s", tc.ct)
		require.Equal(t, tc.hash, HashCanonicalForm(form), "%s", tc.ct)
		require.Equal(t, form, NormalizeCanonicalForm(form))
	}
}
