package cross_stack_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/types/module"

	zeroneapp "github.com/zerone-chain/zerone/app"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	sponsorshiptypes "github.com/zerone-chain/zerone/x/sponsorship/types"
)

func cloneAgentEconomyVersionMap(source module.VersionMap) module.VersionMap {
	cloned := make(module.VersionMap, len(source))
	for name, version := range source {
		cloned[name] = version
	}
	return cloned
}

func TestUpgrade_ExecutablePlansCannotOwnReservedAgentEconomyTransitions(t *testing.T) {
	h := NewTestHarness(t)
	known := h.App.KnownUpgradeNames()
	require.NotEmpty(t, known)
	require.NotContains(t, known, zeroneapp.UpgradeNameAgentEconomyV1)
	require.False(
		t,
		h.App.UpgradeKeeper.HasHandler(zeroneapp.UpgradeNameAgentEconomyV1),
		"the reserved agent-economy plan must remain non-executable",
	)

	_, err := h.App.RunUpgradeHandlerForTests(
		h.Ctx,
		zeroneapp.UpgradeNameAgentEconomyV1,
		h.App.CurrentModuleVersionMap(),
		h.Height(),
	)
	require.ErrorContains(t, err, "no upgrade handler registered")

	target := h.App.CurrentModuleVersionMap()
	require.Equal(t, uint64(7), target[knowledgetypes.ModuleName])
	require.Equal(t, uint64(2), target[sponsorshiptypes.ModuleName])

	boundaries := []struct {
		name       string
		moduleName string
		from       uint64
		to         uint64
	}{
		{
			name:       "knowledge-v6-to-v7",
			moduleName: knowledgetypes.ModuleName,
			from:       6,
			to:         7,
		},
		{
			name:       "sponsorship-v1-to-v2",
			moduleName: sponsorshiptypes.ModuleName,
			from:       1,
			to:         2,
		},
	}

	require.Contains(t, known, zeroneapp.UpgradeNameSDK053IBC10)
	genericPlanCount := 0
	for _, planName := range known {
		require.Truef(
			t,
			h.App.UpgradeKeeper.HasHandler(planName),
			"known upgrade %q must be executable",
			planName,
		)
		if planName == zeroneapp.UpgradeNameSDK053IBC10 {
			continue
		}
		genericPlanCount++

		for _, boundary := range boundaries {
			boundary := boundary
			t.Run(planName+"/"+boundary.name, func(t *testing.T) {
				fromVM := cloneAgentEconomyVersionMap(target)
				fromVM[boundary.moduleName] = boundary.from

				beforeVM, readErr := h.App.UpgradeKeeper.GetModuleVersionMap(h.Ctx)
				require.NoError(t, readErr)
				beforeVM = cloneAgentEconomyVersionMap(beforeVM)

				toVM, runErr := h.App.RunUpgradeHandlerForTests(
					h.Ctx,
					planName,
					fromVM,
					h.Height(),
				)
				require.Error(t, runErr)
				require.Nil(t, toVM)
				require.Contains(
					t,
					runErr.Error(),
					fmt.Sprintf(
						"reserved transition %s %d->%d",
						boundary.moduleName,
						boundary.from,
						boundary.to,
					),
				)
				require.Contains(t, runErr.Error(), zeroneapp.UpgradeNameAgentEconomyV1)

				afterVM, readErr := h.App.UpgradeKeeper.GetModuleVersionMap(h.Ctx)
				require.NoError(t, readErr)
				require.Equal(
					t,
					beforeVM,
					afterVM,
					"reserved-boundary refusal must precede VersionMap seeding",
				)
			})
		}
	}
	require.Equal(t, len(known)-1, genericPlanCount)

	t.Run(zeroneapp.UpgradeNameSDK053IBC10+"/exact-source", func(t *testing.T) {
		seedPreSDKTransitionLineage(t, h)
		h.CommitHMinusOne()

		fromVM := sdk053IBC10SourceVM(h)
		require.Equal(t, uint64(6), fromVM[knowledgetypes.ModuleName])
		require.Equal(t, uint64(1), fromVM[sponsorshiptypes.ModuleName])

		planInfo, buildErr := zeroneapp.BuildSDK053IBC10PlanInfo(nil, nil)
		require.NoError(t, buildErr)
		toVM, runErr := h.App.RunUpgradeHandlerWithInfoForTests(
			h.Ctx,
			zeroneapp.UpgradeNameSDK053IBC10,
			fromVM,
			testH3ActivationHeight,
			planInfo,
		)
		require.ErrorContains(t, runErr, "requires frozen H3 target VersionMap")
		require.Nil(t, toVM)
		require.Empty(
			t,
			h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "migration_v7_complete"),
		)
		require.Empty(
			t,
			h.KnowledgeKeeper.ReadMigrationMarker(
				h.Ctx,
				knowledgetypes.AgentEconomyUpgradeMarker,
			),
		)
	})
}
