package keeper

import (
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

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
