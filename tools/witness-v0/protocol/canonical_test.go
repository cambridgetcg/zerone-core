package protocol

import (
	"bytes"
	"strings"
	"testing"
)

func TestCanonicalJSONKnownAnswers(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"object-order", ` { "z" : 0, "a" : [true, null, "x"] } `, `{"a":[true,null,"x"],"z":0}`},
		{"short-escapes", `{"x":"\u0008\u000c\u000a\u000d\u0009"}`, `{"x":"\b\f\n\r\t"}`},
		{"unicode-utf8-key-order", `{"𐀀":2,"�":1}`, `{"�":1,"𐀀":2}`},
		{"valid-surrogate-pair", `{"x":"\ud800\udc00"}`, `{"x":"𐀀"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CanonicalJSON([]byte(tt.input))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCanonicalJSONRejectsHostileInputs(t *testing.T) {
	tests := map[string]string{
		"duplicate":               `{"a":1,"a":2}`,
		"escaped-duplicate":       `{"a":1,"\u0061":2}`,
		"raw-nul":                 "{\"x\":\"\x00\"}",
		"escaped-nul":             `{"x":"\u0000"}`,
		"lone-high-surrogate":     `{"x":"\ud800"}`,
		"unpaired-high-surrogate": `{"x":"\ud800x"}`,
		"lone-low-surrogate":      `{"x":"\udc00"}`,
		"negative":                `{"x":-1}`,
		"fraction":                `{"x":1.0}`,
		"exponent":                `{"x":1e2}`,
		"unsafe-integer":          `{"x":9007199254740992}`,
		"uint64-overflow-number":  `{"x":18446744073709551616}`,
		"trailing":                `{} {}`,
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := CanonicalJSON([]byte(input)); err == nil {
				t.Fatal("expected rejection")
			}
		})
	}
}

func TestCanonicalJSONBounds(t *testing.T) {
	depthOK := strings.Repeat("[", MaxJSONDepth-1) + "0" + strings.Repeat("]", MaxJSONDepth-1)
	if _, err := CanonicalJSON([]byte(depthOK)); err != nil {
		t.Fatalf("maximum depth: %v", err)
	}
	tooDeep := "[" + depthOK + "]"
	if _, err := CanonicalJSON([]byte(tooDeep)); err == nil {
		t.Fatal("expected depth rejection")
	}
	oversize := bytes.Repeat([]byte{' '}, MaxDocumentBytes+1)
	if _, err := CanonicalJSON(oversize); err == nil {
		t.Fatal("expected size rejection")
	}
	longString := `{"x":"` + strings.Repeat("a", MaxStringBytes+1) + `"}`
	if _, err := CanonicalJSON([]byte(longString)); err == nil {
		t.Fatal("expected string bound rejection")
	}
}

func TestCanonicalUint64Boundary(t *testing.T) {
	if got, err := CanonicalJSON([]byte(`{"x":9007199254740991}`)); err != nil || string(got) != `{"x":9007199254740991}` {
		t.Fatalf("maximum safe bare integer rejected: %s, %v", got, err)
	}
	if !isCanonicalUint("18446744073709551615", false) {
		t.Fatal("max uint64 rejected")
	}
	for _, invalid := range []string{"18446744073709551616", "00", "01", "-1", "+1", "1.0", "1e1", ""} {
		if isCanonicalUint(invalid, true) {
			t.Fatalf("accepted %q", invalid)
		}
	}
}
