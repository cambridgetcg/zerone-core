package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

const UpgradeNameToKFeedbackV1 = "tok-feedback-v1"
const tokFeedbackMarker = "upgrade_marker_tok-feedback-v1"
const tokFeedbackMarkerValue = "completed-h3-to-knowledge7-v1"
const ToKFeedbackAcceptedH3Source = "335bb94f0fd54d3752dcb397263b7e84fb1116b4"
const ToKFeedbackAcceptedH3Tree = "769f67f1cfa108be3d31cace7777cf954f731c42"

// Frozen independently of the candidate module manager. These are the full
// H3 output versions from the accepted source, NOT an executable attestation.
var tokFeedbackH3VersionMap = module.VersionMap{
	"06-solomachine": 0, "07-tendermint": 0,
	"alignment": 1, "auth": 5, "bank": 4, "capture_challenge": 1, "capture_defense": 1,
	"claiming_pot": 2, "consensus": 1, "counterexamples": 1, "creed": 1, "distribution": 3,
	"emergency": 2, "evidence": 1, "feegrant": 2, "genutil": 1, "gov": 5, "home": 1,
	"ibc": 8, "ibcratelimit": 1, "interchainaccounts": 3, "knowledge": 6, "liquiditypool": 5,
	"qualification": 1, "slashing": 4, "sponsorship": 1, "staking": 5, "substrate_bridge": 1,
	"tokens": 1, "training_provenance": 1, "transfer": 6, "trust_score": 1, "upgrade": 2,
	"vesting": 1, "vesting_rewards": 2, "work_creed": 1, "zerone_auth": 1, "zerone_gov": 2,
	"zerone_ontology": 1, "zerone_staking": 1,
}

func requireToKFeedbackVersionMap(vm module.VersionMap, knowledge uint64) error {
	if len(vm) != len(tokFeedbackH3VersionMap) {
		return fmt.Errorf("ToK feedback requires exact full H3 version map cardinality")
	}
	names := make([]string, 0, len(tokFeedbackH3VersionMap))
	for name := range tokFeedbackH3VersionMap {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		want := tokFeedbackH3VersionMap[name]
		if name == "knowledge" {
			want = knowledge
		}
		if got, ok := vm[name]; !ok || got != want {
			return fmt.Errorf("ToK feedback module %q requires %d; found=%t version=%d", name, want, ok, got)
		}
	}
	return nil
}

// This source cannot perform an earlier boundary, including H3's formerly broad
// RunMigrations. Refusing before the loader is essential: handler-only rejection
// cannot undo startup store deletion.
func requireToKFeedbackPlan(name string) error {
	if name != UpgradeNameToKFeedbackV1 {
		return fmt.Errorf("tok-feedback-v1 candidate cannot execute %q; use its exact accepted historical binary", name)
	}
	return nil
}

type tokFeedbackPlanInfo struct {
	Schema   string `json:"schema"`
	H3Source string `json:"h3_source"`
	H3Tree   string `json:"h3_tree"`
}

func validateToKFeedbackPlanInfo(info string) error {
	if len(info) > 1024 {
		return fmt.Errorf("ToK feedback plan info too large")
	}
	var p tokFeedbackPlanInfo
	dec := json.NewDecoder(bytes.NewBufferString(info))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		return err
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("trailing ToK feedback plan info")
	}
	if p.Schema != "zerone.tok-feedback-v1/lineage/v1" || p.H3Source != ToKFeedbackAcceptedH3Source || p.H3Tree != ToKFeedbackAcceptedH3Tree {
		return fmt.Errorf("ToK feedback plan must pin accepted H3 source and tree")
	}
	return nil
}

func (app *ZeroneApp) requireToKFeedbackH3(ctx context.Context, activation int64) error {
	height, err := app.requireCompletedPreSDKTransition(ctx, UpgradeNameToKFeedbackV1, UpgradeNameSDK053IBC10, sdk053IBC10UpgradeMarker, sdk053IBC10UpgradeMarkerValue)
	if err != nil {
		return err
	}
	if height >= activation {
		return fmt.Errorf("ToK feedback requires completed H3 strictly before activation")
	}
	_, native, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(ctx, sdk053IBC10NativeMarker)
	if err != nil {
		return err
	}
	if native {
		return fmt.Errorf("ToK feedback requires migrated H3, not native genesis")
	}
	return app.requireMigratedPreSDKTransitionLineage(ctx, UpgradeNameToKFeedbackV1, height, nil)
}

func (app *ZeroneApp) registerToKFeedbackUpgrade() {
	app.UpgradeKeeper.SetUpgradeHandler(UpgradeNameToKFeedbackV1, func(ctx context.Context, plan upgradetypes.Plan, from module.VersionMap) (module.VersionMap, error) {
		if err := validateToKFeedbackPlanInfo(plan.Info); err != nil {
			return nil, err
		}
		if sdk.UnwrapSDKContext(ctx).BlockHeight() != plan.Height {
			return nil, fmt.Errorf("ToK feedback must execute at plan height")
		}
		if err := requireToKFeedbackVersionMap(from, 6); err != nil {
			return nil, err
		}
		if err := app.requireToKFeedbackH3(ctx, plan.Height); err != nil {
			return nil, err
		}
		if err := requireToKFeedbackVersionMap(app.ModuleManager.GetVersionMap(), 7); err != nil {
			return nil, fmt.Errorf("unreviewed candidate target: %w", err)
		}
		if err := app.requireAbsentToKFeedbackCompletion(ctx); err != nil {
			return nil, err
		}
		cache, write := sdk.UnwrapSDKContext(ctx).CacheContext()
		to, err := app.ModuleManager.RunMigrations(cache, app.configurator, from)
		if err != nil {
			return nil, err
		}
		if err := requireToKFeedbackVersionMap(to, 7); err != nil {
			return nil, err
		}
		if err := app.KnowledgeKeeper.WriteMigrationMarker(cache, tokFeedbackMarker, tokFeedbackMarkerValue); err != nil {
			return nil, err
		}
		write()
		return to, nil
	})
}

// A separate post-H3 preflight avoids running the old pre-H3 emergency-state
// reconciler on hardened state. It verifies every mounted committed IAVL root
// and dry-runs the exact named upgrade, but deliberately reports NO_GO: custody,
// accepted executable identity and stopped-state handoff are external gates.
func (app *ZeroneApp) verifyToKFeedbackPrestate(ctx sdk.Context, plan upgradetypes.Plan) (ActivationPreflightReport, error) {
	root := app.CommitMultiStore().LastCommitID()
	if plan.Height != root.Version+1 {
		return ActivationPreflightReport{}, fmt.Errorf("ToK feedback preflight requires exact H-1")
	}
	skips, skipSHA := app.unsafeSkipUpgradeConfig()
	if len(skips) != 0 {
		return ActivationPreflightReport{}, fmt.Errorf("ToK feedback preflight refuses unsafe skip configuration")
	}
	vm, err := app.UpgradeKeeper.GetModuleVersionMap(ctx)
	if err != nil {
		return ActivationPreflightReport{}, err
	}
	if err := app.verifyNamedActivationPreconditions(ctx, plan, vm, root.Version, nil); err != nil {
		return ActivationPreflightReport{}, err
	}
	names := make([]string, 0, len(app.keys))
	for name := range app.keys {
		names = append(names, name)
	}
	sort.Strings(names)
	ids, err := app.activationSubstoreCommitIDs(names)
	if err != nil {
		return ActivationPreflightReport{}, err
	}
	for _, name := range names {
		if _, err := app.collectVerifiedCommittedRecords(name, ids[name], func(_, _ []byte) (bool, error) { return false, nil }); err != nil {
			return ActivationPreflightReport{}, err
		}
	}
	dry, _ := ctx.CacheContext()
	dry = dry.WithBlockHeight(plan.Height)
	if err := app.UpgradeKeeper.ApplyUpgrade(dry, plan); err != nil {
		return ActivationPreflightReport{}, err
	}
	infoSHA := sha256.Sum256([]byte(plan.Info))
	return ActivationPreflightReport{Schema: "zerone.activation-preflight/v5", Scope: "tok-feedback-source-only-h-minus-one", ActivationReady: false, Height: root.Version, AppHash: fmt.Sprintf("%x", root.Hash), UnsafeSkipUpgradeHeights: skips, UnsafeSkipConfigSHA256: skipSHA, CompletedChecks: []string{"complete_iavl_roots_bound_to_app_hash", "exact_completed_h3_lineage_and_version_map", "exact_tok_feedback_handler_cache_dry_run", "external_custody_and_historical_binary_handoff_NOT_verified"}, PlanName: plan.Name, PlanHeight: plan.Height, PlanInfoSHA256: fmt.Sprintf("%x", infoSHA), SafetySourceVersions: vm}, nil
}

func (app *ZeroneApp) requireAbsentToKFeedbackCompletion(ctx context.Context) error {
	for _, name := range []string{tokFeedbackMarker, "migration_v7_complete"} {
		_, found, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(ctx, name)
		if err != nil {
			return err
		}
		if found {
			return fmt.Errorf("unexpected pre-upgrade marker %s", name)
		}
	}
	done, err := app.UpgradeKeeper.GetDoneHeight(ctx, UpgradeNameToKFeedbackV1)
	if err != nil {
		return err
	}
	if done != 0 {
		return fmt.Errorf("unexpected pre-upgrade ToK done height")
	}
	return nil
}

// ValidateToKFeedbackStartupCoordination is additional to, never a replacement
// for, the unmodified completed/native H3 lineage validator. Knowledge=6 can
// only open at the exact committed H-1 with matching on-chain/disk plan. V7
// restarts require both named completion and migration markers. No unsafe skip.
func (app *ZeroneApp) ValidateToKFeedbackStartupCoordination() error {
	if app.activationPreflightReadOnly {
		return nil
	}
	latest := app.CommitMultiStore().LastCommitID().Version
	if latest == 0 {
		return nil
	}
	ctx := app.NewUncachedContext(true, cmtproto.Header{Height: latest})
	vm, err := app.UpgradeKeeper.GetModuleVersionMap(ctx)
	if err != nil {
		return err
	}
	switch vm["knowledge"] {
	case 6:
		if err := requireToKFeedbackVersionMap(vm, 6); err != nil {
			return err
		}
		plan, err := app.UpgradeKeeper.GetUpgradePlan(ctx)
		if err != nil {
			return err
		}
		disk, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
		if err != nil {
			return err
		}
		if plan.Name != UpgradeNameToKFeedbackV1 || plan.Height != latest+1 || plan.Name != disk.Name || plan.Height != disk.Height || plan.Info != disk.Info {
			return fmt.Errorf("ToK feedback requires exact committed and local plan at H-1")
		}
		if app.UpgradeKeeper.IsSkipHeight(plan.Height) {
			return fmt.Errorf("ToK feedback refuses unsafe skip")
		}
		if err := validateToKFeedbackPlanInfo(plan.Info); err != nil {
			return err
		}
		if err := app.requireAbsentToKFeedbackCompletion(ctx); err != nil {
			return err
		}
		return app.requireToKFeedbackH3(ctx, plan.Height)
	case 7:
		if err := requireToKFeedbackVersionMap(vm, 7); err != nil {
			return err
		}
		done, err := app.UpgradeKeeper.GetDoneHeight(ctx, UpgradeNameToKFeedbackV1)
		if err != nil {
			return err
		}
		if done <= 0 || done > latest {
			return fmt.Errorf("knowledge v7 lacks committed ToK feedback completion")
		}
		for _, marker := range []struct{ name, value string }{{tokFeedbackMarker, tokFeedbackMarkerValue}, {"migration_v7_complete", "true"}} {
			v, found, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(ctx, marker.name)
			if err != nil {
				return err
			}
			if !found || v != marker.value {
				return fmt.Errorf("invalid ToK feedback completion marker %s", marker.name)
			}
		}
		return app.requireToKFeedbackH3(ctx, done)
	default:
		return fmt.Errorf("ToK feedback candidate refuses knowledge version %d", vm["knowledge"])
	}
}
