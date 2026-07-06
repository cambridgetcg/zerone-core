package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// CosmosAccountKeeper defines the expected interface for the standard Cosmos x/auth keeper.
type CosmosAccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	SetAccount(ctx context.Context, acc sdk.AccountI)
	NewAccountWithAddress(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
}

// The BankKeeper dependency was removed with the dormant bootstrap
// auto-claim (2026-07 slim cut): the identity module holds no funds and
// disburses nothing — the real bootstrap path is x/claiming_pot.
