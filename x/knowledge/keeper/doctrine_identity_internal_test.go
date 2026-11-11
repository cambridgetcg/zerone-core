package keeper

import (
	"bytes"
	"context"
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

	"github.com/zerone-chain/zerone/internal/activation"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// doctrineMarkerReads counts reads of the doctrine-identity marker and passes
// every call on to the handoff fault store, so one fixture can both count
// reads and inject a read failure.
type doctrineMarkerReads struct {
	corestore.KVStoreService
	reads *int
}

func (s doctrineMarkerReads) OpenKVStore(ctx context.Context) corestore.KVStore {
	return doctrineMarkerReadsStore{s.KVStoreService.OpenKVStore(ctx), s.reads}
}

type doctrineMarkerReadsStore struct {
	corestore.KVStore
	reads *int
}

func (s doctrineMarkerReadsStore) Get(key []byte) ([]byte, error) {
	if bytes.Equal(key, types.DoctrineIdentityEnabledKey) {
		*s.reads++
	}
	return s.KVStore.Get(key)
}

type doctrineMarkerFixture struct {
	handoffFixture
	reads *int
	epoch sdk.Context // f.ctx at the first fitness-epoch height
}

// newDoctrineMarkerFixture is a native-genesis chain (the marker on) holding a
// claim-derived fact labelled doctrine. No bank or staking keeper is wired:
// these tests stop at the marker or run a fitness epoch that moves no coins.
func newDoctrineMarkerFixture(t *testing.T) doctrineMarkerFixture {
	t.Helper()
	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	key := storetypes.NewKVStoreKey(types.StoreKey)
	ms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())
	ctx := sdk.NewContext(ms, cmtproto.Header{Height: 100}, false, log.NewNopLogger())
	fault, reads := new(handoffFault), new(int)
	service := doctrineMarkerReads{handoffFaultService{runtime.NewKVStoreService(key), fault}, reads}
	k := NewKeeper(service, codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), "authority", nil, nil)
	require.NoError(t, k.InitGenesis(ctx, types.DefaultGenesis()))
	require.NoError(t, k.SetFact(ctx, &types.Fact{
		Id: "fact-labelled", ClaimId: "claim-labelled", Category: types.DoctrineCategory,
		Domain: "physics", Content: "Claim-derived doctrine label", Submitter: "author",
		Status: types.FactStatus_FACT_STATUS_VERIFIED, Energy: 500_000, EnergyCap: 1_000_000,
	}))
	params, err := k.GetParams(ctx)
	require.NoError(t, err)
	ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter()).WithEventManager(sdk.NewEventManager())
	*reads = 0
	return doctrineMarkerFixture{
		handoffFixture: handoffFixture{k: k, ctx: ctx, keys: []*storetypes.KVStoreKey{key}, kFault: fault},
		reads:          reads,
		epoch:          ctx.WithBlockHeader(cmtproto.Header{Height: int64(params.FitnessEpochBlocks)}),
	}
}

// A marker that cannot be read, or holds anything but {1}, is a fault, never
// off. ProcessMetabolism returns it before reading any fact. BeginBlocker
// returns it at entry, before any write, rather than log it at the fitness
// epoch and skip metabolism.
func TestDoctrineIdentityMarkerFaultPropagates(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value []byte // nil: the read itself fails
	}{
		{"unreadable", nil},
		{"malformed-0", []byte{0}},
		{"malformed-2", []byte{2}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newDoctrineMarkerFixture(t)
			if tc.value == nil {
				f.kFault.op, f.kFault.prefix = "get", types.DoctrineIdentityEnabledKey
			} else {
				f.ctx.KVStore(f.keys[0]).Set(types.DoctrineIdentityEnabledKey, tc.value)
			}
			before := f.snapshot(t)
			// A fresh meter and event manager, after the snapshot's own reads.
			meter, events := storetypes.NewInfiniteGasMeter(), sdk.NewEventManager()
			ctx := f.ctx.WithGasMeter(meter).WithEventManager(events)
			epoch := f.epoch.WithGasMeter(meter).WithEventManager(events)

			require.ErrorIs(t, f.k.ProcessMetabolism(ctx, 1), activation.ErrMarkerFault)
			require.ErrorIs(t, f.k.BeginBlocker(epoch), activation.ErrMarkerFault)

			require.Zero(t, meter.GasConsumed(), "the marker reads charge no gas")
			require.Empty(t, events.Events())
			require.Equal(t, before, f.snapshot(t), "no fact is metabolised and BeginBlocker halts before any write")
		})
	}
}

// The marker is read once per entry point. At a fitness epoch BeginBlocker
// reads it at entry and metabolism uses that selection without reading it
// again; the standalone ProcessMetabolism reads it once.
func TestDoctrineIdentityMarkerReadOncePerEntryPoint(t *testing.T) {
	f := newDoctrineMarkerFixture(t)
	require.NoError(t, f.k.BeginBlocker(f.epoch))
	require.Equal(t, 1, *f.reads)
	fact, found := f.k.GetFact(f.epoch, "fact-labelled")
	require.True(t, found)
	require.Less(t, fact.Energy, uint64(500_000), "the selection reached metabolism: the claim-derived fact was metabolised")

	f = newDoctrineMarkerFixture(t)
	require.NoError(t, f.k.ProcessMetabolism(f.ctx, 1))
	require.Equal(t, 1, *f.reads)
}
