package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// AuthMode controls how clients authenticate to the server.
type AuthMode string

const (
	// AuthModePublic — no authentication. Anyone with the URL can read.
	// The privacy is per-item (only readers who know vault-id and
	// manifest-id can fetch them); not per-reader.
	AuthModePublic AuthMode = "public"
	// AuthModeSignedChallenge — client authenticates by signing a
	// server-issued nonce with their own ed25519 keypair, and the
	// resulting signature is checked against an allow-list.
	AuthModeSignedChallenge AuthMode = "signed-challenge"
)

// Authenticator gates incoming requests. nil = public, no checks.
type Authenticator interface {
	// Allowed examines the request and returns nil iff it should
	// proceed. Implementations must NOT modify the request.
	Allowed(r *http.Request) error
}

type publicAuth struct{}

func (publicAuth) Allowed(_ *http.Request) error { return nil }

// signedChallengeAuth verifies signatures from an allow-list of
// client public keys. The flow:
//
//  1. Client GETs /challenge → server returns a fresh nonce (with
//     short TTL).
//  2. Client signs nonce + path with their private key.
//  3. Client sends Authorization: SignedChallenge
//     pubkey=<ed25519:hex>, nonce=<hex>, signature=<hex> on the
//     subsequent request.
//  4. Server verifies pubkey is allow-listed, nonce was issued and
//     not yet consumed, and signature is valid for nonce+path.
type signedChallengeAuth struct {
	allowed map[string]ed25519.PublicKey // hex(pubkey) → pubkey
	mu      sync.Mutex
	// nonces keyed by hex(nonce); value is expiry time. Once consumed,
	// removed.
	nonces  map[string]time.Time
	nonceTTL time.Duration
}

// NewSignedChallengeAuth builds a signed-challenge authenticator.
// allowedKeys is the list of client public keys (operator-pubkey
// format) that may access the server.
func NewSignedChallengeAuth(allowedKeys []string, nonceTTL time.Duration) (*signedChallengeAuth, error) {
	a := &signedChallengeAuth{
		allowed: make(map[string]ed25519.PublicKey),
		nonces:  make(map[string]time.Time),
		nonceTTL: nonceTTL,
	}
	if a.nonceTTL == 0 {
		a.nonceTTL = 5 * time.Minute
	}
	for _, raw := range allowedKeys {
		pub, err := ParseClientPubkey(raw)
		if err != nil {
			return nil, fmt.Errorf("parse allowed key %q: %w", raw, err)
		}
		a.allowed[hex.EncodeToString(pub)] = pub
	}
	if len(a.allowed) == 0 {
		return nil, fmt.Errorf("signed-challenge auth requires at least one allowed pubkey")
	}
	return a, nil
}

// IssueNonce generates and stores a fresh nonce. Returns hex.
func (a *signedChallengeAuth) IssueNonce() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		// Crypto rng failure is fatal; let the caller decide whether
		// to surface it or panic. For practical purposes rand.Read on
		// a healthy OS does not fail.
		return ""
	}
	hexNonce := hex.EncodeToString(buf)
	a.mu.Lock()
	a.nonces[hexNonce] = time.Now().Add(a.nonceTTL)
	a.expireLocked()
	a.mu.Unlock()
	return hexNonce
}

// expireLocked removes nonces past their TTL. Call with mu held.
func (a *signedChallengeAuth) expireLocked() {
	now := time.Now()
	for n, exp := range a.nonces {
		if exp.Before(now) {
			delete(a.nonces, n)
		}
	}
}

// Allowed implements Authenticator. The Authorization header must be:
//
//	SignedChallenge pubkey=<ed25519:hex>, nonce=<hex>, signature=<hex>
//
// Whitespace is tolerated; commas separate fields.
func (a *signedChallengeAuth) Allowed(r *http.Request) error {
	hdr := r.Header.Get("Authorization")
	if hdr == "" {
		return fmt.Errorf("missing Authorization header")
	}
	const prefix = "SignedChallenge "
	if !strings.HasPrefix(hdr, prefix) {
		return fmt.Errorf("unsupported auth scheme")
	}
	fields := parseAuthFields(hdr[len(prefix):])
	pubRaw := fields["pubkey"]
	nonce := fields["nonce"]
	sigHex := fields["signature"]
	if pubRaw == "" || nonce == "" || sigHex == "" {
		return fmt.Errorf("Authorization header missing required field(s)")
	}
	pub, err := ParseClientPubkey(pubRaw)
	if err != nil {
		return fmt.Errorf("parse pubkey: %w", err)
	}
	if _, ok := a.allowed[hex.EncodeToString(pub)]; !ok {
		return fmt.Errorf("pubkey not in allow-list")
	}

	a.mu.Lock()
	exp, hasNonce := a.nonces[nonce]
	if hasNonce {
		// Single-use: consume on success or failure either way.
		delete(a.nonces, nonce)
	}
	a.mu.Unlock()
	if !hasNonce {
		return fmt.Errorf("nonce unknown or already consumed")
	}
	if time.Now().After(exp) {
		return fmt.Errorf("nonce expired")
	}

	sig, err := hex.DecodeString(sigHex)
	if err != nil {
		return fmt.Errorf("decode signature hex: %w", err)
	}
	// Signed payload is "<nonce>:<request-path>" — binds the nonce to
	// the specific resource being requested so a captured signature
	// for /manifest/X cannot be replayed against /item/Y.
	payload := []byte(nonce + ":" + r.URL.Path)
	if !ed25519.Verify(pub, payload, sig) {
		return fmt.Errorf("signature does not verify against payload")
	}
	return nil
}

// parseAuthFields splits "k1=v1, k2=v2, k3=v3" into a map.
func parseAuthFields(s string) map[string]string {
	out := map[string]string{}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		eq := strings.IndexByte(part, '=')
		if eq < 0 {
			continue
		}
		k := strings.TrimSpace(part[:eq])
		v := strings.TrimSpace(part[eq+1:])
		// Strip optional surrounding quotes for tolerance.
		v = strings.Trim(v, `"`)
		out[k] = v
	}
	return out
}
