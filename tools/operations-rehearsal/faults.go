package main

import (
	"fmt"
	"sort"
)

var faultDefinitions = []FaultDefinition{
	{ReasonCode: "PLAN_NAME_MISMATCH", Phase: "preflight", Core: false, Description: "on-chain upgrade plan name differs from the pinned release"},
	{ReasonCode: "PLAN_HEIGHT_MISMATCH", Phase: "preflight", Core: false, Description: "on-chain upgrade height differs from the handoff boundary"},
	{ReasonCode: "PLAN_INFO_MISMATCH", Phase: "preflight", Core: true, Description: "Plan.Info does not match the stopped-database keyset manifest"},
	{ReasonCode: "SOURCE_VERSION_MISMATCH", Phase: "preflight", Core: false, Description: "legacy module versions do not match the migration contract"},
	{ReasonCode: "UNSAFE_SKIP_ATTEMPT", Phase: "preflight", Core: false, Description: "an operator attempts to skip a required store loader"},
	{ReasonCode: "PRESTATE_MUTATION", Phase: "preflight", Core: false, Description: "the source database changes during read-only activation preflight"},
	{ReasonCode: "PRESTATE_SYMLINK_OR_SPECIAL", Phase: "preflight", Core: false, Description: "preflight input contains a symlink or special file"},
	{ReasonCode: "PRESTATE_GENESIS_DRIFT", Phase: "preflight", Core: false, Description: "genesis bytes change during preflight"},
	{ReasonCode: "PRESTATE_UPGRADE_INFO_DRIFT", Phase: "preflight", Core: false, Description: "upgrade-info bytes change during preflight"},
	{ReasonCode: "FEE_STATE_UNSAFE", Phase: "preflight", Core: false, Description: "fee middleware balance or lock makes store retirement unsafe"},
	{ReasonCode: "OBSOLETE_KEYSET_MISMATCH", Phase: "preflight", Core: false, Description: "obsolete IBC key census differs from Plan.Info"},
	{ReasonCode: "ACTIVE_GOVERNANCE_CONFLICT", Phase: "preflight", Core: false, Description: "an active governance action conflicts with migration"},

	{ReasonCode: "OLD_BINARY_DID_NOT_STOP", Phase: "handoff", Core: false, Description: "legacy process remains active at the exact height"},
	{ReasonCode: "OLD_EXIT_EVIDENCE_COUNT", Phase: "handoff", Core: false, Description: "legacy upgrade-needed evidence is missing or duplicated"},
	{ReasonCode: "ARM_AFTER_BOUNDARY", Phase: "handoff", Core: false, Description: "restart policy is armed after the permitted pre-height window"},
	{ReasonCode: "MUTABLE_TARGET_IMAGE", Phase: "handoff", Core: false, Description: "target image is addressed by a mutable tag"},
	{ReasonCode: "MACHINE_CONFIG_DRIFT", Phase: "handoff", Core: true, Description: "machine configuration changed after arming"},
	{ReasonCode: "VOLUME_ID_MISMATCH", Phase: "handoff", Core: false, Description: "machine volume differs from the authorized volume"},
	{ReasonCode: "PREFLIGHT_DIGEST_MISMATCH", Phase: "handoff", Core: false, Description: "activation-preflight report digest differs from the pin"},
	{ReasonCode: "OBSERVER_DISAGREEMENT", Phase: "handoff", Core: false, Description: "independent observer disagrees on chain, H-1 AppHash, or plan"},
	{ReasonCode: "MISSING_RUNTIME_SECRET", Phase: "handoff", Core: false, Description: "required runtime secret or file mount is absent"},
	{ReasonCode: "STAGED_TARGET_DRIFT", Phase: "handoff", Core: false, Description: "staged stopped target tuple differs from authorization"},
	{ReasonCode: "POST_START_APP_HASH_MISMATCH", Phase: "handoff", Core: false, Description: "post-start H AppHash differs from deterministic replay"},
	{ReasonCode: "APPLIED_PLAN_MISSING", Phase: "handoff", Core: false, Description: "the applied upgrade marker is absent or at the wrong height"},

	{ReasonCode: "LOADER_INTERRUPTED", Phase: "upgrade", Core: true, Description: "target process is killed after store-loader staging and before H commit"},
	{ReasonCode: "UPGRADE_HANDLER_FAILED", Phase: "upgrade", Core: false, Description: "upgrade handler returns an error"},
	{ReasonCode: "TARGET_BINARY_MISSING", Phase: "upgrade", Core: false, Description: "authorized target binary cannot be executed"},
	{ReasonCode: "OLD_BINARY_RESTART_ATTEMPT", Phase: "upgrade", Core: false, Description: "automation attempts to restart the legacy binary after H-1"},
	{ReasonCode: "STORAGE_WRITE_FAILURE", Phase: "upgrade", Core: false, Description: "target storage becomes full or unwritable during migration"},
	{ReasonCode: "CONCURRENT_TARGETS", Phase: "upgrade", Core: false, Description: "two target instances attempt to use the validator volume"},

	{ReasonCode: "NON_GUARDIAN_VOTE", Phase: "emergency", Core: false, Description: "an unauthorized account submits an emergency vote"},
	{ReasonCode: "WRAPPED_OR_MIXED_TRANSACTION", Phase: "emergency", Core: false, Description: "an emergency message is wrapped or carries an ordinary tail"},
	{ReasonCode: "NEGATIVE_PRECOMMIT", Phase: "emergency", Core: false, Description: "a negative precommit prevents finalization"},
	{ReasonCode: "VOTE_FLIP", Phase: "emergency", Core: false, Description: "a Guardian attempts to change an immutable vote"},
	{ReasonCode: "COUNCIL_MEMBERSHIP_DRIFT", Phase: "emergency", Core: false, Description: "council membership or activity changes during a ceremony"},
	{ReasonCode: "CEREMONY_DEADLINE_EXPIRED", Phase: "emergency", Core: false, Description: "a ceremony crosses its evidence deadline"},
	{ReasonCode: "ORDINARY_TX_DURING_QUARANTINE", Phase: "emergency", Core: true, Description: "an ordinary transaction is submitted while quarantine is active"},
	{ReasonCode: "RECOVERY_TUPLE_MISMATCH", Phase: "emergency", Core: false, Description: "proposal, submitter, action, plan, or manifest differs from authorization"},
	{ReasonCode: "RECOVERY_DUPLICATE_PROPOSAL", Phase: "emergency", Core: false, Description: "the authorized recovery tuple is submitted more than once"},
	{ReasonCode: "RECOVERY_AUTHORIZATION_REVOKED", Phase: "emergency", Core: true, Description: "a queued recovery proposal loses its authorization"},
	{ReasonCode: "RESUME_EVIDENCE_REPLAY", Phase: "emergency", Core: false, Description: "resume retries unchanged evidence or an old journal head"},
	{ReasonCode: "RESUME_SAME_BLOCK_ADMISSION", Phase: "emergency", Core: false, Description: "ordinary admission is attempted in the resume-finalization block"},
	{ReasonCode: "ICA_CALLBACK_DURING_QUARANTINE", Phase: "emergency", Core: false, Description: "an ICA callback attempts state mutation during quarantine"},

	{ReasonCode: "SOURCE_PROCESS_RUNNING", Phase: "restore", Core: false, Description: "fresh-volume verification runs while the source signer is active"},
	{ReasonCode: "RESTORE_PARTIAL_COPY", Phase: "restore", Core: false, Description: "one or more complete-home files are absent"},
	{ReasonCode: "RESTORE_CONTENT_DRIFT", Phase: "restore", Core: false, Description: "restored file bytes, sizes, or modes differ"},
	{ReasonCode: "RESTORE_SYMLINK_OR_SPECIAL", Phase: "restore", Core: false, Description: "restored home contains a symlink or special file"},
	{ReasonCode: "RESTORE_GENESIS_MISMATCH", Phase: "restore", Core: false, Description: "restored genesis identity differs from the source"},
	{ReasonCode: "RESTORE_IDENTITY_MISMATCH", Phase: "restore", Core: false, Description: "consensus address or node identity differs unexpectedly"},
	{ReasonCode: "RESTORE_SIGNING_STATE_DRIFT", Phase: "restore", Core: false, Description: "validator signing state differs from the clean-stop manifest"},
	{ReasonCode: "RESTORE_DATABASE_SET_MISMATCH", Phase: "restore", Core: false, Description: "application, blockstore, and Comet state files are not one snapshot"},
	{ReasonCode: "RESTORE_CONCURRENT_SIGNER", Phase: "restore", Core: true, Description: "source and destination signer processes overlap"},

	{ReasonCode: "FAIL_STOP_API_FAILURE", Phase: "control_plane", Core: true, Description: "control plane cannot stop a target after a post-start violation"},
}

func faultDefinitionMap() map[string]FaultDefinition {
	result := make(map[string]FaultDefinition, len(faultDefinitions))
	for _, definition := range faultDefinitions {
		result[definition.ReasonCode] = definition
	}
	return result
}

func buildFaultMatrix() (FaultMatrix, error) {
	definitions := append([]FaultDefinition(nil), faultDefinitions...)
	sort.Slice(definitions, func(i, j int) bool {
		return definitions[i].ReasonCode < definitions[j].ReasonCode
	})
	matrix := FaultMatrix{
		Schema:      faultMatrixSchema,
		Definitions: definitions,
	}
	forHash := matrix
	forHash.SHA256 = ""
	digest, err := hashCanonical(forHash)
	if err != nil {
		return FaultMatrix{}, err
	}
	matrix.SHA256 = digest
	return matrix, nil
}

func requiredFaultCodes(mode string) (map[string]struct{}, error) {
	required := make(map[string]struct{})
	switch mode {
	case "quick":
		for _, definition := range faultDefinitions {
			if definition.Core {
				required[definition.ReasonCode] = struct{}{}
			}
		}
	case "full":
		for _, definition := range faultDefinitions {
			required[definition.ReasonCode] = struct{}{}
		}
	default:
		return nil, fmt.Errorf("mode must be quick or full")
	}
	return required, nil
}
