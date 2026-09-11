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
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/internal/reviewmigration"
	knowledgemodule "github.com/zerone-chain/zerone/x/knowledge"
	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

type archivedReviewKnowledgeModule struct{ knowledgemodule.AppModule }

func (archivedReviewKnowledgeModule) ConsensusVersion() uint64 { return 9 }
func freezeReviewCompiledTarget(app *ZeroneApp) {
	app.ModuleManager.Modules["knowledge"] = archivedReviewKnowledgeModule{app.ModuleManager.Modules["knowledge"].(knowledgemodule.AppModule)}
}
func newFrozenReviewFixture(t *testing.T) (*ZeroneApp, sdk.Context, dbm.DB) {
	t.Helper()
	app, ctx, db := newAccountingAuthorityFixture(t)
	freezeReviewCompiledTarget(app)
	ctx.KVStore(app.keys["knowledge"]).Delete([]byte(knowledgekeeper.ClaimRecordsEnabledStoreKey))
	require.NoError(t, app.UpgradeKeeper.SetModuleVersionMap(ctx, reviewNeutralityTargetVersionMap()))
	return app, ctx, db
}

func reviewNeutralitySourceFixture(t *testing.T) (*ZeroneApp, sdk.Context, dbm.DB) {
	t.Helper()
	app, ctx, db := newFrozenReviewFixture(t)
	ctx.KVStore(app.keys["knowledge"]).Delete([]byte(knowledgekeeper.ReviewNeutralityEnabledStoreKey))
	require.NoError(t, app.UpgradeKeeper.SetModuleVersionMap(ctx, reviewNeutralitySourceVersionMap()))
	return app, ctx, db
}

func TestReviewNeutralityOwnsExactVersionBoundary(t *testing.T) {
	app, _, _ := newFrozenReviewFixture(t)
	source, target := reviewNeutralitySourceVersionMap(), reviewNeutralityTargetVersionMap()
	require.Equal(t, target, app.CurrentModuleVersionMap())
	require.Equal(t, uint64(7), survivalHandoffTargetVersionMap()["knowledge"])
	require.Equal(t, uint64(6), accountingAuthorityTargetVersionMap()["knowledge"])
	require.True(t, app.UpgradeKeeper.HasHandler(UpgradeNameKnowledgeReviewNeutralityV1))
	for _, name := range append(app.KnownUpgradeNames(), "unrelated") {
		err := requireReviewNeutralityTransitionOwner(name, source, target)
		if name == UpgradeNameKnowledgeReviewNeutralityV1 {
			require.NoError(t, err)
		} else {
			require.ErrorContains(t, err, "sole owner")
		}
	}
	for _, invalid := range []module.VersionMap{{}, {"knowledge": 10}, {"knowledge": 0}} {
		require.Error(t, requireReviewNeutralityTransitionOwner(UpgradeNameKnowledgeReviewNeutralityV1, invalid, target))
	}
	require.Error(t, requireReviewNeutralityTransitionOwner(UpgradeNameKnowledgeReviewNeutralityV1, target, source))
	_, err := reviewmigration.WithOwner(sdk.Context{}, UpgradeNameSurvivalRewardHandoffV1)
	require.Error(t, err)
}

func TestReviewNeutralityMigrationPreservesLegacyRoundsAndRestarts(t *testing.T) {
	app, ctx, db := reviewNeutralitySourceFixture(t)
	old := &knowledgetypes.VerificationRound{Id: "legacy-review", ClaimId: "legacy-claim", Phase: knowledgetypes.VerificationPhase_VERIFICATION_PHASE_COMMIT,
		CommitDeadline: 100, RevealDeadline: 200, AggregationDeadline: 220}
	claim := &knowledgetypes.Claim{Id: old.ClaimId, VerificationRoundId: old.Id, FactContent: "Preserved legacy claim", Domain: "general", Stake: "100000", Status: knowledgetypes.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION}
	require.NoError(t, app.KnowledgeKeeper.SetClaim(ctx, claim))
	require.NoError(t, app.KnowledgeKeeper.SetVerificationRound(ctx, old))
	key := knowledgetypes.RoundKey(old.Id)
	before := bytes.Clone(ctx.KVStore(app.keys["knowledge"]).Get(key))
	supply := app.BankKeeper.GetSupply(ctx, "uzrn")
	latest := app.CommitMultiStore().Commit().Version
	plan := upgradetypes.Plan{Name: UpgradeNameKnowledgeReviewNeutralityV1, Height: latest + 2}
	require.NoError(t, app.UpgradeKeeper.ScheduleUpgrade(ctx, plan))
	require.NoError(t, app.UpgradeKeeper.DumpUpgradeInfoToDisk(plan.Height, plan))
	require.Error(t, app.ValidateAccountingAuthorityStartup(), "candidate must not execute an early legacy block")
	require.Equal(t, plan.Height-1, app.CommitMultiStore().Commit().Version)
	require.NoError(t, app.ValidateAccountingAuthorityStartup())
	ctx = ctx.WithBlockHeight(plan.Height).WithHeaderInfo(header.Info{Height: plan.Height, ChainID: ctx.ChainID()})
	cache, write := ctx.CacheContext()
	require.NoError(t, app.UpgradeKeeper.ApplyUpgrade(cache, plan))
	enabled, err := app.KnowledgeKeeper.ReviewNeutralityEnabled(ctx)
	require.NoError(t, err)
	require.False(t, enabled, "outer block cache owns the whole migration")
	write()
	require.Equal(t, before, ctx.KVStore(app.keys["knowledge"]).Get(key))
	require.Equal(t, supply, app.BankKeeper.GetSupply(ctx, "uzrn"))
	require.Equal(t, plan.Height, app.CommitMultiStore().Commit().Version)
	require.NoError(t, app.ValidateAccountingAuthorityStartup())
	require.Error(t, app.UpgradeKeeper.ApplyUpgrade(ctx, plan))

	restarted := NewZeroneApp(log.NewNopLogger(), db, nil, false, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()), baseapp.SetChainID(ctx.ChainID()))
	freezeReviewCompiledTarget(restarted)
	require.NoError(t, restarted.LoadLatestVersion())
	require.NoError(t, restarted.ValidateAccountingAuthorityStartup())
	rctx := restarted.NewUncachedContext(false, cmtproto.Header{Height: plan.Height, ChainID: ctx.ChainID()})
	require.Equal(t, before, rctx.KVStore(restarted.keys["knowledge"]).Get(key))
	newClaim := &knowledgetypes.Claim{Id: "new-claim", ReviewPolicyVersion: knowledgetypes.ReviewPolicyNeutral, Domain: "general", FactContent: "New separately authored contribution", Stake: "100000"}
	require.NoError(t, restarted.KnowledgeKeeper.SetClaim(rctx, newClaim))
	newRound, err := restarted.KnowledgeKeeper.CreateVerificationRound(rctx, newClaim)
	require.NoError(t, err)
	require.Equal(t, uint32(2), newRound.CommitmentScheme)
	require.Equal(t, knowledgetypes.ReviewPolicyNeutral, newRound.ReviewPolicyVersion)
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
	freezeReviewCompiledTarget(imported)
	require.NoError(t, imported.LoadLatestVersion())
	ictx := imported.NewUncachedContext(false, cmtproto.Header{ChainID: "record-import"})
	_, err = imported.InitChainer(ictx, &abci.RequestInitChain{ChainId: "record-import", AppStateBytes: raw})
	require.NoError(t, err)
	retained, found := imported.KnowledgeKeeper.GetVerificationRound(ictx, newRound.Id)
	require.True(t, found)
	require.True(t, proto.Equal(newRound, retained), "import keeps the original commitment chain")
	require.Empty(t, imported.KnowledgeKeeper.ReadMigrationMarker(ictx, "migration_v9_complete"))
	done, err := imported.UpgradeKeeper.GetDoneHeight(ictx, plan.Name)
	require.NoError(t, err)
	require.Zero(t, done)
}

func TestReviewNeutralityRejectsSourceOrMigrationCorruptionAtomically(t *testing.T) {
	for _, scenario := range []string{"source already enabled", "mixed version", "missing version", "unknown version", "plan info", "malformed round", "neutral policy predecessor", "corrupt history"} {
		t.Run(scenario, func(t *testing.T) {
			app, ctx, _ := reviewNeutralitySourceFixture(t)
			store := ctx.KVStore(app.keys["knowledge"])
			vm := reviewNeutralitySourceVersionMap()
			switch scenario {
			case "source already enabled":
				require.NoError(t, app.KnowledgeKeeper.EnableReviewNeutrality(ctx))
			case "mixed version":
				vm["vesting_rewards"] = 2
			case "missing version":
				delete(vm, "bank")
			case "unknown version":
				vm["unexpected"] = 1
			case "malformed round":
				store.Set(knowledgetypes.RoundKey("bad"), []byte{255})
			case "neutral policy predecessor":
				round := &knowledgetypes.VerificationRound{Id: "bad", ClaimId: "claim", Phase: knowledgetypes.VerificationPhase_VERIFICATION_PHASE_COMMIT, CommitmentScheme: 2, CommitmentChainId: ctx.ChainID(), ReviewPolicyVersion: knowledgetypes.ReviewPolicyNeutral}
				bz, err := proto.Marshal(round)
				require.NoError(t, err)
				store.Set(knowledgetypes.RoundKey("bad"), bz)
			case "corrupt history":
				store.Set(knowledgetypes.StatusTransitionKey("fact", 1), []byte{255})
			}
			beforeBad := bytes.Clone(store.Get(knowledgetypes.RoundKey("bad")))
			plan := upgradetypes.Plan{Name: UpgradeNameKnowledgeReviewNeutralityV1, Height: app.CommitMultiStore().Commit().Version + 1}
			if scenario == "plan info" {
				plan.Info = "unexpected"
			}
			ctx = ctx.WithBlockHeight(plan.Height).WithHeaderInfo(header.Info{Height: plan.Height, ChainID: ctx.ChainID()})
			_, err := app.RunUpgradeHandlerWithInfoForTests(ctx, plan.Name, vm, plan.Height, plan.Info)
			require.Error(t, err)
			require.Empty(t, app.KnowledgeKeeper.ReadMigrationMarker(ctx, "migration_v9_complete"))
			done, err := app.UpgradeKeeper.GetDoneHeight(ctx, plan.Name)
			require.NoError(t, err)
			require.Zero(t, done)
			require.Equal(t, beforeBad, store.Get(knowledgetypes.RoundKey("bad")))
		})
	}
}

func TestReviewNeutralityTargetRequiresNativeSelectionAndCoherentReceipt(t *testing.T) {
	for _, scenario := range []string{"missing selection", "false selection", "marker only", "done only", "future done", "corrupt enabled"} {
		t.Run(scenario, func(t *testing.T) {
			app, ctx, _ := newFrozenReviewFixture(t)
			if scenario == "missing selection" || scenario == "false selection" {
				genesis := sdk053IBC10GenesisWithValidator(t, app)
				var knowledge map[string]json.RawMessage
				require.NoError(t, json.Unmarshal(genesis["knowledge"], &knowledge))
				delete(knowledge, "review_neutrality_enabled")
				if scenario == "false selection" {
					knowledge["review_neutrality_enabled"] = json.RawMessage("false")
				}
				genesis["knowledge"], _ = json.Marshal(knowledge)
				require.Error(t, validateReviewNeutralityGenesisSelection(genesis))
				return
			}
			if scenario == "marker only" || scenario == "future done" {
				require.NoError(t, app.KnowledgeKeeper.WriteMigrationMarker(ctx, "migration_v9_complete", "true"))
			}
			if scenario == "done only" {
				writeSDK053IBC10TestDoneHeight(app, ctx, UpgradeNameKnowledgeReviewNeutralityV1, 1)
			}
			if scenario == "future done" {
				writeSDK053IBC10TestDoneHeight(app, ctx, UpgradeNameKnowledgeReviewNeutralityV1, 2)
			}
			if scenario == "corrupt enabled" {
				ctx.KVStore(app.keys["knowledge"]).Set([]byte(knowledgekeeper.ReviewNeutralityEnabledStoreKey), []byte{2})
			}
			app.CommitMultiStore().Commit()
			require.Error(t, app.ValidateAccountingAuthorityStartup())
		})
	}
}

func TestReviewNeutralityModuleRefusesUnownedMigration(t *testing.T) {
	app, ctx, _ := reviewNeutralitySourceFixture(t)
	_, err := app.ModuleManager.RunMigrations(ctx, app.configurator, reviewNeutralitySourceVersionMap())
	require.ErrorContains(t, err, "explicit")
	enabled, err := app.KnowledgeKeeper.ReviewNeutralityEnabled(ctx)
	require.NoError(t, err)
	require.False(t, enabled)
}

func TestReviewNeutralityMigrationAuditsRawPoliciesAndPendingRetryReachability(t *testing.T) {
	for _, scenario := range []string{"valid", "claim overflow", "round overflow", "duplicate claim policy", "wrong-wire round policy", "explicit zero claim", "explicit zero round", "explicit zero attribution", "preseeded attribution", "malformed attribution", "missing index", "orphan index", "malformed index", "invalid cursor", "extra cursor key"} {
		t.Run(scenario, func(t *testing.T) {
			app, ctx, _ := reviewNeutralitySourceFixture(t)
			claim := &knowledgetypes.Claim{Id: "pending-old-claim", VerificationRoundId: "pending-old-round", Stake: "100"}
			round := &knowledgetypes.VerificationRound{Id: claim.VerificationRoundId, ClaimId: claim.Id, Phase: knowledgetypes.VerificationPhase_VERIFICATION_PHASE_COMPLETE, Verdict: knowledgetypes.Verdict_VERDICT_REJECT, VerdictBlock: 1, VerifierRewardSettlement: &knowledgetypes.VerifierRewardSettlement{CreatedAtBlock: 1, WithheldTotal: "0", Payments: []*knowledgetypes.VerifierRewardPayment{{Verifier: settlementAddress(199).String(), Amount: "55", Withheld: "0"}}}}
			require.NoError(t, app.KnowledgeKeeper.SetClaim(ctx, claim))
			require.NoError(t, app.KnowledgeKeeper.SetVerificationRound(ctx, round))
			store := ctx.KVStore(app.keys["knowledge"])
			claimKey, roundKey := knowledgetypes.ClaimKey(claim.Id), knowledgetypes.RoundKey(round.Id)
			indexKey := append([]byte{0x82}, []byte(round.Id)...)
			addPolicy := func(key []byte, field protowire.Number, v uint64) {
				store.Set(key, protowire.AppendVarint(protowire.AppendTag(bytes.Clone(store.Get(key)), field, protowire.VarintType), v))
			}
			switch scenario {
			case "claim overflow":
				addPolicy(claimKey, 28, 1<<32)
			case "round overflow":
				addPolicy(roundKey, 16, 1<<32)
			case "duplicate claim policy":
				addPolicy(claimKey, 28, 1)
				addPolicy(claimKey, 28, 0)
			case "wrong-wire round policy":
				store.Set(roundKey, protowire.AppendString(protowire.AppendTag(bytes.Clone(store.Get(roundKey)), 16, protowire.BytesType), "1"))
			case "explicit zero claim":
				addPolicy(claimKey, 28, 0)
			case "explicit zero round":
				addPolicy(roundKey, 16, 0)
			case "explicit zero attribution", "preseeded attribution", "malformed attribution":
				record := &knowledgetypes.ContributionRecord{ModelId: "predecessor-model"}
				data, err := proto.Marshal(record)
				require.NoError(t, err)
				key := append(bytes.Clone(knowledgetypes.ContributionByModelKeyPrefix), []byte(record.ModelId)...)
				if scenario == "malformed attribution" {
					data = []byte{0xff}
				} else {
					var policy uint64
					if scenario == "preseeded attribution" {
						policy = 1
					}
					data = protowire.AppendVarint(protowire.AppendTag(data, 9, protowire.VarintType), policy)
				}
				store.Set(key, data)
			case "missing index":
				store.Delete(indexKey)
			case "orphan index":
				store.Set(append([]byte{0x82}, []byte("absent")...), []byte{1})
			case "malformed index":
				store.Set(indexKey, []byte{2})
			case "invalid cursor":
				store.Set([]byte{0x83}, []byte("invalid"))
			case "extra cursor key":
				store.Set([]byte{0x83, 1}, indexKey)
			case "valid":
				store.Set([]byte{0x83}, append([]byte{0x82}, []byte("already-paid-position")...))
			}
			beforeClaim, beforeRound, beforeIndex := bytes.Clone(store.Get(claimKey)), bytes.Clone(store.Get(roundKey)), bytes.Clone(store.Get(indexKey))
			supply := app.BankKeeper.GetSupply(ctx, "uzrn")
			latest := app.CommitMultiStore().Commit().Version
			plan := upgradetypes.Plan{Name: UpgradeNameKnowledgeReviewNeutralityV1, Height: latest + 2}
			require.NoError(t, app.UpgradeKeeper.ScheduleUpgrade(ctx, plan))
			require.Equal(t, plan.Height-1, app.CommitMultiStore().Commit().Version)
			ctx = ctx.WithBlockHeight(plan.Height).WithHeaderInfo(header.Info{Height: plan.Height, ChainID: ctx.ChainID()})
			err := app.UpgradeKeeper.ApplyUpgrade(ctx, plan)
			enabled, flagErr := app.KnowledgeKeeper.ReviewNeutralityEnabled(ctx)
			require.NoError(t, flagErr)
			vm, vmErr := app.UpgradeKeeper.GetModuleVersionMap(ctx)
			require.NoError(t, vmErr)
			done, doneErr := app.UpgradeKeeper.GetDoneHeight(ctx, plan.Name)
			require.NoError(t, doneErr)
			if scenario == "valid" {
				require.NoError(t, err)
				require.True(t, enabled)
				require.Equal(t, reviewNeutralityTargetVersionMap(), vm)
				require.Equal(t, plan.Height, done)
			} else {
				require.Error(t, err)
				require.False(t, enabled)
				require.Equal(t, reviewNeutralitySourceVersionMap(), vm)
				require.Zero(t, done)
				require.Empty(t, app.KnowledgeKeeper.ReadMigrationMarker(ctx, "migration_v9_complete"))
			}
			require.Equal(t, beforeClaim, store.Get(claimKey))
			require.Equal(t, beforeRound, store.Get(roundKey))
			require.Equal(t, beforeIndex, store.Get(indexKey))
			require.Equal(t, supply, app.BankKeeper.GetSupply(ctx, "uzrn"))
		})
	}
}
