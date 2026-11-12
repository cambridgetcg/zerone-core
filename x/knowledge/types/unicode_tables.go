package types

import (
	"fmt"
	"runtime"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// ConsensusUnicodeVersion is the Unicode version of the character tables that
// every zerone binary on a chain must share.
//
// Claim canonical forms run through norm.NFC (golang.org/x/text), and the v1
// form also through strings.ToLower and fmt's %q; subject, niche and tag index
// keys use strings.ToLower. All of these follow tables compiled into the
// binary: the Go standard library's (unicode, strings and strconv move
// together) and x/text's, which picks its NFC tables by Go release build tag.
// When a Go release adopts a newer Unicode version, code points newly assigned
// in it lower, compose or quote differently, so a claim using them hashes to a
// different canonical key. Two binaries with different tables then disagree on
// claim deduplication and state keys, and the chain forks on the first such
// claim.
//
// Go 1.24.13 and 1.25.14 (the toolchains of the live and trunk builds) ship
// Unicode 15.0.0 in both places; Go 1.27 ships 17.0.0. Raising this pin is a
// consensus change: it needs a named upgrade, and a census of every stored
// value the new tables would change. That covers claim canonical forms and the
// subject, niche and tag keys, and also the common-knowledge key
// (normalizeSubjectForNovelty in x/knowledge/keeper/novelty.go, strings.ToLower),
// %q-formatted emergency audit details (x/emergency/keeper/upgrade_safety.go),
// and the bank metadata symbol ibc-go's transfer keeper stores as
// strings.ToUpper of a received denom's base.
//
// The guard compares Unicode version strings, so its guarantee is the Unicode
// version, not every table. Releases that report the same version can still
// differ elsewhere: go1.24.13 and go1.25.14 differ in the C, Cn and LC category
// tables, and x/text's NFC code changed between v0.34.0 and v0.39.0. Nothing on
// a consensus path reads those today. The exact toolchain pin (go.mod and
// make go-version-check) is the control for everything else; by design the
// runtime guard does not check the Go release.
const ConsensusUnicodeVersion = "15.0.0"

// ValidateUnicodeTables returns an error unless this binary's Unicode tables
// are ConsensusUnicodeVersion. A node must refuse to start on such an error:
// running on would fork it from the chain on the first claim that uses a
// newly assigned code point.
func ValidateUnicodeTables() error {
	return validateUnicodeTables(unicode.Version, norm.Version, runtime.Version())
}

func validateUnicodeTables(stdlibVersion, normVersion, goVersion string) error {
	if stdlibVersion == ConsensusUnicodeVersion && normVersion == ConsensusUnicodeVersion {
		return nil
	}
	return fmt.Errorf(
		"unicode tables differ from the chain's pinned Unicode %s: Go standard library has %s and golang.org/x/text/unicode/norm has %s (built with %s). "+
			"Claim canonical-form hashes and index keys would not match other nodes, so this binary would fork; "+
			"rebuild with the Go toolchain go.mod names (make go-version-check)",
		ConsensusUnicodeVersion, stdlibVersion, normVersion, goVersion,
	)
}
