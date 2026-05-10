package keeper_test

import (
	"crypto/sha256"
	"testing"

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

	"github.com/zerone-chain/zerone/x/work_creed/keeper"
	"github.com/zerone-chain/zerone/x/work_creed/types"
)

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		t.Fatalf("failed to load latest version: %v", err)
	}

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	k := keeper.NewKeeper(runtime.NewKVStoreService(storeKey), cdc, "gov-authority")
	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 1}, false, log.NewNopLogger())

	return k, ctx
}

func samplePin(phase uint32, name string, version uint32, codes []string) *types.PinnedSubCreed {
	hash := sha256.Sum256([]byte(name))
	return &types.PinnedSubCreed{
		Phase:           phase,
		PhaseName:       name,
		Version:         version,
		CanonicalHash:   hash[:],
		AnchoredAtBlock: 0,
		SourceLip:       "",
		CommitmentCodes: codes,
	}
}

func TestSetGetSubCreedPin_Roundtrip(t *testing.T) {
	k, ctx := setupKeeper(t)
	pin := samplePin(0, "foundation", 1, []string{"F1", "F2", "F3"})
	require.NoError(t, k.SetSubCreedPin(ctx, pin))

	got, ok := k.GetLatestSubCreedPin(ctx, 0)
	require.True(t, ok)
	require.Equal(t, pin.PhaseName, got.PhaseName)
	require.Equal(t, pin.Version, got.Version)
	require.Equal(t, pin.CanonicalHash, got.CanonicalHash)
	require.Equal(t, pin.CommitmentCodes, got.CommitmentCodes)
}

func TestSetSubCreedPin_RejectsKnowledgePhase(t *testing.T) {
	k, ctx := setupKeeper(t)
	pin := samplePin(1, "knowledge", 1, []string{})
	err := k.SetSubCreedPin(ctx, pin)
	require.ErrorContains(t, err, "Knowledge phase delegates")
}

func TestSetSubCreedPin_RejectsPhaseOutOfRange(t *testing.T) {
	k, ctx := setupKeeper(t)
	pin := samplePin(9, "out-of-range", 1, []string{})
	err := k.SetSubCreedPin(ctx, pin)
	require.ErrorContains(t, err, "out of range")
}

func TestSetSubCreedPin_RejectsBadHashLength(t *testing.T) {
	k, ctx := setupKeeper(t)
	pin := samplePin(0, "foundation", 1, []string{"F1"})
	pin.CanonicalHash = []byte{0x01, 0x02} // not 32 bytes
	err := k.SetSubCreedPin(ctx, pin)
	require.ErrorContains(t, err, "must be 32 bytes")
}

func TestGetLatestSubCreedPin_AbsentReturnsFalse(t *testing.T) {
	k, ctx := setupKeeper(t)
	_, ok := k.GetLatestSubCreedPin(ctx, 0)
	require.False(t, ok)
}

func TestIterateSubCreedPins_OrdersByPhase(t *testing.T) {
	k, ctx := setupKeeper(t)
	require.NoError(t, k.SetSubCreedPin(ctx, samplePin(7, "substrate", 1, []string{"S1"})))
	require.NoError(t, k.SetSubCreedPin(ctx, samplePin(0, "foundation", 1, []string{"F1"})))
	require.NoError(t, k.SetSubCreedPin(ctx, samplePin(2, "curation", 1, []string{"C1"})))

	var got []uint32
	k.IterateSubCreedPins(ctx, func(p *types.PinnedSubCreed) bool {
		got = append(got, p.Phase)
		return false
	})
	require.Equal(t, []uint32{0, 2, 7}, got)
}

func TestInitExportGenesis_Roundtrip(t *testing.T) {
	k, ctx := setupKeeper(t)
	gs := &types.GenesisState{
		PinnedSubCreeds: []*types.PinnedSubCreed{
			samplePin(0, "foundation", 1, []string{"F1", "F2", "F3"}),
			samplePin(2, "curation", 1, []string{"C1", "C2", "C3"}),
		},
	}
	k.InitGenesis(ctx, gs)
	exported := k.ExportGenesis(ctx)
	require.True(t, gs.Equal(exported), "init+export must roundtrip")
}
