package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

const (
	consolidationNativeLineageMarker = "chain_lineage_native_consolidation-safety-v1"
	consolidationNativeLineageValue  = "genesis"
	consolidationPlanInfoMaxBytes    = 4096
)

const (
	consolidationLineageBootstrap = "uninitialized-bootstrap"
	consolidationLineagePending   = "h1-pending"
	consolidationLineageCompleted = "h1-completed"
	consolidationLineageNative    = "native-h1-sdk050-genesis"
)

var canonicalJSONInteger = regexp.MustCompile(`^(0|-?[1-9][0-9]*)$`)

type migrationMarkerPresenceReader interface {
	ReadMigrationMarkerPresenceChecked(context.Context, string) (string, bool, error)
}

type upgradeDoneHeightReader interface {
	GetDoneHeight(context.Context, string) (int64, error)
}

// liquidityActivationEvidence adapts the two independent global proof
// stores to the intentionally narrow interface held by x/liquiditypool.
type liquidityActivationEvidence struct {
	markers migrationMarkerPresenceReader
	done    upgradeDoneHeightReader
}

func (e liquidityActivationEvidence) ReadMigrationMarkerPresenceChecked(
	ctx context.Context,
	key string,
) (string, bool, error) {
	return e.markers.ReadMigrationMarkerPresenceChecked(ctx, key)
}

func (e liquidityActivationEvidence) GetDoneHeight(ctx context.Context, name string) (int64, error) {
	return e.done.GetDoneHeight(ctx, name)
}

type consolidationStartupEvidence struct {
	latestHeight int64
	versionMap   module.VersionMap

	h1MarkerValue string
	h1MarkerFound bool
	nativeValue   string
	nativeFound   bool
	doneHeight    int64

	onChainPlan      upgradetypes.Plan
	onChainPlanFound bool
	diskPlan         upgradetypes.Plan
	diskPlanFound    bool

	isSkipHeight func(int64) bool
}

// verifyConsolidationStartupLineage is called only after LoadLatestVersion.
// It reads committed state through an uncached context and never writes. The
// returned classification is useful only for logging; any ambiguous evidence
// is an error and prevents NewZeroneApp from returning an ABCI application.
func (app *ZeroneApp) verifyConsolidationStartupLineage() (string, error) {
	evidence, err := app.readConsolidationStartupEvidence()
	if err != nil {
		return "", err
	}
	return validateConsolidationStartupEvidence(evidence, app.ModuleManager.GetVersionMap())
}

func (app *ZeroneApp) readConsolidationStartupEvidence() (
	evidence consolidationStartupEvidence,
	err error,
) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("panic while reading consolidation startup evidence: %v", recovered)
		}
	}()

	evidence.latestHeight = app.LastBlockHeight()
	ctx := app.NewUncachedContext(false, cmtproto.Header{
		Height:  evidence.latestHeight,
		ChainID: app.ChainID(),
	})
	evidence.versionMap, err = app.UpgradeKeeper.GetModuleVersionMap(ctx)
	if err != nil {
		return evidence, fmt.Errorf("read committed module VersionMap: %w", err)
	}
	evidence.h1MarkerValue, evidence.h1MarkerFound, err =
		app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(ctx, consolidationMigrationMarker)
	if err != nil {
		return evidence, fmt.Errorf("read H1 migration marker: %w", err)
	}
	evidence.nativeValue, evidence.nativeFound, err =
		app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(ctx, consolidationNativeLineageMarker)
	if err != nil {
		return evidence, fmt.Errorf("read native lineage marker: %w", err)
	}
	evidence.doneHeight, err = app.UpgradeKeeper.GetDoneHeight(ctx, UpgradeNameConsolidationSafetyV1)
	if err != nil {
		return evidence, fmt.Errorf("read H1 done height: %w", err)
	}

	evidence.onChainPlan, err = app.UpgradeKeeper.GetUpgradePlan(ctx)
	switch {
	case err == nil:
		evidence.onChainPlanFound = true
	case errors.Is(err, upgradetypes.ErrNoUpgradePlanFound):
		err = nil
	default:
		return evidence, fmt.Errorf("read committed upgrade plan: %w", err)
	}

	evidence.diskPlan, err = app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		return evidence, fmt.Errorf("read local upgrade-info.json: %w", err)
	}
	evidence.diskPlanFound = upgradePlanPresent(evidence.diskPlan)
	evidence.isSkipHeight = app.UpgradeKeeper.IsSkipHeight
	return evidence, nil
}

func validateConsolidationStartupEvidence(
	evidence consolidationStartupEvidence,
	target module.VersionMap,
) (string, error) {
	for _, boundary := range consolidationVersionBoundaries {
		version, found := target[boundary.module]
		if !found || version != boundary.after {
			return "", fmt.Errorf(
				"binary target violates H1 contract: require %s=%d; got %d (present=%t)",
				boundary.module,
				boundary.after,
				version,
				found,
			)
		}
	}
	if evidence.latestHeight < 0 {
		return "", fmt.Errorf("negative committed height %d", evidence.latestHeight)
	}
	if evidence.latestHeight == 0 {
		if len(evidence.versionMap) != 0 {
			return "", fmt.Errorf("uninitialized height 0 has a non-empty module VersionMap")
		}
		if evidence.h1MarkerFound || evidence.nativeFound || evidence.doneHeight != 0 ||
			evidence.onChainPlanFound || evidence.diskPlanFound {
			return "", fmt.Errorf("uninitialized height 0 contains lineage, completion, or plan evidence")
		}
		return consolidationLineageBootstrap, nil
	}

	pre := consolidationPreVersionMap(target)
	isPre := versionMapsEqual(evidence.versionMap, pre)
	isPost := versionMapsEqual(evidence.versionMap, target)
	if !isPre && !isPost {
		return "", fmt.Errorf(
			"committed module VersionMap is neither exact H1 prestate nor exact H1 poststate: %s",
			versionMapMismatch(evidence.versionMap, pre, target),
		)
	}

	if isPre {
		return validatePendingConsolidationLineage(evidence)
	}
	if evidence.h1MarkerFound && evidence.nativeFound {
		return "", fmt.Errorf("post-H1 VersionMap has conflicting migrated and native lineage markers")
	}
	if evidence.h1MarkerFound {
		return validateCompletedConsolidationLineage(evidence)
	}
	if evidence.nativeFound {
		return validateNativeConsolidationLineage(evidence)
	}
	return "", fmt.Errorf("post-H1 VersionMap has no explicit migrated or native lineage marker")
}

func validatePendingConsolidationLineage(evidence consolidationStartupEvidence) (string, error) {
	if evidence.h1MarkerFound {
		return "", fmt.Errorf(
			"pending H1 requires migration marker %q truly absent; found %q",
			consolidationMigrationMarker,
			evidence.h1MarkerValue,
		)
	}
	if evidence.nativeFound {
		return "", fmt.Errorf(
			"pending H1 conflicts with native lineage marker %q=%q",
			consolidationNativeLineageMarker,
			evidence.nativeValue,
		)
	}
	if evidence.doneHeight != 0 {
		return "", fmt.Errorf("pending H1 requires done height 0; got %d", evidence.doneHeight)
	}
	if !evidence.onChainPlanFound {
		return "", fmt.Errorf("exact H1 prestate requires a committed H1 plan")
	}
	if err := validatePendingH1Plan(evidence.onChainPlan, evidence.latestHeight); err != nil {
		return "", fmt.Errorf("committed H1 plan: %w", err)
	}
	if !evidence.diskPlanFound {
		return "", fmt.Errorf("exact H1 prestate requires local upgrade-info.json")
	}
	if err := validatePendingH1Plan(evidence.diskPlan, evidence.latestHeight); err != nil {
		return "", fmt.Errorf("local H1 plan: %w", err)
	}
	if !samePlanIdentity(evidence.onChainPlan, evidence.diskPlan) {
		return "", fmt.Errorf(
			"local H1 plan does not exactly match committed name, height, and canonical info",
		)
	}
	if evidence.isSkipHeight != nil && evidence.isSkipHeight(evidence.onChainPlan.Height) {
		return "", fmt.Errorf("H1 height %d is configured for unsafe skip", evidence.onChainPlan.Height)
	}
	return consolidationLineagePending, nil
}

func validateCompletedConsolidationLineage(evidence consolidationStartupEvidence) (string, error) {
	if evidence.h1MarkerValue != "migrated" {
		return "", fmt.Errorf(
			"completed H1 requires %s=migrated; got %q",
			consolidationMigrationMarker,
			evidence.h1MarkerValue,
		)
	}
	if evidence.nativeFound {
		return "", fmt.Errorf("completed H1 conflicts with a present native lineage marker")
	}
	if evidence.doneHeight <= 0 || evidence.doneHeight > evidence.latestHeight {
		return "", fmt.Errorf(
			"completed H1 requires 0 < done height <= latest; got done=%d latest=%d",
			evidence.doneHeight,
			evidence.latestHeight,
		)
	}
	if evidence.isSkipHeight != nil && evidence.isSkipHeight(evidence.doneHeight) {
		return "", fmt.Errorf("completed H1 height %d is configured for unsafe skip", evidence.doneHeight)
	}
	if evidence.onChainPlanFound {
		if err := evidence.onChainPlan.ValidateBasic(); err != nil {
			return "", fmt.Errorf("malformed pending plan after completed H1: %w", err)
		}
		if evidence.onChainPlan.Name == UpgradeNameConsolidationSafetyV1 {
			return "", fmt.Errorf("completed H1 still has a conflicting committed H1 plan")
		}
		if evidence.onChainPlan.Height <= evidence.latestHeight {
			return "", fmt.Errorf(
				"completed H1 has stale committed plan %q at %d (latest %d)",
				evidence.onChainPlan.Name,
				evidence.onChainPlan.Height,
				evidence.latestHeight,
			)
		}
	}
	// x/upgrade intentionally leaves the old binary's halt packet on disk.
	// Its presence is historical evidence only when it still names this exact
	// completed height and its Info remains canonical. Any other disk packet is
	// ambiguous and refused.
	if evidence.diskPlanFound {
		if err := evidence.diskPlan.ValidateBasic(); err != nil {
			return "", fmt.Errorf("malformed historical local H1 plan: %w", err)
		}
		if evidence.diskPlan.Name != UpgradeNameConsolidationSafetyV1 ||
			evidence.diskPlan.Height != evidence.doneHeight {
			return "", fmt.Errorf(
				"local upgrade-info.json conflicts with completed H1 height %d",
				evidence.doneHeight,
			)
		}
		if err := validateCanonicalConsolidationPlanInfo(evidence.diskPlan.Info); err != nil {
			return "", fmt.Errorf("historical local H1 plan info: %w", err)
		}
	}
	return consolidationLineageCompleted, nil
}

func validateNativeConsolidationLineage(evidence consolidationStartupEvidence) (string, error) {
	if evidence.nativeValue != consolidationNativeLineageValue {
		return "", fmt.Errorf(
			"native lineage marker %q has invalid value %q",
			consolidationNativeLineageMarker,
			evidence.nativeValue,
		)
	}
	if evidence.h1MarkerFound {
		return "", fmt.Errorf("native lineage conflicts with a present H1 migration marker")
	}
	if evidence.doneHeight != 0 {
		return "", fmt.Errorf("native lineage requires H1 done height 0; got %d", evidence.doneHeight)
	}
	if evidence.onChainPlanFound {
		if err := evidence.onChainPlan.ValidateBasic(); err != nil {
			return "", fmt.Errorf("malformed committed plan on native lineage: %w", err)
		}
		if evidence.onChainPlan.Name == UpgradeNameConsolidationSafetyV1 {
			return "", fmt.Errorf("native lineage cannot carry a committed H1 migration plan")
		}
		if evidence.onChainPlan.Height <= evidence.latestHeight {
			return "", fmt.Errorf("native lineage has a stale committed upgrade plan")
		}
	}
	if evidence.diskPlanFound {
		return "", fmt.Errorf(
			"native lineage does not accept any local upgrade-info.json packet; a replacement binary must verify its own halted upgrade",
		)
	}
	return consolidationLineageNative, nil
}

func validatePendingH1Plan(plan upgradetypes.Plan, latestHeight int64) error {
	if err := plan.ValidateBasic(); err != nil {
		return err
	}
	if plan.Name != UpgradeNameConsolidationSafetyV1 {
		return fmt.Errorf("require name %q; got %q", UpgradeNameConsolidationSafetyV1, plan.Name)
	}
	if plan.Height != latestHeight+1 {
		return fmt.Errorf("require height latest+1=%d; got %d", latestHeight+1, plan.Height)
	}
	return validateCanonicalConsolidationPlanInfo(plan.Info)
}

// validateCanonicalConsolidationPlanInfo prevents a local halt packet and
// committed plan from being interpreted through different JSON spellings.
// It defines no release fields and proves no artifact identity; a later
// signed release packet must do that. Info is public consensus metadata and
// must never contain secrets.
func validateCanonicalConsolidationPlanInfo(info string) error {
	if len(info) == 0 {
		return fmt.Errorf("info must be a non-empty canonical JSON object")
	}
	if len(info) > consolidationPlanInfoMaxBytes {
		return fmt.Errorf("info exceeds %d-byte limit", consolidationPlanInfoMaxBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader([]byte(info)))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("multiple JSON values are not allowed")
		}
		return fmt.Errorf("decode trailing JSON: %w", err)
	}
	object, ok := value.(map[string]any)
	if !ok || len(object) == 0 {
		return fmt.Errorf("info must be a JSON object with at least one key")
	}
	if err := validateCanonicalJSONNumbers(value); err != nil {
		return err
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("re-encode canonical JSON: %w", err)
	}
	if !bytes.Equal(canonical, []byte(info)) {
		return fmt.Errorf("info is not canonical compact sorted-key JSON")
	}
	return nil
}

func validateCanonicalJSONNumbers(value any) error {
	switch typed := value.(type) {
	case json.Number:
		if !canonicalJSONInteger.MatchString(typed.String()) {
			return fmt.Errorf("JSON number %q is not a canonical integer", typed.String())
		}
	case []any:
		for _, item := range typed {
			if err := validateCanonicalJSONNumbers(item); err != nil {
				return err
			}
		}
	case map[string]any:
		for _, item := range typed {
			if err := validateCanonicalJSONNumbers(item); err != nil {
				return err
			}
		}
	}
	return nil
}

func consolidationPreVersionMap(target module.VersionMap) module.VersionMap {
	pre := make(module.VersionMap, len(target))
	for name, version := range target {
		pre[name] = version
	}
	for _, boundary := range consolidationVersionBoundaries {
		pre[boundary.module] = boundary.before
	}
	return pre
}

func versionMapsEqual(got, want module.VersionMap) bool {
	if len(got) != len(want) {
		return false
	}
	for name, wantVersion := range want {
		gotVersion, found := got[name]
		if !found || gotVersion != wantVersion {
			return false
		}
	}
	return true
}

func versionMapMismatch(got, pre, post module.VersionMap) string {
	for _, name := range sortedVersionMapNames(got) {
		if _, inPre := pre[name]; !inPre {
			return fmt.Sprintf("unknown entry %s=%d", name, got[name])
		}
	}
	for _, name := range sortedVersionMapNames(pre) {
		gotVersion, found := got[name]
		if !found {
			return fmt.Sprintf("missing entry %s", name)
		}
		if gotVersion != pre[name] && gotVersion != post[name] {
			return fmt.Sprintf(
				"entry %s=%d is neither pre=%d nor post=%d",
				name,
				gotVersion,
				pre[name],
				post[name],
			)
		}
	}
	return "entries form a forbidden partial pre/post mixture"
}

func samePlanIdentity(left, right upgradetypes.Plan) bool {
	return left.Name == right.Name && left.Height == right.Height && left.Info == right.Info
}

func upgradePlanPresent(plan upgradetypes.Plan) bool {
	return plan.Name != "" || plan.Height != 0 || plan.Info != "" ||
		!plan.Time.IsZero() || plan.UpgradedClientState != nil
}
