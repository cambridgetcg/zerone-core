// Package journal provides the local, append-only filesystem boundary for
// agent-collaboration receipts. It performs no network access and never
// replaces an existing journal artifact.
package journal

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"

	"golang.org/x/sys/unix"
)

// LockedJournal is a short-lived publication capability created only by
// WithAppendLock. Its unexported state prevents construction by callers, and
// the capability expires before the append lock is released.
type LockedJournal struct {
	journalPath string
	lockPath    string
	lockOwner   fs.FileInfo
	state       *lockedJournalState
}

type lockedJournalState struct {
	mu        sync.Mutex
	active    bool
	published bool
}

const (
	receiptsDirectory = "receipts"
	appendLockName    = ".append.lock"
	receiptNameLength = 20 + 1 + 64 + len(".json")
	maxReceiptEntries = 4096
	maxReceiptBytes   = 128 << 10
)

// ReadRegular reads at most maximum bytes from path. The final path component
// must be a regular file and must not be a symbolic link. When private is true,
// group/world permission bits and additional hard links are also rejected.
func ReadRegular(path string, maximum int, private bool) ([]byte, error) {
	if maximum < 0 {
		return nil, errors.New("maximum byte count must not be negative")
	}
	if uint64(maximum) == uint64(math.MaxInt64) {
		return nil, errors.New("maximum byte count is too large")
	}

	file, info, linkCount, err := openRegularNoFollow(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	limit := int64(maximum)
	if info.Size() > limit {
		return nil, fmt.Errorf("%q exceeds %d-byte limit", path, maximum)
	}
	if private {
		if info.Mode().Perm()&0o077 != 0 {
			return nil, fmt.Errorf("%q has group or world permission bits set", path)
		}
		if linkCount != 1 {
			return nil, fmt.Errorf("%q has %d hard links; private files must have exactly one", path, linkCount)
		}
	}

	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", path, err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%q exceeds %d-byte limit", path, maximum)
	}
	return data, nil
}

// CreateJournal creates a new private journal directory and its receipts
// directory. It refuses to adopt any pre-existing path, including a symlink.
func CreateJournal(path string) error {
	if err := os.Mkdir(path, 0o700); err != nil {
		return fmt.Errorf("create journal %q: %w", path, err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		cleanupErr := os.Remove(path)
		return errors.Join(fmt.Errorf("set journal permissions: %w", err), wrapCleanup(path, cleanupErr))
	}

	receiptsPath := filepath.Join(path, receiptsDirectory)
	if err := os.Mkdir(receiptsPath, 0o700); err != nil {
		cleanupErr := os.Remove(path)
		return errors.Join(fmt.Errorf("create receipts directory: %w", err), wrapCleanup(path, cleanupErr))
	}
	if err := os.Chmod(receiptsPath, 0o700); err != nil {
		removeReceiptsErr := os.Remove(receiptsPath)
		removeJournalErr := os.Remove(path)
		return errors.Join(
			fmt.Errorf("set receipts permissions: %w", err),
			wrapCleanup(receiptsPath, removeReceiptsErr),
			wrapCleanup(path, removeJournalErr),
		)
	}
	return nil
}

// WithAppendLock runs fn while holding the journal's exclusive append lock.
// An existing lock is always treated as authoritative and is never broken.
// The lock is deliberately left behind if fn panics. After fn returns, the
// lock is removed only when it is still the directory created by this call.
func WithAppendLock(path string, fn func(*LockedJournal) error) error {
	if fn == nil {
		return errors.New("append callback must not be nil")
	}
	if err := validateDirectory(path); err != nil {
		return fmt.Errorf("open journal: %w", err)
	}

	lockPath := filepath.Join(path, appendLockName)
	if err := os.Mkdir(lockPath, 0o700); err != nil {
		return fmt.Errorf("acquire append lock %q: %w", lockPath, err)
	}
	lockDirectory, err := openDirectoryNoFollow(lockPath)
	if err != nil {
		// Without a stable descriptor, ownership cannot be established safely.
		// Leave the lock in place and require explicit operator inspection.
		return fmt.Errorf("inspect acquired append lock: %w", err)
	}
	lockOpen := true
	defer func() {
		if lockOpen {
			_ = lockDirectory.Close()
		}
	}()
	if err := lockDirectory.Chmod(0o700); err != nil {
		owner, statErr := lockDirectory.Stat()
		var cleanupErr error
		if statErr == nil {
			cleanupErr = removeOwnedLock(lockPath, owner)
		}
		closeErr := lockDirectory.Close()
		lockOpen = false
		return errors.Join(
			fmt.Errorf("set append-lock permissions: %w", err),
			wrapCleanup(lockPath, cleanupErr),
			statErr,
			closeErr,
		)
	}
	owner, err := lockDirectory.Stat()
	if err != nil {
		return fmt.Errorf("inspect acquired append lock: %w", err)
	}

	locked := &LockedJournal{
		journalPath: path,
		lockPath:    lockPath,
		lockOwner:   owner,
		state:       &lockedJournalState{active: true},
	}
	defer locked.invalidate()
	callbackErr := fn(locked)
	locked.invalidate()
	cleanupErr := removeOwnedLock(lockPath, owner)
	closeErr := lockDirectory.Close()
	lockOpen = false
	if cleanupErr == nil {
		cleanupErr = syncDirectory(path)
	}
	return errors.Join(callbackErr, cleanupErr, closeErr)
}

// ValidateLayout requires the exact closed journal-root layout. When
// appendLockHeld is true, the append lock must be present; otherwise it must
// be absent. Unexpected root files fail closed so secrets cannot hide beside
// a journal that still reports valid.
func ValidateLayout(path string, appendLockHeld bool) error {
	directory, err := openDirectoryNoFollow(path)
	if err != nil {
		return fmt.Errorf("open journal root: %w", err)
	}
	info, statErr := directory.Stat()
	entries, readErr := directory.ReadDir(4)
	closeErr := directory.Close()
	if errors.Is(readErr, io.EOF) {
		readErr = nil
	}
	if err := errors.Join(statErr, readErr, closeErr); err != nil {
		return fmt.Errorf("inspect journal root: %w", err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return errors.New("journal root has group or world permission bits set")
	}
	want := map[string]bool{
		"manifest.json":   false,
		receiptsDirectory: false,
	}
	if appendLockHeld {
		want[appendLockName] = false
	}
	for _, entry := range entries {
		if _, ok := want[entry.Name()]; !ok {
			return fmt.Errorf("unexpected journal-root entry %q", entry.Name())
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("journal-root entry %q is a symbolic link", entry.Name())
		}
		entryInfo, err := entry.Info()
		if err != nil {
			return fmt.Errorf("inspect journal-root entry %q: %w", entry.Name(), err)
		}
		switch entry.Name() {
		case "manifest.json":
			if !entryInfo.Mode().IsRegular() {
				return errors.New("journal manifest is not a regular file")
			}
		case receiptsDirectory, appendLockName:
			if !entryInfo.IsDir() {
				return fmt.Errorf("journal-root entry %q is not a directory", entry.Name())
			}
			if entryInfo.Mode().Perm()&0o077 != 0 {
				return fmt.Errorf("journal-root directory %q has group or world permission bits set", entry.Name())
			}
		}
		want[entry.Name()] = true
	}
	for name, found := range want {
		if !found {
			return fmt.Errorf("journal root is missing required entry %q", name)
		}
	}
	return nil
}

// CreateNoReplace durably stages data in the target directory and publishes it
// with a hard link. The publication fails if path already names anything.
func CreateNoReplace(path string, data []byte, perm fs.FileMode) error {
	if perm != perm.Perm() {
		return fmt.Errorf("file mode %v contains non-permission bits", perm)
	}
	base := filepath.Base(path)
	return createNoReplace(path, data, perm, "."+base+".tmp-*")
}

// ReceiptPaths returns the journal's receipt paths in canonical filename
// order. Any unexpected, duplicate-sequence, symlink, or non-regular entry
// causes the entire listing to fail closed.
func ReceiptPaths(journal string) ([]string, error) {
	journalDirectory, err := openDirectoryNoFollow(journal)
	if err != nil {
		return nil, fmt.Errorf("open journal root: %w", err)
	}
	if err := journalDirectory.Close(); err != nil {
		return nil, fmt.Errorf("close journal root: %w", err)
	}
	receiptsPath := filepath.Join(journal, receiptsDirectory)
	directory, err := openDirectoryNoFollow(receiptsPath)
	if err != nil {
		return nil, fmt.Errorf("open receipts directory: %w", err)
	}
	entries, readErr := directory.ReadDir(maxReceiptEntries + 1)
	closeErr := directory.Close()
	if errors.Is(readErr, io.EOF) {
		readErr = nil
	}
	if err := errors.Join(readErr, closeErr); err != nil {
		return nil, fmt.Errorf("read receipts directory: %w", err)
	}
	if len(entries) > maxReceiptEntries {
		return nil, fmt.Errorf("receipts directory exceeds %d-entry limit", maxReceiptEntries)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	paths := make([]string, 0, len(entries))
	seenSequences := make(map[uint64]string, len(entries))
	for _, entry := range entries {
		sequence, ok := parseReceiptName(entry.Name())
		if !ok {
			return nil, fmt.Errorf("unexpected receipt entry %q", entry.Name())
		}
		if previous, exists := seenSequences[sequence]; exists {
			return nil, fmt.Errorf("receipt sequence %d appears in both %q and %q", sequence, previous, entry.Name())
		}
		seenSequences[sequence] = entry.Name()
		expectedSequence := uint64(len(paths) + 1)
		if sequence != expectedSequence {
			return nil, fmt.Errorf("receipt sequence is not contiguous: expected %d, got %d", expectedSequence, sequence)
		}

		if entry.Type()&fs.ModeSymlink != 0 {
			return nil, fmt.Errorf("receipt entry %q is a symbolic link", entry.Name())
		}
		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("inspect receipt entry %q: %w", entry.Name(), err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("receipt entry %q is not a regular file", entry.Name())
		}
		paths = append(paths, filepath.Join(receiptsPath, entry.Name()))
	}
	return paths, nil
}

// PublishReceipt publishes data as a private, immutable-by-name journal entry.
// It is available only while this capability's append-lock callback is active.
// digest must be exactly 64 lowercase hexadecimal characters.
func (locked *LockedJournal) PublishReceipt(sequence uint64, digest string, data []byte) (string, error) {
	if locked == nil {
		return "", errors.New("receipt publication requires an active append-lock capability")
	}
	if locked.state == nil {
		return "", errors.New("receipt publication requires an active append-lock capability")
	}
	locked.state.mu.Lock()
	defer locked.state.mu.Unlock()
	if !locked.state.active || locked.journalPath == "" || locked.lockOwner == nil {
		return "", errors.New("receipt publication requires an active append-lock capability")
	}
	if locked.state.published {
		return "", errors.New("append-lock capability has already attempted its one receipt publication")
	}
	currentLock, err := os.Lstat(locked.lockPath)
	if err != nil {
		return "", fmt.Errorf("inspect append-lock capability: %w", err)
	}
	if !currentLock.IsDir() || !os.SameFile(locked.lockOwner, currentLock) {
		return "", errors.New("append-lock capability no longer owns its lock")
	}
	if sequence == 0 {
		return "", errors.New("receipt sequence must be positive")
	}
	if !isLowerHex64(digest) {
		return "", errors.New("receipt digest must be exactly 64 lowercase hexadecimal characters")
	}
	if len(data) == 0 || len(data) > maxReceiptBytes {
		return "", fmt.Errorf("receipt document must be between 1 and %d bytes", maxReceiptBytes)
	}
	receiptsPath := filepath.Join(locked.journalPath, receiptsDirectory)
	if err := validateDirectory(receiptsPath); err != nil {
		return "", fmt.Errorf("open receipts directory: %w", err)
	}
	// Consume the one-publication capability before touching the filesystem.
	// An error after publication may be ambiguous, so retrying through this
	// capability would be unsafe.
	locked.state.published = true

	name := fmt.Sprintf("%020d-%s.json", sequence, digest)
	path := filepath.Join(receiptsPath, name)
	if err := createNoReplace(path, data, 0o600, ".receipt-*.tmp"); err != nil {
		return "", fmt.Errorf("publish receipt %q: %w", name, err)
	}
	return path, nil
}

func (locked *LockedJournal) invalidate() {
	if locked == nil {
		return
	}
	if locked.state == nil {
		return
	}
	locked.state.mu.Lock()
	locked.state.active = false
	locked.state.mu.Unlock()
}

func createNoReplace(path string, data []byte, perm fs.FileMode, pattern string) (returnErr error) {
	directoryPath := filepath.Dir(path)
	if err := validateDirectory(directoryPath); err != nil {
		return fmt.Errorf("open destination directory: %w", err)
	}

	temporary, err := os.CreateTemp(directoryPath, pattern)
	if err != nil {
		return fmt.Errorf("create publication temporary file: %w", err)
	}
	temporaryPath := temporary.Name()
	closed := false
	defer func() {
		if !closed {
			if err := temporary.Close(); err != nil {
				returnErr = errors.Join(returnErr, fmt.Errorf("close publication temporary file: %w", err))
			}
		}
		if temporaryPath != "" {
			if err := os.Remove(temporaryPath); err != nil {
				returnErr = errors.Join(returnErr, fmt.Errorf("remove publication temporary file: %w", err))
			} else if err := syncDirectory(directoryPath); err != nil {
				returnErr = errors.Join(returnErr, fmt.Errorf("sync destination after temporary cleanup: %w", err))
			}
		}
	}()

	if err := temporary.Chmod(perm); err != nil {
		return fmt.Errorf("set publication permissions: %w", err)
	}
	if err := writeAll(temporary, data); err != nil {
		return fmt.Errorf("write publication temporary file: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync publication temporary file: %w", err)
	}
	closeErr := temporary.Close()
	closed = true
	if closeErr != nil {
		return fmt.Errorf("close publication temporary file: %w", closeErr)
	}

	if err := os.Link(temporaryPath, path); err != nil {
		return fmt.Errorf("link publication without replacement: %w", err)
	}
	if err := syncDirectory(directoryPath); err != nil {
		return fmt.Errorf("sync published name: %w", err)
	}
	return nil
}

func openRegularNoFollow(path string) (*os.File, fs.FileInfo, uint64, error) {
	initial, err := os.Lstat(path)
	if err != nil {
		return nil, nil, 0, err
	}
	if !initial.Mode().IsRegular() {
		return nil, nil, 0, fmt.Errorf("%q is not a regular file", path)
	}

	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NONBLOCK|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("open %q without following links: %w", path, err)
	}
	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		_ = unix.Close(fd)
		return nil, nil, 0, fmt.Errorf("open %q: invalid file descriptor", path)
	}

	opened, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, nil, 0, fmt.Errorf("inspect opened file %q: %w", path, err)
	}
	if !opened.Mode().IsRegular() || !os.SameFile(initial, opened) {
		_ = file.Close()
		return nil, nil, 0, fmt.Errorf("%q changed while it was being opened", path)
	}
	var raw unix.Stat_t
	if err := unix.Fstat(fd, &raw); err != nil {
		_ = file.Close()
		return nil, nil, 0, fmt.Errorf("inspect hard-link count for %q: %w", path, err)
	}
	return file, opened, uint64(raw.Nlink), nil
}

func openDirectoryNoFollow(path string) (*os.File, error) {
	if path == "" {
		return nil, errors.New("directory path must not be empty")
	}
	cleanPath := filepath.Clean(path)
	initial, err := os.Lstat(cleanPath)
	if err != nil {
		return nil, err
	}
	if !initial.IsDir() {
		return nil, fmt.Errorf("%q is not a directory", cleanPath)
	}

	fd, err := unix.Open(cleanPath, unix.O_RDONLY|unix.O_NONBLOCK|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
	if err != nil {
		return nil, fmt.Errorf("open %q without following links: %w", cleanPath, err)
	}
	directory := os.NewFile(uintptr(fd), cleanPath)
	if directory == nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("open %q: invalid directory descriptor", cleanPath)
	}
	opened, err := directory.Stat()
	if err != nil {
		_ = directory.Close()
		return nil, fmt.Errorf("inspect opened directory %q: %w", cleanPath, err)
	}
	if !opened.IsDir() || !os.SameFile(initial, opened) {
		_ = directory.Close()
		return nil, fmt.Errorf("%q changed while it was being opened", cleanPath)
	}
	return directory, nil
}

func validateDirectory(path string) error {
	directory, err := openDirectoryNoFollow(path)
	if err != nil {
		return err
	}
	return directory.Close()
}

func syncDirectory(path string) error {
	directory, err := openDirectoryNoFollow(path)
	if err != nil {
		return err
	}
	syncErr := directory.Sync()
	closeErr := directory.Close()
	if errors.Is(syncErr, unix.EINVAL) || errors.Is(syncErr, unix.ENOTSUP) {
		syncErr = nil
	}
	return errors.Join(syncErr, closeErr)
}

func removeOwnedLock(path string, owner fs.FileInfo) error {
	current, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect append lock before release: %w", err)
	}
	if !current.IsDir() || !os.SameFile(owner, current) {
		return fmt.Errorf("append lock %q is no longer owned by this operation; refusing to remove it", path)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("release append lock %q: %w", path, err)
	}
	return nil
}

func parseReceiptName(name string) (uint64, bool) {
	if len(name) != receiptNameLength || name[20] != '-' || name[len(name)-len(".json"):] != ".json" {
		return 0, false
	}
	for _, character := range name[:20] {
		if character < '0' || character > '9' {
			return 0, false
		}
	}
	digest := name[21 : len(name)-len(".json")]
	if !isLowerHex64(digest) {
		return 0, false
	}
	sequence, err := strconv.ParseUint(name[:20], 10, 64)
	if err != nil || sequence == 0 || fmt.Sprintf("%020d", sequence) != name[:20] {
		return 0, false
	}
	return sequence, true
}

func isLowerHex64(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, character := range value {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

func writeAll(file *os.File, data []byte) error {
	for len(data) > 0 {
		written, err := file.Write(data)
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
		data = data[written:]
	}
	return nil
}

func wrapCleanup(path string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("clean up %q: %w", path, err)
}
