package protocol

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/bits"
	"reflect"
)

func ParseSettlementBatch(input []byte) (SettlementBatch, []byte, error) {
	canonical, err := CanonicalJSON(input)
	if err != nil {
		return SettlementBatch{}, nil, err
	}
	if !bytes.Equal(input, canonical) {
		return SettlementBatch{}, nil, fmt.Errorf("batch wire bytes are not exact canonical JSON")
	}
	if err := validateSettlementBatchObjectShape(canonical); err != nil {
		return SettlementBatch{}, nil, fmt.Errorf("batch shape: %w", err)
	}
	var batch SettlementBatch
	if err := strictUnmarshal(canonical, &batch); err != nil {
		return SettlementBatch{}, nil, fmt.Errorf("batch shape: %w", err)
	}
	if err := validateBatchRange(batch.FirstSequence, batch.LastSequence, batch.ReceiptCount, batch.DeclaredGaps); err != nil {
		return SettlementBatch{}, nil, err
	}
	count, _ := parseDecimal("receipt_count", batch.ReceiptCount, false)
	if uint64(len(batch.Leaves)) != count {
		return SettlementBatch{}, nil, fmt.Errorf("leaf count %d does not equal receipt_count %d", len(batch.Leaves), count)
	}
	first, _ := parseDecimal("first_sequence", batch.FirstSequence, false)
	last, _ := parseDecimal("last_sequence", batch.LastSequence, false)
	expected := first
	gapIndex := 0
	seenReceipts := make(map[string]struct{}, len(batch.Leaves))
	for i, leaf := range batch.Leaves {
		for gapIndex < len(batch.DeclaredGaps) {
			gapFirst, _ := parseDecimal("gap.first", batch.DeclaredGaps[gapIndex].First, false)
			gapLast, _ := parseDecimal("gap.last", batch.DeclaredGaps[gapIndex].Last, false)
			if expected < gapFirst {
				break
			}
			if expected <= gapLast {
				expected = gapLast + 1
				gapIndex++
				continue
			}
			gapIndex++
		}
		sequence, err := parseDecimal(fmt.Sprintf("leaves[%d].sequence", i), leaf.Sequence, false)
		if err != nil {
			return SettlementBatch{}, nil, err
		}
		if sequence != expected {
			return SettlementBatch{}, nil, fmt.Errorf("leaves[%d].sequence is %d, expected %d", i, sequence, expected)
		}
		if _, err := parseDigest(leaf.ReceiptDigest); err != nil {
			return SettlementBatch{}, nil, fmt.Errorf("leaves[%d].receipt_digest: %w", i, err)
		}
		if _, exists := seenReceipts[leaf.ReceiptDigest]; exists {
			return SettlementBatch{}, nil, fmt.Errorf("leaves[%d].receipt_digest duplicates an earlier receipt", i)
		}
		seenReceipts[leaf.ReceiptDigest] = struct{}{}
		if expected == last {
			expected = 0
		} else {
			expected++
		}
	}
	return batch, canonical, nil
}

func validateSettlementBatchObjectShape(canonical []byte) error {
	if err := requireExactStructKeys(canonical, reflect.TypeOf(SettlementBatch{})); err != nil {
		return err
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(canonical, &top); err != nil {
		return err
	}
	var gaps []json.RawMessage
	if string(top["declared_gaps"]) == "null" {
		return fmt.Errorf("declared_gaps must be an array")
	}
	if err := json.Unmarshal(top["declared_gaps"], &gaps); err != nil {
		return fmt.Errorf("declared_gaps: %w", err)
	}
	for i, raw := range gaps {
		if err := requireExactStructKeys(raw, reflect.TypeOf(Gap{})); err != nil {
			return fmt.Errorf("declared_gaps[%d]: %w", i, err)
		}
	}
	var leaves []json.RawMessage
	if string(top["leaves"]) == "null" {
		return fmt.Errorf("leaves must be an array")
	}
	if err := json.Unmarshal(top["leaves"], &leaves); err != nil {
		return fmt.Errorf("leaves: %w", err)
	}
	for i, raw := range leaves {
		if err := requireExactStructKeys(raw, reflect.TypeOf(SettlementLeaf{})); err != nil {
			return fmt.Errorf("leaves[%d]: %w", i, err)
		}
	}
	return nil
}

func SettlementMerkleRoot(leaves []SettlementLeaf) (string, error) {
	hashes := make([][32]byte, len(leaves))
	for i, leaf := range leaves {
		if _, err := parseDecimal(fmt.Sprintf("leaves[%d].sequence", i), leaf.Sequence, false); err != nil {
			return "", err
		}
		if _, err := parseDigest(leaf.ReceiptDigest); err != nil {
			return "", fmt.Errorf("leaves[%d].receipt_digest: %w", i, err)
		}
		encoded, err := json.Marshal(leaf)
		if err != nil {
			return "", err
		}
		canonical, err := CanonicalJSON(encoded)
		if err != nil {
			return "", err
		}
		leafInput := make([]byte, 0, 1+len(Protocol)+1+len("settlement-leaf")+1+len(canonical))
		leafInput = append(leafInput, 0x00)
		leafInput = append(leafInput, Protocol...)
		leafInput = append(leafInput, 0x00)
		leafInput = append(leafInput, "settlement-leaf"...)
		leafInput = append(leafInput, 0x00)
		leafInput = append(leafInput, canonical...)
		hashes[i] = sha256.Sum256(leafInput)
	}
	root := rfc6962TreeHash(hashes)
	return digestString(root), nil
}

func VerifySettlementBatch(input []byte) (SettlementBatch, string, error) {
	batch, _, err := ParseSettlementBatch(input)
	if err != nil {
		return SettlementBatch{}, "", err
	}
	root, err := SettlementMerkleRoot(batch.Leaves)
	return batch, root, err
}

func rfc6962TreeHash(leaves [][32]byte) [32]byte {
	if len(leaves) == 0 {
		return sha256.Sum256(nil)
	}
	if len(leaves) == 1 {
		return leaves[0]
	}
	k := 1 << (bits.Len(uint(len(leaves)-1)) - 1)
	left := rfc6962TreeHash(leaves[:k])
	right := rfc6962TreeHash(leaves[k:])
	input := make([]byte, 1, 65)
	input[0] = 0x01
	input = append(input, left[:]...)
	input = append(input, right[:]...)
	return sha256.Sum256(input)
}
