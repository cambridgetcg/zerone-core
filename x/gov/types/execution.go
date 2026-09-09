package types

import "fmt"

// ValidateLIPExecutionError validates the additive prospective failure field.
// Empty remains valid for every historical record: migration does not invent
// classifications for past votes or execution outcomes.
func ValidateLIPExecutionError(lip *LIP) error {
	if lip == nil {
		return fmt.Errorf("LIP must not be nil")
	}
	if lip.ExecutionError == "" {
		return nil
	}
	if lip.Stage != StatusFailed {
		return fmt.Errorf("LIP execution_error requires the failed stage")
	}
	switch lip.ExecutionError {
	case "execution_panicked", "execution_failed",
		"creed_payload_missing", "creed_keeper_missing", "creed_anchor_failed",
		"adapter_dispatch_unimplemented", "category_dispatch_unimplemented", "category_unsupported",
		"custom_upgrade_authority_retired", "phase_approval_failed",
		"parameter_changes_missing", "parameter_change_malformed", "parameter_dispatch_failed":
		return nil
	default:
		return fmt.Errorf("LIP execution_error is not a supported reason code")
	}
}
