package keeper_test

import (
	"context"
	"crypto/sha256"
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
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/creed/keeper"
	"github.com/zerone-chain/zerone/x/creed/types"
)

// Observe reads without replacing the real IAVL store or its iterators.
type countedStoreService struct {
	corestore.KVStoreService
	gets *int
}

func (s countedStoreService) OpenKVStore(ctx context.Context) corestore.KVStore {
	return countedStore{KVStore: s.KVStoreService.OpenKVStore(ctx), gets: s.gets}
}

type countedStore struct {
	corestore.KVStore
	gets *int
}

func (s countedStore) Get(key []byte) ([]byte, error) {
	(*s.gets)++
	return s.KVStore.Get(key)
}

func setupHistoryKeeper(t *testing.T) (keeper.Keeper, sdk.Context, *int) {
	t.Helper()
	key := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	state := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	state.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	require.NoError(t, state.LoadLatestVersion())
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	gets := new(int)
	service := countedStoreService{KVStoreService: runtime.NewKVStoreService(key), gets: gets}
	k := keeper.NewKeeper(service, cdc, "gov-authority")
	ctx := sdk.NewContext(state, cmtproto.Header{Height: 1}, false, log.NewNopLogger())
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return k, ctx, gets
}

func historyPin(version uint32) *types.PinnedCreed {
	hash := sha256.Sum256([]byte(fmt.Sprintf("synthetic creed version %d", version)))
	return &types.PinnedCreed{Version: version, CanonicalHash: hash[:], PinnedAtHeight: 1}
}

func seedHistory(t *testing.T, k keeper.Keeper, ctx sdk.Context, versions ...uint32) *types.GenesisState {
	t.Helper()
	gs := types.DefaultGenesis()
	for _, version := range versions {
		gs.History = append(gs.History, historyPin(version))
	}
	if len(gs.History) > 0 {
		gs.GenesisPin = gs.History[len(gs.History)-1]
		gs.History = gs.History[:len(gs.History)-1]
	}
	require.NoError(t, gs.Validate(), "sparse version histories are valid genesis inputs")
	k.InitGenesis(ctx, gs)
	return gs
}

func pinVersions(pins []*types.PinnedCreed) []uint32 {
	out := make([]uint32, len(pins))
	for i, pin := range pins {
		out[i] = pin.Version
	}
	return out
}

func TestPinHistoryReadsRecordsNotVersionGaps(t *testing.T) {
	k, ctx, gets := setupHistoryKeeper(t)
	seedHistory(t, k, ctx, 1, 500, 1000)
	*gets = 0

	page, err := keeper.NewQueryServerImpl(k).PinHistory(ctx, &types.QueryPinHistoryRequest{Limit: 2})
	require.NoError(t, err)
	require.Equal(t, []uint32{1000, 500}, pinVersions(page.Pins))
	require.Equal(t, uint32(500), page.NextStartBeforeVersion)
	require.LessOrEqual(t, *gets, 8, "a two-record page must not probe hundreds of absent versions")
	t.Logf("two-record page: %d point reads", *gets)
}

func TestIteratePinsDescendingReadsRecordsNotVersionGaps(t *testing.T) {
	for _, stopAfter := range []int{0, 2} {
		t.Run(fmt.Sprintf("stop-after-%d", stopAfter), func(t *testing.T) {
			k, ctx, gets := setupHistoryKeeper(t)
			seedHistory(t, k, ctx, 1, 500, 1000)
			*gets = 0
			pins := []*types.PinnedCreed{}
			k.IteratePinsDescending(ctx, func(pin *types.PinnedCreed) bool {
				pins = append(pins, pin)
				return len(pins) == stopAfter
			})
			want := []uint32{1000, 500, 1}
			if stopAfter > 0 {
				want = want[:stopAfter]
			}
			require.Equal(t, want, pinVersions(pins))
			require.LessOrEqual(t, *gets, 8, "iteration must scale with stored pins, not the highest version")
		})
	}
}

func TestExportGenesisReadsRecordsNotVersionGaps(t *testing.T) {
	k, ctx, gets := setupHistoryKeeper(t)
	want := seedHistory(t, k, ctx, 1, 500, 1000)
	*gets = 0

	got := k.ExportGenesis(ctx)
	require.NoError(t, got.Validate())
	require.True(t, proto.Equal(want, got), "export must preserve the current pin and ascending history")
	require.LessOrEqual(t, *gets, 8, "export must not probe absent versions")
	require.LessOrEqual(t, cap(got.History), 8, "history allocation must scale with stored pins")
}

func TestPinHistoryPagination(t *testing.T) {
	k, ctx, _ := setupHistoryKeeper(t)
	seedHistory(t, k, ctx, 1, 256, 500)
	query := keeper.NewQueryServerImpl(k)
	for _, tc := range []struct {
		name   string
		before uint32
		limit  uint32
		want   []uint32
		next   uint32
	}{
		{"newest", 0, 2, []uint32{500, 256}, 256},
		{"beyond-current", 501, 2, []uint32{500, 256}, 256},
		{"current-exclusive", 500, 2, []uint32{256, 1}, 0},
		{"cursor-in-gap", 300, 1, []uint32{256}, 256},
		{"resume-exclusive", 256, 2, []uint32{1}, 0},
		{"oldest-page", 2, 1, []uint32{1}, 0},
		{"before-oldest", 1, 2, []uint32{}, 0},
		{"exact-full-page", 0, 3, []uint32{500, 256, 1}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			page, err := query.PinHistory(ctx, &types.QueryPinHistoryRequest{
				StartBeforeVersion: tc.before, Limit: tc.limit,
			})
			require.NoError(t, err)
			require.Equal(t, tc.want, pinVersions(page.Pins))
			require.Equal(t, tc.next, page.NextStartBeforeVersion)
			for _, pin := range page.Pins {
				stored, found := k.GetPin(ctx, pin.Version)
				require.True(t, found)
				require.True(t, proto.Equal(stored, pin), "history and version queries must return the same record")
			}
		})
	}
}

func TestPinHistoryLimitContract(t *testing.T) {
	k, ctx, _ := setupHistoryKeeper(t)
	versions := make([]uint32, 120)
	for i := range versions {
		versions[i] = uint32(i + 1)
	}
	seedHistory(t, k, ctx, versions...)
	for _, tc := range []struct{ requested, want uint32 }{
		{0, 50}, {1, 1}, {50, 50}, {100, 100}, {101, 50},
	} {
		t.Run(fmt.Sprintf("limit-%d", tc.requested), func(t *testing.T) {
			page, err := keeper.NewQueryServerImpl(k).PinHistory(ctx, &types.QueryPinHistoryRequest{Limit: tc.requested})
			require.NoError(t, err)
			require.Len(t, page.Pins, int(tc.want))
			for i, pin := range page.Pins {
				require.Equal(t, uint32(120-i), pin.Version)
			}
			require.Equal(t, page.Pins[len(page.Pins)-1].Version, page.NextStartBeforeVersion)
		})
	}
}

func TestHistoryAtMaximumVersion(t *testing.T) {
	k, ctx, gets := setupHistoryKeeper(t)
	maxVersion := ^uint32(0)
	want := seedHistory(t, k, ctx, 1, maxVersion)
	*gets = 0
	query := keeper.NewQueryServerImpl(k)

	first, err := query.PinHistory(ctx, &types.QueryPinHistoryRequest{Limit: 1})
	require.NoError(t, err)
	require.Equal(t, []uint32{maxVersion}, pinVersions(first.Pins))
	require.Equal(t, maxVersion, first.NextStartBeforeVersion)
	last, err := query.PinHistory(ctx, &types.QueryPinHistoryRequest{StartBeforeVersion: first.NextStartBeforeVersion, Limit: 1})
	require.NoError(t, err)
	require.Equal(t, []uint32{1}, pinVersions(last.Pins))
	require.Zero(t, last.NextStartBeforeVersion)

	exported := k.ExportGenesis(ctx)
	require.True(t, proto.Equal(want, exported))
	require.LessOrEqual(t, *gets, 8)
	require.LessOrEqual(t, cap(exported.History), 8)
}

func TestHistoryEmptyState(t *testing.T) {
	k, ctx, _ := setupHistoryKeeper(t)
	want := seedHistory(t, k, ctx)
	for _, before := range []uint32{0, 1, 500} {
		page, err := keeper.NewQueryServerImpl(k).PinHistory(ctx, &types.QueryPinHistoryRequest{StartBeforeVersion: before})
		require.NoError(t, err)
		require.Empty(t, page.Pins)
		require.Zero(t, page.NextStartBeforeVersion)
	}
	k.IteratePinsDescending(ctx, func(*types.PinnedCreed) bool {
		t.Fatal("an empty store must not invoke the callback")
		return true
	})
	require.True(t, proto.Equal(want, k.ExportGenesis(ctx)))
}

func TestHistoryExportImportPreservesRecords(t *testing.T) {
	k, ctx, _ := setupHistoryKeeper(t)
	want := seedHistory(t, k, ctx, 1, 256, 1000)
	want.Params.DirectAnchorEnabled = false
	want.GenesisPin.PinnedViaLip = "synthetic-amendment"
	want.GenesisPin.Commitments = []*types.CommitmentEntry{{Number: 1, IntroducedAtHeight: 1}}
	want.CouncilMembers = []*types.CreedCouncilMember{
		{Address: "synthetic-member-a", Active: true, VotingWeightBps: 100},
		{Address: "synthetic-member-b", Active: false, VotingWeightBps: 200},
	}
	require.NoError(t, want.Validate())
	k.InitGenesis(ctx, want)
	before, err := proto.Marshal(want)
	require.NoError(t, err)

	exported := k.ExportGenesis(ctx)
	require.NoError(t, exported.Validate())
	after, err := proto.Marshal(exported)
	require.NoError(t, err)
	require.Equal(t, before, after, "export must preserve protobuf bytes, including params, commitments and council records")

	restored, restoredCtx, _ := setupHistoryKeeper(t)
	restored.InitGenesis(restoredCtx, exported)
	require.True(t, proto.Equal(exported, restored.ExportGenesis(restoredCtx)))
	originalPage, err := keeper.NewQueryServerImpl(k).PinHistory(ctx, &types.QueryPinHistoryRequest{})
	require.NoError(t, err)
	restoredPage, err := keeper.NewQueryServerImpl(restored).PinHistory(restoredCtx, &types.QueryPinHistoryRequest{})
	require.NoError(t, err)
	require.True(t, proto.Equal(originalPage, restoredPage))
}
