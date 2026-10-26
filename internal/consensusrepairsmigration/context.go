// Package consensusrepairsmigration reserves the eleven no-op version steps for
// the atomic consensus-repairs-v1 application handler. This is not release authority.
package consensusrepairsmigration

import (
	"context"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const UpgradeName = "consensus-repairs-v1"

type ownerKey struct{}

func WithOwner(ctx sdk.Context, owner string) (sdk.Context, error) {
	if owner != UpgradeName {
		return ctx, fmt.Errorf("consensus repairs migration requires owner %q", UpgradeName)
	}
	return ctx.WithContext(context.WithValue(ctx.Context(), ownerKey{}, UpgradeName)), nil
}

// Require is the guarded no-op migration registered by each affected module.
// Only the application handler writes markers, after all preconditions pass.
func Require(ctx sdk.Context) error {
	if owner, ok := ctx.Context().Value(ownerKey{}).(string); !ok || owner != UpgradeName {
		return fmt.Errorf("consensus repairs migration requires explicit %q execution context", UpgradeName)
	}
	return nil
}
