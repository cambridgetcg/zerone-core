// Command zerone-query-readiness keeps a small, cached semantic view of the
// private CometBFT origin. Nginx uses its loopback-only endpoints both for the
// public health check and as an auth_request gate before proxying any query.
package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	listenAddress       = "127.0.0.1:18081"
	pollInterval        = 2 * time.Second
	probeTimeout        = 2 * time.Second
	maxProbeAge         = 5 * time.Second
	activeMaxBlockAge   = 30 * time.Second
	activeMaxFutureSkew = 10 * time.Second
	maxStatusBytes      = 64 << 10
	maxCometHeight      = uint64(1<<63 - 1)
)

var upstreamHostPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)*\.internal$`)

type config struct {
	role                     string
	expectedChainID          string
	upstreamHost             string
	statusURL                string
	restSyncingURL           string
	archiveABCIInfoURL       string
	archiveBlockURL          string
	archiveNextBlockURL      string
	expectedArchiveHeight    uint64
	expectedArchiveAppHash   string
	expectedArchiveBlockHash string
	active                   bool
	maxBlockAge              time.Duration
	maxFutureSkew            time.Duration
	maxResponseSize          int64
	resolver                 hostnameResolver
	bindResolvedAddress      bool
}

type hostnameResolver interface {
	LookupIPAddr(context.Context, string) ([]net.IPAddr, error)
}

func configFromEnvironment() (config, error) {
	role := os.Getenv("GATEWAY_ROLE")
	chainID := os.Getenv("EXPECTED_CHAIN_ID")
	host := os.Getenv("UPSTREAM_HOST")
	archiveHeightText := os.Getenv("EXPECTED_ARCHIVE_HEIGHT")
	archiveAppHash := os.Getenv("EXPECTED_ARCHIVE_APP_HASH")
	archiveBlockHash := os.Getenv("EXPECTED_ARCHIVE_BLOCK_HASH")

	active := false
	var archiveHeight uint64
	switch role + ":" + chainID {
	case "zerone-2-query:zerone-2":
		active = true
		if archiveHeightText != "" || archiveAppHash != "" || archiveBlockHash != "" {
			return config{}, errors.New("active gateway forbids archive checkpoint expectations")
		}
	case "zerone-1-archive-query:zerone-1":
		if !isCanonicalPositiveInteger(archiveHeightText) {
			return config{}, errors.New("EXPECTED_ARCHIVE_HEIGHT must be a canonical positive integer")
		}
		var err error
		archiveHeight, err = strconv.ParseUint(archiveHeightText, 10, 64)
		if err != nil || archiveHeight >= maxCometHeight {
			return config{}, errors.New("EXPECTED_ARCHIVE_HEIGHT must permit an A+1 proof")
		}
		if !isLowerHexHash(archiveAppHash) {
			return config{}, errors.New("EXPECTED_ARCHIVE_APP_HASH must be exactly 64 lowercase hex characters")
		}
		if !isLowerHexHash(archiveBlockHash) {
			return config{}, errors.New("EXPECTED_ARCHIVE_BLOCK_HASH must be exactly 64 lowercase hex characters")
		}
	default:
		return config{}, errors.New("GATEWAY_ROLE and EXPECTED_CHAIN_ID are not an approved pair")
	}
	if !upstreamHostPattern.MatchString(host) {
		return config{}, errors.New("UPSTREAM_HOST must be a lowercase Fly .internal DNS name")
	}

	rpcBaseURL := "http://" + host + ":26657"
	cfg := config{
		role:                     role,
		expectedChainID:          chainID,
		upstreamHost:             host,
		statusURL:                rpcBaseURL + "/status",
		restSyncingURL:           "http://" + host + ":1317/cosmos/base/tendermint/v1beta1/syncing",
		expectedArchiveHeight:    archiveHeight,
		expectedArchiveAppHash:   archiveAppHash,
		expectedArchiveBlockHash: archiveBlockHash,
		active:                   active,
		maxBlockAge:              activeMaxBlockAge,
		maxFutureSkew:            activeMaxFutureSkew,
		maxResponseSize:          maxStatusBytes,
		resolver:                 net.DefaultResolver,
		bindResolvedAddress:      true,
	}
	if !active {
		cfg.archiveABCIInfoURL = rpcBaseURL + "/abci_info"
		cfg.archiveBlockURL = rpcBaseURL + "/block?height=" + archiveHeightText
		cfg.archiveNextBlockURL = rpcBaseURL + "/block?height=" + strconv.FormatUint(archiveHeight+1, 10)
	}
	return cfg, nil
}

type statusEnvelope struct {
	Result *struct {
		NodeInfo struct {
			Network string `json:"network"`
		} `json:"node_info"`
		SyncInfo struct {
			LatestBlockHeight string `json:"latest_block_height"`
			LatestBlockHash   string `json:"latest_block_hash"`
			LatestBlockTime   string `json:"latest_block_time"`
			CatchingUp        *bool  `json:"catching_up"`
		} `json:"sync_info"`
	} `json:"result"`
}

type probeResult struct {
	chainID            string
	height             uint64
	catchingUp         bool
	validatedOrigin6PN string
}

func probeStatus(ctx context.Context, client *http.Client, cfg config, now time.Time) (probeResult, error) {
	body, err := fetchBounded(ctx, client, cfg.statusURL, "status", cfg.maxResponseSize)
	if err != nil {
		return probeResult{}, err
	}

	var status statusEnvelope
	if err := json.Unmarshal(body, &status); err != nil {
		return probeResult{}, fmt.Errorf("decode status response: %w", err)
	}
	if status.Result == nil {
		return probeResult{}, errors.New("status response omits result")
	}
	if status.Result.NodeInfo.Network != cfg.expectedChainID {
		return probeResult{}, fmt.Errorf("origin chain ID does not match %q", cfg.expectedChainID)
	}

	heightText := status.Result.SyncInfo.LatestBlockHeight
	if !isCanonicalPositiveInteger(heightText) {
		return probeResult{}, errors.New("latest block height is not a canonical positive integer")
	}
	height, err := strconv.ParseUint(heightText, 10, 64)
	if err != nil {
		return probeResult{}, errors.New("latest block height exceeds uint64")
	}
	if height > maxCometHeight {
		return probeResult{}, errors.New("latest block height exceeds Comet int64 maximum")
	}
	if status.Result.SyncInfo.CatchingUp == nil {
		return probeResult{}, errors.New("status response omits catching_up")
	}
	blockTime, err := time.Parse(time.RFC3339Nano, status.Result.SyncInfo.LatestBlockTime)
	if err != nil {
		return probeResult{}, errors.New("latest block time is malformed")
	}
	if cfg.active {
		if *status.Result.SyncInfo.CatchingUp {
			return probeResult{}, errors.New("active origin is catching up")
		}
		if blockTime.After(now.Add(cfg.maxFutureSkew)) {
			return probeResult{}, errors.New("active latest block time is too far in the future")
		}
		if blockTime.Before(now.Add(-cfg.maxBlockAge)) {
			return probeResult{}, errors.New("active latest block time is stale")
		}
	} else {
		if !*status.Result.SyncInfo.CatchingUp {
			return probeResult{}, errors.New("fresh-key archive is not reporting catching_up")
		}
		if height != cfg.expectedArchiveHeight {
			return probeResult{}, errors.New("archive status height does not match signed checkpoint")
		}
		blockHash, err := normalizeHexHash(status.Result.SyncInfo.LatestBlockHash, "archive status block hash")
		if err != nil {
			return probeResult{}, err
		}
		if blockHash != cfg.expectedArchiveBlockHash {
			return probeResult{}, errors.New("archive status block hash does not match signed checkpoint")
		}
	}

	return probeResult{chainID: cfg.expectedChainID, height: height, catchingUp: *status.Result.SyncInfo.CatchingUp}, nil
}

func fetchBounded(ctx context.Context, client *http.Client, endpoint, label string, maxBytes int64) ([]byte, error) {
	return fetchBoundedAtStatus(ctx, client, endpoint, label, maxBytes, http.StatusOK)
}

func fetchBoundedAtStatus(ctx context.Context, client *http.Client, endpoint, label string, maxBytes int64, expectedStatus int) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("construct %s request: %w", label, err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", label, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != expectedStatus {
		return nil, fmt.Errorf("%s endpoint returned HTTP %d, expected %d", label, resp.StatusCode, expectedStatus)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read %s response: %w", label, err)
	}
	if int64(len(body)) > maxBytes {
		return nil, fmt.Errorf("%s response exceeds %d bytes", label, maxBytes)
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return nil, fmt.Errorf("%s response is empty", label)
	}
	return body, nil
}

func probeRESTSyncing(ctx context.Context, client *http.Client, cfg config, status probeResult) error {
	body, err := fetchBounded(ctx, client, cfg.restSyncingURL, "REST syncing", cfg.maxResponseSize)
	if err != nil {
		return err
	}
	var response struct {
		Syncing *bool `json:"syncing"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("decode REST syncing response: %w", err)
	}
	if response.Syncing == nil {
		return errors.New("REST syncing response omits syncing boolean")
	}
	if *response.Syncing != status.catchingUp {
		return errors.New("REST syncing state disagrees with Comet status")
	}
	return nil
}

func probeArchiveCheckpoint(ctx context.Context, client *http.Client, cfg config) error {
	body, err := fetchBounded(ctx, client, cfg.archiveABCIInfoURL, "archive ABCI info", cfg.maxResponseSize)
	if err != nil {
		return err
	}
	var abci struct {
		Result *struct {
			Response *struct {
				LastBlockHeight  string `json:"last_block_height"`
				LastBlockAppHash string `json:"last_block_app_hash"`
			} `json:"response"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &abci); err != nil {
		return fmt.Errorf("decode archive ABCI info response: %w", err)
	}
	if abci.Result == nil || abci.Result.Response == nil {
		return errors.New("archive ABCI info response omits result.response")
	}
	if abci.Result.Response.LastBlockHeight != strconv.FormatUint(cfg.expectedArchiveHeight, 10) {
		return errors.New("archive ABCI height does not match signed checkpoint")
	}
	appHash, err := normalizeAppHash(abci.Result.Response.LastBlockAppHash)
	if err != nil {
		return err
	}
	if appHash != cfg.expectedArchiveAppHash {
		return errors.New("archive ABCI app hash does not match signed checkpoint")
	}

	body, err = fetchBounded(ctx, client, cfg.archiveBlockURL, "archive A block", cfg.maxResponseSize)
	if err != nil {
		return err
	}
	var archiveBlock struct {
		Result *struct {
			BlockID *struct {
				Hash string `json:"hash"`
			} `json:"block_id"`
			Block *struct {
				Header *struct {
					ChainID string `json:"chain_id"`
					Height  string `json:"height"`
				} `json:"header"`
			} `json:"block"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &archiveBlock); err != nil {
		return fmt.Errorf("decode archive A block response: %w", err)
	}
	if archiveBlock.Result == nil || archiveBlock.Result.BlockID == nil ||
		archiveBlock.Result.Block == nil || archiveBlock.Result.Block.Header == nil {
		return errors.New("archive A block response omits identity fields")
	}
	if archiveBlock.Result.Block.Header.ChainID != cfg.expectedChainID ||
		archiveBlock.Result.Block.Header.Height != strconv.FormatUint(cfg.expectedArchiveHeight, 10) {
		return errors.New("archive A block identity does not match signed checkpoint")
	}
	blockHash, err := normalizeHexHash(archiveBlock.Result.BlockID.Hash, "archive A block hash")
	if err != nil {
		return err
	}
	if blockHash != cfg.expectedArchiveBlockHash {
		return errors.New("archive A block hash does not match signed checkpoint")
	}

	body, err = fetchBoundedAtStatus(ctx, client, cfg.archiveNextBlockURL, "archive A+1 block", cfg.maxResponseSize, http.StatusInternalServerError)
	if err != nil {
		return err
	}
	var nextBlock struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Error   *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Data    string `json:"data"`
		} `json:"error"`
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(body, &nextBlock); err != nil {
		return fmt.Errorf("decode archive A+1 block response: %w", err)
	}
	result := strings.TrimSpace(string(nextBlock.Result))
	expectedErrorData := fmt.Sprintf(
		"height %d must be less than or equal to the current blockchain height %d",
		cfg.expectedArchiveHeight+1, cfg.expectedArchiveHeight,
	)
	if nextBlock.JSONRPC != "2.0" || strings.TrimSpace(string(nextBlock.ID)) != "-1" ||
		nextBlock.Error == nil || nextBlock.Error.Code != -32603 ||
		nextBlock.Error.Message != "Internal error" || nextBlock.Error.Data != expectedErrorData ||
		(result != "" && result != "null") {
		return errors.New("archive unexpectedly exposes A+1 block state")
	}
	return nil
}

func normalizeHexHash(value, label string) (string, error) {
	if len(value) != 64 {
		return "", fmt.Errorf("%s is malformed", label)
	}
	for _, character := range value {
		if !strings.ContainsRune("0123456789abcdefABCDEF", character) {
			return "", fmt.Errorf("%s is malformed", label)
		}
	}
	return strings.ToLower(value), nil
}

func normalizeAppHash(value string) (string, error) {
	if len(value) == 64 {
		for _, character := range value {
			if !strings.ContainsRune("0123456789abcdefABCDEF", character) {
				return "", errors.New("archive ABCI app hash is malformed")
			}
		}
		return strings.ToLower(value), nil
	}
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil || len(decoded) != 32 {
		return "", errors.New("archive ABCI app hash is malformed")
	}
	return hex.EncodeToString(decoded), nil
}

func probeOrigin(ctx context.Context, client *http.Client, cfg config, now time.Time) (probeResult, error) {
	resolvedAddress, err := requireSingle6PN(ctx, cfg.resolver, cfg.upstreamHost)
	if err != nil {
		return probeResult{}, err
	}
	if cfg.bindResolvedAddress {
		client = newHTTPClientForOrigin(cfg.upstreamHost, resolvedAddress.String())
		defer client.CloseIdleConnections()
	}
	if client == nil {
		return probeResult{}, errors.New("origin HTTP client is unavailable")
	}
	status, err := probeStatus(ctx, client, cfg, now)
	if err != nil {
		return probeResult{}, err
	}
	if err := probeRESTSyncing(ctx, client, cfg, status); err != nil {
		return probeResult{}, err
	}
	if !cfg.active {
		if err := probeArchiveCheckpoint(ctx, client, cfg); err != nil {
			return probeResult{}, err
		}
	}
	status.validatedOrigin6PN = resolvedAddress.String()
	return status, nil
}

func requireSingle6PN(ctx context.Context, resolver hostnameResolver, host string) (net.IP, error) {
	if resolver == nil {
		return nil, errors.New("origin DNS resolver is unavailable")
	}
	addresses, err := resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("resolve origin host: %w", err)
	}
	unique := make(map[string]net.IP, len(addresses))
	for _, address := range addresses {
		ip := address.IP
		ipv6 := ip.To16()
		if address.Zone != "" || ip.To4() != nil || ipv6 == nil || ipv6[0] != 0xfd || ipv6[1] != 0xaa {
			return nil, errors.New("origin DNS returned a non-6PN IPv6 address")
		}
		unique[ip.String()] = append(net.IP(nil), ipv6...)
	}
	if len(unique) != 1 {
		return nil, fmt.Errorf("origin DNS must return exactly one unique 6PN address, got %d", len(unique))
	}
	var resolved net.IP
	for _, ip := range unique {
		resolved = ip
	}
	return resolved, nil
}

func isCanonicalPositiveInteger(value string) bool {
	if value == "" || value[0] < '1' || value[0] > '9' {
		return false
	}
	for i := 1; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	return true
}

func isLowerHexHash(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, character := range value {
		if !strings.ContainsRune("0123456789abcdef", character) {
			return false
		}
	}
	return true
}

func isCanonical6PN(value string) bool {
	ip := net.ParseIP(value)
	if ip == nil || ip.To4() != nil {
		return false
	}
	ipv6 := ip.To16()
	return ipv6 != nil && ipv6[0] == 0xfd && ipv6[1] == 0xaa && ipv6.String() == value
}

type readinessSnapshot struct {
	ready              bool
	chainID            string
	height             uint64
	validatedOrigin6PN string
	checkedAt          time.Time
	reason             string
}

type readinessState struct {
	mu       sync.RWMutex
	snapshot readinessSnapshot
}

func (s *readinessState) store(result probeResult, probeErr error, checkedAt time.Time) (bool, readinessSnapshot) {
	next := readinessSnapshot{checkedAt: checkedAt}
	if probeErr == nil {
		if !isCanonical6PN(result.validatedOrigin6PN) {
			next.reason = "semantic probe omitted a canonical validated 6PN address"
		} else {
			next.ready = true
			next.chainID = result.chainID
			next.height = result.height
			next.validatedOrigin6PN = result.validatedOrigin6PN
		}
	} else {
		next.reason = probeErr.Error()
	}

	s.mu.Lock()
	changed := s.snapshot.ready != next.ready ||
		s.snapshot.reason != next.reason ||
		s.snapshot.validatedOrigin6PN != next.validatedOrigin6PN
	s.snapshot = next
	s.mu.Unlock()
	return changed, next
}

func (s *readinessState) load(now time.Time) readinessSnapshot {
	s.mu.RLock()
	snapshot := s.snapshot
	s.mu.RUnlock()
	if snapshot.checkedAt.IsZero() || now.Before(snapshot.checkedAt) || now.Sub(snapshot.checkedAt) > maxProbeAge {
		snapshot.ready = false
		snapshot.chainID = ""
		snapshot.height = 0
		snapshot.validatedOrigin6PN = ""
		snapshot.reason = "semantic status cache is unavailable or expired"
	}
	return snapshot
}

type readinessHandler struct {
	state *readinessState
	now   func() time.Time
}

func (h readinessHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	snapshot := h.state.load(h.now())
	switch r.URL.Path {
	case "/ready":
		if snapshot.ready {
			w.Header().Set("X-Zerone-Validated-6PN", snapshot.validatedOrigin6PN)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusForbidden)
	case "/health":
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json")
		status := http.StatusServiceUnavailable
		response := struct {
			Ready             bool   `json:"ready"`
			ChainID           string `json:"chain_id,omitempty"`
			LatestBlockHeight string `json:"latest_block_height,omitempty"`
		}{Ready: snapshot.ready}
		if snapshot.ready {
			status = http.StatusOK
			response.ChainID = snapshot.chainID
			response.LatestBlockHeight = strconv.FormatUint(snapshot.height, 10)
		}
		w.WriteHeader(status)
		if r.Method == http.MethodHead {
			return
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("write health response: %v", err)
		}
	default:
		http.NotFound(w, r)
	}
}

func newHTTPClient() *http.Client {
	dialer := &net.Dialer{Timeout: probeTimeout, KeepAlive: 30 * time.Second}
	return newHTTPClientWithDialer(dialer.DialContext)
}

func newHTTPClientForOrigin(expectedHost, resolvedAddress string) *http.Client {
	dialer := &net.Dialer{Timeout: probeTimeout, KeepAlive: 30 * time.Second}
	return newHTTPClientWithDialer(func(ctx context.Context, _ string, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("parse origin dial address: %w", err)
		}
		if host != expectedHost {
			return nil, fmt.Errorf("refuse readiness dial to unexpected host %q", host)
		}
		return dialer.DialContext(ctx, "tcp6", net.JoinHostPort(resolvedAddress, port))
	})
}

func newHTTPClientWithDialer(dialContext func(context.Context, string, string) (net.Conn, error)) *http.Client {
	return &http.Client{
		Timeout: probeTimeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return errors.New("origin probe redirects are forbidden")
		},
		Transport: &http.Transport{
			Proxy:                  nil,
			DialContext:            dialContext,
			DisableCompression:     true,
			MaxResponseHeaderBytes: 16 << 10,
			MaxIdleConns:           1,
			MaxIdleConnsPerHost:    1,
			MaxConnsPerHost:        1,
			IdleConnTimeout:        30 * time.Second,
			ResponseHeaderTimeout:  probeTimeout,
		},
	}
}

func pollForever(cfg config, client *http.Client, state *readinessState) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		checkedAt := time.Now().UTC()
		ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
		result, err := probeOrigin(ctx, client, cfg, checkedAt)
		cancel()
		changed, snapshot := state.store(result, err, time.Now().UTC())
		if changed {
			if snapshot.ready {
				log.Printf("origin ready: chain=%s height=%d", snapshot.chainID, snapshot.height)
			} else {
				log.Printf("origin not ready: %s", snapshot.reason)
			}
		}
		<-ticker.C
	}
}

func main() {
	cfg, err := configFromEnvironment()
	if err != nil {
		log.Fatalf("configuration: %v", err)
	}
	if len(os.Args) == 2 && os.Args[1] == "-check-config" {
		return
	}
	if len(os.Args) != 1 {
		log.Fatal("usage: zerone-query-readiness [-check-config]")
	}

	state := &readinessState{}
	go pollForever(cfg, newHTTPClient(), state)

	server := &http.Server{
		Addr:              listenAddress,
		Handler:           readinessHandler{state: state, now: time.Now},
		ReadHeaderTimeout: time.Second,
		ReadTimeout:       2 * time.Second,
		WriteTimeout:      2 * time.Second,
		IdleTimeout:       10 * time.Second,
		MaxHeaderBytes:    4096,
	}
	log.Printf("semantic readiness listening on %s for role=%s", listenAddress, cfg.role)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("serve readiness: %v", err)
	}
}
