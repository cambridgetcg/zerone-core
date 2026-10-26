package keeper

import (
	"context"
	"errors"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	alignmenttypes "github.com/zerone-chain/zerone/x/alignment/types"
	"github.com/zerone-chain/zerone/x/staking/types"
)

var _ alignmenttypes.StakingRegistry = (*AlignmentStakingAdapter)(nil)

// GetRegisteredValidatorCount counts occupied validator keys, including inactive
// validators and malformed records. Unlike IterateValidators, it must not skip
// undecodable records and accidentally claim an occupied registry is empty.
// This is a presence signal, not a validity census; the carrier audits records.
// No marker is read, no state is written, and no active-set limit is applied.
func (a *AlignmentStakingAdapter) GetRegisteredValidatorCount(ctx context.Context) (count uint64, err error) {
	iter := storetypes.KVStorePrefixIterator(a.k.getStore(sdk.UnwrapSDKContext(ctx)), types.ValidatorKeyPrefix)
	defer func() {
		if cause := errors.Join(accountingIteratorError(iter), iter.Close()); cause != nil {
			err = fmt.Errorf("count alignment staking registry: %w", cause)
		}
	}()
	for ; iter.Valid(); iter.Next() {
		count++
	}
	return count, nil
}
