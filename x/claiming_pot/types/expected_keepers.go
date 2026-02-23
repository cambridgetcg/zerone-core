package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// StakingKeeper defines the expected staking module interface for tier checks.
type StakingKeeper interface {
	GetValidatorTier(ctx context.Context, addr string) (uint32, error)
}

// AuthKeeper defines the expected auth module interface for registration age.
type AuthKeeper interface {
	GetRegistrationBlock(ctx context.Context, addr string) (uint64, error)
}

// BankKeeper defines the expected bank module interface for fund transfers.
type BankKeeper interface {
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}
