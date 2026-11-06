package keeper

import (
	"context"

	"github.com/zerone-chain/zerone/internal/activation"
	"github.com/zerone-chain/zerone/x/ontology/types"
)

// ActiveDomainCountEnabled reports whether the active-domain-count repair of
// zerone-core #124 is selected. The marker at types.ActiveDomainCountEnabledKey
// is the only selector. Native genesis writes it and the named carrier upgrade
// consensus-repairs-v1 writes it at its height. No message or parameter toggles
// it, and nothing clears it.
//
// activation.Read holds the contract: the read charges no gas, and an
// unreadable or malformed marker is an error wrapping activation.ErrMarkerFault,
// never legacy state. Callers propagate it before choosing.
func (k Keeper) ActiveDomainCountEnabled(ctx context.Context) (bool, error) {
	return activation.Read(ctx, k.storeService, types.ActiveDomainCountEnabledKey, "active domain count")
}

// EnableActiveDomainCount selects the repair. It is idempotent and rewrites no
// stratum, domain, proposal or link. Native genesis and the
// consensus-repairs-v1 carrier call it; the carrier also bumps this module's
// ConsensusVersion.
func (k Keeper) EnableActiveDomainCount(ctx context.Context) error {
	return activation.Enable(ctx, k.storeService, types.ActiveDomainCountEnabledKey, "active domain count")
}
