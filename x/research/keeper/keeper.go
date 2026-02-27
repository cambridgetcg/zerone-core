package keeper

import (
	"fmt"
	"math/big"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/research/types"
)

// Keeper manages the research module's state.
type Keeper struct {
	cdc          codec.Codec
	storeService store.KVStoreService
	bankKeeper   types.BankKeeper
	authority    string
}

// NewKeeper creates a new research module Keeper.
func NewKeeper(
	storeService store.KVStoreService,
	cdc codec.Codec,
	authority string,
	bk types.BankKeeper,
) Keeper {
	return Keeper{
		cdc:          cdc,
		storeService: storeService,
		bankKeeper:   bk,
		authority:    authority,
	}
}

// prefixEndBytes returns the end key for prefix iteration (exclusive).
func prefixEndBytes(prefix []byte) []byte {
	if len(prefix) == 0 {
		return nil
	}
	end := make([]byte, len(prefix))
	copy(end, prefix)
	for i := len(end) - 1; i >= 0; i-- {
		end[i]++
		if end[i] != 0 {
			return end
		}
	}
	return nil
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// GetAuthority returns the module authority address.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// InitGenesis initializes the module's state from genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, genState *types.GenesisState) {
	if genState.Params != nil {
		k.SetParams(ctx, genState.Params)
	}
	if genState.TreasuryBalance != nil {
		k.SetTreasuryBalance(ctx, genState.TreasuryBalance.Balance)
	}

	for _, sub := range genState.Researches {
		if sub != nil {
			k.SetResearch(ctx, sub)
		}
	}
	for _, bounty := range genState.Bounties {
		if bounty != nil {
			k.SetBounty(ctx, bounty)
		}
	}
	for _, review := range genState.PeerReviews {
		if review != nil {
			k.SetPeerReview(ctx, review)
		}
	}
}

// AutoResolveResearch resolves research submissions that have met review conditions.
func (k Keeper) AutoResolveResearch(ctx sdk.Context) error {
	params := k.GetParams(ctx)
	currentBlock := uint64(ctx.BlockHeight())

	researches := k.GetResearchesByStatus(ctx, types.ResearchStatusUnderReview)
	for _, research := range researches {
		if currentBlock-research.UpdatedAt < params.ReviewPeriodBlocks {
			continue
		}
		if research.ReviewCount < params.MinReviewerCount {
			continue
		}

		stakeInt := new(big.Int)
		stakeInt.SetString(research.Stake, 10)

		if research.AggregateScore >= params.AcceptanceScoreThreshold {
			research.Status = string(types.ResearchStatusAccepted)

			submitterAddr, err := sdk.AccAddressFromBech32(research.Submitter)
			if err != nil {
				k.Logger(ctx).Error("invalid submitter address", "research_id", research.Id, "error", err)
				continue
			}
			coins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(stakeInt)))
			if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, submitterAddr, coins); err != nil {
				k.Logger(ctx).Error("failed to return stake", "research_id", research.Id, "error", err)
				continue
			}
		} else {
			research.Status = string(types.ResearchStatusRejected)

			slashRate := new(big.Int).SetUint64(params.RejectionSlashBps)
			slashAmount := new(big.Int).Mul(stakeInt, slashRate)
			slashAmount.Div(slashAmount, new(big.Int).SetUint64(1000000))

			if slashAmount.Sign() > 0 {
				slashCoins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(slashAmount)))
				if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, "development_fund", slashCoins); err != nil {
					k.Logger(ctx).Error("failed to slash to dev fund", "research_id", research.Id, "error", err)
					continue
				}
			}

			remainder := new(big.Int).Sub(stakeInt, slashAmount)
			if remainder.Sign() > 0 {
				submitterAddr, err := sdk.AccAddressFromBech32(research.Submitter)
				if err != nil {
					k.Logger(ctx).Error("invalid submitter address", "research_id", research.Id, "error", err)
					continue
				}
				returnCoins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(remainder)))
				if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, submitterAddr, returnCoins); err != nil {
					k.Logger(ctx).Error("failed to return remainder", "research_id", research.Id, "error", err)
					continue
				}
			}
		}

		research.UpdatedAt = currentBlock
		k.SetResearch(ctx, research)

		var outcomeStr string
		if research.Status == string(types.ResearchStatusAccepted) {
			outcomeStr = "accepted"
		} else {
			outcomeStr = "rejected"
		}

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				"zerone.research.research_auto_resolved",
				sdk.NewAttribute("research_id", research.Id),
				sdk.NewAttribute("outcome", outcomeStr),
				sdk.NewAttribute("aggregate_score", fmt.Sprintf("%d", research.AggregateScore)),
			),
		)
	}
	return nil
}

// ExportGenesis exports the module's state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	var researches []*types.Research
	k.IterateResearches(ctx, func(r *types.Research) bool {
		researches = append(researches, r)
		return false
	})

	var bounties []*types.Bounty
	k.IterateBounties(ctx, func(b *types.Bounty) bool {
		bounties = append(bounties, b)
		return false
	})

	var peerReviews []*types.PeerReview
	k.IterateResearches(ctx, func(r *types.Research) bool {
		reviews := k.GetReviewsForResearch(ctx, r.Id)
		peerReviews = append(peerReviews, reviews...)
		return false
	})

	params := k.GetParams(ctx)
	return &types.GenesisState{
		Params:          params,
		Researches:      researches,
		Bounties:        bounties,
		PeerReviews:     peerReviews,
		TreasuryBalance: &types.TreasuryBalance{Balance: k.GetTreasuryBalance(ctx)},
	}
}
