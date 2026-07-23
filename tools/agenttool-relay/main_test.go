package main

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

func strPtr(s string) *string { return &s }

// settledInvocation returns a fully-released invocation fixture.
func settledInvocation() *Invocation {
	return &Invocation{
		ID:            "529ff750-fec4-4bfc-863e-8b101afbe1d8",
		ListingID:     "57809f83-c588-4174-ad7e-6ed60c134499",
		BuyerDID:      "did:at:09c5e59e-0374-4d80-a2c1-d8f1acbdfe9a",
		Amount:        53,
		Currency:      "GBP",
		Status:        "released",
		CompletionSig: strPtr("c2lnbmF0dXJl"),
		CreatedAt:     "2026-07-05T10:09:51.018Z",
		CompletedAt:   strPtr("2026-07-05T11:00:00.000Z"),
		SettledAt:     strPtr("2026-07-05T11:00:00.100Z"),
	}
}

// ---------------------------------------------------------------------------
// TestCheckAttestable — table-driven refusal rules
// ---------------------------------------------------------------------------

func TestCheckAttestable(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*Invocation)
		wantErr bool
	}{
		{"released with sig is attestable", func(i *Invocation) {}, false},
		{"escrowed refused", func(i *Invocation) { i.Status = "escrowed" }, true},
		{"acknowledged refused", func(i *Invocation) { i.Status = "acknowledged" }, true},
		{"refunded refused", func(i *Invocation) { i.Status = "refunded" }, true},
		{"disputed refused", func(i *Invocation) { i.Status = "disputed" }, true},
		{"completed (in buyer review) refused", func(i *Invocation) { i.Status = "completed" }, true},
		{"released without sig refused", func(i *Invocation) { i.CompletionSig = nil }, true},
		{"released with empty sig refused", func(i *Invocation) { i.CompletionSig = strPtr("") }, true},
		{"released without settled_at refused", func(i *Invocation) { i.SettledAt = nil }, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inv := settledInvocation()
			tc.mutate(inv)
			err := checkAttestable(inv)
			if (err != nil) != tc.wantErr {
				t.Fatalf("checkAttestable() error = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestContentHash — determinism and sensitivity
// ---------------------------------------------------------------------------

func TestContentHashDeterministic(t *testing.T) {
	h1, canon1, err := contentHash(settledInvocation())
	if err != nil {
		t.Fatal(err)
	}
	h2, canon2, err := contentHash(settledInvocation())
	if err != nil {
		t.Fatal(err)
	}
	if string(h1) != string(h2) {
		t.Fatal("same invocation produced different hashes")
	}
	if string(canon1) != string(canon2) {
		t.Fatal("same invocation produced different canonical bytes")
	}
	if len(h1) != 32 {
		t.Fatalf("expected sha256 (32 bytes), got %d", len(h1))
	}
}

func TestContentHashSensitive(t *testing.T) {
	base, _, _ := contentHash(settledInvocation())
	mutations := map[string]func(*Invocation){
		"amount":         func(i *Invocation) { i.Amount = 54 },
		"buyer":          func(i *Invocation) { i.BuyerDID = "did:at:other" },
		"completion_sig": func(i *Invocation) { i.CompletionSig = strPtr("b3RoZXI=") },
		"status":         func(i *Invocation) { i.Status = "refunded" },
	}
	for name, mutate := range mutations {
		inv := settledInvocation()
		mutate(inv)
		h, _, _ := contentHash(inv)
		if string(h) == string(base) {
			t.Errorf("mutating %s did not change the content hash", name)
		}
	}
}

// ---------------------------------------------------------------------------
// TestWatchState — ledger round-trip and idempotency guards
// ---------------------------------------------------------------------------

func TestWatchStateRoundTrip(t *testing.T) {
	path := t.TempDir() + "/state.json"

	st, err := loadState(path)
	if err != nil {
		t.Fatalf("loadState on missing file: %v", err)
	}
	if len(st.Attested) != 0 {
		t.Fatal("fresh state should be empty")
	}

	st.Attested["inv-1"] = &AttestRecord{TxHash: "ABC", AttestationID: "att-1-1", AttestedAt: "2026-07-05T22:00:00Z"}
	st.Attested["inv-2"] = &AttestRecord{Failures: 5, LastError: "boom"}
	if err := saveState(path, st); err != nil {
		t.Fatalf("saveState: %v", err)
	}

	st2, err := loadState(path)
	if err != nil {
		t.Fatalf("loadState round-trip: %v", err)
	}
	if got := st2.Attested["inv-1"]; got == nil || got.TxHash != "ABC" || got.AttestationID != "att-1-1" {
		t.Fatalf("inv-1 record corrupted: %+v", got)
	}
	if got := st2.Attested["inv-2"]; got == nil || got.Failures != 5 {
		t.Fatalf("inv-2 failure record corrupted: %+v", got)
	}
}

func TestLoadStateCorrupt(t *testing.T) {
	path := t.TempDir() + "/state.json"
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadState(path); err == nil {
		t.Fatal("corrupt state file must error, not silently reset (a reset ledger double-attests)")
	}
}

func TestSplitRoles(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"seller,buyer", 2},
		{"seller", 1},
		{" buyer ", 1},
		{"seller,arbiter", 1},
		{"", 0},
		{"nonsense", 0},
	}
	for _, tc := range cases {
		if got := len(splitRoles(tc.in)); got != tc.want {
			t.Errorf("splitRoles(%q) len = %d, want %d", tc.in, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// TestShouldSkipSubmit — the double-bond guard
// ---------------------------------------------------------------------------

func TestShouldSkipSubmit(t *testing.T) {
	cases := []struct {
		name string
		rec  *AttestRecord
		want bool
	}{
		{"unseen invocation submits", nil, false},
		{"submission_unknown NEVER resubmits (double bond)", &AttestRecord{TxHash: "ABC", Status: statusSubmissionUnknown}, true},
		{"attested skips", &AttestRecord{TxHash: "ABC", Status: statusAttested, AttestationID: "att-1-1"}, true},
		{"legacy attested (no status) skips", &AttestRecord{TxHash: "ABC", AttestationID: "att-1-1"}, true},
		{"parked skips", &AttestRecord{Failures: maxAttestFailures}, true},
		{"failed below threshold retries", &AttestRecord{Failures: maxAttestFailures - 1, LastError: "boom"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldSkipSubmit(tc.rec); got != tc.want {
				t.Fatalf("shouldSkipSubmit(%+v) = %v, want %v", tc.rec, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestDecideReconcile / TestApplyReconcile — the forward-only lifecycle
// ---------------------------------------------------------------------------

// reconcileNow is the fixed wall clock the reconcile tests decide against.
var reconcileNow = time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)

// submittedAgo renders a SubmittedAt timestamp d before reconcileNow.
func submittedAgo(d time.Duration) string {
	return reconcileNow.Add(-d).Format(time.RFC3339)
}

// notFoundProbe is one authoritative "the node says this tx is not found".
func notFoundProbe() txProbe { return txProbe{Outcome: probeNotFound} }

func TestDecideReconcile(t *testing.T) {
	const overWait = minReleaseWait + time.Minute
	const underWait = minReleaseWait - time.Minute
	cases := []struct {
		name           string
		probe          txProbe
		notFoundSoFar  int
		lastMissHeight uint64
		currentHeight  uint64
		elapsed        time.Duration
		want           reconcileAction
	}{
		{"found code 0 attests", txProbe{Outcome: probeFound, Code: 0, AttestationID: "att-9-1"}, 0, 0, 100, 0, reconcileMarkAttested},
		{"found code 0 attests even after misses", txProbe{Outcome: probeFound, Code: 0}, maxReconcileNotFound - 1, 90, 100, overWait, reconcileMarkAttested},
		{"found code !=0 records failure", txProbe{Outcome: probeFound, Code: 5, RawLog: "insufficient bond"}, 0, 0, 100, 0, reconcileRecordFailure},
		{"authoritative miss below threshold waits", notFoundProbe(), 0, 0, 100, overWait, reconcileKeepWaiting},
		{"authoritative miss one below threshold waits", notFoundProbe(), maxReconcileNotFound - 2, 90, 100, overWait, reconcileKeepWaiting},
		{"all gates open releases", notFoundProbe(), maxReconcileNotFound - 1, 99, 100, overWait, reconcileRelease},
		{"probe error is never evidence", txProbe{}, maxReconcileNotFound - 1, 90, 100, overWait, reconcileNoEvidence},
		{"stalled height is never evidence", notFoundProbe(), maxReconcileNotFound - 1, 100, 100, overWait, reconcileNoEvidence},
		{"height gone backward is never evidence", notFoundProbe(), maxReconcileNotFound - 1, 100, 99, overWait, reconcileNoEvidence},
		{"unknown height (query failed) is never evidence", notFoundProbe(), maxReconcileNotFound - 1, 0, 0, overWait, reconcileNoEvidence},
		{"elapsed under the window blocks release at threshold", notFoundProbe(), maxReconcileNotFound - 1, 99, 100, underWait, reconcileKeepWaiting},
		{"elapsed under the window blocks release past threshold", notFoundProbe(), maxReconcileNotFound + 3, 99, 100, underWait, reconcileKeepWaiting},
		{"release fires once the window elapses even past threshold", notFoundProbe(), maxReconcileNotFound + 3, 99, 100, overWait, reconcileRelease},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := decideReconcile(tc.probe, tc.notFoundSoFar, tc.lastMissHeight, tc.currentHeight, tc.elapsed)
			if got != tc.want {
				t.Fatalf("decideReconcile(%+v, %d, %d, %d, %s) = %v, want %v",
					tc.probe, tc.notFoundSoFar, tc.lastMissHeight, tc.currentHeight, tc.elapsed, got, tc.want)
			}
		})
	}
}

func TestClassifyQueryTxFailure(t *testing.T) {
	const hash = "5E4A31F2C0DE"
	cases := []struct {
		name   string
		stderr string
		txHash string
		want   probeOutcome
	}{
		{"cometbft not-found is authoritative", "Error: RPC error -32603 - Internal error: tx (5E4A31F2C0DE) not found", hash, probeNotFound},
		{"sdk cli not-found is authoritative", "Error: no transaction found with hash 5E4A31F2C0DE", hash, probeNotFound},
		{"case difference still authoritative", "error: TX (5e4a31f2c0de) NOT FOUND", hash, probeNotFound},
		{"connection refused is a probe error", `Error: post failed: Post "http://localhost:26657": dial tcp 127.0.0.1:26657: connect: connection refused`, hash, probeError},
		{"empty stderr is a probe error", "", hash, probeError},
		{"not-found naming a DIFFERENT tx is a probe error", "Error: RPC error -32603 - Internal error: tx (FFFF0000) not found", hash, probeError},
		{"indexing disabled is a probe error", "Error: transaction indexing is disabled", hash, probeError},
		{"empty hash can never be authoritative", "Error: tx () not found", "", probeError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyQueryTxFailure(tc.stderr, tc.txHash); got != tc.want {
				t.Fatalf("classifyQueryTxFailure(%q, %q) = %v, want %v", tc.stderr, tc.txHash, got, tc.want)
			}
		})
	}
}

func TestApplyReconcileMarkAttested(t *testing.T) {
	rec := &AttestRecord{TxHash: "ABC", Status: statusSubmissionUnknown, SubmittedAt: "2026-07-23T00:00:00Z", NotFound: 3, LastMissHeight: 90, LastError: "old"}
	action := applyReconcile(rec, txProbe{Outcome: probeFound, Code: 0, AttestationID: "att-9-1"}, 100, reconcileNow)
	if action != reconcileMarkAttested {
		t.Fatalf("action = %v, want reconcileMarkAttested", action)
	}
	if rec.Status != statusAttested || rec.AttestationID != "att-9-1" || rec.TxHash != "ABC" {
		t.Fatalf("attested record wrong: %+v", rec)
	}
	if rec.AttestedAt == "" || rec.LastError != "" || rec.NotFound != 0 || rec.LastMissHeight != 0 {
		t.Fatalf("attested bookkeeping wrong: %+v", rec)
	}
}

func TestApplyReconcileAttestedWithoutEvent(t *testing.T) {
	// code 0 means the bond is posted on-chain: the record must land attested
	// even when the attestation_id cannot be recovered from tx events —
	// releasing it for resubmission here would be the double bond.
	rec := &AttestRecord{TxHash: "ABC", Status: statusSubmissionUnknown}
	if action := applyReconcile(rec, txProbe{Outcome: probeFound, Code: 0}, 100, reconcileNow); action != reconcileMarkAttested {
		t.Fatalf("action = %v, want reconcileMarkAttested", action)
	}
	if rec.Status != statusAttested || rec.AttestationID != "" || rec.TxHash != "ABC" {
		t.Fatalf("record wrong: %+v", rec)
	}
}

func TestApplyReconcileRecordFailure(t *testing.T) {
	rec := &AttestRecord{TxHash: "ABC", Status: statusSubmissionUnknown, SubmittedAt: "2026-07-23T00:00:00Z", NotFound: 2, LastMissHeight: 90}
	action := applyReconcile(rec, txProbe{Outcome: probeFound, Code: 5, RawLog: "insufficient bond"}, 100, reconcileNow)
	if action != reconcileRecordFailure {
		t.Fatalf("action = %v, want reconcileRecordFailure", action)
	}
	if rec.TxHash != "" || rec.Status != "" || rec.SubmittedAt != "" || rec.NotFound != 0 || rec.LastMissHeight != 0 {
		t.Fatalf("failed record not released cleanly: %+v", rec)
	}
	if rec.Failures != 1 || !strings.Contains(rec.LastError, "code 5") {
		t.Fatalf("failure bookkeeping wrong: %+v", rec)
	}
}

func TestApplyReconcileNotFoundLifecycle(t *testing.T) {
	// A vanished tx stays submission_unknown for maxReconcileNotFound-1
	// authoritative misses (each with the chain advancing, window elapsed)
	// and is released for resubmission only on the last.
	rec := &AttestRecord{TxHash: "ABC", Status: statusSubmissionUnknown, SubmittedAt: submittedAgo(minReleaseWait + time.Minute)}
	height := uint64(100)
	for i := 1; i < maxReconcileNotFound; i++ {
		height++
		if action := applyReconcile(rec, notFoundProbe(), height, reconcileNow); action != reconcileKeepWaiting {
			t.Fatalf("miss %d: action = %v, want reconcileKeepWaiting", i, action)
		}
		if rec.Status != statusSubmissionUnknown || rec.TxHash != "ABC" || rec.NotFound != i || rec.LastMissHeight != height {
			t.Fatalf("miss %d: record wrong: %+v", i, rec)
		}
	}
	height++
	if action := applyReconcile(rec, notFoundProbe(), height, reconcileNow); action != reconcileRelease {
		t.Fatalf("final miss: action = %v, want reconcileRelease", action)
	}
	if rec.TxHash != "" || rec.Status != "" || rec.Failures != 1 || rec.NotFound != 0 || rec.LastMissHeight != 0 {
		t.Fatalf("released record wrong: %+v", rec)
	}
	if !strings.Contains(rec.LastError, "released for resubmission") {
		t.Fatalf("release reason not recorded: %+v", rec)
	}
}

func TestApplyReconcileProbeErrorCountsNothing(t *testing.T) {
	// Node down, connection refused, binary failure: the probe proved
	// nothing, so the record must not move — misses neither grow nor reset.
	rec := &AttestRecord{TxHash: "ABC", Status: statusSubmissionUnknown, SubmittedAt: submittedAgo(minReleaseWait + time.Hour), NotFound: 7, LastMissHeight: 90}
	before := *rec
	for i := 0; i < maxReconcileNotFound*2; i++ {
		if action := applyReconcile(rec, txProbe{}, 100+uint64(i), reconcileNow); action != reconcileNoEvidence {
			t.Fatalf("probe error %d: action = %v, want reconcileNoEvidence", i, action)
		}
	}
	if *rec != before {
		t.Fatalf("probe errors mutated the record: %+v → %+v", before, *rec)
	}
}

func TestApplyReconcileStalledChainCountsNothing(t *testing.T) {
	// Consensus halt: `query tx` cannot see the tx sitting in the mempool,
	// so an authoritative not-found while the height is not advancing is no
	// evidence — arbitrarily many such probes must never release the record.
	rec := &AttestRecord{TxHash: "ABC", Status: statusSubmissionUnknown, SubmittedAt: submittedAgo(minReleaseWait + time.Hour), NotFound: maxReconcileNotFound - 1, LastMissHeight: 100}
	before := *rec
	for i := 0; i < maxReconcileNotFound*2; i++ {
		if action := applyReconcile(rec, notFoundProbe(), 100, reconcileNow); action != reconcileNoEvidence {
			t.Fatalf("stalled probe %d: action = %v, want reconcileNoEvidence", i, action)
		}
	}
	if action := applyReconcile(rec, notFoundProbe(), 0, reconcileNow); action != reconcileNoEvidence {
		t.Fatal("unknown height (failed height query) must also count nothing")
	}
	if *rec != before {
		t.Fatalf("stalled-chain probes mutated the record: %+v → %+v", before, *rec)
	}
	// The moment the chain advances again, the same probe counts.
	if action := applyReconcile(rec, notFoundProbe(), 101, reconcileNow); action != reconcileRelease {
		t.Fatalf("advancing chain after the stall: action = %v, want reconcileRelease", action)
	}
}

func TestApplyReconcileElapsedGateBlocksEarlyRelease(t *testing.T) {
	// Ten authoritative misses inside the first minutes (fast RELAY_INTERVAL)
	// must NOT release: the wall-clock window since SubmittedAt rules.
	rec := &AttestRecord{TxHash: "ABC", Status: statusSubmissionUnknown, SubmittedAt: submittedAgo(5 * time.Minute)}
	height := uint64(100)
	for i := 1; i <= maxReconcileNotFound+2; i++ {
		height++
		if action := applyReconcile(rec, notFoundProbe(), height, reconcileNow); action != reconcileKeepWaiting {
			t.Fatalf("early miss %d: action = %v, want reconcileKeepWaiting (elapsed gate)", i, action)
		}
	}
	if rec.Status != statusSubmissionUnknown || rec.TxHash != "ABC" || rec.NotFound != maxReconcileNotFound+2 {
		t.Fatalf("record wrong under the elapsed gate: %+v", rec)
	}
	// Same record, once the window has truly elapsed: the next counted miss
	// releases.
	rec.SubmittedAt = submittedAgo(minReleaseWait)
	height++
	if action := applyReconcile(rec, notFoundProbe(), height, reconcileNow); action != reconcileRelease {
		t.Fatalf("post-window miss: action = %v, want reconcileRelease", action)
	}
}

func TestApplyReconcileUnknownSubmittedAtNeverReleases(t *testing.T) {
	// No parseable SubmittedAt → elapsed reads as zero → release is blocked.
	// Safe direction: never resubmit on unknown timing.
	rec := &AttestRecord{TxHash: "ABC", Status: statusSubmissionUnknown, NotFound: maxReconcileNotFound - 1, LastMissHeight: 90}
	if action := applyReconcile(rec, notFoundProbe(), 100, reconcileNow); action != reconcileKeepWaiting {
		t.Fatalf("action = %v, want reconcileKeepWaiting", action)
	}
}

func TestApplyReconcileFoundResetsMissCount(t *testing.T) {
	// "N consecutive" means a found probe resets the count: a failure release
	// followed by a new submission must start counting from zero again.
	rec := &AttestRecord{TxHash: "ABC", Status: statusSubmissionUnknown, NotFound: maxReconcileNotFound - 1, LastMissHeight: 90}
	applyReconcile(rec, txProbe{Outcome: probeFound, Code: 7, RawLog: "seq mismatch"}, 100, reconcileNow)
	if rec.NotFound != 0 || rec.LastMissHeight != 0 {
		t.Fatalf("found probe must reset the consecutive-miss count: %+v", rec)
	}
}

// ---------------------------------------------------------------------------
// TestLoadStateLegacy — pre-lifecycle state files must load unchanged
// ---------------------------------------------------------------------------

func TestLoadStateLegacy(t *testing.T) {
	path := t.TempDir() + "/state.json"
	legacy := `{
  "attested": {
    "inv-old": {"tx_hash": "OLD", "attestation_id": "att-2419-2", "attested_at": "2026-07-05T22:00:00Z"},
    "inv-parked": {"failures": 5, "last_error": "boom"}
  }
}`
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := loadState(path)
	if err != nil {
		t.Fatalf("legacy state must load: %v", err)
	}
	old := st.Attested["inv-old"]
	if old == nil || old.TxHash != "OLD" || old.Status != "" {
		t.Fatalf("legacy attested record corrupted: %+v", old)
	}
	if !shouldSkipSubmit(old) {
		t.Fatal("legacy attested record must never resubmit")
	}
	if old.Status == statusSubmissionUnknown {
		t.Fatal("legacy record must not be mistaken for an in-flight broadcast")
	}
	if !shouldSkipSubmit(st.Attested["inv-parked"]) {
		t.Fatal("legacy parked record must stay parked")
	}
}

func TestWatchStateLifecycleRoundTrip(t *testing.T) {
	path := t.TempDir() + "/state.json"
	st := &WatchState{Attested: map[string]*AttestRecord{
		"inv-flight": {TxHash: "ABC", Status: statusSubmissionUnknown, SubmittedAt: "2026-07-23T09:00:00Z", NotFound: 4, LastMissHeight: 66300},
	}}
	if err := saveState(path, st); err != nil {
		t.Fatal(err)
	}
	st2, err := loadState(path)
	if err != nil {
		t.Fatal(err)
	}
	got := st2.Attested["inv-flight"]
	if got == nil || got.Status != statusSubmissionUnknown || got.TxHash != "ABC" || got.NotFound != 4 || got.SubmittedAt == "" {
		t.Fatalf("in-flight record did not survive the round trip: %+v", got)
	}
	if got.LastMissHeight != 66300 {
		t.Fatalf("last counted-miss height did not survive the round trip: %+v", got)
	}
}

// ---------------------------------------------------------------------------
// TestAlertAuthExpired — sentinel file + once-per-hour rate limit
// ---------------------------------------------------------------------------

func TestAlertAuthExpired(t *testing.T) {
	path := t.TempDir() + "/alerts/RELAY-AUTH-ALERT"
	if !alertAuthExpired(path, "https://api.example/v1/invocations?role=seller") {
		t.Fatal("first 401 must alert")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("sentinel file not written: %v", err)
	}
	if !strings.Contains(string(data), "RELAY-AUTH-EXPIRED") {
		t.Fatalf("sentinel missing the distinctive marker: %s", data)
	}
	if alertAuthExpired(path, "https://api.example/v1/invocations?role=seller") {
		t.Fatal("second 401 within the hour must be rate-limited")
	}
}

// ---------------------------------------------------------------------------
// TestWitnessWriteback — flag-gated, best-effort, correct wire shape
// ---------------------------------------------------------------------------

func TestWitnessWritebackDisabledByDefault(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("writeback must not fire when the flag is off")
	}))
	defer srv.Close()
	cfg := Config{API: srv.URL, APIKey: "k", ChainID: "zerone-1", Adapter: "agenttool-invocation-v1"}
	witnessWriteback(cfg, "inv-1", &AttestRecord{TxHash: "ABC", AttestationID: "att-9-1"})
}

func TestWitnessWritebackWireShape(t *testing.T) {
	var gotMethod, gotPath, gotAuth string
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	cfg := Config{API: srv.URL, APIKey: "secret-key", ChainID: "zerone-1", Adapter: "agenttool-invocation-v1", WitnessWriteback: true}
	rec := &AttestRecord{TxHash: "ABC123", AttestationID: "att-9-1", Status: statusAttested}
	witnessWriteback(cfg, "inv-1", rec)
	if gotMethod != http.MethodPost || gotPath != "/v1/invocations/inv-1/witness" {
		t.Fatalf("wrong route: %s %s", gotMethod, gotPath)
	}
	if gotAuth != "Bearer secret-key" {
		t.Fatalf("wrong auth: %q", gotAuth)
	}
	want := map[string]string{"chain_id": "zerone-1", "tx_hash": "ABC123", "attestation_id": "att-9-1", "adapter_id": "agenttool-invocation-v1"}
	for k, v := range want {
		if gotBody[k] != v {
			t.Fatalf("body[%s] = %q, want %q (full: %v)", k, gotBody[k], v, gotBody)
		}
	}
}

func TestWitnessWritebackFailureLeavesRecordAlone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound) // route not deployed on live yet
	}))
	defer srv.Close()
	cfg := Config{API: srv.URL, APIKey: "k", ChainID: "zerone-1", Adapter: "agenttool-invocation-v1", WitnessWriteback: true}
	rec := &AttestRecord{TxHash: "ABC", AttestationID: "att-9-1", Status: statusAttested, AttestedAt: "2026-07-23T09:00:00Z"}
	before := *rec
	witnessWriteback(cfg, "inv-1", rec)
	if *rec != before {
		t.Fatalf("writeback failure mutated the attested record: %+v → %+v", before, rec)
	}
}

// ---------------------------------------------------------------------------
// TestBuildLinkRoundTrip — the load-bearing contract with the CLI
// ---------------------------------------------------------------------------

// The submit-attestation command reads the link file with encoding/json into
// types.SubstrateLink (client/cli/tx.go readJSONFile). This pins that the
// relay's output survives that exact decode with every field intact.
func TestBuildLinkRoundTrip(t *testing.T) {
	cfg := loadConfig()
	inv := settledInvocation()
	linkBytes, err := buildLink(cfg, inv, 2222)
	if err != nil {
		t.Fatal(err)
	}

	var link types.SubstrateLink
	if err := json.Unmarshal(linkBytes, &link); err != nil {
		t.Fatalf("link JSON does not decode into types.SubstrateLink: %v", err)
	}
	if link.Source == nil {
		t.Fatal("decoded link has nil source")
	}
	if link.Source.SourceId != inv.ID {
		t.Errorf("source_id = %q, want %q", link.Source.SourceId, inv.ID)
	}
	if link.Source.FetchedAtBlock != 2222 {
		t.Errorf("fetched_at_block = %d, want 2222", link.Source.FetchedAtBlock)
	}
	wantHash, _, _ := contentHash(inv)
	if base64.StdEncoding.EncodeToString(wantHash) != base64.StdEncoding.EncodeToString(link.Source.ContentHash) {
		t.Error("decoded content_hash does not match recomputed canonical hash")
	}
	if len(link.CitedFacts) != 0 || len(link.PendingClaims) != 0 {
		t.Error("witness-only link must carry no cited facts or pending claims")
	}
}
