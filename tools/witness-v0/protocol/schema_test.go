package protocol

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

func TestEmbeddedSchemasAreStrictCanonicalJSON(t *testing.T) {
	entries, err := schemaFS.ReadDir("schemas")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 12 {
		t.Fatalf("got %d schema files, want 12", len(entries))
	}
	for _, entry := range entries {
		contents, err := schemaFS.ReadFile("schemas/" + entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := CanonicalJSON(contents); err != nil {
			t.Fatalf("%s: %v", entry.Name(), err)
		}
	}
}

func TestSchemaDecimalStringsEnforceUint64LexicalMaximum(t *testing.T) {
	entries, _ := schemaFS.ReadDir("schemas")
	for _, entry := range entries {
		contents, _ := schemaFS.ReadFile("schemas/" + entry.Name())
		var schema map[string]any
		if err := json.Unmarshal(contents, &schema); err != nil {
			t.Fatal(err)
		}
		defs, _ := schema["$defs"].(map[string]any)
		for _, name := range []string{"positive", "uint"} {
			definition, ok := defs[name].(map[string]any)
			if !ok {
				continue
			}
			base := regexp.MustCompile(definition["pattern"].(string))
			notObject, ok := definition["not"].(map[string]any)
			if !ok {
				t.Fatalf("%s %s lacks overflow exclusion", entry.Name(), name)
			}
			overflow := regexp.MustCompile(notObject["pattern"].(string))
			accepts := func(value string) bool { return base.MatchString(value) && !overflow.MatchString(value) }
			if !accepts("18446744073709551615") {
				t.Fatalf("%s %s rejects uint64 max", entry.Name(), name)
			}
			if accepts("18446744073709551616") || accepts("99999999999999999999") {
				t.Fatalf("%s %s accepts uint64 overflow", entry.Name(), name)
			}
			if name == "positive" && accepts("0") {
				t.Fatalf("%s positive accepts zero", entry.Name())
			}
			if name == "uint" && !accepts("0") {
				t.Fatalf("%s uint rejects zero", entry.Name())
			}
		}
	}
}

func TestProtocolSchemasRejectDoubleSlashLexically(t *testing.T) {
	for _, name := range []string{"kingdom-release-root.schema.json", "agenttool-settlement-root.schema.json", "wake-public-checkpoint.schema.json"} {
		contents, _ := schemaFS.ReadFile("schemas/" + name)
		if !strings.Contains(string(contents), `^(?!.*//)`) {
			t.Fatalf("%s does not exclude //", name)
		}
	}
	for _, value := range []string{"agenttool.receipt//1", "agenttool.receipt/1//x"} {
		if err := validateProtocolID("test", value); err == nil {
			t.Fatalf("runtime accepted %q", value)
		}
	}
}

func TestCollaborationSchemaDeclaresNormativeCrossFieldEquality(t *testing.T) {
	contents, err := schemaFS.ReadFile("schemas/collaboration-checkpoint.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contents), "event_head_sequence to equal event_count") {
		t.Fatal("collaboration schema does not disclose the normative full-prefix equality enforced by the verifier")
	}
}
