package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type admissionFunc func(context.Context, *http.Request, int, int) (func(), error)

func (a admissionFunc) Acquire(c context.Context, r *http.Request, rate, concurrency int) (func(), error) {
	return a(c, r, rate, concurrency)
}
func admitted(t *testing.T) Admission {
	return admissionFunc(func(ctx context.Context, r *http.Request, rate, concurrency int) (func(), error) {
		t.Helper()
		if rate != 2 || concurrency != 8 {
			t.Fatalf("wrong admission limits %d %d", rate, concurrency)
		}
		if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > pointTimeout {
			t.Fatal("missing admission deadline")
		}
		return func() {}, nil
	})
}
func newTestFacade(t *testing.T, handler http.Handler, admission Admission) *facade {
	t.Helper()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/status" {
			freshStatus(w)
			return
		}
		handler.ServeHTTP(w, r)
	}))
	t.Cleanup(upstream.Close)
	f, err := newFacade(upstream.URL, "zerone-1", admission, nil)
	if err != nil {
		t.Fatal(err)
	}
	return f
}
func request(f *facade, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	f.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
	return w
}
func pointResponse(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cosmos-Block-Height", "1000")
	w.Header().Set("X-Zerone-Expected-Chain", "zerone-1")
	fmt.Fprint(w, body)
}

func TestFacadePointAndDigest(t *testing.T) {
	f := newTestFacade(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/zerone/knowledge/v1/facts/fact-a" || r.URL.RawQuery != "" || r.Method != "GET" {
			t.Errorf("unexpected upstream request: %s", r.URL)
		}
		pointResponse(w, `{ "fact": {"id":"fact-a", "content":"<hello>"}}`)
	}), admitted(t))
	w := request(f, "/facts/fact-a")
	if w.Code != 200 {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	var value struct {
		Payload json.RawMessage `json:"payload"`
		Digest  string          `json:"payloadDigest"`
		Height  string          `json:"blockHeight"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(value.Payload)
	if value.Digest != hex.EncodeToString(hash[:]) || value.Height != "1000" {
		t.Fatal("digest/actual height mismatch")
	}
	if w.Body.Len() > pointBytes {
		t.Fatal("oversized point response")
	}
}
func TestFacadeDefaultDenyAndAdmissionFailure(t *testing.T) {
	var calls atomic.Int32
	f := newTestFacade(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1) }), nil)
	if w := request(f, "/facts/fact-a"); w.Code != 503 {
		t.Fatal(w.Code)
	}
	f.admission = admitted(t)
	for _, path := range []string{"/facts", "/facts/fact-a?track_query=true", "/facts/fact-a?", "/facts/a%2Fb", "/facts/" + strings.Repeat("x", 129), "/bundle_tok", "/status", "/snapshots/../x"} {
		if w := request(f, path); w.Code != 400 && w.Code != 404 {
			t.Fatalf("accepted %s: %d", path, w.Code)
		}
	}
	f.admission = admissionFunc(func(context.Context, *http.Request, int, int) (func(), error) { return nil, errAdmissionLimited })
	if w := request(f, "/facts/fact-a"); w.Code != 429 {
		t.Fatal(w.Code)
	}
	if calls.Load() != 0 {
		t.Fatal("refused request touched upstream")
	}
}
func TestFacadeRejectsBadBodiesAndProvenance(t *testing.T) {
	for name, body := range map[string]string{"oversize": strings.Repeat("x", pointBytes+1), "wrong id": `{"fact":{"id":"fact-b"}}`, "duplicate": `{"fact":{"id":"fact-a","id":"fact-b"}}`, "missing": `{}`, "nested": strings.Repeat("[", 65) + strings.Repeat("]", 65)} {
		t.Run(name, func(t *testing.T) {
			f := newTestFacade(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { pointResponse(w, body) }), admitted(t))
			if w := request(f, "/facts/fact-a"); w.Code != 502 {
				t.Fatal(w.Code)
			}
		})
	}
	f := newTestFacade(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"fact":{"id":"fact-a"}}`)
	}), admitted(t))
	if w := request(f, "/facts/fact-a"); w.Code != 502 {
		t.Fatal("accepted unknown provenance")
	}
}
func TestPointHeadersMandatoryAndUnambiguous(t *testing.T) {
	for _, kind := range []string{"missing chain", "wrong chain", "duplicate chain", "conflicting heights", "duplicate height", "overflow height", "zero height"} {
		t.Run(kind, func(t *testing.T) {
			f := newTestFacade(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Cosmos-Block-Height", "1000")
				w.Header().Set("X-Zerone-Expected-Chain", "zerone-1")
				switch kind {
				case "missing chain":
					w.Header().Del("X-Zerone-Expected-Chain")
				case "wrong chain":
					w.Header().Set("X-Zerone-Expected-Chain", "wrong")
				case "duplicate chain":
					w.Header().Add("X-Zerone-Expected-Chain", "zerone-1")
				case "conflicting heights":
					w.Header().Set("Grpc-Metadata-X-Cosmos-Block-Height", "999")
				case "duplicate height":
					w.Header().Add("X-Cosmos-Block-Height", "1000")
				case "overflow height":
					w.Header().Set("X-Cosmos-Block-Height", "9223372036854775808")
				case "zero height":
					w.Header().Set("X-Cosmos-Block-Height", "0")
				}
				fmt.Fprint(w, `{"fact":{"id":"fact-a"}}`)
			}), admitted(t))
			if w := request(f, "/facts/fact-a"); w.Code != 502 {
				t.Fatalf("accepted %s: %d", kind, w.Code)
			}
		})
	}
}

func TestFacadeFiveSecondBodyDeadline(t *testing.T) {
	f := newTestFacade(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}), admitted(t))
	start := time.Now()
	w := request(f, "/facts/fact-a")
	if w.Code != 504 || time.Since(start) > 6*time.Second {
		t.Fatalf("unbounded body wait: %d %v", w.Code, time.Since(start))
	}
}
func TestFacadeImmutableSnapshot(t *testing.T) {
	f := newTestFacade(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("snapshot fetched upstream") }), admitted(t))
	raw, metadata := publicationFixture(t, f.upstream.String())
	body, err := buildSnapshot(raw, metadata, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(body)
	id := hex.EncodeToString(hash[:])
	if err := f.addSnapshot(id, body); err != nil {
		t.Fatal(err)
	}
	body[0] = 'x'
	w := request(f, "/snapshots/"+id)
	if w.Code != 200 || !strings.Contains(w.Header().Get("Cache-Control"), "immutable") {
		t.Fatal("missing immutable response")
	}
	if err := f.addSnapshot(id, body); err == nil {
		t.Fatal("accepted changed bytes")
	}
	if w := request(f, "/snapshots/"+strings.Repeat("0", 64)); w.Code != 404 {
		t.Fatal(w.Code)
	}
}

// This test supplies one centralized FAKE admission authority for all requests.
// It verifies handler lease lifetime and limit contract under concurrent load;
// it does not establish any deployed fleet-wide enforcement or client identity.
func TestFacadeAdmissionLeaseLoad(t *testing.T) {
	var mu sync.Mutex
	active, maxActive := 0, 0
	perClient := map[string]int{}
	var released atomic.Int32
	control := admissionFunc(func(ctx context.Context, r *http.Request, rate, concurrency int) (func(), error) {
		mu.Lock()
		defer mu.Unlock()
		client := r.RemoteAddr // local test transport identity, never user header
		if perClient[client] >= rate || active >= concurrency {
			return nil, errAdmissionLimited
		}
		perClient[client]++
		active++
		if active > maxActive {
			maxActive = active
		}
		return func() { mu.Lock(); active--; mu.Unlock(); released.Add(1) }, nil
	})
	var entered atomic.Int32
	done := make(chan struct{})
	f := newTestFacade(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entered.Add(1)
		<-done
		pointResponse(w, `{"fact":{"id":"fact-a"}}`)
	}), control)
	results := make(chan int, 8)
	for i := 0; i < 8; i++ {
		go func(i int) {
			r := httptest.NewRequest("GET", "/facts/fact-a", nil)
			r.RemoteAddr = fmt.Sprintf("client-%d", i)
			w := httptest.NewRecorder()
			f.ServeHTTP(w, r)
			results <- w.Code
		}(i)
	}
	deadline := time.Now().Add(time.Second)
	for entered.Load() < 8 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if entered.Load() != 8 {
		close(done)
		t.Fatal("did not reach eight upstreams")
	}
	if w := request(f, "/facts/fact-a"); w.Code != 429 {
		close(done)
		t.Fatal("ninth read admitted")
	}
	close(done)
	for i := 0; i < 8; i++ {
		if code := <-results; code != 200 {
			t.Fatal(code)
		}
	}
	if released.Load() != 8 || maxActive != 8 {
		t.Fatal("concurrency lease leak")
	}
	// After slots free, a single client still receives only two admissions.
	for i := 0; i < 3; i++ {
		w := request(f, "/facts/fact-a")
		want := 200
		if i == 2 {
			want = 429
		}
		if w.Code != want {
			t.Fatalf("request %d: %d", i, w.Code)
		}
	}
}
