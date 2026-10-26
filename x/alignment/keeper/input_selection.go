package keeper

import (
	"context"

	"github.com/zerone-chain/zerone/x/alignment/types"
)

// SelectInputs reads each wired marker once, before EndBlock can write or emit
// events. It deliberately ignores any incoming context selection: each entry
// point must take a fresh snapshot of committed activation state.
func (k Keeper) SelectInputs(ctx context.Context) (context.Context, error) {
	var selection types.InputSelection
	var err error
	if reader, ok := k.vestingRewardsKeeper.(types.BankSupplySelector); ok {
		selection.BankSupply, err = reader.AlignmentBankSupplyEnabled(ctx)
		if err != nil {
			return nil, err
		}
	}
	if reader, ok := k.ontologyKeeper.(types.ActiveDomainCountSelector); ok {
		selection.ActiveDomainCount, err = reader.ActiveDomainCountEnabled(ctx)
		if err != nil {
			return nil, err
		}
	}
	return types.WithInputSelection(ctx, selection), nil
}
