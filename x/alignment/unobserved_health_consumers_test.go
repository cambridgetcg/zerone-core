package alignment

import (
	"context"
	"fmt"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/alignment/keeper"
	"github.com/zerone-chain/zerone/x/alignment/types"
	capturekeeper "github.com/zerone-chain/zerone/x/capture_defense/keeper"
	capturetypes "github.com/zerone-chain/zerone/x/capture_defense/types"
	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// The unchanged pacing body from 6a56b621 is the gas reference. The production
// adapter and both real consumers run against identical stores below.
type q1LegacyPacing struct{ k keeper.Keeper }

func (a q1LegacyPacing) GetGlobalPacingMultiplier(ctx context.Context) (uint64, uint64) {
	state := a.k.GetState(ctx)
	if state == nil || !state.Enabled {
		return types.BPS, types.BPS
	}
	switch state.PreviousCategory {
	case types.CategoryHealthy:
		return types.BPS, types.BPS
	case types.CategoryDegraded:
		return 750_000, 1_500_000
	case types.CategoryCritical:
		return 500_000, 2_000_000
	default:
		return types.BPS, types.BPS
	}
}

func TestUnobservedHealthActualPacingConsumersDoNotReadMarker(t *testing.T) {
	for _, on := range []bool{false, true} {
		for _, occupied := range []bool{false, true} {
			t.Run(fmt.Sprintf("marker=%t/occupied=%t", on, occupied), func(t *testing.T) {
				db := dbm.NewMemDB()
				t.Cleanup(func() { require.NoError(t, db.Close()) })
				keys := storetypes.NewKVStoreKeys(types.StoreKey, knowledgetypes.StoreKey, capturetypes.StoreKey)
				ms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
				for _, key := range keys {
					ms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
				}
				require.NoError(t, ms.LoadLatestVersion())
				ctx := sdk.NewContext(ms, cmtproto.Header{Height: 100}, false, log.NewNopLogger())
				cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
				service := &q1Store{KVStoreService: runtime.NewKVStoreService(keys[types.StoreKey])}
				inputs := &q1Inputs{}
				if occupied {
					inputs.registered = 1
				}
				ak := keeper.NewKeeper(service, cdc, "authority", nil, inputs, inputs, inputs, inputs)
				gs := types.DefaultGenesis()
				gs.UnobservedHealthEnabled = false
				ak.InitGenesis(ctx, gs)
				if on {
					require.NoError(t, ak.EnableUnobservedHealth(ctx))
				}
				require.NoError(t, NewAppModule(cdc, ak).EndBlock(ctx))
				kk := knowledgekeeper.NewKeeper(runtime.NewKVStoreService(keys[knowledgetypes.StoreKey]), cdc, "authority", nil, nil)
				kp := knowledgetypes.DefaultParams()
				kp.MinReviewFee = "100000"
				kp.ClaimCooldownBlocks = 50
				require.NoError(t, kk.SetParams(ctx, &kp))
				ck := capturekeeper.NewKeeper(runtime.NewKVStoreService(keys[capturetypes.StoreKey]), cdc, "authority")
				cp := capturetypes.DefaultParams()
				cp.DecayEpochBlocks = 0
				cp.RiskAnalysisInterval = 0
				ck.SetParams(ctx, cp)
				// Corrupt Q1 after EndBlock: no consumer may select it again.
				ctx.KVStore(keys[types.StoreKey]).Set(types.UnobservedHealthEnabledKey, []byte{2})
				service.reads, inputs.registryReads = 0, 0
				adapter := keeper.NewAlignmentPacingAdapter(ak)
				kk.SetPacingKeeper(adapter)
				ck.SetPacingKeeper(adapter)
				actual := ctx.WithGasMeter(storetypes.NewInfiniteGasMeter()).WithEventManager(sdk.NewEventManager())
				fee, cooldown := kk.GetEffectiveMinReviewFee(actual), kk.GetEffectiveCooldown(actual, "physics")
				require.NoError(t, ck.BeginBlocker(actual))
				reference := ctx.WithGasMeter(storetypes.NewInfiniteGasMeter()).WithEventManager(sdk.NewEventManager())
				kk.SetPacingKeeper(q1LegacyPacing{ak})
				ck.SetPacingKeeper(q1LegacyPacing{ak})
				require.Equal(t, kk.GetEffectiveMinReviewFee(reference), fee)
				require.Equal(t, kk.GetEffectiveCooldown(reference, "physics"), cooldown)
				require.NoError(t, ck.BeginBlocker(reference))
				require.Equal(t, reference.GasMeter().GasConsumed(), actual.GasMeter().GasConsumed())
				require.Equal(t, reference.EventManager().Events(), actual.EventManager().Events())
				require.Zero(t, service.reads)
				require.Zero(t, inputs.registryReads)
				if on && !occupied {
					require.Equal(t, "100000", fee)
					require.Equal(t, uint64(50), cooldown)
				} else {
					require.Equal(t, "200000", fee)
					require.Equal(t, uint64(100), cooldown)
				}
				t.Logf("fee=%s cooldown=%d caller_gas=%d legacy_gas=%d marker_reads=%d", fee, cooldown, actual.GasMeter().GasConsumed(), reference.GasMeter().GasConsumed(), service.reads)
			})
		}
	}
}

func TestUnobservedHealthEndBlockHysteresisEventsAndCadence(t *testing.T) {
	f := newQ1Fixture(t, true)
	f.inputs.registered = 1
	p := f.k.GetParams(f.ctx)
	// Isolate one controllable score; still exercise ObserveAll and EndBlock.
	p.WeightKnowledgeQuality, p.WeightEconomicStability, p.WeightGovernanceParticipation = 0, 0, types.BPS
	p.WeightNetworkSecurity, p.WeightStakingRatio = 0, 0
	p.CorrectionConfidenceMinSamples = 1_000_000 // isolate the health-frequency switch
	f.k.SetParams(f.ctx, p)
	for _, step := range []struct {
		height             int64
		domains            uint64
		occupied           bool
		category, event    string
		creation, analysis uint64
		accelerated        bool
	}{
		{100, 100, true, types.CategoryHealthy, "", types.BPS, types.BPS, false},
		{200, 80, true, types.CategoryHealthy, "", types.BPS, types.BPS, false},
		{300, 50, true, types.CategoryDegraded, "zerone.alignment.network_health_degraded", 750_000, 1_500_000, true},
		{350, 80, true, types.CategoryDegraded, "", 750_000, 1_500_000, true},
		{400, 0, true, types.CategoryCritical, "zerone.alignment.network_health_critical", 500_000, 2_000_000, true},
		{450, 100, true, types.CategoryHealthy, "zerone.alignment.network_health_recovered", types.BPS, types.BPS, false},
		{500, 0, true, types.CategoryCritical, "zerone.alignment.network_health_critical", 500_000, 2_000_000, true},
		{550, 0, false, types.CategoryNotObserved, "", types.BPS, types.BPS, false},
	} {
		f.inputs.domains = step.domains
		if step.occupied {
			f.inputs.registered = 1
		} else {
			f.inputs.registered = 0
		}
		ctx := f.ctx.WithBlockHeight(step.height).WithEventManager(sdk.NewEventManager())
		require.NoError(t, f.am.EndBlock(ctx))
		state := f.k.GetState(ctx)
		require.Equal(t, uint64(step.height), state.LastObservationHeight)
		require.Equal(t, step.category, state.PreviousCategory)
		require.Equal(t, step.accelerated, state.DegradedFrequencyActive)
		c, a := f.k.GetGlobalPacingMultiplier(ctx)
		require.Equal(t, step.creation, c)
		require.Equal(t, step.analysis, a)
		var transitionEvents []sdk.Event
		for _, event := range ctx.EventManager().Events() {
			if event.Type == "zerone.alignment.network_health_degraded" || event.Type == "zerone.alignment.network_health_critical" || event.Type == "zerone.alignment.network_health_recovered" {
				transitionEvents = append(transitionEvents, event)
			}
		}
		if step.event == "" {
			require.Empty(t, transitionEvents)
			continue
		}
		require.Len(t, transitionEvents, 1)
		require.Equal(t, step.event, transitionEvents[0].Type)
		attributes := map[string]string{}
		for _, attr := range transitionEvents[0].Attributes {
			attributes[attr.Key] = attr.Value
		}
		require.Equal(t, fmt.Sprint(step.creation), attributes["creation_multiplier_bps"])
		require.Equal(t, fmt.Sprint(step.analysis), attributes["analysis_multiplier_bps"])
	}
}
