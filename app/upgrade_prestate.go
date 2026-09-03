package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"sort"
	"time"

	"cosmossdk.io/collections"
	storeiavl "cosmossdk.io/store/iavl"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	sdkgovtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/cosmos/iavl"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"

	claimingpottypes "github.com/zerone-chain/zerone/x/claiming_pot/types"
	emergencykeeper "github.com/zerone-chain/zerone/x/emergency/keeper"
	emergencytypes "github.com/zerone-chain/zerone/x/emergency/types"
	zeronegovtypes "github.com/zerone-chain/zerone/x/gov/types"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	liquiditypooltypes "github.com/zerone-chain/zerone/x/liquiditypool/types"
	vestingrewardstypes "github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

const (
	maxSelectedPrestateRecordCount = 100_000
	maxSelectedPrestateRecordBytes = 64 << 20
)

type upgradePrestateRecord struct {
	Key   []byte
	Value []byte
}

type verifiedActivationPrestate struct {
	EmergencySnapshot         emergencykeeper.OperationsSafetySnapshot
	SDKGovExecutableProposals []upgradePrestateRecord
	CustomGovLIPs             []upgradePrestateRecord
	CustomGovStake            customUpgradeStakeSummary
}

type customUpgradeStakeEntry struct {
	LIPID      string `json:"lip_id"`
	StakedUzrn string `json:"staked_uzrn"`
}

type customUpgradeStakeSummary struct {
	Entries        []customUpgradeStakeEntry
	TotalUzrn      string
	ManifestSHA256 string
}

// ActivationPreflightReport is the stable JSON result of running the exact
// activation-time complete-IAVL verifier against a stopped node or copied
// application database.
type ActivationPreflightReport struct {
	Schema                          string            `json:"schema"`
	Scope                           string            `json:"scope"`
	ActivationReady                 bool              `json:"activation_ready"`
	ChainID                         string            `json:"chain_id,omitempty"`
	GenesisSHA256                   string            `json:"genesis_sha256,omitempty"`
	UpgradeInfoSHA256               string            `json:"upgrade_info_sha256,omitempty"`
	ReportSHA256                    string            `json:"report_sha256,omitempty"`
	PlanName                        string            `json:"plan_name,omitempty"`
	PlanHeight                      int64             `json:"plan_height,omitempty"`
	PlanInfoSHA256                  string            `json:"plan_info_sha256,omitempty"`
	BlocksUntilActivation           int64             `json:"blocks_until_activation,omitempty"`
	UnsafeSkipUpgradeHeights        []int64           `json:"unsafe_skip_upgrade_heights"`
	UnsafeSkipConfigSHA256          string            `json:"unsafe_skip_config_sha256"`
	SourceVersionMapSHA256          string            `json:"source_version_map_sha256,omitempty"`
	H2PlanIdentitySHA256            string            `json:"h2_plan_identity_sha256,omitempty"`
	SourceDataManifestSHA256        string            `json:"source_data_manifest_sha256,omitempty"`
	SourceDataFileCount             int               `json:"source_data_file_count,omitempty"`
	SourceDataBytes                 int64             `json:"source_data_bytes,omitempty"`
	CompletedChecks                 []string          `json:"completed_checks"`
	Height                          int64             `json:"height"`
	AppHash                         string            `json:"app_hash"`
	SafetySourceVersions            map[string]uint64 `json:"safety_source_versions"`
	EmergencySnapshotRecords        int               `json:"emergency_snapshot_records"`
	EmergencySnapshotBytes          int               `json:"emergency_snapshot_bytes"`
	SDKGovExecutableProposalRecords int               `json:"sdk_gov_executable_proposal_records"`
	SDKGovExecutableProposalBytes   int               `json:"sdk_gov_executable_proposal_bytes"`
	CustomGovUpgradeLIPRecords      int               `json:"custom_gov_upgrade_lip_records"`
	CustomGovUpgradeLIPBytes        int               `json:"custom_gov_upgrade_lip_bytes"`
	CustomGovUpgradeLIPIDs          []string          `json:"custom_gov_upgrade_lip_ids"`
	CustomGovUnattributedStakeUzrn  string            `json:"custom_gov_unattributed_stake_uzrn"`
	CustomGovStakeManifestSHA256    string            `json:"custom_gov_stake_manifest_sha256"`
	CompleteIAVLVerificationMillis  int64             `json:"complete_iavl_verification_millis"`
}

type prestateRecordSelector func(key, value []byte) (bool, error)

type committedIAVLExporter interface {
	LastCommitID() storetypes.CommitID
	Export(version int64) (*iavl.Exporter, error)
}

type rootCommitInfoReader interface {
	LastCommitID() storetypes.CommitID
	GetCommitInfo(version int64) (*storetypes.CommitInfo, error)
}

type exportedNodeSummary struct {
	hash   [sha256.Size]byte
	height int8
	size   int64
	minKey []byte
	maxKey []byte
}

// VerifyActivationPrestate runs the plan-independent committed-state checks.
// Its report deliberately has activation_ready=false; operators should use
// VerifyScheduledActivationPrestate (the CLI default) for a complete H-1 gate.
func (app *ZeroneApp) VerifyActivationPrestate() (
	ActivationPreflightReport,
	error,
) {
	return app.verifyActivationPrestate(false)
}

// VerifyScheduledActivationPrestate verifies the exact scheduled named plan at
// H-1, including every plan.Info, source-lineage, balance, and legacy-store
// precondition enforced by its activation handler.
func (app *ZeroneApp) VerifyScheduledActivationPrestate() (
	ActivationPreflightReport,
	error,
) {
	return app.verifyActivationPrestate(true)
}

func (app *ZeroneApp) verifyActivationPrestate(
	requireScheduledPlan bool,
) (
	ActivationPreflightReport,
	error,
) {
	started := time.Now()
	prestate, err := app.collectAndVerifyActivationPrestate()
	if err != nil {
		return ActivationPreflightReport{}, err
	}
	rootStore, ok := app.CommitMultiStore().(rootCommitInfoReader)
	if !ok {
		return ActivationPreflightReport{}, fmt.Errorf(
			"root commit store has type %T, not the required CommitInfo reader",
			app.CommitMultiStore(),
		)
	}
	rootID := rootStore.LastCommitID()
	ctx := app.NewUncachedContext(
		true,
		cmtproto.Header{Height: rootID.Version},
	)
	versionMap, err := app.UpgradeKeeper.GetModuleVersionMap(ctx)
	if err != nil {
		return ActivationPreflightReport{}, fmt.Errorf(
			"read source module version map: %w",
			err,
		)
	}
	if err := requireActivationSafetySourceVersions(
		"activation-preflight",
		versionMap,
	); err != nil {
		return ActivationPreflightReport{}, err
	}
	if err := app.ensureNoActiveEmergencyGovProposals(
		ctx,
		prestate.SDKGovExecutableProposals,
	); err != nil {
		return ActivationPreflightReport{}, fmt.Errorf(
			"activation-preflight SDK governance authority audit failed: %w",
			err,
		)
	}
	if err := ensureNoUnattributedCustomUpgradeStake(
		prestate.CustomGovStake,
	); err != nil {
		return ActivationPreflightReport{}, fmt.Errorf(
			"activation-preflight: %w",
			err,
		)
	}
	safetySourceVersions := map[string]uint64{
		emergencytypes.ModuleName: versionMap[emergencytypes.ModuleName],
		sdkgovtypes.ModuleName:    versionMap[sdkgovtypes.ModuleName],
		zeronegovtypes.ModuleName: versionMap[zeronegovtypes.ModuleName],
	}
	emergencyBytes := 0
	for _, record := range prestate.EmergencySnapshot.Records {
		emergencyBytes += len(record.Key) + len(record.Value)
	}
	customGovIDs := make([]string, len(prestate.CustomGovStake.Entries))
	for i, entry := range prestate.CustomGovStake.Entries {
		customGovIDs[i] = entry.LIPID
	}
	skipHeights, skipConfigSHA256 := app.unsafeSkipUpgradeConfig()
	report := ActivationPreflightReport{
		Schema:                   "zerone.activation-preflight/v5",
		Scope:                    "common-safety",
		ActivationReady:          false,
		UnsafeSkipUpgradeHeights: skipHeights,
		UnsafeSkipConfigSHA256:   skipConfigSHA256,
		CompletedChecks: []string{
			"complete_iavl_roots_bound_to_app_hash",
			"exact_safety_source_versions",
			"sdk_governance_emergency_authority_audit",
			"zero_unattributed_custom_upgrade_stake",
			"effective_unsafe_skip_configuration_bound",
		},
		Height:                          rootID.Version,
		AppHash:                         fmt.Sprintf("%x", rootID.Hash),
		SafetySourceVersions:            safetySourceVersions,
		EmergencySnapshotRecords:        len(prestate.EmergencySnapshot.Records),
		EmergencySnapshotBytes:          emergencyBytes,
		SDKGovExecutableProposalRecords: len(prestate.SDKGovExecutableProposals),
		SDKGovExecutableProposalBytes: recordSetBytes(
			prestate.SDKGovExecutableProposals,
		),
		CustomGovUpgradeLIPRecords:     len(prestate.CustomGovLIPs),
		CustomGovUpgradeLIPBytes:       recordSetBytes(prestate.CustomGovLIPs),
		CustomGovUpgradeLIPIDs:         customGovIDs,
		CustomGovUnattributedStakeUzrn: prestate.CustomGovStake.TotalUzrn,
		CustomGovStakeManifestSHA256:   prestate.CustomGovStake.ManifestSHA256,
		CompleteIAVLVerificationMillis: time.Since(started).Milliseconds(),
	}
	if !requireScheduledPlan {
		return report, nil
	}

	plan, err := app.UpgradeKeeper.GetUpgradePlan(ctx)
	if err != nil {
		return ActivationPreflightReport{}, fmt.Errorf(
			"read scheduled activation plan: %w",
			err,
		)
	}
	if err := plan.ValidateBasic(); err != nil {
		return ActivationPreflightReport{}, fmt.Errorf(
			"scheduled activation plan is invalid: %w",
			err,
		)
	}
	if plan.Height != rootID.Version+1 {
		return ActivationPreflightReport{}, fmt.Errorf(
			"scheduled plan %q activates at height %d but committed preflight height is %d; run the final readiness gate against exact H-1",
			plan.Name,
			plan.Height,
			rootID.Version,
		)
	}
	if app.UpgradeKeeper.IsSkipHeight(plan.Height) {
		return ActivationPreflightReport{}, fmt.Errorf(
			"scheduled plan %q at height %d is configured in --unsafe-skip-upgrades; activation readiness is false",
			plan.Name,
			plan.Height,
		)
	}
	var lineageEvidence preSDKTransitionLineageEvidence
	if err := app.verifyNamedActivationPreconditions(
		ctx,
		plan,
		versionMap,
		rootID.Version,
		&lineageEvidence,
	); err != nil {
		return ActivationPreflightReport{}, err
	}
	dryRunCtx, _ := ctx.CacheContext()
	dryRunCtx = dryRunCtx.WithBlockHeight(plan.Height)
	if err := app.UpgradeKeeper.ApplyUpgrade(
		context.WithValue(
			dryRunCtx,
			sdk053IBC10PreflightDryRunContextKey{},
			true,
		),
		plan,
	); err != nil {
		return ActivationPreflightReport{}, fmt.Errorf(
			"scheduled upgrade %q exact handler dry-run failed: %w",
			plan.Name,
			err,
		)
	}

	infoDigest := sha256.Sum256([]byte(plan.Info))
	report.Scope = "scheduled-plan-h-minus-one"
	report.ActivationReady = true
	report.PlanName = plan.Name
	report.PlanHeight = plan.Height
	report.PlanInfoSHA256 = fmt.Sprintf("%x", infoDigest)
	report.SourceVersionMapSHA256 = sdk053IBC10SourceVersionMapSHA256
	report.H2PlanIdentitySHA256 = lineageEvidence.h2PlanIdentitySHA256
	report.BlocksUntilActivation = plan.Height - rootID.Version
	report.CompletedChecks = append(
		report.CompletedChecks,
		"scheduled_plan_exact_h_minus_one",
		"named_handler_plan_specific_preconditions",
		"exact_full_source_module_version_map",
		"ordered_h1_h2_marker_and_done_height_proofs",
		"h2_plan_identity_state_evidence",
		"founder_renunciation_zero_poststate",
		"scheduled_height_not_unsafe_skipped",
		"exact_upgrade_handler_cache_dry_run",
	)
	return report, nil
}

func (app *ZeroneApp) unsafeSkipUpgradeConfig() ([]int64, string) {
	heights := make([]int64, 0, len(app.unsafeSkipUpgradeHeights))
	for height, skipped := range app.unsafeSkipUpgradeHeights {
		if skipped {
			heights = append(heights, height)
		}
	}
	sort.Slice(heights, func(i, j int) bool {
		return heights[i] < heights[j]
	})
	hasher := sha256.New()
	_, _ = hasher.Write([]byte("zerone/unsafe-skip-upgrades/v1\x00"))
	var scalar [8]byte
	binary.BigEndian.PutUint64(scalar[:], uint64(len(heights)))
	_, _ = hasher.Write(scalar[:])
	for _, height := range heights {
		binary.BigEndian.PutUint64(scalar[:], uint64(height))
		_, _ = hasher.Write(scalar[:])
	}
	return heights, fmt.Sprintf("%x", hasher.Sum(nil))
}

func (app *ZeroneApp) verifyNamedActivationPreconditions(
	ctx sdk.Context,
	plan upgradetypes.Plan,
	versionMap map[string]uint64,
	committedHeight int64,
	lineageEvidence *preSDKTransitionLineageEvidence,
) error {
	switch plan.Name {
	case UpgradeNameSDK053IBC10:
		manifest, err := parseSDK053IBC10PlanInfo(plan.Info)
		if err != nil {
			return fmt.Errorf(
				"scheduled upgrade %q has invalid plan info: %w",
				plan.Name,
				err,
			)
		}
		if err := requireSDK053IBC10SourceVersions(plan.Name, versionMap); err != nil {
			return err
		}
		if err := app.requirePreSDKTransitionLineage(
			ctx,
			plan.Name,
			plan.Height,
			versionMap,
			lineageEvidence,
		); err != nil {
			return err
		}
		feeBalances := app.BankKeeper.GetAllBalances(
			ctx,
			authtypes.NewModuleAddress(legacyIBCFeeStoreKey),
		)
		if !feeBalances.IsZero() {
			return fmt.Errorf(
				"scheduled upgrade %q cannot remove legacy IBC fee middleware while module account %q holds %s",
				plan.Name,
				legacyIBCFeeStoreKey,
				feeBalances,
			)
		}
		if app.sdk053IBC10LegacyStoresAtStartup[legacyScheduleStoreKey] {
			return fmt.Errorf(
				"scheduled upgrade %q refuses committed retired scheduler store %q; reconcile old-format liabilities first",
				plan.Name,
				legacyScheduleStoreKey,
			)
		}
		if err := app.requireNoLegacyScheduleBalance(ctx, "scheduled upgrade "+plan.Name); err != nil {
			return err
		}
		if err := app.requireFreshScheduleBalanceZero(ctx, "scheduled upgrade "+plan.Name); err != nil {
			return err
		}
		if _, err := app.verifyObsoleteIBCChannelState(manifest); err != nil {
			return fmt.Errorf(
				"scheduled upgrade %q failed to verify obsolete IBC v8 channel state: %w",
				plan.Name,
				err,
			)
		}
		legacyCapabilityKey, capabilityMounted :=
			app.keys[legacyCapabilityStoreKey]
		legacyFeeKey, feeMounted := app.keys[legacyIBCFeeStoreKey]
		if !capabilityMounted || legacyCapabilityKey == nil ||
			!feeMounted || legacyFeeKey == nil {
			return fmt.Errorf(
				"scheduled upgrade %q preflight requires both read-only legacy store mounts %q and %q",
				plan.Name,
				legacyCapabilityStoreKey,
				legacyIBCFeeStoreKey,
			)
		}
		if _, err := app.activationSubstoreCommitIDs([]string{
			legacyCapabilityStoreKey,
			legacyIBCFeeStoreKey,
		}); err != nil {
			return fmt.Errorf(
				"scheduled upgrade %q cannot prove both legacy roots in H-1 CommitInfo: %w",
				plan.Name,
				err,
			)
		}
		if err := rejectLegacyIBCFeeLock(
			app.CommitMultiStore().GetCommitKVStore(legacyFeeKey),
			committedHeight,
		); err != nil {
			return err
		}
		app.sdk053IBC10LoaderProof = &sdk053IBC10StoreLoaderProof{
			upgradeHeight:       plan.Height,
			preUpgradeVersion:   committedHeight,
			legacyRootsComplete: true,
			feeLockAbsent:       true,
			preflightOnly:       true,
		}
	default:
		return fmt.Errorf(
			"scheduled plan %q is not supported by the guarded activation preflight",
			plan.Name,
		)
	}
	return nil
}

const (
	consolidationSafetyMarker             = "upgrade_marker_consolidation-safety-v1"
	consolidationSafetyMarkerValue        = "migrated"
	consolidationSafetyNativeMarker       = "chain_lineage_native_consolidation-safety-v1"
	founderRenunciationMarker             = "upgrade_marker_founder-renunciation-v1"
	founderRenunciationMarkerValue        = "migrated"
	founderRenunciationNativeMarker       = "chain_lineage_native_founder-renunciation-v1"
	founderRenunciationPlanIdentityMarker = "upgrade_plan_identity_founder-renunciation-v1"
	sdk053IBC10SourceVersionMapSHA256     = "de3f0e0d9769adf2a7375f921d78f25365bc2f9a8b42d8c80de5982affa20127"
)

type preSDKTransitionLineageEvidence struct {
	h1DoneHeight         int64
	h2DoneHeight         int64
	h2PlanIdentitySHA256 string
}

type sdk053IBC10SourceModuleVersion struct {
	name    string
	version uint64
}

// sdk053IBC10ExactSourceVersionMap is the complete module VersionMap emitted
// by the reviewed H2 pre-SDK release. It is intentionally a frozen list rather
// than a clone of this binary's target map: deriving the source from H3 would
// let a later module addition or version bump silently ride the H3 plan.
var sdk053IBC10ExactSourceVersionMap = []sdk053IBC10SourceModuleVersion{
	{name: "alignment", version: 1},
	{name: "auth", version: 5},
	{name: "bank", version: 4},
	{name: legacyCapabilityStoreKey, version: 1},
	{name: "capture_challenge", version: 1},
	{name: "capture_defense", version: 1},
	{name: claimingpottypes.ModuleName, version: 2},
	{name: "consensus", version: 1},
	{name: "counterexamples", version: 1},
	{name: "creed", version: 1},
	{name: "distribution", version: 3},
	{name: emergencytypes.ModuleName, version: 1},
	{name: "evidence", version: 1},
	{name: "feegrant", version: 2},
	{name: legacyIBCFeeStoreKey, version: 2},
	{name: "genutil", version: 1},
	{name: sdkgovtypes.ModuleName, version: 5},
	{name: "home", version: 1},
	{name: ibcexported.ModuleName, version: 6},
	{name: "ibcratelimit", version: 1},
	{name: icatypes.ModuleName, version: 3},
	{name: knowledgetypes.ModuleName, version: 6},
	{name: liquiditypooltypes.ModuleName, version: 5},
	{name: "qualification", version: 1},
	{name: "slashing", version: 4},
	{name: "sponsorship", version: 1},
	{name: "staking", version: 5},
	{name: "substrate_bridge", version: 1},
	{name: "tokens", version: 1},
	{name: "training_provenance", version: 1},
	{name: ibctransfertypes.ModuleName, version: 5},
	{name: "trust_score", version: 1},
	{name: upgradetypes.ModuleName, version: 2},
	{name: "vesting", version: 1},
	{name: vestingrewardstypes.ModuleName, version: 2},
	{name: "work_creed", version: 1},
	{name: "zerone_auth", version: 1},
	{name: zeronegovtypes.ModuleName, version: 2},
	{name: "zerone_ontology", version: 1},
	{name: "zerone_staking", version: 1},
}

// requirePreSDKTransitionLineage consumes two distinct state proofs before H3:
// exact H2 output bytes as represented by the full VersionMap, plus the
// checked marker and immutable x/upgrade done height for each named H1/H2
// transition. These state facts prove ordered named-plan completion; they do
// not cryptographically attest which executable produced the state. Release
// source/binary provenance remains a separate operator verification.
func (app *ZeroneApp) requirePreSDKTransitionLineage(
	ctx context.Context,
	planName string,
	activationHeight int64,
	fromVM map[string]uint64,
	evidence *preSDKTransitionLineageEvidence,
) error {
	if err := requireSDK053IBC10ExactSourceVersionMap(planName, fromVM); err != nil {
		return err
	}
	if err := app.requireAbsentH3TransitionEvidence(ctx, planName); err != nil {
		return err
	}
	return app.requireMigratedPreSDKTransitionLineage(
		ctx,
		planName,
		activationHeight,
		evidence,
	)
}

func (app *ZeroneApp) requireMigratedPreSDKTransitionLineage(
	ctx context.Context,
	planName string,
	activationHeight int64,
	evidence *preSDKTransitionLineageEvidence,
) error {
	if activationHeight <= 0 {
		return fmt.Errorf(
			"upgrade %q requires a positive H3 activation height: got %d",
			planName,
			activationHeight,
		)
	}
	if err := app.requireAbsentPreSDKNativeLineage(ctx, planName); err != nil {
		return err
	}

	h1DoneHeight, err := app.requireCompletedPreSDKTransition(
		ctx,
		planName,
		UpgradeNameConsolidationSafetyV1,
		consolidationSafetyMarker,
		consolidationSafetyMarkerValue,
	)
	if err != nil {
		return err
	}
	h2DoneHeight, err := app.requireCompletedPreSDKTransition(
		ctx,
		planName,
		UpgradeNameFounderRenunciationV1,
		founderRenunciationMarker,
		founderRenunciationMarkerValue,
	)
	if err != nil {
		return err
	}
	h2PlanIdentitySHA256, err := app.requireFounderRenunciationPlanIdentity(
		ctx,
		planName,
	)
	if err != nil {
		return err
	}
	if h1DoneHeight >= h2DoneHeight || h2DoneHeight >= activationHeight {
		return fmt.Errorf(
			"upgrade %q requires ordered activation heights H1(%q) < H2(%q) < H3(%q): got %d, %d, %d",
			planName,
			UpgradeNameConsolidationSafetyV1,
			UpgradeNameFounderRenunciationV1,
			UpgradeNameSDK053IBC10,
			h1DoneHeight,
			h2DoneHeight,
			activationHeight,
		)
	}
	if err := app.requireFounderRenunciationZeroPoststate(ctx, planName); err != nil {
		return err
	}
	if evidence != nil {
		*evidence = preSDKTransitionLineageEvidence{
			h1DoneHeight:         h1DoneHeight,
			h2DoneHeight:         h2DoneHeight,
			h2PlanIdentitySHA256: h2PlanIdentitySHA256,
		}
	}
	return nil
}

func (app *ZeroneApp) requireAbsentH3TransitionEvidence(
	ctx context.Context,
	planName string,
) error {
	for _, marker := range []string{
		sdk053IBC10UpgradeMarker,
		sdk053IBC10NativeMarker,
	} {
		value, found, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(
			ctx,
			marker,
		)
		if err != nil {
			return fmt.Errorf(
				"upgrade %q cannot verify H3 marker %q absence before transition seal: %w",
				planName,
				marker,
				err,
			)
		}
		if found {
			return fmt.Errorf(
				"upgrade %q requires pre-H3 marker %q to be truly absent: found value %q",
				planName,
				marker,
				value,
			)
		}
	}
	doneHeight, err := app.UpgradeKeeper.GetDoneHeight(ctx, UpgradeNameSDK053IBC10)
	if err != nil {
		return fmt.Errorf(
			"upgrade %q cannot verify pre-H3 done height: %w",
			planName,
			err,
		)
	}
	if doneHeight != 0 {
		return fmt.Errorf(
			"upgrade %q requires pre-H3 done height exactly 0: got %d",
			planName,
			doneHeight,
		)
	}
	return nil
}

func (app *ZeroneApp) requireAbsentPreSDKNativeLineage(
	ctx context.Context,
	planName string,
) error {
	for _, prerequisite := range []struct {
		upgrade string
		marker  string
	}{
		{
			upgrade: UpgradeNameConsolidationSafetyV1,
			marker:  consolidationSafetyNativeMarker,
		},
		{
			upgrade: UpgradeNameFounderRenunciationV1,
			marker:  founderRenunciationNativeMarker,
		},
	} {
		value, found, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(
			ctx,
			prerequisite.marker,
		)
		if err != nil {
			return fmt.Errorf(
				"upgrade %q cannot verify prerequisite %q native lineage marker %q absence: %w",
				planName,
				prerequisite.upgrade,
				prerequisite.marker,
				err,
			)
		}
		if found {
			return fmt.Errorf(
				"upgrade %q requires migrated prerequisite %q native lineage marker %q to be truly absent: found value %q",
				planName,
				prerequisite.upgrade,
				prerequisite.marker,
				value,
			)
		}
	}
	return nil
}

func (app *ZeroneApp) requireNativeH3LineagePoststate(
	ctx context.Context,
	planName string,
) error {
	if err := app.requireNoLegacyScheduleBalance(ctx, planName+" native lineage"); err != nil {
		return err
	}
	for _, marker := range []string{
		consolidationSafetyMarker,
		consolidationSafetyNativeMarker,
		founderRenunciationMarker,
		founderRenunciationNativeMarker,
		founderRenunciationPlanIdentityMarker,
	} {
		value, found, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(
			ctx,
			marker,
		)
		if err != nil {
			return fmt.Errorf(
				"native H3 lineage cannot verify pre-SDK marker %q absence: %w",
				marker,
				err,
			)
		}
		if found {
			return fmt.Errorf(
				"native H3 lineage requires pre-SDK marker %q to be truly absent: found value %q",
				marker,
				value,
			)
		}
	}
	for _, prerequisite := range []string{
		UpgradeNameConsolidationSafetyV1,
		UpgradeNameFounderRenunciationV1,
	} {
		doneHeight, err := app.UpgradeKeeper.GetDoneHeight(ctx, prerequisite)
		if err != nil {
			return fmt.Errorf(
				"native H3 lineage cannot read prerequisite %q done height: %w",
				prerequisite,
				err,
			)
		}
		if doneHeight != 0 {
			return fmt.Errorf(
				"native H3 lineage requires prerequisite %q done height exactly 0: got %d",
				prerequisite,
				doneHeight,
			)
		}
	}
	return app.requireFounderRenunciationZeroPoststate(ctx, planName)
}

func (app *ZeroneApp) requireFounderRenunciationPlanIdentity(
	ctx context.Context,
	planName string,
) (string, error) {
	value, found, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(
		ctx,
		founderRenunciationPlanIdentityMarker,
	)
	if err != nil {
		return "", fmt.Errorf(
			"upgrade %q cannot verify prerequisite %q plan identity marker %q: %w",
			planName,
			UpgradeNameFounderRenunciationV1,
			founderRenunciationPlanIdentityMarker,
			err,
		)
	}
	if !found {
		return "", fmt.Errorf(
			"upgrade %q requires prerequisite %q plan identity marker %q to be present",
			planName,
			UpgradeNameFounderRenunciationV1,
			founderRenunciationPlanIdentityMarker,
		)
	}
	decoded, decodeErr := hex.DecodeString(value)
	if decodeErr != nil || len(decoded) != sha256.Size || hex.EncodeToString(decoded) != value {
		return "", fmt.Errorf(
			"upgrade %q requires prerequisite %q plan identity marker %q to contain exactly %d lowercase hexadecimal characters: got %q",
			planName,
			UpgradeNameFounderRenunciationV1,
			founderRenunciationPlanIdentityMarker,
			sha256.Size*2,
			value,
		)
	}
	return value, nil
}

// requireFounderRenunciationZeroPoststate proves the concrete H2 economic
// poststate instead of inferring it from vesting_rewards=2 and named-plan
// markers. The exact Params read never falls back to defaults, and GetAccount
// is deliberately used instead of the lazy-creating GetModuleAccount.
func (app *ZeroneApp) requireFounderRenunciationZeroPoststate(
	ctx context.Context,
	planName string,
) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	params, err := app.VestingRewardsKeeper.GetStoredParamsChecked(sdkCtx)
	if err != nil {
		return fmt.Errorf(
			"upgrade %q cannot read persisted founder-renunciation Params: %w",
			planName,
			err,
		)
	}
	if err := vestingrewardstypes.ValidateParams(params); err != nil {
		return fmt.Errorf(
			"upgrade %q requires valid persisted founder-renunciation v2 Params: %w",
			planName,
			err,
		)
	}
	if params.FounderShareBps != 0 ||
		params.FounderAddress != "" ||
		params.BlockReward != "0" ||
		params.FloorReward != "0" ||
		params.EmptyBlockRewardRate != 0 {
		return fmt.Errorf(
			"upgrade %q requires founder-renunciation zero poststate: founder_share_bps=%d founder_address_empty=%t block_reward=%q floor_reward=%q empty_block_reward_rate=%d",
			planName,
			params.FounderShareBps,
			params.FounderAddress == "",
			params.BlockReward,
			params.FloorReward,
			params.EmptyBlockRewardRate,
		)
	}

	account := app.AccountKeeper.GetAccount(
		sdkCtx,
		authtypes.NewModuleAddress(vestingrewardstypes.ModuleName),
	)
	if account == nil {
		return nil
	}
	moduleAccount, ok := account.(sdk.ModuleAccountI)
	if !ok {
		return fmt.Errorf(
			"upgrade %q requires existing %q account to implement ModuleAccountI: got %T",
			planName,
			vestingrewardstypes.ModuleName,
			account,
		)
	}
	if moduleAccount.GetName() != vestingrewardstypes.ModuleName {
		return fmt.Errorf(
			"upgrade %q requires existing %q module account name: got %q",
			planName,
			vestingrewardstypes.ModuleName,
			moduleAccount.GetName(),
		)
	}
	if permissions := moduleAccount.GetPermissions(); len(permissions) != 0 {
		return fmt.Errorf(
			"upgrade %q requires existing %q module account to have exact empty permissions: got %v",
			planName,
			vestingrewardstypes.ModuleName,
			permissions,
		)
	}
	return nil
}

func (app *ZeroneApp) requireCompletedPreSDKTransition(
	ctx context.Context,
	planName string,
	prerequisiteUpgrade string,
	markerName string,
	markerValue string,
) (int64, error) {
	marker, found, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(
		ctx,
		markerName,
	)
	if err != nil {
		return 0, fmt.Errorf(
			"upgrade %q cannot verify prerequisite %q marker %q: %w",
			planName,
			prerequisiteUpgrade,
			markerName,
			err,
		)
	}
	if !found {
		return 0, fmt.Errorf(
			"upgrade %q requires prerequisite %q marker %q to be present",
			planName,
			prerequisiteUpgrade,
			markerName,
		)
	}
	if marker != markerValue {
		return 0, fmt.Errorf(
			"upgrade %q requires prerequisite %q marker %q=%q: got %q",
			planName,
			prerequisiteUpgrade,
			markerName,
			markerValue,
			marker,
		)
	}
	doneHeight, err := app.UpgradeKeeper.GetDoneHeight(ctx, prerequisiteUpgrade)
	if err != nil {
		return 0, fmt.Errorf(
			"upgrade %q cannot verify prerequisite %q done height: %w",
			planName,
			prerequisiteUpgrade,
			err,
		)
	}
	if doneHeight <= 0 {
		return 0, fmt.Errorf(
			"upgrade %q requires prerequisite %q done height greater than zero: got %d",
			planName,
			prerequisiteUpgrade,
			doneHeight,
		)
	}
	return doneHeight, nil
}

func requireSDK053IBC10ExactSourceVersionMap(
	planName string,
	fromVM map[string]uint64,
) error {
	expectedNames := make(map[string]struct{}, len(sdk053IBC10ExactSourceVersionMap))
	for index, expected := range sdk053IBC10ExactSourceVersionMap {
		if expected.name == "" ||
			(index > 0 && sdk053IBC10ExactSourceVersionMap[index-1].name >= expected.name) {
			return fmt.Errorf(
				"upgrade %q has a non-canonical internal source VersionMap pin at index %d",
				planName,
				index,
			)
		}
		expectedNames[expected.name] = struct{}{}
		version, present := fromVM[expected.name]
		if !present || version != expected.version {
			return fmt.Errorf(
				"upgrade %q requires exact full source VersionMap entry %q=%d: got %d (present=%t)",
				planName,
				expected.name,
				expected.version,
				version,
				present,
			)
		}
	}
	extraNames := make([]string, 0)
	for name := range fromVM {
		if _, expected := expectedNames[name]; !expected {
			extraNames = append(extraNames, name)
		}
	}
	if len(extraNames) > 0 {
		sort.Strings(extraNames)
		return fmt.Errorf(
			"upgrade %q refuses unexpected full source VersionMap entries: %v",
			planName,
			extraNames,
		)
	}
	if len(fromVM) != len(sdk053IBC10ExactSourceVersionMap) {
		return fmt.Errorf(
			"upgrade %q requires exact full source VersionMap size %d: got %d",
			planName,
			len(sdk053IBC10ExactSourceVersionMap),
			len(fromVM),
		)
	}
	actualDigest, err := canonicalSDK053IBC10SourceVersionMapSHA256(fromVM)
	if err != nil {
		return fmt.Errorf(
			"upgrade %q cannot canonicalize exact full source VersionMap: %w",
			planName,
			err,
		)
	}
	if actualDigest != sdk053IBC10SourceVersionMapSHA256 {
		return fmt.Errorf(
			"upgrade %q exact full source VersionMap digest mismatch: require %s, got %s",
			planName,
			sdk053IBC10SourceVersionMapSHA256,
			actualDigest,
		)
	}
	return nil
}

func canonicalSDK053IBC10SourceVersionMapSHA256(
	versionMap map[string]uint64,
) (string, error) {
	canonicalJSON, err := json.Marshal(versionMap)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(canonicalJSON)
	return fmt.Sprintf("%x", digest), nil
}

func requireSDK053IBC10SourceVersions(
	planName string,
	fromVM map[string]uint64,
) error {
	expected := []struct {
		name    string
		version uint64
	}{
		{ibcexported.ModuleName, 6},
		{ibctransfertypes.ModuleName, 5},
		{icatypes.ModuleName, 3},
		{legacyCapabilityStoreKey, 1},
		{legacyIBCFeeStoreKey, 2},
	}
	for _, moduleVersion := range expected {
		version, ok := fromVM[moduleVersion.name]
		if !ok {
			return fmt.Errorf(
				"upgrade %q requires source module %q at consensus version %d: module is absent from version map",
				planName,
				moduleVersion.name,
				moduleVersion.version,
			)
		}
		if version != moduleVersion.version {
			return fmt.Errorf(
				"upgrade %q requires source module %q at consensus version %d: got %d",
				planName,
				moduleVersion.name,
				moduleVersion.version,
				version,
			)
		}
	}
	return nil
}

func recordSetBytes(records []upgradePrestateRecord) int {
	total := 0
	for _, record := range records {
		total += len(record.Key) + len(record.Value)
	}
	return total
}

func summarizeCustomUpgradeStake(
	records []upgradePrestateRecord,
) (customUpgradeStakeSummary, error) {
	entries := make([]customUpgradeStakeEntry, 0, len(records))
	total := new(big.Int)
	for _, record := range records {
		var lip zeronegovtypes.LIP
		if err := json.Unmarshal(record.Value, &lip); err != nil {
			return customUpgradeStakeSummary{}, fmt.Errorf(
				"decode committed custom governance LIP at key %x: %w",
				record.Key,
				err,
			)
		}
		if lip.Id == "" ||
			!bytes.Equal(record.Key, zeronegovtypes.LIPKey(lip.Id)) {
			return customUpgradeStakeSummary{}, fmt.Errorf(
				"committed custom governance LIP key %x does not match encoded id %q",
				record.Key,
				lip.Id,
			)
		}
		stakeText := lip.StakedAmount
		if stakeText == "" {
			stakeText = "0"
		}
		stake, ok := new(big.Int).SetString(stakeText, 10)
		if !ok || stake.Sign() < 0 {
			return customUpgradeStakeSummary{}, fmt.Errorf(
				"committed custom governance LIP %q has invalid aggregate stake %q",
				lip.Id,
				lip.StakedAmount,
			)
		}
		total.Add(total, stake)
		entries = append(entries, customUpgradeStakeEntry{
			LIPID:      lip.Id,
			StakedUzrn: stake.String(),
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].LIPID < entries[j].LIPID
	})
	manifest, err := json.Marshal(entries)
	if err != nil {
		return customUpgradeStakeSummary{}, fmt.Errorf(
			"encode custom governance stake manifest: %w",
			err,
		)
	}
	digest := sha256.Sum256(manifest)
	return customUpgradeStakeSummary{
		Entries:        entries,
		TotalUzrn:      total.String(),
		ManifestSHA256: fmt.Sprintf("%x", digest),
	}, nil
}

func ensureNoUnattributedCustomUpgradeStake(
	summary customUpgradeStakeSummary,
) error {
	if summary.TotalUzrn == "" || summary.TotalUzrn == "0" {
		return nil
	}
	ids := make([]string, len(summary.Entries))
	for i, entry := range summary.Entries {
		ids[i] = entry.LIPID
	}
	return fmt.Errorf(
		"custom software-authority retirement would strand %s uzrn across LIPs %v without a claimant ledger (stake manifest sha256 %s); reconcile the balance before activation",
		summary.TotalUzrn,
		ids,
		summary.ManifestSHA256,
	)
}

// collectAndVerifyActivationPrestate exports each complete committed IAVL
// tree and recomputes its root hash from the export stream. This avoids both
// cacheMergeIterator's hidden parent errors and IAVL v1.2.2's silently
// truncated iterator traversal without putting mutable H-1 state into the
// already immutable software-upgrade plan.Info.
func (app *ZeroneApp) collectAndVerifyActivationPrestate() (verifiedActivationPrestate, error) {
	expectedCommitIDs, err := app.activationSubstoreCommitIDs([]string{
		emergencytypes.StoreKey,
		sdkgovtypes.StoreKey,
		zeronegovtypes.StoreKey,
	})
	if err != nil {
		return verifiedActivationPrestate{}, err
	}
	emergencyRecords, err := app.collectVerifiedCommittedRecords(
		emergencytypes.StoreKey,
		expectedCommitIDs[emergencytypes.StoreKey],
		func(key, _ []byte) (bool, error) {
			return emergencykeeper.IsOperationsSafetySnapshotKey(key), nil
		},
	)
	if err != nil {
		return verifiedActivationPrestate{}, err
	}
	emergencyInput := make(
		[]emergencykeeper.OperationsSafetyRecord,
		len(emergencyRecords),
	)
	for i, record := range emergencyRecords {
		emergencyInput[i] = emergencykeeper.OperationsSafetyRecord{
			Key:   record.Key,
			Value: record.Value,
		}
	}
	emergencySnapshot, err := emergencykeeper.NewOperationsSafetySnapshot(
		emergencyInput,
	)
	if err != nil {
		return verifiedActivationPrestate{}, fmt.Errorf(
			"construct verified emergency operations snapshot: %w",
			err,
		)
	}

	sdkGovSelectedRecords, err := app.collectVerifiedCommittedRecords(
		sdkgovtypes.StoreKey,
		expectedCommitIDs[sdkgovtypes.StoreKey],
		func(key, value []byte) (bool, error) {
			if bytes.HasPrefix(
				key,
				sdkgovtypes.ActiveProposalQueuePrefix.Bytes(),
			) || bytes.HasPrefix(
				key,
				sdkgovtypes.InactiveProposalQueuePrefix.Bytes(),
			) {
				return true, nil
			}
			if !bytes.HasPrefix(
				key,
				sdkgovtypes.ProposalsKeyPrefix.Bytes(),
			) {
				return false, nil
			}
			_, proposal, err := decodeCommittedSDKGovProposal(key, value)
			if err != nil {
				return false, err
			}
			switch proposal.Status {
			case govv1.StatusDepositPeriod, govv1.StatusVotingPeriod:
				return true, nil
			default:
				return false, nil
			}
		},
	)
	if err != nil {
		return verifiedActivationPrestate{}, err
	}
	queueRecords := make(
		[]upgradePrestateRecord,
		0,
		len(sdkGovSelectedRecords),
	)
	sdkGovRecords := make(
		[]upgradePrestateRecord,
		0,
		len(sdkGovSelectedRecords),
	)
	for _, record := range sdkGovSelectedRecords {
		switch {
		case bytes.HasPrefix(
			record.Key,
			sdkgovtypes.ActiveProposalQueuePrefix.Bytes(),
		), bytes.HasPrefix(
			record.Key,
			sdkgovtypes.InactiveProposalQueuePrefix.Bytes(),
		):
			queueRecords = append(queueRecords, record)
		case bytes.HasPrefix(
			record.Key,
			sdkgovtypes.ProposalsKeyPrefix.Bytes(),
		):
			sdkGovRecords = append(sdkGovRecords, record)
		default:
			return verifiedActivationPrestate{}, fmt.Errorf(
				"SDK governance prestate selected out-of-domain key %x",
				record.Key,
			)
		}
	}
	queuedProposals, err := sdkGovQueueProposalReferences(
		queueRecords,
	)
	if err != nil {
		return verifiedActivationPrestate{}, err
	}
	foundQueuedProposalIDs := make(
		map[uint64]struct{},
		len(queuedProposals),
	)
	for _, record := range sdkGovRecords {
		proposalID, proposal, err := decodeCommittedSDKGovProposal(
			record.Key,
			record.Value,
		)
		if err != nil {
			return verifiedActivationPrestate{}, err
		}
		if queueReference, queued := queuedProposals[proposalID]; queued {
			if err := validateQueuedSDKGovProposal(
				proposalID,
				proposal,
				queueReference,
			); err != nil {
				return verifiedActivationPrestate{}, err
			}
			foundQueuedProposalIDs[proposalID] = struct{}{}
		}
	}
	for proposalID := range queuedProposals {
		if _, found := foundQueuedProposalIDs[proposalID]; found {
			continue
		}
		key := sdkGovProposalKey(proposalID)
		value, err := app.getBoundCommittedValue(
			sdkgovtypes.StoreKey,
			expectedCommitIDs[sdkgovtypes.StoreKey],
			key,
		)
		if err != nil {
			return verifiedActivationPrestate{}, err
		}
		if value == nil {
			return verifiedActivationPrestate{}, fmt.Errorf(
				"active SDK governance queue references missing proposal %d",
				proposalID,
			)
		}
		_, proposal, err := decodeCommittedSDKGovProposal(key, value)
		if err != nil {
			return verifiedActivationPrestate{}, err
		}
		if err := validateQueuedSDKGovProposal(
			proposalID,
			proposal,
			queuedProposals[proposalID],
		); err != nil {
			return verifiedActivationPrestate{}, err
		}
	}
	customGovRecords, err := app.collectVerifiedCommittedRecords(
		zeronegovtypes.StoreKey,
		expectedCommitIDs[zeronegovtypes.StoreKey],
		func(key, value []byte) (bool, error) {
			if bytes.Equal(
				key,
				zeronegovtypes.EmergencyTransitionHoldKey,
			) {
				return false, fmt.Errorf(
					"irreparable custom governance source state pre-seeds reserved emergency transition hold key %x before operations-safety activation",
					key,
				)
			}
			if !bytes.HasPrefix(key, zeronegovtypes.LIPKeyPrefix) {
				return false, nil
			}
			var lip zeronegovtypes.LIP
			if err := json.Unmarshal(value, &lip); err != nil {
				return false, fmt.Errorf(
					"decode committed custom governance LIP at key %x: %w",
					key,
					err,
				)
			}
			expectedKey := zeronegovtypes.LIPKey(lip.Id)
			if lip.Id == "" || !bytes.Equal(key, expectedKey) {
				return false, fmt.Errorf(
					"committed custom governance LIP key %x does not match id %q",
					key,
					lip.Id,
				)
			}
			return lip.Category == zeronegovtypes.CategoryUpgrade &&
				!zeronegovtypes.IsTerminal(lip.Stage), nil
		},
	)
	if err != nil {
		return verifiedActivationPrestate{}, err
	}
	customGovStake, err := summarizeCustomUpgradeStake(customGovRecords)
	if err != nil {
		return verifiedActivationPrestate{}, err
	}

	return verifiedActivationPrestate{
		EmergencySnapshot:         emergencySnapshot,
		SDKGovExecutableProposals: sdkGovRecords,
		CustomGovLIPs:             customGovRecords,
		CustomGovStake:            customGovStake,
	}, nil
}

type sdkGovQueueProposalReference struct {
	Deadline time.Time
	Status   govv1.ProposalStatus
	Label    string
}

func sdkGovQueueProposalReferences(
	records []upgradePrestateRecord,
) (map[uint64]sdkGovQueueProposalReference, error) {
	pairCodec := collections.PairKeyCodec(sdk.TimeKey, collections.Uint64Key)
	references := make(
		map[uint64]sdkGovQueueProposalReference,
		len(records),
	)
	for _, record := range records {
		var (
			prefix []byte
			status govv1.ProposalStatus
			label  string
		)
		switch {
		case bytes.HasPrefix(
			record.Key,
			sdkgovtypes.ActiveProposalQueuePrefix.Bytes(),
		):
			prefix = sdkgovtypes.ActiveProposalQueuePrefix.Bytes()
			status = govv1.StatusVotingPeriod
			label = "active"
		case bytes.HasPrefix(
			record.Key,
			sdkgovtypes.InactiveProposalQueuePrefix.Bytes(),
		):
			prefix = sdkgovtypes.InactiveProposalQueuePrefix.Bytes()
			status = govv1.StatusDepositPeriod
			label = "inactive"
		default:
			return nil, fmt.Errorf(
				"invalid committed SDK governance queue record key=%x value_bytes=%d",
				record.Key,
				len(record.Value),
			)
		}
		suffix := record.Key[len(prefix):]
		read, queueKey, err := pairCodec.Decode(suffix)
		if err != nil || read != len(suffix) {
			return nil, fmt.Errorf(
				"decode committed %s SDK governance queue key %x: read=%d bytes=%d: %w",
				label,
				record.Key,
				read,
				len(suffix),
				err,
			)
		}
		valueID, err := collections.Uint64Value.Decode(record.Value)
		if err != nil {
			return nil, fmt.Errorf(
				"decode committed %s SDK governance queue value at key %x: %w",
				label,
				record.Key,
				err,
			)
		}
		keyID := queueKey.K2()
		if keyID == 0 || keyID != valueID {
			return nil, fmt.Errorf(
				"%s SDK governance queue key proposal %d does not match value %d",
				label,
				keyID,
				valueID,
			)
		}
		if prior, duplicate := references[valueID]; duplicate {
			return nil, fmt.Errorf(
				"SDK governance proposal %d appears in multiple queue records (%s and %s)",
				valueID,
				prior.Label,
				label,
			)
		}
		references[valueID] = sdkGovQueueProposalReference{
			Deadline: queueKey.K1(),
			Status:   status,
			Label:    label,
		}
	}
	return references, nil
}

func activeSDKGovQueueProposalDeadlines(
	records []upgradePrestateRecord,
) (map[uint64]time.Time, error) {
	references, err := sdkGovQueueProposalReferences(records)
	if err != nil {
		return nil, err
	}
	deadlines := make(map[uint64]time.Time, len(references))
	for proposalID, reference := range references {
		if reference.Status != govv1.StatusVotingPeriod {
			return nil, fmt.Errorf(
				"expected active SDK governance queue record for proposal %d, got %s",
				proposalID,
				reference.Label,
			)
		}
		deadlines[proposalID] = reference.Deadline
	}
	return deadlines, nil
}

func sdkGovProposalKey(proposalID uint64) []byte {
	prefix := sdkgovtypes.ProposalsKeyPrefix.Bytes()
	key := make([]byte, len(prefix)+8)
	copy(key, prefix)
	binary.BigEndian.PutUint64(key[len(prefix):], proposalID)
	return key
}

func decodeCommittedSDKGovProposal(
	key, value []byte,
) (uint64, govv1.Proposal, error) {
	var proposal govv1.Proposal
	prefix := sdkgovtypes.ProposalsKeyPrefix.Bytes()
	if !bytes.HasPrefix(key, prefix) || len(key) != len(prefix)+8 {
		return 0, proposal, fmt.Errorf(
			"invalid committed SDK governance proposal key %x",
			key,
		)
	}
	proposalID := binary.BigEndian.Uint64(key[len(prefix):])
	if err := proposal.Unmarshal(value); err != nil {
		return 0, proposal, fmt.Errorf(
			"decode committed SDK governance proposal envelope %d: %w",
			proposalID,
			err,
		)
	}
	if proposal.Id != proposalID {
		return 0, proposal, fmt.Errorf(
			"committed SDK governance proposal key %d does not match encoded id %d",
			proposalID,
			proposal.Id,
		)
	}
	return proposalID, proposal, nil
}

func validateQueuedSDKGovProposal(
	proposalID uint64,
	proposal govv1.Proposal,
	reference sdkGovQueueProposalReference,
) error {
	var proposalDeadline *time.Time
	switch reference.Status {
	case govv1.StatusDepositPeriod:
		proposalDeadline = proposal.DepositEndTime
	case govv1.StatusVotingPeriod:
		proposalDeadline = proposal.VotingEndTime
	default:
		return fmt.Errorf(
			"%s SDK governance queue proposal %d has unsupported expected status %s",
			reference.Label,
			proposalID,
			reference.Status,
		)
	}
	if proposal.Status != reference.Status ||
		proposalDeadline == nil ||
		!proposalDeadline.Equal(reference.Deadline) {
		return fmt.Errorf(
			"%s SDK governance queue proposal %d is incoherent: status=%s expected_status=%s proposal_deadline=%v queue_deadline=%s",
			reference.Label,
			proposalID,
			proposal.Status,
			reference.Status,
			proposalDeadline,
			reference.Deadline.UTC().Format(time.RFC3339Nano),
		)
	}
	return nil
}

func (app *ZeroneApp) getBoundCommittedValue(
	storeName string,
	expectedCommitID storetypes.CommitID,
	key []byte,
) ([]byte, error) {
	storeKey, ok := app.keys[storeName]
	if !ok || storeKey == nil {
		return nil, fmt.Errorf("store key %q is not mounted", storeName)
	}
	committed := app.CommitMultiStore().GetCommitKVStore(storeKey)
	if committed == nil {
		return nil, fmt.Errorf("committed store %q is unavailable", storeName)
	}
	exportStore, ok := committed.(committedIAVLExporter)
	if !ok {
		return nil, fmt.Errorf(
			"committed store %q has type %T, not the required regular-IAVL store",
			storeName,
			committed,
		)
	}
	if err := validateActivationSubstoreCommitID(
		storeName,
		exportStore.LastCommitID(),
		expectedCommitID,
	); err != nil {
		return nil, err
	}
	return bytes.Clone(committed.Get(key)), nil
}

func (app *ZeroneApp) collectVerifiedCommittedRecords(
	storeName string,
	expectedCommitID storetypes.CommitID,
	selectRecord prestateRecordSelector,
) ([]upgradePrestateRecord, error) {
	key, ok := app.keys[storeName]
	if !ok || key == nil {
		return nil, fmt.Errorf("store key %q is not mounted", storeName)
	}
	committed := app.CommitMultiStore().GetCommitKVStore(key)
	if committed == nil {
		return nil, fmt.Errorf("committed store %q is unavailable", storeName)
	}
	exportStore, ok := committed.(committedIAVLExporter)
	if !ok {
		return nil, fmt.Errorf(
			"committed store %q has type %T, not the required regular-IAVL exporter",
			storeName,
			committed,
		)
	}
	commitID := exportStore.LastCommitID()
	if err := validateActivationSubstoreCommitID(
		storeName,
		commitID,
		expectedCommitID,
	); err != nil {
		return nil, err
	}
	if commitID.Version <= 0 || len(commitID.Hash) != sha256.Size {
		return nil, fmt.Errorf(
			"committed store %q has invalid commit id version=%d hash_bytes=%d",
			storeName,
			commitID.Version,
			len(commitID.Hash),
		)
	}
	emptyRoot := sha256.Sum256(nil)
	if bytes.Equal(commitID.Hash, emptyRoot[:]) {
		return nil, nil
	}

	exporter, err := exportStore.Export(commitID.Version)
	if err != nil {
		return nil, fmt.Errorf(
			"open complete committed export for store %q at version %d: %w",
			storeName,
			commitID.Version,
			err,
		)
	}
	defer exporter.Close()

	records, root, err := verifyIAVLExportStream(
		exporter.Next,
		selectRecord,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"verify complete committed export for store %q: %w",
			storeName,
			err,
		)
	}
	if !bytes.Equal(root, commitID.Hash) {
		return nil, fmt.Errorf(
			"complete committed export for store %q does not reconstruct its root: expected %x, got %x",
			storeName,
			commitID.Hash,
			root,
		)
	}
	return records, nil
}

func (app *ZeroneApp) activationSubstoreCommitIDs(
	requiredStoreNames []string,
) (map[string]storetypes.CommitID, error) {
	rootStore, ok := app.CommitMultiStore().(rootCommitInfoReader)
	if !ok {
		return nil, fmt.Errorf(
			"root commit store has type %T, not the required CommitInfo reader",
			app.CommitMultiStore(),
		)
	}
	rootID := rootStore.LastCommitID()
	if rootID.Version <= 0 || len(rootID.Hash) != sha256.Size {
		return nil, fmt.Errorf(
			"root commit id is invalid: version=%d hash_bytes=%d",
			rootID.Version,
			len(rootID.Hash),
		)
	}
	commitInfo, err := rootStore.GetCommitInfo(rootID.Version)
	if err != nil {
		return nil, fmt.Errorf(
			"read root CommitInfo at version %d: %w",
			rootID.Version,
			err,
		)
	}
	return validateActivationRootCommitInfo(
		rootID,
		commitInfo,
		requiredStoreNames,
	)
}

func validateActivationRootCommitInfo(
	rootID storetypes.CommitID,
	commitInfo *storetypes.CommitInfo,
	requiredStoreNames []string,
) (map[string]storetypes.CommitID, error) {
	if commitInfo == nil {
		return nil, errors.New("root CommitInfo is nil")
	}
	if commitInfo.Version != rootID.Version {
		return nil, fmt.Errorf(
			"root CommitInfo version %d does not match root commit id version %d",
			commitInfo.Version,
			rootID.Version,
		)
	}
	computedRoot := commitInfo.Hash()
	if len(computedRoot) != sha256.Size ||
		len(rootID.Hash) != sha256.Size ||
		!bytes.Equal(computedRoot, rootID.Hash) {
		return nil, fmt.Errorf(
			"root CommitInfo hash %x does not match root commit id hash %x",
			computedRoot,
			rootID.Hash,
		)
	}

	required := make(map[string]struct{}, len(requiredStoreNames))
	for _, name := range requiredStoreNames {
		if name == "" {
			return nil, errors.New("required activation store name is empty")
		}
		if _, duplicate := required[name]; duplicate {
			return nil, fmt.Errorf(
				"required activation store name %q is duplicated",
				name,
			)
		}
		required[name] = struct{}{}
	}
	found := make(map[string]storetypes.CommitID, len(required))
	seenAll := make(map[string]struct{}, len(commitInfo.StoreInfos))
	for _, storeInfo := range commitInfo.StoreInfos {
		if _, duplicate := seenAll[storeInfo.Name]; duplicate {
			return nil, fmt.Errorf(
				"root CommitInfo contains duplicate store %q",
				storeInfo.Name,
			)
		}
		seenAll[storeInfo.Name] = struct{}{}
		if _, needed := required[storeInfo.Name]; !needed {
			continue
		}
		if storeInfo.CommitId.Version != rootID.Version ||
			len(storeInfo.CommitId.Hash) != sha256.Size {
			return nil, fmt.Errorf(
				"root CommitInfo store %q has invalid commit id version=%d hash_bytes=%d",
				storeInfo.Name,
				storeInfo.CommitId.Version,
				len(storeInfo.CommitId.Hash),
			)
		}
		found[storeInfo.Name] = storetypes.CommitID{
			Version: storeInfo.CommitId.Version,
			Hash:    bytes.Clone(storeInfo.CommitId.Hash),
		}
	}
	for name := range required {
		if _, ok := found[name]; !ok {
			return nil, fmt.Errorf(
				"root CommitInfo is missing required store %q",
				name,
			)
		}
	}
	return found, nil
}

func validateActivationSubstoreCommitID(
	storeName string,
	actual, expected storetypes.CommitID,
) error {
	if actual.Version != expected.Version ||
		!bytes.Equal(actual.Hash, expected.Hash) {
		return fmt.Errorf(
			"loaded committed store %q is not the subroot bound by root CommitInfo: expected version=%d hash=%x, got version=%d hash=%x",
			storeName,
			expected.Version,
			expected.Hash,
			actual.Version,
			actual.Hash,
		)
	}
	return nil
}

func verifyIAVLExport(
	nodes []*iavl.ExportNode,
	selectRecord prestateRecordSelector,
) ([]upgradePrestateRecord, []byte, error) {
	index := 0
	return verifyIAVLExportStream(
		func() (*iavl.ExportNode, error) {
			if index >= len(nodes) {
				return nil, iavl.ErrorExportDone
			}
			node := nodes[index]
			index++
			return node, nil
		},
		selectRecord,
	)
}

func verifyIAVLExportStream(
	next func() (*iavl.ExportNode, error),
	selectRecord prestateRecordSelector,
) ([]upgradePrestateRecord, []byte, error) {
	if next == nil {
		return nil, nil, errors.New("IAVL export reader is nil")
	}
	if selectRecord == nil {
		return nil, nil, errors.New("IAVL export record selector is nil")
	}
	stack := make([]exportedNodeSummary, 0)
	records := make([]upgradePrestateRecord, 0)
	var leafCount uint64
	selectedBytes := 0

	for index := 0; ; index++ {
		node, nextErr := next()
		if errors.Is(nextErr, iavl.ErrorExportDone) {
			break
		}
		if nextErr != nil {
			return nil, nil, fmt.Errorf(
				"read IAVL export node %d: %w",
				index,
				nextErr,
			)
		}
		if node == nil {
			return nil, nil, fmt.Errorf("export node %d is nil", index)
		}
		if node.Version <= 0 {
			return nil, nil, fmt.Errorf(
				"export node %d has non-positive version %d",
				index,
				node.Version,
			)
		}
		if node.Height < 0 {
			return nil, nil, fmt.Errorf(
				"export node %d has negative height %d",
				index,
				node.Height,
			)
		}
		if node.Height == 0 {
			if len(node.Key) == 0 || node.Value == nil {
				return nil, nil, fmt.Errorf(
					"leaf export node %d has an empty key or nil value",
					index,
				)
			}
			hash, err := hashExportedIAVLNode(
				node.Height,
				1,
				node.Version,
				node.Key,
				node.Value,
				nil,
				nil,
			)
			if err != nil {
				return nil, nil, err
			}
			stack = append(stack, exportedNodeSummary{
				hash:   hash,
				height: 0,
				size:   1,
				minKey: bytes.Clone(node.Key),
				maxKey: bytes.Clone(node.Key),
			})
			leafCount++
			selected, selectErr := selectRecord(node.Key, node.Value)
			if selectErr != nil {
				return nil, nil, fmt.Errorf(
					"select IAVL export leaf %d key %x: %w",
					index,
					node.Key,
					selectErr,
				)
			}
			if selected {
				if len(records) >= maxSelectedPrestateRecordCount {
					return nil, nil, fmt.Errorf(
						"selected prestate exceeds %d records",
						maxSelectedPrestateRecordCount,
					)
				}
				if len(node.Key) > maxSelectedPrestateRecordBytes-selectedBytes {
					return nil, nil, fmt.Errorf(
						"selected prestate exceeds %d aggregate key/value bytes",
						maxSelectedPrestateRecordBytes,
					)
				}
				selectedBytes += len(node.Key)
				if len(node.Value) > maxSelectedPrestateRecordBytes-selectedBytes {
					return nil, nil, fmt.Errorf(
						"selected prestate exceeds %d aggregate key/value bytes",
						maxSelectedPrestateRecordBytes,
					)
				}
				selectedBytes += len(node.Value)
				records = append(records, upgradePrestateRecord{
					Key:   bytes.Clone(node.Key),
					Value: bytes.Clone(node.Value),
				})
			}
			continue
		}

		if node.Value != nil {
			return nil, nil, fmt.Errorf(
				"inner export node %d has a non-nil value",
				index,
			)
		}
		if len(stack) < 2 {
			return nil, nil, fmt.Errorf(
				"inner export node %d has fewer than two completed child subtrees",
				index,
			)
		}
		right := stack[len(stack)-1]
		left := stack[len(stack)-2]
		stack = stack[:len(stack)-2]
		expectedHeight := left.height
		if right.height > expectedHeight {
			expectedHeight = right.height
		}
		expectedHeight++
		if node.Height != expectedHeight {
			return nil, nil, fmt.Errorf(
				"inner export node %d height is %d, expected %d from children",
				index,
				node.Height,
				expectedHeight,
			)
		}
		delta := int(left.height) - int(right.height)
		if delta < -1 || delta > 1 {
			return nil, nil, fmt.Errorf(
				"inner export node %d violates AVL balance: left=%d right=%d",
				index,
				left.height,
				right.height,
			)
		}
		if bytes.Compare(left.maxKey, right.minKey) >= 0 {
			return nil, nil, fmt.Errorf(
				"inner export node %d children are not in strict key order",
				index,
			)
		}
		if !bytes.Equal(node.Key, right.minKey) {
			return nil, nil, fmt.Errorf(
				"inner export node %d separator key %x does not match right-subtree minimum %x",
				index,
				node.Key,
				right.minKey,
			)
		}
		size := left.size + right.size
		if size <= 1 {
			return nil, nil, fmt.Errorf(
				"inner export node %d has invalid derived size %d",
				index,
				size,
			)
		}
		hash, err := hashExportedIAVLNode(
			node.Height,
			size,
			node.Version,
			nil,
			nil,
			left.hash[:],
			right.hash[:],
		)
		if err != nil {
			return nil, nil, err
		}
		stack = append(stack, exportedNodeSummary{
			hash:   hash,
			height: node.Height,
			size:   size,
			minKey: left.minKey,
			maxKey: right.maxKey,
		})
	}
	if leafCount == 0 || len(stack) != 1 {
		return nil, nil, fmt.Errorf(
			"IAVL export is incomplete or structurally invalid: leaves=%d residual_subtrees=%d",
			leafCount,
			len(stack),
		)
	}
	return records, bytes.Clone(stack[0].hash[:]), nil
}

func hashExportedIAVLNode(
	height int8,
	size, version int64,
	key, value, leftHash, rightHash []byte,
) ([sha256.Size]byte, error) {
	var zero [sha256.Size]byte
	hasher := sha256.New()
	if err := writeIAVLVarint(hasher, int64(height)); err != nil {
		return zero, err
	}
	if err := writeIAVLVarint(hasher, size); err != nil {
		return zero, err
	}
	if err := writeIAVLVarint(hasher, version); err != nil {
		return zero, err
	}
	if height == 0 {
		if len(key) == 0 || value == nil {
			return zero, errors.New("cannot hash IAVL leaf with empty key or nil value")
		}
		if err := writeIAVLBytes(hasher, key); err != nil {
			return zero, err
		}
		valueHash := sha256.Sum256(value)
		if err := writeIAVLBytes(hasher, valueHash[:]); err != nil {
			return zero, err
		}
	} else {
		if len(leftHash) != sha256.Size || len(rightHash) != sha256.Size {
			return zero, errors.New("cannot hash IAVL inner node without two 32-byte child hashes")
		}
		if err := writeIAVLBytes(hasher, leftHash); err != nil {
			return zero, err
		}
		if err := writeIAVLBytes(hasher, rightHash); err != nil {
			return zero, err
		}
	}
	sum := hasher.Sum(nil)
	copy(zero[:], sum)
	return zero, nil
}

func writeIAVLVarint(writer io.Writer, value int64) error {
	var buffer [binary.MaxVarintLen64]byte
	count := binary.PutVarint(buffer[:], value)
	_, err := writer.Write(buffer[:count])
	return err
}

func writeIAVLBytes(writer io.Writer, value []byte) error {
	var buffer [binary.MaxVarintLen64]byte
	count := binary.PutUvarint(buffer[:], uint64(len(value)))
	if _, err := writer.Write(buffer[:count]); err != nil {
		return err
	}
	_, err := writer.Write(value)
	return err
}

func (app *ZeroneApp) retireCommittedCustomUpgradeLIPs(
	ctx sdk.Context,
	records []upgradePrestateRecord,
) (uint64, error) {
	lips := make([]*zeronegovtypes.LIP, 0, len(records))
	for _, record := range records {
		if !bytes.HasPrefix(record.Key, zeronegovtypes.LIPKeyPrefix) {
			return 0, fmt.Errorf(
				"custom governance snapshot contains out-of-domain key %x",
				record.Key,
			)
		}
		var lip zeronegovtypes.LIP
		if err := json.Unmarshal(record.Value, &lip); err != nil {
			return 0, fmt.Errorf(
				"decode committed custom governance LIP at key %x: %w",
				record.Key,
				err,
			)
		}
		if lip.Id == "" ||
			!bytes.Equal(record.Key, zeronegovtypes.LIPKey(lip.Id)) {
			return 0, fmt.Errorf(
				"committed custom governance LIP key %x does not match encoded id %q",
				record.Key,
				lip.Id,
			)
		}
		lips = append(lips, &lip)
	}
	return app.ZeroneGovKeeper.RetireCustomUpgradeLIPs(ctx, lips)
}

var _ committedIAVLExporter = (*storeiavl.Store)(nil)
