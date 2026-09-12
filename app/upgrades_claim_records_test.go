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
	"github.com/zerone-chain/zerone/internal/claimrecordmigration"
	knowledgemodule "github.com/zerone-chain/zerone/x/knowledge"
	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	"google.golang.org/protobuf/proto"
)

type archivedClaimRecordsKnowledgeModule struct{ knowledgemodule.AppModule }

func (archivedClaimRecordsKnowledgeModule) ConsensusVersion() uint64 { return 10 }
func freezeClaimRecordsCompiledTarget(app *ZeroneApp) {
	app.ModuleManager.Modules["knowledge"] = archivedClaimRecordsKnowledgeModule{app.ModuleManager.Modules["knowledge"].(knowledgemodule.AppModule)}
}
func newFrozenClaimRecordsFixture(t *testing.T) (*ZeroneApp, sdk.Context, dbm.DB) {
	t.Helper()
	app, ctx, db := newAccountingAuthorityFixture(t)
	freezeClaimRecordsCompiledTarget(app)
	ctx.KVStore(app.keys[knowledgetypes.StoreKey]).Delete([]byte(knowledgekeeper.FundSettlementEnabledStoreKey))
	require.NoError(t, app.UpgradeKeeper.SetModuleVersionMap(ctx, claimRecordsTargetVersionMap()))
	return app, ctx, db
}

func claimRecordsSourceFixture(t *testing.T) (*ZeroneApp, sdk.Context, dbm.DB) {
	t.Helper()
	app, ctx, db := newFrozenClaimRecordsFixture(t)
	ctx.KVStore(app.keys[knowledgetypes.StoreKey]).Delete([]byte(knowledgekeeper.ClaimRecordsEnabledStoreKey))
	require.NoError(t, app.UpgradeKeeper.SetModuleVersionMap(ctx, claimRecordsSourceVersionMap()))
	return app, ctx, db
}

func TestClaimRecordsOwnsExactVersionBoundary(t *testing.T) {
	app, _, _ := newFrozenClaimRecordsFixture(t)
	source, target := claimRecordsSourceVersionMap(), claimRecordsTargetVersionMap()
	require.Equal(t, target, app.CurrentModuleVersionMap())
	require.Equal(t, uint64(9), reviewNeutralityTargetVersionMap()["knowledge"])
	require.True(t, app.UpgradeKeeper.HasHandler(UpgradeNameKnowledgeClaimRecordsV1))
	for _, name := range append(app.KnownUpgradeNames(), "unrelated") {
		for _, guard := range []func(string, module.VersionMap, module.VersionMap) error{requireClaimRecordsTransitionOwner, requireReviewNeutralityTransitionOwner, requireRecordIntegrityTransitionOwner, requireSurvivalHandoffTransitionOwner} {
			err := guard(name, source, target)
			if name == UpgradeNameKnowledgeClaimRecordsV1 {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, "sole owner")
			}
		}
	}
	for _, invalid := range []module.VersionMap{{}, {"knowledge": 11}, {"knowledge": 0}, {"knowledge": 8}} {
		require.Error(t, requireClaimRecordsTransitionOwner(UpgradeNameKnowledgeClaimRecordsV1, invalid, target))
	}
	require.Error(t, requireClaimRecordsTransitionOwner(UpgradeNameKnowledgeClaimRecordsV1, target, source))
	_, err := claimrecordmigration.WithOwner(sdk.Context{}, UpgradeNameKnowledgeReviewNeutralityV1)
	require.Error(t, err)
}

func TestClaimRecordsMigrationPreservesRecordsObligationsAndRestart(t *testing.T) {
	app, ctx, db := claimRecordsSourceFixture(t)
	claim := &knowledgetypes.Claim{Id: "old-claim", VerificationRoundId: "old-round", FactContent: "Historical content", Domain: "physics", Stake: "100", ArgumentText: "Already recorded argument", EvidenceIds: []string{"old-evidence"}}
	round := &knowledgetypes.VerificationRound{Id: claim.VerificationRoundId, ClaimId: claim.Id, Phase: knowledgetypes.VerificationPhase_VERIFICATION_PHASE_COMPLETE, Verdict: knowledgetypes.Verdict_VERDICT_REJECT, VerdictBlock: 1, VerifierRewardSettlement: &knowledgetypes.VerifierRewardSettlement{CreatedAtBlock: 1, WithheldTotal: "0", Payments: []*knowledgetypes.VerifierRewardPayment{{Verifier: settlementAddress(212).String(), Amount: "55", Withheld: "0"}}}}
	require.NoError(t, app.KnowledgeKeeper.SetClaim(ctx, claim))
	require.NoError(t, app.KnowledgeKeeper.SetVerificationRound(ctx, round))
	dropped := &knowledgetypes.Claim{Id: "old-contradiction", FactContent: "Recorded counter assertion", Stake: "100", Domain: "physics"}
	require.NoError(t, app.KnowledgeKeeper.SetClaim(ctx, dropped))
	before := claimRecordsStores(t, app, ctx)
	latest := app.CommitMultiStore().Commit().Version
	plan := upgradetypes.Plan{Name: UpgradeNameKnowledgeClaimRecordsV1, Height: latest + 2}
	require.NoError(t, app.UpgradeKeeper.ScheduleUpgrade(ctx, plan))
	require.NoError(t, app.UpgradeKeeper.DumpUpgradeInfoToDisk(plan.Height, plan))
	require.Error(t, app.ValidateAccountingAuthorityStartup(), "new binary cannot execute an early v9 block")
	require.Equal(t, plan.Height-1, app.CommitMultiStore().Commit().Version)
	require.NoError(t, app.ValidateAccountingAuthorityStartup())
	ctx = ctx.WithBlockHeight(plan.Height).WithHeaderInfo(header.Info{Height: plan.Height, ChainID: ctx.ChainID()})
	cache, write := ctx.CacheContext()
	require.NoError(t, app.UpgradeKeeper.ApplyUpgrade(cache, plan))
	enabled, err := app.KnowledgeKeeper.ClaimRecordsEnabled(ctx)
	require.NoError(t, err)
	require.False(t, enabled, "outer block cache owns activation")
	require.Equal(t, before, claimRecordsStores(t, app, ctx))
	write()
	after := claimRecordsStores(t, app, ctx)
	delete(after, knowledgetypes.StoreKey+":"+knowledgekeeper.ClaimRecordsEnabledStoreKey)
	delete(after, knowledgetypes.StoreKey+":"+string(append([]byte{0x7f, 0x01}, []byte("migration_v10_complete")...)))
	require.Equal(t, before, after, "only activation and its completion marker change knowledge/bank/auth state")
	require.Equal(t, plan.Height, app.CommitMultiStore().Commit().Version)
	require.NoError(t, app.ValidateAccountingAuthorityStartup())
	require.Error(t, app.UpgradeKeeper.ApplyUpgrade(ctx, plan))
	restarted := NewZeroneApp(log.NewNopLogger(), db, nil, false, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()), baseapp.SetChainID(ctx.ChainID()))
	freezeClaimRecordsCompiledTarget(restarted)
	require.NoError(t, restarted.LoadLatestVersion())
	require.NoError(t, restarted.ValidateAccountingAuthorityStartup())
	rctx := restarted.NewUncachedContext(false, cmtproto.Header{Height: plan.Height, ChainID: ctx.ChainID()})
	retained, found := restarted.KnowledgeKeeper.GetVerificationRound(rctx, round.Id)
	require.True(t, found)
	require.True(t, proto.Equal(round, retained))
	old, found := restarted.KnowledgeKeeper.GetClaim(rctx, dropped.Id)
	require.True(t, found)
	require.Empty(t, old.ArgumentText)
	require.Empty(t, old.EvidenceIds)
	genesis, err := restarted.ModuleManager.ExportGenesis(rctx, restarted.appCodec)
	require.NoError(t, err)
	lineage, err := restarted.accountingAuthorityGenesisMetadata(rctx, plan.Height)
	require.NoError(t, err)
	genesis[accountingAuthorityGenesisKey], err = json.Marshal(lineage)
	require.NoError(t, err)
	raw, err := json.Marshal(genesis)
	require.NoError(t, err)
	imported := NewZeroneApp(log.NewNopLogger(), dbm.NewMemDB(), nil, false, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()), baseapp.SetChainID("claim-record-import"))
	freezeClaimRecordsCompiledTarget(imported)
	require.NoError(t, imported.LoadLatestVersion())
	ictx := imported.NewUncachedContext(false, cmtproto.Header{ChainID: "claim-record-import"})
	_, err = imported.InitChainer(ictx, &abci.RequestInitChain{ChainId: "claim-record-import", AppStateBytes: raw})
	require.NoError(t, err)
	enabled, err = imported.KnowledgeKeeper.ClaimRecordsEnabled(ictx)
	require.NoError(t, err)
	require.True(t, enabled)
	retained, found = imported.KnowledgeKeeper.GetVerificationRound(ictx, round.Id)
	require.True(t, found)
	require.True(t, proto.Equal(round, retained), "pending financial plan survives unchanged")
	require.Empty(t, imported.KnowledgeKeeper.ReadMigrationMarker(ictx, "migration_v10_complete"))
	done, err := imported.UpgradeKeeper.GetDoneHeight(ictx, plan.Name)
	require.NoError(t, err)
	require.Zero(t, done)
}

func TestClaimRecordsRejectsWrongSourceAndPrematureActivation(t *testing.T) {
	for _, scenario := range []string{"enabled", "corrupt flag", "missing neutrality", "missing integrity", "mixed version", "missing version", "extra version", "plan info", "preseeded done marker", "wrong height"} {
		t.Run(scenario, func(t *testing.T) {
			app, ctx, _ := claimRecordsSourceFixture(t)
			vm := claimRecordsSourceVersionMap()
			store := ctx.KVStore(app.keys[knowledgetypes.StoreKey])
			switch scenario {
			case "enabled":
				require.NoError(t, app.KnowledgeKeeper.EnableClaimRecords(ctx))
			case "corrupt flag":
				store.Set([]byte(knowledgekeeper.ClaimRecordsEnabledStoreKey), []byte{0})
			case "missing neutrality":
				store.Delete([]byte(knowledgekeeper.ReviewNeutralityEnabledStoreKey))
			case "missing integrity":
				store.Delete([]byte(knowledgekeeper.RecordIntegrityEnabledStoreKey))
			case "mixed version":
				vm["vesting_rewards"] = 2
			case "missing version":
				delete(vm, "bank")
			case "extra version":
				vm["unexpected"] = 1
			case "preseeded done marker":
				require.NoError(t, app.KnowledgeKeeper.WriteMigrationMarker(ctx, "migration_v10_complete", "true"))
			}
			before := claimRecordsStores(t, app, ctx)
			plan := upgradetypes.Plan{Name: UpgradeNameKnowledgeClaimRecordsV1, Height: app.CommitMultiStore().Commit().Version + 1}
			if scenario == "plan info" {
				plan.Info = "unexpected"
			}
			ctx = ctx.WithBlockHeight(plan.Height).WithHeaderInfo(header.Info{Height: plan.Height, ChainID: ctx.ChainID()})
			if scenario == "wrong height" {
				ctx = ctx.WithBlockHeight(plan.Height + 1)
			}
			_, err := app.RunUpgradeHandlerWithInfoForTests(ctx, plan.Name, vm, plan.Height, plan.Info)
			require.Error(t, err)
			require.Equal(t, before, claimRecordsStores(t, app, ctx))
			done, err := app.UpgradeKeeper.GetDoneHeight(ctx, plan.Name)
			require.NoError(t, err)
			require.Zero(t, done)
		})
	}
}

func TestClaimRecordsNativeSelectionAndRestartReceipt(t *testing.T) {
	for _, scenario := range []string{"missing selection", "false selection", "missing flag", "corrupt flag", "marker only", "done only", "future done"} {
		t.Run(scenario, func(t *testing.T) {
			app, ctx, _ := newFrozenClaimRecordsFixture(t)
			if scenario == "missing selection" || scenario == "false selection" {
				genesis := sdk053IBC10GenesisWithValidator(t, app)
				var knowledge map[string]json.RawMessage
				require.NoError(t, json.Unmarshal(genesis["knowledge"], &knowledge))
				delete(knowledge, "claim_records_enabled")
				if scenario == "false selection" {
					knowledge["claim_records_enabled"] = json.RawMessage("false")
				}
				genesis["knowledge"], _ = json.Marshal(knowledge)
				require.Error(t, validateClaimRecordsGenesisSelection(genesis))
				return
			}
			if scenario == "missing flag" {
				ctx.KVStore(app.keys[knowledgetypes.StoreKey]).Delete([]byte(knowledgekeeper.ClaimRecordsEnabledStoreKey))
			}
			if scenario == "corrupt flag" {
				ctx.KVStore(app.keys[knowledgetypes.StoreKey]).Set([]byte(knowledgekeeper.ClaimRecordsEnabledStoreKey), []byte{2})
			}
			if scenario == "marker only" || scenario == "future done" {
				require.NoError(t, app.KnowledgeKeeper.WriteMigrationMarker(ctx, "migration_v10_complete", "true"))
			}
			if scenario == "done only" {
				writeSDK053IBC10TestDoneHeight(app, ctx, UpgradeNameKnowledgeClaimRecordsV1, 1)
			}
			if scenario == "future done" {
				writeSDK053IBC10TestDoneHeight(app, ctx, UpgradeNameKnowledgeClaimRecordsV1, 2)
			}
			app.CommitMultiStore().Commit()
			require.Error(t, app.ValidateAccountingAuthorityStartup())
		})
	}
}

func TestClaimRecordsModuleRefusesUnownedMigration(t *testing.T) {
	app, ctx, _ := claimRecordsSourceFixture(t)
	before := bytes.Clone(ctx.KVStore(app.keys[knowledgetypes.StoreKey]).Get([]byte(knowledgekeeper.ClaimRecordsEnabledStoreKey)))
	_, err := app.ModuleManager.RunMigrations(ctx, app.configurator, claimRecordsSourceVersionMap())
	require.ErrorContains(t, err, "explicit")
	require.Equal(t, before, ctx.KVStore(app.keys[knowledgetypes.StoreKey]).Get([]byte(knowledgekeeper.ClaimRecordsEnabledStoreKey)))
}
