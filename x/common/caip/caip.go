// Package caip contains small, state-free helpers for Chain Agnostic
// Improvement Proposal identifiers.
package caip

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

const cosmosNamespace = "cosmos"

var directCosmosReferencePattern = regexp.MustCompile(`^[-a-zA-Z0-9]{1,32}$`)

// CosmosChainID returns the CAIP-2 identifier for a Cosmos chain ID using the
// Cosmos namespace profile.
//
// Directly representable IDs such as "zerone-2" become "cosmos:zerone-2".
// Longer IDs, non-ASCII IDs, IDs containing characters outside the Cosmos
// direct-reference grammar, and IDs beginning with the reserved "hashed-"
// prefix use the profile's hashed fallback.
func CosmosChainID(chainID string) (string, error) {
	if chainID == "" {
		return "", fmt.Errorf("cosmos chain ID is required")
	}

	reference := chainID
	if !directCosmosReferencePattern.MatchString(chainID) || strings.HasPrefix(chainID, "hashed-") {
		sum := sha256.Sum256([]byte(chainID))
		reference = "hashed-" + hex.EncodeToString(sum[:8])
	}
	return cosmosNamespace + ":" + reference, nil
}
