package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BankKeeper defines the expected bank module keeper interface.
type BankKeeper interface {
	SendCoins(ctx context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}

// KnowledgeKeeper defines the expected knowledge module keeper interface.
// Adapters bridge the concrete knowledge keeper to this interface.
type KnowledgeKeeper interface {
	GetFactConfidence(ctx context.Context, factId string) (confidence uint64, found bool)
}

// BillingKeeper defines the expected billing module keeper interface.
type BillingKeeper interface {
	// Placeholder — billing integration for BVM queries is future work.
}

// HomeKeeper defines the expected home module keeper interface.
type HomeKeeper interface {
	// Placeholder — home integration for BVM is future work.
}
