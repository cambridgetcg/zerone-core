package app

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"google.golang.org/protobuf/proto"

	claimingpottypes "github.com/zerone-chain/zerone/x/claiming_pot/types"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	liquiditypooltypes "github.com/zerone-chain/zerone/x/liquiditypool/types"
	vestingrewardstypes "github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

const (
	founderRenunciationNativeLineageMarker = "chain_lineage_native_founder-renunciation-v1"
	founderRenunciationNativeLineageValue  = "genesis"
)

const (
	// These values freeze the complete executable module surface consumed by
	// H3. encoding/json deterministically sorts string map keys.
	FounderRenunciationTargetVersionMapEntries = 40
	FounderRenunciationTargetVersionMapSHA256  = "de3f0e0d9769adf2a7375f921d78f25365bc2f9a8b42d8c80de5982affa20127"
)

const (
	founderRenunciationLineageBootstrap = "uninitialized-bootstrap"
	founderRenunciationLineagePending   = "h2-pending-after-h1"
	founderRenunciationLineageCompleted = "h2-completed-after-h1"
	founderRenunciationLineageNative    = "native-h2-sdk050-genesis"
)

type founderRenunciationStartupEvidence struct {
	latestHeight int64
	versionMap   module.VersionMap

	h1MarkerValue string
	h1MarkerFound bool
	h1NativeValue string
	h1NativeFound bool
	h1DoneHeight  int64

	h2MarkerValue string
	h2MarkerFound bool
	h2NativeValue string
	h2NativeFound bool
	h2DoneHeight  int64

	params *vestingrewardstypes.Params

	vestingAccountFound bool
	vestingPermissions  []string

	onChainPlan      upgradetypes.Plan
	onChainPlanFound bool
	diskPlan         upgradetypes.Plan
	diskPlanFound    bool

	isSkipHeight func(int64) bool
}

// verifyFounderRenunciationStartupLineage replaces the H1-only startup gate.
// It is called after LoadLatestVersion and accepts only four explicit states:
// empty bootstrap, exact H1-complete/H2-pending, exact H2-complete, or a chain
// born natively from this H2 binary. It never writes chain state.
func (app *ZeroneApp) verifyFounderRenunciationStartupLineage() (string, error) {
	evidence, err := app.readFounderRenunciationStartupEvidence()
	if err != nil {
		return "", err
	}
	return validateFounderRenunciationStartupEvidence(
		evidence,
		app.ModuleManager.GetVersionMap(),
	)
}

func (app *ZeroneApp) readFounderRenunciationStartupEvidence() (
	founderRenunciationStartupEvidence,
	error,
) {
	latestHeight := app.LastBlockHeight()
	ctx := app.NewUncachedContext(false, cmtproto.Header{
		Height:  latestHeight,
		ChainID: app.ChainID(),
	})
	return app.readFounderRenunciationEvidence(ctx, latestHeight)
}

func (app *ZeroneApp) readFounderRenunciationEvidence(
	ctx sdk.Context,
	latestHeight int64,
) (evidence founderRenunciationStartupEvidence, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("panic while reading founder-renunciation evidence: %v", recovered)
		}
	}()

	evidence.latestHeight = latestHeight
	evidence.versionMap, err = app.UpgradeKeeper.GetModuleVersionMap(ctx)
	if err != nil {
		return evidence, fmt.Errorf("read committed module VersionMap: %w", err)
	}
	evidence.h1MarkerValue, evidence.h1MarkerFound, err =
		app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(ctx, consolidationMigrationMarker)
	if err != nil {
		return evidence, fmt.Errorf("read H1 migration marker: %w", err)
	}
	evidence.h1NativeValue, evidence.h1NativeFound, err =
		app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(ctx, consolidationNativeLineageMarker)
	if err != nil {
		return evidence, fmt.Errorf("read H1 native lineage marker: %w", err)
	}
	evidence.h1DoneHeight, err =
		app.UpgradeKeeper.GetDoneHeight(ctx, UpgradeNameConsolidationSafetyV1)
	if err != nil {
		return evidence, fmt.Errorf("read H1 done height: %w", err)
	}

	evidence.h2MarkerValue, evidence.h2MarkerFound, err =
		app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(ctx, founderRenunciationMigrationMarker)
	if err != nil {
		return evidence, fmt.Errorf("read H2 migration marker: %w", err)
	}
	evidence.h2NativeValue, evidence.h2NativeFound, err =
		app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(
			ctx,
			founderRenunciationNativeLineageMarker,
		)
	if err != nil {
		return evidence, fmt.Errorf("read H2 native lineage marker: %w", err)
	}
	evidence.h2DoneHeight, err =
		app.UpgradeKeeper.GetDoneHeight(ctx, UpgradeNameFounderRenunciationV1)
	if err != nil {
		return evidence, fmt.Errorf("read H2 done height: %w", err)
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

	// An empty height-zero store is the only state in which params and module
	// account evidence do not yet exist. Every persisted chain must prove them.
	if latestHeight > 0 {
		evidence.params, err = app.VestingRewardsKeeper.GetStoredParamsChecked(ctx)
		if err != nil {
			return evidence, fmt.Errorf("read strict vesting_rewards params: %w", err)
		}
		evidence.vestingAccountFound, evidence.vestingPermissions, err =
			app.readVestingRewardsModuleAccountPermissions(ctx)
		if err != nil {
			return evidence, err
		}
	}

	return evidence, nil
}

func (app *ZeroneApp) readVestingRewardsModuleAccountPermissions(
	ctx sdk.Context,
) (bool, []string, error) {
	address := authtypes.NewModuleAddress(vestingrewardstypes.ModuleName)
	existing := app.AccountKeeper.GetAccount(ctx, address)
	if existing == nil {
		return false, nil, nil
	}
	moduleAccount, ok := existing.(sdk.ModuleAccountI)
	if !ok {
		return false, nil, fmt.Errorf(
			"account at vesting_rewards module address is not a module account",
		)
	}
	if moduleAccount.GetName() != vestingrewardstypes.ModuleName {
		return false, nil, fmt.Errorf(
			"account at vesting_rewards module address has module name %q",
			moduleAccount.GetName(),
		)
	}
	return true, append([]string(nil), moduleAccount.GetPermissions()...), nil
}

func validateFounderRenunciationStartupEvidence(
	evidence founderRenunciationStartupEvidence,
	target module.VersionMap,
) (string, error) {
	if err := validateFounderRenunciationTarget(target); err != nil {
		return "", err
	}
	if evidence.latestHeight < 0 {
		return "", fmt.Errorf("negative committed height %d", evidence.latestHeight)
	}
	if evidence.latestHeight == 0 {
		if len(evidence.versionMap) != 0 {
			return "", fmt.Errorf("uninitialized height 0 has a non-empty module VersionMap")
		}
		if evidence.h1MarkerFound || evidence.h1NativeFound ||
			evidence.h2MarkerFound || evidence.h2NativeFound ||
			evidence.h1DoneHeight != 0 || evidence.h2DoneHeight != 0 ||
			evidence.onChainPlanFound || evidence.diskPlanFound ||
			evidence.params != nil || evidence.vestingAccountFound {
			return "", fmt.Errorf(
				"uninitialized height 0 contains lineage, completion, plan, params, or account evidence",
			)
		}
		return founderRenunciationLineageBootstrap, nil
	}

	pre := founderRenunciationPreVersionMap(target)
	isPre := versionMapsEqual(evidence.versionMap, pre)
	isPost := versionMapsEqual(evidence.versionMap, target)
	if !isPre && !isPost {
		return "", fmt.Errorf(
			"committed module VersionMap is neither exact H2 prestate nor exact H2 poststate: %s",
			versionMapMismatch(evidence.versionMap, pre, target),
		)
	}
	if isPre {
		return validatePendingFounderRenunciationLineage(evidence)
	}
	if evidence.h1NativeFound || evidence.h2NativeFound {
		return validateNativeFounderRenunciationLineage(evidence)
	}
	return validateCompletedFounderRenunciationLineage(evidence)
}

func validateFounderRenunciationTarget(target module.VersionMap) error {
	canonical, err := json.Marshal(target)
	if err != nil {
		return fmt.Errorf("marshal canonical H2 target VersionMap: %w", err)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(canonical))
	if len(target) != FounderRenunciationTargetVersionMapEntries ||
		digest != FounderRenunciationTargetVersionMapSHA256 {
		return fmt.Errorf(
			"binary target violates canonical H2 surface: entries=%d sha256=%s; require entries=%d sha256=%s",
			len(target),
			digest,
			FounderRenunciationTargetVersionMapEntries,
			FounderRenunciationTargetVersionMapSHA256,
		)
	}
	expected := map[string]uint64{
		knowledgetypes.ModuleName:      6,
		claimingpottypes.ModuleName:    2,
		liquiditypooltypes.ModuleName:  5,
		vestingrewardstypes.ModuleName: 2,
	}
	for name, want := range expected {
		got, found := target[name]
		if !found || got != want {
			return fmt.Errorf(
				"binary target violates H1→H2 contract: require %s=%d; got %d (present=%t)",
				name,
				want,
				got,
				found,
			)
		}
	}
	return nil
}

func founderRenunciationPreVersionMap(target module.VersionMap) module.VersionMap {
	pre := make(module.VersionMap, len(target))
	for name, version := range target {
		pre[name] = version
	}
	pre[vestingrewardstypes.ModuleName] = 1
	return pre
}

func validatePendingFounderRenunciationLineage(
	evidence founderRenunciationStartupEvidence,
) (string, error) {
	if evidence.h1MarkerValue != "migrated" || !evidence.h1MarkerFound {
		return "", fmt.Errorf("pending H2 requires the exact H1 migrated marker")
	}
	if evidence.h1NativeFound {
		return "", fmt.Errorf("pending H2 conflicts with a present H1 native marker")
	}
	if evidence.h1DoneHeight <= 0 || evidence.h1DoneHeight > evidence.latestHeight {
		return "", fmt.Errorf(
			"pending H2 requires 0 < H1 done <= latest; got H1=%d latest=%d",
			evidence.h1DoneHeight,
			evidence.latestHeight,
		)
	}
	if evidence.h2MarkerFound {
		return "", fmt.Errorf(
			"pending H2 requires migration marker %q truly absent; found %q",
			founderRenunciationMigrationMarker,
			evidence.h2MarkerValue,
		)
	}
	if evidence.h2NativeFound {
		return "", fmt.Errorf("pending H2 conflicts with a present H2 native marker")
	}
	if evidence.h2DoneHeight != 0 {
		return "", fmt.Errorf("pending H2 requires done height 0; got %d", evidence.h2DoneHeight)
	}
	if err := validateMigratableFounderRenunciationParams(evidence.params); err != nil {
		return "", fmt.Errorf("pending H2 params: %w", err)
	}
	if err := validatePendingFounderRenunciationPermissions(evidence); err != nil {
		return "", fmt.Errorf("pending H2 permissions: %w", err)
	}
	if !evidence.onChainPlanFound {
		return "", fmt.Errorf("exact H2 prestate requires a committed H2 plan")
	}
	if err := validatePendingFounderRenunciationPlan(
		evidence.onChainPlan,
		evidence.latestHeight,
	); err != nil {
		return "", fmt.Errorf("committed H2 plan: %w", err)
	}
	if !evidence.diskPlanFound {
		return "", fmt.Errorf("exact H2 prestate requires local upgrade-info.json")
	}
	if err := validatePendingFounderRenunciationPlan(
		evidence.diskPlan,
		evidence.latestHeight,
	); err != nil {
		return "", fmt.Errorf("local H2 plan: %w", err)
	}
	if !samePlanIdentity(evidence.onChainPlan, evidence.diskPlan) {
		return "", fmt.Errorf(
			"local H2 plan does not exactly match committed name, height, and canonical info",
		)
	}
	if evidence.h1DoneHeight >= evidence.onChainPlan.Height {
		return "", fmt.Errorf(
			"pending H2 requires H1 done before H2; got H1=%d H2=%d",
			evidence.h1DoneHeight,
			evidence.onChainPlan.Height,
		)
	}
	if isUnsafeFounderRenunciationSkip(evidence, evidence.h1DoneHeight) {
		return "", fmt.Errorf("H1 height %d is configured for unsafe skip", evidence.h1DoneHeight)
	}
	if isUnsafeFounderRenunciationSkip(evidence, evidence.onChainPlan.Height) {
		return "", fmt.Errorf(
			"H2 height %d is configured for unsafe skip",
			evidence.onChainPlan.Height,
		)
	}
	return founderRenunciationLineagePending, nil
}

func validateCompletedFounderRenunciationLineage(
	evidence founderRenunciationStartupEvidence,
) (string, error) {
	if evidence.h1MarkerValue != "migrated" || !evidence.h1MarkerFound {
		return "", fmt.Errorf("completed H2 requires the exact H1 migrated marker")
	}
	if evidence.h2MarkerValue != "migrated" || !evidence.h2MarkerFound {
		return "", fmt.Errorf("completed H2 requires the exact H2 migrated marker")
	}
	if evidence.h1NativeFound || evidence.h2NativeFound {
		return "", fmt.Errorf("completed H2 conflicts with a present native lineage marker")
	}
	if evidence.h1DoneHeight <= 0 ||
		evidence.h1DoneHeight >= evidence.h2DoneHeight ||
		evidence.h2DoneHeight > evidence.latestHeight {
		return "", fmt.Errorf(
			"completed H2 requires 0 < H1 done < H2 done <= latest; got H1=%d H2=%d latest=%d",
			evidence.h1DoneHeight,
			evidence.h2DoneHeight,
			evidence.latestHeight,
		)
	}
	if isUnsafeFounderRenunciationSkip(evidence, evidence.h1DoneHeight) {
		return "", fmt.Errorf("H1 height %d is configured for unsafe skip", evidence.h1DoneHeight)
	}
	if isUnsafeFounderRenunciationSkip(evidence, evidence.h2DoneHeight) {
		return "", fmt.Errorf("H2 height %d is configured for unsafe skip", evidence.h2DoneHeight)
	}
	if err := validateRetiredFounderRenunciationParams(evidence.params); err != nil {
		return "", fmt.Errorf("completed H2 params: %w", err)
	}
	if evidence.vestingAccountFound && len(evidence.vestingPermissions) != 0 {
		return "", fmt.Errorf(
			"completed H2 vesting_rewards module account retains permissions %v",
			evidence.vestingPermissions,
		)
	}
	if err := validatePostFounderRenunciationPlans(evidence, false); err != nil {
		return "", err
	}
	return founderRenunciationLineageCompleted, nil
}

func validateNativeFounderRenunciationLineage(
	evidence founderRenunciationStartupEvidence,
) (string, error) {
	if evidence.h1MarkerFound || evidence.h2MarkerFound {
		return "", fmt.Errorf("native H2 lineage conflicts with a present migration marker")
	}
	if !evidence.h1NativeFound ||
		evidence.h1NativeValue != consolidationNativeLineageValue {
		return "", fmt.Errorf("native H2 lineage requires the exact inherited H1 native marker")
	}
	if !evidence.h2NativeFound ||
		evidence.h2NativeValue != founderRenunciationNativeLineageValue {
		return "", fmt.Errorf("native H2 lineage requires the exact H2 native marker")
	}
	if evidence.h1DoneHeight != 0 || evidence.h2DoneHeight != 0 {
		return "", fmt.Errorf(
			"native H2 lineage requires both done heights 0; got H1=%d H2=%d",
			evidence.h1DoneHeight,
			evidence.h2DoneHeight,
		)
	}
	if err := validateRetiredFounderRenunciationParams(evidence.params); err != nil {
		return "", fmt.Errorf("native H2 params: %w", err)
	}
	if evidence.vestingAccountFound && len(evidence.vestingPermissions) != 0 {
		return "", fmt.Errorf(
			"native H2 vesting_rewards module account retains permissions %v",
			evidence.vestingPermissions,
		)
	}
	if err := validatePostFounderRenunciationPlans(evidence, true); err != nil {
		return "", err
	}
	return founderRenunciationLineageNative, nil
}

func validatePostFounderRenunciationPlans(
	evidence founderRenunciationStartupEvidence,
	native bool,
) error {
	lineage := "completed H2"
	if native {
		lineage = "native H2"
	}
	if evidence.onChainPlanFound {
		if err := evidence.onChainPlan.ValidateBasic(); err != nil {
			return fmt.Errorf("malformed committed plan after %s: %w", lineage, err)
		}
		if evidence.onChainPlan.Name == UpgradeNameConsolidationSafetyV1 ||
			evidence.onChainPlan.Name == UpgradeNameFounderRenunciationV1 {
			return fmt.Errorf("%s carries a conflicting H1/H2 migration plan", lineage)
		}
		if evidence.onChainPlan.Height <= evidence.latestHeight {
			return fmt.Errorf(
				"%s has stale committed plan %q at %d (latest %d)",
				lineage,
				evidence.onChainPlan.Name,
				evidence.onChainPlan.Height,
				evidence.latestHeight,
			)
		}
	}
	if native {
		if evidence.diskPlanFound {
			return fmt.Errorf("native H2 lineage does not accept local upgrade-info.json")
		}
		return nil
	}
	if !evidence.diskPlanFound {
		return nil
	}
	if err := evidence.diskPlan.ValidateBasic(); err != nil {
		return fmt.Errorf("malformed historical local H2 plan: %w", err)
	}
	if evidence.diskPlan.Name != UpgradeNameFounderRenunciationV1 ||
		evidence.diskPlan.Height != evidence.h2DoneHeight {
		return fmt.Errorf(
			"local upgrade-info.json conflicts with completed H2 height %d",
			evidence.h2DoneHeight,
		)
	}
	if err := validateCanonicalConsolidationPlanInfo(evidence.diskPlan.Info); err != nil {
		return fmt.Errorf("historical local H2 plan info: %w", err)
	}
	return nil
}

func validatePendingFounderRenunciationPlan(
	plan upgradetypes.Plan,
	latestHeight int64,
) error {
	if err := plan.ValidateBasic(); err != nil {
		return err
	}
	if plan.Name != UpgradeNameFounderRenunciationV1 {
		return fmt.Errorf("require name %q; got %q", UpgradeNameFounderRenunciationV1, plan.Name)
	}
	if plan.Height != latestHeight+1 {
		return fmt.Errorf("require height latest+1=%d; got %d", latestHeight+1, plan.Height)
	}
	return validateCanonicalConsolidationPlanInfo(plan.Info)
}

func validateFounderRenunciationPlanAtHeight(
	plan upgradetypes.Plan,
	height int64,
) error {
	if err := plan.ValidateBasic(); err != nil {
		return err
	}
	if plan.Name != UpgradeNameFounderRenunciationV1 {
		return fmt.Errorf("require name %q; got %q", UpgradeNameFounderRenunciationV1, plan.Name)
	}
	if plan.Height != height {
		return fmt.Errorf("require current height %d; got %d", height, plan.Height)
	}
	return validateCanonicalConsolidationPlanInfo(plan.Info)
}

func validateMigratableFounderRenunciationParams(params *vestingrewardstypes.Params) error {
	if params == nil {
		return fmt.Errorf("strict persisted params are missing")
	}
	migrated := proto.Clone(params).(*vestingrewardstypes.Params)
	migrated.FounderShareBps = 0
	migrated.FounderAddress = ""
	migrated.BlockReward = "0"
	migrated.FloorReward = "0"
	migrated.EmptyBlockRewardRate = 0
	if err := vestingrewardstypes.ValidateParams(migrated); err != nil {
		return fmt.Errorf("legacy params cannot migrate to valid v2 state: %w", err)
	}
	return nil
}

func validateRetiredFounderRenunciationParams(params *vestingrewardstypes.Params) error {
	if params == nil {
		return fmt.Errorf("strict persisted params are missing")
	}
	if err := vestingrewardstypes.ValidateParams(params); err != nil {
		return err
	}
	if params.FounderShareBps != 0 || params.FounderAddress != "" ||
		params.BlockReward != "0" || params.FloorReward != "0" ||
		params.EmptyBlockRewardRate != 0 {
		return fmt.Errorf("retired founder/reward fields are not pinned to zero")
	}
	return nil
}

func validatePendingFounderRenunciationPermissions(
	evidence founderRenunciationStartupEvidence,
) error {
	// H1's deterministic permission reconciliation skips accounts that have
	// never been created. If the vesting account exists, however, its exact H1
	// Minter/Burner set is part of the reviewed prestate; H2 must not silently
	// normalize arbitrary auth drift under its narrower plan.
	if !evidence.vestingAccountFound {
		return nil
	}
	want := []string{authtypes.Minter, authtypes.Burner}
	if !equalStringSets(evidence.vestingPermissions, want) {
		return fmt.Errorf(
			"existing vesting_rewards module account requires exact H1 permissions %v; got %v",
			want,
			evidence.vestingPermissions,
		)
	}
	return nil
}

func isUnsafeFounderRenunciationSkip(
	evidence founderRenunciationStartupEvidence,
	height int64,
) bool {
	return evidence.isSkipHeight != nil && evidence.isSkipHeight(height)
}

func (app *ZeroneApp) handleFounderRenunciationUpgrade(
	ctx context.Context,
	plan upgradetypes.Plan,
	fromVM module.VersionMap,
) (toVM module.VersionMap, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			toVM = nil
			err = fmt.Errorf("panic while proving founder-renunciation upgrade: %v", recovered)
		}
	}()

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	evidence, err := app.readFounderRenunciationEvidence(sdkCtx, sdkCtx.BlockHeight())
	if err != nil {
		return nil, err
	}
	target := app.ModuleManager.GetVersionMap()
	if err := validateFounderRenunciationHandlerEvidence(
		evidence,
		plan,
		fromVM,
		target,
	); err != nil {
		return nil, err
	}

	app.Logger().Info(fmt.Sprintf(
		"applying upgrade %q at height %d",
		plan.Name,
		plan.Height,
	))
	toVM, err = app.ModuleManager.RunMigrations(ctx, app.configurator, fromVM)
	if err != nil {
		return nil, err
	}
	if !versionMapsEqual(toVM, target) {
		return nil, fmt.Errorf(
			"upgrade %q produced invalid full poststate: %s",
			plan.Name,
			versionMapMismatch(toVM, target, target),
		)
	}
	params, err := app.VestingRewardsKeeper.GetStoredParamsChecked(sdkCtx)
	if err != nil {
		return nil, fmt.Errorf("read migrated vesting_rewards params: %w", err)
	}
	if err := validateRetiredFounderRenunciationParams(params); err != nil {
		return nil, fmt.Errorf("invalid migrated vesting_rewards params: %w", err)
	}

	// H2 is scoped to vesting_rewards. Reconcile only that existing account,
	// after every read-only proof and migration but before the completion
	// marker; do not carry unrelated auth-state repair under this plan.
	if err := app.reconcileVestingRewardsModuleAccountPermissions(sdkCtx); err != nil {
		return nil, err
	}
	found, permissions, err := app.readVestingRewardsModuleAccountPermissions(sdkCtx)
	if err != nil {
		return nil, err
	}
	if found && len(permissions) != 0 {
		return nil, fmt.Errorf(
			"vesting_rewards module permissions remain after reconciliation: %v",
			permissions,
		)
	}
	h1Marker, h1Found, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(
		ctx,
		consolidationMigrationMarker,
	)
	if err != nil {
		return nil, fmt.Errorf("re-read H1 migration marker: %w", err)
	}
	if !h1Found || h1Marker != "migrated" {
		return nil, fmt.Errorf("H1 migration marker changed during H2 execution")
	}
	h2Marker, h2Found, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(
		ctx,
		founderRenunciationMigrationMarker,
	)
	if err != nil {
		return nil, fmt.Errorf("re-read H2 migration marker: %w", err)
	}
	if h2Found {
		return nil, fmt.Errorf(
			"H2 migration marker appeared before final write with value %q",
			h2Marker,
		)
	}
	if err := app.KnowledgeKeeper.WriteMigrationMarker(
		ctx,
		founderRenunciationMigrationMarker,
		"migrated",
	); err != nil {
		return nil, err
	}
	return toVM, nil
}

// reconcileVestingRewardsModuleAccountPermissions removes the historical
// Minter/Burner permissions from the one existing account owned by H2. It
// never lazily creates an account and never touches another module.
func (app *ZeroneApp) reconcileVestingRewardsModuleAccountPermissions(
	ctx sdk.Context,
) error {
	address := authtypes.NewModuleAddress(vestingrewardstypes.ModuleName)
	existing := app.AccountKeeper.GetAccount(ctx, address)
	if existing == nil {
		return nil
	}
	moduleAccount, ok := existing.(sdk.ModuleAccountI)
	if !ok {
		return fmt.Errorf("vesting_rewards address is not a module account")
	}
	if moduleAccount.GetName() != vestingrewardstypes.ModuleName {
		return fmt.Errorf(
			"vesting_rewards address has module name %q",
			moduleAccount.GetName(),
		)
	}
	if len(moduleAccount.GetPermissions()) == 0 {
		return nil
	}
	rebuilt := authtypes.NewModuleAccount(
		authtypes.NewBaseAccount(
			moduleAccount.GetAddress(),
			nil,
			moduleAccount.GetAccountNumber(),
			moduleAccount.GetSequence(),
		),
		vestingrewardstypes.ModuleName,
	)
	app.AccountKeeper.SetModuleAccount(ctx, rebuilt)
	app.Logger().Info(
		"retired vesting_rewards module account permissions",
		"was",
		moduleAccount.GetPermissions(),
	)
	return nil
}

func validateFounderRenunciationHandlerEvidence(
	evidence founderRenunciationStartupEvidence,
	plan upgradetypes.Plan,
	fromVM module.VersionMap,
	target module.VersionMap,
) error {
	if err := validateFounderRenunciationTarget(target); err != nil {
		return err
	}
	height := evidence.latestHeight
	if height <= 0 {
		return fmt.Errorf("H2 handler requires a positive current height")
	}
	if err := validateFounderRenunciationPlanAtHeight(plan, height); err != nil {
		return fmt.Errorf("handler H2 plan: %w", err)
	}
	pre := founderRenunciationPreVersionMap(target)
	if !versionMapsEqual(fromVM, pre) {
		return fmt.Errorf(
			"H2 handler requires exact full V1 VersionMap: %s",
			versionMapMismatch(fromVM, pre, target),
		)
	}
	if !versionMapsEqual(evidence.versionMap, pre) ||
		!versionMapsEqual(evidence.versionMap, fromVM) {
		return fmt.Errorf("committed VersionMap does not exactly match handler fromVM")
	}
	if !evidence.h1MarkerFound || evidence.h1MarkerValue != "migrated" {
		return fmt.Errorf("H2 handler requires exact H1 migrated marker")
	}
	if evidence.h1NativeFound || evidence.h2NativeFound {
		return fmt.Errorf("H2 handler refuses native lineage markers")
	}
	if evidence.h1DoneHeight <= 0 || evidence.h1DoneHeight >= height {
		return fmt.Errorf(
			"H2 handler requires 0 < H1 done < H2 height; got H1=%d H2=%d",
			evidence.h1DoneHeight,
			height,
		)
	}
	if evidence.h2MarkerFound {
		return fmt.Errorf(
			"H2 handler requires migration marker truly absent; found %q",
			evidence.h2MarkerValue,
		)
	}
	if evidence.h2DoneHeight != 0 {
		return fmt.Errorf("H2 handler requires done height 0; got %d", evidence.h2DoneHeight)
	}
	if err := validateMigratableFounderRenunciationParams(evidence.params); err != nil {
		return fmt.Errorf("H2 handler params: %w", err)
	}
	if err := validatePendingFounderRenunciationPermissions(evidence); err != nil {
		return fmt.Errorf("H2 handler permissions: %w", err)
	}
	if !evidence.onChainPlanFound {
		return fmt.Errorf("H2 handler requires committed plan evidence")
	}
	if err := validateFounderRenunciationPlanAtHeight(
		evidence.onChainPlan,
		height,
	); err != nil {
		return fmt.Errorf("committed H2 plan: %w", err)
	}
	if !samePlanIdentity(plan, evidence.onChainPlan) {
		return fmt.Errorf("handler plan does not exactly match committed H2 plan")
	}
	if !evidence.diskPlanFound {
		return fmt.Errorf("H2 handler requires local upgrade-info.json")
	}
	if err := validateFounderRenunciationPlanAtHeight(
		evidence.diskPlan,
		height,
	); err != nil {
		return fmt.Errorf("local H2 plan: %w", err)
	}
	if !samePlanIdentity(plan, evidence.diskPlan) {
		return fmt.Errorf("handler plan does not exactly match local H2 plan")
	}
	if isUnsafeFounderRenunciationSkip(evidence, evidence.h1DoneHeight) {
		return fmt.Errorf("H1 height %d is configured for unsafe skip", evidence.h1DoneHeight)
	}
	if isUnsafeFounderRenunciationSkip(evidence, height) {
		return fmt.Errorf("H2 height %d is configured for unsafe skip", height)
	}
	return nil
}
