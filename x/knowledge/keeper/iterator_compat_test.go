package keeper

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	corestore "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

type iteratorCompatDiagnostic struct {
	corestore.Iterator
	valid bool
	err   error
}

func (it iteratorCompatDiagnostic) Valid() bool  { return it.valid }
func (it iteratorCompatDiagnostic) Error() error { return it.err }

func TestFeedbackIteratorErrorExactEOF(t *testing.T) {
	eof := errors.New("invalid cacheMergeIterator")
	failure := errors.New("exposed iterator failure")
	for _, tc := range []struct {
		name  string
		valid bool
		err   error
		clean bool
	}{
		{"valid without error", true, nil, true},
		{"exhausted without error", false, nil, true},
		{"exact exhausted cache diagnostic", false, eof, true},
		{"same diagnostic while valid", true, eof, false},
		{"exposed failure while valid", true, failure, false},
		{"exposed failure after exhaustion", false, failure, false},
		{"wrapped diagnostic", false, fmt.Errorf("backend: %w", eof), false},
		{"joined diagnostic and failure", false, errors.Join(eof, failure), false},
		{"diagnostic prefix only", false, errors.New("invalid cacheMergeIterator: backend failed"), false},
		{"other exhausted diagnostic", false, errors.New("invalid iterator"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := feedbackIteratorError(iteratorCompatDiagnostic{valid: tc.valid, err: tc.err})
			if tc.clean {
				require.NoError(t, err)
			} else {
				require.Same(t, tc.err, err, "preserve the exposed error, not only its text")
			}
		})
	}
}

// Like the receipt fault harness, use real IAVL storage. Unlike an uncommitted
// root-store fixture, reload a committed version through the SDK query cache.
func newIteratorCompatKeeper(t *testing.T) (Keeper, sdk.Context, storetypes.CommitMultiStore) {
	t.Helper()
	key := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	ms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())
	k := NewKeeper(runtime.NewKVStoreService(key), codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), "authority", nil, nil)
	ctx := sdk.NewContext(ms, cmtproto.Header{Height: 1}, false, log.NewNopLogger())
	return k, ctx, ms
}

func committedIteratorCompatContext(t *testing.T, ctx sdk.Context, ms storetypes.CommitMultiStore) sdk.Context {
	t.Helper()
	commit := ms.Commit()
	require.Positive(t, commit.Version)
	cache, err := ms.CacheMultiStoreWithVersion(commit.Version)
	require.NoError(t, err)
	return ctx.WithMultiStore(cache).WithBlockHeight(commit.Version)
}

func TestToKFeedbackFactQueryCommittedCacheIteratorEOF(t *testing.T) {
	for _, nonempty := range []bool{false, true} {
		t.Run(fmt.Sprintf("relations=%t", nonempty), func(t *testing.T) {
			k, ctx, ms := newIteratorCompatKeeper(t)
			fact := &types.Fact{Id: "root", Status: types.FactStatus_FACT_STATUS_VERIFIED}
			require.NoError(t, k.SetFactSkipTransition(ctx, fact))
			outgoing := &types.FactRelation{SourceFactId: "root", TargetFactId: "child", Relation: types.RelationType_RELATION_TYPE_SUPPORTS}
			incoming := &types.FactRelation{SourceFactId: "parent", TargetFactId: "root", Relation: types.RelationType_RELATION_TYPE_SUPPORTS}
			if nonempty {
				for _, id := range []string{"parent", "child"} {
					require.NoError(t, k.SetFactSkipTransition(ctx, &types.Fact{Id: id, Status: types.FactStatus_FACT_STATUS_VERIFIED}))
				}
				require.NoError(t, k.SetFactRelation(ctx, outgoing))
				require.NoError(t, k.SetFactRelation(ctx, incoming))
			}
			ctx = committedIteratorCompatContext(t, ctx, ms)
			for _, prefix := range [][]byte{types.FactRelationsBySourcePrefix("root"), types.FactRelationsByTargetPrefix("root")} {
				it, err := k.storeService.OpenKVStore(ctx).Iterator(prefix, prefixEndBytes(prefix))
				require.NoError(t, err)
				n := 0
				for ; it.Valid(); it.Next() {
					require.NoError(t, it.Error())
					n++
				}
				want := 0
				if nonempty {
					want = 1
				}
				require.Equal(t, want, n)
				require.EqualError(t, it.Error(), "invalid cacheMergeIterator")
				require.NoError(t, feedbackIteratorError(it))
				require.NoError(t, it.Close())
			}
			response, err := NewQueryServerImpl(k).Fact(ctx, &types.QueryFactRequest{Id: "root", TrackQuery: true, Querier: "obsolete-tracking-field"})
			require.NoError(t, err)
			if nonempty {
				require.Len(t, response.Fact.OutgoingRelations, 1)
				require.Len(t, response.Fact.IncomingRelations, 1)
				require.True(t, proto.Equal(outgoing, response.Fact.OutgoingRelations[0]))
				require.True(t, proto.Equal(incoming, response.Fact.IncomingRelations[0]))
			} else {
				require.Empty(t, response.Fact.OutgoingRelations)
				require.Empty(t, response.Fact.IncomingRelations)
			}
			persisted, found := k.GetFact(ctx, "root")
			require.True(t, found)
			require.True(t, proto.Equal(fact, persisted), "hydration and obsolete tracking fields must not change stored facts")
		})
	}
}

func TestFactUseReceiptCommittedCacheIteratorEOF(t *testing.T) {
	for _, count := range []int{0, 1} {
		t.Run(fmt.Sprintf("receipts=%d", count), func(t *testing.T) {
			k, ctx, ms := newIteratorCompatKeeper(t)
			consumer := sdk.AccAddress(bytes.Repeat([]byte{1}, 20)).String()
			p := types.DefaultParams()
			p.FitnessEpochBlocks = 10
			p.FitnessWeightQueryBps, p.FitnessWeightSatisfactionBps, p.MetabolismEnergyPerQuery = 0, 0, 0
			require.NoError(t, k.SetParams(ctx, &p))
			require.NoError(t, k.SetFactSkipTransition(ctx, &types.Fact{Id: "root"}))
			var receipts []*types.FactUseReceipt
			if count > 0 {
				receipts = append(receipts, &types.FactUseReceipt{Version: types.FactUseReceiptVersion, Consumer: consumer, FactId: "root", UseHeight: 1, ExpiryHeight: 20, Rating: types.FactUseRating_FACT_USE_RATING_UNRATED})
			}
			require.NoError(t, k.InitFactUseReceipts(ctx, receipts, &types.FactUsePruningState{EverReported: count > 0}))
			ctx = committedIteratorCompatContext(t, ctx, ms)
			found, err := k.HasRetainedFactUseReceipts(ctx)
			require.NoError(t, err)
			require.Equal(t, count > 0, found)
			exported, err := k.ExportFactUseReceipts(ctx)
			require.NoError(t, err)
			require.Len(t, exported, count)
			if count > 0 {
				require.True(t, proto.Equal(receipts[0], exported[0]))
			}
			total, global, account, err := k.factUseAdmissionCounts(ctx, 0, consumer)
			require.NoError(t, err)
			require.Equal(t, uint64(count), total)
			require.Equal(t, total, global)
			require.Equal(t, total, account)
			require.NoError(t, k.PruneFactUseReceipts(ctx.WithBlockHeight(20)))
			found, err = k.HasRetainedFactUseReceipts(ctx)
			require.NoError(t, err)
			require.False(t, found)
		})
	}
}

// Inject above the SDK cache boundary: cacheMergeIterator itself does not
// forward errors from its parent/cache iterators, so those cannot be tested as
// exposed errors or recovered by the keeper compatibility helper.
type iteratorCompatProbeService struct {
	corestore.KVStoreService
	iteratorErr, closeErr error
	closed                int
}

type iteratorCompatProbeStore struct {
	corestore.KVStore
	service *iteratorCompatProbeService
}

type iteratorCompatProbe struct {
	corestore.Iterator
	service *iteratorCompatProbeService
}

func (s *iteratorCompatProbeService) OpenKVStore(ctx context.Context) corestore.KVStore {
	return iteratorCompatProbeStore{s.KVStoreService.OpenKVStore(ctx), s}
}

func (s iteratorCompatProbeStore) Iterator(start, end []byte) (corestore.Iterator, error) {
	it, err := s.KVStore.Iterator(start, end)
	if err != nil {
		return nil, err
	}
	return iteratorCompatProbe{it, s.service}, nil
}

func (it iteratorCompatProbe) Error() error {
	if it.service.iteratorErr != nil {
		return it.service.iteratorErr
	}
	return it.Iterator.Error()
}

func (it iteratorCompatProbe) Close() error {
	it.service.closed++
	return errors.Join(it.Iterator.Close(), it.service.closeErr)
}

func TestReadIteratorExposedError(t *testing.T) {
	for _, nonempty := range []bool{false, true} {
		t.Run(fmt.Sprintf("relations=%t", nonempty), func(t *testing.T) {
			k, ctx, ms := newIteratorCompatKeeper(t)
			require.NoError(t, k.SetFactSkipTransition(ctx, &types.Fact{Id: "root"}))
			if nonempty {
				require.NoError(t, k.SetFactRelation(ctx, &types.FactRelation{SourceFactId: "root", TargetFactId: "child", Relation: types.RelationType_RELATION_TYPE_SUPPORTS}))
			}
			ctx = committedIteratorCompatContext(t, ctx, ms)
			failure := errors.New("exposed backend iterator failure")
			probe := &iteratorCompatProbeService{KVStoreService: k.storeService, iteratorErr: failure}
			k.storeService = probe
			response, err := NewQueryServerImpl(k).Fact(ctx, &types.QueryFactRequest{Id: "root"})
			require.Nil(t, response, "do not return empty/partial relation history on an exposed failure")
			require.Equal(t, codes.Internal, status.Code(err))
			require.ErrorContains(t, err, failure.Error())
			require.Equal(t, 1, probe.closed)
		})
	}
}

func TestFactUseIteratorCompatibilityPreservesCheckedClose(t *testing.T) {
	k, ctx, ms := newIteratorCompatKeeper(t)
	ctx = committedIteratorCompatContext(t, ctx, ms)
	failure := errors.New("exposed iterator failure")
	closeFailure := errors.New("iterator close failure")
	for _, iteratorErr := range []error{nil, failure} {
		probe := &iteratorCompatProbeService{KVStoreService: k.storeService, iteratorErr: iteratorErr, closeErr: closeFailure}
		wrapped := k
		wrapped.storeService = probe
		_, err := wrapped.HasRetainedFactUseReceipts(ctx)
		require.ErrorIs(t, err, closeFailure, "normal EOF compatibility must not suppress close failure")
		if iteratorErr != nil {
			require.ErrorIs(t, err, iteratorErr)
		}
		require.Equal(t, 1, probe.closed)
	}
}
