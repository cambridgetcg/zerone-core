// Package activation reads and writes the default-off markers that select a
// consensus repair. Each marker is one key in its module's store: absent means
// the repair is off, the single byte 1 means it is on, and any other value is a
// fault. Only native genesis and the carrier upgrade write a marker, and
// nothing clears one, so a fault means the store no longer holds what consensus
// wrote. Callers read a marker once per entry point and propagate a fault,
// never treating it as off.
//
// The gas-free read applies to the consensus-repairs-v1 and -v2 markers only
// (app.ConsensusRepairsV1Markers and app.ConsensusRepairsV2Markers). Older
// knowledge markers (ClaimIntegrityEnabled at 0x87, record integrity, review
// neutrality, claim records, fund settlement) read through the caller's gas
// meter on live chains; moving them onto Read would change gas and needs its
// own upgrade.
package activation

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	corestore "cosmossdk.io/core/store"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ErrMarkerFault is wrapped by every error Read and Enable return: a store
// failure, or a stored value other than the single byte 1. It lets a caller
// tell a marker fault, which must propagate, from the errors its legacy path
// already returns.
var ErrMarkerFault = errors.New("activation marker fault")

// faultError keeps each module's marker error text exactly as it was before
// the readers were shared, while wrapping ErrMarkerFault and any store error.
type faultError struct {
	msg   string
	cause error
}

func (e *faultError) Error() string { return e.msg }

func (e *faultError) Unwrap() []error {
	if e.cause == nil {
		return []error{ErrMarkerFault}
	}
	return []error{ErrMarkerFault, e.cause}
}

// Read reports whether the marker at key is set. name labels its errors
// ("malformed <name> marker", "read <name> marker: <cause>").
//
// The read runs under a fresh infinite gas meter, so a chain without the
// marker spends exactly the gas it spent before the marker existed. Absent
// reads as off; anything but absent or the single byte 1 is an error wrapping
// ErrMarkerFault, never off.
func Read(ctx context.Context, store corestore.KVStoreService, key []byte, name string) (bool, error) {
	ungassed := sdk.UnwrapSDKContext(ctx).WithGasMeter(storetypes.NewInfiniteGasMeter())
	bz, err := store.OpenKVStore(ungassed).Get(key)
	if err != nil {
		return false, &faultError{msg: fmt.Sprintf("read %s marker: %v", name, err), cause: err}
	}
	if bz == nil {
		return false, nil
	}
	if !bytes.Equal(bz, []byte{1}) {
		return false, &faultError{msg: fmt.Sprintf("malformed %s marker", name)}
	}
	return true, nil
}

// Enable sets the marker at key. It is idempotent, writes nothing else, and
// never repairs a malformed marker: Read's fault is returned instead. The write
// is charged to ctx's gas meter.
func Enable(ctx context.Context, store corestore.KVStoreService, key []byte, name string) error {
	enabled, err := Read(ctx, store, key, name)
	if err != nil || enabled {
		return err
	}
	if err := store.OpenKVStore(ctx).Set(key, []byte{1}); err != nil {
		return &faultError{msg: fmt.Sprintf("write %s marker: %v", name, err), cause: err}
	}
	return nil
}

// StoreKeyService serves Read and Enable for a keeper that holds a raw store
// key rather than a KVStoreService. It opens the context's own store for key,
// as runtime.NewKVStoreService does for a concrete *KVStoreKey, so reads and
// writes meter gas the same way.
func StoreKeyService(key storetypes.StoreKey) corestore.KVStoreService {
	return storeKeyService{key: key}
}

type storeKeyService struct{ key storetypes.StoreKey }

func (s storeKeyService) OpenKVStore(ctx context.Context) corestore.KVStore {
	return storeKeyStore{sdk.UnwrapSDKContext(ctx).KVStore(s.key)}
}

// storeKeyStore gives an SDK store the core interface. Like the SDK's own
// adapter, it never returns an error: a failing SDK store panics instead.
type storeKeyStore struct{ kv storetypes.KVStore }

func (s storeKeyStore) Get(key []byte) ([]byte, error) { return s.kv.Get(key), nil }
func (s storeKeyStore) Has(key []byte) (bool, error)   { return s.kv.Has(key), nil }
func (s storeKeyStore) Set(key, value []byte) error    { s.kv.Set(key, value); return nil }
func (s storeKeyStore) Delete(key []byte) error        { s.kv.Delete(key); return nil }

func (s storeKeyStore) Iterator(start, end []byte) (corestore.Iterator, error) {
	return s.kv.Iterator(start, end), nil
}

func (s storeKeyStore) ReverseIterator(start, end []byte) (corestore.Iterator, error) {
	return s.kv.ReverseIterator(start, end), nil
}
