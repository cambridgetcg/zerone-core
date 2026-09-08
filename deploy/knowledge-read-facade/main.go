// Finite, read-only serving. No listener starts without an explicit validated
// configuration. The concrete admission profile is SINGLETON, not multi-replica.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	pointBytes        = 64 * 1024
	snapshotBytes     = 256 * 1024
	pointTimeout      = 5 * time.Second
	clientRate        = 2
	globalConcurrency = 8
	freshnessWindow   = 30 * time.Second
	futureSkew        = 10 * time.Second
)

var factIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
var digestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
var heightPattern = regexp.MustCompile(`^[1-9][0-9]{0,18}$`)

// Every route shares this gate for the entire response lifetime. The concrete
// implementation only coordinates this process. Deployment must be a singleton.
type Admission interface {
	Acquire(context.Context, *http.Request, int, int) (func(), error)
}

var errAdmissionLimited = errors.New("admission limited")

type facade struct {
	upstream     *url.URL
	statusOrigin *url.URL
	chainID      string
	client       *http.Client
	admission    Admission
	snapshots    map[string][]byte // initialization only; never mutated while serving
	head         []byte
	validUntil   time.Time // nonzero only for the reviewed singleton profile
	now          func() time.Time
}

func fixedOrigin(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.RawPath != "" || u.Path != "" || u.Opaque != "" {
		return nil, errors.New("one exact credential-free fixed origin required")
	}
	return u, nil
}
func newFacade(upstream, chainID string, admission Admission, transport http.RoundTripper) (*facade, error) {
	target, err := fixedOrigin(upstream)
	if err != nil {
		return nil, err
	}
	if !regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,49}$`).MatchString(chainID) {
		return nil, errors.New("expected chain ID required")
	}
	if transport == nil {
		transport = boundedTransport()
	}
	return &facade{upstream: target, statusOrigin: target, chainID: chainID, admission: admission, snapshots: map[string][]byte{}, now: time.Now, client: &http.Client{Timeout: pointTimeout, Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func response(w http.ResponseWriter, r *http.Request, code int, body []byte) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(code)
	if r.Method != "HEAD" {
		_, _ = w.Write(body)
	}
}
func refuse(w http.ResponseWriter, r *http.Request, code int, message string) {
	body, _ := json.Marshal(map[string]string{"error": message})
	response(w, r, code, body)
}
func (f *facade) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), pointTimeout)
	defer cancel()
	if f.admission == nil || (!f.validUntil.IsZero() && !f.now().Before(f.validUntil)) {
		refuse(w, r, 503, "singleton admission or deployment review unavailable")
		return
	}
	release, err := f.admission.Acquire(ctx, r, clientRate, globalConcurrency)
	if err != nil {
		code := 503
		if errors.Is(err, errAdmissionLimited) {
			code = 429
		}
		refuse(w, r, code, "read admission unavailable")
		return
	}
	if release == nil {
		refuse(w, r, 503, "invalid admission lease")
		return
	}
	defer release()
	if r.Method != "GET" && r.Method != "HEAD" {
		refuse(w, r, 405, "GET or HEAD required")
		return
	}
	if len(r.RequestURI) > 256 || r.URL.RawQuery != "" || r.URL.ForceQuery || r.URL.RawPath != "" || strings.Contains(r.RequestURI, "%") || r.ContentLength != 0 || len(r.TransferEncoding) > 0 {
		refuse(w, r, 400, "query parameters, encoded paths and bodies are forbidden")
		return
	}
	if r.URL.Path == "/head" {
		if len(f.head) == 0 {
			refuse(w, r, 404, "no selected publication")
			return
		}
		response(w, r, 200, f.head)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/snapshots/") {
		id := strings.TrimPrefix(r.URL.Path, "/snapshots/")
		body, ok := f.snapshots[id]
		if !digestPattern.MatchString(id) || !ok {
			refuse(w, r, 404, "snapshot not found")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.Header().Set("ETag", `"`+id+`"`)
		if r.Method != "HEAD" {
			_, _ = w.Write(body)
		}
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/facts/")
	if !strings.HasPrefix(r.URL.Path, "/facts/") || !factIDPattern.MatchString(id) {
		refuse(w, r, 404, "route not found")
		return
	}
	body, headers, code, err := f.read(ctx, f.upstream, "/zerone/knowledge/v1/facts/"+id)
	if err != nil {
		refuse(w, r, code, "fact unavailable")
		return
	}
	height := headers.Get("Grpc-Metadata-X-Cosmos-Block-Height")
	alternate := headers.Get("X-Cosmos-Block-Height")
	if height == "" {
		height = alternate
	}
	if !validHeight(height, true) || (alternate != "" && height != alternate) || len(headers.Values("Grpc-Metadata-X-Cosmos-Block-Height")) > 1 || len(headers.Values("X-Cosmos-Block-Height")) > 1 {
		refuse(w, r, 502, "point height is unavailable or conflicting")
		return
	}
	// Preserve the original fixed-origin chain binding, and independently require
	// fresh paired RPC status below. This header remains an assertion, not proof;
	// neither source is selected by incoming requests or publication contents.
	if values := headers.Values("X-Zerone-Expected-Chain"); len(values) != 1 || values[0] != f.chainID {
		refuse(w, r, 502, "point chain assertion mismatch")
		return
	}
	if err := validatePointJSON(body, id); err != nil {
		refuse(w, r, 502, "invalid fact response")
		return
	}
	statusBody, _, code, err := f.read(ctx, f.statusOrigin, "/status")
	if err != nil {
		refuse(w, r, code, "fixed status unavailable")
		return
	}
	status, err := validateStatus(statusBody, f.chainID, height, f.now())
	if err != nil {
		refuse(w, r, 502, "point provenance or freshness unavailable")
		return
	}
	body, err = json.Marshal(json.RawMessage(body))
	if err != nil {
		refuse(w, r, 502, "invalid fact encoding")
		return
	}
	envelope, err := json.Marshal(struct {
		Schema        string            `json:"schema"`
		ChainID       string            `json:"chainId"`
		Height        string            `json:"blockHeight"`
		PayloadDigest string            `json:"payloadDigest"`
		DigestScope   string            `json:"payloadDigestScope"`
		ObservedAt    string            `json:"observedAt"`
		Status        statusObservation `json:"freshness"`
		Provenance    string            `json:"provenance"`
		Coverage      string            `json:"coverage"`
		Payload       json.RawMessage   `json:"payload"`
	}{"zerone.fact-point/v1", f.chainID, height, hashBytes(body), "SHA-256 of exact embedded compact JSON payload bytes; not a chain proof", f.now().UTC().Format(time.RFC3339Nano), status, "fixed operator-paired REST/RPC observations; no light-client, release or custody proof", "one fact plus bounded canonical adjacency; whole graph NOT_CLAIMED", body})
	if err != nil || len(envelope) > pointBytes {
		refuse(w, r, 502, "point envelope exceeds 64 KiB")
		return
	}
	response(w, r, 200, envelope)
}
func (f *facade) read(ctx context.Context, origin *url.URL, path string) ([]byte, http.Header, int, error) {
	target := *origin
	target.Path = path
	req, err := http.NewRequestWithContext(ctx, "GET", target.String(), nil)
	if err != nil {
		return nil, nil, 502, err
	}
	req.Header.Set("Accept", "application/json")
	res, err := f.client.Do(req)
	if err != nil {
		return nil, nil, 504, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		code := 502
		switch res.StatusCode {
		case 400, 404, 408, 429, 503, 504:
			code = res.StatusCode
		}
		return nil, nil, code, errors.New("upstream unavailable")
	}
	if strings.TrimSpace(strings.ToLower(strings.Split(res.Header.Get("Content-Type"), ";")[0])) != "application/json" || res.ContentLength > pointBytes {
		return nil, nil, 502, errors.New("invalid upstream media or length")
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, pointBytes+1))
	if err != nil || ctx.Err() != nil {
		return nil, nil, 504, errors.New("body unavailable")
	}
	if len(body) > pointBytes {
		return nil, nil, 502, errors.New("oversize body")
	}
	return body, res.Header, 200, nil
}
func hashBytes(body []byte) string { hash := sha256.Sum256(body); return hex.EncodeToString(hash[:]) }

// Shared bounded JSON traversal: duplicates and depth remain rejected before
// identity decoding. Go's case-insensitive struct decoder is NEVER an ID check.
func validateJSON(body []byte, maximum int) error {
	if len(body) > maximum || !utf8.Valid(body) {
		return errors.New("invalid UTF-8 or byte bound")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var walk func(int) error
	walk = func(depth int) error {
		if depth > 64 {
			return errors.New("JSON depth")
		}
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delim, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for decoder.More() {
				key, err := decoder.Token()
				if err != nil {
					return err
				}
				name, ok := key.(string)
				if !ok || seen[name] {
					return errors.New("duplicate key")
				}
				seen[name] = true
				if err := walk(depth + 1); err != nil {
					return err
				}
			}
		case '[':
			for decoder.More() {
				if err := walk(depth + 1); err != nil {
					return err
				}
			}
		default:
			return errors.New("invalid delimiter")
		}
		_, err = decoder.Token()
		return err
	}
	if err := walk(0); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return errors.New("trailing JSON")
	}
	return nil
}
func exactMember(body []byte, key string) (json.RawMessage, error) {
	var value map[string]json.RawMessage
	if err := json.Unmarshal(body, &value); err != nil || value == nil {
		return nil, errors.New("object required")
	}
	for name := range value {
		if name != key && strings.EqualFold(name, key) {
			return nil, errors.New("case alias forbidden")
		}
	}
	result, ok := value[key]
	if !ok || bytes.Equal(result, []byte("null")) {
		return nil, errors.New("exact member required")
	}
	return result, nil
}
func validatePointJSON(body []byte, id string) error {
	if err := validateJSON(body, pointBytes); err != nil {
		return err
	}
	fact, err := exactMember(body, "fact")
	if err != nil {
		return err
	}
	identity, err := exactMember(fact, "id")
	if err != nil {
		return err
	}
	var actual string
	if err := json.Unmarshal(identity, &actual); err != nil {
		return err
	}
	if actual != id {
		return errors.New("fact ID mismatch")
	}
	return nil
}

type statusObservation struct {
	StatusHeight string `json:"statusHeight"`
	BlockTime    string `json:"blockTime"`
	CatchingUp   bool   `json:"catchingUp"`
	LiveStatus   string `json:"liveStatus"`
}

func validateStatus(body []byte, chain, height string, now time.Time) (statusObservation, error) {
	var out statusObservation
	if err := validateJSON(body, pointBytes); err != nil {
		return out, err
	}
	result, err := exactMember(body, "result")
	if err != nil {
		return out, err
	}
	node, err := exactMember(result, "node_info")
	if err != nil {
		return out, err
	}
	network, err := exactMember(node, "network")
	if err != nil {
		return out, err
	}
	var chainID string
	if json.Unmarshal(network, &chainID) != nil || chainID != chain {
		return out, errors.New("wrong chain")
	}
	syncInfo, err := exactMember(result, "sync_info")
	if err != nil {
		return out, err
	}
	h, err := exactMember(syncInfo, "latest_block_height")
	if err != nil {
		return out, err
	}
	blockTime, err := exactMember(syncInfo, "latest_block_time")
	if err != nil {
		return out, err
	}
	catching, err := exactMember(syncInfo, "catching_up")
	if err != nil {
		return out, err
	}
	if json.Unmarshal(h, &out.StatusHeight) != nil || json.Unmarshal(blockTime, &out.BlockTime) != nil || json.Unmarshal(catching, &out.CatchingUp) != nil || out.CatchingUp || !nearHeights(height, out.StatusHeight) {
		return out, errors.New("invalid status")
	}
	timestamp, err := time.Parse(time.RFC3339Nano, out.BlockTime)
	if err != nil || now.Sub(timestamp) > freshnessWindow || timestamp.Sub(now) > futureSkew {
		return out, errors.New("stale or future block")
	}
	out.LiveStatus = "fresh fixed-origin observation; not finality proof"
	return out, nil
}
func validHeight(h string, positive bool) bool {
	if h == "0" {
		return !positive
	}
	if !heightPattern.MatchString(h) {
		return false
	}
	_, err := strconv.ParseInt(h, 10, 64)
	return err == nil
}
func nearHeights(a, b string) bool {
	if !validHeight(a, true) || !validHeight(b, true) {
		return false
	}
	x, _ := strconv.ParseInt(a, 10, 64)
	y, _ := strconv.ParseInt(b, 10, 64)
	d := x - y
	if d < 0 {
		d = -d
	}
	return d <= 128
}
func (f *facade) addSnapshot(id string, body []byte) error {
	if !digestPattern.MatchString(id) || hashBytes(body) != id || len(f.snapshots) >= maxPublications {
		return errors.New("invalid immutable snapshot or storage cap")
	}
	if _, err := validateSnapshot(body, f.chainID); err != nil {
		return err
	}
	if _, exists := f.snapshots[id]; exists {
		return errors.New("snapshot already registered")
	}
	f.snapshots[id] = bytes.Clone(body)
	return nil
}
