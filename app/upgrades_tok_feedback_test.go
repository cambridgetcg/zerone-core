package app

import (
	"cosmossdk.io/core/header"
	"cosmossdk.io/log"
	upgrade "cosmossdk.io/x/upgrade"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/stretchr/testify/require"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	vestingrewardstypes "github.com/zerone-chain/zerone/x/vesting_rewards/types"
	"testing"
)

const testToKInfo = `{"schema":"zerone.tok-feedback-v1/lineage/v1","h3_source":"335bb94f0fd54d3752dcb397263b7e84fb1116b4","h3_tree":"769f67f1cfa108be3d31cace7777cf954f731c42"}`

func copyToKH3Map() module.VersionMap {
	m := module.VersionMap{}
	for n, v := range tokFeedbackH3VersionMap {
		m[n] = v
	}
	return m
}

// Explicit synthetic state fixture, not accepted-H3 executable/handoff proof.
// It exercises the actual startup checks, SDK height dispatch and commit store.
func newToKStartupFixture(t *testing.T) (*ZeroneApp, sdk.Context, upgradetypes.Plan) {
	t.Helper()
	a := NewZeroneApp(log.NewNopLogger(), dbm.NewMemDB(), nil, false, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()))
	require.NoError(t, a.LoadLatestVersion())
	ctx := a.NewUncachedContext(false, cmtproto.Header{Height: 1})
	require.NoError(t, a.UpgradeKeeper.SetModuleVersionMap(ctx, copyToKH3Map()))
	a.VestingRewardsKeeper.InitGenesis(ctx, vestingrewardstypes.DefaultGenesis())
	p := knowledgetypes.DefaultParams()
	p.FactUseMaxPerConsumerEpoch = 0
	p.FactUseMaxPerEpoch = 0
	require.NoError(t, a.KnowledgeKeeper.SetParams(ctx, &p))
	seedSDK053IBC10CompletedTestLineage(t, a, ctx)
	plan := upgradetypes.Plan{Name: UpgradeNameToKFeedbackV1, Height: 5, Info: testToKInfo}
	require.NoError(t, a.UpgradeKeeper.ScheduleUpgrade(ctx, plan))
	require.NoError(t, a.UpgradeKeeper.DumpUpgradeInfoToDisk(plan.Height, plan))
	for h := int64(1); h <= 4; h++ {
		require.Equal(t, h, a.CommitMultiStore().Commit().Version)
	}
	ctx = a.NewUncachedContext(false, cmtproto.Header{Height: 4})
	return a, ctx, plan
}

func TestToKFeedbackFrozenFullVersionMap(t *testing.T) {
	a := NewZeroneApp(log.NewNopLogger(), dbm.NewMemDB(), nil, false, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()))
	require.NoError(t, requireToKFeedbackVersionMap(a.ModuleManager.GetVersionMap(), 7))
	for name := range tokFeedbackH3VersionMap {
		vm := copyToKH3Map()
		delete(vm, name)
		require.Error(t, requireToKFeedbackVersionMap(vm, 6))
		vm = copyToKH3Map()
		vm[name]++
		require.Error(t, requireToKFeedbackVersionMap(vm, 6))
	}
	vm := copyToKH3Map()
	vm["surprise"] = 0
	require.Error(t, requireToKFeedbackVersionMap(vm, 6))
}

func TestToKFeedbackHeightDispatchAndCommittedStartup(t *testing.T) {
	a, ctx, plan := newToKStartupFixture(t)
	require.NoError(t, a.ValidateSDK053IBC10StartupCoordination())
	require.NoError(t, a.ValidateToKFeedbackStartupCoordination())
	report, err := a.VerifyScheduledActivationPrestate()
	require.NoError(t, err)
	require.False(t, report.ActivationReady)
	require.Equal(t, "tok-feedback-source-only-h-minus-one", report.Scope)
	_, err = upgrade.PreBlocker(ctx.WithHeaderInfo(header.Info{Height: 4}), a.UpgradeKeeper)
	// SDK refuses a candidate with a registered handler one block before H;
	// the process must start at H-1 state but must not execute H-1 under new code.
	require.Error(t, err)
	vm, err := a.UpgradeKeeper.GetModuleVersionMap(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 6, vm["knowledge"])
	ctx = ctx.WithBlockHeight(plan.Height).WithHeaderInfo(header.Info{Height: plan.Height})
	_, err = upgrade.PreBlocker(ctx, a.UpgradeKeeper)
	require.NoError(t, err)
	require.Equal(t, int64(5), a.CommitMultiStore().Commit().Version)
	require.NoError(t, a.ValidateSDK053IBC10StartupCoordination())
	require.NoError(t, a.ValidateToKFeedbackStartupCoordination())
	vm, err = a.UpgradeKeeper.GetModuleVersionMap(ctx)
	require.NoError(t, err)
	require.NoError(t, requireToKFeedbackVersionMap(vm, 7))
	p, err := a.KnowledgeKeeper.GetParams(ctx)
	require.NoError(t, err)
	require.False(t, p.FactUseEnabled)
	require.EqualValues(t, 100, p.FactUseMaxPerConsumerEpoch)
	// Restart recognition cannot excuse loss of old H3 lineage.
	ctx.KVStore(a.keys["knowledge"]).Set(append([]byte{0x7f, 0x01}, []byte(sdk053IBC10UpgradeMarker)...), []byte("wrong"))
	require.Error(t, a.ValidateSDK053IBC10StartupCoordination())
	require.Error(t, a.ValidateToKFeedbackStartupCoordination())
}

func TestToKFeedbackStartupRejectsWrongPlanLineageAndEarlyV7(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*ZeroneApp, sdk.Context)
	}{
		{"early-v7", func(a *ZeroneApp, c sdk.Context) {
			m := copyToKH3Map()
			m["knowledge"] = 7
			require.NoError(t, a.UpgradeKeeper.SetModuleVersionMap(c, m))
		}},
		{"wrong-h3-marker", func(a *ZeroneApp, c sdk.Context) {
			c.KVStore(a.keys["knowledge"]).Set(append([]byte{0x7f, 0x01}, []byte(sdk053IBC10UpgradeMarker)...), []byte("migrated"))
		}},
		{"native-h3", func(a *ZeroneApp, c sdk.Context) {
			require.NoError(t, a.KnowledgeKeeper.WriteMigrationMarker(c, sdk053IBC10NativeMarker, "genesis"))
		}},
		{"wrong-local-plan", func(a *ZeroneApp, c sdk.Context) {
			require.NoError(t, a.UpgradeKeeper.DumpUpgradeInfoToDisk(5, upgradetypes.Plan{Name: UpgradeNameSDK053IBC10, Height: 5, Info: testToKInfo}))
		}},
		{"marker-before-H", func(a *ZeroneApp, c sdk.Context) {
			require.NoError(t, a.KnowledgeKeeper.WriteMigrationMarker(c, "migration_v7_complete", "true"))
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			a, c, _ := newToKStartupFixture(t)
			test.change(a, c)
			require.Error(t, a.ValidateToKFeedbackStartupCoordination())
		})
	}
}

func TestToKFeedbackRefusesLegacyRootsBeforeLoader(t *testing.T) {
	a, _, _ := newToKStartupFixture(t)
	oldDB, before := commitLegacySDK053IBC10Roots(t)
	a.db = oldDB
	require.NoError(t, a.UpgradeKeeper.DumpUpgradeInfoToDisk(2, upgradetypes.Plan{Name: UpgradeNameSDK053IBC10, Height: 2}))
	require.ErrorContains(t, a.RegisterStoreUpgrades(), "refuses legacy H1/H2/H3 stores before loader")
	names, err := committedLegacyStoreNames(oldDB)
	require.NoError(t, err)
	require.True(t, names[legacyCapabilityStoreKey])
	require.True(t, names[legacyIBCFeeStoreKey])
	require.EqualValues(t, 1, before.Version)
}

func TestToKFeedbackCandidateCannotExecuteHistoricalH3(t *testing.T) {
	a, ctx, _ := newToKStartupFixture(t)
	for _, name := range []string{UpgradeNameConsolidationSafetyV1, UpgradeNameFounderRenunciationV1, UpgradeNameSDK053IBC10, UpgradeNameAgenttoolSeamV1} {
		require.Error(t, requireToKFeedbackPlan(name))
	}
	// Direct application cannot use H3's broad RunMigrations to advance v7.
	err := a.UpgradeKeeper.ApplyUpgrade(ctx.WithBlockHeight(5), upgradetypes.Plan{Name: UpgradeNameSDK053IBC10, Height: 5, Info: "not-a-proof"})
	require.ErrorContains(t, err, "candidate cannot execute")
	vm, err := a.UpgradeKeeper.GetModuleVersionMap(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 6, vm["knowledge"])
}
