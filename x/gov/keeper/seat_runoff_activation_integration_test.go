package keeper_test

import (
	"encoding/json"
	"fmt"
	"testing"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/gov/types"
)

type seatTallyOutcome struct {
	State  []byte
	Events sdk.Events
	Gas    uint64
	Panic  string
}

func TestSeatRunoffGenesisDispatch(t *testing.T) {
	for _, ordered := range []bool{false, true} {
		t.Run(fmt.Sprintf("ordered=%t", ordered), func(t *testing.T) {
			f := newSeatRunoffReplayFixture(t, []uint32{0}, false)
			genesis := types.DefaultGenesisState()
			// DefaultGenesis selects new-chain behavior. A pre-carrier fixture
			// must explicitly turn this off rather than inherit that default.
			genesis.SeatRunoffOrderingEnabled = ordered
			genesis.ResearchFundGovernance = f.keeper.GetResearchFundGovernanceState(f.ctx)
			genesis.SeatElections = f.keeper.GetAllSeatElections(f.ctx)
			genesis.SeatElectionVotes = f.keeper.GetAllSeatElectionVotes(f.ctx)
			genesis.NextSeatElectionNumber = f.keeper.GetNextSeatElectionID(f.ctx)
			raw, err := json.Marshal(genesis)
			require.NoError(t, err)
			f.module.InitGenesis(f.ctx, nil, raw)
			enabled, err := f.keeper.SeatRunoffOrderingEnabled(f.ctx)
			require.NoError(t, err)
			require.Equal(t, ordered, enabled)
			ctx := f.replayContext()
			require.NoError(t, f.module.BeginBlock(ctx))
			f.checkRunoffs(t, ctx, [][2]uint64{{3, 4}})
		})
	}
}

func observeSeatTally(t *testing.T, f seatRunoffReplayFixture, ctx sdk.Context, tally func(sdk.Context)) seatTallyOutcome {
	t.Helper()
	var outcome seatTallyOutcome
	func() {
		defer func() {
			if p := recover(); p != nil {
				outcome.Panic = fmt.Sprintf("%T: %v", p, p)
			}
		}()
		tally(ctx)
	}()
	outcome.Gas = ctx.GasMeter().GasConsumed()
	outcome.Events = ctx.EventManager().Events()
	outcome.State = seatRunoffStoreBytes(t, ctx.WithGasMeter(storetypes.NewInfiniteGasMeter()), f.key)
	return outcome
}

// The reference is the old executable tally, not an expected result derived
// from the new dispatcher. Compare bytes, ordered events, gas, and panic/error
// values (the old block hook has no error return), including out-of-gas.
func TestSeatRunoffMarkerOffMatchesLegacy(t *testing.T) {
	for _, name := range []string{"not_due", "single_pass", "single_fail", "contested_clear", "contested_fail", "create_runoff", "resolve_runoff", "runoff_fail", "out_of_gas"} {
		t.Run(name, func(t *testing.T) {
			f := newSeatRunoffReplayFixture(t, []uint32{0}, false)
			for i, original := range f.originals[0] {
				prop := proto.Clone(original).(*types.SeatElectionProposal)
				switch name {
				case "not_due":
					prop.VotingEndBlock++
				case "single_pass", "single_fail":
					if i == 1 {
						prop.Stage = types.SeatStageFailed
					}
					if name == "single_fail" {
						prop.YesStake = "0"
					}
				case "contested_clear":
					if i == 1 {
						prop.YesStake = "400"
					}
				case "contested_fail":
					prop.YesStake = "0"
				case "resolve_runoff", "runoff_fail":
					prop.IsRunoff = true
					if name == "runoff_fail" {
						prop.YesStake = "0"
					}
				}
				f.keeper.SetSeatElection(f.ctx, prop)
			}
			limit := uint64(10_000_000)
			if name == "out_of_gas" {
				limit = 1
			}
			legacy := observeSeatTally(t, f, f.replayContext().WithGasMeter(storetypes.NewGasMeter(limit)), f.keeper.TallySeatElectionsLegacyReference)
			gated := observeSeatTally(t, f, f.replayContext().WithGasMeter(storetypes.NewGasMeter(limit)), f.keeper.TallySeatElections)
			require.Equal(t, legacy, gated)
			if name == "out_of_gas" {
				require.Equal(t, "types.ErrorOutOfGas: {ReadFlat}", gated.Panic)
			} else {
				require.Empty(t, gated.Panic)
			}
		})
	}
}

// Two independent map traversals cannot be compared trial-for-trial. Compare
// their complete observed outcome sets instead, without sorting the events or
// rewriting any ID. Bounded sampling does not control Go's map randomizer.
func TestSeatRunoffMarkerOffRetainsLegacyOutcomeSet(t *testing.T) {
	f := newSeatRunoffReplayFixture(t, []uint32{1, 0}, false)
	collect := func(tally func(sdk.Context)) map[string]bool {
		outcomes := map[string]bool{}
		for trial := 0; trial < 512; trial++ {
			outcome := observeSeatTally(t, f, f.replayContext(), tally)
			require.Empty(t, outcome.Panic)
			encoded, err := json.Marshal(outcome)
			require.NoError(t, err)
			outcomes[string(encoded)] = true
		}
		return outcomes
	}
	legacy := collect(f.keeper.TallySeatElectionsLegacyReference)
	gated := collect(f.keeper.TallySeatElections)
	require.Len(t, legacy, 2, "bounded legacy sample should include both seat orders")
	require.Equal(t, legacy, gated, "off must preserve old raw bytes, event order and gas for both allocations")
}

func TestSeatRunoffMalformedMarkerFailsClosed(t *testing.T) {
	for _, bad := range [][]byte{{0}, {2}, {1, 1}, {0xff}} {
		t.Run(fmt.Sprintf("%x", bad), func(t *testing.T) {
			f := newSeatRunoffReplayFixture(t, []uint32{1, 0}, false)
			ctx := f.replayContext()
			ctx.KVStore(f.key).Set(types.SeatRunoffOrderingEnabledKey, bad)
			before := seatRunoffStoreBytes(t, ctx, f.key)
			metered := ctx.WithGasMeter(storetypes.NewGasMeter(1))
			require.PanicsWithError(t, "malformed seat-runoff ordering marker", func() { f.keeper.TallySeatElections(metered) })
			require.Zero(t, metered.GasMeter().GasConsumed())
			require.Equal(t, before, seatRunoffStoreBytes(t, ctx, f.key))
			require.Empty(t, ctx.EventManager().Events())
			require.PanicsWithError(t, "malformed seat-runoff ordering marker", func() { _ = f.module.BeginBlock(ctx) })
			require.Equal(t, before, seatRunoffStoreBytes(t, ctx, f.key))
		})
	}
}

func TestSeatRunoffActivationPreservesLegacyState(t *testing.T) {
	f := newSeatRunoffReplayFixture(t, []uint32{1, 0}, false)
	ctx := f.replayContext()
	// Build a legal old-order state using the old tally: seat 1 first, then 0.
	// Staggering deadlines avoids relying on a lucky random map traversal.
	for _, original := range f.originals[0] {
		prop := proto.Clone(original).(*types.SeatElectionProposal)
		prop.VotingEndBlock++
		f.keeper.SetSeatElection(ctx, prop)
	}
	f.keeper.TallySeatElectionsLegacyReference(ctx)
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	f.keeper.TallySeatElectionsLegacyReference(ctx)
	for id, seat := range map[uint64]uint32{5: 1, 6: 1, 7: 0, 8: 0} {
		prop, found := f.keeper.GetSeatElection(ctx, id)
		require.True(t, found)
		require.Equal(t, seat, prop.SeatIndex)
		require.True(t, prop.IsRunoff)
	}
	// Real votes on old-order children must survive activation under their IDs.
	for _, id := range []uint64{5, 7} {
		_, err := f.keeper.VoteSeatElection(ctx, &types.MsgVoteSeatElection{Voter: testAddr("voter0"), ProposalId: id, Option: types.VoteYes})
		require.NoError(t, err)
	}
	legacyBytes := seatRunoffStoreBytes(t, ctx, f.key)
	votes := f.keeper.GetAllSeatElectionVotes(ctx)
	parents := f.keeper.GetSeatElectionsByStage(ctx, types.SeatStageRunoff)
	require.Len(t, parents, 4)
	require.Equal(t, uint64(9), f.keeper.GetNextSeatElectionID(ctx))
	require.NoError(t, f.keeper.EnableSeatRunoffOrdering(ctx))
	// A discarded view without just the marker is byte-identical to the old
	// state: proposal IDs, stages, parent references, votes and counter included.
	withoutMarker, _ := ctx.CacheContext()
	withoutMarker.KVStore(f.key).Delete(types.SeatRunoffOrderingEnabledKey)
	require.Equal(t, legacyBytes, seatRunoffStoreBytes(t, withoutMarker, f.key))
	activatedBytes := seatRunoffStoreBytes(t, ctx, f.key)
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	require.NoError(t, f.module.BeginBlock(ctx))
	require.Equal(t, activatedBytes, seatRunoffStoreBytes(t, ctx, f.key), "parents already in runoff must not be collected again")

	child, found := f.keeper.GetSeatElection(ctx, 7)
	require.True(t, found)
	ctx = ctx.WithBlockHeight(int64(child.VotingEndBlock)).WithEventManager(sdk.NewEventManager())
	require.NoError(t, f.module.BeginBlock(ctx))
	require.Equal(t, uint64(9), f.keeper.GetNextSeatElectionID(ctx), "resolveRunoff allocates no IDs")
	for _, event := range ctx.EventManager().Events() {
		require.NotEqual(t, "zerone.gov.seat_election_runoff_created", event.Type)
	}
	for _, parent := range parents {
		got, found := f.keeper.GetSeatElection(ctx, parent.ProposalId)
		require.True(t, found)
		require.True(t, proto.Equal(parent, got))
	}
	for _, vote := range votes {
		got, found := f.keeper.GetSeatElectionVote(ctx, vote.ProposalId, vote.Voter)
		require.True(t, found)
		require.True(t, proto.Equal(vote, got))
	}
	for _, id := range []uint64{5, 7} {
		got, found := f.keeper.GetSeatElection(ctx, id)
		require.True(t, found)
		require.Equal(t, types.SeatStagePassed, got.Stage)
	}

	// Synthetic fresh due proposals continue from the stored counter, even
	// though nomination order is reversed; old records are not renumbered.
	for _, seat := range []uint32{1, 0} {
		for _, original := range f.originals[seat] {
			prop := proto.Clone(original).(*types.SeatElectionProposal)
			prop.ProposalId = f.keeper.GetNextSeatElectionID(ctx)
			prop.VotingEndBlock = uint64(ctx.BlockHeight())
			f.keeper.SetNextSeatElectionID(ctx, prop.ProposalId+1)
			f.keeper.SetSeatElection(ctx, prop)
		}
	}
	f.keeper.TallySeatElections(ctx)
	for id, seat := range map[uint64]uint32{13: 0, 14: 0, 15: 1, 16: 1} {
		prop, found := f.keeper.GetSeatElection(ctx, id)
		require.True(t, found)
		require.Equal(t, seat, prop.SeatIndex)
		require.True(t, prop.IsRunoff)
	}
	require.Equal(t, uint64(17), f.keeper.GetNextSeatElectionID(ctx))
}

// Keep the pre-existing rollback/seat-growth behavior in both states. This
// characterization does not endorse it as policy; any repair is a separate PR.
func TestSeatRunoffRollbackSeatGrowthPreserved(t *testing.T) {
	for _, ordered := range []bool{false, true} {
		t.Run(fmt.Sprintf("ordered=%t", ordered), func(t *testing.T) {
			f := newSeatRunoffReplayFixture(t, []uint32{2, 0, 1}, ordered)
			ctx := f.replayContext()
			f.keeper.ExecuteRollback(ctx)
			require.Len(t, f.keeper.GetResearchFundGovernanceState(ctx).CommunitySeats, 1)
			require.NoError(t, f.module.BeginBlock(ctx))
			var end uint64
			for _, prop := range f.keeper.GetSeatElectionsByStage(ctx, types.SeatStageVoting) {
				end = prop.VotingEndBlock
				if prop.Candidate == f.originals[prop.SeatIndex][0].Candidate {
					_, err := f.keeper.VoteSeatElection(ctx, &types.MsgVoteSeatElection{Voter: testAddr("voter0"), ProposalId: prop.ProposalId, Option: types.VoteYes})
					require.NoError(t, err)
				}
			}
			require.NotZero(t, end)
			ctx = ctx.WithBlockHeight(int64(end))
			require.NoError(t, f.module.BeginBlock(ctx))
			state := f.keeper.GetResearchFundGovernanceState(ctx)
			require.Equal(t, types.ResearchFundPhase_RESEARCH_FUND_PHASE_OBSERVER, state.CurrentPhase)
			require.Len(t, state.CommunitySeats, 3)
			for seat, address := range state.CommunitySeats {
				require.Equal(t, f.originals[seat][0].Candidate, address)
			}
			require.Equal(t, uint64(13), f.keeper.GetNextSeatElectionID(ctx))
		})
	}
}
