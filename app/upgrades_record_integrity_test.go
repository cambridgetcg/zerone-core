package app

import (
	"bytes"
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
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/internal/recordmigration"
	knowledgemodule "github.com/zerone-chain/zerone/x/knowledge"
	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

type archivedRecordKnowledgeModule struct{ knowledgemodule.AppModule }

func (archivedRecordKnowledgeModule) ConsensusVersion() uint64 { return 8 }
func freezeRecordCompiledTarget(app *ZeroneApp) {
	app.ModuleManager.Modules["knowledge"] = archivedRecordKnowledgeModule{app.ModuleManager.Modules["knowledge"].(knowledgemodule.AppModule)}
}
func newFrozenRecordFixture(t *testing.T) (*ZeroneApp, sdk.Context, dbm.DB) {
	t.Helper()
	app, ctx, db := newAccountingAuthorityFixture(t)
	freezeRecordCompiledTarget(app)
	ctx.KVStore(app.keys["knowledge"]).Delete([]byte(knowledgekeeper.FundSettlementEnabledStoreKey))
	ctx.KVStore(app.keys["knowledge"]).Delete([]byte(knowledgekeeper.ClaimRecordsEnabledStoreKey))
	ctx.KVStore(app.keys["knowledge"]).Delete([]byte(knowledgekeeper.ReviewNeutralityEnabledStoreKey))
	require.NoError(t, app.UpgradeKeeper.SetModuleVersionMap(ctx, recordIntegrityTargetVersionMap()))
	return app, ctx, db
}

func recordIntegritySourceFixture(t *testing.T) (*ZeroneApp, sdk.Context, dbm.DB) {
	t.Helper()
	app, ctx, db := newFrozenRecordFixture(t)
	ctx.KVStore(app.keys["knowledge"]).Delete([]byte(knowledgekeeper.RecordIntegrityEnabledStoreKey))
	require.NoError(t, app.UpgradeKeeper.SetModuleVersionMap(ctx, recordIntegritySourceVersionMap()))
	return app, ctx, db
}

func TestRecordIntegrityOwnsExactVersionBoundary(t *testing.T) {
	app, _, _ := newFrozenRecordFixture(t)
	source, target := recordIntegritySourceVersionMap(), recordIntegrityTargetVersionMap()
	require.Equal(t, target, app.CurrentModuleVersionMap())
	require.Equal(t, uint64(7), survivalHandoffTargetVersionMap()["knowledge"])
	require.Equal(t, uint64(6), accountingAuthorityTargetVersionMap()["knowledge"])
	require.True(t, app.UpgradeKeeper.HasHandler(UpgradeNameKnowledgeRecordIntegrityV1))
	for _, name := range append(app.KnownUpgradeNames(), "unrelated") {
		err := requireRecordIntegrityTransitionOwner(name, source, target)
		if name == UpgradeNameKnowledgeRecordIntegrityV1 {
			require.NoError(t, err)
		} else {
			require.ErrorContains(t, err, "sole owner")
		}
	}
	for _, invalid := range []module.VersionMap{{}, {"knowledge": 9}, {"knowledge": 0}} {
		require.Error(t, requireRecordIntegrityTransitionOwner(UpgradeNameKnowledgeRecordIntegrityV1, invalid, target))
	}
	require.Error(t, requireRecordIntegrityTransitionOwner(UpgradeNameKnowledgeRecordIntegrityV1, target, source))
	_, err := recordmigration.WithOwner(sdk.Context{}, UpgradeNameSurvivalRewardHandoffV1)
	require.Error(t, err)
}

func TestRecordIntegrityMigrationPreservesLegacyRoundsAndRestarts(t *testing.T) {
	app, ctx, db := recordIntegritySourceFixture(t)
	old := &knowledgetypes.VerificationRound{Id: "legacy-review", ClaimId: "legacy-claim", Phase: knowledgetypes.VerificationPhase_VERIFICATION_PHASE_COMMIT,
		CommitDeadline: 100, RevealDeadline: 200, AggregationDeadline: 220}
	claim := &knowledgetypes.Claim{Id: old.ClaimId, VerificationRoundId: old.Id, FactContent: "Preserved legacy claim", Domain: "general", Stake: "100000", Status: knowledgetypes.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION}
	require.NoError(t, app.KnowledgeKeeper.SetClaim(ctx, claim))
	require.NoError(t, app.KnowledgeKeeper.SetVerificationRound(ctx, old))
	key := knowledgetypes.RoundKey(old.Id)
	before := bytes.Clone(ctx.KVStore(app.keys["knowledge"]).Get(key))
	supply := app.BankKeeper.GetSupply(ctx, "uzrn")
	latest := app.CommitMultiStore().Commit().Version
	plan := upgradetypes.Plan{Name: UpgradeNameKnowledgeRecordIntegrityV1, Height: latest + 2}
	require.NoError(t, app.UpgradeKeeper.ScheduleUpgrade(ctx, plan))
	require.NoError(t, app.UpgradeKeeper.DumpUpgradeInfoToDisk(plan.Height, plan))
	require.Error(t, app.ValidateAccountingAuthorityStartup(), "candidate must not execute an early legacy block")
	require.Equal(t, plan.Height-1, app.CommitMultiStore().Commit().Version)
	require.NoError(t, app.ValidateAccountingAuthorityStartup())
	ctx = ctx.WithBlockHeight(plan.Height).WithHeaderInfo(header.Info{Height: plan.Height, ChainID: ctx.ChainID()})
	cache, write := ctx.CacheContext()
	require.NoError(t, app.UpgradeKeeper.ApplyUpgrade(cache, plan))
	enabled, err := app.KnowledgeKeeper.RecordIntegrityEnabled(ctx)
	require.NoError(t, err)
	require.False(t, enabled, "outer block cache owns the whole migration")
	write()
	require.Equal(t, before, ctx.KVStore(app.keys["knowledge"]).Get(key))
	require.Equal(t, supply, app.BankKeeper.GetSupply(ctx, "uzrn"))
	require.Equal(t, plan.Height, app.CommitMultiStore().Commit().Version)
	require.NoError(t, app.ValidateAccountingAuthorityStartup())
	require.Error(t, app.UpgradeKeeper.ApplyUpgrade(ctx, plan))

	restarted := NewZeroneApp(log.NewNopLogger(), db, nil, false, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()), baseapp.SetChainID(ctx.ChainID()))
	freezeRecordCompiledTarget(restarted)
	require.NoError(t, restarted.LoadLatestVersion())
	require.NoError(t, restarted.ValidateAccountingAuthorityStartup())
	rctx := restarted.NewUncachedContext(false, cmtproto.Header{Height: plan.Height, ChainID: ctx.ChainID()})
	require.Equal(t, before, rctx.KVStore(restarted.keys["knowledge"]).Get(key))
	newClaim := &knowledgetypes.Claim{Id: "new-claim", Domain: "general", FactContent: "New separately authored contribution", Stake: "100000"}
	require.NoError(t, restarted.KnowledgeKeeper.SetClaim(rctx, newClaim))
	newRound, err := restarted.KnowledgeKeeper.CreateVerificationRound(rctx, newClaim)
	require.NoError(t, err)
	require.Equal(t, uint32(2), newRound.CommitmentScheme)
	require.Equal(t, rctx.ChainID(), newRound.CommitmentChainId)
	downgraded := proto.Clone(newRound).(*knowledgetypes.VerificationRound)
	downgraded.CommitmentScheme, downgraded.CommitmentChainId = 0, ""
	require.Error(t, restarted.KnowledgeKeeper.SetVerificationRound(rctx, downgraded))

	genesis, err := restarted.ModuleManager.ExportGenesis(rctx, restarted.appCodec)
	require.NoError(t, err)
	lineage, err := restarted.accountingAuthorityGenesisMetadata(rctx, plan.Height)
	require.NoError(t, err)
	genesis[accountingAuthorityGenesisKey], err = json.Marshal(lineage)
	require.NoError(t, err)
	raw, err := json.Marshal(genesis)
	require.NoError(t, err)
	imported := NewZeroneApp(log.NewNopLogger(), dbm.NewMemDB(), nil, false, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()), baseapp.SetChainID("record-import"))
	freezeRecordCompiledTarget(imported)
	require.NoError(t, imported.LoadLatestVersion())
	ictx := imported.NewUncachedContext(false, cmtproto.Header{ChainID: "record-import"})
	_, err = imported.InitChainer(ictx, &abci.RequestInitChain{ChainId: "record-import", AppStateBytes: raw})
	require.NoError(t, err)
	retained, found := imported.KnowledgeKeeper.GetVerificationRound(ictx, newRound.Id)
	require.True(t, found)
	require.True(t, proto.Equal(newRound, retained), "import keeps the original commitment chain")
	require.Empty(t, imported.KnowledgeKeeper.ReadMigrationMarker(ictx, "migration_v8_complete"))
	done, err := imported.UpgradeKeeper.GetDoneHeight(ictx, plan.Name)
	require.NoError(t, err)
	require.Zero(t, done)
}

func TestRecordIntegrityRejectsSourceOrMigrationCorruptionAtomically(t *testing.T) {
	for _, scenario := range []string{"source already enabled", "mixed version", "missing version", "unknown version", "plan info", "malformed round", "scheme 2 predecessor", "corrupt history"} {
		t.Run(scenario, func(t *testing.T) {
			app, ctx, _ := recordIntegritySourceFixture(t)
			store := ctx.KVStore(app.keys["knowledge"])
			vm := recordIntegritySourceVersionMap()
			switch scenario {
			case "source already enabled":
				require.NoError(t, app.KnowledgeKeeper.EnableRecordIntegrity(ctx))
			case "mixed version":
				vm["vesting_rewards"] = 2
			case "missing version":
				delete(vm, "bank")
			case "unknown version":
				vm["unexpected"] = 1
			case "malformed round":
				store.Set(knowledgetypes.RoundKey("bad"), []byte{255})
			case "scheme 2 predecessor":
				round := &knowledgetypes.VerificationRound{Id: "bad", ClaimId: "claim", Phase: knowledgetypes.VerificationPhase_VERIFICATION_PHASE_COMMIT, CommitmentScheme: 2, CommitmentChainId: ctx.ChainID()}
				bz, err := proto.Marshal(round)
				require.NoError(t, err)
				store.Set(knowledgetypes.RoundKey("bad"), bz)
			case "corrupt history":
				store.Set(knowledgetypes.StatusTransitionKey("fact", 1), []byte{255})
			}
			beforeBad := bytes.Clone(store.Get(knowledgetypes.RoundKey("bad")))
			plan := upgradetypes.Plan{Name: UpgradeNameKnowledgeRecordIntegrityV1, Height: app.CommitMultiStore().Commit().Version + 1}
			if scenario == "plan info" {
				plan.Info = "unexpected"
			}
			ctx = ctx.WithBlockHeight(plan.Height).WithHeaderInfo(header.Info{Height: plan.Height, ChainID: ctx.ChainID()})
			_, err := app.RunUpgradeHandlerWithInfoForTests(ctx, plan.Name, vm, plan.Height, plan.Info)
			require.Error(t, err)
			require.Empty(t, app.KnowledgeKeeper.ReadMigrationMarker(ctx, "migration_v8_complete"))
			done, err := app.UpgradeKeeper.GetDoneHeight(ctx, plan.Name)
			require.NoError(t, err)
			require.Zero(t, done)
			require.Equal(t, beforeBad, store.Get(knowledgetypes.RoundKey("bad")))
		})
	}
}

func TestRecordIntegrityTargetRequiresNativeSelectionAndCoherentReceipt(t *testing.T) {
	for _, scenario := range []string{"missing selection", "false selection", "marker only", "done only", "future done", "corrupt enabled"} {
		t.Run(scenario, func(t *testing.T) {
			app, ctx, _ := newFrozenRecordFixture(t)
			if scenario == "missing selection" || scenario == "false selection" {
				genesis := sdk053IBC10GenesisWithValidator(t, app)
				var knowledge map[string]json.RawMessage
				require.NoError(t, json.Unmarshal(genesis["knowledge"], &knowledge))
				delete(knowledge, "record_integrity_enabled")
				if scenario == "false selection" {
					knowledge["record_integrity_enabled"] = json.RawMessage("false")
				}
				genesis["knowledge"], _ = json.Marshal(knowledge)
				require.Error(t, validateRecordIntegrityGenesisSelection(genesis))
				return
			}
			if scenario == "marker only" || scenario == "future done" {
				require.NoError(t, app.KnowledgeKeeper.WriteMigrationMarker(ctx, "migration_v8_complete", "true"))
			}
			if scenario == "done only" {
				writeSDK053IBC10TestDoneHeight(app, ctx, UpgradeNameKnowledgeRecordIntegrityV1, 1)
			}
			if scenario == "future done" {
				writeSDK053IBC10TestDoneHeight(app, ctx, UpgradeNameKnowledgeRecordIntegrityV1, 2)
			}
			if scenario == "corrupt enabled" {
				ctx.KVStore(app.keys["knowledge"]).Set([]byte(knowledgekeeper.RecordIntegrityEnabledStoreKey), []byte{2})
			}
			app.CommitMultiStore().Commit()
			require.Error(t, app.ValidateAccountingAuthorityStartup())
		})
	}
}

func TestRecordIntegrityModuleRefusesUnownedMigration(t *testing.T) {
	app, ctx, _ := recordIntegritySourceFixture(t)
	_, err := app.ModuleManager.RunMigrations(ctx, app.configurator, recordIntegritySourceVersionMap())
	require.ErrorContains(t, err, "explicit")
	enabled, err := app.KnowledgeKeeper.RecordIntegrityEnabled(ctx)
	require.NoError(t, err)
	require.False(t, enabled)
}
