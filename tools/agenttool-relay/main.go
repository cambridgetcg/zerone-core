// Package main implements the agenttool → ZERONE attestation relay
// (agenttool bridge, slice 02).
//
// Slice 01 woke the substrate_bridge CLI and proved the path by hand: a
// registered agenttool-invocation-v1 adapter accepted a hand-built
// attestation on a live localnet. This tool automates that path for the
// one event class the wire-spec (agenttool docs/ZERONE-WIRE.md §2.3) names
// first: a completed marketplace invocation. It fetches the invocation
// from an agenttool instance, canonicalizes the economically-load-bearing
// fields, and submits a MsgSubmitExternalAttestation whose SubstrateLink
// source carries the invocation's identity and content hash — a completed
// agent invocation, witnessed on-chain with real provenance.
//
// The relay witnesses; it does not vouch. It refuses invocations that have
// not settled (escrow released), because an unsettled invocation is not yet
// a fact about value having moved.
//
// Configuration is via environment variables:
//
//	RELAY_API_KEY   (required) agenttool bearer token (invocations are identity-scoped)
//	RELAY_HOME      (required for broadcast) zeroned home directory
//	RELAY_CHAIN_ID  (required for broadcast) chain ID for tx signing
//	RELAY_FROM      (required for broadcast) signing key name
//	RELAY_API       agenttool API base (default https://api.agenttool.dev)
//	RELAY_NODE      CometBFT RPC endpoint (default tcp://localhost:26657)
//	RELAY_KEYRING_BACKEND keyring backend (default test)
//	RELAY_ADAPTER   adapter ID (default agenttool-invocation-v1)
//	RELAY_WORK_CLASS work class ID (default agenttool.invocation)
//	RELAY_BOND_UZRN bond in uzrn (default 1000000 — adapter min, no pending claims)
//	RELAY_FEES      tx fees (default 200000uzrn — localnet min-gas-price × default gas)
//	RELAY_BINARY    zeroned binary path (default zeroned)
//
// Usage:
//
//	agenttool-relay -invocation <id>            fetch, verify, attest
//	agenttool-relay -invocation <id> -dry-run   fetch, verify, print link + command
package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

// Config holds all relay runtime settings, populated from environment variables.
type Config struct {
	API            string
	APIKey         string
	Node           string
	Home           string
	ChainID        string
	From           string
	KeyringBackend string
	Adapter        string
	WorkClass      string
	BondUzrn       string
	Fees           string
	Binary         string
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// loadConfig reads configuration from environment variables. RELAY_API_KEY is
// always required; the signing trio (RELAY_HOME, RELAY_CHAIN_ID, RELAY_FROM)
// is required only when broadcasting (checked in main, after -dry-run is known).
func loadConfig() Config {
	return Config{
		API:            strings.TrimRight(envOr("RELAY_API", "https://api.agenttool.dev"), "/"),
		APIKey:         os.Getenv("RELAY_API_KEY"),
		Node:           envOr("RELAY_NODE", "tcp://localhost:26657"),
		Home:           os.Getenv("RELAY_HOME"),
		ChainID:        os.Getenv("RELAY_CHAIN_ID"),
		From:           os.Getenv("RELAY_FROM"),
		KeyringBackend: envOr("RELAY_KEYRING_BACKEND", "test"),
		Adapter:        envOr("RELAY_ADAPTER", "agenttool-invocation-v1"),
		WorkClass:      envOr("RELAY_WORK_CLASS", "agenttool.invocation"),
		BondUzrn:       envOr("RELAY_BOND_UZRN", "1000000"),
		Fees:           envOr("RELAY_FEES", "200000uzrn"),
		Binary:         envOr("RELAY_BINARY", "zeroned"),
	}
}

// ---------------------------------------------------------------------------
// agenttool invocation
// ---------------------------------------------------------------------------

// Invocation is the subset of agenttool's GET /v1/invocations/:id response
// the relay reads. Unknown fields are ignored on purpose: the witnessed
// canonical form is the explicit subset below, not the whole document.
type Invocation struct {
	ID            string  `json:"id"`
	ListingID     string  `json:"listing_id"`
	BuyerDID      string  `json:"buyer_did"`
	Amount        int64   `json:"amount"`
	Currency      string  `json:"currency"`
	Status        string  `json:"status"`
	CompletionSig *string `json:"completion_sig"`
	CreatedAt     string  `json:"created_at"`
	CompletedAt   *string `json:"completed_at"`
	SettledAt     *string `json:"settled_at"`
}

// fetchInvocation retrieves one invocation from the agenttool API.
func fetchInvocation(cfg Config, id string) (*Invocation, error) {
	url := cfg.API + "/v1/invocations/" + id
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch invocation: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("agenttool returned %d for %s: %s", resp.StatusCode, url, truncate(string(body), 300))
	}
	var inv Invocation
	if err := json.Unmarshal(body, &inv); err != nil {
		return nil, fmt.Errorf("parse invocation: %w", err)
	}
	if inv.ID != id {
		return nil, fmt.Errorf("agenttool returned invocation %q, wanted %q", inv.ID, id)
	}
	return &inv, nil
}

// checkAttestable enforces the relay's refusal rule: only a settled
// invocation with a seller completion signature is a witnessable fact.
// A refunded, declined, escrowed, or still-in-review invocation is not
// completed work, and attesting it would put an untruth on a truth chain.
func checkAttestable(inv *Invocation) error {
	if inv.Status != "settled" {
		return fmt.Errorf("invocation %s has status %q — only settled invocations are attestable (escrow released, work delivered)", inv.ID, inv.Status)
	}
	if inv.CompletionSig == nil || *inv.CompletionSig == "" {
		return fmt.Errorf("invocation %s is settled but carries no completion_sig — refusing to witness unsigned work", inv.ID)
	}
	return nil
}

// canonicalInvocation is the witnessed form: the economically-load-bearing
// fields, in a fixed struct order, marshaled without indentation. The
// sha256 of these bytes is the SubstrateLink source content_hash; anyone
// can re-derive it from the public invocation record (M2 re-derivability,
// same discipline as keeper.ComputeLinkHash).
type canonicalInvocation struct {
	Amount        int64   `json:"amount"`
	BuyerDID      string  `json:"buyer_did"`
	CompletedAt   *string `json:"completed_at"`
	CompletionSig *string `json:"completion_sig"`
	CreatedAt     string  `json:"created_at"`
	Currency      string  `json:"currency"`
	ID            string  `json:"id"`
	ListingID     string  `json:"listing_id"`
	SettledAt     *string `json:"settled_at"`
	Status        string  `json:"status"`
}

// contentHash returns sha256 over the canonical JSON form of the invocation.
func contentHash(inv *Invocation) ([]byte, []byte, error) {
	canon, err := json.Marshal(canonicalInvocation{
		Amount:        inv.Amount,
		BuyerDID:      inv.BuyerDID,
		CompletedAt:   inv.CompletedAt,
		CompletionSig: inv.CompletionSig,
		CreatedAt:     inv.CreatedAt,
		Currency:      inv.Currency,
		ID:            inv.ID,
		ListingID:     inv.ListingID,
		SettledAt:     inv.SettledAt,
		Status:        inv.Status,
	})
	if err != nil {
		return nil, nil, err
	}
	sum := sha256.Sum256(canon)
	return sum[:], canon, nil
}

// ---------------------------------------------------------------------------
// SubstrateLink assembly
// ---------------------------------------------------------------------------

// linkJSON mirrors types.SubstrateLink's std-encoding/json shape (the CLI
// reads the file with encoding/json, so field names follow the pb.go json
// tags and []byte is base64). adapter_id and link_hash are omitted: the
// submit-attestation command sets both itself.
type linkJSON struct {
	Source sourceJSON `json:"source"`
}

type sourceJSON struct {
	SourceID       string `json:"source_id"`
	SourceURL      string `json:"source_url"`
	ContentHash    string `json:"content_hash"`
	FetchedAtBlock uint64 `json:"fetched_at_block,omitempty"`
}

// buildLink assembles the SubstrateLink JSON for one invocation.
func buildLink(cfg Config, inv *Invocation, height uint64) ([]byte, error) {
	hash, _, err := contentHash(inv)
	if err != nil {
		return nil, err
	}
	link := linkJSON{
		Source: sourceJSON{
			SourceID:       inv.ID,
			SourceURL:      cfg.API + "/v1/invocations/" + inv.ID,
			ContentHash:    base64.StdEncoding.EncodeToString(hash),
			FetchedAtBlock: height,
		},
	}
	return json.MarshalIndent(link, "", "  ")
}

// chainHeight asks the CometBFT RPC for the latest block height, so the
// link records when the relay observed the invocation. Best-effort: on any
// failure it returns 0 (the field is optional provenance, not consensus).
func chainHeight(node string) uint64 {
	url := strings.Replace(node, "tcp://", "http://", 1) + "/status"
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	var status struct {
		Result struct {
			SyncInfo struct {
				LatestBlockHeight string `json:"latest_block_height"`
			} `json:"sync_info"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return 0
	}
	var h uint64
	_, _ = fmt.Sscanf(status.Result.SyncInfo.LatestBlockHeight, "%d", &h)
	return h
}

// ---------------------------------------------------------------------------
// Broadcast
// ---------------------------------------------------------------------------

// submitAttestation shells out to zeroned to broadcast the attestation.
// It returns the tx hash on success.
func submitAttestation(cfg Config, linkFile string) (string, error) {
	// #nosec G204 — arguments come from validated config and a temp path we created
	cmd := exec.Command(cfg.Binary, "tx", "substrate_bridge", "submit-attestation",
		cfg.Adapter, cfg.WorkClass, linkFile, cfg.BondUzrn,
		"--from", cfg.From,
		"--keyring-backend", cfg.KeyringBackend,
		"--home", cfg.Home,
		"--chain-id", cfg.ChainID,
		"--node", cfg.Node,
		"--fees", cfg.Fees,
		"--broadcast-mode", "sync",
		"--output", "json",
		"--yes",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("zeroned tx failed: %w: %s", err, string(out))
	}
	var result map[string]interface{}
	if err := json.Unmarshal(out, &result); err != nil {
		return "", fmt.Errorf("parse tx output: %w: %s", err, string(out))
	}
	if code, ok := result["code"].(float64); ok && code != 0 {
		rawLog, _ := result["raw_log"].(string)
		return "", fmt.Errorf("tx error code %d: %s", int(code), rawLog)
	}
	txHash, _ := result["txhash"].(string)
	if txHash == "" {
		return "", fmt.Errorf("no txhash in response: %s", string(out))
	}
	return txHash, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	invocationID := flag.String("invocation", "", "agenttool invocation ID to attest")
	dryRun := flag.Bool("dry-run", false, "build and print the link + command without broadcasting")
	flag.Parse()

	if *invocationID == "" {
		log.Fatal("usage: agenttool-relay -invocation <id> [-dry-run]")
	}
	cfg := loadConfig()
	if cfg.APIKey == "" {
		log.Fatal("RELAY_API_KEY is required")
	}
	if !*dryRun {
		if cfg.Home == "" || cfg.ChainID == "" || cfg.From == "" {
			log.Fatal("RELAY_HOME, RELAY_CHAIN_ID, RELAY_FROM are required to broadcast (or pass -dry-run)")
		}
	}

	inv, err := fetchInvocation(cfg, *invocationID)
	if err != nil {
		log.Fatalf("fetch: %v", err)
	}
	if err := checkAttestable(inv); err != nil {
		log.Fatalf("refused: %v", err)
	}

	height := chainHeight(cfg.Node)
	linkBytes, err := buildLink(cfg, inv, height)
	if err != nil {
		log.Fatalf("build link: %v", err)
	}

	if *dryRun {
		fmt.Printf("invocation %s: settled, attestable\nlink:\n%s\n\nwould run: %s tx substrate_bridge submit-attestation %s %s <link.json> %s\n",
			inv.ID, linkBytes, cfg.Binary, cfg.Adapter, cfg.WorkClass, cfg.BondUzrn)
		return
	}

	tmp, err := os.CreateTemp("", "agenttool-link-*.json")
	if err != nil {
		log.Fatalf("temp file: %v", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(linkBytes); err != nil {
		log.Fatalf("write link: %v", err)
	}
	if err := tmp.Close(); err != nil {
		log.Fatalf("close link file: %v", err)
	}

	txHash, err := submitAttestation(cfg, filepath.Clean(tmp.Name()))
	if err != nil {
		log.Fatalf("submit: %v", err)
	}
	fmt.Printf("attested invocation %s → tx %s (adapter %s, class %s, bond %s uzrn, observed at block %d)\n",
		inv.ID, txHash, cfg.Adapter, cfg.WorkClass, cfg.BondUzrn, height)
}
