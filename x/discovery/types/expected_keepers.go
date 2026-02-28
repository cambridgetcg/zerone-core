package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BankKeeper defines the expected bank module interface.
type BankKeeper interface {
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}

// PacingKeeper provides global pacing signals for adaptive discovery timing (R29-6).
type PacingKeeper interface {
	GetGlobalPacingMultiplier(ctx context.Context) (creationBps, analysisBps uint64)
}
