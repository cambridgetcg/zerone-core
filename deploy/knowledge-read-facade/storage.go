package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// Root-relative operations cannot escape the fixed operator-selected directory.
// Publishers and startup readers cooperate through flock on this ONE host.
// This is not a distributed filesystem or inventory verifier.
func openStore(path string) (*os.Root, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return nil, errors.New("absolute clean snapshot root required")
	}
	if err := os.Mkdir(path, 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return nil, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("snapshot root must be a real directory")
	}
	return os.OpenRoot(path)
}
func lockStore(root *os.Root, name string) (*os.File, error) {
	if info, err := root.Lstat(name); err == nil && !info.Mode().IsRegular() {
		return nil, errors.New("lock must be regular")
	}
	f, err := root.OpenFile(name, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, fmt.Errorf("singleton/store already held: %w", err)
	}
	return f, nil
}
func unlockStore(f *os.File) { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN); _ = f.Close() }
func storeFile(root *os.Root, name string, maximum int) ([]byte, error) {
	info, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > int64(maximum) {
		return nil, errors.New("invalid immutable regular file")
	}
	f, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return readBounded(f, maximum)
}
func inventoryStore(root *os.Root) (objects, heads []string, err error) {
	d, err := root.Open(".")
	if err != nil {
		return nil, nil, err
	}
	defer d.Close()
	entries, err := d.ReadDir(2*maxPublications + 4)
	if err != nil {
		return nil, nil, err
	}
	if len(entries) > 2*maxPublications+2 {
		return nil, nil, errors.New("store entry cap")
	}
	for _, entry := range entries {
		name := entry.Name()
		info, e := entry.Info()
		if e != nil {
			return nil, nil, e
		}
		if !info.Mode().IsRegular() {
			return nil, nil, errors.New("non-regular store entry")
		}
		if name == ".store.lock" || name == ".serve.lock" {
			if info.Size() != 0 {
				return nil, nil, errors.New("unexpected lock content")
			}
			continue
		}
		id := strings.TrimSuffix(name, ".json")
		if strings.HasPrefix(id, "head-") {
			if !digestPattern.MatchString(strings.TrimPrefix(id, "head-")) || info.Size() > 4096 {
				return nil, nil, errors.New("invalid head file")
			}
			heads = append(heads, name)
		} else {
			if !digestPattern.MatchString(id) || info.Size() > snapshotBytes {
				return nil, nil, errors.New("invalid snapshot file")
			}
			objects = append(objects, name)
		}
	}
	if len(objects) > maxPublications || len(heads) > maxPublications {
		return nil, nil, errors.New("store publication cap")
	}
	return objects, heads, nil
}
func writeImmutable(root *os.Root, name string, body []byte) error {
	// Link makes complete, fsynced bytes visible with NO_REPLACE semantics. A
	// crash leftover .pending is refused by inventory; only an operator recovers.
	pending, err := root.OpenFile(".pending", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer root.Remove(".pending")
	if _, err := pending.Write(body); err != nil {
		pending.Close()
		return err
	}
	if err := pending.Sync(); err != nil {
		pending.Close()
		return err
	}
	if err := pending.Close(); err != nil {
		return err
	}
	if err := root.Link(".pending", name); err != nil {
		return err
	}
	if err := root.Remove(".pending"); err != nil {
		return err
	}
	directory, err := root.Open(".")
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
func publish(rootPath string, raw []byte, m publicationMetadata, now time.Time) (string, string, error) {
	body, err := buildSnapshot(raw, m, now)
	if err != nil {
		return "", "", err
	}
	id := hashBytes(body)
	head, err := compactJSON(publicationHead{"zerone.knowledge-publication-head/v1", id, m.ChainID, m.BlockHeight, m.ObservedAt, "fixed bounded observation; not a latest-chain pointer or proof"})
	if err != nil {
		return "", "", err
	}
	headID := hashBytes(head)
	root, err := openStore(rootPath)
	if err != nil {
		return "", "", err
	}
	defer root.Close()
	lock, err := lockStore(root, ".store.lock")
	if err != nil {
		return "", "", err
	}
	defer unlockStore(lock)
	objects, heads, err := inventoryStore(root)
	if err != nil {
		return "", "", err
	}
	if len(objects) >= maxPublications || len(heads) >= maxPublications {
		return "", "", errors.New("publication cap: provision a separately reviewed new root; no automatic eviction")
	}
	if err := writeImmutable(root, id+".json", body); err != nil {
		return "", "", err
	}
	if err := writeImmutable(root, "head-"+headID+".json", head); err != nil {
		return "", "", err
	}
	return id, headID, nil
}
func loadPublications(f *facade, root *os.Root, headID string) error {
	if !digestPattern.MatchString(headID) {
		return errors.New("explicit content-addressed head required")
	}
	lock, err := lockStore(root, ".store.lock")
	if err != nil {
		return err
	}
	defer unlockStore(lock)
	objects, _, err := inventoryStore(root)
	if err != nil {
		return err
	}
	for _, name := range objects {
		body, err := storeFile(root, name, snapshotBytes)
		if err != nil {
			return err
		}
		if err := f.addSnapshot(strings.TrimSuffix(name, ".json"), body); err != nil {
			return err
		}
	}
	body, err := storeFile(root, "head-"+headID+".json", 4096)
	if err != nil {
		return err
	}
	if hashBytes(body) != headID {
		return errors.New("head digest mismatch")
	}
	var head publicationHead
	if err := decodeExact(body, &head, 4096); err != nil {
		return err
	}
	snapshot, ok := f.snapshots[head.SnapshotID]
	if !ok {
		return errors.New("head snapshot missing")
	}
	s, err := validateSnapshot(snapshot, f.chainID)
	if err != nil {
		return err
	}
	if head != (publicationHead{"zerone.knowledge-publication-head/v1", head.SnapshotID, s.ChainID, s.BlockHeight, s.ObservedAt, "fixed bounded observation; not a latest-chain pointer or proof"}) {
		return errors.New("head metadata mismatch")
	}
	if s.Publication.RESTOrigin != f.upstream.String() || s.Publication.RPCOrigin != f.statusOrigin.String() {
		return errors.New("selected publication origins differ from fixed serving origins")
	}
	f.head = body
	return nil
}
