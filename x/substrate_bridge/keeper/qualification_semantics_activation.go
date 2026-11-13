package keeper

import (
	"context"

	"github.com/zerone-chain/zerone/internal/activation"
	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

// QualificationSemanticsEnabled reports whether the qualification-status
// semantics repair of zerone-core #126 is selected. The marker at
// types.QualificationSemanticsEnabledKey is the only selector. Native genesis
// writes it and the named carrier upgrade consensus-repairs-v1 writes it at its
// height. No message or parameter toggles it, and nothing clears it.
//
// activation.Read holds the contract: the read charges no gas, and an
// unreadable or malformed marker is an error wrapping activation.ErrMarkerFault,
// never legacy state. Callers propagate it before choosing.
func (k Keeper) QualificationSemanticsEnabled(ctx context.Context) (bool, error) {
	return activation.Read(ctx, activation.StoreKeyService(k.storeKey), types.QualificationSemanticsEnabledKey, "qualification semantics")
}

// EnableQualificationSemantics selects the repair. It is idempotent and
// rewrites no adapter, attestation, source reference, bond or pending reward.
// Native genesis and the consensus-repairs-v1 carrier call it; the carrier also
// bumps this module's ConsensusVersion.
func (k Keeper) EnableQualificationSemantics(ctx context.Context) error {
	return activation.Enable(ctx, activation.StoreKeyService(k.storeKey), types.QualificationSemanticsEnabledKey, "qualification semantics")
}
