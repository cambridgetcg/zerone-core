package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/zerone-chain/zerone/internal/consensusrepairsmigration"
	gov "github.com/zerone-chain/zerone/x/gov/types"
)

const UpgradeNameConsensusRepairsV1 = consensusrepairsmigration.UpgradeName
const consensusRepairsReceiptKey = "upgrade_receipt_consensus-repairs-v1"

func consensusRepairsSourceVersionMap() module.VersionMap { return claimIntegrityTargetVersionMap() }
func consensusRepairsTargetVersionMap() module.VersionMap {
	vm := consensusRepairsSourceVersionMap()
	for name, version := range consensusRepairsVersions() {
		vm[name] = version
	}
	return vm
}
func consensusRepairsVersions() module.VersionMap {
	return module.VersionMap{"zerone_gov": 4, "knowledge": 13, "capture_defense": 2, "qualification": 2, "capture_challenge": 2, "vesting_rewards": 4, "alignment": 2, "zerone_ontology": 2, "emergency": 3, "substrate_bridge": 2, "ibcratelimit": 2}
}

// Every caller, including older broad migrations, must respect the complete rung.
func requireConsensusRepairsTransitionOwner(name string, from, target module.VersionMap) error {
	if from["knowledge"] >= 14 || target["knowledge"] >= 14 || from["alignment"] >= 3 || target["alignment"] >= 3 {
		return requireConsensusRepairsV2TransitionOwner(name, from, target)
	}
	selected := false
	for key, version := range consensusRepairsVersions() {
		if from[key] >= version || target[key] >= version {
			selected = true
		}
	}
	if !selected {
		return nil
	} // Frozen predecessors retain their own guards.
	if reflect.DeepEqual(from, consensusRepairsTargetVersionMap()) && reflect.DeepEqual(from, target) {
		return nil
	}
	if name != UpgradeNameConsensusRepairsV1 || !reflect.DeepEqual(from, consensusRepairsSourceVersionMap()) || !reflect.DeepEqual(target, consensusRepairsTargetVersionMap()) {
		return fmt.Errorf("upgrade %q cannot carry consensus repairs; sole owner is %q with exact source and target maps", name, UpgradeNameConsensusRepairsV1)
	}
	return nil
}

type consensusRepairsMarkerReceipt struct {
	Module  string `json:"module"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

// Permanent receipts contain one fixed-shape summary per census. Detailed
// findings, arbitrary status strings and rollback identities stay offline.
type consensusRepairsCensusSummary struct {
	ConsensusRepairsSize
	Digest           string `json:"digest"`
	HeldFindings     int    `json:"held_findings"`
	Blockers         int    `json:"blockers"`
	PendingRollbacks int    `json:"pending_rollbacks"`
}

type consensusRepairsCensusReceipt struct {
	Height   int64                                    `json:"height"`
	Digest   string                                   `json:"digest"`
	Records  int                                      `json:"records"`
	Bytes    int                                      `json:"bytes"`
	Censuses map[string]consensusRepairsCensusSummary `json:"censuses"`
}

func (report ConsensusRepairsCensus) receipt() (consensusRepairsCensusReceipt, error) {
	return report.receiptFor(consensusRepairsCensusNames)
}

func (report ConsensusRepairsCensus) receiptFor(names []string) (consensusRepairsCensusReceipt, error) {
	compact := consensusRepairsCensusReceipt{
		Height: report.Height, Digest: report.Digest, Records: report.Records, Bytes: report.Bytes,
		Censuses: make(map[string]consensusRepairsCensusSummary, len(names)),
	}
	for _, name := range names {
		compact.Censuses[name] = consensusRepairsCensusSummary{ConsensusRepairsSize: report.Censuses[name], Digest: report.Digests[name]}
	}
	for _, finding := range report.Findings {
		summary, ok := compact.Censuses[finding.Census]
		if !ok || finding.Outcome != "held" {
			return compact, fmt.Errorf("consensus repairs cannot receipt unclassified or blocking finding")
		}
		summary.HeldFindings++
		compact.Censuses[finding.Census] = summary
	}
	if summary, ok := compact.Censuses["halt_reasons"]; ok {
		summary.PendingRollbacks = len(report.PendingRollbacks)
		compact.Censuses["halt_reasons"] = summary
	}
	return compact, nil
}

type consensusRepairsReceipt struct {
	Height     int64                           `json:"height"`
	ChainID    string                          `json:"chain_id"`
	PlanDigest string                          `json:"plan_digest"`
	Markers    []consensusRepairsMarkerReceipt `json:"markers"`
	Census     consensusRepairsCensusReceipt   `json:"census"`
}

func consensusRepairsPlanDigest(plan upgradetypes.Plan) string {
	encoded, _ := json.Marshal(plan)
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

func (app *ZeroneApp) validateConsensusRepairsSource(ctx sdk.Context, plan upgradetypes.Plan, from module.VersionMap) (ConsensusRepairsCensus, error) {
	var empty ConsensusRepairsCensus
	if plan.Name != UpgradeNameConsensusRepairsV1 || plan.Height <= 0 || plan.Info != "" || !plan.Time.IsZero() || plan.UpgradedClientState != nil {
		return empty, fmt.Errorf("consensus repairs requires its named height plan with empty info")
	}
	if ctx.ChainID() != "zerone-dev-1" && !strings.HasPrefix(ctx.ChainID(), "zerone-rehearsal-") {
		return empty, fmt.Errorf("consensus repairs requires dev-1 or an explicitly isolated zerone-rehearsal- lineage")
	}
	if !reflect.DeepEqual(from, consensusRepairsSourceVersionMap()) || !reflect.DeepEqual(app.ModuleManager.GetVersionMap(), consensusRepairsTargetVersionMap()) {
		return empty, fmt.Errorf("consensus repairs requires exact source and compiled target maps")
	}
	stored, err := app.UpgradeKeeper.GetModuleVersionMap(ctx)
	if err != nil || !reflect.DeepEqual(stored, from) {
		return empty, fmt.Errorf("consensus repairs stored source map mismatch: %v", err)
	}
	if err := app.validateSDK053IBC10CompletedOrNativeLineage(); err != nil {
		return empty, err
	}
	latest := app.CommitMultiStore().LastCommitID().Version
	lineage, err := app.accountingAuthorityGenesisMetadata(ctx, latest)
	if err != nil || lineage.Origin == "legacy" {
		return empty, fmt.Errorf("consensus repairs requires completed accounting lineage: %v", err)
	}
	if err := app.validateClaimIntegrityCompleted(ctx, latest); err != nil {
		return empty, err
	}
	done, err := app.UpgradeKeeper.GetDoneHeight(ctx, plan.Name)
	if err != nil || done != 0 {
		return empty, fmt.Errorf("consensus repairs already applied or unreadable: %v", err)
	}
	_, present, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(ctx, consensusRepairsReceiptKey)
	if err != nil || present {
		return empty, fmt.Errorf("consensus repairs source receipt already present or unreadable: %v", err)
	}
	if app.UpgradeKeeper.IsSkipHeight(plan.Height) {
		return empty, fmt.Errorf("consensus repairs cannot be unsafe-skipped")
	}
	scheduled, err := app.UpgradeKeeper.GetUpgradePlan(ctx)
	if err != nil || !reflect.DeepEqual(scheduled, plan) {
		return empty, fmt.Errorf("consensus repairs requires the exact on-chain plan: %v", err)
	}
	return app.CensusConsensusRepairsV1(ctx)
}

func (app *ZeroneApp) registerConsensusRepairsUpgrade() {
	app.UpgradeKeeper.SetUpgradeHandler(UpgradeNameConsensusRepairsV1, func(goCtx context.Context, plan upgradetypes.Plan, from module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(goCtx)
		if ctx.BlockHeight() != plan.Height || ctx.HeaderInfo().Height != plan.Height || app.CommitMultiStore().LastCommitID().Version != plan.Height-1 {
			return nil, fmt.Errorf("consensus repairs requires exact committed H-1/H boundary")
		}
		census, err := app.validateConsensusRepairsSource(ctx, plan, from)
		if err != nil {
			return nil, err
		}
		// Hash the source merged with the eleven expected writes BEFORE any
		// migrations or marker writers run. No full image is retained at H.
		compact, err := census.receipt()
		if err != nil {
			return nil, err
		}
		receipt := consensusRepairsReceipt{Height: plan.Height, ChainID: ctx.ChainID(), PlanDigest: consensusRepairsPlanDigest(plan), Census: compact}
		markers := app.ConsensusRepairsV1Markers()
		expectedWrites := make(map[string]map[string][]byte)
		for i, loc := range consensusRepairsMarkerLocations() {
			if expectedWrites[loc.store] == nil {
				expectedWrites[loc.store] = make(map[string][]byte)
			}
			expectedWrites[loc.store][string(loc.key)] = []byte{1}
			// These expected true values commit only after every Enable/readback succeeds.
			receipt.Markers = append(receipt.Markers, consensusRepairsMarkerReceipt{markers[i].Module, markers[i].Name, true})
		}
		encoded, err := json.Marshal(receipt)
		if err != nil {
			return nil, err
		}
		expectedWrites["knowledge"][string(append([]byte{0x7f, 0x01}, []byte(consensusRepairsReceiptKey)...))] = encoded
		expected, claimBefore, err := app.consensusRepairsStream(ctx, expectedWrites, nil)
		if err != nil {
			return nil, err
		}
		cache, write := ctx.CacheContext()
		authorized, err := consensusrepairsmigration.WithOwner(cache, plan.Name)
		if err != nil {
			return nil, err
		}
		to, err := app.ModuleManager.RunMigrations(authorized, app.configurator, from)
		if err != nil {
			return nil, err
		}
		if !reflect.DeepEqual(to, consensusRepairsTargetVersionMap()) {
			return nil, fmt.Errorf("consensus repairs unexpected migration target")
		}
		for _, marker := range markers {
			if err := marker.Enable(cache); err != nil {
				return nil, err
			}
			enabled, err := marker.Enabled(cache)
			if err != nil {
				return nil, err
			}
			if !enabled {
				return nil, fmt.Errorf("consensus repairs marker readback off: %s.%s", marker.Module, marker.Name)
			}
		}
		if err := app.KnowledgeKeeper.WriteMigrationMarker(cache, consensusRepairsReceiptKey, string(encoded)); err != nil {
			return nil, err
		}
		after, claimAfter, err := app.consensusRepairsStream(cache, nil, nil)
		if err != nil {
			return nil, err
		}
		// The migrations are guarded no-ops. Verify exact bytes before committing;
		// x/upgrade records its version map/done height only after this returns.
		if expected != after || claimBefore != claimAfter {
			return nil, fmt.Errorf("consensus repairs unexpected store diff (including claim-integrity 0x87)")
		}
		for _, name := range consensusRepairsCensusNames {
			cache.EventManager().EmitEvent(sdk.NewEvent("consensus_repairs_census", sdk.NewAttribute("census", name), sdk.NewAttribute("digest", census.Digests[name])))
		}
		write()
		return to, nil
	})
}

func validateConsensusRepairsGenesisSelection(genesis GenesisState) error {
	return ValidateConsensusRepairsGenesisSelection(genesis)
}

// ValidateConsensusRepairsGenesisSelection is shared by InitChain and the
// offline launch preflight. Require one exact spelling across JSON codecs.
func ValidateConsensusRepairsGenesisSelection(genesis GenesisState) error {
	for _, marker := range append(ConsensusRepairsV1GenesisSelections(), ConsensusRepairsV2GenesisSelections()...) {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(genesis[marker.Module], &fields); err != nil {
			return fmt.Errorf("consensus repairs genesis %s: %w", marker.Module, err)
		}
		alias := consensusRepairsCamelCase(marker.Name)
		names := make([]string, 0, len(fields))
		for name := range fields {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if (name != marker.Name && strings.EqualFold(name, marker.Name)) || strings.EqualFold(name, alias) {
				return fmt.Errorf("native %s genesis requires explicit %s=true with exact spelling; alias %q forbidden", marker.Module, marker.Name, name)
			}
		}
		var value *bool
		if err := json.Unmarshal(fields[marker.Name], &value); err != nil || value == nil || !*value {
			return fmt.Errorf("native %s genesis requires explicit %s=true", marker.Module, marker.Name)
		}
	}
	return gov.ValidateRetainedGenesisFieldNames(genesis["zerone_gov"])
}

func consensusRepairsCamelCase(name string) string {
	parts := strings.Split(name, "_")
	for i := 1; i < len(parts); i++ {
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return strings.Join(parts, "")
}

func (app *ZeroneApp) validateConsensusRepairsStartupVersions(ctx sdk.Context, vm module.VersionMap, latest int64) error {
	if reflect.DeepEqual(vm, consensusRepairsTargetVersionMap()) {
		if err := app.validateClaimIntegrityCompleted(ctx, latest); err != nil {
			return err
		}
		for _, marker := range app.ConsensusRepairsV1Markers() {
			on, err := marker.Enabled(ctx)
			if err != nil {
				return err
			}
			if !on {
				return fmt.Errorf("consensus repairs target requires %s.%s", marker.Module, marker.Name)
			}
		}
		done, err := app.UpgradeKeeper.GetDoneHeight(ctx, UpgradeNameConsensusRepairsV1)
		if err != nil {
			return err
		}
		raw, present, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(ctx, consensusRepairsReceiptKey)
		if err != nil {
			return err
		}
		if done == 0 && !present {
			return nil
		} // Native/imported selection, never invented history.
		var receipt consensusRepairsReceipt
		if done <= 0 || done > latest || !present || json.Unmarshal([]byte(raw), &receipt) != nil {
			return fmt.Errorf("consensus repairs inconsistent receipt/done height")
		}
		expectedPlan := upgradetypes.Plan{Name: UpgradeNameConsensusRepairsV1, Height: done}
		if receipt.Height != done || receipt.ChainID != ctx.ChainID() || receipt.PlanDigest != consensusRepairsPlanDigest(expectedPlan) || receipt.Census.Height != done {
			return fmt.Errorf("consensus repairs receipt identity mismatch")
		}
		digest, err := hex.DecodeString(receipt.Census.Digest)
		if err != nil || len(digest) != sha256.Size {
			return fmt.Errorf("consensus repairs invalid census digest")
		}
		if len(receipt.Census.Censuses) != len(consensusRepairsCensusNames) {
			return fmt.Errorf("consensus repairs incomplete census summaries")
		}
		for _, name := range consensusRepairsCensusNames {
			summary := receipt.Census.Censuses[name]
			digest, err := hex.DecodeString(summary.Digest)
			if err != nil || len(digest) != sha256.Size {
				return fmt.Errorf("consensus repairs invalid %s census digest", name)
			}
			if summary.Blockers != 0 || summary.HeldFindings < 0 || summary.PendingRollbacks < 0 || summary.Records < 0 || summary.Bytes < 0 || summary.Records > consensusRepairsMaxRecords || summary.Bytes > consensusRepairsMaxBytes {
				return fmt.Errorf("consensus repairs invalid %s census counts or blocker", name)
			}
		}
		markers := app.ConsensusRepairsV1Markers()
		if len(receipt.Markers) != len(markers) {
			return fmt.Errorf("consensus repairs incomplete marker receipt")
		}
		for i, marker := range markers {
			if receipt.Markers[i] != (consensusRepairsMarkerReceipt{marker.Module, marker.Name, true}) {
				return fmt.Errorf("consensus repairs marker receipt mismatch")
			}
		}
		return nil
	}
	if !reflect.DeepEqual(vm, consensusRepairsSourceVersionMap()) {
		return fmt.Errorf("consensus repairs startup requires exact source or target map")
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
		return fmt.Errorf("consensus repairs source startup requires exact on-chain and local H-1 plan")
	}
	_, err = app.validateConsensusRepairsSource(ctx, plan, vm)
	return err
}

type consensusRepairsMarkerLocation struct {
	store string
	key   []byte
}

func consensusRepairsMarkerLocations() []consensusRepairsMarkerLocation {
	return []consensusRepairsMarkerLocation{{"zerone_gov", []byte{0x18}}, {"knowledge", []byte{0x90}}, {"capture_defense", []byte{0x08}}, {"qualification", []byte{0x40}}, {"capture_challenge", []byte{0x05}}, {"vesting_rewards", []byte{0x0c}}, {"zerone_ontology", []byte{0x06}}, {"emergency", []byte{0x14}}, {"substrate_bridge", []byte{0x90}}, {"zerone_ibcrl", []byte{0x04}}}
}
