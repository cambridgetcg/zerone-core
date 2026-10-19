package keeper

import (
	"bytes"
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	qualificationtypes "github.com/zerone-chain/zerone/x/qualification/types"
	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

// QualificationSemanticsEnabled selects meaning-based qualification admission.
// Absence preserves legacy admission; malformed bytes must not select it.
func (k Keeper) QualificationSemanticsEnabled(ctx context.Context) (bool, error) {
	bz := sdk.UnwrapSDKContext(ctx).KVStore(k.storeKey).Get(types.QualificationSemanticsKey)
	if bz == nil {
		return false, nil
	}
	if !bytes.Equal(bz, []byte{1}) {
		return false, fmt.Errorf("malformed substrate bridge qualification semantics marker")
	}
	return true, nil
}

// EnableQualificationSemantics is a one-way, idempotent keeper-only seam for a
// separately authorized upgrade carrier. No message, parameter or automatic
// fresh-chain initialization enables it. It changes no native qualification,
// adapter, attestation, source reference, bond or pending reward. Genesis keeps
// the marker in StateEntries so an export/import cannot silently downgrade it.
func (k Keeper) EnableQualificationSemantics(ctx context.Context) error {
	enabled, err := k.QualificationSemanticsEnabled(ctx)
	if err != nil || enabled {
		return err
	}
	sdk.UnwrapSDKContext(ctx).KVStore(k.storeKey).Set(types.QualificationSemanticsKey, []byte{1})
	return nil
}

func (k Keeper) checkQualification(ctx context.Context, submitter string, adapter *types.AdapterRegistration) error {
	if adapter.RequiredQualificationDomain == "" {
		return nil // An open adapter has no qualification requirement.
	}
	enabled, err := k.QualificationSemanticsEnabled(ctx)
	if err != nil {
		return err
	}
	if k.qualificationKeeper == nil {
		if enabled {
			return types.ErrInsufficientQualification
		}
		return nil // Preserve the legacy unwired-keeper behavior until activation.
	}
	qual, found := k.qualificationKeeper.GetDomainQualification(ctx, submitter, adapter.RequiredQualificationDomain)
	if !found || qual == nil {
		return types.ErrInsufficientQualification
	}
	if !enabled {
		// Historical numeric comparison is deliberately confined to this branch.
		if uint32(qual.Status) < uint32(adapter.MinQualificationStatus) {
			return types.ErrInsufficientQualification
		}
		return nil
	}
	if !qualificationMeetsMinimum(qual.Status, adapter.MinQualificationStatus) {
		return types.ErrInsufficientQualification
	}
	return nil
}

// qualificationMeetsMinimum compares meanings, never numbers across enums.
// UNSPECIFIED leaves the rank floor unset, not the domain's good-standing check.
// Native PROBATIONARY can satisfy that floor or PROBATIONARY, but not ACTIVE.
// Native qualification has no DISTINGUISHED status, so no record can meet it.
// Unknown minima and every non-good-standing/unknown native status fail closed.
func qualificationMeetsMinimum(status qualificationtypes.QualificationStatus, minimum types.QualificationStatus) bool {
	switch minimum {
	case types.QualificationStatus_QUALIFICATION_STATUS_UNSPECIFIED,
		types.QualificationStatus_QUALIFICATION_STATUS_PROBATIONARY:
		return status == qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_ACTIVE ||
			status == qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_PROBATIONARY
	case types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE:
		return status == qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_ACTIVE
	case types.QualificationStatus_QUALIFICATION_STATUS_DISTINGUISHED:
		return false
	default:
		return false
	}
}
