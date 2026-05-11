package keeper

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

func (k Keeper) LinkPendingClaim(ctx context.Context, pendingClaimID, attestationID string) error {
	store := sdk.UnwrapSDKContext(ctx).KVStore(k.storeKey)
	store.Set(types.PendingFactIndexKey(pendingClaimID), []byte(attestationID))
	store.Set(types.AttestationPendingClaimsKey(attestationID, pendingClaimID), []byte{0x01})
	return nil
}

func (k Keeper) GetAttestationForPendingClaim(ctx context.Context, pendingClaimID string) (string, bool) {
	bz := sdk.UnwrapSDKContext(ctx).KVStore(k.storeKey).Get(types.PendingFactIndexKey(pendingClaimID))
	if bz == nil {
		return "", false
	}
	return string(bz), true
}

func (k Keeper) PendingClaimsFor(ctx context.Context, attestationID string) []string {
	store := sdk.UnwrapSDKContext(ctx).KVStore(k.storeKey)
	prefix := types.AttestationPendingClaimsPrefixFor(attestationID)
	iter := storetypes.KVStorePrefixIterator(store, prefix)
	defer iter.Close()
	var out []string
	prefixLen := len(prefix)
	for ; iter.Valid(); iter.Next() {
		out = append(out, string(iter.Key()[prefixLen:]))
	}
	return out
}

func (k Keeper) UnlinkPendingClaim(ctx context.Context, pendingClaimID, attestationID string) error {
	store := sdk.UnwrapSDKContext(ctx).KVStore(k.storeKey)
	store.Delete(types.PendingFactIndexKey(pendingClaimID))
	store.Delete(types.AttestationPendingClaimsKey(attestationID, pendingClaimID))
	return nil
}
