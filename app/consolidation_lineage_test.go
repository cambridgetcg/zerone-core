package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	corestore "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/stretchr/testify/require"

	claimingpottypes "github.com/zerone-chain/zerone/x/claiming_pot/types"
	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	liquiditypooltypes "github.com/zerone-chain/zerone/x/liquiditypool/types"
	vestingrewardstypes "github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

type startupReadErrorStoreService struct {
	err error
}

func (s startupReadErrorStoreService) OpenKVStore(context.Context) corestore.KVStore {
	return startupReadErrorStore{err: s.err}
}

type startupReadErrorStore struct {
	err error
}

func (s startupReadErrorStore) Get([]byte) ([]byte, error) { return nil, s.err }
func (s startupReadErrorStore) Has([]byte) (bool, error)   { return false, s.err }
func (s startupReadErrorStore) Set([]byte, []byte) error   { return s.err }
func (s startupReadErrorStore) Delete([]byte) error        { return s.err }
func (s startupReadErrorStore) Iterator([]byte, []byte) (corestore.Iterator, error) {
	return nil, s.err
}
func (s startupReadErrorStore) ReverseIterator([]byte, []byte) (corestore.Iterator, error) {
	return nil, s.err
}

const canonicalH1PlanInfo = `{"packet":"ipfs://consolidation-h1","sha256":"0123456789abcdef"}`

func startupTargetVM() module.VersionMap {
	return module.VersionMap{
		"bank":                         4,
		knowledgetypes.ModuleName:      6,
		claimingpottypes.ModuleName:    2,
		liquiditypooltypes.ModuleName:  5,
		vestingrewardstypes.ModuleName: 1,
	}
}

func pendingStartupEvidence() consolidationStartupEvidence {
	target := startupTargetVM()
	plan := upgradetypes.Plan{
		Name:   UpgradeNameConsolidationSafetyV1,
		Height: 101,
		Info:   canonicalH1PlanInfo,
	}
	return consolidationStartupEvidence{
		latestHeight:     100,
		versionMap:       consolidationPreVersionMap(target),
		onChainPlan:      plan,
		onChainPlanFound: true,
		diskPlan:         plan,
		diskPlanFound:    true,
	}
}

func completedStartupEvidence() consolidationStartupEvidence {
	return consolidationStartupEvidence{
		latestHeight:  120,
		versionMap:    startupTargetVM(),
		h1MarkerValue: "migrated",
		h1MarkerFound: true,
		doneHeight:    101,
		diskPlan: upgradetypes.Plan{
			Name:   UpgradeNameConsolidationSafetyV1,
			Height: 101,
			Info:   canonicalH1PlanInfo,
		},
		diskPlanFound: true,
	}
}

func nativeStartupEvidence() consolidationStartupEvidence {
	return consolidationStartupEvidence{
		latestHeight: 100,
		versionMap:   startupTargetVM(),
		nativeValue:  consolidationNativeLineageValue,
		nativeFound:  true,
	}
}

func cloneStartupEvidence(source consolidationStartupEvidence) consolidationStartupEvidence {
	clone := source
	clone.versionMap = make(module.VersionMap, len(source.versionMap))
	for name, version := range source.versionMap {
		clone.versionMap[name] = version
	}
	return clone
}

func TestConsolidationStartupLineageAcceptsOnlyExactThreeStates(t *testing.T) {
	target := startupTargetVM()

	bootstrap := consolidationStartupEvidence{
		latestHeight: 0,
		versionMap:   module.VersionMap{},
	}
	lineage, err := validateConsolidationStartupEvidence(bootstrap, target)
	require.NoError(t, err)
	require.Equal(t, consolidationLineageBootstrap, lineage)

	lineage, err = validateConsolidationStartupEvidence(pendingStartupEvidence(), target)
	require.NoError(t, err)
	require.Equal(t, consolidationLineagePending, lineage)

	lineage, err = validateConsolidationStartupEvidence(completedStartupEvidence(), target)
	require.NoError(t, err)
	require.Equal(t, consolidationLineageCompleted, lineage)

	lineage, err = validateConsolidationStartupEvidence(nativeStartupEvidence(), target)
	require.NoError(t, err)
	require.Equal(t, consolidationLineageNative, lineage)

	wrongBinaryTarget := startupTargetVM()
	wrongBinaryTarget[liquiditypooltypes.ModuleName] = 4
	_, err = validateConsolidationStartupEvidence(pendingStartupEvidence(), wrongBinaryTarget)
	require.ErrorContains(t, err, "binary target violates H1 contract")
}

func TestConsolidationPendingStartupRejectsPartialEvidenceAndPlanAmbiguity(t *testing.T) {
	target := startupTargetVM()
	tests := []struct {
		name    string
		mutate  func(*consolidationStartupEvidence)
		message string
	}{
		{
			name: "missing version key",
			mutate: func(e *consolidationStartupEvidence) {
				delete(e.versionMap, "bank")
			},
			message: "missing entry bank",
		},
		{
			name: "unknown version key",
			mutate: func(e *consolidationStartupEvidence) {
				e.versionMap["forged"] = 1
			},
			message: "unknown entry forged=1",
		},
		{
			name: "partial boundary",
			mutate: func(e *consolidationStartupEvidence) {
				e.versionMap[knowledgetypes.ModuleName] = 6
			},
			message: "forbidden partial",
		},
		{
			name: "intermediate boundary",
			mutate: func(e *consolidationStartupEvidence) {
				e.versionMap[liquiditypooltypes.ModuleName] = 4
			},
			message: "neither pre=3 nor post=5",
		},
		{
			name: "unrelated catchup",
			mutate: func(e *consolidationStartupEvidence) {
				e.versionMap["bank"] = 3
			},
			message: "entry bank=3",
		},
		{
			name: "present empty marker",
			mutate: func(e *consolidationStartupEvidence) {
				e.h1MarkerFound = true
			},
			message: "truly absent",
		},
		{
			name: "forged marker",
			mutate: func(e *consolidationStartupEvidence) {
				e.h1MarkerFound = true
				e.h1MarkerValue = "forged"
			},
			message: "truly absent",
		},
		{
			name: "preseeded migrated marker",
			mutate: func(e *consolidationStartupEvidence) {
				e.h1MarkerFound = true
				e.h1MarkerValue = "migrated"
			},
			message: "truly absent",
		},
		{
			name: "native marker conflict",
			mutate: func(e *consolidationStartupEvidence) {
				e.nativeFound = true
				e.nativeValue = consolidationNativeLineageValue
			},
			message: "conflicts with native",
		},
		{
			name: "done before migration",
			mutate: func(e *consolidationStartupEvidence) {
				e.doneHeight = 99
			},
			message: "requires done height 0",
		},
		{
			name: "missing committed plan",
			mutate: func(e *consolidationStartupEvidence) {
				e.onChainPlanFound = false
			},
			message: "requires a committed H1 plan",
		},
		{
			name: "stale committed plan",
			mutate: func(e *consolidationStartupEvidence) {
				e.onChainPlan.Height = 100
			},
			message: "latest+1=101",
		},
		{
			name: "future committed plan",
			mutate: func(e *consolidationStartupEvidence) {
				e.onChainPlan.Height = 102
			},
			message: "latest+1=101",
		},
		{
			name: "wrong committed name",
			mutate: func(e *consolidationStartupEvidence) {
				e.onChainPlan.Name = "not-h1"
			},
			message: "require name",
		},
		{
			name: "missing disk plan",
			mutate: func(e *consolidationStartupEvidence) {
				e.diskPlanFound = false
			},
			message: "requires local upgrade-info.json",
		},
		{
			name: "disk height mismatch",
			mutate: func(e *consolidationStartupEvidence) {
				e.diskPlan.Height = 102
			},
			message: "latest+1=101",
		},
		{
			name: "disk info mismatch",
			mutate: func(e *consolidationStartupEvidence) {
				e.diskPlan.Info = `{"packet":"different"}`
			},
			message: "does not exactly match",
		},
		{
			name: "unsafe skip",
			mutate: func(e *consolidationStartupEvidence) {
				e.isSkipHeight = func(height int64) bool { return height == 101 }
			},
			message: "unsafe skip",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			evidence := cloneStartupEvidence(pendingStartupEvidence())
			tc.mutate(&evidence)
			_, err := validateConsolidationStartupEvidence(evidence, target)
			require.ErrorContains(t, err, tc.message)
		})
	}
}

func TestConsolidationCompletedAndNativeRestartRejectForgedLineage(t *testing.T) {
	target := startupTargetVM()
	tests := []struct {
		name     string
		evidence consolidationStartupEvidence
		mutate   func(*consolidationStartupEvidence)
		message  string
	}{
		{
			name:     "completed empty marker",
			evidence: completedStartupEvidence(),
			mutate:   func(e *consolidationStartupEvidence) { e.h1MarkerValue = "" },
			message:  "requires upgrade_marker_consolidation-safety-v1=migrated",
		},
		{
			name:     "completed forged marker",
			evidence: completedStartupEvidence(),
			mutate:   func(e *consolidationStartupEvidence) { e.h1MarkerValue = "forged" },
			message:  "requires upgrade_marker_consolidation-safety-v1=migrated",
		},
		{
			name:     "marker without done",
			evidence: completedStartupEvidence(),
			mutate:   func(e *consolidationStartupEvidence) { e.doneHeight = 0 },
			message:  "0 < done height",
		},
		{
			name:     "future done",
			evidence: completedStartupEvidence(),
			mutate:   func(e *consolidationStartupEvidence) { e.doneHeight = 121 },
			message:  "0 < done height",
		},
		{
			name:     "completed skip",
			evidence: completedStartupEvidence(),
			mutate: func(e *consolidationStartupEvidence) {
				e.isSkipHeight = func(height int64) bool { return height == 101 }
			},
			message: "unsafe skip",
		},
		{
			name:     "both lineage markers",
			evidence: completedStartupEvidence(),
			mutate: func(e *consolidationStartupEvidence) {
				e.nativeFound = true
				e.nativeValue = consolidationNativeLineageValue
			},
			message: "conflicting migrated and native",
		},
		{
			name:     "completed pending H1 plan",
			evidence: completedStartupEvidence(),
			mutate: func(e *consolidationStartupEvidence) {
				e.onChainPlanFound = true
				e.onChainPlan = upgradetypes.Plan{
					Name: UpgradeNameConsolidationSafetyV1, Height: 121, Info: canonicalH1PlanInfo,
				}
			},
			message: "conflicting committed H1 plan",
		},
		{
			name:     "completed conflicting disk",
			evidence: completedStartupEvidence(),
			mutate:   func(e *consolidationStartupEvidence) { e.diskPlan.Height = 102 },
			message:  "conflicts with completed H1 height",
		},
		{
			name:     "native empty marker",
			evidence: nativeStartupEvidence(),
			mutate:   func(e *consolidationStartupEvidence) { e.nativeValue = "" },
			message:  "invalid value",
		},
		{
			name:     "native forged marker",
			evidence: nativeStartupEvidence(),
			mutate:   func(e *consolidationStartupEvidence) { e.nativeValue = "forged" },
			message:  "invalid value",
		},
		{
			name:     "native done conflict",
			evidence: nativeStartupEvidence(),
			mutate:   func(e *consolidationStartupEvidence) { e.doneHeight = 1 },
			message:  "requires H1 done height 0",
		},
		{
			name:     "native H1 plan conflict",
			evidence: nativeStartupEvidence(),
			mutate: func(e *consolidationStartupEvidence) {
				e.onChainPlanFound = true
				e.onChainPlan = upgradetypes.Plan{
					Name: UpgradeNameConsolidationSafetyV1, Height: 101, Info: canonicalH1PlanInfo,
				}
			},
			message: "cannot carry a committed H1",
		},
		{
			name:     "native rejects ambiguous unrelated disk packet",
			evidence: nativeStartupEvidence(),
			mutate: func(e *consolidationStartupEvidence) {
				e.diskPlanFound = true
				e.diskPlan = upgradetypes.Plan{
					Name: "future-upgrade", Height: 101, Info: "opaque",
				}
			},
			message: "does not accept any local upgrade-info.json",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			evidence := cloneStartupEvidence(tc.evidence)
			tc.mutate(&evidence)
			_, err := validateConsolidationStartupEvidence(evidence, target)
			require.ErrorContains(t, err, tc.message)
		})
	}

	postWithoutLineage := completedStartupEvidence()
	postWithoutLineage.h1MarkerFound = false
	postWithoutLineage.h1MarkerValue = ""
	postWithoutLineage.doneHeight = 0
	_, err := validateConsolidationStartupEvidence(postWithoutLineage, target)
	require.ErrorContains(t, err, "no explicit migrated or native lineage marker")
}

func TestCanonicalConsolidationPlanInfo(t *testing.T) {
	require.NoError(t, validateCanonicalConsolidationPlanInfo(canonicalH1PlanInfo))
	require.NoError(t, validateCanonicalConsolidationPlanInfo(`{"count":12,"nested":{"ok":true}}`))

	tests := []string{
		``,
		`{}`,
		`[]`,
		` {"packet":"x"}`,
		`{"z":1,"a":2}`,
		`{"a":1,"a":1}`,
		`{"a":1} `,
		`{"a":1}{"b":2}`,
		`{"a":1.0}`,
		`{"a":1e0}`,
		`{"a":-0}`,
		`not-json`,
		`{"a":"unterminated}`,
		`{"payload":"` + strings.Repeat("x", consolidationPlanInfoMaxBytes) + `"}`,
	}
	for _, info := range tests {
		require.Error(t, validateCanonicalConsolidationPlanInfo(info), info)
	}
}

func TestConsolidationStartupEvidenceReadErrorsFailClosedWithoutWrites(t *testing.T) {
	app := NewZeroneApp(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		nil,
		true,
		simtestutil.NewAppOptionsWithFlagHome(t.TempDir()),
	)
	commitBefore := app.LastCommitID()
	app.KnowledgeKeeper = knowledgekeeper.NewKeeper(
		startupReadErrorStoreService{err: errors.New("marker read failed")},
		nil,
		"",
		nil,
		nil,
	)

	_, err := app.readConsolidationStartupEvidence()
	require.ErrorContains(t, err, "read H1 migration marker")
	require.ErrorContains(t, err, "marker read failed")
	commitAfter := app.LastCommitID()
	require.Equal(t, commitBefore.Version, commitAfter.Version)
	require.Equal(t, commitBefore.Hash, commitAfter.Hash, "startup evidence reads must not commit state")
}

func TestConsolidationStartupMalformedDoneStorePanicBecomesRefusal(t *testing.T) {
	app := NewZeroneApp(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		nil,
		true,
		simtestutil.NewAppOptionsWithFlagHome(t.TempDir()),
	)
	ctx := app.NewUncachedContext(false, cmtproto.Header{})
	upgradeStore := ctx.KVStore(app.keys[upgradetypes.StoreKey])
	upgradeStore.Set([]byte{upgradetypes.DoneByte}, []byte{1})

	_, err := app.readConsolidationStartupEvidence()
	require.ErrorContains(t, err, "panic while reading consolidation startup evidence")
	stored := upgradeStore.Get([]byte{upgradetypes.DoneByte})
	require.Equal(t, []byte{1}, stored, "refusal must not repair or rewrite malformed state")
}

func TestConsolidationStartupRejectsMalformedDiskPacket(t *testing.T) {
	home := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(home, "data"), 0o700))
	require.NoError(t, os.WriteFile(
		filepath.Join(home, "data", upgradetypes.UpgradeInfoFilename),
		[]byte(`{"name":`),
		0o600,
	))

	require.Panics(t, func() {
		_ = NewZeroneApp(
			log.NewNopLogger(),
			dbm.NewMemDB(),
			nil,
			true,
			simtestutil.NewAppOptionsWithFlagHome(home),
		)
	})
}
