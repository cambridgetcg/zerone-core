package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/cockroachdb/pebble"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

const applicationDBName = "application"

type openedPhysicalDB interface {
	physicalDB
	Close() error
}

// openCopiedApplicationDB deliberately accepts only a separately copied node
// home and opens its application database through a backend-enforced read-only
// handle. In particular, it refuses the default live home and aliases into it.
func openCopiedApplicationDB(home, backend string) (openedPhysicalDB, error) {
	if home == "" {
		return nil, errors.New("--home is required")
	}
	if !filepath.IsAbs(home) {
		return nil, errors.New("--home must be an absolute path")
	}
	cleanHome := filepath.Clean(home)
	if cleanHome == string(filepath.Separator) {
		return nil, errors.New("--home must not be the filesystem root")
	}

	protectedHomes, err := protectedUserHomes()
	if err != nil {
		return nil, err
	}
	for _, protectedHome := range protectedHomes {
		defaultNodeHome := filepath.Join(protectedHome, ".zeroned")
		if pathIsEqualOrDescendant(filepath.Clean(defaultNodeHome), cleanHome) {
			return nil, fmt.Errorf(
				"refusing Zerone's default live home or descendant %q; pass a separately copied home",
				cleanHome,
			)
		}
	}

	homeInfo, err := os.Lstat(cleanHome)
	if err != nil {
		return nil, fmt.Errorf("inspect copied home %q: %w", cleanHome, err)
	}
	if homeInfo.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("copied home %q must not be a symlink", cleanHome)
	}
	if !homeInfo.IsDir() {
		return nil, fmt.Errorf("copied home %q must be a directory", cleanHome)
	}

	resolvedHome, err := filepath.EvalSymlinks(cleanHome)
	if err != nil {
		return nil, fmt.Errorf(
			"resolve copied home %q before checking live-home aliases: %w",
			cleanHome,
			err,
		)
	}
	for _, protectedHome := range protectedHomes {
		resolvedUserHome, resolveErr := filepath.EvalSymlinks(filepath.Clean(protectedHome))
		if resolveErr != nil {
			return nil, fmt.Errorf("resolve protected user home before checking --home safety: %w", resolveErr)
		}
		resolvedDefaultNodeHome := filepath.Join(resolvedUserHome, ".zeroned")
		if resolved, defaultErr := filepath.EvalSymlinks(resolvedDefaultNodeHome); defaultErr == nil {
			resolvedDefaultNodeHome = resolved
		} else if !errors.Is(defaultErr, os.ErrNotExist) {
			return nil, fmt.Errorf("resolve Zerone's default live home before checking --home safety: %w", defaultErr)
		}
		identityMatch, identityErr := pathIsSameFileOrDescendant(resolvedDefaultNodeHome, resolvedHome)
		if identityErr != nil {
			return nil, fmt.Errorf("compare copied home to Zerone's default live home: %w", identityErr)
		}
		if pathIsEqualOrDescendant(filepath.Clean(resolvedDefaultNodeHome), filepath.Clean(resolvedHome)) || identityMatch {
			return nil, fmt.Errorf(
				"refusing resolved alias to Zerone's default live home or descendant %q; pass a separately copied home",
				cleanHome,
			)
		}
	}

	dataDir := filepath.Join(cleanHome, "data")
	dbPath := filepath.Join(dataDir, applicationDBName+dbm.DBFileSuffix)
	dataInfo, err := os.Lstat(dataDir)
	if err != nil {
		return nil, fmt.Errorf("inspect copied data directory %q: %w", dataDir, err)
	}
	if dataInfo.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("copied data directory %q must not be a symlink", dataDir)
	}
	if !dataInfo.IsDir() {
		return nil, fmt.Errorf("copied data path %q must be a directory", dataDir)
	}
	info, err := os.Lstat(dbPath)
	if err != nil {
		return nil, fmt.Errorf("inspect copied application DB %q: %w", dbPath, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("copied application DB %q must not be a symlink", dbPath)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("copied application DB %q must be a directory", dbPath)
	}

	var opened openedPhysicalDB
	switch backend {
	case string(dbm.GoLevelDBBackend):
		db, err := dbm.NewGoLevelDBWithOpts(
			applicationDBName,
			dataDir,
			&opt.Options{ReadOnly: true, ErrorIfMissing: true},
		)
		if err != nil {
			return nil, fmt.Errorf("open copied GoLevelDB application DB read-only: %w", err)
		}
		opened = db

	case string(dbm.PebbleDBBackend):
		db, err := pebble.Open(
			dbPath,
			&pebble.Options{ReadOnly: true, ErrorIfNotExists: true},
		)
		if err != nil {
			return nil, fmt.Errorf("open copied Pebble application DB read-only: %w", err)
		}
		opened = &pebbleReadDB{db: db}

	default:
		return nil, fmt.Errorf(
			"--backend must be %q or %q: got %q",
			dbm.GoLevelDBBackend,
			dbm.PebbleDBBackend,
			backend,
		)
	}
	postOpenInfo, err := os.Stat(dbPath)
	if err != nil || !os.SameFile(info, postOpenInfo) {
		closeErr := opened.Close()
		if err == nil {
			err = errors.New("copied application DB path changed while it was being opened")
		}
		return nil, errors.Join(err, closeErr)
	}
	return opened, nil
}

func protectedUserHomes() ([]string, error) {
	environmentHome, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve environment user home before checking --home safety: %w", err)
	}
	account, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("resolve OS account home before checking --home safety: %w", err)
	}
	candidates := []string{environmentHome, account.HomeDir}
	seen := make(map[string]struct{}, len(candidates))
	homes := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate == "" || !filepath.IsAbs(candidate) {
			return nil, errors.New("protected user home is empty or not absolute")
		}
		clean := filepath.Clean(candidate)
		if _, duplicate := seen[clean]; duplicate {
			continue
		}
		seen[clean] = struct{}{}
		homes = append(homes, clean)
	}
	return homes, nil
}

func pathIsEqualOrDescendant(base, candidate string) bool {
	relative, err := filepath.Rel(base, candidate)
	if err != nil {
		return false
	}
	return relative == "." ||
		(relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}

func pathIsSameFileOrDescendant(base, candidate string) (bool, error) {
	baseInfo, err := os.Stat(base)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	current := filepath.Clean(candidate)
	for {
		info, statErr := os.Stat(current)
		if statErr != nil {
			return false, statErr
		}
		if os.SameFile(baseInfo, info) {
			return true, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return false, nil
		}
		current = parent
	}
}

type pebbleReadDB struct {
	db *pebble.DB
}

func (db *pebbleReadDB) Get(key []byte) ([]byte, error) {
	if len(key) == 0 {
		return nil, errors.New("key cannot be empty")
	}
	value, closer, err := db.db.Get(key)
	if errors.Is(err, pebble.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	copied := bytes.Clone(value)
	if err := closer.Close(); err != nil {
		return nil, fmt.Errorf("close Pebble value handle: %w", err)
	}
	return copied, nil
}

func (db *pebbleReadDB) Iterator(start, end []byte) (dbm.Iterator, error) {
	return db.newIterator(start, end, false)
}

func (db *pebbleReadDB) ReverseIterator(start, end []byte) (dbm.Iterator, error) {
	return db.newIterator(start, end, true)
}

func (db *pebbleReadDB) newIterator(start, end []byte, reverse bool) (dbm.Iterator, error) {
	if len(start) == 0 || len(end) == 0 {
		return nil, errors.New("bounded Pebble iterator requires non-empty start and end")
	}
	if bytes.Compare(start, end) >= 0 {
		return nil, errors.New("Pebble iterator start must be before end")
	}
	start = bytes.Clone(start)
	end = bytes.Clone(end)
	iterator, err := db.db.NewIter(&pebble.IterOptions{
		LowerBound: start,
		UpperBound: end,
	})
	if err != nil {
		return nil, err
	}
	if reverse {
		iterator.Last()
	} else {
		iterator.First()
	}
	return &pebbleReadIterator{
		iterator: iterator,
		start:    start,
		end:      end,
		reverse:  reverse,
	}, nil
}

func (db *pebbleReadDB) Close() error {
	return db.db.Close()
}

type pebbleReadIterator struct {
	iterator *pebble.Iterator
	start    []byte
	end      []byte
	reverse  bool
}

func (iterator *pebbleReadIterator) Domain() ([]byte, []byte) {
	return bytes.Clone(iterator.start), bytes.Clone(iterator.end)
}

func (iterator *pebbleReadIterator) Valid() bool {
	return iterator.iterator.Valid()
}

func (iterator *pebbleReadIterator) Next() {
	if !iterator.Valid() {
		panic("iterator is invalid")
	}
	if iterator.reverse {
		iterator.iterator.Prev()
	} else {
		iterator.iterator.Next()
	}
}

func (iterator *pebbleReadIterator) Key() []byte {
	if !iterator.Valid() {
		panic("iterator is invalid")
	}
	return bytes.Clone(iterator.iterator.Key())
}

func (iterator *pebbleReadIterator) Value() []byte {
	if !iterator.Valid() {
		panic("iterator is invalid")
	}
	return bytes.Clone(iterator.iterator.Value())
}

func (iterator *pebbleReadIterator) Error() error {
	return iterator.iterator.Error()
}

func (iterator *pebbleReadIterator) Close() error {
	return iterator.iterator.Close()
}
