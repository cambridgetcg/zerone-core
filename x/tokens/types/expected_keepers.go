package types

import (
	"context"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BankKeeper defines the expected bank keeper interface for wrap/unwrap operations.
type BankKeeper interface {
	MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
	BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	GetSupply(ctx context.Context, denom string) sdk.Coin
}

// VestingRewardsKeeper is the chain's single cap-gated mint entry point
// (x/vesting_rewards.MintWithCap). Emission-period minting routes through it
// so no schedule can push total supply past the 222,222,222 ZRN cap. Wired
// post-init in app.go; nil = direct-mint fallback (isolated unit tests only).
type VestingRewardsKeeper interface {
	MintWithCap(ctx sdk.Context, recipientModule string, amount *big.Int) (*big.Int, error)
}
