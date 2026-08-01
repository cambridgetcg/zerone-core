package app

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	vestingrewardstypes "github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

const canonicalH2PlanInfo = `{"packet":"ipfs://founder-renunciation-h2","sha256":"abcdef0123456789"}`
const canonicalH2TargetVersionMap = `{"alignment":1,"auth":5,"bank":4,"capability":1,"capture_challenge":1,"capture_defense":1,"claiming_pot":2,"consensus":1,"counterexamples":1,"creed":1,"distribution":3,"emergency":1,"evidence":1,"feegrant":2,"feeibc":2,"genutil":1,"gov":5,"home":1,"ibc":6,"ibcratelimit":1,"interchainaccounts":3,"knowledge":6,"liquiditypool":5,"qualification":1,"slashing":4,"sponsorship":1,"staking":5,"substrate_bridge":1,"tokens":1,"training_provenance":1,"transfer":5,"trust_score":1,"upgrade":2,"vesting":1,"vesting_rewards":2,"work_creed":1,"zerone_auth":1,"zerone_gov":2,"zerone_ontology":1,"zerone_staking":1}`

func founderStartupTargetVM() module.VersionMap {
	var target module.VersionMap
	if err := json.Unmarshal([]byte(canonicalH2TargetVersionMap), &target); err != nil {
		panic(err)
	}
	return target
}

func TestFounderRenunciationCanonicalTargetVersionMap(t *testing.T) {
	app := newRestartTestApp(t, dbm.NewMemDB(), t.TempDir())
	target := app.CurrentModuleVersionMap()
	require.Len(t, target, FounderRenunciationTargetVersionMapEntries)
	canonical, err := json.Marshal(target)
	require.NoError(t, err)
	require.Equal(t, canonicalH2TargetVersionMap, string(canonical))
	digest := sha256.Sum256(canonical)
	require.Equal(t, FounderRenunciationTargetVersionMapSHA256, fmt.Sprintf("%x", digest))
	require.NoError(t, validateFounderRenunciationTarget(target))
}

func legacyFounderParams() *vestingrewardstypes.Params {
	params := vestingrewardstypes.DefaultParams()
	params.FounderShareBps = 70_000
	params.BlockReward = "10000000"
	params.FloorReward = "100000"
	return params
}

func pendingFounderStartupEvidence() founderRenunciationStartupEvidence {
	target := founderStartupTargetVM()
	plan := upgradetypes.Plan{
		Name:   UpgradeNameFounderRenunciationV1,
		Height: 101,
		Info:   canonicalH2PlanInfo,
	}
	return founderRenunciationStartupEvidence{
		latestHeight:        100,
		versionMap:          founderRenunciationPreVersionMap(target),
		h1MarkerValue:       "migrated",
		h1MarkerFound:       true,
		h1DoneHeight:        90,
		params:              legacyFounderParams(),
		vestingAccountFound: true,
		vestingPermissions:  []string{authtypes.Minter, authtypes.Burner},
		onChainPlan:         plan,
		onChainPlanFound:    true,
		diskPlan:            plan,
		diskPlanFound:       true,
	}
}

func completedFounderStartupEvidence() founderRenunciationStartupEvidence {
	return founderRenunciationStartupEvidence{
		latestHeight:  120,
		versionMap:    founderStartupTargetVM(),
		h1MarkerValue: "migrated",
		h1MarkerFound: true,
		h1DoneHeight:  90,
		h2MarkerValue: "migrated",
		h2MarkerFound: true,
		h2DoneHeight:  101,
		params:        vestingrewardstypes.DefaultParams(),
		diskPlan: upgradetypes.Plan{
			Name:   UpgradeNameFounderRenunciationV1,
			Height: 101,
			Info:   canonicalH2PlanInfo,
		},
		diskPlanFound: true,
	}
}

func nativeFounderStartupEvidence() founderRenunciationStartupEvidence {
	return founderRenunciationStartupEvidence{
		latestHeight:  100,
		versionMap:    founderStartupTargetVM(),
		h1NativeValue: consolidationNativeLineageValue,
		h1NativeFound: true,
		h2NativeValue: founderRenunciationNativeLineageValue,
		h2NativeFound: true,
		params:        vestingrewardstypes.DefaultParams(),
	}
}

func cloneFounderStartupEvidence(
	source founderRenunciationStartupEvidence,
) founderRenunciationStartupEvidence {
	clone := source
	clone.versionMap = make(module.VersionMap, len(source.versionMap))
	for name, version := range source.versionMap {
		clone.versionMap[name] = version
	}
	if source.params != nil {
		clone.params = proto.Clone(source.params).(*vestingrewardstypes.Params)
	}
	clone.vestingPermissions = append([]string(nil), source.vestingPermissions...)
	return clone
}

func TestFounderRenunciationStartupAcceptsOnlyFourExactStates(t *testing.T) {
	target := founderStartupTargetVM()
	bootstrap := founderRenunciationStartupEvidence{
		latestHeight: 0,
		versionMap:   module.VersionMap{},
	}

	tests := []struct {
		name     string
		evidence founderRenunciationStartupEvidence
		lineage  string
	}{
		{"bootstrap", bootstrap, founderRenunciationLineageBootstrap},
		{"pending", pendingFounderStartupEvidence(), founderRenunciationLineagePending},
		{"completed", completedFounderStartupEvidence(), founderRenunciationLineageCompleted},
		{"native", nativeFounderStartupEvidence(), founderRenunciationLineageNative},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lineage, err := validateFounderRenunciationStartupEvidence(tc.evidence, target)
			require.NoError(t, err)
			require.Equal(t, tc.lineage, lineage)
		})
	}

	wrongTarget := founderStartupTargetVM()
	wrongTarget[vestingrewardstypes.ModuleName] = 1
	_, err := validateFounderRenunciationStartupEvidence(
		pendingFounderStartupEvidence(),
		wrongTarget,
	)
	require.ErrorContains(t, err, "binary target violates canonical H2 surface")
}

func TestPendingFounderRenunciationRejectsEveryAmbiguousEvidenceClass(t *testing.T) {
	target := founderStartupTargetVM()
	tests := []struct {
		name    string
		mutate  func(*founderRenunciationStartupEvidence)
		message string
	}{
		{"missing VM entry", func(e *founderRenunciationStartupEvidence) {
			delete(e.versionMap, "bank")
		}, "missing entry bank"},
		{"unknown VM entry", func(e *founderRenunciationStartupEvidence) {
			e.versionMap["forged"] = 1
		}, "unknown entry forged=1"},
		{"partial H2 VM", func(e *founderRenunciationStartupEvidence) {
			e.versionMap[vestingrewardstypes.ModuleName] = 2
		}, "completed H2 requires"},
		{"missing H1 marker", func(e *founderRenunciationStartupEvidence) {
			e.h1MarkerFound = false
			e.h1MarkerValue = ""
		}, "exact H1 migrated marker"},
		{"empty H1 marker", func(e *founderRenunciationStartupEvidence) {
			e.h1MarkerValue = ""
		}, "exact H1 migrated marker"},
		{"forged H1 marker", func(e *founderRenunciationStartupEvidence) {
			e.h1MarkerValue = "forged"
		}, "exact H1 migrated marker"},
		{"H1 native conflict", func(e *founderRenunciationStartupEvidence) {
			e.h1NativeFound = true
			e.h1NativeValue = consolidationNativeLineageValue
		}, "H1 native marker"},
		{"missing H1 done", func(e *founderRenunciationStartupEvidence) {
			e.h1DoneHeight = 0
		}, "0 < H1 done"},
		{"future H1 done", func(e *founderRenunciationStartupEvidence) {
			e.h1DoneHeight = 102
		}, "0 < H1 done"},
		{"present empty H2 marker", func(e *founderRenunciationStartupEvidence) {
			e.h2MarkerFound = true
		}, "truly absent"},
		{"preseeded H2 marker", func(e *founderRenunciationStartupEvidence) {
			e.h2MarkerFound = true
			e.h2MarkerValue = "migrated"
		}, "truly absent"},
		{"H2 native conflict", func(e *founderRenunciationStartupEvidence) {
			e.h2NativeFound = true
			e.h2NativeValue = founderRenunciationNativeLineageValue
		}, "H2 native marker"},
		{"H2 done before migration", func(e *founderRenunciationStartupEvidence) {
			e.h2DoneHeight = 101
		}, "done height 0"},
		{"missing params", func(e *founderRenunciationStartupEvidence) {
			e.params = nil
		}, "strict persisted params are missing"},
		{"unmigratable params", func(e *founderRenunciationStartupEvidence) {
			e.params.BlocksPerRewardEpoch = 0
		}, "cannot migrate"},
		{"permission drift", func(e *founderRenunciationStartupEvidence) {
			e.vestingPermissions = []string{authtypes.Minter}
		}, "exact H1 permissions"},
		{"missing committed plan", func(e *founderRenunciationStartupEvidence) {
			e.onChainPlanFound = false
		}, "requires a committed H2 plan"},
		{"wrong committed name", func(e *founderRenunciationStartupEvidence) {
			e.onChainPlan.Name = UpgradeNameConsolidationSafetyV1
		}, "require name"},
		{"stale committed plan", func(e *founderRenunciationStartupEvidence) {
			e.onChainPlan.Height = 100
		}, "latest+1=101"},
		{"future committed plan", func(e *founderRenunciationStartupEvidence) {
			e.onChainPlan.Height = 102
		}, "latest+1=101"},
		{"noncanonical committed info", func(e *founderRenunciationStartupEvidence) {
			e.onChainPlan.Info = `{"z":1,"a":2}`
		}, "not canonical"},
		{"missing disk plan", func(e *founderRenunciationStartupEvidence) {
			e.diskPlanFound = false
		}, "requires local upgrade-info.json"},
		{"disk plan mismatch", func(e *founderRenunciationStartupEvidence) {
			e.diskPlan.Info = `{"packet":"different"}`
		}, "does not exactly match"},
		{"H1 unsafe skip", func(e *founderRenunciationStartupEvidence) {
			e.isSkipHeight = func(height int64) bool { return height == 90 }
		}, "H1 height 90"},
		{"H2 unsafe skip", func(e *founderRenunciationStartupEvidence) {
			e.isSkipHeight = func(height int64) bool { return height == 101 }
		}, "H2 height 101"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			evidence := cloneFounderStartupEvidence(pendingFounderStartupEvidence())
			tc.mutate(&evidence)
			_, err := validateFounderRenunciationStartupEvidence(evidence, target)
			require.ErrorContains(t, err, tc.message)
		})
	}
}

func TestPendingFounderRenunciationAllowsNeverCreatedVestingAccount(t *testing.T) {
	evidence := pendingFounderStartupEvidence()
	evidence.vestingAccountFound = false
	evidence.vestingPermissions = nil
	lineage, err := validateFounderRenunciationStartupEvidence(
		evidence,
		founderStartupTargetVM(),
	)
	require.NoError(t, err)
	require.Equal(t, founderRenunciationLineagePending, lineage)
}

func TestCompletedFounderRenunciationRejectsForgedPoststate(t *testing.T) {
	target := founderStartupTargetVM()
	tests := []struct {
		name    string
		mutate  func(*founderRenunciationStartupEvidence)
		message string
	}{
		{"missing H1 marker", func(e *founderRenunciationStartupEvidence) {
			e.h1MarkerFound = false
			e.h1MarkerValue = ""
		}, "exact H1 migrated marker"},
		{"empty H2 marker", func(e *founderRenunciationStartupEvidence) {
			e.h2MarkerValue = ""
		}, "exact H2 migrated marker"},
		{"native conflict", func(e *founderRenunciationStartupEvidence) {
			e.h2NativeFound = true
		}, "native H2 lineage conflicts"},
		{"equal done heights", func(e *founderRenunciationStartupEvidence) {
			e.h1DoneHeight = e.h2DoneHeight
		}, "0 < H1 done < H2 done"},
		{"future H2 done", func(e *founderRenunciationStartupEvidence) {
			e.h2DoneHeight = 121
		}, "0 < H1 done < H2 done"},
		{"legacy params", func(e *founderRenunciationStartupEvidence) {
			e.params = legacyFounderParams()
		}, "block rewards are permanently retired"},
		{"nonempty module permissions", func(e *founderRenunciationStartupEvidence) {
			e.vestingAccountFound = true
			e.vestingPermissions = []string{"minter"}
		}, "retains permissions"},
		{"H1 skip", func(e *founderRenunciationStartupEvidence) {
			e.isSkipHeight = func(height int64) bool { return height == 90 }
		}, "unsafe skip"},
		{"H2 skip", func(e *founderRenunciationStartupEvidence) {
			e.isSkipHeight = func(height int64) bool { return height == 101 }
		}, "unsafe skip"},
		{"conflicting H2 plan", func(e *founderRenunciationStartupEvidence) {
			e.onChainPlanFound = true
			e.onChainPlan = upgradetypes.Plan{
				Name: UpgradeNameFounderRenunciationV1, Height: 121, Info: canonicalH2PlanInfo,
			}
		}, "conflicting H1/H2"},
		{"stale unrelated plan", func(e *founderRenunciationStartupEvidence) {
			e.onChainPlanFound = true
			e.onChainPlan = upgradetypes.Plan{
				Name: "future", Height: 120, Info: canonicalH2PlanInfo,
			}
		}, "stale committed plan"},
		{"conflicting disk height", func(e *founderRenunciationStartupEvidence) {
			e.diskPlan.Height = 102
		}, "conflicts with completed H2"},
		{"noncanonical disk info", func(e *founderRenunciationStartupEvidence) {
			e.diskPlan.Info = `{"z":1,"a":2}`
		}, "historical local H2 plan info"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			evidence := cloneFounderStartupEvidence(completedFounderStartupEvidence())
			tc.mutate(&evidence)
			_, err := validateFounderRenunciationStartupEvidence(evidence, target)
			require.ErrorContains(t, err, tc.message)
		})
	}
}

func TestNativeFounderRenunciationRequiresBothNativeMarkersAndNoMigrationProof(t *testing.T) {
	target := founderStartupTargetVM()
	tests := []struct {
		name    string
		mutate  func(*founderRenunciationStartupEvidence)
		message string
	}{
		{"missing H1 native", func(e *founderRenunciationStartupEvidence) {
			e.h1NativeFound = false
		}, "exact inherited H1 native marker"},
		{"forged H1 native", func(e *founderRenunciationStartupEvidence) {
			e.h1NativeValue = "forged"
		}, "exact inherited H1 native marker"},
		{"missing H2 native", func(e *founderRenunciationStartupEvidence) {
			e.h2NativeFound = false
		}, "exact H2 native marker"},
		{"forged H2 native", func(e *founderRenunciationStartupEvidence) {
			e.h2NativeValue = "forged"
		}, "exact H2 native marker"},
		{"H1 migration conflict", func(e *founderRenunciationStartupEvidence) {
			e.h1MarkerFound = true
			e.h1MarkerValue = "migrated"
		}, "migration marker"},
		{"H2 migration conflict", func(e *founderRenunciationStartupEvidence) {
			e.h2MarkerFound = true
			e.h2MarkerValue = "migrated"
		}, "migration marker"},
		{"done conflict", func(e *founderRenunciationStartupEvidence) {
			e.h2DoneHeight = 1
		}, "both done heights 0"},
		{"legacy params", func(e *founderRenunciationStartupEvidence) {
			e.params = legacyFounderParams()
		}, "block rewards are permanently retired"},
		{"permissions conflict", func(e *founderRenunciationStartupEvidence) {
			e.vestingAccountFound = true
			e.vestingPermissions = []string{"burner"}
		}, "retains permissions"},
		{"disk plan conflict", func(e *founderRenunciationStartupEvidence) {
			e.diskPlanFound = true
			e.diskPlan = upgradetypes.Plan{
				Name: "future", Height: 101, Info: canonicalH2PlanInfo,
			}
		}, "does not accept local"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			evidence := cloneFounderStartupEvidence(nativeFounderStartupEvidence())
			tc.mutate(&evidence)
			_, err := validateFounderRenunciationStartupEvidence(evidence, target)
			require.ErrorContains(t, err, tc.message)
		})
	}
}

func TestFounderRenunciationHandlerEvidenceRequiresExactLivePlanAndPrestate(t *testing.T) {
	target := founderStartupTargetVM()
	evidence := pendingFounderStartupEvidence()
	evidence.latestHeight = 101
	plan := evidence.onChainPlan
	pre := founderRenunciationPreVersionMap(target)
	require.NoError(t, validateFounderRenunciationHandlerEvidence(
		evidence,
		plan,
		pre,
		target,
	))

	tests := []struct {
		name    string
		mutate  func(*founderRenunciationStartupEvidence, *upgradetypes.Plan, module.VersionMap)
		message string
	}{
		{"handler plan mismatch", func(
			e *founderRenunciationStartupEvidence,
			p *upgradetypes.Plan,
			_ module.VersionMap,
		) {
			p.Info = `{"packet":"different"}`
		}, "handler plan does not exactly match"},
		{"committed plan absent", func(
			e *founderRenunciationStartupEvidence,
			_ *upgradetypes.Plan,
			_ module.VersionMap,
		) {
			e.onChainPlanFound = false
		}, "requires committed plan"},
		{"disk plan absent", func(
			e *founderRenunciationStartupEvidence,
			_ *upgradetypes.Plan,
			_ module.VersionMap,
		) {
			e.diskPlanFound = false
		}, "requires local upgrade-info.json"},
		{"wrong fromVM", func(
			_ *founderRenunciationStartupEvidence,
			_ *upgradetypes.Plan,
			vm module.VersionMap,
		) {
			vm[vestingrewardstypes.ModuleName] = 2
		}, "exact full V1 VersionMap"},
		{"committed VM mismatch", func(
			e *founderRenunciationStartupEvidence,
			_ *upgradetypes.Plan,
			_ module.VersionMap,
		) {
			e.versionMap["bank"] = 3
		}, "committed VersionMap"},
		{"permission drift", func(
			e *founderRenunciationStartupEvidence,
			_ *upgradetypes.Plan,
			_ module.VersionMap,
		) {
			e.vestingPermissions = []string{authtypes.Burner}
		}, "exact H1 permissions"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := cloneFounderStartupEvidence(evidence)
			p := plan
			vm := make(module.VersionMap, len(pre))
			for name, version := range pre {
				vm[name] = version
			}
			tc.mutate(&e, &p, vm)
			err := validateFounderRenunciationHandlerEvidence(e, p, vm, target)
			require.ErrorContains(t, err, tc.message)
		})
	}
}
