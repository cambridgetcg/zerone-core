package cross_stack_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/stretchr/testify/require"

	zeroneapp "github.com/zerone-chain/zerone/app"
	vestingkeeper "github.com/zerone-chain/zerone/x/vesting_rewards/keeper"
	vestingtypes "github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

func TestFounderRenunciationNamedUpgradeBoundary(t *testing.T) {
	h := NewTestHarness(t)
	require.NoError(t, h.VestingRewardsKeeper.SetParams(h.Ctx, vestingtypes.DefaultParams()))

	current := h.App.CurrentModuleVersionMap()
	fromVM := make(module.VersionMap, len(current))
	for name, version := range current {
		fromVM[name] = version
	}
	fromVM[vestingtypes.ModuleName] = 1

	toVM, err := h.App.RunUpgradeHandlerForTests(
		h.Ctx,
		zeroneapp.UpgradeNameFounderRenunciationV1,
		fromVM,
		h.Height(),
	)
	require.NoError(t, err)
	require.Equal(t, uint64(2), toVM[vestingtypes.ModuleName])
	require.Equal(t, "migrated", h.KnowledgeKeeper.ReadMigrationMarker(
		h.Ctx,
		"upgrade_marker_founder-renunciation-v1",
	))

	params := h.VestingRewardsKeeper.GetParams(h.Ctx)
	require.Zero(t, params.FounderShareBps)
	require.Empty(t, params.FounderAddress)

	// Ordinary parameter governance cannot restore the retired recipient after
	// the named migration. A rejected update must leave stored state unchanged.
	proposed := vestingtypes.DefaultParams()
	proposed.FounderShareBps = 1
	proposed.FounderAddress = "zrn1attemptedrecipient"
	server := vestingkeeper.NewMsgServerImpl(h.VestingRewardsKeeper)
	_, err = server.UpdateParams(h.Ctx, &vestingtypes.MsgUpdateParams{
		Authority: h.VestingRewardsKeeper.GetAuthority(),
		Params:    proposed,
	})
	require.Error(t, err)
	after := h.VestingRewardsKeeper.GetParams(h.Ctx)
	require.Zero(t, after.FounderShareBps)
	require.Empty(t, after.FounderAddress)
}

func TestFounderRenunciationUpgradeIsAdvertisedLast(t *testing.T) {
	h := NewTestHarness(t)
	names := h.App.KnownUpgradeNames()
	require.NotEmpty(t, names)
	require.Equal(t, zeroneapp.UpgradeNameFounderRenunciationV1, names[len(names)-1])
	require.True(t, h.App.UpgradeKeeper.HasHandler(zeroneapp.UpgradeNameFounderRenunciationV1))
}

func TestFounderRenunciationCannotRideAnUnrelatedUpgrade(t *testing.T) {
	h := NewTestHarness(t)
	require.NoError(t, h.VestingRewardsKeeper.SetParams(h.Ctx, vestingtypes.DefaultParams()))

	current := h.App.CurrentModuleVersionMap()
	fromVM := make(module.VersionMap, len(current))
	for name, version := range current {
		fromVM[name] = version
	}
	fromVM[vestingtypes.ModuleName] = 1

	_, err := h.App.RunUpgradeHandlerForTests(
		h.Ctx,
		zeroneapp.UpgradeNameConsolidationSafetyV1,
		fromVM,
		h.Height(),
	)
	require.ErrorContains(t, err, "cannot activate vesting_rewards v2")
	require.Zero(t, h.VestingRewardsKeeper.GetParams(h.Ctx).FounderShareBps)
	require.Empty(t, h.VestingRewardsKeeper.GetParams(h.Ctx).FounderAddress)
}

func TestFounderRenunciationRejectsAlreadyMigratedPrestateWithoutMarker(t *testing.T) {
	h := NewTestHarness(t)
	current := h.App.CurrentModuleVersionMap()

	_, err := h.App.RunUpgradeHandlerForTests(
		h.Ctx,
		zeroneapp.UpgradeNameFounderRenunciationV1,
		current,
		h.Height(),
	)
	require.ErrorContains(t, err, "requires exact prestate vesting_rewards=1")
	require.Empty(t, h.KnowledgeKeeper.ReadMigrationMarker(
		h.Ctx,
		"upgrade_marker_founder-renunciation-v1",
	))
}

func TestFounderRenunciationRejectsUnrelatedModuleCatchUpWithoutMarker(t *testing.T) {
	h := NewTestHarness(t)
	current := h.App.CurrentModuleVersionMap()
	fromVM := make(module.VersionMap, len(current))
	for name, version := range current {
		fromVM[name] = version
	}
	fromVM[vestingtypes.ModuleName] = 1
	require.Greater(t, fromVM["knowledge"], uint64(0))
	fromVM["knowledge"]--

	_, err := h.App.RunUpgradeHandlerForTests(
		h.Ctx,
		zeroneapp.UpgradeNameFounderRenunciationV1,
		fromVM,
		h.Height(),
	)
	require.ErrorContains(t, err, "refuses unrelated migration")
	require.ErrorContains(t, err, "knowledge")
	require.Empty(t, h.KnowledgeKeeper.ReadMigrationMarker(
		h.Ctx,
		"upgrade_marker_founder-renunciation-v1",
	))
}
