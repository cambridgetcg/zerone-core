package keeper

import (
	"bytes"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/gov/types"
)

// AccountingSafetyEnabled separates prospective atomic execution from the
// historical rules. Corrupt marker bytes must never select a permissive branch.
func (k Keeper) AccountingSafetyEnabled(ctx sdk.Context) bool {
	bz := ctx.KVStore(k.storeKey).Get(types.AccountingSafetyKey)
	if bz == nil {
		return false
	}
	if !bytes.Equal(bz, []byte{1}) {
		panic("invalid governance accounting safety marker")
	}
	return true
}

// ValidateAccountingSafety proves that the repaired execution marker is present
// and exact without changing state. Startup coordination uses it after activation.
func (k Keeper) ValidateAccountingSafety(ctx sdk.Context) error {
	if !bytes.Equal(ctx.KVStore(k.storeKey).Get(types.AccountingSafetyKey), []byte{1}) {
		return fmt.Errorf("governance accounting safety marker is missing or invalid")
	}
	return nil
}

// EnableAccountingSafety is an idempotent keeper-only activation seam for native
// genesis and the separately owned accounting-authority migration. It changes
// no historical proposal, vote, aggregate escrow, or target module authority.
func (k Keeper) EnableAccountingSafety(ctx sdk.Context) error {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.AccountingSafetyKey)
	if bz != nil && !bytes.Equal(bz, []byte{1}) {
		return fmt.Errorf("invalid governance accounting safety marker")
	}
	store.Set(types.AccountingSafetyKey, []byte{1})
	return nil
}

// lipExecutionError separates bounded consensus-visible codes from diagnostic
// handler errors. New codes require source review; no handler selects a code.
type lipExecutionError struct {
	code  string
	cause error
}

func lipExecutionFailure(code string, cause error) error {
	return &lipExecutionError{code: code, cause: cause}
}

func (e *lipExecutionError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.code, e.cause)
	}
	return e.code
}

func executionErrorCode(err error) string {
	if err == nil {
		return ""
	}
	if failure, ok := err.(*lipExecutionError); ok {
		return failure.code
	}
	if _, ok := err.(*paramExecutionError); ok {
		return "parameter_dispatch_failed"
	}
	return "execution_failed"
}
