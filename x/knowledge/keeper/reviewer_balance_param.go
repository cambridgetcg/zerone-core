package keeper

import (
	"context"
	"fmt"

	sdkmath "cosmossdk.io/math"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// reviewerMinBalance returns the bank balance, in uzrn, that SubmitCommitment
// requires of a reviewer. SubmitCommitment calls it once, before it reads the
// reviewer's balance.
//
// With the reviewer-balance-param marker off it returns the built-in 100 ZRN
// and reads no parameter, so the commit path spends exactly the gas it spent
// before: the marker read itself is gas-free (internal/activation). With the
// marker on it reads Params.MinReviewerBalance (gas-metered, like any param
// read); an empty value stands for the same built-in 100 ZRN. A marker fault is
// returned, never read as off.
//
// What this does not do: it sets no vote weight, bond or reputation, and it is
// not Sybil resistance. The check is a balance snapshot at commit time; nothing
// is locked, and the same coins can seat one account and then another. A lower
// figure lowers the cost of holding many reviewer seats. Governance changes the
// figure only through MsgUpdateParams.
func (k Keeper) reviewerMinBalance(ctx context.Context) (sdkmath.Int, error) {
	enabled, err := k.ReviewerBalanceParamEnabled(ctx)
	if err != nil {
		return sdkmath.Int{}, err
	}
	if !enabled {
		return sdkmath.NewIntFromUint64(types.BuiltInMinReviewerBalanceUzrn), nil
	}
	params, err := k.GetParams(ctx)
	if err != nil {
		return sdkmath.Int{}, fmt.Errorf("load params: %w", err)
	}
	amount, err := types.ParseMinReviewerBalance(params.MinReviewerBalance)
	if err != nil {
		return sdkmath.Int{}, fmt.Errorf("stored params: %w", err)
	}
	return sdkmath.NewIntFromUint64(amount), nil
}

// checkReviewerBalanceParamChange refuses a runtime change of
// min_reviewer_balance while the reviewer-balance-param marker is off. With
// the marker off execution never reads the field, and before this field
// existed no parameter update could carry it, so an off chain keeps refusing
// it rather than store a control it would silently ignore. An update that
// leaves the field as it is does not read the marker at all, so every existing
// update runs as before.
//
// The refusal is not at the same stage as before. A binary without the field
// refuses a tx that carries it at decode (unknownproto: field 161 is an unknown
// critical field), before the ante handler runs. This binary decodes it, and a
// governance proposal that wraps such an update is stored and fails here, at
// execution. Fees, sequences, stored proposals and result codes then differ
// from the old binary for the same block. So even with the marker off this
// binary cannot run beside the old one: it has to be adopted at a coordinated
// upgrade height, like any change to a tx-reachable message.
func (k Keeper) checkReviewerBalanceParamChange(ctx context.Context, current, proposed *types.Params) error {
	if proposed.MinReviewerBalance == current.MinReviewerBalance {
		return nil
	}
	enabled, err := k.ReviewerBalanceParamEnabled(ctx)
	if err != nil {
		return err
	}
	if !enabled {
		return fmt.Errorf("min_reviewer_balance requires the reviewer-balance-param activation")
	}
	return nil
}
