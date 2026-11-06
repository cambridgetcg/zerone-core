package app

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"cosmossdk.io/core/header"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/stretchr/testify/require"
)

// Adapted from the independent CR2 handler-review probes supplied with CR2b.
// The scheduled plan itself carries Info, so plan equality cannot mask the guard.
func TestConsensusRepairsV2ScheduledInfoRefused(t *testing.T) {
	a, ctx, _ := consensusRepairsV2Fixture(t)
	plan := upgradetypes.Plan{Name: UpgradeNameConsensusRepairsV2, Height: a.CommitMultiStore().Commit().Version + 1, Info: "extra"}
	require.NoError(t, a.UpgradeKeeper.ScheduleUpgrade(ctx, plan))
	require.NoError(t, a.UpgradeKeeper.DumpUpgradeInfoToDisk(plan.Height, plan))
	ctx = ctx.WithBlockHeight(plan.Height).WithHeaderInfo(header.Info{Height: plan.Height})
	before := claimIntegrityAllStores(t, a, ctx)
	require.ErrorContains(t, a.ValidateAccountingAuthorityStartup(), "empty info")
	require.ErrorContains(t, a.UpgradeKeeper.ApplyUpgrade(ctx, plan), "empty info")
	require.Equal(t, before, claimIntegrityAllStores(t, a, ctx))
}

func TestConsensusRepairsV2PreEnabledMarkerRefused(t *testing.T) {
	for _, loc := range consensusRepairsV2MarkerLocations() {
		t.Run(fmt.Sprintf("%s/%x", loc.store, loc.key), func(t *testing.T) {
			a, ctx, _ := consensusRepairsV2Fixture(t)
			ctx.KVStore(a.keys[loc.store]).Set(loc.key, []byte{1})
			ctx, plan := scheduleRepairsV2(t, a, ctx)
			before := claimIntegrityAllStores(t, a, ctx)
			require.ErrorContains(t, a.ValidateAccountingAuthorityStartup(), "already present")
			require.ErrorContains(t, a.UpgradeKeeper.ApplyUpgrade(ctx, plan), "already present")
			require.Equal(t, before, claimIntegrityAllStores(t, a, ctx))
		})
	}
}

func TestConsensusRepairsV2TargetMarkerLossRefusesRestart(t *testing.T) {
	for _, loc := range consensusRepairsV2MarkerLocations() {
		t.Run(fmt.Sprintf("%s/%x", loc.store, loc.key), func(t *testing.T) {
			a, ctx, _ := consensusRepairsV2Fixture(t)
			ctx, plan := scheduleRepairsV2(t, a, ctx)
			require.NoError(t, a.UpgradeKeeper.ApplyUpgrade(ctx, plan))
			ctx.KVStore(a.keys[loc.store]).Delete(loc.key)
			a.CommitMultiStore().Commit()
			require.ErrorContains(t, a.ValidateAccountingAuthorityStartup(), "v2 target requires")
		})
	}
}

func TestConsensusRepairsV2EachMigrationGuarded(t *testing.T) {
	for _, mod := range []string{"alignment", "knowledge"} {
		t.Run(mod, func(t *testing.T) {
			a, ctx, _ := consensusRepairsV2Fixture(t)
			vm := consensusRepairsV2TargetVersionMap()
			vm[mod] = consensusRepairsV2SourceVersionMap()[mod]
			before := claimIntegrityAllStores(t, a, ctx)
			_, err := a.ModuleManager.RunMigrations(ctx, a.configurator, vm)
			require.ErrorContains(t, err, "explicit")
			require.Equal(t, before, claimIntegrityAllStores(t, a, ctx))
		})
	}
}

// Applied-v1 lineage fixture, not evidence of an actual dev-1 upgrade. The
// receipt/done tuple exercises the non-native route used by an upgraded source.
func TestConsensusRepairsV2AfterAppliedV1(t *testing.T) {
	for _, variant := range []string{"applied", "receipt-no-done", "done-no-receipt", "v1-receipt-blocker"} {
		t.Run(variant, func(t *testing.T) {
			a, ctx, _ := consensusRepairsV2Fixture(t)
			seedRepairsV2Calibration(t, a, ctx)
			v1h := a.CommitMultiStore().Commit().Version
			if variant != "receipt-no-done" {
				writeSDK053IBC10TestDoneHeight(a, ctx, UpgradeNameConsensusRepairsV1, uint64(v1h))
			}
			r := consensusRepairsReceipt{Height: v1h, ChainID: ctx.ChainID(), PlanDigest: consensusRepairsPlanDigest(upgradetypes.Plan{Name: UpgradeNameConsensusRepairsV1, Height: v1h})}
			r.Census = consensusRepairsCensusReceipt{Height: v1h, Digest: strings.Repeat("ab", 32), Censuses: map[string]consensusRepairsCensusSummary{}}
			for _, n := range consensusRepairsCensusNames {
				s := consensusRepairsCensusSummary{Digest: strings.Repeat("cd", 32)}
				if variant == "v1-receipt-blocker" && n == "halt_reasons" {
					s.Blockers = 1
				}
				r.Census.Censuses[n] = s
			}
			for _, m := range a.ConsensusRepairsV1Markers() {
				r.Markers = append(r.Markers, consensusRepairsMarkerReceipt{m.Module, m.Name, true})
			}
			raw, err := json.Marshal(r)
			require.NoError(t, err)
			if variant != "done-no-receipt" {
				require.NoError(t, a.KnowledgeKeeper.WriteMigrationMarker(ctx, consensusRepairsReceiptKey, string(raw)))
			}
			ctx, plan := scheduleRepairsV2(t, a, ctx)
			if variant != "applied" {
				before := claimIntegrityAllStores(t, a, ctx)
				require.Error(t, a.ValidateConsensusRepairsV2SourceEligibility(ctx))
				require.Error(t, a.ValidateAccountingAuthorityStartup())
				require.Error(t, a.UpgradeKeeper.ApplyUpgrade(ctx, plan))
				require.Equal(t, before, claimIntegrityAllStores(t, a, ctx))
				return
			}
			require.NoError(t, a.ValidateConsensusRepairsV2SourceEligibility(ctx))
			require.NoError(t, a.ValidateAccountingAuthorityStartup())
			require.NoError(t, a.UpgradeKeeper.ApplyUpgrade(ctx, plan))
			a.CommitMultiStore().Commit()
			require.NoError(t, a.ValidateAccountingAuthorityStartup())
			require.Equal(t, string(raw), a.KnowledgeKeeper.ReadMigrationMarker(ctx, consensusRepairsReceiptKey))
		})
	}
}
