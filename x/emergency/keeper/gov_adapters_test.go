package keeper_test

import (
	"bytes"
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
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/emergency/keeper"
	"github.com/zerone-chain/zerone/x/emergency/types"
	govkeeper "github.com/zerone-chain/zerone/x/gov/keeper"
	govtypes "github.com/zerone-chain/zerone/x/gov/types"
)

// The fixture composes the real store-backed adapter with both real governance
// predicates. Stake eligibility is synthetic; no node, signer or bank is used.
func setupHaltReasonKeepers(t *testing.T) (keeper.Keeper, govkeeper.Keeper, sdk.Context, *storetypes.KVStoreKey) {
	t.Helper()
	emergencyKey := storetypes.NewKVStoreKey(types.StoreKey)
	govKey := storetypes.NewKVStoreKey(govtypes.StoreKey)
	db := dbm.NewMemDB()
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	ms.SetIAVLSyncPruning(true)
	ms.MountStoreWithDB(emergencyKey, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(govKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())
	ctx := sdk.NewContext(ms, cmtproto.Header{Height: 50_000_000}, false, log.NewNopLogger())
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	k := keeper.NewKeeper(runtime.NewKVStoreService(emergencyKey), cdc, "authority", nil)
	g := govkeeper.NewKeeper(cdc, govKey, "authority", nil, haltReasonGovStaking{})
	g.SetEmergencyKeeper(keeper.NewGovEmergencyAdapter(k))
	g.SetResearchFundGovernanceState(ctx, &govtypes.ResearchFundGovernanceState{
		CurrentPhase:             govtypes.ResearchFundPhase_RESEARCH_FUND_PHASE_BALANCED,
		PhaseStartedAtBlock:      40_000_000,
		ProposalsExecutedInPhase: 10,
	})
	for i := 0; i < 50; i++ {
		g.RecordDistinctVoter(ctx, fmt.Sprintf("fixture-voter-%d", i))
	}
	return k, g, ctx, emergencyKey
}

type haltReasonGovStaking struct{}

func (haltReasonGovStaking) GetTotalBondedStake(context.Context) (string, error) { return "0", nil }
func (haltReasonGovStaking) GetDelegatorTotalBonded(context.Context, string) (string, error) {
	return "0", nil
}
func (haltReasonGovStaking) CountActiveGuardians(context.Context) (uint64, error)  { return 22, nil }
func (haltReasonGovStaking) IsGuardian(context.Context, string) (bool, error)      { return false, nil }
func (haltReasonGovStaking) IsJailed(context.Context, string) (bool, error)        { return false, nil }
func (haltReasonGovStaking) GetSlashCount(context.Context, string) (uint64, error) { return 0, nil }

func storeHaltReasonCeremony(t *testing.T, k keeper.Keeper, ctx sdk.Context, id, reason, phase string, start uint64) {
	t.Helper()
	payload, err := proto.Marshal(&types.EmergencyHaltProposal{
		Id: id, Proposer: testCouncilAddr(1), Reason: reason, ProposedAtBlock: start,
	})
	require.NoError(t, err)
	require.NoError(t, k.SetCeremony(ctx, &types.EmergencyCeremony{
		Id: id, Type: string(types.CeremonyHalt), Phase: phase, StartBlock: start, ProposalData: payload,
	}))
}

func haltReasonStoreBytes(t *testing.T, ctx sdk.Context, key *storetypes.KVStoreKey) [][]byte {
	t.Helper()
	it := ctx.KVStore(key).Iterator(nil, nil)
	defer it.Close()
	var result [][]byte
	for ; it.Valid(); it.Next() {
		result = append(result, bytes.Clone(it.Key()), bytes.Clone(it.Value()))
	}
	return result
}

func TestGovEmergencyAdapterReasonAndPhaseWindow(t *testing.T) {
	for _, tc := range []struct {
		name, reason, phase string
		start               uint64
		legacy, enabled     uint64
	}{
		{"unrelated", "validator_key_compromise_unrelated_to_research_fund", "finalized", 45_000_000, 1, 0},
		{"exact", "research_fund", "finalized", 45_000_000, 1, 1},
		{"substring_is_not_match", "research_fund_misuse", "finalized", 45_000_000, 1, 0},
		{"case_is_not_match", "Research_Fund", "finalized", 45_000_000, 1, 0},
		{"whitespace_is_not_match", "research_fund ", "finalized", 45_000_000, 1, 0},
		{"empty_reason", "", "finalized", 45_000_000, 1, 0},
		{"prevote", "research_fund", "prevote", 45_000_000, 0, 0},
		{"precommit", "research_fund", "precommit", 45_000_000, 0, 0},
		{"failed", "research_fund", "failed", 45_000_000, 0, 0},
		{"old_matching", "research_fund", "finalized", 100, 1, 0},
		{"before_phase_boundary", "research_fund", "finalized", 39_999_999, 1, 0},
		{"at_phase_boundary", "research_fund", "finalized", 40_000_000, 1, 1},
		{"at_current_height", "research_fund", "finalized", 50_000_000, 1, 1},
		{"future_record", "research_fund", "finalized", 50_000_001, 1, 0},
		{"maximum_height_record", "research_fund", "finalized", ^uint64(0), 1, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			k, g, ctx, key := setupHaltReasonKeepers(t)
			storeHaltReasonCeremony(t, k, ctx, "halt", tc.reason, tc.phase, tc.start)
			for _, mode := range []struct {
				name  string
				count uint64
			}{{"legacy", tc.legacy}, {"enabled", tc.enabled}} {
				t.Run(mode.name, func(t *testing.T) {
					if mode.name == "enabled" {
						require.NoError(t, k.EnableHaltReasonFilter(ctx))
					}
					before := haltReasonStoreBytes(t, ctx, key)
					a := keeper.NewGovEmergencyAdapter(k)
					require.Equal(t, mode.count, a.CountHaltsForReason(ctx, "research_fund", 40_000_000))
					conditions, allMet := g.CheckPhaseExitConditions(ctx)
					require.Equal(t, mode.count, conditions.EmergencyHaltsFromMisuse)
					require.Equal(t, mode.count == 0, allMet, "all other exit conditions are satisfied")
					if mode.count == 0 {
						require.ErrorIs(t, g.ValidatePhaseRollback(ctx), govtypes.ErrNoRollbackJustification)
					} else {
						require.NoError(t, g.ValidatePhaseRollback(ctx))
					}
					require.Equal(t, before, haltReasonStoreBytes(t, ctx, key), "counts must not rewrite history")
				})
			}
		})
	}
}

func TestGovEmergencyAdapterWindowMovesWithPhase(t *testing.T) {
	k, g, ctx, _ := setupHaltReasonKeepers(t)
	storeHaltReasonCeremony(t, k, ctx, "old", "research_fund", "finalized", 100)
	require.NoError(t, k.EnableHaltReasonFilter(ctx))
	state := g.GetResearchFundGovernanceState(ctx)
	state.PhaseStartedAtBlock = 100
	g.SetResearchFundGovernanceState(ctx, state)
	// Age alone does not expire a crisis in the same phase. There is no invented
	// rolling TTL; the scope follows the retained phase boundary.
	require.NoError(t, g.ValidatePhaseRollback(ctx))
	conditions, allMet := g.CheckPhaseExitConditions(ctx)
	require.Equal(t, uint64(1), conditions.EmergencyHaltsFromMisuse)
	require.False(t, allMet)
	// The production rollback writer establishes the next phase's lower bound.
	g.ExecuteRollback(ctx)
	require.Equal(t, uint64(ctx.BlockHeight()), g.GetResearchFundGovernanceState(ctx).PhaseStartedAtBlock)
	require.ErrorIs(t, g.ValidatePhaseRollback(ctx), govtypes.ErrNoRollbackJustification)
	conditions, _ = g.CheckPhaseExitConditions(ctx)
	require.Zero(t, conditions.EmergencyHaltsFromMisuse)
}

func TestGovEmergencyAdapterMixedHistoryAndBounds(t *testing.T) {
	k, _, ctx, _ := setupHaltReasonKeepers(t)
	storeHaltReasonCeremony(t, k, ctx, "match-a", "research_fund", "finalized", 40_000_000)
	storeHaltReasonCeremony(t, k, ctx, "match-b", "research_fund", "finalized", 50_000_000)
	storeHaltReasonCeremony(t, k, ctx, "other", "validator_keys", "finalized", 45_000_000)
	storeHaltReasonCeremony(t, k, ctx, "old", "research_fund", "finalized", 100)
	storeHaltReasonCeremony(t, k, ctx, "prevote", "research_fund", "prevote", 45_000_000)
	for _, kind := range []types.CeremonyType{types.CeremonyResume, types.CeremonyRevert} {
		require.NoError(t, k.SetCeremony(ctx, &types.EmergencyCeremony{
			Id: string(kind), Type: string(kind), Phase: "finalized", StartBlock: 45_000_000,
		}))
	}
	a := keeper.NewGovEmergencyAdapter(k)
	require.Equal(t, uint64(4), a.CountHaltsForReason(ctx, "", ^uint64(0)), "legacy ignores both arguments")
	require.NoError(t, k.EnableHaltReasonFilter(ctx))
	require.Equal(t, uint64(2), a.CountHaltsForReason(ctx, "research_fund", 40_000_000))
	require.Equal(t, uint64(3), a.CountHaltsForReason(ctx, "research_fund", 0))
	require.Equal(t, uint64(1), a.CountHaltsForReason(ctx, "validator_keys", 40_000_000))
	require.Zero(t, a.CountHaltsForReason(ctx, "absent", 0))
	require.Zero(t, a.CountHaltsForReason(ctx, "", 0))
	require.Zero(t, a.CountHaltsForReason(ctx, "research_fund", 50_000_001))
	require.Zero(t, a.CountHaltsForReason(ctx, "research_fund", ^uint64(0)))
	require.Panics(t, func() { a.CountHaltsForReason(ctx.WithBlockHeight(-1), "research_fund", 0) })
}

func TestGovEmergencyAdapterActualCeremony(t *testing.T) {
	// Exercise the actual producer, two voting phases and stored proposal bytes,
	// separately from the deliberately small persisted-history matrix above.
	for _, reason := range []string{"research_fund", "validator_key_compromise_unrelated_to_research_fund"} {
		t.Run(reason, func(t *testing.T) {
			_, g, ctx, key := setupHaltReasonKeepers(t)
			mock := &mockStakingKeeper{}
			guardian := testCouncilAddr(1)
			mock.addGuardian(guardian, "100000000000")
			cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
			k := keeper.NewKeeper(runtime.NewKVStoreService(key), cdc, "authority", mock)
			params := types.DefaultParams()
			params.MinGuardianStake = "1"
			params.MinDistinctVoters = 1
			k.SetParams(ctx, &params)
			require.NoError(t, k.EnableHaltReasonFilter(ctx))
			g.SetEmergencyKeeper(keeper.NewGovEmergencyAdapter(k))
			ctx = types.WithAuthenticatedEmergencyTx(ctx)
			server := keeper.NewMsgServerImpl(k)
			response, err := server.ProposeHalt(ctx, &types.MsgProposeHalt{Proposer: guardian, Reason: reason})
			require.NoError(t, err)
			a := keeper.NewGovEmergencyAdapter(k)
			for _, phase := range []types.CeremonyPhase{types.PhasePrevote, types.PhasePrecommit} {
				ceremony, found := k.GetCeremony(ctx, response.ProposalId)
				require.True(t, found)
				require.Equal(t, string(phase), ceremony.Phase)
				require.Zero(t, a.CountHaltsForReason(ctx, "research_fund", 40_000_000))
				_, err = server.VoteHalt(ctx, &types.MsgVoteHalt{Voter: guardian, ProposalId: response.ProposalId, Approve: true})
				require.NoError(t, err)
			}
			ceremony, found := k.GetCeremony(ctx, response.ProposalId)
			require.True(t, found)
			require.Equal(t, string(types.PhaseFinalized), ceremony.Phase)
			if reason == "research_fund" {
				require.Equal(t, uint64(1), a.CountHaltsForReason(ctx, "research_fund", 40_000_000))
				require.NoError(t, g.ValidatePhaseRollback(ctx))
			} else {
				require.Zero(t, a.CountHaltsForReason(ctx, "research_fund", 40_000_000))
				require.ErrorIs(t, g.ValidatePhaseRollback(ctx), govtypes.ErrNoRollbackJustification)
			}
		})
	}
}
