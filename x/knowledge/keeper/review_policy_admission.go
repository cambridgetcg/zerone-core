package keeper

import "context"

// reviewPolicyAtAdmission freezes the applicable terms before a transaction
// collects funds or changes records. Existing claims keep their stored policy.
func (k Keeper) reviewPolicyAtAdmission(ctx context.Context) (uint32, error) {
	enabled, err := k.ReviewNeutralityEnabled(ctx)
	if err != nil {
		return 0, err
	}
	if enabled {
		return 1, nil
	}
	return 0, nil
}
