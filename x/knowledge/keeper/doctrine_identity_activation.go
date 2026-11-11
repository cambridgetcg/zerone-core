package keeper

import (
	"context"

	"github.com/zerone-chain/zerone/internal/activation"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// DoctrineIdentityEnabled reports whether the doctrine-identity repair is
// selected: doctrine is judged by a fact's provenance (isDoctrineFact) instead
// of its free-text category label. The marker selects the metabolism
// exemption, the doctrine rescue, the refusal of the category "doctrine" at
// admission, the empty category of a new challenge of doctrine, and the two
// confidence steps that leave doctrine alone. doctrine_identity.go and
// doctrine_reserved.go name each entry point. The marker at
// types.DoctrineIdentityEnabledKey is the only selector. Native genesis writes
// it. No carrier upgrade writes it yet; a consensus-repairs-v3 carrier will,
// at its height. No message or param toggles it, and nothing clears it.
//
// activation.Read holds the contract: the read charges no gas, and an
// unreadable or malformed marker is an error wrapping activation.ErrMarkerFault,
// never legacy state. Callers propagate it before choosing.
func (k Keeper) DoctrineIdentityEnabled(ctx context.Context) (bool, error) {
	return activation.Read(ctx, k.storeService, types.DoctrineIdentityEnabledKey, "doctrine-identity")
}

// EnableDoctrineIdentity selects the repair. It is idempotent and rewrites no
// fact, category, energy or status. Native genesis calls it; a
// consensus-repairs-v3 carrier will call it and bump this module's
// ConsensusVersion.
func (k Keeper) EnableDoctrineIdentity(ctx context.Context) error {
	return activation.Enable(ctx, k.storeService, types.DoctrineIdentityEnabledKey, "doctrine-identity")
}
