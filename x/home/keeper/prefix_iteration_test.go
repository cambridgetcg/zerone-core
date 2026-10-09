package keeper_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	corestore "cosmossdk.io/core/store"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/home/keeper"
	"github.com/zerone-chain/zerone/x/home/types"
)

type homeIteratorRead struct {
	start, end []byte
	closed     bool
	nexts      int
}

type homeObservedService struct {
	corestore.KVStoreService
	reads []*homeIteratorRead
	fail  bool
}

func (s *homeObservedService) OpenKVStore(ctx context.Context) corestore.KVStore {
	return homeObservedStore{KVStore: s.KVStoreService.OpenKVStore(ctx), owner: s}
}

type homeObservedStore struct {
	corestore.KVStore
	owner *homeObservedService
}

func (s homeObservedStore) Iterator(start, end []byte) (corestore.Iterator, error) {
	read := &homeIteratorRead{start: bytes.Clone(start), end: bytes.Clone(end)}
	s.owner.reads = append(s.owner.reads, read)
	if s.owner.fail {
		return nil, errors.New("synthetic iterator-open failure")
	}
	iter, err := s.KVStore.Iterator(start, end)
	if err != nil {
		return nil, err
	}
	return &homeObservedIterator{Iterator: iter, read: read}, nil
}

type homeObservedIterator struct {
	corestore.Iterator
	read *homeIteratorRead
}

func (i *homeObservedIterator) Close() error {
	i.read.closed = true
	return i.Iterator.Close()
}

func (i *homeObservedIterator) Next() {
	i.read.nexts++
	i.Iterator.Next()
}

// These are the actual five Home prefix families, not arbitrary byte prefixes.
// In particular, this does not claim that Home's former helper and the SDK
// agree for generic prefixes ending in 0xff.
func TestHomePrefixIterationContract(t *testing.T) {
	const homeID = "home-1"
	tests := []struct {
		name     string
		prefix   []byte
		neighbor []byte
		value    func(string) proto.Message
		visit    func(keeper.Keeper, sdk.Context, func(string) bool)
		canStop  bool
	}{
		{
			name: "homes", prefix: types.HomeKeyPrefix, neighbor: types.SessionKeyPrefix, canStop: true,
			value: func(id string) proto.Message { return &types.AgentHome{HomeId: id} },
			visit: func(k keeper.Keeper, ctx sdk.Context, cb func(string) bool) {
				k.IterateHomes(ctx, func(v *types.AgentHome) bool { return cb(v.HomeId) })
			},
		},
		{
			name: "sessions", prefix: types.SessionPrefixKey(homeID), neighbor: types.SessionPrefixKey("home-10"), canStop: true,
			value: func(id string) proto.Message { return &types.ActiveSession{HomeId: homeID, SessionId: id} },
			visit: func(k keeper.Keeper, ctx sdk.Context, cb func(string) bool) {
				k.IterateSessions(ctx, homeID, func(v *types.ActiveSession) bool { return cb(v.SessionId) })
			},
		},
		{
			name: "keys", prefix: types.KeyRegPrefixKey(homeID), neighbor: types.KeyRegPrefixKey("home-10"), canStop: true,
			value: func(id string) proto.Message { return &types.KeyRegistration{KeyHash: id} },
			visit: func(k keeper.Keeper, ctx sdk.Context, cb func(string) bool) {
				k.IterateKeyRegistrations(ctx, homeID, func(v *types.KeyRegistration) bool { return cb(v.KeyHash) })
			},
		},
		{
			name: "alerts", prefix: types.AlertPrefixKey(homeID), neighbor: types.AlertPrefixKey("home-10"), canStop: true,
			value: func(id string) proto.Message { return &types.Alert{HomeId: homeID, AlertId: id} },
			visit: func(k keeper.Keeper, ctx sdk.Context, cb func(string) bool) {
				k.IterateAlerts(ctx, homeID, func(v *types.Alert) bool { return cb(v.AlertId) })
			},
		},
		{
			name: "spending-limits", prefix: types.SpendLimitPrefixKey(homeID), neighbor: types.SpendLimitPrefixKey("home-10"),
			value: func(id string) proto.Message { return &types.SpendingLimit{KeyType: id, MaxAmount: "7"} },
			visit: func(k keeper.Keeper, ctx sdk.Context, cb func(string) bool) {
				for _, v := range k.GetSpendingLimitsByHome(ctx, homeID) {
					cb(v.KeyType)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var observed *homeObservedService
			k, ctx, _ := setupKeeperWithStoreWrapper(t, func(s corestore.KVStoreService) corestore.KVStoreService {
				observed = &homeObservedService{KVStoreService: s}
				return observed
			})
			kv := observed.OpenKVStore(ctx)
			prefixBefore := bytes.Clone(tc.prefix)
			end := storetypes.PrefixEndBytes(tc.prefix)
			put := func(key []byte, id string) {
				t.Helper()
				value, err := proto.Marshal(tc.value(id))
				require.NoError(t, err)
				require.NoError(t, kv.Set(key, value))
			}
			key := func(prefix []byte, suffix string) []byte { return append(bytes.Clone(prefix), []byte(suffix)...) }
			// Valid protobuf sentinels make an incorrect range observable instead
			// of relying on a foreign record accidentally failing to decode.
			lower := bytes.Clone(tc.prefix)
			lower[len(lower)-1]--
			put(key(lower, "outside"), "below-range")
			put(end, "exclusive-end")
			put(key(tc.neighbor, "outside"), "neighbor")

			read := func(stop bool) []string {
				t.Helper()
				observed.reads = nil
				var ids []string
				tc.visit(k, ctx, func(id string) bool {
					ids = append(ids, id)
					return stop
				})
				require.Len(t, observed.reads, 1)
				require.Equal(t, tc.prefix, observed.reads[0].start)
				require.Equal(t, end, observed.reads[0].end)
				require.Equal(t, !observed.fail, observed.reads[0].closed)
				require.Equal(t, prefixBefore, tc.prefix, "prefix input must not be mutated")
				return ids
			}

			require.Empty(t, read(false), "foreign ranges do not make an empty collection nonempty")
			put(key(tc.prefix, "z"), "z")
			put(key(tc.prefix, "a"), "a")
			require.NoError(t, kv.Set(key(tc.prefix, "m"), []byte{0xff}))
			require.Equal(t, []string{"a", "z"}, read(false), "ordered own records only; malformed record skipped")
			if tc.canStop {
				require.Equal(t, []string{"a"}, read(true))
				require.Zero(t, observed.reads[0].nexts, "stop before advancing the iterator")
			}
			observed.fail = true
			require.Empty(t, read(false), "preserve the existing iterator-open error behavior")
		})
	}
}
