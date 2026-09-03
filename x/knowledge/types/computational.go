package types

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
)

// Computational commitment digests deliberately follow the existing chain
// convention: bare, lowercase SHA-256 hex (no "sha256:" multihash prefix).
const SHA256HexLength = 64

// ValidateSHA256Hex validates the single canonical wire representation used
// for computational commitments. Rejecting uppercase and algorithm prefixes
// prevents semantically identical commitments from acquiring different IDs.
func ValidateSHA256Hex(field, value string) error {
	if len(value) != SHA256HexLength {
		return fmt.Errorf("%s must be %d lowercase SHA-256 hex characters", field, SHA256HexLength)
	}
	if value != strings.ToLower(value) {
		return fmt.Errorf("%s must use lowercase SHA-256 hex", field)
	}
	if _, err := hex.DecodeString(value); err != nil {
		return fmt.Errorf("%s must be valid lowercase SHA-256 hex: %w", field, err)
	}
	return nil
}

// Validate requires the complete seven-digest commitment. Partial
// computational provenance is fail-closed: omitting any root makes exact work
// binding and settlement replay protection impossible.
func (c *ComputationalCommitment) Validate() error {
	if c == nil {
		return fmt.Errorf("computational_commitment is required")
	}
	fields := []struct {
		name  string
		value string
	}{
		{"work_spec_hash", c.WorkSpecHash},
		{"acceptance_hash", c.AcceptanceHash},
		{"input_root", c.InputRoot},
		{"environment_root", c.EnvironmentRoot},
		{"artifact_root", c.ArtifactRoot},
		{"evidence_root", c.EvidenceRoot},
		{"work_receipt_hash", c.WorkReceiptHash},
	}
	for _, field := range fields {
		if err := ValidateSHA256Hex(field.name, field.value); err != nil {
			return err
		}
	}
	return nil
}

// ValidateComputationalClaim enforces the epistemic type wall: a complete
// commitment is mandatory for computational claims and forbidden everywhere
// else. A commitment records provenance; it does not confer truth standing.
func ValidateComputationalClaim(claimType ClaimType, c *ComputationalCommitment) error {
	if claimType == ClaimType_CLAIM_TYPE_COMPUTATIONAL {
		return c.Validate()
	}
	if c != nil {
		return fmt.Errorf("computational_commitment is only valid for CLAIM_TYPE_COMPUTATIONAL")
	}
	return nil
}

// ComputeWorkReceiptHash defines the terminal receipt binding used by
// sponsorship v2. It commits the full sponsor-facing contract, result roots,
// and the eventual chain payee. Evidence may contain signatures and detailed
// evaluation records off chain; the consensus receipt is their compact anchor.
//
// Preimage (all values uint64 big-endian length-prefixed UTF-8):
//
//	"ZRN.work.receipt.v1\0", work_spec_hash, acceptance_hash, input_root,
//	environment_root, artifact_root, evidence_root, payee.
func ComputeWorkReceiptHash(c *ComputationalCommitment, payee string) string {
	h := sha256.New()
	h.Write([]byte("ZRN.work.receipt.v1\x00"))
	writePart := func(value string) {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(value)))
		h.Write(size[:])
		h.Write([]byte(value))
	}
	writePart(c.WorkSpecHash)
	writePart(c.AcceptanceHash)
	writePart(c.InputRoot)
	writePart(c.EnvironmentRoot)
	writePart(c.ArtifactRoot)
	writePart(c.EvidenceRoot)
	writePart(payee)
	return hex.EncodeToString(h.Sum(nil))
}

// ValidateWorkReceiptBinding rejects a receipt that can be replayed after
// changing the artifact, evidence, environment, sponsor contract, or payee.
func ValidateWorkReceiptBinding(c *ComputationalCommitment, payee string) error {
	if c == nil {
		return fmt.Errorf("computational_commitment is required")
	}
	if got, want := c.WorkReceiptHash, ComputeWorkReceiptHash(c, payee); got != want {
		return fmt.Errorf("work_receipt_hash does not bind the stored computational commitment and payee: got %s want %s", got, want)
	}
	return nil
}
