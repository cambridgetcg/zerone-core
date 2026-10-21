package keeper_test

import (
	"math/big"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	alignment "github.com/zerone-chain/zerone/x/alignment"
	"github.com/zerone-chain/zerone/x/alignment/types"
	stakingkeeper "github.com/zerone-chain/zerone/x/staking/keeper"
	stakingtypes "github.com/zerone-chain/zerone/x/staking/types"
)

// Pin the marker-off characterization. Q1's on-path matrix lives in
// unobserved_health_integration_test.go: active count and stake alone still
// cannot distinguish these two registries.
func TestInputSelectionEmptyRegistryLegacyBeforeQ1(t *testing.T) {
	empty := newInputHarness(t, true, true, false)
	inactive := newInputHarness(t, true, true, true)
	operator := sdk.AccAddress("input_operator______").String()
	val, found := inactive.staking.GetValidator(inactive.ctx, operator)
	require.True(t, found)
	val.IsActive = false
	inactive.staking.SetValidator(inactive.ctx, val)
	for i, h := range []inputHarness{empty, inactive} {
		var records int
		h.staking.IterateValidators(h.ctx, func(*stakingtypes.Validator) bool {
			records++
			return false
		})
		require.Equal(t, i, records, "the fixtures have different registry membership")
		h.vesting.SetTotalMinted(h.ctx, new(big.Int))
		adapter := stakingkeeper.NewAlignmentStakingAdapter(h.staking)
		require.Zero(t, adapter.GetTotalStaked(h.ctx).Sign())
		require.Zero(t, adapter.GetActiveValidatorCount(h.ctx))
		require.Equal(t, uint64(111), adapter.GetTargetValidatorCount(h.ctx))
		ctx := h.ctx.WithEventManager(sdk.NewEventManager())
		require.NoError(t, alignment.NewAppModule(h.cdc, h.align).EndBlock(ctx))
		obs, found := h.align.GetObservation(ctx, 100)
		require.True(t, found)
		require.Zero(t, obs.EconomicStability)
		require.Zero(t, obs.StakingRatio)
		health, found := h.align.GetHealthIndex(ctx, 100)
		require.True(t, found)
		require.Equal(t, types.CategoryCritical, health.Category)
		creation, analysis := h.align.GetGlobalPacingMultiplier(ctx)
		require.Equal(t, uint64(500_000), creation)
		require.Equal(t, uint64(2_000_000), analysis)
		var criticalEvents int
		for _, event := range ctx.EventManager().Events() {
			if event.Type == "zerone.alignment.network_health_critical" {
				criticalEvents++
			}
		}
		require.Zero(t, criticalEvents, "Q1 is off: preserve the missing first-critical event")
		t.Logf("LEGACY registry_records=%d active=0 bonded=0 ledger=0 economic=%d staking=%d composite=%d category=%s pacing=%d/%d first_critical_events=%d",
			records, obs.EconomicStability, obs.StakingRatio, health.CompositeScore, health.Category, creation, analysis, criticalEvents)
	}
}
