package keeper_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	corestore "cosmossdk.io/core/store"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/internal/activation"
	"github.com/zerone-chain/zerone/x/emergency/keeper"
	"github.com/zerone-chain/zerone/x/emergency/types"
	govmodule "github.com/zerone-chain/zerone/x/gov"
	govkeeper "github.com/zerone-chain/zerone/x/gov/keeper"
	govtypes "github.com/zerone-chain/zerone/x/gov/types"
)

func TestHaltReasonMalformedEvidenceDoesNotPanic(t *testing.T) {
	wrongID, err := proto.Marshal(&types.EmergencyHaltProposal{Id: "wrong", Reason: types.HaltReasonResearchFund})
	require.NoError(t, err)
	for _, tc := range []struct {
		name    string
		payload []byte
	}{
		{"invalid_wire", []byte{0xff}}, {"absent", nil}, {"wrong_id", wrongID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			k, g, ctx, _ := setupHaltReasonKeepers(t)
			require.NoError(t, k.SetCeremony(ctx, &types.EmergencyCeremony{
				Id: "halt", Type: "halt", Phase: "finalized", StartBlock: 45_000_000, ProposalData: tc.payload,
			}))
			a := keeper.NewGovEmergencyAdapter(k)
			require.Equal(t, uint64(1), a.CountHaltsForReason(ctx, "ignored"))
			require.NoError(t, k.EnableHaltReasonScoping(ctx))
			selected := g.WithHaltReasonScoping(true)
			_, err := a.CountResearchFundHalts(ctx, 40_000_000)
			require.ErrorIs(t, err, types.ErrHaltEvidence)
			conditions, met := selected.CheckPhaseExitConditions(ctx)
			require.False(t, met)
			require.Equal(t, uint64(1), conditions.EmergencyHaltsFromMisuse)
			require.ErrorIs(t, selected.ValidatePhaseRollback(ctx), types.ErrHaltEvidence)
			count, err := a.CountResearchFundHalts(ctx, 46_000_000)
			require.NoError(t, err)
			require.Zero(t, count, "out-of-window payload is irrelevant")
			g.SetPhaseTransitionMeta(ctx, &govtypes.PhaseTransitionProposal{
				LipID: "pending", TargetPhase: govtypes.ResearchFundPhase_RESEARCH_FUND_PHASE_FULL_GOVERNANCE,
				Stage: govtypes.PhaseTransitionStagePending, ActivationBlock: uint64(ctx.BlockHeight()),
			})
			require.NoError(t, govmodule.NewAppModule(nil, g).BeginBlock(ctx))
			meta, found := g.GetPhaseTransitionMeta(ctx, "pending")
			require.True(t, found)
			require.Equal(t, govtypes.PhaseTransitionStageCancelled, meta.Stage)
		})
	}
}

func TestHaltReasonPendingRollbackRevalidated(t *testing.T) {
	for _, tc := range []struct {
		name, reason string
		start        uint64
		cancel       bool
	}{
		{"old", types.HaltReasonResearchFund, 100, true},
		{"unrelated_unknown", "validator_keys", 45_000_000, true},
		{"prose_unknown", "research_fund_misuse", 45_000_000, true},
		{"valid", types.HaltReasonResearchFund, 45_000_000, false},
	} {
		for _, enabled := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/enabled=%t", tc.name, enabled), func(t *testing.T) {
				k, g, ctx, _ := setupHaltReasonKeepers(t)
				storeHaltReasonCeremony(t, k, ctx, "halt", tc.reason, "finalized", tc.start)
				// Admission occurs under the actual legacy predicate.
				require.NoError(t, g.ValidatePhaseRollback(ctx))
				g.SetPhaseTransitionMeta(ctx, &govtypes.PhaseTransitionProposal{
					LipID: "rollback", TargetPhase: govtypes.ResearchFundPhase_RESEARCH_FUND_PHASE_OBSERVER,
					Stage: govtypes.PhaseTransitionStagePending, IsRollback: true, ActivationBlock: uint64(ctx.BlockHeight()),
				})
				if enabled {
					require.NoError(t, k.EnableHaltReasonScoping(ctx))
				}
				require.NoError(t, govmodule.NewAppModule(nil, g).BeginBlock(ctx))
				meta, _ := g.GetPhaseTransitionMeta(ctx, "rollback")
				state := g.GetResearchFundGovernanceState(ctx)
				if enabled && tc.cancel {
					require.Equal(t, govtypes.PhaseTransitionStageCancelled, meta.Stage)
					require.Equal(t, govtypes.ResearchFundPhase_RESEARCH_FUND_PHASE_BALANCED, state.CurrentPhase)
					require.NotEmpty(t, meta.CancelReason)
				} else {
					require.Equal(t, govtypes.PhaseTransitionStageActivated, meta.Stage)
					require.Equal(t, govtypes.ResearchFundPhase_RESEARCH_FUND_PHASE_OBSERVER, state.CurrentPhase)
				}
			})
		}
	}
}

type haltReadService struct {
	corestore.KVStoreService
	reads          int
	getErr, setErr error
}

func (s *haltReadService) OpenKVStore(ctx context.Context) corestore.KVStore {
	return haltReadStore{KVStore: s.KVStoreService.OpenKVStore(ctx), service: s}
}

type haltReadStore struct {
	corestore.KVStore
	service *haltReadService
}

func (s haltReadStore) Get(key []byte) ([]byte, error) {
	if bytes.Equal(key, types.HaltReasonScopingEnabledKey) {
		s.service.reads++
		if s.service.getErr != nil {
			return nil, s.service.getErr
		}
	}
	return s.KVStore.Get(key)
}
func (s haltReadStore) Set(key, value []byte) error {
	if bytes.Equal(key, types.HaltReasonScopingEnabledKey) && s.service.setErr != nil {
		return s.service.setErr
	}
	return s.KVStore.Set(key, value)
}

func haltEntry(g govkeeper.Keeper, ctx sdk.Context, entry string) (any, error) {
	switch entry {
	case "forward", "rollback", "text":
		category := govtypes.CategoryPhaseTransition
		if entry == "rollback" {
			category = govtypes.CategoryPhaseRollback
		}
		if entry == "text" {
			category = govtypes.CategoryText
		}
		return govkeeper.NewMsgServerImpl(g).SubmitLIP(ctx, &govtypes.MsgSubmitLIP{
			Title: "fixture", Proposer: testCouncilAddr(1), Category: category,
			Description: fmt.Sprintf(`{"target_phase":%d}`, govtypes.ResearchFundPhase_RESEARCH_FUND_PHASE_FULL_GOVERNANCE), InitialStake: "1000000000000",
		})
	case "query":
		return govkeeper.NewQueryServerImpl(g).ResearchFundGovernance(ctx, &govtypes.QueryResearchFundGovernanceRequest{})
	case "begin":
		return nil, govmodule.NewAppModule(nil, g).BeginBlock(ctx)
	default:
		panic("unknown test entry")
	}
}

func TestHaltReasonMarkerOnceBeforeEntryWrites(t *testing.T) {
	for _, mode := range []string{"off", "on", "malformed", "read_error"} {
		for _, entry := range []string{"forward", "rollback", "text", "query", "begin"} {
			t.Run(mode+"/"+entry, func(t *testing.T) {
				k, g, ctx, ek, gk := setupHaltReasonStores(t)
				if mode == "on" {
					require.NoError(t, k.EnableHaltReasonScoping(ctx))
				}
				if mode == "malformed" {
					ctx.KVStore(ek).Set(types.HaltReasonScopingEnabledKey, []byte{2})
				}
				service := &haltReadService{KVStoreService: runtime.NewKVStoreService(ek)}
				if mode == "read_error" {
					service.getErr = errors.New("injected read failure")
				}
				k = keeper.NewKeeper(service, nil, "authority", nil)
				g.SetEmergencyKeeper(keeper.NewGovEmergencyAdapter(k))
				beforeE, beforeG := haltReasonStoreBytes(t, ctx, ek), haltReasonStoreBytes(t, ctx, gk)
				ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter()).WithEventManager(sdk.NewEventManager())
				_, err := haltEntry(g, ctx, entry)
				require.Equal(t, 1, service.reads, "including SubmitLIP's two condition checks and all BeginBlock loops")
				if mode == "malformed" || mode == "read_error" {
					require.ErrorIs(t, err, activation.ErrMarkerFault)
					require.Zero(t, ctx.GasMeter().GasConsumed())
					require.Empty(t, ctx.EventManager().Events())
					require.Equal(t, beforeE, haltReasonStoreBytes(t, ctx, ek))
					require.Equal(t, beforeG, haltReasonStoreBytes(t, ctx, gk))
				} else if entry == "rollback" {
					require.ErrorIs(t, err, govtypes.ErrNoRollbackJustification)
				} else {
					require.NoError(t, err)
				}
			})
		}
	}
}

// The reference adapter invokes the verbatim 77521a0d counting body, without
// the new reader. Real entry points must be identical when its marker is absent.
type haltLegacyReference struct{ *keeper.GovEmergencyAdapter }

func (haltLegacyReference) HaltReasonScopingEnabled(context.Context) (bool, error) { return false, nil }

func TestHaltReasonOffEntryParity(t *testing.T) {
	for _, entry := range []string{"forward", "rollback", "text", "query", "begin"} {
		for _, withHalt := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/halt=%t", entry, withHalt), func(t *testing.T) {
				type result struct {
					Response, Error string
					Events          sdk.Events
					Gas             uint64
					Emergency, Gov  [][]byte
				}
				run := func(reference bool) result {
					k, g, ctx, ek, gk := setupHaltReasonStores(t)
					if withHalt {
						storeHaltReasonCeremony(t, k, ctx, "old", "unrelated prose", "finalized", 100)
					}
					if entry == "begin" {
						g.SetPhaseTransitionMeta(ctx, &govtypes.PhaseTransitionProposal{LipID: "pending", Stage: govtypes.PhaseTransitionStagePending, ActivationBlock: uint64(ctx.BlockHeight()), IsRollback: withHalt, TargetPhase: govtypes.ResearchFundPhase_RESEARCH_FUND_PHASE_OBSERVER})
					}
					if reference {
						g.SetEmergencyKeeper(haltLegacyReference{keeper.NewGovEmergencyAdapter(k)})
					}
					ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter()).WithEventManager(sdk.NewEventManager())
					response, err := haltEntry(g, ctx, entry)
					r := result{Gas: ctx.GasMeter().GasConsumed(), Events: ctx.EventManager().Events()}
					if err != nil {
						r.Error = err.Error()
					}
					bz, marshalErr := json.Marshal(response)
					require.NoError(t, marshalErr)
					r.Response = string(bz)
					r.Emergency, r.Gov = haltReasonStoreBytes(t, ctx, ek), haltReasonStoreBytes(t, ctx, gk)
					return r
				}
				want, got := run(true), run(false)
				require.Equal(t, want, got, "state, events, errors, results and gas")
				t.Logf("legacy/off gas=%d entry=%s halt=%t", got.Gas, entry, withHalt)
			})
		}
	}
}

func TestHaltReasonEnableOnlyMarkerAndCacheIsolation(t *testing.T) {
	k, g, ctx, ek, gk := setupHaltReasonStores(t)
	storeHaltReasonCeremony(t, k, ctx, "old", "prose", "finalized", 100)
	beforeE, beforeG := haltReasonStoreBytes(t, ctx, ek), haltReasonStoreBytes(t, ctx, gk)
	cache, write := ctx.CacheContext()
	require.NoError(t, k.EnableHaltReasonScoping(cache))
	on, err := k.HaltReasonScopingEnabled(ctx)
	require.NoError(t, err)
	require.False(t, on)
	write()
	require.Equal(t, append(beforeE, types.HaltReasonScopingEnabledKey, []byte{1}), haltReasonStoreBytes(t, ctx, ek))
	require.Equal(t, beforeG, haltReasonStoreBytes(t, ctx, gk))
	require.NoError(t, k.EnableHaltReasonScoping(ctx))
	require.NoError(t, g.RequireNoEmergencyTransitionHold(ctx))
	_, _, err = g.EnsureEmergencyTransitionHold(ctx, "fixture-incident")
	require.NoError(t, err)
	_, err = haltEntry(g, ctx, "forward")
	require.ErrorIs(t, err, govtypes.ErrEmergencyTransitionHold)
}

func TestHaltReasonEnableDoesNotRepairFaults(t *testing.T) {
	for _, mode := range []string{"malformed", "read", "write"} {
		t.Run(mode, func(t *testing.T) {
			_, _, ctx, key := setupHaltReasonKeepers(t)
			service := &haltReadService{KVStoreService: runtime.NewKVStoreService(key)}
			failure := errors.New("injected marker failure")
			if mode == "read" {
				service.getErr = failure
			}
			if mode == "write" {
				service.setErr = failure
			}
			if mode == "malformed" {
				ctx.KVStore(key).Set(types.HaltReasonScopingEnabledKey, []byte{2})
			}
			k := keeper.NewKeeper(service, nil, "authority", nil)
			before := haltReasonStoreBytes(t, ctx, key)
			err := k.EnableHaltReasonScoping(ctx)
			require.ErrorIs(t, err, activation.ErrMarkerFault)
			if mode != "malformed" {
				require.ErrorIs(t, err, failure)
			}
			require.Equal(t, before, haltReasonStoreBytes(t, ctx, key))
		})
	}
}
