package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

const maxClients = 4096
const maxConnections = 32

type clientWindow struct {
	starts [clientRate]time.Time
	count  int
}
type singletonAdmission struct {
	mu      sync.Mutex
	clients map[netip.Prefix]clientWindow
	active  int
	now     func() time.Time
}

func newSingletonAdmission() *singletonAdmission {
	return &singletonAdmission{clients: make(map[netip.Prefix]clientWindow), now: time.Now}
}
func peerClient(remote string) (netip.Prefix, error) {
	host, _, err := net.SplitHostPort(remote)
	if err != nil {
		return netip.Prefix{}, err
	}
	addr, err := netip.ParseAddr(host)
	if err != nil || addr.Zone() != "" {
		return netip.Prefix{}, errors.New("TCP peer IP required")
	}
	addr = addr.Unmap()
	bits := 128
	if addr.Is4() {
		bits = 32
	} else {
		bits = 64
	}
	return netip.PrefixFrom(addr, bits).Masked(), nil
}
func (a *singletonAdmission) Acquire(ctx context.Context, r *http.Request, rate, concurrency int) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if rate != clientRate || concurrency != globalConcurrency {
		return nil, errors.New("unsupported limits")
	}
	client, err := peerClient(r.RemoteAddr)
	if err != nil {
		return nil, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	now := a.now()
	// At most 4096 entries are scanned; active-window identities are NEVER evicted
	// to admit a new client. Capacity exhaustion fails closed, including snapshots.
	if len(a.clients) >= maxClients {
		for key, window := range a.clients {
			if now.Sub(window.starts[window.count-1]) >= time.Second {
				delete(a.clients, key)
			}
		}
	}
	window, exists := a.clients[client]
	kept := clientWindow{}
	for i := 0; i < window.count; i++ {
		if now.Sub(window.starts[i]) < time.Second {
			kept.starts[kept.count] = window.starts[i]
			kept.count++
		}
	}
	if kept.count >= clientRate || a.active >= globalConcurrency || (!exists && len(a.clients) >= maxClients) {
		return nil, errAdmissionLimited
	}
	kept.starts[kept.count] = now
	kept.count++
	a.clients[client] = kept
	a.active++
	var once sync.Once
	return func() { once.Do(func() { a.mu.Lock(); a.active--; a.mu.Unlock() }) }, nil
}
func boundedTransport() *http.Transport {
	return &http.Transport{
		Proxy:        nil, // Do not turn environment variables into another origin.
		DialContext:  (&net.Dialer{Timeout: 2 * time.Second, KeepAlive: 15 * time.Second}).DialContext,
		MaxIdleConns: 16, MaxIdleConnsPerHost: 8, MaxConnsPerHost: 8, IdleConnTimeout: 5 * time.Second,
		ResponseHeaderTimeout: pointTimeout, TLSHandshakeTimeout: 2 * time.Second, MaxResponseHeaderBytes: 8192, DisableCompression: true,
	}
}

type serveConfig struct {
	Schema                 string `json:"schema"`
	Profile                string `json:"profile"`
	Listen                 string `json:"listen"`
	RESTOrigin             string `json:"restOrigin"`
	RPCOrigin              string `json:"rpcOrigin"`
	ChainID                string `json:"chainId"`
	SnapshotRoot           string `json:"snapshotRoot"`
	HeadID                 string `json:"headId"`
	InstanceID             string `json:"instanceId"`
	DeploymentReviewPath   string `json:"deploymentReviewPath"`
	DeploymentReviewSHA256 string `json:"deploymentReviewSha256"`
	TLSCertificatePath     string `json:"tlsCertificatePath"`
	TLSPrivateKeyPath      string `json:"tlsPrivateKeyPath"`
}

// This is an operator-supplied review receipt, NOT an implementation that can
// discover other hosts or independently verify an inventory. Production remains
// subject to the separately performed inventory/drain/no-overlap review.
type deploymentReview struct {
	Schema                   string   `json:"schema"`
	InstanceID               string   `json:"instanceId"`
	ActiveInstances          []string `json:"activeInstances"`
	PreviousInstancesDrained bool     `json:"previousInstancesDrained"`
	NoOverlap                bool     `json:"noOverlap"`
	DirectPeerIngress        bool     `json:"directPeerIngress"`
	VerifiedAt               string   `json:"verifiedAt"`
	ValidUntil               string   `json:"validUntil"`
	Scope                    string   `json:"scope"`
}

func validateConfig(c serveConfig, now time.Time) (time.Time, error) {
	if c.Schema != "zerone.knowledge-read-config/v1" || c.ChainID != "zerone-1" || !factIDPattern.MatchString(c.InstanceID) || !digestPattern.MatchString(c.HeadID) {
		return time.Time{}, errors.New("invalid explicit configuration identity")
	}
	host, port, err := net.SplitHostPort(c.Listen)
	if err != nil {
		return time.Time{}, err
	}
	ip, err := netip.ParseAddr(host)
	if err != nil || ip.Zone() != "" {
		return time.Time{}, errors.New("literal listen IP required")
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 0 || n > 65535 {
		return time.Time{}, errors.New("invalid listen port")
	}
	rest, err := fixedOrigin(c.RESTOrigin)
	if err != nil {
		return time.Time{}, err
	}
	rpc, err := fixedOrigin(c.RPCOrigin)
	if err != nil {
		return time.Time{}, err
	}
	switch c.Profile {
	case "loopback-preview":
		if !ip.IsLoopback() || c.DeploymentReviewPath != "" || c.DeploymentReviewSHA256 != "" || c.TLSCertificatePath != "" || c.TLSPrivateKeyPath != "" {
			return time.Time{}, errors.New("preview requires loopback and no invented deployment review")
		}
		for _, u := range []string{rest.Hostname(), rpc.Hostname()} {
			addr, err := netip.ParseAddr(u)
			if err != nil || !addr.IsLoopback() {
				return time.Time{}, errors.New("preview upstreams must be literal loopback IPs")
			}
		}
		if rest.Scheme != "http" || rpc.Scheme != "http" {
			return time.Time{}, errors.New("preview requires explicit local HTTP origins")
		}
		return time.Time{}, nil
	case "singleton-direct":
		if rest.Scheme != "https" || rpc.Scheme != "https" || n == 0 || !digestPattern.MatchString(c.DeploymentReviewSHA256) || c.TLSCertificatePath == "" || c.TLSPrivateKeyPath == "" {
			return time.Time{}, errors.New("singleton requires fixed HTTPS origins, nonzero port and reviewed inventory digest")
		}
		body, err := readFileBounded(c.DeploymentReviewPath, 4096)
		if err != nil {
			return time.Time{}, err
		}
		if hashBytes(body) != c.DeploymentReviewSHA256 {
			return time.Time{}, errors.New("deployment review digest mismatch")
		}
		var review deploymentReview
		if err := decodeExact(body, &review, 4096); err != nil {
			return time.Time{}, err
		}
		verified, err := time.Parse(time.RFC3339Nano, review.VerifiedAt)
		if err != nil {
			return time.Time{}, err
		}
		until, err := time.Parse(time.RFC3339Nano, review.ValidUntil)
		if err != nil {
			return time.Time{}, err
		}
		if review.Schema != "zerone.knowledge-singleton-review/v1" || review.InstanceID != c.InstanceID || len(review.ActiveInstances) != 1 || review.ActiveInstances[0] != c.InstanceID || !review.PreviousInstancesDrained || !review.NoOverlap || !review.DirectPeerIngress || review.Scope != "operator-reviewed singleton inventory and drain; assertion, not independently verified by this server" || now.Sub(verified) > 5*time.Minute || verified.After(now) || !until.After(now) || until.Sub(verified) > time.Hour {
			return time.Time{}, errors.New("singleton inventory/drain/no-overlap review missing, inconsistent or expired")
		}
		return until, nil
	default:
		return time.Time{}, errors.New("no serving profile selected")
	}
}

// Bound accepted sockets without starting an unbounded goroutine per accept.
// Kernel backlog is outside this process bound. Closing unblocks a full listener.
type cappedListener struct {
	net.Listener
	slots  chan struct{}
	closed chan struct{}
	once   sync.Once
}
type cappedConn struct {
	net.Conn
	release func()
	once    sync.Once
}

func (c *cappedConn) Close() error { err := c.Conn.Close(); c.once.Do(c.release); return err }
func (l *cappedListener) Accept() (net.Conn, error) {
	select {
	case l.slots <- struct{}{}:
	case <-l.closed:
		return nil, net.ErrClosed
	}
	c, err := l.Listener.Accept()
	if err != nil {
		<-l.slots
		return nil, err
	}
	return &cappedConn{Conn: c, release: func() { <-l.slots }}, nil
}
func (l *cappedListener) Close() error {
	l.once.Do(func() { close(l.closed) })
	return l.Listener.Close()
}
func serve(ctx context.Context, c serveConfig, out io.Writer) error {
	validUntil, err := validateConfig(c, time.Now())
	if err != nil {
		return err
	}
	root, err := openStore(c.SnapshotRoot)
	if err != nil {
		return err
	}
	defer root.Close()
	singletonLock, err := lockStore(root, ".serve.lock")
	if err != nil {
		return err
	}
	defer unlockStore(singletonLock)
	transport := boundedTransport()
	defer transport.CloseIdleConnections()
	f, err := newFacade(c.RESTOrigin, c.ChainID, newSingletonAdmission(), transport)
	if err != nil {
		return err
	}
	f.statusOrigin, err = fixedOrigin(c.RPCOrigin)
	if err != nil {
		return err
	}
	f.validUntil = validUntil
	if err := loadPublications(f, root, c.HeadID); err != nil {
		return err
	}
	// Admission history is in-memory. Enforce a full empty one-second window on
	// EVERY restart while holding the singleton lock; never overlap two instances.
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
	}
	var tlsConfig *tls.Config
	if c.Profile == "singleton-direct" {
		cert, err := readFileBounded(c.TLSCertificatePath, 16384)
		if err != nil {
			return err
		}
		key, err := readFileBounded(c.TLSPrivateKeyPath, 16384)
		if err != nil {
			return err
		}
		pair, err := tls.X509KeyPair(cert, key)
		if err != nil {
			return errors.New("invalid operator-provisioned TLS certificate/key")
		}
		tlsConfig = &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{pair}, NextProtos: []string{"http/1.1"}}
	}
	raw, err := net.Listen("tcp", c.Listen)
	if err != nil {
		return err
	}
	listener := &cappedListener{Listener: raw, slots: make(chan struct{}, maxConnections), closed: make(chan struct{})}
	server := &http.Server{Handler: f, ReadHeaderTimeout: 2 * time.Second, ReadTimeout: pointTimeout, WriteTimeout: pointTimeout, IdleTimeout: pointTimeout, MaxHeaderBytes: 8192, BaseContext: func(net.Listener) context.Context { return ctx }}
	ready, _ := compactJSON(map[string]string{"event": "listening", "address": listener.Addr().String(), "profile": c.Profile, "limitScope": "one process, one instance; TCP peer IPv4 or IPv6 /64; no forwarded-header identity"})
	if _, err := fmt.Fprintln(out, string(ready)); err != nil {
		listener.Close()
		return err
	}
	stopped := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			shutdown, cancel := context.WithTimeout(context.Background(), pointTimeout)
			defer cancel()
			if server.Shutdown(shutdown) != nil {
				_ = server.Close()
			}
		case <-stopped:
		}
	}()
	var servingListener net.Listener = listener
	if tlsConfig != nil {
		servingListener = tls.NewListener(listener, tlsConfig)
	}
	err = server.Serve(servingListener)
	close(stopped)
	// Serve returns when listeners close; wait for active responses before releasing
	// the singleton lock even during a graceful shutdown.
	shutdown, cancel := context.WithTimeout(context.Background(), pointTimeout)
	defer cancel()
	if server.Shutdown(shutdown) != nil {
		_ = server.Close()
	}
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
func run(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("NO_GO: no listener started; use publish or serve with an explicit --config")
	}
	command := args[0]
	if command != "publish" && command != "serve" {
		return errors.New("expected publish or serve")
	}
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	configPath := flags.String("config", "", "explicit operator config")
	projectionPath := flags.String("projection", "", "canonical actual-query projection file (publish only)")
	metadataPath := flags.String("metadata", "", "explicit observation metadata file (publish only)")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 || *configPath == "" {
		return errors.New("explicit --config required; no positional arguments")
	}
	body, err := readFileBounded(*configPath, 8192)
	if err != nil {
		return err
	}
	var config serveConfig
	if err := decodeExact(body, &config, 8192); err != nil {
		return err
	}
	if _, err := validateConfig(config, time.Now()); err != nil {
		return err
	}
	if command == "serve" {
		if *projectionPath != "" || *metadataPath != "" {
			return errors.New("serve never reads caller publication files")
		}
		return serve(ctx, config, out)
	}
	raw, err := readFileBounded(*projectionPath, snapshotBytes)
	if err != nil {
		return err
	}
	meta, err := readFileBounded(*metadataPath, 4096)
	if err != nil {
		return err
	}
	var m publicationMetadata
	if err := decodeExact(meta, &m, 4096); err != nil {
		return err
	}
	if m.RESTOrigin != config.RESTOrigin || m.RPCOrigin != config.RPCOrigin || m.ChainID != config.ChainID {
		return errors.New("publication metadata differs from operator-approved config")
	}
	id, head, err := publish(config.SnapshotRoot, raw, m, time.Now())
	if err != nil {
		return err
	}
	result, _ := compactJSON(map[string]string{"snapshotId": id, "headId": head, "scope": "immutable operator-asserted observation; no chain proof or activation"})
	_, err = fmt.Fprintln(out, string(result))
	return err
}
func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := run(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
