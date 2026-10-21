package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/zerone-chain/zerone/app/consensusrepairsv2migration"
	knowledge "github.com/zerone-chain/zerone/x/knowledge/types"
	"google.golang.org/protobuf/proto"
)

const UpgradeNameConsensusRepairsV2 = consensusrepairsv2migration.UpgradeName
const consensusRepairsV2ReceiptKey = "upgrade_receipt_consensus-repairs-v2"

func consensusRepairsV2SourceVersionMap() module.VersionMap {
	return consensusRepairsTargetVersionMap()
}
func consensusRepairsV2TargetVersionMap() module.VersionMap {
	vm := consensusRepairsV2SourceVersionMap()
	vm["alignment"], vm["knowledge"] = 3, 14
	return vm
}

func requireConsensusRepairsV2TransitionOwner(name string, from, target module.VersionMap) error {
	if reflect.DeepEqual(from, consensusRepairsV2TargetVersionMap()) && reflect.DeepEqual(from, target) {
		return nil
	}
	if name != UpgradeNameConsensusRepairsV2 || !reflect.DeepEqual(from, consensusRepairsV2SourceVersionMap()) || !reflect.DeepEqual(target, consensusRepairsV2TargetVersionMap()) {
		return fmt.Errorf("upgrade %q cannot carry consensus repairs v2; sole owner is %q with exact source and target maps", name, UpgradeNameConsensusRepairsV2)
	}
	return nil
}

func consensusRepairsV2MarkerLocations() []consensusRepairsMarkerLocation {
	return []consensusRepairsMarkerLocation{{"alignment", []byte{0x09}}, {"knowledge", []byte{0x91}}, {"knowledge", []byte{0x92}}, {"knowledge", []byte{0x93}}}
}

// V1's target branch validates its markers and receipt without consulting the
// compiled map. Reuse that frozen completed-history contract, not its census
// (which correctly refuses already-enabled v1 markers).
func (app *ZeroneApp) validateConsensusRepairsV1Completed(ctx sdk.Context, latest int64) error {
	return app.validateConsensusRepairsStartupVersions(ctx, consensusRepairsTargetVersionMap(), latest)
}

func (app *ZeroneApp) validateConsensusRepairsV2Source(ctx sdk.Context, plan upgradetypes.Plan, from module.VersionMap) (ConsensusRepairsCensus, []*knowledge.AgentCalibration, error) {
	var empty ConsensusRepairsCensus
	fail := func(err error) (ConsensusRepairsCensus, []*knowledge.AgentCalibration, error) { return empty, nil, err }
	if plan.Name != UpgradeNameConsensusRepairsV2 || plan.Height <= 0 || plan.Info != "" || !plan.Time.IsZero() || plan.UpgradedClientState != nil {
		return fail(fmt.Errorf("consensus repairs v2 requires its named height plan with empty info"))
	}
	if ctx.ChainID() != "zerone-dev-1" && !strings.HasPrefix(ctx.ChainID(), "zerone-rehearsal-") {
		return fail(fmt.Errorf("consensus repairs v2 requires dev-1 or an explicitly isolated zerone-rehearsal- lineage"))
	}
	if !reflect.DeepEqual(from, consensusRepairsV2SourceVersionMap()) || !reflect.DeepEqual(app.ModuleManager.GetVersionMap(), consensusRepairsV2TargetVersionMap()) {
		return fail(fmt.Errorf("consensus repairs v2 requires exact source and compiled target maps"))
	}
	stored, err := app.UpgradeKeeper.GetModuleVersionMap(ctx)
	if err != nil || !reflect.DeepEqual(stored, from) {
		return fail(fmt.Errorf("consensus repairs v2 stored source map mismatch: %v", err))
	}
	if err := app.validateSDK053IBC10CompletedOrNativeLineage(); err != nil {
		return fail(err)
	}
	latest := app.CommitMultiStore().LastCommitID().Version
	lineage, err := app.accountingAuthorityGenesisMetadata(ctx, latest)
	if err != nil || lineage.Origin == "legacy" {
		return fail(fmt.Errorf("consensus repairs v2 requires completed accounting lineage: %v", err))
	}
	if err := app.validateConsensusRepairsV1Completed(ctx, latest); err != nil {
		return fail(err)
	}
	done, err := app.UpgradeKeeper.GetDoneHeight(ctx, plan.Name)
	if err != nil || done != 0 {
		return fail(fmt.Errorf("consensus repairs v2 already applied or unreadable: %v", err))
	}
	_, present, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(ctx, consensusRepairsV2ReceiptKey)
	if err != nil || present {
		return fail(fmt.Errorf("consensus repairs v2 receipt already present or unreadable: %v", err))
	}
	if app.UpgradeKeeper.IsSkipHeight(plan.Height) {
		return fail(fmt.Errorf("consensus repairs v2 cannot be unsafe-skipped"))
	}
	scheduled, err := app.UpgradeKeeper.GetUpgradePlan(ctx)
	if err != nil || !reflect.DeepEqual(scheduled, plan) {
		return fail(fmt.Errorf("consensus repairs v2 requires exact on-chain plan: %v", err))
	}
	return app.censusConsensusRepairsV2(ctx)
}

func (app *ZeroneApp) registerConsensusRepairsV2Upgrade() {
	app.UpgradeKeeper.SetUpgradeHandler(UpgradeNameConsensusRepairsV2, func(goCtx context.Context, plan upgradetypes.Plan, from module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(goCtx)
		if ctx.BlockHeight() != plan.Height || ctx.HeaderInfo().Height != plan.Height || app.CommitMultiStore().LastCommitID().Version != plan.Height-1 {
			return nil, fmt.Errorf("consensus repairs v2 requires exact committed H-1/H boundary")
		}
		census, calibrations, err := app.validateConsensusRepairsV2Source(ctx, plan, from)
		if err != nil {
			return nil, err
		}
		compact, err := census.receiptFor(consensusRepairsV2CensusNames)
		if err != nil {
			return nil, err
		}
		receipt := consensusRepairsReceipt{Height: plan.Height, ChainID: ctx.ChainID(), PlanDigest: consensusRepairsPlanDigest(plan), Census: compact}
		markers := app.ConsensusRepairsV2Markers()
		writes := map[string]map[string][]byte{"alignment": {}, "knowledge": {}}
		for i, loc := range consensusRepairsV2MarkerLocations() {
			writes[loc.store][string(loc.key)] = []byte{1}
			receipt.Markers = append(receipt.Markers, consensusRepairsMarkerReceipt{markers[i].Module, markers[i].Name, true})
		}
		// The census strictly decoded canonical records and changed only the
		// score in memory. Commit the expected score bytes BEFORE invoking any
		// migration or writer; never bless the writer's actual output as expected.
		for _, calibration := range calibrations {
			raw, err := (proto.MarshalOptions{Deterministic: true}).Marshal(calibration)
			if err != nil {
				return nil, err
			}
			writes["knowledge"][string(knowledge.AgentCalibrationKey(calibration.Address))] = raw
		}
		raw, err := json.Marshal(receipt)
		if err != nil {
			return nil, err
		}
		writes["knowledge"][string(append([]byte{0x7f, 1}, []byte(consensusRepairsV2ReceiptKey)...))] = raw
		expected, claimBefore, err := app.consensusRepairsStream(ctx, writes, nil)
		if err != nil {
			return nil, err
		}
		cache, write := ctx.CacheContext()
		authorized, err := consensusrepairsv2migration.WithOwner(cache, plan.Name)
		if err != nil {
			return nil, err
		}
		to, err := app.ModuleManager.RunMigrations(authorized, app.configurator, from)
		if err != nil {
			return nil, err
		}
		if !reflect.DeepEqual(to, consensusRepairsV2TargetVersionMap()) {
			return nil, fmt.Errorf("consensus repairs v2 unexpected migration target")
		}
		// Q5 recompute and 0x92 must be indivisible, including on a later marker
		// or receipt failure. No historical fact snapshots are rewritten.
		for _, calibration := range calibrations {
			if err := app.KnowledgeKeeper.SetAgentCalibration(cache, calibration); err != nil {
				return nil, err
			}
		}
		for _, marker := range markers {
			if err := marker.Enable(cache); err != nil {
				return nil, err
			}
			on, err := marker.Enabled(cache)
			if err != nil {
				return nil, err
			}
			if !on {
				return nil, fmt.Errorf("consensus repairs v2 marker readback off: %s.%s", marker.Module, marker.Name)
			}
		}
		if err := app.KnowledgeKeeper.WriteMigrationMarker(cache, consensusRepairsV2ReceiptKey, string(raw)); err != nil {
			return nil, err
		}
		after, claimAfter, err := app.consensusRepairsStream(cache, nil, nil)
		if err != nil {
			return nil, err
		}
		if expected != after || claimBefore != claimAfter {
			return nil, fmt.Errorf("consensus repairs v2 unexpected store diff (including claim-integrity 0x87)")
		}
		for _, name := range consensusRepairsV2CensusNames {
			cache.EventManager().EmitEvent(sdk.NewEvent("consensus_repairs_v2_census", sdk.NewAttribute("census", name), sdk.NewAttribute("digest", census.Digests[name])))
		}
		write()
		return to, nil
	})
}

func (app *ZeroneApp) validateConsensusRepairsV2StartupVersions(ctx sdk.Context, vm module.VersionMap, latest int64) error {
	if reflect.DeepEqual(vm, consensusRepairsV2TargetVersionMap()) {
		if err := app.validateConsensusRepairsV1Completed(ctx, latest); err != nil {
			return err
		}
		markers := app.ConsensusRepairsV2Markers()
		for _, marker := range markers {
			on, err := marker.Enabled(ctx)
			if err != nil || !on {
				return fmt.Errorf("consensus repairs v2 target requires %s.%s: %v", marker.Module, marker.Name, err)
			}
		}
		done, err := app.UpgradeKeeper.GetDoneHeight(ctx, UpgradeNameConsensusRepairsV2)
		if err != nil {
			return err
		}
		raw, present, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(ctx, consensusRepairsV2ReceiptKey)
		if err != nil {
			return err
		}
		if done == 0 && !present {
			return nil
		} // Native/imported, not invented history.
		var receipt consensusRepairsReceipt
		if done <= 0 || done > latest || !present || json.Unmarshal([]byte(raw), &receipt) != nil {
			return fmt.Errorf("consensus repairs v2 inconsistent receipt/done height")
		}
		if receipt.Height != done || receipt.ChainID != ctx.ChainID() || receipt.PlanDigest != consensusRepairsPlanDigest(upgradetypes.Plan{Name: UpgradeNameConsensusRepairsV2, Height: done}) || receipt.Census.Height != done {
			return fmt.Errorf("consensus repairs v2 receipt identity mismatch")
		}
		digest, err := hex.DecodeString(receipt.Census.Digest)
		if err != nil || len(digest) != sha256.Size || receipt.Census.Records < 0 || receipt.Census.Bytes < 0 {
			return fmt.Errorf("consensus repairs v2 invalid census digest/counts")
		}
		if len(receipt.Census.Censuses) != len(consensusRepairsV2CensusNames) || len(receipt.Markers) != len(markers) {
			return fmt.Errorf("consensus repairs v2 incomplete receipt")
		}
		for _, name := range consensusRepairsV2CensusNames {
			summary, ok := receipt.Census.Censuses[name]
			digest, err := hex.DecodeString(summary.Digest)
			if !ok || err != nil || len(digest) != sha256.Size || summary.Blockers != 0 || summary.PendingRollbacks != 0 || summary.HeldFindings < 0 || summary.Records < 0 || summary.Records > consensusRepairsMaxRecords || summary.Bytes < 0 || summary.Bytes > consensusRepairsMaxBytes {
				return fmt.Errorf("consensus repairs v2 invalid %s summary", name)
			}
		}
		for i, marker := range markers {
			if receipt.Markers[i] != (consensusRepairsMarkerReceipt{marker.Module, marker.Name, true}) {
				return fmt.Errorf("consensus repairs v2 marker receipt mismatch")
			}
		}
		return nil
	}
	if !reflect.DeepEqual(vm, consensusRepairsV2SourceVersionMap()) {
		return fmt.Errorf("consensus repairs v2 startup requires exact source or target map")
	}
	local, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		return err
	}
	plan, err := app.UpgradeKeeper.GetUpgradePlan(ctx)
	if err != nil {
		return err
	}
	if latest == int64(^uint64(0)>>1) || plan.Height != latest+1 || local.Name != plan.Name || local.Height != plan.Height || local.Info != plan.Info {
		return fmt.Errorf("consensus repairs v2 source startup requires exact on-chain and local H-1 plan")
	}
	_, _, err = app.validateConsensusRepairsV2Source(ctx, plan, vm)
	return err
}
