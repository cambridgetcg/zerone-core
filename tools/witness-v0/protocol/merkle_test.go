package protocol

import (
	"encoding/json"
	"strings"
	"testing"
)

func knownBatch() SettlementBatch {
	return SettlementBatch{
		FirstSequence: "1", LastSequence: "4", ReceiptCount: "3",
		DeclaredGaps: []Gap{{First: "2", Last: "2"}},
		Leaves: []SettlementLeaf{
			{Sequence: "1", ReceiptDigest: "sha256:" + strings.Repeat("1", 64)},
			{Sequence: "3", ReceiptDigest: "sha256:" + strings.Repeat("2", 64)},
			{Sequence: "4", ReceiptDigest: "sha256:" + strings.Repeat("3", 64)},
		},
	}
}

func TestRFC6962SettlementMerkleKnownAnswers(t *testing.T) {
	empty, err := SettlementMerkleRoot(nil)
	if err != nil {
		t.Fatal(err)
	}
	if empty != "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Fatalf("empty root: %s", empty)
	}
	batch := knownBatch()
	root, err := SettlementMerkleRoot(batch.Leaves)
	if err != nil {
		t.Fatal(err)
	}
	if root != "sha256:8f0995e7dcee1737603969d8e03ecccc1d7dbe5777c5bc4dc1d42d4b20025b52" {
		t.Fatalf("three-leaf root: %s", root)
	}
	encoded, _ := jsonBytes(batch)
	parsed, verifiedRoot, err := VerifySettlementBatch(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if verifiedRoot != root || len(parsed.Leaves) != 3 {
		t.Fatal("batch verification changed root or leaves")
	}
}

func TestSettlementBatchRejectsDuplicateReceiptDigest(t *testing.T) {
	batch := knownBatch()
	batch.Leaves[1].ReceiptDigest = batch.Leaves[0].ReceiptDigest
	encoded, _ := jsonBytes(batch)
	if _, _, err := VerifySettlementBatch(encoded); err == nil || !strings.Contains(err.Error(), "duplicates") {
		t.Fatalf("expected duplicate receipt rejection, got %v", err)
	}
}

func TestSettlementBatchCoverageAndGapRules(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*SettlementBatch)
	}{
		{"wrong-count", func(b *SettlementBatch) { b.ReceiptCount = "2" }},
		{"wrong-leaf-sequence", func(b *SettlementBatch) { b.Leaves[1].Sequence = "2" }},
		{"unordered-gaps", func(b *SettlementBatch) {
			b.FirstSequence = "1"
			b.LastSequence = "6"
			b.ReceiptCount = "3"
			b.DeclaredGaps = []Gap{{First: "5", Last: "5"}, {First: "2", Last: "3"}}
		}},
		{"adjacent-gaps", func(b *SettlementBatch) {
			b.FirstSequence = "1"
			b.LastSequence = "6"
			b.ReceiptCount = "3"
			b.DeclaredGaps = []Gap{{First: "2", Last: "2"}, {First: "3", Last: "3"}}
		}},
		{"gap-outside", func(b *SettlementBatch) { b.DeclaredGaps[0].First = "5"; b.DeclaredGaps[0].Last = "5" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batch := knownBatch()
			tt.mutate(&batch)
			encoded, _ := jsonBytes(batch)
			if _, _, err := VerifySettlementBatch(encoded); err == nil {
				t.Fatal("expected rejection")
			}
		})
	}
}

func TestSettlementBatchRequiresExactObjectKeys(t *testing.T) {
	batch := knownBatch()
	encoded, _ := jsonBytes(batch)
	for _, path := range [][]string{{"declared_gaps"}, {"leaves"}, {"declared_gaps", "first"}, {"leaves", "receipt_digest"}} {
		var top map[string]any
		if err := json.Unmarshal(encoded, &top); err != nil {
			t.Fatal(err)
		}
		if len(path) == 1 {
			delete(top, path[0])
		} else if path[0] == "declared_gaps" {
			delete(top["declared_gaps"].([]any)[0].(map[string]any), path[1])
		} else {
			delete(top["leaves"].([]any)[0].(map[string]any), path[1])
		}
		mutated, _ := jsonBytes(top)
		if _, _, err := VerifySettlementBatch(mutated); err == nil || !strings.Contains(err.Error(), "missing required") {
			t.Fatalf("path %v: expected missing rejection, got %v", path, err)
		}
	}

	for _, target := range []string{"top", "gap", "leaf"} {
		var top map[string]any
		if err := json.Unmarshal(encoded, &top); err != nil {
			t.Fatal(err)
		}
		switch target {
		case "top":
			top["unexpected"] = true
		case "gap":
			top["declared_gaps"].([]any)[0].(map[string]any)["unexpected"] = true
		case "leaf":
			top["leaves"].([]any)[0].(map[string]any)["unexpected"] = true
		}
		mutated, _ := jsonBytes(top)
		if _, _, err := VerifySettlementBatch(mutated); err == nil || !strings.Contains(err.Error(), "unknown field") {
			t.Fatalf("target %s: expected extra-key rejection, got %v", target, err)
		}
	}
}

func TestSettlementBatchRequiresExactCanonicalWireBytes(t *testing.T) {
	encoded, _ := jsonBytes(knownBatch())
	if _, _, err := VerifySettlementBatch(append(encoded, '\n')); err == nil || !strings.Contains(err.Error(), "wire bytes") {
		t.Fatalf("expected exact-wire rejection, got %v", err)
	}
}
