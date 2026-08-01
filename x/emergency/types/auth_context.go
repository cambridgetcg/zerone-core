package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// authenticatedEmergencyTxKey is intentionally private. Only the application
// ante chain can mark a context after proving that the transaction contains
// direct, isolated emergency messages. EndBlock governance execution, IBC
// callbacks, authz wrappers, and direct message-router calls do not inherit the
// marker.
type authenticatedEmergencyTxKey struct{}

// WithAuthenticatedEmergencyTx marks the SDK context passed through the ante
// chain for direct emergency coordination messages. Signature verification
// still runs later in the ante chain before any message is executed.
func WithAuthenticatedEmergencyTx(ctx sdk.Context) sdk.Context {
	return ctx.WithValue(authenticatedEmergencyTxKey{}, true)
}

// HasAuthenticatedEmergencyTx reports whether the application ante chain
// authenticated this execution path as a direct emergency transaction.
func HasAuthenticatedEmergencyTx(ctx context.Context) bool {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	authenticated, _ := sdkCtx.Value(authenticatedEmergencyTxKey{}).(bool)
	return authenticated
}
