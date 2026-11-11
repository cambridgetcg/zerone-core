package keeper_test

// Regression tests for DET-1 (consensus review 2026-11-05): TallySeatElections
// grouped due elections in a Go map keyed by seat index and ranged over it.
// When two or more seats needed runoffs in one block, the map order decided
// which seat's runoff proposals took which IDs from the shared counter, so
// identical prestate committed different store hashes from run to run. These
// tests began as the review's scratch reproduction, which failed with two
// distinct store hashes on every commit it was run against.

import (
	"encoding/hex"
	"fmt"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/zerone-chain/zerone/x/gov/keeper"
	"github.com/zerone-chain/zerone/x/gov/types"
)

// seatRunoffRuns is how many times each scenario is replayed from identical
// prestate. With two seats the old map traversal picked either order about
// half the time, so 64 runs miss a regression with probability about 2^-63.
const seatRunoffRuns = 64

// seatRunoffScenario describes contested seats whose votes all end at
// votingEnd. Seats are written in the order given, two candidates each, so a
// scenario can put a higher seat index at the lower ProposalIds.
type seatRunoffScenario struct {
	seats       []uint32
	yesStakes   [2]string
	votingEnd   uint64
	tallyHeight int64
	// viaBeginBlocker drives the real governance BeginBlocker instead of
	// calling TallySeatElections directly.
	viaBeginBlocker bool
}

// runSeatRunoffScenario builds a fresh committed IAVL store, runs one tally,
// commits, and returns the store hash with the seat index of each runoff
// proposal created, keyed by its ID.
func runSeatRunoffScenario(t *testing.T, sc seatRunoffScenario) (string, map[uint64]uint32) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	cms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	cms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := cms.LoadLatestVersion(); err != nil {
		t.Fatal(err)
	}
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	mock := newMockStakingKeeperWithGuardian("1000000000000")
	k := keeper.NewKeeper(cdc, storeKey, "authority", nil, mock)

	ctx := sdk.NewContext(cms, cmtproto.Header{Height: 10}, false, log.NewNopLogger())
	k.SetParams(ctx, types.DefaultParams())

	st := k.GetResearchFundGovernanceState(ctx)
	st.CurrentPhase = types.ResearchFundPhase_RESEARCH_FUND_PHASE_BALANCED
	st.CommunitySeats = []string{"", "", ""}
	st.SeatTermEndBlocks = []uint64{1 << 40, 1 << 40, 1 << 40}
	k.SetResearchFundGovernanceState(ctx, st)

	id := uint64(1)
	for _, seat := range sc.seats {
		for c, yes := range sc.yesStakes {
			cand := testAddr(fmt.Sprintf("seat%dcand%d", seat, c))
			mock.guardians[cand] = true
			k.SetSeatElection(ctx, &types.SeatElectionProposal{
				ProposalId:        id,
				Proposer:          testAddr("nominator"),
				Candidate:         cand,
				SeatIndex:         seat,
				Stage:             types.SeatStageVoting,
				YesStake:          yes,
				NoStake:           "0",
				AbstainStake:      "0",
				VotingEndBlock:    sc.votingEnd,
				CandidateAccepted: true,
			})
			id++
		}
	}
	firstRunoffID := id
	k.SetNextSeatElectionID(ctx, firstRunoffID)
	cms.Commit()

	ctx = sdk.NewContext(cms, cmtproto.Header{Height: sc.tallyHeight}, false, log.NewNopLogger())
	if sc.viaBeginBlocker {
		k.BeginBlocker(ctx)
	} else {
		k.TallySeatElections(ctx)
	}

	runoffs := make(map[uint64]uint32)
	for rid := firstRunoffID; rid < firstRunoffID+uint64(2*len(sc.seats)); rid++ {
		r, ok := k.GetSeatElection(ctx, rid)
		if !ok {
			t.Fatalf("runoff proposal %d missing", rid)
		}
		if !r.IsRunoff {
			t.Fatalf("proposal %d is not a runoff", rid)
		}
		runoffs[rid] = r.SeatIndex
	}
	cms.Commit()
	return hex.EncodeToString(cms.LastCommitID().Hash), runoffs
}

// expectedAscendingRunoffs returns the runoff ID to seat mapping that ascending
// seat order produces: each seat takes the next two IDs from firstID.
func expectedAscendingRunoffs(firstID uint64, sortedSeats []uint32) map[uint64]uint32 {
	want := make(map[uint64]uint32)
	for i, seat := range sortedSeats {
		want[firstID+uint64(2*i)] = seat
		want[firstID+uint64(2*i)+1] = seat
	}
	return want
}

func assertSeatRunoffsDeterministic(t *testing.T, sc seatRunoffScenario, want map[uint64]uint32) {
	t.Helper()
	hashes := make(map[string]map[uint64]uint32)
	for i := 0; i < seatRunoffRuns; i++ {
		h, got := runSeatRunoffScenario(t, sc)
		hashes[h] = got
		for rid, seat := range want {
			if got[rid] != seat {
				t.Fatalf("run %d: runoff %d is on seat %d, want seat %d (all: %v)", i, rid, got[rid], seat, got)
			}
		}
	}
	if len(hashes) != 1 {
		for h, m := range hashes {
			t.Logf("store hash %s runoffs %v", h[:16], m)
		}
		t.Fatalf("NONDETERMINISTIC: %d distinct store hashes from identical prestate", len(hashes))
	}
}

// The review's scratch reproduction: seats 0 and 1, each with two accepted
// candidates within the 5% runoff margin, tallied directly.
func TestTallySeatElections_TwoSeatRunoffsAreDeterministic(t *testing.T) {
	sc := seatRunoffScenario{
		seats:       []uint32{0, 1},
		yesStakes:   [2]string{"500000000000", "490000000000"},
		votingEnd:   100,
		tallyHeight: 100,
	}
	assertSeatRunoffsDeterministic(t, sc, expectedAscendingRunoffs(5, []uint32{0, 1}))
}

// The verifier's form: the two seats that open vacant when BALANCED begins,
// driven through the real governance BeginBlocker.
func TestBeginBlocker_SeatRunoffsAreDeterministic(t *testing.T) {
	sc := seatRunoffScenario{
		seats:           []uint32{1, 2},
		yesStakes:       [2]string{"600000000000", "580000000000"},
		votingEnd:       50,
		tallyHeight:     50,
		viaBeginBlocker: true,
	}
	assertSeatRunoffsDeterministic(t, sc, expectedAscendingRunoffs(5, []uint32{1, 2}))
}

// Seat order is the seat index, not the order the elections were created in:
// seat 2 holds the lowest ProposalIds here, yet seat 0 takes the first runoff
// IDs.
func TestTallySeatElections_RunoffOrderIgnoresCreationOrder(t *testing.T) {
	sc := seatRunoffScenario{
		seats:       []uint32{2, 1, 0},
		yesStakes:   [2]string{"500000000000", "490000000000"},
		votingEnd:   100,
		tallyHeight: 100,
	}
	assertSeatRunoffsDeterministic(t, sc, expectedAscendingRunoffs(7, []uint32{0, 1, 2}))
}
