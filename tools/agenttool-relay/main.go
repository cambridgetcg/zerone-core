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
//	RELAY_GAS       declared gas limit (default 120000 — measured attestations use ~93k)
//	RELAY_FEES      tx fees (default RELAY_GAS×1uzrn — the consensus floor is 1uzrn/gas)
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
//	RELAY_WITNESS_WRITEBACK  set to "1" to report each confirmed attestation
//	                back to agenttool (POST /v1/invocations/{id}/witness).
//	                Default off: the route is not deployed on live yet.
//
// Auth monitoring: a 401 from the agenttool API logs RELAY-AUTH-EXPIRED and
// refreshes the sentinel file ~/.zerone-agent/RELAY-AUTH-ALERT (at most once
// per hour), so external monitoring can watch a file instead of parsing logs.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
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
	"sync"
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
	Gas            string
	Fees           string
	Binary         string
	StatePath      string
	IntervalSec    int
	Roles          []string
	// WitnessWriteback reports confirmed attestations back to agenttool.
	// Flag-gated (RELAY_WITNESS_WRITEBACK=1) because the /witness route is
	// not deployed on the live API yet.
	WitnessWriteback bool
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
		Gas:            envOr("RELAY_GAS", "120000"),
		Fees:           envOr("RELAY_FEES", envOr("RELAY_GAS", "120000")+"uzrn"),
		Binary:         envOr("RELAY_BINARY", "zeroned"),
		StatePath:      envOr("RELAY_STATE", defaultStatePath()),
		IntervalSec:    envInt("RELAY_INTERVAL", 90),
		Roles:          splitRoles(envOr("RELAY_ROLES", "seller,buyer")),

		WitnessWriteback: os.Getenv("RELAY_WITNESS_WRITEBACK") == "1",
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
	if resp.StatusCode == http.StatusUnauthorized {
		alertAuthExpired(authAlertPath(), url)
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
	if resp.StatusCode == http.StatusUnauthorized {
		alertAuthExpired(authAlertPath(), url)
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
// Auth alert — a file external monitoring can watch
// ---------------------------------------------------------------------------

// authAlertPath is the sentinel external monitoring watches for auth expiry.
func authAlertPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "RELAY-AUTH-ALERT"
	}
	return filepath.Join(home, ".zerone-agent", "RELAY-AUTH-ALERT")
}

// alertAuthExpired logs a distinctive RELAY-AUTH-EXPIRED line and refreshes
// the sentinel file, at most once per hour. The rate limit is keyed on the
// sentinel's mtime so it survives restarts without extra state. Returns
// whether the alert fired (rate-limit not yet elapsed → false).
func alertAuthExpired(path, url string) bool {
	if fi, err := os.Stat(path); err == nil && time.Since(fi.ModTime()) < time.Hour {
		return false
	}
	log.Printf("RELAY-AUTH-EXPIRED: agenttool API returned 401 for %s — RELAY_API_KEY needs rotation", url)
	msg := fmt.Sprintf("%s RELAY-AUTH-EXPIRED: agenttool API returned 401 for %s — RELAY_API_KEY needs rotation\n",
		time.Now().UTC().Format(time.RFC3339), url)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		log.Printf("auth-alert: mkdir %s: %v", filepath.Dir(path), err)
		return true
	}
	if err := os.WriteFile(path, []byte(msg), 0o600); err != nil {
		log.Printf("auth-alert: write %s: %v", path, err)
	}
	return true
}

// ---------------------------------------------------------------------------
// Watch state — the attested ledger
// ---------------------------------------------------------------------------

// maxAttestFailures parks an invocation after this many failed submit
// attempts, so a permanently-rejected attestation cannot storm the chain.
const maxAttestFailures = 5

// Attestation lifecycle statuses. The lifecycle is forward-only: once a tx
// hash is on the wire the record can only move forward (reconcile by hash),
// never backward into a resubmit — a resubmit while the first tx can still
// confirm is a duplicate attestation and a double bond.
const (
	// statusSubmissionUnknown: the broadcast succeeded (hash on the wire,
	// bond possibly already locked) but inclusion has not been observed.
	// Reconciled by tx hash on later polls; NEVER resubmitted.
	statusSubmissionUnknown = "submission_unknown"
	// statusAttested: inclusion observed, tx executed with code 0.
	statusAttested = "attested"
)

// maxReconcileNotFound: a submission_unknown record is released for
// resubmission only after this many CONSECUTIVE AUTHORITATIVE not-found
// probes — the node itself answered "tx not found" (probeNotFound). A probe
// that merely failed (node down, RPC error, unparseable output) proves
// nothing and counts for nothing. Each miss additionally counts only if the
// chain height advanced strictly past the previous counted miss (a halted
// chain cannot commit the tx, so its absence from the index proves nothing —
// `query tx` only sees committed txs and the tx may sit invisible in the
// mempool through the halt) and release further requires minReleaseWait of
// wall-clock time since submission. Any found result resets the count.
const maxReconcileNotFound = 10

// minReleaseWait: a submission_unknown record may be released for
// resubmission only after at least this much wall-clock time since
// SubmittedAt, no matter how many authoritative misses have accumulated.
// This keeps the release window independent of RELAY_INTERVAL: ten fast
// polls must never outrun mempool retention, and a resubmission while the
// original tx can still confirm is a duplicate attestation and a double bond.
const minReleaseWait = 15 * time.Minute

// AttestRecord is one ledger entry: a successful attestation (Status
// "attested"), an in-flight broadcast awaiting reconciliation (Status
// "submission_unknown"), or a failure trail (Failures, parked at
// maxAttestFailures). Legacy records written before the lifecycle existed
// carry TxHash with no Status — they were only persisted after observed
// inclusion, so an empty Status with TxHash set reads as attested.
type AttestRecord struct {
	TxHash        string `json:"tx_hash,omitempty"`
	AttestationID string `json:"attestation_id,omitempty"`
	AttestedAt    string `json:"attested_at,omitempty"`
	Status        string `json:"status,omitempty"`
	SubmittedAt   string `json:"submitted_at,omitempty"`
	NotFound      int    `json:"reconcile_not_found,omitempty"`
	// LastMissHeight is the chain height at which the last COUNTED not-found
	// probe was taken. A new miss counts only when the current height is
	// strictly greater — the chain committed blocks in which the tx had a
	// real chance to appear and did not. Optional and backward-compatible:
	// absent on older records it reads as 0, so the first counted miss
	// simply records the current height.
	LastMissHeight uint64 `json:"last_miss_height,omitempty"`
	Failures       int    `json:"failures,omitempty"`
	LastError      string `json:"last_error,omitempty"`
}

// shouldSkipSubmit is the fresh-submit guard: any record with a tx hash on
// the wire (attested, legacy-attested, or submission_unknown) or parked at
// the failure threshold must never re-enter the submit path.
func shouldSkipSubmit(rec *AttestRecord) bool {
	return rec != nil && (rec.TxHash != "" || rec.Failures >= maxAttestFailures)
}

// probeOutcome classifies one `zeroned query tx` observation. The zero value
// is probeError on purpose: an unclassified probe must never read as
// evidence of absence.
type probeOutcome int

const (
	// probeError: the probe itself failed — exec failure, RPC/transport
	// error, timeout, unparseable output. This proves NOTHING about the tx
	// and must never count toward release.
	probeError probeOutcome = iota
	// probeFound: the node knows the tx; Code carries its execution result.
	probeFound
	// probeNotFound: the CLI ran, the node answered, and the answer named
	// THIS tx as not found — authoritative evidence of absence right now
	// (though still not proof it can never confirm: it may sit in a mempool).
	probeNotFound
)

// txProbe is one observation of `zeroned query tx <hash>`. Outcome says what
// the probe proved; on probeFound, Code carries the execution code and, when
// code 0, AttestationID is recovered from the
// external_attestation_submitted event.
type txProbe struct {
	Outcome       probeOutcome
	Code          int
	RawLog        string
	AttestationID string
}

// reconcileAction is the decision for one submission_unknown reconcile pass.
type reconcileAction int

const (
	// reconcileKeepWaiting: authoritative not-found with the chain advanced,
	// but the release gates not yet all open — stay submission_unknown,
	// count the miss.
	reconcileKeepWaiting reconcileAction = iota
	// reconcileMarkAttested: tx found with code 0 — the bond is posted and
	// the attestation exists on-chain.
	reconcileMarkAttested
	// reconcileRecordFailure: tx found with code != 0 — it executed and
	// failed, state changes reverted, safe to retry through the normal path.
	reconcileRecordFailure
	// reconcileRelease: every gate open — maxReconcileNotFound consecutive
	// authoritative misses, each with the chain advancing, and minReleaseWait
	// elapsed since submission — release the record for resubmission via the
	// failure path.
	reconcileRelease
	// reconcileNoEvidence: the probe proved nothing (probe error, height
	// unknown, or chain not advancing since the last counted miss) — change
	// NOTHING: the miss count neither grows nor resets.
	reconcileNoEvidence
)

// decideReconcile is the pure decision function for a submission_unknown
// record: given one tx probe, the counted consecutive-miss state so far, the
// current chain height (0 = height query failed) and the wall-clock time the
// record has been in flight, pick the forward-only transition.
func decideReconcile(probe txProbe, notFoundSoFar int, lastMissHeight, currentHeight uint64, elapsed time.Duration) reconcileAction {
	switch probe.Outcome {
	case probeFound:
		if probe.Code == 0 {
			return reconcileMarkAttested
		}
		return reconcileRecordFailure
	case probeNotFound:
		// A miss counts only when the chain provably advanced past the last
		// counted miss: new blocks were committed in which the tx had a real
		// chance to appear and did not. A halted chain (or a failed height
		// query, currentHeight 0) turns absence-from-the-index into no
		// evidence at all — the tx may sit invisible in the mempool.
		if currentHeight == 0 || currentHeight <= lastMissHeight {
			return reconcileNoEvidence
		}
		if notFoundSoFar+1 >= maxReconcileNotFound && elapsed >= minReleaseWait {
			return reconcileRelease
		}
		return reconcileKeepWaiting
	default: // probeError
		return reconcileNoEvidence
	}
}

// sinceSubmit reports how long a record has been in flight. An absent or
// unparseable SubmittedAt reads as zero elapsed, which blocks release — the
// safe direction: never resubmit on unknown timing.
func sinceSubmit(rec *AttestRecord, now time.Time) time.Duration {
	t, err := time.Parse(time.RFC3339, rec.SubmittedAt)
	if err != nil {
		return 0
	}
	return now.Sub(t)
}

// applyReconcile applies the decided transition to the record and returns
// the action taken. currentHeight is the latest chain height (0 = unknown);
// now supplies the wall clock. Pure over its inputs.
func applyReconcile(rec *AttestRecord, probe txProbe, currentHeight uint64, now time.Time) reconcileAction {
	action := decideReconcile(probe, rec.NotFound, rec.LastMissHeight, currentHeight, sinceSubmit(rec, now))
	switch action {
	case reconcileMarkAttested:
		rec.Status = statusAttested
		rec.AttestationID = probe.AttestationID
		rec.AttestedAt = now.Format(time.RFC3339)
		rec.LastError = ""
		rec.NotFound = 0
		rec.LastMissHeight = 0
	case reconcileRecordFailure:
		rec.Failures++
		rec.LastError = fmt.Sprintf("tx %s executed with code %d: %s", rec.TxHash, probe.Code, truncate(probe.RawLog, 300))
		rec.TxHash = ""
		rec.Status = ""
		rec.SubmittedAt = ""
		rec.NotFound = 0
		rec.LastMissHeight = 0
	case reconcileRelease:
		rec.Failures++
		rec.LastError = fmt.Sprintf("tx %s authoritatively not found in %d consecutive probes with the chain advancing and %s elapsed since submission — released for resubmission", rec.TxHash, rec.NotFound+1, minReleaseWait)
		rec.TxHash = ""
		rec.Status = ""
		rec.SubmittedAt = ""
		rec.NotFound = 0
		rec.LastMissHeight = 0
	case reconcileKeepWaiting:
		rec.NotFound++
		rec.LastMissHeight = currentHeight
	case reconcileNoEvidence:
		// Deliberately nothing: an inconclusive probe neither counts toward
		// release nor resets the counted misses.
	}
	return action
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

// watchOnce runs a single poll pass: reconcile every in-flight broadcast
// first (a submission_unknown record blocks nothing but must resolve before
// its invocation could ever be released for retry), then list each
// configured role, confirm every unseen released invocation by ID, attest
// it, and record the outcome. Submits are sequential and wait for
// inclusion, so the signing account's sequence never races itself.
func watchOnce(cfg Config, st *WatchState) {
	reconcilePending(cfg, st)
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
			if shouldSkipSubmit(rec) {
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

			txHash, err := broadcastAttestation(cfg, inv)
			if err != nil {
				rec.Failures++
				rec.LastError = err.Error()
				log.Printf("watch: attest %s failed (%d/%d): %v", inv.ID, rec.Failures, maxAttestFailures, err)
				logIfParked(inv.ID, rec)
				if err := saveState(cfg.StatePath, st); err != nil {
					log.Printf("watch: save state: %v", err)
				}
				continue
			}

			// Forward-only lifecycle: the hash is on the wire and the bond
			// may already be locked. Persist BEFORE waiting, so a timeout,
			// crash, or restart can only reconcile this tx by hash — never
			// resubmit it (a resubmit while the first tx can still confirm
			// is a duplicate attestation and a double bond).
			rec.TxHash = txHash
			rec.Status = statusSubmissionUnknown
			rec.SubmittedAt = time.Now().UTC().Format(time.RFC3339)
			rec.NotFound = 0
			rec.LastMissHeight = 0
			rec.LastError = ""
			if err := saveState(cfg.StatePath, st); err != nil {
				// The on-disk ledger is the ONLY defense against
				// double-bonding this tx across a restart. Fail the pass
				// loudly and submit nothing further: every additional
				// broadcast would be another hash the disk cannot remember.
				// The record stays in memory, so reconcilePending still
				// resolves this tx by hash on later passes of this process.
				log.Printf("RELAY-STATE-WRITE-FAILED: cannot persist ledger %s after broadcast of %s (tx %s): %v — aborting this pass, no further submissions until the ledger writes again", cfg.StatePath, inv.ID, txHash, err)
				return
			}

			probe, werr := waitForInclusion(cfg, txHash)
			if werr != nil {
				log.Printf("watch: attest %s: %v — held as %s for reconciliation, will not resubmit", inv.ID, werr, statusSubmissionUnknown)
				continue
			}
			// waitForInclusion only ever hands back a probeFound, so the
			// height and wall-clock gates are unused here; height 0 keeps
			// any other outcome on the no-evidence (no-count) path.
			action := applyReconcile(rec, probe, 0, time.Now().UTC())
			logReconcile(inv.ID, rec, probe, action, txHash)
			if err := saveState(cfg.StatePath, st); err != nil {
				log.Printf("watch: save state: %v", err)
			}
			if action == reconcileMarkAttested {
				witnessWriteback(cfg, inv.ID, rec)
			}
		}
	}
}

// reconcilePending resolves submission_unknown records — broadcasts whose
// inclusion was never observed (wait timeout, crash, restart). Each is
// probed by tx hash; the forward-only transitions are decided by
// decideReconcile. Runs before any new submits so a released record can
// re-enter the submit path in the same pass only after the old tx is
// provably absent.
func reconcilePending(cfg Config, st *WatchState) {
	now := time.Now().UTC()
	var height uint64
	heightKnown := false
	for id, rec := range st.Attested {
		if rec.Status != statusSubmissionUnknown || rec.TxHash == "" {
			continue
		}
		if !heightKnown {
			// One height reading per pass, fetched lazily: a not-found probe
			// counts only against a chain that provably advanced, and 0
			// (height query failed) counts nothing.
			height = chainHeight(cfg.Node)
			heightKnown = true
		}
		txHash := rec.TxHash
		probe := queryTx(cfg, txHash)
		action := applyReconcile(rec, probe, height, now)
		logReconcile(id, rec, probe, action, txHash)
		if err := saveState(cfg.StatePath, st); err != nil {
			log.Printf("watch: save state: %v", err)
		}
		if action == reconcileMarkAttested {
			witnessWriteback(cfg, id, rec)
		}
	}
}

// logReconcile narrates one lifecycle transition; txHash is the hash before
// applyReconcile (failure/release transitions clear rec.TxHash).
func logReconcile(invID string, rec *AttestRecord, probe txProbe, action reconcileAction, txHash string) {
	switch action {
	case reconcileMarkAttested:
		if rec.AttestationID == "" {
			log.Printf("watch: attested %s (tx %s) but no attestation_id in tx events — backfill by querying the tx", invID, txHash)
		} else {
			log.Printf("watch: attested %s → %s (tx %s)", invID, rec.AttestationID, txHash)
		}
	case reconcileRecordFailure:
		log.Printf("watch: attest %s failed on-chain (%d/%d): tx %s code %d: %s", invID, rec.Failures, maxAttestFailures, txHash, probe.Code, truncate(probe.RawLog, 300))
		logIfParked(invID, rec)
	case reconcileRelease:
		log.Printf("watch: reconcile %s: tx %s authoritatively absent (%d+ consecutive misses, chain advancing, ≥%s since submit) — released for resubmission (%d/%d failures)", invID, txHash, maxReconcileNotFound, minReleaseWait, rec.Failures, maxAttestFailures)
		logIfParked(invID, rec)
	case reconcileKeepWaiting:
		suffix := ""
		if rec.NotFound >= maxReconcileNotFound {
			suffix = fmt.Sprintf(" (miss threshold met; %s wall-clock window since submit not yet elapsed)", minReleaseWait)
		}
		log.Printf("watch: reconcile %s: tx %s authoritatively not found (%d/%d counted misses) — staying %s%s", invID, txHash, rec.NotFound, maxReconcileNotFound, statusSubmissionUnknown, suffix)
	case reconcileNoEvidence:
		if probe.Outcome == probeError {
			log.Printf("watch: reconcile %s: tx %s probe failed (node unreachable, CLI error, or unparseable output) — no evidence, miss count stays %d/%d", invID, txHash, rec.NotFound, maxReconcileNotFound)
		} else {
			log.Printf("watch: reconcile %s: tx %s not found but chain height has not advanced past %d (halted, or height unknown) — not counted, miss count stays %d/%d", invID, txHash, rec.LastMissHeight, rec.NotFound, maxReconcileNotFound)
		}
	}
}

// logIfParked emits the distinctive RELAY-PARKED line external monitoring
// greps for, once per transition into the parked state. The instruction is
// verify-first on purpose: after a reconcile-driven release the original tx
// may STILL confirm once a halted chain resumes, so clearing the ledger
// entry before checking the chain risks a duplicate attestation and a
// double bond.
func logIfParked(invID string, rec *AttestRecord) {
	if rec.Failures == maxAttestFailures {
		log.Printf("RELAY-PARKED: invocation %s reached %d failures — will not retry. Do NOT clear the ledger entry until you have verified on-chain that NO attestation exists for this invocation: run `zeroned query tx <hash> --node <rpc> --output json` for every tx hash in the failure trail (a tx released during a halt can still confirm after resume); only when all report not found is it safe to clear the entry and retry; last error: %s", invID, rec.Failures, rec.LastError)
	}
}

// broadcastAttestation builds the link and broadcasts one attestation,
// returning the tx hash. It does NOT wait for inclusion — the caller must
// persist the hash first (forward-only lifecycle), then observe.
func broadcastAttestation(cfg Config, inv *Invocation) (string, error) {
	height := chainHeight(cfg.Node)
	linkBytes, err := buildLink(cfg, inv, height)
	if err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp("", "agenttool-link-*.json")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(linkBytes); err != nil {
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	return submitAttestation(cfg, tmp.Name())
}

// classifyQueryTxFailure decides what a failed `zeroned query tx` run
// proved. Authoritative not-found requires the CLI to have named THIS tx as
// missing: CometBFT answers `tx (HASH) not found` (rpc/core/tx.go) and the
// SDK query-tx command answers `no transaction found with hash HASH`
// (x/auth/client/cli/query.go) — both carry the hash. Anything else (binary
// missing, connection refused, timeout, tx indexing disabled, unrelated
// errors) proves nothing about the tx and must never count toward release.
func classifyQueryTxFailure(stderr, txHash string) probeOutcome {
	if txHash == "" {
		return probeError
	}
	s := strings.ToLower(stderr)
	if !strings.Contains(s, strings.ToLower(txHash)) {
		return probeError
	}
	// The two message shapes the stack actually emits for a missing tx; note
	// the SDK's "no transaction found" does NOT contain the substring
	// "not found", so both must be matched explicitly.
	if strings.Contains(s, "not found") || strings.Contains(s, "no transaction found") {
		return probeNotFound
	}
	return probeError
}

// queryTx probes the node once for a tx by hash, classifying the result into
// found / authoritatively-not-found / probe-error (see probeOutcome). Only
// the caller turns repeated authoritative absence into a release decision.
func queryTx(cfg Config, txHash string) txProbe {
	// #nosec G204 — arguments come from validated config
	out, err := exec.Command(cfg.Binary, "query", "tx", txHash,
		"--node", cfg.Node, "--output", "json").Output()
	if err != nil {
		// Output() captures stderr on the ExitError; any non-exit error
		// (binary missing, etc.) has no stderr and classifies as probeError.
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return txProbe{Outcome: classifyQueryTxFailure(string(ee.Stderr), txHash)}
		}
		return txProbe{}
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
		// The CLI exited 0 but printed something we cannot parse: treat as a
		// probe error, never as evidence of absence.
		return txProbe{}
	}
	probe := txProbe{Outcome: probeFound, Code: tx.Code, RawLog: tx.RawLog}
	for _, e := range tx.Events {
		if e.Type != "external_attestation_submitted" {
			continue
		}
		for _, a := range e.Attributes {
			if a.Key == "attestation_id" {
				probe.AttestationID = a.Value
			}
		}
	}
	return probe
}

// waitForInclusion polls the node until the tx is found (either outcome) or
// the window closes. A timeout is NOT a failure: the caller already holds
// the record in submission_unknown and later polls reconcile it.
func waitForInclusion(cfg Config, txHash string) (txProbe, error) {
	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(2 * time.Second)
		if probe := queryTx(cfg, txHash); probe.Outcome == probeFound {
			return probe, nil
		}
	}
	return txProbe{}, fmt.Errorf("tx %s not found on-chain within 45s", txHash)
}

// ---------------------------------------------------------------------------
// Witness writeback — report confirmed attestations back to agenttool
// ---------------------------------------------------------------------------

// writebackRouteMissing gates the "route not deployed" log to once per run:
// the route is being rebuilt in a parallel agenttool PR and 404 is expected
// on live until it ships.
var writebackRouteMissing sync.Once

// witnessWriteback POSTs the confirmed attestation back to agenttool.
// Flag-gated (cfg.WitnessWriteback) and strictly best-effort: the attested
// state is already persisted before this is called, and no writeback
// outcome may ever change it.
func witnessWriteback(cfg Config, invID string, rec *AttestRecord) {
	if !cfg.WitnessWriteback {
		return
	}
	body, err := json.Marshal(map[string]string{
		"chain_id":       cfg.ChainID,
		"tx_hash":        rec.TxHash,
		"attestation_id": rec.AttestationID,
		"adapter_id":     cfg.Adapter,
	})
	if err != nil {
		log.Printf("writeback %s: marshal: %v", invID, err)
		return
	}
	url := cfg.API + "/v1/invocations/" + invID + "/witness"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		log.Printf("writeback %s: %v", invID, err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("writeback %s: %v", invID, err)
		return
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	switch {
	case resp.StatusCode == http.StatusNotFound:
		writebackRouteMissing.Do(func() {
			log.Printf("writeback: route not deployed (404 for %s) — skipping for the rest of this run", url)
		})
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		log.Printf("writeback %s: witnessed (tx %s, %s)", invID, rec.TxHash, rec.AttestationID)
	default:
		log.Printf("writeback %s: agenttool returned %d: %s", invID, resp.StatusCode, truncate(string(respBody), 300))
	}
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
		"--gas", cfg.Gas,
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
