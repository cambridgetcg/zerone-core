package types

import "fmt"

const (
	ReviewPolicyLegacy  uint32 = 0
	ReviewPolicyNeutral uint32 = 1
)

func ValidateReviewPolicyVersion(version uint32, enabled bool) error {
	switch version {
	case ReviewPolicyLegacy:
		return nil
	case ReviewPolicyNeutral:
		if enabled {
			return nil
		}
		return fmt.Errorf("neutral review policy requires activation")
	default:
		return fmt.Errorf("unsupported review policy version %d", version)
	}
}
