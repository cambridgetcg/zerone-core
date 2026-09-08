package cross_stack_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/stretchr/testify/require"
	zeroneapp "github.com/zerone-chain/zerone/app"
)

// Frozen SOURCE-suite coverage, not an accepted H3 executable, stopped-state
// handoff, custody review, or production release proof. These unchanged tests
// exercise historical handlers against the code they were written for. The
// candidate separately tests retirement, pure migration helpers and the named
// feedback boundary; no production guard is disabled or adapted.
const historicalInvariantSource = "89553a0132dafa1ba9670f53b8a3195b33a6e720"

func TestFrozenHistoricalUpgradeInvariants(t *testing.T) {
	root, err := filepath.Abs("../..")
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	// CI must fetch this exact public git object before tests; this test performs
	// no network access and fails (never skips) if its prerequisite is missing.
	archive := exec.CommandContext(ctx, "git", "-C", root, "archive", historicalInvariantSource)
	wire, err := archive.Output()
	require.NoError(t, err, "missing pinned historical source object %s", historicalInvariantSource)
	dir := t.TempDir()
	unpack := exec.CommandContext(ctx, "tar", "-x", "-C", dir)
	unpack.Stdin = bytes.NewReader(wire)
	out, err := unpack.CombinedOutput()
	require.NoError(t, err, "%s", out)
	pattern := "^(TestUpgrade_|TestSDK053IBC10|TestActivationPreflightCommonVerifierIsReadOnly|TestStandardSDKGovernanceSchedulesAndExecutesUpgradeAcrossRestart|TestIncident_P0_ChainHaltWithNamedUpgrade|TestResilience_FullDrillP0)"
	command := exec.CommandContext(ctx, filepath.Join(runtime.GOROOT(), "bin", "go"), "test", "-mod=readonly", "-p", "2", "-json", "./tests/cross_stack", "-run", pattern, "-count=1")
	command.Dir = dir
	command.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off")
	out, err = command.CombinedOutput()
	require.NoError(t, err, "frozen historical suite failed: %s", out)
	passed := map[string]bool{}
	for _, line := range bytes.Split(out, []byte("\n")) {
		var event struct{ Action, Test string }
		if json.Unmarshal(line, &event) == nil && event.Action == "pass" {
			passed[event.Test] = true
		}
	}
	// Fail on an accidentally narrowed regex or absent historical test, not just
	// the subprocess exit status (go test with zero selected tests also succeeds).
	for _, name := range []string{
		"TestSDK053IBC10ActivationAuditsTerminalProposalInActiveSDKGovQueue",
		"TestSDK053IBC10ActivationRejectsPendingAuthzWrappedEmergencyGovProposal",
		"TestSDK053IBC10ActivationSkipsUnknownMessagesOnlyInTerminalGovProposals",
		"TestIncident_P0_ChainHaltWithNamedUpgrade", "TestResilience_FullDrillP0",
		"TestStandardSDKGovernanceSchedulesAndExecutesUpgradeAcrossRestart",
		"TestUpgrade_ChainVersionReportWellFormed", "TestUpgrade_CompassionCalibrationV1RefreshesScores",
		"TestUpgrade_SDK053IBC10RunsIBCStateMigrations",
		"TestSDK053IBC10FounderRenunciationPoststateAcceptsAbsentModuleAccount",
		"TestSDK053IBC10ScheduledPreflightRefusesPersistedUnconsolidatedVersion",
		"TestSDK053IBC10RefusesUnattributedCustomUpgradeStakeBeforeMutation",
		"TestSDK053IBC10RefusesPreseededCustomGovernanceHoldKey",
		"TestUpgrade_SDK053IBC10RefusesLegacyFeeBalance", "TestUpgrade_SubstrateDedupeV1SeedsAndArms",
		"TestUpgrade_AgenttoolSeamV1DeclaresAxisBounds", "TestUpgrade_CurrentSDKHandlersCannotCarryFounderRenunciation",
	} {
		require.True(t, passed[name], "pinned test must actually execute: %s", name)
	}
	t.Logf("unchanged historical source %s: %d passing test/subtest receipts; NOT binary handoff evidence", historicalInvariantSource, len(passed))
}

func candidateStoreSnapshot(t *testing.T, h *TestHarness) map[string]map[string]string {
	t.Helper()
	out := map[string]map[string]string{}
	names := []string{"acc", "icacontroller", "icahost"} // store/module aliases in app.go
	for name := range h.App.CurrentModuleVersionMap() {
		names = append(names, name)
	}
	for _, name := range names {
		key := h.App.GetStoreKeyForTests(name)
		if key == nil || reflect.ValueOf(key).IsNil() {
			continue
		} // stateless modules have no mounted KV store
		it := h.Ctx.KVStore(key).Iterator(nil, nil)
		records := map[string]string{}
		for ; it.Valid(); it.Next() {
			records[string(it.Key())] = string(it.Value())
		}
		require.NoError(t, it.Close())
		out[name] = records
	}
	return out
}

func assertCandidateRetiresPlan(t *testing.T, h *TestHarness, name string) {
	t.Helper()
	before, root := candidateStoreSnapshot(t, h), h.App.LastCommitID()
	// Use the actual SDK dispatcher, not the convenience helper that seeds a
	// historical VersionMap. The refusal must precede ALL module writes.
	err := h.App.UpgradeKeeper.ApplyUpgrade(h.Ctx, upgradetypes.Plan{Name: name, Height: h.Ctx.BlockHeight()})
	require.ErrorContains(t, err, fmt.Sprintf("tok-feedback-v1 candidate cannot execute %q; use its exact accepted historical binary", name))
	require.Equal(t, before, candidateStoreSnapshot(t, h))
	require.Equal(t, root, h.App.LastCommitID())
}

func TestToKFeedbackCandidateRetiresAllHistoricalHandlersWithoutMutation(t *testing.T) {
	h := NewTestHarness(t)
	for _, name := range h.App.KnownUpgradeNames() {
		if name == zeroneapp.UpgradeNameToKFeedbackV1 {
			continue
		}
		t.Run(name, func(t *testing.T) { assertCandidateRetiresPlan(t, h, name) })
	}
}

const syntheticFeedbackPlanInfo = `{"schema":"zerone.tok-feedback-v1/lineage/v1","h3_source":"335bb94f0fd54d3752dcb397263b7e84fb1116b4","h3_tree":"769f67f1cfa108be3d31cace7777cf954f731c42"}`

// Explicitly synthetic H3 unit fixture, NOT an old binary execution or custody
// proof. No accepted scenario fact or round is manufactured by this helper.
func seedSyntheticCompletedH3(t *testing.T, h *TestHarness) {
	t.Helper()
	seedPreSDKTransitionLineage(t, h)
	require.NoError(t, h.KnowledgeKeeper.WriteMigrationMarker(h.Ctx, "upgrade_marker_sdk-0.53-ibc-10", "migrated-with-loader-proof-v1"))
	h.SeedCompletedUpgrade(zeroneapp.UpgradeNameSDK053IBC10, 3)
	vm := h.App.CurrentModuleVersionMap()
	vm["knowledge"] = 6
	require.NoError(t, h.App.UpgradeKeeper.SetModuleVersionMap(h.Ctx, vm))
	h.currentHeight = 4
	h.Ctx = h.Ctx.WithBlockHeight(4)
}

func runSyntheticFeedbackUpgrade(t *testing.T, h *TestHarness) {
	t.Helper()
	vm, err := h.App.UpgradeKeeper.GetModuleVersionMap(h.Ctx)
	require.NoError(t, err)
	require.EqualValues(t, 6, vm["knowledge"])
	to, err := h.App.RunUpgradeHandlerWithInfoForTests(h.Ctx, zeroneapp.UpgradeNameToKFeedbackV1, vm, h.Height(), syntheticFeedbackPlanInfo)
	require.NoError(t, err)
	require.EqualValues(t, 7, to["knowledge"])
	require.Equal(t, "completed-h3-to-knowledge7-v1", h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "upgrade_marker_tok-feedback-v1"))
	require.Empty(t, h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "upgrade_marker_v1.0.1"))
}
