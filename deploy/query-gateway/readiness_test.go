package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func statusBody(chainID, height string, catchingUp any, blockTime string) string {
	return statusBodyWithHash(chainID, height, strings.Repeat("1", 64), catchingUp, blockTime)
}

func statusBodyWithHash(chainID, height, blockHash string, catchingUp any, blockTime string) string {
	return fmt.Sprintf(`{"result":{"node_info":{"network":%q},"sync_info":{"latest_block_height":%q,"latest_block_hash":%q,"latest_block_time":%q,"catching_up":%s}}}`,
		chainID, height, blockHash, blockTime, mustJSON(catchingUp))
}

func mustJSON(value any) string {
	b, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func probeConfig(serverURL string, active bool) config {
	cfg := config{
		expectedChainID: "zerone-2",
		upstreamHost:    "origin.internal",
		statusURL:       serverURL,
		active:          active,
		maxBlockAge:     activeMaxBlockAge,
		maxFutureSkew:   activeMaxFutureSkew,
		maxResponseSize: maxStatusBytes,
		resolver: resolverFunc(func(context.Context, string) ([]net.IPAddr, error) {
			return []net.IPAddr{{IP: net.ParseIP("fdaa::1")}}, nil
		}),
	}
	if !active {
		cfg.expectedArchiveHeight = 7
		cfg.expectedArchiveBlockHash = strings.Repeat("1", 64)
	}
	return cfg
}

func probeBody(t *testing.T, body string, active bool, now time.Time) (probeResult, error) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()
	return probeStatus(context.Background(), server.Client(), probeConfig(server.URL, active), now)
}

func TestProbeStatusAcceptsFreshActiveOrigin(t *testing.T) {
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	result, err := probeBody(t, statusBody("zerone-2", "42", false, now.Add(-time.Second).Format(time.RFC3339Nano)), true, now)
	if err != nil {
		t.Fatalf("probeStatus() error = %v", err)
	}
	if result.chainID != "zerone-2" || result.height != 42 {
		t.Fatalf("probeStatus() = %+v", result)
	}
}

func TestConfigFromEnvironmentAllowsOnlyApprovedRoleAndOrigin(t *testing.T) {
	t.Setenv("GATEWAY_ROLE", "zerone-2-query")
	t.Setenv("EXPECTED_CHAIN_ID", "zerone-2")
	t.Setenv("UPSTREAM_HOST", "zerone-2-edge.internal")
	t.Setenv("EXPECTED_ARCHIVE_HEIGHT", "")
	t.Setenv("EXPECTED_ARCHIVE_APP_HASH", "")
	t.Setenv("EXPECTED_ARCHIVE_BLOCK_HASH", "")
	cfg, err := configFromEnvironment()
	if err != nil {
		t.Fatalf("configFromEnvironment() error = %v", err)
	}
	if !cfg.active || !cfg.bindResolvedAddress || cfg.statusURL != "http://zerone-2-edge.internal:26657/status" {
		t.Fatalf("configFromEnvironment() = %+v", cfg)
	}

	t.Setenv("EXPECTED_CHAIN_ID", "zerone-1")
	if _, err := configFromEnvironment(); err == nil || !strings.Contains(err.Error(), "approved pair") {
		t.Fatalf("role/chain mismatch error = %v", err)
	}
	t.Setenv("EXPECTED_CHAIN_ID", "zerone-2")
	t.Setenv("UPSTREAM_HOST", "edge.internal;invalid")
	if _, err := configFromEnvironment(); err == nil || !strings.Contains(err.Error(), ".internal") {
		t.Fatalf("unsafe host error = %v", err)
	}

	t.Setenv("GATEWAY_ROLE", "zerone-1-archive-query")
	t.Setenv("EXPECTED_CHAIN_ID", "zerone-1")
	t.Setenv("UPSTREAM_HOST", "zerone-1-archive.internal")
	t.Setenv("EXPECTED_ARCHIVE_HEIGHT", "42")
	t.Setenv("EXPECTED_ARCHIVE_APP_HASH", strings.Repeat("ab", 32))
	t.Setenv("EXPECTED_ARCHIVE_BLOCK_HASH", strings.Repeat("cd", 32))
	cfg, err = configFromEnvironment()
	if err != nil {
		t.Fatalf("archive configFromEnvironment() error = %v", err)
	}
	if cfg.active || cfg.expectedArchiveHeight != 42 || !strings.HasSuffix(cfg.archiveNextBlockURL, "height=43") {
		t.Fatalf("archive configFromEnvironment() = %+v", cfg)
	}
	t.Setenv("EXPECTED_ARCHIVE_BLOCK_HASH", "")
	if _, err := configFromEnvironment(); err == nil || !strings.Contains(err.Error(), "BLOCK_HASH") {
		t.Fatalf("missing archive block hash error = %v", err)
	}
	t.Setenv("EXPECTED_ARCHIVE_BLOCK_HASH", strings.Repeat("cd", 32))

	t.Setenv("EXPECTED_ARCHIVE_APP_HASH", strings.Repeat("AB", 32))
	if _, err := configFromEnvironment(); err == nil || !strings.Contains(err.Error(), "lowercase") {
		t.Fatalf("uppercase archive hash error = %v", err)
	}
	t.Setenv("EXPECTED_ARCHIVE_APP_HASH", strings.Repeat("ab", 32))
	t.Setenv("EXPECTED_ARCHIVE_BLOCK_HASH", strings.Repeat("CD", 32))
	if _, err := configFromEnvironment(); err == nil || !strings.Contains(err.Error(), "BLOCK_HASH") {
		t.Fatalf("uppercase archive block hash error = %v", err)
	}
	t.Setenv("EXPECTED_ARCHIVE_BLOCK_HASH", strings.Repeat("cd", 32))
	t.Setenv("EXPECTED_ARCHIVE_HEIGHT", "18446744073709551615")
	if _, err := configFromEnvironment(); err == nil || !strings.Contains(err.Error(), "A+1") {
		t.Fatalf("maximum archive height error = %v", err)
	}
}

type resolverFunc func(context.Context, string) ([]net.IPAddr, error)

func (resolve resolverFunc) LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error) {
	return resolve(ctx, host)
}

func TestRequireSingle6PN(t *testing.T) {
	tests := []struct {
		name      string
		addresses []net.IPAddr
		err       error
		want      string
	}{
		{name: "one 6PN", addresses: []net.IPAddr{{IP: net.ParseIP("fdaa::42")}}},
		{name: "duplicate 6PN", addresses: []net.IPAddr{{IP: net.ParseIP("fdaa::42")}, {IP: net.ParseIP("fdaa::42")}}},
		{name: "no addresses", want: "exactly one"},
		{name: "two Machines", addresses: []net.IPAddr{{IP: net.ParseIP("fdaa::42")}, {IP: net.ParseIP("fdaa::43")}}, want: "exactly one"},
		{name: "IPv4", addresses: []net.IPAddr{{IP: net.ParseIP("192.0.2.1")}}, want: "non-6PN"},
		{name: "other IPv6", addresses: []net.IPAddr{{IP: net.ParseIP("fd00::1")}}, want: "non-6PN"},
		{name: "scoped IPv6", addresses: []net.IPAddr{{IP: net.ParseIP("fdaa::42"), Zone: "eth0"}}, want: "non-6PN"},
		{name: "lookup failure", err: errorsForTest("DNS unavailable"), want: "DNS unavailable"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resolver := resolverFunc(func(context.Context, string) ([]net.IPAddr, error) {
				return test.addresses, test.err
			})
			address, err := requireSingle6PN(context.Background(), resolver, "origin.internal")
			if test.want == "" {
				if err != nil {
					t.Fatalf("requireSingle6PN() error = %v", err)
				}
				if address.String() != "fdaa::42" {
					t.Fatalf("requireSingle6PN() address = %v", address)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("requireSingle6PN() error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestProbeOriginRejectsMultipleMachinesBeforeHTTP(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests++
	}))
	defer server.Close()

	cfg := probeConfig(server.URL, true)
	cfg.resolver = resolverFunc(func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("fdaa::42")}, {IP: net.ParseIP("fdaa::43")}}, nil
	})
	_, err := probeOrigin(context.Background(), server.Client(), cfg, time.Now())
	if err == nil || !strings.Contains(err.Error(), "exactly one") {
		t.Fatalf("probeOrigin() DNS error = %v", err)
	}
	if requests != 0 {
		t.Fatalf("origin received %d requests before DNS topology rejection", requests)
	}
}

func TestBoundOriginClientUsesValidatedAddressOnly(t *testing.T) {
	listener, err := net.Listen("tcp6", "[::1]:0")
	if err != nil {
		t.Skipf("IPv6 loopback unavailable: %v", err)
	}
	wantHost := ""
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != wantHost {
			t.Errorf("origin Host header = %q", r.Host)
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	server.Listener = listener
	server.Start()
	defer server.Close()

	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("split listener address: %v", err)
	}
	wantHost = "origin.internal:" + port
	client := newHTTPClientForOrigin("origin.internal", "::1")
	defer client.CloseIdleConnections()
	response, err := client.Get("http://" + wantHost + "/status")
	if err != nil {
		t.Fatalf("bound origin request: %v", err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("bound origin status = %d", response.StatusCode)
	}

	unexpectedResponse, err := client.Get("http://other.internal:" + port + "/status")
	if unexpectedResponse != nil {
		_ = unexpectedResponse.Body.Close()
	}
	if err == nil || !strings.Contains(err.Error(), "unexpected host") {
		t.Fatalf("unexpected-host dial error = %v", err)
	}
}

func TestProbeStatusRejectsUnsafeActiveResponses(t *testing.T) {
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	fresh := now.Add(-time.Second).Format(time.RFC3339Nano)
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "wrong chain", body: statusBody("zerone-1", "42", false, fresh), want: "chain ID"},
		{name: "zero height", body: statusBody("zerone-2", "0", false, fresh), want: "positive integer"},
		{name: "leading zero height", body: statusBody("zerone-2", "042", false, fresh), want: "positive integer"},
		{name: "negative height", body: statusBody("zerone-2", "-1", false, fresh), want: "positive integer"},
		{name: "overflow height", body: statusBody("zerone-2", "18446744073709551616", false, fresh), want: "uint64"},
		{name: "above Comet int64 height", body: statusBody("zerone-2", "9223372036854775808", false, fresh), want: "Comet int64"},
		{name: "catching up", body: statusBody("zerone-2", "42", true, fresh), want: "catching up"},
		{name: "missing catching up", body: `{"result":{"node_info":{"network":"zerone-2"},"sync_info":{"latest_block_height":"42","latest_block_time":"` + fresh + `"}}}`, want: "omits catching_up"},
		{name: "stale block", body: statusBody("zerone-2", "42", false, now.Add(-activeMaxBlockAge-time.Nanosecond).Format(time.RFC3339Nano)), want: "stale"},
		{name: "future block", body: statusBody("zerone-2", "42", false, now.Add(activeMaxFutureSkew+time.Nanosecond).Format(time.RFC3339Nano)), want: "future"},
		{name: "malformed time", body: statusBody("zerone-2", "42", false, "yesterday"), want: "malformed"},
		{name: "malformed JSON", body: `{`, want: "decode"},
		{name: "multiple JSON values", body: statusBody("zerone-2", "42", false, fresh) + `{}`, want: "decode"},
		{name: "missing result", body: `{}`, want: "omits result"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := probeBody(t, test.body, true, now)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("probeStatus() error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestProbeStatusRejectsOversizedAndHTTPErrorResponses(t *testing.T) {
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oversized" {
			_, _ = w.Write([]byte(strings.Repeat("x", maxStatusBytes+1)))
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	cfg := probeConfig(server.URL+"/oversized", true)
	if _, err := probeStatus(context.Background(), server.Client(), cfg, now); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("oversized probe error = %v", err)
	}
	cfg.statusURL = server.URL + "/unavailable"
	if _, err := probeStatus(context.Background(), server.Client(), cfg, now); err == nil || !strings.Contains(err.Error(), "HTTP 503") {
		t.Fatalf("HTTP failure probe error = %v", err)
	}
}

func TestProbeStatusRejectsRedirect(t *testing.T) {
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/elsewhere", http.StatusFound)
	}))
	defer server.Close()
	if _, err := probeStatus(context.Background(), newHTTPClient(), probeConfig(server.URL, true), now); err == nil || !strings.Contains(err.Error(), "redirects") {
		t.Fatalf("redirect probe error = %v", err)
	}
}

func TestProbeStatusAllowsStaleCatchingArchive(t *testing.T) {
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	result, err := probeBody(t, statusBody("zerone-2", "7", true, now.Add(-365*24*time.Hour).Format(time.RFC3339Nano)), false, now)
	if err != nil {
		t.Fatalf("archive probe error = %v", err)
	}
	if result.height != 7 {
		t.Fatalf("archive height = %d", result.height)
	}
}

func TestProbeStatusRejectsStructurallyInvalidArchiveStatus(t *testing.T) {
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name string
		body string
		want string
	}{
		{name: "missing catching up", body: `{"result":{"node_info":{"network":"zerone-2"},"sync_info":{"latest_block_height":"7","latest_block_time":"2025-09-03T12:00:00Z"}}}`, want: "omits catching_up"},
		{name: "malformed time", body: statusBody("zerone-2", "7", true, "yesterday"), want: "malformed"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := probeBody(t, test.body, false, now)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("archive probe error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestProbeOriginRequiresBoundedConsistentREST(t *testing.T) {
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		restStatus int
		restBody   string
		want       string
	}{
		{name: "healthy", restStatus: http.StatusOK, restBody: `{"syncing":false}`},
		{name: "REST down", restStatus: http.StatusServiceUnavailable, restBody: `{}`, want: "HTTP 503"},
		{name: "malformed REST", restStatus: http.StatusOK, restBody: `{`, want: "decode REST"},
		{name: "oversized REST", restStatus: http.StatusOK, restBody: strings.Repeat("x", maxStatusBytes+1), want: "exceeds"},
		{name: "missing boolean", restStatus: http.StatusOK, restBody: `{}`, want: "omits syncing"},
		{name: "disagrees", restStatus: http.StatusOK, restBody: `{"syncing":true}`, want: "disagrees"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/status":
					_, _ = w.Write([]byte(statusBody("zerone-2", "42", false, now.Add(-time.Second).Format(time.RFC3339Nano))))
				case "/syncing":
					w.WriteHeader(test.restStatus)
					_, _ = w.Write([]byte(test.restBody))
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			cfg := probeConfig(server.URL+"/status", true)
			cfg.restSyncingURL = server.URL + "/syncing"
			result, err := probeOrigin(context.Background(), server.Client(), cfg, now)
			if test.want == "" {
				if err != nil || result.height != 42 || result.validatedOrigin6PN != "fdaa::1" {
					t.Fatalf("probeOrigin() = %+v, %v", result, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("probeOrigin() error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestProbeOriginHonorsOneDeadlineAcrossProbes(t *testing.T) {
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/status":
			_, _ = w.Write([]byte(statusBody("zerone-2", "42", false, now.Format(time.RFC3339Nano))))
		case "/syncing":
			<-r.Context().Done()
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	cfg := probeConfig(server.URL+"/status", true)
	cfg.restSyncingURL = server.URL + "/syncing"
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, err := probeOrigin(ctx, server.Client(), cfg, now)
	if err == nil || !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Fatalf("deadline probe error = %v", err)
	}
	if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
		t.Fatalf("shared deadline took %s", elapsed)
	}
}

func TestProbeOriginBindsExactArchiveCheckpoint(t *testing.T) {
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	expectedHash := strings.Repeat("ab", 32)
	expectedBlockHash := strings.Repeat("cd", 32)
	base64Hash := base64.StdEncoding.EncodeToString(bytesForTest(0xab, 32))
	missingAPlusOne := `{"jsonrpc":"2.0","id":-1,"error":{"code":-32603,"message":"Internal error","data":"height 43 must be less than or equal to the current blockchain height 42"}}`
	tests := []struct {
		name            string
		statusHeight    string
		catchingUp      bool
		restSyncing     bool
		abciHeight      string
		abciHash        string
		statusBlockHash string
		omitStatusHash  bool
		blockAHash      string
		blockAChainID   string
		blockAHeight    string
		omitBlockA      bool
		nextBlockStatus int
		nextBlock       string
		want            string
	}{
		{name: "hex hash", statusHeight: "42", catchingUp: true, restSyncing: true, abciHeight: "42", abciHash: strings.ToUpper(expectedHash), nextBlockStatus: http.StatusInternalServerError, nextBlock: missingAPlusOne},
		{name: "base64 hash", statusHeight: "42", catchingUp: true, restSyncing: true, abciHeight: "42", abciHash: base64Hash, nextBlockStatus: http.StatusInternalServerError, nextBlock: missingAPlusOne},
		{name: "wrong status height", statusHeight: "41", catchingUp: true, restSyncing: true, abciHeight: "42", abciHash: expectedHash, nextBlockStatus: http.StatusInternalServerError, nextBlock: missingAPlusOne, want: "status height"},
		{name: "archive not catching", statusHeight: "42", catchingUp: false, restSyncing: false, abciHeight: "42", abciHash: expectedHash, nextBlockStatus: http.StatusInternalServerError, nextBlock: missingAPlusOne, want: "not reporting catching_up"},
		{name: "REST disagrees", statusHeight: "42", catchingUp: true, restSyncing: false, abciHeight: "42", abciHash: expectedHash, nextBlockStatus: http.StatusInternalServerError, nextBlock: missingAPlusOne, want: "disagrees"},
		{name: "wrong ABCI height", statusHeight: "42", catchingUp: true, restSyncing: true, abciHeight: "41", abciHash: expectedHash, nextBlockStatus: http.StatusInternalServerError, nextBlock: missingAPlusOne, want: "ABCI height"},
		{name: "wrong app hash", statusHeight: "42", catchingUp: true, restSyncing: true, abciHeight: "42", abciHash: strings.Repeat("cd", 32), nextBlockStatus: http.StatusInternalServerError, nextBlock: missingAPlusOne, want: "app hash does not match"},
		{name: "malformed app hash", statusHeight: "42", catchingUp: true, restSyncing: true, abciHeight: "42", abciHash: "not-a-hash", nextBlockStatus: http.StatusInternalServerError, nextBlock: missingAPlusOne, want: "app hash is malformed"},
		{name: "oversized ABCI", statusHeight: "42", catchingUp: true, restSyncing: true, abciHeight: "42", abciHash: strings.Repeat("x", maxStatusBytes+1), nextBlockStatus: http.StatusInternalServerError, nextBlock: missingAPlusOne, want: "exceeds"},
		{name: "missing status block hash", statusHeight: "42", catchingUp: true, restSyncing: true, abciHeight: "42", abciHash: expectedHash, omitStatusHash: true, nextBlockStatus: http.StatusInternalServerError, nextBlock: missingAPlusOne, want: "status block hash is malformed"},
		{name: "wrong status block hash", statusHeight: "42", catchingUp: true, restSyncing: true, abciHeight: "42", abciHash: expectedHash, statusBlockHash: strings.Repeat("de", 32), nextBlockStatus: http.StatusInternalServerError, nextBlock: missingAPlusOne, want: "status block hash does not match"},
		{name: "missing A block", statusHeight: "42", catchingUp: true, restSyncing: true, abciHeight: "42", abciHash: expectedHash, omitBlockA: true, nextBlockStatus: http.StatusInternalServerError, nextBlock: missingAPlusOne, want: "omits identity"},
		{name: "wrong A block chain", statusHeight: "42", catchingUp: true, restSyncing: true, abciHeight: "42", abciHash: expectedHash, blockAChainID: "zerone-2", nextBlockStatus: http.StatusInternalServerError, nextBlock: missingAPlusOne, want: "A block identity"},
		{name: "wrong A block height", statusHeight: "42", catchingUp: true, restSyncing: true, abciHeight: "42", abciHash: expectedHash, blockAHeight: "41", nextBlockStatus: http.StatusInternalServerError, nextBlock: missingAPlusOne, want: "A block identity"},
		{name: "wrong A block hash", statusHeight: "42", catchingUp: true, restSyncing: true, abciHeight: "42", abciHash: expectedHash, blockAHash: strings.Repeat("de", 32), nextBlockStatus: http.StatusInternalServerError, nextBlock: missingAPlusOne, want: "A block hash does not match"},
		{name: "A plus one exists", statusHeight: "42", catchingUp: true, restSyncing: true, abciHeight: "42", abciHash: expectedHash, nextBlockStatus: http.StatusOK, nextBlock: `{"result":{"block":{"header":{"height":"43"}}}}`, want: "expected 500"},
		{name: "A plus one ambiguous", statusHeight: "42", catchingUp: true, restSyncing: true, abciHeight: "42", abciHash: expectedHash, nextBlockStatus: http.StatusInternalServerError, nextBlock: `{"jsonrpc":"2.0","id":-1,"result":null}`, want: "exposes A+1"},
		{name: "wrong A plus one error", statusHeight: "42", catchingUp: true, restSyncing: true, abciHeight: "42", abciHash: expectedHash, nextBlockStatus: http.StatusInternalServerError, nextBlock: `{"jsonrpc":"2.0","id":-1,"error":{"code":-32603,"message":"Internal error","data":"different"}}`, want: "exposes A+1"},
		{name: "oversized A plus one error", statusHeight: "42", catchingUp: true, restSyncing: true, abciHeight: "42", abciHash: expectedHash, nextBlockStatus: http.StatusInternalServerError, nextBlock: strings.Repeat("x", maxStatusBytes+1), want: "exceeds"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			statusBlockHash := expectedBlockHash
			if test.statusBlockHash != "" {
				statusBlockHash = test.statusBlockHash
			}
			if test.omitStatusHash {
				statusBlockHash = ""
			}
			blockAHash := expectedBlockHash
			if test.blockAHash != "" {
				blockAHash = test.blockAHash
			}
			blockAChainID := "zerone-1"
			if test.blockAChainID != "" {
				blockAChainID = test.blockAChainID
			}
			blockAHeight := "42"
			if test.blockAHeight != "" {
				blockAHeight = test.blockAHeight
			}
			requestedBlocks := make(map[string]int)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/status":
					_, _ = w.Write([]byte(statusBodyWithHash("zerone-1", test.statusHeight, statusBlockHash, test.catchingUp, now.Add(-365*24*time.Hour).Format(time.RFC3339Nano))))
				case "/syncing":
					_, _ = fmt.Fprintf(w, `{"syncing":%t}`, test.restSyncing)
				case "/abci_info":
					_, _ = fmt.Fprintf(w, `{"result":{"response":{"last_block_height":%q,"last_block_app_hash":%q}}}`, test.abciHeight, test.abciHash)
				case "/block":
					height := r.URL.Query().Get("height")
					requestedBlocks[height]++
					switch height {
					case "42":
						if test.omitBlockA {
							_, _ = w.Write([]byte(`{"result":{}}`))
							return
						}
						_, _ = fmt.Fprintf(w, `{"result":{"block_id":{"hash":%q},"block":{"header":{"chain_id":%q,"height":%q}}}}`, strings.ToUpper(blockAHash), blockAChainID, blockAHeight)
					case "43":
						w.WriteHeader(test.nextBlockStatus)
						_, _ = w.Write([]byte(test.nextBlock))
					default:
						t.Errorf("block height = %q", r.URL.Query().Get("height"))
						http.NotFound(w, r)
					}
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()

			cfg := probeConfig(server.URL+"/status", false)
			cfg.expectedChainID = "zerone-1"
			cfg.expectedArchiveHeight = 42
			cfg.expectedArchiveAppHash = expectedHash
			cfg.expectedArchiveBlockHash = expectedBlockHash
			cfg.restSyncingURL = server.URL + "/syncing"
			cfg.archiveABCIInfoURL = server.URL + "/abci_info"
			cfg.archiveBlockURL = server.URL + "/block?height=42"
			cfg.archiveNextBlockURL = server.URL + "/block?height=43"
			result, err := probeOrigin(context.Background(), server.Client(), cfg, now)
			if test.want == "" {
				if err != nil || result.height != 42 || !result.catchingUp ||
					result.validatedOrigin6PN != "fdaa::1" {
					t.Fatalf("archive probeOrigin() = %+v, %v", result, err)
				}
				if requestedBlocks["42"] != 1 || requestedBlocks["43"] != 1 {
					t.Fatalf("archive block requests = %+v, want one request each for A and A+1", requestedBlocks)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("archive probeOrigin() error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func bytesForTest(value byte, count int) []byte {
	result := make([]byte, count)
	for i := range result {
		result[i] = value
	}
	return result
}

func TestReadinessHTTPBehavior(t *testing.T) {
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	state := &readinessState{}
	state.store(probeResult{
		chainID:            "zerone-2",
		height:             42,
		validatedOrigin6PN: "fdaa::42",
	}, nil, now)
	handler := readinessHandler{state: state, now: func() time.Time { return now }}

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantBody   string
	}{
		{name: "auth ready", method: http.MethodGet, path: "/ready", wantStatus: http.StatusNoContent},
		{name: "health ready", method: http.MethodGet, path: "/health", wantStatus: http.StatusOK, wantBody: `{"ready":true,"chain_id":"zerone-2","latest_block_height":"42"}`},
		{name: "head health", method: http.MethodHead, path: "/health", wantStatus: http.StatusOK},
		{name: "method closed", method: http.MethodPost, path: "/health", wantStatus: http.StatusMethodNotAllowed},
		{name: "unknown path", method: http.MethodGet, path: "/unknown", wantStatus: http.StatusNotFound},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(test.method, test.path, nil))
			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
			if test.wantBody != "" && strings.TrimSpace(recorder.Body.String()) != test.wantBody {
				t.Fatalf("body = %q, want %q", recorder.Body.String(), test.wantBody)
			}
			if test.method == http.MethodHead && recorder.Body.Len() != 0 {
				t.Fatalf("HEAD body = %q", recorder.Body.String())
			}
			if test.path == "/ready" && recorder.Header().Get("X-Zerone-Validated-6PN") != "fdaa::42" {
				t.Fatalf("ready validated origin = %q", recorder.Header().Get("X-Zerone-Validated-6PN"))
			}
		})
	}
}

func TestReadinessFailsClosedOnProbeErrorAndExpiredCache(t *testing.T) {
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		setup func(*readinessState)
	}{
		{name: "never probed", setup: func(*readinessState) {}},
		{name: "probe error", setup: func(state *readinessState) {
			state.store(probeResult{}, errorsForTest("wrong chain"), now)
		}},
		{name: "expired success", setup: func(state *readinessState) {
			state.store(probeResult{
				chainID:            "zerone-2",
				height:             42,
				validatedOrigin6PN: "fdaa::42",
			}, nil, now.Add(-maxProbeAge-time.Nanosecond))
		}},
		{name: "successful semantics without bound origin", setup: func(state *readinessState) {
			state.store(probeResult{chainID: "zerone-2", height: 42}, nil, now)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := &readinessState{}
			test.setup(state)
			handler := readinessHandler{state: state, now: func() time.Time { return now }}
			for _, endpoint := range []struct {
				path       string
				wantStatus int
			}{
				{path: "/ready", wantStatus: http.StatusForbidden},
				{path: "/health", wantStatus: http.StatusServiceUnavailable},
			} {
				recorder := httptest.NewRecorder()
				handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, endpoint.path, nil))
				if recorder.Code != endpoint.wantStatus {
					t.Fatalf("%s status = %d, want %d", endpoint.path, recorder.Code, endpoint.wantStatus)
				}
				if strings.Contains(recorder.Body.String(), "wrong chain") {
					t.Fatalf("%s leaked private probe reason: %q", endpoint.path, recorder.Body.String())
				}
				if recorder.Header().Get("X-Zerone-Validated-6PN") != "" {
					t.Fatalf("%s leaked stale/unvalidated origin address", endpoint.path)
				}
			}
		})
	}
}

type testError string

func (e testError) Error() string { return string(e) }

func errorsForTest(message string) error { return testError(message) }
