package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// pathEscape escapes a manifest or item id for use in URL paths. The
// raw IDs may contain '#' (e.g. "love-corpus#1") which would otherwise
// be interpreted as a URL fragment.
func pathEscape(id string) string { return url.PathEscape(id) }

// helper: write a private key PEM file, return its path and matching pubkey.
func mustWriteKey(t *testing.T, dir string) (string, ed25519.PublicKey) {
	t.Helper()
	keyPath := filepath.Join(dir, "operator.pem")
	pub, err := GenerateKeyPair(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	return keyPath, pub
}

// helper: build a server with one manifest under "manifest1/".
func newTestServer(t *testing.T, mode AuthMode, allowedKeys []string) (*Server, ed25519.PublicKey, string) {
	t.Helper()
	dir := t.TempDir()
	itemsRoot := filepath.Join(dir, "items")
	if err := os.MkdirAll(filepath.Join(itemsRoot, "v1"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(itemsRoot, "v1", "a.txt"), []byte("alpha"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(itemsRoot, "v1", "b.txt"), []byte("beta"), 0o644); err != nil {
		t.Fatal(err)
	}
	keyPath, pub := mustWriteKey(t, dir)
	cfg := &Config{
		ListenAddress:     ":0",
		PrivateKeyPath:    keyPath,
		VaultID:           "love-corpus",
		ItemsRoot:         itemsRoot,
		AuthMode:          mode,
		AllowedClientKeys: allowedKeys,
		Manifests: []ManifestSpec{
			{ID: "love-corpus#1", Version: "1.0", ItemRoot: "v1"},
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	srv, err := NewServer(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = srv.Close() })
	return srv, pub, "love-corpus#1"
}

func TestServer_PublicMode_ManifestSignedAndVerifiable(t *testing.T) {
	srv, pub, manifestID := newTestServer(t, AuthModePublic, nil)
	ts := httptest.NewServer(srv)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/manifest/" + pathEscape(manifestID))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var sm SignedManifest
	if err := json.NewDecoder(resp.Body).Decode(&sm); err != nil {
		t.Fatal(err)
	}
	if sm.ManifestID != manifestID {
		t.Fatalf("manifest id mismatch")
	}
	if err := VerifyManifest(pub, sm); err != nil {
		t.Fatalf("manifest signature: %v", err)
	}
}

func TestServer_HealthAndPubkey_Unauthenticated(t *testing.T) {
	srv, _, _ := newTestServer(t, AuthModePublic, nil)
	ts := httptest.NewServer(srv)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz status=%d", resp.StatusCode)
	}

	resp, err = http.Get(ts.URL + "/pubkey")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("pubkey status=%d", resp.StatusCode)
	}
	var pk map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&pk); err != nil {
		t.Fatal(err)
	}
	if pk["vault_id"] != "love-corpus" {
		t.Fatalf("unexpected vault_id: %v", pk)
	}
	if !strings.HasPrefix(pk["pubkey"], "ed25519:") {
		t.Fatalf("expected ed25519:hex pubkey, got %v", pk)
	}
}

func TestServer_Item_Roundtrip(t *testing.T) {
	srv, _, manifestID := newTestServer(t, AuthModePublic, nil)
	ts := httptest.NewServer(srv)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/item/" + pathEscape(manifestID) + "/a.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("item status=%d", resp.StatusCode)
	}
	bz, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(bz) != "alpha" {
		t.Fatalf("item bytes mismatch: %q", bz)
	}
}

func TestServer_PathTraversal_Blocked(t *testing.T) {
	srv, _, manifestID := newTestServer(t, AuthModePublic, nil)
	ts := httptest.NewServer(srv)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/item/" + pathEscape(manifestID) + "/../../../etc/passwd")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Fatalf("expected non-200 for traversal attempt, got %d", resp.StatusCode)
	}
}

func TestServer_SignedChallenge_DeniesUnauthenticated(t *testing.T) {
	// Generate a client key but DON'T put it on the allow-list.
	_, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	// Instead, build the server with a different allow-listed key.
	allowedPub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	srv, _, manifestID := newTestServer(t, AuthModeSignedChallenge,
		[]string{EncodePubkey(allowedPub)})
	ts := httptest.NewServer(srv)
	defer ts.Close()

	// No Authorization header → denied.
	resp, err := http.Get(ts.URL + "/manifest/" + pathEscape(manifestID))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestServer_SignedChallenge_AllowsAuthenticatedClient(t *testing.T) {
	clientPub, clientPriv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	srv, _, manifestID := newTestServer(t, AuthModeSignedChallenge,
		[]string{EncodePubkey(clientPub)})
	ts := httptest.NewServer(srv)
	defer ts.Close()

	// Step 1: fetch a nonce.
	nResp, err := http.Get(ts.URL + "/challenge")
	if err != nil {
		t.Fatal(err)
	}
	defer nResp.Body.Close()
	var nonceBody struct{ Nonce string }
	if err := json.NewDecoder(nResp.Body).Decode(&nonceBody); err != nil {
		t.Fatal(err)
	}
	if nonceBody.Nonce == "" {
		t.Fatal("empty nonce")
	}

	// Step 2: sign nonce + decoded path. The server reads r.URL.Path
	// (decoded form), so the client must sign the decoded form.
	decodedPath := "/manifest/" + manifestID
	payload := []byte(nonceBody.Nonce + ":" + decodedPath)
	sig := ed25519.Sign(clientPriv, payload)
	authHdr := "SignedChallenge pubkey=" + EncodePubkey(clientPub) +
		", nonce=" + nonceBody.Nonce +
		", signature=" + hex.EncodeToString(sig)

	// Step 3: request manifest with Authorization. Wire URL is encoded.
	req, _ := http.NewRequest("GET", ts.URL+"/manifest/"+pathEscape(manifestID), nil)
	req.Header.Set("Authorization", authHdr)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bz, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, bz)
	}
}

func TestServer_SignedChallenge_NonceSingleUse(t *testing.T) {
	clientPub, clientPriv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	srv, _, manifestID := newTestServer(t, AuthModeSignedChallenge,
		[]string{EncodePubkey(clientPub)})
	ts := httptest.NewServer(srv)
	defer ts.Close()

	// Issue nonce.
	nResp, _ := http.Get(ts.URL + "/challenge")
	var nonceBody struct{ Nonce string }
	_ = json.NewDecoder(nResp.Body).Decode(&nonceBody)
	nResp.Body.Close()

	decodedPath := "/manifest/" + manifestID
	wireURL := ts.URL + "/manifest/" + pathEscape(manifestID)
	payload := []byte(nonceBody.Nonce + ":" + decodedPath)
	sig := ed25519.Sign(clientPriv, payload)
	authHdr := "SignedChallenge pubkey=" + EncodePubkey(clientPub) +
		", nonce=" + nonceBody.Nonce +
		", signature=" + hex.EncodeToString(sig)

	// First use: ok.
	req, _ := http.NewRequest("GET", wireURL, nil)
	req.Header.Set("Authorization", authHdr)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("first use should succeed, got %d", resp.StatusCode)
	}

	// Second use of the same nonce: denied.
	req2, _ := http.NewRequest("GET", wireURL, nil)
	req2.Header.Set("Authorization", authHdr)
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusUnauthorized {
		t.Fatalf("second use should fail, got %d", resp2.StatusCode)
	}
}

func TestServer_ContentHash_MatchesOnDisk(t *testing.T) {
	// The hash the server prints at startup (via buildManifests) and
	// the hash a separate `hash` invocation computes over the same
	// directory must be identical — the operator's on-chain commit is
	// trustworthy only if the two paths agree.
	srv, _, manifestID := newTestServer(t, AuthModePublic, nil)
	signed := srv.manifests[manifestID]
	got, err := ContentHash(signed.ManifestBody)
	if err != nil {
		t.Fatal(err)
	}
	// Recompute by walking the directory the same way `corpus-vault hash`
	// would.
	body, err := BuildManifestFromDir(manifestID, srv.cfg.VaultID,
		filepath.Join(srv.cfg.ItemsRoot, srv.cfg.Manifests[0].ItemRoot),
		"/item/"+manifestID+"/")
	if err != nil {
		t.Fatal(err)
	}
	want, err := ContentHash(body)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("server hash %s vs walker hash %s", got, want)
	}
}

func TestSignedChallengeAuth_NonceTTL(t *testing.T) {
	clientPub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	auth, err := NewSignedChallengeAuth([]string{EncodePubkey(clientPub)}, 1*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	nonce := auth.IssueNonce()
	if nonce == "" {
		t.Fatal("empty nonce")
	}
	time.Sleep(10 * time.Millisecond)
	auth.mu.Lock()
	auth.expireLocked()
	_, exists := auth.nonces[nonce]
	auth.mu.Unlock()
	if exists {
		t.Fatalf("nonce should have expired")
	}
}
