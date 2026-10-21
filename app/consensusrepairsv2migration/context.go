// Package consensusrepairsv2migration reserves the two no-op version steps
// for the atomic consensus-repairs-v2 handler. It confers no release authority.
// It is an app-owned leaf package so neither module imports the application.
package consensusrepairsv2migration

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const UpgradeName = "consensus-repairs-v2"

type ownerKey struct{}

func WithOwner(ctx sdk.Context, owner string) (sdk.Context, error) {
	if owner != UpgradeName {
		return ctx, fmt.Errorf("consensus repairs v2 migration requires owner %q", UpgradeName)
	}
	return ctx.WithContext(context.WithValue(ctx.Context(), ownerKey{}, UpgradeName)), nil
}

// Require writes nothing. V1's distinct owner token cannot authorize this step.
func Require(ctx sdk.Context) error {
	if owner, ok := ctx.Context().Value(ownerKey{}).(string); !ok || owner != UpgradeName {
		return fmt.Errorf("consensus repairs v2 migration requires explicit %q execution context", UpgradeName)
	}
	return nil
}
