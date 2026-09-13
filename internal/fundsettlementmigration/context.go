// Package fundsettlementmigration scopes future funding-term activation to its
// named application handler. This process-local context is not release authority.
package fundsettlementmigration

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const UpgradeName = "knowledge-fund-settlement-v1"

type ownerKey struct{}

func WithOwner(ctx sdk.Context, owner string) (sdk.Context, error) {
	if owner != UpgradeName {
		return ctx, fmt.Errorf("fund settlement migration requires owner %q", UpgradeName)
	}
	return ctx.WithContext(context.WithValue(ctx.Context(), ownerKey{}, UpgradeName)), nil
}

func Require(ctx sdk.Context) error {
	if owner, ok := ctx.Context().Value(ownerKey{}).(string); !ok || owner != UpgradeName {
		return fmt.Errorf("fund settlement migration requires explicit %q execution context", UpgradeName)
	}
	return nil
}
