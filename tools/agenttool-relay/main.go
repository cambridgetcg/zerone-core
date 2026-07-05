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
//	agenttool-relay -watch                      daemon: poll for newly released
//	                                            invocations and attest each once
//	agenttool-relay -watch -once                single poll pass (cron-friendly)
//
// Watch mode additionally reads:
//
//	RELAY_STATE     attested-ledger path (default ~/.zerone-agent/agenttool-relay-state.json)
//	RELAY_INTERVAL  poll interval seconds (default 90)
//	RELAY_ROLES     comma-separated invocation roles to watch (default seller,buyer)
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
	"strconv"
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
	StatePath      string
	IntervalSec    int
	Roles          []string
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
		StatePath:      envOr("RELAY_STATE", defaultStatePath()),
		IntervalSec:    envInt("RELAY_INTERVAL", 90),
		Roles:          splitRoles(envOr("RELAY_ROLES", "seller,buyer")),
	}
}

func defaultStatePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "agenttool-relay-state.json"
	}
	return filepath.Join(home, ".zerone-agent", "agenttool-relay-state.json")
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

func splitRoles(s string) []string {
	var roles []string
	for _, r := range strings.Split(s, ",") {
		r = strings.TrimSpace(r)
		if r == "seller" || r == "buyer" {
			roles = append(roles, r)
		}
	}
	return roles
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

// checkAttestable enforces the relay's refusal rule: only a released
// invocation with a seller completion signature is a witnessable fact.
// agenttool's terminal states are "released" (escrow paid to the seller,
// settled_at set) and "refunded"; a refunded, escrowed, acknowledged, or
// still-in-review invocation is not completed work, and attesting it
// would put an untruth on a truth chain.
func checkAttestable(inv *Invocation) error {
	if inv.Status != "released" {
		return fmt.Errorf("invocation %s has status %q — only released invocations are attestable (escrow released, work delivered)", inv.ID, inv.Status)
	}
	if inv.CompletionSig == nil || *inv.CompletionSig == "" {
		return fmt.Errorf("invocation %s is released but carries no completion_sig — refusing to witness unsigned work", inv.ID)
	}
	if inv.SettledAt == nil || *inv.SettledAt == "" {
		return fmt.Errorf("invocation %s is released but settled_at is unset — refusing to witness an unsettled release", inv.ID)
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

// listInvocations retrieves every invocation visible to the project for one
// role. The endpoint supports no status/since filtering or pagination — the
// daemon filters client-side and confirms candidates via fetchInvocation
// (GET-by-id is also the path that runs agenttool's lazy SLA sweep).
func listInvocations(cfg Config, role string) ([]Invocation, error) {
	url := cfg.API + "/v1/invocations?role=" + role
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list invocations (%s): %w", role, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("agenttool returned %d for %s: %s", resp.StatusCode, url, truncate(string(body), 300))
	}
	var out struct {
		Invocations []Invocation `json:"invocations"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("parse invocation list: %w", err)
	}
	return out.Invocations, nil
}

// ---------------------------------------------------------------------------
// Watch state — the attested ledger
// ---------------------------------------------------------------------------

// maxAttestFailures parks an invocation after this many failed submit
// attempts, so a permanently-rejected attestation cannot storm the chain.
const maxAttestFailures = 5

// AttestRecord is one ledger entry: either a successful attestation
// (TxHash set) or a parked failure (Failures reached maxAttestFailures).
type AttestRecord struct {
	TxHash        string `json:"tx_hash,omitempty"`
	AttestationID string `json:"attestation_id,omitempty"`
	AttestedAt    string `json:"attested_at,omitempty"`
	Failures      int    `json:"failures,omitempty"`
	LastError     string `json:"last_error,omitempty"`
}

// WatchState is the persisted ledger keyed by invocation ID. It is what
// makes the daemon idempotent across restarts: an invocation is attested
// exactly once no matter how often it reappears in the list.
type WatchState struct {
	Attested map[string]*AttestRecord `json:"attested"`
}

func loadState(path string) (*WatchState, error) {
	st := &WatchState{Attested: map[string]*AttestRecord{}}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return st, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, st); err != nil {
		return nil, fmt.Errorf("corrupt state file %s: %w", path, err)
	}
	if st.Attested == nil {
		st.Attested = map[string]*AttestRecord{}
	}
	return st, nil
}

// saveState persists the ledger atomically (write temp + rename), so a
// crash mid-write can never truncate the ledger into double-attesting.
func saveState(path string, st *WatchState) error {
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ---------------------------------------------------------------------------
// Watch loop
// ---------------------------------------------------------------------------

// watchOnce runs a single poll pass: list each configured role, confirm
// every unseen released invocation by ID, attest it, and record the
// outcome. Submits are sequential and wait for inclusion, so the signing
// account's sequence never races itself.
func watchOnce(cfg Config, st *WatchState) {
	seen := map[string]bool{}
	for _, role := range cfg.Roles {
		rows, err := listInvocations(cfg, role)
		if err != nil {
			log.Printf("watch: %v", err)
			continue
		}
		for i := range rows {
			row := &rows[i]
			if seen[row.ID] || row.Status != "released" {
				continue
			}
			seen[row.ID] = true
			rec := st.Attested[row.ID]
			if rec != nil && (rec.TxHash != "" || rec.Failures >= maxAttestFailures) {
				continue
			}
			if rec == nil {
				rec = &AttestRecord{}
				st.Attested[row.ID] = rec
			}

			// Confirm via GET-by-id: authoritative status + full fields.
			inv, err := fetchInvocation(cfg, row.ID)
			if err != nil {
				log.Printf("watch: confirm %s: %v", row.ID, err)
				continue
			}
			if err := checkAttestable(inv); err != nil {
				log.Printf("watch: %v", err)
				continue
			}

			txHash, attID, err := attestInclusion(cfg, inv)
			if err != nil {
				rec.Failures++
				rec.LastError = err.Error()
				log.Printf("watch: attest %s failed (%d/%d): %v", inv.ID, rec.Failures, maxAttestFailures, err)
				if rec.Failures >= maxAttestFailures {
					log.Printf("watch: parking %s — will not retry (clear the ledger entry to retry)", inv.ID)
				}
			} else {
				rec.TxHash = txHash
				rec.AttestationID = attID
				rec.AttestedAt = time.Now().UTC().Format(time.RFC3339)
				rec.LastError = ""
				log.Printf("watch: attested %s → %s (tx %s)", inv.ID, attID, txHash)
			}
			if err := saveState(cfg.StatePath, st); err != nil {
				log.Printf("watch: save state: %v", err)
			}
		}
	}
}

// attestInclusion submits one attestation and waits until the tx is found
// on-chain, returning the tx hash and the attestation ID from the
// external_attestation_submitted event.
func attestInclusion(cfg Config, inv *Invocation) (string, string, error) {
	height := chainHeight(cfg.Node)
	linkBytes, err := buildLink(cfg, inv, height)
	if err != nil {
		return "", "", err
	}
	tmp, err := os.CreateTemp("", "agenttool-link-*.json")
	if err != nil {
		return "", "", err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(linkBytes); err != nil {
		return "", "", err
	}
	if err := tmp.Close(); err != nil {
		return "", "", err
	}
	txHash, err := submitAttestation(cfg, tmp.Name())
	if err != nil {
		return "", "", err
	}
	attID, err := waitForInclusion(cfg, txHash)
	if err != nil {
		return txHash, "", err
	}
	return txHash, attID, nil
}

// waitForInclusion polls the node until the tx executes, then returns the
// attestation_id attribute of the external_attestation_submitted event.
func waitForInclusion(cfg Config, txHash string) (string, error) {
	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(2 * time.Second)
		// #nosec G204 — arguments come from validated config
		out, err := exec.Command(cfg.Binary, "query", "tx", txHash,
			"--node", cfg.Node, "--output", "json").Output()
		if err != nil {
			continue // not yet indexed
		}
		var tx struct {
			Code   int    `json:"code"`
			RawLog string `json:"raw_log"`
			Events []struct {
				Type       string `json:"type"`
				Attributes []struct {
					Key   string `json:"key"`
					Value string `json:"value"`
				} `json:"attributes"`
			} `json:"events"`
		}
		if err := json.Unmarshal(out, &tx); err != nil {
			continue
		}
		if tx.Code != 0 {
			return "", fmt.Errorf("tx %s executed with code %d: %s", txHash, tx.Code, truncate(tx.RawLog, 300))
		}
		for _, e := range tx.Events {
			if e.Type != "external_attestation_submitted" {
				continue
			}
			for _, a := range e.Attributes {
				if a.Key == "attestation_id" {
					return a.Value, nil
				}
			}
		}
		return "", fmt.Errorf("tx %s executed but carried no external_attestation_submitted event", txHash)
	}
	return "", fmt.Errorf("tx %s not found on-chain within 45s", txHash)
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
	watch := flag.Bool("watch", false, "daemon mode: poll for newly released invocations and attest each once")
	once := flag.Bool("once", false, "with -watch: run a single poll pass and exit")
	flag.Parse()

	if *invocationID == "" && !*watch {
		log.Fatal("usage: agenttool-relay -invocation <id> [-dry-run] | agenttool-relay -watch [-once]")
	}
	cfg := loadConfig()
	if cfg.APIKey == "" {
		log.Fatal("RELAY_API_KEY is required")
	}
	if *watch || !*dryRun {
		if cfg.Home == "" || cfg.ChainID == "" || cfg.From == "" {
			log.Fatal("RELAY_HOME, RELAY_CHAIN_ID, RELAY_FROM are required to broadcast (or pass -dry-run)")
		}
	}

	if *watch {
		if len(cfg.Roles) == 0 {
			log.Fatal("RELAY_ROLES must include seller and/or buyer")
		}
		st, err := loadState(cfg.StatePath)
		if err != nil {
			log.Fatalf("load state: %v", err)
		}
		log.Printf("watch: roles=%s interval=%ds state=%s adapter=%s",
			strings.Join(cfg.Roles, ","), cfg.IntervalSec, cfg.StatePath, cfg.Adapter)
		for {
			watchOnce(cfg, st)
			if *once {
				return
			}
			time.Sleep(time.Duration(cfg.IntervalSec) * time.Second)
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
