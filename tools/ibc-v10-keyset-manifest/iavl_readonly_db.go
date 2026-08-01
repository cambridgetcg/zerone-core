package main

import (
	"errors"
	"fmt"
	"sync/atomic"

	dbm "github.com/cosmos/cosmos-db"
)

var errReadOnlyIAVLCensus = errors.New("IBC census database is read-only")

// readOnlyPhysicalDB expands the deliberately narrow physicalDB reader into
// the dbm.DB API required by IAVL. Every mutation surface fails before it can
// reach the copied source. Close is intentionally a no-op: the CLI owns the
// source database lifetime.
type readOnlyPhysicalDB struct {
	source        physicalDB
	writeAttempts int64
}

var _ dbm.DB = (*readOnlyPhysicalDB)(nil)

func newReadOnlyPhysicalDB(source physicalDB) *readOnlyPhysicalDB {
	return &readOnlyPhysicalDB{source: source}
}

func (db *readOnlyPhysicalDB) Get(key []byte) ([]byte, error) {
	if db.source == nil {
		return nil, errors.New("read-only IBC census source is nil")
	}
	return db.source.Get(key)
}

func (db *readOnlyPhysicalDB) Has(key []byte) (bool, error) {
	value, err := db.Get(key)
	if err != nil {
		return false, err
	}
	return value != nil, nil
}

func (db *readOnlyPhysicalDB) Iterator(start, end []byte) (dbm.Iterator, error) {
	if db.source == nil {
		return nil, errors.New("read-only IBC census source is nil")
	}
	return db.source.Iterator(start, end)
}

func (db *readOnlyPhysicalDB) ReverseIterator(start, end []byte) (dbm.Iterator, error) {
	if db.source == nil {
		return nil, errors.New("read-only IBC census source is nil")
	}
	return db.source.ReverseIterator(start, end)
}

func (db *readOnlyPhysicalDB) Set(_, _ []byte) error {
	return db.refuseWrite("Set")
}

func (db *readOnlyPhysicalDB) SetSync(_, _ []byte) error {
	return db.refuseWrite("SetSync")
}

func (db *readOnlyPhysicalDB) Delete(_ []byte) error {
	return db.refuseWrite("Delete")
}

func (db *readOnlyPhysicalDB) DeleteSync(_ []byte) error {
	return db.refuseWrite("DeleteSync")
}

func (db *readOnlyPhysicalDB) Close() error {
	return nil
}

func (db *readOnlyPhysicalDB) NewBatch() dbm.Batch {
	return &readOnlyBatch{owner: db}
}

func (db *readOnlyPhysicalDB) NewBatchWithSize(_ int) dbm.Batch {
	return &readOnlyBatch{owner: db}
}

func (db *readOnlyPhysicalDB) Print() error {
	return errors.New("printing the copied IBC census database is disabled")
}

func (db *readOnlyPhysicalDB) Stats() map[string]string {
	return map[string]string{"mode": "read-only-ibc-census"}
}

func (db *readOnlyPhysicalDB) WriteAttempts() int64 {
	return atomic.LoadInt64(&db.writeAttempts)
}

func (db *readOnlyPhysicalDB) refuseWrite(operation string) error {
	atomic.AddInt64(&db.writeAttempts, 1)
	return fmt.Errorf("%s: %w", operation, errReadOnlyIAVLCensus)
}

type readOnlyBatch struct {
	owner  *readOnlyPhysicalDB
	closed uint32
}

var _ dbm.Batch = (*readOnlyBatch)(nil)

func (batch *readOnlyBatch) Set(_, _ []byte) error {
	return batch.refuseWrite("batch Set")
}

func (batch *readOnlyBatch) Delete(_ []byte) error {
	return batch.refuseWrite("batch Delete")
}

func (batch *readOnlyBatch) Write() error {
	return batch.refuseWrite("batch Write")
}

func (batch *readOnlyBatch) WriteSync() error {
	return batch.refuseWrite("batch WriteSync")
}

func (batch *readOnlyBatch) Close() error {
	atomic.StoreUint32(&batch.closed, 1)
	return nil
}

func (batch *readOnlyBatch) GetByteSize() (int, error) {
	if atomic.LoadUint32(&batch.closed) != 0 {
		return 0, errors.New("read-only IBC census batch is closed")
	}
	return 0, nil
}

func (batch *readOnlyBatch) refuseWrite(operation string) error {
	if batch.owner == nil {
		return fmt.Errorf("%s: %w", operation, errReadOnlyIAVLCensus)
	}
	return batch.owner.refuseWrite(operation)
}
