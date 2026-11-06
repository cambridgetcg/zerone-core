package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/gov/types"
)

// TallySeatElectionsLegacyReference is the verbatim pre-ordering tally from
// 778cd69ce8a5bd159a0d32a67124d42e8c576c35, except for this test-only name.
// Keep the old map loop and per-seat dispatch independent of the gated loop.
func (k Keeper) TallySeatElectionsLegacyReference(ctx sdk.Context) {
	currentHeight := uint64(ctx.BlockHeight())
	params := k.GetParams(ctx)

	// Collect voting-stage proposals whose voting has ended.
	var readyToTally []*types.SeatElectionProposal
	k.IterateSeatElections(ctx, func(prop *types.SeatElectionProposal) bool {
		if prop.Stage == types.SeatStageVoting && prop.VotingEndBlock > 0 && currentHeight >= prop.VotingEndBlock {
			readyToTally = append(readyToTally, prop)
		}
		return false
	})

	if len(readyToTally) == 0 {
		return
	}

	// Group by seat index.
	bySeat := make(map[uint32][]*types.SeatElectionProposal)
	for _, prop := range readyToTally {
		bySeat[prop.SeatIndex] = append(bySeat[prop.SeatIndex], prop)
	}

	for seatIndex, candidates := range bySeat {
		if len(candidates) == 1 {
			// Single candidate: standard quorum + support check.
			prop := candidates[0]
			quorumMet, passed := k.checkSeatElectionQuorum(ctx, prop, params)
			if quorumMet && passed {
				prop.Stage = types.SeatStagePassed
				k.SetSeatElection(ctx, prop)
				if err := k.InstallCommunitySeat(ctx, prop.SeatIndex, prop.Candidate, currentHeight); err != nil {
					k.Logger(ctx).Error("failed to install community seat",
						"proposal_id", prop.ProposalId,
						"candidate", prop.Candidate,
						"error", err,
					)
				}
			} else {
				prop.Stage = types.SeatStageFailed
				k.SetSeatElection(ctx, prop)
			}

			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					"zerone.gov.seat_election_tallied",
					sdk.NewAttribute("proposal_id", fmt.Sprintf("%d", prop.ProposalId)),
					sdk.NewAttribute("outcome", prop.Stage),
					sdk.NewAttribute("yes_stake", prop.YesStake),
					sdk.NewAttribute("no_stake", prop.NoStake),
				),
			)
		} else if candidates[0].IsRunoff {
			// Runoff: pick highest yes_stake among those passing quorum+support.
			k.resolveRunoff(ctx, seatIndex, candidates, params, currentHeight)
		} else {
			// Multiple candidates for same seat: rank by yes_stake with quorum filter.
			k.resolveContestedSeat(ctx, seatIndex, candidates, currentHeight)
		}
	}
}
