package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func freshStatus(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"result":{"node_info":{"network":"zerone-1"},"sync_info":{"latest_block_height":"1000","latest_block_time":%q,"catching_up":false}}}`, time.Now().UTC().Format(time.RFC3339Nano))
}
func publicationFixture(t *testing.T, origin string) ([]byte, publicationMetadata) {
	t.Helper()
	p := projection{geometrySchema, projectionSource{"zerone-1", "1000", "1000", false, queryPath, false, false, "NOT_CLAIMED", 1, 1, false}, []projectedFact{{"fact-a", "<A bounded record>", "general", "", "FACT_STATUS_VERIFIED", "CLAIM_TYPE_ASSERTION", 0, "900", "900", 0, 0, 0, ""}}, []projectedRelation{{"fact-a", "external", "RELATION_TYPE_CITES", "INFERENCE_TYPE_CITATION", 0, "900", ""}}}
	raw, err := compactJSON(p)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	return raw, publicationMetadata{metadataSchema, "zerone-1", "1000", "1000", now, now, origin, origin, strings.Repeat("a", 40), hashBytes(raw), provenanceScope}
}
func TestExactCaseSensitiveIdentity(t *testing.T) {
	for _, body := range []string{
		`{"fact":{"id":"wrong-fact","ID":"fact-a"}}`,
		`{"fact":{"id":"wrong-fact"},"FACT":{"id":"fact-a"}}`,
		`{"FACT":{"id":"fact-a"}}`, `{"fact":{"ID":"fact-a"}}`,
		`{"fact":{"id":"fact-a","ID":"fact-a"}}`, `{"fact":{"id":"fact-a"},"Fact":{"id":"fact-a"}}`,
		`{"fact":{"id":"fact-a","id":"fact-a"}}`,
	} {
		t.Run(body, func(t *testing.T) {
			f := newTestFacade(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { pointResponse(w, body) }), admitted(t))
			if w := request(f, "/facts/fact-a"); w.Code != 502 {
				t.Fatalf("case alias accepted: %d %s", w.Code, w.Body)
			}
		})
	}
	if err := validatePointJSON([]byte(`{"fact":{"id":"fact-a","content":"normal"}}`), "fact-a"); err != nil {
		t.Fatal(err)
	}
}
func TestTypeScriptPublicationCrossLanguageBytes(t *testing.T) {
	body, err := os.ReadFile("testdata/typescript-publication.json")
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := validateSnapshot(body, "zerone-1")
	if err != nil {
		t.Fatal("TypeScript publication rejected", err)
	}
	observed, err := time.Parse(time.RFC3339Nano, snapshot.ObservedAt)
	if err != nil {
		t.Fatal(err)
	}
	rebuilt, err := buildSnapshot(snapshot.Payload, snapshot.Publication, observed)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(body, rebuilt) {
		t.Fatal("Go/TypeScript publication bytes differ")
	}
	if hashBytes(body) != "bbd7749824ac776397ed1ca0fbc5c12287fef25069da14b2c346dec05b08bb1c" {
		t.Fatal("deterministic publication vector changed")
	}
}

func TestImmutablePublishLoadAndNoOverwrite(t *testing.T) {
	raw, m := publicationFixture(t, "http://127.0.0.1:1")
	rootPath := filepath.Join(t.TempDir(), "snapshots")
	id, head, err := publish(rootPath, raw, m, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(filepath.Join(rootPath, id+".json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := publish(rootPath, raw, m, time.Now()); err == nil {
		t.Fatal("overwrote publication")
	}
	after, _ := os.ReadFile(filepath.Join(rootPath, id+".json"))
	if !bytes.Equal(original, after) {
		t.Fatal("changed immutable bytes")
	}
	f, err := newFacade(m.RESTOrigin, "zerone-1", admitted(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	root, err := openStore(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := loadPublications(f, root, head); err != nil {
		t.Fatal(err)
	}
	if w := request(f, "/head"); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if w := request(f, "/snapshots/"+id); w.Code != 200 || !bytes.Equal(w.Body.Bytes(), original) {
		t.Fatal("snapshot serving mismatch")
	}
	for _, mutation := range []string{"wrong digest", "wrong chain", "wrong root", "case alias", "duplicate", "over-depth", "over-bytes"} {
		t.Run(mutation, func(t *testing.T) {
			changed := bytes.Clone(original)
			switch mutation {
			case "wrong digest":
				changed = bytes.Replace(changed, []byte(`"payloadDigest":"`), []byte(`"payloadDigest":"0`), 1)
			case "wrong chain":
				changed = bytes.Replace(changed, []byte(`"chainId":"zerone-1"`), []byte(`"chainId":"other"`), 1)
			case "wrong root":
				changed = bytes.Replace(changed, []byte(`"topologyRoot":"`), []byte(`"topologyRoot":"0`), 1)
			case "case alias":
				changed = bytes.Replace(changed, []byte(`"chainId"`), []byte(`"ChainId"`), 1)
			case "duplicate":
				changed = append([]byte(`{"schema":"bad",`), changed[1:]...)
			case "over-depth":
				changed = []byte(strings.Repeat("[", 65) + strings.Repeat("]", 65))
			case "over-bytes":
				changed = []byte(strings.Repeat("x", snapshotBytes+1))
			}
			if _, err := validateSnapshot(changed, "zerone-1"); err == nil {
				t.Fatal("invalid publication accepted")
			}
		})
	}
}
func TestPublisherRejectsInvalidProjectionMetadataAndStorage(t *testing.T) {
	raw, m := publicationFixture(t, "http://127.0.0.1:1")
	for _, name := range []string{"stale", "future", "wrong height", "wrong digest", "false proof", "caller url", "source identity", "syncing", "edge conflict", "node cap", "byte cap", "null field", "duplicate field", "unknown field"} {
		t.Run(name, func(t *testing.T) {
			body := bytes.Clone(raw)
			meta := m
			switch name {
			case "stale":
				meta.ObservedAt = time.Now().Add(-time.Minute).UTC().Format(time.RFC3339Nano)
			case "future":
				meta.ObservedAt = time.Now().Add(time.Minute).UTC().Format(time.RFC3339Nano)
			case "wrong height":
				meta.BlockHeight = "999"
			case "wrong digest":
				meta.ProjectionSHA256 = strings.Repeat("0", 64)
			case "false proof":
				meta.Provenance = "verified chain proof"
			case "caller url":
				meta.RESTOrigin = "http://127.0.0.1:1/?execute=bad"
			case "source identity":
				meta.SourceCommit = "moving-main"
			case "syncing":
				body = bytes.Replace(body, []byte(`"catchingUp":false`), []byte(`"catchingUp":true`), 1)
			case "edge conflict":
				var p projection
				_ = json.Unmarshal(body, &p)
				p.Relations = append(p.Relations, p.Relations[0])
				body, _ = compactJSON(p)
			case "node cap":
				var p projection
				_ = json.Unmarshal(body, &p)
				for len(p.Facts) < 129 {
					f := p.Facts[0]
					f.ID = fmt.Sprintf("fact-%04d", len(p.Facts))
					p.Facts = append(p.Facts, f)
				}
				body, _ = compactJSON(p)
			case "byte cap":
				body = []byte(strings.Repeat("x", snapshotBytes+1))
			case "null field":
				body = bytes.Replace(body, []byte(`"energy":0`), []byte(`"energy":null`), 1)
			case "duplicate field":
				body = bytes.Replace(body, []byte(`"energy":0`), []byte(`"energy":0,"energy":0`), 1)
			case "unknown field":
				body = bytes.Replace(body, []byte(`"energy":0`), []byte(`"energy":0,"legacyEdges":[]`), 1)
			}
			if name != "wrong digest" {
				meta.ProjectionSHA256 = hashBytes(body)
			}
			if _, err := buildSnapshot(body, meta, time.Now()); err == nil {
				t.Fatal("invalid publisher input accepted")
			}
		})
	}
	rootPath := t.TempDir()
	root, err := openStore(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := os.Symlink("/dev/null", filepath.Join(rootPath, strings.Repeat("a", 64)+".json")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := publish(rootPath, raw, m, time.Now()); err == nil {
		t.Fatal("symlink accepted")
	}
}
func TestPublicationCapacityAndInterruptedWriteFailClosed(t *testing.T) {
	raw, m := publicationFixture(t, "http://127.0.0.1:1")
	rootPath := t.TempDir()
	for i := 0; i < maxPublications; i++ {
		m.SourceCommit = fmt.Sprintf("%040x", i)
		if _, _, err := publish(rootPath, raw, m, time.Now()); err != nil {
			t.Fatal(i, err)
		}
	}
	m.SourceCommit = strings.Repeat("f", 40)
	if _, _, err := publish(rootPath, raw, m, time.Now()); err == nil {
		t.Fatal("unbounded store")
	}
	rootPath = t.TempDir()
	if err := os.WriteFile(filepath.Join(rootPath, ".pending"), []byte("interrupted"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := publish(rootPath, raw, m, time.Now()); err == nil {
		t.Fatal("ignored interrupted publication")
	}
}
