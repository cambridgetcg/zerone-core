package cross_stack_test

import (
	"context"
	"errors"
	"testing"

	"cosmossdk.io/core/header"
	corestore "cosmossdk.io/core/store"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/stretchr/testify/require"

	zeroneapp "github.com/zerone-chain/zerone/app"
	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	vestingkeeper "github.com/zerone-chain/zerone/x/vesting_rewards/keeper"
	vestingtypes "github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

func TestFounderRenunciationNamedUpgradeBoundary(t *testing.T) {
	h := NewTestHarness(t)
	h.Ctx = h.Ctx.WithHeaderInfo(header.Info{
		Height:  h.Height(),
		ChainID: testChainID,
	})
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
	marker, err := h.KnowledgeKeeper.ReadMigrationMarkerChecked(
		h.Ctx,
		"upgrade_marker_founder-renunciation-v1",
	)
	require.NoError(t, err)
	require.Equal(t, "migrated", marker)
	doneHeight, err := h.App.UpgradeKeeper.GetDoneHeight(
		h.Ctx,
		zeroneapp.UpgradeNameFounderRenunciationV1,
	)
	require.NoError(t, err)
	require.Equal(t, h.Height(), doneHeight,
		"activation proof includes x/upgrade's completed height")

	params := h.VestingRewardsKeeper.GetParams(h.Ctx)
	require.Zero(t, params.FounderShareBps)
	require.Empty(t, params.FounderAddress)
	require.Equal(t, "0", params.BlockReward)
	require.Equal(t, "0", params.FloorReward)
	require.Zero(t, params.EmptyBlockRewardRate)

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

func TestFounderRenunciationRejectsPreseededMarkerBeforeMigration(t *testing.T) {
	for _, markerValue := range []string{"forged", "migrated"} {
		t.Run(markerValue, func(t *testing.T) {
			h := NewTestHarness(t)
			require.NoError(t, h.KnowledgeKeeper.WriteMigrationMarker(
				h.Ctx,
				"upgrade_marker_founder-renunciation-v1",
				markerValue,
			))

			current := h.App.CurrentModuleVersionMap()
			fromVM := make(module.VersionMap, len(current))
			for name, version := range current {
				fromVM[name] = version
			}
			fromVM[vestingtypes.ModuleName] = 1

			_, err := h.App.RunUpgradeHandlerForTests(
				h.Ctx,
				zeroneapp.UpgradeNameFounderRenunciationV1,
				fromVM,
				h.Height(),
			)
			require.ErrorContains(t, err, "requires migration marker")
			require.ErrorContains(t, err, "to be absent before execution")

			storedMarker, readErr := h.KnowledgeKeeper.ReadMigrationMarkerChecked(
				h.Ctx,
				"upgrade_marker_founder-renunciation-v1",
			)
			require.NoError(t, readErr)
			require.Equal(t, markerValue, storedMarker,
				"the founder handler must not rewrite pre-existing evidence")

			storedVM, vmErr := h.App.UpgradeKeeper.GetModuleVersionMap(h.Ctx)
			require.NoError(t, vmErr)
			require.Equal(t, uint64(1), storedVM[vestingtypes.ModuleName],
				"marker refusal must happen before vesting_rewards migration")
			doneHeight, doneErr := h.App.UpgradeKeeper.GetDoneHeight(
				h.Ctx,
				zeroneapp.UpgradeNameFounderRenunciationV1,
			)
			require.NoError(t, doneErr)
			require.Zero(t, doneHeight,
				"a preseeded marker cannot manufacture x/upgrade completion proof")
			require.False(t,
				storedVM[vestingtypes.ModuleName] == 2 &&
					storedMarker == "migrated" &&
					doneHeight == h.Height(),
				"activation proof requires module v2, the exact marker, and the done height",
			)
		})
	}
}

func TestFounderRenunciationRejectsHistoricalEmptyMarkerBeforeMigration(t *testing.T) {
	h := NewTestHarness(t)
	h.App.KnowledgeKeeper = knowledgekeeper.NewKeeper(
		emptyMigrationMarkerStoreService{},
		nil,
		"",
		nil,
		nil,
	)

	current := h.App.CurrentModuleVersionMap()
	fromVM := make(module.VersionMap, len(current))
	for name, version := range current {
		fromVM[name] = version
	}
	fromVM[vestingtypes.ModuleName] = 1

	_, err := h.App.RunUpgradeHandlerForTests(
		h.Ctx,
		zeroneapp.UpgradeNameFounderRenunciationV1,
		fromVM,
		h.Height(),
	)
	require.ErrorContains(t, err, "requires migration marker")
	require.ErrorContains(t, err, "to be absent before execution")

	storedVM, vmErr := h.App.UpgradeKeeper.GetModuleVersionMap(h.Ctx)
	require.NoError(t, vmErr)
	require.Equal(t, uint64(1), storedVM[vestingtypes.ModuleName],
		"a present legacy empty marker must be refused before migration")
	doneHeight, doneErr := h.App.UpgradeKeeper.GetDoneHeight(
		h.Ctx,
		zeroneapp.UpgradeNameFounderRenunciationV1,
	)
	require.NoError(t, doneErr)
	require.Zero(t, doneHeight)
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

func TestFounderRenunciationRejectsUnknownModuleVersionEntry(t *testing.T) {
	h := NewTestHarness(t)
	current := h.App.CurrentModuleVersionMap()
	fromVM := make(module.VersionMap, len(current)+1)
	for name, version := range current {
		fromVM[name] = version
	}
	fromVM[vestingtypes.ModuleName] = 1
	fromVM["unknown-founder-sidecar"] = 1

	_, err := h.App.RunUpgradeHandlerForTests(
		h.Ctx,
		zeroneapp.UpgradeNameFounderRenunciationV1,
		fromVM,
		h.Height(),
	)
	require.ErrorContains(t, err, "refuses unknown module version entries")
	require.ErrorContains(t, err, "unknown-founder-sidecar")
	require.Empty(t, h.KnowledgeKeeper.ReadMigrationMarker(
		h.Ctx,
		"upgrade_marker_founder-renunciation-v1",
	))
	doneHeight, doneErr := h.App.UpgradeKeeper.GetDoneHeight(
		h.Ctx,
		zeroneapp.UpgradeNameFounderRenunciationV1,
	)
	require.NoError(t, doneErr)
	require.Zero(t, doneHeight)
}

type emptyMigrationMarkerStoreService struct{}

func (emptyMigrationMarkerStoreService) OpenKVStore(context.Context) corestore.KVStore {
	return emptyMigrationMarkerStore{}
}

type emptyMigrationMarkerStore struct{}

func (emptyMigrationMarkerStore) Get([]byte) ([]byte, error) {
	return []byte{}, nil
}

func (emptyMigrationMarkerStore) Has([]byte) (bool, error) {
	return true, nil
}

func (emptyMigrationMarkerStore) Set([]byte, []byte) error {
	return errors.New("empty migration marker test store is read-only")
}

func (emptyMigrationMarkerStore) Delete([]byte) error {
	return errors.New("empty migration marker test store is read-only")
}

func (emptyMigrationMarkerStore) Iterator([]byte, []byte) (corestore.Iterator, error) {
	return nil, errors.New("empty migration marker test store does not support iteration")
}

func (emptyMigrationMarkerStore) ReverseIterator([]byte, []byte) (corestore.Iterator, error) {
	return nil, errors.New("empty migration marker test store does not support reverse iteration")
}
