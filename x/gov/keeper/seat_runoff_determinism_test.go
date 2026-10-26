package keeper_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	gov "github.com/zerone-chain/zerone/x/gov"
	"github.com/zerone-chain/zerone/x/gov/keeper"
	"github.com/zerone-chain/zerone/x/gov/types"
)

// Pin ascending seat-index allocation of the shared runoff ID counter.
// The 2026-10-12 block-processing review identified the unsorted bySeat map;
// PR #115 reproduced two raw-state outcomes from identical governance bytes.
// Two seats enter runoffs in the same BeginBlock (EndBlock is a no-op).
//
// Each trial executes the real module wrapper against a fresh, discarded cache
// of exactly the same governance bytes. Repetition avoids relying on one lucky
// map traversal; it is bounded sampling, not control over Go's map randomizer.
// The fixture uses real nomination/acceptance/voting handlers with synthetic
// staking observations/history. It does not run signed transactions, the app
// manager, FinalizeBlock/Commit, a node, or a live-chain replay.
func TestSeatRunoffReplayDeterminism(t *testing.T) {
	f := newSeatRunoffReplayFixture(t, []uint32{0, 1}, true)
	before := seatRunoffStoreBytes(t, f.ctx, f.key)
	const trials = 512
	type outcome struct {
		ids    [][2]uint64
		nextID uint64
		state  []byte
		count  int
	}
	var outcomes []outcome

	for trial := 0; trial < trials; trial++ {
		ctx := f.replayContext()
		require.True(t, bytes.Equal(before, seatRunoffStoreBytes(t, ctx, f.key)), "trial %d prestate drift", trial)
		require.NoError(t, f.module.BeginBlock(ctx))
		ids := f.checkRunoffs(t, ctx, [][2]uint64{{5, 6}, {7, 8}})
		nextID := f.keeper.GetNextSeatElectionID(ctx)
		after := seatRunoffStoreBytes(t, ctx, f.key)
		if trial == 0 {
			// The marker is the only added byte-level state relative to the
			// PR #115 ascending-order golden; never rewrite election history.
			withoutMarker, _ := ctx.CacheContext()
			withoutMarker.KVStore(f.key).Delete(types.SeatRunoffOrderingEnabledKey)
			require.Equal(t, "efd5abbee21b2e449748dc4dcf8ada3e62f1476065469c22b586a13f13821e6e", fmt.Sprintf("%x", sha256.Sum256(seatRunoffStoreBytes(t, withoutMarker, f.key))))
		}
		matched := false
		for i := range outcomes {
			if bytes.Equal(after, outcomes[i].state) {
				require.Equal(t, outcomes[i].ids, ids)
				require.Equal(t, outcomes[i].nextID, nextID)
				outcomes[i].count++
				matched = true
				break
			}
		}
		if !matched {
			outcomes = append(outcomes, outcome{ids: ids, nextID: nextID, state: after, count: 1})
		}
		// Never write the trial cache: the next replay must not inherit IDs,
		// stage transitions, events, or gas accounting from this execution.
		require.True(t, bytes.Equal(before, seatRunoffStoreBytes(t, f.ctx, f.key)), "trial %d changed parent", trial)
	}

	t.Logf("identical-prestate trials=%d prestate_sha256=%x raw_state_outcomes=%d", trials, sha256.Sum256(before), len(outcomes))
	for _, observed := range outcomes {
		t.Logf("seat0=%v seat1=%v next_id=%d trials=%d raw_state_sha256=%x", observed.ids[0], observed.ids[1], observed.nextID, observed.count, sha256.Sum256(observed.state))
		// Ascending-seat golden including the foundation's enabled marker.
		require.Equal(t, "df85a7c9a7d219a3a45a4effffcdd0e1a92be78eec0ca618148d3c35b1eb1f83", fmt.Sprintf("%x", sha256.Sum256(observed.state)))
	}
	// Compare raw key/value bytes, not just event ordering or a Go map's
	// presentation. The digest above is NOT an IAVL/AppHash.
	if len(outcomes) != 1 {
		t.Fatalf("identical state produced %d runoff ID/raw-state outcomes; seat traversal is not consensus-deterministic", len(outcomes))
	}
}

func TestSeatRunoffReplayControls(t *testing.T) {
	f := newSeatRunoffReplayFixture(t, []uint32{0, 1}, true)
	before := seatRunoffStoreBytes(t, f.ctx, f.key)

	t.Run("EndBlock_is_not_the_tally_hook", func(t *testing.T) {
		ctx := f.replayContext()
		require.NoError(t, f.module.EndBlock(ctx))
		require.Equal(t, before, seatRunoffStoreBytes(t, ctx, f.key))
		require.Equal(t, uint64(5), f.keeper.GetNextSeatElectionID(ctx))
		require.Empty(t, ctx.EventManager().Events())
	})

	t.Run("before_voting_deadline_allocates_nothing", func(t *testing.T) {
		ctx := f.replayContext().WithBlockHeight(f.ctx.BlockHeight() - 1)
		require.NoError(t, f.module.BeginBlock(ctx))
		require.Equal(t, before, seatRunoffStoreBytes(t, ctx, f.key))
		require.Equal(t, uint64(5), f.keeper.GetNextSeatElectionID(ctx))
	})

	t.Run("same_cache_second_call_does_not_allocate_again", func(t *testing.T) {
		ctx := f.replayContext()
		require.NoError(t, f.module.BeginBlock(ctx))
		ids := f.checkRunoffs(t, ctx, [][2]uint64{{5, 6}, {7, 8}})
		after := seatRunoffStoreBytes(t, ctx, f.key)
		ctx = ctx.WithEventManager(sdk.NewEventManager())
		require.NoError(t, f.module.BeginBlock(ctx))
		require.Equal(t, after, seatRunoffStoreBytes(t, ctx, f.key))
		require.Equal(t, uint64(9), f.keeper.GetNextSeatElectionID(ctx))
		for _, event := range ctx.EventManager().Events() {
			require.NotEqual(t, "zerone.gov.seat_election_runoff_created", event.Type)
		}
		t.Logf("unchanged second call: seat0=%v seat1=%v", ids[0], ids[1])
	})

	require.Equal(t, before, seatRunoffStoreBytes(t, f.ctx, f.key))
}

// Nomination/proposal order must not accidentally define the canonical seat
// order. The ascending-nomination replay alone cannot distinguish these rules.
func TestSeatRunoffReplayCanonicalSeatOrder(t *testing.T) {
	for _, tc := range []struct {
		name            string
		nominationOrder []uint32
		wantParents     [][2]uint64
		wantRunoffs     [][2]uint64
	}{
		{
			name:            "reverse_nomination_order",
			nominationOrder: []uint32{1, 0},
			wantParents:     [][2]uint64{{3, 4}, {1, 2}},
			wantRunoffs:     [][2]uint64{{5, 6}, {7, 8}},
		},
		{
			name:            "three_balanced_seats",
			nominationOrder: []uint32{2, 0, 1},
			wantParents:     [][2]uint64{{3, 4}, {5, 6}, {1, 2}},
			wantRunoffs:     [][2]uint64{{7, 8}, {9, 10}, {11, 12}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newSeatRunoffReplayFixture(t, tc.nominationOrder, true)
			require.Equal(t, types.ResearchFundPhase_RESEARCH_FUND_PHASE_BALANCED, f.keeper.GetResearchFundPhase(f.ctx))
			for seat, parents := range tc.wantParents {
				require.Equal(t, parents, [2]uint64{f.originals[seat][0].ProposalId, f.originals[seat][1].ProposalId}, "fixture nomination order for seat %d", seat)
			}
			require.Equal(t, tc.wantRunoffs[0][0], f.keeper.GetNextSeatElectionID(f.ctx))
			before := seatRunoffStoreBytes(t, f.ctx, f.key)
			ctx := f.replayContext()
			require.NoError(t, f.module.BeginBlock(ctx))
			ids := f.checkRunoffs(t, ctx, tc.wantRunoffs)
			require.Equal(t, before, seatRunoffStoreBytes(t, f.ctx, f.key))
			t.Logf("nomination_order=%v runoff_ids_by_seat=%v next_id=%d", tc.nominationOrder, ids, f.keeper.GetNextSeatElectionID(ctx))
		})
	}
}

func TestSeatRunoffReplayObserverTalliesAllDueSeats(t *testing.T) {
	f := newSeatRunoffReplayFixture(t, []uint32{2, 0, 1}, true)
	ctx := f.replayContext()
	// Exercise the real rollback, not just a phase-field edit. Continuing due
	// elections after seat truncation is existing behavior, not a new policy.
	f.keeper.ExecuteRollback(ctx)
	state := f.keeper.GetResearchFundGovernanceState(ctx)
	require.Len(t, state.CommunitySeats, 1)
	require.Len(t, state.SeatTermEndBlocks, 1)
	require.Equal(t, uint64(ctx.BlockHeight())+types.RollbackCooldownBlocks, state.RollbackCooldownUntil)
	require.Equal(t, types.ResearchFundPhase_RESEARCH_FUND_PHASE_OBSERVER, f.keeper.GetResearchFundPhase(ctx))
	require.NoError(t, f.module.BeginBlock(ctx))
	ids := f.checkRunoffs(t, ctx, [][2]uint64{{7, 8}, {9, 10}, {11, 12}})
	require.Equal(t, types.ResearchFundPhase_RESEARCH_FUND_PHASE_OBSERVER, f.keeper.GetResearchFundPhase(ctx))
	t.Logf("OBSERVER tally: ids=%v next_id=%d", ids, f.keeper.GetNextSeatElectionID(ctx))
}

func TestSeatRunoffReplaySparseDueSeats(t *testing.T) {
	f := newSeatRunoffReplayFixture(t, []uint32{2, 0, 1}, true)
	ctx := f.replayContext()
	for i, original := range f.originals[1] {
		prop := proto.Clone(original).(*types.SeatElectionProposal)
		prop.VotingEndBlock++ // seat 1 exists but is not yet due
		f.keeper.SetSeatElection(ctx, prop)
		f.originals[1][i] = prop
	}
	require.NoError(t, f.module.BeginBlock(ctx))
	f.checkRunoffs(t, ctx, [][2]uint64{{7, 8}, {}, {9, 10}})
}

type seatRunoffReplayFixture struct {
	keeper    keeper.Keeper
	module    gov.AppModule
	ctx       sdk.Context
	key       *storetypes.KVStoreKey
	originals [][2]*types.SeatElectionProposal
}

// nominationOrder is a permutation of the contiguous seat indices to exercise.
func newSeatRunoffReplayFixture(t *testing.T, nominationOrder []uint32, ordered bool) seatRunoffReplayFixture {
	t.Helper()
	require.NotEmpty(t, nominationOrder)
	require.LessOrEqual(t, len(nominationOrder), 3)
	db := dbm.NewMemDB()
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	// Unlike the older fixture helpers, do not leave asynchronous IAVL pruning
	// workers alive when repeatedly constructing the fixture.
	ms.SetIAVLSyncPruning(true)
	key := storetypes.NewKVStoreKey(types.StoreKey)
	ms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	staking := newMockStakingKeeperWithGuardian("1000")
	k := keeper.NewKeeper(cdc, key, "authority", nil, staking)
	ctx := sdk.NewContext(ms, cmtproto.Header{
		ChainID: "seat-runoff-replay-test", Height: 100, Time: time.Unix(1700000000, 0).UTC(),
	}, false, log.NewNopLogger())
	k.SetParams(ctx, types.DefaultParams())
	if ordered {
		require.NoError(t, k.EnableSeatRunoffOrdering(ctx))
	}
	state := k.GetResearchFundGovernanceState(ctx)
	// BALANCED permits three seats; OBSERVER would reject new nominations for
	// seats 1 and 2. Tallying already-admitted elections is a separate contract.
	state.CurrentPhase = types.ResearchFundPhase_RESEARCH_FUND_PHASE_BALANCED
	state.CommunitySeats = []string{"", "", ""}
	state.SeatTermEndBlocks = []uint64{0, 0, 0}
	k.SetResearchFundGovernanceState(ctx, state)
	f := seatRunoffReplayFixture{
		keeper: k, module: gov.NewAppModule(cdc, k), key: key,
		originals: make([][2]*types.SeatElectionProposal, len(nominationOrder)),
	}

	for _, seat := range nominationOrder {
		require.Less(t, seat, uint32(len(f.originals)))
		require.Nil(t, f.originals[seat][0], "duplicate seat in nomination order")
		for candidate := range f.originals[seat] {
			addr := testAddr(fmt.Sprintf("s%d-c%d", seat, candidate))
			staking.guardians[addr] = true
			seedCandidateVotes(t, k, ctx, addr, int(types.MinCandidateGovernanceVotes))
			nomination, err := k.NominateSeatElection(ctx, &types.MsgNominateSeatElection{
				Proposer: testAddr("nominator"), Candidate: addr, SeatIndex: uint32(seat),
				Statement: fmt.Sprintf("Synthetic seat %d candidate %d", seat, candidate),
			})
			require.NoError(t, err)
			_, err = k.AcceptSeatNomination(ctx, &types.MsgAcceptSeatNomination{Candidate: addr, ProposalId: nomination.ProposalId})
			require.NoError(t, err)
			prop, found := k.GetSeatElection(ctx, nomination.ProposalId)
			require.True(t, found)
			f.originals[seat][candidate] = prop
		}
	}

	ctx = ctx.WithBlockHeight(int64(f.originals[0][0].DiscussionEndBlock))
	require.NoError(t, f.module.BeginBlock(ctx)) // real discussion -> voting transition
	for seat := range f.originals {
		for candidate, prop := range f.originals[seat] {
			voter := testAddr(fmt.Sprintf("voter%d", candidate))
			staking.delegations[voter] = []string{"500", "490"}[candidate]
			_, err := k.VoteSeatElection(ctx, &types.MsgVoteSeatElection{Voter: voter, ProposalId: prop.ProposalId, Option: types.VoteYes})
			require.NoError(t, err)
			voted, found := k.GetSeatElection(ctx, prop.ProposalId)
			require.True(t, found)
			require.Equal(t, types.SeatStageVoting, voted.Stage)
			require.Equal(t, []string{"500", "490"}[candidate], voted.YesStake)
			f.originals[seat][candidate] = voted
		}
	}
	// Both candidates meet default quorum/support; 500 versus 490 is within
	// the runoff margin. All proposals expire at precisely this height.
	f.ctx = ctx.WithBlockHeight(int64(f.originals[0][0].VotingEndBlock)).WithEventManager(sdk.NewEventManager())
	for _, seat := range f.originals {
		for _, prop := range seat {
			require.Equal(t, uint64(f.ctx.BlockHeight()), prop.VotingEndBlock)
		}
	}
	require.Equal(t, uint64(2*len(f.originals)+1), k.GetNextSeatElectionID(f.ctx))
	return f
}

func (f seatRunoffReplayFixture) replayContext() sdk.Context {
	ctx, _ := f.ctx.CacheContext()
	return ctx.WithGasMeter(storetypes.NewInfiniteGasMeter()).WithBlockGasMeter(storetypes.NewInfiniteGasMeter())
}

// checkRunoffs pins the exact transition independently of the determinism
// assertion: every original changes stage, two children per seat are written,
// parent/candidate identities survive, and the shared counter advances exactly.
func (f seatRunoffReplayFixture) checkRunoffs(t *testing.T, ctx sdk.Context, wantIDs [][2]uint64) [][2]uint64 {
	t.Helper()
	require.Len(t, wantIDs, len(f.originals))
	firstID := uint64(2*len(f.originals) + 1)
	dueSeats := 0
	for _, ids := range wantIDs {
		if ids != [2]uint64{} {
			dueSeats++
		}
	}
	nextID := firstID + uint64(2*dueSeats)
	require.Len(t, f.keeper.GetAllSeatElections(ctx), 2*(len(f.originals)+dueSeats))
	require.Equal(t, nextID, f.keeper.GetNextSeatElectionID(ctx))
	for seatIndex, seat := range f.originals {
		for _, original := range seat {
			got, found := f.keeper.GetSeatElection(ctx, original.ProposalId)
			require.True(t, found)
			expected := proto.Clone(original).(*types.SeatElectionProposal)
			if wantIDs[seatIndex] != [2]uint64{} {
				expected.Stage = types.SeatStageRunoff
			}
			require.True(t, proto.Equal(expected, got), "original proposal %d changed beyond its stage", original.ProposalId)
		}
	}
	ids := make([][2]uint64, len(f.originals))
	var seatOrder []uint32
	for id := firstID; id < nextID; id++ {
		got, found := f.keeper.GetSeatElection(ctx, id)
		require.True(t, found)
		require.Less(t, got.SeatIndex, uint32(len(f.originals)))
		candidate := int((id - firstID) % 2)
		seat := got.SeatIndex
		original := f.originals[seat][candidate]
		expected := &types.SeatElectionProposal{
			ProposalId: id, Proposer: original.Proposer, Candidate: original.Candidate,
			SeatIndex: seat, Statement: original.Statement, Stage: types.SeatStageVoting,
			YesStake: "0", NoStake: "0", AbstainStake: "0",
			VotingEndBlock: uint64(ctx.BlockHeight()) + types.SeatVotingBlocks,
			CreatedAtBlock: uint64(ctx.BlockHeight()), CandidateAccepted: true, IsRunoff: true,
			RunoffParentIds: []uint64{f.originals[seat][0].ProposalId, f.originals[seat][1].ProposalId},
		}
		require.True(t, proto.Equal(expected, got), "unexpected runoff proposal %d: %v", id, got)
		require.Zero(t, ids[seat][candidate], "duplicate child for seat/candidate")
		ids[seat][candidate] = id
		if candidate == 0 {
			seatOrder = append(seatOrder, seat)
		}
	}
	require.Equal(t, wantIDs, ids, "runoff IDs must follow ascending seat index, not nomination/proposal order")
	var runoffEvents sdk.Events
	for _, event := range ctx.EventManager().Events() {
		if event.Type == "zerone.gov.seat_election_runoff_created" {
			runoffEvents = append(runoffEvents, event)
		}
	}
	require.Len(t, runoffEvents, dueSeats)
	for i, event := range runoffEvents {
		seat := seatOrder[i]
		require.Equal(t, sdk.NewEvent("zerone.gov.seat_election_runoff_created",
			sdk.NewAttribute("seat_index", fmt.Sprint(seat)),
			sdk.NewAttribute("runoff_proposal_1", fmt.Sprint(ids[seat][0])),
			sdk.NewAttribute("runoff_proposal_2", fmt.Sprint(ids[seat][1])),
			sdk.NewAttribute("candidate_1", f.originals[seat][0].Candidate),
			sdk.NewAttribute("candidate_2", f.originals[seat][1].Candidate),
		), event)
	}
	return ids
}

// Length-framed raw key/value bytes in the store iterator's canonical key order.
// No JSON re-encoding, map formatting, exporter defaults, or event bytes enter it.
func seatRunoffStoreBytes(t *testing.T, ctx sdk.Context, key *storetypes.KVStoreKey) []byte {
	t.Helper()
	iter := ctx.KVStore(key).Iterator(nil, nil)
	defer func() { require.NoError(t, iter.Close()) }()
	var snapshot []byte
	var previous []byte
	for ; iter.Valid(); iter.Next() {
		require.True(t, previous == nil || bytes.Compare(previous, iter.Key()) < 0, "non-canonical store iterator")
		previous = bytes.Clone(iter.Key())
		for _, field := range [][]byte{iter.Key(), iter.Value()} {
			snapshot = binary.BigEndian.AppendUint64(snapshot, uint64(len(field)))
			snapshot = append(snapshot, field...)
		}
	}
	return snapshot
}
