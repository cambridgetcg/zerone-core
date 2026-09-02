package types

import (
	"bytes"
	"testing"
)

func TestSourceRefKeyCompatibility(t *testing.T) {
	adapterID := "a\x00b"
	sourceID := "source"
	want := append([]byte{0x8E, 0x00, 0x00, 0x00, 0x03}, []byte(adapterID+sourceID)...)
	if got := SourceRefKey(adapterID, sourceID); !bytes.Equal(got, want) {
		t.Fatalf("SourceRefKey() = %x, want %x", got, want)
	}

	if bytes.Equal(SourceRefKey("a", "bc"), SourceRefKey("ab", "c")) {
		t.Fatal("length-prefixed adapter IDs must keep concatenation-equivalent pairs distinct")
	}
	if !bytes.Equal(DedupeArmedKey, []byte{0x8F}) {
		t.Fatalf("DedupeArmedKey = %x, want 8f", DedupeArmedKey)
	}
}

func TestErrDedupeNotArmedCompatibility(t *testing.T) {
	if got := ErrDedupeNotArmed.Codespace(); got != ModuleName {
		t.Fatalf("codespace = %q, want %q", got, ModuleName)
	}
	if got := ErrDedupeNotArmed.ABCICode(); got != 25 {
		t.Fatalf("ABCI code = %d, want 25", got)
	}
}
