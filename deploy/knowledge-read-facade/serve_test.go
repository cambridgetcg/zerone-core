package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestConcreteSlidingWindowClientBoundAndPeerIdentity(t *testing.T) {
	a := newSingletonAdmission()
	now := time.Now()
	a.now = func() time.Time { return now }
	req := httptest.NewRequest("GET", "/head", nil)
	req.RemoteAddr = "127.0.0.1:1000"
	for i := 0; i < 2; i++ {
		release, err := a.Acquire(context.Background(), req, 2, 8)
		if err != nil {
			t.Fatal(err)
		}
		release()
		release()
	}
	req.RemoteAddr = "127.0.0.1:2000"
	req.Header.Set("X-Forwarded-For", "8.8.8.8")
	if _, err := a.Acquire(context.Background(), req, 2, 8); err != errAdmissionLimited {
		t.Fatal("port/header bypass", err)
	}
	now = now.Add(time.Second - time.Nanosecond)
	if _, err := a.Acquire(context.Background(), req, 2, 8); err != errAdmissionLimited {
		t.Fatal("early reset")
	}
	now = now.Add(time.Nanosecond)
	release, err := a.Acquire(context.Background(), req, 2, 8)
	if err != nil {
		t.Fatal(err)
	}
	release()
	for i := 1; i < maxClients; i++ {
		req.RemoteAddr = fmt.Sprintf("10.%d.%d.%d:1", (i>>16)&255, (i>>8)&255, i&255)
		release, err := a.Acquire(context.Background(), req, 2, 8)
		if err != nil {
			t.Fatal(i, err)
		}
		release()
	}
	req.RemoteAddr = "192.0.2.1:1"
	if _, err := a.Acquire(context.Background(), req, 2, 8); err != errAdmissionLimited {
		t.Fatal("tracking overflow")
	}
	if len(a.clients) != maxClients || a.active != 0 {
		t.Fatal("unbounded/leaked state")
	}
	now = now.Add(time.Second)
	release, err = a.Acquire(context.Background(), req, 2, 8)
	if err != nil {
		t.Fatal(err)
	}
	release()
	if len(a.clients) != 1 {
		t.Fatal("expired identities not reclaimed")
	}
	for _, pair := range [][2]string{{"[::ffff:192.0.2.1]:1", "192.0.2.1:2"}, {"[2001:db8::1]:1", "[2001:db8::ffff]:2"}} {
		x, _ := peerClient(pair[0])
		y, _ := peerClient(pair[1])
		if x != y {
			t.Fatal("alias splits identity")
		}
	}
}
func TestConcreteAdmissionConcurrencyAndCancellation(t *testing.T) {
	a := newSingletonAdmission()
	var wg sync.WaitGroup
	ready := make(chan func(), 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r := httptest.NewRequest("GET", "/head", nil)
			r.RemoteAddr = fmt.Sprintf("192.0.2.%d:1", i+1)
			release, err := a.Acquire(context.Background(), r, 2, 8)
			if err != nil {
				t.Error(err)
				return
			}
			ready <- release
		}(i)
	}
	wg.Wait()
	if len(ready) != 8 {
		t.Fatal("eight admissions not acquired")
	}
	r := httptest.NewRequest("GET", "/head", nil)
	r.RemoteAddr = "198.51.100.1:1"
	if _, err := a.Acquire(context.Background(), r, 2, 8); err != errAdmissionLimited {
		t.Fatal("ninth acquired")
	}
	for i := 0; i < 8; i++ {
		(<-ready)()
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := a.Acquire(ctx, r, 2, 8); err != context.Canceled {
		t.Fatal("canceled admission")
	}
}
func TestFreshStatusFailures(t *testing.T) {
	base := fmt.Sprintf(`{"result":{"node_info":{"network":"zerone-1"},"sync_info":{"latest_block_height":"1000","latest_block_time":%q,"catching_up":false}}}`, time.Now().UTC().Format(time.RFC3339Nano))
	for name, body := range map[string]string{
		"wrong chain":       strings.Replace(base, "zerone-1", "zerone-2", 1),
		"height drift":      strings.Replace(base, `"1000"`, `"500"`, 1),
		"syncing":           strings.Replace(base, `"catching_up":false`, `"catching_up":true`, 1),
		"null":              strings.Replace(base, `"catching_up":false`, `"catching_up":null`, 1),
		"case":              strings.Replace(base, `"network"`, `"Network"`, 1),
		"conflicting alias": strings.Replace(base, `"network":"zerone-1"`, `"network":"wrong","NETWORK":"zerone-1"`, 1),
		"duplicate":         strings.Replace(base, `"network":"zerone-1"`, `"network":"zerone-1","network":"zerone-1"`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := validateStatus([]byte(body), "zerone-1", "1000", time.Now()); err == nil {
				t.Fatal("bad status accepted")
			}
		})
	}
	for _, now := range []time.Time{time.Now().Add(time.Minute), time.Now().Add(-time.Minute)} {
		if _, err := validateStatus([]byte(base), "zerone-1", "1000", now); err == nil {
			t.Fatal("time skew accepted")
		}
	}
}
func previewConfig(root, origin string) serveConfig {
	return serveConfig{Schema: "zerone.knowledge-read-config/v1", Profile: "loopback-preview", Listen: "127.0.0.1:0", RESTOrigin: origin, RPCOrigin: origin, ChainID: "zerone-1", SnapshotRoot: root, HeadID: strings.Repeat("0", 64), InstanceID: "local-synthetic"}
}
func TestConfigurationIsExplicitAndFailClosed(t *testing.T) {
	c := previewConfig(t.TempDir(), "http://127.0.0.1:1")
	if _, err := validateConfig(c, time.Now()); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"wildcard", "external", "path", "userinfo", "query", "missing profile", "production receipt", "different chain"} {
		t.Run(name, func(t *testing.T) {
			changed := c
			switch name {
			case "wildcard":
				changed.Listen = "0.0.0.0:8080"
			case "external":
				changed.RESTOrigin = "http://example.invalid"
			case "path":
				changed.RESTOrigin += "/root"
			case "userinfo":
				changed.RESTOrigin = "http://user@127.0.0.1:1"
			case "query":
				changed.RPCOrigin += "?url=bad"
			case "missing profile":
				changed.Profile = ""
			case "production receipt":
				changed.Profile = "singleton-direct"
				changed.RESTOrigin = "https://example.invalid"
				changed.RPCOrigin = changed.RESTOrigin
			case "different chain":
				changed.ChainID = "zerone-2"
			}
			if _, err := validateConfig(changed, time.Now()); err == nil {
				t.Fatal("bad config accepted")
			}
		})
	}
	var out bytes.Buffer
	if err := run(context.Background(), nil, &out); err == nil || out.Len() != 0 {
		t.Fatal("default listener")
	}
	b, _ := compactJSON(c)
	b = bytes.Replace(b, []byte(`"profile"`), []byte(`"Profile"`), 1)
	if decodeExact(b, &c, 8192) == nil {
		t.Fatal("case-insensitive config")
	}
}
func TestSingletonReviewReceiptDoesNotInventFleetVerification(t *testing.T) {
	now := time.Now().UTC()
	review := deploymentReview{"zerone.knowledge-singleton-review/v1", "one", []string{"one"}, true, true, true, now.Format(time.RFC3339Nano), now.Add(time.Hour).Format(time.RFC3339Nano), "operator-reviewed singleton inventory and drain; assertion, not independently verified by this server"}
	c := previewConfig(t.TempDir(), "https://operator-approved.invalid")
	c.Profile = "singleton-direct"
	c.Listen = "127.0.0.1:8080"
	c.InstanceID = "one"
	c.TLSCertificatePath = "/operator-provisioned/not-read-in-parser-test.pem"
	c.TLSPrivateKeyPath = "/operator-provisioned/not-read-in-parser-test.key"
	c.DeploymentReviewPath = filepath.Join(t.TempDir(), "review.json")
	check := func(review deploymentReview) error {
		b, _ := compactJSON(review)
		if err := os.WriteFile(c.DeploymentReviewPath, b, 0600); err != nil {
			t.Fatal(err)
		}
		c.DeploymentReviewSHA256 = hashBytes(b)
		_, err := validateConfig(c, now)
		return err
	}
	if err := check(review); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"two replicas", "not drained", "overlap", "proxy identity", "stale", "expired"} {
		t.Run(name, func(t *testing.T) {
			r := review
			switch name {
			case "two replicas":
				r.ActiveInstances = []string{"one", "two"}
			case "not drained":
				r.PreviousInstancesDrained = false
			case "overlap":
				r.NoOverlap = false
			case "proxy identity":
				r.DirectPeerIngress = false
			case "stale":
				r.VerifiedAt = now.Add(-6 * time.Minute).Format(time.RFC3339Nano)
			case "expired":
				r.ValidUntil = now.Format(time.RFC3339Nano)
			}
			if check(r) == nil {
				t.Fatal("missing inventory gate accepted")
			}
		})
	}
}

// TLS uses only net/http/httptest's synthetic certificate/key on loopback. The
// receipt below is explicitly a fixture, not external inventory evidence.
func TestLocalSyntheticTLSProfileSnapshot(t *testing.T) {
	fixture := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("snapshot unexpectedly queried origin") }))
	defer fixture.Close()
	rootPath := filepath.Join(t.TempDir(), "store")
	raw, metadata := publicationFixture(t, fixture.URL)
	id, head, err := publish(rootPath, raw, metadata, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	config := previewConfig(rootPath, fixture.URL)
	config.Profile = "singleton-direct"
	config.HeadID = head
	reserved, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	config.Listen = reserved.Addr().String()
	reserved.Close()
	directory := t.TempDir()
	config.TLSCertificatePath = filepath.Join(directory, "synthetic-cert.pem")
	config.TLSPrivateKeyPath = filepath.Join(directory, "synthetic-key.pem")
	pair := fixture.TLS.Certificates[0]
	key, err := x509.MarshalPKCS8PrivateKey(pair.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	for path, body := range map[string][]byte{config.TLSCertificatePath: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: pair.Certificate[0]}), config.TLSPrivateKeyPath: pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: key})} {
		if err := os.WriteFile(path, body, 0600); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now().UTC()
	review := deploymentReview{"zerone.knowledge-singleton-review/v1", config.InstanceID, []string{config.InstanceID}, true, true, true, now.Format(time.RFC3339Nano), now.Add(time.Minute).Format(time.RFC3339Nano), "operator-reviewed singleton inventory and drain; assertion, not independently verified by this server"}
	reviewBody, _ := compactJSON(review)
	config.DeploymentReviewPath = filepath.Join(directory, "synthetic-review.json")
	config.DeploymentReviewSHA256 = hashBytes(reviewBody)
	if err := os.WriteFile(config.DeploymentReviewPath, reviewBody, 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	reader, writer := io.Pipe()
	done := make(chan error, 1)
	go func() { done <- serve(ctx, config, writer); writer.Close() }()
	defer func() {
		cancel()
		reader.Close()
		if err := <-done; err != nil {
			t.Error(err)
		}
	}()
	scanner := bufio.NewScanner(reader)
	if !scanner.Scan() {
		t.Fatal("TLS readiness unavailable")
	}
	client := fixture.Client()
	client.Timeout = 5 * time.Second
	res, err := client.Get("https://" + config.Listen + "/snapshots/" + id)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil || res.StatusCode != 200 || hashBytes(body) != id {
		t.Fatal("TLS immutable response mismatch", err, res.StatusCode)
	}
}

func TestAcceptedConnectionsBounded(t *testing.T) {
	raw, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	l := &cappedListener{Listener: raw, slots: make(chan struct{}, 2), closed: make(chan struct{})}
	defer l.Close()
	var clients, accepted []net.Conn
	defer func() {
		for _, c := range clients {
			c.Close()
		}
		for _, c := range accepted {
			c.Close()
		}
	}()
	for i := 0; i < 2; i++ {
		client, err := net.Dial("tcp", l.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		clients = append(clients, client)
		conn, err := l.Accept()
		if err != nil {
			t.Fatal(err)
		}
		accepted = append(accepted, conn)
	}
	result := make(chan error, 1)
	go func() {
		conn, err := l.Accept()
		if conn != nil {
			conn.Close()
		}
		result <- err
	}()
	select {
	case <-result:
		t.Fatal("connection cap bypassed")
	case <-time.After(20 * time.Millisecond):
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-result:
		if err == nil {
			t.Fatal("closed listener admitted")
		}
	case <-time.After(time.Second):
		t.Fatal("full accept not unblocked")
	}
}

// This launches the actual CLI binary (with the race detector), invokes publish,
// opens a real loopback socket, and drives real admission and upstream transport.
// Even the eight-in-flight check uses one real TCP peer, spaced 510ms apart, not
// fake forwarded client identities or a fake admission provider.
func TestCLIProductPublishServeLimitsAndShutdown(t *testing.T) {
	directory := t.TempDir()
	binary := filepath.Join(directory, "knowledge-read")
	build := exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go"), "build", "-race", "-mod=readonly", "-p", "2", "-o", binary, ".")
	build.Env = append(os.Environ(), "GOPROXY=off", "GOSUMDB=off", "GOTOOLCHAIN=local")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v %s", err, output)
	}
	var mode atomic.Int32
	var upstreamCalls atomic.Int32
	entered := make(chan struct{}, 16)
	unblock := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalls.Add(1)
		if r.URL.Path == "/status" {
			freshStatus(w)
			return
		}
		if r.URL.Path != "/zerone/knowledge/v1/facts/fact-a" || r.URL.RawQuery != "" || r.Method != "GET" {
			t.Error("unapproved upstream", r.URL)
			w.WriteHeader(500)
			return
		}
		switch mode.Load() {
		case 1:
			entered <- struct{}{}
			select {
			case <-unblock:
			case <-r.Context().Done():
				return
			}
		case 2:
			pointResponse(w, `{"fact":{"id":"wrong","ID":"fact-a"}}`)
			return
		case 3:
			entered <- struct{}{}
			<-r.Context().Done()
			return
		}
		pointResponse(w, `{"fact":{"id":"fact-a","content":"http://127.0.0.1:9/not-fetched"}}`)
	}))
	defer upstream.Close()
	raw, m := publicationFixture(t, upstream.URL)
	config := previewConfig(filepath.Join(directory, "store"), upstream.URL)
	configPath := filepath.Join(directory, "config.json")
	rawPath := filepath.Join(directory, "projection.json")
	metaPath := filepath.Join(directory, "metadata.json")
	write := func(path string, b []byte) {
		t.Helper()
		if err := os.WriteFile(path, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	saveConfig := func() { b, _ := compactJSON(config); write(configPath, b) }
	saveConfig()
	write(rawPath, raw)
	mb, _ := compactJSON(m)
	write(metaPath, mb)
	output, err := exec.Command(binary, "publish", "--config", configPath, "--projection", rawPath, "--metadata", metaPath).CombinedOutput()
	if err != nil {
		t.Fatalf("CLI publish %v %s", err, output)
	}
	var published struct {
		SnapshotID string `json:"snapshotId"`
		HeadID     string `json:"headId"`
	}
	if json.Unmarshal(output, &published) != nil || !digestPattern.MatchString(published.HeadID) {
		t.Fatalf("bad publication output %s", output)
	}
	if upstreamCalls.Load() != 0 {
		t.Fatal("publisher fetched receipt contents")
	}
	t.Logf("CLI immutable publish snapshot=%s head=%s; no network", published.SnapshotID, published.HeadID)
	config.HeadID = published.HeadID
	saveConfig()
	cmd := exec.Command(binary, "serve", "--config", configPath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	waited := make(chan error, 1)
	go func() { waited <- cmd.Wait() }()
	stopped := false
	t.Cleanup(func() {
		if !stopped {
			_ = cmd.Process.Kill()
			<-waited
		}
	})
	lines := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		if scanner.Scan() {
			lines <- scanner.Text()
		}
	}()
	var ready struct {
		Address string `json:"address"`
	}
	select {
	case line := <-lines:
		if json.Unmarshal([]byte(line), &ready) != nil || ready.Address == "" {
			t.Fatal("invalid startup", line)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("listener never started")
	}
	host, _, _ := net.SplitHostPort(ready.Address)
	if ip, err := netip.ParseAddr(host); err != nil || !ip.IsLoopback() {
		t.Fatal("non-loopback smoke listener")
	}
	client := &http.Client{Timeout: 6 * time.Second}
	defer client.CloseIdleConnections()
	call := func(path string) (int, []byte, error) {
		res, err := client.Get("http://" + ready.Address + path)
		if err != nil {
			return 0, nil, err
		}
		defer res.Body.Close()
		b, err := io.ReadAll(io.LimitReader(res.Body, snapshotBytes+1))
		return res.StatusCode, b, err
	}
	expect := func(path string, want int) []byte {
		t.Helper()
		code, b, err := call(path)
		if err != nil || code != want {
			t.Fatalf("%s: got %d want %d err=%v body=%s", path, code, want, err, b)
		}
		return b
	}
	point := expect("/facts/fact-a", 200)
	var pointValue struct {
		Payload json.RawMessage `json:"payload"`
		Digest  string          `json:"payloadDigest"`
	}
	if json.Unmarshal(point, &pointValue) != nil || pointValue.Digest != hashBytes(pointValue.Payload) {
		t.Fatal("CLI point digest")
	}
	snapshot := expect("/snapshots/"+published.SnapshotID, 200)
	if hashBytes(snapshot) != published.SnapshotID {
		t.Fatal("CLI immutable digest")
	}
	expect("/head", 429)
	t.Log("CLI point and snapshot succeed; third mixed-route read receives429")
	time.Sleep(1050 * time.Millisecond)
	expect("/head", 200)
	mode.Store(2)
	expect("/facts/fact-a", 502)
	mode.Store(0)
	t.Log("CLI conflicting lowercase identity refused502")
	second, err := exec.Command(binary, "serve", "--config", configPath).CombinedOutput()
	if err == nil || !bytes.Contains(second, []byte("already held")) {
		t.Fatalf("overlapping CLI accepted: %v %s", err, second)
	}
	time.Sleep(1050 * time.Millisecond)
	mode.Store(1)
	results := make(chan error, 8)
	for i := 0; i < 8; i++ {
		go func() {
			code, b, err := call("/facts/fact-a")
			if err == nil && code != 200 {
				err = fmt.Errorf("code=%d body=%s", code, b)
			}
			results <- err
		}()
		select {
		case <-entered:
		case <-time.After(time.Second):
			close(unblock)
			t.Fatal("request did not reach real upstream", i)
		}
		time.Sleep(510 * time.Millisecond)
	}
	expect("/facts/fact-a", 429)
	expect("/snapshots/"+published.SnapshotID, 429)
	close(unblock)
	for i := 0; i < 8; i++ {
		if err := <-results; err != nil {
			t.Fatal("in-flight read", i, err)
		}
	}
	t.Log("CLI measured load:8 held requests succeed; ninth point and snapshot both429; one real TCP peer")
	time.Sleep(1050 * time.Millisecond)
	mode.Store(3)
	go func() { _, _, err := call("/facts/fact-a"); results <- err }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("shutdown request not started")
	}
	start := time.Now()
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-waited:
		stopped = true
		if err != nil {
			t.Fatalf("CLI shutdown %v %s", err, stderr.String())
		}
	case <-time.After(7 * time.Second):
		t.Fatal("shutdown deadline exceeded")
	}
	<-results
	if bytes.Contains(stderr.Bytes(), []byte("DATA RACE")) {
		t.Fatal(stderr.String())
	}
	t.Logf("CLI active-request cancellation and shutdown completed in %s", time.Since(start))
	if _, err := net.DialTimeout("tcp", ready.Address, 100*time.Millisecond); err == nil {
		t.Fatal("listener still open")
	}
}
