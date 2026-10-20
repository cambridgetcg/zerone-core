package keeper

import (
	"bytes"
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/capture_challenge/types"
)

// ChallengeAccountingEnabled reports whether the rejected-stake accounting and
// expiry-refund repair of zerone-core #121 is selected. The marker at
// types.ChallengeAccountingEnabledKey is the only selector. Native genesis
// writes it and the named carrier upgrade consensus-repairs-v1 writes it at its
// height. No message or parameter toggles it, and nothing clears it.
//
// The read uses its own gas meter, so a chain without the marker spends exactly
// the gas it spent before this code existed. An unreadable or malformed marker
// is an error, never legacy state; callers propagate it before choosing.
func (k Keeper) ChallengeAccountingEnabled(ctx context.Context) (bool, error) {
	ungassed := sdk.UnwrapSDKContext(ctx).WithGasMeter(storetypes.NewInfiniteGasMeter())
	bz, err := k.storeService.OpenKVStore(ungassed).Get(types.ChallengeAccountingEnabledKey)
	if err != nil {
		return false, fmt.Errorf("read challenge accounting marker: %w", err)
	}
	if bz == nil {
		return false, nil
	}
	if !bytes.Equal(bz, []byte{1}) {
		return false, fmt.Errorf("malformed challenge accounting marker")
	}
	return true, nil
}

// EnableChallengeAccounting selects the repair. It is idempotent and rewrites
// no challenge or bounty pool. Native genesis and the consensus-repairs-v1
// carrier call it; the carrier also bumps this module's ConsensusVersion.
func (k Keeper) EnableChallengeAccounting(ctx context.Context) error {
	enabled, err := k.ChallengeAccountingEnabled(ctx)
	if err != nil || enabled {
		return err
	}
	if err := k.storeService.OpenKVStore(ctx).Set(types.ChallengeAccountingEnabledKey, []byte{1}); err != nil {
		return fmt.Errorf("write challenge accounting marker: %w", err)
	}
	return nil
}
