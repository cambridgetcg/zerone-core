package types

import (
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ValidateRecoveryAuthorization validates the fixed-size capability written
// only by a finalized Guardian recovery-authorization ceremony.
func ValidateRecoveryAuthorization(
	authorization *EmergencyRecoveryAuthorization,
) error {
	if authorization == nil {
		return nil
	}
	for name, value := range map[string]string{
		"halt_ceremony_id":          authorization.HaltCeremonyId,
		"authorization_ceremony_id": authorization.AuthorizationCeremonyId,
	} {
		if value == "" ||
			strings.TrimSpace(value) != value ||
			len(value) > 512 {
			return fmt.Errorf(
				"recovery authorization %s must be non-empty, trimmed, and at most 512 bytes",
				name,
			)
		}
	}
	if authorization.SdkGovProposalId == 0 {
		return fmt.Errorf(
			"recovery authorization sdk_gov_proposal_id must be positive",
		)
	}
	if !IsLowerSHA256(authorization.ActionSha256) {
		return fmt.Errorf(
			"recovery authorization action_sha256 must be a canonical lowercase SHA-256",
		)
	}
	if !IsLowerSHA256(authorization.RecoveryManifestSha256) {
		return fmt.Errorf(
			"recovery authorization recovery_manifest_sha256 must be a canonical lowercase SHA-256",
		)
	}
	if !IsLowerSHA256(authorization.UpgradePlanSha256) {
		return fmt.Errorf(
			"recovery authorization upgrade_plan_sha256 must be a canonical lowercase SHA-256",
		)
	}
	if _, err := sdk.AccAddressFromBech32(
		authorization.AuthorizedSubmitter,
	); err != nil {
		return fmt.Errorf(
			"recovery authorization has invalid authorized_submitter: %w",
			err,
		)
	}
	switch authorization.ActionType {
	case "software_upgrade", "cancel_upgrade":
	default:
		return fmt.Errorf(
			"recovery authorization has invalid action_type %q",
			authorization.ActionType,
		)
	}
	if authorization.AuthorizedAtBlock == 0 {
		return fmt.Errorf(
			"recovery authorization authorized_at_block must be positive",
		)
	}
	if authorization.Generation == 0 {
		return fmt.Errorf(
			"recovery authorization generation must be positive",
		)
	}
	if (authorization.TerminalAtBlock == 0) !=
		(authorization.Outcome == "") {
		return fmt.Errorf(
			"recovery authorization terminal_at_block and outcome must be both present or both absent",
		)
	}
	if authorization.TerminalAtBlock != 0 &&
		authorization.TerminalAtBlock <
			authorization.AuthorizedAtBlock {
		return fmt.Errorf(
			"recovery authorization terminal_at_block cannot precede authorized_at_block",
		)
	}
	switch authorization.Outcome {
	case "", "passed", "failed", "rejected", "revoked":
	default:
		return fmt.Errorf(
			"recovery authorization has invalid terminal outcome %q",
			authorization.Outcome,
		)
	}
	return nil
}
