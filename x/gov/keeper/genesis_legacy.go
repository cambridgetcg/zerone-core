package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/gov/types"
)

// Legacy genesis bodies from 6a56b621082064483af1ced7718462766d9ae756
// x/gov/keeper/state.go, verbatim except for function names. FX8 is genesis-only:
// wrappers add retained records, not a live activation marker or a new runtime path.
func (k Keeper) initGenesisLegacy(ctx sdk.Context, gs *types.GenesisState) {
	k.SetParams(ctx, gs.Params)
	k.SetNextLIPNumber(ctx, gs.NextLipNumber)
	if gs.EmergencyTransitionHold != nil {
		if err := k.SetEmergencyTransitionHold(
			ctx,
			gs.EmergencyTransitionHold,
		); err != nil {
			panic("invalid custom-governance emergency transition hold: " + err.Error())
		}
	} else {
		k.clearEmergencyTransitionHold(ctx)
	}

	for _, lip := range gs.Lips {
		if err := types.ValidateLIPExecutionError(lip); err != nil {
			panic(err)
		}
		lipToStore := lip
		if lip != nil &&
			lip.Category == types.CategoryUpgrade &&
			!types.IsTerminal(lip.Stage) {
			normalized := proto.Clone(lip).(*types.LIP)
			normalized.Stage = types.StatusFailed
			lipToStore = normalized
		}
		k.SetLIP(ctx, lipToStore)
	}
	for _, vote := range gs.Votes {
		k.SetVote(ctx, vote)
	}

	for _, gup := range gs.UpgradePlans {
		k.setUpgradePlanFromGenesis(ctx, gup.LipId, gup.Plan)
	}

	// Restore research fund voters from params if set.
	if gs.Params != nil && gs.Params.ResearchFundVoters != nil {
		v := gs.Params.ResearchFundVoters
		if v.Voter1 != "" && v.Voter2 != "" {
			k.SetResearchFundVoters(ctx, v)
		}
	}

	// Restore research fund governance state.
	if gs.ResearchFundGovernance != nil {
		k.SetResearchFundGovernanceState(ctx, gs.ResearchFundGovernance)
	} else {
		k.SetResearchFundGovernanceState(ctx, types.DefaultResearchFundGovernanceState())
	}

	// Restore seat elections.
	for _, se := range gs.SeatElections {
		k.SetSeatElection(ctx, se)
	}
	for _, v := range gs.SeatElectionVotes {
		k.SetSeatElectionVote(ctx, v)
	}
	if gs.NextSeatElectionNumber > 0 {
		k.SetNextSeatElectionID(ctx, gs.NextSeatElectionNumber)
	}

	// Only an explicit true writes the marker; absence keeps legacy semantics.
	if gs.SeatRunoffOrderingEnabled {
		if err := k.EnableSeatRunoffOrdering(ctx); err != nil {
			panic("invalid governance genesis: " + err.Error())
		}
	}
}

func (k Keeper) exportGenesisLegacy(ctx sdk.Context) *types.GenesisState {
	var allLIPs []*types.LIP
	k.IterateLIPs(ctx, func(lip *types.LIP) bool {
		allLIPs = append(allLIPs, lip)
		return false
	})

	params := k.GetParams(ctx)

	// Export research voters into params.
	voters := k.GetResearchFundVoters(ctx)
	if voters != nil {
		params.ResearchFundVoters = voters
	}

	var upgradePlans []*types.GenesisUpgradePlan
	k.IterateUpgradePlans(ctx, func(lipID string, plan *types.UpgradePlan) bool {
		upgradePlans = append(upgradePlans, &types.GenesisUpgradePlan{
			LipId: lipID,
			Plan:  plan,
		})
		return false
	})

	seatRunoffOrdering, err := k.SeatRunoffOrderingEnabled(ctx)
	if err != nil {
		panic(err)
	}

	return &types.GenesisState{
		AccountingSafetyEnabled: k.AccountingSafetyEnabled(ctx),
		Params:                  params,
		Lips:                    allLIPs,
		Votes:                   k.GetAllVotes(ctx),
		NextLipNumber:           k.GetNextLIPNumber(ctx),
		UpgradePlans:            upgradePlans,
		ResearchFundGovernance:  k.GetResearchFundGovernanceState(ctx),
		SeatElections:           k.GetAllSeatElections(ctx),
		SeatElectionVotes:       k.GetAllSeatElectionVotes(ctx),
		NextSeatElectionNumber:  k.GetNextSeatElectionID(ctx),
		EmergencyTransitionHold: func() *types.EmergencyTransitionHold {
			hold, found, err := k.GetEmergencyTransitionHold(ctx)
			if err != nil {
				panic(err)
			}
			if !found {
				return nil
			}
			return hold
		}(),
		SeatRunoffOrderingEnabled: seatRunoffOrdering,
	}
}
