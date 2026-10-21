package app

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/core/header"
	"cosmossdk.io/log"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/stretchr/testify/require"
	"github.com/zerone-chain/zerone/app/consensusrepairsv2migration"
	"github.com/zerone-chain/zerone/internal/consensusrepairsmigration"
	knowledgemodule "github.com/zerone-chain/zerone/x/knowledge"
	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledge "github.com/zerone-chain/zerone/x/knowledge/types"
)

type archivedRepairsV1KnowledgeModule struct{ knowledgemodule.AppModule }

func (archivedRepairsV1KnowledgeModule) ConsensusVersion() uint64 { return 13 }
func freezeConsensusRepairsV1CompiledTarget(a *ZeroneApp) {
	a.ModuleManager.Modules["knowledge"] = archivedRepairsV1KnowledgeModule{a.ModuleManager.Modules["knowledge"].(knowledgemodule.AppModule)}
	original := a.ModuleManager.Modules["alignment"]
	a.ModuleManager.Modules["alignment"] = archivedConsensusSibling{original.(appmodule.AppModule), original.(module.HasGenesis), 2}
}

func consensusRepairsV2Fixture(t *testing.T) (*ZeroneApp, sdk.Context, dbm.DB) {
	t.Helper()
	db := dbm.NewMemDB()
	a := NewZeroneApp(log.NewNopLogger(), db, nil, true, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()), baseapp.SetChainID("zerone-dev-1"))
	raw, err := json.Marshal(sdk053IBC10GenesisWithValidator(t, a))
	require.NoError(t, err)
	ctx := a.NewUncachedContext(false, cmtproto.Header{ChainID: "zerone-dev-1"})
	_, err = a.InitChainer(ctx, &abci.RequestInitChain{ChainId: ctx.ChainID(), AppStateBytes: raw})
	require.NoError(t, err)
	for _, loc := range consensusRepairsV2MarkerLocations() {
		ctx.KVStore(a.keys[loc.store]).Delete(loc.key)
	}
	require.NoError(t, a.UpgradeKeeper.SetModuleVersionMap(ctx, consensusRepairsV2SourceVersionMap()))
	return a, ctx, db
}
func scheduleRepairsV2(t *testing.T, a *ZeroneApp, ctx sdk.Context) (sdk.Context, upgradetypes.Plan) {
	t.Helper()
	plan := upgradetypes.Plan{Name: UpgradeNameConsensusRepairsV2, Height: a.CommitMultiStore().Commit().Version + 1}
	require.NoError(t, a.UpgradeKeeper.ScheduleUpgrade(ctx, plan))
	require.NoError(t, a.UpgradeKeeper.DumpUpgradeInfoToDisk(plan.Height, plan))
	return ctx.WithBlockHeight(plan.Height).WithHeaderInfo(header.Info{Height: plan.Height, ChainID: ctx.ChainID()}), plan
}
func seedRepairsV2Calibration(t *testing.T, a *ZeroneApp, ctx sdk.Context) *knowledge.AgentCalibration {
	t.Helper()
	c := &knowledge.AgentCalibration{Address: "author", TotalSubmissions: 2, Accepted: 2, DisprovenCount: 1, CalibrationScoreBps: 500_000}
	require.NoError(t, a.KnowledgeKeeper.SetAgentCalibration(ctx, c))
	return c
}

func TestConsensusRepairsV2ExactVersionBoundary(t *testing.T) {
	a, ctx, _ := consensusRepairsV2Fixture(t)
	source, target := consensusRepairsV2SourceVersionMap(), consensusRepairsV2TargetVersionMap()
	require.Equal(t, target, a.CurrentModuleVersionMap())
	var changed []string
	for _, entry := range a.BuildChainVersionReport().Modules {
		if source[entry.ModuleName] != target[entry.ModuleName] {
			changed = append(changed, entry.ModuleName)
			require.Equal(t, source[entry.ModuleName]+1, target[entry.ModuleName])
		}
	}
	require.Equal(t, []string{"alignment", "knowledge"}, changed)
	guards := []func(string, module.VersionMap, module.VersionMap) error{requireConsensusRepairsTransitionOwner, requireAccountingTransitionOwner, requireSurvivalHandoffTransitionOwner, requireRecordIntegrityTransitionOwner, requireReviewNeutralityTransitionOwner, requireClaimRecordsTransitionOwner, requireFundSettlementTransitionOwner, requireClaimIntegrityTransitionOwner}
	for _, guard := range guards {
		for _, name := range append(a.KnownUpgradeNames(), "unknown") {
			err := guard(name, source, target)
			if name == UpgradeNameConsensusRepairsV2 {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, "sole owner")
			}
		}
		require.NoError(t, guard("no-op", target, target))
		for key := range source {
			bad := consensusRepairsV2SourceVersionMap()
			delete(bad, key)
			require.Error(t, guard(UpgradeNameConsensusRepairsV2, bad, target))
		}
		require.Error(t, guard(UpgradeNameConsensusRepairsV2, target, source))
	}
	before := claimIntegrityAllStores(t, a, ctx)
	_, err := a.ModuleManager.RunMigrations(ctx, a.configurator, source)
	require.ErrorContains(t, err, "explicit")
	v1, err := consensusrepairsmigration.WithOwner(ctx, UpgradeNameConsensusRepairsV1)
	require.NoError(t, err)
	_, err = a.ModuleManager.RunMigrations(v1, a.configurator, source)
	require.ErrorContains(t, err, "explicit")
	require.Equal(t, before, claimIntegrityAllStores(t, a, ctx))
}

func TestConsensusRepairsV2AtomicRecomputePreBlockAndRestart(t *testing.T) {
	a, ctx, db := consensusRepairsV2Fixture(t)
	old := seedRepairsV2Calibration(t, a, ctx)
	report, err := a.CensusConsensusRepairsV2(ctx)
	require.NoError(t, err)
	require.Len(t, report.Censuses, 6)
	ctx, plan := scheduleRepairsV2(t, a, ctx)
	require.NoError(t, a.ValidateAccountingAuthorityStartup())
	before := claimIntegrityAllStores(t, a, ctx)
	cache, write := ctx.CacheContext()
	_, err = a.PotPreBlocker(cache, &abci.RequestFinalizeBlock{Height: plan.Height})
	require.NoError(t, err)
	require.Equal(t, before, claimIntegrityAllStores(t, a, ctx))
	updated, found := a.KnowledgeKeeper.GetAgentCalibration(cache, "author")
	require.True(t, found)
	require.Equal(t, knowledgekeeper.ComputeSmoothedAgentCalibrationScore(old), updated.CalibrationScoreBps)
	require.NotEqual(t, old.CalibrationScoreBps, updated.CalibrationScoreBps)
	write()
	after := claimIntegrityAllStores(t, a, ctx)
	for _, loc := range consensusRepairsV2MarkerLocations() {
		key := loc.store + ":" + string(loc.key)
		require.Equal(t, []byte{1}, after[key])
		delete(after, key)
	}
	calKey := "knowledge:" + string(knowledge.AgentCalibrationKey("author"))
	delete(before, calKey)
	delete(after, calKey)
	raw := a.KnowledgeKeeper.ReadMigrationMarker(ctx, consensusRepairsV2ReceiptKey)
	var receipt consensusRepairsReceipt
	require.NoError(t, json.Unmarshal([]byte(raw), &receipt))
	require.Len(t, receipt.Markers, 4)
	require.Len(t, receipt.Census.Censuses, 6)
	require.Less(t, len(raw), 8000, "receipt is summaries, never detailed inventories")
	delete(after, "knowledge:"+string(append([]byte{0x7f, 1}, []byte(consensusRepairsV2ReceiptKey)...)))
	// SDK bookkeeping is outside the carrier's own streamed postcondition.
	for key := range before {
		if strings.HasPrefix(key, "upgrade:") {
			delete(before, key)
		}
	}
	for key := range after {
		if strings.HasPrefix(key, "upgrade:") {
			delete(after, key)
		}
	}
	require.Equal(t, before, after, "all business rows, v1 history and 0x87 unchanged")
	vm, err := a.UpgradeKeeper.GetModuleVersionMap(ctx)
	require.NoError(t, err)
	require.Equal(t, consensusRepairsV2TargetVersionMap(), module.VersionMap(vm))
	a.CommitMultiStore().Commit()
	require.NoError(t, a.ValidateAccountingAuthorityStartup())
	restarted := NewZeroneApp(log.NewNopLogger(), db, nil, true, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()), baseapp.SetChainID(ctx.ChainID()))
	require.NoError(t, restarted.ValidateAccountingAuthorityStartup())
	require.Error(t, a.UpgradeKeeper.ApplyUpgrade(ctx, plan))
}

func TestConsensusRepairsV2LateFailureIsAtomic(t *testing.T) {
	for _, fault := range []string{"last marker", "insert", "delete", "rewrite", "receipt"} {
		t.Run(fault, func(t *testing.T) {
			a, ctx, _ := consensusRepairsV2Fixture(t)
			seedRepairsV2Calibration(t, a, ctx)
			key := []byte("\xffcr2-existing")
			ctx.KVStore(a.keys["bank"]).Set(key, []byte{1, 2})
			cfg := module.NewConfigurator(a.appCodec, a.MsgServiceRouter(), a.GRPCQueryRouter())
			require.NoError(t, cfg.RegisterMigration("knowledge", 13, consensusrepairsv2migration.Require))
			require.NoError(t, cfg.RegisterMigration("alignment", 2, func(cache sdk.Context) error {
				if err := consensusrepairsv2migration.Require(cache); err != nil {
					return err
				}
				switch fault {
				case "last marker":
					cache.KVStore(a.keys["knowledge"]).Set([]byte{0x93}, []byte{2})
				case "insert":
					cache.KVStore(a.keys["bank"]).Set([]byte("\xffcr2-insert"), []byte{3})
				case "delete":
					cache.KVStore(a.keys["bank"]).Delete(key)
				case "rewrite":
					cache.KVStore(a.keys["bank"]).Set(key, []byte{3, 4})
				case "receipt":
					return a.KnowledgeKeeper.WriteMigrationMarker(cache, consensusRepairsV2ReceiptKey, "conflict")
				}
				return nil
			}))
			a.configurator = cfg
			ctx, plan := scheduleRepairsV2(t, a, ctx)
			before := claimIntegrityAllStores(t, a, ctx)
			events := append(sdk.Events{}, ctx.EventManager().Events()...)
			err := a.UpgradeKeeper.ApplyUpgrade(ctx, plan)
			require.Error(t, err)
			if fault == "insert" || fault == "delete" || fault == "rewrite" {
				require.ErrorContains(t, err, "unexpected store diff")
			}
			require.Equal(t, before, claimIntegrityAllStores(t, a, ctx))
			require.Equal(t, events, ctx.EventManager().Events())
		})
	}
}

func TestConsensusRepairsV2CensusBlockersAndReadOnly(t *testing.T) {
	for _, fault := range []string{"calibration", "orphan escrow", "underbacking", "address counter", "epoch counter", "gov zero", "gov max", "gov malformed", "registry", "active index", "marker"} {
		t.Run(fault, func(t *testing.T) {
			a, ctx, _ := consensusRepairsV2Fixture(t)
			seedRepairsV2Calibration(t, a, ctx)
			store := ctx.KVStore(a.keys["knowledge"])
			switch fault {
			case "calibration":
				store.Set([]byte("\x5awrong"), []byte{255})
			case "orphan escrow":
				store.Set([]byte("\x6borphan"), []byte("0"))
			case "underbacking":
				store.Set([]byte("\x6bb"), []byte("1"))
				repairPutProto(t, a, ctx, "knowledge", knowledge.AugmentationBountyKey("b"), &knowledge.AugmentationBounty{Id: "b"})
			case "address counter":
				store.Set([]byte("\x35address"), make([]byte, 9))
			case "epoch counter":
				store.Set([]byte("\x36bad"), make([]byte, 8))
			case "gov zero":
				ctx.KVStore(a.keys["zerone_gov"]).Set([]byte{8}, make([]byte, 8))
			case "gov max":
				ctx.KVStore(a.keys["zerone_gov"]).Set([]byte{8}, binary.BigEndian.AppendUint64(nil, ^uint64(0)))
			case "gov malformed":
				ctx.KVStore(a.keys["zerone_gov"]).Set([]byte{8}, []byte{})
			case "registry":
				ctx.KVStore(a.keys["zerone_staking"]).Set([]byte("\x01bad"), []byte("null"))
			case "active index":
				store.Set([]byte("\x21missing"), []byte{1})
			case "marker":
				store.Set([]byte{0x92}, []byte{})
			}
			ctx, plan := scheduleRepairsV2(t, a, ctx)
			before := claimIntegrityAllStores(t, a, ctx)
			report, err := a.CensusConsensusRepairsV2(ctx)
			require.Error(t, err)
			require.NotEmpty(t, report.Findings)
			require.Error(t, a.ValidateAccountingAuthorityStartup())
			require.Error(t, a.UpgradeKeeper.ApplyUpgrade(ctx, plan))
			require.Equal(t, before, claimIntegrityAllStores(t, a, ctx))
		})
	}
	a, ctx, _ := consensusRepairsV2Fixture(t)
	a.BankKeeper = writingCensusBank{a.BankKeeper, func(c context.Context) {
		sdk.UnwrapSDKContext(c).KVStore(a.keys["knowledge"]).Set([]byte("unwanted"), []byte{1})
	}}
	before := claimIntegrityAllStores(t, a, ctx)
	_, err := a.CensusConsensusRepairsV2(ctx)
	require.NoError(t, err)
	require.Equal(t, before, claimIntegrityAllStores(t, a, ctx))
}

func TestConsensusRepairsV2LargeStateAndSelectedBounds(t *testing.T) {
	a, ctx, _ := consensusRepairsV2Fixture(t)
	base, err := a.CensusConsensusRepairsV2(ctx)
	require.NoError(t, err)
	for i := 0; i < 200_001; i++ {
		ctx.KVStore(a.keys["alignment"]).Set(binary.BigEndian.AppendUint64([]byte{3}, uint64(i)), []byte{1})
	}
	ctx.KVStore(a.keys["bank"]).Set([]byte("\xfflarge"), []byte(strings.Repeat("x", 65<<20)))
	reads, err := consensusRepairsV2Reads()
	require.NoError(t, err)
	var report ConsensusRepairsCensus
	rows, err := a.consensusRepairsCensusRowsFor(ctx, &report, consensusRepairsV2CensusNames, reads)
	require.NoError(t, err)
	require.Equal(t, 2, len(rows["alignment"]), "retain only exact params/state, not unrelated observations")
	require.NotContains(t, rows["bank"], "\xfflarge")
	require.Equal(t, base.Censuses, report.Censuses)
	require.Equal(t, base.Digests, report.Digests)
	ctx, plan := scheduleRepairsV2(t, a, ctx)
	start := time.Now()
	require.NoError(t, a.UpgradeKeeper.ApplyUpgrade(ctx, plan))
	t.Logf("synthetic large-state handler: %s; records=%d bytes=%d", time.Since(start), report.Records, report.Bytes)
	b, bctx, _ := consensusRepairsV2Fixture(t)
	bctx.KVStore(b.keys["knowledge"]).Set([]byte("\x5atoo-large"), []byte(strings.Repeat("x", 64<<20)))
	_, err = b.CensusConsensusRepairsV2(bctx)
	require.ErrorContains(t, err, "resource ceiling")
}

func TestConsensusRepairsV2GenesisAliasesRegistryAndReadback(t *testing.T) {
	a, ctx, _ := consensusRepairsV2Fixture(t)
	for i, marker := range a.ConsensusRepairsV2Markers() {
		require.Equal(t, ActivationSelection{marker.Module, marker.Name}, ConsensusRepairsV2GenesisSelections()[i])
		cache, _ := ctx.CacheContext()
		before := claimIntegrityAllStores(t, a, cache)
		require.NoError(t, marker.Enable(cache))
		after := claimIntegrityAllStores(t, a, cache)
		loc := consensusRepairsV2MarkerLocations()[i]
		key := loc.store + ":" + string(loc.key)
		require.Equal(t, []byte{1}, after[key])
		delete(after, key)
		require.Equal(t, before, after)
		for _, alias := range []string{strings.ToUpper(marker.Name), consensusRepairsCamelCase(marker.Name), strings.Replace(marker.Name, "s", "ſ", 1)} {
			g := sdk053IBC10GenesisWithValidator(t, a)
			raw := g[marker.Module]
			g[marker.Module] = json.RawMessage(fmt.Sprintf(`%s,%q:false}`, raw[:len(raw)-1], alias))
			require.Error(t, ValidateConsensusRepairsGenesisSelection(g))
			fresh := NewZeroneApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()))
			encoded, err := json.Marshal(g)
			require.NoError(t, err)
			_, err = fresh.InitChainer(fresh.NewUncachedContext(false, cmtproto.Header{}), &abci.RequestInitChain{AppStateBytes: encoded})
			require.Error(t, err)
		}
		// Independently exercise post-InitGenesis readback, bypassing no validator.
		fresh := NewZeroneApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()))
		original := fresh.ModuleManager.Modules[marker.Module]
		fresh.ModuleManager.Modules[marker.Module] = repairGenesisFault{archivedConsensusSibling{original.(appmodule.AppModule), original.(module.HasGenesis), fresh.CurrentModuleVersionMap()[marker.Module]}, func(c sdk.Context) { c.KVStore(fresh.keys[loc.store]).Delete(loc.key) }}
		encoded, err := json.Marshal(sdk053IBC10GenesisWithValidator(t, fresh))
		require.NoError(t, err)
		_, err = fresh.InitChainer(fresh.NewUncachedContext(false, cmtproto.Header{}), &abci.RequestInitChain{AppStateBytes: encoded})
		require.ErrorContains(t, err, "readback")
	}
}
