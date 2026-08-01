package journal

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestReadRegularBoundsAndPrivatePolicy(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "private.json")
	if err := os.WriteFile(path, []byte("abcd"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatalf("set fixture mode: %v", err)
	}

	data, err := ReadRegular(path, 4, true)
	if err != nil {
		t.Fatalf("read exact bound: %v", err)
	}
	if string(data) != "abcd" {
		t.Fatalf("read bytes = %q, want abcd", data)
	}
	if _, err := ReadRegular(path, 3, true); err == nil {
		t.Fatal("expected oversized input rejection")
	}
	if _, err := ReadRegular(path, -1, true); err == nil {
		t.Fatal("expected negative bound rejection")
	}

	if err := os.Chmod(path, 0o640); err != nil {
		t.Fatalf("set group-readable mode: %v", err)
	}
	if _, err := ReadRegular(path, 4, true); err == nil {
		t.Fatal("expected private permission rejection")
	}
	if _, err := ReadRegular(path, 4, false); err != nil {
		t.Fatalf("non-private read should allow group permissions: %v", err)
	}

	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatalf("restore private mode: %v", err)
	}
	alias := filepath.Join(directory, "private-alias.json")
	if err := os.Link(path, alias); err != nil {
		t.Skipf("hard links are unavailable on this filesystem: %v", err)
	}
	if _, err := ReadRegular(path, 4, true); err == nil {
		t.Fatal("expected private hard-link rejection")
	}
	if _, err := ReadRegular(path, 4, false); err != nil {
		t.Fatalf("non-private read should allow hard links: %v", err)
	}
}

func TestReadRegularRejectsSymlinkDirectoryAndFIFONonBlocking(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "target")
	if err := os.WriteFile(target, []byte("data"), 0o600); err != nil {
		t.Fatalf("write target: %v", err)
	}

	symlink := filepath.Join(directory, "symlink")
	if err := os.Symlink(target, symlink); err != nil {
		t.Fatalf("create symlink: %v", err)
	}
	if _, err := ReadRegular(symlink, 16, false); err == nil {
		t.Fatal("expected symlink rejection")
	}
	if _, err := ReadRegular(directory, 16, false); err == nil {
		t.Fatal("expected directory rejection")
	}

	fifo := filepath.Join(directory, "fifo")
	if err := unix.Mkfifo(fifo, 0o600); err != nil {
		t.Skipf("FIFO creation unavailable: %v", err)
	}
	result := make(chan error, 1)
	go func() {
		_, err := ReadRegular(fifo, 16, false)
		result <- err
	}()
	select {
	case err := <-result:
		if err == nil {
			t.Fatal("expected FIFO rejection")
		}
	case <-time.After(time.Second):
		t.Fatal("ReadRegular blocked while rejecting a FIFO")
	}
}

func TestCreateJournalIsCreateNewAndPrivate(t *testing.T) {
	parent := t.TempDir()
	path := filepath.Join(parent, "journal")
	if err := CreateJournal(path); err != nil {
		t.Fatalf("create journal: %v", err)
	}
	assertDirectoryMode(t, path, 0o700)
	assertDirectoryMode(t, filepath.Join(path, receiptsDirectory), 0o700)

	if err := CreateJournal(path); err == nil {
		t.Fatal("expected existing journal rejection")
	}

	target := filepath.Join(parent, "target")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatalf("create symlink target: %v", err)
	}
	alias := filepath.Join(parent, "journal-link")
	if err := os.Symlink(target, alias); err != nil {
		t.Fatalf("create journal symlink: %v", err)
	}
	if err := CreateJournal(alias); err == nil {
		t.Fatal("expected pre-existing symlink rejection")
	}
	info, err := os.Lstat(alias)
	if err != nil {
		t.Fatalf("lstat journal symlink: %v", err)
	}
	if info.Mode()&fs.ModeSymlink == 0 {
		t.Fatal("journal symlink was unexpectedly replaced")
	}
}

func TestValidateLayoutRejectsUnexpectedRootEntriesAndLockDrift(t *testing.T) {
	journalPath := createTestJournal(t)
	if err := CreateNoReplace(filepath.Join(journalPath, "manifest.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateLayout(journalPath, false); err != nil {
		t.Fatalf("validate clean layout: %v", err)
	}
	secret := filepath.Join(journalPath, "alpha.private.json")
	if err := os.WriteFile(secret, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateLayout(journalPath, false); err == nil || !strings.Contains(err.Error(), "unexpected") {
		t.Fatalf("unexpected root entry error = %v", err)
	}
	if err := os.Remove(secret); err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(journalPath, appendLockName)
	if err := os.Mkdir(lockPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := ValidateLayout(journalPath, false); err == nil {
		t.Fatal("layout without expected lock accepted lock entry")
	}
	if err := ValidateLayout(journalPath, true); err != nil {
		t.Fatalf("layout with expected lock: %v", err)
	}
}

func TestCreateNoReplacePublishesWholeFileWithoutReplacement(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "manifest.json")
	if err := CreateNoReplace(path, []byte("first"), 0o640); err != nil {
		t.Fatalf("create file: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read published file: %v", err)
	}
	if string(data) != "first" {
		t.Fatalf("published data = %q, want first", data)
	}
	assertFileMode(t, path, 0o640)

	if err := CreateNoReplace(path, []byte("second"), 0o600); err == nil {
		t.Fatal("expected existing target rejection")
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("re-read published file: %v", err)
	}
	if string(data) != "first" {
		t.Fatalf("existing file changed to %q", data)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("list publication directory: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "manifest.json" {
		t.Fatalf("temporary publication files remain: %v", entryNames(entries))
	}

	target := filepath.Join(directory, "target")
	if err := os.WriteFile(target, []byte("target"), 0o600); err != nil {
		t.Fatalf("write symlink target: %v", err)
	}
	link := filepath.Join(directory, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("create target symlink: %v", err)
	}
	if err := CreateNoReplace(link, []byte("replacement"), 0o600); err == nil {
		t.Fatal("expected symlink target rejection")
	}
	targetData, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read symlink target: %v", err)
	}
	if string(targetData) != "target" {
		t.Fatalf("symlink target was modified: %q", targetData)
	}
}

func TestCreateNoReplaceRejectsSymlinkParent(t *testing.T) {
	root := t.TempDir()
	realDirectory := filepath.Join(root, "real")
	if err := os.Mkdir(realDirectory, 0o700); err != nil {
		t.Fatalf("create real directory: %v", err)
	}
	alias := filepath.Join(root, "alias")
	if err := os.Symlink(realDirectory, alias); err != nil {
		t.Fatalf("create directory symlink: %v", err)
	}
	if err := CreateNoReplace(filepath.Join(alias, "key"), []byte("secret"), 0o600); err == nil {
		t.Fatal("expected symlink parent rejection")
	}
	if _, err := os.Lstat(filepath.Join(realDirectory, "key")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unexpected target through symlink parent: %v", err)
	}
}

func TestWithAppendLockContentionCleanupAndStaleLock(t *testing.T) {
	t.Run("contention and callback error", func(t *testing.T) {
		journalPath := createTestJournal(t)
		lockPath := filepath.Join(journalPath, appendLockName)
		callbackErr := errors.New("callback failed")
		nestedCalled := false
		err := WithAppendLock(journalPath, func(_ *LockedJournal) error {
			assertDirectoryMode(t, lockPath, 0o700)
			nestedErr := WithAppendLock(journalPath, func(_ *LockedJournal) error {
				nestedCalled = true
				return nil
			})
			if nestedErr == nil {
				t.Fatal("expected nested lock contention")
			}
			return callbackErr
		})
		if !errors.Is(err, callbackErr) {
			t.Fatalf("callback error was not preserved: %v", err)
		}
		if nestedCalled {
			t.Fatal("contending callback was run")
		}
		if _, err := os.Lstat(lockPath); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("owned lock remains after normal callback return: %v", err)
		}
	})

	t.Run("stale lock is authoritative", func(t *testing.T) {
		journalPath := createTestJournal(t)
		lockPath := filepath.Join(journalPath, appendLockName)
		if err := os.Mkdir(lockPath, 0o700); err != nil {
			t.Fatalf("create stale lock: %v", err)
		}
		called := false
		if err := WithAppendLock(journalPath, func(_ *LockedJournal) error {
			called = true
			return nil
		}); err == nil {
			t.Fatal("expected stale lock rejection")
		}
		if called {
			t.Fatal("callback ran despite stale lock")
		}
		if info, err := os.Lstat(lockPath); err != nil || !info.IsDir() {
			t.Fatalf("stale lock was removed or changed: info=%v err=%v", info, err)
		}
	})

	t.Run("panic deliberately leaves lock", func(t *testing.T) {
		journalPath := createTestJournal(t)
		lockPath := filepath.Join(journalPath, appendLockName)
		func() {
			defer func() {
				if recover() == nil {
					t.Fatal("expected callback panic")
				}
			}()
			_ = WithAppendLock(journalPath, func(_ *LockedJournal) error {
				panic("stop")
			})
		}()
		if info, err := os.Lstat(lockPath); err != nil || !info.IsDir() {
			t.Fatalf("panic lock was removed or changed: info=%v err=%v", info, err)
		}
	})

	t.Run("replacement lock is not removed", func(t *testing.T) {
		journalPath := createTestJournal(t)
		lockPath := filepath.Join(journalPath, appendLockName)
		err := WithAppendLock(journalPath, func(_ *LockedJournal) error {
			if err := os.Remove(lockPath); err != nil {
				return err
			}
			if err := os.Mkdir(lockPath, 0o700); err != nil {
				return err
			}
			return nil
		})
		if err == nil {
			t.Fatal("expected replacement-lock ownership error")
		}
		if info, statErr := os.Lstat(lockPath); statErr != nil || !info.IsDir() {
			t.Fatalf("replacement lock was disturbed: info=%v err=%v", info, statErr)
		}
	})
}

func TestWithAppendLockRejectsNilCallbackWithoutCreatingLock(t *testing.T) {
	journalPath := createTestJournal(t)
	if err := WithAppendLock(journalPath, nil); err == nil {
		t.Fatal("expected nil callback rejection")
	}
	if _, err := os.Lstat(filepath.Join(journalPath, appendLockName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("nil callback left a lock: %v", err)
	}
}

func TestPublishReceiptAndReceiptPaths(t *testing.T) {
	journalPath := createTestJournal(t)
	digestA := strings.Repeat("a", 64)
	digestB := strings.Repeat("b", 64)
	var pathOne, pathTwo string
	var expired *LockedJournal
	lateStart := make(chan struct{})
	lateResult := make(chan error, 1)
	err := WithAppendLock(journalPath, func(locked *LockedJournal) error {
		copied := *locked
		expired = &copied
		invalidDigests := []string{
			strings.Repeat("A", 64),
			strings.Repeat("g", 64),
			strings.Repeat("a", 63),
		}
		for _, digest := range invalidDigests {
			if _, err := locked.PublishReceipt(3, digest, []byte("bad")); err == nil {
				t.Fatalf("expected invalid digest %q rejection", digest)
			}
		}
		if _, err := locked.PublishReceipt(0, digestA, []byte("bad")); err == nil {
			t.Fatal("expected sequence-zero rejection")
		}
		if _, err := locked.PublishReceipt(3, digestA, make([]byte, maxReceiptBytes+1)); err == nil {
			t.Fatal("expected oversized receipt rejection")
		}
		var err error
		pathTwo, err = locked.PublishReceipt(2, digestB, []byte(`{"sequence":2}`))
		if err != nil {
			t.Fatalf("publish sequence two: %v", err)
		}
		if _, err := locked.PublishReceipt(1, digestA, []byte("second-publication")); err == nil || !strings.Contains(err.Error(), "already attempted") {
			t.Fatalf("second publication through one capability error = %v", err)
		}
		go func(capability *LockedJournal) {
			<-lateStart
			_, err := capability.PublishReceipt(3, digestA, []byte("late"))
			lateResult <- err
		}(expired)
		return nil
	})
	if err != nil {
		t.Fatalf("locked receipt publication: %v", err)
	}
	close(lateStart)
	if err := <-lateResult; err == nil || !strings.Contains(err.Error(), "active append-lock") {
		t.Fatalf("expired publication capability error = %v", err)
	}
	if err := WithAppendLock(journalPath, func(locked *LockedJournal) error {
		var err error
		pathOne, err = locked.PublishReceipt(1, digestA, []byte(`{"sequence":1}`))
		return err
	}); err != nil {
		t.Fatalf("publish sequence one: %v", err)
	}
	if err := WithAppendLock(journalPath, func(locked *LockedJournal) error {
		_, err := locked.PublishReceipt(1, digestA, []byte("replacement"))
		return err
	}); err == nil {
		t.Fatal("expected duplicate receipt publication rejection")
	}
	assertFileMode(t, pathOne, 0o600)
	assertFileMode(t, pathTwo, 0o600)

	paths, err := ReceiptPaths(journalPath)
	if err != nil {
		t.Fatalf("list receipts: %v", err)
	}
	want := []string{pathOne, pathTwo}
	if len(paths) != len(want) {
		t.Fatalf("receipt path count = %d, want %d: %v", len(paths), len(want), paths)
	}
	for index := range want {
		if paths[index] != want[index] {
			t.Fatalf("receipt path %d = %q, want %q", index, paths[index], want[index])
		}
	}

	data, err := ReadRegular(pathOne, 1024, true)
	if err != nil {
		t.Fatalf("read private receipt: %v", err)
	}
	if string(data) != `{"sequence":1}` {
		t.Fatalf("sequence-one receipt = %q", data)
	}
	data, err = os.ReadFile(pathOne)
	if err != nil {
		t.Fatalf("re-read sequence-one receipt: %v", err)
	}
	if string(data) != `{"sequence":1}` {
		t.Fatalf("duplicate publication changed receipt to %q", data)
	}

	entries, err := os.ReadDir(filepath.Join(journalPath, receiptsDirectory))
	if err != nil {
		t.Fatalf("list receipt files: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("publication temporary files remain: %v", entryNames(entries))
	}

}

func TestLockedJournalSerializesOnePublication(t *testing.T) {
	journalPath := createTestJournal(t)
	start := make(chan struct{})
	results := make(chan error, 2)
	if err := WithAppendLock(journalPath, func(locked *LockedJournal) error {
		for _, digest := range []string{strings.Repeat("c", 64), strings.Repeat("d", 64)} {
			go func(digest string) {
				<-start
				_, err := locked.PublishReceipt(1, digest, []byte("one"))
				results <- err
			}(digest)
		}
		close(start)
		successes := 0
		failures := 0
		for range 2 {
			if err := <-results; err == nil {
				successes++
			} else {
				failures++
			}
		}
		if successes != 1 || failures != 1 {
			t.Fatalf("concurrent publication results: successes=%d failures=%d", successes, failures)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	paths, err := ReceiptPaths(journalPath)
	if err != nil || len(paths) != 1 {
		t.Fatalf("published paths = %v, err=%v", paths, err)
	}
}

func TestReceiptPathsFailsClosedOnUnexpectedEntries(t *testing.T) {
	tests := []struct {
		name   string
		create func(t *testing.T, receiptsPath string)
	}{
		{
			name: "unexpected filename",
			create: func(t *testing.T, receiptsPath string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(receiptsPath, ".DS_Store"), nil, 0o600); err != nil {
					t.Fatalf("write unexpected file: %v", err)
				}
			},
		},
		{
			name: "exact-name directory",
			create: func(t *testing.T, receiptsPath string) {
				t.Helper()
				name := "00000000000000000001-" + strings.Repeat("1", 64) + ".json"
				if err := os.Mkdir(filepath.Join(receiptsPath, name), 0o700); err != nil {
					t.Fatalf("create receipt-shaped directory: %v", err)
				}
			},
		},
		{
			name: "exact-name symlink",
			create: func(t *testing.T, receiptsPath string) {
				t.Helper()
				target := filepath.Join(filepath.Dir(receiptsPath), "target")
				if err := os.WriteFile(target, []byte("target"), 0o600); err != nil {
					t.Fatalf("write symlink target: %v", err)
				}
				name := "00000000000000000001-" + strings.Repeat("2", 64) + ".json"
				if err := os.Symlink(target, filepath.Join(receiptsPath, name)); err != nil {
					t.Fatalf("create receipt-shaped symlink: %v", err)
				}
			},
		},
		{
			name: "overflow sequence",
			create: func(t *testing.T, receiptsPath string) {
				t.Helper()
				name := "99999999999999999999-" + strings.Repeat("3", 64) + ".json"
				if err := os.WriteFile(filepath.Join(receiptsPath, name), nil, 0o600); err != nil {
					t.Fatalf("write overflow-sequence file: %v", err)
				}
			},
		},
		{
			name: "sequence zero",
			create: func(t *testing.T, receiptsPath string) {
				t.Helper()
				name := "00000000000000000000-" + strings.Repeat("4", 64) + ".json"
				if err := os.WriteFile(filepath.Join(receiptsPath, name), nil, 0o600); err != nil {
					t.Fatalf("write sequence-zero file: %v", err)
				}
			},
		},
		{
			name: "sequence gap",
			create: func(t *testing.T, receiptsPath string) {
				t.Helper()
				name := "00000000000000000002-" + strings.Repeat("5", 64) + ".json"
				if err := os.WriteFile(filepath.Join(receiptsPath, name), nil, 0o600); err != nil {
					t.Fatalf("write sequence-gap file: %v", err)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			journalPath := createTestJournal(t)
			test.create(t, filepath.Join(journalPath, receiptsDirectory))
			if _, err := ReceiptPaths(journalPath); err == nil {
				t.Fatal("expected fail-closed receipt listing")
			}
		})
	}
}

func TestReceiptPathsRejectsDuplicateSequence(t *testing.T) {
	journalPath := createTestJournal(t)
	if err := WithAppendLock(journalPath, func(locked *LockedJournal) error {
		if _, err := locked.PublishReceipt(7, strings.Repeat("7", 64), []byte("one")); err != nil {
			t.Fatalf("publish first receipt: %v", err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := WithAppendLock(journalPath, func(locked *LockedJournal) error {
		_, err := locked.PublishReceipt(7, strings.Repeat("8", 64), []byte("two"))
		return err
	}); err != nil {
		t.Fatalf("publish second receipt: %v", err)
	}
	if _, err := ReceiptPaths(journalPath); err == nil {
		t.Fatal("expected duplicate sequence rejection")
	}
}

func TestReceiptPathsRejectsSymlinkReceiptsDirectory(t *testing.T) {
	root := t.TempDir()
	journalPath := filepath.Join(root, "journal")
	if err := os.Mkdir(journalPath, 0o700); err != nil {
		t.Fatalf("create journal fixture: %v", err)
	}
	realReceipts := filepath.Join(root, "real-receipts")
	if err := os.Mkdir(realReceipts, 0o700); err != nil {
		t.Fatalf("create receipts target: %v", err)
	}
	if err := os.Symlink(realReceipts, filepath.Join(journalPath, receiptsDirectory)); err != nil {
		t.Fatalf("create receipts symlink: %v", err)
	}
	if _, err := ReceiptPaths(journalPath); err == nil {
		t.Fatal("expected symlink receipts-directory rejection")
	}
}

func TestDirectoryEntryPointsRejectCleanedJournalSymlinkSpellings(t *testing.T) {
	root := t.TempDir()
	realJournal := filepath.Join(root, "real-journal")
	if err := CreateJournal(realJournal); err != nil {
		t.Fatal(err)
	}
	if err := CreateNoReplace(filepath.Join(realJournal, "manifest.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "journal-alias")
	if err := os.Symlink(realJournal, alias); err != nil {
		t.Skipf("directory symlinks are unavailable: %v", err)
	}

	for name, spelling := range map[string]string{
		"trailing slash": alias + string(os.PathSeparator),
		"dot suffix":     alias + string(os.PathSeparator) + ".",
	} {
		t.Run(name, func(t *testing.T) {
			if err := ValidateLayout(spelling, false); err == nil {
				t.Fatal("ValidateLayout followed a journal symlink")
			}
			called := false
			if err := WithAppendLock(spelling, func(_ *LockedJournal) error {
				called = true
				return nil
			}); err == nil {
				t.Fatal("WithAppendLock followed a journal symlink")
			}
			if called {
				t.Fatal("append callback ran through a journal symlink")
			}
			if _, err := ReceiptPaths(spelling); err == nil {
				t.Fatal("ReceiptPaths followed a journal symlink")
			}
			if _, err := os.Lstat(filepath.Join(realJournal, appendLockName)); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("symlink rejection changed real journal lock state: %v", err)
			}
		})
	}
}

func createTestJournal(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "journal")
	if err := CreateJournal(path); err != nil {
		t.Fatalf("create test journal: %v", err)
	}
	return path
}

func assertDirectoryMode(t *testing.T, path string, want fs.FileMode) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("lstat directory %q: %v", path, err)
	}
	if !info.IsDir() {
		t.Fatalf("%q is not a directory: %v", path, info.Mode())
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != want {
		t.Fatalf("directory %q mode = %o, want %o", path, info.Mode().Perm(), want)
	}
}

func assertFileMode(t *testing.T, path string, want fs.FileMode) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("lstat file %q: %v", path, err)
	}
	if !info.Mode().IsRegular() {
		t.Fatalf("%q is not a regular file: %v", path, info.Mode())
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != want {
		t.Fatalf("file %q mode = %o, want %o", path, info.Mode().Perm(), want)
	}
}

func entryNames(entries []os.DirEntry) []string {
	names := make([]string, len(entries))
	for index, entry := range entries {
		names[index] = entry.Name()
	}
	return names
}
