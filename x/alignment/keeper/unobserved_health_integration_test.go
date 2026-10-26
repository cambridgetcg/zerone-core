package keeper_test

import (
	"fmt"
	"testing"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	alignment "github.com/zerone-chain/zerone/x/alignment"
	"github.com/zerone-chain/zerone/x/alignment/types"
	stakingkeeper "github.com/zerone-chain/zerone/x/staking/keeper"
	stakingtypes "github.com/zerone-chain/zerone/x/staking/types"
)

func q1Events(ctx sdk.Context, eventType string) []sdk.Event {
	var result []sdk.Event
	for _, event := range ctx.EventManager().Events() {
		if event.Type == eventType {
			result = append(result, event)
		}
	}
	return result
}

func q1Attribute(t *testing.T, event sdk.Event, key string) string {
	t.Helper()
	for _, attr := range event.Attributes {
		if attr.Key == key {
			return attr.Value
		}
	}
	t.Fatalf("missing attribute %s", key)
	return ""
}

// Exercise real custom-staking, bank-supply and ontology adapters, not just
// mocks. Inactive and malformed occupied registries must not become unknown.
func TestUnobservedHealthRealRegistryAndInputMarkerMatrix(t *testing.T) {
	for _, bankSupply := range []bool{false, true} {
		for _, activeDomains := range []bool{false, true} {
			for _, registry := range []string{"empty", "inactive", "malformed"} {
				t.Run(fmt.Sprintf("supply=%t/domains=%t/%s", bankSupply, activeDomains, registry), func(t *testing.T) {
					h := newInputHarness(t, bankSupply, activeDomains, registry == "inactive")
					if registry == "inactive" {
						val, found := h.staking.GetValidator(h.ctx, sdk.AccAddress("input_operator______").String())
						require.True(t, found)
						val.IsActive = false
						h.staking.SetValidator(h.ctx, val)
					}
					if registry == "malformed" {
						h.ctx.KVStore(h.keys[stakingtypes.StoreKey]).Set(stakingtypes.ValidatorKey("malformed"), []byte("bad json"))
					}
					adapter := stakingkeeper.NewAlignmentStakingAdapter(h.staking)
					count, err := adapter.GetRegisteredValidatorCount(h.ctx)
					require.NoError(t, err)
					wantCount := uint64(1)
					if registry == "empty" {
						wantCount = 0
					}
					require.Equal(t, wantCount, count)
					require.Zero(t, adapter.GetActiveValidatorCount(h.ctx))
					require.Zero(t, adapter.GetTotalStaked(h.ctx).Sign())
					require.NoError(t, h.align.EnableUnobservedHealth(h.ctx))
					before := h.snapshot(t)
					wantCategory, creation, analysis := types.CategoryCritical, uint64(500_000), uint64(2_000_000)
					if registry == "empty" {
						wantCategory, creation, analysis = types.CategoryNotObserved, types.BPS, types.BPS
					}
					for _, height := range []int64{100, 200} {
						h.vs.reads, h.os.reads = 0, 0
						ctx := h.ctx.WithBlockHeight(height).WithGasMeter(storetypes.NewInfiniteGasMeter()).WithEventManager(sdk.NewEventManager())
						require.NoError(t, alignment.NewAppModule(h.cdc, h.align).EndBlock(ctx))
						require.Equal(t, 1, h.vs.reads)
						require.Equal(t, 1, h.os.reads)
						health, found := h.align.GetHealthIndex(ctx, uint64(height))
						require.True(t, found)
						require.Equal(t, wantCategory, health.Category)
						c, a := h.align.GetGlobalPacingMultiplier(ctx)
						require.Equal(t, creation, c)
						require.Equal(t, analysis, a)
						events := q1Events(ctx, "zerone.alignment.network_health_critical")
						if registry != "empty" && height == 100 {
							require.Len(t, events, 1, "born critical emits at the first genuine sample")
							require.Equal(t, "500000", q1Attribute(t, events[0], "creation_multiplier_bps"))
							require.Equal(t, "2000000", q1Attribute(t, events[0], "analysis_multiplier_bps"))
						} else {
							require.Empty(t, events)
						}
						observations := q1Events(ctx, "zerone.alignment.observation_recorded")
						require.Len(t, observations, 1)
						require.Equal(t, wantCategory, q1Attribute(t, observations[0], "category"))
						require.Equal(t, fmt.Sprint(creation), q1Attribute(t, observations[0], "creation_multiplier_bps"))
						require.Equal(t, registry != "empty", h.align.GetState(ctx).DegradedFrequencyActive)
					}
					after := h.snapshot(t)
					delete(before, types.StoreKey)
					delete(after, types.StoreKey)
					require.Equal(t, before, after, "Q1 must not rewrite input stores")
				})
			}
		}
	}
}

func TestUnobservedHealthRegistryTransitionsAndGenesisRoundTrip(t *testing.T) {
	h := newInputHarness(t, true, true, true)
	operator := sdk.AccAddress("input_operator______").String()
	val, found := h.staking.GetValidator(h.ctx, operator)
	require.True(t, found)
	val.IsActive = false
	h.staking.DeleteValidator(h.ctx, operator)
	require.NoError(t, h.align.EnableUnobservedHealth(h.ctx))
	am := alignment.NewAppModule(h.cdc, h.align)
	for _, step := range []struct {
		height   int64
		occupied bool
	}{{100, false}, {200, true}, {300, true}, {400, false}} {
		if step.occupied {
			h.staking.SetValidator(h.ctx, val)
		} else {
			h.staking.DeleteValidator(h.ctx, operator)
		}
		ctx := h.ctx.WithBlockHeight(step.height).WithEventManager(sdk.NewEventManager())
		require.NoError(t, am.EndBlock(ctx))
		want := types.CategoryNotObserved
		if step.occupied {
			want = types.CategoryCritical
		}
		require.Equal(t, want, h.align.GetState(ctx).PreviousCategory)
		critical := q1Events(ctx, "zerone.alignment.network_health_critical")
		if step.height == 200 {
			require.Len(t, critical, 1)
		} else {
			require.Empty(t, critical)
		}
		// Round-trip the actual module JSON. Stored category and pacing need no
		// schema extension, and import must not require a live registry read.
		raw := am.ExportGenesis(ctx, h.cdc)
		reloaded := newUnobservedHealthFixture(t)
		reloaded.initGenesis(t, raw)
		require.Equal(t, h.align.GetState(ctx), reloaded.k.GetState(reloaded.ctx))
		c, a := h.align.GetGlobalPacingMultiplier(ctx)
		rc, ra := reloaded.k.GetGlobalPacingMultiplier(reloaded.ctx)
		require.Equal(t, c, rc)
		require.Equal(t, a, ra)
	}
	old, found := h.align.GetHealthIndex(h.ctx, 200)
	require.True(t, found)
	require.Equal(t, types.CategoryCritical, old.Category, "later emptiness does not rewrite history")
}

func TestObservedHealthThresholdBoundaries(t *testing.T) {
	k, _, ctx := setupKeeper(t)
	params := k.GetParams(ctx)
	for _, previous := range []string{"", types.CategoryNotObserved, types.CategoryCritical, types.CategoryDegraded, types.CategoryHealthy} {
		for _, composite := range []uint64{0, params.CriticalThreshold - 1, params.CriticalThreshold, params.DegradedThreshold - 1, params.DegradedThreshold, params.HealthyThreshold - 1, params.HealthyThreshold, types.BPS} {
			want := types.CategoryDegraded
			if composite < params.CriticalThreshold {
				want = types.CategoryCritical
			} else if composite >= params.HealthyThreshold || (composite >= params.DegradedThreshold && previous == types.CategoryHealthy) {
				want = types.CategoryHealthy
			}
			require.Equal(t, want, k.CategorizeObservedHealth(ctx, composite, previous), "%s/%d", previous, composite)
		}
	}
	// A parameter change moves the entry boundary, not just correction magnitudes.
	params.DegradedThreshold = 600_000
	k.SetParams(ctx, params)
	require.Equal(t, types.CategoryDegraded, k.CategorizeObservedHealth(ctx, 500_000, types.CategoryHealthy))
	require.Equal(t, types.CategoryHealthy, k.CategorizeObservedHealth(ctx, 600_000, types.CategoryHealthy))
}
