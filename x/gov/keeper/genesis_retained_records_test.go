package keeper_test

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	govmodule "github.com/zerone-chain/zerone/x/gov"
	"github.com/zerone-chain/zerone/x/gov/types"
)

func retainedGenesisFixture() *types.GenesisState {
	gs := types.DefaultGenesisState()
	gs.AccountingSafetyEnabled = false
	gs.SeatRunoffOrderingEnabled = false
	gs.Params.ResearchFundVoters = &types.ResearchFundVoters{Voter1: testAddr("core1"), Voter2: testAddr("core2")}
	gs.ResearchFundGovernance.CommunitySeats = []string{testAddr("seat")}
	gs.ResearchFundGovernance.SeatTermEndBlocks = []uint64{9000}
	gs.ResearchFundGovernance.PhaseStartedAtBlock = 10
	gs.ResearchFundGovernance.CurrentPhase = types.ResearchFundPhase_RESEARCH_FUND_PHASE_OBSERVER
	gs.ResearchFundGovernance.ProposalsExecutedInPhase = 7
	next := uint64(23)
	gs.NextResearchSpendNumber = &next
	// Deliberately not sorted. All retained stages, including an execution
	// failure still in voting, must survive without re-evaluation.
	for i, stage := range []types.ResearchSpendStage{types.ResearchStageExpired, types.ResearchStageRejected,
		types.ResearchStageExecuted, types.ResearchStageVoting, types.ResearchStageDiscussion} {
		gs.ResearchSpendProposals = append(gs.ResearchSpendProposals, &types.ResearchSpendProposal{
			ProposalId: uint64(10 - i), Proposer: testAddr("core1"), Recipient: testAddr("recipient"),
			Title: "retained", Description: "historical", Amount: "1234567", Justification: "research",
			Stage: string(stage), CreatedAt: 20, VotingStartsAt: 30, VotingEndsAt: 5000,
			Voter1Vote: "yes", Voter1Reason: "first reason", Voter1VotedAt: 31,
			Voter2Vote: "no", Voter2Reason: "second reason", Voter2VotedAt: 32,
			ExecutedAt: 40, ExecutionErr: "historical execution error",
		})
	}
	gs.ResearchCommunityVotes = []*types.GenesisResearchCommunityVote{
		{ProposalId: 8, Voter: testAddr("former-seat"), Vote: "no"},
		{ProposalId: 7, Voter: testAddr("seat"), Vote: "yes"},
	}
	gs.DistinctVoters = []*types.GenesisDistinctVoter{
		{Voter: testAddr("second"), FirstVotedAtBlock: 77},
		{Voter: testAddr("first"), FirstVotedAtBlock: 0},
	}
	for i, stage := range []string{types.PhaseTransitionStagePending, types.PhaseTransitionStageActivated,
		types.PhaseTransitionStageCancelled, types.PhaseTransitionStageFailed} {
		id := fmt.Sprintf("LIP-%d", i+1)
		rollback := i%2 != 0
		category := types.CategoryPhaseTransition
		if rollback {
			category = types.CategoryPhaseRollback
		}
		gs.Lips = append(gs.Lips, &types.LIP{Id: id, Category: category, Stage: types.StatusPassed})
		gs.PhaseTransitions = append(gs.PhaseTransitions, &types.GenesisPhaseTransition{
			LipId: id, TargetPhase: types.ResearchFundPhase_RESEARCH_FUND_PHASE_BALANCED,
			ConditionsSnapshot: &types.PhaseTransitionConditions{
				DistinctLipVoters: 4, ActiveGuardians: 5, ResearchFundBalance: "6789", ChainAgeBlocks: 100,
				ProposalsExecutedInPhase: 6, CommunitySeatParticipation: 7, EmergencyHaltsFromMisuse: 8,
			},
			Stage: stage, ActivationBlock: 8000, IsRollback: rollback, CancelReason: "retained reason",
		})
	}
	gs.PhaseTransitions[1].ConditionsSnapshot = nil // rollback producer omits it
	gs.NextLipNumber = 5
	return gs
}

func retainedJSON(t *testing.T, gs *types.GenesisState) json.RawMessage {
	t.Helper()
	bz, err := json.Marshal(gs)
	require.NoError(t, err)
	return bz
}

func seedRetainedLiveState(t *testing.T, f *seatRunoffOrderingFixture, gs *types.GenesisState) {
	t.Helper()
	legacy := proto.Clone(gs).(*types.GenesisState)
	legacy.ResearchSpendProposals, legacy.NextResearchSpendNumber = nil, nil
	legacy.ResearchCommunityVotes, legacy.DistinctVoters, legacy.PhaseTransitions = nil, nil, nil
	f.initGenesis(t, retainedJSON(t, legacy))
	for _, p := range gs.ResearchSpendProposals {
		f.k.SetResearchSpendProposal(f.ctx, p)
	}
	f.k.SetNextResearchSpendID(f.ctx, *gs.NextResearchSpendNumber)
	for _, v := range gs.ResearchCommunityVotes {
		f.k.SetResearchCommunityVote(f.ctx, v.ProposalId, v.Voter, v.Vote)
	}
	for _, v := range gs.DistinctVoters {
		f.k.RecordDistinctVoter(f.ctx.WithBlockHeight(int64(v.FirstVotedAtBlock)), v.Voter)
	}
	for _, p := range gs.PhaseTransitions {
		f.k.SetPhaseTransitionMeta(f.ctx, &types.PhaseTransitionProposal{
			LipID: p.LipId, TargetPhase: p.TargetPhase, ConditionsSnapshot: p.ConditionsSnapshot,
			Stage: p.Stage, ActivationBlock: p.ActivationBlock, IsRollback: p.IsRollback, CancelReason: p.CancelReason,
		})
	}
}

func TestRetainedGenesisModuleRoundTrip(t *testing.T) {
	for _, marker := range []bool{false, true} {
		t.Run(fmt.Sprintf("existing-seat-marker-%t", marker), func(t *testing.T) {
			source := newSeatRunoffOrderingFixture(t)
			gs := retainedGenesisFixture()
			gs.SeatRunoffOrderingEnabled = marker
			seedRetainedLiveState(t, source, gs)
			before := source.snapshot(t)
			raw := source.am.ExportGenesis(source.ctx, source.cdc)
			require.Equal(t, raw, source.am.ExportGenesis(source.ctx, source.cdc), "deterministic export")
			require.Equal(t, before, source.snapshot(t), "export is read-only")
			var exported types.GenesisState
			require.NoError(t, json.Unmarshal(raw, &exported))
			require.Len(t, exported.ResearchSpendProposals, 5)
			require.Len(t, exported.ResearchCommunityVotes, 2)
			require.Len(t, exported.DistinctVoters, 2)
			require.Len(t, exported.PhaseTransitions, 4)
			for i, p := range exported.ResearchSpendProposals {
				require.Equal(t, uint64(6+i), p.ProposalId, "byte-key ordering")
			}

			target := newSeatRunoffOrderingFixture(t)
			target.ctx = target.ctx.WithBlockHeight(7777).WithEventManager(sdk.NewEventManager())
			vesting := &mockVestingKeeper{}
			target.k.SetVestingKeeper(vesting)
			target.am = govmodule.NewAppModule(nil, target.k)
			target.initGenesis(t, raw)
			require.Equal(t, before, target.snapshot(t), "every stored byte survives a different import height")
			require.Equal(t, raw, target.am.ExportGenesis(target.ctx, target.cdc))
			require.Empty(t, target.ctx.EventManager().Events(), "no lifecycle replay")
			require.False(t, vesting.disburseCalled)
			require.Equal(t, uint64(77), binary.BigEndian.Uint64(target.ctx.KVStore(target.key).Get(types.DistinctVoterKey(testAddr("second")))))
			require.Equal(t, uint64(0), binary.BigEndian.Uint64(target.ctx.KVStore(target.key).Get(types.DistinctVoterKey(testAddr("first")))))
			require.Equal(t, uint64(2), target.k.CountDistinctVoters(target.ctx))
			require.Equal(t, uint64(1), target.k.CountCommunitySeatVotes(target.ctx, target.k.GetResearchFundGovernanceState(target.ctx)))
			require.Equal(t, "no", target.k.GetResearchCommunityVote(target.ctx, 8, testAddr("former-seat")))
			require.True(t, target.k.HasActivePhaseTransitionLIP(target.ctx))
			require.ErrorIs(t, target.k.ValidatePhaseTransition(target.ctx, types.ResearchFundPhase_RESEARCH_FUND_PHASE_BALANCED), types.ErrPendingTransitionExists)
			response, err := target.k.SubmitResearchSpend(target.ctx, &types.MsgSubmitResearchSpend{
				Proposer: testAddr("core1"), Recipient: testAddr("recipient"), Title: "next", Amount: "1",
			})
			require.NoError(t, err)
			require.Equal(t, uint64(23), response.ProposalId, "counter is retained, not maxID+1")
		})
	}
}

// This trace also runs against the foundation via a Go overlay, to compare
// state, events and gas with the actual original InitGenesis implementation.
func TestRetainedGenesisLegacyImport(t *testing.T) {
	f := newSeatRunoffOrderingFixture(t)
	gs := types.DefaultGenesisState()
	gs.AccountingSafetyEnabled, gs.SeatRunoffOrderingEnabled = false, false
	gs.Params.ResearchFundVoters = &types.ResearchFundVoters{Voter1: testAddr("core1"), Voter2: testAddr("core2")}
	gs.Lips = []*types.LIP{{Id: "LIP-1", Category: types.CategoryText, Stage: types.StatusDraft}}
	gs.NextLipNumber = 2
	raw := retainedJSON(t, gs)
	for _, field := range []string{"research_spend_proposals", "next_research_spend_number", "research_community_votes", "distinct_voters", "phase_transitions"} {
		require.NotContains(t, string(raw), field)
	}
	f.initGenesis(t, raw)
	gas := f.ctx.GasMeter().GasConsumed()
	require.Nil(t, f.ctx.KVStore(f.key).Get(types.ResearchSpendCounterKey))
	require.Equal(t, uint64(1), f.k.GetNextResearchSpendID(f.ctx))
	require.Empty(t, f.ctx.EventManager().Events())
	trace, err := json.Marshal(struct {
		State  map[string]string
		Gas    uint64
		Events sdk.Events
	}{f.snapshot(t), gas, f.ctx.EventManager().Events()})
	require.NoError(t, err)
	t.Logf("LEGACY_TRACE %s", trace)
}

func TestRetainedGenesisCounterPresence(t *testing.T) {
	for _, value := range []*uint64{nil, proto.Uint64(0), proto.Uint64(1), proto.Uint64(99)} {
		f := newSeatRunoffOrderingFixture(t)
		gs := types.DefaultGenesisState()
		gs.NextResearchSpendNumber = value
		f.initGenesis(t, retainedJSON(t, gs))
		exported := f.k.ExportGenesis(f.ctx)
		require.Equal(t, value, exported.NextResearchSpendNumber)
		require.Equal(t, value != nil, f.ctx.KVStore(f.key).Has(types.ResearchSpendCounterKey))
		target := newSeatRunoffOrderingFixture(t)
		target.initGenesis(t, f.am.ExportGenesis(f.ctx, f.cdc))
		require.Equal(t, f.snapshot(t), target.snapshot(t))
	}
}

func TestRetainedGenesisInvalidImportBeforeWrites(t *testing.T) {
	for _, mutate := range []func(*types.GenesisState){
		func(gs *types.GenesisState) { gs.ResearchSpendProposals = append(gs.ResearchSpendProposals, nil) },
		func(gs *types.GenesisState) { gs.NextResearchSpendNumber = nil },
		func(gs *types.GenesisState) { gs.ResearchCommunityVotes[0].ProposalId = 99 },
		func(gs *types.GenesisState) { gs.DistinctVoters = append(gs.DistinctVoters, gs.DistinctVoters[0]) },
		func(gs *types.GenesisState) { gs.PhaseTransitions[0].LipId = "absent" },
	} {
		gs := retainedGenesisFixture()
		mutate(gs)
		f := newSeatRunoffOrderingFixture(t)
		before := f.snapshot(t)
		raw := retainedJSON(t, gs)
		require.Error(t, f.am.ValidateGenesis(f.cdc, nil, raw))
		require.Panics(t, func() { f.am.InitGenesis(f.ctx, f.cdc, raw) })
		require.Equal(t, before, f.snapshot(t), "direct InitGenesis also validates before any write")
		require.Empty(t, f.ctx.EventManager().Events())
	}
}

func TestRetainedGenesisCorruptExportFailsClosed(t *testing.T) {
	cases := []struct {
		name    string
		corrupt func(*seatRunoffOrderingFixture)
	}{
		{"proposal-json", func(f *seatRunoffOrderingFixture) { f.ctx.KVStore(f.key).Set(types.ResearchSpendKey(6), []byte("{")) }},
		{"proposal-null", func(f *seatRunoffOrderingFixture) {
			f.ctx.KVStore(f.key).Set(types.ResearchSpendKey(6), []byte("null"))
		}},
		{"proposal-identity", func(f *seatRunoffOrderingFixture) {
			f.ctx.KVStore(f.key).Set(types.ResearchSpendKey(5), f.ctx.KVStore(f.key).Get(types.ResearchSpendKey(6)))
		}},
		{"proposal-key-width", func(f *seatRunoffOrderingFixture) { f.ctx.KVStore(f.key).Set([]byte{6, 1}, []byte("{}")) }},
		{"unknown-json-field", func(f *seatRunoffOrderingFixture) {
			b := f.ctx.KVStore(f.key).Get(types.ResearchSpendKey(6))
			f.ctx.KVStore(f.key).Set(types.ResearchSpendKey(6), append([]byte(`{"unknown":1,`), b[1:]...))
		}},
		{"trailing-json", func(f *seatRunoffOrderingFixture) {
			b := f.ctx.KVStore(f.key).Get(types.ResearchSpendKey(6))
			f.ctx.KVStore(f.key).Set(types.ResearchSpendKey(6), append(b, []byte(" {}")...))
		}},
		{"counter-width", func(f *seatRunoffOrderingFixture) { f.ctx.KVStore(f.key).Set(types.ResearchSpendCounterKey, []byte{1}) }},
		{"counter-absent", func(f *seatRunoffOrderingFixture) { f.ctx.KVStore(f.key).Delete(types.ResearchSpendCounterKey) }},
		{"counter-stale", func(f *seatRunoffOrderingFixture) { f.k.SetNextResearchSpendID(f.ctx, 10) }},
		{"community-key", func(f *seatRunoffOrderingFixture) { f.ctx.KVStore(f.key).Set([]byte{15, 1}, []byte("yes")) }},
		{"community-separator", func(f *seatRunoffOrderingFixture) {
			key := types.ResearchCommunityVoteKey(8, testAddr("seat"))
			key[9] = 1
			f.ctx.KVStore(f.key).Set(key, []byte("yes"))
		}},
		{"community-vote", func(f *seatRunoffOrderingFixture) { f.k.SetResearchCommunityVote(f.ctx, 8, testAddr("seat"), "maybe") }},
		{"community-orphan", func(f *seatRunoffOrderingFixture) { f.k.SetResearchCommunityVote(f.ctx, 90, testAddr("seat"), "yes") }},
		{"distinct-height", func(f *seatRunoffOrderingFixture) {
			f.ctx.KVStore(f.key).Set(types.DistinctVoterKey(testAddr("first")), []byte{1})
		}},
		{"distinct-key", func(f *seatRunoffOrderingFixture) {
			f.ctx.KVStore(f.key).Set(types.DistinctVoterKey(""), make([]byte, 8))
		}},
		{"transition-json", func(f *seatRunoffOrderingFixture) {
			f.ctx.KVStore(f.key).Set(types.PhaseTransitionKey("LIP-1"), []byte("{"))
		}},
		{"transition-null", func(f *seatRunoffOrderingFixture) {
			f.ctx.KVStore(f.key).Set(types.PhaseTransitionKey("LIP-1"), []byte("null"))
		}},
		{"transition-identity", func(f *seatRunoffOrderingFixture) {
			f.ctx.KVStore(f.key).Set(types.PhaseTransitionKey("other"), f.ctx.KVStore(f.key).Get(types.PhaseTransitionKey("LIP-1")))
		}},
		{"transition-orphan", func(f *seatRunoffOrderingFixture) { f.k.DeleteLIP(f.ctx, "LIP-1") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newSeatRunoffOrderingFixture(t)
			seedRetainedLiveState(t, f, retainedGenesisFixture())
			tc.corrupt(f)
			before := f.snapshot(t)
			require.Panics(t, func() { f.am.ExportGenesis(f.ctx, f.cdc) })
			require.Equal(t, before, f.snapshot(t), "no repair or deletion during export")
		})
	}
}

func TestRetainedGenesisPreservesLiveSkipBehavior(t *testing.T) {
	f := newSeatRunoffOrderingFixture(t)
	f.initGenesis(t, f.am.DefaultGenesis(f.cdc))
	f.ctx.KVStore(f.key).Set(types.ResearchSpendKey(1), []byte("{"))
	f.ctx.KVStore(f.key).Set(types.PhaseTransitionKey("bad"), []byte("{"))
	require.Empty(t, f.k.GetAllResearchSpendProposals(f.ctx))
	require.Empty(t, f.k.GetAllPhaseTransitionMeta(f.ctx))
	require.NotPanics(t, func() { f.k.ProcessResearchSpendExpiry(f.ctx, 100) })
	require.NotPanics(t, func() { f.k.BeginBlockPhaseTransition(f.ctx) })
	require.Panics(t, func() { f.k.ExportGenesis(f.ctx) })
}

func TestRetainedGenesisRejectsNULVoters(t *testing.T) {
	gs := retainedGenesisFixture()
	gs.ResearchCommunityVotes[0].Voter += "\x00suffix"
	require.Error(t, gs.Validate())
	gs = retainedGenesisFixture()
	gs.DistinctVoters[0].Voter = string(bytes.Repeat([]byte{0}, 20))
	require.Error(t, gs.Validate())
}

type retainedFaultStore struct {
	storetypes.MultiStore
	prefix     []byte
	fault      error
	closeFault bool
	steps      int
	closed     bool
}

func (s *retainedFaultStore) GetKVStore(key storetypes.StoreKey) storetypes.KVStore {
	return retainedFaultKV{KVStore: s.MultiStore.GetKVStore(key), owner: s}
}

type retainedFaultKV struct {
	storetypes.KVStore
	owner *retainedFaultStore
}

func (s retainedFaultKV) Iterator(start, end []byte) storetypes.Iterator {
	iter := s.KVStore.Iterator(start, end)
	if bytes.Equal(start, s.owner.prefix) {
		return &retainedFaultIterator{Iterator: iter, owner: s.owner}
	}
	return iter
}

type retainedFaultIterator struct {
	storetypes.Iterator
	owner *retainedFaultStore
}

func (i *retainedFaultIterator) Next() {
	i.owner.steps++
	i.Iterator.Next()
}

func (i *retainedFaultIterator) Valid() bool {
	return (i.owner.closeFault || i.owner.steps == 0) && i.Iterator.Valid()
}

func (i *retainedFaultIterator) Error() error {
	if !i.owner.closeFault && i.owner.steps > 0 {
		return i.owner.fault
	}
	return i.Iterator.Error()
}

func (i *retainedFaultIterator) Close() error {
	i.owner.closed = true
	err := i.Iterator.Close()
	if i.owner.closeFault {
		return errors.Join(err, i.owner.fault)
	}
	return err
}

func TestRetainedGenesisIteratorFaults(t *testing.T) {
	for _, prefix := range [][]byte{types.ResearchSpendKeyPrefix, types.ResearchCommunityVotePrefix, types.DistinctVoterKeyPrefix, types.PhaseTransitionKeyPrefix} {
		for _, closeFault := range []bool{false, true} {
			t.Run(fmt.Sprintf("prefix-%x-close-%t", prefix, closeFault), func(t *testing.T) {
				f := newSeatRunoffOrderingFixture(t)
				seedRetainedLiveState(t, f, retainedGenesisFixture())
				fault := errors.New("injected retained iterator failure")
				ms := &retainedFaultStore{MultiStore: f.ctx.MultiStore(), prefix: prefix, fault: fault, closeFault: closeFault}
				ctx := f.ctx.WithMultiStore(ms)
				func() {
					defer func() {
						err, ok := recover().(error)
						require.True(t, ok, "export must panic with an error")
						require.ErrorIs(t, err, fault)
					}()
					f.am.ExportGenesis(ctx, f.cdc)
				}()
				require.Positive(t, ms.steps, "fault follows at least one valid row")
				require.True(t, ms.closed)
			})
		}
	}
}

func TestRetainedGenesisCacheIteratorEOF(t *testing.T) {
	f := newSeatRunoffOrderingFixture(t)
	seedRetainedLiveState(t, f, retainedGenesisFixture())
	want := f.am.ExportGenesis(f.ctx, f.cdc)
	cacheCtx, _ := f.ctx.CacheContext()
	// Force both the parent-store and cache iterator arms to participate.
	f.k.RecordDistinctVoter(cacheCtx, testAddr("cache-voter"))
	var got types.GenesisState
	require.NoError(t, json.Unmarshal(f.am.ExportGenesis(cacheCtx, f.cdc), &got))
	require.Len(t, got.DistinctVoters, 3)
	require.Equal(t, want, f.am.ExportGenesis(f.ctx, f.cdc), "uncommitted cache does not mutate source")
}

func TestRetainedGenesisDefersTransitionToBlockHook(t *testing.T) {
	source := newSeatRunoffOrderingFixture(t)
	gs := retainedGenesisFixture()
	// A due transition is not executed or re-approved during import. The
	// existing hook must still recheck and cancel it when conditions are unmet.
	gs.PhaseTransitions[0].ActivationBlock = 100
	seedRetainedLiveState(t, source, gs)
	target := newSeatRunoffOrderingFixture(t)
	target.ctx = target.ctx.WithBlockHeight(500)
	target.initGenesis(t, source.am.ExportGenesis(source.ctx, source.cdc))
	pending, found := target.k.GetPhaseTransitionMeta(target.ctx, "LIP-1")
	require.True(t, found)
	require.Equal(t, types.PhaseTransitionStagePending, pending.Stage)
	require.Empty(t, target.ctx.EventManager().Events())
	target.k.BeginBlockPhaseTransition(target.ctx)
	cancelled, found := target.k.GetPhaseTransitionMeta(target.ctx, "LIP-1")
	require.True(t, found)
	require.Equal(t, types.PhaseTransitionStageCancelled, cancelled.Stage)
	require.Equal(t, "exit conditions no longer met", cancelled.CancelReason)
}
