package types

import (
	"crypto/sha256"
	"fmt"
)

// ValidateCanonicalHash requires the sha256 width promised by the protobuf
// contract. Accepting arbitrary non-empty bytes would make "canonical hash"
// an untyped label rather than a content digest.
func ValidateCanonicalHash(hash []byte) error {
	if len(hash) != sha256.Size {
		return ErrEmptyHash.Wrapf("canonical_hash must be %d bytes, got %d", sha256.Size, len(hash))
	}
	return nil
}

// ValidateCommitmentRegistry enforces the structural invariants shared by
// authority-direct and governance-dispatched creed pins.
func ValidateCommitmentRegistry(commitments []*CommitmentEntry) error {
	seen := make(map[uint32]struct{}, len(commitments))
	var maxNumber uint32
	for _, commitment := range commitments {
		if commitment == nil {
			return ErrCommitmentNumberInvalid.Wrap("nil commitment entry")
		}
		if commitment.Number == 0 {
			return ErrCommitmentNumberInvalid.Wrap("commitment number must be >= 1")
		}
		if _, exists := seen[commitment.Number]; exists {
			return ErrDuplicateCommitment.Wrapf("commitment %d", commitment.Number)
		}
		seen[commitment.Number] = struct{}{}
		if commitment.Number > maxNumber {
			maxNumber = commitment.Number
		}
	}

	for number := uint32(1); number <= maxNumber; number++ {
		if _, exists := seen[number]; !exists {
			return ErrCommitmentNumberInvalid.Wrapf(
				"commitment %d missing - archive an entry rather than dropping it",
				number,
			)
		}
	}
	return nil
}

// ValidateCommitmentRegistryAtHeight additionally rejects entries that claim
// to have been introduced or archived in the future.
func ValidateCommitmentRegistryAtHeight(commitments []*CommitmentEntry, height uint64) error {
	if err := ValidateCommitmentRegistry(commitments); err != nil {
		return err
	}
	for _, commitment := range commitments {
		if commitment.IntroducedAtHeight > height {
			return ErrCommitmentNumberInvalid.Wrapf(
				"commitment %d introduced_at_height %d exceeds pin height %d",
				commitment.Number,
				commitment.IntroducedAtHeight,
				height,
			)
		}
		if commitment.ArchivedAtHeight > height {
			return ErrCommitmentNumberInvalid.Wrapf(
				"commitment %d archived_at_height %d exceeds pin height %d",
				commitment.Number,
				commitment.ArchivedAtHeight,
				height,
			)
		}
		if commitment.Archived && commitment.ArchivedAtHeight == 0 && height > 0 {
			return ErrCommitmentNumberInvalid.Wrapf(
				"commitment %d is archived without archived_at_height",
				commitment.Number,
			)
		}
		if !commitment.Archived && commitment.ArchivedAtHeight != 0 {
			return ErrCommitmentNumberInvalid.Wrapf(
				"commitment %d has archived_at_height but is not archived",
				commitment.Number,
			)
		}
		if commitment.ArchivedAtHeight != 0 &&
			commitment.ArchivedAtHeight < commitment.IntroducedAtHeight {
			return fmt.Errorf(
				"commitment %d archived_at_height precedes introduced_at_height",
				commitment.Number,
			)
		}
	}
	return nil
}
