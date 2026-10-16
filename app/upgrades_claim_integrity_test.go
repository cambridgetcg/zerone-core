package app

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"sort"
	"testing"
	"time"

	"cosmossdk.io/core/header"
	"cosmossdk.io/log"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/stretchr/testify/require"
	"github.com/zerone-chain/zerone/internal/claimintegritymigration"
	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	"google.golang.org/protobuf/proto"
)

// knowledge-claim-integrity-v1 (plan §3.3-§3.4, §6 Task 6). The handler owns
// exactly knowledge 11->12, writes only marker 0x87 and its receipt, and the
// v12 binary admits v11 state only at the exact scheduled H-1.

const claimIntegrityReceiptName = "migration_v12_complete"

func claimIntegrityReceiptRow() string {
	return knowledgetypes.StoreKey + ":" + string(append([]byte{0x7f, 0x01}, claimIntegrityReceiptName...))
}

func claimIntegrityMarkerRow() string {
	return knowledgetypes.StoreKey + ":" + knowledgekeeper.ClaimIntegrityEnabledStoreKey
}

// claimIntegrityAllStores images every mounted KV store, in sorted store order.
func claimIntegrityAllStores(t *testing.T, app *ZeroneApp, ctx sdk.Context) map[string][]byte {
	t.Helper()
	names := make([]string, 0, len(app.keys))
	for name := range app.keys {
		names = append(names, name)
	}
	sort.Strings(names)
	rows := make(map[string][]byte)
	for _, name := range names {
		it := ctx.KVStore(app.keys[name]).Iterator(nil, nil)
		for ; it.Valid(); it.Next() {
			rows[name+":"+string(it.Key())] = bytes.Clone(it.Value())
		}
		require.NoError(t, it.Close())
	}
	return rows
}

// newClaimIntegrityFixture is newAccountingAuthorityFixture with explicit
// unsafe-skip heights, so the handler's own skip refusal can be exercised.
func newClaimIntegrityFixture(t *testing.T, skip []int) (*ZeroneApp, sdk.Context, dbm.DB) {
	t.Helper()
	db := dbm.NewMemDB()
	app := NewZeroneApp(log.NewNopLogger(), db, nil, true, simtestutil.AppOptionsMap{flags.FlagHome: t.TempDir(), server.FlagUnsafeSkipUpgrades: skip}, baseapp.SetChainID("accounting-fixture"))
	genesis, err := json.Marshal(sdk053IBC10GenesisWithValidator(t, app))
	require.NoError(t, err)
	ctx := app.NewUncachedContext(false, cmtproto.Header{ChainID: "accounting-fixture"})
	_, err = app.InitChainer(ctx, &abci.RequestInitChain{ChainId: "accounting-fixture", AppStateBytes: genesis})
	require.NoError(t, err)
	return app, ctx, db
}

// claimIntegritySourceFixture models a completed knowledge-11 store: every
// earlier selection on, no 0x87, and the exact fund-settlement target map.
func claimIntegritySourceFixture(t *testing.T, skip ...int) (*ZeroneApp, sdk.Context, dbm.DB) {
	t.Helper()
	app, ctx, db := newClaimIntegrityFixture(t, skip)
	ctx.KVStore(app.keys[knowledgetypes.StoreKey]).Delete([]byte(knowledgekeeper.ClaimIntegrityEnabledStoreKey))
	require.NoError(t, app.UpgradeKeeper.SetModuleVersionMap(ctx, claimIntegritySourceVersionMap()))
	return app, ctx, db
}

func TestClaimIntegrityOwnsExactVersionBoundary(t *testing.T) {
	app, _, _ := newAccountingAuthorityFixture(t)
	source, target := claimIntegritySourceVersionMap(), claimIntegrityTargetVersionMap()
	require.Equal(t, target, app.CurrentModuleVersionMap())
	require.Equal(t, fundSettlementTargetVersionMap(), source)
	require.Equal(t, uint64(11), source["knowledge"])
	require.Equal(t, uint64(12), target["knowledge"])
	require.Len(t, target, len(source))
	for name, version := range source {
		if name != "knowledge" {
			require.Equal(t, version, target[name], "only knowledge moves at 11->12: %s", name)
		}
	}
	require.True(t, app.UpgradeKeeper.HasHandler(UpgradeNameKnowledgeClaimIntegrityV1))
	require.Equal(t, "knowledge-claim-integrity-v1", UpgradeNameKnowledgeClaimIntegrityV1)
	var listed bool
	for _, entry := range app.BuildChainVersionReport().KnownUpgrades {
		if entry.UpgradeName == UpgradeNameKnowledgeClaimIntegrityV1 {
			listed = true
			require.Contains(t, entry.Description, "knowledge 11->12")
		}
	}
	require.True(t, listed, "BuildChainVersionReport lists the claim-integrity upgrade")
	guards := []func(string, module.VersionMap, module.VersionMap) error{requireClaimIntegrityTransitionOwner, requireFundSettlementTransitionOwner, requireClaimRecordsTransitionOwner, requireReviewNeutralityTransitionOwner, requireRecordIntegrityTransitionOwner, requireSurvivalHandoffTransitionOwner}
	for _, name := range append(app.KnownUpgradeNames(), "unrelated") {
		for _, guard := range guards {
			err := guard(name, source, target)
			if name == UpgradeNameKnowledgeClaimIntegrityV1 {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, "sole owner", "every older plan name is refused for 11->12: %s", name)
			}
		}
	}
	// A jump over the v11 boundary has no owner at all.
	for _, name := range append(app.KnownUpgradeNames(), "unrelated") {
		require.Error(t, requireSurvivalHandoffTransitionOwner(name, fundSettlementSourceVersionMap(), target), name)
	}
	// The earlier boundary keeps its own sole owner.
	require.NoError(t, requireSurvivalHandoffTransitionOwner(UpgradeNameKnowledgeFundSettlementV1, fundSettlementSourceVersionMap(), fundSettlementTargetVersionMap()))
	require.ErrorContains(t, requireSurvivalHandoffTransitionOwner(UpgradeNameKnowledgeClaimIntegrityV1, fundSettlementSourceVersionMap(), fundSettlementTargetVersionMap()), "sole owner")
	for _, invalid := range []module.VersionMap{{}, {"knowledge": 13}, {"knowledge": 0}, {"knowledge": 10}} {
		require.Error(t, requireClaimIntegrityTransitionOwner(UpgradeNameKnowledgeClaimIntegrityV1, invalid, target))
	}
	require.Error(t, requireClaimIntegrityTransitionOwner(UpgradeNameKnowledgeClaimIntegrityV1, target, source))
	_, err := claimintegritymigration.WithOwner(sdk.Context{}, UpgradeNameKnowledgeFundSettlementV1)
	require.Error(t, err)
}

func TestClaimIntegrityMigrationPreservesEveryByteAndRestart(t *testing.T) {
	app, ctx, db := claimIntegritySourceFixture(t)
	claim := &knowledgetypes.Claim{Id: "old-claim", VerificationRoundId: "old-round", FactContent: "Historical content", Domain: "physics", Stake: "100", ArgumentText: "Already recorded argument", EvidenceIds: []string{"old-evidence"}}
	round := &knowledgetypes.VerificationRound{Id: claim.VerificationRoundId, ClaimId: claim.Id, Phase: knowledgetypes.VerificationPhase_VERIFICATION_PHASE_COMPLETE, Verdict: knowledgetypes.Verdict_VERDICT_REJECT, VerdictBlock: 1, VerifierRewardSettlement: &knowledgetypes.VerifierRewardSettlement{CreatedAtBlock: 1, WithheldTotal: "0", Payments: []*knowledgetypes.VerifierRewardPayment{{Verifier: settlementAddress(213).String(), Amount: "55", Withheld: "0"}}}}
	require.NoError(t, app.KnowledgeKeeper.SetClaim(ctx, claim))
	require.NoError(t, app.KnowledgeKeeper.SetVerificationRound(ctx, round))
	fact := &knowledgetypes.Fact{Id: "old-contested", Domain: "physics", Category: "empirical", Status: knowledgetypes.FactStatus_FACT_STATUS_CONTESTED, Content: "A legacy fact contested at admission", Submitter: settlementAddress(214).String()}
	require.NoError(t, app.KnowledgeKeeper.SetFact(ctx, fact))
	latest := app.CommitMultiStore().Commit().Version
	plan := upgradetypes.Plan{Name: UpgradeNameKnowledgeClaimIntegrityV1, Height: latest + 2}
	require.NoError(t, app.UpgradeKeeper.ScheduleUpgrade(ctx, plan))
	require.NoError(t, app.UpgradeKeeper.DumpUpgradeInfoToDisk(plan.Height, plan))
	require.Error(t, app.ValidateAccountingAuthorityStartup(), "the v12 binary cannot execute an early v11 block")
	require.Equal(t, plan.Height-1, app.CommitMultiStore().Commit().Version)
	require.NoError(t, app.ValidateAccountingAuthorityStartup(), "the v12 binary admits v11 state at the exact scheduled H-1")
	before := claimIntegrityAllStores(t, app, ctx)
	require.NotContains(t, before, claimIntegrityMarkerRow())
	ctx = ctx.WithBlockHeight(plan.Height).WithHeaderInfo(header.Info{Height: plan.Height, ChainID: ctx.ChainID()})
	cache, write := ctx.CacheContext()
	require.NoError(t, app.UpgradeKeeper.ApplyUpgrade(cache, plan))
	enabled, err := app.KnowledgeKeeper.ClaimIntegrityEnabled(ctx)
	require.NoError(t, err)
	require.False(t, enabled, "outer block cache owns activation")
	require.Equal(t, before, claimIntegrityAllStores(t, app, ctx))
	write()
	after := claimIntegrityAllStores(t, app, ctx)
	require.Equal(t, []byte{1}, after[claimIntegrityMarkerRow()])
	require.Equal(t, []byte("true"), after[claimIntegrityReceiptRow()])
	delete(after, claimIntegrityMarkerRow())
	delete(after, claimIntegrityReceiptRow())
	// x/upgrade's own bookkeeping for one applied plan: the plan is cleared,
	// its done height and the knowledge version are recorded, and the
	// protocol version advances.
	upgradeRow := func(key []byte) string { return upgradetypes.StoreKey + ":" + string(key) }
	doneKey := make([]byte, 9+len(plan.Name))
	doneKey[0] = upgradetypes.DoneByte
	binary.BigEndian.PutUint64(doneKey[1:9], uint64(plan.Height))
	copy(doneKey[9:], plan.Name)
	knowledgeVersion := upgradeRow(append([]byte{upgradetypes.VersionMapByte}, "knowledge"...))
	require.Contains(t, before, upgradeRow(upgradetypes.PlanKey()))
	require.NotContains(t, after, upgradeRow(upgradetypes.PlanKey()))
	require.Equal(t, []byte{1}, after[upgradeRow(doneKey)])
	require.Equal(t, binary.BigEndian.AppendUint64(nil, 11), before[knowledgeVersion])
	require.Equal(t, binary.BigEndian.AppendUint64(nil, 12), after[knowledgeVersion])
	for _, rows := range []map[string][]byte{before, after} {
		delete(rows, upgradeRow(upgradetypes.PlanKey()))
		delete(rows, upgradeRow(doneKey))
		delete(rows, knowledgeVersion)
		delete(rows, upgradeRow([]byte{upgradetypes.ProtocolVersionByte}))
	}
	require.Equal(t, before, after, "only 0x87 and its receipt change any store beyond x/upgrade bookkeeping")
	vm, err := app.UpgradeKeeper.GetModuleVersionMap(ctx)
	require.NoError(t, err)
	require.Equal(t, claimIntegrityTargetVersionMap(), module.VersionMap(vm))
	require.Equal(t, plan.Height, app.CommitMultiStore().Commit().Version)
	require.NoError(t, app.ValidateAccountingAuthorityStartup(), "restart at the applied target is accepted")
	require.Error(t, app.UpgradeKeeper.ApplyUpgrade(ctx, plan))
	restarted := NewZeroneApp(log.NewNopLogger(), db, nil, true, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()), baseapp.SetChainID(ctx.ChainID()))
	require.NoError(t, restarted.ValidateAccountingAuthorityStartup())
	rctx := restarted.NewUncachedContext(false, cmtproto.Header{Height: plan.Height, ChainID: ctx.ChainID()})
	enabled, err = restarted.KnowledgeKeeper.ClaimIntegrityEnabled(rctx)
	require.NoError(t, err)
	require.True(t, enabled)
	retained, found := restarted.KnowledgeKeeper.GetVerificationRound(rctx, round.Id)
	require.True(t, found)
	require.True(t, proto.Equal(round, retained))
	legacy, found := restarted.KnowledgeKeeper.GetFact(rctx, fact.Id)
	require.True(t, found)
	require.Equal(t, knowledgetypes.FactStatus_FACT_STATUS_CONTESTED, legacy.Status, "activation rewrites no record")
	genesis, err := restarted.ModuleManager.ExportGenesis(rctx, restarted.appCodec)
	require.NoError(t, err)
	lineage, err := restarted.accountingAuthorityGenesisMetadata(rctx, plan.Height)
	require.NoError(t, err)
	genesis[accountingAuthorityGenesisKey], err = json.Marshal(lineage)
	require.NoError(t, err)
	var selection struct {
		Enabled *bool `json:"claim_integrity_enabled"`
	}
	require.NoError(t, json.Unmarshal(genesis["knowledge"], &selection))
	require.NotNil(t, selection.Enabled)
	require.True(t, *selection.Enabled, "export carries the explicit selection")
	raw, err := json.Marshal(genesis)
	require.NoError(t, err)
	imported := NewZeroneApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()), baseapp.SetChainID("claim-integrity-import"))
	ictx := imported.NewUncachedContext(false, cmtproto.Header{ChainID: "claim-integrity-import"})
	_, err = imported.InitChainer(ictx, &abci.RequestInitChain{ChainId: "claim-integrity-import", AppStateBytes: raw})
	require.NoError(t, err)
	enabled, err = imported.KnowledgeKeeper.ClaimIntegrityEnabled(ictx)
	require.NoError(t, err)
	require.True(t, enabled)
	require.Empty(t, imported.KnowledgeKeeper.ReadMigrationMarker(ictx, claimIntegrityReceiptName), "import invents no upgrade receipt")
	done, err := imported.UpgradeKeeper.GetDoneHeight(ictx, plan.Name)
	require.NoError(t, err)
	require.Zero(t, done)
	imported.CommitMultiStore().Commit()
	require.NoError(t, imported.ValidateAccountingAuthorityStartup(), "imported target restarts as native genesis")
}

func TestClaimIntegrityRejectsWrongSourceAndPrematureActivation(t *testing.T) {
	for _, tc := range []struct{ scenario, want string }{
		{"enabled", "claim integrity predecessor already enabled or unreadable"},
		{"corrupt flag", "malformed claim integrity marker"},
		{"missing fund settlement", "fund settlement target requires explicit enabled marker"},
		{"fund settlement receipt only", "fund settlement target has inconsistent migration marker and done height"},
		{"fund settlement future done", "fund settlement target has inconsistent migration marker and done height"},
		{"missing claim records", "claim records target requires explicit enabled marker"},
		{"preseeded receipt", "claim integrity predecessor already marked or unreadable"},
		{"preseeded done height", "claim integrity predecessor already applied or unreadable"},
		{"mixed version", "claim integrity requires exact completed fund-settlement source and compiled target version maps"},
		{"missing version", "claim integrity requires exact completed fund-settlement source and compiled target version maps"},
		{"extra version", "claim integrity requires exact completed fund-settlement source and compiled target version maps"},
		{"plan info", "claim integrity requires its named height plan with empty info and no deprecated fields"},
		{"plan time", "claim integrity requires its named height plan with empty info and no deprecated fields"},
		{"wrong height", "claim integrity requires exact committed H-1/H boundary"},
		{"premature execution", "claim integrity requires exact committed H-1/H boundary"},
		{"skip height", "claim integrity cannot be unsafe-skipped"},
		{"older plan name fund settlement", "fund settlement requires exact completed claim-records source and compiled target version maps"},
		{"older plan name claim records", "claim records requires exact completed review-neutrality source and compiled target version maps"},
	} {
		scenario, want := tc.scenario, tc.want
		t.Run(scenario, func(t *testing.T) {
			var skip []int
			if scenario == "skip height" {
				skip = []int{2} // InitChain at 0, one commit, plan at latest+1.
			}
			app, ctx, _ := claimIntegritySourceFixture(t, skip...)
			store := ctx.KVStore(app.keys[knowledgetypes.StoreKey])
			switch scenario {
			case "enabled":
				require.NoError(t, app.KnowledgeKeeper.EnableClaimIntegrity(ctx))
			case "corrupt flag":
				store.Set([]byte(knowledgekeeper.ClaimIntegrityEnabledStoreKey), []byte{0})
			case "missing fund settlement":
				store.Delete([]byte(knowledgekeeper.FundSettlementEnabledStoreKey))
			case "fund settlement receipt only":
				require.NoError(t, app.KnowledgeKeeper.WriteMigrationMarker(ctx, "migration_v11_complete", "true"))
			case "fund settlement future done":
				require.NoError(t, app.KnowledgeKeeper.WriteMigrationMarker(ctx, "migration_v11_complete", "true"))
				writeSDK053IBC10TestDoneHeight(app, ctx, UpgradeNameKnowledgeFundSettlementV1, 99)
			case "missing claim records":
				store.Delete([]byte(knowledgekeeper.ClaimRecordsEnabledStoreKey))
			case "preseeded receipt":
				require.NoError(t, app.KnowledgeKeeper.WriteMigrationMarker(ctx, claimIntegrityReceiptName, "true"))
			case "preseeded done height":
				writeSDK053IBC10TestDoneHeight(app, ctx, UpgradeNameKnowledgeClaimIntegrityV1, 1)
			case "mixed version":
				vm := claimIntegritySourceVersionMap()
				vm["vesting_rewards"] = 2
				require.NoError(t, app.UpgradeKeeper.SetModuleVersionMap(ctx, vm))
			case "missing version":
				ctx.KVStore(app.keys[upgradetypes.StoreKey]).Delete(append([]byte{upgradetypes.VersionMapByte}, "bank"...))
			case "extra version":
				vm := claimIntegritySourceVersionMap()
				vm["unexpected"] = 1
				require.NoError(t, app.UpgradeKeeper.SetModuleVersionMap(ctx, vm))
			}
			latest := app.CommitMultiStore().Commit().Version
			plan := upgradetypes.Plan{Name: UpgradeNameKnowledgeClaimIntegrityV1, Height: latest + 1}
			switch scenario {
			case "plan info":
				plan.Info = "unexpected"
			case "plan time":
				plan.Time = time.Unix(1, 0).UTC()
			case "premature execution":
				plan.Height = latest + 2
			case "older plan name fund settlement":
				plan.Name = UpgradeNameKnowledgeFundSettlementV1
			case "older plan name claim records":
				plan.Name = UpgradeNameKnowledgeClaimRecordsV1
			}
			if scenario == "skip height" {
				require.True(t, app.UpgradeKeeper.IsSkipHeight(plan.Height))
			}
			ctx = ctx.WithBlockHeight(plan.Height).WithHeaderInfo(header.Info{Height: plan.Height, ChainID: ctx.ChainID()})
			if scenario == "wrong height" {
				ctx = ctx.WithBlockHeight(plan.Height + 1)
			}
			before := claimIntegrityAllStores(t, app, ctx)
			err := app.UpgradeKeeper.ApplyUpgrade(ctx, plan)
			require.ErrorContains(t, err, want)
			require.Equal(t, before, claimIntegrityAllStores(t, app, ctx), "a refused activation writes nothing")
			done, err := app.UpgradeKeeper.GetDoneHeight(ctx, UpgradeNameKnowledgeClaimIntegrityV1)
			require.NoError(t, err)
			if scenario != "preseeded done height" {
				require.Zero(t, done)
			}
		})
	}
}

func TestClaimIntegrityNativeSelectionAndRestartReceipt(t *testing.T) {
	for _, scenario := range []string{"missing selection", "false selection"} {
		t.Run(scenario, func(t *testing.T) {
			app := NewZeroneApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()), baseapp.SetChainID("claim-integrity-native"))
			genesis := sdk053IBC10GenesisWithValidator(t, app)
			var knowledge map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(genesis["knowledge"], &knowledge))
			require.JSONEq(t, "true", string(knowledge["claim_integrity_enabled"]), "DefaultGenesis selects v12 explicitly")
			delete(knowledge, "claim_integrity_enabled")
			if scenario == "false selection" {
				knowledge["claim_integrity_enabled"] = json.RawMessage("false")
			}
			var err error
			genesis["knowledge"], err = json.Marshal(knowledge)
			require.NoError(t, err)
			require.ErrorContains(t, validateClaimIntegrityGenesisSelection(genesis), "claim_integrity_enabled=true")
			raw, err := json.Marshal(genesis)
			require.NoError(t, err)
			ctx := app.NewUncachedContext(false, cmtproto.Header{ChainID: "claim-integrity-native"})
			_, err = app.InitChainer(ctx, &abci.RequestInitChain{ChainId: "claim-integrity-native", AppStateBytes: raw})
			require.ErrorContains(t, err, "claim_integrity_enabled=true", "native genesis without the flag is refused")
			require.Nil(t, ctx.KVStore(app.keys[knowledgetypes.StoreKey]).Get([]byte(knowledgekeeper.ClaimIntegrityEnabledStoreKey)))
		})
	}
	for _, tc := range []struct{ scenario, want string }{
		{"native", ""},
		{"missing flag", "claim integrity target requires explicit enabled marker"},
		{"corrupt flag", "malformed claim integrity marker"},
		{"marker only", "claim integrity target has inconsistent migration marker and done height"},
		{"done only", "claim integrity target has inconsistent migration marker and done height"},
		{"future done", "claim integrity target has inconsistent migration marker and done height"},
		{"fund settlement receipt only", "fund settlement target has inconsistent migration marker and done height"},
	} {
		scenario, want := tc.scenario, tc.want
		t.Run(scenario, func(t *testing.T) {
			app, ctx, db := newAccountingAuthorityFixture(t)
			store := ctx.KVStore(app.keys[knowledgetypes.StoreKey])
			switch scenario {
			case "missing flag":
				store.Delete([]byte(knowledgekeeper.ClaimIntegrityEnabledStoreKey))
			case "corrupt flag":
				store.Set([]byte(knowledgekeeper.ClaimIntegrityEnabledStoreKey), []byte{2})
			case "marker only":
				require.NoError(t, app.KnowledgeKeeper.WriteMigrationMarker(ctx, claimIntegrityReceiptName, "true"))
			case "done only":
				writeSDK053IBC10TestDoneHeight(app, ctx, UpgradeNameKnowledgeClaimIntegrityV1, 1)
			case "future done":
				require.NoError(t, app.KnowledgeKeeper.WriteMigrationMarker(ctx, claimIntegrityReceiptName, "true"))
				writeSDK053IBC10TestDoneHeight(app, ctx, UpgradeNameKnowledgeClaimIntegrityV1, 2)
			case "fund settlement receipt only":
				require.NoError(t, app.KnowledgeKeeper.WriteMigrationMarker(ctx, "migration_v11_complete", "true"))
			}
			app.CommitMultiStore().Commit()
			if scenario != "native" {
				require.ErrorContains(t, app.ValidateAccountingAuthorityStartup(), want)
				return
			}
			require.NoError(t, app.ValidateAccountingAuthorityStartup(), "native v12 genesis restarts at the target")
			restarted := NewZeroneApp(log.NewNopLogger(), db, nil, true, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()))
			require.NoError(t, restarted.ValidateAccountingAuthorityStartup())
		})
	}
}

func TestClaimIntegrityModuleRefusesUnownedMigration(t *testing.T) {
	app, ctx, _ := claimIntegritySourceFixture(t)
	before := claimIntegrityAllStores(t, app, ctx)
	_, err := app.ModuleManager.RunMigrations(ctx, app.configurator, claimIntegritySourceVersionMap())
	require.ErrorContains(t, err, "explicit")
	require.ErrorContains(t, err, UpgradeNameKnowledgeClaimIntegrityV1)
	require.Equal(t, before, claimIntegrityAllStores(t, app, ctx))
}
