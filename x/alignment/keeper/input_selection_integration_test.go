package keeper_test

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
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/internal/activation"
	alignment "github.com/zerone-chain/zerone/x/alignment"
	"github.com/zerone-chain/zerone/x/alignment/keeper"
	"github.com/zerone-chain/zerone/x/alignment/types"
	ontologykeeper "github.com/zerone-chain/zerone/x/ontology/keeper"
	ontologytypes "github.com/zerone-chain/zerone/x/ontology/types"
	stakingkeeper "github.com/zerone-chain/zerone/x/staking/keeper"
	stakingtypes "github.com/zerone-chain/zerone/x/staking/types"
	vestingkeeper "github.com/zerone-chain/zerone/x/vesting_rewards/keeper"
	vestingtypes "github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

// The wrappers instrument actual store reads, not adapter method calls. This
// catches a sensor/adapter that silently re-reads its marker after selection.
type inputMarkerStore struct {
	corestore.KVStoreService
	key          []byte
	reads        int
	fault        error
	dataFaultKey []byte
	dataFault    error
}

func (s *inputMarkerStore) OpenKVStore(ctx context.Context) corestore.KVStore {
	return inputMarkerKV{KVStore: s.KVStoreService.OpenKVStore(ctx), owner: s}
}

type inputMarkerKV struct {
	corestore.KVStore
	owner *inputMarkerStore
}

func (s inputMarkerKV) Get(key []byte) ([]byte, error) {
	if bytes.Equal(key, s.owner.key) {
		s.owner.reads++
		if s.owner.fault != nil {
			return nil, s.owner.fault
		}
	}
	if bytes.Equal(key, s.owner.dataFaultKey) && s.owner.dataFault != nil {
		return nil, s.owner.dataFault
	}
	return s.KVStore.Get(key)
}

type inputFlags struct{}

func (inputFlags) GetFlaggedDomainCount(context.Context) uint64 { return 1 }

type inputHarness struct {
	ctx      sdk.Context
	cdc      codec.Codec
	keys     map[string]*storetypes.KVStoreKey
	bank     bankkeeper.BaseKeeper
	vesting  vestingkeeper.Keeper
	ontology ontologykeeper.Keeper
	staking  stakingkeeper.Keeper
	align    keeper.Keeper
	supply   *vestingkeeper.AlignmentVestingRewardsAdapter
	domains  *ontologykeeper.AlignmentOntologyAdapter
	vs, os   *inputMarkerStore
}

// All supply/stake/domain and alignment stores/adapters are real and in-memory.
// Knowledge is unwired; a synthetic flag exercises the second domain-count
// consumer, which is intentionally unwired in the production app.
func newInputHarness(t *testing.T, bankSupply, activeDomains, registered bool) inputHarness {
	t.Helper()
	keys := storetypes.NewKVStoreKeys(types.StoreKey, vestingtypes.StoreKey,
		ontologytypes.StoreKey, stakingtypes.StoreKey, authtypes.StoreKey, banktypes.StoreKey)
	db := dbm.NewMemDB()
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	ms.SetIAVLSyncPruning(true)
	for _, key := range keys {
		ms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	}
	require.NoError(t, ms.LoadLatestVersion())
	ctx := sdk.NewContext(ms, cmtproto.Header{Height: 100}, false, log.NewNopLogger())
	registry := codectypes.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(registry)
	banktypes.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress("gov").String()
	prefix := sdk.GetConfig().GetBech32AccountAddrPrefix()
	accounts := authkeeper.NewAccountKeeper(cdc, runtime.NewKVStoreService(keys[authtypes.StoreKey]),
		authtypes.ProtoBaseAccount, map[string][]string{
			"input_fund": {authtypes.Minter, authtypes.Burner}, stakingtypes.ModuleName: nil,
		}, addresscodec.NewBech32Codec(prefix), prefix, authority)
	accounts.InitGenesis(ctx, *authtypes.DefaultGenesisState())
	bank := bankkeeper.NewBaseKeeper(cdc, runtime.NewKVStoreService(keys[banktypes.StoreKey]),
		accounts, map[string]bool{}, authority, log.NewNopLogger())
	bank.InitGenesis(ctx, banktypes.DefaultGenesisState())
	require.NoError(t, bank.MintCoins(ctx, "input_fund", sdk.NewCoins(sdk.NewInt64Coin("uzrn", 1_000_000_000))))
	vs := &inputMarkerStore{KVStoreService: runtime.NewKVStoreService(keys[vestingtypes.StoreKey]), key: vestingtypes.AlignmentBankSupplyEnabledKey}
	vesting := vestingkeeper.NewKeeper(cdc, vs, bank, nil, authority)
	vg := vestingtypes.DefaultGenesis()
	vg.AlignmentBankSupplyEnabled = false
	vesting.InitGenesis(ctx, vg)
	vesting.SetTotalMinted(ctx, big.NewInt(200_000_000))
	if bankSupply {
		require.NoError(t, vesting.EnableAlignmentBankSupply(ctx))
	}
	os := &inputMarkerStore{KVStoreService: runtime.NewKVStoreService(keys[ontologytypes.StoreKey]), key: ontologytypes.ActiveDomainCountEnabledKey}
	ontology := ontologykeeper.NewKeeper(cdc, os, bank, authority)
	og := ontologytypes.DefaultGenesis()
	og.ActiveDomainCountEnabled = false
	og.Domains = []*ontologytypes.Domain{
		{Name: "a", Status: "active"}, {Name: "b", Status: "active"},
		{Name: "c", Status: "proposed"}, {Name: "d", Status: "active_pending"},
	}
	ontology.InitGenesis(ctx, og)
	if activeDomains {
		require.NoError(t, ontology.EnableActiveDomainCount(ctx))
	}
	staking := stakingkeeper.NewKeeper(cdc, keys[stakingtypes.StoreKey], accounts, bank, authority)
	staking.InitGenesis(ctx, stakingtypes.DefaultGenesisState())
	if registered {
		operator := sdk.AccAddress("input_operator______")
		require.NoError(t, bank.SendCoinsFromModuleToAccount(ctx, "input_fund", operator,
			sdk.NewCoins(sdk.NewInt64Coin("uzrn", 100_000_000))))
		_, err := stakingkeeper.NewMsgServerImpl(staking).RegisterValidator(ctx, &stakingtypes.MsgRegisterValidator{
			Operator: operator.String(), ConsensusPubkey: "fixture-only", SelfDelegation: "100000000",
		})
		require.NoError(t, err)
	}
	supply := vestingkeeper.NewAlignmentVestingRewardsAdapter(vesting)
	domains := ontologykeeper.NewAlignmentOntologyAdapter(ontology)
	align := keeper.NewKeeper(runtime.NewKVStoreService(keys[types.StoreKey]), cdc, authority,
		nil, stakingkeeper.NewAlignmentStakingAdapter(staking), domains, nil, supply)
	align.SetCaptureDefenseKeeper(inputFlags{})
	ag := types.DefaultGenesis()
	ag.UnobservedHealthEnabled = false
	align.InitGenesis(ctx, ag)
	vs.reads, os.reads = 0, 0
	return inputHarness{ctx, cdc, keys, bank, vesting, ontology, staking, align, supply, domains, vs, os}
}

func (h inputHarness) snapshot(t *testing.T) map[string]map[string]string {
	t.Helper()
	rows := make(map[string]map[string]string)
	for name, key := range h.keys {
		rows[name] = make(map[string]string)
		it := h.ctx.KVStore(key).Iterator(nil, nil)
		for ; it.Valid(); it.Next() {
			rows[name][string(it.Key())] = string(it.Value())
		}
		require.NoError(t, it.Close())
	}
	return rows
}

func TestInputSelectionMarkerMatrix(t *testing.T) {
	for _, bankSupply := range []bool{false, true} {
		for _, activeDomains := range []bool{false, true} {
			t.Run(fmt.Sprintf("supply=%t/domains=%t", bankSupply, activeDomains), func(t *testing.T) {
				h := newInputHarness(t, bankSupply, activeDomains, true)
				before := h.snapshot(t)
				ctx := h.ctx.WithGasMeter(storetypes.NewInfiniteGasMeter()).WithEventManager(sdk.NewEventManager())
				require.NoError(t, alignment.NewAppModule(h.cdc, h.align).EndBlock(ctx))
				require.Equal(t, 1, h.vs.reads, "two supply sensors must share one marker read")
				require.Equal(t, 1, h.os.reads, "both domain consumers must share one marker read")
				obs, found := h.align.GetObservation(h.ctx, 100)
				require.True(t, found)
				wantRatio := uint64(500_000)
				if bankSupply {
					wantRatio = 100_000
				}
				count := uint64(4)
				if activeDomains {
					count = 2
				}
				require.Equal(t, wantRatio, obs.EconomicStability)
				require.Equal(t, wantRatio, obs.StakingRatio)
				require.Equal(t, count*types.BPS/100*700_000/types.BPS, obs.GovernanceParticipation)
				require.Equal(t, (types.BPS/111)*(types.BPS-types.BPS/count)/types.BPS, obs.NetworkSecurity)
				after := h.snapshot(t)
				delete(before, types.StoreKey)
				delete(after, types.StoreKey)
				require.Equal(t, before, after, "observations do not migrate input state")
			})
		}
	}
}

// These methods retain the pre-#122/#124 bodies from 77521a0d. Deliberately no
// marker capability: the reference run exercises the old adapter contract.
type inputLegacySupply struct{ k vestingkeeper.Keeper }

func (a inputLegacySupply) GetTotalSupply(ctx context.Context) *big.Int {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	_ = sdkCtx
	// Access the bank keeper through vesting_rewards to get total supply.
	// The vesting_rewards keeper wraps bankKeeper.GetSupply internally.
	// Since bankKeeper is private, we use GetTotalMinted as a proxy.
	totalMinted := a.k.GetTotalMinted(sdk.UnwrapSDKContext(ctx))
	if totalMinted == nil {
		return new(big.Int)
	}
	return totalMinted
}

type inputLegacyDomains struct{ k ontologykeeper.Keeper }

func (a inputLegacyDomains) GetDomainCount(ctx context.Context) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	domains := a.k.GetAllDomains(sdkCtx)
	return uint64(len(domains))
}

func TestInputSelectionOffMatchesLegacyStateEventsErrorsGas(t *testing.T) {
	for _, scenario := range []string{"observation", "zero-ledger-empty-registry", "disabled", "halted", "unscheduled", "legacy-read-error", "malformed-params", "out-of-gas"} {
		t.Run(scenario, func(t *testing.T) {
			a := newInputHarness(t, false, false, scenario != "zero-ledger-empty-registry")
			b := newInputHarness(t, false, false, scenario != "zero-ledger-empty-registry")
			legacy := keeper.NewKeeper(runtime.NewKVStoreService(b.keys[types.StoreKey]), b.cdc, "authority", nil,
				stakingkeeper.NewAlignmentStakingAdapter(b.staking), inputLegacyDomains{b.ontology}, nil, inputLegacySupply{b.vesting})
			legacy.SetCaptureDefenseKeeper(inputFlags{})
			if scenario == "halted" {
				a.align = keeper.NewKeeper(runtime.NewKVStoreService(a.keys[types.StoreKey]), a.cdc, "authority", nil,
					stakingkeeper.NewAlignmentStakingAdapter(a.staking), a.domains, &mockEmergencyKeeper{halted: true}, a.supply)
				legacy = keeper.NewKeeper(runtime.NewKVStoreService(b.keys[types.StoreKey]), b.cdc, "authority", nil,
					stakingkeeper.NewAlignmentStakingAdapter(b.staking), inputLegacyDomains{b.ontology}, &mockEmergencyKeeper{halted: true}, inputLegacySupply{b.vesting})
			}
			for _, h := range []inputHarness{a, b} {
				if scenario == "disabled" {
					state := h.align.GetState(h.ctx)
					state.Enabled = false
					h.align.SetState(h.ctx, state)
				}
				if scenario == "zero-ledger-empty-registry" {
					h.vesting.SetTotalMinted(h.ctx, new(big.Int))
				}
				if scenario == "legacy-read-error" {
					h.vs.dataFaultKey = vestingtypes.TotalMintedKey
					h.vs.dataFault = errors.New("synthetic non-marker read error")
				}
				if scenario == "malformed-params" {
					// Legacy GetParams falls back rather than introducing a halt.
					h.ctx.KVStore(h.keys[types.StoreKey]).Set(types.ParamsKey, []byte("bad json"))
				}
			}
			ac := a.ctx.WithGasMeter(storetypes.NewInfiniteGasMeter()).WithEventManager(sdk.NewEventManager())
			bc := b.ctx.WithGasMeter(storetypes.NewInfiniteGasMeter()).WithEventManager(sdk.NewEventManager())
			if scenario == "unscheduled" {
				ac, bc = ac.WithBlockHeight(101), bc.WithBlockHeight(101)
			}
			if scenario == "out-of-gas" {
				ac, bc = ac.WithGasMeter(storetypes.NewGasMeter(1)), bc.WithGasMeter(storetypes.NewGasMeter(1))
			}
			run := func(am alignment.AppModule, ctx sdk.Context) (err error, fault interface{}) {
				defer func() { fault = recover() }()
				err = am.EndBlock(ctx)
				return
			}
			ae, ap := run(alignment.NewAppModule(a.cdc, a.align), ac)
			be, bp := run(alignment.NewAppModule(b.cdc, legacy), bc)
			require.Equal(t, be, ae)
			require.Equal(t, bp, ap, "legacy panic/error handling must not change")
			require.Equal(t, bc.GasMeter().GasConsumed(), ac.GasMeter().GasConsumed())
			require.Equal(t, bc.EventManager().Events(), ac.EventManager().Events())
			require.Equal(t, b.snapshot(t), a.snapshot(t))
			require.Equal(t, 1, a.vs.reads)
			require.Equal(t, 1, a.os.reads)
			require.Zero(t, b.vs.reads)
			require.Zero(t, b.os.reads)
		})
	}
}

func TestInputSelectionFaultsBeforeWrites(t *testing.T) {
	for _, marker := range []string{"supply", "domains"} {
		for _, mode := range []string{"malformed", "unreadable"} {
			for _, stage := range []string{"scheduled", "unscheduled", "disabled"} {
				t.Run(marker+"/"+mode+"/"+stage, func(t *testing.T) {
					h := newInputHarness(t, false, false, true)
					if stage == "disabled" {
						h.align.SetState(h.ctx, &types.AlignmentState{Enabled: false})
					}
					service, key := h.vs, h.keys[vestingtypes.StoreKey]
					if marker == "domains" {
						service, key = h.os, h.keys[ontologytypes.StoreKey]
					}
					cause := errors.New("synthetic marker store read failure")
					if mode == "malformed" {
						h.ctx.KVStore(key).Set(service.key, []byte{2})
					} else {
						service.fault = cause
					}
					before := h.snapshot(t)
					ctx := h.ctx.WithGasMeter(storetypes.NewInfiniteGasMeter()).WithEventManager(sdk.NewEventManager())
					if stage == "unscheduled" {
						ctx = ctx.WithBlockHeight(101)
					}
					err := alignment.NewAppModule(h.cdc, h.align).EndBlock(ctx)
					require.ErrorIs(t, err, activation.ErrMarkerFault)
					if mode == "unreadable" {
						require.ErrorIs(t, err, cause)
					}
					require.Equal(t, before, h.snapshot(t))
					require.Empty(t, ctx.EventManager().Events())
					require.Zero(t, ctx.GasMeter().GasConsumed())
				})
			}
		}
	}
}

func TestInputSelectionStandaloneAndSnapshot(t *testing.T) {
	h := newInputHarness(t, false, false, true)
	require.Equal(t, big.NewInt(200_000_000), h.supply.GetTotalSupply(h.ctx))
	require.Equal(t, uint64(4), h.domains.GetDomainCount(h.ctx))
	require.Equal(t, 1, h.vs.reads)
	require.Equal(t, 1, h.os.reads)
	legacyCtx, err := h.align.SelectInputs(h.ctx)
	require.NoError(t, err)
	require.NoError(t, h.vesting.EnableAlignmentBankSupply(h.ctx))
	require.NoError(t, h.ontology.EnableActiveDomainCount(h.ctx))
	// Standalone calls see the markers; the earlier entry snapshot stays legacy.
	require.Equal(t, big.NewInt(1_000_000_000), h.supply.GetTotalSupply(h.ctx))
	require.Equal(t, uint64(2), h.domains.GetDomainCount(h.ctx))
	require.Equal(t, big.NewInt(200_000_000), h.supply.GetTotalSupply(legacyCtx))
	require.Equal(t, uint64(4), h.domains.GetDomainCount(legacyCtx))
	selected, err := h.align.SelectInputs(h.ctx)
	require.NoError(t, err)
	h.ctx.KVStore(h.keys[vestingtypes.StoreKey]).Set(vestingtypes.AlignmentBankSupplyEnabledKey, []byte{2})
	h.ctx.KVStore(h.keys[ontologytypes.StoreKey]).Set(ontologytypes.ActiveDomainCountEnabledKey, []byte{2})
	h.vs.reads, h.os.reads = 0, 0
	require.Equal(t, big.NewInt(1_000_000_000), h.supply.GetTotalSupply(selected))
	require.Equal(t, uint64(2), h.domains.GetDomainCount(selected))
	require.Zero(t, h.vs.reads)
	require.Zero(t, h.os.reads)
	for _, read := range []func(){func() { h.supply.GetTotalSupply(h.ctx) }, func() { h.domains.GetDomainCount(h.ctx) }} {
		func() {
			defer func() {
				fault, ok := recover().(error)
				require.True(t, ok, "standalone interface must not conceal a marker fault")
				require.ErrorIs(t, fault, activation.ErrMarkerFault)
			}()
			read()
		}()
	}
	// Even an incoming valid-looking snapshot cannot bypass entry-point faults.
	require.ErrorIs(t, alignment.NewAppModule(h.cdc, h.align).EndBlock(selected), activation.ErrMarkerFault)
}
