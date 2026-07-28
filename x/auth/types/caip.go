package types

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"

	"github.com/cosmos/cosmos-sdk/types/bech32"
)

const (
	// CosmosCAIPNamespace is the CAIP-2 namespace assigned to Cosmos chains.
	CosmosCAIPNamespace = "cosmos"

	hashedChainReferencePrefix = "hashed-"
	zeroneAccountAddressPrefix = "zrn"
	cosmosAccountAddressBytes  = 20
)

var directCosmosChainReference = regexp.MustCompile(`^[-a-zA-Z0-9]{1,32}$`)

// CosmosChainReference returns the chain reference described by the draft
// Cosmos CAIP-2 namespace profile. Chain IDs that cannot be represented
// directly are reduced to its deterministic "hashed-" reference.
func CosmosChainReference(chainID string) (string, error) {
	if chainID == "" {
		return "", fmt.Errorf("chain ID cannot be empty")
	}

	if directCosmosChainReference.MatchString(chainID) &&
		!strings.HasPrefix(chainID, hashedChainReferencePrefix) {
		return chainID, nil
	}

	sum := sha256.Sum256([]byte(chainID))
	return fmt.Sprintf("%s%x", hashedChainReferencePrefix, sum[:8]), nil
}

// CAIP2ChainID returns the CAIP-2 identifier for a Cosmos chain.
func CAIP2ChainID(chainID string) (string, error) {
	reference, err := CosmosChainReference(chainID)
	if err != nil {
		return "", err
	}
	return CosmosCAIPNamespace + ":" + reference, nil
}

// CAIP10AccountID returns a CAIP-10-syntax identifier for a canonical Zerone
// account address on the supplied Cosmos chain. The generic CAIP-10 format is
// Final; the Cosmos namespace's address profile remains Draft and does not yet
// name Zerone's zrn HRP.
func CAIP10AccountID(chainID, address string) (string, error) {
	hrp, addressBytes, err := bech32.DecodeAndConvert(address)
	if err != nil {
		return "", fmt.Errorf("invalid Zerone account address: %w", err)
	}
	if hrp != zeroneAccountAddressPrefix {
		return "", fmt.Errorf("invalid Zerone account prefix: expected %q, got %q", zeroneAccountAddressPrefix, hrp)
	}
	if len(addressBytes) != cosmosAccountAddressBytes {
		return "", fmt.Errorf("invalid Zerone account address length: expected %d bytes, got %d", cosmosAccountAddressBytes, len(addressBytes))
	}
	canonical, err := bech32.ConvertAndEncode(zeroneAccountAddressPrefix, addressBytes)
	if err != nil {
		return "", fmt.Errorf("cannot canonicalize Zerone account address: %w", err)
	}
	if address != canonical {
		return "", fmt.Errorf("account address must use canonical bech32 form %q", canonical)
	}

	caip2, err := CAIP2ChainID(chainID)
	if err != nil {
		return "", err
	}
	return caip2 + ":" + address, nil
}
