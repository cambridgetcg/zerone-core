package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"time"

	cmtjson "github.com/cometbft/cometbft/libs/json"
	cmttypes "github.com/cometbft/cometbft/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"golang.org/x/sys/unix"
)

// Walk directory descriptors rather than lstat-then-open paths. Every component
// is no-follow. Only the final runtime directory must be owner-only; ancestor
// directories may be standard shared system parents, but may not be symlinks.
func openDir(path string, private bool) *os.File {
	require(filepath.IsAbs(path) && filepath.Clean(path) == path, "unsafe_path")
	fd, e := unix.Open("/", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	require(e == nil, "unsafe_path")
	for _, part := range strings.Split(strings.TrimPrefix(path, "/"), "/") {
		if part == "" {
			continue
		}
		next, e := unix.Openat(fd, part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
		unix.Close(fd)
		require(e == nil, "unsafe_path")
		fd = next
	}
	f := os.NewFile(uintptr(fd), "directory")
	st, e := f.Stat()
	if e != nil {
		f.Close()
		panic(failure("unsafe_path"))
	}
	if private {
		s, ok := st.Sys().(*syscall.Stat_t)
		if !ok || s.Uid != uint32(os.Geteuid()) || st.Mode().Perm() != 0700 {
			f.Close()
			panic(failure("unsafe_path"))
		}
	}
	return f
}
func openPrivateDir(path string) *os.File { return openDir(path, true) }
func openRegular(path string, private bool) *os.File {
	require(filepath.IsAbs(path) && filepath.Clean(path) == path, "unsafe_path")
	d := openDir(filepath.Dir(path), private)
	defer d.Close()
	fd, e := unix.Openat(int(d.Fd()), filepath.Base(path), unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, 0)
	require(e == nil, "unsafe_path")
	f := os.NewFile(uintptr(fd), "input")
	st, e := f.Stat()
	if e != nil || !st.Mode().IsRegular() {
		f.Close()
		panic(failure("unsafe_path"))
	}
	s, ok := st.Sys().(*syscall.Stat_t)
	if !ok || s.Nlink != 1 || (private && (s.Uid != uint32(os.Geteuid()) || st.Mode().Perm() != 0600)) {
		f.Close()
		panic(failure("unsafe_path"))
	}
	return f
}
func openPrivateFile(path string) *os.File { return openRegular(path, true) }
func sameFileSnapshot(a, b os.FileInfo) bool {
	if !os.SameFile(a, b) || a.Size() != b.Size() || a.Mode() != b.Mode() || a.ModTime() != b.ModTime() {
		return false
	}
	x, xok := a.Sys().(*syscall.Stat_t)
	y, yok := b.Sys().(*syscall.Stat_t)
	if !xok || !yok || x.Uid != y.Uid || x.Gid != y.Gid || x.Nlink != y.Nlink {
		return false
	}
	// Darwin and Linux expose the same POSIX change time under different names.
	// Read access time may change, but change time must not.
	for _, field := range []string{"Ctim", "Ctimespec"} {
		left := reflect.ValueOf(x).Elem().FieldByName(field)
		right := reflect.ValueOf(y).Elem().FieldByName(field)
		if left.IsValid() && right.IsValid() {
			return reflect.DeepEqual(left.Interface(), right.Interface())
		}
	}
	return false
}

// Check cancellation between bounded reads as well as after CPU-only work.
// Regular-file syscalls themselves are not preemptible by a Go context; at most
// one 32KiB read is in flight, not an uninterruptible copy of a 1GiB artifact.
const fileReadChunk = 32 << 10

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := budgetError(r.ctx); err != nil {
		return 0, err
	}
	if len(p) > fileReadChunk {
		p = p[:fileReadChunk]
	}
	n, err := r.r.Read(p)
	if canceled := budgetError(r.ctx); canceled != nil {
		return n, canceled
	}
	return n, err
}
func budgetError(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// The wall-clock budget must hold even before the context timer goroutine
	// has had an opportunity to deliver its cancellation.
	if deadline, ok := ctx.Deadline(); ok && !time.Now().Before(deadline) {
		return context.DeadlineExceeded
	}
	return nil
}
func checkDeadline(ctx context.Context) { require(budgetError(ctx) == nil, "limit_exceeded") }

func boundedRead(ctx context.Context, f *os.File, max int64) []byte {
	checkDeadline(ctx)
	before, e := f.Stat()
	require(e == nil && before.Size() <= max, "limit_exceeded")
	b, e := io.ReadAll(io.LimitReader(contextReader{ctx, f}, max+1))
	checkDeadline(ctx)
	require(e == nil && int64(len(b)) <= max, "limit_exceeded")
	after, e := f.Stat()
	require(e == nil && sameFileSnapshot(before, after), "unsafe_path")
	a, ok := after.Sys().(*syscall.Stat_t)
	require(ok && a.Nlink == 1, "unsafe_path")
	checkDeadline(ctx)
	return b
}
func readPrivate(ctx context.Context, path string, max int64) []byte {
	checkDeadline(ctx)
	f := openPrivateFile(path)
	defer f.Close()
	return boundedRead(ctx, f, max)
}
func readPublic(ctx context.Context, path string, max int64) []byte {
	checkDeadline(ctx)
	f := openRegular(path, false)
	defer f.Close()
	return boundedRead(ctx, f, max)
}
func checkNewPrivatePath(path string) {
	require(filepath.IsAbs(path) && filepath.Clean(path) == path, "unsafe_path")
	d := openPrivateDir(filepath.Dir(path))
	defer d.Close()
	var st unix.Stat_t
	err := unix.Fstatat(int(d.Fd()), filepath.Base(path), &st, unix.AT_SYMLINK_NOFOLLOW)
	if err == nil {
		panic(failure("already_exists"))
	}
	require(err == unix.ENOENT, "unsafe_path")
}

func writePrivateExclusive(path string, b []byte) {
	require(filepath.IsAbs(path) && filepath.Clean(path) == path, "unsafe_path")
	d := openPrivateDir(filepath.Dir(path))
	defer d.Close()
	fd, e := unix.Openat(int(d.Fd()), filepath.Base(path), unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0600)
	if e == unix.EEXIST {
		panic(failure("already_exists"))
	}
	require(e == nil, "unsafe_path")
	f := os.NewFile(uintptr(fd), "signed transaction")
	defer f.Close()
	n, e := f.Write(b)
	require(e == nil && n == len(b) && f.Sync() == nil, "signing_unknown")
	st, e := f.Stat()
	require(e == nil && st.Mode().Perm() == 0600, "unsafe_path")
	// Directory persistence is part of success. Failed persistence never deletes a
	// possible signed result or reopens the caller's one-use signing allowance.
	require(d.Sync() == nil, "signing_unknown")
}
func hashArtifact(ctx context.Context, path string) string {
	checkDeadline(ctx)
	f := openRegular(path, false)
	defer f.Close()
	before, e := f.Stat()
	require(e == nil && before.Size() <= 1<<30, "limit_exceeded")
	h := sha256.New()
	n, e := io.Copy(h, io.LimitReader(contextReader{ctx, f}, (1<<30)+1))
	checkDeadline(ctx)
	require(e == nil && n <= 1<<30, "limit_exceeded")
	after, e := f.Stat()
	require(e == nil && sameFileSnapshot(before, after), "unsafe_path")
	checkDeadline(ctx)
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

type trustedArtifacts struct{ genesis *cmttypes.GenesisDoc }

func loadTrust(ctx context.Context, r object, opts options) trustedArtifacts {
	require(opts.trustFile != "", "profile_mismatch")
	trust := parseJSON(readPrivate(ctx, opts.trustFile, maxJSON))
	exact(trust, "protocol profile_id node genesis_file runtime_file source_manifest_file disposable_test")
	p := obj(r["profile"])
	require(str(trust, "protocol") == "zerone-seed-trust/0.1" && str(trust, "profile_id") == str(p, "profile_id"), "profile_mismatch")
	trustedNode := obj(trust["node"])
	require(str(trustedNode, "node_trust_id") == str(obj(r["policy"]), "node_trust_id"), "profile_mismatch")
	if n, ok := r["node"]; ok {
		require(bytes.Equal(canonical(trustedNode), canonical(n)), "profile_mismatch")
	}
	disposable, ok := trust["disposable_test"].(bool)
	require(ok, "invalid_request")
	require(disposable == opts.disposable, "profile_mismatch")
	if disposable {
		require(strings.HasPrefix(str(p, "chain_reference"), "seed-local-") && str(obj(trust["node"]), "mode") == "local", "profile_mismatch")
	}
	executable, e := os.Executable()
	require(e == nil, "profile_mismatch")
	// Executable resolution is explicit self-measurement, not ambient key discovery.
	executable, e = filepath.EvalSymlinks(executable)
	require(e == nil, "profile_mismatch")
	require(hashArtifact(ctx, executable) == str(p, "helper_sha256") && hashArtifact(ctx, str(trust, "runtime_file")) == str(p, "runtime_sha256"), "profile_mismatch")
	source := readPublic(ctx, str(trust, "source_manifest_file"), maxJSON)
	require(digest(source) == str(p, "source_digest"), "profile_mismatch")
	require(bytes.Equal(canonical(parseJSON(source)), source), "profile_mismatch")
	genesis := readPublic(ctx, str(trust, "genesis_file"), maxJSON)
	require(digest(genesis) == str(p, "genesis_hash"), "profile_mismatch")
	// SDK v0.53 CLI files are AppGenesis (numeric initial_height, consensus
	// envelope). Decode only the already bounded and hash-pinned bytes, never
	// reopen the path through AppGenesisFromFile or rewrite its trusted encoding.
	appGenesis, e := genutiltypes.AppGenesisFromReader(bytes.NewReader(genesis))
	require(e == nil && appGenesis.ValidateAndComplete() == nil, "profile_mismatch")
	gd, e := appGenesis.ToGenesisDoc()
	require(e == nil && gd.ChainID == str(p, "chain_reference"), "profile_mismatch")
	checkDeadline(ctx)
	return trustedArtifacts{genesis: gd}
}
func sameGenesis(a, b *cmttypes.GenesisDoc) bool {
	if a == nil || b == nil {
		return false
	}
	x, e := cmtjson.Marshal(a)
	if e != nil {
		return false
	}
	y, e := cmtjson.Marshal(b)
	return e == nil && bytes.Equal(x, y)
}
