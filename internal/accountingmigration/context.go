// Package accountingmigration owns the process-local capability for the
// accounting-authority migration. It is not release or network authority.
// Only the named, precondition-checked upgrade handler issues this capability.
package accountingmigration

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const UpgradeName = "accounting-authority-v1"

type ownerKey struct{}

// WithOwner scopes module migration execution to its one reserved owner. The
// unexported context key cannot be supplied by transaction or configuration data.
func WithOwner(ctx sdk.Context, owner string) (sdk.Context, error) {
	if owner != UpgradeName {
		return ctx, fmt.Errorf("accounting migration requires owner %q, got %q", UpgradeName, owner)
	}
	return ctx.WithContext(context.WithValue(ctx.Context(), ownerKey{}, UpgradeName)), nil
}

// Require rejects broad RunMigrations calls without the owner capability.
func Require(ctx sdk.Context) error {
	if owner, ok := ctx.Context().Value(ownerKey{}).(string); !ok || owner != UpgradeName {
		return fmt.Errorf("accounting migration requires explicit %q execution context", UpgradeName)
	}
	return nil
}
