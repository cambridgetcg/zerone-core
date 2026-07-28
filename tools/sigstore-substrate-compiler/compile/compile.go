// Package compile deterministically projects an already-verified Sigstore
// bundle and its in-toto DSSE payload into Zerone's witness-only
// substrate-link shape.
//
// This package deliberately performs no signature verification. Its public
// API accepts only the opaque value returned by the verification package.
package compile

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"

	"github.com/zerone-chain/zerone/tools/sigstore-substrate-compiler/verification"
	substratebridgekeeper "github.com/zerone-chain/zerone/x/substrate_bridge/keeper"
	substratebridgetypes "github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

const (
	// AdapterID is fixed so it cannot drift from the governance registration.
	AdapterID = "sigstore-in-toto-v1"

	// MaxPayloadBytes bounds the verified statement held in memory. The bundle
	// verifier applies the same ceiling before doing cryptographic work.
	MaxPayloadBytes = 8 << 20

	// MaxBundleBytes bounds the exact Sigstore proof material committed by the
	// ExternalSource content hash.
	MaxBundleBytes = 16 << 20

	maxSourceURLBytes = 2048
)

// Input accepts only the opaque attestation returned by
// verification.VerifyBundle. This prevents callers from pairing proof bytes
// from one bundle with a payload from another.
type Input struct {
	Attestation    *verification.VerifiedAttestation
	SourceURL      string
	FetchedAtBlock uint64
}

// validateMaterial rejects inputs that would produce an ambiguous or
// unauditable source. sourceURL is unauthenticated audit metadata and is
// constrained to a public HTTPS URL with no credential-bearing components.
func validateMaterial(bundleJSON, dssePayload []byte, sourceURL string) error {
	if len(bundleJSON) == 0 {
		return fmt.Errorf("bundle JSON must be non-empty")
	}
	if len(bundleJSON) > MaxBundleBytes {
		return fmt.Errorf("bundle JSON exceeds %d-byte limit", MaxBundleBytes)
	}
	if len(dssePayload) == 0 {
		return fmt.Errorf("DSSE payload must be non-empty")
	}
	if len(dssePayload) > MaxPayloadBytes {
		return fmt.Errorf("DSSE payload exceeds %d-byte limit", MaxPayloadBytes)
	}
	if sourceURL == "" {
		return fmt.Errorf("source URL must be non-empty")
	}
	if sourceURL != strings.TrimSpace(sourceURL) {
		return fmt.Errorf("source URL must not have surrounding whitespace")
	}
	if len(sourceURL) > maxSourceURLBytes {
		return fmt.Errorf("source URL exceeds %d-byte limit", maxSourceURLBytes)
	}
	parsed, err := url.Parse(sourceURL)
	if err != nil {
		return fmt.Errorf("source URL must be an absolute HTTPS URL: %w", err)
	}
	if parsed.Scheme != "https" || parsed.Host == "" {
		return fmt.Errorf("source URL must be an absolute HTTPS URL")
	}
	if parsed.User != nil {
		return fmt.Errorf("source URL must not contain credentials")
	}
	if parsed.RawQuery != "" || parsed.ForceQuery {
		return fmt.Errorf("source URL must not contain a query")
	}
	if parsed.Fragment != "" {
		return fmt.Errorf("source URL must not contain a fragment")
	}
	return nil
}

// Compile hashes the exact verified DSSE payload for payload-byte source
// identity and the exact accepted bundle bytes for the proof-material content
// hash. It creates no claims, citations, or recursion weight, so the result
// itself asks for no knowledge creation or economic weight.
func Compile(input Input) (*substratebridgetypes.SubstrateLink, error) {
	bundleJSON, dssePayload, err := input.Attestation.CompilationMaterial()
	if err != nil {
		return nil, fmt.Errorf("invalid compiler input: %w", err)
	}
	return compileMaterial(bundleJSON, dssePayload, input.SourceURL, input.FetchedAtBlock)
}

// compileMaterial is the deterministic core kept private so the public API
// cannot break the verifier's bundle/payload pairing.
func compileMaterial(bundleJSON, dssePayload []byte, sourceURL string, fetchedAtBlock uint64) (*substratebridgetypes.SubstrateLink, error) {
	if err := validateMaterial(bundleJSON, dssePayload, sourceURL); err != nil {
		return nil, fmt.Errorf("invalid compiler input: %w", err)
	}

	payloadHash := sha256.Sum256(dssePayload)
	bundleHash := sha256.Sum256(bundleJSON)
	bundleHashBytes := append([]byte(nil), bundleHash[:]...)
	sourceID := "sha256:" + hex.EncodeToString(payloadHash[:])

	link := &substratebridgetypes.SubstrateLink{
		CitedFacts:      nil,
		PendingClaims:   nil,
		RecursionWeight: nil,
		AdapterId:       AdapterID,
		Source: &substratebridgetypes.ExternalSource{
			AdapterId:      AdapterID,
			SourceId:       sourceID,
			SourceUrl:      sourceURL,
			ContentHash:    bundleHashBytes,
			FetchedAtBlock: fetchedAtBlock,
		},
	}
	link.LinkHash = substratebridgekeeper.ComputeLinkHash(link)
	return link, nil
}
