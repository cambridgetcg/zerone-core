// Package contentid contains state-free helpers for content-addressed
// identifiers used at Zerone's client boundaries.
package contentid

import (
	"fmt"

	cid "github.com/ipfs/go-cid"
)

const MaxMemoryCIDBytes = 256

// ParseCanonicalV1 parses value as a CIDv1 and requires Zerone's chosen
// lowercase-base32 text representation. CID itself permits other multibase
// strings for the same identifier. This helper deliberately does not impose a
// codec or multihash policy; callers must choose those semantics for their
// artifact class.
func ParseCanonicalV1(value string) (cid.Cid, error) {
	if value == "" {
		return cid.Undef, fmt.Errorf("CID is required")
	}

	parsed, err := cid.Decode(value)
	if err != nil {
		return cid.Undef, fmt.Errorf("decode CID: %w", err)
	}
	if parsed.Version() != 1 {
		return cid.Undef, fmt.Errorf("CID version must be 1, got %d", parsed.Version())
	}

	canonical := parsed.String()
	if value != canonical {
		return cid.Undef, fmt.Errorf("CIDv1 must use Zerone's lowercase-base32 representation %q", canonical)
	}
	return parsed, nil
}

// ParseMemoryV1 applies x/home's current text bound in addition to Zerone's
// CIDv1 representation rules. The bound is a client preflight constraint; it
// does not change validator handling of historical opaque values.
func ParseMemoryV1(value string) (cid.Cid, error) {
	if len(value) > MaxMemoryCIDBytes {
		return cid.Undef, fmt.Errorf("memory CID exceeds the on-chain %d-byte limit", MaxMemoryCIDBytes)
	}
	return ParseCanonicalV1(value)
}
