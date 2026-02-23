package v2

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/zerone-chain/zerone/x/knowledge/keeper"
)

// Migrate performs the v2 migration for the knowledge module.
func Migrate(ctx sdk.Context, k keeper.Keeper) error {
	// No-op for initial version. Future migrations go here.
	return nil
}
