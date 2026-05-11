package keeper_test

import (
	"context"
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

	"github.com/zerone-chain/zerone/x/contribution/keeper"
	"github.com/zerone-chain/zerone/x/contribution/types"
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
	storeService := runtime.NewKVStoreService(storeKey)
	k := keeper.NewKeeper(storeService, cdc, "gov-authority")

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 1}, false, log.NewNopLogger())
	return k, ctx
}

func sample32(seed byte) []byte {
	id := sha256.Sum256([]byte{seed})
	return id[:]
}

func sampleContribution(id []byte, contributor string, class types.ContributionClass, phase types.LifecyclePhase) *types.Contribution {
	return &types.Contribution{
		Id:          id,
		Contributor: contributor,
		Class:       class,
		Phase:       phase,
		Status:      types.ContributionStatus_STATUS_SUBMITTED,
	}
}

func TestKeeper_StoreRoundtrip(t *testing.T) {
	k, ctx := setupKeeper(t)
	c := sampleContribution(sample32(0x01), "zrn1abc", types.ContributionClass_KNOWLEDGE_CLAIM, types.LifecyclePhase_PHASE_KNOWLEDGE)
	require.NoError(t, k.WriteContribution(ctx, c))

	got, ok := k.GetContribution(ctx, c.Id)
	require.True(t, ok)
	require.Equal(t, c.Contributor, got.Contributor)
	require.Equal(t, c.Class, got.Class)
}

func TestKeeper_GetByBackRef(t *testing.T) {
	k, ctx := setupKeeper(t)
	c := sampleContribution(sample32(0x02), "zrn1abc", types.ContributionClass_KNOWLEDGE_CLAIM, types.LifecyclePhase_PHASE_KNOWLEDGE)
	c.BackRef = "claim-42"
	require.NoError(t, k.WriteContribution(ctx, c))

	got, ok := k.GetContributionByBackRef(ctx, "claim-42")
	require.True(t, ok)
	require.Equal(t, c.Id, got.Id)
}

func TestKeeper_TransitionStatus_ForwardOnly(t *testing.T) {
	k, ctx := setupKeeper(t)
	c := sampleContribution(sample32(0x03), "zrn1abc", types.ContributionClass_KNOWLEDGE_CLAIM, types.LifecyclePhase_PHASE_KNOWLEDGE)
	require.NoError(t, k.WriteContribution(ctx, c))

	// Forward transitions OK.
	require.NoError(t, k.TransitionStatus(ctx, c, types.ContributionStatus_STATUS_CLASSIFIED))
	require.NoError(t, k.TransitionStatus(ctx, c, types.ContributionStatus_STATUS_VERIFIED))

	// Backwards rejected.
	err := k.TransitionStatus(ctx, c, types.ContributionStatus_STATUS_SUBMITTED)
	require.ErrorIs(t, err, types.ErrInvalidStatusTransition)
}

func TestKeeper_RegisterAdapter_DuplicatePanics(t *testing.T) {
	k, _ := setupKeeper(t)
	a1 := &fakeAdapter{class: types.ContributionClass_KNOWLEDGE_CLAIM}
	a2 := &fakeAdapter{class: types.ContributionClass_KNOWLEDGE_CLAIM}
	k.RegisterAdapter(a1)
	require.Panics(t, func() { k.RegisterAdapter(a2) })
}

func TestKeeper_GetAdapter_NotRegistered(t *testing.T) {
	k, _ := setupKeeper(t)
	_, ok := k.GetAdapter(types.ContributionClass_TOOL)
	require.False(t, ok)
}

func TestKeeper_GenesisRoundtrip(t *testing.T) {
	k, ctx := setupKeeper(t)
	c1 := sampleContribution(sample32(0x10), "zrn1aa", types.ContributionClass_KNOWLEDGE_CLAIM, types.LifecyclePhase_PHASE_KNOWLEDGE)
	c2 := sampleContribution(sample32(0x11), "zrn1bb", types.ContributionClass_TOOL, types.LifecyclePhase_PHASE_TOOLS)
	gs := &types.GenesisState{Contributions: []*types.Contribution{c1, c2}}

	k.InitGenesis(ctx, gs)
	exported := k.ExportGenesis(ctx)
	require.Len(t, exported.Contributions, 2)
}

// ── helpers ──

type fakeAdapter struct {
	class types.ContributionClass
}

func (f *fakeAdapter) Class() types.ContributionClass { return f.class }
func (f *fakeAdapter) Classify(_ context.Context, _ *types.Contribution) error { return nil }
func (f *fakeAdapter) SubstrateLink(_ context.Context, _ *types.Contribution) (uint32, error) {
	return 0, nil
}
func (f *fakeAdapter) Verify(_ context.Context, _ *types.Contribution) (uint32, error) {
	return 0, nil
}
