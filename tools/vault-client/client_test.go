package vaultclient

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// newTestKeyPair generates a fresh ed25519 key pair for testing.
func newTestKeyPair(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generating test key pair: %v", err)
	}
	return pub, priv
}

// tlsClient returns a VaultClient whose HTTP transport trusts the TLS
// certificate of the given httptest.Server.
func tlsClient(t *testing.T, ts *httptest.Server, opts ...Option) *VaultClient {
	t.Helper()
	c := NewVaultClient(ts.URL, opts...)
	// httptest.NewTLSServer provides a client pre-configured to trust its cert.
	c.httpClient = ts.Client()
	// Reapply timeout option if present (ts.Client() replaces the http.Client).
	for _, o := range opts {
		o(c)
	}
	return c
}

// ---------- Tests ----------

func TestGetPublicKey(t *testing.T) {
	pub, _ := newTestKeyPair(t)
	pubHex := hex.EncodeToString(pub)

	var hitCount int32

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/public-key" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		atomic.AddInt32(&hitCount, 1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"public_key":"%s"}`, pubHex)
	}))
	defer ts.Close()

	client := tlsClient(t, ts)

	// First call should hit the server.
	got, err := client.GetPublicKey()
	if err != nil {
		t.Fatalf("GetPublicKey() error: %v", err)
	}
	if !pub.Equal(got) {
		t.Fatalf("GetPublicKey() returned wrong key\n  got:  %x\n  want: %x", got, pub)
	}

	// Second call should return the cached key without hitting the server.
	got2, err := client.GetPublicKey()
	if err != nil {
		t.Fatalf("GetPublicKey() cached call error: %v", err)
	}
	if !pub.Equal(got2) {
		t.Fatal("GetPublicKey() cached call returned different key")
	}

	if n := atomic.LoadInt32(&hitCount); n != 1 {
		t.Fatalf("expected server to be hit exactly once, got %d", n)
	}
}

func TestRequestSignature(t *testing.T) {
	_, priv := newTestKeyPair(t)

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sign" || r.Method != http.MethodPost {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read error", http.StatusInternalServerError)
			return
		}

		var req struct {
			Payload string `json:"payload"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}

		// We just sign a deterministic message so we can verify the hex decode path.
		sig := ed25519.Sign(priv, []byte("test-payload"))

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"signature":"%s"}`, hex.EncodeToString(sig))
	}))
	defer ts.Close()

	client := tlsClient(t, ts)

	sig, err := client.RequestSignature([]byte("test-payload"))
	if err != nil {
		t.Fatalf("RequestSignature() error: %v", err)
	}

	if len(sig) != ed25519.SignatureSize {
		t.Fatalf("signature has wrong length %d, want %d", len(sig), ed25519.SignatureSize)
	}
}

func TestVerifyVaultIdentity(t *testing.T) {
	pub, priv := newTestKeyPair(t)
	pubHex := hex.EncodeToString(pub)

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/public-key":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"public_key":"%s"}`, pubHex)

		case "/v1/challenge":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "read error", http.StatusInternalServerError)
				return
			}

			var req struct {
				Nonce string `json:"nonce"`
			}
			if err := json.Unmarshal(body, &req); err != nil {
				http.Error(w, "bad json", http.StatusBadRequest)
				return
			}

			nonce, err := hex.DecodeString(req.Nonce)
			if err != nil {
				http.Error(w, "bad hex", http.StatusBadRequest)
				return
			}

			sig := ed25519.Sign(priv, nonce)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"signature":"%s"}`, hex.EncodeToString(sig))

		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer ts.Close()

	client := tlsClient(t, ts)

	if err := client.VerifyVaultIdentity(); err != nil {
		t.Fatalf("VerifyVaultIdentity() error: %v", err)
	}
}

func TestRetryOnNetworkFailure(t *testing.T) {
	_, priv := newTestKeyPair(t)

	var attempt int32

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sign" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		n := atomic.AddInt32(&attempt, 1)
		if n <= 2 {
			// First two attempts return 500.
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		// Third attempt succeeds.
		sig := ed25519.Sign(priv, []byte("retry-payload"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"signature":"%s"}`, hex.EncodeToString(sig))
	}))
	defer ts.Close()

	client := tlsClient(t, ts,
		WithMaxRetries(3),
		WithRetryDelay(10*time.Millisecond), // fast retries for tests
	)

	sig, err := client.RequestSignature([]byte("retry-payload"))
	if err != nil {
		t.Fatalf("RequestSignature() with retries error: %v", err)
	}

	if len(sig) != ed25519.SignatureSize {
		t.Fatalf("signature has wrong length %d after retry", len(sig))
	}

	if n := atomic.LoadInt32(&attempt); n != 3 {
		t.Fatalf("expected 3 attempts, got %d", n)
	}
}

func TestTimeout(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sleep longer than the client timeout.
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := tlsClient(t, ts,
		WithTimeout(100*time.Millisecond),
		WithMaxRetries(0), // no retries so the test finishes quickly
	)

	_, err := client.RequestSignature([]byte("timeout-payload"))
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}
