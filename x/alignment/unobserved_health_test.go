package alignment

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"

	corestore "cosmossdk.io/core/store"
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

	"github.com/zerone-chain/zerone/internal/activation"
	"github.com/zerone-chain/zerone/x/alignment/keeper"
	"github.com/zerone-chain/zerone/x/alignment/types"
)

type q1Store struct {
	corestore.KVStoreService
	reads     int
	fault     error
	dataFault error
}

type q1KV struct {
	corestore.KVStore
	owner *q1Store
}

func (s *q1Store) OpenKVStore(ctx context.Context) corestore.KVStore {
	return q1KV{s.KVStoreService.OpenKVStore(ctx), s}
}

func (s q1KV) Get(key []byte) ([]byte, error) {
	if bytes.Equal(key, types.UnobservedHealthEnabledKey) {
		s.owner.reads++
		if s.owner.fault != nil {
			return nil, s.owner.fault
		}
	}
	if bytes.Equal(key, types.ParamsKey) && s.owner.dataFault != nil {
		return nil, s.owner.dataFault
	}
	return s.KVStore.Get(key)
}

type q1Inputs struct {
	registered, active, domains uint64
	registryReads               int
	registryError               error
	halted                      bool
}

func (s *q1Inputs) GetTotalStaked(context.Context) *big.Int        { return new(big.Int).SetUint64(s.active) }
func (s *q1Inputs) GetTotalSupply(context.Context) *big.Int        { return big.NewInt(111) }
func (s *q1Inputs) GetActiveValidatorCount(context.Context) uint64 { return s.active }
func (s *q1Inputs) GetTargetValidatorCount(context.Context) uint64 { return 111 }
func (s *q1Inputs) GetDomainCount(context.Context) uint64          { return s.domains }
func (s *q1Inputs) IsHalted(context.Context) bool                  { return s.halted }
func (s *q1Inputs) GetRegisteredValidatorCount(context.Context) (uint64, error) {
	s.registryReads++
	return s.registered, s.registryError
}

type q1Fixture struct {
	ctx     sdk.Context
	key     *storetypes.KVStoreKey
	service *q1Store
	inputs  *q1Inputs
	k       keeper.Keeper
	am      AppModule
}

func newQ1Fixture(t *testing.T, on bool) q1Fixture {
	t.Helper()
	key := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	ms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())
	ctx := sdk.NewContext(ms, cmtproto.Header{Height: 100}, false, log.NewNopLogger())
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	service := &q1Store{KVStoreService: runtime.NewKVStoreService(key)}
	inputs := &q1Inputs{}
	k := keeper.NewKeeper(service, cdc, "authority", nil, inputs, inputs, inputs, inputs)
	genesis := types.DefaultGenesis()
	genesis.UnobservedHealthEnabled = false
	k.InitGenesis(ctx, genesis)
	if on {
		require.NoError(t, k.EnableUnobservedHealth(ctx))
	}
	service.reads = 0
	return q1Fixture{ctx, key, service, inputs, k, NewAppModule(cdc, k)}
}

func (f q1Fixture) snapshot(t *testing.T) map[string]string {
	t.Helper()
	it := f.ctx.KVStore(f.key).Iterator(nil, nil)
	defer func() { require.NoError(t, it.Close()) }()
	rows := map[string]string{}
	for ; it.Valid(); it.Next() {
		rows[string(it.Key())] = string(it.Value())
	}
	return rows
}

func q1Run(fn func(context.Context) error, ctx sdk.Context) (err error, fault any) {
	defer func() { fault = recover() }()
	err = fn(ctx)
	return
}

func TestUnobservedHealthOffMatchesFrozenLegacyStateEventsErrorsGas(t *testing.T) {
	for _, scenario := range []string{"empty", "inactive", "healthy", "transition", "disabled", "halted", "unscheduled", "zero-interval", "malformed-params", "read-error", "out-of-gas"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := newQ1Fixture(t, false), newQ1Fixture(t, false)
			for _, f := range []q1Fixture{a, b} {
				switch scenario {
				case "inactive":
					f.inputs.registered = 1
				case "healthy":
					f.inputs.registered, f.inputs.active, f.inputs.domains = 111, 111, 100
				case "transition":
					f.k.SetState(f.ctx, &types.AlignmentState{Enabled: true, PreviousCategory: types.CategoryHealthy})
				case "disabled":
					f.k.SetState(f.ctx, &types.AlignmentState{Enabled: false})
				case "halted":
					f.inputs.halted = true
				case "zero-interval":
					p := f.k.GetParams(f.ctx)
					p.ObservationIntervalBlocks = 0
					f.k.SetParams(f.ctx, p)
				case "malformed-params":
					f.ctx.KVStore(f.key).Set(types.ParamsKey, []byte("bad json"))
				case "read-error":
					f.service.dataFault = errors.New("synthetic params read failure")
				}
			}
			for _, height := range []int64{100, 150, 200} {
				if scenario == "unscheduled" {
					height++
				}
				ac := a.ctx.WithBlockHeight(height).WithGasMeter(storetypes.NewInfiniteGasMeter()).WithEventManager(sdk.NewEventManager())
				bc := b.ctx.WithBlockHeight(height).WithGasMeter(storetypes.NewInfiniteGasMeter()).WithEventManager(sdk.NewEventManager())
				if scenario == "out-of-gas" {
					ac, bc = ac.WithGasMeter(storetypes.NewGasMeter(1)), bc.WithGasMeter(storetypes.NewGasMeter(1))
				}
				a.service.reads, b.service.reads = 0, 0
				ae, ap := q1Run(a.am.EndBlock, ac)
				be, bp := q1Run(b.am.endBlockLegacy, bc)
				require.Equal(t, be, ae)
				require.Equal(t, bp, ap)
				require.Equal(t, bc.GasMeter().GasConsumed(), ac.GasMeter().GasConsumed())
				require.Equal(t, bc.EventManager().Events(), ac.EventManager().Events())
				require.Equal(t, b.snapshot(t), a.snapshot(t))
				require.Equal(t, 1, a.service.reads)
				require.Zero(t, b.service.reads)
				require.Zero(t, a.inputs.registryReads, "marker-off must not add registry reads")
			}
		})
	}
}

func TestUnobservedHealthMarkerFaultsBeforeEffects(t *testing.T) {
	for _, stage := range []string{"scheduled", "unscheduled", "disabled", "halted"} {
		for _, mode := range []string{"zero", "malformed", "extra-byte", "unreadable"} {
			t.Run(stage+"/"+mode, func(t *testing.T) {
				f := newQ1Fixture(t, false)
				if stage == "disabled" {
					f.k.SetState(f.ctx, &types.AlignmentState{Enabled: false})
				}
				if stage == "halted" {
					f.inputs.halted = true
				}
				cause := errors.New("synthetic marker read failure")
				switch mode {
				case "zero":
					f.ctx.KVStore(f.key).Set(types.UnobservedHealthEnabledKey, []byte{0})
				case "malformed":
					f.ctx.KVStore(f.key).Set(types.UnobservedHealthEnabledKey, []byte{2})
				case "extra-byte":
					f.ctx.KVStore(f.key).Set(types.UnobservedHealthEnabledKey, []byte{1, 0})
				case "unreadable":
					f.service.fault = cause
				}
				before := f.snapshot(t)
				ctx := f.ctx.WithGasMeter(storetypes.NewInfiniteGasMeter()).WithEventManager(sdk.NewEventManager())
				if stage == "unscheduled" {
					ctx = ctx.WithBlockHeight(101)
				}
				err := f.am.EndBlock(ctx)
				require.ErrorIs(t, err, activation.ErrMarkerFault)
				if mode == "unreadable" {
					require.ErrorIs(t, err, cause)
				}
				require.Equal(t, 1, f.service.reads)
				require.Zero(t, f.inputs.registryReads)
				require.Zero(t, ctx.GasMeter().GasConsumed())
				require.Empty(t, ctx.EventManager().Events())
				require.Equal(t, before, f.snapshot(t))
			})
		}
	}
}

func TestUnobservedHealthSelectionAndStoredPacing(t *testing.T) {
	for _, registered := range []uint64{0, 1} {
		t.Run(fmt.Sprint(registered), func(t *testing.T) {
			f := newQ1Fixture(t, true)
			f.inputs.registered = registered
			ctx := f.ctx.WithGasMeter(storetypes.NewInfiniteGasMeter()).WithEventManager(sdk.NewEventManager())
			require.NoError(t, f.am.EndBlock(ctx))
			require.Equal(t, 1, f.service.reads)
			require.Equal(t, 1, f.inputs.registryReads)
			wantCategory, wantCreation, wantAnalysis := types.CategoryNotObserved, types.BPS, types.BPS
			if registered > 0 {
				wantCategory, wantCreation, wantAnalysis = types.CategoryCritical, 500_000, 2_000_000
			}
			require.Equal(t, wantCategory, f.k.GetState(ctx).PreviousCategory)
			f.ctx.KVStore(f.key).Set(types.UnobservedHealthEnabledKey, []byte{2})
			f.service.reads, f.inputs.registryReads = 0, 0
			for i := 0; i < 3; i++ {
				creation, analysis := f.k.GetGlobalPacingMultiplier(ctx)
				require.Equal(t, wantCreation, creation)
				require.Equal(t, wantAnalysis, analysis)
			}
			require.Zero(t, f.service.reads, "gas-metered helpers must not read even a corrupt marker")
			require.Zero(t, f.inputs.registryReads)
		})
	}
}

func TestUnobservedHealthRegistryFailureBeforeEffects(t *testing.T) {
	f := newQ1Fixture(t, true)
	f.inputs.registryError = errors.New("synthetic registry iterator failure")
	before := f.snapshot(t)
	ctx := f.ctx.WithEventManager(sdk.NewEventManager())
	err := f.am.EndBlock(ctx)
	require.ErrorIs(t, err, f.inputs.registryError)
	require.ErrorIs(t, err, types.ErrHealthObservationFault)
	require.Empty(t, ctx.EventManager().Events())
	require.Equal(t, before, f.snapshot(t))
}

func TestUnobservedHealthActivationRetainsHistoryUntilScheduledObservation(t *testing.T) {
	f := newQ1Fixture(t, false)
	require.NoError(t, f.am.EndBlock(f.ctx))
	require.Equal(t, types.CategoryCritical, f.k.GetState(f.ctx).PreviousCategory)
	old, found := f.k.GetHealthIndex(f.ctx, 100)
	require.True(t, found)
	require.NoError(t, f.k.EnableUnobservedHealth(f.ctx))
	before := f.snapshot(t)
	ctx := f.ctx.WithBlockHeight(101).WithEventManager(sdk.NewEventManager())
	require.NoError(t, f.am.EndBlock(ctx))
	require.Equal(t, before, f.snapshot(t), "Enable does not recompute pacing; the carrier must select its activation prestate")
	require.Empty(t, ctx.EventManager().Events())
	ctx = ctx.WithBlockHeight(200)
	require.NoError(t, f.am.EndBlock(ctx))
	state := f.k.GetState(ctx)
	require.Equal(t, types.CategoryNotObserved, state.PreviousCategory)
	require.False(t, state.DegradedFrequencyActive)
	pacing, err := keeper.NewQueryServerImpl(f.k).GlobalPacing(ctx, &types.QueryGlobalPacingRequest{})
	require.NoError(t, err)
	require.Equal(t, "not_observed", pacing.HealthCategory, "fourth category reaches the real client query")
	require.Equal(t, types.BPS, pacing.CreationMultiplierBps)
	require.Equal(t, types.BPS, pacing.AnalysisMultiplierBps)
	c, a := f.k.GetGlobalPacingMultiplier(ctx)
	require.Equal(t, types.BPS, c)
	require.Equal(t, types.BPS, a)
	retained, found := f.k.GetHealthIndex(ctx, 100)
	require.True(t, found)
	require.Equal(t, old, retained)
}
