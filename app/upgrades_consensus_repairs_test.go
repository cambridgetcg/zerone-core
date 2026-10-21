package app

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"cosmossdk.io/core/appmodule"
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
	"github.com/zerone-chain/zerone/internal/activation"
	cc "github.com/zerone-chain/zerone/x/capture_challenge/types"
	cd "github.com/zerone-chain/zerone/x/capture_defense/types"
	knowledgemodule "github.com/zerone-chain/zerone/x/knowledge"
	knowledge "github.com/zerone-chain/zerone/x/knowledge/types"
	qualification "github.com/zerone-chain/zerone/x/qualification/types"
	staking "github.com/zerone-chain/zerone/x/staking/types"
	"google.golang.org/protobuf/proto"
)

// Historical release tests pin all newly bumped siblings, not just knowledge.
// Concrete knowledge wrappers below still supply that release's exact version.
type archivedConsensusSibling struct {
	appmodule.AppModule
	module.HasGenesis
	version uint64
}

func (m archivedConsensusSibling) ConsensusVersion() uint64 { return m.version }

// Version pinning must not silently remove real block hooks from historical
// release fixtures. Modules without a hook still do nothing.
func (m archivedConsensusSibling) BeginBlock(ctx context.Context) error {
	if hook, ok := m.AppModule.(appmodule.HasBeginBlocker); ok {
		return hook.BeginBlock(ctx)
	}
	return nil
}

func (m archivedConsensusSibling) EndBlock(ctx context.Context) error {
	if hook, ok := m.AppModule.(appmodule.HasEndBlocker); ok {
		return hook.EndBlock(ctx)
	}
	return nil
}

type archivedClaimIntegrityKnowledgeModule struct{ knowledgemodule.AppModule }

func (archivedClaimIntegrityKnowledgeModule) ConsensusVersion() uint64 { return 12 }
func freezeConsensusRepairsSiblingTargets(app *ZeroneApp) {
	source := consensusRepairsSourceVersionMap()
	for name := range consensusRepairsVersions() {
		if name != "knowledge" {
			original := app.ModuleManager.Modules[name]
			app.ModuleManager.Modules[name] = archivedConsensusSibling{original.(appmodule.AppModule), original.(module.HasGenesis), source[name]}
		}
	}
}
func freezeClaimIntegrityCompiledTarget(app *ZeroneApp) {
	freezeConsensusRepairsSiblingTargets(app)
	current := app.ModuleManager.Modules["knowledge"]
	if frozen, ok := current.(archivedRepairsV1KnowledgeModule); ok {
		current = frozen.AppModule
	}
	app.ModuleManager.Modules["knowledge"] = archivedClaimIntegrityKnowledgeModule{current.(knowledgemodule.AppModule)}
}

func consensusRepairsFixture(t *testing.T, skip ...int) (*ZeroneApp, sdk.Context, dbm.DB) {
	t.Helper()
	db := dbm.NewMemDB()
	app := NewZeroneApp(log.NewNopLogger(), db, nil, true, simtestutil.AppOptionsMap{flags.FlagHome: t.TempDir(), server.FlagUnsafeSkipUpgrades: skip}, baseapp.SetChainID("zerone-dev-1"))
	freezeConsensusRepairsV1CompiledTarget(app)
	genesis, err := json.Marshal(sdk053IBC10GenesisWithValidator(t, app))
	require.NoError(t, err)
	ctx := app.NewUncachedContext(false, cmtproto.Header{ChainID: "zerone-dev-1"})
	_, err = app.InitChainer(ctx, &abci.RequestInitChain{ChainId: ctx.ChainID(), AppStateBytes: genesis})
	require.NoError(t, err)
	for _, loc := range consensusRepairsMarkerLocations() {
		ctx.KVStore(app.keys[loc.store]).Delete(loc.key)
	}
	// Synthetic registry and nonzero ledger, not a dev-1 measurement. Empty or
	// inactive registries alone do not change the zero-stake sensor reading.
	validator := &staking.Validator{OperatorAddress: settlementAddress(240).String(), TotalStake: "0", SelfDelegation: "0", DelegatedStake: "0"}
	raw, err := json.Marshal(validator)
	require.NoError(t, err)
	ctx.KVStore(app.keys["zerone_staking"]).Set(append([]byte{1}, []byte(validator.OperatorAddress)...), raw)
	ctx.KVStore(app.keys["vesting_rewards"]).Set([]byte{0x0b}, []byte("1"))
	require.NoError(t, app.UpgradeKeeper.SetModuleVersionMap(ctx, consensusRepairsSourceVersionMap()))
	return app, ctx, db
}
func TestConsensusRepairsExactVersionBoundary(t *testing.T) {
	app, _, _ := consensusRepairsFixture(t)
	source, target := consensusRepairsSourceVersionMap(), consensusRepairsTargetVersionMap()
	require.Equal(t, target, app.CurrentModuleVersionMap())
	require.Equal(t, uint64(1), target["home"])
	changes := 0
	for name, version := range source {
		if target[name] != version {
			changes++
			require.Equal(t, version+1, target[name])
		}
	}
	require.Equal(t, 11, changes)
	guards := []func(string, module.VersionMap, module.VersionMap) error{requireConsensusRepairsTransitionOwner, requireAccountingTransitionOwner, requireSurvivalHandoffTransitionOwner, requireRecordIntegrityTransitionOwner, requireReviewNeutralityTransitionOwner, requireClaimRecordsTransitionOwner, requireFundSettlementTransitionOwner, requireClaimIntegrityTransitionOwner}
	for _, guard := range guards {
		for _, name := range append(app.KnownUpgradeNames(), "unrelated") {
			err := guard(name, source, target)
			if name == UpgradeNameConsensusRepairsV1 {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, "sole owner")
			}
		}
	}
	for name := range target {
		bad := consensusRepairsSourceVersionMap()
		delete(bad, name)
		require.Error(t, requireConsensusRepairsTransitionOwner(UpgradeNameConsensusRepairsV1, bad, target))
	}
	require.Error(t, requireConsensusRepairsTransitionOwner(UpgradeNameConsensusRepairsV1, target, source))
}
func TestConsensusRepairsAtomicUpgradeDiffAndRestart(t *testing.T) {
	app, ctx, db := consensusRepairsFixture(t)
	census, err := app.CensusConsensusRepairsV1(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, census.Digest)
	latest := app.CommitMultiStore().Commit().Version
	plan := upgradetypes.Plan{Name: UpgradeNameConsensusRepairsV1, Height: latest + 2}
	require.NoError(t, app.UpgradeKeeper.ScheduleUpgrade(ctx, plan))
	require.NoError(t, app.UpgradeKeeper.DumpUpgradeInfoToDisk(plan.Height, plan))
	require.ErrorContains(t, app.ValidateAccountingAuthorityStartup(), "H-1")
	require.Equal(t, plan.Height-1, app.CommitMultiStore().Commit().Version)
	require.NoError(t, app.ValidateAccountingAuthorityStartup())
	ctx = ctx.WithBlockHeight(plan.Height).WithHeaderInfo(header.Info{Height: plan.Height, ChainID: ctx.ChainID()})
	before := claimIntegrityAllStores(t, app, ctx)
	cache, write := ctx.CacheContext()
	require.NoError(t, app.UpgradeKeeper.ApplyUpgrade(cache, plan))
	require.Equal(t, before, claimIntegrityAllStores(t, app, ctx))
	write()
	after := claimIntegrityAllStores(t, app, ctx)
	for _, marker := range app.ConsensusRepairsV1Markers() {
		on, err := marker.Enabled(ctx)
		require.NoError(t, err)
		require.True(t, on)
	}
	for _, loc := range consensusRepairsMarkerLocations() {
		row := loc.store + ":" + string(loc.key)
		require.Equal(t, []byte{1}, after[row])
		delete(after, row)
	}
	receiptRaw, present, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(ctx, consensusRepairsReceiptKey)
	require.NoError(t, err)
	require.True(t, present)
	var receipt consensusRepairsReceipt
	require.NoError(t, json.Unmarshal([]byte(receiptRaw), &receipt))
	require.Len(t, receipt.Markers, 10)
	require.Equal(t, plan.Height, receipt.Height)
	delete(after, "knowledge:"+string(append([]byte{0x7f, 1}, []byte(consensusRepairsReceiptKey)...)))
	doneKey := append([]byte{upgradetypes.DoneByte}, binary.BigEndian.AppendUint64(nil, uint64(plan.Height))...)
	doneKey = append(doneKey, []byte(plan.Name)...)
	allowed := []string{"upgrade:" + string(upgradetypes.PlanKey()), "upgrade:" + string(doneKey), "upgrade:" + string([]byte{upgradetypes.ProtocolVersionByte})}
	for name := range consensusRepairsVersions() {
		allowed = append(allowed, "upgrade:"+string(append([]byte{upgradetypes.VersionMapByte}, []byte(name)...)))
	}
	for _, key := range allowed {
		delete(before, key)
		delete(after, key)
	}
	require.Equal(t, before, after, "exact all-store diff excludes every business record and claim-integrity 0x87")
	vm, err := app.UpgradeKeeper.GetModuleVersionMap(ctx)
	require.NoError(t, err)
	require.Equal(t, consensusRepairsTargetVersionMap(), module.VersionMap(vm))
	app.CommitMultiStore().Commit()
	require.NoError(t, app.ValidateAccountingAuthorityStartup())
	restarted := NewZeroneApp(log.NewNopLogger(), db, nil, false, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()), baseapp.SetChainID(ctx.ChainID()))
	freezeConsensusRepairsV1CompiledTarget(restarted)
	require.NoError(t, restarted.LoadLatestVersion())
	require.NoError(t, restarted.ValidateAccountingAuthorityStartup())
	require.Error(t, app.UpgradeKeeper.ApplyUpgrade(ctx, plan))
	genesis, err := app.ModuleManager.ExportGenesis(ctx, app.appCodec)
	require.NoError(t, err)
	lineage, err := app.accountingAuthorityGenesisMetadata(ctx, plan.Height)
	require.NoError(t, err)
	genesis[accountingAuthorityGenesisKey], err = json.Marshal(lineage)
	require.NoError(t, err)
	require.NoError(t, validateConsensusRepairsGenesisSelection(genesis))
	raw, err := json.Marshal(genesis)
	require.NoError(t, err)
	imported := NewZeroneApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()), baseapp.SetChainID("zerone-rehearsal-import"))
	ictx := imported.NewUncachedContext(false, cmtproto.Header{ChainID: "zerone-rehearsal-import"})
	_, err = imported.InitChainer(ictx, &abci.RequestInitChain{ChainId: ictx.ChainID(), AppStateBytes: raw})
	require.NoError(t, err)
	require.Empty(t, imported.KnowledgeKeeper.ReadMigrationMarker(ictx, consensusRepairsReceiptKey), "import invents no applied receipt")
	done, err := imported.UpgradeKeeper.GetDoneHeight(ictx, UpgradeNameConsensusRepairsV1)
	require.NoError(t, err)
	require.Zero(t, done)
	imported.CommitMultiStore().Commit()
	require.NoError(t, imported.ValidateAccountingAuthorityStartup())
}
func TestConsensusRepairsCensusBlockersLeaveEveryMarkerOff(t *testing.T) {
	for _, scenario := range []string{"pool deficit", "bounty identity", "augmentation identity", "empty marker", "corrupt marker", "neutral discontinuity", "occupied generation", "corrupt halt"} {
		t.Run(scenario, func(t *testing.T) {
			app, ctx, _ := consensusRepairsFixture(t)
			switch scenario {
			case "pool deficit":
				raw, err := proto.Marshal(&cc.DomainBountyPool{Domain: "physics", Balance: "1"})
				require.NoError(t, err)
				ctx.KVStore(app.keys["capture_challenge"]).Set([]byte("\x02physics"), raw)
			case "bounty identity":
				raw, err := proto.Marshal(&knowledge.AugmentationBounty{Id: "bad\x00id"})
				require.NoError(t, err)
				ctx.KVStore(app.keys["knowledge"]).Set([]byte("\x62different"), raw)
			case "augmentation identity":
				raw, err := proto.Marshal(&knowledge.Augmentation{Id: "bad\x00id"})
				require.NoError(t, err)
				ctx.KVStore(app.keys["knowledge"]).Set([]byte("\x63different"), raw)
			case "empty marker":
				ctx.KVStore(app.keys["qualification"]).Set([]byte{0x40}, []byte{})
			case "corrupt marker":
				ctx.KVStore(app.keys["qualification"]).Set([]byte{0x40}, []byte{2})
			case "neutral discontinuity":
				ctx.KVStore(app.keys["vesting_rewards"]).Delete([]byte{0x0b})
			case "occupied generation":
				ctx.KVStore(app.keys["zerone_ibcrl"]).Set([]byte("\x03channel-0\x00uzrn"), make([]byte, 8))
			case "corrupt halt":
				ctx.KVStore(app.keys["emergency"]).Set([]byte("\x02bad"), []byte{255})
			}
			latest := app.CommitMultiStore().Commit().Version
			plan := upgradetypes.Plan{Name: UpgradeNameConsensusRepairsV1, Height: latest + 1}
			require.NoError(t, app.UpgradeKeeper.ScheduleUpgrade(ctx, plan))
			require.NoError(t, app.UpgradeKeeper.DumpUpgradeInfoToDisk(plan.Height, plan))
			require.Error(t, app.ValidateAccountingAuthorityStartup(), "refuse before H")
			ctx = ctx.WithBlockHeight(plan.Height).WithHeaderInfo(header.Info{Height: plan.Height})
			before := claimIntegrityAllStores(t, app, ctx)
			report, err := app.CensusConsensusRepairsV1(ctx)
			require.Error(t, err)
			require.NotEmpty(t, report.Findings)
			require.Error(t, app.UpgradeKeeper.ApplyUpgrade(ctx, plan))
			require.Equal(t, before, claimIntegrityAllStores(t, app, ctx))
			for _, marker := range app.ConsensusRepairsV1Markers() {
				on, _ := marker.Enabled(ctx)
				require.False(t, on)
			}
		})
	}
}
func TestConsensusRepairsExpiredAmountsAreHeldNotMoved(t *testing.T) {
	app, ctx, _ := consensusRepairsFixture(t)
	p := &qualification.DomainQualification{Validator: settlementAddress(241).String(), Domain: "physics", Pathway: qualification.QualificationPathway_QUALIFICATION_PATHWAY_STAKE, Status: qualification.QualificationStatus_QUALIFICATION_STATUS_EXPIRED, StakedAmount: "42"}
	raw, err := proto.Marshal(p)
	require.NoError(t, err)
	key := []byte("\x01" + p.Validator + "/" + p.Domain)
	ctx.KVStore(app.keys["qualification"]).Set(key, raw)
	before := claimIntegrityAllStores(t, app, ctx)
	report, err := app.CensusConsensusRepairsV1(ctx)
	require.NoError(t, err)
	require.Equal(t, before, claimIntegrityAllStores(t, app, ctx))
	encoded, err := json.Marshal(report)
	require.NoError(t, err)
	require.Contains(t, string(encoded), "EXPIRED retained amount=42")
}
func TestConsensusRepairsRequiresAllExplicitGenesisSelections(t *testing.T) {
	app, _, _ := newAccountingAuthorityFixture(t)
	require.Len(t, app.ConsensusRepairsV1Markers(), 10)
	for _, marker := range app.ConsensusRepairsV1Markers() {
		for _, selection := range []string{"absent", "false", "null", "uppercase", "mixed-case", "camelCase", "alias-after-canonical"} {
			t.Run(marker.Module+"/"+selection, func(t *testing.T) {
				genesis := sdk053IBC10GenesisWithValidator(t, app)
				var fields map[string]json.RawMessage
				require.NoError(t, json.Unmarshal(genesis[marker.Module], &fields))
				delete(fields, marker.Name)
				switch selection {
				case "uppercase":
					fields[marker.Name] = json.RawMessage("true")
					fields[strings.ToUpper(marker.Name)] = json.RawMessage("false")
				case "mixed-case":
					fields[marker.Name] = json.RawMessage("true")
					fields[strings.ToUpper(marker.Name[:1])+marker.Name[1:]] = json.RawMessage("false")
				case "camelCase":
					fields[marker.Name] = json.RawMessage("true")
					fields[consensusRepairsCamelCase(marker.Name)] = json.RawMessage("false")
				case "alias-after-canonical":
					fields[marker.Name] = json.RawMessage("true")
				case "false", "null":
					fields[marker.Name] = json.RawMessage(selection)
				}
				var err error
				genesis[marker.Module], err = json.Marshal(fields)
				require.NoError(t, err)
				if selection == "alias-after-canonical" {
					// Pin the review's exact last-value-wins bypass, independent
					// of encoding/json's sorted map serialization order.
					original := genesis[marker.Module]
					genesis[marker.Module] = json.RawMessage(fmt.Sprintf(`%s,%q:false}`, original[:len(original)-1], strings.ToUpper(marker.Name)))
				}
				raw, err := json.Marshal(genesis)
				require.NoError(t, err)
				fresh := NewZeroneApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()))
				ctx := fresh.NewUncachedContext(false, cmtproto.Header{})
				before := claimIntegrityAllStores(t, fresh, ctx)
				_, err = fresh.InitChainer(ctx, &abci.RequestInitChain{AppStateBytes: raw})
				require.ErrorContains(t, err, marker.Name+"=true")
				require.Equal(t, before, claimIntegrityAllStores(t, fresh, ctx))
			})
		}
	}
}
func TestConsensusRepairsUnownedMigrationWritesNothing(t *testing.T) {
	app, ctx, _ := consensusRepairsFixture(t)
	before := claimIntegrityAllStores(t, app, ctx)
	_, err := app.ModuleManager.RunMigrations(ctx, app.configurator, consensusRepairsSourceVersionMap())
	require.ErrorContains(t, err, "explicit")
	require.Equal(t, before, claimIntegrityAllStores(t, app, ctx))
}
func TestConsensusRepairsComposedChallengeFault(t *testing.T) {
	app, ctx, _ := newAccountingAuthorityFixture(t)
	for i := 0; i < 10; i++ {
		app.CaptureDefenseKeeper.SetVerificationHistory(ctx, &cd.VerificationHistoryEntry{Domain: "monopoly", RoundId: fmt.Sprint(i), Validators: []string{settlementAddress(242).String()}, Verdicts: []bool{true}, SubmitBlocks: []uint64{1, 1}})
	}
	ctx.KVStore(app.keys["capture_challenge"]).Set(cc.ChallengeAccountingEnabledKey, []byte{2})
	before := claimIntegrityAllStores(t, app, ctx)
	cache, _ := ctx.WithBlockHeight(1000).CacheContext()
	err := app.CaptureDefenseKeeper.BeginBlocker(cache)
	require.ErrorIs(t, err, activation.ErrMarkerFault)
	require.ErrorContains(t, err, "auto-challenge")
	require.ErrorContains(t, err, "challenge accounting")
	require.Equal(t, before, claimIntegrityAllStores(t, app, ctx), "block cache discarded on propagated fault")
	require.True(t, bytes.Equal(ctx.KVStore(app.keys["capture_challenge"]).Get(cc.ChallengeAccountingEnabledKey), []byte{2}))
}
