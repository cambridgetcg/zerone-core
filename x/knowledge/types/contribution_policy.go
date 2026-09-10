package types

import "fmt"

// ValidateContributionRecord distinguishes an archived heuristic valuation
// from an owner declaration. It does not certify that the declared use occurred.
func ValidateContributionRecord(record *ContributionRecord) error {
	if record == nil || record.ModelId == "" {
		return fmt.Errorf("invalid contribution record")
	}
	switch record.AttributionPolicyVersion {
	case 0:
		// Preserve the historical representation without reinterpreting it.
	case 1:
		if record.ComputedTvw != 0 || len(record.PerFactCalibrationBps) != 0 {
			return fmt.Errorf("owner-declared contribution cannot contain computed value or calibration snapshots")
		}
	default:
		return fmt.Errorf("unknown contribution attribution policy %d", record.AttributionPolicyVersion)
	}
	return nil
}
