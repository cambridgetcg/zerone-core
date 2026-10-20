package keeper

import (
	"context"

	qualificationtypes "github.com/zerone-chain/zerone/x/qualification/types"
	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

// checkQualification is the activated path only. The message entry point has
// already read the marker; helpers and per-record checks must not read it again.
func (k Keeper) checkQualification(ctx context.Context, submitter string, adapter *types.AdapterRegistration) error {
	if adapter.RequiredQualificationDomain == "" {
		return nil // An open adapter has no qualification requirement.
	}
	if k.qualificationKeeper == nil {
		return types.ErrInsufficientQualification
	}
	qual, found := k.qualificationKeeper.GetDomainQualification(ctx, submitter, adapter.RequiredQualificationDomain)
	if !found || qual == nil {
		return types.ErrInsufficientQualification
	}
	if !qualificationMeetsMinimum(qual.Status, adapter.MinQualificationStatus) {
		return types.ErrInsufficientQualification
	}
	return nil
}

// qualificationMeetsMinimum compares meanings, never numbers across enums.
// UNSPECIFIED leaves the rank floor unset, not the domain's admissibility check.
// Native PROBATIONARY can satisfy that floor or PROBATIONARY, but not ACTIVE.
// This includes accuracy-decay demotions; it is intentionally not IsQualified,
// which selects ACTIVE only. The bridge enum is not the native enum.
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

// Registration validation is prospective. Do not put it in WriteAdapter or
// genesis restoration: historical adapters and their economic records must
// still round-trip, including minima that cannot admit new work after activation.
func validateAdapterQualificationMinimum(adapter *types.AdapterRegistration) error {
	switch adapter.MinQualificationStatus {
	case types.QualificationStatus_QUALIFICATION_STATUS_UNSPECIFIED,
		types.QualificationStatus_QUALIFICATION_STATUS_PROBATIONARY,
		types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE:
		return nil
	default:
		return types.ErrInvalidQualificationMinimum.Wrapf("minimum %d has no native qualification meaning", adapter.MinQualificationStatus)
	}
}
