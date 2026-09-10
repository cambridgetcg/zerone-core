// Package reviewmigration scopes the knowledge record migration to its named
// application handler. This process-local context is not release authority.
package reviewmigration

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const UpgradeName = "knowledge-review-neutrality-v1"

type ownerKey struct{}

func WithOwner(ctx sdk.Context, owner string) (sdk.Context, error) {
	if owner != UpgradeName {
		return ctx, fmt.Errorf("review neutrality migration requires owner %q", UpgradeName)
	}
	return ctx.WithContext(context.WithValue(ctx.Context(), ownerKey{}, UpgradeName)), nil
}

func Require(ctx sdk.Context) error {
	if owner, ok := ctx.Context().Value(ownerKey{}).(string); !ok || owner != UpgradeName {
		return fmt.Errorf("review neutrality migration requires explicit %q execution context", UpgradeName)
	}
	return nil
}
