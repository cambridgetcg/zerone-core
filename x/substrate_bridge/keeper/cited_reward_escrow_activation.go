package keeper

import (
	"context"

	"github.com/zerone-chain/zerone/internal/activation"
	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

// CitedRewardEscrowEnabled reports whether the cited-reward escrow repair is
// selected: the reward for an attestation that cites existing facts waits in
// the existing witness escrow window instead of minting at settlement, and the
// sweep releases it only while each cited fact still stands. The
// marker at types.CitedRewardEscrowEnabledKey is the only selector. Native
// genesis writes it. No carrier upgrade writes it yet; a consensus-repairs-v3
// carrier will, at its height. No message or parameter toggles it, and
// nothing clears it.
//
// activation.Read holds the contract: the read charges no gas, and an
// unreadable or malformed marker is an error wrapping activation.ErrMarkerFault,
// never legacy state. Callers propagate it before choosing.
func (k Keeper) CitedRewardEscrowEnabled(ctx context.Context) (bool, error) {
	return activation.Read(ctx, activation.StoreKeyService(k.storeKey), types.CitedRewardEscrowEnabledKey, "cited-reward escrow")
}

// EnableCitedRewardEscrow selects the repair. It is idempotent and rewrites no
// adapter, attestation, source reference, bond or pending reward. Native
// genesis calls it; a consensus-repairs-v3 carrier will call it and bump this
// module's ConsensusVersion.
func (k Keeper) EnableCitedRewardEscrow(ctx context.Context) error {
	return activation.Enable(ctx, activation.StoreKeyService(k.storeKey), types.CitedRewardEscrowEnabledKey, "cited-reward escrow")
}
