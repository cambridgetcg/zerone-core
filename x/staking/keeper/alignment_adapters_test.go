package keeper_test

import (
	"errors"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/staking/keeper"
	"github.com/zerone-chain/zerone/x/staking/types"
)

type alignmentRegistryFaults struct {
	terminal, close error
	closed          bool
}

type alignmentRegistryMultiStore struct {
	storetypes.MultiStore
	faults *alignmentRegistryFaults
}

func (s alignmentRegistryMultiStore) GetKVStore(key storetypes.StoreKey) storetypes.KVStore {
	return alignmentRegistryKV{s.MultiStore.GetKVStore(key), s.faults}
}

type alignmentRegistryKV struct {
	storetypes.KVStore
	faults *alignmentRegistryFaults
}

func (s alignmentRegistryKV) Iterator(start, end []byte) storetypes.Iterator {
	return alignmentRegistryIterator{s.KVStore.Iterator(start, end), s.faults}
}

type alignmentRegistryIterator struct {
	storetypes.Iterator
	faults *alignmentRegistryFaults
}

func (i alignmentRegistryIterator) Error() error {
	if i.faults.terminal != nil {
		return i.faults.terminal
	}
	return i.Iterator.Error()
}
func (i alignmentRegistryIterator) Close() error {
	i.faults.closed = true
	return errors.Join(i.Iterator.Close(), i.faults.close)
}

func TestAlignmentRegistryCountIncludesInactiveAndMalformedKeys(t *testing.T) {
	key := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	ms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())
	ctx := sdk.NewContext(ms, cmtproto.Header{Height: 100}, false, log.NewNopLogger())
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	adapter := keeper.NewAlignmentStakingAdapter(keeper.NewKeeper(cdc, key, nil, nil, "authority"))
	count, err := adapter.GetRegisteredValidatorCount(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
	// Other store prefixes, including DID lookup keys, do not count.
	ctx.KVStore(key).Set(types.ValidatorByDIDKey("did:fixture"), []byte("active"))
	ctx.KVStore(key).Set(types.ValidatorKey("active"), []byte(`{"is_active":true}`))
	ctx.KVStore(key).Set(types.ValidatorKey("inactive"), []byte(`{"is_active":false}`))
	ctx.KVStore(key).Set(types.ValidatorKey("malformed"), []byte("bad json"))
	count, err = adapter.GetRegisteredValidatorCount(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(3), count)
	cache, _ := ctx.CacheContext()
	count, err = adapter.GetRegisteredValidatorCount(cache)
	require.NoError(t, err, "normal SDK cache-iterator exhaustion is not a backend fault")
	require.Equal(t, uint64(3), count)

	for _, mode := range []string{"terminal", "close", "both"} {
		t.Run(mode, func(t *testing.T) {
			terminalCause, closeCause := errors.New("synthetic iterator failure"), errors.New("synthetic close failure")
			faults := &alignmentRegistryFaults{}
			if mode != "close" {
				faults.terminal = terminalCause
			}
			if mode != "terminal" {
				faults.close = closeCause
			}
			faultCtx := ctx.WithMultiStore(alignmentRegistryMultiStore{ctx.MultiStore(), faults})
			_, err := adapter.GetRegisteredValidatorCount(faultCtx)
			require.Error(t, err)
			if mode != "close" {
				require.ErrorIs(t, err, terminalCause)
			}
			if mode != "terminal" {
				require.ErrorIs(t, err, closeCause)
			}
			require.True(t, faults.closed)
		})
	}
}
