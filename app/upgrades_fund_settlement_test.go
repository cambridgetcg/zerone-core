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
	"github.com/zerone-chain/zerone/internal/fundsettlementmigration"
	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	"google.golang.org/protobuf/proto"
)

func fundSettlementSourceFixture(t *testing.T) (*ZeroneApp, sdk.Context, dbm.DB) {
	t.Helper()
	app, ctx, db := newAccountingAuthorityFixture(t)
	ctx.KVStore(app.keys[knowledgetypes.StoreKey]).Delete([]byte(knowledgekeeper.FundSettlementEnabledStoreKey))
	require.NoError(t, app.UpgradeKeeper.SetModuleVersionMap(ctx, fundSettlementSourceVersionMap()))
	return app, ctx, db
}

func TestFundSettlementOwnsExactVersionBoundary(t *testing.T) {
	app, _, _ := newAccountingAuthorityFixture(t)
	source, target := fundSettlementSourceVersionMap(), fundSettlementTargetVersionMap()
	require.Equal(t, target, app.CurrentModuleVersionMap())
	require.Equal(t, uint64(10), claimRecordsTargetVersionMap()["knowledge"])
	require.True(t, app.UpgradeKeeper.HasHandler(UpgradeNameKnowledgeFundSettlementV1))
	for _, name := range append(app.KnownUpgradeNames(), "unrelated") {
		for _, guard := range []func(string, module.VersionMap, module.VersionMap) error{requireFundSettlementTransitionOwner, requireClaimRecordsTransitionOwner, requireReviewNeutralityTransitionOwner, requireRecordIntegrityTransitionOwner, requireSurvivalHandoffTransitionOwner} {
			err := guard(name, source, target)
			if name == UpgradeNameKnowledgeFundSettlementV1 {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, "sole owner")
			}
		}
	}
	for _, invalid := range []module.VersionMap{{}, {"knowledge": 12}, {"knowledge": 0}, {"knowledge": 9}} {
		require.Error(t, requireFundSettlementTransitionOwner(UpgradeNameKnowledgeFundSettlementV1, invalid, target))
	}
	require.Error(t, requireFundSettlementTransitionOwner(UpgradeNameKnowledgeFundSettlementV1, target, source))
	_, err := fundsettlementmigration.WithOwner(sdk.Context{}, UpgradeNameKnowledgeReviewNeutralityV1)
	require.Error(t, err)
}

func TestFundSettlementMigrationPreservesRecordsObligationsAndRestart(t *testing.T) {
	app, ctx, db := fundSettlementSourceFixture(t)
	claim := &knowledgetypes.Claim{Id: "old-claim", VerificationRoundId: "old-round", FactContent: "Historical content", Domain: "physics", Stake: "100", ArgumentText: "Already recorded argument", EvidenceIds: []string{"old-evidence"}}
	round := &knowledgetypes.VerificationRound{Id: claim.VerificationRoundId, ClaimId: claim.Id, Phase: knowledgetypes.VerificationPhase_VERIFICATION_PHASE_COMPLETE, Verdict: knowledgetypes.Verdict_VERDICT_REJECT, VerdictBlock: 1, VerifierRewardSettlement: &knowledgetypes.VerifierRewardSettlement{CreatedAtBlock: 1, WithheldTotal: "0", Payments: []*knowledgetypes.VerifierRewardPayment{{Verifier: settlementAddress(212).String(), Amount: "55", Withheld: "0"}}}}
	require.NoError(t, app.KnowledgeKeeper.SetClaim(ctx, claim))
	require.NoError(t, app.KnowledgeKeeper.SetVerificationRound(ctx, round))
	dropped := &knowledgetypes.Claim{Id: "old-contradiction", FactContent: "Recorded counter assertion", Stake: "100", Domain: "physics"}
	require.NoError(t, app.KnowledgeKeeper.SetClaim(ctx, dropped))
	before := claimRecordsStores(t, app, ctx)
	latest := app.CommitMultiStore().Commit().Version
	plan := upgradetypes.Plan{Name: UpgradeNameKnowledgeFundSettlementV1, Height: latest + 2}
	require.NoError(t, app.UpgradeKeeper.ScheduleUpgrade(ctx, plan))
	require.NoError(t, app.UpgradeKeeper.DumpUpgradeInfoToDisk(plan.Height, plan))
	require.Error(t, app.ValidateAccountingAuthorityStartup(), "new binary cannot execute an early v10 block")
	require.Equal(t, plan.Height-1, app.CommitMultiStore().Commit().Version)
	require.NoError(t, app.ValidateAccountingAuthorityStartup())
	ctx = ctx.WithBlockHeight(plan.Height).WithHeaderInfo(header.Info{Height: plan.Height, ChainID: ctx.ChainID()})
	cache, write := ctx.CacheContext()
	require.NoError(t, app.UpgradeKeeper.ApplyUpgrade(cache, plan))
	enabled, err := app.KnowledgeKeeper.FundSettlementEnabled(ctx)
	require.NoError(t, err)
	require.False(t, enabled, "outer block cache owns activation")
	require.Equal(t, before, claimRecordsStores(t, app, ctx))
	write()
	after := claimRecordsStores(t, app, ctx)
	delete(after, knowledgetypes.StoreKey+":"+knowledgekeeper.FundSettlementEnabledStoreKey)
	delete(after, knowledgetypes.StoreKey+":"+string(append([]byte{0x7f, 0x01}, []byte("migration_v11_complete")...)))
	require.Equal(t, before, after, "only activation and its completion marker change knowledge/bank/auth state")
	require.Equal(t, plan.Height, app.CommitMultiStore().Commit().Version)
	require.NoError(t, app.ValidateAccountingAuthorityStartup())
	require.Error(t, app.UpgradeKeeper.ApplyUpgrade(ctx, plan))
	restarted := NewZeroneApp(log.NewNopLogger(), db, nil, true, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()), baseapp.SetChainID(ctx.ChainID()))
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
	imported := NewZeroneApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()), baseapp.SetChainID("fund-settlement-import"))
	ictx := imported.NewUncachedContext(false, cmtproto.Header{ChainID: "fund-settlement-import"})
	_, err = imported.InitChainer(ictx, &abci.RequestInitChain{ChainId: "fund-settlement-import", AppStateBytes: raw})
	require.NoError(t, err)
	enabled, err = imported.KnowledgeKeeper.FundSettlementEnabled(ictx)
	require.NoError(t, err)
	require.True(t, enabled)
	retained, found = imported.KnowledgeKeeper.GetVerificationRound(ictx, round.Id)
	require.True(t, found)
	require.True(t, proto.Equal(round, retained), "pending financial plan survives unchanged")
	require.Empty(t, imported.KnowledgeKeeper.ReadMigrationMarker(ictx, "migration_v11_complete"))
	done, err := imported.UpgradeKeeper.GetDoneHeight(ictx, plan.Name)
	require.NoError(t, err)
	require.Zero(t, done)
}

func TestFundSettlementRejectsWrongSourceAndPrematureActivation(t *testing.T) {
	for _, scenario := range []string{"enabled", "corrupt flag", "missing claim records", "missing neutrality", "missing integrity", "raw claim funding", "raw round refund", "mixed version", "missing version", "extra version", "plan info", "preseeded done marker", "wrong height"} {
		t.Run(scenario, func(t *testing.T) {
			app, ctx, _ := fundSettlementSourceFixture(t)
			vm := fundSettlementSourceVersionMap()
			store := ctx.KVStore(app.keys[knowledgetypes.StoreKey])
			switch scenario {
			case "enabled":
				require.NoError(t, app.KnowledgeKeeper.EnableFundSettlement(ctx))
			case "corrupt flag":
				store.Set([]byte(knowledgekeeper.FundSettlementEnabledStoreKey), []byte{0})
			case "missing claim records":
				store.Delete([]byte(knowledgekeeper.ClaimRecordsEnabledStoreKey))
			case "raw claim funding":
				claim := &knowledgetypes.Claim{Id: "preseeded-terms", Stake: "100", Domain: "physics"}
				raw, err := proto.Marshal(claim)
				require.NoError(t, err)
				// Field29 is unknown to the exact v10 source, including explicit empty.
				store.Set(knowledgetypes.ClaimKey(claim.Id), append(raw, 0xea, 0x01, 0x00))
			case "raw round refund":
				claim := &knowledgetypes.Claim{Id: "preseeded-refund-claim", Stake: "100", Domain: "physics"}
				require.NoError(t, app.KnowledgeKeeper.SetClaim(ctx, claim))
				round := &knowledgetypes.VerificationRound{Id: "preseeded-refund", ClaimId: claim.Id, Phase: knowledgetypes.VerificationPhase_VERIFICATION_PHASE_COMPLETE, Verdict: knowledgetypes.Verdict_VERDICT_REJECT, VerdictBlock: 1}
				raw, err := proto.Marshal(round)
				require.NoError(t, err)
				store.Set(knowledgetypes.RoundKey(round.Id), append(raw, 0x8a, 0x01, 0x00))
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
				require.NoError(t, app.KnowledgeKeeper.WriteMigrationMarker(ctx, "migration_v11_complete", "true"))
			}
			before := claimRecordsStores(t, app, ctx)
			plan := upgradetypes.Plan{Name: UpgradeNameKnowledgeFundSettlementV1, Height: app.CommitMultiStore().Commit().Version + 1}
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

func TestFundSettlementNativeSelectionAndRestartReceipt(t *testing.T) {
	for _, scenario := range []string{"missing selection", "false selection", "missing flag", "corrupt flag", "marker only", "done only", "future done"} {
		t.Run(scenario, func(t *testing.T) {
			app, ctx, _ := newAccountingAuthorityFixture(t)
			if scenario == "missing selection" || scenario == "false selection" {
				genesis := sdk053IBC10GenesisWithValidator(t, app)
				var knowledge map[string]json.RawMessage
				require.NoError(t, json.Unmarshal(genesis["knowledge"], &knowledge))
				delete(knowledge, "fund_settlement_enabled")
				if scenario == "false selection" {
					knowledge["fund_settlement_enabled"] = json.RawMessage("false")
				}
				genesis["knowledge"], _ = json.Marshal(knowledge)
				require.Error(t, validateFundSettlementGenesisSelection(genesis))
				return
			}
			if scenario == "missing flag" {
				ctx.KVStore(app.keys[knowledgetypes.StoreKey]).Delete([]byte(knowledgekeeper.FundSettlementEnabledStoreKey))
			}
			if scenario == "corrupt flag" {
				ctx.KVStore(app.keys[knowledgetypes.StoreKey]).Set([]byte(knowledgekeeper.FundSettlementEnabledStoreKey), []byte{2})
			}
			if scenario == "marker only" || scenario == "future done" {
				require.NoError(t, app.KnowledgeKeeper.WriteMigrationMarker(ctx, "migration_v11_complete", "true"))
			}
			if scenario == "done only" {
				writeSDK053IBC10TestDoneHeight(app, ctx, UpgradeNameKnowledgeFundSettlementV1, 1)
			}
			if scenario == "future done" {
				writeSDK053IBC10TestDoneHeight(app, ctx, UpgradeNameKnowledgeFundSettlementV1, 2)
			}
			app.CommitMultiStore().Commit()
			require.Error(t, app.ValidateAccountingAuthorityStartup())
		})
	}
}

func TestFundSettlementModuleRefusesUnownedMigration(t *testing.T) {
	app, ctx, _ := fundSettlementSourceFixture(t)
	before := bytes.Clone(ctx.KVStore(app.keys[knowledgetypes.StoreKey]).Get([]byte(knowledgekeeper.FundSettlementEnabledStoreKey)))
	_, err := app.ModuleManager.RunMigrations(ctx, app.configurator, fundSettlementSourceVersionMap())
	require.ErrorContains(t, err, "explicit")
	require.Equal(t, before, ctx.KVStore(app.keys[knowledgetypes.StoreKey]).Get([]byte(knowledgekeeper.FundSettlementEnabledStoreKey)))
}
