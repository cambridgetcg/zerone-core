package keeper

import "context"

// GetCurrentPinVersion returns the latest pinned creed version, or 0 if
// no pin has been recorded. This legacy compatibility helper is retained for
// downstream interfaces that used the retired generic x/contribution adapter;
// it is not app-wired in the current runtime.
//
// Thin alias for GetCurrentVersion preserving the contribution-side
// vocabulary ("PinVersion") used by the truth-floor attestation.
func (k Keeper) GetCurrentPinVersion(ctx context.Context) uint32 {
	return k.GetCurrentVersion(ctx)
}
