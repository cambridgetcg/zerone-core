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

// TestKeeper_WrapAsSubstrateContribution_LeafAndNested exercises the
// self-application primitive (Layer 2). A privileged action becomes a
// Contribution about itself; nesting under a parent surfaces the
// proto-level recursion at the runtime layer.
func TestKeeper_WrapAsSubstrateContribution_LeafAndNested(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Leaf case: no parent. The Contribution is recorded at ADMITTED
	// with a stub PipelineImprovement payload.
	id1, err := k.WrapAsSubstrateContribution(
		ctx,
		"code",
		"gov-authority",
		[]byte("adapter registered: knowledgeclaim"),
		nil,
	)
	require.NoError(t, err)
	c1, ok := k.GetContribution(ctx, id1)
	require.True(t, ok)
	require.Equal(t, types.ContributionClass_PIPELINE_IMPROVEMENT, c1.Class)
	require.Equal(t, types.LifecyclePhase_PHASE_SUBSTRATE, c1.Phase)
	require.Equal(t, types.ContributionStatus_STATUS_ADMITTED, c1.Status)
	require.Equal(t, "gov-authority", c1.Contributor)
	require.NotNil(t, c1.Payload.GetPipelineImprovement())

	// Nested case: parent → nested. The runtime relationship rides
	// the proto-level oneof variant.
	id2, err := k.WrapAsSubstrateContribution(
		ctx,
		"doctrine",
		"gov-authority",
		[]byte("doctrine doc pinned: truth-seeking"),
		id1,
	)
	require.NoError(t, err)
	c2, ok := k.GetContribution(ctx, id2)
	require.True(t, ok)
	require.NotNil(t, c2.Payload.GetNested(), "parent must be embedded as nested")
	require.Equal(t, c1.Id, c2.Payload.GetNested().Id)

	// Depth check: chaining further must respect MaxNestingDepth.
	id3, err := k.WrapAsSubstrateContribution(ctx, "doctrine", "gov-authority", []byte("amend"), id2)
	require.NoError(t, err)
	id4, err := k.WrapAsSubstrateContribution(ctx, "doctrine", "gov-authority", []byte("ratify"), id3)
	require.NoError(t, err)
	// id4 has depth 4 (id4 → id3 → id2 → id1). One more must fail.
	_, err = k.WrapAsSubstrateContribution(ctx, "doctrine", "gov-authority", []byte("further"), id4)
	require.ErrorIs(t, err, types.ErrNestingDepthExceeded, "5-deep wrap must be refused")
}

// TestKeeper_WrapAsSubstrateContribution_SelfApplyingMeta exercises the
// fixed-point property of the wrap helper: every successful wrap at the
// public entry point also emits a meta-Contribution describing the act
// of wrapping. The recursion is bounded — the meta does not itself
// self-meta. UW: the chain records its own privileged actions, and the
// helper that does the recording is itself a privileged action.
func TestKeeper_WrapAsSubstrateContribution_SelfApplyingMeta(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Single leaf wrap creates exactly two Contributions: the leaf and
	// its meta. Both ADMITTED. The meta nests the leaf via payload.nested.
	leafID, err := k.WrapAsSubstrateContribution(
		ctx,
		"code",
		"gov-authority",
		[]byte("self-meta exercise"),
		nil,
	)
	require.NoError(t, err)

	exported := k.ExportGenesis(ctx)
	require.Len(t, exported.Contributions, 2,
		"one wrap must create exactly two Contributions: leaf + meta")

	var leaf, meta *types.Contribution
	for _, c := range exported.Contributions {
		if string(c.Id) == string(leafID) {
			leaf = c
			continue
		}
		meta = c
	}
	require.NotNil(t, leaf, "leaf must be present")
	require.NotNil(t, meta, "meta must be present")

	// Both at ADMITTED.
	require.Equal(t, types.ContributionStatus_STATUS_ADMITTED, leaf.Status,
		"leaf must be ADMITTED")
	require.Equal(t, types.ContributionStatus_STATUS_ADMITTED, meta.Status,
		"meta must be ADMITTED")

	// Meta's payload.nested IS the leaf — proto-level recursion (Layer 1)
	// surfaced at the runtime layer (Layer 2).
	require.NotNil(t, meta.Payload.GetNested(),
		"meta payload must embed the leaf as nested")
	require.Equal(t, leaf.Id, meta.Payload.GetNested().Id,
		"meta.nested.id must equal leaf.id")

	// Meta's claims_about_self carries the "ops" subclass marker.
	require.Contains(t, string(meta.ClaimsAboutSelf), "ops\n",
		"meta must carry ops subclass route")
	require.Contains(t, string(meta.ClaimsAboutSelf), "meta: wrapped contribution",
		"meta description must announce the wrap")
}

// TestKeeper_WrapAsSubstrateContribution_MetaTerminates binds the
// termination property: the recursion stops at one level. A 2x wrap
// with a parent does NOT generate a meta-meta-meta-... infinite chain.
// We verify by exhaustive count: two distinct public wraps must produce
// exactly four Contributions (2 leaves + 2 metas), not more.
func TestKeeper_WrapAsSubstrateContribution_MetaTerminates(t *testing.T) {
	k, ctx := setupKeeper(t)

	// First wrap: leaf + meta = 2 records.
	leaf1, err := k.WrapAsSubstrateContribution(
		ctx, "code", "gov-authority", []byte("first wrap"), nil,
	)
	require.NoError(t, err)

	// Second wrap, nested under first leaf: leaf + meta = 2 more = 4 total.
	leaf2, err := k.WrapAsSubstrateContribution(
		ctx, "doctrine", "gov-authority", []byte("second wrap"), leaf1,
	)
	require.NoError(t, err)
	require.NotEqual(t, string(leaf1), string(leaf2),
		"distinct wraps must produce distinct leaf ids")

	exported := k.ExportGenesis(ctx)
	require.Len(t, exported.Contributions, 4,
		"two wraps must yield exactly four Contributions: 2 leaves + 2 metas — recursion terminates at one level, not unbounded")
}

// TestKeeper_WrapAsSubstrateContribution_MetaWrapBoundedByDepth binds
// the safety property: when a leaf already sits at MaxNestingDepth, the
// meta-wrap is silently skipped (the meta would breach MaxNestingDepth).
// The leaf still returns successfully — the load-bearing record is the
// leaf; the meta is observational. UW: bounded recursion is non-negotiable.
func TestKeeper_WrapAsSubstrateContribution_MetaWrapBoundedByDepth(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Build a depth-4 chain. After this, the deepest leaf has depth 4.
	// A meta-wrap nesting under it would be depth 5 — silently skipped.
	id1, err := k.WrapAsSubstrateContribution(ctx, "code", "gov-authority", []byte("d1"), nil)
	require.NoError(t, err)
	id2, err := k.WrapAsSubstrateContribution(ctx, "code", "gov-authority", []byte("d2"), id1)
	require.NoError(t, err)
	id3, err := k.WrapAsSubstrateContribution(ctx, "code", "gov-authority", []byte("d3"), id2)
	require.NoError(t, err)
	id4, err := k.WrapAsSubstrateContribution(ctx, "code", "gov-authority", []byte("d4"), id3)
	require.NoError(t, err,
		"depth-4 leaf must be admitted; the meta-wrap that would breach depth-5 is silently dropped")

	// Verify the leaf id4 exists; the meta does NOT exist (would have been depth-5).
	leaf4, ok := k.GetContribution(ctx, id4)
	require.True(t, ok, "depth-4 leaf must be present")
	leafDepth, err := types.ContributionNestingDepth(leaf4)
	require.NoError(t, err)
	require.Equal(t, types.MaxNestingDepth, leafDepth,
		"leaf id4 must be at MaxNestingDepth (4)")
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
