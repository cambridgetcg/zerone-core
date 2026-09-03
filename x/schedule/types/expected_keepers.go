package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BankKeeper is deliberately transfer-only: schedules cannot mint or burn.
type BankKeeper interface {
	SendCoinsFromAccountToModule(context.Context, sdk.AccAddress, string, sdk.Coins) error
	SendCoinsFromModuleToAccount(context.Context, string, sdk.AccAddress, sdk.Coins) error
	SendCoinsFromModuleToModule(context.Context, string, string, sdk.Coins) error
	SpendableCoins(context.Context, sdk.AccAddress) sdk.Coins
	GetAllBalances(context.Context, sdk.AccAddress) sdk.Coins
	BlockedAddr(sdk.AccAddress) bool
}

// EmergencyKeeper supplies the canonical application-quarantine state. Due
// transfers pause while ordinary owner cancellation transactions are blocked.
type EmergencyKeeper interface {
	IsHalted(context.Context) bool
	GetQuarantineReleaseBlock(context.Context) uint64
}
