package app

import (
	"encoding/binary"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/core/header"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/stretchr/testify/require"
	"github.com/zerone-chain/zerone/internal/activation"
	"github.com/zerone-chain/zerone/internal/consensusrepairsmigration"
)

func TestConsensusRepairsLaterFailureDiscardsEnabledMarkers(t *testing.T) {
	for _, fault := range []string{"fourth marker faults", "unexpected migration write", "capture_challenge insert", "bank insert", "existing row delete", "same-length rewrite"} {
		t.Run(fault, func(t *testing.T) {
			app, ctx, _ := consensusRepairsFixture(t)
			// A non-marker, unselected row pins deletion and byte replacement,
			// independently of the separate knowledge 0x87 digest.
			existingKey := []byte("\xffpostcondition-existing")
			ctx.KVStore(app.keys["capture_challenge"]).Set(existingKey, []byte{1, 2})
			// Isolated configurator fault injection. All real production migrations
			// are guarded no-ops; simulate an accidental future side effect/error.
			cfg := module.NewConfigurator(app.appCodec, app.MsgServiceRouter(), app.GRPCQueryRouter())
			for name, version := range consensusRepairsVersions() {
				name := name
				require.NoError(t, cfg.RegisterMigration(name, version-1, func(cache sdk.Context) error {
					if err := consensusrepairsmigration.Require(cache); err != nil {
						return err
					}
					if name == "alignment" {
						switch fault {
						case "fourth marker faults":
							cache.KVStore(app.keys["qualification"]).Set([]byte{0x40}, []byte{2})
						case "unexpected migration write":
							cache.KVStore(app.keys["knowledge"]).Set([]byte{0x87}, []byte{2})
						case "capture_challenge insert":
							key := []byte("\xffunexpected-insert")
							require.False(t, cache.KVStore(app.keys["capture_challenge"]).Has(key))
							cache.KVStore(app.keys["capture_challenge"]).Set(key, []byte{3})
						case "bank insert":
							key := []byte("\xffunexpected-insert")
							require.False(t, cache.KVStore(app.keys["bank"]).Has(key))
							cache.KVStore(app.keys["bank"]).Set(key, []byte{3})
						case "existing row delete":
							require.True(t, cache.KVStore(app.keys["capture_challenge"]).Has(existingKey))
							cache.KVStore(app.keys["capture_challenge"]).Delete(existingKey)
						case "same-length rewrite":
							require.Len(t, cache.KVStore(app.keys["capture_challenge"]).Get(existingKey), 2)
							cache.KVStore(app.keys["capture_challenge"]).Set(existingKey, []byte{3, 4})
						}
					}
					return nil
				}))
			}
			app.configurator = cfg
			latest := app.CommitMultiStore().Commit().Version
			plan := upgradetypes.Plan{Name: UpgradeNameConsensusRepairsV1, Height: latest + 1}
			require.NoError(t, app.UpgradeKeeper.ScheduleUpgrade(ctx, plan))
			ctx = ctx.WithBlockHeight(plan.Height).WithHeaderInfo(header.Info{Height: plan.Height})
			before := claimIntegrityAllStores(t, app, ctx)
			err := app.UpgradeKeeper.ApplyUpgrade(ctx, plan)
			if fault == "fourth marker faults" {
				require.ErrorIs(t, err, activation.ErrMarkerFault)
			} else {
				require.ErrorContains(t, err, "unexpected store diff")
			}
			require.Equal(t, before, claimIntegrityAllStores(t, app, ctx))
			for _, marker := range app.ConsensusRepairsV1Markers() {
				on, err := marker.Enabled(ctx)
				require.NoError(t, err)
				require.False(t, on)
			}
		})
	}
}

func TestConsensusRepairsRejectsWrongSourcePlanAndHistory(t *testing.T) {
	for _, scenario := range []string{"wrong chain", "info", "time", "wrong source", "wrong compiled", "extra version", "receipt", "done", "claim marker absent", "predecessor receipt", "skip", "unscheduled", "wrong height"} {
		t.Run(scenario, func(t *testing.T) {
			var skip []int
			if scenario == "skip" {
				skip = []int{2}
			}
			app, ctx, _ := consensusRepairsFixture(t, skip...)
			switch scenario {
			case "wrong chain":
				ctx = ctx.WithChainID("zerone-1")
			case "wrong source":
				vm := consensusRepairsSourceVersionMap()
				vm["knowledge"] = 11
				require.NoError(t, app.UpgradeKeeper.SetModuleVersionMap(ctx, vm))
			case "wrong compiled":
				freezeClaimIntegrityCompiledTarget(app)
			case "extra version":
				require.NoError(t, app.UpgradeKeeper.SetModuleVersionMap(ctx, module.VersionMap{"unexpected": 1}))
			case "receipt":
				require.NoError(t, app.KnowledgeKeeper.WriteMigrationMarker(ctx, consensusRepairsReceiptKey, "{}"))
			case "done":
				writeSDK053IBC10TestDoneHeight(app, ctx, UpgradeNameConsensusRepairsV1, 1)
			case "claim marker absent":
				ctx.KVStore(app.keys["knowledge"]).Delete([]byte{0x87})
			case "predecessor receipt":
				require.NoError(t, app.KnowledgeKeeper.WriteMigrationMarker(ctx, "migration_v12_complete", "true"))
			}
			latest := app.CommitMultiStore().Commit().Version
			plan := upgradetypes.Plan{Name: UpgradeNameConsensusRepairsV1, Height: latest + 1}
			if scenario == "info" {
				plan.Info = "unexpected"
			}
			if scenario == "time" {
				plan.Time = time.Unix(1, 0)
			}
			// Schedule the ordinary plan; malformed direct invocation must not match it.
			if scenario != "unscheduled" && scenario != "done" {
				require.NoError(t, app.UpgradeKeeper.ScheduleUpgrade(ctx, upgradetypes.Plan{Name: plan.Name, Height: plan.Height}))
			}
			ctx = ctx.WithBlockHeight(plan.Height).WithHeaderInfo(header.Info{Height: plan.Height})
			if scenario == "wrong height" {
				ctx = ctx.WithBlockHeight(plan.Height + 1)
			}
			before := claimIntegrityAllStores(t, app, ctx)
			require.Error(t, app.UpgradeKeeper.ApplyUpgrade(ctx, plan))
			require.Equal(t, before, claimIntegrityAllStores(t, app, ctx))
		})
	}
}

func TestConsensusRepairsReceiptFaultsRefuseRestart(t *testing.T) {
	for _, scenario := range []string{"missing receipt", "missing done", "wrong height", "false marker readback", "missing readback", "census blocker", "missing census", "invalid census digest", "negative census count", "invalid digest", "wrong chain"} {
		t.Run(scenario, func(t *testing.T) {
			app, ctx, _ := consensusRepairsFixture(t)
			latest := app.CommitMultiStore().Commit().Version
			plan := upgradetypes.Plan{Name: UpgradeNameConsensusRepairsV1, Height: latest + 1}
			require.NoError(t, app.UpgradeKeeper.ScheduleUpgrade(ctx, plan))
			ctx = ctx.WithBlockHeight(plan.Height).WithHeaderInfo(header.Info{Height: plan.Height})
			require.NoError(t, app.UpgradeKeeper.ApplyUpgrade(ctx, plan))
			raw := app.KnowledgeKeeper.ReadMigrationMarker(ctx, consensusRepairsReceiptKey)
			var receipt consensusRepairsReceipt
			require.NoError(t, json.Unmarshal([]byte(raw), &receipt))
			switch scenario {
			case "missing receipt":
				ctx.KVStore(app.keys["knowledge"]).Delete(append([]byte{0x7f, 1}, []byte(consensusRepairsReceiptKey)...))
			case "missing done":
				key := append([]byte{upgradetypes.DoneByte}, binary.BigEndian.AppendUint64(nil, uint64(plan.Height))...)
				key = append(key, []byte(plan.Name)...)
				ctx.KVStore(app.keys["upgrade"]).Delete(key)
			case "wrong height":
				receipt.Height++
			case "false marker readback":
				receipt.Markers[0].Enabled = false
			case "missing readback":
				receipt.Markers = receipt.Markers[:9]
			case "census blocker":
				summary := receipt.Census.Censuses["reformulations"]
				summary.Blockers = 1
				receipt.Census.Censuses["reformulations"] = summary
			case "missing census":
				delete(receipt.Census.Censuses, "seat_runoffs")
			case "invalid census digest":
				summary := receipt.Census.Censuses["seat_runoffs"]
				summary.Digest = "not-a-digest"
				receipt.Census.Censuses["seat_runoffs"] = summary
			case "negative census count":
				summary := receipt.Census.Censuses["seat_runoffs"]
				summary.Records = -1
				receipt.Census.Censuses["seat_runoffs"] = summary
			case "invalid digest":
				receipt.Census.Digest = "not-a-digest"
			case "wrong chain":
				receipt.ChainID = "zerone-1"
			}
			if scenario != "missing receipt" && scenario != "missing done" {
				data, err := json.Marshal(receipt)
				require.NoError(t, err)
				// Simulate corrupted storage, not a legal marker rewrite. The real
				// WriteMigrationMarker correctly refuses conflicting receipts.
				ctx.KVStore(app.keys["knowledge"]).Set(append([]byte{0x7f, 1}, []byte(consensusRepairsReceiptKey)...), data)
			}
			app.CommitMultiStore().Commit()
			require.Error(t, app.ValidateAccountingAuthorityStartup())
		})
	}
}

func TestConsensusRepairsCensusBoundsAndDeterminism(t *testing.T) {
	app, ctx, _ := consensusRepairsFixture(t)
	a, err := app.CensusConsensusRepairsV1(ctx)
	require.NoError(t, err)
	b, err := app.CensusConsensusRepairsV1(ctx)
	require.NoError(t, err)
	require.Equal(t, a, b)
	cache, _ := ctx.CacheContext()
	cache.KVStore(app.keys["knowledge"]).Set([]byte("oversized-census-fixture"), []byte(strings.Repeat("x", consensusRepairsMaxBytes)))
	_, err = app.CensusConsensusRepairsV1(cache)
	require.NoError(t, err, "unrelated bytes have no whole-state ceiling")
	cache.KVStore(app.keys["knowledge"]).Set([]byte("\x62oversized-census-fixture"), []byte(strings.Repeat("x", consensusRepairsMaxBytes)))
	_, err = app.CensusConsensusRepairsV1(cache)
	require.ErrorContains(t, err, "resource ceiling")
	before := claimIntegrityAllStores(t, app, ctx)
	require.NotContains(t, before, "knowledge:oversized-census-fixture")
}
