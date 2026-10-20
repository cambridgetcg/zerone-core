package keeper

import (
	"bytes"
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

// AlignmentBankSupplyEnabled reports whether the alignment bank-supply repair
// of zerone-core #122 is selected. The marker at
// types.AlignmentBankSupplyEnabledKey is the only selector. Native genesis
// writes it and the named carrier upgrade consensus-repairs-v1 writes it at its
// height. No message or parameter toggles it, and nothing clears it.
//
// The read uses its own gas meter, so a chain without the marker spends exactly
// the gas it spent before this code existed. An unreadable or malformed marker
// is an error, never legacy state; callers propagate it before choosing.
func (k Keeper) AlignmentBankSupplyEnabled(ctx context.Context) (bool, error) {
	ungassed := sdk.UnwrapSDKContext(ctx).WithGasMeter(storetypes.NewInfiniteGasMeter())
	bz, err := k.storeService.OpenKVStore(ungassed).Get(types.AlignmentBankSupplyEnabledKey)
	if err != nil {
		return false, fmt.Errorf("read alignment bank-supply marker: %w", err)
	}
	if bz == nil {
		return false, nil
	}
	if !bytes.Equal(bz, []byte{1}) {
		return false, fmt.Errorf("malformed alignment bank-supply marker")
	}
	return true, nil
}

// EnableAlignmentBankSupply selects the repair. It is idempotent and rewrites
// no vesting schedule, clawback record or minted-supply ledger. Native genesis
// and the consensus-repairs-v1 carrier call it; the carrier also bumps this
// module's ConsensusVersion.
func (k Keeper) EnableAlignmentBankSupply(ctx context.Context) error {
	enabled, err := k.AlignmentBankSupplyEnabled(ctx)
	if err != nil || enabled {
		return err
	}
	if err := k.storeService.OpenKVStore(ctx).Set(types.AlignmentBankSupplyEnabledKey, []byte{1}); err != nil {
		return fmt.Errorf("write alignment bank-supply marker: %w", err)
	}
	return nil
}
