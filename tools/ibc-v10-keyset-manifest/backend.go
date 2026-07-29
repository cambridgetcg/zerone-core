package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
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
	userHome, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve user home before checking --home safety: %w", err)
	}
	defaultNodeHome := filepath.Join(userHome, ".zeroned")
	if pathIsEqualOrDescendant(filepath.Clean(defaultNodeHome), cleanHome) {
		return nil, fmt.Errorf(
			"refusing Zerone's default live home or descendant %q; pass a separately copied home",
			cleanHome,
		)
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

	switch backend {
	case string(dbm.GoLevelDBBackend):
		db, err := dbm.NewGoLevelDBWithOpts(
			applicationDBName,
			dataDir,
			&opt.Options{
				ReadOnly:       true,
				ErrorIfMissing: true,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("open copied GoLevelDB application DB read-only: %w", err)
		}
		return db, nil

	case string(dbm.PebbleDBBackend):
		db, err := pebble.Open(
			dbPath,
			&pebble.Options{
				ReadOnly:         true,
				ErrorIfNotExists: true,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("open copied Pebble application DB read-only: %w", err)
		}
		return &pebbleReadDB{db: db}, nil

	default:
		return nil, fmt.Errorf(
			"--backend must be %q or %q: got %q",
			dbm.GoLevelDBBackend,
			dbm.PebbleDBBackend,
			backend,
		)
	}
}

func pathIsEqualOrDescendant(base, candidate string) bool {
	relative, err := filepath.Rel(base, candidate)
	if err != nil {
		return false
	}
	return relative == "." ||
		(relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
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
	iterator.First()
	return &pebbleReadIterator{
		iterator: iterator,
		start:    start,
		end:      end,
	}, nil
}

func (db *pebbleReadDB) Close() error {
	return db.db.Close()
}

type pebbleReadIterator struct {
	iterator *pebble.Iterator
	start    []byte
	end      []byte
}

func (iterator *pebbleReadIterator) Domain() ([]byte, []byte) {
	return iterator.start, iterator.end
}

func (iterator *pebbleReadIterator) Valid() bool {
	return iterator.iterator.Valid()
}

func (iterator *pebbleReadIterator) Next() {
	if !iterator.Valid() {
		panic("iterator is invalid")
	}
	iterator.iterator.Next()
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
