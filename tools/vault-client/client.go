package vaultclient

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Option configures a VaultClient.
type Option func(*VaultClient)

// VaultClient communicates with a remote vault service over HTTPS to retrieve
// public keys, request signatures, and verify vault identity via challenge.
type VaultClient struct {
	endpoint   string
	httpClient *http.Client
	pubKey     ed25519.PublicKey
	maxRetries int
	retryDelay time.Duration
}

// WithMaxRetries sets the maximum number of retries for transient failures.
func WithMaxRetries(n int) Option {
	return func(c *VaultClient) {
		c.maxRetries = n
	}
}

// WithRetryDelay sets the base delay between retries (doubles on each attempt).
func WithRetryDelay(d time.Duration) Option {
	return func(c *VaultClient) {
		c.retryDelay = d
	}
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *VaultClient) {
		c.httpClient.Timeout = d
	}
}

// NewVaultClient creates a VaultClient targeting the given endpoint.
// The endpoint must begin with "https://".
func NewVaultClient(endpoint string, opts ...Option) *VaultClient {
	c := &VaultClient{
		endpoint:   strings.TrimRight(endpoint, "/"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
		maxRetries: 3,
		retryDelay: 500 * time.Millisecond,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// ---------- request / response helpers ----------

type publicKeyResponse struct {
	PublicKey string `json:"public_key"`
}

type signRequest struct {
	Payload string `json:"payload"`
}

type signResponse struct {
	Signature string `json:"signature"`
}

type challengeRequest struct {
	Nonce string `json:"nonce"`
}

type challengeResponse struct {
	Signature string `json:"signature"`
}

// ---------- public API ----------

// GetPublicKey fetches the vault's ed25519 public key. The result is cached;
// subsequent calls return the cached key without hitting the network.
func (c *VaultClient) GetPublicKey() (ed25519.PublicKey, error) {
	if c.pubKey != nil {
		return c.pubKey, nil
	}

	if !strings.HasPrefix(c.endpoint, "https://") {
		return nil, fmt.Errorf("vault: endpoint must start with \"https://\", got %q", c.endpoint)
	}

	resp, err := c.httpClient.Get(c.endpoint + "/v1/public-key")
	if err != nil {
		return nil, fmt.Errorf("vault: GET /v1/public-key: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vault: GET /v1/public-key: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("vault: reading public-key response: %w", err)
	}

	var pkResp publicKeyResponse
	if err := json.Unmarshal(body, &pkResp); err != nil {
		return nil, fmt.Errorf("vault: decoding public-key response: %w", err)
	}

	decoded, err := hex.DecodeString(pkResp.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("vault: decoding public-key hex: %w", err)
	}

	if len(decoded) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("vault: public key has wrong length %d, want %d", len(decoded), ed25519.PublicKeySize)
	}

	c.pubKey = ed25519.PublicKey(decoded)
	return c.pubKey, nil
}

// RequestSignature asks the vault to sign an arbitrary payload. The call is
// retried with exponential backoff on 5xx responses and transient network
// errors up to maxRetries times.
func (c *VaultClient) RequestSignature(payload []byte) ([]byte, error) {
	if !strings.HasPrefix(c.endpoint, "https://") {
		return nil, fmt.Errorf("vault: endpoint must start with \"https://\", got %q", c.endpoint)
	}

	reqBody, err := json.Marshal(signRequest{
		Payload: base64.StdEncoding.EncodeToString(payload),
	})
	if err != nil {
		return nil, fmt.Errorf("vault: marshalling sign request: %w", err)
	}

	var lastErr error
	delay := c.retryDelay

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(delay)
			delay *= 2
		}

		resp, err := c.httpClient.Post(
			c.endpoint+"/v1/sign",
			"application/json",
			strings.NewReader(string(reqBody)),
		)
		if err != nil {
			lastErr = fmt.Errorf("vault: POST /v1/sign: %w", err)
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("vault: POST /v1/sign: server error %d", resp.StatusCode)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("vault: POST /v1/sign: unexpected status %d", resp.StatusCode)
		}

		if readErr != nil {
			return nil, fmt.Errorf("vault: reading sign response: %w", readErr)
		}

		var sigResp signResponse
		if err := json.Unmarshal(body, &sigResp); err != nil {
			return nil, fmt.Errorf("vault: decoding sign response: %w", err)
		}

		sig, err := hex.DecodeString(sigResp.Signature)
		if err != nil {
			return nil, fmt.Errorf("vault: decoding signature hex: %w", err)
		}

		return sig, nil
	}

	return nil, fmt.Errorf("vault: POST /v1/sign: exhausted %d retries: %w", c.maxRetries, lastErr)
}

// VerifyVaultIdentity generates a random 32-byte nonce, sends it to the vault
// as a challenge, and verifies the returned signature against the cached (or
// freshly fetched) public key.
func (c *VaultClient) VerifyVaultIdentity() error {
	if !strings.HasPrefix(c.endpoint, "https://") {
		return fmt.Errorf("vault: endpoint must start with \"https://\", got %q", c.endpoint)
	}

	pubKey, err := c.GetPublicKey()
	if err != nil {
		return fmt.Errorf("vault: fetching public key for challenge: %w", err)
	}

	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("vault: generating nonce: %w", err)
	}

	reqBody, err := json.Marshal(challengeRequest{
		Nonce: hex.EncodeToString(nonce),
	})
	if err != nil {
		return fmt.Errorf("vault: marshalling challenge request: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.endpoint+"/v1/challenge",
		"application/json",
		strings.NewReader(string(reqBody)),
	)
	if err != nil {
		return fmt.Errorf("vault: POST /v1/challenge: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("vault: POST /v1/challenge: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("vault: reading challenge response: %w", err)
	}

	var chalResp challengeResponse
	if err := json.Unmarshal(body, &chalResp); err != nil {
		return fmt.Errorf("vault: decoding challenge response: %w", err)
	}

	sig, err := hex.DecodeString(chalResp.Signature)
	if err != nil {
		return fmt.Errorf("vault: decoding challenge signature hex: %w", err)
	}

	if !ed25519.Verify(pubKey, nonce, sig) {
		return fmt.Errorf("vault: challenge signature verification failed")
	}

	return nil
}
