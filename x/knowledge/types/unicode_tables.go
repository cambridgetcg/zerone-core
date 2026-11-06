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
// consensus change: it needs a named upgrade, and a census of stored claims
// whose canonical form the new tables would change.
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
