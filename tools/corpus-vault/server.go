package main

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Server is the corpus-vault HTTP server.
type Server struct {
	cfg       *Config
	priv      ed25519.PrivateKey
	pub       ed25519.PublicKey
	auth      Authenticator
	signedAuth *signedChallengeAuth // non-nil iff auth_mode=signed-challenge
	mux       *http.ServeMux

	// Cached manifests, keyed by manifest ID. Built at startup.
	manifests map[string]SignedManifest

	// Per-manifest item root, keyed by manifest ID. Used by
	// /item/{id} to locate the requested file on disk.
	itemRoots map[string]string

	// Optional access log file.
	accessLog *os.File
	mu        sync.Mutex
}

// NewServer builds a Server from config.
func NewServer(cfg *Config) (*Server, error) {
	priv, err := LoadPrivateKey(cfg.PrivateKeyPath)
	if err != nil {
		return nil, err
	}
	s := &Server{
		cfg:       cfg,
		priv:      priv,
		pub:       priv.Public().(ed25519.PublicKey),
		mux:       http.NewServeMux(),
		manifests: map[string]SignedManifest{},
		itemRoots: map[string]string{},
	}

	switch cfg.AuthMode {
	case AuthModePublic, "":
		s.auth = publicAuth{}
	case AuthModeSignedChallenge:
		ttl := cfg.NonceTTL
		if ttl == 0 {
			ttl = 5 * time.Minute
		}
		sa, err := NewSignedChallengeAuth(cfg.AllowedClientKeys, ttl)
		if err != nil {
			return nil, err
		}
		s.signedAuth = sa
		s.auth = sa
	default:
		return nil, fmt.Errorf("unknown auth_mode: %s", cfg.AuthMode)
	}

	if cfg.AccessLogPath != "" {
		f, err := os.OpenFile(cfg.AccessLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, fmt.Errorf("open access log: %w", err)
		}
		s.accessLog = f
	}

	if err := s.buildManifests(); err != nil {
		return nil, err
	}
	s.registerRoutes()
	return s, nil
}

// buildManifests walks each configured manifest's item directory,
// hashes the files, and signs the resulting body. The signed
// manifest is held in memory; restart the server to pick up new
// items.
func (s *Server) buildManifests() error {
	for _, ms := range s.cfg.Manifests {
		root := filepath.Join(s.cfg.ItemsRoot, ms.ItemRoot)
		body, err := BuildManifestFromDir(ms.ID, s.cfg.VaultID, root, "/item/"+ms.ID+"/")
		if err != nil {
			return fmt.Errorf("build manifest %s: %w", ms.ID, err)
		}
		sig, err := SignManifest(s.priv, body)
		if err != nil {
			return fmt.Errorf("sign manifest %s: %w", ms.ID, err)
		}
		s.manifests[ms.ID] = SignedManifest{ManifestBody: body, Signature: sig}
		s.itemRoots[ms.ID] = root

		// Print on-chain content_hash so the operator knows what to
		// publish via MsgPublishManifest.
		ch, _ := ContentHash(body)
		log.Printf("manifest %q built — content_hash=%s items=%d", ms.ID, ch, len(body.Items))
	}
	return nil
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/healthz", s.handleHealth)
	s.mux.HandleFunc("/pubkey", s.handlePubkey)
	s.mux.HandleFunc("/challenge", s.handleChallenge)
	s.mux.HandleFunc("/manifest/", s.handleManifest)
	s.mux.HandleFunc("/item/", s.handleItem)
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// Run starts the HTTP server. Blocks.
func (s *Server) Run() error {
	log.Printf("corpus-vault listening on %s (auth=%s, manifests=%d)",
		s.cfg.ListenAddress, s.cfg.AuthMode, len(s.manifests))
	log.Printf("operator public key (publish on-chain): %s", EncodePubkey(s.pub))
	srv := &http.Server{
		Addr:              s.cfg.ListenAddress,
		Handler:           s,
		ReadHeaderTimeout: 10 * time.Second,
	}
	return srv.ListenAndServe()
}

// Close releases resources (access log).
func (s *Server) Close() error {
	if s.accessLog != nil {
		return s.accessLog.Close()
	}
	return nil
}

// ─── handlers ───────────────────────────────────────────────────────

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// handlePubkey is unauthenticated and returns the operator's public
// key. Useful for debugging and for clients that don't yet have the
// on-chain registration but want to talk to the server. It does NOT
// substitute for the on-chain operator_pubkey — readers should still
// fetch that from the chain to verify they're talking to the right
// vault.
func (s *Server) handlePubkey(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	resp := map[string]string{
		"vault_id":  s.cfg.VaultID,
		"pubkey":    EncodePubkey(s.pub),
		"auth_mode": string(s.cfg.AuthMode),
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// handleChallenge issues a fresh nonce for signed-challenge auth.
func (s *Server) handleChallenge(w http.ResponseWriter, _ *http.Request) {
	if s.signedAuth == nil {
		http.Error(w, "challenge endpoint disabled (auth_mode is not signed-challenge)", http.StatusNotFound)
		return
	}
	nonce := s.signedAuth.IssueNonce()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"nonce": nonce,
	})
}

// handleManifest returns a signed manifest. Path: /manifest/{id}.
func (s *Server) handleManifest(w http.ResponseWriter, r *http.Request) {
	if err := s.auth.Allowed(r); err != nil {
		s.logAccess(r, "/manifest", "denied: "+err.Error())
		http.Error(w, "unauthorised: "+err.Error(), http.StatusUnauthorized)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/manifest/")
	if id == "" {
		http.Error(w, "manifest id required", http.StatusBadRequest)
		return
	}
	m, ok := s.manifests[id]
	if !ok {
		s.logAccess(r, "/manifest/"+id, "not found")
		http.Error(w, "manifest not found", http.StatusNotFound)
		return
	}
	s.logAccess(r, "/manifest/"+id, "ok")
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(m)
}

// handleItem returns raw bytes of one item. Path: /item/{manifest_id}/{path...}
//
// The manifest_id segment scopes the lookup to a specific manifest's
// configured item root. Items are NOT cross-manifest — if you publish
// the same item under two manifests, you're shipping two copies and
// they have two URLs.
func (s *Server) handleItem(w http.ResponseWriter, r *http.Request) {
	if err := s.auth.Allowed(r); err != nil {
		s.logAccess(r, "/item", "denied: "+err.Error())
		http.Error(w, "unauthorised: "+err.Error(), http.StatusUnauthorized)
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/item/")
	slash := strings.IndexByte(rest, '/')
	if slash <= 0 {
		http.Error(w, "expected /item/{manifest_id}/{path}", http.StatusBadRequest)
		return
	}
	manifestID := rest[:slash]
	itemPath := rest[slash+1:]
	root, ok := s.itemRoots[manifestID]
	if !ok {
		http.Error(w, "manifest not found", http.StatusNotFound)
		return
	}
	full := filepath.Join(root, filepath.FromSlash(itemPath))
	// Refuse path traversal.
	cleaned, err := filepath.Abs(full)
	if err != nil {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		http.Error(w, "server config error", http.StatusInternalServerError)
		return
	}
	if !strings.HasPrefix(cleaned, rootAbs+string(filepath.Separator)) && cleaned != rootAbs {
		http.Error(w, "path traversal denied", http.StatusBadRequest)
		return
	}
	f, err := os.Open(cleaned)
	if err != nil {
		s.logAccess(r, r.URL.Path, "item not found")
		http.Error(w, "item not found", http.StatusNotFound)
		return
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		http.Error(w, "stat: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size()))
	s.logAccess(r, r.URL.Path, "ok")
	_, _ = io.Copy(w, f)
}

// logAccess writes a single JSON line per request to the access log
// (if configured). The format is intentionally minimal so an operator
// can grep it; promote interesting lines to on-chain
// MsgRecordAccess at their discretion.
func (s *Server) logAccess(r *http.Request, target, outcome string) {
	if s.accessLog == nil {
		return
	}
	entry := map[string]string{
		"time":      time.Now().UTC().Format(time.RFC3339),
		"remote":    r.RemoteAddr,
		"method":    r.Method,
		"target":    target,
		"outcome":   outcome,
		"vault_id":  s.cfg.VaultID,
		"auth_mode": string(s.cfg.AuthMode),
		"client_pubkey": authClientPubkey(r),
	}
	bz, _ := json.Marshal(entry)
	s.mu.Lock()
	defer s.mu.Unlock()
	_, _ = s.accessLog.Write(append(bz, '\n'))
}

// authClientPubkey returns the pubkey field of a SignedChallenge
// header, or "" if none.
func authClientPubkey(r *http.Request) string {
	hdr := r.Header.Get("Authorization")
	const prefix = "SignedChallenge "
	if !strings.HasPrefix(hdr, prefix) {
		return ""
	}
	for _, part := range strings.Split(hdr[len(prefix):], ",") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "pubkey=") {
			return strings.Trim(part[len("pubkey="):], `"`)
		}
	}
	return ""
}
