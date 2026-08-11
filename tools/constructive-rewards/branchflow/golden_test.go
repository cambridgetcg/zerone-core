package branchflow

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestReferenceGoldenVector(t *testing.T) {
	requestBytes, err := os.ReadFile("testdata/reference_request.json")
	if err != nil {
		t.Fatalf("read request fixture: %v", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(requestBytes))
	decoder.DisallowUnknownFields()
	var request Request
	if err := decoder.Decode(&request); err != nil {
		t.Fatalf("decode request fixture: %v", err)
	}
	result, err := Allocate(request)
	if err != nil {
		t.Fatalf("Allocate golden request: %v", err)
	}
	got, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	want, err := os.ReadFile("testdata/reference_result.json")
	if err != nil {
		t.Fatalf("read result fixture: %v", err)
	}
	want = bytes.TrimSpace(want)
	if !bytes.Equal(got, want) {
		t.Fatalf("golden drift:\nwant %s\n got %s", want, got)
	}

	replayed, err := Allocate(request)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	replayedJSON, err := json.Marshal(replayed)
	if err != nil {
		t.Fatalf("marshal replay: %v", err)
	}
	if !bytes.Equal(replayedJSON, got) {
		t.Fatalf("replay was not byte-identical:\nfirst %s\nagain %s", got, replayedJSON)
	}
}
