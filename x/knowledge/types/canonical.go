package types

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// claimTypeFuncName maps ClaimType enum values to canonical form function names.
var claimTypeFuncName = map[ClaimType]string{
	ClaimType_CLAIM_TYPE_UNSPECIFIED: "assert",
	ClaimType_CLAIM_TYPE_ASSERTION:  "assert",
	ClaimType_CLAIM_TYPE_RELATION:   "relate",
	ClaimType_CLAIM_TYPE_DEFINITION: "define",
	ClaimType_CLAIM_TYPE_CONSTRAINT: "constrain",
	ClaimType_CLAIM_TYPE_NEGATION:   "negate",
	ClaimType_CLAIM_TYPE_OBSERVATION: "observe",
}

// BuildCanonicalForm constructs a canonical form from a claim's type and structure.
// Returns empty string if structure is insufficient (nil or missing subject).
//
// Grammar:
//
//	ASSERTION:   assert(domain, subject, predicate[, scope])
//	RELATION:    relate(domain, subject, relation_verb, object[, scope])
//	DEFINITION:  define(domain, term, meaning)
//	CONSTRAINT:  constrain(domain, subject, constraint[, scope])
//	NEGATION:    negate(domain, subject, predicate[, scope])
//	OBSERVATION: observe(domain, subject, predicate, temporal_scope)
func BuildCanonicalForm(claimType ClaimType, structure *ClaimStructure, domain string) string {
	if structure == nil || structure.Subject == "" {
		return ""
	}

	funcName, ok := claimTypeFuncName[claimType]
	if !ok {
		funcName = "assert"
	}

	var args []string
	args = append(args, quoteArg(domain))

	switch claimType {
	case ClaimType_CLAIM_TYPE_DEFINITION:
		// define(domain, term, meaning)
		args = append(args, quoteArg(structure.Subject))
		if structure.Predicate != "" {
			args = append(args, quoteArg(structure.Predicate))
		}

	case ClaimType_CLAIM_TYPE_RELATION:
		// relate(domain, subject, relation_verb, object[, scope])
		args = append(args, quoteArg(structure.Subject))
		if structure.Predicate != "" {
			args = append(args, quoteArg(structure.Predicate))
		}
		if structure.Object != "" {
			args = append(args, quoteArg(structure.Object))
		}
		if structure.Scope != "" {
			args = append(args, quoteArg(structure.Scope))
		}

	case ClaimType_CLAIM_TYPE_OBSERVATION:
		// observe(domain, subject, predicate, temporal_scope)
		args = append(args, quoteArg(structure.Subject))
		if structure.Predicate != "" {
			args = append(args, quoteArg(structure.Predicate))
		}
		if structure.TemporalScope != "" {
			args = append(args, quoteArg(structure.TemporalScope))
		}

	default:
		// assert, constrain, negate: (domain, subject, predicate[, scope])
		args = append(args, quoteArg(structure.Subject))
		if structure.Predicate != "" {
			args = append(args, quoteArg(structure.Predicate))
		}
		if structure.Scope != "" {
			args = append(args, quoteArg(structure.Scope))
		}
	}

	form := fmt.Sprintf("%s(%s)", funcName, strings.Join(args, ", "))
	return NormalizeCanonicalForm(form)
}

// CanonicalFormV2Prefix marks a canonical form v2 (knowledge v12, claim
// integrity). A lookup form with this prefix is hashed verbatim after NFC and
// trimming; any other form keeps the v1 normalisation.
const CanonicalFormV2Prefix = "zrn.cf.v2/"

// BuildCanonicalFormV2 constructs the v2 canonical form that claim-integrity
// chains derive from a claim's type and structure:
//
//	zrn.cf.v2/<CLAIM_TYPE_…>(domain="…",subject="…",predicate="…",object="…",scope="…",temporal_scope="…")
//
// Every field is present, named, normalised (normalizeCanonicalFieldV2) and
// quoted on its own; the form is never post-processed, lowercased or
// collapsed as a whole. Tags and negatable do not take part. UNSPECIFIED is
// defaulted to ASSERTION, as at admission. Returns "" when the structure is nil
// or its normalised subject is empty.
func BuildCanonicalFormV2(claimType ClaimType, structure *ClaimStructure, domain string) string {
	if structure == nil || normalizeCanonicalFieldV2(structure.Subject) == "" {
		return ""
	}
	if claimType == ClaimType_CLAIM_TYPE_UNSPECIFIED {
		claimType = ClaimType_CLAIM_TYPE_ASSERTION
	}
	return CanonicalFormV2Prefix + claimType.String() +
		"(domain=" + quoteCanonicalFieldV2(domain) +
		",subject=" + quoteCanonicalFieldV2(structure.Subject) +
		",predicate=" + quoteCanonicalFieldV2(structure.Predicate) +
		",object=" + quoteCanonicalFieldV2(structure.Object) +
		",scope=" + quoteCanonicalFieldV2(structure.Scope) +
		",temporal_scope=" + quoteCanonicalFieldV2(structure.TemporalScope) + ")"
}

// NormalizeCanonicalFormForLookup prepares a caller-supplied canonical form
// for hashing. A v2 form (CanonicalFormV2Prefix after NFC and trim) is used
// verbatim after NFC and trim; anything else is a v1 form and gets
// NormalizeCanonicalForm.
func NormalizeCanonicalFormForLookup(form string) string {
	trimmed := strings.TrimSpace(norm.NFC.String(form))
	if strings.HasPrefix(trimmed, CanonicalFormV2Prefix) {
		return trimmed
	}
	return NormalizeCanonicalForm(form)
}

// normalizeCanonicalFieldV2 is NFC, then trim, then every run of
// unicode.IsSpace collapsed to one U+0020. Case is kept.
func normalizeCanonicalFieldV2(v string) string {
	return strings.Join(strings.Fields(norm.NFC.String(v)), " ")
}

// quoteCanonicalFieldV2 quotes a normalised field with strconv.QuoteToASCII,
// so the whole form is ASCII. Two reasons, both found while building v2:
//   - strconv.Quote escapes only non-printable runes, and an escape ending in
//     a hex letter can compose with a following combining mark under NFC
//     (ZWSP then U+0307 gives `\u200b` + U+0307 -> `\u200` + U+1E03). The
//     lookup normaliser would then change a form the chain built, and
//     FactByCanonical or a declared form would miss it.
//   - which runes strconv.Quote treats as printable follows the Go
//     toolchain's Unicode tables, so a newly assigned code point would change
//     the form, and the dedup decision, between binaries. QuoteToASCII
//     escapes every non-ASCII rune whatever the Unicode version.
func quoteCanonicalFieldV2(v string) string {
	return strconv.QuoteToASCII(normalizeCanonicalFieldV2(v))
}

// NormalizeCanonicalForm applies normalization rules to a canonical form string:
// 1. All strings lowercased
// 2. Leading/trailing whitespace trimmed
// 3. Internal whitespace collapsed to single space
// 4. UTF-8 normalized to NFC
func NormalizeCanonicalForm(form string) string {
	// NFC normalization
	form = norm.NFC.String(form)
	// Lowercase
	form = strings.ToLower(form)
	// Trim
	form = strings.TrimSpace(form)
	// Collapse internal whitespace
	form = collapseWhitespace(form)
	return form
}

// HashCanonicalForm returns the SHA-256 hex digest of a normalized canonical form.
func HashCanonicalForm(form string) string {
	h := sha256.Sum256([]byte(form))
	return hex.EncodeToString(h[:])
}

// quoteArg wraps a canonical form argument in double quotes.
func quoteArg(s string) string {
	s = strings.TrimSpace(s)
	return fmt.Sprintf("%q", s)
}

// collapseWhitespace replaces runs of whitespace with a single space,
// except inside quoted strings.
func collapseWhitespace(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inQuote := false
	prevSpace := false
	for _, r := range s {
		if r == '"' {
			inQuote = !inQuote
			b.WriteRune(r)
			prevSpace = false
			continue
		}
		if !inQuote && unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteRune(' ')
			}
			prevSpace = true
		} else {
			b.WriteRune(r)
			prevSpace = false
		}
	}
	return b.String()
}
