package app

import (
	"encoding/json"
	"testing"

	"cosmossdk.io/core/header"
	"cosmossdk.io/log"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/stretchr/testify/require"
)

func TestConsensusRepairsV2HMinusOneOnlyAndPredecessorRefusal(t *testing.T) {
	a, ctx, db := consensusRepairsV2Fixture(t)
	latest := a.CommitMultiStore().Commit().Version
	plan := upgradetypes.Plan{Name: UpgradeNameConsensusRepairsV2, Height: latest + 2}
	require.NoError(t, a.UpgradeKeeper.ScheduleUpgrade(ctx, plan))
	require.NoError(t, a.UpgradeKeeper.DumpUpgradeInfoToDisk(plan.Height, plan))
	require.ErrorContains(t, a.ValidateAccountingAuthorityStartup(), "H-1")
	require.Equal(t, plan.Height-1, a.CommitMultiStore().Commit().Version)
	require.NoError(t, a.ValidateAccountingAuthorityStartup())
	ctx = ctx.WithBlockHeight(plan.Height).WithHeaderInfo(header.Info{Height: plan.Height, ChainID: ctx.ChainID()})
	require.NoError(t, a.UpgradeKeeper.ApplyUpgrade(ctx, plan))
	a.CommitMultiStore().Commit()
	old := NewZeroneApp(log.NewNopLogger(), db, nil, false, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()), baseapp.SetChainID(ctx.ChainID()))
	freezeConsensusRepairsV1CompiledTarget(old)
	require.NoError(t, old.LoadLatestVersion())
	require.ErrorContains(t, old.ValidateAccountingAuthorityStartup(), "source or target map", "synthetic frozen v1 version contract cannot accept v2 history")
}

func TestConsensusRepairsV2HistoricalCalibrationHandlerCannotUndoQ5(t *testing.T) {
	a, ctx, _ := consensusRepairsV2Fixture(t)
	seedRepairsV2Calibration(t, a, ctx)
	ctx, plan := scheduleRepairsV2(t, a, ctx)
	require.NoError(t, a.UpgradeKeeper.ApplyUpgrade(ctx, plan))
	a.CommitMultiStore().Commit()
	historical := upgradetypes.Plan{Name: UpgradeNameCompassionCalibrationV1, Height: plan.Height + 1}
	require.NoError(t, a.UpgradeKeeper.ScheduleUpgrade(ctx, historical))
	ctx = ctx.WithBlockHeight(historical.Height).WithHeaderInfo(header.Info{Height: historical.Height})
	before := claimIntegrityAllStores(t, a, ctx)
	err := a.UpgradeKeeper.ApplyUpgrade(ctx, historical)
	require.ErrorContains(t, err, "require knowledge=6", "the existing frozen pre-SDK guard prevents a legacy-formula rewrite; no old handler edit needed")
	require.Equal(t, before, claimIntegrityAllStores(t, a, ctx))
}

func TestConsensusRepairsV2ExportImportDoesNotInventHistory(t *testing.T) {
	a, ctx, _ := consensusRepairsV2Fixture(t)
	seedRepairsV2Calibration(t, a, ctx)
	ctx, plan := scheduleRepairsV2(t, a, ctx)
	require.NoError(t, a.UpgradeKeeper.ApplyUpgrade(ctx, plan))
	a.CommitMultiStore().Commit()
	g, err := a.ModuleManager.ExportGenesis(ctx, a.appCodec)
	require.NoError(t, err)
	lineage, err := a.accountingAuthorityGenesisMetadata(ctx, plan.Height)
	require.NoError(t, err)
	g[accountingAuthorityGenesisKey], err = json.Marshal(lineage)
	require.NoError(t, err)
	raw, err := json.Marshal(g)
	require.NoError(t, err)
	b := NewZeroneApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()), baseapp.SetChainID("zerone-rehearsal-v2-import"))
	bctx := b.NewUncachedContext(false, cmtproto.Header{ChainID: "zerone-rehearsal-v2-import"})
	_, err = b.InitChainer(bctx, &abci.RequestInitChain{ChainId: bctx.ChainID(), AppStateBytes: raw})
	require.NoError(t, err)
	for _, m := range b.ConsensusRepairsV2Markers() {
		on, err := m.Enabled(bctx)
		require.NoError(t, err)
		require.True(t, on)
	}
	require.Empty(t, b.KnowledgeKeeper.ReadMigrationMarker(bctx, consensusRepairsV2ReceiptKey))
	done, err := b.UpgradeKeeper.GetDoneHeight(bctx, UpgradeNameConsensusRepairsV2)
	require.NoError(t, err)
	require.Zero(t, done)
	want, found := a.KnowledgeKeeper.GetAgentCalibration(ctx, "author")
	require.True(t, found)
	got, found := b.KnowledgeKeeper.GetAgentCalibration(bctx, "author")
	require.True(t, found)
	require.Equal(t, want.CalibrationScoreBps, got.CalibrationScoreBps)
	b.CommitMultiStore().Commit()
	require.NoError(t, b.ValidateAccountingAuthorityStartup())
}
